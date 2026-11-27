"""Derive an independent schema-location fixture from the immutable source.

This maintenance command does not run in ordinary gates. Tests compare the
reviewed fixture with model-derived coverage, including every location's keys.
"""

import argparse
import hashlib
import json
from pathlib import Path


def locations(schema: dict) -> dict[str, list[str]]:
    result = {}

    def escape(value: str) -> str:
        return value.replace("~", "~0").replace("/", "~1")

    def visit(node: dict, pointer: str) -> None:
        result[pointer] = sorted(node)
        for keyword in ("$defs", "properties"):
            for name, child in node.get(keyword, {}).items():
                visit(child, pointer + "/" + keyword + "/" + escape(name))
        for keyword in ("items", "additionalProperties"):
            child = node.get(keyword)
            if isinstance(child, dict):
                visit(child, pointer + "/" + keyword)
        for keyword in ("allOf", "anyOf", "oneOf"):
            for index, child in enumerate(node.get(keyword, [])):
                visit(child, pointer + "/" + keyword + "/" + str(index))

    visit(schema, "#")
    return result


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    body = (root / "schema/inputs/2026-07-28/schema.json").read_bytes()
    digest = hashlib.sha256(body).hexdigest()
    if digest != "907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551":
        raise ValueError("source schema does not match the reviewed pin")
    nodes = locations(json.loads(body))
    document = {"version": 1, "schema-sha256": digest, "nodes": nodes}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(document, indent=2, sort_keys=True) + "\n")
    print(f"schema node inventory: {len(nodes)} locations including the document root")


if __name__ == "__main__":
    main()
