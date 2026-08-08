#!/usr/bin/env python3
# fen_from_image.py - wraps Chess_diagram_to_FEN get_fen for on-demand diagram import

from __future__ import annotations

import ctypes.util
import io
import os
import sys
from pathlib import Path
from typing import Any

from PIL import Image

# _resolve_vendor_dir - prefers env, then sibling _local_ outside the git tree, then in-tree clone
def _resolve_vendor_dir() -> Path:
    env = os.environ.get("CHESS_DIAGRAM_TO_FEN_DIR", "").strip()
    if env:
        return Path(env).expanduser().resolve()
    here = Path(__file__).resolve().parent
    candidates = (
        here.parent.parent / "_local_Chess_diagram_to_FEN",
        here / "Chess_diagram_to_FEN",
    )
    for path in candidates:
        if path.is_dir():
            return path
    return candidates[0]


_VENDOR_DIR = _resolve_vendor_dir()
_CAIRO_CANDIDATES = (
    Path("/opt/homebrew/opt/cairo/lib/libcairo.2.dylib"),
    Path("/usr/local/opt/cairo/lib/libcairo.2.dylib"),
)
_CAIRO_FIND_NAMES = frozenset({"cairo", "cairo-2", "libcairo-2", "libcairo.2"})
_cairo_ready = False
_orig_find_library = ctypes.util.find_library

# analyser API uses xianqi; upstream Chess_diagram_to_FEN uses xiangqi
_GAME_ALIASES = {
    "chess": "chess",
    "xianqi": "xiangqi",
    "xiangqi": "xiangqi",
    "shogi": "shogi",
}

SUPPORTED_DIAGRAM_GAMES = frozenset(_GAME_ALIASES)
_LIMITS_NOTE = (
    "Chess strongest; Xiangqi OK; Shogi weaker / no pieces-in-hand"
)


class FenFromImageError(Exception):
    def __init__(self, message: str, error_kind: str = "validation") -> None:
        super().__init__(message)
        self.error_kind = error_kind


# resolve_diagram_game - maps analyser game_type aliases to Chess_diagram_to_FEN keys
def resolve_diagram_game(raw: str) -> str:
    key = (raw or "").strip().lower()
    if key not in _GAME_ALIASES:
        raise FenFromImageError(
            f'Unsupported game: {raw!r}. Use "chess", "xianqi", or "shogi"'
        )
    return _GAME_ALIASES[key]


# _ensure_vendor_importable - puts vendor root on sys.path once for get_fen import
def _ensure_vendor_importable() -> None:
    vendor = str(_VENDOR_DIR)
    if not _VENDOR_DIR.is_dir():
        raise FenFromImageError(
            f"Chess_diagram_to_FEN vendor missing at {_VENDOR_DIR}",
            "internal",
        )
    if vendor not in sys.path:
        sys.path.insert(0, vendor)


# _find_library - returns homebrew libcairo for cairocffi names; else defers to ctypes
def _find_library(name: str) -> str | None:
    if name in _CAIRO_FIND_NAMES:
        for path in _CAIRO_CANDIDATES:
            if path.is_file():
                return str(path)
    return _orig_find_library(name)


# _ensure_cairo_library - points cairocffi at homebrew libcairo (conda shells miss it)
def _ensure_cairo_library() -> None:
    global _cairo_ready
    if _cairo_ready:
        return

    if not any(p.is_file() for p in _CAIRO_CANDIDATES):
        raise FenFromImageError(
            "libcairo not found; install with: brew install cairo",
            "internal",
        )

    ctypes.util.find_library = _find_library
    _cairo_ready = True


# fen_from_image_bytes - runs diagram recognition and returns fen fields
def fen_from_image_bytes(image_bytes: bytes, game: str) -> dict[str, Any]:
    if not image_bytes:
        raise FenFromImageError("Empty image upload")

    resolved = resolve_diagram_game(game)

    try:
        img = Image.open(io.BytesIO(image_bytes))
        img.load()
    except Exception as exc:
        raise FenFromImageError(f"Invalid image: {exc}") from exc

    _ensure_vendor_importable()
    _ensure_cairo_library()
    # vendor code resolves some paths relative to cwd
    prev = os.getcwd()
    try:
        os.chdir(_VENDOR_DIR)
        # import here so /analyze startup does not load torch/cairo
        from chess_diagram_to_fen import get_fen

        result = get_fen(
            img=img,
            game=resolved,
            auto_rotate_image=True,
            auto_rotate_board=True,
        )
    except FenFromImageError:
        raise
    except FileNotFoundError as exc:
        raise FenFromImageError(str(exc), "internal") from exc
    except ValueError as exc:
        raise FenFromImageError(str(exc)) from exc
    except Exception as exc:
        raise FenFromImageError(
            f"Diagram recognition failed: {exc}",
            "internal",
        ) from exc
    finally:
        os.chdir(prev)

    if result is None:
        raise FenFromImageError("No board detected in image", "recognition")

    fen = getattr(result, "fen", None)
    if not fen or not str(fen).strip():
        raise FenFromImageError(
            "Board detected but FEN could not be recovered",
            "recognition",
        )

    return {
        "fen": str(fen).strip(),
        "game": resolved,
        "board_is_flipped": getattr(result, "board_is_flipped", None),
        "image_rotation_angle": getattr(result, "image_rotation_angle", None),
        "limits_note": _LIMITS_NOTE,
    }
