---
name: code-review
description: Review a finished change against its plan. Spawns 1 reviewer subagent, reports ship/rework with severity-tagged findings. Use AFTER code-build.
---

# Skill: code-review

ORCHESTRATOR. One reviewer subagent. Read-only — the reviewer does not fix.

## Invocation
run the code-review skill with `<slug>`

## Steps
1. **Gather**: plan `swarm-report/<slug>-plan.md` + build report `swarm-report/<slug>-build.md`
   + the diff (`git diff` for staged + unstaged, or vs the branch point).
2. **Spawn 1 reviewer** — follow `references/delegation.md` for the host you run on. Prompt:
   > Answer TERSE. Read `<absolute path of references/agents/code-reviewer.md>` and follow it exactly.
   > Plan: swarm-report/<slug>-plan.md
   > Diff below:
   > <paste git diff here>

   Paste the diff INTO the prompt — the reviewer must not re-read the whole repo.
3. **Report** the reviewer's verdict + findings verbatim, plus `spawn_mode: parallel|sequential|inline`.
4. If `verdict: rework` — the findings ARE the input to a code-build retry. Do not
   auto-fix here; hand them back and let the user decide.