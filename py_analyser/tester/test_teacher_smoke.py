#!/usr/bin/env python3
# test_teacher_smoke.py - offline teacher smoke; set TEACHER_SMOKE_OLLAMA=1 for live ollama

from __future__ import annotations

import json
import os
import sys
import unittest
from pathlib import Path

CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

DATA_DIR = Path(PARENT_DIR) / "data"
FEN_AFTER_E4 = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"


_TERMS_TONE_FILES = (
    "chess_terms.json",
    "chess_tone.json",
    "xiangqi_terms.json",
    "xiangqi_tone.json",
    "shogi_terms.json",
    "shogi_tone.json",
)


class TestTeacherSmokeOffline(unittest.TestCase):
    def test_terms_and_tone_json_load(self) -> None:
        for name in _TERMS_TONE_FILES:
            path = DATA_DIR / name
            self.assertTrue(path.is_file(), f"missing {path}")
            data = json.loads(path.read_text(encoding="utf-8"))
            self.assertIsInstance(data, dict)
            for level in ("beginner", "intermediate", "advanced"):
                self.assertIn(level, data, f"{name} missing {level}")

    def test_prompts_differ_beginner_vs_advanced(self) -> None:
        from teacher_prompt import build_teacher_prompt

        kwargs = dict(
            fen=FEN_AFTER_E4,
            move_uci="e2e4",
            move_san="e4",
            move_history=["White: e2e4"],
            human_color="white",
        )
        beg = build_teacher_prompt(**kwargs, skill_level="beginner")
        adv = build_teacher_prompt(**kwargs, skill_level="advanced")
        self.assertNotEqual(beg, adv)
        self.assertIn("friendly teacher", beg)
        self.assertIn("prophylaxis", adv)

    def test_variant_prompts_use_own_terms(self) -> None:
        from teacher_prompt import build_teacher_prompt

        kwargs = dict(
            fen=FEN_AFTER_E4,
            move_uci="e2e4",
            move_san="e4",
            move_history=[],
            skill_level="intermediate",
        )
        chess = build_teacher_prompt(**kwargs, game_type="chess")
        xq = build_teacher_prompt(**kwargs, game_type="xianqi")
        sh = build_teacher_prompt(**kwargs, game_type="shogi")
        self.assertIn("palace", xq)
        self.assertIn("drop", sh)
        self.assertNotIn("palace", chess)
        # shared anti-hallucination text may say "drops"; chess Ok-terms must not list drop
        self.assertRegex(chess, r"Ok terms:.*\bdevelopment\b")
        self.assertNotRegex(chess, r"Ok terms:.*\bdrop\b")

    def test_explain_heuristic_nonempty(self) -> None:
        os.environ["LLM_PROVIDER"] = "heuristic"
        import server

        server.app.config["TESTING"] = True
        client = server.app.test_client()
        resp = client.post(
            "/explain",
            json={
                "request_id": "smoke-heuristic",
                "fen": FEN_AFTER_E4,
                "color": "black",
                "game_type": "chess",
                "skill_level": "intermediate",
                "move_uci": "e2e4",
                "move_history": ["White: e2e4"],
                "human_color": "white",
            },
        )
        self.assertEqual(resp.status_code, 200, resp.get_json())
        body = resp.get_json()
        self.assertEqual(body.get("status"), "ok")
        self.assertTrue(str(body.get("explanation") or "").strip())
        self.assertEqual(body.get("move_san"), "e4")
        self.assertTrue(str(body.get("explanation")).startswith("You played e4."))

    def test_quick_explain_is_instant_ground_truth(self) -> None:
        os.environ["LLM_PROVIDER"] = "heuristic"
        import server

        server.app.config["TESTING"] = True
        client = server.app.test_client()
        resp = client.post(
            "/explain",
            json={
                "request_id": "smoke-quick",
                "fen": FEN_AFTER_E4,
                "color": "black",
                "game_type": "chess",
                "move_uci": "e2e4",
                "move_history": ["White: e2e4"],
                "human_color": "white",
                "quick": True,
            },
        )
        self.assertEqual(resp.status_code, 200, resp.get_json())
        body = resp.get_json()
        self.assertEqual(body.get("source"), "quick")
        self.assertTrue(str(body.get("explanation")).startswith("You played e4."))
        self.assertIn("latency_ms", body)
        self.assertLess(int(body["latency_ms"]), 500)


@unittest.skipUnless(
    os.getenv("TEACHER_SMOKE_OLLAMA", "").strip() == "1",
    "set TEACHER_SMOKE_OLLAMA=1 with Ollama running to enable live smoke",
)
class TestTeacherSmokeOllama(unittest.TestCase):
    def test_beginner_and_advanced_ollama(self) -> None:
        os.environ["LLM_PROVIDER"] = "ollama"
        import importlib
        import llm_providers
        import server

        importlib.reload(llm_providers)
        importlib.reload(server)
        server.app.config["TESTING"] = True
        client = server.app.test_client()

        payloads = {}
        for level in ("beginner", "advanced"):
            resp = client.post(
                "/explain",
                json={
                    "request_id": f"smoke-ollama-{level}",
                    "fen": FEN_AFTER_E4,
                    "color": "black",
                    "game_type": "chess",
                    "skill_level": level,
                    "move_uci": "e2e4",
                    "move_history": ["White: e2e4"],
                    "human_color": "white",
                    "concept_hints": [
                        "Position is roughly balanced.",
                        "Engine suggested replies (side to move): e5, c5, Nf6.",
                    ],
                },
            )
            self.assertEqual(resp.status_code, 200, resp.get_json())
            body = resp.get_json()
            self.assertEqual(body.get("status"), "ok")
            self.assertEqual(body.get("skill_level"), level)
            self.assertEqual(body.get("source"), "ollama", body)
            text = str(body.get("explanation") or "").strip()
            self.assertTrue(text)
            self.assertTrue(text.startswith("You played e4."), text)
            self.assertEqual(body.get("move_san"), "e4")
            payloads[level] = text

        # Soft: levels may produce same short line; still require both succeeded with ollama.
        self.assertTrue(payloads["beginner"])
        self.assertTrue(payloads["advanced"])


if __name__ == "__main__":
    unittest.main()
