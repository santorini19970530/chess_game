#!/usr/bin/env python3
# analyzer.py - facade: board facts, analyze payloads, and coach ground-truth helpers

from __future__ import annotations

import argparse
import json
import math
import re
import time
import uuid
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple

import chess

PIECE_VALUES = {
    chess.PAWN: 100,
    chess.KNIGHT: 320,
    chess.BISHOP: 330,
    chess.ROOK: 500,
    chess.QUEEN: 900,
    chess.KING: 0,
}


# MoveSuggestion - one ranked move suggestion with uci, san, and score
@dataclass(frozen=True)
class MoveSuggestion:
    rank: int
    uci: str
    san: str
    score: int


# parse_color - maps a color string to chess.WHITE or chess.BLACK
def parse_color(color: str) -> chess.Color:
    normalized = color.strip().lower()
    if normalized in {"white", "w"}:
        return chess.WHITE
    if normalized in {"black", "b"}:
        return chess.BLACK
    raise ValueError('color must be "white" or "black"')


# material_score - returns material balance from the given side's perspective
def material_score(board: chess.Board, perspective: chess.Color) -> int:
    white_total = 0
    black_total = 0
    for piece_type, value in PIECE_VALUES.items():
        white_total += len(board.pieces(piece_type, chess.WHITE)) * value
        black_total += len(board.pieces(piece_type, chess.BLACK)) * value
    return white_total - black_total if perspective == chess.WHITE else black_total - white_total


# material_totals - returns white and black material totals in centipawn-like units
def material_totals(board: chess.Board) -> Dict[str, int]:
    white_total = 0
    black_total = 0
    for piece_type, value in PIECE_VALUES.items():
        white_total += len(board.pieces(piece_type, chess.WHITE)) * value
        black_total += len(board.pieces(piece_type, chess.BLACK)) * value
    return {"white": white_total, "black": black_total}


# evaluate_position - scores a chess position from the given side's perspective
def evaluate_position(board: chess.Board, perspective: chess.Color) -> int:
    if board.is_checkmate():
        # side to move in checkmate loses
        return -100_000 if board.turn == perspective else 100_000
    if board.is_stalemate() or board.is_insufficient_material():
        return 0

    score = material_score(board, perspective)

    # small tactical/initiative bonuses
    if board.is_check():
        score += 35 if board.turn != perspective else -35

    # mobility bonus for perspective side
    current_turn = board.turn
    board.turn = perspective
    perspective_mobility = board.legal_moves.count()
    board.turn = not perspective
    opponent_mobility = board.legal_moves.count()
    board.turn = current_turn
    score += (perspective_mobility - opponent_mobility) * 2

    return score


# cp_to_win_chance - maps a centipawn-like score to a win probability
def cp_to_win_chance(cp_score: int) -> float:
    return 1.0 / (1.0 + math.exp(-cp_score / 300.0))


# build_health_summary - builds material and check fields for an analyze payload
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


# build_threat_summary - returns a short threat/initiative note for the position
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


_PIECE_NAMES = {
    chess.PAWN: "pawn",
    chess.KNIGHT: "knight",
    chess.BISHOP: "bishop",
    chess.ROOK: "rook",
    chess.QUEEN: "queen",
    chess.KING: "king",
}


# normalize_history_uci - strips session labels like 'White: e2e4' down to bare uci
def normalize_history_uci(raw: str) -> str:
    s = str(raw or "").strip()
    if ":" in s:
        s = s.split(":", 1)[1].strip()
    return s.lower()


_SHOGI_KIND = {
    "p": "pawn",
    "l": "lance",
    "n": "knight",
    "s": "silver",
    "g": "gold",
    "b": "bishop",
    "r": "rook",
    "k": "king",
}
_SHOGI_PROMO_KIND = {
    "p": "tokin",
    "l": "promoted lance",
    "n": "promoted knight",
    "s": "promoted silver",
    "b": "horse",
    "r": "dragon",
}
_XIANGQI_KIND = {
    "k": "general",
    "a": "advisor",
    "b": "elephant",
    "e": "elephant",
    "n": "horse",
    "r": "chariot",
    "c": "cannon",
    "p": "soldier",
}


# _variant_board_grid - maps (file, rank) to piece kind from fen placement
def _variant_board_grid(fen: str, *, files: int, ranks: int) -> Dict[Tuple[int, int], str]:
    placement = (fen or "").split()[0] if fen else ""
    if "[" in placement:
        placement = placement.split("[", 1)[0]
    rows = placement.split("/")
    if len(rows) != ranks:
        return {}
    out: Dict[Tuple[int, int], str] = {}
    for i, row in enumerate(rows):
        rank = ranks - i
        file_i = 1
        j = 0
        while j < len(row) and file_i <= files:
            ch = row[j]
            if ch.isdigit():
                file_i += int(ch)
                j += 1
                continue
            promoted = False
            if ch == "+":
                promoted = True
                j += 1
                if j >= len(row):
                    break
                ch = row[j]
            kind_key = ch.lower()
            if ranks == 9:  # shogi
                kind = (
                    _SHOGI_PROMO_KIND.get(kind_key)
                    if promoted
                    else _SHOGI_KIND.get(kind_key)
                )
            else:
                kind = _XIANGQI_KIND.get(kind_key)
            if kind and 1 <= file_i <= files:
                out[(file_i, rank)] = kind
            file_i += 1
            j += 1
    return out


# _variant_move_label - builds a human label for xianqi/shogi uci from the post-move fen
def _variant_move_label(fen: str, target: str, gt: str) -> str:
    key = "shogi" if gt == "shogi" else "xianqi"
    drop = re.match(r"^([plnsgbr])[*@]([a-i])([1-9])$", target)
    if drop and key == "shogi":
        kind = _SHOGI_KIND.get(drop.group(1), "piece")
        return f"drop {kind} → {drop.group(2)}{drop.group(3)}"

    board = re.match(r"^([a-i])(\d{1,2})([a-i])(\d{1,2})(\+?)$", target)
    if not board:
        return target
    ff, fr = board.group(1), board.group(2)
    tf, tr, promo = board.group(3), int(board.group(4)), board.group(5)
    files, ranks = (9, 9) if key == "shogi" else (9, 10)
    grid = _variant_board_grid(fen, files=files, ranks=ranks)
    kind = grid.get((ord(tf) - ord("a") + 1, tr), "piece")
    suffix = " (promote)" if promo == "+" else ""
    return f"{kind} {ff}{fr}→{tf}{tr}{suffix}"


# _chess_ground_from_before_board - builds chess ground-truth dict from a before-move board
def _chess_ground_from_before_board(
    board: chess.Board,
    move: chess.Move,
    target: str,
    fen: str,
    hist: List[str],
) -> Dict[str, str]:
    piece = board.piece_at(move.from_square)
    piece_name = _PIECE_NAMES.get(piece.piece_type, "piece") if piece else "piece"
    mover = "White" if piece and piece.color == chess.WHITE else "Black"
    from_sq = chess.square_name(move.from_square)
    to_sq = chess.square_name(move.to_square)
    san = board.san(move)
    is_capture = board.is_capture(move)
    if board.is_en_passant(move):
        capture_bit = ", capturing pawn en passant"
    elif is_capture:
        victim = board.piece_at(move.to_square)
        vname = _PIECE_NAMES.get(victim.piece_type, "piece") if victim else "piece"
        capture_bit = f", capturing {vname} on {to_sq}"
    else:
        capture_bit = ", no capture"

    board.push(move)
    attacked: List[str] = []
    for sq in board.attacks(move.to_square):
        hit = board.piece_at(sq)
        if hit is not None and piece is not None and hit.color != piece.color:
            attacked.append(f"{_PIECE_NAMES.get(hit.piece_type, 'piece')} on {chess.square_name(sq)}")
    if attacked:
        attack_bit = " After the move, that piece attacks: " + ", ".join(attacked) + "."
    else:
        attack_bit = " After the move, that piece attacks no enemy piece."

    fen_ok = True
    fen_core = (fen or "").split()
    if fen_core:
        fen_ok = board.board_fen() == fen_core[0]
    mismatch = "" if fen_ok else " (replay FEN mismatch — still prefer these move facts over invention)."

    recent = " ".join(hist[-6:]) if hist else target
    summary = (
        f"GROUND TRUTH (never contradict): {mover} moved {piece_name} {from_sq}→{to_sq} "
        f"(SAN {san}, UCI {target}){capture_bit}.{attack_bit} "
        f"Recent UCI history: {recent}.{mismatch}"
    )
    return {"summary": summary, "san": san, "uci": target}


# _chess_ground_from_after_fen - tip/diagram sessions: fen is after the move; undo uci for san
def _chess_ground_from_after_fen(fen: str, target: str, hist: List[str]) -> Optional[Dict[str, str]]:
    try:
        after = chess.Board(fen)
        move = chess.Move.from_uci(target)
    except Exception:
        return None
    piece = after.piece_at(move.to_square)
    if piece is None:
        return None
    # rebuild the before-move board by reversing the tip ply
    before = after.copy(stack=False)
    before.remove_piece_at(move.to_square)
    if move.promotion:
        before.set_piece_at(move.from_square, chess.Piece(chess.PAWN, piece.color))
    else:
        before.set_piece_at(move.from_square, piece)
    before.turn = piece.color
    if move not in before.legal_moves:
        # capture: restore a missing enemy piece on the destination and retry
        for pt in (chess.PAWN, chess.KNIGHT, chess.BISHOP, chess.ROOK, chess.QUEEN):
            before.set_piece_at(move.to_square, chess.Piece(pt, not piece.color))
            if move in before.legal_moves:
                break
            before.remove_piece_at(move.to_square)
        else:
            return None
    if move not in before.legal_moves:
        return None
    return _chess_ground_from_before_board(before, move, target, fen, hist)


# build_move_ground_truth - builds grounded last-move facts for the explain prompt
def build_move_ground_truth(
    fen: str,
    move_uci: str,
    move_history: Optional[List[str]] = None,
    game_type: str = "chess",
) -> Dict[str, str]:
    gt = (game_type or "chess").strip().lower()
    target = normalize_history_uci(move_uci)
    hist = [normalize_history_uci(m) for m in (move_history or []) if str(m).strip()]
    hist = [m for m in hist if m]
    if not target and hist:
        target = hist[-1]
    if not target:
        return {"summary": "GROUND TRUTH: last move unknown — do not invent tactics, attacks, or captures."}

    if gt in {"xianqi", "xiangqi", "shogi"}:
        label = "Xiangqi" if gt in {"xianqi", "xiangqi"} else "Shogi"
        move_lab = _variant_move_label(fen, target, gt)
        squares = " ".join(dict.fromkeys(re.findall(r"[a-i]\d{1,2}", f"{move_lab} {target}")))
        summary = (
            f"GROUND TRUTH: last {label} move {move_lab} (UCI {target}). "
            f"Only mention squares {squares or target}. "
            f"Do not invent other pieces, squares, captures, or Chess ideas (centre/castling)."
        )
        return {"summary": summary, "san": move_lab, "uci": target}

    try:
        board = chess.Board()
        if hist and hist[-1] == target:
            for u in hist[:-1]:
                board.push_uci(u)
        elif hist and target in hist:
            for u in hist:
                if u == target:
                    break
                board.push_uci(u)
        elif hist:
            # history present but target not in it — replay all then fail closed
            for u in hist:
                board.push_uci(u)
            return {
                "summary": (
                    f"GROUND TRUTH: explained move {target} is not the tip of move_history "
                    f"({hist[-1]}); do not invent attacks. Recent UCI: {' '.join(hist[-6:])}."
                ),
                "uci": target,
            }

        move = chess.Move.from_uci(target)
        if move not in board.legal_moves:
            # tip FEN games: history alone starts from classic opening — undo from after-fen instead
            from_after = _chess_ground_from_after_fen(fen, target, hist)
            if from_after is not None:
                return from_after
            return {
                "summary": f"GROUND TRUTH: {target} is not legal in the replayed position — do not invent tactics.",
                "uci": target,
            }

        return _chess_ground_from_before_board(board, move, target, fen, hist)
    except Exception:
        from_after = _chess_ground_from_after_fen(fen, target, hist)
        if from_after is not None:
            return from_after
        return {
            "summary": f"GROUND TRUTH: last move UCI {target}; do not invent piece attacks or captures.",
            "uci": target,
        }


# build_concept_hints - builds up to max_hints cues from an analyze-shaped dict (threat, material, replies)
def build_concept_hints(
    analysis: Optional[Dict[str, object]] = None,
    *,
    max_hints: int = 3,
) -> List[str]:
    if not analysis or max_hints <= 0:
        return []

    hints: List[str] = []

    threat = str(analysis.get("threat_summary") or "").strip()
    if threat:
        hints.append(threat)

    balance = analysis.get("health_summary")
    material_cp: Optional[int] = None
    if isinstance(balance, dict):
        raw = balance.get("material_balance_white_minus_black")
        try:
            material_cp = int(raw)  # type: ignore[arg-type]
        except (TypeError, ValueError):
            material_cp = None
    if material_cp is None:
        raw_eval = analysis.get("eval_cp_white")
        try:
            material_cp = int(raw_eval)  # type: ignore[arg-type]
        except (TypeError, ValueError):
            material_cp = None
    if material_cp is not None and abs(material_cp) >= 100:
        if material_cp > 0:
            hints.append("White is ahead on material / evaluation.")
        else:
            hints.append("Black is ahead on material / evaluation.")

    labels: List[str] = []
    suggestions = analysis.get("suggested_moves")
    if isinstance(suggestions, list):
        for item in suggestions[:3]:
            if isinstance(item, dict):
                lab = str(item.get("san") or item.get("uci") or "").strip()
            else:
                lab = str(getattr(item, "san", None) or getattr(item, "uci", "") or "").strip()
            if lab:
                labels.append(lab)
    if not labels:
        best = str(analysis.get("best_move_uci") or "").strip()
        if best:
            labels = [best]
    if labels:
        hints.append("Engine suggested replies (side to move): " + ", ".join(labels) + ".")

    # dedupe while preserving order (threat can repeat material idea)
    seen: set[str] = set()
    unique: List[str] = []
    for h in hints:
        key = h.lower()
        if key in seen:
            continue
        seen.add(key)
        unique.append(h)
        if len(unique) >= max_hints:
            break
    return unique


# build_explanation_fallback - builds offline coach text when the llm path is unavailable
def build_explanation_fallback(
    fen: str,
    color: str,
    move_uci: str,
    move_san: str | None = None,
    game_type: str = "chess",
) -> str:
    move_text = move_san or move_uci
    gt = (game_type or "chess").strip().lower()
    token = normalize_history_uci(move_uci or "")
    position_only = token in {"", "position", "diagram"}

    if gt in {"xianqi", "xiangqi", "shogi"}:
        label = "Xiangqi" if gt in {"xianqi", "xiangqi"} else "Shogi"
        if position_only:
            return (
                f"Tip {label} position — re-check checks, captures, and piece safety. "
                f"Next: use the suggested moves for the side to move."
            )
        lab = _variant_move_label(fen, token, gt) or move_text
        return (
            f"{lab} is a legal-looking {label} move. "
            f"Re-check checks, captures, and piece safety before committing."
        )

    board = chess.Board(fen)
    requested = parse_color(color)
    board.turn = requested

    threat = build_threat_summary(board, evaluate_position(board, chess.WHITE))
    material = material_score(board, requested)
    sign = "ahead" if material > 50 else ("behind" if material < -50 else "level")
    if position_only:
        return (
            f"Tip position — material looks {sign}. {threat} "
            "Next: check the suggested moves for the side to move, and watch checks, captures, and loose pieces."
        )
    return (
        f"{move_text} keeps material {sign}. {threat} "
        "Next: check the suggested moves for replies, and watch checks, captures, and loose pieces."
    )


# _analyze_position_variant - analyzes xianqi/shogi via fairy-stockfish uci; never chess.Board(fen)
def _analyze_position_variant(
    fen: str,
    color: str,
    top_k: int,
    request_id: str | None,
    game_type: str,
    profile: str = "intermediate",
) -> Dict[str, object]:
    started_at = time.perf_counter()
    requested_color = parse_color(color)
    eval_cp_white = 0
    suggestions: List[MoveSuggestion] = []
    source = "fairy-stockfish"
    # leave empty on success — a stub threat line looked like fs endorsed the llm
    threat = ""

    try:
        from move_suggest import FairyStockfishVariantSuggest, MoveSuggestContext

        suggestions, score = FairyStockfishVariantSuggest().suggest_with_eval(
            MoveSuggestContext(
                fen=fen,
                color="white",
                top_k=top_k,
                profile=profile,
                game_type=game_type,
            )
        )
        if score is not None:
            eval_cp_white = score
    except Exception:
        # fs down / timeout: keep service up with empty suggestions
        source = "fallback"
        threat = "Fairy-Stockfish unavailable; variant analysis fallback."
        suggestions = []
        eval_cp_white = 0

    win_chance_white = cp_to_win_chance(eval_cp_white)
    win_chance_black = 1.0 - win_chance_white
    best_move_uci = suggestions[0].uci if suggestions else None
    side_to_move = "white" if fen.split()[1:2] == ["w"] else (
        "black" if fen.split()[1:2] == ["b"] else "white"
    )
    latency_ms = int((time.perf_counter() - started_at) * 1000)

    return {
        "request_id": request_id or str(uuid.uuid4()),
        "status": "ok",
        "source": source,
        "fen": fen,
        "evaluated_for_color": "white" if requested_color == chess.WHITE else "black",
        "health_summary": {
            "material_white": 0,
            "material_black": 0,
            "material_balance_white_minus_black": 0,
            "side_to_move": side_to_move,
            "white_in_check": False,
            "black_in_check": False,
        },
        "is_check": False,
        "is_checkmate": False,
        "is_stalemate": False,
        "eval_cp_white": eval_cp_white,
        "win_chance_white": round(win_chance_white, 4),
        "win_chance_black": round(win_chance_black, 4),
        "threat_summary": threat,
        "best_move_uci": best_move_uci,
        "suggested_moves": [item.__dict__ for item in suggestions],
        "latency_ms": latency_ms,
        "game_type": game_type,
    }


# analyze_position - builds the shared analyze payload for chess or variant game types
def analyze_position(
    fen: str,
    color: str,
    top_k: int = 5,
    request_id: str | None = None,
    game_type: str = "chess",
    profile: str = "intermediate",
) -> Dict[str, object]:
    gt = (game_type or "chess").strip().lower()
    if gt in {"xianqi", "shogi"}:
        return _analyze_position_variant(
            fen, color, top_k, request_id, gt, profile=profile
        )

    from move_suggest import HeuristicSuggest, MoveSuggestContext

    started_at = time.perf_counter()
    board = chess.Board(fen)
    requested_color = parse_color(color)
    suggestions = HeuristicSuggest().suggest(
        MoveSuggestContext(fen=fen, color=color, top_k=top_k, game_type="chess")
    )

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
        "game_type": "chess",
    }


# _build_parser - builds the cli argument parser for standalone analyzer runs
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


# main - runs the standalone analyzer cli and prints suggestions
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
