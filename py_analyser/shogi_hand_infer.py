#!/usr/bin/env python3
# shogi_hand_infer.py - infers shogi hand pieces from board inventory after diagram import

from __future__ import annotations

# per-side starting counts (kings never enter hand)
_START = {"r": 1, "b": 1, "g": 2, "s": 2, "n": 2, "l": 2, "p": 9}
_ORDER = ("r", "b", "g", "s", "n", "l", "p")


# split_shogi_placement_and_hands - splits placement token into board path and hand text
def split_shogi_placement_and_hands(raw: str) -> tuple[str, str]:
    text = (raw or "").strip()
    start = text.find("[")
    if start < 0:
        return text, ""
    end = text.find("]", start)
    if end < 0:
        return text[:start], text[start + 1 :]
    return text[:start], text[start + 1 : end]


# _count_board_by_side - counts unpromoted piece types on the board for white and black
def _count_board_by_side(placement: str) -> tuple[dict[str, int], dict[str, int]]:
    white: dict[str, int] = {k: 0 for k in _START}
    black: dict[str, int] = {k: 0 for k in _START}
    i = 0
    n = len(placement)
    while i < n:
        ch = placement[i]
        if ch in "123456789/" or ch.isspace():
            i += 1
            continue
        if ch == "+":
            i += 1
            if i >= n:
                break
            ch = placement[i]
        kind = ch.lower()
        if kind in _START:
            if ch.isupper():
                white[kind] += 1
            elif ch.islower():
                black[kind] += 1
        i += 1
    return white, black


# _infer_hand_counts - deficit heuristic with repair so hands match missing totals
def _infer_hand_counts(
    white_board: dict[str, int], black_board: dict[str, int]
) -> tuple[dict[str, int], dict[str, int]]:
    white_hand = {k: 0 for k in _START}
    black_hand = {k: 0 for k in _START}
    for kind, start in _START.items():
        total = start * 2
        on_board = white_board.get(kind, 0) + black_board.get(kind, 0)
        missing = max(0, total - on_board)
        # black short vs start → white captured those; white short → black hand
        wh = max(0, start - black_board.get(kind, 0))
        bh = max(0, start - white_board.get(kind, 0))
        while wh + bh > missing:
            if wh >= bh and wh > 0:
                wh -= 1
            elif bh > 0:
                bh -= 1
            else:
                break
        while wh + bh < missing:
            wh += 1
        white_hand[kind] = wh
        black_hand[kind] = bh
    return white_hand, black_hand


# _format_hand_bracket - builds go-compatible hand text (repeated letters, white then black)
def _format_hand_bracket(white_hand: dict[str, int], black_hand: dict[str, int]) -> str:
    parts: list[str] = []
    for kind in _ORDER:
        parts.append(kind.upper() * white_hand.get(kind, 0))
    for kind in _ORDER:
        parts.append(kind.lower() * black_hand.get(kind, 0))
    return "".join(parts)


# apply_inferred_shogi_hands - rewrites shogi fen hand field from board material
def apply_inferred_shogi_hands(fen: str) -> str:
    text = (fen or "").strip()
    if not text:
        return text
    parts = text.split()
    placement, _old = split_shogi_placement_and_hands(parts[0])
    if placement.count("/") != 8:
        return text
    white_board, black_board = _count_board_by_side(placement)
    white_hand, black_hand = _infer_hand_counts(white_board, black_board)
    hand = _format_hand_bracket(white_hand, black_hand)
    parts[0] = f"{placement}[{hand}]"
    return " ".join(parts)


if __name__ == "__main__":
    start = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"
    assert apply_inferred_shogi_hands(start).startswith(
        "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[]"
    )
    one_pawn = "lnsgkgsnl/1r5b1/pppp1pppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"
    out = apply_inferred_shogi_hands(one_pawn)
    assert "[P]" in out.split()[0], out
    print("ok", out.split()[0])
