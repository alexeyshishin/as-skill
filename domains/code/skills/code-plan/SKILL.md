---
name: code-plan
description: Design a feature before writing code. Spawns planner + skeptic subagents that read the Memory Bank, argue, and write a plan to swarm-report/. Use BEFORE /code-build, for any feature touching 2+ files or with unclear scope. Skip for typos and one-line fixes.
---

# Skill: /code-plan

You are the ORCHESTRATOR. You do not design the plan yourself — you run two subagents
and merge their output. No code edits here.

## Invocation
`/code-plan "<one-sentence feature>"`

## Steps

1. **Slugify** the feature → `<slug>` (kebab-case, short).
2. **Read context yourself** (cheap, keeps subagents focused): `.memory-bank/index.md`
   if it exists. If it does not, tell the user: "No Memory Bank found — plan will rest
   only on your description." and continue.
3. **Spawn 2 subagents in parallel** — one message, two `Task` calls,
   `subagent_type: general-purpose`. Prefix EACH prompt with:
   > Answer TERSE: terse, drop filler, keep all technical substance.

   - **planner** — "Read `.claude/agents/planner.md` and follow it exactly. Feature: <verbatim>."
   - **skeptic** — "Read `.claude/agents/skeptic.md` and follow it exactly. Feature: <verbatim>."

   (Sharper, slightly pricier variant: run planner first, then pass its YAML into the
   skeptic prompt so it critiques the concrete plan. One extra round-trip. Use it when
   the feature is risky.)
4. **Merge** (orchestrator only): take the planner's plan, apply the skeptic's HIGH/MED
   fixes, and list anything unresolved under **Blockers**.
5. **Write** `swarm-report/<slug>-plan.md`:
   ```
   # Plan: <feature>   (slug: <slug>)
   ## TL;DR
   ## Acceptance criteria  — observable "done" conditions; /code-review checks these
   ## Plan                 — merged steps + affected files + tests
   ## Blockers             — skeptic HIGH not yet resolved (human decides)
   ## Out of scope
   ## Assumptions
   ```
6. **Report to user**: TL;DR + any Blockers. If Blockers exist, ask the user to resolve
   them before `/code-build`.