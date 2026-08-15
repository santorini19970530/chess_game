#!/usr/bin/env python3
# test_fs_nnue.py - NNUE path resolve + shared UCI command helper

from __future__ import annotations

import os
import unittest
from pathlib import Path
from unittest import mock

from fs_engine import (
    DEFAULT_NNUE_FILENAME,
    FS_BINARY_PATH,
    classify_eval_mode,
    nnue_eval_probe,
    nnue_evidence_check,
    nnue_uci_commands,
    resolve_nnue_path,
)


class TestFSNNUEConfig(unittest.TestCase):
    # test_env_path_wins_when_file_exists - env path wins and builds Use NNUE + EvalFile lines
    def test_env_path_wins_when_file_exists(self) -> None:
        path = Path(self.id().replace(".", "_") + ".nnue")
        self.addCleanup(lambda: path.unlink(missing_ok=True))
        path.write_bytes(b"not-a-real-net")
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": str(path)}):
            self.assertEqual(resolve_nnue_path(), str(path))
            cmds = nnue_uci_commands()
        self.assertEqual(cmds[0], "setoption name Use NNUE value true")
        self.assertEqual(cmds[1], f"setoption name EvalFile value {path}")

    # test_missing_env_path_returns_empty_commands - missing env file yields no UCI options
    def test_missing_env_path_returns_empty_commands(self) -> None:
        missing = "/tmp/does-not-exist-" + DEFAULT_NNUE_FILENAME
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": missing}):
            self.assertEqual(resolve_nnue_path(), "")
            self.assertEqual(nnue_uci_commands(), [])

    # test_evidence_unset_path_fails - unset env is not pretrained-model evidence
    def test_evidence_unset_path_fails(self) -> None:
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": ""}, clear=False):
            os.environ.pop("FAIRY_STOCKFISH_NNUE_PATH", None)
            with self.assertRaises(RuntimeError) as ctx:
                nnue_evidence_check()
        self.assertIn("unset", str(ctx.exception))

    # test_evidence_missing_file_fails - requested path that does not exist fails evidence
    def test_evidence_missing_file_fails(self) -> None:
        missing = "/tmp/does-not-exist-" + DEFAULT_NNUE_FILENAME
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": missing}):
            with self.assertRaises(RuntimeError) as ctx:
                nnue_evidence_check()
        self.assertIn("missing", str(ctx.exception))

    # test_evidence_tiny_file_rejected - placeholder bytes are not a real SF14 net
    def test_evidence_tiny_file_rejected(self) -> None:
        path = Path(self.id().replace(".", "_") + ".nnue")
        self.addCleanup(lambda: path.unlink(missing_ok=True))
        path.write_bytes(b"not-a-real-net")
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": str(path)}):
            with self.assertRaises(RuntimeError) as ctx:
                nnue_evidence_check()
        self.assertIn("rejected", str(ctx.exception))

    # test_classify_eval_mode_nnue - verify() NNUE line is pretrained-model evidence
    def test_classify_eval_mode_nnue(self) -> None:
        text = (
            "info string NNUE evaluation using /tmp/nn-3475407dc199.nnue enabled\n"
            "Final evaluation       +0.16 (white side) [with scaled NNUE, hybrid, ...]"
        )
        self.assertEqual(classify_eval_mode(text), "nnue")

    # test_classify_eval_mode_classical - verify() classical line is not pretrained evidence
    def test_classify_eval_mode_classical(self) -> None:
        self.assertEqual(
            classify_eval_mode("info string classical evaluation enabled"),
            "classical",
        )

    # test_eval_probe_chess_uses_nnue - live UCI eval on Chess startpos uses the SF14 net
    def test_eval_probe_chess_uses_nnue(self) -> None:
        nnue = _live_nnue_path()
        if not nnue or not os.path.exists(FS_BINARY_PATH):
            self.skipTest("Fairy-Stockfish binary or nn-3475407dc199.nnue not present")
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": nnue}):
            mode, lines = nnue_eval_probe("chess")
        self.assertEqual(mode, "nnue", lines)

    # test_eval_probe_xiangqi_and_shogi_stay_classical - this Chess net does not apply
    def test_eval_probe_xiangqi_and_shogi_stay_classical(self) -> None:
        nnue = _live_nnue_path()
        if not nnue or not os.path.exists(FS_BINARY_PATH):
            self.skipTest("Fairy-Stockfish binary or nn-3475407dc199.nnue not present")
        with mock.patch.dict(os.environ, {"FAIRY_STOCKFISH_NNUE_PATH": nnue}):
            for variant in ("xiangqi", "shogi"):
                mode, lines = nnue_eval_probe(variant)
                self.assertEqual(mode, "classical", f"{variant}: {lines}")


# _live_nnue_path - real SF14 net if present
def _live_nnue_path() -> str:
    env = os.environ.get("FAIRY_STOCKFISH_NNUE_PATH", "").strip()
    if env and os.path.isfile(env):
        return env
    here = Path(__file__).resolve().parent
    for cand in (
        here.parents[3] / "_local_nnue" / DEFAULT_NNUE_FILENAME,
        here.parents[2] / "_local_nnue" / DEFAULT_NNUE_FILENAME,
    ):
        if cand.is_file():
            return str(cand)
    return ""


if __name__ == "__main__":
    unittest.main()
