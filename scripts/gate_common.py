"""Shared, non-mutating execution and Go test accounting for local gates."""

from __future__ import annotations

import json
import os
import platform
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time

MODULES = {"runtime": ".", "generator": "cmd/mcpgen"}


def environment(*, network: bool = False) -> dict[str, str]:
    env = dict(os.environ)
    for key in ("GOOS", "GOARCH", "GOEXPERIMENT", "GOENV", "GOTOOLDIR"):
        env.pop(key, None)
    env.update(CGO_ENABLED="0", GOWORK="off", GOTOOLCHAIN="local",
               GOFLAGS="-mod=readonly", GOENV="off")
    if not network:
        env.update(GOPROXY="off", GOSUMDB="off")
    return env


def run(args: list[str], cwd: Path, *, network: bool = False,
        timeout: int = 600, gobin: Path | None = None,
        env_updates: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    """Retain logs on disk; bound memory and terminate the process tree on timeout."""
    limit = 32 * 1024 * 1024
    directory = Path(tempfile.mkdtemp(prefix="mcp-gate-log-"))
    options = {"start_new_session": True} if os.name != "nt" else {
        "creationflags": subprocess.CREATE_NEW_PROCESS_GROUP}
    env = environment(network=network)
    if env_updates:
        if set(env_updates) - {"GO_MCP_LINTER", "GO_MCP_REVIVE"}:
            raise ValueError("execution overrides are limited to prepared tool selection")
        env.update(env_updates)
    if gobin is not None:
        env["GOBIN"] = str(gobin)
    try:
        if os.name == "nt" and args[0] in {"npm", "npx"}:
            executable = shutil.which(args[0])
            node = shutil.which("node")
            script = Path(executable).parent / "node_modules/npm/bin" / (args[0] + "-cli.js") if executable else None
            if not node or script is None or not script.is_file():
                raise OSError(f"native {args[0]} JavaScript entry point unavailable")
            args = [node, str(script), *args[1:]]
        with (directory / "stdout").open("wb") as out, (directory / "stderr").open("wb") as err:
            process = subprocess.Popen(args, cwd=cwd, env=env,
                                       stdout=out, stderr=err, **options)
            deadline = time.monotonic() + timeout
            failure = ""
            try:
                while process.poll() is None:
                    if time.monotonic() >= deadline:
                        failure = "command timed out"
                        break
                    if out.tell() + err.tell() > limit:
                        failure = "command log limit exceeded"
                        break
                    time.sleep(0.05)
            except BaseException:
                terminate_tree(process)
                raise
            if failure:
                terminate_tree(process)
            code = process.wait()
        with (directory / "stdout").open("rb") as handle:
            stdout = handle.read(limit).decode("utf-8", errors="replace")
        with (directory / "stderr").open("rb") as handle:
            stderr = handle.read(limit).decode("utf-8", errors="replace")
        if failure or (directory / "stdout").stat().st_size + (directory / "stderr").stat().st_size > limit:
            code = 124 if failure == "command timed out" else 125
            stderr += "\n" + (failure or "command log limit exceeded")
        result = subprocess.CompletedProcess(args, code, stdout, stderr)
        result.log_directory = str(directory)
        return result
    except OSError as exc:
        return subprocess.CompletedProcess(args, 127, "", str(exc))


def terminate_tree(process: subprocess.Popen) -> None:
    """Do not release a gate/preparation lock while its timed-out child survives."""
    if os.name == "nt":
        killed = subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"],
                                capture_output=True, timeout=30, check=False)
        if killed.returncode and process.poll() is None:
            raise RuntimeError("cannot terminate command tree")
    else:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    process.wait(timeout=30)


def report(case: str, errors: list[str], detail: str = "") -> int:
    for error in errors:
        print(f"FAIL {case}: {error}")
    if not errors:
        print(f"pass {case}" + (f": {detail}" if detail else ""))
    return len(errors)


def selected_modules(selection: str) -> list[tuple[str, str]]:
    return list(MODULES.items()) if selection == "all" else [(selection, MODULES[selection])]


def account_tests(stdout: str, expected_packages: set[str],
                  no_test_packages: set[str] | None = None,
                  required_tests: set[tuple[str, str]] | None = None) -> tuple[list[str], int]:
    """Reconcile every package and test, including parallel/subtest lifecycles."""
    errors: list[str] = []
    no_test_packages = no_test_packages or set()
    packages: dict[str, str] = {}
    tests: dict[tuple[str, str], str] = {}
    passed = 0
    for number, line in enumerate(stdout.splitlines(), 1):
        if not line.strip():
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            errors.append(f"line {number}: malformed JSON")
            continue
        if not isinstance(event, dict):
            errors.append(f"line {number}: event is not an object")
            continue
        action, package, test = (event.get(key) for key in ("Action", "Package", "Test"))
        if not isinstance(package, str) or package not in expected_packages:
            errors.append(f"line {number}: unknown or missing package {package!r}")
            continue
        if not isinstance(action, str) or action not in {
            "start", "run", "pause", "cont", "output", "pass", "fail", "skip", "bench"
        }:
            errors.append(f"line {number}: unsupported action {action!r}")
            continue
        if action == "output":
            if not isinstance(event.get("Output"), str):
                errors.append(f"{package}: malformed output event")
            continue
        if test is not None:
            if not isinstance(test, str) or not test:
                errors.append(f"{package}: invalid test identity")
                continue
            key = (package, test)
            state = tests.get(key)
            if packages.get(package) != "start":
                errors.append(f"{package}/{test}: test outside active package")
            if action == "run" and state is None:
                tests[key] = "run"
            elif action == "pause" and state == "run":
                tests[key] = "pause"
            elif action == "cont" and state == "pause":
                tests[key] = "run"
            elif action in {"pass", "fail", "skip"} and state == "run":
                tests[key] = action
                if action == "pass":
                    passed += 1
                else:
                    errors.append(f"{package}/{test}: unexpected {action}")
            else:
                errors.append(f"{package}/{test}: duplicate or invalid {state!r} -> {action}")
        elif action == "start" and package not in packages:
            packages[package] = "start"
        elif action in {"pass", "fail", "skip"} and packages.get(package) == "start":
            packages[package] = action
            empty_package = action == "skip" and package in no_test_packages and not any(key[0] == package for key in tests)
            if empty_package:
                packages[package] = "pass"
            elif action != "pass":
                errors.append(f"{package}: unexpected package {action}")
        else:
            errors.append(f"{package}: duplicate or invalid package transition {action}")
    for package in sorted(expected_packages):
        if packages.get(package) != "pass":
            errors.append(f"{package}: missing successful terminal package result")
    for (package, test), state in sorted(tests.items()):
        if state not in {"pass", "fail", "skip"}:
            errors.append(f"{package}/{test}: missing terminal test result")
    for package, test in sorted(required_tests or set()):
        if tests.get((package, test)) != "pass":
            errors.append(f"{package}/{test}: missing required passing test")
    if not passed:
        errors.append("no passing tests executed in selected module")
    return errors, passed


def package_inventory(root: Path) -> tuple[set[str], set[str], list[str]]:
    result = run(["go", "list", "-f", "{{.ImportPath}} {{len .TestGoFiles}} {{len .XTestGoFiles}}", "./..."], root)
    names, no_tests = set(), set()
    errors = []
    for line in result.stdout.splitlines():
        fields = line.split()
        if len(fields) != 3 or not all(value.isdecimal() for value in fields[1:]):
            errors.append("malformed package inventory")
            continue
        names.add(fields[0])
        if fields[1:] == ["0", "0"]:
            no_tests.add(fields[0])
    if result.returncode or not names:
        errors.append(f"package inventory failed ({result.returncode}): {result.stderr.strip()}")
    return names, no_tests, errors


def test_module(root: Path, name: str) -> int:
    vet = run(["go", "vet", "./..."], root)
    failures = report(f"unit.{name}.vet", [vet.stdout + vet.stderr] if vet.returncode else [])
    packages, no_tests, errors = package_inventory(root)
    if errors:
        return failures + report(f"unit.{name}", errors)
    try:
        repository = root if name == "runtime" else root.parent.parent
        required = required_inventory(repository, name)
    except (OSError, ValueError, KeyError, TypeError) as exc:
        return failures + report(f"unit.{name}", [f"required test inventory: {exc}"])
    result = run(["go", "test", "-vet=off", "-count=1", "-json", "./..."], root,
                 env_updates={"GO_MCP_LINTER": linter_executable(repository),
                              "GO_MCP_REVIVE": revive_executable(repository)})
    errors, passed = account_tests(result.stdout, packages, no_tests, required)
    errors.extend(failed_test_diagnostics(result.stdout))
    if result.returncode:
        logs = getattr(result, "log_directory", "unavailable")
        errors.append(f"go test exit {result.returncode}; complete logs: {logs}: {(result.stdout + result.stderr)[-2000:]}")
    if not errors and getattr(result, "log_directory", None):
        print("test-logs " + json.dumps({"module": name, "directory": result.log_directory}), flush=True)
    return failures + report(f"unit.{name}", errors, f"{passed} tests passed; {len(packages)} packages accounted")


def failed_test_diagnostics(stream: str) -> list[str]:
    """Return bounded failed-test details, independent of temporary log lifetime.

    Accounting owns stream validity. This only recovers diagnostic output and
    never turns a failed or malformed stream into a successful result.
    """
    details = {}
    failed = set()
    for line in stream.splitlines():
        try:
            event = json.loads(line)
        except (json.JSONDecodeError, ValueError):
            continue
        if not isinstance(event, dict):
            continue
        package, test = event.get("Package"), event.get("Test")
        if not isinstance(package, str) or not isinstance(test, str):
            continue
        key = (package, test)
        if event.get("Action") == "fail":
            failed.add(key)
        if event.get("Action") == "output" and isinstance(event.get("Output"), str):
            details[key] = (details.get(key, "") + event["Output"])[-8192:]
    output = []
    budget = 65536
    for key in sorted(failed):
        message = f"{key[0]}/{key[1]} retained failure detail:\n{details.get(key, '')}"
        if not budget:
            break
        output.append(message[:budget])
        budget -= len(output[-1])
    return output


def required_inventory(repository: Path, name: str) -> set[tuple[str, str]]:
    """Reviewed expectations are independent of observed test results."""
    document = json.loads((repository / "tools/test-inventory.json").read_text())
    if document.get("version") != 1:
        raise ValueError("unsupported test inventory")
    rows = document["modules"][name]
    required = {(row["package"], row["test"]) for row in rows}
    if not required or len(required) != len(rows):
        raise ValueError("empty or duplicate expected tests")
    return required


def linter_executable(root: Path) -> str:
    """One tool-selection policy for analyzer runs and generated-candidate tests."""
    prepared = linter_path(root)
    return os.environ.get("GO_MCP_LINTER") or (str(prepared) if prepared.is_file() else shutil.which("golangci-lint")) or "golangci-lint"


def linter(root: Path, pins: dict) -> tuple[str, list[str]]:
    executable = linter_executable(root)
    result = run([executable, "--version"], root)
    pin = pins["lint"]["golangci-lint"]
    version = re.search(r"version (\S+) built with go(\S+)", result.stdout)
    errors = []
    if result.returncode or not version or version.groups() != (pin["version"], pin["built-with-go"]):
        errors.append(f"required golangci-lint {pin['version']} built with go{pin['built-with-go']}; {result.stdout}{result.stderr}")
    return executable, errors


def linter_path(root: Path) -> Path:
    suffix = ".exe" if os.name == "nt" else ""
    return root / ".cache/tools" / (platform.system() + "-" + platform.machine()) / ("golangci-lint" + suffix)


def revive_path(root: Path) -> Path:
    suffix = ".exe" if os.name == "nt" else ""
    return linter_path(root).parent / ("revive" + suffix)


def revive_executable(root: Path) -> str:
    prepared = revive_path(root)
    return os.environ.get("GO_MCP_REVIVE") or (str(prepared) if prepared.is_file() else shutil.which("revive")) or "revive"


def documentation_pin(root: Path) -> dict:
    document = json.loads((root / "tools/documentation-lint.json").read_text())
    if document["version"] != 1:
        raise ValueError("unsupported documentation lint pin")
    return document["revive"]


def revive(root: Path, pins: dict, executable: str | None = None) -> tuple[str, list[str]]:
    executable = executable or revive_executable(root)
    try:
        pin = documentation_pin(root)
    except (OSError, ValueError, KeyError, TypeError) as exc:
        return executable, [f"documentation lint pin unavailable: {exc}"]
    result = run(["go", "version", "-m", executable], root)
    lines = [line.split() for line in result.stdout.splitlines()]
    builder = pins["go"]["toolchain"]
    compiler = bool(lines and lines[0][-1:] == [builder])
    module = any(row[:3] == ["mod", pin["module"], pin["version"]] for row in lines)
    errors = [] if not result.returncode and compiler and module else [
        f"required Revive {pin['version']} built with {builder}; {result.stdout}{result.stderr}"]
    return executable, errors


def lint_module(root: Path, module: Path, name: str, executable: str, documentation: str | None = None) -> int:
    result = run([executable, "run", "--config", str(root / ".golangci.yml")], module)
    errors = [f"exit {result.returncode}: {result.stdout}{result.stderr}"] if result.returncode else []
    failures = report(f"lint.{name}", errors, "0 issues")
    result = run([documentation or revive_executable(root), "-config", str(root / ".revive.toml"),
                  "-formatter", "friendly", "-set_exit_status", "./..."], module)
    errors = [f"exit {result.returncode}: {result.stdout}{result.stderr}"] if result.returncode else []
    return failures + report(f"lint.{name}.documentation", errors, "0 issues; generated files included")


def go_files(module: Path) -> list[Path]:
    """Include generated, untracked and platform files; stop at nested modules."""
    files = []
    for directory, children, names in os.walk(module):
        base = Path(directory)
        children[:] = sorted(child for child in children
                             if not child.startswith(".") and child not in {"vendor", "node_modules"}
                             and not (base / child).is_symlink()
                             and not (base / child / "go.mod").exists())
        files.extend(base / name for name in sorted(names) if name.endswith(".go"))
    return files


def precheck(root: Path, selection: str, pins: dict,
             checks: frozenset[str] = frozenset({"fmt", "vet", "test", "lint"})) -> int:
    if not checks or checks - {"fmt", "vet", "test", "lint"}:
        raise ValueError("unknown or empty module check selection")
    executable, errors = linter(root, pins) if "lint" in checks else ("", [])
    documentation, doc_errors = revive(root, pins) if "lint" in checks else ("", [])
    errors += doc_errors
    failures = report("pre-add.linter", errors) if "lint" in checks else 0
    for name, relative in selected_modules(selection):
        module = root / relative
        if not (module / "go.mod").is_file():
            failures += report(f"pre-add.{name}", [f"missing module at {relative}"])
            continue
        files = go_files(module) if "fmt" in checks else []
        if "fmt" in checks and not files:
            failures += report(f"pre-add.{name}.gofmt", ["empty source inventory"])
        # Bounded argument lengths also work on native Windows.
        for offset in range(0, len(files), 50):
            result = run(["gofmt", "-l", *map(str, files[offset:offset + 50])], module)
            problems = []
            if result.returncode:
                problems.append(f"gofmt exit {result.returncode}: {result.stderr}")
            if result.stdout.strip():
                problems.append(f"gofmt would rewrite: {result.stdout.strip()}")
            failures += report(f"pre-add.{name}.gofmt", problems)
        if "test" in checks:
            # Test accounting owns the one standalone vet invocation as well.
            failures += test_module(module, name)
        elif "vet" in checks:
            result = run(["go", "vet", "./..."], module)
            failures += report(f"unit.{name}.vet", [result.stdout + result.stderr] if result.returncode else [])
        if "lint" not in checks:
            continue
        if not errors:
            failures += lint_module(root, module, name, executable, documentation)
        else:
            failures += report(f"lint.{name}", ["pinned linter unavailable"])
    return failures
