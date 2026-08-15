#!/usr/bin/env python3
# test_fs_nnue.py - NNUE path resolve + shared UCI command helper

from __future__ import annotations

import os
import unittest
from pathlib import Path
from unittest import mock

from fs_engine import (
    DEFAULT_NNUE_FILENAME,
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


if __name__ == "__main__":
    unittest.main()
