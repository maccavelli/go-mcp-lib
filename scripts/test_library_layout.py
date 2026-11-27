"""Scratch layout, import, consumer and relocation-conflict controls."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import gate_common as common
import library_layout as layout
from qualify_layout import protected_changes


def fixture(root: Path) -> None:
    (root / "go.mod").write_text(f"module {layout.MODULE}\n\ngo 1.27.0\n")
    generator = root / "cmd/mcpgen"
    generator.mkdir(parents=True)
    (generator / "go.mod").write_text(f"module {layout.MODULE}/cmd/mcpgen\n\ngo 1.27.0\n")
    for relative, name in layout.PACKAGES.items():
        directory = root / relative
        directory.mkdir(parents=True)
        (directory / "owner.go").write_text(f"package {name}\n")
        if relative != "protocol":
            (directory / "owner_test.go").write_text(f"package {name}\nimport \"testing\"\nfunc TestOwner(t *testing.T) {{}}\n")


class TestLibraryLayout(unittest.TestCase):
    def caught(self, case, errors, expected):
        found = [error for error in errors if expected in error]
        self.assertTrue(found, f"FAIL {case}: intended diagnostic absent: {errors}")
        print(f"CONTROL {case}: FAIL {found[0]}")

    def test_valid_inventory_and_broken_ownership(self):
        for case in ("valid", "root-file", "missing-package", "wrong-owner", "missing-production", "missing-tests", "extra-module", "module-identity", "malformed-module", "legacy-bridge"):
            with self.subTest(case=case), tempfile.TemporaryDirectory(prefix="mcp-layout-control-") as temporary:
                root = Path(temporary)
                fixture(root)
                expected = ""
                if case == "root-file":
                    (root / "old.go").write_text("package old\n")
                    expected = "root Go files are forbidden"
                elif case == "missing-package":
                    (root / "mcp").rename(root / "other")
                    expected = "missing owned package: mcp"
                elif case == "wrong-owner":
                    (root / "protocol/owner.go").write_text("package wrong\n")
                    expected = "package owner or production source missing: protocol"
                elif case == "missing-tests":
                    (root / "mcp/owner_test.go").unlink()
                    expected = "colocated tests missing: mcp"
                elif case == "missing-production":
                    (root / "mcp/owner.go").unlink()
                    expected = "package owner or production source missing: mcp"
                elif case == "extra-module":
                    (root / "extra").mkdir()
                    (root / "extra/go.mod").write_text("module example.invalid/extra\n")
                    expected = "module boundary inventory"
                elif case == "module-identity":
                    (root / "cmd/mcpgen/go.mod").write_text("module example.invalid/other\n")
                    expected = "module identity changed: cmd/mcpgen/go.mod"
                elif case == "malformed-module":
                    (root / "cmd/mcpgen/go.mod").write_text("invalid module fixture\n")
                    expected = "module identity unavailable: cmd/mcpgen/go.mod"
                elif case == "legacy-bridge":
                    (root / "internal/sdkbridge").mkdir()
                    (root / "internal/sdkbridge/old.go").write_text("package sdkbridge\n")
                    expected = "legacy root bridge package is forbidden"
                names, no_tests, errors = common.package_inventory(root, check_layout=True)
                if expected:
                    self.caught(case, errors, expected)
                else:
                    self.assertEqual(errors, [])
                    self.assertEqual(names, {layout.MODULE + "/" + name for name in layout.PACKAGES})
                    self.assertEqual(no_tests, {layout.MODULE + "/protocol"})

    def test_forbidden_import_edges(self):
        cases = (("codec-runtime", "codec", "mcp", "independent capability imports runtime"),
                 ("protocol-runtime", "protocol", "mcp", "independent capability imports runtime"),
                 ("bridge-owner", "mcp/internal/sdkbridge", "mcp", "private bridge imports its owning runtime"),
                 ("command-import", "codec", "cmd/helper", "library imports command tooling"))
        for case, owner, imported, expected in cases:
            with self.subTest(case=case), tempfile.TemporaryDirectory(prefix="mcp-edge-control-") as temporary:
                root = Path(temporary)
                fixture(root)
                if imported == "cmd/helper":
                    (root / imported).mkdir()
                    (root / imported / "helper.go").write_text("package helper\n")
                name = layout.PACKAGES[owner]
                (root / owner / "owner.go").write_text(f'package {name}\nimport _ "{layout.MODULE}/{imported}"\n')
                self.caught(case, common.package_inventory(root, check_layout=True)[2], expected)

    def test_private_import_edge_metadata_controls(self):
        # The compiler owns internal access. Pin the policy diagnostic as well
        # with exact Go metadata-shaped fixtures rather than another live query.
        with tempfile.TemporaryDirectory(prefix="mcp-private-edge-") as temporary:
            root = Path(temporary)
            fixture(root)
            for owner, expected in (("codec", "independent capability imports runtime"),
                                    ("protocol", "independent capability imports runtime"),
                                    ("sibling", "sibling imports runtime-private implementation")):
                with self.subTest(owner=owner):
                    packages = [{"ImportPath": layout.MODULE + "/" + owner,
                                 "Imports": [layout.MODULE + "/mcp/internal/sdkbridge"]}]
                    self.caught(owner + "-private", layout.validate(root, packages), expected)

    def test_public_consumer_and_legacy_refusal(self):
        with tempfile.TemporaryDirectory(prefix="mcp-consumer-control-") as temporary:
            root = Path(temporary) / "library"
            root.mkdir()
            fixture(root)
            consumer = Path(temporary) / "consumer"
            consumer.mkdir()
            replacement = json.dumps(str(root))
            (consumer / "go.mod").write_text(f"module example.invalid/consumer\n\ngo 1.27.0\nrequire {layout.MODULE} v0.0.0\nreplace {layout.MODULE} => {replacement}\n")
            source = consumer / "consumer.go"
            source.write_text("package consumer\nimport (\n" + "".join(f'_ "{layout.MODULE}/{name}"\n' for name in ("mcp", "codec", "protocol")) + ")\n")
            result = common.run(["go", "build", "."], consumer)
            self.assertEqual(result.returncode, 0, "FAIL canonical consumer: " + result.stdout + result.stderr)
            source.write_text(f'package consumer\nimport _ "{layout.MODULE}"\n')
            result = common.run(["go", "build", "."], consumer)
            output = result.stdout + result.stderr
            self.assertNotEqual(result.returncode, 0, "FAIL legacy-root consumer accepted")
            self.assertIn("cannot find module providing package " + layout.MODULE, output)
            self.assertIn("import lookup disabled by -mod=readonly", output)
            print("CONTROL legacy-consumer: FAIL compiler refused the absent root package")

    def test_owner_and_sibling_internal_consumers(self):
        with tempfile.TemporaryDirectory(prefix="mcp-internal-consumer-") as temporary:
            root = Path(temporary)
            fixture(root)
            imported = layout.MODULE + "/mcp/internal/sdkbridge"
            (root / "mcp/owner.go").write_text(f'package mcp\nimport _ "{imported}"\n')
            result = common.run(["go", "build", "./mcp"], root)
            self.assertEqual(result.returncode, 0, "FAIL owner internal import: " + result.stdout + result.stderr)
            (root / "sibling").mkdir()
            (root / "sibling/consumer.go").write_text(f'package sibling\nimport _ "{imported}"\n')
            result = common.run(["go", "build", "./sibling"], root)
            self.assertNotEqual(result.returncode, 0, "FAIL sibling internal import accepted")
            self.assertIn("use of internal package", result.stdout + result.stderr)
            self.assertIn("not allowed", result.stdout + result.stderr)
            print("CONTROL sibling-internal-consumer: FAIL compiler enforced runtime-private scope")

    def test_metadata_stream_controls(self):
        good = json.dumps({"ImportPath": "fixture/p", "GoFiles": ["owner.go"]})
        for case, body in (("malformed", "{"), ("non-object", "[]"), ("missing-name", "{}"),
                           ("duplicate", good + good), ("wrong-files", good.replace('["owner.go"]', '"owner.go"'))):
            with self.subTest(case=case), patch.object(common, "run", return_value=subprocess.CompletedProcess([], 0, body, "")):
                self.caught(case, common.package_inventory(Path.cwd())[2], "malformed package inventory")

    def test_protected_baseline_control(self):
        repository = Path(__file__).resolve().parent.parent
        with tempfile.TemporaryDirectory(prefix="mcp-protected-control-") as temporary:
            root = Path(temporary) / "snapshot"
            # Reuse the existing object database without a scratch commit or
            # hook override; only the fresh clone's files are ever changed.
            result = common.run(["git", "clone", "--shared", "--quiet", str(repository), str(root)], repository)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(protected_changes(root, "HEAD"), [])
            self.caught("missing-baseline", protected_changes(root, "invalid-baseline-control"), "protected baseline unavailable")
            source = root / "protocol/doc.go"
            source.write_text(source.read_text() + "\n// Deliberate scratch contract edit.\n")
            self.caught("protected-contract", protected_changes(root, "HEAD"), "protected contract")


class TestLibraryLayoutConflict(unittest.TestCase):
    def test_preflight_preserves_all_work(self):
        for case in ("valid", "destination", "unrelated-destination", "parent", "edited-source", "extra-source", "missing-source", "mapping", "source-alias", "destination-alias", "parent-alias"):
            with self.subTest(case=case), tempfile.TemporaryDirectory(prefix="mcp-move-control-") as temporary:
                root = Path(temporary)
                source = root / "owner.go"
                source.write_text("package mcp\n")
                moves = {"owner.go": "mcp/owner.go"}
                identities = {"owner.go": hashlib.sha256(source.read_bytes()).hexdigest()}
                expected = ""
                if case == "destination":
                    (root / "mcp").mkdir()
                    (root / "mcp/owner.go").write_text("preserve destination\n")
                    expected = "destination conflict"
                elif case == "unrelated-destination":
                    (root / "mcp").mkdir()
                    (root / "mcp/unrelated.txt").write_text("preserve unrelated work\n")
                    expected = "destination tree contains unreviewed work"
                elif case == "parent":
                    (root / "mcp").write_text("preserve parent\n")
                    expected = "destination parent conflict"
                elif case == "edited-source":
                    source.write_text("preserve edited source\n")
                    expected = "unreviewed source change"
                elif case == "extra-source":
                    (root / "extra.go").write_text("preserve extra source\n")
                    expected = "unexpected source inventory"
                elif case == "missing-source":
                    source.unlink()
                    expected = "unexpected source inventory"
                elif case == "mapping":
                    moves["owner.go"] = "../outside.go"
                    expected = "invalid relocation mapping"
                elif case == "source-alias":
                    (root / "target.txt").write_bytes(source.read_bytes())
                    source.unlink()
                    source.symlink_to("target.txt")
                    expected = "source is missing or aliased"
                elif case == "destination-alias":
                    (root / "mcp").mkdir()
                    (root / "target.txt").write_text("preserve alias target\n")
                    (root / "mcp/owner.go").symlink_to("../target.txt")
                    expected = "destination conflict"
                elif case == "parent-alias":
                    (root / "elsewhere").mkdir()
                    (root / "mcp").symlink_to("elsewhere", target_is_directory=True)
                    expected = "destination parent conflict"
                before = self.inventory(root)
                errors = layout.preflight_moves(root, moves, identities)
                self.assertEqual(before, self.inventory(root))
                if expected:
                    self.assertTrue(any(expected in error for error in errors), f"FAIL {case}: {errors}")
                    print(f"CONTROL move-{case}: FAIL {expected}; all source/destination bytes preserved")
                else:
                    self.assertEqual(errors, [])

    @staticmethod
    def inventory(root):
        return {path.relative_to(root): ("link", str(path.readlink())) if path.is_symlink()
                else ("file", path.read_bytes()) if path.is_file() else ("directory",)
                for path in root.rglob("*")}

    def test_aliased_source_parent_preserves_work(self):
        with tempfile.TemporaryDirectory(prefix="mcp-source-parent-control-") as temporary:
            root = Path(temporary)
            (root / "store/sdkbridge").mkdir(parents=True)
            (root / "store/sdkbridge/codec.go").write_text("package sdkbridge\n")
            (root / "internal").symlink_to("store", target_is_directory=True)
            source = "internal/sdkbridge/codec.go"
            before = self.inventory(root)
            errors = layout.preflight_moves(root, {source: "mcp/" + source},
                                            {source: hashlib.sha256((root / source).read_bytes()).hexdigest()})
            self.assertTrue(any("source parent conflict" in error for error in errors), errors)
            self.assertEqual(before, self.inventory(root))
            print("CONTROL source-parent-alias: FAIL source parent conflict; all work preserved")


if __name__ == "__main__":
    unittest.main(verbosity=2)
