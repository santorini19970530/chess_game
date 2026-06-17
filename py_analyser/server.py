#!/usr/bin/env python3
"""
Persistent Python analyzer service.

Endpoints:
  - GET /health
  - POST /analyze
"""

from __future__ import annotations

import json
import os
import socket
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone
from typing import Any

from flask import Flask, jsonify, request

from analyzer import (
    analyze_position,
    build_explanation_fallback,
    build_history_payload,
    build_policy_payload,
    build_value_payload,
)


app = Flask(__name__)


SUPPORTED_GAME_TYPES = {"chess"}


def _error_response(
    request_id: str | None,
    message: str,
    error_kind: str = "validation",
    status_code: int = 400,
) -> tuple:
    return (
        jsonify(
            {
                "request_id": request_id,
                "status": "error",
                "error_kind": error_kind,
                "message": message,
            }
        ),
        status_code,
    )


def _parse_common_payload(payload: dict[str, Any]) -> tuple[dict[str, Any] | None, tuple | None]:
    request_id = str(payload.get("request_id", "")).strip() or None
    fen = str(payload.get("fen", "")).strip()
    color = str(payload.get("color", "")).strip().lower()
    game_type = str(payload.get("game_type", "")).strip().lower()
    variant = str(payload.get("variant", "")).strip().lower() or game_type
    move_number_raw = payload.get("move_number", 0)
    move_history_raw = payload.get("move_history", [])
    profile = str(payload.get("profile", "")).strip().lower() or "intermediate"

    if not fen:
        return None, _error_response(request_id, 'Missing required field: "fen"')
    if color not in {"white", "black", "w", "b"}:
        return None, _error_response(request_id, 'Invalid "color". Use "white" or "black".')
    if not game_type:
        return None, _error_response(request_id, 'Missing required field: "game_type"')
    if game_type not in SUPPORTED_GAME_TYPES:
        return None, _error_response(
            request_id, f'Unsupported "game_type": {game_type}', "validation", 400
        )
    if variant and variant != game_type:
        return None, _error_response(
            request_id,
            f'Unsupported "variant": {variant} for game_type "{game_type}"',
            "validation",
            400,
        )
    try:
        move_number = int(move_number_raw)
    except (TypeError, ValueError):
        return None, _error_response(request_id, '"move_number" must be an integer.')

    if not isinstance(move_history_raw, list):
        return None, _error_response(request_id, '"move_history" must be an array of strings.')
    move_history = [str(item) for item in move_history_raw]

    return (
        {
            "request_id": request_id,
            "fen": fen,
            "color": color,
            "game_type": game_type,
            "variant": variant or game_type,
            "move_number": move_number,
            "move_history": move_history,
            "profile": profile,
        },
        None,
    )


def _extract_move_fields(payload: dict[str, Any]) -> tuple[str | None, str | None, tuple | None]:
    move_uci = str(payload.get("move_uci", "")).strip() or None
    move_san = str(payload.get("move_san", "")).strip() or None
    if not move_uci and not move_san:
        rid = str(payload.get("request_id", "")).strip() or None
        return None, None, _error_response(rid, 'Missing required field: "move_uci" or "move_san"')
    return move_uci, move_san, None


@app.get("/health")
def health() -> tuple:
    return (
        jsonify(
            {
                "status": "ok",
                "service": "py_analyser",
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
        ),
        200,
    )


@app.post("/analyze")
def analyze() -> tuple:
    payload = request.get_json(silent=True) or {}
    request_id = payload.get("request_id")
    fen = str(payload.get("fen", "")).strip()
    color = str(payload.get("color", "")).strip().lower()
    top_k = payload.get("top_k", 5)

    if not fen:
        return jsonify({"error": 'Missing required field: "fen"'}), 400
    if color not in {"white", "black", "w", "b"}:
        return jsonify({"error": 'Invalid "color". Use "white" or "black".'}), 400

    try:
        top_k_value = int(top_k)
    except (TypeError, ValueError):
        return jsonify({"error": '"top_k" must be an integer.'}), 400

    try:
        result = analyze_position(
            fen=fen,
            color=color,
            top_k=top_k_value,
            request_id=str(request_id) if request_id else None,
        )
    except ValueError as exc:
        # Covers invalid FEN / color parser errors from analyzer.
        return jsonify({"error": str(exc)}), 400
    except Exception:
        return jsonify({"error": "Internal analyzer error"}), 500

    return jsonify(result), 200


@app.post("/history")
def history() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    try:
        result = build_history_payload(
            fen=common["fen"],
            color=common["color"],
            move_history=common["move_history"],
            request_id=common["request_id"],
            profile=common.get("profile", "intermediate"),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


@app.post("/policy")
def policy() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    top_k = payload.get("top_k", 5)
    try:
        top_k_value = int(top_k)
    except (TypeError, ValueError):
        return _error_response(common["request_id"], '"top_k" must be an integer.')
    top_k_value = min(20, max(1, top_k_value))

    try:
        result = build_policy_payload(
            fen=common["fen"],
            color=common["color"],
            top_k=top_k_value,
            request_id=common["request_id"],
            profile=common.get("profile", "intermediate"),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


@app.post("/value")
def value() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    try:
        result = build_value_payload(
            fen=common["fen"],
            color=common["color"],
            request_id=common["request_id"],
            profile=common.get("profile", "intermediate"),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


@app.post("/explain")
def explain() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    move_uci, move_san, merr = _extract_move_fields(payload)
    if merr is not None:
        return merr

    started_at = time.perf_counter()
    ollama_url = "http://localhost:11434/api/generate"
    model = os.getenv("OLLAMA_MODEL", "gemma2:2b")
    move_text = move_san or move_uci or ""
    history = common.get("move_history", [])
    history_str = " ".join(history[-6:]) if history else "(no prior moves)"

    prompt = (
        f"You are a calm chess coach for an intermediate club player. "
        f"Explain the move {move_text} in 2-4 short sentences. "
        f"FEN: {common['fen']}. Recent moves: {history_str}. "
        f"Mention whether it creates threats, changes material balance, or improves safety. "
        f"Keep the tone educational and encouraging; avoid engine jargon."
    )

    source = "ollama"
    explanation = ""
    try:
        req = urllib.request.Request(
            ollama_url,
            data=json.dumps({"model": model, "prompt": prompt, "stream": False}).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=9) as resp:
            if resp.status != 200:
                raise urllib.error.HTTPError(ollama_url, resp.status, "bad status", {}, None)
            data = json.loads(resp.read().decode("utf-8"))
            explanation = (data.get("response") or "").strip()
            if not explanation:
                raise ValueError("empty response from ollama")
    except (urllib.error.URLError, socket.timeout, TimeoutError, json.JSONDecodeError, ValueError):
        explanation = build_explanation_fallback(
            fen=common["fen"],
            color=common["color"],
            move_uci=move_uci or "",
            move_san=move_san,
        )
        source = "heuristic_fallback"
    except Exception:
        # Any unexpected error also falls back; never crash the endpoint.
        explanation = build_explanation_fallback(
            fen=common["fen"],
            color=common["color"],
            move_uci=move_uci or "",
            move_san=move_san,
        )
        source = "heuristic_fallback"
    latency_ms = int((time.perf_counter() - started_at) * 1000)

    return (
        jsonify(
            {
                "request_id": common["request_id"] or uuid.uuid4().hex,
                "status": "ok",
                "source": source,
                "explanation": explanation,
                "move_uci": move_uci,
                "move_san": move_san,
                "latency_ms": latency_ms,
            }
        ),
        200,
    )


def main() -> None:
    host = os.getenv("PY_ANALYSER_HOST", "127.0.0.1")
    port = int(os.getenv("PY_ANALYSER_PORT", "8001"))
    debug = os.getenv("PY_ANALYSER_DEBUG", "0") == "1"
    app.run(host=host, port=port, debug=debug)


if __name__ == "__main__":
    main()
elif os.getenv("PY_EXPLAIN_SELFCHECK"):
    # ponytail: one tiny runnable check that the /explain route is registered
    assert any(r.rule == "/explain" for r in app.url_map.iter_rules())
