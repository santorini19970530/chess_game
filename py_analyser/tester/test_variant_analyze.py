#!/usr/bin/env python3
"""issue0049 step 1: /analyze for xianqi / shogi (no chess.Board on variant FEN)."""

from __future__ import annotations

import os
import sys
import unittest

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

import analyzer  # noqa: E402
import server  # noqa: E402

XIANGQI_START = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
SHOGI_START = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"

REQUIRED_FIELDS = {
    "request_id",
    "status",
    "source",
    "fen",
    "evaluated_for_color",
    "health_summary",
    "eval_cp_white",
    "win_chance_white",
    "win_chance_black",
    "best_move_uci",
    "suggested_moves",
    "latency_ms",
}


class TestVariantAnalyze(unittest.TestCase):
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()

    def test_analyze_position_xianqi_does_not_use_chess_board(self) -> None:
        # Without game_type wiring this raises "expected 8 rows" from chess.Board.
        result = analyzer.analyze_position(
            fen=XIANGQI_START,
            color="white",
            top_k=3,
            request_id="xq-unit",
            game_type="xianqi",
        )
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["request_id"], "xq-unit")
        self.assertTrue(REQUIRED_FIELDS.issubset(result.keys()))
        self.assertIsInstance(result["suggested_moves"], list)

    def test_analyze_http_xianqi_returns_schema(self) -> None:
        response = self.client.post(
            "/analyze",
            json={
                "request_id": "xq-http",
                "fen": XIANGQI_START,
                "color": "white",
                "top_k": 3,
                "game_type": "xianqi",
            },
        )
        self.assertEqual(response.status_code, 200, response.get_json())
        payload = response.get_json()
        self.assertEqual(payload["status"], "ok")
        self.assertTrue(REQUIRED_FIELDS.issubset(payload.keys()))
        self.assertNotIn("expected 8 rows", str(payload).lower())

    def test_analyze_http_shogi_returns_schema(self) -> None:
        response = self.client.post(
            "/analyze",
            json={
                "request_id": "sh-http",
                "fen": SHOGI_START,
                "color": "white",
                "top_k": 3,
                "game_type": "shogi",
            },
        )
        self.assertEqual(response.status_code, 200, response.get_json())
        payload = response.get_json()
        self.assertEqual(payload["status"], "ok")
        self.assertTrue(REQUIRED_FIELDS.issubset(payload.keys()))

    def test_analyze_chess_unchanged_without_game_type(self) -> None:
        response = self.client.post(
            "/analyze",
            json={
                "request_id": "chess-default",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "top_k": 3,
            },
        )
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.get_json()["status"], "ok")

    def test_uci_score_as_white_matches_chess_white_perspective(self) -> None:
        # Same mapping chess uses: eval_cp is always White-minus-Black style.
        self.assertEqual(analyzer.uci_score_as_white(120, XIANGQI_START), 120)
        black_to_move = XIANGQI_START.replace(" w ", " b ", 1)
        self.assertEqual(analyzer.uci_score_as_white(120, black_to_move), -120)

    def test_win_chance_uses_shared_cp_mapping(self) -> None:
        # Chess and variants must share cp_to_win_chance (not a separate formula).
        cp = 150
        expected = analyzer.cp_to_win_chance(cp)
        self.assertAlmostEqual(expected + (1.0 - expected), 1.0, places=9)
        self.assertGreater(expected, 0.5)


if __name__ == "__main__":
    unittest.main()
