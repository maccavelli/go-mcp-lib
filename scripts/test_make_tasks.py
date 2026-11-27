"""Make selectors, queue admission and real Make failure controls in scratch."""

import contextlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import gate_common
import make_tasks


class TestMakeFoundation(unittest.TestCase):
    def test_combined_goals_share_owners(self):
        goals = ["pre-add-check", "lint", "vet", "test", "docs-check", "check-records", "markdownlint", "generate-check", "build-check"]
        checks, leaves = make_tasks.selections(goals)
        self.assertEqual(checks, {"fmt", "vet", "test", "lint"})
        self.assertEqual(leaves, {"records", "markdown", "generated"})
        with patch.object(make_tasks, "precheck", return_value=0) as precheck, patch.object(make_tasks, "command", return_value=0) as commands:
            self.assertEqual(make_tasks.execute(goals, "all", checks, leaves), 0)
        precheck.assert_called_once()
        self.assertEqual([call.args[0] for call in commands.call_args_list], ["generated", "records", "markdown"])
        with self.assertRaisesRegex(ValueError, "unknown Make goal"):
            make_tasks.selections(["invented"])
        print("CONTROL unknown-goal: FAIL unknown Make goal")

    def test_failed_leaf_fails_batch_without_hiding_later_results(self):
        goals = ["lint", "check-records"]
        with patch.object(make_tasks, "precheck", return_value=1), patch.object(make_tasks, "command", return_value=0) as later:
            self.assertEqual(make_tasks.execute(goals, "all", *make_tasks.selections(goals)), 1)
        later.assert_called_once()
        print("CONTROL failed-leaf: FAIL make batch")

    def test_lint_selects_both_modules_without_hidden_tests(self):
        pins = json.loads((make_tasks.ROOT / "tools/versions.json").read_text())
        with patch.object(gate_common, "linter", return_value=("pinned", [])), patch.object(gate_common, "revive", return_value=("docs", [])), patch.object(gate_common, "lint_module", return_value=0) as lint, patch.object(gate_common, "run") as commands, patch.object(gate_common, "test_module") as tests, contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(gate_common.precheck(make_tasks.ROOT, "all", pins, frozenset({"lint"})), 0)
        self.assertEqual([call.args[2] for call in lint.call_args_list], ["runtime", "generator"])
        commands.assert_not_called()
        tests.assert_not_called()

    def test_help_never_resolves_tools_and_worker_requires_queue(self):
        with patch.object(make_tasks, "execute") as execute, patch.object(make_tasks, "print_help") as help_text:
            self.assertEqual(make_tasks.main([]), 0)
            help_text.assert_called_once()
            execute.assert_not_called()
        with patch.dict(os.environ, {}, clear=True), contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(make_tasks.main(["--worker", "test"]), 1)
        self.assertIn("queue ticket", output.getvalue())
        print("CONTROL unadmitted-worker: FAIL heavy worker requires queue ticket")

    def test_documentation_tool_identity_controls(self):
        pins = json.loads((make_tasks.ROOT / "tools/versions.json").read_text())
        valid = "fixture: go1.27.4\n\tpath\tgithub.com/mgechev/revive\n\tmod\tgithub.com/mgechev/revive\tv1.17.1\th1:fixture\n"
        for name, output, code in (("valid", valid, 0),
                                   ("wrong-version", valid.replace("v1.17.1", "v1.0.0"), 0),
                                   ("wrong-builder", valid.replace("go1.27.4", "go1.26.0"), 0),
                                   ("missing-module", "fixture: go1.27.4\n", 0),
                                   ("failed-reader", valid, 1)):
            with self.subTest(case=name), patch.object(gate_common, "run", return_value=subprocess.CompletedProcess([], code, output, "")):
                errors = gate_common.revive(make_tasks.ROOT, pins, "fixture")[1]
                self.assertEqual(bool(errors), name != "valid")
                if errors:
                    print(f"CONTROL {name}: FAIL documentation analyzer identity")

    def test_actual_make_dispatch_and_failure(self):
        executable = shutil.which("make")
        self.assertIsNotNone(executable, "FAIL Make tool unavailable")
        with tempfile.TemporaryDirectory(prefix="mcp-make-control-") as directory:
            root = Path(directory)
            shutil.copy2(make_tasks.ROOT / "Makefile", root / "Makefile")
            (root / "scripts").mkdir()
            (root / "scripts/make_tasks.py").write_text(
                "import json,sys\nfrom pathlib import Path\n"
                "p=Path('probe.json')\ns=json.loads(p.read_text())\n"
                "s['calls'].append(sys.argv[1:])\np.write_text(json.dumps(s))\n"
                "raise SystemExit(s['exit'])\n")
            for goals in (("lint", "test"), ("pre-add-check",), ("pre-add-check", "lint", "test")):
                for broken in (False, True):
                    with self.subTest(goals=goals, broken=broken):
                        (root / "probe.json").write_text(json.dumps({"exit": int(broken), "calls": []}))
                        result = subprocess.run([executable, *goals, "PYTHON=" + shutil.which("python3")], cwd=root, capture_output=True, text=True, check=False)
                        self.assertEqual(result.returncode != 0, broken, result.stdout + result.stderr)
                        calls = json.loads((root / "probe.json").read_text())["calls"]
                        self.assertEqual(calls, [["--module", "all", *goals]])
                        if broken:
                            print(f"CONTROL make-broken {' '.join(goals)}: FAIL propagated nonzero dispatcher result")

    def test_hook_discovery_rejects_shared_only_target(self):
        def assert_discoverable(path):
            self.assertTrue(any(line.startswith("pre-add-check:") for line in path.read_text().splitlines()),
                            "FAIL pre-add hook cannot discover literal target")

        source = (make_tasks.ROOT / "Makefile").read_text()
        with tempfile.TemporaryDirectory(prefix="mcp-hook-control-") as directory:
            path = Path(directory) / "Makefile"
            path.write_text(source)
            assert_discoverable(path)
            literal = "pre-add-check: _local-checks\n"
            self.assertEqual(source.count(literal), 1)
            broken = source.replace(literal, "", 1).replace("\nall help setup setup-network", "\npre-add-check all help setup setup-network", 1)
            path.write_text(broken)
            with self.assertRaisesRegex(AssertionError, "FAIL pre-add hook cannot discover literal target"):
                assert_discoverable(path)
            print("CONTROL shared-only-target: FAIL pre-add hook cannot discover literal target")


if __name__ == "__main__":
    unittest.main(verbosity=2)
