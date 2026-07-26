from __future__ import annotations

import json
import os
import re
import urllib.error
import urllib.request
from functools import lru_cache
from pathlib import Path
from typing import Any, Protocol

from analyzer import build_explanation_fallback, build_move_ground_truth

_DATA_DIR = Path(__file__).resolve().parent / "data"
_SKILL_LEVELS = frozenset({"beginner", "intermediate", "advanced"})
# Player-facing text must never echo engine FEN (prompt leak from small local models).
_FEN_TOKEN = re.compile(
    r"(?i)\bFEN\s*:\s*"
    r"(?:[rnbqkpRNBQKP1-8]+/){7}[rnbqkpRNBQKP1-8]+"
    r"(?:\s+[wb]\s+(?:[KQkq]+|-)\s+(?:[a-h][1-8]|-)\s+\d+\s+\d+)?"
)
_BARE_FEN = re.compile(
    r"\b(?:[rnbqkpRNBQKP1-8]+/){7}[rnbqkpRNBQKP1-8]+"
    r"(?:\s+[wb]\s+(?:[KQkq]+|-)\s+(?:[a-h][1-8]|-)\s+\d+\s+\d+)?\b"
)
_INTERNAL_FEN_NOISE = re.compile(
    r"(?i)\(?\s*internal board(?:\s+fen)?[^)\n]*\)?\.?"
)
_UCI_TOKEN = re.compile(r"^[a-h](?:[1-9]|10)[a-h](?:[1-9]|10)[qrbn]?$", re.I)


def looks_like_uci(move: str | None) -> bool:
    return bool(_UCI_TOKEN.match(str(move or "").strip()))


def sanitize_explanation(text: str) -> str:
    """Strip leaked FEN / label noise from coach text."""
    cleaned = _FEN_TOKEN.sub(" ", text or "")
    cleaned = _BARE_FEN.sub(" ", cleaned)
    cleaned = _INTERNAL_FEN_NOISE.sub(" ", cleaned)
    cleaned = re.sub(r"(?i)\bzwischenzug\b", "in-between idea", cleaned)
    cleaned = re.sub(r"[ \t]+", " ", cleaned)
    cleaned = re.sub(r"\n{3,}", "\n\n", cleaned)
    return cleaned.strip()


_PIECE_NAMES = (
    "promoted silver",
    "promoted knight",
    "promoted lance",
    "tokin",
    "dragon",
    "horse",
    "pawn",
    "lance",
    "knight",
    "silver",
    "gold",
    "bishop",
    "rook",
    "queen",
    "king",
    "general",
    "advisor",
    "elephant",
    "chariot",
    "cannon",
    "soldier",
    "piece",
)


def _piece_from_move_label(move_san: str | None) -> str:
    s = (move_san or "").strip().lower()
    if not s:
        return "piece"
    if s.startswith("drop "):
        rest = s[5:].replace("→", " ").strip()
        tok = rest.split()[0] if rest else "piece"
        return tok if tok in {p.split()[-1] for p in _PIECE_NAMES} or tok in _PIECE_NAMES else "piece"
    for name in _PIECE_NAMES:
        if s.startswith(name):
            return name
    return "piece"


def _is_drop_label(move_san: str | None) -> bool:
    return (move_san or "").strip().lower().startswith("drop ")


def _replies_from_cues(concept_hints: list[str] | None) -> list[str]:
    for raw in concept_hints or []:
        c = str(raw).strip()
        if "engine suggested replies" not in c.lower():
            continue
        tail = c.split(":", 1)[-1].strip().rstrip(".")
        return [p.strip() for p in re.split(r"[,;]", tail) if p.strip()][:2]
    return []


def safe_coach_followup(
    *,
    ground_summary: str = "",
    concept_hints: list[str] | None = None,
    last_mover: str = "white",
    human_color: str | None = None,
    move_san: str | None = None,
) -> str:
    """Deterministic second sentence when Ollama follow-up is stripped or empty."""
    allow = (ground_summary or "").lower()
    mover = _normalize_side(last_mover)
    human = _normalize_side(human_color) if human_color else ""
    piece = _piece_from_move_label(move_san)
    replies = _replies_from_cues(concept_hints)
    for raw in concept_hints or []:
        cl = str(raw).lower()
        if "check" in cl and "roughly balanced" not in cl and "engine suggested" not in cl:
            return "There is a check idea in the cues — deal with that first."
    if replies:
        shown = ", ".join(replies)
        if human and mover and human != mover:
            return f"That {piece} move is done — candidate replies include {shown}."
        return f"Candidate replies include {shown}."
    if "capturing" in allow:
        return "That was a capture — check whether the piece is safe."
    if "attacks:" in allow:
        return "It now presses the pieces listed in the ground truth."
    if _is_drop_label(move_san):
        if human and mover and human != mover:
            return f"A {piece} was dropped — ask what it threatens, then use suggested moves."
        return f"You dropped a {piece} — check whether it is safe and what it attacks."
    if human and mover and human != mover:
        return f"Ask what that {piece} move threatens, then check the suggested replies."
    if "xiangqi" in allow or "shogi" in allow:
        return f"Solid {piece} move — watch checks, captures, and piece safety."
    return "Watch checks, captures, and loose pieces."


def finalize_explanation(
    text: str,
    *,
    move_san: str | None,
    last_mover: str,
    human_color: str | None = None,
    ground_summary: str = "",
    concept_hints: list[str] | None = None,
) -> str:
    """Force correct opener + drop follow-up fluff that invents pieces/forks."""
    cleaned = sanitize_explanation(text)
    san = (move_san or "").strip()
    if not san:
        return cleaned

    mover = _normalize_side(last_mover)
    human = _normalize_side(human_color) if human_color else ""
    if human and mover == human:
        opener = f"You played {san}."
    else:
        opener = f"{mover.capitalize()} played {san}."

    body = cleaned
    if re.match(re.escape(opener), body, flags=re.IGNORECASE):
        rest = body[len(opener) :].lstrip(" \n")
    else:
        # Multi-word labels e.g. "pawn g7→g6" — do not stop at first token.
        rest = re.sub(
            r"^(You|Black|White)\s+played\s+[^.]+?\.\s*",
            "",
            body,
            count=1,
            flags=re.IGNORECASE,
        ).strip()

    allow = (ground_summary or "").lower()
    moved_piece = ""
    pm = re.search(
        r"(?:moved|move)\s+(drop\s+)?([a-z]+(?:\s+[a-z]+)?)",
        ground_summary or "",
        flags=re.IGNORECASE,
    )
    if pm:
        moved_piece = (pm.group(2) or "").lower()
    # Also "silver g1→f2" style labels.
    if not moved_piece:
        pm2 = re.search(
            r"\b(drop\s+)?(pawn|lance|knight|silver|gold|bishop|rook|king|tokin|horse|dragon|"
            r"general|advisor|elephant|chariot|cannon|soldier|promoted\s+\w+|piece)\b",
            ground_summary or "",
            flags=re.IGNORECASE,
        )
        if pm2:
            moved_piece = pm2.group(2).lower()

    cue_blob = " ".join(str(h) for h in (concept_hints or [])).lower()
    allow_blob = f"{allow} {cue_blob} {san.lower()}"
    allowed_squares = set(re.findall(r"\b[a-i]\d{1,2}\b", allow_blob))
    allowed_moves = set(re.findall(r"\b[plnsgbr][*@][a-i][1-9]\b", allow_blob, flags=re.I))
    allowed_moves |= set(re.findall(r"\b[a-i]\d{1,2}[a-i]\d{1,2}\+?\b", allow_blob, flags=re.I))

    _PIECE = (
        r"pawn|lance|knight|silver|gold|bishop|rook|queen|king|tokin|horse|dragon|"
        r"general|advisor|elephant|chariot|cannon|soldier"
    )

    follow: list[str] = []
    for part in re.split(r"(?<=[.!?])\s+", rest) if rest else []:
        sentence = part.strip()
        if not sentence:
            continue
        sl = sentence.lower()
        if "fork" in sl and "fork" not in allow and "attacks:" not in allow:
            continue
        if re.search(r"\bknight fork\b", sl) and "knight" not in allow:
            continue
        # Invented squares (e.g. "pawn on d1" when move was f7e6).
        claimed_sq = set(re.findall(r"\b[a-i]\d{1,2}\b", sl))
        if claimed_sq and not claimed_sq.issubset(allowed_squares):
            continue
        # Invented drops / UCI not present in ground truth or cues.
        claimed_moves = set(re.findall(r"\b[plnsgbr][*@][a-i][1-9]\b", sl, flags=re.I))
        claimed_moves |= set(re.findall(r"\b[a-i]\d{1,2}[a-i]\d{1,2}\+?\b", sl, flags=re.I))
        if claimed_moves and not claimed_moves.issubset(allowed_moves):
            continue
        if ("xiangqi" in allow or "shogi" in allow) and re.search(
            r"\b(cent(?:er|re)|castling|en passant)\b", sl
        ):
            continue
        # Retract / undo the move just played (illegal nonsense, esp. pawns).
        if re.search(
            r"\b(back to (its )?original|moving it back|move it back|"
            r"return(ing)? (it )?to|undo(ing)? (the )?move)\b",
            sl,
        ):
            continue
        yours = re.search(rf"\byour ({_PIECE})\b", sl)
        if yours:
            claimed = yours.group(1)
            # After opponent's move, "your <that piece>" is wrong POV.
            if human and mover and human != mover:
                continue
            if moved_piece and claimed not in moved_piece and claimed not in allow:
                continue
        # Drop mid-generation cutoffs (num_predict) like "so let's"
        if not re.search(r"[.!?]$", sentence):
            continue
        # Empty coach filler (not a teaching idea).
        if re.search(
            r"\b(now )?it'?s your (turn|move)\b|"
            r"\byour (turn|move)[!.,]?\s*$|"
            r"\b(white|black) to move\b|"
            r"\bsafe hand\b",
            sl,
        ):
            continue
        if re.fullmatch(r"placeholder\.?", sl):
            continue
        follow.append(sentence)
        break  # at most one follow-up sentence

    if follow:
        return f"{opener} {' '.join(follow)}".strip()
    # Prefer a boring true line over opener-only silence after filters strip LLM junk.
    idea = safe_coach_followup(
        ground_summary=ground_summary,
        concept_hints=concept_hints,
        last_mover=mover,
        human_color=human or None,
        move_san=san,
    )
    return f"{opener} {idea}".strip()


def _game_label(game_type: str) -> str:
    gt = (game_type or "chess").strip().lower()
    if gt in {"xianqi", "xiangqi"}:
        return "Xiangqi (Chinese chess)"
    if gt == "shogi":
        return "Shogi (Japanese chess)"
    return "Chess"


def _terms_tone_game_key(game_type: str) -> str:
    """Filename stem for terms/tone JSON (session id xianqi → xiangqi_*)."""
    gt = (game_type or "chess").strip().lower()
    if gt in {"xianqi", "xiangqi"}:
        return "xiangqi"
    if gt == "shogi":
        return "shogi"
    return "chess"


def _terms_tone_filenames(game_type: str) -> tuple[str, str]:
    """Return terms/tone filenames for game_type; fall back to chess if missing."""
    key = _terms_tone_game_key(game_type)
    terms = f"{key}_terms.json"
    tone = f"{key}_tone.json"
    if not (_DATA_DIR / terms).is_file() or not (_DATA_DIR / tone).is_file():
        return "chess_terms.json", "chess_tone.json"
    return terms, tone


def normalize_skill_level(skill_level: str | None) -> str:
    level = (skill_level or "").strip().lower()
    return level if level in _SKILL_LEVELS else "intermediate"


@lru_cache(maxsize=16)
def _load_json_file(filename: str) -> dict[str, Any]:
    path = _DATA_DIR / filename
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        return {}
    return data


def _level_block(filename: str, skill_level: str) -> dict[str, Any]:
    data = _load_json_file(filename)
    level = normalize_skill_level(skill_level)
    block = data.get(level) or data.get("intermediate") or {}
    return block if isinstance(block, dict) else {}


def _normalize_side(color: str | None) -> str:
    c = (color or "").strip().lower()
    if c in {"black", "b"}:
        return "black"
    return "white"


def side_to_move_from_fen(fen: str, fallback: str = "white") -> str:
    parts = (fen or "").split()
    if len(parts) >= 2 and parts[1] in {"w", "b"}:
        return "white" if parts[1] == "w" else "black"
    return _normalize_side(fallback)


# Back-compat alias for older callers/tests.
_side_to_move_from_fen = side_to_move_from_fen


def build_teacher_prompt(
    *,
    fen: str,
    move_uci: str,
    move_san: str | None,
    move_history: list[str] | None,
    game_type: str = "chess",
    skill_level: str = "intermediate",
    side_to_move: str = "white",
    human_color: str | None = None,
    concept_hints: list[str] | None = None,
) -> str:
    """Assemble Ollama prompt from {game}_terms.json + {game}_tone.json by skill_level."""
    level = normalize_skill_level(skill_level)
    terms_file, tone_file = _terms_tone_filenames(game_type)
    terms = _level_block(terms_file, level)
    tone = _level_block(tone_file, level)

    ground = build_move_ground_truth(
        fen=fen,
        move_uci=move_uci,
        move_history=move_history,
        game_type=game_type,
    )
    # Prefer real SAN from board replay; ignore UCI mistakenly sent as move_san.
    move_text = (ground.get("san") or "").strip()
    if not move_text:
        candidate = (move_san or "").strip()
        move_text = candidate if candidate and not looks_like_uci(candidate) else (move_uci or "").strip()
    history = move_history or []
    history_str = " ".join(history[-6:]) if history else "(no prior moves)"
    game = _game_label(game_type)
    to_move = side_to_move_from_fen(fen, side_to_move)
    last_mover = "black" if to_move == "white" else "white"
    human = _normalize_side(human_color) if human_color else ""
    cues = [str(h).strip() for h in (concept_hints or []) if str(h).strip()][:3]

    voice = str(tone.get("voice") or f"{game} coach").strip()
    style_rules = tone.get("style_rules") or []
    if isinstance(style_rules, list):
        style_line = "; ".join(str(r).strip() for r in style_rules if str(r).strip())
    else:
        style_line = str(style_rules).strip()

    allowed = terms.get("allowed_terms") or []
    # Keep prompt short — long glossaries slow local Ollama prefill a lot.
    if isinstance(allowed, list):
        terms_line = ", ".join([str(t).strip() for t in allowed if str(t).strip()][:8])
    else:
        terms_line = ""

    you_or_side = "You played" if human and last_mover == human else f"{last_mover.capitalize()} played"
    parts = [
        f"You are a {voice} for {game}. Skill: {level}.",
        str(ground.get("summary") or "GROUND TRUTH: unavailable."),
        f'Only explain {move_text}. {to_move.capitalize()} to move. '
        f'Open with "{you_or_side} {move_text}." then ONE short idea (≤45 words total).',
        "Use ONLY GROUND TRUTH + cues. No invented forks/pieces/captures/drops. "
        "No next-move UCI/SAN/drop unless that exact token appears in cues. "
        "If unsure, one clause pointing at suggested replies — never invent tactics.",
        f"Recent moves: {history_str}.",
    ]
    if cues:
        parts.append("Cues: " + " | ".join(cues) + ".")
    if human in {"white", "black"}:
        if last_mover == human:
            parts.append(f"Speak as 'you' ({human}).")
        else:
            parts.append(
                f"Advise YOU ({human}) after naming {last_mover}'s move; "
                f"do not call {last_mover}'s piece 'your' piece."
            )
    if style_line:
        parts.append(f"Style: {style_line}.")
    if terms_line:
        parts.append(f"Ok terms: {terms_line}.")
    return " ".join(parts)


def build_quick_coach_line(
    *,
    fen: str,
    move_uci: str,
    move_san: str | None,
    move_history: list[str] | None,
    game_type: str = "chess",
    human_color: str | None = None,
    side_to_move: str = "white",
) -> str:
    """Instant coach text from ground truth (no Ollama). Fills notes while LLM runs."""
    ground = build_move_ground_truth(
        fen=fen,
        move_uci=move_uci,
        move_history=move_history,
        game_type=game_type,
    )
    san = (ground.get("san") or move_san or move_uci or "").strip()
    summary = str(ground.get("summary") or "").lower()
    if "capturing" in summary:
        idea = "That was a capture — check whether the piece is safe."
    elif "attacks:" in summary:
        idea = "It now presses the pieces listed in the ground truth."
    elif "attacks no enemy piece" in summary:
        idea = "Quiet move — watch development and safety."
    elif "xiangqi" in summary or "shogi" in summary:
        idea = "Watch checks, captures, and piece safety."
    else:
        idea = "Watch checks, captures, and loose pieces."
    to_move = side_to_move_from_fen(fen, side_to_move)
    last_mover = "black" if to_move == "white" else "white"
    mover = _normalize_side(last_mover)
    human = _normalize_side(human_color) if human_color else ""
    if human and mover == human:
        seed = f"You played {san}. {idea}"
    else:
        seed = f"{mover.capitalize()} played {san}. {idea}"
    return finalize_explanation(
        seed,
        move_san=san,
        last_mover=last_mover,
        human_color=human_color,
        ground_summary=str(ground.get("summary") or ""),
    )


class LLMProvider(Protocol):
    """Minimal protocol for LLM explanation providers."""

    name: str

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        ...


class OllamaProvider:
    """Calls a local Ollama instance (default path)."""

    name = "ollama"

    def __init__(self, model: str | None = None, timeout: float | None = None) -> None:
        self.model = model or os.getenv("OLLAMA_MODEL", "gemma2:2b")
        timeout_ms_raw = os.getenv("OLLAMA_TIMEOUT_MS", "15000").strip()
        try:
            timeout_ms = max(1000, int(timeout_ms_raw))
        except ValueError:
            timeout_ms = 15000
        self.timeout = timeout if timeout is not None else (timeout_ms / 1000.0)
        self.url = os.getenv("OLLAMA_URL", "http://localhost:11434/api/generate").strip()

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        prompt = build_teacher_prompt(
            fen=fen,
            move_uci=move_uci,
            move_san=move_san,
            move_history=move_history,
            game_type=game_type,
            skill_level=skill_level,
            side_to_move=color,
            human_color=human_color,
            concept_hints=concept_hints,
        )

        # Cap tokens: 40 often cut mid-sentence; ~80 fits opener + one short idea.
        try:
            num_predict = max(48, int(os.getenv("OLLAMA_NUM_PREDICT", "96")))
        except ValueError:
            num_predict = 96
        try:
            num_ctx = max(512, int(os.getenv("OLLAMA_NUM_CTX", "1024")))
        except ValueError:
            num_ctx = 1024
        body = {
            "model": self.model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "num_predict": num_predict,
                "num_ctx": num_ctx,
                "temperature": 0.3,
            },
        }
        req = urllib.request.Request(
            self.url,
            data=json.dumps(body).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=self.timeout) as resp:
            if resp.status != 200:
                raise urllib.error.HTTPError(self.url, resp.status, "bad status", {}, None)
            data = json.loads(resp.read().decode("utf-8"))
            text = (data.get("response") or "").strip()
            if not text:
                raise ValueError("empty response from ollama")
            return sanitize_explanation(text)


class HeuristicProvider:
    """Pure rule-based fallback (no external dependency)."""

    name = "heuristic"

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None = None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        _ = skill_level
        _ = human_color
        _ = concept_hints
        return build_explanation_fallback(
            fen=fen,
            color=color,
            move_uci=move_uci or "",
            move_san=move_san,
            game_type=game_type,
        )


def get_llm_provider() -> LLMProvider:
    """Factory: returns provider based on LLM_PROVIDER env var."""
    name = os.getenv("LLM_PROVIDER", "ollama").lower().strip()
    if name in ("heuristic", "fallback", "offline"):
        return HeuristicProvider()
    return OllamaProvider()


if __name__ == "__main__" or os.getenv("LLM_PROVIDER_SELFCHECK"):
    p1 = get_llm_provider()
    p2 = HeuristicProvider()
    assert hasattr(p1, "explain") and hasattr(p2, "explain")
    fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
    beg = build_teacher_prompt(
        fen=fen, move_uci="e2e4", move_san="e4", move_history=[], skill_level="beginner"
    )
    adv = build_teacher_prompt(
        fen=fen, move_uci="e2e4", move_san="e4", move_history=[], skill_level="advanced"
    )
    assert "Skill level: beginner" in beg and "everyday" in beg.lower()
    assert "Skill level: advanced" in adv and "prophylaxis" in adv
    assert beg != adv
