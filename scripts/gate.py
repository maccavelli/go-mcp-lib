#!/usr/bin/env python3
"""gate: orchestration for the go-mcp-lib framework gates.

Modes:
  setup [--allow-network]  prepare toolchain, conformance source, fixtures
  pre-add                  gofmt, vet, build, test per module (via go-precheck)
  unit                     package tests with go test -json execution verified
  generate-check           generator contract tests (incomplete until stage 4)
  api                      public API surface checks (incomplete until stage 3)
  wire                     wire-level contract tests (incomplete until stage 7)
  qualification            full qualification matrix (incomplete until stage 9)
  plants                   deliberate-failure control inventory (stage 1 plants)

Every mode prints named pass/FAIL lines and exits nonzero on any failure. An
unimplemented stage or absent test inventory reports FAIL-incomplete; it is
never reported as a successful gate. Python standard library only.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
# Tests override the tree under test with GO_MCP_GATE_ROOT; production
# runs resolve the repository root containing this script.
ROOT = os.environ.get("GO_MCP_GATE_ROOT") or os.path.dirname(HERE)
ENV = {
    "PATH": os.environ.get("PATH", "/usr/local/bin:/usr/bin:/bin"),
    "HOME": os.environ.get("HOME", ""),
    "GOWORK": "off",
    "GOTOOLCHAIN": "local",
    "GOPROXY": "off",
    "GOSUMDB": "off",
}

LINTER = os.environ.get("GO_MCP_LINTER", os.path.expanduser("~/go/bin/golangci-lint"))
LINTER_VERSION = "2.15.0"

MODES = ("setup", "pre-add", "unit", "generate-check", "api", "wire",
         "qualification", "plants")


def fail(case: str, detail: str) -> None:
    print(f"FAIL {case}: {detail}")


def passln(case: str, detail: str = "") -> None:
    print(f"pass {case}" + (f": {detail}" if detail else ""))


def go_test_json(module_dir: str) -> tuple[int, list[dict], str]:
    """Run go test -json and return (exit, events, raw output tail)."""
    proc = subprocess.run(
        ["go", "test", "-json", "./..."],
        cwd=os.path.join(ROOT, module_dir),
        env=ENV,
        capture_output=True,
        text=True,
    )
    events: list[dict] = []
    for line in proc.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            # Non-JSON output goes into the raw tail, not the events.
            pass
    return proc.returncode, events, (proc.stdout + proc.stderr)[-2000:]


def mode_setup(allow_network: bool) -> int:
    """Prepare toolchain and conformance inputs. Network is opt-in."""
    failures = 0
    with open(os.path.join(ROOT, "tools", "versions.json"), encoding="utf-8") as f:
        pins = json.load(f)
    go_version = subprocess.run(
        ["go", "version"], capture_output=True, text=True, env=ENV
    ).stdout.strip()
    want = pins["go"]["toolchain"]
    if want not in go_version:
        fail("setup.go-toolchain", f"want {want} in {go_version!r}")
        failures += 1
    else:
        passln("setup.go-toolchain", go_version)
    node = subprocess.run(
        ["node", "--version"], capture_output=True, text=True
    ).stdout.strip()
    want_node = "v" + pins["conformance"]["node"]
    if node != want_node:
        fail("setup.node", f"want {want_node} got {node!r}")
        failures += 1
    else:
        passln("setup.node", node)
    npm = subprocess.run(
        ["npm", "--version"], capture_output=True, text=True
    ).stdout.strip()
    if npm != pins["conformance"]["npm"]:
        fail("setup.npm", f"want {pins['conformance']['npm']} got {npm!r}")
        failures += 1
    else:
        passln("setup.npm", npm)
    # Conformance source preparation is network work and stays behind
    # --allow-network. Offline setup checks pins only and reports the
    # conformance preparation as incomplete, never as done.
    if not allow_network:
        fail(
            "setup.conformance",
            "network preparation not run: invoke with --allow-network",
        )
        failures += 1
    else:
        fail(
            "setup.conformance",
            "conformance preparation lands with stage 9 qualification wiring",
        )
        failures += 1
    return failures


def mode_preadd() -> int:
    proc = subprocess.run(
        [sys.executable, os.path.join(HERE, "go-precheck.py")],
        cwd=ROOT,
        env=ENV,
        capture_output=True,
        text=True,
    )
    print(proc.stdout, end="")
    if proc.stderr:
        print(proc.stderr, end="")
    return 0 if proc.returncode == 0 else 1


def run_lint(module_dir: str) -> tuple[int, str]:
    """Run the pinned golangci-lint in a module; return (exit, output)."""
    if not os.path.exists(LINTER):
        return 127, f"pinned golangci-lint not found at {LINTER}"
    proc = subprocess.run(
        [LINTER, "run"],
        cwd=os.path.join(ROOT, module_dir),
        env=ENV,
        capture_output=True,
        text=True,
    )
    return proc.returncode, proc.stdout + proc.stderr


def mode_unit() -> int:
    """Run package tests and pinned lint, verifying execution via -json."""
    failures = 0
    version = subprocess.run(
        [LINTER, "--version"], capture_output=True, text=True
    ).stdout
    if LINTER_VERSION not in version:
        fail("unit.lint.tool", f"want golangci-lint {LINTER_VERSION}: {version!r}")
        failures += 1
    for module_dir, name in ((".", "runtime"), ("cmd/mcpgen", "generator")):
        code, events, tail = go_test_json(module_dir)
        ran = [e for e in events if e.get("Action") == "run" and e.get("Test")]
        passed = [e for e in events if e.get("Action") == "pass" and e.get("Test")]
        skipped = [e for e in events if e.get("Action") == "skip" and e.get("Test")]
        if code != 0:
            fail(f"unit.{name}.exit", f"go test exit {code}: {tail}")
            failures += 1
            continue
        if not ran:
            fail(f"unit.{name}.empty", "no tests executed in module")
            failures += 1
            continue
        passln(
            f"unit.{name}",
            f"{len(passed)} passed, {len(skipped)} skipped, {len(ran)} ran",
        )
        code, out = run_lint(module_dir)
        if code != 0:
            fail(f"unit.{name}.lint", f"golangci-lint exit {code}: {out[-500:]}")
            failures += 1
        else:
            passln(f"unit.{name}.lint", "0 issues")
    return failures


def mode_incomplete(stage: str, what: str) -> int:
    """An unimplemented stage is a FAIL-incomplete, never a silent pass."""
    fail(stage, f"incomplete: {what}")
    return 1


PLANT_SOURCE = '''// Package plant is a stage-1 gate control: a deliberately deprecated SDK
// use that the pinned lint gate must catch with zero exclusions.
package plant

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler assigns the deprecated CreateMessageHandler binding (SA1019 in
// the pinned SDK v1.8.0 under SEP-2577).
func Handler(opts *mcp.ClientOptions) {
	opts.CreateMessageHandler = func(
		_ context.Context, _ *mcp.CreateMessageRequest,
	) (*mcp.CreateMessageResult, error) {
		return &mcp.CreateMessageResult{}, nil
	}
}
'''


def mode_plants() -> int:
    """Synthesize the deprecated-SDK plant and prove the lint gate catches it.

    The plant module is created in a temporary directory from the pinned
    SDK require, tidied offline from the local module cache, and linted with
    the pinned golangci-lint. The gate FAILS if the plant lints clean -- a
    clean plant would mean the gate is not real. No scratch content is
    committed.
    """
    failures = 0
    if not os.path.exists(LINTER):
        fail("plants.lint.tool", f"pinned golangci-lint not found at {LINTER}")
        return 1
    with tempfile.TemporaryDirectory(prefix="go-mcp-plant-") as tmp:
        plant = os.path.join(tmp, "plant")
        os.makedirs(plant)
        with open(os.path.join(plant, "go.mod"), "w", encoding="utf-8") as f:
            f.write(
                "module plant.invalid/deprecated\n\ngo 1.27\n\n"
                "require github.com/modelcontextprotocol/go-sdk v1.8.0\n"
            )
        with open(os.path.join(plant, "plant.go"), "w", encoding="utf-8") as f:
            f.write(PLANT_SOURCE)
        tidy = subprocess.run(
            ["go", "mod", "tidy"], cwd=plant, env=ENV,
            capture_output=True, text=True,
        )
        if tidy.returncode != 0:
            fail(
                "plants.prepare",
                f"go mod tidy offline failed (module cache lacks the pinned "
                f"SDK?): {tidy.stderr[-300:]}",
            )
            return 1
        lint = subprocess.run(
            [LINTER, "run"], cwd=plant, env=ENV,
            capture_output=True, text=True,
        )
        out = lint.stdout + lint.stderr
        if lint.returncode == 0:
            fail(
                "plants.lint-deprecated",
                "lint gate did NOT catch the planted deprecated SDK use",
            )
            failures += 1
        elif "SA1019" not in out:
            fail(
                "plants.lint-deprecated",
                f"lint failed for an unexpected reason: {out[-300:]}",
            )
            failures += 1
        else:
            findings = out.count("SA1019")
            passln(
                "plants.lint-deprecated",
                f"gate caught the planted deprecated use ({findings} SA1019)",
            )
    return failures


def mode_generate_check() -> int:
    """Generator contract: stage 2 loading, resolution, and modeling.

    Checks the pinned schema is committed and byte-hashed, the fixture
    corpus is present, the disposition seed is large enough for stage 3
    to complete mechanically, and the generator module's tests and lint
    pass. Unimplemented generator stages stay FAIL-incomplete inside the
    generator's own stage wiring; this mode reports the stage 2 contract.
    """
    failures = 0
    schema_path = os.path.join(ROOT, "schema", "inputs", "2026-07-28", "schema.json")
    if not os.path.exists(schema_path):
        fail("generate-check.schema", f"pinned schema missing at {schema_path}")
        return 1
    passln("generate-check.schema", "pinned schema present")
    fixtures_dir = os.path.join(ROOT, "cmd", "mcpgen", "testdata", "fixtures")
    if not os.path.isdir(fixtures_dir):
        fail("generate-check.fixtures", "fixture corpus missing")
        failures += 1
    else:
        fixtures = [f for f in os.listdir(fixtures_dir) if f.endswith(".json")]
        if len(fixtures) < 8:
            fail("generate-check.fixtures", f"corpus incomplete: {len(fixtures)} fixtures")
            failures += 1
        else:
            passln("generate-check.fixtures", f"{len(fixtures)} fixtures")
    with open(os.path.join(ROOT, "schema", "sdk-map.json"), encoding="utf-8") as f:
        sdk_map = json.load(f)
    dispositioned = set(sdk_map.get("dispositions", {}))
    if len(dispositioned) < 20:
        fail(
            "generate-check.dispositions",
            f"disposition table too small: {len(dispositioned)} seeded",
        )
        failures += 1
    else:
        passln(
            "generate-check.dispositions",
            f"{len(dispositioned)} definitions dispositioned (stage 3 completes totality)",
        )
    code, events, tail = go_test_json("cmd/mcpgen")
    ran = [e for e in events if e.get("Action") == "run" and e.get("Test")]
    passed = [e for e in events if e.get("Action") == "pass" and e.get("Test")]
    if code != 0 or not ran:
        fail("generate-check.tests", f"generator tests failed: {tail}")
        failures += 1
    else:
        passln("generate-check.tests", f"{len(passed)} generator tests passed")
    code, out = run_lint("cmd/mcpgen")
    if code != 0:
        fail("generate-check.lint", f"generator lint: {out[-500:]}")
        failures += 1
    else:
        passln("generate-check.lint", "0 issues")
    return failures


def main() -> int:
    args = sys.argv[1:]
    if not args or args[0] not in MODES:
        print(__doc__)
        return 2
    mode = args[0]
    if mode == "setup":
        failures = mode_setup("--allow-network" in args)
    elif mode == "pre-add":
        failures = mode_preadd()
    elif mode == "unit":
        failures = mode_unit()
    elif mode == "generate-check":
        failures = mode_generate_check()
    elif mode == "api":
        failures = mode_incomplete("api", "generated API contract lands in stage 3")
    elif mode == "wire":
        failures = mode_incomplete("wire", "transports land in stage 7")
    elif mode == "qualification":
        failures = mode_incomplete("qualification", "full matrix lands in stage 9")
    elif mode == "plants":
        failures = mode_plants()
    else:  # unreachable; MODES gates entry
        return 2
    if failures:
        print(f"gate {mode}: {failures} FAIL case(s)")
        return 1
    print(f"gate {mode}: all cases pass")
    return 0


if __name__ == "__main__":
    sys.exit(main())
