#!/usr/bin/env python3
# test_fen_from_image.py - light api checks for diagram→fen (no torch unless live env set)

from __future__ import annotations

import os
import sys
import unittest
from io import BytesIO
from pathlib import Path


CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
PARENT_DIR = os.path.dirname(CURRENT_DIR)
if PARENT_DIR not in sys.path:
    sys.path.insert(0, PARENT_DIR)

import fen_from_image  # noqa: E402
import server  # noqa: E402

_FIXTURE = (
    Path(PARENT_DIR).parent
    / "gameplay_capture"
    / "chess"
    / "chess-08082026.webp"
)
_VENDOR_PY = Path(PARENT_DIR).parent.parent / "_local_Chess_diagram_to_FEN" / ".venv" / "bin" / "python"


# _torch_available - true only when this interpreter can import torch
def _torch_available() -> bool:
    try:
        import torch  # noqa: F401

        return True
    except ImportError:
        return False


class TestFenFromImageAPI(unittest.TestCase):
    # setUp - flask test client
    def setUp(self) -> None:
        server.app.config["TESTING"] = True
        self.client = server.app.test_client()

    # test_route_registered - /fen_from_image is on the app map
    def test_route_registered(self) -> None:
        rules = {r.rule for r in server.app.url_map.iter_rules()}
        self.assertIn("/fen_from_image", rules)

    # test_xianqi_alias_maps_to_xiangqi - analyser spelling maps to vendor key
    def test_xianqi_alias_maps_to_xiangqi(self) -> None:
        self.assertEqual(fen_from_image.resolve_diagram_game("xianqi"), "xiangqi")

    # test_chess_vision_checkpoint_names_when_vendor_present - five Chess best_model_*.pth prefixes
    def test_chess_vision_checkpoint_names_when_vendor_present(self) -> None:
        names = fen_from_image.chess_vision_checkpoints()
        if not names:
            self.skipTest(f"no chess .pth under {fen_from_image._VENDOR_DIR}")
        prefixes = (
            "best_model_existence_",
            "best_model_image_rotation_",
            "best_model_orientation_",
            "best_model_position_",
            "best_model_quad_",
        )
        joined = " ".join(names)
        for prefix in prefixes:
            self.assertTrue(
                any(n.startswith(prefix) for n in names),
                f"missing {prefix}*.pth in {names}",
            )
        self.assertIn("1.0/models.zip", fen_from_image.VISION_WEIGHTS_RELEASE)
        self.assertTrue(joined)

    # test_rejects_missing_image - no file → 400
    def test_rejects_missing_image(self) -> None:
        response = self.client.post("/fen_from_image", data={"game": "chess"})
        self.assertEqual(response.status_code, 400)
        payload = response.get_json()
        self.assertEqual(payload.get("status"), "error")

    # test_rejects_bad_image_bytes - garbage bytes → validation
    def test_rejects_bad_image_bytes(self) -> None:
        response = self.client.post(
            "/fen_from_image",
            data={"game": "chess", "image": (BytesIO(b"not-an-image"), "x.png")},
            content_type="multipart/form-data",
        )
        self.assertEqual(response.status_code, 400)
        payload = response.get_json()
        self.assertEqual(payload.get("error_kind"), "validation")

    @unittest.skipUnless(
        os.getenv("PY_FEN_FROM_IMAGE_LIVE") == "1",
        "set PY_FEN_FROM_IMAGE_LIVE=1 to run fixture recognition",
    )
    @unittest.skipUnless(
        _torch_available(),
        f"use vendor venv python (has torch): {_VENDOR_PY}",
    )
    # test_chess_fixture_returns_fen - live torch path returns a FEN with ranks
    def test_chess_fixture_returns_fen(self) -> None:
        self.assertTrue(_FIXTURE.is_file(), f"missing fixture {_FIXTURE}")
        self.assertTrue(
            fen_from_image._VENDOR_DIR.is_dir(),
            f"missing vendor {fen_from_image._VENDOR_DIR}",
        )
        response = self.client.post(
            "/fen_from_image",
            data={
                "game": "chess",
                "image": (_FIXTURE.open("rb"), _FIXTURE.name),
            },
            content_type="multipart/form-data",
        )
        payload = response.get_json()
        self.assertEqual(
            response.status_code,
            200,
            f"expected 200, got {response.status_code}: {payload}",
        )
        self.assertEqual(payload.get("status"), "ok")
        self.assertIn("/", payload.get("fen", ""))


if __name__ == "__main__":
    unittest.main()
