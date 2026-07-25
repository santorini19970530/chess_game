#!/usr/bin/env python3
"""issue0045: terms/tone JSON + prompt assembly by skill_level."""

from __future__ import annotations

import os
import sys
import unittest

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from llm_providers import (  # noqa: E402
    HeuristicProvider,
    build_teacher_prompt,
    finalize_explanation,
    normalize_skill_level,
    sanitize_explanation,
)


class TestTeacherPrompt(unittest.TestCase):
    FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

    def test_normalize_skill_level(self) -> None:
        self.assertEqual(normalize_skill_level(None), "intermediate")
        self.assertEqual(normalize_skill_level("BEGINNER"), "beginner")
        self.assertEqual(normalize_skill_level("n00b"), "intermediate")

    def test_prompts_differ_by_skill_level(self) -> None:
        kwargs = dict(fen=self.FEN, move_uci="e2e4", move_san="e4", move_history=[])
        beg = build_teacher_prompt(**kwargs, skill_level="beginner")
        mid = build_teacher_prompt(**kwargs, skill_level="intermediate")
        adv = build_teacher_prompt(**kwargs, skill_level="advanced")
        self.assertEqual(len({beg, mid, adv}), 3)
        self.assertIn("friendly teacher", beg)
        self.assertIn("fork", mid)
        self.assertIn("prophylaxis", adv)
        self.assertNotIn("avoid engine jargon", adv.lower())

    def test_prompt_pov_for_human_white_after_black_move(self) -> None:
        # After Black plays e6, White to move — advice must target White, not Black's plan.
        fen = "rnbqkbnr/pppp1ppp/4p3/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2"
        prompt = build_teacher_prompt(
            fen=fen,
            move_uci="e7e6",
            move_san="e6",
            move_history=["e2e4", "e7e6"],
            skill_level="beginner",
            side_to_move="white",
            human_color="white",
        )
        self.assertIn("Black played e6", prompt)
        self.assertIn("White to move", prompt)
        self.assertIn("Advise YOU (white)", prompt)
        self.assertIn("≤45 words", prompt)

    def test_prompt_injects_concept_hints(self) -> None:
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        with_cues = build_teacher_prompt(
            fen=fen,
            move_uci="e2e4",
            move_san="e4",
            move_history=["e2e4"],
            skill_level="intermediate",
            concept_hints=[
                "Black king is in check.",
                "Engine suggested replies (side to move): e5, Nf6.",
            ],
        )
        bare = build_teacher_prompt(
            fen=fen,
            move_uci="e2e4",
            move_san="e4",
            move_history=["e2e4"],
            skill_level="intermediate",
        )
        self.assertIn("GROUND TRUTH", with_cues)
        self.assertIn("attacks no enemy piece", with_cues)
        self.assertIn("Cues:", with_cues)
        self.assertIn("Black king is in check.", with_cues)
        self.assertIn("Engine suggested replies", with_cues)
        self.assertIn("No invented forks", with_cues)
        self.assertIn('Open with "White played e4."', with_cues)
        self.assertNotIn("Cues:", bare)
        self.assertNotIn("Black king is in check.", bare)
        self.assertIn("GROUND TRUTH", bare)
        self.assertIn("Only explain e4", with_cues)

    def test_sanitize_strips_leaked_fen(self) -> None:
        raw = (
            "Black played c6 for structure. "
            "FEN: rnbqkbnr/pp3ppp/2p5/3pp3/4P3/3P1P2/PPP3PP/RNBQKBNR w KQkq - 0 4"
        )
        cleaned = sanitize_explanation(raw)
        self.assertIn("Black played c6", cleaned)
        self.assertNotIn("FEN:", cleaned)
        self.assertNotIn("rnbqkbnr/pp3ppp", cleaned)

    def test_sanitize_strips_internal_fen_and_zwischenzug(self) -> None:
        raw = "Nice idea. (Internal board FEN remains unchanged) A zwischenzug helps."
        cleaned = sanitize_explanation(raw)
        self.assertNotIn("Internal board", cleaned)
        self.assertNotIn("zwischenzug", cleaned.lower())
        self.assertIn("in-between idea", cleaned)

    def test_finalize_forces_opener_and_drops_fake_fork(self) -> None:
        ground = (
            "GROUND TRUTH (never contradict): White moved pawn e2→e4 (SAN e4, UCI e2e4), "
            "no capture. After the move, that piece attacks no enemy piece."
        )
        raw = (
            "You played Qh4+. Watch for a knight fork on f2 putting your king in danger."
        )
        out = finalize_explanation(
            raw,
            move_san="e4",
            last_mover="white",
            human_color="white",
            ground_summary=ground,
        )
        self.assertTrue(out.startswith("You played e4."))
        self.assertNotIn("knight fork", out.lower())
        self.assertNotIn("Qh4", out)

    def test_heuristic_still_works(self) -> None:
        text = HeuristicProvider().explain(
            fen=self.FEN,
            color="white",
            move_uci="e2e4",
            move_san="e4",
            move_history=[],
            game_type="chess",
            skill_level="advanced",
        )
        self.assertTrue(str(text).strip())


if __name__ == "__main__":
    unittest.main()
