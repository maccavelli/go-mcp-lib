#!/usr/bin/env python3
"""Prove failed-test details survive later output and omission is detected."""

from pathlib import Path
import sys

from gate_common import report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    case = "test_gate.TestRepairA02GateAccounting.test_failed_test_diagnostics_survive_later_output"
    control = run([sys.executable, "-B", "-m", "unittest", case], root / "scripts")
    failures = report("diagnostics.valid", [] if control.returncode == 0 else [control.stdout + control.stderr])
    plant = f"""import unittest
from unittest.mock import patch
import gate_common
suite = unittest.defaultTestLoader.loadTestsFromName({case!r})
with patch.object(gate_common, 'failed_test_diagnostics', return_value=[]):
    result = unittest.TextTestRunner(verbosity=2).run(suite)
raise SystemExit(not result.wasSuccessful())
"""
    broken = run([sys.executable, "-B", "-c", plant], root / "scripts")
    caught = broken.returncode != 0 and "fixture owned validation failed" in broken.stderr
    failures += report("diagnostics.omission-plant", [] if caught else [broken.stdout + broken.stderr or "plant was accepted"],
                       "FAIL fixture owned validation failed; omission detected")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
