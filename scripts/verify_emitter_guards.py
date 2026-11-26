#!/usr/bin/env python3
"""Prove emitter collision and optional-presence checks detect scratch bypasses.

The unchanged controls belong to the generator pre-add suite. This command runs
only deliberately broken copies, so normal validation is not scanned twice.
"""

from pathlib import Path
import shutil
import tempfile

from gate_common import linter_executable, report, run


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    cases = (
        ("name-collision", "internal/emit/compiler.go", "^TestRepairA10NameGuards$/^case-fold$",
         "if previous, exists := c.claimed[key]; exists {",
         "if previous, exists := c.claimed[key]; exists && false {",
         "FAIL emission guard case-fold"),
        ("optional-empty", "internal/emit/compiler.go", "^TestRepairA10GeneratedWire$",
         'tag += ",omitzero"', 'tag += ",omitempty"',
         "FAIL present empty object omitted"),
        ("ownership-hash", "internal/emit/emit.go", "^TestRepairA10TotalBindings$",
         "manifest.SHA256[file.Name] = hex.EncodeToString(sum[:])",
         'manifest.SHA256[file.Name] = hex.EncodeToString(sum[:]) + "broken"',
         "FAIL manifest content identity"),
        ("binding-omission", "internal/emit/compiler.go", "^TestRepairA10TotalBindings$",
         "c.bindings = append(c.bindings, Binding{name, node.Pointer, goName, disposition})",
         'if name != "JSONValue" { c.bindings = append(c.bindings, Binding{name, node.Pointer, goName, disposition}) }',
         "FAIL incomplete bindings or hashed inventory"),
        ("document-snapshot", "internal/model/model.go", "^TestRepairA10DocumentSnapshot$",
         "Document: append([]byte(nil), doc.Bytes...)", "Document: doc.Bytes",
         "FAIL model schema bytes share mutable loader storage"),
        ("intersection-required", "internal/emit/intersection.go", "^TestRepairA10IntersectionProjection$",
         "required[name] = true", 'if name != "message" { required[name] = true }',
         "FAIL intersection requiredness"),
        ("intersection-dag", "internal/emit/intersection.go", "^TestRepairA10IntersectionProjection$/^shared-reference-dag$",
         "if visited[node] {", "if false {",
         "FAIL shared intersection DAG expanded repeatedly"),
        ("alias-identity", "internal/emit/compiler.go", "^TestRepairA10GeneratedWire$",
         'typeDecl(goName, selector("mcp", name), comment, true)',
         'typeDecl(goName, selector("mcp", name), comment, false)',
         "as mcp."),
        ("generated-deprecation", "internal/emit/emit.go", "^TestRepairA10GeneratedWire$",
         "files = append(files, File{item.name, body})",
         'if item.name == "aliases.go" { body = append(body, []byte("\\nfunc GeneratedDeprecatedProbe() any { return new(mcp.CreateMessageResult) }\\n")...) }; files = append(files, File{item.name, body})',
         "SA1019"),
        ("field-initialism", "internal/emit/emit.go", "^TestKeywordNamesSafe$",
         '"id": "ID"', '"id": "Id"',
         "FAIL Go field initialism"),
    )
    members = ("go.mod", "go.sum", "tools/versions.json", "schema/generation-profile.json",
               "schema/inputs/2026-07-28/schema.json", "schema/inputs/2026-07-28/manifest.json",
               "schema/sdk-map.json", "schema/semantics.json", ".golangci.yml", "protocol/doc.go")
    failures = 0
    for name, relative_source, test, anchor, replacement, diagnostic in cases:
        with tempfile.TemporaryDirectory(prefix="mcp-emitter-plant-") as temporary:
            candidate = Path(temporary)
            module = candidate / "cmd/mcpgen"
            shutil.copytree(root / "cmd/mcpgen", module,
                            ignore=shutil.ignore_patterns(".cache", "mcpgen", "*.test", "*.exe"))
            shutil.copytree(root / "internal/validation", candidate / "internal/validation")
            for relative in members:
                destination = candidate / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / relative, destination)
            source = module / relative_source
            text = source.read_text()
            if text.count(anchor) != 1:
                failures += report(f"emitter.{name}.plant-shape", ["guard anchor changed"])
                continue
            source.write_text(text.replace(anchor, replacement, 1))
            result = run(["go", "test", "-vet=off", "-count=1", "-run", test,
                          "./internal/emit"], module, timeout=600,
                         env_updates={"GO_MCP_LINTER": linter_executable(root)})
            output = result.stdout + result.stderr
            caught = result.returncode != 0 and diagnostic in output
            failures += report(f"emitter.{name}.plant", [] if caught else [output],
                               f"bypass caught: {diagnostic}")
    return int(bool(failures))


if __name__ == "__main__":
    raise SystemExit(main())
