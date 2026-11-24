#!/usr/bin/env python3
"""go-precheck: fast pre-add checks for the go-mcp-lib framework.

Enumerates the root and generator Go modules explicitly, then runs gofmt,
go vet, and package tests per module with GOWORK=off. Commands run through
subprocess argument arrays with a controlled environment; no shell, no
network. Exit code is nonzero if any check fails, and every failed check
prints a named FAIL line.
"""

from __future__ import annotations

import os
import subprocess
import sys

# Modules are enumerated explicitly: root `go test ./...` does not cover
# cmd/mcpgen because it is a separate module.
MODULES = [
    {"name": "runtime", "dir": "."},
    {"name": "generator", "dir": "cmd/mcpgen"},
]

ENV = {
    "PATH": os.environ.get("PATH", "/usr/local/bin:/usr/bin:/bin"),
    "HOME": os.environ.get("HOME", ""),
    "GOWORK": "off",
    "GOTOOLCHAIN": "local",
    "GOPROXY": "off",
    "GOSUMDB": "off",
    # GOCACHE is deliberately inherited so local builds reuse cache.
}

# Case names are the contract consumed by scripts/gate.py mode `pre-add`.
PRE_ADD_CASES = ("gofmt", "vet", "build", "test")


def run(module: dict, args: list[str]) -> tuple[int, str]:
    """Run a go command in a module dir; return (exit code, output)."""
    proc = subprocess.run(
        ["go", *args],
        cwd=module["dir"],
        env=ENV,
        capture_output=True,
        text=True,
    )
    return proc.returncode, proc.stdout + proc.stderr


def prechecks() -> int:
    """Run pre-add checks for every module; return failure count."""
    failures = 0
    for module in MODULES:
        name = module["name"]
        if not os.path.isdir(module["dir"]) or not os.path.exists(
            os.path.join(module["dir"], "go.mod")
        ):
            print(f"FAIL {name}: missing module at {module['dir']}")
            failures += 1
            continue
        code, out = run(module, ["fmt", "./..."])
        dirty = [ln for ln in out.splitlines() if ln.strip()]
        if code != 0:
            print(f"FAIL {name}: gofmt errored: {out.strip()}")
            failures += 1
        elif dirty:
            print(f"FAIL {name}: gofmt would rewrite {len(dirty)} file(s)")
            for ln in dirty:
                print(f"  {ln}")
            failures += 1
        else:
            print(f"pass {name}: gofmt clean")
        code, out = run(module, ["vet", "./..."])
        if code != 0:
            print(f"FAIL {name}: go vet: {out.strip()}")
            failures += 1
        else:
            print(f"pass {name}: go vet clean")
        code, out = run(module, ["build", "./..."])
        if code != 0:
            print(f"FAIL {name}: go build: {out.strip()}")
            failures += 1
        else:
            print(f"pass {name}: go build clean")
        code, out = run(module, ["test", "./..."])
        if code != 0:
            print(f"FAIL {name}: go test: {out.strip()}")
            failures += 1
        else:
            print(f"pass {name}: go test clean")
    return failures


def main() -> int:
    if len(sys.argv) > 2:
        print(__doc__)
        return 2
    failures = prechecks()
    if failures:
        print(f"pre-add: {failures} FAIL case(s)")
        return 1
    print("pre-add: all cases pass")
    return 0


if __name__ == "__main__":
    sys.exit(main())
