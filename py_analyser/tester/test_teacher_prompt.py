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

from llm_providers import HeuristicProvider, build_teacher_prompt, normalize_skill_level  # noqa: E402


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
        self.assertIn("Black just played", prompt)
        self.assertIn("White is to move", prompt)
        self.assertIn("human plays White", prompt)
        self.assertIn("advise YOU (White) only", prompt)
        self.assertIn("at most 2 short sentences", prompt)
        self.assertIn("under 45 words", prompt)

    def test_prompt_injects_concept_hints(self) -> None:
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        with_cues = build_teacher_prompt(
            fen=fen,
            move_uci="e2e4",
            move_san="e4",
            move_history=["e2e4"],
            skill_level="intermediate",
            concept_hints=["Black king is in check.", "Top suggestion for the side to move: e7e5."],
        )
        bare = build_teacher_prompt(
            fen=fen,
            move_uci="e2e4",
            move_san="e4",
            move_history=["e2e4"],
            skill_level="intermediate",
        )
        self.assertIn("Position cues", with_cues)
        self.assertIn("Black king is in check.", with_cues)
        self.assertIn("e7e5", with_cues)
        self.assertNotIn("Position cues", bare)

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
