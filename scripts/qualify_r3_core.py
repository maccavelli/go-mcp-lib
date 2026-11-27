"""Qualify the publication core; this is not the full R3 exit or promotion gate."""

import argparse
from pathlib import Path
import sys

from gate_common import report, run
from qualify_r1 import source_identity


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("R3 core identity", before, flush=True)
    failures = 0
    for name, command in (
        ("preparation", [sys.executable, "-B", "scripts/gate.py", "setup",
                         *(["--allow-network"] if args.allow_network else [])]),
        ("generator-pre-add", [sys.executable, "-B", "scripts/go-precheck.py", "--module", "generator"]),
        ("publication-plants", [sys.executable, "-B", "scripts/verify_publication_guards.py"]),
        ("candidate-plants", [sys.executable, "-B", "scripts/verify_candidate_guards.py"]),
    ):
        result = run(command, root, network=name == "preparation" and args.allow_network, timeout=1800)
        print(result.stdout, end="", flush=True)
        print(result.stderr, end="", file=sys.stderr, flush=True)
        failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
    failures += report("source-preservation", [] if source_identity(root) == before else ["source changed"])
    print("R3 core failures", failures, flush=True)
    print("R3 CLI integration, native matrix and verified promotion remain separate.", flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
