#!/usr/bin/env python3
# explain_finalize.py - ExplainFinalizer helpers for coach text (beside LLMProvider)

from __future__ import annotations

import re

# player-facing text must never echo engine fen (prompt leak from small local models)
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


# ExplainFinalizer - sanitizes and finalizes coach text around the last move
class ExplainFinalizer:
    # sanitize - strips leaked fen and label noise from coach text
    def sanitize(self, text: str) -> str:
        cleaned = _FEN_TOKEN.sub(" ", text or "")
        cleaned = _BARE_FEN.sub(" ", cleaned)
        cleaned = _INTERNAL_FEN_NOISE.sub(" ", cleaned)
        cleaned = re.sub(r"(?i)\bzwischenzug\b", "in-between idea", cleaned)
        cleaned = re.sub(r"[ \t]+", " ", cleaned)
        cleaned = re.sub(r"\n{3,}", "\n\n", cleaned)
        return cleaned.strip()

    # finalize - forces a correct opener and drops invented follow-up fluff
    def finalize(
        self,
        text: str,
        *,
        move_san: str | None,
        last_mover: str,
        human_color: str | None = None,
        ground_summary: str = "",
        concept_hints: list[str] | None = None,
    ) -> str:
        cleaned = self.sanitize(text)
        san = (move_san or "").strip()
        if not san:
            return cleaned

        mover = self._normalize_side(last_mover)
        human = self._normalize_side(human_color) if human_color else ""
        if human and mover == human:
            opener = f"You played {san}."
        else:
            opener = f"{mover.capitalize()} played {san}."

        body = cleaned
        if re.match(re.escape(opener), body, flags=re.IGNORECASE):
            rest = body[len(opener) :].lstrip(" \n")
        else:
            # multi-word labels e.g. "pawn g7→g6" — do not stop at first token
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
        # also "silver g1→f2" style labels
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

        piece_alt = (
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
            # invented squares (e.g. "pawn on d1" when move was f7e6)
            claimed_sq = set(re.findall(r"\b[a-i]\d{1,2}\b", sl))
            if claimed_sq and not claimed_sq.issubset(allowed_squares):
                continue
            # invented drops / uci not present in ground truth or cues
            claimed_moves = set(re.findall(r"\b[plnsgbr][*@][a-i][1-9]\b", sl, flags=re.I))
            claimed_moves |= set(re.findall(r"\b[a-i]\d{1,2}[a-i]\d{1,2}\+?\b", sl, flags=re.I))
            if claimed_moves and not claimed_moves.issubset(allowed_moves):
                continue
            if ("xiangqi" in allow or "shogi" in allow) and re.search(
                r"\b(cent(?:er|re)|castling|en passant)\b", sl
            ):
                continue
            # retract / undo the move just played (illegal nonsense, esp. pawns)
            if re.search(
                r"\b(back to (its )?original|moving it back|move it back|"
                r"return(ing)? (it )?to|undo(ing)? (the )?move)\b",
                sl,
            ):
                continue
            yours = re.search(rf"\byour ({piece_alt})\b", sl)
            if yours:
                claimed = yours.group(1)
                # after opponent's move, "your <that piece>" is wrong pov
                if human and mover and human != mover:
                    continue
                if moved_piece and claimed not in moved_piece and claimed not in allow:
                    continue
            # drop mid-generation cutoffs (num_predict) like "so let's"
            if not re.search(r"[.!?]$", sentence):
                continue
            # empty coach filler (not a teaching idea)
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
        # prefer a boring true line over opener-only silence after filters strip llm junk
        idea = self._safe_coach_followup(
            ground_summary=ground_summary,
            concept_hints=concept_hints,
            last_mover=mover,
            human_color=human or None,
            move_san=san,
        )
        return f"{opener} {idea}".strip()

    # _normalize_side - maps color strings to white or black
    def _normalize_side(self, color: str | None) -> str:
        c = (color or "").strip().lower()
        if c in {"black", "b"}:
            return "black"
        return "white"

    # _piece_from_move_label - extracts a piece word from a move label for follow-up text
    def _piece_from_move_label(self, move_san: str | None) -> str:
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

    # _is_drop_label - reports whether a move label is a shogi drop
    def _is_drop_label(self, move_san: str | None) -> bool:
        return (move_san or "").strip().lower().startswith("drop ")

    # _replies_from_cues - pulls up to two suggested reply tokens from concept hints
    def _replies_from_cues(self, concept_hints: list[str] | None) -> list[str]:
        for raw in concept_hints or []:
            c = str(raw).strip()
            if "engine suggested replies" not in c.lower():
                continue
            tail = c.split(":", 1)[-1].strip().rstrip(".")
            return [p.strip() for p in re.split(r"[,;]", tail) if p.strip()][:2]
        return []

    # _safe_coach_followup - builds a deterministic second sentence when llm follow-up is empty
    def _safe_coach_followup(
        self,
        *,
        ground_summary: str = "",
        concept_hints: list[str] | None = None,
        last_mover: str = "white",
        human_color: str | None = None,
        move_san: str | None = None,
    ) -> str:
        allow = (ground_summary or "").lower()
        mover = self._normalize_side(last_mover)
        human = self._normalize_side(human_color) if human_color else ""
        piece = self._piece_from_move_label(move_san)
        replies = self._replies_from_cues(concept_hints)
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
        if self._is_drop_label(move_san):
            if human and mover and human != mover:
                return f"A {piece} was dropped — ask what it threatens, then use suggested moves."
            return f"You dropped a {piece} — check whether it is safe and what it attacks."
        if human and mover and human != mover:
            return f"Ask what that {piece} move threatens, then check the suggested replies."
        if "xiangqi" in allow or "shogi" in allow:
            return f"Solid {piece} move — watch checks, captures, and piece safety."
        return "Watch checks, captures, and loose pieces."


_FINALIZER = ExplainFinalizer()


# sanitize_explanation - public wrapper around ExplainFinalizer.sanitize
def sanitize_explanation(text: str) -> str:
    return _FINALIZER.sanitize(text)


# finalize_explanation - public wrapper around ExplainFinalizer.finalize
def finalize_explanation(
    text: str,
    *,
    move_san: str | None,
    last_mover: str,
    human_color: str | None = None,
    ground_summary: str = "",
    concept_hints: list[str] | None = None,
) -> str:
    return _FINALIZER.finalize(
        text,
        move_san=move_san,
        last_mover=last_mover,
        human_color=human_color,
        ground_summary=ground_summary,
        concept_hints=concept_hints,
    )


# _normalize_side - public-module alias for callers that only need color mapping
def _normalize_side(color: str | None) -> str:
    return _FINALIZER._normalize_side(color)
