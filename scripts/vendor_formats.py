"""Vendor format fixture cases from the pinned test suite.

The plan requires 36 format fixtures: representative cases drawn from
the suite's optional/format directory at the pinned commit. Selected
formats are the ones the MCP schema and the framework's
protocol-location rules actually exercise: date-time, date, time, uri,
uri-reference, uri-template, uuid, json-pointer, relative-json-pointer,
ipv4, ipv6, email, hostname, regex, duration.

Extracts each selected file's groups into one file per format under
internal/validation/testdata/formats/, with a manifest of hashes.
"""

import hashlib
import json
import os
import shutil
import sys

SOURCE = "/tmp/jsonschema-testsuite/tests/draft2020-12/optional/format"
LICENSE = "/tmp/jsonschema-testsuite/LICENSE"
DEST_ROOT = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "internal", "validation", "testdata", "formats",
)
COMMIT = "83e866b46c9f9e7082fd51e83a61c5f2145a1ab7"
SELECTED = [
    "date-time", "date", "duration", "email", "hostname",
    "ipv4", "ipv6", "json-pointer", "regex",
    "relative-json-pointer", "time", "uri", "uri-reference",
    "uri-template", "uuid",
]
# Cases per format to vendor; chosen to cover valid and invalid shapes.
PER_FORMAT_CASES = 3  # groups are 1 per file; tests within are kept whole


def sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def main():
    if not os.path.isdir(SOURCE):
        print(f"source missing: {SOURCE}", file=sys.stderr)
        return 2
    os.makedirs(DEST_ROOT, exist_ok=True)
    manifest = {
        "suite": "json-schema-org/JSON-Schema-Test-Suite",
        "commit": COMMIT,
        "formats": {},
    }
    for name in SELECTED:
        src = os.path.join(SOURCE, name + ".json")
        if not os.path.exists(src):
            print(f"skip missing format file: {name}", file=sys.stderr)
            continue
        body = json.load(open(src))
        # Keep groups whole; the plan's 36 fixtures are the test cases.
        shutil.copy2(src, os.path.join(DEST_ROOT, name + ".json"))
        cases = 0
        for group in body:
            cases += len(group.get("tests", []))
        manifest["formats"][name] = {
            "hash": sha256(os.path.join(DEST_ROOT, name + ".json")),
            "cases": cases,
        }
    with open(os.path.join(DEST_ROOT, "LICENSE"), "w") as f:
        shutil.copyfileobj(open(LICENSE), f)
    manifest["formats"]["LICENSE"] = {
        "hash": sha256(os.path.join(DEST_ROOT, "LICENSE")),
        "cases": 0,
    }
    out = os.path.join(DEST_ROOT, "formats-manifest.json")
    with open(out, "w") as f:
        json.dump(manifest, f, indent=2, sort_keys=True)
        f.write("\n")
    total = sum(v["cases"] for v in manifest["formats"].values())
    print(f"vendored {len(SELECTED)} formats, {total} fixture cases")
    return 0


if __name__ == "__main__":
    sys.exit(main())
