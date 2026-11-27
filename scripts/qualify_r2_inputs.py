#!/usr/bin/env python3
"""Check R2 input repairs and plants; this is not the complete R2 exit gate."""

import argparse
import json
from pathlib import Path
import sys

from gate_common import report, run
from qualify_r1 import source_identity
from r2_evidence import migration_evidence


def checks(allow_network: bool = False, mapping_report: bool = False) -> tuple:
    return (
        ("preparation", [sys.executable, "-B", "scripts/gate.py", "setup",
                         *(["--allow-network"] if allow_network else [])]),
        ("gate-runner-regressions", [sys.executable, "-B", "scripts/test_gate.py"]),
        ("evidence-regressions", [sys.executable, "-B", "scripts/test_r2_evidence.py"]),
        ("diagnostic-plants", [sys.executable, "-B", "scripts/verify_diagnostic_guards.py"]),
        ("generator-pre-add", [sys.executable, "-B", "scripts/go-precheck.py", "--module", "generator"]),
        ("loader-model-plants", [sys.executable, "-B", "scripts/verify_loader_guards.py"]),
        ("reference-plants", [sys.executable, "-B", "scripts/verify_reference_guards.py"]),
        ("alias-plants", [sys.executable, "-B", "scripts/verify_alias_guards.py"]),
        ("profile-plants", [sys.executable, "-B", "scripts/verify_profile_guards.py"]),
        ("emitter-plants", [sys.executable, "-B", "scripts/verify_emitter_guards.py"]),
        ("candidate-plants", [sys.executable, "-B", "scripts/verify_candidate_guards.py"]),
        *([("mapping-evidence", [sys.executable, "-B", "scripts/capture_r2_mapping.py"])] if mapping_report else []),
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    parser.add_argument("--mapping-report", action="store_true", help="capture fresh SDK mapping evidence")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    before = source_identity(root)
    print("R2 input identity", before, flush=True)
    commands = checks(args.allow_network, args.mapping_report)
    failures = 0
    migration = None
    for name, command in commands:
        print("R2 input check", name, flush=True)
        result = run(command, root, network=name == "preparation" and args.allow_network, timeout=1800)
        print(result.stdout, end="", flush=True)
        print(result.stderr, end="", file=sys.stderr, flush=True)
        failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
        if name == "generator-pre-add" and result.returncode == 0:
            try:
                migration = migration_evidence(result.stdout)
            except (OSError, ValueError, KeyError, TypeError) as exc:
                failures += report("migration-evidence", [str(exc)])
    after = source_identity(root)
    failures += report("source-preservation", [] if before == after else [f"source changed: {before} -> {after}"])
    if not failures and migration is not None:
        target = root / ".cache" / "r2-candidate-migration.json"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(json.dumps(migration, indent=2) + "\n")
        print("R2_MIGRATION " + json.dumps(migration, separators=(",", ":")), flush=True)
    print("R2 input check failures", failures, flush=True)
    print("Complete R2 candidate and publication acceptance remain separate.", flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
