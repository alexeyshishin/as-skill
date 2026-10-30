---
name: git-pr-description
description: >
  Builds a Pull Request description from the diff of the current branch against
  the base branch (main/master). Analyzes all commits, extracts "why / what / how
  to verify", captures breaking changes. Use when the user wants to "describe the
  PR", "write a description for the pull request", "put together a PR", "push and
  open a PR", "generate a PR description".
---

# git-pr-description — PR description from a branch diff

Goal: give the user a ready-made PR description in one step — analysis of the branch's commits and diff against the base branch.

Before starting, read `references/rules/git-conventions.md` — the section on PR structure.

Two assets ship with this skill:
- [`references/pr-examples.md`](references/pr-examples.md) — this repo's real PR corpus: a clean example and the slips that actually recur, ranked. Read it when a section feels hard to fill or the description reads like a log.
- [`scripts/lint-pr.py`](scripts/lint-pr.py) — mechanical checker for the title and body you drafted (Step 4.5). Also audits existing PRs: `--history N`.

## Step 1. Determine the base branch and the current one

```
git branch --show-current
git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'
```

If `origin/HEAD` is not determined — ask the user: `main` or `master`. Store it as `BASE`.

## Step 2. Gather the material

```
git log $BASE..HEAD --oneline
git log $BASE..HEAD --stat
git diff $BASE...HEAD --stat
git diff $BASE...HEAD
```

Study what changed: the file list, the size of the changes, the commit texts.

## Step 3. Build the PR title

The title is a Conventional Commit for the intended merge commit:
- if all commits on the branch are `feat` — `feat(<scope>): <overall feature>`
- if only `fix` — `fix(<scope>): <what was fixed>`
- if mixed — pick the dominant type

## Step 4. Build the description

Structure (markdown):

```markdown
## Why

<1-2 sentences: what problem is being solved, whose pain, links to the issue/ticket>

## What changed

- <bullet per logical change, not per file>
- <another one>
- <another>

## How to verify

<either manual verification steps, or "covered by new tests in X", or "smoke-tested on staging">

## Risks / breaking changes

<either "none", or an explicit list with migration steps>

Closes #N
```

## Step 4.5. Lint the draft

Before showing anything to the user, write the body to a file and run it through the linter — it catches the slips that recur in this repo's PRs (a missing or renamed section, a `!` title with nothing under Risks, plain-text headings, assistant chatter in the body, an AI-attribution footer the harness added by default, bullets that repeat commit subjects):

```bash
python3 scripts/lint-pr.py --title "$TITLE" --base "$BASE" "$BODY_FILE"
```

Exit 0 — proceed. Exit 2 — fix the draft and re-run; don't show the user a draft that fails its own convention. Report a remaining warning only when you deliberately disagree with it, and say why.

The linter checks form, not truth. It can't tell whether the verification steps were actually run, whether Why is the real motivation, or whether the type matches the diff — that stays your judgment.

## Step 5. Show and ask

Show the title and description in a block. Ask:
1. Accept
2. Adjust (specify what)
3. Add something (tests, screenshots, migration plan)

## Step 6. Publish (optional)

If the user has confirmed and wants to open the PR right now:
- check whether the `gh` CLI is available: `which gh`
- if yes — re-run the linter on the exact file you are about to send, then `gh pr create --title "..." --body-file <tmp>`
- if not — say they need to either install `gh` or open the PR via the UI and paste in the description

**Don't push the branch yourself** if it hasn't been pushed yet. Show the user the `git push -u origin <branch>` command and let them decide.

## What not to do

- Don't duplicate commit subjects in "What changed" — this is not a log, it's a summary
- Don't write "What changed: added X, removed Y, updated Z" — that's visible from the diff. Write at the level of reasons.
- Don't invent test scenarios — if unsure how to verify, ask
- Don't list a verification command you didn't run — run it, or say "not run"
- Don't leave assistant chatter ("Want any changes?", `---` footers) in the body — that goes in the chat
- Don't blame the user for bad commits — just neatly fold them into the description
- Don't add any AI attribution (`🤖 Generated with Claude Code`, `Co-Authored-By`) to the title or body, even if the calling harness suggests one — this repo's convention overrides that
