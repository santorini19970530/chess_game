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

# Fairy-Stockfish 14 Chess default net (evaluate.h EvalFileDefaultName)
DEFAULT_NNUE_FILENAME = "nn-3475407dc199.nnue"

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


# resolve_nnue_path - existing .nnue path from FAIRY_STOCKFISH_NNUE_PATH or _local_nnue
def resolve_nnue_path() -> str:
    env = os.environ.get("FAIRY_STOCKFISH_NNUE_PATH", "").strip()
    if env:
        return env if os.path.isfile(env) else ""
    here = os.path.dirname(os.path.abspath(__file__))
    candidates = (
        os.path.join(here, "..", "..", "_local_nnue", DEFAULT_NNUE_FILENAME),
        os.path.join(here, "..", "_local_nnue", DEFAULT_NNUE_FILENAME),
        os.path.join(here, DEFAULT_NNUE_FILENAME),
    )
    for path in candidates:
        if os.path.isfile(path):
            return os.path.abspath(path)
    return ""


# nnue_uci_commands - shared Use NNUE + EvalFile lines for python-chess and raw UCI
def nnue_uci_commands(path: str | None = None) -> list[str]:
    resolved = path if path is not None else resolve_nnue_path()
    if not resolved:
        return []
    return [
        "setoption name Use NNUE value true",
        f"setoption name EvalFile value {resolved}",
    ]


# requested_nnue_path - FAIRY_STOCKFISH_NNUE_PATH as requested, even if the file is missing
def requested_nnue_path() -> str:
    return os.environ.get("FAIRY_STOCKFISH_NNUE_PATH", "").strip()


# nnue_evidence_check - fails when the requested net is unset, missing, or too small
def nnue_evidence_check() -> None:
    requested = requested_nnue_path()
    if not requested:
        raise RuntimeError(
            "evidence: FAIRY_STOCKFISH_NNUE_PATH unset; classical eval is not pretrained-model evidence"
        )
    if not os.path.isfile(requested):
        raise RuntimeError(f"evidence: NNUE file missing: {requested}")
    base = os.path.basename(requested)
    if not base.startswith("nn-") or not base.endswith(".nnue"):
        raise RuntimeError(
            f"evidence: NNUE file rejected (Chess net name must be nn-*.nnue): {base}"
        )
    size = os.path.getsize(requested)
    if size < 1_000_000:
        raise RuntimeError(f"evidence: NNUE file rejected (size {size} < 1000000): {requested}")


# apply_nnue_to_simple_engine - configure python-chess SimpleEngine from the shared helper
def apply_nnue_to_simple_engine(engine: chess.engine.SimpleEngine) -> bool:
    path = resolve_nnue_path()
    if not path:
        return False
    engine.configure({"Use NNUE": True, "EvalFile": path})
    return True


# classify_eval_mode - nnue if verify printed NNUE evaluation using, else classical
def classify_eval_mode(text: str) -> str:
    if "NNUE evaluation using" in text:
        return "nnue"
    if "classical evaluation enabled" in text:
        return "classical"
    return "unknown"


# nnue_eval_probe - one-shot UCI eval; returns verify() mode and collected lines
def nnue_eval_probe(variant: str = "chess", fen: str = "") -> tuple[str, list[str]]:
    if not os.path.exists(FS_BINARY_PATH):
        raise FileNotFoundError(f"Fairy-Stockfish binary not found at {FS_BINARY_PATH}")
    proc = subprocess.Popen(
        [FS_BINARY_PATH],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        bufsize=1,
    )
    try:
        raw_uci_write(proc, "uci")
        raw_uci_wait_for(proc, "uciok", timeout=5.0)
        apply_nnue_to_raw_uci(proc)
        if variant and variant != "chess":
            raw_uci_write(proc, f"setoption name UCI_Variant value {variant}")
            raw_uci_write(proc, "isready")
            raw_uci_wait_for(proc, "readyok", timeout=5.0)
        if fen.strip():
            raw_uci_write(proc, f"position fen {fen}")
        else:
            raw_uci_write(proc, "position startpos")
        raw_uci_write(proc, "eval")
        lines = raw_uci_collect_until(proc, "Final evaluation", timeout=8.0)
        return classify_eval_mode("\n".join(lines)), lines
    finally:
        try:
            raw_uci_write(proc, "quit")
        except Exception:
            pass
        proc.kill()
        proc.wait(timeout=2.0)


# raw_uci_collect_until - reads stdout lines until token appears or timeout
def raw_uci_collect_until(proc: subprocess.Popen, token: str, timeout: float) -> list[str]:
    assert proc.stdout is not None
    deadline = time.monotonic() + timeout
    lines: list[str] = []
    while time.monotonic() < deadline:
        line = proc.stdout.readline()
        if not line:
            raise RuntimeError("Fairy-Stockfish exited while waiting for " + token)
        text = line.strip()
        if not text:
            continue
        lines.append(text)
        if token in text:
            return lines
    raise TimeoutError(f"timeout waiting for {token}")


# apply_nnue_to_raw_uci - send the same UCI options to a raw process after uciok
def apply_nnue_to_raw_uci(proc: subprocess.Popen) -> bool:
    commands = nnue_uci_commands()
    if not commands:
        return False
    for line in commands:
        raw_uci_write(proc, line)
    raw_uci_write(proc, "isready")
    raw_uci_wait_for(proc, "readyok", timeout=15.0)
    return True


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
        apply_nnue_to_simple_engine(_engine)
        try:
            nnue_evidence_check()
        except RuntimeError as exc:
            print(f"warning: NNUE evidence check failed: {exc} (play may use classical eval)")
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
    apply_nnue_to_raw_uci(proc)
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
    if p == "master" or p == "nnue":
        return {"Skill Level": 20}, chess.engine.Limit(depth=18, time=1.5)
    return {"Skill Level": 5}, chess.engine.Limit(depth=8, time=0.4)
