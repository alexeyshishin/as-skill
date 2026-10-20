---
name: code-debug
description: Find the root cause of a bug or failing test. Spawns 1 debugger subagent that reproduces, ladders hypotheses, isolates with evidence, and applies the minimal fix. Use when a test fails, /build reports blocked, or "X stopped working". Does NOT design features (that's /plan).
---

# Skill: /debug

ORCHESTRATOR. One debugger subagent. It diagnoses before it touches anything.

## Invocation
`/debug "<error text / failing test / what broke>"`  (optionally `/debug <slug>` to tie it
to a feature build)

## Steps
1. **Gather the symptom**: the error/stack/failing-test name from the user, plus the build
   report `swarm-report/<slug>-build.md` if a slug was given.
2. **Spawn 1 debugger** (`Task`, `general-purpose`). Prompt:
   > Answer TERSE. Read `.claude/agents/debugger.md` and follow it exactly.
   > Symptom: <verbatim error / failing test>. Feature slug (if any): <slug>.
3. **Report** the debugger's return verbatim: root cause, evidence, fix, and the
   re-run output.
4. **Route the outcome:**
   - `fixed` → suggest `/review <slug>` to check the fix.
   - `root_cause_found` (found but not fixed) or `stuck` → show the hypothesis ladder;
     the user decides the fix, or re-run `/build <slug>` with the root cause as input.
   - `cannot_reproduce` → ask the user for exact repro steps; do not guess-patch.

## Where it sits in the loop
`/plan → /build → /review`. If a test fails or review says `rework` → `/debug` →
back to `/build`. The Stop test-gate hook blocks "done" until tests actually run.