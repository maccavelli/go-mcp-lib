# 0007-REPORT — Stage 4 execution: CLI, deterministic publication, and recovery

Execution evidence for stage 4 of
[0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md).

| Item | Value |
| :--- | :--- |
| Stage | 4 of 10 — CLI, deterministic publication, recovery |
| Source revision | `8709316` on `main` plus uncommitted stage-4 tree |
| Go | go1.27.4 darwin/arm64, `GOWORK=off`, `GOTOOLCHAIN=local` |
| Verification | Python 3.15.2 stdlib scripts; no heredocs anywhere in this stage |

## Deliverables

| Path | Content |
| :--- | :--- |
| `cmd/mcpgen/cli.go` | Exit-code contract (0 help/success, 2 usage, 1 failure), `--format text\|json` printer with versioned envelope, strict flag sets, command help |
| `cmd/mcpgen/commands.go` | `inspect`, `check`, `generate`, `diff`, `update`, `recover`, `diagnostic`; unknown flags and positionals exit 2 |
| `cmd/mcpgen/check.go` | Scratch render + byte comparison; distinguishes drift, unowned collisions, interrupted publications, and first generation |
| `cmd/mcpgen/publish.go` | Journal-based publication: staged candidate hashes, deterministic order, verified-stale removal, manifest last, mtime preservation on unchanged bytes, sync before replacement |
| `cmd/mcpgen/lock.go` | Exclusive per-destination lock; an existing holder is never stolen on age; unreadable lock is an operator conflict |
| `cmd/mcpgen/recover.go` | `recover --action inspect\|complete\|restore`; candidate-hash verification regardless of journal state; restore refuses files whose originals are unrecoverable |
| `cmd/mcpgen/acquire_cmd.go` | `update`: bounded, hash-verified, candidate-only acquisition; never promotes |
| `scripts/verify_recovery.py` | Nine-point fault-injection and recovery verification over the real CLI binary |
| `scripts/verify_determinism.py` | Twin-scratch regeneration: byte-identical outputs, no identity leaks, mtime preservation |
| `scripts/verify_update.py` | Update contract: positive path plus conflict, mismatch, missing-flags, and unreachable-source plants |

## Verification

| Check | Result |
| :--- | :--- |
| Mixed-tree detection (removed owned file) | check fails exit 1 naming the interrupted file |
| Regenerate completes an interrupted tree | pass, journal-verified, sentinel untouched |
| Live lock (holder `pid-99999-live-generator`) | refused without stealing, exit 1 |
| Tampered journal candidate hash | complete refused loudly |
| Twin scratch generation | 4/4 files byte-identical |
| Identity leak scan over outputs | clean: no machine paths, account names, timestamps |
| Mtime preservation on regeneration | pass after fixing unconditional rewrites |
| `update` positive + 4 negative contracts | all pass; mismatch and failed fetch write nothing |
| Both modules | 8 generator packages + runtime green, `0 issues.` lint each |
| All repo gates (`pre-add`, `generate-check`, `api`, `unit`, `plants`, 14 orchestration tests) | exit 0 |

## In-flight corrections

- `check` initially printed drift but exited 0, and `generate` published
  over drift: both contract defects were found by the verification scripts
  and fixed (drift exits 1; generate refuses).
- First generation was impossible: `checkTree` required an existing
  destination. A missing destination is now classified as first
  generation, completable by design.
- `recover complete` short-circuited on `already-complete` before
  verifying hashes; verification now happens regardless of journal state,
  so a tampered journal is refused even when marked complete.
- mtime preservation was broken by unconditional rewrites of unchanged
  files, including the manifest; unchanged bytes are now left untouched.

## Limits

- Fault injection is at command granularity (simulated interrupted
  states), not signal-level kills of a child generator mid-rename; the
  journal makes such states recoverable and `check` detects them, but a
  kill-timing test is future hardening for stage 9's qualification matrix.
- `diff` compares emitted manifests and revisions; semantic-rule
  (semantics.json) diffing extends in stage 5 with the vocabulary
  vocabulary checks it gates.
- The journal records hashes, not original bytes, so `restore` cannot
  resurrect a replaced file from the journal alone — by design, that is a
  conflict requiring operator resolution, and the PLAN's "recover or
  complete" contract is satisfied by git history.
- Identity scan: clean across all outputs and this report.
