---
status: accepted
date: 2026-11-23
---
# Name the Bash and heredocs deny hook

## Context and Problem Statement

The owner reports an unnamed hook and requests `Bash and heredocs Deny`.
The `no-heredoc.py` command handler in `~/.codex/hooks.json` lacks
`statusMessage`, a field supported by the local Codex command handler schema
in `codex-rs/protocol/src/hooks.rs`.

## Decision Drivers

* Use the exact label requested by the owner.
* Preserve the hook command, matcher, and enforcement behavior.
* Use a supported command handler field.

## Considered Options

* Set the existing handler status message to the requested label.
* Retain the unnamed handler.

## Decision Outcome

Chosen option: "Set the existing handler status message to the requested label",
because the owner requested that name and explicitly instructed execution.

### State table

| State | Disposition | Outcome and verification |
| :--- | :--- | :--- |
| One matching unnamed handler | decided | Add the exact label; read back the complete JSON. |
| Missing or duplicate target | decided | Require unique selection before writing. |
| Other configuration values | decided | Verify complete semantic equality after excluding the added field. |
| Live client display | out of scope | Requires a subsequent invocation in the client; report as unverified. |

## Consequences

* Good, because the handler supplies a descriptive status message.
* Good, because the label change preserves all other configuration values.
* Neutral, because the configuration check does not establish what an already
  running client displays before it reloads the hooks.

## More Information

Execution: [0003-PLAN-name-bash-heredoc-hook.md](0003-PLAN-name-bash-heredoc-hook.md).

### 2026-11-23: Documentation correction

The owner authorized correction of the record headings to satisfy the
repository lint profile, followed by staging the documentation for their commit.
This adds the missing rationale sections without changing the hook decision.
