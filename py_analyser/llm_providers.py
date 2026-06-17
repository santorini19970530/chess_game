from __future__ import annotations

import json
import os
import socket
import urllib.error
import urllib.request
from typing import Protocol

from analyzer import build_explanation_fallback


class LLMProvider(Protocol):
    """Minimal protocol for LLM explanation providers."""

    name: str

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
    ) -> str:
        ...


class OllamaProvider:
    """Calls a local Ollama instance (default path)."""

    name = "ollama"

    def __init__(self, model: str | None = None, timeout: float = 9.0) -> None:
        self.model = model or os.getenv("OLLAMA_MODEL", "gemma2:2b")
        self.timeout = timeout
        self.url = "http://localhost:11434/api/generate"

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None,
    ) -> str:
        move_text = move_san or move_uci or ""
        history = move_history or []
        history_str = " ".join(history[-6:]) if history else "(no prior moves)"

        prompt = (
            f"You are a calm chess coach for an intermediate club player. "
            f"Explain the move {move_text} in 2-4 short sentences. "
            f"FEN: {fen}. Recent moves: {history_str}. "
            f"Mention whether it creates threats, changes material balance, or improves safety. "
            f"Keep the tone educational and encouraging; avoid engine jargon."
        )

        req = urllib.request.Request(
            self.url,
            data=json.dumps({"model": self.model, "prompt": prompt, "stream": False}).encode("utf-8"),
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
            return text


class HeuristicProvider:
    """Pure rule-based fallback (no external dependency)."""

    name = "heuristic"

    def explain(
        self,
        *,
        fen: str,
        color: str,
        move_uci: str,
        move_san: str | None,
        move_history: list[str] | None = None,
    ) -> str:
        return build_explanation_fallback(
            fen=fen,
            color=color,
            move_uci=move_uci or "",
            move_san=move_san,
        )


def get_llm_provider() -> LLMProvider:
    """Factory: returns provider based on LLM_PROVIDER env var."""
    name = os.getenv("LLM_PROVIDER", "ollama").lower().strip()
    if name in ("heuristic", "fallback", "offline"):
        return HeuristicProvider()
    return OllamaProvider()