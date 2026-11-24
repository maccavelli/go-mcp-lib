"""Tests for scripts/gate.py and scripts/go-precheck.py orchestration.

Controlled subprocess fixtures only: no network, no shell, argument arrays.
Each test runs the scripts against a prepared scratch tree and asserts the
named FAIL/pass lines and exit codes, including the honest-incomplete
contract: an unimplemented stage or absent inventory never reports success.
"""

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
GATE = ROOT / "scripts" / "gate.py"
PRECHECK = ROOT / "scripts" / "go-precheck.py"


def run_script(
    script: Path, args: list[str], cwd: Path, env: dict[str, str] | None = None
) -> tuple[int, str]:
    full_env = dict(os.environ)
    if env:
        full_env.update(env)
    proc = subprocess.run(
        [sys.executable, str(script), *args],
        cwd=cwd,
        env=full_env,
        capture_output=True,
        text=True,
    )
    return proc.returncode, proc.stdout + proc.stderr


def run_gate(args: list[str], root: Path) -> tuple[int, str]:
    """Run gate.py against an explicit tree under test."""
    return run_script(GATE, args, cwd=root, env={"GO_MCP_GATE_ROOT": str(root)})


def make_modules(base: Path, source: str, gen_source: str | None = None) -> Path:
    """Write the enumerated module layout: runtime at ., generator at cmd/mcpgen."""
    (base / "internal").mkdir(parents=True, exist_ok=True)
    (base / "go.mod").write_text("module example.invalid/root\n\ngo 1.27\n")
    (base / "internal" / "source.go").write_text(source)
    gen = base / "cmd" / "mcpgen"
    gen.mkdir(parents=True, exist_ok=True)
    (gen / "go.mod").write_text("module example.invalid/gen\n\ngo 1.27\n")
    (gen / "main.go").write_text(gen_source or "package main\n\nfunc main() {}\n")
    return base


BAD_SOURCE = "package internal\n\nfunc Broken() { return }\n"
GOOD_SOURCE = "package internal\n\nfunc Ok() int { return 1 }\n"

# A deliberately deprecated SDK use: CreateMessageHandler is SA1019-marked
# in the pinned SDK v1.8.0. The lint gate must catch it with no exclusion.
DEPRECATED_SOURCE = """package internal

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Handler(opts *mcp.ClientOptions) {
	opts.CreateMessageHandler = func(
		_ context.Context, _ *mcp.CreateMessageRequest,
	) (*mcp.CreateMessageResult, error) {
		return &mcp.CreateMessageResult{}, nil
	}
}
"""


class GateModes(unittest.TestCase):
    """gate.py mode dispatch, honest-incomplete, and unit verification."""

    def test_unknown_mode_exits_2(self):
        code, out = run_script(GATE, ["not-a-mode"], ROOT)
        self.assertEqual(code, 2)
        self.assertIn("modes:", out.lower())
        self.assertIn("pre-add", out)

    def test_unimplemented_modes_fail_incomplete(self):
        for mode in ("generate-check", "api", "wire", "qualification"):
            with self.subTest(mode=mode):
                code, out = run_script(GATE, [mode], ROOT)
                self.assertEqual(code, 1)
                self.assertIn(f"FAIL {mode}: incomplete", out)

    def test_setup_offline_fails_conformance(self):
        code, out = run_script(GATE, ["setup"], ROOT)
        self.assertEqual(code, 1)
        self.assertIn("FAIL setup.conformance", out)
        self.assertIn("--allow-network", out)

    def test_unit_passes_on_real_repo(self):
        code, out = run_script(GATE, ["unit"], ROOT)
        self.assertEqual(code, 0)
        self.assertIn("pass unit.runtime", out)
        self.assertIn("pass unit.generator", out)

    def test_unit_empty_module_fails_loudly(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = make_modules(Path(tmp), GOOD_SOURCE)
            code, out = run_gate(["unit"], base)
            self.assertEqual(code, 1)
            self.assertIn("FAIL unit.runtime.empty", out)

    def test_unit_exit_failure_names_module(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = make_modules(Path(tmp), GOOD_SOURCE)
            (base / "internal" / "source_test.go").write_text(
                "package internal\n\nimport \"testing\"\n\n"
                "func TestBroken(t *testing.T) { t.Fatal(\"planted\") }\n",
            )
            code, out = run_gate(["unit"], base)
            self.assertEqual(code, 1)
            self.assertIn("FAIL unit.runtime.exit", out)


class PrecheckModules(unittest.TestCase):
    """go-precheck.py against controlled module fixtures."""

    def test_clean_module_passes(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = make_modules(Path(tmp), GOOD_SOURCE)
            code, out = run_script(PRECHECK, [], base)
            self.assertEqual(code, 0, out)
            self.assertIn("pre-add: all cases pass", out)

    def test_vet_failure_fails_by_name(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = make_modules(
                Path(tmp),
                "package internal\n\nimport \"fmt\"\n\n"
                "func Vet() { fmt.Printf(\"%d\") }\n",
            )
            code, out = run_script(PRECHECK, [], base)
            self.assertEqual(code, 1)
            self.assertIn("FAIL runtime: go vet", out)

    def test_missing_module_fails_by_name(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            # No go.mod anywhere: both enumerated modules are missing.
            code, out = run_script(PRECHECK, [], base)
            self.assertEqual(code, 1)
            self.assertIn("FAIL runtime: missing module at .", out)
            self.assertIn("FAIL generator: missing module at cmd/mcpgen", out)

    def test_test_failure_aggregates(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = make_modules(Path(tmp), GOOD_SOURCE)
            (base / "internal" / "source_test.go").write_text(
                "package internal\n\nimport \"testing\"\n\n"
                "func TestBroken(t *testing.T) { t.Fatal(\"planted\") }\n",
            )
            code, out = run_script(PRECHECK, [], base)
            self.assertEqual(code, 1)
            self.assertIn("FAIL runtime: go test", out)


class VersionsRecord(unittest.TestCase):
    """tools/versions.json shape and pin consistency with the plan."""

    def test_versions_pins_present(self):
        with open(ROOT / "tools" / "versions.json", encoding="utf-8") as f:
            pins = json.load(f)
        self.assertEqual(pins["go"]["toolchain"], "go1.27.4")
        self.assertEqual(
            pins["runtime-dependencies"]["github.com/modelcontextprotocol/go-sdk"]["version"],
            "v1.8.0",
        )
        self.assertEqual(
            pins["lint"]["golangci-lint"]["exclusions"], "none"
        )


if __name__ == "__main__":
    unittest.main()
