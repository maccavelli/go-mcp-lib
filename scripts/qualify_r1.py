#!/usr/bin/env python3
"""Run one R1 qualification pass; invoke through the fleet's heavy-run queue."""

import argparse
import hashlib
from pathlib import Path
import subprocess
import sys

from gate_common import report


def source_identity(root: Path) -> str:
    result = subprocess.run(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"],
                            cwd=root, capture_output=True, check=True)
    digest = hashlib.sha256()
    for name in sorted(set(result.stdout.split(b"\0"))):
        if not name:
            continue
        path = root / name.decode()
        digest.update(name + b"\0")
        if path.is_symlink():
            digest.update(str(path.readlink()).encode())
        elif path.is_file():
            digest.update(path.read_bytes())
        else:
            digest.update(b"missing")
        digest.update(b"\0")
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("R1 input identity", before, flush=True)
    commands = [
        ("harness", [sys.executable, "-B", "-m", "unittest", "discover", "-s", "scripts", "-p", "test_gate.py", "-v"]),
        ("inventory", [sys.executable, "-B", "scripts/acceptance_inventory.py"]),
        ("preparation", [sys.executable, "-B", "scripts/gate.py", "setup", *(["--allow-network"] if args.allow_network else [])]),
        ("pre-add", [sys.executable, "-B", "scripts/go-precheck.py"]),
        ("actual-plants", [sys.executable, "-B", "scripts/gate.py", "plants"]),
        ("fixture-bindings", [sys.executable, "-B", "scripts/verify_fixture_bindings.py"]),
    ]
    failures = 0
    for name, command in commands:
        print("R1 command", name, command, flush=True)
        try:
            result = subprocess.run(command, cwd=root, timeout=1800, check=False)
            failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
        except (OSError, subprocess.TimeoutExpired) as exc:
            failures += report(name, [str(exc)])
    after = source_identity(root)
    failures += report("source-preservation", [] if before == after else [f"source changed: {before} -> {after}"])
    print("R1 qualification failures", failures, flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
