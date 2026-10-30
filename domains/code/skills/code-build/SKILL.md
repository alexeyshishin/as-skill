---
name: code-build
description: Implement an already-approved plan from swarm-report/<slug>-plan.md. Routes each task to the executing agent that owns its file scope (frontend / backend / devops / mobile), runs the tests, writes an implementation report. Use AFTER code-plan, never before.
---

# Skill: code-build

ORCHESTRATOR. You route work to executing agents by file scope. No code edits in the
main loop.

## Invocation
run the code-build skill with `<slug>`

## Steps
1. **Find the plan**: `swarm-report/<slug>-plan.md`. Missing → abort:
   "Run the code-plan skill for \"<feature>\" first."
2. **Check Blockers** in the plan. Any unresolved HIGH → abort and name it. The user must
   resolve or explicitly waive it.
3. **Route by scope**: read the plan's `affected_files`. Match each against the Executing
   table — the `## Исполнители для /code-build` section of the project's `AGENTS.md`
   (written by code-init), else of `CLAUDE.md` (frontend / backend /
   devops / mobile). Group files by agent. No match for a file → ask the user which agent
   owns it.
4. **Spawn the matched exec agents** — one subagent each, in parallel when a feature
   spans several layers — follow `references/delegation.md` for the host you run on.
   Prompt each:
   > Answer TERSE. Read `<absolute path of references/agents/<scope>.md>` and follow it exactly.
   > Plan: swarm-report/<slug>-plan.md. Your scope: <the files for this agent>.
   If one agent's output changes a contract another needs (e.g. backend `api_changes`),
   pass that note into the dependent agent's prompt — or run backend first, then frontend.
5. **Verify**: read each agent's `tests_result`. Any `status: blocked` or failing test →
   do NOT claim success. Report the failures verbatim and stop. Offer a code-build
   retry after the fix.
6. **Write** `swarm-report/<slug>-build.md`: per-agent changed files, test commands +
   real results, cross-layer notes, `spawn_mode: parallel|sequential|inline`.
7. **Report** to user: status per scope + test results. Quote real output — no "done"
   without proof.