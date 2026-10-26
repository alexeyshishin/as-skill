# Release notes — format and anti-slop rules

Companion to `SKILL.md`. The script (`scripts/release.py`) enforces the mechanical half of this file; the rest is judgment.

## Version: what decides the bump

The script takes the highest level among commits since the last `vX.Y.Z` tag reachable from HEAD:

| commit | level |
|--------|-------|
| `!` after type/scope, or a `BREAKING CHANGE:` footer | MAJOR (MINOR while `0.x`) |
| `feat` | MINOR |
| `fix`, `perf`, `docs`, `refactor`, `revert` | PATCH |
| `build`, `ci`, `chore`, `test`, `style` | none |
| not a Conventional Commit | none — listed separately, read the diff |

No previous tag → `v0.1.0`. Only `build`/`ci`/`chore`/`test` since the last tag → nothing to release (exit 3).

The script reads headers, not diffs. Override with `--bump` when the header lies:

- a `fix`/`refactor` that renames or removes a public flag, endpoint, exported symbol, config key → breaking
- a `feat` that only touches internals nobody outside the repo calls → `patch`
- a non-conventional commit that adds or fixes user-facing behaviour → whatever it actually is

Say which commit forced the override and why. Never bump down past a real breaking change.

## Anatomy

```
<summary line>

## Breaking changes
- <scope>: <what broke> — <what to do instead>

## Features
- <scope>: <subject> (#PR)

## Fixes
- <scope>: <subject> (#PR)

## Performance
- ...

## Other
- ...

Full diff: https://github.com/<owner>/<repo>/compare/vA...vB
```

Sections in this order, only the non-empty ones, exactly these headings. `build`/`ci`/`chore`/`test` don't appear at all — release notes are for whoever upgrades, not for whoever maintains the pipeline.

## Summary line

One sentence. The single change a reader most needs to know about. Max 120 characters. In the language of the project's commits.

| Bad | Why | Good |
|-----|-----|------|
| `This release brings several improvements to the CLI.` | filler opener, says nothing | `` `as-skill uninstall` now removes hook registrations from `settings.json`. `` |
| `We're excited to ship v2.0!` | sells, no fact | `Drop the v1 API; all endpoints now live under /v2.` |
| `Bug fixes and performance improvements.` | the classic empty line | `Fix token refresh race that logged users out under load.` |
| `New dry-run flag, faster install, and many fixes.` | three changes in one line | `` Add `--dry-run` to preview an install without writing files. `` |

Picking the change: breaking change first if there is one; otherwise the `feat` with the widest reach; otherwise the most visible `fix`. A docs-only release says what the docs now cover.

## Bullets

Start from the commit subjects the script emits. Rewrite one only when a reader outside the repo can't parse it — e.g. an internal name with no context. Rules:

- one bullet = one commit; never invent a bullet without a commit behind it
- keep the `(#PR)` the script extracted
- a breaking bullet ends with the migration in one clause: `— use X instead`
- drop an `Other` bullet nobody upgrading cares about (a typo fix in a code comment); keep docs changes that cover new behaviour
- no bold lead-ins (`**New:**`), no nested bullets, no prose paragraphs between sections

## Slop

The linter blocks these as errors — they sell, inflate, or fill space:

- excitement and self-congratulation: `excited`, `thrilled`, `happy to announce`, `proud to`, `major milestone`
- marketing adjectives: `seamless`, `cutting-edge`, `game-changing`, `revolutionary`, `best-in-class`, `blazing fast`
- AI-flavoured verbs: `elevate`, `empower`, `unlock`, `supercharge`, `leverage`, `delve`
- empty catch-alls: `bug fixes and improvements`, `various fixes`, `minor tweaks`, `and much more`, `under the hood`, `a host of`
- `This release brings…` / `In this version…` as the opener
- emoji (including in headings), `!`, AI attribution (`Generated with Claude Code`, `Co-Authored-By`)
- Russian equivalents: `рады представить`, `долгожданный`, `бесшовный`, `различные улучшения`, `и многое другое`

Warnings — usually a vague word where a concrete one belongs: `robust`, `powerful`, `comprehensive`, `significantly`, `enhanced`, `streamlined`, `various`, `we`/`our`. Fix it or keep it with a reason (e.g. `various` inside a quoted flag name).

Text inside backticks is not checked — `unlock()` as a function name is fine.

## Full example

Bad:

```
🎉 We're thrilled to announce v2.3.0! This release brings exciting new features and various improvements.

## ✨ What's New
- Various bug fixes and performance improvements
- Added a powerful new dry-run mode

## 🐛 Bug Fixes

🤖 Generated with Claude Code
```

Good:

```
Add `--dry-run` to preview an install without writing files.

## Features
- cli: add `--dry-run` flag (#42)

## Fixes
- auth: fix race in token refresh (#41)

Full diff: https://github.com/o/r/compare/v2.2.0...v2.3.0
```
