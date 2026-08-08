#!/usr/bin/env python3
# test_move_suggest_strategy.py - characterizes MoveSuggestStrategy FS → Heuristic chain

from __future__ import annotations

import os
import sys
import unittest
from unittest import mock


CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from move_suggest import (  # noqa: E402
    FairyStockfishSuggest,
    HeuristicSuggest,
    MoveSuggestContext,
    select_suggestions,
)


FEN_START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"


class TestMoveSuggestStrategy(unittest.TestCase):
    # test_strategy_names - checks concrete strategy name constants
    def test_strategy_names(self) -> None:
        self.assertEqual(FairyStockfishSuggest.name, "fairy_stockfish")
        self.assertEqual(HeuristicSuggest.name, "heuristic")

    # test_heuristic_returns_ranked_moves - checks heuristic suggest returns ranked uci
    def test_heuristic_returns_ranked_moves(self) -> None:
        out = HeuristicSuggest().suggest(
            MoveSuggestContext(fen=FEN_START, color="white", top_k=3)
        )
        self.assertGreaterEqual(len(out), 1)
        self.assertEqual(out[0].rank, 1)
        self.assertTrue(out[0].uci)

    # test_select_falls_back_to_heuristic_when_fs_soft_misses - checks FS empty triggers heuristic
    def test_select_falls_back_to_heuristic_when_fs_soft_misses(self) -> None:
        ctx = MoveSuggestContext(fen=FEN_START, color="white", top_k=3, profile="intermediate")
        with mock.patch.object(FairyStockfishSuggest, "suggest", return_value=[]):
            suggestions, name = select_suggestions(ctx)
        self.assertEqual(name, "heuristic")
        self.assertGreaterEqual(len(suggestions), 1)

    # test_select_uses_fs_when_it_returns_moves - checks FS hit skips heuristic
    def test_select_uses_fs_when_it_returns_moves(self) -> None:
        from analyzer import MoveSuggestion

        fake = [MoveSuggestion(rank=1, uci="e2e4", san="e4", score=20)]
        ctx = MoveSuggestContext(fen=FEN_START, color="white", top_k=1)
        with mock.patch.object(FairyStockfishSuggest, "suggest", return_value=fake):
            suggestions, name = select_suggestions(ctx)
        self.assertEqual(name, "fairy_stockfish")
        self.assertEqual(suggestions[0].uci, "e2e4")


if __name__ == "__main__":
    unittest.main()
