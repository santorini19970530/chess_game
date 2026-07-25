from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from functools import lru_cache
from pathlib import Path
from typing import Any, Protocol

from analyzer import build_explanation_fallback

_DATA_DIR = Path(__file__).resolve().parent / "data"
_SKILL_LEVELS = frozenset({"beginner", "intermediate", "advanced"})


def _game_label(game_type: str) -> str:
    gt = (game_type or "chess").strip().lower()
    if gt == "xianqi":
        return "Xiangqi (Chinese chess)"
    if gt == "shogi":
        return "Shogi (Japanese chess)"
    return "Chess"


def normalize_skill_level(skill_level: str | None) -> str:
    level = (skill_level or "").strip().lower()
    return level if level in _SKILL_LEVELS else "intermediate"


@lru_cache(maxsize=4)
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


def _side_to_move_from_fen(fen: str, fallback: str = "white") -> str:
    parts = (fen or "").split()
    if len(parts) >= 2 and parts[1] in {"w", "b"}:
        return "white" if parts[1] == "w" else "black"
    return _normalize_side(fallback)


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
) -> str:
    """Assemble Ollama prompt from chess_terms.json + chess_tone.json by skill_level."""
    level = normalize_skill_level(skill_level)
    terms = _level_block("chess_terms.json", level)
    tone = _level_block("chess_tone.json", level)

    move_text = move_san or move_uci or ""
    history = move_history or []
    history_str = " ".join(history[-6:]) if history else "(no prior moves)"
    game = _game_label(game_type)
    to_move = _side_to_move_from_fen(fen, side_to_move)
    last_mover = "black" if to_move == "white" else "white"
    human = _normalize_side(human_color) if human_color else ""

    voice = str(tone.get("voice") or f"{game} coach").strip()
    style_rules = tone.get("style_rules") or []
    if isinstance(style_rules, list):
        style_line = "; ".join(str(r).strip() for r in style_rules if str(r).strip())
    else:
        style_line = str(style_rules).strip()

    allowed = terms.get("allowed_terms") or []
    if isinstance(allowed, list):
        terms_line = ", ".join(str(t).strip() for t in allowed if str(t).strip())
    else:
        terms_line = ""

    definitions = terms.get("definitions") or {}
    def_bits: list[str] = []
    if isinstance(definitions, dict):
        for key, val in definitions.items():
            k, v = str(key).strip(), str(val).strip()
            if k and v:
                def_bits.append(f"{k}: {v}")
    defs_line = "; ".join(def_bits[:6])
    guidance = str(terms.get("guidance") or "").strip()

    parts = [
        f"You are a {voice}.",
        f"This is {game} — use the correct piece names and rules.",
        f"Skill level: {level}.",
        f"{last_mover.capitalize()} just played {move_text}. {to_move.capitalize()} is to move now.",
        "Hard length limit: at most 2 short sentences, under 45 words total. "
        "No lists, no numbered steps, no section headers, no 'Practical next-step advice' labels.",
        "Only mention pieces/squares that match the FEN; do not invent pieces.",
        f"FEN after the move: {fen}. Recent moves: {history_str}.",
    ]
    if human in {"white", "black"}:
        if last_mover == human:
            parts.append(
                f"The human plays {human.capitalize()} and just moved. "
                f"Speak to them as 'you': what you did, then what to watch next."
            )
        else:
            parts.append(
                f"The human plays {human.capitalize()}; the opponent just moved. "
                f"One clause on what {last_mover.capitalize()} did, then advise YOU ({human.capitalize()}) only. "
                f"Do not give a plan for {last_mover.capitalize()}."
            )
    else:
        parts.append(
            f"Explain the last move in third person, then advise {to_move.capitalize()} only."
        )
    if style_line:
        parts.append(f"Style: {style_line}.")
    if terms_line:
        parts.append(f"Preferred terms when accurate: {terms_line}.")
    if defs_line:
        parts.append(f"Term hints: {defs_line}.")
    if guidance:
        parts.append(guidance)
    return " ".join(parts)


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
        )

        req = urllib.request.Request(
            self.url,
            data=json.dumps({"model": self.model, "prompt": prompt, "stream": False}).encode("utf-8"),
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
            return text


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
    ) -> str:
        _ = skill_level
        _ = human_color
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
