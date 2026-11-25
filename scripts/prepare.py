"""Explicit, pinned preparation; offline verification never downloads inputs."""

from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import tarfile
import tempfile
import time

from gate_common import MODULES, linter, linter_path, report, run


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def tree_digest(root: Path, *, allow_links: bool = False) -> str:
    value = hashlib.sha256()
    files = sorted(path for path in root.rglob("*") if path.is_file())
    if not files:
        raise ValueError(f"empty artifact tree: {root.name}")
    for path in files:
        if path.is_symlink():
            if not allow_links or not path.resolve().is_relative_to(root.resolve()):
                raise ValueError("artifact symlink is not permitted")
            value.update(path.relative_to(root).as_posix().encode() + b"\0link\0" + os.readlink(path).encode() + b"\0")
            continue
        value.update(path.relative_to(root).as_posix().encode() + b"\0")
        value.update(path.read_bytes() + b"\0")
    return value.hexdigest()


def verify_source(source: Path, pin: dict) -> None:
    for filename, expected in ((pin["requirements-file"], pin["requirements-sha256"]),
                               ("package-lock.json", pin["lockfile-sha256"])):
        if digest(source / filename) != expected:
            raise ValueError(f"conformance {filename} hash mismatch")
    package = json.loads((source / "package.json").read_text())
    if package["version"] != pin["package-version"] or package["bin"]["conformance"] != "dist/index.js":
        raise ValueError("conformance package identity mismatch")


def verify_prepared(source: Path, pin: dict, tools: dict) -> None:
    verify_source(source, pin)
    receipt = json.loads((source / "prepared.json").read_text())
    if receipt.get("pin") != pin or receipt.get("tools") != tools:
        raise ValueError("prepared pin/tool identity mismatch")
    if receipt.get("runner-sha256") != tree_digest(source / "dist"):
        raise ValueError("prepared runner hash mismatch")
    if receipt.get("dependencies-sha256") != tree_digest(source / "node_modules", allow_links=True):
        raise ValueError("prepared runtime dependency hash mismatch")
    if not (source / "dist/index.js").is_file():
        raise ValueError("prepared runner missing")


def checked(args: list[str], directory: Path, *, network: bool = False) -> str:
    result = run(args, directory, network=network)
    if result.returncode:
        raise ValueError(f"{args[0]} {args[1]} exit {result.returncode}: {(result.stdout + result.stderr)[-1500:]}")
    return result.stdout.strip()


def prepare_conformance(root: Path, pin: dict, tools: dict, network: bool) -> Path:
    parent = root / ".cache/prepared"
    destination = parent / ("conformance-" + pin["commit"])
    if destination.exists():
        try:
            verify_prepared(destination, pin, tools)
            return destination
        except (OSError, ValueError, KeyError):
            if not network:
                raise
    if not network:
        raise ValueError("conformance not prepared; run setup --allow-network explicitly")
    parent.mkdir(parents=True, exist_ok=True)
    # Exclusive creation avoids concurrent promotion; interruption leaves a named
    # lock, never a successful receipt. Do not remove another invocation's lock.
    lock = parent / "conformance.lock"
    with lock.open("x") as handle:
        handle.write(f"pid={os.getpid()}\n")
    try:
        if destination.exists():
            destination.rename(parent / (destination.name + f"-invalid-{time.time_ns()}"))
        with tempfile.TemporaryDirectory(prefix="conformance-candidate-", dir=parent) as temporary:
            work = Path(temporary)
            repository = work / "repository"
            repository.mkdir()
            checked(["git", "init", "--quiet"], repository)
            checked(["git", "fetch", "--depth=1", "https://" + pin["repository"] + ".git", pin["commit"]], repository, network=True)
            commit = checked(["git", "rev-parse", "FETCH_HEAD"], repository)
            if commit != pin["commit"]:
                raise ValueError("fetched conformance commit mismatch")
            archive = work / "source.tar"
            checked(["git", "archive", "--format=tar", "--output", str(archive), commit], repository)
            source = work / "source"
            source.mkdir()
            with tarfile.open(archive) as tar:
                for member in tar.getmembers():
                    target = (source / member.name).resolve()
                    if not target.is_relative_to(source.resolve()) or not (member.isfile() or member.isdir()):
                        raise ValueError("unsafe conformance archive member")
                tar.extractall(source, filter="data")
            verify_source(source, pin)
            # Ignore package lifecycle scripts; build exactly once explicitly.
            checked(["npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"], source, network=True)
            verify_source(source, pin)
            checked(["npm", "run", "build"], source)
            verify_source(source, pin)
            checked(["node", "dist/index.js", "--help"], source)
            receipt = {"pin": pin, "tools": tools, "runner-sha256": tree_digest(source / "dist"),
                       "dependencies-sha256": tree_digest(source / "node_modules", allow_links=True)}
            (source / "prepared.json").write_text(json.dumps(receipt, sort_keys=True, indent=2) + "\n")
            verify_prepared(source, pin, tools)
            source.rename(destination)
    finally:
        lock.unlink()
    return destination


def prepare_module(root: Path, relative: str, pins: dict, name: str, network: bool) -> list[str]:
    module = root / relative
    problems = []
    with tempfile.TemporaryDirectory(prefix="mcp-module-prepare-") as temporary:
        candidate = Path(temporary)
        for filename in ("go.mod", "go.sum"):
            if (module / filename).exists():
                shutil.copy2(module / filename, candidate / filename)
        before = {p.name: digest(p) for p in candidate.iterdir()}
        if network:
            result = run(["go", "mod", "download"], candidate, network=True)
            if result.returncode:
                problems.append(result.stdout + result.stderr)
        metadata = run(["go", "list", "-m", "-json", "all"], candidate, network=network)
        if metadata.returncode:
            problems.append(metadata.stdout + metadata.stderr)
        else:
            decoder, offset, modules = json.JSONDecoder(), 0, {}
            body = metadata.stdout
            while offset < len(body):
                offset += len(body[offset:]) - len(body[offset:].lstrip())
                if offset == len(body):
                    break
                entry, offset = decoder.raw_decode(body, offset)
                if entry.get("Replace"):
                    problems.append("unreviewed module replacement")
                modules[entry["Path"]] = entry.get("Version")
            if name == "runtime":
                for path, spec in pins["runtime-dependencies"].items():
                    if modules.get(path) != spec["version"]:
                        problems.append(f"dependency pin mismatch: {path}")
        verify = run(["go", "mod", "verify"], candidate)
        if verify.returncode:
            problems.append(verify.stdout + verify.stderr)
        after = {p.name: digest(p) for p in candidate.iterdir()}
        if after != before:
            problems.append("isolated module metadata changed; review lockfile update")
    return problems


def prepare(root: Path, pins: dict, network: bool) -> int:
    failures = 0
    identities = {}
    specs = (("go", ["go", "version"], pins["go"]["toolchain"]),
             ("node", ["node", "--version"], "v" + pins["conformance"]["node"]),
             ("npm", ["npm", "--version"], pins["conformance"]["npm"]))
    for name, command, expected in specs:
        result = run(command, root)
        actual = result.stdout.strip()
        matches = actual.split()[2:3] == [expected] if name == "go" else actual == expected
        errors = [] if not result.returncode and matches else [f"want {expected}; got {actual!r}; {result.stderr}"]
        failures += report(f"setup.{name}", errors, actual)
        if not errors:
            identities[name] = actual
    executable, errors = linter(root, pins)
    if errors and network and not os.environ.get("GO_MCP_LINTER"):
        destination = linter_path(root)
        destination.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix="linter-candidate-", dir=destination.parent) as temporary:
            candidate = Path(temporary)
            result = run(["go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v" + pins["lint"]["golangci-lint"]["version"]],
                         candidate, network=True, gobin=candidate)
            if result.returncode:
                errors = [f"pinned linter acquisition failed: {result.stdout}{result.stderr}"]
            else:
                built = candidate / destination.name
                version = run([str(built), "--version"], candidate)
                pin = pins["lint"]["golangci-lint"]
                identity = re.search(r"version (\S+) built with go(\S+)", version.stdout)
                if version.returncode or not identity or identity.groups() != (pin["version"], pin["built-with-go"]):
                    errors = ["acquired linter identity mismatch"]
                else:
                    built.replace(destination)
                    executable, errors = linter(root, pins)
    failures += report("setup.linter", errors)
    if not errors:
        try:
            identities["linter-sha256"] = digest(Path(executable))
        except OSError as exc:
            failures += report("setup.linter.identity", [str(exc)])
    for name, relative in MODULES.items():
        try:
            problems = prepare_module(root, relative, pins, name, network)
        except (OSError, ValueError, KeyError, TypeError) as exc:
            problems = [f"invalid module preparation: {exc}"]
        failures += report(f"setup.{name}.dependencies", problems)
    try:
        schema = root / "schema/inputs" / pins["mcp-schema"]["revision"] / "schema.json"
        if digest(schema) != pins["mcp-schema"]["sha256"]:
            raise ValueError("schema input hash mismatch")
        report("setup.schema", [])
    except (OSError, ValueError) as exc:
        failures += report("setup.schema", [str(exc)])
    if len(identities) != 4:
        failures += report("setup.conformance", ["required tool identities unavailable; no preparation attempted"])
    else:
        try:
            source = prepare_conformance(root, pins["conformance"], identities, network)
            report("setup.conformance", [], f"verified commit {pins['conformance']['commit']}; runner {tree_digest(source / 'dist')}")
        except (OSError, ValueError, KeyError, tarfile.TarError) as exc:
            failures += report("setup.conformance", [str(exc)])
    return failures
