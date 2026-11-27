"""Qualify the approved runtime layout in an unchanged queued native snapshot."""

import argparse
from pathlib import Path
import sys

from gate_common import report, run
from qualify_r1 import source_identity
from qualify_r2_inputs import checks


def protected_changes(root: Path, baseline: str) -> list[str]:
    resolved = run(["git", "rev-parse", "--verify", "--end-of-options", baseline + "^{commit}"], root)
    if resolved.returncode:
        return ["protected baseline unavailable: " + resolved.stderr]
    result = run(["git", "diff", "--exit-code", resolved.stdout.strip(), "--", "protocol", "schema", "internal/validation",
                  "go.mod", "go.sum", "cmd/mcpgen/go.mod", "cmd/mcpgen/go.sum", "tools/versions.json",
                  "tools/r2-candidate-migration.json", "tools/r2-candidate-migration-pre-lint.json"], root)
    return [] if result.returncode == 0 else ["protected contract, validation, module or pin changed: " + result.stdout + result.stderr]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    parser.add_argument("--baseline", required=True, help="coherent prerequisite commit")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("L1 qualification identity", before, flush=True)
    failures = report("layout.protected-baseline", protected_changes(root, args.baseline))
    commands = (*checks(args.allow_network),
                ("layout-controls", [sys.executable, "-B", "scripts/test_library_layout.py"]),
                ("make-foundation-regressions", [sys.executable, "-B", "scripts/test_make_tasks.py"]),
                ("make-lint-plants", [sys.executable, "-B", "scripts/verify_make_lint.py"]),
                ("publication-cli-plants", [sys.executable, "-B", "scripts/verify_publication_guards.py"]))
    for name, command in commands:
        print("L1 qualification check", name, flush=True)
        result = run(command, root, network=name == "preparation" and args.allow_network, timeout=1800)
        print(result.stdout, end="", flush=True)
        print(result.stderr, end="", file=sys.stderr, flush=True)
        failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
    failures += report("layout.protected-final", protected_changes(root, args.baseline))
    failures += report("source-preservation", [] if source_identity(root) == before else ["source changed"])
    print("L1 qualification failures", failures, flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
