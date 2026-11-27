"""Qualify R3 in an unchanged snapshot before live publication; not the exit gate."""

import argparse
from pathlib import Path
import sys

from gate_common import report, run
from qualify_r1 import source_identity
from qualify_r2_inputs import checks


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    parser.add_argument("--race", action="store_true", help="also race-test actual CLI fixtures on the eligible physical Linux runner")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("R3 qualification identity", before, flush=True)
    failures = 0
    commands = (*checks(args.allow_network),
                ("make-foundation-regressions", [sys.executable, "-B", "scripts/test_make_tasks.py"]),
                ("make-lint-plants", [sys.executable, "-B", "scripts/verify_make_lint.py"]),
                ("publication-cli-plants", [sys.executable, "-B", "scripts/verify_publication_guards.py"]))
    for name, command in commands:
        print("R3 qualification check", name, flush=True)
        result = run(command, root, network=name == "preparation" and args.allow_network, timeout=1800)
        print(result.stdout, end="", flush=True)
        print(result.stderr, end="", file=sys.stderr, flush=True)
        failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
    if args.race:
        result = run(["env", "CGO_ENABLED=1", "go", "test", "-race", "-vet=off", "-count=1",
                      "-run", "^TestRepairA13(Recovery|CLIScope|GitBounds)$", "."],
                     root / "cmd/mcpgen", timeout=1800)
        print(result.stdout, end="", flush=True)
        print(result.stderr, end="", file=sys.stderr, flush=True)
        failures += report("actual-cli-race", [] if result.returncode == 0 else [f"exit {result.returncode}"])
    failures += report("source-preservation", [] if source_identity(root) == before else ["source changed"])
    print("R3 qualification failures", failures, flush=True)
    print("Live promotion, post-publication checks and coherent commit remain separate.", flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
