#!/usr/bin/env python3
"""Exercise the complete proposed product in scratch and retain test diagnostics."""

from pathlib import Path
import sys

from gate_common import linter_executable, report, run
from qualify_r1 import source_identity


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("R2 product identity", before, flush=True)
    result = run(
        ["go", "test", "-vet=off", "-count=1", "-v", "-run", "^TestRepairA11ActualProduct$", "."],
        root / "cmd/mcpgen", timeout=1800,
        env_updates={"GO_MCP_LINTER": linter_executable(root)},
    )
    print(result.stdout, end="", flush=True)
    print(result.stderr, end="", file=sys.stderr, flush=True)
    failures = report("actual-product", [] if result.returncode == 0 else [f"exit {result.returncode}"])
    failures += report("source-preservation", [] if before == source_identity(root) else ["source changed"])
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
