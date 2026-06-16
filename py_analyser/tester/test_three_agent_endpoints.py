#!/usr/bin/env python3
"""Verification tests for /history, /policy, and /value endpoints."""

from __future__ import annotations

import os
import sys
import unittest


CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

import server  # noqa: E402


class TestThreeAgentEndpoints(unittest.TestCase):
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()
        self.base_payload = {
            "request_id": "three-agent-test",
            "game_id": "game-three-agent",
            "game_type": "chess",
            "variant": "chess",
            "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
            "color": "white",
            "move_number": 1,
            "move_history": [],
        }

    def test_history_endpoint_returns_expected_shape(self) -> None:
        response = self.client.post("/history", json=self.base_payload)
        self.assertEqual(response.status_code, 200)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "ok")
        self.assertEqual(payload.get("request_id"), "three-agent-test")
        self.assertIn(payload.get("phase"), {"opening", "middlegame", "endgame"})
        self.assertIsInstance(payload.get("features"), dict)
        self.assertIsInstance(payload.get("tags"), list)
        self.assertIsInstance(payload.get("latency_ms"), int)

    def test_policy_endpoint_returns_candidates(self) -> None:
        req = dict(self.base_payload)
        req["top_k"] = 3
        response = self.client.post("/policy", json=req)
        self.assertEqual(response.status_code, 200)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "ok")
        self.assertEqual(payload.get("request_id"), "three-agent-test")
        self.assertIsInstance(payload.get("candidates"), list)
        self.assertGreaterEqual(len(payload["candidates"]), 1)
        first = payload["candidates"][0]
        self.assertIn("uci", first)
        self.assertIn("san", first)
        self.assertIn("score_cp", first)
        self.assertIn("prob", first)

    def test_value_endpoint_returns_score_fields(self) -> None:
        response = self.client.post("/value", json=self.base_payload)
        self.assertEqual(response.status_code, 200)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "ok")
        self.assertEqual(payload.get("request_id"), "three-agent-test")
        self.assertIn("score_cp", payload)
        self.assertIn("mate_in", payload)
        self.assertIn("value", payload)
        self.assertIn("win_chance_white", payload)
        self.assertIn("win_chance_black", payload)

    def test_common_validation_rejects_missing_game_type(self) -> None:
        req = dict(self.base_payload)
        req.pop("game_type")
        response = self.client.post("/history", json=req)
        self.assertEqual(response.status_code, 400)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "error")
        self.assertEqual(payload.get("error_kind"), "validation")

    def test_common_validation_rejects_unsupported_game_type(self) -> None:
        req = dict(self.base_payload)
        req["game_type"] = "xiangqi"
        req["variant"] = "xiangqi"
        response = self.client.post("/policy", json=req)
        self.assertEqual(response.status_code, 400)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "error")
        self.assertEqual(payload.get("error_kind"), "validation")


if __name__ == "__main__":
    unittest.main()
