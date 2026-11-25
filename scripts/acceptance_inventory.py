"""Reconcile acceptance ownership with frozen schema, requirements and PLAN.

--write explicitly regenerates the reviewable inventory; the default is readonly.
An inventory pass establishes coverage ownership, not product qualification.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import re

from gate_common import report
from prepare import digest

PLAN = "docs/decisions/0002-PLAN-build-schema-driven-mcp-framework.md"
REQUIREMENTS = "tools/fixtures/conformance-requirements-2026-07-28.yaml"
INVENTORY = "tools/acceptance-inventory.json"

# Independently reviewed method directions; an added method requires review.
METHOD_DIRECTIONS = {
    "tools/call": "client to server", "tools/list": "client to server",
    "completion/complete": "client to server", "server/discover": "client to server",
    "prompts/get": "client to server", "prompts/list": "client to server",
    "resources/read": "client to server", "resources/list": "client to server",
    "resources/templates/list": "client to server", "subscriptions/listen": "client to server",
    "sampling/createMessage": "server input request / client owned fulfillment",
    "roots/list": "server input request / client owned fulfillment",
    "elicitation/create": "server input request / client owned fulfillment",
    "notifications/cancelled": "client to server; server to client on stdio only to terminate subscriptions/listen",
    "notifications/progress": "request receiver to request sender",
    "notifications/message": "server to client",
    "notifications/prompts/list_changed": "server to client",
    "notifications/resources/list_changed": "server to client",
    "notifications/resources/updated": "server to client",
    "notifications/subscriptions/acknowledged": "server to client",
    "notifications/tools/list_changed": "server to client",
}


def definition_usage(definitions: dict) -> dict[str, list[str]]:
    """Trace method envelopes and associated results through the pinned graph."""
    usages = {name: set() for name in definitions}

    def references(node):
        if isinstance(node, dict):
            for key, value in node.items():
                if key == "$ref" and isinstance(value, str) and value.startswith("#/$defs/"):
                    yield value.removeprefix("#/$defs/").split("/")[0]
                else:
                    yield from references(value)
        elif isinstance(node, list):
            for value in node:
                yield from references(value)

    for name, node in definitions.items():
        method = node.get("properties", {}).get("method", {}).get("const")
        if not method or name == "ClientNotification":
            continue
        if method not in METHOD_DIRECTIONS:
            raise ValueError(f"unreviewed method direction: {method}")
        pending = [name]
        result = name.removesuffix("Request") + "Result"
        if name.endswith("Request") and result in definitions:
            pending.append(result)
        seen = set()
        while pending:
            current = pending.pop()
            if current in seen:
                continue
            if current not in definitions:
                raise ValueError(f"unresolved inventory reference: {current}")
            seen.add(current)
            usages[current].add(method)
            pending.extend(references(definitions[current]))
    return {name: sorted(methods) for name, methods in usages.items()}


def scenarios(body: str) -> list[dict]:
    """Read the verified frozen requirements' list subset, not arbitrary YAML."""
    result = []
    section = ""
    for line in body.splitlines():
        if line in {"server:", "client:", "not_scored:"}:
            section = line[:-1]
        elif line.startswith("  - "):
            value = line[4:]
            if section in {"server", "client"}:
                result.append({"name": value, "role": section, "scoring": "required"})
            elif section == "not_scored" and value.startswith("scenario: "):
                result.append({"name": value[10:], "role": "", "scoring": ""})
            else:
                raise ValueError("unsupported frozen scenario list")
        elif line.startswith("    leg: "):
            result[-1]["role"] = line[9:]
        elif line.startswith("    reason: "):
            result[-1]["scoring"] = line[12:]
    if not result or any(not row["role"] or not row["scoring"] for row in result):
        raise ValueError("incomplete scenario role/scoring inventory")
    keys = {(row["role"], row["name"]) for row in result}
    if len(keys) != len(result):
        raise ValueError("duplicate scenario identity")
    return result


def build_inventory(root: Path, pins: dict) -> dict:
    schema_path = root / "schema/inputs" / pins["mcp-schema"]["revision"] / "schema.json"
    requirements = root / REQUIREMENTS
    if digest(schema_path) != pins["mcp-schema"]["sha256"]:
        raise ValueError("schema identity mismatch")
    if digest(requirements) != pins["conformance"]["requirements-sha256"]:
        raise ValueError("requirements identity mismatch")
    definitions = json.loads(schema_path.read_text())["$defs"]
    if len(definitions) != pins["mcp-schema"]["definitions"]:
        raise ValueError("schema definition inventory mismatch")
    obligations = []
    for line in (root / PLAN).read_text().splitlines():
        cells = [cell.strip() for cell in line.strip("|").split("|")]
        if cells and re.fullmatch(r"(?:D\d{2}|G\d{2}|CI\d{2})", cells[0]):
            obligations.append({"id": cells[0], "owner": cells[1].replace("`", ""),
                                "expected": cells[-1], "source": PLAN + "#verification",
                                "status": "out-of-scope" if cells[0] == "CI08" else "unqualified"})
    expected = {f"D{i:02}" for i in range(1, 7)} | {f"G{i:02}" for i in range(1, 26)} | {f"CI{i:02}" for i in range(1, 9)}
    if {row["id"] for row in obligations} != expected or len(obligations) != len(expected):
        raise ValueError("missing or duplicate decision/qualification obligation")
    definition_rows, method_rows = [], []
    usage = definition_usage(definitions)
    for name, node in sorted(definitions.items()):
        reference = f"schema/inputs/{pins['mcp-schema']['revision']}/schema.json#/$defs/{name}"
        definition_rows.append({"name": name, "source": reference,
                                "role": "method contract" if usage[name] else "protocol framing or reusable value",
                                "methods": usage[name],
                                "direction": sorted({METHOD_DIRECTIONS[m] for m in usage[name]}) or ["both peers: framing/value validation"],
                                "transport": "stdio and HTTP",
                                "owner": "R2: TestRepairA10GeneratedContract",
                                "expected": "explicit reviewed binding preserving schema semantics",
                                "status": "unqualified"})
        method = node.get("properties", {}).get("method", {}).get("const")
        if method and name != "ClientNotification":
            direction = METHOD_DIRECTIONS[method]
            method_rows.append({"name": method, "definition": name, "source": reference,
                                "role": "client and server", "direction": direction,
                                "transport": "stdio and HTTP", "owner": "R5: TestRepairA19CanonicalRuntime",
                                "expected": "supported modern path or explicit legacy rejection; preserve owned fulfillment",
                                "status": "unqualified"})
    scenario_rows = scenarios(requirements.read_text())
    for row in scenario_rows:
        row.update(source=f"conformance@{pins['conformance']['commit']}/{pins['conformance']['requirements-file']}",
                   direction="peer exercises implementation's " + row["role"] + " role",
                   transport="HTTP (official runner); applicable stdio parity owned by G08/D06",
                   owner="Stage 9: official scenario " + row["name"],
                   expected="execute and report; require pass" if row["scoring"] == "required" else "execute and report without counting toward pass rate",
                   status="unqualified")
    return {"version": 1, "qualification": "incomplete; ownership is not execution evidence",
            "schema-sha256": pins["mcp-schema"]["sha256"],
            "requirements-sha256": pins["conformance"]["requirements-sha256"],
            "obligations": sorted(obligations, key=lambda row: row["id"]),
            "definitions": definition_rows, "methods": method_rows, "scenarios": scenario_rows}


def check_inventory(root: Path, pins: dict) -> int:
    try:
        expected = build_inventory(root, pins)
        actual = json.loads((root / INVENTORY).read_text())
        if actual != expected:
            raise ValueError("missing, altered or stale ownership entry; review inventory diff")
        counts = ", ".join(f"{len(expected[key])} {key}" for key in ("obligations", "definitions", "methods", "scenarios"))
        return report("inventory", [], counts + "; product remains unqualified")
    except (OSError, ValueError, KeyError, TypeError) as exc:
        return report("inventory", [str(exc)])


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    pins = json.loads((root / "tools/versions.json").read_text())
    if args.write:
        content = json.dumps(build_inventory(root, pins), indent=2, sort_keys=True) + "\n"
        target = root / INVENTORY
        if not target.exists() or target.read_text() != content:
            target.write_text(content)
    return check_inventory(root, pins)


if __name__ == "__main__":
    raise SystemExit(main())
