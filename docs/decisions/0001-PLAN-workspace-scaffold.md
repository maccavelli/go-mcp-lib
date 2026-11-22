---
status: "in-progress"
date: 2026-11-22
associated-madr: "0001-MADR-workspace-scaffold.md"
---

# Implement the fleet workspace scaffold

Associated MADR: [0001-MADR-workspace-scaffold.md](0001-MADR-workspace-scaffold.md)

## Goal

Install the fleet workspace scaffold in `go-mcp-lib` exactly as decided in the associated MADR:
line-ending and ignore baselines, docs gates, the scaffold record pair, the docs tree, and the
agent files with pointers for all five harnesses.

## Scope

- Fleet scaffold files only: `.gitattributes`, `.gitignore` merge, `.markdownlint-cli2.jsonc`,
  `scripts/check_records.py`, `docs/`, `CLAUDE.md`, the `AGENTS.md` symlink, the five harness
  pointer trees, `opencode.json`, and the two harness `.gitignore` files.
- No product source, no CI workflow, no `.editorconfig`, no Makefile targets (no `Makefile`
  exists; splice per the MADR's state table when one lands).

## Grounding Reference

The first step of every execution pass re-grounds the MADR's asserted facts at HEAD:

- the working tree is clean at the commit the scaffold lands on;
- a whole-repository scan for `NNNN-{MADR,PLAN,REPORT,GATES}-*.md` finds no record other than
  this pair, so `0001` is the next number and the `--next` gate must print `0002` once the pair
  exists;
- `git rev-parse --path-format=absolute --git-path hooks` is `~/.global-git-hooks`, matching the
  MADR's commit-policy premise.

## Prerequisites

- Owner approval received (`proceed`, 2026-11-22); the MADR is `accepted`.
- The plant suite runs only in a scratch copy built from the skill's templates, never in the
  working tree.
- The stages below map to the MADR's decided state-table rows:

  | MADR row (decided) | Pinned by |
  | :--- | :--- |
  | Fresh clone, all five harness homes present | pointer `cmp` checks and the stage 4 gate run |
  | Clone missing user-skill homes | pointer files present and `cmp`-identical, checked in stage 3 |
  | Repo gains a `Makefile` later | recorded in the MADR state table itself; deliberately no commit in this PLAN |

## Stage 1 — line endings and ignores

1. Re-ground at HEAD per Grounding Reference.
2. Copy the template `.gitattributes`; `cmp` it afterwards.
3. Merge the template gitignore fragment into `.gitignore`, appending missing entries.

Exit criterion: both copies `cmp`-identical, no pre-existing ignore entry dropped.

## Stage 2 — docs gates

1. Copy `.markdownlint-cli2.jsonc` with the legacy override's `filter` left empty; `cmp`.
2. Copy `scripts/check_records.py`; `cmp`.
3. Write `docs/README.md` from its template (repo name, the `0001` record rows, the
   "I want to…" matrix) and `docs/architecture.md` with `{{WHAT_IT_IS}}` and
   `{{WHAT_IS_NOT_HERE}}` filled from the repository as it is now — a Go MCP library whose
   structure is named by its first stack MADR, not invented here.
4. Root `README.md`: add a link to `docs/README.md`; do not otherwise touch it.

Exit criterion: byte-for-byte copies `cmp`-identical; no unfilled `{{…}}` slot anywhere.

## Stage 3 — agent files last

1. Write `CLAUDE.md` from the template, filling `{{SCOPE}}` with the adopted scope sentence and
   leaving `{{DEPENDENCIES}}`/`{{PREADD}}` as "named in the first stack MADR".
2. Link `AGENTS.md` as a relative symlink to `CLAUDE.md` from the repo root; `git add` both.
3. Copy every pointer file for Claude, Grok, OpenCode, Codex and Kilo (the
   `madr-and-plan-*` set and the `coordinator-agents` set), `templates/opencode.json`, and the
   `.claude/` and `.kilo/` gitignores; `cmp` each.

Exit criterion: all copies `cmp`-identical; `git ls-files -s AGENTS.md` shows mode `120000` with
the link reading exactly `CLAUDE.md` once staged.

Step 11 as planned: ~~Index the `0001` pair in `docs/README.md` via
`scripts/check_records.py --write-index`.~~

*Deviation, 2026-11-22 (owner resolution 1):* the vendored `scripts/check_records.py`
supports only `--next`, `--check` and `--check-all`; `--write-index` exits 2
(`unrecognized arguments`). The hand-written index from stage 2 stands — it already carries
both `0001` rows with their statuses — and is verified by the stage 4 gates instead. This
matches `CLAUDE.md`'s rule that `docs/README.md` indexes every record by hand. Files added
to the phase's scope: none; the tree is unchanged by this resolution.

Exit criterion (unchanged by the deviation): the index in `docs/README.md` carries the `0001`
pair with their current statuses, and `--check-all` passes.

## Stage 4 — gates and plants

1. Run the verification commands below; every exit status 0.
2. Run the plant suite (`tests/test_scaffold_plants.py` from the skill directory) in a scratch
   copy, never in the working tree; its unplanted control must pass and each plant must fail as
   expected.
3. Identifier scan on every added file: placeholders only.

Exit criterion: no blocker, every MADR decided row pinned by a result above, plants caught.

## Verification

```bash
python3 scripts/check_records.py --next          # 0002
python3 scripts/check_records.py --check-all     # exit 0
npx --yes markdownlint-cli2@0.23.2               # exit 0
git ls-files --eol                               # every tracked text file: attr/text=auto eol=lf
git ls-files -s CLAUDE.md AGENTS.md              # 100644 CLAUDE.md, 120000 AGENTS.md → CLAUDE.md
```

Read each whole result before concluding. A new gate counts as used only after stage 4's plants
have shown it failing on a planted input in the scratch copy.

## Rollback

The scaffold lands in a single commit on `main` per the owner's approval; no rollout beyond that.
Rollback is `git revert` of the scaffold commit — no product code depends on any scaffold file
yet, so the revert is clean and total.

## Completion Criteria

- Every stage 4 command exits 0 and its output has been read in full.
- The plant suite's control passes and every planted failure is observed in the scratch copy.
- `cmp` is silent for all byte-for-byte copies; the only edits are the named deltas
  (`.gitignore` merge, `CLAUDE.md` slots, the two `docs/architecture.md` slots, empty legacy
  `filter`).
- No file under `docs/decisions/` or `docs/reports/` other than the `0001` pair.

## Execution Record

**2026-11-22 — grounding.** Clean working tree at `0bbaa13`; whole-repo scan found only this
pair, so `0001` is in force; hooks path is `~/.global-git-hooks`, matching the MADR's
commit-policy premise.

**2026-11-22 — stage 1.** `.gitattributes` and `.gitignore` copied; both `cmp`-identical (the
repo had no `.gitignore`, so the merge is the fragment unchanged).

**2026-11-22 — stage 2.** `.markdownlint-cli2.jsonc` and `scripts/check_records.py` copied,
both `cmp`-identical; legacy override `filter` left empty (greenfield). `docs/README.md` and
`docs/architecture.md` written with all slots filled; root `README.md` gained only the docs
link.

**2026-11-22 — stage 3.** `CLAUDE.md` written with `{{SCOPE}}` filled and
`{{DEPENDENCIES}}`/`{{PREADD}}` set to "named in the first stack MADR". `AGENTS.md` linked
relative. All 13 pointer/config copies `cmp`-identical. The `--write-index` deviation above was
found and recorded at step 11; the owner chose resolution 1.

**2026-11-22 — stage 4, first pass.** `--next` exit 0 printing `0002`; `--check-all` exit 0;
markdownlint exit 1 with 13 issues in the two records — the gate caught the record profile
being violated by both drafts: the MADR carried `### Consequences` where the profile requires
the H2 `## Consequences`, and the PLAN lacked the required `## Grounding Reference` /
`## Prerequisites` / stage / `## Rollback` / `## Completion Criteria` heading order and used
one continuing list number across headings. Both records were restructured to conform (this
restructure also moved the state table and Confirmation to top level); the deviation entry was
re-set as prose after its own `11.` list marker re-triggered MD029. Third markdownlint run:
exit 0, 0 issues; `--check-all` re-run: exit 0. Counted as the second build pass on the stage;
no third pass was needed.

**2026-11-22 — stage 4, second pass.** All scaffold files staged (`git add -A`, per the PLAN's
step for the mode assertion). `git ls-files --eol`: every tracked file `attr/text=auto eol=lf`,
index and working tree. `git ls-files -s CLAUDE.md AGENTS.md`: `100644` and `120000`, link
reads exactly `CLAUDE.md`. Plant suite (`pytest tests/ -q` from the skill directory): 129
passed in 14.86s — the unplanted control passes every gate and each planted defect is caught in
the scratch copy. Identifier scan over every staged line, read from the environment (local
account name, hostname, home-path prefix): clean; records use `<user>` placeholders.
