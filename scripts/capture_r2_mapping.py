#!/usr/bin/env python3
"""Capture fresh compiler-backed R2 mapping evidence and exercise its guards."""

import copy
import hashlib
import json
from pathlib import Path

from gate_common import report, run


def validate(root: Path, envelope: dict) -> list[str]:
    if not isinstance(envelope, dict):
        return ["inspect envelope must be an object"]
    problems = []
    pins = json.loads((root / "tools/versions.json").read_text())
    pin = pins["mcp-schema"]
    schema = root / "schema/inputs" / pin["revision"] / "schema.json"
    schema_bytes = schema.read_bytes()
    definitions = set(json.loads(schema_bytes)["$defs"])
    if hashlib.sha256(schema_bytes).hexdigest() != pin["sha256"]:
        problems.append("schema bytes differ from independent pin")
    if envelope.get("version") != 1 or envelope.get("command") != "inspect" or envelope.get("status") != "ok":
        problems.append("invalid inspect envelope")
    data = envelope.get("data", {})
    if not isinstance(data, dict):
        return problems + ["inspect data must be an object"]
    if any(data.get(key) != value for key, value in (
        ("revision", pin["revision"]), ("mapping-revision", pin["revision"]),
        ("schema-sha256", pin["sha256"]), ("source-commit", pin["upstream-commit"]),
        ("definitions", pin["definitions"]),
    )):
        problems.append("schema provenance mismatch")
    audit = data.get("sdk-audit", {})
    if not isinstance(audit, dict):
        return problems + ["SDK audit must be an object"]
    if audit.get("revision") != pin["revision"] or audit.get("sdk-version") != "v1.8.0" or audit.get("sdk-module") != "github.com/modelcontextprotocol/go-sdk":
        problems.append("SDK mapping identity mismatch")
    digest = audit.get("sdk-context-sha256", "")
    if not isinstance(digest, str) or len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
        problems.append("missing SDK context identity")
    sums = [line.split()[2] for line in (root / "go.sum").read_text().splitlines()
            if line.startswith("github.com/modelcontextprotocol/go-sdk v1.8.0 ")]
    if len(sums) != 1 or audit.get("sdk-module-sum") != sums[0]:
        problems.append("SDK checksum mismatch")
    dispositions = audit.get("dispositions", {})
    if not isinstance(dispositions, dict):
        return problems + ["dispositions must be an object"]
    if set(dispositions) != definitions:
        problems.append("missing or extra definition disposition")
    if any(value not in {"alias", "owned", "owned-codec"} for value in dispositions.values()):
        problems.append("unknown disposition")
    if any(dispositions.get(name) == "alias" for name in audit.get("sdk-deprecations-verified", [])):
        problems.append("deprecated alias accepted")
    if any(dispositions.get(name) != "owned-codec" for name in ("CreateMessageResult", "ListRootsResult")):
        problems.append("required codec ownership changed")
    return problems


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    result = run(["go", "-C", "cmd/mcpgen", "run", ".", "inspect", "--repo", "../..", "--format", "json"], root)
    if result.returncode:
        return int(bool(report("mapping.inspect", [result.stdout + result.stderr])))
    try:
        document = json.loads(result.stdout)
        failures = report("mapping.control", validate(root, document))
        changes = (
            ("missing-definition", lambda doc: doc["data"]["sdk-audit"]["dispositions"].pop("Tool"), "missing or extra"),
            ("unknown-disposition", lambda doc: doc["data"]["sdk-audit"]["dispositions"].update(Tool="skip"), "unknown disposition"),
            ("deprecated-alias", lambda doc: doc["data"]["sdk-audit"]["dispositions"].update(CreateMessageResult="alias"), "deprecated alias"),
            ("SDK-version", lambda doc: doc["data"]["sdk-audit"].update({"sdk-version": "v0.0.0"}), "SDK mapping identity"),
            ("SDK-checksum", lambda doc: doc["data"]["sdk-audit"].update({"sdk-module-sum": "h1:wrong"}), "SDK checksum"),
            ("SDK-context", lambda doc: doc["data"]["sdk-audit"].update({"sdk-context-sha256": ""}), "SDK context identity"),
            ("source-commit", lambda doc: doc["data"].update({"source-commit": "unreviewed"}), "provenance mismatch"),
            ("malformed-audit", lambda doc: doc["data"].update({"sdk-audit": None}), "SDK audit must be an object"),
            ("malformed-dispositions", lambda doc: doc["data"]["sdk-audit"].update(dispositions=None), "dispositions must be an object"),
        )
        for name, mutate, expected in changes:
            broken = copy.deepcopy(document)
            mutate(broken)
            errors = validate(root, broken)
            caught = any(expected in error for error in errors)
            failures += report(f"mapping.plant.{name}", [] if caught else [f"expected {expected}: {errors}"], "broken report rejected")
        if not failures:
            path = root / ".cache/r2-sdk-audit.json"
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps(document, indent=2, sort_keys=True) + "\n")
            print("R2_AUDIT_JSON_BEGIN")
            print(path.read_text(), end="")
            print("R2_AUDIT_JSON_END")
        return int(bool(failures))
    except (KeyError, TypeError, ValueError, OSError) as exc:
        return int(bool(report("mapping.capture", [str(exc)])))


if __name__ == "__main__":
    raise SystemExit(main())
