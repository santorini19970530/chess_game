#!/usr/bin/env python3
# server.py - flask http edge for analyze, explain, play agents, and diagram fen

from __future__ import annotations

import json
import os
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from flask import Flask, jsonify, request

from agents import AgentContext, run_agent
from analyzer import (
    analyze_position,
    build_concept_hints,
)
from fen_from_image import FenFromImageError, fen_from_image_bytes
from llm_providers import get_llm_provider


app = Flask(__name__)


# History/Policy/Value stay chess-only. Coach (/analyze, /explain) accepts variants.
# /fen_from_image is on-demand vision only — never called from the live /move path.
SUPPORTED_GAME_TYPES = {"chess"}
COACH_GAME_TYPES = {"chess", "xianqi", "shogi"}
SKILL_LEVELS = frozenset({"beginner", "intermediate", "advanced"})
DEFAULT_SKILL_LEVEL = "intermediate"
MAX_CONCEPT_HINTS = 3
_EXPLAIN_LOG_DIR = Path(__file__).resolve().parent / "data" / "explain_logs"
_MAX_DIAGRAM_BYTES = 12 * 1024 * 1024


# _append_explain_log - appends one json line for qa evidence; set EXPLAIN_LOG=0 to disable
def _append_explain_log(entry: dict[str, Any]) -> None:
    if os.getenv("EXPLAIN_LOG", "1").strip() == "0":
        return
    try:
        _EXPLAIN_LOG_DIR.mkdir(parents=True, exist_ok=True)
        path = _EXPLAIN_LOG_DIR / "explain.jsonl"
        with path.open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(entry, ensure_ascii=False) + "\n")
    except OSError:
        pass


# _parse_skill_level - maps missing or invalid skill_level to intermediate for old clients
def _parse_skill_level(payload: dict[str, Any]) -> str:
    raw = payload.get("skill_level", None)
    if raw is None or str(raw).strip() == "":
        return DEFAULT_SKILL_LEVEL
    level = str(raw).strip().lower()
    if level in SKILL_LEVELS:
        return level
    return DEFAULT_SKILL_LEVEL


# _parse_concept_hints - pulls up to three concept hints from payload or analysis
def _parse_concept_hints(payload: dict[str, Any]) -> list[str]:
    raw = payload.get("concept_hints", None)
    if isinstance(raw, list):
        out: list[str] = []
        for item in raw:
            text = str(item).strip()
            if not text:
                continue
            out.append(text)
            if len(out) >= MAX_CONCEPT_HINTS:
                break
        return out
    if isinstance(raw, str) and raw.strip():
        return [raw.strip()]

    analysis = payload.get("analysis")
    if isinstance(analysis, dict):
        return build_concept_hints(analysis, max_hints=MAX_CONCEPT_HINTS)
    return []


# _error_response - builds a standard json error body and http status tuple
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


# _parse_common_payload - validates shared fen/color/game_type fields for agent and coach routes
def _parse_common_payload(
    payload: dict[str, Any],
    allowed_game_types: set[str] | None = None,
) -> tuple[dict[str, Any] | None, tuple | None]:
    request_id = str(payload.get("request_id", "")).strip() or None
    fen = str(payload.get("fen", "")).strip()
    color = str(payload.get("color", "")).strip().lower()
    game_type = str(payload.get("game_type", "")).strip().lower()
    variant = str(payload.get("variant", "")).strip().lower() or game_type
    move_number_raw = payload.get("move_number", 0)
    move_history_raw = payload.get("move_history", [])
    profile = str(payload.get("profile", "")).strip().lower() or "intermediate"
    allowed = allowed_game_types if allowed_game_types is not None else SUPPORTED_GAME_TYPES

    if not fen:
        return None, _error_response(request_id, 'Missing required field: "fen"')
    if color not in {"white", "black", "w", "b"}:
        return None, _error_response(request_id, 'Invalid "color". Use "white" or "black".')
    if not game_type:
        return None, _error_response(request_id, 'Missing required field: "game_type"')
    if game_type not in allowed:
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


# _extract_move_fields - reads move_uci/move_san or returns a validation error response
def _extract_move_fields(payload: dict[str, Any]) -> tuple[str | None, str | None, tuple | None]:
    move_uci = str(payload.get("move_uci", "")).strip() or None
    move_san = str(payload.get("move_san", "")).strip() or None
    if not move_uci and not move_san:
        rid = str(payload.get("request_id", "")).strip() or None
        return None, None, _error_response(rid, 'Missing required field: "move_uci" or "move_san"')
    return move_uci, move_san, None


# health - returns a simple ok status payload for liveness checks
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


# fen_from_image - accepts a board diagram image and returns a FEN-like string
@app.post("/fen_from_image")
def fen_from_image() -> tuple:
    request_id = (
        str(request.form.get("request_id", "")).strip()
        or str(request.args.get("request_id", "")).strip()
        or None
    )
    game_raw = (
        request.form.get("game")
        or request.form.get("type")
        or request.form.get("game_type")
        or ""
    )
    upload = request.files.get("image") or request.files.get("file")
    if upload is None:
        return _error_response(
            request_id,
            'Missing multipart file field: "image"',
        )
    if not str(game_raw).strip():
        return _error_response(
            request_id,
            'Missing required field: "game" (chess / xianqi / shogi)',
        )

    image_bytes = upload.read(_MAX_DIAGRAM_BYTES + 1)
    if len(image_bytes) > _MAX_DIAGRAM_BYTES:
        return _error_response(
            request_id,
            f"Image too large (max {_MAX_DIAGRAM_BYTES} bytes)",
        )

    try:
        result = fen_from_image_bytes(image_bytes, str(game_raw))
    except FenFromImageError as exc:
        status = 400 if exc.error_kind in {"validation", "recognition"} else 500
        return _error_response(request_id, str(exc), exc.error_kind, status)
    except Exception:
        return _error_response(
            request_id,
            "Internal diagram recognition error",
            "internal",
            500,
        )

    return (
        jsonify(
            {
                "request_id": request_id,
                "status": "ok",
                "fen": result["fen"],
                "game": result["game"],
                "board_is_flipped": result.get("board_is_flipped"),
                "image_rotation_angle": result.get("image_rotation_angle"),
                "limits_note": result.get("limits_note"),
            }
        ),
        200,
    )


# analyze - runs position analysis and returns the shared coach schema
@app.post("/analyze")
def analyze() -> tuple:
    payload = request.get_json(silent=True) or {}
    request_id = payload.get("request_id")
    fen = str(payload.get("fen", "")).strip()
    color = str(payload.get("color", "")).strip().lower()
    top_k = payload.get("top_k", 5)
    game_type = str(payload.get("game_type", "chess")).strip().lower() or "chess"
    profile = str(payload.get("profile", "intermediate")).strip().lower() or "intermediate"

    if not fen:
        return jsonify({"error": 'Missing required field: "fen"'}), 400
    if color not in {"white", "black", "w", "b"}:
        return jsonify({"error": 'Invalid "color". Use "white" or "black".'}), 400
    if game_type not in {"chess", "xianqi", "shogi"}:
        return jsonify({"error": f'Unsupported "game_type": {game_type}'}), 400

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
            game_type=game_type,
            profile=profile,
        )
    except ValueError as exc:
        # covers invalid fen / color parser errors from analyzer
        return jsonify({"error": str(exc)}), 400
    except Exception:
        return jsonify({"error": "Internal analyzer error"}), 500

    return jsonify(result), 200


# history - runs the history AgentStrategy and returns its payload
@app.post("/history")
def history() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    try:
        result = run_agent(
            "history",
            AgentContext(
                fen=common["fen"],
                color=common["color"],
                move_history=common["move_history"],
                request_id=common["request_id"],
                profile=common.get("profile", "intermediate"),
            ),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


# policy - runs the policy AgentStrategy and returns candidate moves
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
        result = run_agent(
            "policy",
            AgentContext(
                fen=common["fen"],
                color=common["color"],
                top_k=top_k_value,
                request_id=common["request_id"],
                profile=common.get("profile", "intermediate"),
            ),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


# value - runs the value AgentStrategy and returns score fields
@app.post("/value")
def value() -> tuple:
    payload = request.get_json(silent=True) or {}
    common, err = _parse_common_payload(payload)
    if err is not None:
        return err

    assert common is not None
    try:
        result = run_agent(
            "value",
            AgentContext(
                fen=common["fen"],
                color=common["color"],
                request_id=common["request_id"],
                profile=common.get("profile", "intermediate"),
            ),
        )
    except ValueError as exc:
        return _error_response(common["request_id"], str(exc), "validation", 400)
    except Exception:
        return _error_response(common["request_id"], "Internal analyzer error", "internal", 500)
    return jsonify(result), 200


# explain - returns coach text via LLMProvider with finalize/sanitize applied
@app.post("/explain")
def explain() -> tuple:
    payload = dict(request.get_json(silent=True) or {})
    # optional `game` alias + default chess for terms/tone file selection
    if not str(payload.get("game_type", "")).strip():
        game_alias = str(payload.get("game", "")).strip().lower()
        payload["game_type"] = game_alias or "chess"

    common, err = _parse_common_payload(payload, allowed_game_types=COACH_GAME_TYPES)
    if err is not None:
        return err

    assert common is not None
    move_uci, move_san, merr = _extract_move_fields(payload)
    if merr is not None:
        return merr

    skill_level = _parse_skill_level(payload)
    concept_hints = _parse_concept_hints(payload)
    human_color = str(payload.get("human_color", "")).strip().lower() or None
    if human_color not in {"white", "black", "w", "b"}:
        human_color = None
    elif human_color in {"w", "b"}:
        human_color = "white" if human_color == "w" else "black"
    started_at = time.perf_counter()
    provider = get_llm_provider()
    history = common.get("move_history", [])
    game_type = common["game_type"]

    from analyzer import build_move_ground_truth
    from explain_finalize import finalize_explanation
    from teacher_prompt import looks_like_uci, side_to_move_from_fen

    ground = build_move_ground_truth(
        fen=common["fen"],
        move_uci=move_uci or "",
        move_history=history,
        game_type=game_type,
    )
    # prefer board-derived san; go often sends uci in the san field
    if ground.get("san"):
        move_san = ground["san"]
    elif move_san and looks_like_uci(move_san):
        # keep a label so finalize never skips filters on tip-FEN explains
        move_san = ground.get("uci") or move_uci or None

    quick = bool(payload.get("quick")) or str(payload.get("mode", "")).strip().lower() == "quick"
    to_move = side_to_move_from_fen(common["fen"], common["color"])
    last_mover = "black" if to_move == "white" else "white"

    if quick:
        from teacher_prompt import build_quick_coach_line

        explanation = build_quick_coach_line(
            fen=common["fen"],
            move_uci=move_uci or "",
            move_san=move_san,
            move_history=history,
            game_type=game_type,
            human_color=human_color,
            side_to_move=common["color"],
        )
        source = "quick"
    else:
        source = getattr(provider, "name", "ollama")
        explanation = ""
        try:
            explanation = provider.explain(
                fen=common["fen"],
                color=common["color"],
                move_uci=move_uci or "",
                move_san=move_san,
                move_history=history,
                game_type=game_type,
                skill_level=skill_level,
                human_color=human_color,
                concept_hints=concept_hints or None,
            )
        except Exception:
            # any failure (ollama down, timeout, bad response) falls back to heuristic
            from analyzer import build_explanation_fallback as _fallback

            explanation = _fallback(
                fen=common["fen"],
                color=common["color"],
                move_uci=move_uci or "",
                move_san=move_san,
                game_type=game_type,
            )
            source = "heuristic_fallback"
        explanation = finalize_explanation(
            explanation,
            move_san=move_san,
            last_mover=last_mover,
            human_color=human_color,
            ground_summary=str(ground.get("summary") or ""),
            concept_hints=concept_hints or None,
        )
    latency_ms = int((time.perf_counter() - started_at) * 1000)
    request_id = common["request_id"] or uuid.uuid4().hex
    _append_explain_log(
        {
            "timestamp_utc": datetime.now(timezone.utc).isoformat(),
            "request_id": request_id,
            "source": source,
            "skill_level": skill_level,
            "game_type": game_type,
            "fen": common["fen"],
            "color": common["color"],
            "move_uci": move_uci,
            "move_san": move_san,
            "move_history": history[-8:] if isinstance(history, list) else [],
            "concept_hints": concept_hints,
            "explanation": explanation,
            "latency_ms": latency_ms,
            "human_color": human_color,
        }
    )

    return (
        jsonify(
            {
                "request_id": request_id,
                "status": "ok",
                "source": source,
                "explanation": explanation,
                "move_uci": move_uci,
                "move_san": move_san,
                "latency_ms": latency_ms,
                "game_type": game_type,
                "skill_level": skill_level,
                "concept_hints": concept_hints,
            }
        ),
        200,
    )

# main - starts the flask analyser service from env host/port/debug
def main() -> None:
    host = os.getenv("PY_ANALYSER_HOST", "127.0.0.1")
    port = int(os.getenv("PY_ANALYSER_PORT", "8001"))
    debug = os.getenv("PY_ANALYSER_DEBUG", "0") == "1"
    app.run(host=host, port=port, debug=debug)

if __name__ == "__main__":
    main()
elif os.getenv("PY_EXPLAIN_SELFCHECK"):
    # one tiny runnable check that coach + diagram routes are registered
    rules = {r.rule for r in app.url_map.iter_rules()}
    assert "/explain" in rules
    assert "/fen_from_image" in rules