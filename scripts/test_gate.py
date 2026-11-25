"""R1 failure controls: scratch inputs and controlled subprocess results only."""

from __future__ import annotations

import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))
import acceptance_inventory as inventory
import gate
import gate_common as common
import prepare

ROOT = Path(__file__).resolve().parent.parent
PINS = json.loads((ROOT / "tools/versions.json").read_text())


def stream(*events: tuple[str, str | None], package: str = "fixture/p") -> str:
    return "\n".join(json.dumps(dict(Action=action, Package=package, **({"Test": test} if test else {})))
                     for action, test in events)


GOOD = stream(("start", None), ("run", "TestCase"), ("pass", "TestCase"), ("pass", None))


class TestRepairA02GateAccounting(unittest.TestCase):
    def test_complete_and_parallel(self):
        self.assertEqual(common.account_tests(GOOD, {"fixture/p"}), ([], 1))
        parallel = stream(("start", None), ("run", "TestCase"), ("pause", "TestCase"),
                          ("cont", "TestCase"), ("pass", "TestCase"), ("pass", None))
        self.assertEqual(common.account_tests(parallel, {"fixture/p"}), ([], 1))

    def test_broken_streams(self):
        cases = {
            "empty": "", "malformed": GOOD + "\n{", "nonobject": GOOD + "\n[]",
            "missing-package": GOOD.replace('"fixture/p"', '"other"'),
            "all-skipped": GOOD.replace('"pass", "Package": "fixture/p", "Test"', '"skip", "Package": "fixture/p", "Test"'),
            "unfinished": stream(("start", None), ("run", "TestCase"), ("pass", None)),
            "failed": GOOD.replace('"pass"', '"fail"', 1),
            "duplicate": GOOD + "\n" + stream(("pass", "TestCase")),
            "no-run": stream(("start", None), ("pass", "TestCase"), ("pass", None)),
            "no-package-end": stream(("start", None), ("run", "TestCase"), ("pass", "TestCase")),
            "unknown-action": GOOD + "\n" + stream(("mystery", None)),
        }
        for name, body in cases.items():
            with self.subTest(case=name):
                errors, _ = common.account_tests(body, {"fixture/p"})
                self.assertTrue(errors, name)
                print(f"CONTROL {name}: FAIL {errors[0]}")

    def test_missing_second_package(self):
        errors, _ = common.account_tests(GOOD, {"fixture/p", "fixture/missing"})
        self.assertTrue(any("fixture/missing" in error for error in errors))

    def test_required_case_and_literal_cache_output(self):
        literal = GOOD.replace('"Action": "pass", "Package": "fixture/p"}',
                               '"Action":"output","Package":"fixture/p","Output":"(cached) is test data"}\n'
                               '{"Action":"pass","Package":"fixture/p"}')
        self.assertEqual(common.account_tests(literal, {"fixture/p"}), ([], 1))
        errors, _ = common.account_tests(GOOD, {"fixture/p"}, required_tests={("fixture/p", "TestRequired")})
        self.assertTrue(any("missing required passing test" in error for error in errors))
        print("CONTROL missing-required-case: FAIL missing required passing test")

    def test_timeout_retains_partial_output(self):
        result = common.run([sys.executable, "-u", "-c", "import time; print('partial'); time.sleep(30)"], ROOT, timeout=0.2)
        self.assertEqual(result.returncode, 124)
        self.assertIn("partial", result.stdout)
        self.assertIn("timed out", result.stderr)

    def test_package_without_tests_is_explicit(self):
        extra = stream(("start", None), ("skip", None), package="fixture/data")
        self.assertEqual(common.account_tests(GOOD + "\n" + extra, {"fixture/p", "fixture/data"}, {"fixture/data"}), ([], 1))
        self.assertTrue(common.account_tests(GOOD + "\n" + extra, {"fixture/p", "fixture/data"})[0])
        self.assertTrue(common.account_tests(extra, {"fixture/data"}, {"fixture/data"})[0])

    def test_fresh_command_exit_and_environment(self):
        calls = []
        def execute(args, cwd, **kwargs):
            calls.append(args)
            return subprocess.CompletedProcess(args, 0, "fixture/p 1 0\n" if args[1] == "list" else ("" if args[1] == "vet" else GOOD), "")
        with patch.object(common, "run", side_effect=execute), patch.object(common, "required_inventory", return_value={("fixture/p", "TestCase")}), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(common.test_module(ROOT, "runtime"), 0)
        self.assertIn(["go", "test", "-vet=off", "-count=1", "-json", "./..."], calls)
        self.assertEqual(calls.count(["go", "vet", "./..."]), 1)
        with patch.dict(os.environ, {"CGO_ENABLED": "1", "GOFLAGS": "-race", "GOWORK": "other", "GOOS": "foreign", "GOARCH": "wrong", "GOEXPERIMENT": "nojsonv2"}):
            env = common.environment()
            self.assertEqual((env["CGO_ENABLED"], env["GOFLAGS"], env["GOWORK"]), ("0", "-mod=readonly", "off"))
            self.assertFalse({"GOOS", "GOARCH", "GOEXPERIMENT"} & env.keys())
        with patch.object(common, "run", return_value=subprocess.CompletedProcess([], 9, "", "broken")), contextlib.redirect_stdout(io.StringIO()):
            self.assertGreater(common.test_module(ROOT, "runtime"), 0)

    def test_strict_arguments_and_module_selection(self):
        for args in (["unit", "--module", "wrong"], ["api", "--module", "runtime"], ["unit", "--allow-network"], ["unit", "--unknown"]):
            with self.subTest(args=args), contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as error:
                gate.main(args)
            self.assertEqual(error.exception.code, 2)
        with patch.object(gate, "test_module", return_value=0) as tests, patch.object(gate, "linter", return_value=("lint", [])), patch.object(gate, "lint_module", return_value=0), contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(gate.main(["unit", "--module", "generator"]), 0)
            self.assertEqual(tests.call_args.args[1], "generator")
            self.assertEqual(tests.call_count, 1)

    def test_incomplete_modes_remain_nonzero(self):
        for mode in ("api", "generate-check", "wire", "qualification"):
            with self.subTest(mode=mode), contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertEqual(gate.main([mode]), 1)
                self.assertIn("incomplete", output.getvalue())


class TestRepairA03LintAndPreAdd(unittest.TestCase):
    def test_gofmt_is_readonly_and_nested_modules_are_separate(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "bad.go"
            source.write_text("package fixture\nfunc X(){ }\n")
            (root / "go.mod").write_text("module fixture\n")
            nested = root / "cmd/other"
            nested.mkdir(parents=True)
            (nested / "go.mod").write_text("module other\n")
            (nested / "other.go").write_text("package other\n")
            before = (source.read_bytes(), source.stat().st_mtime_ns)
            self.assertEqual(common.go_files(root), [source])
            result = common.run(["gofmt", "-l", str(source)], root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("bad.go", result.stdout)
            self.assertEqual(before, (source.read_bytes(), source.stat().st_mtime_ns))

    def test_failed_empty_formatter_and_missing_linter_do_not_stop_checks(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "go.mod").write_text("module fixture\n")
            (root / "source.go").write_text("package fixture\n")
            with patch.object(common, "linter", return_value=("missing", ["missing tool"])), patch.object(common, "run", return_value=subprocess.CompletedProcess([], 1, "", "")) as commands, patch.object(common, "test_module", return_value=1) as tests, contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertGreaterEqual(common.precheck(root, "runtime", PINS), 4)
                self.assertTrue(tests.called)
                self.assertIn("gofmt exit 1", output.getvalue())
                self.assertFalse(any(call.args[0][1:2] == ["vet"] for call in commands.call_args_list))

    def test_wrong_linter_version_rejected(self):
        with patch.object(common, "run", return_value=subprocess.CompletedProcess([], 0, "version 2.15.00 built with go1.27.4", "")):
            self.assertTrue(common.linter(ROOT, PINS)[1])


class TestRepairA01Evidence(unittest.TestCase):
    def test_complete_inventory_and_missing_owner_plant(self):
        expected = inventory.build_inventory(ROOT, PINS)
        self.assertEqual(len(expected["definitions"]), 155)
        self.assertEqual(len(expected["obligations"]), 39)
        self.assertEqual(len(expected["scenarios"]), 89)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "tools").mkdir()
            altered = json.loads(json.dumps(expected))
            altered["definitions"].pop()
            (root / inventory.INVENTORY).write_text(json.dumps(altered))
            with patch.object(inventory, "build_inventory", return_value=expected), contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertGreater(inventory.check_inventory(root, PINS), 0)
                self.assertIn("missing, altered or stale", output.getvalue())

    def test_frozen_requirement_tamper(self):
        with patch.object(inventory, "digest", return_value="wrong"), self.assertRaisesRegex(ValueError, "identity mismatch"):
            inventory.build_inventory(ROOT, PINS)


class TestRepairA04Preparation(unittest.TestCase):
    def test_module_preparation_isolated_and_network_is_explicit(self):
        metadata = "\n".join(json.dumps({"Path": name, "Version": pin["version"]})
                             for name, pin in PINS["runtime-dependencies"].items())
        for network in (False, True):
            calls = []
            def execute(args, cwd, **kwargs):
                self.assertNotEqual(cwd.resolve(), ROOT)
                calls.append((args, kwargs.get("network", False)))
                return subprocess.CompletedProcess(args, 0, metadata if args[1] == "list" else "", "")
            with self.subTest(network=network), patch.object(prepare, "run", side_effect=execute):
                self.assertEqual(prepare.prepare_module(ROOT, ".", PINS, "runtime", network), [])
                self.assertIn((["go", "list", "-m", "-json", "all"], network), calls)
                self.assertEqual(any(args == ["go", "mod", "download"] for args, _ in calls), network)

    def test_module_metadata_mutation_cannot_change_live_lock(self):
        before = (ROOT / "go.sum").read_bytes()
        metadata = "\n".join(json.dumps({"Path": name, "Version": pin["version"]})
                             for name, pin in PINS["runtime-dependencies"].items())
        def execute(args, cwd, **kwargs):
            if args[1:] == ["mod", "download"]:
                (cwd / "go.sum").write_text("deliberately changed scratch lock\n")
            return subprocess.CompletedProcess(args, 0, metadata if args[1] == "list" else "", "")
        with patch.object(prepare, "run", side_effect=execute):
            errors = prepare.prepare_module(ROOT, ".", PINS, "runtime", True)
            self.assertTrue(any("isolated module metadata changed" in error for error in errors))
            print("CONTROL module-mutation: FAIL isolated module metadata changed")
        self.assertEqual((ROOT / "go.sum").read_bytes(), before)

    def prepared_fixture(self, root):
        (root / "requirements").mkdir(parents=True)
        (root / "requirements/spec.yaml").write_text("server:\n  - valid\n")
        (root / "package-lock.json").write_text("{}\n")
        (root / "package.json").write_text(json.dumps({"version": "1", "bin": {"conformance": "dist/index.js"}}))
        (root / "dist").mkdir()
        (root / "dist/index.js").write_text("export {};\n")
        (root / "node_modules").mkdir()
        (root / "node_modules/runtime.js").write_text("export {};\n")
        pin = {"commit": "a" * 40, "package-version": "1", "requirements-file": "requirements/spec.yaml",
               "requirements-sha256": prepare.digest(root / "requirements/spec.yaml"),
               "lockfile-sha256": prepare.digest(root / "package-lock.json")}
        receipt = {"pin": pin, "tools": {"node": "1"}, "runner-sha256": prepare.tree_digest(root / "dist"),
                   "dependencies-sha256": prepare.tree_digest(root / "node_modules", allow_links=True)}
        (root / "prepared.json").write_text(json.dumps(receipt))
        return pin

    def test_missing_preparation_offline_never_runs_command(self):
        with tempfile.TemporaryDirectory() as temporary, patch.object(prepare, "checked") as command:
            with self.assertRaisesRegex(ValueError, "not prepared"):
                prepare.prepare_conformance(Path(temporary), {"commit": "a" * 40}, {}, False)
            command.assert_not_called()
            self.assertEqual(list(Path(temporary).iterdir()), [])

    def test_hash_tool_and_partial_receipt_controls(self):
        for case in ("runner", "dependencies", "lock", "requirements", "tools", "partial"):
            with self.subTest(case=case), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                pin = self.prepared_fixture(root)
                prepare.verify_prepared(root, pin, {"node": "1"})
                targets = {"runner": "dist/index.js", "dependencies": "node_modules/runtime.js", "lock": "package-lock.json", "requirements": "requirements/spec.yaml", "partial": "prepared.json"}
                if case in targets:
                    (root / targets[case]).write_text("broken")
                tools = {"node": "2" if case == "tools" else "1"}
                with self.assertRaises((ValueError, KeyError)):
                    prepare.verify_prepared(root, pin, tools)

    def test_interrupted_acquisition_has_no_promotion(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            pin = {"commit": "a" * 40, "repository": "example.invalid/conformance"}
            with patch.object(prepare, "checked", side_effect=KeyboardInterrupt):
                with self.assertRaises(KeyboardInterrupt):
                    prepare.prepare_conformance(root, pin, {}, True)
            self.assertFalse((root / ".cache/prepared/conformance.lock").exists())
            self.assertFalse((root / ".cache/prepared" / ("conformance-" + pin["commit"])).exists())

    def test_missing_executable_is_named_result(self):
        result = common.run(["nonexistent-r1-control-executable"], ROOT)
        self.assertEqual(result.returncode, 127)


if __name__ == "__main__":
    unittest.main(verbosity=2)
