#!/usr/bin/env python3
# test_shogi_hand_infer.py - checks board-inventory hand inference for diagram import

from __future__ import annotations

import os
import sys
import unittest

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from shogi_hand_infer import apply_inferred_shogi_hands, split_shogi_placement_and_hands


class TestShogiHandInfer(unittest.TestCase):
    # test_start_position_keeps_empty_hands - checks full start board yields empty hand bracket
    def test_start_position_keeps_empty_hands(self) -> None:
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"
        out = apply_inferred_shogi_hands(fen)
        placement, hands = split_shogi_placement_and_hands(out.split()[0])
        self.assertEqual(placement.count("/"), 8)
        self.assertEqual(hands, "")

    # test_missing_black_pawn_goes_to_white_hand - checks one off-board black pawn becomes white hand P
    def test_missing_black_pawn_goes_to_white_hand(self) -> None:
        fen = "lnsgkgsnl/1r5b1/pppp1pppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"
        out = apply_inferred_shogi_hands(fen)
        _placement, hands = split_shogi_placement_and_hands(out.split()[0])
        self.assertEqual(hands, "P")

    # test_missing_white_gold_goes_to_black_hand - checks one off-board white gold becomes black hand g
    def test_missing_white_gold_goes_to_black_hand(self) -> None:
        fen = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSK1GSNL w - - 0 1"
        out = apply_inferred_shogi_hands(fen)
        _placement, hands = split_shogi_placement_and_hands(out.split()[0])
        self.assertEqual(hands, "g")

    # test_promoted_piece_counts_as_base_type - checks +p counts as pawn so full pawn inventory stays empty-handed
    def test_promoted_piece_counts_as_base_type(self) -> None:
        fen = "lnsgkgsnl/1r5b1/pppppppp+p/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"
        out = apply_inferred_shogi_hands(fen)
        _placement, hands = split_shogi_placement_and_hands(out.split()[0])
        self.assertEqual(hands, "")


if __name__ == "__main__":
    unittest.main()
