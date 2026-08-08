#!/usr/bin/env python3
# move_suggest.py - move suggest strategy family (fs → heuristic), mirrors go aimove strategy

from __future__ import annotations

from dataclasses import dataclass
from typing import Dict, List, Optional, Protocol, Tuple, TYPE_CHECKING

import chess

if TYPE_CHECKING:
    from analyzer import MoveSuggestion


# MoveSuggestContext - shared inputs for move-suggestion strategies
@dataclass
class MoveSuggestContext:
    fen: str
    color: str
    top_k: int = 5
    profile: str = "intermediate"
    game_type: str = "chess"


# MoveSuggestStrategy - protocol for ranked move suggestion providers
class MoveSuggestStrategy(Protocol):
    name: str

    # suggest - returns ranked moves; empty list is a soft-miss for the selector
    def suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        raise NotImplementedError


# HeuristicSuggest - ranks chess legal moves with the heuristic evaluator
class HeuristicSuggest:
    name = "heuristic"

    # suggest - ranks chess legal moves with the heuristic evaluator
    def suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        from analyzer import MoveSuggestion, evaluate_position, parse_color

        board = chess.Board(ctx.fen)
        target_color = parse_color(ctx.color)

        # analyze from requested player's perspective even if fen turn differs
        analysis_board = board.copy(stack=False)
        analysis_board.turn = target_color

        if analysis_board.is_game_over():
            return []

        scored: List[MoveSuggestion] = []
        for move in analysis_board.legal_moves:
            san = analysis_board.san(move)
            analysis_board.push(move)
            score = evaluate_position(analysis_board, target_color)
            analysis_board.pop()
            scored.append(MoveSuggestion(rank=0, uci=move.uci(), san=san, score=score))

        scored.sort(key=lambda item: item.score, reverse=True)
        top = scored[: max(1, ctx.top_k)]
        ranked: List[MoveSuggestion] = []
        for idx, item in enumerate(top, start=1):
            ranked.append(
                MoveSuggestion(rank=idx, uci=item.uci, san=item.san, score=item.score)
            )
        return ranked


# FairyStockfishSuggest - runs Fairy-Stockfish MultiPV for chess
class FairyStockfishSuggest:
    name = "fairy_stockfish"

    # suggest - runs Fairy-Stockfish MultiPV for chess; empty on engine failure
    def suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        try:
            return self._suggest(ctx)
        except Exception:
            return []

    # _suggest - runs fairy-stockfish multipv for chess; raises on engine failure
    def _suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        from analyzer import MoveSuggestion, parse_color
        from fs_engine import get_engine, profile_to_uci_options

        board = chess.Board(ctx.fen)
        target_color = parse_color(ctx.color)
        board.turn = target_color

        if board.is_game_over():
            return []

        engine = get_engine()
        options, limit = profile_to_uci_options(ctx.profile)
        engine.configure(options)

        multipv = max(1, min(ctx.top_k, 10))
        analysis = engine.analyse(board, limit, multipv=multipv)

        suggestions: List[MoveSuggestion] = []
        for idx, info in enumerate(analysis, start=1):
            move = info.get("pv", [None])[0]
            if move is None:
                continue
            score = info.get("score")
            cp = score.white().score(mate_score=100000) if score else 0
            san = board.san(move)
            suggestions.append(
                MoveSuggestion(rank=idx, uci=move.uci(), san=san, score=cp)
            )

        # pad with legal moves when multipv returns fewer than top_k (still fs path)
        if len(suggestions) < ctx.top_k:
            for move in list(board.legal_moves)[len(suggestions) : ctx.top_k]:
                san = board.san(move)
                suggestions.append(
                    MoveSuggestion(
                        rank=len(suggestions) + 1, uci=move.uci(), san=san, score=0
                    )
                )

        return suggestions[: ctx.top_k]


# FairyStockfishVariantSuggest - runs raw-UCI MultiPV for xianqi/shogi
class FairyStockfishVariantSuggest:
    name = "fairy_stockfish_variant"

    # suggest - returns variant MultiPV moves without the eval score
    def suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        suggestions, _eval = self.suggest_with_eval(ctx)
        return suggestions

    # suggest_with_eval - runs raw-UCI MultiPV for xianqi/shogi and returns eval cp
    def suggest_with_eval(
        self, ctx: MoveSuggestContext
    ) -> Tuple[List["MoveSuggestion"], Optional[int]]:
        import time

        from analyzer import MoveSuggestion
        from fs_engine import (
            parse_info_multipv_pv,
            parse_info_score_cp,
            raw_uci_ensure,
            raw_uci_lock,
            raw_uci_write,
            uci_score_as_white,
            uci_variant_name,
        )

        fen = ctx.fen
        variant = uci_variant_name(ctx.game_type)
        # match chess analyze stability: true eval, not profile-handicapped search
        _ = ctx.profile  # kept for api compatibility with callers
        skill = 20
        limit = chess.engine.Limit(depth=10, time=0.5)
        multipv = max(1, min(ctx.top_k, 10))
        go_parts = []
        if limit.time is not None:
            go_parts.append(f"movetime {int(limit.time * 1000)}")
        if limit.depth is not None:
            go_parts.append(f"depth {int(limit.depth)}")
        go_cmd = "go " + " ".join(go_parts) if go_parts else "go depth 10"

        with raw_uci_lock:
            proc = raw_uci_ensure(variant)
            raw_uci_write(proc, f"setoption name Skill Level value {skill}")
            raw_uci_write(proc, f"setoption name MultiPV value {multipv}")
            raw_uci_write(proc, f"position fen {fen}")
            raw_uci_write(proc, go_cmd)

            assert proc.stdout is not None
            seen: Dict[int, MoveSuggestion] = {}
            eval_cp_white: Optional[int] = None
            deadline = time.monotonic() + 12.0
            while time.monotonic() < deadline:
                line = proc.stdout.readline()
                if not line:
                    break
                line = line.strip()
                if line.startswith("bestmove"):
                    break
                if not line.startswith("info"):
                    continue
                fields = line.split()
                idx, move = parse_info_multipv_pv(fields)
                score = parse_info_score_cp(fields)
                if move is None:
                    continue
                if score is not None and idx == 1:
                    eval_cp_white = uci_score_as_white(score, fen)
                if 1 <= idx <= multipv:
                    seen[idx] = MoveSuggestion(
                        rank=idx, uci=move, san=move, score=score if score is not None else 0
                    )

            suggestions = [seen[i] for i in range(1, multipv + 1) if i in seen]
            return suggestions[: ctx.top_k], eval_cp_white


# select_suggestions - runs FS then Heuristic for chess; variant FS only (no chess Board fallback)
def select_suggestions(ctx: MoveSuggestContext) -> Tuple[List["MoveSuggestion"], str]:
    gt = (ctx.game_type or "chess").strip().lower()
    if gt in {"xianqi", "xiangqi", "shogi"}:
        strat = FairyStockfishVariantSuggest()
        try:
            suggestions, _eval = strat.suggest_with_eval(ctx)
            if suggestions:
                return suggestions, strat.name
        except Exception:
            pass
        return [], "fallback"

    fs = FairyStockfishSuggest()
    out = fs.suggest(ctx)
    if out:
        return out, fs.name
    heuristic = HeuristicSuggest()
    return heuristic.suggest(ctx), heuristic.name
