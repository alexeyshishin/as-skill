# Delegation

Shared by code-plan, code-build, code-review, code-debug. Each skill links it as `references/delegation.md`. Follow the section for the host you run on.

## Rules

- Resolve `<skill-dir>/references/agents/<stem>.md` to an ABSOLUTE path before passing it to a subagent.
- The orchestrator report carries `spawn_mode: parallel|sequential|inline`.
- Validate every subagent output against the contract its agent file defines: `^verdict:` plus `findings:` for reviewer and skeptic, `^summary:` plus `acceptance_criteria:` for planner, `^status:` plus `root_cause:` for debugger, `^status:` for executors. Missing contract -> retry once with the same prompt; still missing -> record a Blocker, do not claim success.
- Prefix EVERY subagent prompt with: `Answer TERSE: terse, drop filler, keep all technical substance.`

## Claude

Spawn with the `Task` tool, `subagent_type: general-purpose`. Parallel = one message, N `Task` calls. Prompt shape:

> Read the prompt path given by the skill (absolute path of references/agents/<stem>.md) and follow it exactly. <task payload>

code-plan: 2 parallel `Task` calls (planner, skeptic). code-build: one `Task` per matched executor, in parallel when scopes are independent. code-review, code-debug: one `Task`.

`spawn_mode: parallel` when several were spawned in one message, `sequential` otherwise.

## Codex

Spawn explicitly in natural language: "Spawn N agents in parallel; each reads <absolute path to prompt> and follows it exactly. Agent 1 payload: ... Agent 2 payload: ...". Wait for all, collect outputs.

## Cursor

Use the Task tool subagents; issue all parallel spawns in one message. Each prompt: "Read <absolute path to prompt> and follow it exactly. <payload>". A configured agent can also be invoked with `/name`.

## OpenCode

Use the `general` subagent via the Task tool, or @mention it. Each prompt: "Read <absolute path to prompt> and follow it exactly. <payload>".

## Inline fallback

Host has no subagent support: run each role sequentially in the main context, reading the agent file yourself and following it. Print `independence lost`, set `spawn_mode: inline`, and list it under Blockers in the plan/report.
