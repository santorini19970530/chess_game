#!/usr/bin/env python3
# agents.py - agent strategy family for history / policy / value

from __future__ import annotations

import math
import time
import uuid
from dataclasses import dataclass
from typing import Any, Dict, List, Protocol

import chess


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
        from analyzer import evaluate_position, parse_color

        started_at = time.perf_counter()
        board = chess.Board(ctx.fen)
        requested_color = parse_color(ctx.color)
        move_history = ctx.move_history or []
        _ = ctx.profile

        perspective_eval = evaluate_position(board, requested_color)
        features = {
            "is_check": board.is_check(),
            "is_checkmate": board.is_checkmate(),
            "is_stalemate": board.is_stalemate(),
            "material_delta_cp": perspective_eval,
            "move_count": len(move_history),
        }
        tags: List[str] = []
        if board.is_check():
            tags.append("check_pressure")
        if abs(perspective_eval) < 80:
            tags.append("balanced")
        elif perspective_eval > 0:
            tags.append("advantage")
        else:
            tags.append("disadvantage")
        if self._phase_from_board(board) == "opening":
            tags.append("book_like")

        latency_ms = int((time.perf_counter() - started_at) * 1000)
        return {
            "request_id": ctx.request_id or str(uuid.uuid4()),
            "status": "ok",
            "source": "rule_based_v1",
            "phase": self._phase_from_board(board),
            "features": features,
            "tags": tags,
            "latency_ms": latency_ms,
        }

    # _phase_from_board - classifies a chess position as opening, middlegame, or endgame
    def _phase_from_board(self, board: chess.Board) -> str:
        non_pawn_material = 0
        for piece_type in (chess.KNIGHT, chess.BISHOP, chess.ROOK, chess.QUEEN):
            non_pawn_material += len(board.pieces(piece_type, chess.WHITE))
            non_pawn_material += len(board.pieces(piece_type, chess.BLACK))

        if board.fullmove_number <= 10:
            return "opening"
        if non_pawn_material <= 6:
            return "endgame"
        return "middlegame"


# PolicyAgent - builds the policy-agent candidates payload via move suggestions
class PolicyAgent:
    name = "policy"

    # run - builds the policy-agent candidates payload via move suggestions
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        from move_suggest import MoveSuggestContext, select_suggestions

        started_at = time.perf_counter()
        # profile controls fairy-stockfish uci options inside select_suggestions
        suggestions, _name = select_suggestions(
            MoveSuggestContext(
                fen=ctx.fen,
                color=ctx.color,
                top_k=ctx.top_k,
                profile=ctx.profile,
                game_type="chess",
            )
        )

        if not suggestions:
            candidates: List[Dict[str, Any]] = []
        else:
            max_score = max(item.score for item in suggestions)
            exp_scores = [math.exp((item.score - max_score) / 100.0) for item in suggestions]
            total = sum(exp_scores) or 1.0
            candidates = []
            for item, exp_val in zip(suggestions, exp_scores):
                candidates.append(
                    {
                        "rank": item.rank,
                        "uci": item.uci,
                        "san": item.san,
                        "score_cp": item.score,
                        "prob": round(exp_val / total, 6),
                    }
                )

        best_move_uci = candidates[0]["uci"] if candidates else None
        latency_ms = int((time.perf_counter() - started_at) * 1000)
        return {
            "request_id": ctx.request_id or str(uuid.uuid4()),
            "status": "ok",
            "source": "fairy-stockfish",
            "best_move_uci": best_move_uci,
            "candidates": candidates,
            "latency_ms": latency_ms,
        }


# ValueAgent - builds the value-agent score and win-chance payload
class ValueAgent:
    name = "value"

    # run - builds the value-agent score and win-chance payload
    def run(self, ctx: AgentContext) -> Dict[str, Any]:
        from analyzer import cp_to_win_chance, evaluate_position, parse_color

        started_at = time.perf_counter()
        board = chess.Board(ctx.fen)
        _ = parse_color(ctx.color)  # validated for consistency with shared api contract
        _ = ctx.profile
        score_cp = evaluate_position(board, chess.WHITE)
        value = math.tanh(score_cp / 400.0)
        win_chance_white = cp_to_win_chance(score_cp)
        win_chance_black = 1.0 - win_chance_white
        latency_ms = int((time.perf_counter() - started_at) * 1000)

        return {
            "request_id": ctx.request_id or str(uuid.uuid4()),
            "status": "ok",
            "source": "heuristic",
            "score_cp": int(score_cp),
            "mate_in": 0,
            "value": round(float(value), 6),
            "win_chance_white": round(float(win_chance_white), 6),
            "win_chance_black": round(float(win_chance_black), 6),
            "latency_ms": latency_ms,
        }


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
