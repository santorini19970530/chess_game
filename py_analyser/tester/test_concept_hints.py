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

from analyzer import build_concept_hints, build_move_ground_truth  # noqa: E402
import server  # noqa: E402


class TestMoveGroundTruth(unittest.TestCase):
    def test_quiet_pawn_push_attacks_nothing(self) -> None:
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        gt = build_move_ground_truth(fen, "e2e4", ["e2e4"], "chess")
        summary = gt["summary"]
        self.assertIn("GROUND TRUTH", summary)
        self.assertIn("SAN e4", summary)
        self.assertIn("no capture", summary)
        self.assertIn("attacks no enemy piece", summary)
        self.assertEqual(gt.get("san"), "e4")

    def test_labeled_session_history_replays(self) -> None:
        # Go MoveHistory uses "White: e2e4" labels — must still ground-truth.
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        gt = build_move_ground_truth(fen, "e2e4", ["White: e2e4"], "chess")
        self.assertEqual(gt.get("san"), "e4")
        self.assertIn("pawn e2→e4", gt["summary"])
        self.assertIn("attacks no enemy piece", gt["summary"])

    def test_shogi_labels_piece_from_post_move_fen(self) -> None:
        # After sente silver g1→f2 (piece sits on f2 in this FEN).
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B3S1R1/LNSGKG1NL[] b - - 0 1"
        gt = build_move_ground_truth(fen, "g1f2", ["g1f2"], "shogi")
        self.assertEqual(gt.get("san"), "silver g1→f2")
        self.assertIn("silver g1→f2", gt["summary"])
        self.assertNotEqual(gt.get("san"), "g1f2")

    def test_shogi_drop_label(self) -> None:
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[P] w - - 0 1"
        gt = build_move_ground_truth(fen, "P*e5", ["P*e5"], "shogi")
        self.assertEqual(gt.get("san"), "drop pawn → e5")

    def test_does_not_claim_far_bishop_attack_on_f3(self) -> None:
        # Opening-ish: after 1.e4 e5 2.Nf3 Nc6 3.Bb5 a6 4.Ba4 Nf6 5.O-O Be7 6.Re1 b5 7.Bb3 d6 8.c3 O-O 9.h3 Nb8 …
        # Simpler: white plays f3 in a position with black bishop nowhere attacked by that pawn.
        hist = ["e2e4", "e7e5", "f2f3"]
        board_fen = "rnbqkbnr/pppp1ppp/8/4p3/4P3/5P2/PPPP2PP/RNBQKBNR b KQkq - 0 2"
        gt = build_move_ground_truth(board_fen, "f2f3", hist, "chess")
        self.assertIn("pawn f2→f3", gt["summary"])
        self.assertIn("no capture", gt["summary"])
        self.assertIn("attacks no enemy piece", gt["summary"])
        self.assertNotIn("bishop", gt["summary"].lower())


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
        self.assertEqual(hints[2], "Engine suggested replies (side to move): e4.")

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
                "Engine suggested replies (side to move): g1f3.",
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
        self.assertIn("Engine suggested replies (side to move): d2d4.", hints)


if __name__ == "__main__":
    unittest.main()
