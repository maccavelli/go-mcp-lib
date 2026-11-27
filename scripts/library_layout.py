"""Runtime package ownership and non-mutating relocation preflight."""

from __future__ import annotations

import hashlib
import os
from pathlib import Path

MODULE = "github.com/maccavelli/go-mcp-lib"
PACKAGES = {"mcp": "mcp", "mcp/internal/sdkbridge": "sdkbridge",
            "codec": "codec", "protocol": "protocol", "internal/validation": "validation"}


def module_path(path: Path) -> str:
    for line in path.read_text().splitlines():
        fields = line.split()
        if fields[:1] == ["module"] and len(fields) == 2:
            return fields[1]
    raise ValueError("missing module directive")


def module_files(root: Path) -> set[str]:
    paths = set()
    for directory, children, names in os.walk(root):
        base = Path(directory)
        children[:] = [name for name in children if not name.startswith((".", "_"))
                       and name not in {"vendor", "node_modules"} and not (base / name).is_symlink()]
        if "go.mod" in names:
            paths.add((base / "go.mod").relative_to(root).as_posix())
    return paths


def validate(root: Path, packages: list[dict]) -> list[str]:
    """Consume the existing Go metadata query; do not run another source scan."""
    errors = []
    root_files = sorted(path.name for path in root.glob("*.go"))
    if root_files:
        errors.append("root Go files are forbidden: " + ", ".join(root_files))
    if module_files(root) != {"go.mod", "cmd/mcpgen/go.mod"}:
        errors.append("module boundary inventory must be root and cmd/mcpgen only")
    for relative, expected in (("go.mod", MODULE), ("cmd/mcpgen/go.mod", MODULE + "/cmd/mcpgen")):
        try:
            if module_path(root / relative) != expected:
                errors.append("module identity changed: " + relative)
        except (OSError, ValueError) as exc:
            errors.append(f"module identity unavailable: {relative}: {exc}")
    by_name = {package["ImportPath"]: package for package in packages}
    runtime = MODULE + "/mcp"
    private = runtime + "/internal/"
    legacy = MODULE + "/internal/sdkbridge"
    for relative, name in PACKAGES.items():
        path = MODULE + "/" + relative
        package = by_name.get(path)
        if package is None:
            errors.append("missing owned package: " + relative)
            continue
        if package.get("Name") != name or not package.get("GoFiles"):
            errors.append("package owner or production source missing: " + relative)
        if relative != "protocol" and not (package.get("TestGoFiles") or package.get("XTestGoFiles")):
            errors.append("colocated tests missing: " + relative)
    for package in packages:
        owner = package["ImportPath"]
        if owner == legacy or owner.startswith(legacy + "/"):
            errors.append("legacy root bridge package is forbidden")
        for imported in package.get("Imports", []):
            reason = ""
            if imported.startswith(MODULE + "/cmd/"):
                reason = "library imports command tooling"
            elif owner in {MODULE + "/codec", MODULE + "/protocol"} and (imported == runtime or imported.startswith(private)):
                reason = "independent capability imports runtime"
            elif owner.startswith(private) and imported == runtime:
                reason = "private bridge imports its owning runtime"
            elif owner != runtime and not owner.startswith(runtime + "/") and imported.startswith(private):
                reason = "sibling imports runtime-private implementation"
            if reason:
                errors.append(f"forbidden import: {owner} -> {imported}: {reason}")
    return errors


def preflight_moves(root: Path, moves: dict[str, str], identities: dict[str, str]) -> list[str]:
    """Reject conflicts and changed inventories before any caller renames files."""
    errors = []
    actual = {path.relative_to(root).as_posix() for path in root.glob("*.go")}
    actual.update(path.relative_to(root).as_posix() for path in (root / "internal/sdkbridge").glob("*.go"))
    if not moves or set(moves) != actual or set(identities) != actual:
        errors.append("unexpected source inventory; preserve all work")
    destination_root = root / "mcp"
    if destination_root.is_symlink() or (destination_root.exists() and
            (not destination_root.is_dir() or any(destination_root.iterdir()))):
        errors.append("destination tree contains unreviewed work")
    for source, destination in moves.items():
        path = Path(source)
        valid = path.suffix == ".go" and path.parent.as_posix() in {".", "internal/sdkbridge"}
        if not valid or destination != "mcp/" + source:
            errors.append("invalid relocation mapping: " + source)
            continue
        original, target = root / source, root / destination
        if original.is_symlink() or not original.is_file():
            errors.append("source is missing or aliased: " + source)
        elif hashlib.sha256(original.read_bytes()).hexdigest() != identities.get(source):
            errors.append("unreviewed source change: " + source)
        for parent in original.parents:
            if parent == root:
                break
            if parent.is_symlink():
                errors.append("source parent conflict: " + source)
                break
        if target.exists() or target.is_symlink():
            errors.append("destination conflict: " + destination)
        for parent in target.parents:
            if parent == root:
                break
            if parent.is_symlink() or (parent.exists() and not parent.is_dir()):
                errors.append("destination parent conflict: " + destination)
                break
    return errors
