#!/usr/bin/env python3
"""issue0047 step 1: tiny concept hints from /analyze-shaped payloads."""

from __future__ import annotations

import os
import sys
import unittest

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

os.environ["LLM_PROVIDER"] = "heuristic"

from analyzer import build_concept_hints  # noqa: E402
import server  # noqa: E402


class TestConceptHints(unittest.TestCase):
    def test_empty_analysis(self) -> None:
        self.assertEqual(build_concept_hints(None), [])
        self.assertEqual(build_concept_hints({}), [])

    def test_ordered_tiny_set(self) -> None:
        hints = build_concept_hints(
            {
                "threat_summary": "Black king is in check.",
                "health_summary": {"material_balance_white_minus_black": 350},
                "suggested_moves": [{"uci": "e2e4", "san": "e4"}],
            }
        )
        self.assertEqual(len(hints), 3)
        self.assertEqual(hints[0], "Black king is in check.")
        self.assertIn("White is ahead", hints[1])
        self.assertEqual(hints[2], "Top suggestion for the side to move: e4.")

    def test_skips_balanced_material(self) -> None:
        hints = build_concept_hints(
            {
                "threat_summary": "Position is roughly balanced.",
                "eval_cp_white": 20,
                "best_move_uci": "g1f3",
            }
        )
        self.assertEqual(
            hints,
            [
                "Position is roughly balanced.",
                "Top suggestion for the side to move: g1f3.",
            ],
        )

    def test_max_hints_cap(self) -> None:
        hints = build_concept_hints(
            {
                "threat_summary": "White has the initiative.",
                "eval_cp_white": -200,
                "suggested_moves": [{"uci": "e7e5", "san": "e5"}],
            },
            max_hints=2,
        )
        self.assertEqual(len(hints), 2)


class TestExplainConceptHintsContract(unittest.TestCase):
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()
        os.environ["LLM_PROVIDER"] = "heuristic"

    def test_explain_without_hints_still_ok(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "hints-0",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "game_type": "chess",
                "move_uci": "e2e4",
            },
        )
        self.assertEqual(response.status_code, 200, response.get_json())
        body = response.get_json()
        self.assertEqual(body["status"], "ok")
        self.assertEqual(body.get("concept_hints"), [])

    def test_explain_accepts_concept_hints_list(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "hints-1",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "game_type": "chess",
                "move_uci": "e2e4",
                "concept_hints": [
                    "Black king is in check.",
                    "",
                    "Top suggestion for the side to move: e4.",
                    "extra ignored",
                    "also ignored",
                ],
            },
        )
        self.assertEqual(response.status_code, 200)
        hints = response.get_json()["concept_hints"]
        self.assertEqual(
            hints,
            [
                "Black king is in check.",
                "Top suggestion for the side to move: e4.",
                "extra ignored",
            ],
        )

    def test_explain_builds_hints_from_analysis_object(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "hints-2",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "game_type": "chess",
                "move_uci": "e2e4",
                "analysis": {
                    "threat_summary": "White has the initiative.",
                    "best_move_uci": "d2d4",
                },
            },
        )
        self.assertEqual(response.status_code, 200)
        hints = response.get_json()["concept_hints"]
        self.assertIn("White has the initiative.", hints)
        self.assertIn("Top suggestion for the side to move: d2d4.", hints)


if __name__ == "__main__":
    unittest.main()
