"""Vendor the pinned JSON Schema Test Suite corpus.

Copies the draft2020-12 test files and the remotes directory (the local
resources the suite's remote-ref cases point at via localhost:1234)
from the pinned checkout of the suite (commit
83e866b46c9f9e7082fd51e83a61c5f2145a1ab7) into the repository, records
each file's SHA-256 in a manifest, and writes the suite license.
Idempotent: re-running refreshes from the source checkout.
"""

import hashlib
import json
import os
import shutil
import sys

SOURCE = "/tmp/jsonschema-testsuite/tests/draft2020-12"
REMOTES = "/tmp/jsonschema-testsuite/remotes"
LICENSE = "/tmp/jsonschema-testsuite/LICENSE"
DEST = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "internal", "validation", "testdata", "draft2020-12",
)
COMMIT = "83e866b46c9f9e7082fd51e83a61c5f2145a1ab7"


def sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def main():
    if not os.path.isdir(SOURCE):
        print(f"source checkout missing: {SOURCE}", file=sys.stderr)
        return 2
    os.makedirs(DEST, exist_ok=True)
    manifest = {
        "suite": "json-schema-org/JSON-Schema-Test-Suite",
        "commit": COMMIT,
        "draft": "draft2020-12",
        "files": {},
    }
    names = sorted(n for n in os.listdir(SOURCE) if n.endswith(".json"))
    for name in names:
        shutil.copy2(os.path.join(SOURCE, name), os.path.join(DEST, name))
        manifest["files"][name] = sha256(os.path.join(DEST, name))
    remotes_dest = os.path.join(DEST, "remotes")
    shutil.rmtree(remotes_dest, ignore_errors=True)
    shutil.copytree(REMOTES, remotes_dest, ignore=shutil.ignore_patterns("draft3", "draft-next", "draft2019-09", "draft4", "draft6", "draft7"))
    for root, _, files in os.walk(remotes_dest):
        for name in sorted(files):
            rel = os.path.relpath(os.path.join(root, name), DEST)
            manifest["files"][rel] = sha256(os.path.join(root, name))
    with open(os.path.join(DEST, "LICENSE"), "w") as f:
        shutil.copyfileobj(open(LICENSE), f)
    manifest["files"]["LICENSE"] = sha256(os.path.join(DEST, "LICENSE"))
    out = os.path.join(DEST, "corpus-manifest.json")
    with open(out, "w") as f:
        json.dump(manifest, f, indent=2, sort_keys=True)
        f.write("\n")
    print(f"vendored {len(names)} files + remotes + LICENSE + manifest")
    return 0


if __name__ == "__main__":
    sys.exit(main())
