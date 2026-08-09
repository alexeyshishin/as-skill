---
name: code-setup
description: Install this mini dev-loop harness into a target project. Copies the agents, skills (plan/build/review/debug), the test-gate hook, settings, and AGENTS.md into the given project path, then seeds a Memory Bank index if none exists. Run this from a clone of the harness repo — pass the absolute path to the project you want to equip.
---

# Skill: /setup

Install the harness into another project. You run in a clone of the harness repo; the
target is a different directory the user names.

## Invocation
`/setup <absolute-path-to-target-project>`

If no path is given, ask for one. Do not guess.

## Steps

1. **Resolve paths.**
   - `SRC` = this harness repo root (`$CLAUDE_PROJECT_DIR`).
   - `DST` = the path argument. Expand `~`. Abort if it does not exist or is not a
     directory: "Target `<path>` not found — give the absolute path to your project."
   - Refuse if `DST` equals `SRC` (don't install the harness into itself).

2. **Git check.** If `DST` is not a git repo (`git -C "$DST" rev-parse` fails), tell the
   user and offer to `git init` it. Do not init without a yes.

3. **Copy the harness files** into `DST` (create parent dirs as needed):
   - `AGENTS.md`
   - `.claude/agents/` — the whole directory.
   - `.claude/skills/plan`, `.claude/skills/build`, `.claude/skills/review`,
     `.claude/skills/debug`, `.claude/skills/design-cover`, `.claude/skills/visual-verify`
     — but NOT `.claude/skills/setup` (the target never re-installs).
   - `.claude/hooks/test-gate.sh` and `.claude/hooks/visual-gate.sh` — then `chmod +x` both.
   - `.claude/design-gate.json.example` — the example only. The real
     `.claude/design-gate.json` is the project's to write (`/design-cover` writes it), and
     until it exists the visual gate is inert.
   - `swarm-report/` — create it with a `.gitkeep`.
   Example:
   ```bash
   mkdir -p "$DST/.claude/agents" "$DST/.claude/skills" "$DST/.claude/hooks" "$DST/swarm-report"
   cp -R "$SRC/.claude/agents/." "$DST/.claude/agents/"
   for s in plan build review debug design-cover visual-verify; do cp -R "$SRC/.claude/skills/$s" "$DST/.claude/skills/"; done
   for h in test-gate visual-gate; do cp "$SRC/.claude/hooks/$h.sh" "$DST/.claude/hooks/" && chmod +x "$DST/.claude/hooks/$h.sh"; done
   cp "$SRC/.claude/design-gate.json.example" "$DST/.claude/"
   cp "$SRC/AGENTS.md" "$DST/AGENTS.md"
   touch "$DST/swarm-report/.gitkeep"
   ```

4. **Wire settings.json (merge, don't clobber).**
   - If `DST/.claude/settings.json` is absent → copy `SRC/.claude/settings.json`.
   - If present → merge both Stop hooks (`test-gate`, `visual-gate`) into the existing file
     with `jq`, without dropping the user's other hooks. Dedupe by command string — a
     re-run must not register the same hook twice. Verify the result is valid JSON.

4b. **Append the design-loop ignores** to `DST/.gitignore` (append-only, never rewrite):
   `swarm-report/visual/` and `design-ref/**/*.png` if the user does not want reference
   exports in git. Ask before adding the second one — some teams do commit them.

5. **Seed the Memory Bank (never overwrite).**
   - If `DST/.memory-bank/` does not exist → copy `SRC/.memory-bank/index.md` into it as a
     starter template and tell the user to fill it in.
   - If it exists → leave it untouched; the harness reads whatever is already there.

6. **Report.** List exactly what was written, note whether the Memory Bank was seeded or
   left alone, and give the next step:
   > Installed. Open `<DST>` in Claude Code and run `/plan "<your first feature>"`.

## Notes
- Idempotent: re-running overwrites the copied agents/skills/hook with the current version
  but never touches `.memory-bank/` or the user's own settings hooks.
- This skill only moves files. It runs no build and edits no user code.