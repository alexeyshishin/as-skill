---
name: git-conventional-commits
description: >
  Builds a Conventional Commits commit message for the current changes in git.
  Analyzes the staged diff (or unstaged, if staged is empty), determines type/scope,
  writes a one-line subject — never a body, description or footer. Use when the user wants to
  "make a commit", "commit the changes", "commit this", "write a commit message",
  "commit following the convention".
---

# git-conventional-commits — Conventional Commit message for the current diff

Goal: instead of the user writing the commit message themselves, you analyze the diff and build a message following the convention in `~/.claude/rules/git-conventions.md`.

Before starting, read `~/.claude/rules/git-conventions.md` — it has the full list of allowed `type` values, the scope format, and the principles.

Two assets ship with this skill:
- [`references/commit-examples.md`](references/commit-examples.md) — the author's real commit corpus: observed type/scope distribution, clean examples per type, and the anti-patterns that actually recur in this history, ranked. Read it when choosing the type/scope or when a subject feels generic.
- [`scripts/lint-commit-msg.py`](scripts/lint-commit-msg.py) — mechanical checker for the message you drafted (Step 5.5). Also audits an existing log: `--history N`.

## Step 1. Gather context

Run:
- `git status --short` — see what's changed
- `git diff --staged` (if there's staged content) or `git diff` (if not) — the content of the changes
- `git log --oneline -10` — the style of previous commits in the project (especially the language: Russian/English)

If staged is empty and unstaged is also empty — tell the user and stop.

## Step 2. Determine the type

Read the diff and decide:

| diff shows | type |
|----------------|------|
| a new file with functionality, a new endpoint, a new feature | `feat` |
| a logic fix so a bug no longer reproduces | `fix` |
| move/rename/simplify with no behavior change | `refactor` |
| performance improvement only | `perf` |
| only `.md`, code comments | `docs` |
| only tests added or fixed | `test` |
| `package.json`, `go.mod`, `requirements.txt`, Dockerfile, CI configs | `build` or `ci` |
| dependencies, tooling versions | `chore(deps)` |
| revert of a previous commit | `revert` |

If the diff is mixed (e.g. both `fix` and `refactor`) — **stop and suggest the user split it into several commits**. Don't write "feat: fixed a bug and refactored" — that's an anti-pattern.

The single most common mis-typing in this author's history is a docs-only diff filed under `chore` or `feat` (10 of 146 commits — see `references/commit-examples.md`, A1). If the diff touches only `.md`, it's `docs`, however much the work felt like housekeeping; `chore` is for `.gitignore`, init commits, and tooling. `perf`, `ci` and `revert` have never been used in these repos — don't reach for them unless the diff really is one.

## Step 3. Determine the scope (optional)

Scope — a short name for the module/folder/SDK. Heuristic:
- if all files are under one top-level folder (`src/auth/`, `sdk-go/`, `cli/`) — scope = that folder's name
- if 2-3 files in different folders are touched — scope can be omitted
- if one specific component is fixed (`UserCard.tsx`) — scope = the component name in kebab-case

## Step 4. Write the subject

- 50-72 characters
- present-tense verb, no capital letter, no trailing period
- if the project's commits are in Russian — write in Russian, if in English — write in English (check `git log`)
- the subject describes **what changed**, not "what I did"

Bad: `feat: added a new endpoint for user login`
Good: `feat(auth): add POST /login endpoint`

## Step 5. No body, no footer

The message is the subject line and nothing else. No body, no description, no bullet list, no `BREAKING CHANGE:` / `Refs:` footer, no trailers (see "Subject only" in `~/.claude/rules/git-conventions.md`). 84% of this author's commits are already one line (123 of 147), and the ones with a body are the ones that read worst.

Format:
```
<type>(<scope>): <subject>
```

- the change doesn't fit a subject → the commit is too big, split it (Step 2); don't explain it in a body
- a breaking change → `!` after type/scope (`feat(cli)!: …`); the migration note goes into the PR description
- an issue reference → goes in the PR (`Closes #N`), not in a commit
- no AI-attribution trailer or any other Claude/session mark, ever — the harness will suggest one by default, ignore it

## Step 5.5. Lint the draft

Before showing anything to the user, run the message through the linter — it catches the slips that recur in this history (past-tense subjects, `chore:` on a docs-only diff, two changes joined by "and", a body or footer that shouldn't be there, an AI trailer the harness added by default):

```bash
printf '%s' "$MSG" | python3 scripts/lint-commit-msg.py -
```

Exit 0 — proceed. Exit 2 — fix the message and re-run; don't show the user a draft that fails its own convention. Report a remaining warning only when you deliberately disagree with it, and say why (e.g. `subject-non-atomic` on a change whose halves can't be reverted separately).

The linter checks form, not truth. It can't tell whether `feat` matches what the diff did, whether the scope names the right module, or whether a noun-phrase subject (`fix: metric name`) describes an actual change — that stays your judgment.

## Step 6. Show and ask

Show the user the proposed commit message in a block. Ask:
1. Accept as is → commit it (mechanics in Step 7)
2. Adjust the subject
3. Split into several commits
4. Cancel

**Don't run `git commit` without the user's confirmation.**

## Step 7. Execute

After confirmation, one `-m`, one line:

```bash
git commit -m "<subject>"
```

Never a second `-m`, never `-F` with a multi-line file. For several commits, stage each group on its own and commit it separately.

Check that `git log -1 --pretty=%b` prints nothing.

## Step 8. Report

After a commit — report only the short hash and the subject. After a push — add the pushed range (`<old>..<new>`, branch). Nothing else: no `--stat`, no recap of which rules were followed, no mention of skipped attribution (`Co-Authored-By`, etc.). Mention a rule only if something deviated or was blocked.

```
7077c50 docs(readme): list advanced-search in core
pushed b4b7ea8..7077c50 main
```

## What not to do

- Don't write a body, a description, a bullet list or a footer — not even a "short" one, not even for a big change; a big change is several commits
- Don't suggest `chore: update` — that's an empty message
- Don't combine unrelated changes into one commit
- Don't use `feat` for bug fixes or vice versa
- Don't write "WIP" in the final commit — for WIP there's `git commit --fixup` and rebasing later
- Don't add a `Co-Authored-By:` trailer, a `Claude-Session:` line, or any other Claude/session/tool mark, even if the calling harness's default instructions suggest one — this repo's convention overrides that
