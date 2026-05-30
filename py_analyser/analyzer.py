#!/usr/bin/env python3
"""
Standalone chess analyzer.

Input:
  - FEN string
  - player color ("white" or "black")
Output:
  - top suggested moves with simple heuristic scores

No backend/frontend integration is required to use this file.
"""

from __future__ import annotations

import argparse
import json
import math
import time
import uuid
from dataclasses import dataclass
from typing import Dict, List

import chess

PIECE_VALUES = {
    chess.PAWN: 100,
    chess.KNIGHT: 320,
    chess.BISHOP: 330,
    chess.ROOK: 500,
    chess.QUEEN: 900,
    chess.KING: 0,
}


@dataclass(frozen=True)
class MoveSuggestion:
    rank: int
    uci: str
    san: str
    score: int


def parse_color(color: str) -> chess.Color:
    normalized = color.strip().lower()
    if normalized in {"white", "w"}:
        return chess.WHITE
    if normalized in {"black", "b"}:
        return chess.BLACK
    raise ValueError('color must be "white" or "black"')


def material_score(board: chess.Board, perspective: chess.Color) -> int:
    white_total = 0
    black_total = 0
    for piece_type, value in PIECE_VALUES.items():
        white_total += len(board.pieces(piece_type, chess.WHITE)) * value
        black_total += len(board.pieces(piece_type, chess.BLACK)) * value
    return white_total - black_total if perspective == chess.WHITE else black_total - white_total


def material_totals(board: chess.Board) -> Dict[str, int]:
    white_total = 0
    black_total = 0
    for piece_type, value in PIECE_VALUES.items():
        white_total += len(board.pieces(piece_type, chess.WHITE)) * value
        black_total += len(board.pieces(piece_type, chess.BLACK)) * value
    return {"white": white_total, "black": black_total}


def evaluate_position(board: chess.Board, perspective: chess.Color) -> int:
    if board.is_checkmate():
        # Side to move in checkmate loses.
        return -100_000 if board.turn == perspective else 100_000
    if board.is_stalemate() or board.is_insufficient_material():
        return 0

    score = material_score(board, perspective)

    # Small tactical/initiative bonuses.
    if board.is_check():
        score += 35 if board.turn != perspective else -35

    # Mobility bonus for perspective side.
    current_turn = board.turn
    board.turn = perspective
    perspective_mobility = board.legal_moves.count()
    board.turn = not perspective
    opponent_mobility = board.legal_moves.count()
    board.turn = current_turn
    score += (perspective_mobility - opponent_mobility) * 2

    return score


def suggest_moves(fen: str, color: str, top_k: int = 5) -> List[MoveSuggestion]:
    board = chess.Board(fen)
    target_color = parse_color(color)

    # Analyze from requested player's perspective even if FEN turn differs.
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
    top = scored[: max(1, top_k)]
    ranked: List[MoveSuggestion] = []
    for idx, item in enumerate(top, start=1):
        ranked.append(
            MoveSuggestion(rank=idx, uci=item.uci, san=item.san, score=item.score)
        )
    return ranked


def cp_to_win_chance(cp_score: int) -> float:
    # Logistic mapping from centipawn-like score to probability.
    return 1.0 / (1.0 + math.exp(-cp_score / 300.0))


def build_health_summary(board: chess.Board) -> Dict[str, object]:
    totals = material_totals(board)
    side_to_move = "white" if board.turn == chess.WHITE else "black"
    side_in_check = board.is_check()
    return {
        "material_white": totals["white"],
        "material_black": totals["black"],
        "material_balance_white_minus_black": totals["white"] - totals["black"],
        "side_to_move": side_to_move,
        "white_in_check": side_in_check and board.turn == chess.WHITE,
        "black_in_check": side_in_check and board.turn == chess.BLACK,
    }


def build_threat_summary(board: chess.Board, eval_cp_white: int) -> str:
    if board.is_checkmate():
        winner = "black" if board.turn == chess.WHITE else "white"
        return f"{winner} has a forced checkmate."
    if board.is_stalemate():
        return "Position is stalemate."
    if board.is_check():
        checked_side = "white" if board.turn == chess.WHITE else "black"
        return f"{checked_side} king is in check."
    if eval_cp_white > 150:
        return "White has the initiative."
    if eval_cp_white < -150:
        return "Black has the initiative."
    return "Position is roughly balanced."


def analyze_position(
    fen: str, color: str, top_k: int = 5, request_id: str | None = None
) -> Dict[str, object]:
    started_at = time.perf_counter()
    board = chess.Board(fen)
    requested_color = parse_color(color)
    suggestions = suggest_moves(fen, color, top_k)

    eval_cp_white = evaluate_position(board, chess.WHITE)
    win_chance_white = cp_to_win_chance(eval_cp_white)
    win_chance_black = 1.0 - win_chance_white
    best_move_uci = suggestions[0].uci if suggestions else None
    latency_ms = int((time.perf_counter() - started_at) * 1000)

    return {
        "request_id": request_id or str(uuid.uuid4()),
        "status": "ok",
        "source": "heuristic",
        "fen": fen,
        "evaluated_for_color": "white" if requested_color == chess.WHITE else "black",
        "health_summary": build_health_summary(board),
        "is_check": board.is_check(),
        "is_checkmate": board.is_checkmate(),
        "is_stalemate": board.is_stalemate(),
        "eval_cp_white": eval_cp_white,
        "win_chance_white": round(win_chance_white, 4),
        "win_chance_black": round(win_chance_black, 4),
        "threat_summary": build_threat_summary(board, eval_cp_white),
        "best_move_uci": best_move_uci,
        "suggested_moves": [item.__dict__ for item in suggestions],
        "latency_ms": latency_ms,
    }


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Suggest chess moves from FEN and player color.")
    parser.add_argument("--fen", required=True, help="FEN position string")
    parser.add_argument("--color", required=True, choices=["white", "black", "w", "b"], help="Player color")
    parser.add_argument("--top-k", type=int, default=5, help="Number of suggestions to return")
    parser.add_argument(
        "--format",
        default="json",
        choices=["json", "text"],
        help="Output format",
    )
    return parser


def main() -> None:
    parser = _build_parser()
    args = parser.parse_args()

    result = analyze_position(args.fen, args.color, args.top_k)
    if args.format == "text":
        print("Health summary:")
        print(json.dumps(result["health_summary"], indent=2))
        print(
            f'Win chance: white={result["win_chance_white"]:.4f}, black={result["win_chance_black"]:.4f}'
        )

        suggestions = result["suggested_moves"]
        if not suggestions:
            print("No legal moves.")
            return
        print("Suggested moves:")
        for idx, item in enumerate(suggestions, start=1):
            print(f'{idx}. {item["uci"]} ({item["san"]}) score={item["score"]}')
        return

    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
