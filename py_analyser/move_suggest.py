#!/usr/bin/env python3
# move_suggest.py - MoveSuggestStrategy family (FS → Heuristic), mirrors Go AIMoveStrategy

from __future__ import annotations

from dataclasses import dataclass
from typing import List, Optional, Protocol, Tuple, TYPE_CHECKING

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
        from analyzer import _heuristic_suggest_impl

        return _heuristic_suggest_impl(ctx.fen, ctx.color, ctx.top_k)


# FairyStockfishSuggest - runs Fairy-Stockfish MultiPV for chess
class FairyStockfishSuggest:
    name = "fairy_stockfish"

    # suggest - runs Fairy-Stockfish MultiPV for chess; empty on engine failure
    def suggest(self, ctx: MoveSuggestContext) -> List["MoveSuggestion"]:
        from analyzer import _fs_suggest_impl

        try:
            return _fs_suggest_impl(ctx.fen, ctx.color, ctx.top_k, ctx.profile)
        except Exception:
            return []


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
        from analyzer import _fs_variant_suggest_impl

        return _fs_variant_suggest_impl(ctx.fen, ctx.game_type, ctx.top_k, ctx.profile)


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
