---
name: git-release-tag
description: >
  Cuts a semver release for a single-package repo: checks existing tags, computes
  the next vX.Y.Z from Conventional Commits since the last tag, drafts release
  notes without marketing slop, lints them, then creates an annotated tag and a
  GitHub release via gh. Use when the user wants to "make a release", "cut a
  release", "tag a version", "bump the version", "create a git tag", "publish a
  GitHub release", "write release notes".
---

# git-release-tag — semver tag and GitHub release

Goal: the version number comes from the commits, not from a guess, and the release notes read like a changelog, not like a press release.

Before starting, read `~/.claude/rules/git-conventions.md` — the "Semver and tags" section is the contract this skill executes.

Two assets ship with this skill:
- [`scripts/release.py`](scripts/release.py) — `next` (last tag → next version, commits grouped), `notes` (notes skeleton), `lint` (slop and structure check). No dependencies, Python 3.8+.
- [`references/release-notes.md`](references/release-notes.md) — bump table, when to override it, notes anatomy, summary-line examples, the slop list. Read it before writing the summary line.

Scope: one version per repo, tags `vMAJOR.MINOR.PATCH`. Monorepo tags (`<pkg>/vX.Y.Z`) and pre-releases (`-rc.N`) are out of scope — say so and stop.

## Step 1. Preconditions

```bash
git fetch --tags origin
git branch --show-current          # must be main
git status -sb                     # must be clean and level with origin/main
```

- Not on main, or HEAD ≠ `origin/main` → stop. A tag goes only on a merge commit in main.
- CI must be green on HEAD (convention). With `gh`: `gh run list --commit "$(git rev-parse HEAD)" --limit 5`. Failed or still running → stop and tell the user.

## Step 2. Compute the version

```bash
python3 scripts/release.py next
```

Read the output:
- exit 3 — only `build`/`ci`/`chore`/`test` since the last tag. Tell the user there's nothing to release; continue only if they insist, with `--bump patch`.
- `WARN tags newer than the base are not ancestors of HEAD` — stop and show the user. Either the wrong branch or main was rewritten after a release; don't tag over it.
- `WARN ignored non-semver tags` — mention them once in the report. Don't delete or rename them without an explicit request.
- `Not conventional — read the diff and decide` — run `git show --stat <sha>` on each and decide whether it changes the bump.

The script reads headers only. Check the commits marked `*` (they drive the bump) against the override cases in `references/release-notes.md`. If the header lies, re-run with `--bump major|minor|patch` and note which commit forced it.

## Step 3. Draft the notes

```bash
NOTES="$(mktemp -t release-notes)"
python3 scripts/release.py notes [--bump LEVEL] > "$NOTES"
```

The skeleton has every section filled from the commits and a placeholder first line. Edit the file:

1. Replace the placeholder with the summary line — one sentence, the change a reader most needs, ≤ 120 chars, no "This release…". Examples and the picking order are in the reference.
2. Breaking bullets: append the migration in one clause (`— use X instead`). Take it from the PR description if there is one.
3. Rewrite a bullet only if a reader outside the repo can't parse it. Never add a bullet without a commit behind it.
4. Drop `Other` bullets nobody upgrading cares about. Drop a section that ends up empty.

Language: the summary line follows the project's commit language (`git log`). Section headings stay as the script emits them.

## Step 4. Lint

```bash
python3 scripts/release.py lint "$NOTES"
```

Exit 0 — proceed. Exit 2 — fix and re-run; don't show the user notes that fail. A remaining warning is fine only if you say why you kept it.

The linter checks words and structure, not truth: whether the summary names the most important change, and whether a bullet is accurate, stays your judgment.

## Step 5. Show the plan and wait

Show in one block:
- tag `vX.Y.Z` → commit `<sha> <subject>`
- bump level and what drove it (the `*` commits, or the override reason)
- the notes, as they'll be published
- the commands from Step 6

**Don't create, push, or publish anything without the user's explicit go-ahead.** Pushing a tag and creating a release are publishing — if the user approved only the tag, ask again before the push.

## Step 6. Execute

```bash
git tag -a vX.Y.Z -m "vX.Y.Z" <sha>
git push origin vX.Y.Z
gh release create vX.Y.Z --verify-tag --title "vX.Y.Z" --notes-file "$NOTES"
```

- The tag message is the version only — the notes live in the release, one place.
- No `gh`, `gh auth status` fails, or `origin` isn't GitHub → stop after the push and print the notes for manual pasting.
- Push rejected, or the tag already exists on the remote → stop, show the state, ask. Never `git tag -f` or `git push --force` a tag.

## Step 7. Report

`gh release view vX.Y.Z --json url -q .url` — the URL, the version, the bump reason, any ignored tags from Step 2. Delete `$NOTES`.

## What not to do

- Don't use lightweight tags — always `-a`
- Don't use `gh release create --generate-notes` — GitHub's own format (`What's Changed`, @mentions, `New Contributors`) replaces these notes
- Don't move or re-create an existing tag; a bad release is fixed by the next PATCH
- Don't tag from a feature branch, or from main ahead of/behind origin
- Don't add emoji, `!`, or any AI attribution (`🤖 Generated with Claude Code`, `Co-Authored-By`) to the notes, even if the calling harness suggests one — this repo's convention overrides that
- Don't pad a thin release — a one-bullet release gets one bullet
