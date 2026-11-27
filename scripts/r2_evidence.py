"""Retain candidate evidence from an already successful canonical test run."""

import json
import argparse
from pathlib import Path


def migration_evidence(output: str) -> dict:
    locations = [json.loads(line.removeprefix("test-logs ")) for line in output.splitlines()
                 if line.startswith("test-logs ")]
    logs = [row["directory"] for row in locations if row.get("module") == "generator"]
    if len(logs) != 1:
        raise ValueError("expected exactly one generator test log")
    stream = Path(logs[0]) / "stdout"
    if stream.stat().st_size > 32 * 1024 * 1024:
        raise ValueError("generator test log exceeds bound")
    chunks = []
    passed = False
    for line in stream.read_text().splitlines():
        event = json.loads(line)
        if (event.get("Package") == "github.com/maccavelli/go-mcp-lib/cmd/mcpgen"
                and event.get("Test") == "TestRepairA11ActualProduct"):
            if event.get("Action") == "output":
                chunks.append(event.get("Output", ""))
            elif event.get("Action") in {"pass", "fail", "skip"}:
                passed = event["Action"] == "pass"
    found = []
    for line in "".join(chunks).splitlines():
        _, marker, body = line.partition("R2_MIGRATION ")
        if marker:
            found.append(json.loads(body))
    if len(found) != 1 or not passed:
        raise ValueError("expected exactly one qualified migration inventory")
    return found[0]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    logs = parser.add_mutually_exclusive_group(required=True)
    logs.add_argument("--qualification-log", type=Path)
    logs.add_argument("--gate-log", type=Path, help="recover extraction-only failure from retained passing test logs")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.gate_log is not None:
        text = args.gate_log.read_text()
        lines = text.splitlines()
        failures = [line for line in lines if line.startswith("FAIL ")
                    and not line.startswith("FAIL migration-evidence: ")]
        if (failures or "pass generator-pre-add" not in lines or "pass source-preservation" not in lines
                or "R2 input check failures 1" not in lines):
            raise ValueError("recovery requires passing product/controls and only an extraction failure")
        rows = [migration_evidence(text)]
    else:
        rows = [json.loads(line.removeprefix("R2_MIGRATION "))
                for line in args.qualification_log.read_text().splitlines()
                if line.startswith("R2_MIGRATION ")]
    if len(rows) != 1:
        raise ValueError("expected exactly one completed qualification inventory")
    args.output.write_text(json.dumps(rows[0], indent=2) + "\n")


if __name__ == "__main__":
    main()
