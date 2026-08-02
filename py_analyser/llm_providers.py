#!/usr/bin/env python3
# llm_providers.py - LLMProvider Strategy (Ollama / Heuristic)

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from typing import Any, Protocol

from analyzer import build_explanation_fallback
from explain_finalize import sanitize_explanation
from teacher_prompt import build_teacher_prompt


# LLMProvider - protocol for coach explanation providers
class LLMProvider(Protocol):
    name: str

    # explain - returns coach text for the last move in context
    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        raise NotImplementedError


# OllamaProvider - calls a local ollama instance for coach text
class OllamaProvider:
    name = "ollama"

    # __init__ - configures model, timeout, and generate url from env defaults
    def __init__(self, model: str | None = None, timeout: float | None = None) -> None:
        self.model = model or os.getenv("OLLAMA_MODEL", "gemma2:2b")
        timeout_ms_raw = os.getenv("OLLAMA_TIMEOUT_MS", "15000").strip()
        try:
            timeout_ms = max(1000, int(timeout_ms_raw))
        except ValueError:
            timeout_ms = 15000
        self.timeout = timeout if timeout is not None else (timeout_ms / 1000.0)
        self.url = os.getenv("OLLAMA_URL", "http://localhost:11434/api/generate").strip()

    # explain - generates coach text via ollama and sanitizes the response
    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        prompt = build_teacher_prompt(
            fen=fen,
            move_uci=move_uci,
            move_san=move_san,
            move_history=move_history,
            game_type=game_type,
            skill_level=skill_level,
            side_to_move=color,
            human_color=human_color,
            concept_hints=concept_hints,
        )
        body = self._request_body(prompt)
        req = urllib.request.Request(
            self.url,
            data=json.dumps(body).encode("utf-8"),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=self.timeout) as resp:
            if resp.status != 200:
                raise urllib.error.HTTPError(self.url, resp.status, "bad status", {}, None)
            data = json.loads(resp.read().decode("utf-8"))
            text = (data.get("response") or "").strip()
            if not text:
                raise ValueError("empty response from ollama")
            return sanitize_explanation(text)

    # _request_body - builds the ollama generate payload with token caps from env
    def _request_body(self, prompt: str) -> dict[str, Any]:
        # cap tokens: 40 often cut mid-sentence; ~80 fits opener + one short idea
        try:
            num_predict = max(48, int(os.getenv("OLLAMA_NUM_PREDICT", "96")))
        except ValueError:
            num_predict = 96
        try:
            num_ctx = max(512, int(os.getenv("OLLAMA_NUM_CTX", "1024")))
        except ValueError:
            num_ctx = 1024
        return {
            "model": self.model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "num_predict": num_predict,
                "num_ctx": num_ctx,
                "temperature": 0.3,
            },
        }


# HeuristicProvider - rule-based coach fallback with no external dependency
class HeuristicProvider:
    name = "heuristic"

    # explain - returns heuristic fallback coach text for the last move
    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None = None,
        game_type: str = "chess",
        skill_level: str = "intermediate",
        human_color: str | None = None,
        concept_hints: list[str] | None = None,
    ) -> str:
        _ = skill_level
        _ = human_color
        _ = concept_hints
        return build_explanation_fallback(
            fen=fen,
            color=color,
            move_uci=move_uci or "",
            move_san=move_san,
            game_type=game_type,
        )


# get_llm_provider - returns the llm provider selected by LLM_PROVIDER env
def get_llm_provider() -> LLMProvider:
    name = os.getenv("LLM_PROVIDER", "ollama").lower().strip()
    if name in ("heuristic", "fallback", "offline"):
        return HeuristicProvider()
    return OllamaProvider()


if __name__ == "__main__" or os.getenv("LLM_PROVIDER_SELFCHECK"):
    p1 = get_llm_provider()
    p2 = HeuristicProvider()
    assert hasattr(p1, "explain") and hasattr(p2, "explain")
    fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
    beg = build_teacher_prompt(
        fen=fen, move_uci="e2e4", move_san="e4", move_history=[], skill_level="beginner"
    )
    adv = build_teacher_prompt(
        fen=fen, move_uci="e2e4", move_san="e4", move_history=[], skill_level="advanced"
    )
    assert "Skill level: beginner" in beg and "everyday" in beg.lower()
    assert "Skill level: advanced" in adv and "prophylaxis" in adv
    assert beg != adv
