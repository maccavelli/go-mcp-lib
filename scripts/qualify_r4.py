"""Qualify R4 through existing check owners; optionally prepare reviewed generation.

Run generation preparation on the live checkout under its heavy-run queue.
Ordinary qualification runs on immutable native fleet snapshots. Neither mode
commits, pushes, modifies inputs/pins, or relaxes any existing check.
"""

import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

from gate_common import report, run
from qualify_r1 import source_identity


def command(root: Path, name: str, args: list[str], *, cwd: Path | None = None, network: bool = False):
    print("R4 qualification check", name, flush=True)
    result = run(args, cwd or root, network=network, timeout=1800)
    print(result.stdout, end="", flush=True)
    print(result.stderr, end="", file=sys.stderr, flush=True)
    return result


def go_json(value) -> bytes:
    body = json.dumps(value, ensure_ascii=False, indent=2) + "\n"
    for char, replacement in (("<", r"\u003c"), (">", r"\u003e"), ("&", r"\u0026"), ("\u2028", r"\u2028"), ("\u2029", r"\u2029")):
        body = body.replace(char, replacement)
    return body.encode()


def expected_api_hash(root: Path) -> str:
    original = (root / "protocol/api.json").read_bytes()
    document = json.loads(original)
    if go_json(document) != original:
        raise ValueError("API inventory cannot be reproduced byte-for-byte")
    template = (root / "cmd/mcpgen/internal/emit/support.go.txt").read_text()
    block = template.split("import (\n", 1)[1].split("\n)", 1)[0]
    imports = {}
    for line in block.splitlines():
        if not line.strip():
            continue
        match = re.fullmatch(r'\s*(?:(\w+)\s+)?"([^"]+)"\s*', line)
        if match is None:
            raise ValueError("unrecognized codec template import")
        alias, path = match.groups()
        name = alias or path.rsplit("/", 1)[-1]
        if name in imports:
            raise ValueError("duplicate codec template import")
        imports[name] = path
    affected = 0
    for declaration in document["declarations"]:
        if declaration["file"] == "codecs.go":
            declaration["imports"] = dict(sorted(imports.items()))
            affected += 1
    if not imports or not affected:
        raise ValueError("missing codec template/API inventory")
    return hashlib.sha256(go_json(document)).hexdigest()


def reviewed_migration(output: str, code: int, api_hash: str) -> dict:
    data = json.loads(output)["data"]
    migration = data["migration"]
    targets = {"darwin/arm64", "linux/amd64", "windows/amd64"}
    reports = migration["compile-qualification"]
    if len(reports) != 2 or {r["candidate"] for r in reports} != {".", "cmd/mcpgen"}:
        raise ValueError("missing candidate module qualification")
    for candidate in reports:
        if not candidate["passed"] or len(candidate["targets"]) != 3 or {r["target"] for r in candidate["targets"]} != targets:
            raise ValueError("incomplete target qualification")
        if any(not r["package"] or not r["tests"] for r in candidate["targets"]):
            raise ValueError("failed source/test compilation")
    changed = {name for name in migration["proposed-sha256"]
               if migration["previous-sha256"].get(name) != migration["proposed-sha256"][name]}
    removed = set(migration["previous-sha256"]) - set(migration["proposed-sha256"])
    if changed != {"api.json", "codecs.go", "generated-manifest.json"} or removed:
        raise ValueError("R4 generation must change only serialization support, its exact API imports and its manifest")
    if migration["proposed-sha256"]["api.json"] != api_hash:
        raise ValueError("API drift exceeds the exact codec import metadata update")
    if code != 1 or set(data["drift"]) != changed or data["unowned"] or data["interrupted"]:
        raise ValueError("expected only reviewed serialization drift")
    return migration


def prepare_generation(root: Path) -> int:
    result = command(root, "review-generation", ["go", "run", ".", "check", "--repo", "../..", "--format", "json"], cwd=root / "cmd/mcpgen")
    try:
        migration = reviewed_migration(result.stdout, result.returncode, expected_api_hash(root))
        review = root / "tools/r2-candidate-migration.json"
        archive = root / "tools/r2-candidate-migration-pre-r4.json"
        previous = review.read_bytes()
        if archive.exists() and archive.read_bytes() != previous:
            raise ValueError("existing pre-R4 review archive differs")
        archive.write_bytes(previous)
        body = json.dumps(migration, indent=2) + "\n"
        review.write_text(body)
        if review.read_text() != body or archive.read_bytes() != previous:
            raise ValueError("review write did not land")
        print("R4 reviewed generated manifest", migration["proposed-sha256"]["generated-manifest.json"], flush=True)
    except (KeyError, ValueError, TypeError, OSError) as error:
        return int(bool(report("r4.review-generation", [str(error)])))
    result = command(root, "publish-generation", ["go", "run", ".", "generate", "--repo", "../..", "--format", "json"], cwd=root / "cmd/mcpgen")
    failures = report("r4.publish-generation", [] if result.returncode == 0 else [f"exit {result.returncode}"])
    for name, expected in migration["proposed-sha256"].items():
        path = root / "protocol" / name
        if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != expected:
            failures += report("r4.published-identity", [name])
    return int(bool(failures))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--allow-network", action="store_true", help="explicit pinned preparation only")
    parser.add_argument("--prepare-generation", action="store_true", help="review exact candidate hashes and publish through the qualified CLI")
    parser.add_argument("--verify-after-generation", action="store_true", help="also qualify the prepared generation in this same admitted snapshot")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    if args.verify_after_generation and not args.prepare_generation:
        parser.error("--verify-after-generation requires --prepare-generation")
    if args.prepare_generation:
        failure = prepare_generation(root)
        if failure or not args.verify_after_generation:
            return failure
    before = source_identity(root)
    print("R4 qualification identity", before, flush=True)
    failures = 0
    for name, args_list in (
        ("preparation", [sys.executable, "-B", "scripts/gate.py", "setup", *(["--allow-network"] if args.allow_network else [])]),
        ("pre-add", [sys.executable, "-B", "scripts/go-precheck.py", "--module", "all"]),
        ("raw-plants", [sys.executable, "-B", "scripts/verify_raw_guards.py"]),
        ("validator-plants", [sys.executable, "-B", "scripts/verify_mutations.py"]),
        ("sdk-plants", [sys.executable, "-B", "scripts/verify_sdk_guards.py"]),
        ("review-controls", [sys.executable, "-B", "scripts/test_r4_review.py"]),
        ("generated-check", ["go", "run", ".", "check", "--repo", "../..", "--format", "json"]),
    ):
        result = command(root, name, args_list, cwd=root / "cmd/mcpgen" if name == "generated-check" else root,
                         network=name == "preparation" and args.allow_network)
        failures += report(name, [] if result.returncode == 0 else [f"exit {result.returncode}"])
    failures += report("source-preservation", [] if source_identity(root) == before else ["source changed"])
    print("R4 qualification failures", failures, flush=True)
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
