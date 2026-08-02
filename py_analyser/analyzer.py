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
import os
import random
import re
import subprocess
import threading
import time
import uuid
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple

import chess
import chess.engine


# Fairy-Stockfish binary path (override via environment variable)
FS_BINARY_PATH: str = os.environ.get(
    "FAIRY_STOCKFISH_PATH",
    os.path.join(os.path.dirname(__file__), "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
)

# Session game_type → Fairy-Stockfish UCI_Variant name.
_GAME_TYPE_TO_UCI_VARIANT = {
    "chess": "chess",
    "xianqi": "xiangqi",
    "shogi": "shogi",
}

_engine: Optional[chess.engine.SimpleEngine] = None
_raw_uci_lock = threading.Lock()
_raw_uci_proc: Optional[subprocess.Popen] = None
_raw_uci_variant: Optional[str] = None


def _get_engine() -> chess.engine.SimpleEngine:
    """Return a singleton Fairy-Stockfish engine instance (opened once)."""
    global _engine
    if _engine is None:
        if not os.path.exists(FS_BINARY_PATH):
            raise FileNotFoundError(
                f"Fairy-Stockfish binary not found at {FS_BINARY_PATH}. "
                "Set FAIRY_STOCKFISH_PATH environment variable to the correct path."
            )
        _engine = chess.engine.SimpleEngine.popen_uci(FS_BINARY_PATH)
    return _engine


def uci_variant_name(game_type: str) -> str:
    key = (game_type or "chess").strip().lower()
    return _GAME_TYPE_TO_UCI_VARIANT.get(key, key)


def _raw_uci_ensure(variant: str) -> subprocess.Popen:
    """Singleton raw UCI process for variant FENs (python-chess Board is chess-only)."""
    global _raw_uci_proc, _raw_uci_variant
    if _raw_uci_proc is not None and _raw_uci_proc.poll() is None:
        if _raw_uci_variant != variant:
            _raw_uci_write(_raw_uci_proc, f"setoption name UCI_Variant value {variant}")
            _raw_uci_write(_raw_uci_proc, "isready")
            _raw_uci_wait_for(_raw_uci_proc, "readyok", timeout=5.0)
            _raw_uci_variant = variant
        return _raw_uci_proc

    if not os.path.exists(FS_BINARY_PATH):
        raise FileNotFoundError(
            f"Fairy-Stockfish binary not found at {FS_BINARY_PATH}. "
            "Set FAIRY_STOCKFISH_PATH environment variable to the correct path."
        )
    proc = subprocess.Popen(
        [FS_BINARY_PATH],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        bufsize=1,
    )
    _raw_uci_write(proc, "uci")
    _raw_uci_wait_for(proc, "uciok", timeout=5.0)
    _raw_uci_write(proc, f"setoption name UCI_Variant value {variant}")
    _raw_uci_write(proc, "isready")
    _raw_uci_wait_for(proc, "readyok", timeout=5.0)
    _raw_uci_proc = proc
    _raw_uci_variant = variant
    return proc


def _raw_uci_write(proc: subprocess.Popen, line: str) -> None:
    assert proc.stdin is not None
    proc.stdin.write(line + "\n")
    proc.stdin.flush()


def _raw_uci_wait_for(proc: subprocess.Popen, token: str, timeout: float) -> None:
    assert proc.stdout is not None
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        line = proc.stdout.readline()
        if not line:
            raise RuntimeError("Fairy-Stockfish exited while waiting for " + token)
        if token in line:
            return
    raise TimeoutError(f"timeout waiting for {token}")


def _parse_info_score_cp(fields: List[str]) -> Optional[int]:
    for i, f in enumerate(fields):
        if f == "score" and i + 2 < len(fields):
            if fields[i + 1] == "cp":
                try:
                    return int(fields[i + 2])
                except ValueError:
                    return None
            if fields[i + 1] == "mate":
                try:
                    mate = int(fields[i + 2])
                except ValueError:
                    return None
                return 100_000 if mate > 0 else -100_000
    return None


def _parse_info_multipv_pv(fields: List[str]) -> Tuple[int, Optional[str]]:
    multipv = 1
    move: Optional[str] = None
    for i, f in enumerate(fields):
        if f == "multipv" and i + 1 < len(fields):
            try:
                multipv = int(fields[i + 1])
            except ValueError:
                multipv = 1
        if f == "pv" and i + 1 < len(fields):
            move = fields[i + 1]
    return multipv, move


def uci_score_as_white(score_cp: int, fen: str) -> int:
    """UCI info scores are from the side to move; chess analyze uses White POV."""
    parts = fen.split()
    if len(parts) >= 2 and parts[1].lower() == "b":
        return -int(score_cp)
    return int(score_cp)


# _fs_variant_suggest_impl - runs raw-UCI MultiPV for xianqi/shogi and returns white-pov eval
def _fs_variant_suggest_impl(
    fen: str,
    game_type: str,
    top_k: int = 5,
    profile: str = "intermediate",
) -> Tuple[List[MoveSuggestion], Optional[int]]:
    variant = uci_variant_name(game_type)
    # match chess analyze stability: true eval, not profile-handicapped search
    _ = profile  # kept for API compatibility with callers
    skill = 20
    limit = chess.engine.Limit(depth=10, time=0.5)
    multipv = max(1, min(top_k, 10))
    go_parts = []
    if limit.time is not None:
        go_parts.append(f"movetime {int(limit.time * 1000)}")
    if limit.depth is not None:
        go_parts.append(f"depth {int(limit.depth)}")
    go_cmd = "go " + " ".join(go_parts) if go_parts else "go depth 10"

    with _raw_uci_lock:
        proc = _raw_uci_ensure(variant)
        _raw_uci_write(proc, f"setoption name Skill Level value {skill}")
        _raw_uci_write(proc, f"setoption name MultiPV value {multipv}")
        _raw_uci_write(proc, f"position fen {fen}")
        _raw_uci_write(proc, go_cmd)

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
            idx, move = _parse_info_multipv_pv(fields)
            score = _parse_info_score_cp(fields)
            if move is None:
                continue
            if score is not None and idx == 1:
                eval_cp_white = uci_score_as_white(score, fen)
            if 1 <= idx <= multipv:
                seen[idx] = MoveSuggestion(
                    rank=idx, uci=move, san=move, score=score if score is not None else 0
                )

        suggestions = [seen[i] for i in range(1, multipv + 1) if i in seen]
        return suggestions[:top_k], eval_cp_white


# suggest_moves_fs_variant - wraps FairyStockfishVariantSuggest for existing callers
def suggest_moves_fs_variant(
    fen: str,
    game_type: str,
    top_k: int = 5,
    profile: str = "intermediate",
) -> Tuple[List[MoveSuggestion], Optional[int]]:
    from move_suggest import FairyStockfishVariantSuggest, MoveSuggestContext

    return FairyStockfishVariantSuggest().suggest_with_eval(
        MoveSuggestContext(
            fen=fen,
            color="white",
            top_k=top_k,
            profile=profile,
            game_type=game_type,
        )
    )


def _profile_to_uci_options(profile: str) -> tuple[dict, chess.engine.Limit]:
    """Map strength profile to Fairy-Stockfish UCI options and search limits."""
    p = (profile or "intermediate").lower()
    if p == "beginner":
        return {"Skill Level": 0}, chess.engine.Limit(depth=5, time=0.2)
    if p == "intermediate":
        return {"Skill Level": 5}, chess.engine.Limit(depth=8, time=0.4)
    if p == "advanced":
        return {"Skill Level": 15}, chess.engine.Limit(depth=12, time=0.8)
    if p == "master":
        return {"Skill Level": 20}, chess.engine.Limit(depth=18, time=1.5)
    # default
    return {"Skill Level": 5}, chess.engine.Limit(depth=8, time=0.4)


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


# _heuristic_suggest_impl - ranks chess legal moves with the heuristic evaluator
def _heuristic_suggest_impl(fen: str, color: str, top_k: int = 5) -> List[MoveSuggestion]:
    board = chess.Board(fen)
    target_color = parse_color(color)

    # analyze from requested player's perspective even if FEN turn differs
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


# _fs_suggest_impl - runs Fairy-Stockfish MultiPV for chess; raises on engine failure
def _fs_suggest_impl(
    fen: str,
    color: str,
    top_k: int = 5,
    profile: str = "intermediate",
) -> List[MoveSuggestion]:
    board = chess.Board(fen)
    target_color = parse_color(color)
    board.turn = target_color

    if board.is_game_over():
        return []

    engine = _get_engine()
    options, limit = _profile_to_uci_options(profile)
    engine.configure(options)

    multipv = max(1, min(top_k, 10))
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

    # pad with legal moves when MultiPV returns fewer than top_k (still FS path)
    if len(suggestions) < top_k:
        for move in list(board.legal_moves)[len(suggestions) : top_k]:
            san = board.san(move)
            suggestions.append(
                MoveSuggestion(rank=len(suggestions) + 1, uci=move.uci(), san=san, score=0)
            )

    return suggestions[:top_k]


# suggest_moves - wraps HeuristicSuggest for existing callers
def suggest_moves(fen: str, color: str, top_k: int = 5) -> List[MoveSuggestion]:
    from move_suggest import HeuristicSuggest, MoveSuggestContext

    return HeuristicSuggest().suggest(
        MoveSuggestContext(fen=fen, color=color, top_k=top_k, game_type="chess")
    )


# suggest_moves_fs - wraps select_suggestions (FS then Heuristic) for existing callers
def suggest_moves_fs(
    fen: str,
    color: str,
    top_k: int = 5,
    profile: str = "intermediate",
) -> List[MoveSuggestion]:
    from move_suggest import MoveSuggestContext, select_suggestions

    suggestions, _name = select_suggestions(
        MoveSuggestContext(
            fen=fen,
            color=color,
            top_k=top_k,
            profile=profile,
            game_type="chess",
        )
    )
    return suggestions


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


_PIECE_NAMES = {
    chess.PAWN: "pawn",
    chess.KNIGHT: "knight",
    chess.BISHOP: "bishop",
    chess.ROOK: "rook",
    chess.QUEEN: "queen",
    chess.KING: "king",
}


def normalize_history_uci(raw: str) -> str:
    """Strip session labels like 'White: e2e4' / 'Black: e7e5' → bare UCI."""
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


def _variant_board_grid(fen: str, *, files: int, ranks: int) -> Dict[Tuple[int, int], str]:
    """Map (file, rank) → piece kind from SFEN/FEN placement (post-move board)."""
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


def _variant_move_label(fen: str, target: str, gt: str) -> str:
    """Human label for xianqi/shogi UCI using post-move FEN (piece on destination)."""
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


def build_move_ground_truth(
    fen: str,
    move_uci: str,
    move_history: Optional[List[str]] = None,
    game_type: str = "chess",
) -> Dict[str, str]:
    """Deterministic facts about the move just played (chess via python-chess).

    Returns at least ``summary``; may include ``san`` / ``uci``. Never raises.
    """
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
            # History present but target not in it — replay all then fail closed.
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
            return {
                "summary": f"GROUND TRUTH: {target} is not legal in the replayed position — do not invent tactics.",
                "uci": target,
            }

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
    except Exception:
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

    # Dedupe while preserving order (threat can repeat material idea).
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


def build_explanation_fallback(
    fen: str,
    color: str,
    move_uci: str,
    move_san: str | None = None,
    game_type: str = "chess",
) -> str:
    move_text = move_san or move_uci
    gt = (game_type or "chess").strip().lower()
    if gt in {"xianqi", "xiangqi", "shogi"}:
        label = "Xiangqi" if gt in {"xianqi", "xiangqi"} else "Shogi"
        lab = _variant_move_label(fen, normalize_history_uci(move_uci or ""), gt) or move_text
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
    return (
        f"{move_text} keeps material {sign}. {threat} "
        "Next: check the suggested moves for replies, and watch checks, captures, and loose pieces."
    )


def _analyze_position_variant(
    fen: str,
    color: str,
    top_k: int,
    request_id: str | None,
    game_type: str,
    profile: str = "intermediate",
) -> Dict[str, object]:
    """Analyze xianqi/shogi via Fairy-Stockfish UCI — never chess.Board(fen)."""
    started_at = time.perf_counter()
    requested_color = parse_color(color)
    eval_cp_white = 0
    suggestions: List[MoveSuggestion] = []
    source = "fairy-stockfish"
    # Empty on success — a stub like "Position evaluated with Fairy-Stockfish."
    # sat between suggestions and coach and looked like FS endorsed the LLM.
    threat = ""

    try:
        suggestions, score = suggest_moves_fs_variant(fen, game_type, top_k, profile)
        if score is not None:
            eval_cp_white = score
    except Exception:
        # FS down / timeout: keep service up with empty suggestions.
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
        "game_type": "chess",
    }


def _phase_from_board(board: chess.Board) -> str:
    non_pawn_material = 0
    for piece_type in (chess.KNIGHT, chess.BISHOP, chess.ROOK, chess.QUEEN):
        non_pawn_material += len(board.pieces(piece_type, chess.WHITE))
        non_pawn_material += len(board.pieces(piece_type, chess.BLACK))

    if board.fullmove_number <= 10:
        return "opening"
    if non_pawn_material <= 6:
        return "endgame"
    return "middlegame"


# _history_agent_impl - builds history-agent features and tags for a chess position
def _history_agent_impl(
    fen: str,
    color: str,
    move_history: List[str] | None = None,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    started_at = time.perf_counter()
    board = chess.Board(fen)
    requested_color = parse_color(color)
    move_history = move_history or []
    _ = profile

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
    if _phase_from_board(board) == "opening":
        tags.append("book_like")

    latency_ms = int((time.perf_counter() - started_at) * 1000)
    return {
        "request_id": request_id or str(uuid.uuid4()),
        "status": "ok",
        "source": "rule_based_v1",
        "phase": _phase_from_board(board),
        "features": features,
        "tags": tags,
        "latency_ms": latency_ms,
    }


# _policy_agent_impl - builds policy-agent candidates from FS→Heuristic suggestions
def _policy_agent_impl(
    fen: str,
    color: str,
    top_k: int = 5,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    started_at = time.perf_counter()

    # profile controls Fairy-Stockfish UCI options inside select_suggestions
    suggestions = suggest_moves_fs(fen, color, top_k, profile)

    if not suggestions:
        candidates = []
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
        "request_id": request_id or str(uuid.uuid4()),
        "status": "ok",
        "source": "fairy-stockfish",
        "best_move_uci": best_move_uci,
        "candidates": candidates,
        "latency_ms": latency_ms,
    }


# _value_agent_impl - builds value-agent score and win-chance for a chess position
def _value_agent_impl(
    fen: str,
    color: str,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    started_at = time.perf_counter()
    board = chess.Board(fen)
    _ = parse_color(color)  # validated for consistency with shared API contract
    _ = profile
    score_cp = evaluate_position(board, chess.WHITE)
    value = math.tanh(score_cp / 400.0)
    win_chance_white = cp_to_win_chance(score_cp)
    win_chance_black = 1.0 - win_chance_white
    latency_ms = int((time.perf_counter() - started_at) * 1000)

    return {
        "request_id": request_id or str(uuid.uuid4()),
        "status": "ok",
        "source": "heuristic",
        "score_cp": int(score_cp),
        "mate_in": 0,
        "value": round(float(value), 6),
        "win_chance_white": round(float(win_chance_white), 6),
        "win_chance_black": round(float(win_chance_black), 6),
        "latency_ms": latency_ms,
    }


# build_history_payload - wraps HistoryAgent for existing callers
def build_history_payload(
    fen: str,
    color: str,
    move_history: List[str] | None = None,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    from agents import AgentContext, HistoryAgent

    return HistoryAgent().run(
        AgentContext(
            fen=fen,
            color=color,
            move_history=move_history,
            request_id=request_id,
            profile=profile,
        )
    )


# build_policy_payload - wraps PolicyAgent for existing callers
def build_policy_payload(
    fen: str,
    color: str,
    top_k: int = 5,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    from agents import AgentContext, PolicyAgent

    return PolicyAgent().run(
        AgentContext(
            fen=fen,
            color=color,
            top_k=top_k,
            request_id=request_id,
            profile=profile,
        )
    )


# build_value_payload - wraps ValueAgent for existing callers
def build_value_payload(
    fen: str,
    color: str,
    request_id: str | None = None,
    profile: str = "intermediate",
) -> Dict[str, object]:
    from agents import AgentContext, ValueAgent

    return ValueAgent().run(
        AgentContext(
            fen=fen,
            color=color,
            request_id=request_id,
            profile=profile,
        )
    )


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
