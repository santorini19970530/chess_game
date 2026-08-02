#!/usr/bin/env python3
# run_all.py - discovers tester suite and exits hard so Flask threads cannot hang
from __future__ import annotations

import os
import unittest


def main() -> None:
    suite = unittest.defaultTestLoader.discover(
        start_dir=os.path.dirname(os.path.abspath(__file__)),
        pattern="test_*.py",
    )
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    # Flask test_client can leave non-daemon threads; hard-exit after the report
    os._exit(0 if result.wasSuccessful() else 1)


if __name__ == "__main__":
    main()
