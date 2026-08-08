#!/usr/bin/env python3
# test_agent_strategy.py - characterizes AgentStrategy history / policy / value registry

from __future__ import annotations

import os
import sys
import unittest


CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

from agents import (  # noqa: E402
    AGENTS,
    AgentContext,
    HistoryAgent,
    PolicyAgent,
    ValueAgent,
    run_agent,
)


FEN_START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"


class TestAgentStrategy(unittest.TestCase):
    # test_registry_names - checks the three theme agents are registered by name
    def test_registry_names(self) -> None:
        self.assertEqual(set(AGENTS.keys()), {"history", "policy", "value"})
        self.assertEqual(HistoryAgent.name, "history")
        self.assertEqual(PolicyAgent.name, "policy")
        self.assertEqual(ValueAgent.name, "value")

    # test_run_agent_history_shape - checks history agent returns status and tags
    def test_run_agent_history_shape(self) -> None:
        out = run_agent(
            "history",
            AgentContext(fen=FEN_START, color="white", request_id="hist-1", move_history=[]),
        )
        self.assertEqual(out.get("status"), "ok")
        self.assertEqual(out.get("request_id"), "hist-1")
        self.assertIn("tags", out)
        self.assertIn("features", out)

    # test_run_agent_value_shape - checks value agent returns score fields
    def test_run_agent_value_shape(self) -> None:
        out = run_agent(
            "value",
            AgentContext(fen=FEN_START, color="white", request_id="val-1"),
        )
        self.assertEqual(out.get("status"), "ok")
        self.assertIn("score_cp", out)
        self.assertIn("win_chance_white", out)

    # test_run_agent_unknown_kind - checks unknown agent kind raises ValueError
    def test_run_agent_unknown_kind(self) -> None:
        with self.assertRaises(ValueError):
            run_agent("not-an-agent", AgentContext(fen=FEN_START, color="white"))


if __name__ == "__main__":
    unittest.main()
