---
status: "accepted"
date: 2026-11-22
decision-makers: "<owner>"
consulted: "<user>"
informed: "<user>"
---

# Adopt the fleet workspace scaffold

## Context and Problem Statement

`go-mcp-lib` is a Go library for building MCP (Model Context Protocol) servers and clients, with
transport and tooling kept in-repo as an adjacent stack. The repository currently contains only a
stub `README.md` and an initial commit; it has no agent layer, no docs tree, and no record
tooling. Without one, every clone and every harness gets a different set of rules, mutations
happen with no decision record, and there is no repository-wide numbering scheme for records
before the first real decision arrives.

The fleet maintains a standard workspace scaffold: `CLAUDE.md` as the canonical instruction file
with `AGENTS.md` as a symlink to it, thin pointer files for five harnesses (Claude, Grok,
OpenCode, Codex, Kilo), a `docs/` tree with a ToC and architecture page, `scripts/check_records.py`
as the authority on record numbering and links, and a markdownlint profile that treats records
differently from ordinary prose.

## Decision Drivers

- Any clone, on any of the five harnesses, must get the same gates: write a MADR/PLAN pair before
  mutating, and follow the docs-tree layout.
- The instruction file must not drift: one canonical `CLAUDE.md`, `AGENTS.md` reachable only as a
  link.
- Record numbering is repository-wide from `0001`; the scaffold pair takes `0001` so later
  decisions never reuse it.
- The scaffold must be re-verifiable: `check_records.py`, markdownlint, and mode checks run as
  plain commands with no host-specific dependencies.

## Considered Options

* Adopt the fleet workspace scaffold as specified (chosen)
* Ad-hoc agent layer: write `CLAUDE.md` by hand now, add tooling when it hurts
* No agent layer: leave the repo as a plain product repository

## Decision Outcome

Chosen option: "Adopt the fleet workspace scaffold as specified", because the repository is
greenfield — the scaffold lands before any product source, so there is nothing to migrate and no
legacy records to exempt. The ad-hoc option defers a decision that must be unanimous across
clones, and retrofitting the docs tree after records exist costs a legacy-filter adoption plan;
the no-agent option leaves the mutate-only-with-a-pair rule unenforced in this repository.

## Consequences

Good, because a fresh clone on any harness reaches the same gates through the same files, and
`AGENTS.md` cannot drift from `CLAUDE.md` because it is mode `120000`.

Good, because `check_records.py` is in place from `0001`, so the first product decision records
its own number correctly with no adoption plan.

Bad, because the repository carries fourteen scaffolding files before any product source exists,
which is weight for a repo with one commit.

Bad, because the docs tree's architecture page describes a repository that does not yet have its
library structure, so it must be amended by the first stack MADR.

## Confirmation

The scaffold is in force when `python3 scripts/check_records.py --check-all` exits 0,
`npx --yes markdownlint-cli2@0.23.2` exits 0, and `git ls-files -s CLAUDE.md AGENTS.md` shows
`100644` and `120000` respectively.

## State table

| Input or state | Ruling |
| :--- | :--- |
| Fresh clone, all five harness homes present | decided — in-repo pointers load the writer skills by `name:`; test: gate run in Verification of the PLAN |
| Clone missing user-skill homes (`~/.agents/skills` absent) | decided — in-repo pointer files still exist and name the skills, so the gate is discoverable without the homes; test: pointer files present and `cmp`-identical |
| Repo gains a `Makefile` later | decided — splice the portable `SHELL` and `check-records`/`markdownlint` targets at that time; pinned by the first Makefile commit |
| Repo with pre-existing records adopting the lint profile | out of scope — greenfield here; the legacy override's `filter` stays empty and is filled only per the refresh procedure |
| Existing instruction file (`CLAUDE.md` or regular `AGENTS.md`) | out of scope — repository is greenfield; the refresh procedure governs if it ever applies |
| Product source or CI in the scaffold commit | out of scope — the bootstrap exception covers the pair, `CLAUDE.md` with its link, and the pointer trees only |

## More Information

Template provenance, apply order, and the verification gates live in the fleet
`workspace-scaffolding` skill; the implementation steps are in
[0001-PLAN-workspace-scaffold.md](0001-PLAN-workspace-scaffold.md).
