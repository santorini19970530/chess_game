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
