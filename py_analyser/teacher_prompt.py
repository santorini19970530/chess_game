#!/usr/bin/env python3
# teacher_prompt.py - terms/tone assembly and quick coach lines

from __future__ import annotations

import json
import re
from functools import lru_cache
from pathlib import Path
from typing import Any

from analyzer import build_move_ground_truth
from explain_finalize import _normalize_side, finalize_explanation

_DATA_DIR = Path(__file__).resolve().parent / "data"
_SKILL_LEVELS = frozenset({"beginner", "intermediate", "advanced"})
_UCI_TOKEN = re.compile(r"^[a-h](?:[1-9]|10)[a-h](?:[1-9]|10)[qrbn]?$", re.I)


# TeacherPrompt - builds teacher prompts and quick coach lines from terms/tone json
class TeacherPrompt:
    # build - assembles the ollama prompt from terms/tone json by skill level
    def build(
        self,
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
        level = self.normalize_skill_level(skill_level)
        terms_file, tone_file = self._terms_tone_filenames(game_type)
        terms = self._level_block(terms_file, level)
        tone = self._level_block(tone_file, level)

        ground = build_move_ground_truth(
            fen=fen,
            move_uci=move_uci,
            move_history=move_history,
            game_type=game_type,
        )
        # prefer real san from board replay; ignore uci mistakenly sent as move_san
        move_text = (ground.get("san") or "").strip()
        if not move_text:
            candidate = (move_san or "").strip()
            move_text = (
                candidate
                if candidate and not self.looks_like_uci(candidate)
                else (move_uci or "").strip()
            )
        history = move_history or []
        history_str = " ".join(history[-6:]) if history else "(no prior moves)"
        game = self._game_label(game_type)
        to_move = self.side_to_move_from_fen(fen, side_to_move)
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
        # keep prompt short — long glossaries slow local ollama prefill a lot
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

    # build_quick - builds instant coach text from ground truth without ollama
    def build_quick(
        self,
        *,
        fen: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
        game_type: str = "chess",
        human_color: str | None = None,
        side_to_move: str = "white",
    ) -> str:
        ground = build_move_ground_truth(
            fen=fen,
            move_uci=move_uci,
            move_history=move_history,
            game_type=game_type,
        )
        san = (ground.get("san") or move_san or move_uci or "").strip()
        summary = str(ground.get("summary") or "").lower()
        to_move = self.side_to_move_from_fen(fen, side_to_move)
        # tip / diagram loads have no last ply — coach the position, not a fake "played" move
        if not self.looks_like_uci(move_uci) or (move_uci or "").strip().lower() in {
            "position",
            "diagram",
        }:
            seed = (
                f"Tip position — {to_move} to move. "
                f"Watch checks, captures, and loose pieces; use the suggested moves."
            )
            return finalize_explanation(
                seed,
                move_san="",
                last_mover="",
                human_color=human_color,
                ground_summary=str(ground.get("summary") or ""),
            )
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

    # looks_like_uci - reports whether a string looks like a bare uci move token
    def looks_like_uci(self, move: str | None) -> bool:
        return bool(_UCI_TOKEN.match(str(move or "").strip()))

    # normalize_skill_level - maps missing or invalid skill_level to intermediate
    def normalize_skill_level(self, skill_level: str | None) -> str:
        level = (skill_level or "").strip().lower()
        return level if level in _SKILL_LEVELS else "intermediate"

    # side_to_move_from_fen - reads side to move from fen, else uses the fallback color
    def side_to_move_from_fen(self, fen: str, fallback: str = "white") -> str:
        parts = (fen or "").split()
        if len(parts) >= 2 and parts[1] in {"w", "b"}:
            return "white" if parts[1] == "w" else "black"
        return _normalize_side(fallback)

    # _game_label - returns a short display label for a game type
    def _game_label(self, game_type: str) -> str:
        gt = (game_type or "chess").strip().lower()
        if gt in {"xianqi", "xiangqi"}:
            return "Xiangqi (Chinese chess)"
        if gt == "shogi":
            return "Shogi (Japanese chess)"
        return "Chess"

    # _terms_tone_game_key - returns the filename stem for terms/tone json by game type
    def _terms_tone_game_key(self, game_type: str) -> str:
        gt = (game_type or "chess").strip().lower()
        if gt in {"xianqi", "xiangqi"}:
            return "xiangqi"
        if gt == "shogi":
            return "shogi"
        return "chess"

    # _terms_tone_filenames - returns terms/tone filenames for game_type, falling back to chess
    def _terms_tone_filenames(self, game_type: str) -> tuple[str, str]:
        key = self._terms_tone_game_key(game_type)
        terms = f"{key}_terms.json"
        tone = f"{key}_tone.json"
        if not (_DATA_DIR / terms).is_file() or not (_DATA_DIR / tone).is_file():
            return "chess_terms.json", "chess_tone.json"
        return terms, tone

    # _load_json_file - loads a terms/tone json file from the data directory
    @staticmethod
    @lru_cache(maxsize=16)
    def _load_json_file(filename: str) -> dict[str, Any]:
        path = _DATA_DIR / filename
        data = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(data, dict):
            return {}
        return data

    # _level_block - returns the skill-level block from a terms or tone json file
    def _level_block(self, filename: str, skill_level: str) -> dict[str, Any]:
        data = self._load_json_file(filename)
        level = self.normalize_skill_level(skill_level)
        block = data.get(level) or data.get("intermediate") or {}
        return block if isinstance(block, dict) else {}


_TEACHER = TeacherPrompt()


# looks_like_uci - public wrapper around TeacherPrompt.looks_like_uci
def looks_like_uci(move: str | None) -> bool:
    return _TEACHER.looks_like_uci(move)


# normalize_skill_level - public wrapper around TeacherPrompt.normalize_skill_level
def normalize_skill_level(skill_level: str | None) -> str:
    return _TEACHER.normalize_skill_level(skill_level)


# side_to_move_from_fen - public wrapper around TeacherPrompt.side_to_move_from_fen
def side_to_move_from_fen(fen: str, fallback: str = "white") -> str:
    return _TEACHER.side_to_move_from_fen(fen, fallback)


# back-compat alias for older callers/tests
_side_to_move_from_fen = side_to_move_from_fen


# _terms_tone_filenames - public-module alias used by tests
def _terms_tone_filenames(game_type: str) -> tuple[str, str]:
    return _TEACHER._terms_tone_filenames(game_type)


# build_teacher_prompt - public wrapper around TeacherPrompt.build
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
    return _TEACHER.build(
        fen=fen,
        move_uci=move_uci,
        move_san=move_san,
        move_history=move_history,
        game_type=game_type,
        skill_level=skill_level,
        side_to_move=side_to_move,
        human_color=human_color,
        concept_hints=concept_hints,
    )


# build_quick_coach_line - public wrapper around TeacherPrompt.build_quick
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
    return _TEACHER.build_quick(
        fen=fen,
        move_uci=move_uci,
        move_san=move_san,
        move_history=move_history,
        game_type=game_type,
        human_color=human_color,
        side_to_move=side_to_move,
    )
