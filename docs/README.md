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
| 0002 | MADR | [Build a schema-driven MCP framework](decisions/0002-MADR-build-schema-driven-mcp-framework.md) | accepted |
| 0002 | PLAN | [Implement the schema-driven MCP framework](decisions/0002-PLAN-build-schema-driven-mcp-framework.md) | in-progress |
| 0002 | REPORT | [Qualify MCP framework boundaries](reports/0002-REPORT-qualify-mcp-framework-boundaries.md) | complete |
| 0002 | REPORT | [Stage 0 grounding and framework inventory](reports/0002-REPORT-stage-0-framework-inventory.md) | complete — B01 resolved by Option D |
| 0003 | MADR | [Name the Bash and heredocs deny hook](decisions/0003-MADR-name-bash-heredoc-hook.md) | accepted |
| 0003 | PLAN | [Name the Bash and heredocs deny hook](decisions/0003-PLAN-name-bash-heredoc-hook.md) | implemented |
| 0004 | REPORT | [Stage 1 execution: modules, tool pins, and honest gates](reports/0004-REPORT-stage-1-modules-tool-pins-gates.md) | stage 1 complete |
| 0005 | REPORT | [Stage 2 execution: lossless loading, resolution, and mappings](reports/0005-REPORT-stage-2-loading-resolution-mappings.md) | stage 2 complete |
| 0006 | REPORT | [Stage 3 execution: emission and API contract qualification](reports/0006-REPORT-stage-3-emission-api-contract.md) | stage 3 complete |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| understand how this fits together | [architecture.md](architecture.md) |
| know the rules agents follow in this repository | [CLAUDE.md](../CLAUDE.md) |
| know why the repository has the files and gates it has | [0001-MADR-workspace-scaffold.md](decisions/0001-MADR-workspace-scaffold.md) |
