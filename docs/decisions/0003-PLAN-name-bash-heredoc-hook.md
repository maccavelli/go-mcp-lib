---
status: implemented
date: 2026-11-23
associated-madr: 0003-MADR-name-bash-heredoc-hook.md
---
# Name the Bash and heredocs deny hook

Associated MADR: [0003-MADR-name-bash-heredoc-hook.md](0003-MADR-name-bash-heredoc-hook.md).

## Grounding Reference

The associated MADR records the requested label and the configuration states.
The local Codex source, `codex-rs/protocol/src/hooks.rs`, defines the command
handler field `statusMessage`. The repository `.markdownlint-cli2.jsonc` defines
the required record headings; `scripts/check_records.py` validates record
placement, names, statuses, and relative links.

## Goal

Set the existing hook label to `Bash and heredocs Deny`.

## Scope

The matching handler in `~/.codex/hooks.json`, this record pair, and its index.

## Prerequisites

* The owner has approved the label and execution.
* The target configuration can be read and contains exactly one matching handler.
* Writing the user configuration requires filesystem authorization.
* Documentation checks use Python 3.12 or later and markdownlint-cli2 0.23.2.

## Implementation Steps

### Phase 1: Set and verify the label

1. Re-read the configuration and confirm the command handler schema supports
   `statusMessage`.
2. Require exactly one Bash handler invoking `no-heredoc.py`.
3. Add the requested `statusMessage` using Python.
4. Read the complete result; verify the exact label and semantic equality of
   every other setting.

Exit criterion: no blocker, every decided row checked, malformed selection
rejected before writing. No persistent gate or test is introduced.

### Phase 2: Correct record formatting and stage documentation

1. Re-read the record pair and the repository heading requirements.
2. Add the missing MADR and PLAN sections and record this approved correction.
3. Read the complete edited pair and run the record validator and pinned
   Markdown linter over the documentation being staged.
4. Check whitespace and identifiers, then stage the six documentation files
   reviewed by the owner: `docs/README.md`, this pair,
   [0002-MADR-build-schema-driven-mcp-framework.md](0002-MADR-build-schema-driven-mcp-framework.md),
   [0002-PLAN-build-schema-driven-mcp-framework.md](0002-PLAN-build-schema-driven-mcp-framework.md),
   and [0002-REPORT-qualify-mcp-framework-boundaries.md](../reports/0002-REPORT-qualify-mcp-framework-boundaries.md).
5. Verify the staged file list and confirm the staged content matches the checked
   working files. Leave the commit to the owner.

Exit criterion: no blocker, the existing checks pass, and exactly the reviewed
documentation is staged. No new gate, assertion, or test is introduced.

## Verification

| Decided state | Verification |
| :--- | :--- |
| One matching unnamed handler | Parse the saved JSON and compare the label to the exact requested value. |
| Missing or duplicate target | Exercise both cases on in-memory copies and require rejection before writing. |
| Other configuration values | Remove the added field from a copy and compare the complete JSON with the original. |

Report client display verification separately. For documentation, run
`python3 scripts/check_records.py --check-all`, markdownlint-cli2 0.23.2,
and the Git whitespace check before staging.

## Rollout

The configuration is available on the next client hook configuration load.

## Rollback

Rollback removes only the added field, with owner authorization.
Documentation correction does not change runtime behavior; preserve unrelated
work if any record edit must be reversed.

## Completion Criteria

* The saved label is exactly `Bash and heredocs Deny`.
* Missing and duplicate target fixtures are rejected before writing.
* All other configuration values are unchanged.
* Required documentation checks pass and the six reviewed files are staged.
* The owner retains responsibility for committing; client display verification
  remains explicitly unverified.

## Execution Record

### 2026-11-23: Phase 1, pass 1

The owner requested the name and then instructed execution to proceed.
Inspection confirmed the target and schema field. The label was saved and the
complete JSON read back. Verification output:

```text
PASS: missing target rejected: Expected exactly one matching handler
PASS: duplicate target rejected: Expected exactly one matching handler
PASS: exact label saved; all other settings unchanged
```

The malformed cases used in-memory copies. The client display has not been
observed after reloading. Records remain uncommitted; the local configuration
is outside the repository.

The existing hook rejected inline Python due to its length. The same edit ran
successfully from a one-use scratch Python file as instructed by that hook.

### 2026-11-23: Approved deviation and Phase 2, pass 1

The staging review found pre-existing formatting errors in this pair:
markdownlint-cli2 0.23.2 reported `MD043/required-headings` at line 36 of
`0003-MADR-name-bash-heredoc-hook.md` and line 62 of
`0003-PLAN-name-bash-heredoc-hook.md`. The MADR lacked `Decision Drivers` and
`Consequences`; the PLAN lacked `Grounding Reference`, `Prerequisites`,
`Rollback`, and `Completion Criteria`.

The owner authorized adding the required sections, checking standards, and
staging the documentation for their commit. Phase 2 records that resolution;
the runtime decision and completed Phase 1 steps are unchanged.

The existing record validator exited 0 with no output. The pinned linter
reported:

```text
markdownlint-cli2 v0.23.2 (markdownlint v0.41.1)
Linting: 22 files
Summary: 0 issues in 0 files
```

The configured globs include the repository documentation as well as the six
explicit files. The whitespace check exited 0. A scan of the six files for the
local account name, known fleet host identifiers, and user-directory paths
returned no matches. No new checks were introduced.

The queue wrapper exited 3 because its queue directory was not writable.
Following its documented fallback, the record validator ran directly.
