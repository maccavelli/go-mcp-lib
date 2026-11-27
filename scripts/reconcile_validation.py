"""Reconcile the original qualification cases with the immutable vendored corpus.

The original SDK-aligned dependency is an independent oracle, never a runtime
dependency on this script. Numbers compare as Decimal, without binary rounding.
"""

import argparse
from decimal import Decimal
import hashlib
import json
from pathlib import Path

FILES = "defs ref type const enum properties required additionalProperties items maxItems minimum maximum allOf anyOf".split()


def load(path: Path):
    return json.loads(path.read_text(), parse_int=Decimal, parse_float=Decimal)


def schema_identity(schema):
    # The vendored suite explicitly adds the same default dialect at its root.
    if isinstance(schema, dict):
        result = dict(schema)
        result.pop("$schema", None)
        return result
    return schema


def reconcile(source: Path, target: Path) -> tuple[dict, list[str]]:
    result = {"source": "github.com/google/jsonschema-go@v0.4.3/jsonschema/testdata/draft2020-12",
              "upstream-commit": "83e866b46c9f9e7082fd51e83a61c5f2145a1ab7", "files": {}, "cases": []}
    errors = []
    for name in FILES:
        filename = name + ".json"
        old_path, new_path = source / filename, target / filename
        try:
            old, new = load(old_path), load(new_path)
            result["files"][filename] = {"original-sha256": hashlib.sha256(old_path.read_bytes()).hexdigest(),
                                         "vendored-sha256": hashlib.sha256(new_path.read_bytes()).hexdigest()}
            current = {g["description"]: g for g in new}
            for group in old:
                other = current[group["description"]]
                if schema_identity(group["schema"]) != schema_identity(other["schema"]):
                    raise ValueError("schema changed: " + group["description"])
                tests = {t["description"]: t for t in other["tests"]}
                for case in group["tests"]:
                    matched = tests[case["description"]]
                    if case["data"] != matched["data"] or case["valid"] != matched["valid"]:
                        raise ValueError("instance/outcome changed: " + case["description"])
                    result["cases"].append({"file": filename, "group": group["description"],
                                            "test": case["description"], "valid": case["valid"]})
        except (OSError, KeyError, ValueError) as error:
            errors.append(f"FAIL reconcile {filename}: {error}")
    if len(result["cases"]) != 421:
        errors.append(f"FAIL reconcile original cases: {len(result['cases'])} != 421")
    return result, errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True, help="prepared original draft2020-12 fixture directory")
    parser.add_argument("--write", action="store_true", help="write the independently reconciled ledger")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    target = root / "internal/validation/testdata/draft2020-12"
    result, errors = reconcile(args.source, target)
    path = target.parent / "original-assertions.json"
    if not errors:
        body = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
        if args.write:
            path.write_text(body)
            if path.read_text() != body:
                errors.append("FAIL reconcile ledger write did not land")
        elif path.read_text() != body:
            errors.append("FAIL reconcile ledger differs from independent source")
    for error in errors:
        print(error)
    if not errors:
        print("pass validation.reconciliation: 421 original schema/instance/outcome identities preserved")
    return int(bool(errors))


if __name__ == "__main__":
    raise SystemExit(main())
