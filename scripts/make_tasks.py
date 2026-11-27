"""Portable Make dispatch; existing Python gates own each validation leaf."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

import gate
from gate_common import precheck, report, run

ROOT = Path(__file__).resolve().parent.parent
MODULE_GOALS = {"fmt-check": {"fmt"}, "lint": {"lint"}, "vet": {"vet"},
                "test": {"vet", "test"}, "pre-add-check": {"fmt", "vet", "test", "lint"}}
LEAF_GOALS = {"setup": {"setup"}, "setup-network": {"setup"},
              "check-records": {"records"}, "markdownlint": {"markdown"},
              "docs-check": {"records", "markdown"}, "inventory-check": {"inventory"},
              "generate-check": {"generated"}, "build-check": {"generated"},
              "gate-selftest": {"selftest", "lint-controls"}, "lint-selftest": {"lint-controls"},
              "help": set(), "all": set()}
GOALS = {**MODULE_GOALS, **LEAF_GOALS}


def selections(goals: list[str]) -> tuple[frozenset[str], frozenset[str]]:
    checks, leaves = set(), set()
    for goal in goals:
        if goal not in GOALS:
            raise ValueError(f"unknown Make goal: {goal}")
        if goal in MODULE_GOALS:
            checks.update(MODULE_GOALS[goal])
        else:
            leaves.update(LEAF_GOALS[goal])
    return frozenset(checks), frozenset(leaves)


def print_help() -> None:
    print("Local validation: make <target> [MODULE=all|runtime|generator]")
    print("Normal Go checks use CGO=0, readonly independent modules and offline dependencies.")
    print("lint covers the native host; the native fleet qualification covers every supported tuple.")
    for name in GOALS:
        if name != "all":
            print("  " + name)
    print("setup-network explicitly prepares pinned dependencies and the Markdown tool.")
    print("Complete preflight, race/security integration and unfinished product gates remain in the PLAN.")


def command(case: str, args: list[str], cwd: Path = ROOT, *, network: bool = False) -> int:
    result = run(args, cwd, network=network, timeout=1800)
    print(result.stdout, end="", flush=True)
    print(result.stderr, end="", file=sys.stderr, flush=True)
    return report(case, [] if result.returncode == 0 else [f"exit {result.returncode}"])


def execute(goals: list[str], module: str, checks: frozenset[str], leaves: frozenset[str]) -> int:
    pins = json.loads((ROOT / "tools/versions.json").read_text())
    failures = 0
    if {"help", "all"} & set(goals):
        print_help()
    if "setup" in leaves:
        online = "setup-network" in goals
        failures += gate.main(["setup", *(["--allow-network"] if online else [])])
        if online:
            version = pins["lint"]["markdownlint-cli2"]["version"]
            failures += command("setup.markdown", ["npx", "--yes", f"markdownlint-cli2@{version}", "--help"], network=True)
    if checks:
        failures += precheck(ROOT, module, pins, checks)
    if "generated" in leaves:
        failures += command("generated", ["go", "run", ".", "check", "--repo", "../..", "--format", "json"], ROOT / "cmd/mcpgen")
    if "inventory" in leaves:
        failures += gate.main(["inventory"])
    if "records" in leaves:
        failures += command("records", [sys.executable, "-B", "scripts/check_records.py", "--check-all"])
    if "markdown" in leaves:
        version = pins["lint"]["markdownlint-cli2"]["version"]
        failures += command("markdown", ["npx", "--offline", "--yes", f"markdownlint-cli2@{version}"])
    if "selftest" in leaves:
        for name in ("test_gate.py", "test_r2_evidence.py", "test_make_tasks.py", "test_library_layout.py"):
            failures += command(name, [sys.executable, "-B", "scripts/" + name])
    if "lint-controls" in leaves:
        failures += command("lint-controls", [sys.executable, "-B", "scripts/verify_make_lint.py"])
    return int(bool(failures))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--module", choices=("all", "runtime", "generator"), default="all")
    parser.add_argument("--worker", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("goals", nargs="*", choices=None)
    args = parser.parse_args(argv)
    goals = args.goals or ["help"]
    try:
        checks, leaves = selections(goals)
        if not checks and not leaves:
            print_help()
            return 0
        heavy = bool("test" in checks or leaves & {"generated", "selftest", "setup", "lint-controls"})
        ticket = os.environ.get("DEV_HOST_QUEUE_TICKET")
        if args.worker and heavy and not ticket:
            raise ValueError("heavy worker requires an admitted host queue ticket")
        if heavy and not ticket:
            runner = Path.home() / ".agents/skills/available-dev-hosts/scripts/run-on-host"
            if not runner.is_file():
                raise ValueError("fleet runner unavailable; install the operator's available-dev-hosts tooling")
            result = subprocess.run([sys.executable, str(runner), "--need", "any", "--label", "mcp-make",
                                     "--", "python3", "-B", "scripts/make_tasks.py", "--worker",
                                     "--module", args.module, *goals], cwd=ROOT, check=False)
            return result.returncode
        return execute(goals, args.module, checks, leaves)
    except (OSError, ValueError, KeyError, TypeError) as exc:
        return int(bool(report("make", [str(exc)])))


if __name__ == "__main__":
    raise SystemExit(main())
