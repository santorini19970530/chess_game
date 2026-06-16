#!/usr/bin/env python3
"""API verification tests for the Python analyzer service."""

from __future__ import annotations

import os
import sys
import unittest


CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

import server  # noqa: E402


class TestAnalyzerServiceAPI(unittest.TestCase):
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()

    def test_health_returns_200_and_json_shape(self) -> None:
        response = self.client.get("/health")
        self.assertEqual(response.status_code, 200)
        payload = response.get_json()
        self.assertIsInstance(payload, dict)
        self.assertEqual(payload.get("status"), "ok")
        self.assertEqual(payload.get("service"), "py_analyser")
        self.assertIsInstance(payload.get("timestamp"), str)
        self.assertTrue(payload["timestamp"].strip())

    def test_analyze_returns_required_schema(self) -> None:
        response = self.client.post(
            "/analyze",
            json={
                "request_id": "analyzer-service-test",
                "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
                "color": "white",
                "top_k": 3,
            },
        )
        self.assertEqual(response.status_code, 200)
        payload = response.get_json()
        self.assertIsInstance(payload, dict)

        required_top_level_fields = {
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
        self.assertTrue(required_top_level_fields.issubset(set(payload.keys())))
        self.assertEqual(payload["status"], "ok")
        self.assertEqual(payload["request_id"], "analyzer-service-test")
        self.assertEqual(payload["evaluated_for_color"], "white")
        self.assertIsInstance(payload["health_summary"], dict)
        self.assertIsInstance(payload["suggested_moves"], list)
        self.assertGreaterEqual(len(payload["suggested_moves"]), 1)

    def test_analyze_rejects_missing_fen(self) -> None:
        response = self.client.post("/analyze", json={"color": "white"})
        self.assertEqual(response.status_code, 400)
        payload = response.get_json()
        self.assertIn("error", payload)


if __name__ == "__main__":
    unittest.main()
