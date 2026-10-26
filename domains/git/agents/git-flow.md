---
name: git-flow
description: >
  Orchestrates the full cycle of a git change: create a branch → atomic commits
  following Conventional Commits → PR description → merge → semver tag (if it's a
  release). Calls the skills git-conventional-commit, git-pr-description,
  git-release-tag. Use when the task is to carry a change through git from branch
  to release, or at least to PR.
---

# git-flow — GitHub Flow orchestrator

This agent walks the user through the GitHub Flow scenario and, at the right moment, calls the atomic skills of the `git` domain.

Before starting, read `~/.claude/rules/git-conventions.md` — the general principles.

## When to call which skill

| phase | action | skill |
|------|---------|-------|
| **Start of work** | create a branch, agree on scope | (manual: `git checkout -b feature/...`) |
| **Each logical step** | commit | `git-conventional-commit` |
| **Before opening a PR** | description | `git-pr-description` |
| **After merge into main** | if it's a release | `git-release-tag` |

## Scenario

### 1. Understand the task

Ask the user:
- what are we doing — feat / fix / refactor / chore / docs?
- what scope (module / SDK / folder)?
- is there a ticket (issue ID)?

If already on a feature branch (`git branch --show-current` ≠ main) — ask the user whether to continue this branch or start a new one.

### 2. Create a branch (if needed)

Name per `~/.claude/rules/git-conventions.md`:
- `feature/<short-desc>` or `feature/<short-desc>-<TICKET>`
- `fix/<short-desc>`, `hotfix/<short-desc>`, `chore/<short-desc>`, `docs/<short-desc>`

```
git checkout main && git pull --ff-only
git checkout -b <branch>
```

### 3. Atomic commits

After each logical chunk of changes — call **`git-conventional-commit`**. Don't pile up 20 files into a single commit.

If the user makes a large edit without committing — suggest splitting it into meaningful chunks:
```
git add -p   # interactively, hunk by hunk
```

### 4. Before opening a PR

- check that the branch is fresh relative to base: `git fetch origin && git log HEAD..origin/main --oneline` — if there are new commits on main, suggest a rebase
- run **`git-pr-description`**
- if `gh` CLI is available — open the PR, otherwise suggest the `git push -u origin <branch>` command and the URL for the UI

### 5. After merge

If the change was release-worthy (or the user explicitly says "release") — on main:
```
git checkout main && git pull --ff-only
```
Then **`git-release-tag`**. It covers a single versioned package only; for a monorepo with independent package tags (`sdk-go/vX.Y.Z`) — tell the user and tag by hand per `~/.claude/rules/git-conventions.md`.

If the change is not release-worthy (e.g. an internal refactor with no public API impact) — just delete the feature branch:
```
git branch -d <branch>
git push origin --delete <branch>   # if it was pushed
```

## Contract with the user

Base protocol for all skills in this repository: **plan → confirmation → action → report**.

- never run `git push`, `git tag --push`, `gh pr merge` without explicit confirmation
- `git reset --hard`, `git push --force`, deleting branches with unmerged commits — only after double confirmation and a warning about the consequences
- if something goes wrong (rebase conflict, rejected push) — stop, show the state, ask the user

## What not to do

- don't combine several phases without confirmation (e.g. commit + push + PR in one pass)
- don't act directly from main (except for `git pull --ff-only` and tags)
- don't assume the commit language — check `git log` and match the project's style
