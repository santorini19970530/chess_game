#!/usr/bin/env python3
"""
Persistent Python analyzer service.

Endpoints:
  - GET /health
  - POST /analyze
"""

from __future__ import annotations

import os
from datetime import datetime, timezone

from flask import Flask, jsonify, request

from analyzer import analyze_position


app = Flask(__name__)


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


def main() -> None:
    host = os.getenv("PY_ANALYSER_HOST", "127.0.0.1")
    port = int(os.getenv("PY_ANALYSER_PORT", "8001"))
    debug = os.getenv("PY_ANALYSER_DEBUG", "0") == "1"
    app.run(host=host, port=port, debug=debug)


if __name__ == "__main__":
    main()
