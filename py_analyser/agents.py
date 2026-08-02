#!/usr/bin/env python3
# agents.py - AgentStrategy family for history / policy / value (theme three play agents)

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, List, Protocol


# AgentContext - shared inputs for history / policy / value agents
@dataclass
class AgentContext:
    fen: str
    color: str
    request_id: str | None = None
    profile: str = "intermediate"
    move_history: List[str] | None = None
    top_k: int = 5
    game_type: str = "chess"


# AgentStrategy - protocol for one of the three play-agent payloads
class AgentStrategy(Protocol):
    name: str

    # run - builds one agent payload dict for the shared analyzer contract
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        raise NotImplementedError


# HistoryAgent - builds the history-agent features and tags payload
class HistoryAgent:
    name = "history"

    # run - builds the history-agent features and tags payload
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        from analyzer import _history_agent_impl

        return _history_agent_impl(
            fen=ctx.fen,
            color=ctx.color,
            move_history=ctx.move_history,
            request_id=ctx.request_id,
            profile=ctx.profile,
        )


# PolicyAgent - builds the policy-agent candidates payload via move suggestions
class PolicyAgent:
    name = "policy"

    # run - builds the policy-agent candidates payload via move suggestions
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        from analyzer import _policy_agent_impl

        return _policy_agent_impl(
            fen=ctx.fen,
            color=ctx.color,
            top_k=ctx.top_k,
            request_id=ctx.request_id,
            profile=ctx.profile,
        )


# ValueAgent - builds the value-agent score and win-chance payload
class ValueAgent:
    name = "value"

    # run - builds the value-agent score and win-chance payload
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        from analyzer import _value_agent_impl

        return _value_agent_impl(
            fen=ctx.fen,
            color=ctx.color,
            request_id=ctx.request_id,
            profile=ctx.profile,
        )


AGENTS: Dict[str, AgentStrategy] = {
    "history": HistoryAgent(),
    "policy": PolicyAgent(),
    "value": ValueAgent(),
}


# run_agent - looks up a named AgentStrategy and runs it
def run_agent(kind: str, ctx: AgentContext) -> Dict[str, Any]:
    key = (kind or "").strip().lower()
    agent = AGENTS.get(key)
    if agent is None:
        raise ValueError(f'unknown agent kind "{kind}"')
    return agent.run(ctx)
