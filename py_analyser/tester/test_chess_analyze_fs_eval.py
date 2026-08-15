#!/usr/bin/env python3
# test_chess_analyze_fs_eval.py - checks chess /analyze uses one fairy-stockfish multipv for eval + suggestions

from __future__ import annotations

import os
import sys
import unittest
from unittest import mock

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

import analyzer  # noqa: E402
from analyzer import MoveSuggestion  # noqa: E402
from move_suggest import FairyStockfishSuggest  # noqa: E402


FEN_START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
FEN_BLACK = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"


class TestChessAnalyzeFsEval(unittest.TestCase):
    # test_analyze_uses_fs_multipv_score_as_eval_cp_white - mocked top pv score becomes eval + win%
    def test_analyze_uses_fs_multipv_score_as_eval_cp_white(self) -> None:
        fake = [MoveSuggestion(rank=1, uci="e2e4", san="e4", score=42)]
        with mock.patch.object(
            FairyStockfishSuggest,
            "suggest_with_eval",
            return_value=(fake, 42),
        ) as mocked:
            result = analyzer.analyze_position(
                fen=FEN_START,
                color="white",
                top_k=3,
                request_id="fs-eval-1",
                game_type="chess",
            )
        mocked.assert_called_once()
        self.assertEqual(result["eval_cp_white"], 42)
        self.assertEqual(result["win_chance_white"], round(analyzer.cp_to_win_chance(42), 4))
        self.assertEqual(result["win_chance_black"], round(1.0 - analyzer.cp_to_win_chance(42), 4))
        self.assertEqual(result["best_move_uci"], "e2e4")
        self.assertEqual(result["suggested_moves"][0]["uci"], "e2e4")
        self.assertEqual(result["source"], "fairy-stockfish")
        self.assertEqual(result["evaluation_source"], "fairy-stockfish")

    # test_analyze_fs_failure_falls_back_to_heuristic - engine miss keeps playable heuristic payload
    def test_analyze_fs_failure_falls_back_to_heuristic(self) -> None:
        with mock.patch.object(
            FairyStockfishSuggest,
            "suggest_with_eval",
            side_effect=RuntimeError("engine down"),
        ):
            result = analyzer.analyze_position(
                fen=FEN_START,
                color="white",
                top_k=3,
                request_id="fs-eval-fallback",
                game_type="chess",
            )
        self.assertEqual(result["source"], "heuristic")
        self.assertEqual(result["evaluation_source"], "heuristic-fallback")
        self.assertIn("eval_cp_white", result)
        self.assertGreaterEqual(len(result["suggested_moves"]), 1)

    # test_analyze_black_to_move_keeps_white_pov_score - black stm still stores white-perspective cp
    def test_analyze_black_to_move_keeps_white_pov_score(self) -> None:
        fake = [MoveSuggestion(rank=1, uci="e7e5", san="e5", score=-15)]
        with mock.patch.object(
            FairyStockfishSuggest,
            "suggest_with_eval",
            return_value=(fake, -15),
        ):
            result = analyzer.analyze_position(
                fen=FEN_BLACK,
                color="black",
                top_k=1,
                request_id="fs-eval-black",
                game_type="chess",
            )
        self.assertEqual(result["eval_cp_white"], -15)
        self.assertEqual(result["win_chance_white"], round(analyzer.cp_to_win_chance(-15), 4))


if __name__ == "__main__":
    unittest.main()
