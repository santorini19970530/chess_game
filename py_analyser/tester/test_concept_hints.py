#!/usr/bin/env python3
# test_concept_hints.py - checks tiny concept hints from analyze-shaped payloads

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
    # test_quiet_pawn_push_attacks_nothing - checks e2e4 ground truth reports quiet push with no attacks
    def test_quiet_pawn_push_attacks_nothing(self) -> None:
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        gt = build_move_ground_truth(fen, "e2e4", ["e2e4"], "chess")
        summary = gt["summary"]
        self.assertIn("GROUND TRUTH", summary)
        self.assertIn("SAN e4", summary)
        self.assertIn("no capture", summary)
        self.assertIn("attacks no enemy piece", summary)
        self.assertEqual(gt.get("san"), "e4")

    # test_labeled_session_history_replays - checks White: e2e4 history labels still ground-truth
    def test_labeled_session_history_replays(self) -> None:
        fen = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
        gt = build_move_ground_truth(fen, "e2e4", ["White: e2e4"], "chess")
        self.assertEqual(gt.get("san"), "e4")
        self.assertIn("pawn e2→e4", gt["summary"])
        self.assertIn("attacks no enemy piece", gt["summary"])

    # test_shogi_labels_piece_from_post_move_fen - checks shogi san uses piece on post-move fen
    def test_shogi_labels_piece_from_post_move_fen(self) -> None:
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B3S1R1/LNSGKG1NL[] b - - 0 1"
        gt = build_move_ground_truth(fen, "g1f2", ["g1f2"], "shogi")
        self.assertEqual(gt.get("san"), "silver g1→f2")
        self.assertIn("silver g1→f2", gt["summary"])
        self.assertNotEqual(gt.get("san"), "g1f2")

    # test_shogi_drop_label - checks shogi drop uci becomes a drop pawn label
    def test_shogi_drop_label(self) -> None:
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[P] w - - 0 1"
        gt = build_move_ground_truth(fen, "P*e5", ["P*e5"], "shogi")
        self.assertEqual(gt.get("san"), "drop pawn → e5")

    # test_does_not_claim_far_bishop_attack_on_f3 - checks quiet f2f3 does not invent bishop attacks
    def test_does_not_claim_far_bishop_attack_on_f3(self) -> None:
        hist = ["e2e4", "e7e5", "f2f3"]
        board_fen = "rnbqkbnr/pppp1ppp/8/4p3/4P3/5P2/PPPP2PP/RNBQKBNR b KQkq - 0 2"
        gt = build_move_ground_truth(board_fen, "f2f3", hist, "chess")
        self.assertIn("pawn f2→f3", gt["summary"])
        self.assertIn("no capture", gt["summary"])
        self.assertIn("attacks no enemy piece", gt["summary"])
        self.assertNotIn("bishop", gt["summary"].lower())

    # test_tip_fen_san_from_after_position - checks diagram tip fen yields san when history cannot replay from start
    def test_tip_fen_san_from_after_position(self) -> None:
        after = "r4r1k/p1qb1Bpp/1ppb4/5pBQ/3P4/8/P1P2PPP/1R2R1K1 b - - 1 1"
        gt = build_move_ground_truth(after, "c4f7", ["c4f7"], "chess")
        self.assertIn(gt.get("san"), {"Bf7", "Bxf7"})
        self.assertIn("bishop c4→f7", gt["summary"])
        self.assertIn("SAN", gt["summary"])


class TestConceptHints(unittest.TestCase):
    # test_empty_analysis - checks empty analyze payloads yield no hints
    def test_empty_analysis(self) -> None:
        self.assertEqual(build_concept_hints(None), [])
        self.assertEqual(build_concept_hints({}), [])

    # test_ordered_tiny_set - checks threat, material, and reply cues stay ordered and capped
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

    # test_skips_balanced_material - checks near-zero eval does not add a material cue
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

    # test_max_hints_cap - checks max_hints truncates the cue list
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
    # setUp - configures the flask test client and heuristic provider
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()
        os.environ["LLM_PROVIDER"] = "heuristic"

    # test_explain_without_hints_still_ok - checks /explain works with an empty concept_hints list
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

    # test_explain_accepts_concept_hints_list - checks blanks are dropped and hints are capped at three
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

    # test_explain_builds_hints_from_analysis_object - checks analysis payloads become concept_hints
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
