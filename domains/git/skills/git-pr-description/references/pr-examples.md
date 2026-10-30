# PR corpus — what this repo's pull requests actually look like

Mined from the 7 PRs of `as-skill` on 2026-10-30 (#1–#7, all states). Every one
was graded by `scripts/lint-pr.py --history 10`: 0 of 7 clean. The sample is
small, so read it as a list of observed slips, not as statistics.

The convention itself lives in [`git-conventions`](../../../rules/git-conventions.md#pull-request).
This file is the evidence of where it gets missed.

## What a clean PR looks like

Title is a Conventional Commit for the merge commit; body has exactly four
`##` sections in this order; nothing else.

```markdown
## Why

Core skills lived outside `domains/` with no manifest, so `core` was the one
domain-shaped concept that did not behave like a domain: a second
install/list/check code path and a standing exception to "`domains/` is the
single source of truth".

## What changed

- `core` is an ordinary domain: `domains/core/manifest.yaml`, installed, listed and validated through the shared per-domain path.
- The parallel core-only install machinery and the `--with-core` flag are gone.
- Harness root detection no longer names a specific domain, so renaming one cannot break it again.
- Old lockfile rows are migrated on load, so reinstalling does not duplicate entries.

## How to verify

`cd tools && go test ./...`; then `as-skill add core -p <scratch> -y` and
`as-skill remove core -p <scratch> -y` leave the scratch project unchanged.

## Risks / breaking changes

**Breaking:** `--with-core` is removed. Replace `add code --with-core` with `add code core`.
```

Source: PR #6, reshaped — the original had no verify and no risks section.

## Anti-patterns, ranked by how often they recur

### A1. A required section is missing — 20 findings in 7 PRs

The most common slip by a wide margin. Only #2 carries all four sections as `##`.
Variants seen:

- **No section at all** — #1 is two bullets and nothing else.
- **Plain-text headings** — #3, #4, #5 write `Why` / `What changed` on their own line instead of `## Why`; they render as paragraphs and the structure is lost.
- **Renamed section** — #7's first version had `## What`, `## Notes` instead of `## What changed`, `## Risks / breaking changes`. Notes is not a substitute: the reviewer looks for the breaking-change section by name.
- **No verify** — #3, #4, #5, #6: nothing says how to check the change.

Fix: take the template from `SKILL.md` Step 4 verbatim; write `none` under Risks instead of dropping it.

### A2. Breaking title, undocumented break — 1 finding

#6 is `refactor(core)!:` — the `!` promises a break — but has no Risks section
telling the reader what breaks and how to migrate. The migration note is the
whole reason the PR description exists for a breaking change (commits are
subject-only, so this is the only place for it).

Fix: `!` in the title ⇒ Risks lists the break and the migration step. Risks
that says "none" under a `!` title is a contradiction.

### A3. Assistant chatter left in the body — 1 finding

#2 ends with `---` and `Want any changes to this before I open it (once gh is
authenticated)?` — a question to the user pasted into the PR text. The
confirmation step belongs to the conversation, not the description.

Fix: show the draft, ask in chat, publish only the description itself.

### A4. Title is not a Conventional Commit — 1 finding

#1 is `Docs/agents md cleanup`. The title becomes the merge commit subject, so
it follows the same grammar as a commit: `docs(agents): mandate caveman, add readme`.

### A5. AI attribution footer — seen, caught by hand

The first version of #7 ended with a `🤖 Generated with Claude Code` line
copied in from the harness default. Removed by hand; `lint-pr.py` now flags any
line that starts with it (`ai-attribution`), plus `Co-Authored-By:` and
`Claude-Session:`. A PR whose prose merely *talks about* the trailer (#5) is not
flagged — only a line that is the trailer.

### A6. Verification that was never run

Not a lint rule — the linter cannot tell. The first version of #7 listed
`go build`, `go test`, and `--dry-run` commands that had not been executed.
Either run them and list what passed, or say "not run" — never present an
untested command as the verification.

## Register

- Language follows the repo: this repo's PR text is English, README prose is Russian.
- Bullets summarize a logical change in a sentence; the 3-line-plus bullet with
  sub-bullets per file (#3) is the same file-listing the diff already shows.
- Why is 1–2 sentences of motivation; the linter warns past 3 (`why-long`).
