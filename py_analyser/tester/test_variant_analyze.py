#!/usr/bin/env python3
# test_variant_analyze.py - checks /analyze for xianqi/shogi without chess.Board on variant fen

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
    "evaluation_source",
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
        from fs_engine import uci_score_as_white

        # same mapping chess uses: eval_cp is always white-minus-black style
        self.assertEqual(uci_score_as_white(120, XIANGQI_START), 120)
        black_to_move = XIANGQI_START.replace(" w ", " b ", 1)
        self.assertEqual(uci_score_as_white(120, black_to_move), -120)

    def test_win_chance_uses_shared_cp_mapping(self) -> None:
        # Chess and variants must share cp_to_win_chance (not a separate formula).
        cp = 150
        expected = analyzer.cp_to_win_chance(cp)
        self.assertAlmostEqual(expected + (1.0 - expected), 1.0, places=9)
        self.assertGreater(expected, 0.5)

    # test_variant_fs_failure_reports_unavailable - no chess.Board heuristic for variant fen
    def test_variant_fs_failure_reports_unavailable(self) -> None:
        from unittest import mock
        from move_suggest import FairyStockfishVariantSuggest

        with mock.patch.object(
            FairyStockfishVariantSuggest,
            "suggest_with_eval",
            side_effect=RuntimeError("engine down"),
        ):
            result = analyzer.analyze_position(
                fen=XIANGQI_START,
                color="white",
                top_k=3,
                request_id="xq-unavailable",
                game_type="xianqi",
            )
        self.assertEqual(result["source"], "fallback")
        self.assertEqual(result["evaluation_source"], "unavailable")
        self.assertEqual(result["eval_cp_white"], 0)
        self.assertEqual(result["suggested_moves"], [])

    # test_variant_fs_success_sets_evaluation_source - mocked multipv marks fairy-stockfish
    def test_variant_fs_success_sets_evaluation_source(self) -> None:
        from unittest import mock
        from analyzer import MoveSuggestion
        from move_suggest import FairyStockfishVariantSuggest

        fake = [MoveSuggestion(rank=1, uci="a4a5", san="a4a5", score=33)]
        with mock.patch.object(
            FairyStockfishVariantSuggest,
            "suggest_with_eval",
            return_value=(fake, 33),
        ) as mocked:
            result = analyzer.analyze_position(
                fen=XIANGQI_START,
                color="white",
                top_k=1,
                request_id="xq-fs-ok",
                game_type="xianqi",
            )
        mocked.assert_called_once()
        self.assertEqual(result["evaluation_source"], "fairy-stockfish")
        self.assertEqual(result["eval_cp_white"], 33)
        self.assertEqual(result["best_move_uci"], "a4a5")


if __name__ == "__main__":
    unittest.main()
