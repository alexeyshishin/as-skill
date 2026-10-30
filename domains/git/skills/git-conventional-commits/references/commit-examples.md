# Commit corpus — what this author's history actually looks like

Mined from 10 of the user's own repositories (`as-blog`, `as-gitverse-api`,
`as-homelab`, `as-is-organization`, `as-is-practice`, `as-mlops`, `as-practice`,
`as-scraper`, `as-skill`, `project-school-hq`) on 2026-10-26: 155 commits, 147
gradeable after merge/fixup subjects are dropped, 100 of them clean under
`scripts/lint-commit-msg.py`.

**The register is one line.** 123 of 147 non-merge commits (84%) are
subject-only. The 24 that carry a body or footer are the weak spot, not the
model to copy — see A0.

The point of this file is calibration, not doctrine. The convention itself lives
in [`git-conventions`](../../../rules/git-conventions.md) — this is the evidence
of how it gets applied and where it slips, so a generated message lands in the
author's own register instead of a generic one.

## Register, measured

Observed type distribution:

| type | count | note |
|------|-------|------|
| `feat` | 60 | the default reach — and over-reached (see A3) |
| `docs` | 35 | heavy `README.md` maintenance, typical of this author |
| `chore` | 28 | init commits, `.gitignore`, renames |
| `fix` | 11 | |
| `refactor` | 9 | concentrated in `as-skill`'s CLI work |
| `build` | 2 | |
| `test` | 1 | |
| `perf`, `ci`, `revert` | 0 | never used yet — don't force them |

Observed scopes, all lowercase, all one word except `sdk`-style compounds:
`code`, `cli`, `git`, `core`, `obsidian`, `cms`, `site`, `notes`, `install`,
`docs`, `devops`, `content`. Scope is omitted more often than not (≈80% of
commits have none) and that is within the convention — when in doubt, omit.

Language: English throughout, in every repo. There is no
Russian-subject repo in this corpus, so Step 1's "check `git log` for the
language" will say English here; keep the check anyway for other projects.

## Good examples, by type

These are real commits that pass clean. Use them as the shape to imitate.

```
feat(auth): add POST /login endpoint              ← the convention's own example
feat: add available memory metric                 as-homelab
feat: add cleaner for old dashboards              as-homelab
feat: convert default hours into academic hours   as-scraper
feat(install): persist as-skill on PATH across new terminals   as-skill
fix: delete useless legend                        as-homelab
fix(cms): correct duplicated /alpine/ path in apk mirror sed   as-blog
fix(code): register test-gate hook in project settings, not global   as-skill
refactor(cli): extract registry package           as-skill
refactor(core): move core/skills into domains/core as a real domain   as-skill
docs: fix links in README.md                      as-practice
docs(git): forbid Claude-Session link in commit trailers   as-skill
build(cms): switch apk mirror off the fastly-fronted host   as-blog
test: drop stale core/skills fixture from harness-root sniff tests   as-skill
chore(deps): update express to 4.19               convention example
```

Pattern worth copying: the strongest subjects in this history name **the thing
and the delta** (`extract registry package`, `switch apk mirror off the
fastly-fronted host`), not the activity (`update`, `change`, `fill out`).

### When the change "needs" an explanation

It doesn't get a body. `as-blog` `9392586` has a seven-line body about Fastly,
TLS handshakes and `.npmrc`; the subject alone is already the right commit:

```
build(cms): switch apk mirror off the fastly-fronted host
```

The reasoning belongs in the PR description or in a comment next to the changed
line, where it survives a rebase and is read in context. A commit that can't be
summed up in one line is a commit that should have been two.

## Anti-patterns, with the real commits

Ranked by how often they actually occur here. Each is a lint code from
`scripts/lint-commit-msg.py`.

### A0. `body-present` — 23 occurrences, the most common error

Anything after the header: a paragraph, a bullet list, `Refs:`,
`BREAKING CHANGE:`, a trailer. Seen in `as-blog` (both substantive commits) and
throughout `as-skill`'s recent history — typically a wall of "what changed per
file" that the diff already shows, e.g. `docs: document core as a real domain,
drop --with-core` with a four-bullet file-by-file list, or
`docs(git): forbid AI attribution in commits and PRs` with its own bullet list
(the commit that prompted this rule; amended to the bare subject).

Fix, every time: delete everything after line 1. If the subject then feels
inadequate, sharpen the subject or split the commit. A breaking change is the
`!` in the header — the migration note goes in the PR.

### A1. `type-mismatch-docs` — 10 occurrences, the most common slip

A docs-only change filed under `chore` or `feat`.

| actual | should have been |
|--------|------------------|
| `chore: update README.md` (`as-practice`, ×2) | `docs: …` + what changed |
| `chore: update content-documentation skill` (`as-skill`) | `docs(content): …` |
| `feat: update note` (`project-school-hq`) | `docs(notes): …` — the shape the author later got right in `docs(notes): add note for 05.10.2026` |
| `feat: translate all md files into English` (`as-skill`) | `docs: translate all md files into English` |
| `chore(docs): rename note` (`project-school-hq`) | `docs(notes): rename …` |

Rule of thumb for this author: if the diff is only `.md`, it is `docs`, even
when the work felt like housekeeping. `chore` is for `.gitignore`, init
commits, and tooling — see the clean `chore: add swarm-report into .gitignore`.

### A2. `subject-non-atomic` — 7 occurrences

The subject announces two changes with `and` or a comma, which means the commit
should have been two.

| actual | fix |
|--------|-----|
| `feat: add test-gate hook registration, add exec agents structures` | two commits |
| `feat(obsidian): fix manifest yaml and update skills` | a `fix` and a `feat`, split |
| `fix: remove installer from tools and add install into root of whole project` | two commits (also 75 chars, over the limit) |
| `chore: add basic readme.md and fix devops domain structure` | `docs:` + `fix:` |
| `docs: update README.md, add SECURITY and CONTRIBUTING files` | acceptable as one docs commit; prefer `docs: add SECURITY and CONTRIBUTING, refresh README` |

Not every conjunction is a defect — `docs: document core as a real domain, drop
--with-core` describes one coherent documentation change. Judge whether the two
halves could be reverted independently; if yes, split.

### A3. `subject-mood` — 5 occurrences, all the same word

Past tense instead of the imperative, concentrated in one sitting:

```
feat: filled out CLI tool and install.sh
feat(devops): filled out agents
feat(obsidian): filled out agents, rules, skills and manifest.yaml
feat(git): filled out agents, rules, skills and manifest.yaml
feat(content): filled out agents, rules, skills and manifest.yaml
```

Rewrite: `feat(obsidian): add agents, rules and skills`. Note these are also A2
violations — "filled out X, Y, Z and W" is four changes.

### A4. `header-too-long` — 3 occurrences

```
refactor: split into modular monolith (schedule + enrollment modules, shared kernel)   84 chars
refactor(code): drop code-setup skill, install code domain via CLI directly            75 chars
fix: remove installer from tools and add install into root of whole project            75 chars
```

The first one is a good commit with the detail crammed into the subject:
`refactor: split into a modular monolith` is enough — the module names are in
the diff. No body to move them to; shorten the subject instead.

### A5. `ai-attribution` — 2 occurrences

`as-blog` `a390984` and `9392586` both carry
`Co-Authored-By: Claude <noreply@anthropic.com>`. The convention now bans this
outright ([no-ai-attribution](../../../rules/git-conventions.md#no-ai-attribution)),
and the harness will keep proposing it by default — strip it every time. Same
for `Claude-Session:` lines, `🤖 Generated with …`, and any `claude.ai/…` URL.

History note: `as-skill` `b262289` was `feat(git): add Co-Authored-By trailer
for AI-made commits`, later reversed by `docs(git): forbid Claude-Session link
in commit trailers` and the current no-attribution rule. A trailer is reported
as `ai-attribution` rather than a generic `body-present` so the message names the
actual problem; the linter matches whole-line trailer form only, so prose that
merely mentions one is not mistaken for it.

### A6. `subject-vague` / `subject-too-short` — 3 occurrences

```
fix: some UI            as-blog
feat: some UI fix       as-skill
build: dashboard        as-homelab
```

`some`, `various`, `minor`, a bare `update` — these cost the reader a `git show`
to learn anything. Name the component: `fix(web): correct card spacing on
mobile`.

### A7. Repeated subjects — an amend that never happened

Non-lint-fatal but a real signal, reported by `--history`:

```
feat: change type of metric      as-homelab, 3 consecutive commits
fix: metric name                 as-homelab, 2
feat: add first chapter about HA-postgresql   as-practice, 2
docs: update README.md           as-is-practice, 3 consecutive
```

Three commits with one subject means three attempts at one change. Those wanted
`git commit --amend` or `git commit --fixup` plus a rebase before pushing.

Related: `fix: metric name` and `chore: rename panels` pass the linter but are
noun phrases, not changes. A regex cannot catch this; a reviewer can. Prefer
`fix(dashboard): correct the available-memory metric name`.

### A8. Breaking change without `!` — retired as a lint code

`as-skill` `8476c8a` carried a `BREAKING CHANGE:` footer with no `!` in the
header, so tooling computing the next semver saw a `refactor` and proposed a
PATCH bump. With footers banned, the footer half of that mistake can no longer
happen: a breaking change is the `!`, as in
`feat(cli)!: default as-skill install to symlink mode` — and nothing else.
Whether a change *is* breaking stays your call; the linter can't see it.

## Using the linter

```bash
python3 scripts/lint-commit-msg.py -                 # message on stdin
python3 scripts/lint-commit-msg.py .git/COMMIT_EDITMSG
python3 scripts/lint-commit-msg.py --history 50      # audit a repo's log
python3 scripts/lint-commit-msg.py --history 50 --repo ../other-project --json
```

Exit 0 clean, 2 on errors (`--strict` also fails on warnings), 1 on usage
problems — so it drops into a `commit-msg` hook unchanged:

```bash
#!/bin/sh
exec python3 "scripts/lint-commit-msg.py" "$1"
```

What it cannot judge, and you must: whether the type matches what the diff
really did, whether the scope is the right module, whether a noun-phrase subject
describes an actual change, and whether a change marked with `!` really breaks
something.
