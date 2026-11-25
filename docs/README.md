# Documentation

This tree holds three kinds of document:

- reports that measured something;
- the decisions behind the work;
- guides you follow to do a thing.

- **[architecture.md](architecture.md)** — how the pieces fit together, as they are now.
- **guides/** — task-shaped walkthroughs. None exists yet; the first real guide creates this directory.
- **[decisions/](decisions/)** — numbered MADR/PLAN pairs.
- **[reports/](reports/)** — measured findings and qualification evidence.

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Adopt the fleet workspace scaffold](decisions/0001-MADR-workspace-scaffold.md) | accepted |
| 0001 | PLAN | [Implement the fleet workspace scaffold](decisions/0001-PLAN-workspace-scaffold.md) | in-progress |
| 0002 | MADR | [Build a schema-driven MCP framework](decisions/0002-MADR-build-schema-driven-mcp-framework.md) | accepted — local-cicd-scaffolding scope recorded |
| 0002 | PLAN | [Implement the schema-driven MCP framework](decisions/0002-PLAN-build-schema-driven-mcp-framework.md) | in-progress — R1 qualified on four environments; R2 next |
| 0002 | REPORT | [Qualify MCP framework boundaries](reports/0002-REPORT-qualify-mcp-framework-boundaries.md) | complete |
| 0002 | REPORT | [Stage 0 grounding and framework inventory](reports/0002-REPORT-stage-0-framework-inventory.md) | B01 resolved; stage exit incomplete per completion audit |
| 0003 | MADR | [Name the Bash and heredocs deny hook](decisions/0003-MADR-name-bash-heredoc-hook.md) | accepted |
| 0003 | PLAN | [Name the Bash and heredocs deny hook](decisions/0003-PLAN-name-bash-heredoc-hook.md) | implemented |
| 0004 | REPORT | [Stage 1 execution: modules, tool pins, and honest gates](reports/0004-REPORT-stage-1-modules-tool-pins-gates.md) | historical completion claim; repairs required per audit |
| 0005 | REPORT | [Stage 2 execution: lossless loading, resolution, and mappings](reports/0005-REPORT-stage-2-loading-resolution-mappings.md) | historical completion claim; repairs required per audit |
| 0006 | REPORT | [Stage 3 execution: emission and API contract qualification](reports/0006-REPORT-stage-3-emission-api-contract.md) | historical completion claim; repairs required per audit |
| 0007 | REPORT | [Stage 4 execution: CLI, deterministic publication, and recovery](reports/0007-REPORT-stage-4-cli-publication-recovery.md) | historical completion claim; repairs required per audit |
| 0008 | REPORT | [Stage 5 execution: strict JSON codec and canonical validator](reports/0008-REPORT-stage-5-strict-json-validation.md) | historical completion claim; repairs required per audit |
| 0009 | REPORT | [CI/CD standardization assessment](reports/0009-REPORT-cicd-standardization-assessment.md) | complete — owner decisions recorded |
| 0010 | REPORT | [Stage 6 execution: owned runtime APIs and feature matrix](reports/0010-REPORT-stage-6-runtime-api-feature-matrix.md) | historical completion claim; repairs required per audit |
| 0011 | REPORT | [Pre-Stage 7 completion audit](reports/0011-REPORT-pre-stage-7-completion-audit.md) | complete — prerequisite repairs recommended |
| 0012 | REPORT | [Local gate duplication and execution costs](reports/0012-REPORT-local-gate-duplication-assessment.md) | assessment complete — consolidation and timings pending |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| understand how this fits together | [architecture.md](architecture.md) |
| know the rules agents follow in this repository | [CLAUDE.md](../CLAUDE.md) |
| know why the repository has the files and gates it has | [0001-MADR-workspace-scaffold.md](decisions/0001-MADR-workspace-scaffold.md) |
| review the selected CI/CD policy | [CI/CD standardization assessment](reports/0009-REPORT-cicd-standardization-assessment.md) |
| understand why Stage 7 is waiting | [Pre-Stage 7 completion audit](reports/0011-REPORT-pre-stage-7-completion-audit.md) |
| review the prerequisite repair work | [Proposed R1–R5 amendment](decisions/0002-PLAN-build-schema-driven-mcp-framework.md#proposed-repair-amendment--r1r5-before-stage-7) |
| compare repair implementation options | [R1–R5 research, recommendations and required work](decisions/0002-PLAN-build-schema-driven-mcp-framework.md#research-refinement--2026-11-25) |
| streamline local checks without losing coverage | [Local gate duplication assessment](reports/0012-REPORT-local-gate-duplication-assessment.md) |
