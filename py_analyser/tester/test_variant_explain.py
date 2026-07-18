#!/usr/bin/env python3
"""issue0049 step 2: /explain for xianqi / shogi (no chess.Board on variant FEN)."""

from __future__ import annotations

import os
import sys
import unittest

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

# Force offline path so tests do not need Ollama.
os.environ["LLM_PROVIDER"] = "heuristic"

import analyzer  # noqa: E402
import server  # noqa: E402

XIANGQI_START = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
SHOGI_START = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"


class TestVariantExplain(unittest.TestCase):
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()
        os.environ["LLM_PROVIDER"] = "heuristic"

    def test_fallback_xianqi_does_not_use_chess_board(self) -> None:
        text = analyzer.build_explanation_fallback(
            fen=XIANGQI_START,
            color="white",
            move_uci="h2e2",
            move_san=None,
            game_type="xianqi",
        )
        self.assertIsInstance(text, str)
        self.assertTrue(text.strip())
        self.assertIn("xiangqi", text.lower())

    def test_explain_http_xianqi_ok(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "xq-explain",
                "fen": XIANGQI_START,
                "color": "white",
                "game_type": "xianqi",
                "move_uci": "h2e2",
                "move_number": 1,
                "move_history": [],
            },
        )
        self.assertEqual(response.status_code, 200, response.get_json())
        payload = response.get_json()
        self.assertEqual(payload["status"], "ok")
        self.assertTrue(str(payload.get("explanation", "")).strip())
        self.assertNotIn("expected 8 rows", str(payload).lower())

    def test_explain_http_shogi_ok(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "sh-explain",
                "fen": SHOGI_START,
                "color": "white",
                "game_type": "shogi",
                "move_uci": "c3c4",
                "move_number": 1,
                "move_history": [],
            },
        )
        self.assertEqual(response.status_code, 200, response.get_json())
        payload = response.get_json()
        self.assertEqual(payload["status"], "ok")
        self.assertTrue(str(payload.get("explanation", "")).strip())

    def test_explain_chess_still_works(self) -> None:
        response = self.client.post(
            "/explain",
            json={
                "request_id": "chess-explain",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "game_type": "chess",
                "move_uci": "e2e4",
                "move_number": 1,
                "move_history": [],
            },
        )
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.get_json()["status"], "ok")

    def test_history_still_rejects_xianqi(self) -> None:
        # HPV stay chess-only; coach pipe is separate.
        response = self.client.post(
            "/history",
            json={
                "request_id": "hpv-xq",
                "fen": XIANGQI_START,
                "color": "white",
                "game_type": "xianqi",
                "move_number": 1,
                "move_history": [],
            },
        )
        self.assertEqual(response.status_code, 400)


if __name__ == "__main__":
    unittest.main()
