#!/usr/bin/env python3
# fs_engine.py - fairy-stockfish process helpers (chess engine + raw uci for variants)

from __future__ import annotations

import os
import subprocess
import threading
import time
from typing import List, Optional, Tuple

import chess
import chess.engine

# fairy-stockfish binary path (override via environment variable)
FS_BINARY_PATH: str = os.environ.get(
    "FAIRY_STOCKFISH_PATH",
    os.path.join(os.path.dirname(__file__), "Fairy-Stockfish-fairy_sf_14", "src", "stockfish"),
)

# session game_type → fairy-stockfish UCI_Variant name
_GAME_TYPE_TO_UCI_VARIANT = {
    "chess": "chess",
    "xianqi": "xiangqi",
    "shogi": "shogi",
}

_engine: Optional[chess.engine.SimpleEngine] = None
raw_uci_lock = threading.Lock()
_raw_uci_proc: Optional[subprocess.Popen] = None
_raw_uci_variant: Optional[str] = None


# get_engine - returns a singleton fairy-stockfish engine instance (opened once)
def get_engine() -> chess.engine.SimpleEngine:
    global _engine
    if _engine is None:
        if not os.path.exists(FS_BINARY_PATH):
            raise FileNotFoundError(
                f"Fairy-Stockfish binary not found at {FS_BINARY_PATH}. "
                "Set FAIRY_STOCKFISH_PATH environment variable to the correct path."
            )
        _engine = chess.engine.SimpleEngine.popen_uci(FS_BINARY_PATH)
    return _engine


# uci_variant_name - maps session game_type to a fairy-stockfish UCI_Variant name
def uci_variant_name(game_type: str) -> str:
    key = (game_type or "chess").strip().lower()
    return _GAME_TYPE_TO_UCI_VARIANT.get(key, key)


# raw_uci_ensure - returns a singleton raw uci process for variant fens (chess.Board is chess-only)
def raw_uci_ensure(variant: str) -> subprocess.Popen:
    global _raw_uci_proc, _raw_uci_variant
    if _raw_uci_proc is not None and _raw_uci_proc.poll() is None:
        if _raw_uci_variant != variant:
            raw_uci_write(_raw_uci_proc, f"setoption name UCI_Variant value {variant}")
            raw_uci_write(_raw_uci_proc, "isready")
            raw_uci_wait_for(_raw_uci_proc, "readyok", timeout=5.0)
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
    raw_uci_write(proc, "uci")
    raw_uci_wait_for(proc, "uciok", timeout=5.0)
    raw_uci_write(proc, f"setoption name UCI_Variant value {variant}")
    raw_uci_write(proc, "isready")
    raw_uci_wait_for(proc, "readyok", timeout=5.0)
    _raw_uci_proc = proc
    _raw_uci_variant = variant
    return proc


# raw_uci_write - writes one line to a raw uci process stdin
def raw_uci_write(proc: subprocess.Popen, line: str) -> None:
    assert proc.stdin is not None
    proc.stdin.write(line + "\n")
    proc.stdin.flush()


# raw_uci_wait_for - reads stdout until token appears or timeout
def raw_uci_wait_for(proc: subprocess.Popen, token: str, timeout: float) -> None:
    assert proc.stdout is not None
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        line = proc.stdout.readline()
        if not line:
            raise RuntimeError("Fairy-Stockfish exited while waiting for " + token)
        if token in line:
            return
    raise TimeoutError(f"timeout waiting for {token}")


# parse_info_score_cp - extracts a centipawn or mate score from a uci info field list
def parse_info_score_cp(fields: List[str]) -> Optional[int]:
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


# parse_info_multipv_pv - extracts multipv index and first pv move from a uci info line
def parse_info_multipv_pv(fields: List[str]) -> Tuple[int, Optional[str]]:
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


# uci_score_as_white - converts a side-to-move uci score into white-pov centipawns
def uci_score_as_white(score_cp: int, fen: str) -> int:
    parts = fen.split()
    if len(parts) >= 2 and parts[1].lower() == "b":
        return -int(score_cp)
    return int(score_cp)


# profile_to_uci_options - maps strength profile to fairy-stockfish options and search limits
def profile_to_uci_options(profile: str) -> tuple[dict, chess.engine.Limit]:
    p = (profile or "intermediate").lower()
    if p == "beginner":
        return {"Skill Level": 0}, chess.engine.Limit(depth=5, time=0.2)
    if p == "intermediate":
        return {"Skill Level": 5}, chess.engine.Limit(depth=8, time=0.4)
    if p == "advanced":
        return {"Skill Level": 15}, chess.engine.Limit(depth=12, time=0.8)
    if p == "master":
        return {"Skill Level": 20}, chess.engine.Limit(depth=18, time=1.5)
    return {"Skill Level": 5}, chess.engine.Limit(depth=8, time=0.4)
