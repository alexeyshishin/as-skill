---
name: docs-readme
description: >
  Writes or updates a project's README.md (English, the source of truth) and
  README_RUS.md (its full Russian translation) from the actual code: detects the
  project type, picks the sections, takes every command from the project's own
  manifests, scripts and CI, keeps EN and RU in sync, lints both and clears AI
  slop through docs-humanizer. Use when the
  user wants to "write a README", "update the README", "make a README for this
  repo", "translate the README into Russian", "sync README_RUS", "напиши
  README", "обнови README", "переведи README на русский". Not for tutorials or
  articles (content-documentation, content-article-draft); a review of an
  existing README is docs-readme-reviewer.
---

# docs-readme — README.md + README_RUS.md from the code

Goal: a README a stranger can act on in a minute — what it is, how to install, how to use — in English, with a Russian translation that never lags behind.

Before starting, read `references/rules/docs-conventions.md` — sections README and EN ↔ RU.

Assets:
- [`references/readme-skeletons.md`](references/readme-skeletons.md) — project-type detection, sections per type, EN ↔ RU heading map, a skeleton pair.
- [`scripts/check-readme.py`](scripts/check-readme.py) — mechanical checker for one README or an EN + RU pair (Step 6).
- the `docs-humanizer` skill, README mode — AI-slop scan and spot fixes for both files (Step 6.5).

## Step 1. Pick the mode

| situation | mode |
|-----------|------|
| no `README.md` | **new** — write EN, then RU |
| `README.md` exists, the user wants it updated or rewritten | **update** — fix what is stale or missing, keep the rest |
| EN changed, RU missing or behind | **sync** — translate only the EN parts RU lacks |
| only RU asked for, no EN | write EN first: EN is the source |

For **update** and **sync** read both files in full first, and `git log -3 --format='%h %cs %s' -- README.md README_RUS.md` to see which one moved last.

## Step 2. Gather facts from the code

Everything in the README must trace back to a file. Read:

- manifests: `package.json`, `pyproject.toml`, `setup.cfg`, `go.mod`, `Cargo.toml`, `pom.xml`, `build.gradle*`, `*.gemspec`, `composer.json`, `Dockerfile`, `docker-compose*.yml`, `Chart.yaml` — name, description, runtime versions, entry points
- commands: `Makefile`, `justfile`, `Taskfile.yml`, `scripts` in `package.json`, CI (`.github/workflows/*`, `.gitlab-ci.yml`) — the real install, build, test, run commands
- interface: CLI flag definitions, exported API, HTTP routes, environment variables the code reads (`os.Getenv`, `process.env`, `os.environ`)
- existing docs: `docs/`, `CONTRIBUTING.md`, `CHANGELOG.md`, `*_RUS.md` — link them, don't inline them
- license: the `LICENSE*` / `COPYING*` file — name its SPDX id; no license file → say so and point to `docs-license`, never write a license name by guess
- remote: `git remote get-url origin` — for badge and link URLs

Run a command only when it is read-only and cheap (`--help`, `--version`, a dry run) — to confirm a flag or capture real output. Never run installs into the system, deploys, or anything that writes outside the repo; such commands are copied verbatim from their source file.

## Step 3. Ask what the code can't tell

One message, only the gaps: who the project is for and why it exists (if no description anywhere), whether there is a screenshot to show, which install path is preferred when there are several. Don't ask what a file already answers. Don't fill a gap with an invented fact or a `TODO`.

## Step 4. Write README.md

Detect the type and take its section set from `references/readme-skeletons.md`; order per `docs-conventions`. Then:

- opening line first: what + for whom, searchable nouns up front
- Install: prerequisites with versions from the manifests, then one command
- Usage: the smallest example with real output
- flags, options, env vars — tables; the why — prose
- badges only for what exists (no CI badge without CI), at most 3
- anything long (architecture, benchmarks, FAQ) — link to a doc, or leave out

In **update** mode keep the author's structure and voice; change only what is wrong, stale or missing.

## Step 5. Write README_RUS.md

Translate the final EN per `docs-conventions` → EN ↔ RU: same headings, tables, images, code; comments and `<placeholders>` inside code may be translated; terms stay English; anchors point to RU headings; links go to `_RUS` versions when they exist; language switcher in both files.

In **sync** mode translate only the EN parts RU lacks or that changed; don't re-translate untouched paragraphs.

## Step 6. Lint

Before overwriting a README that has uncommitted changes or isn't tracked by git — ask. Then write the files and run:

```bash
python3 scripts/check-readme.py README.md
```

It pairs `README_RUS.md` automatically (`--ru PATH` for another name, `--no-ru` for EN alone). Exit 0 — proceed. Exit 2 — fix and re-run; don't show a draft that fails its own convention. Report a remaining warning only when you deliberately disagree with it, and say why (e.g. `missing-section` Usage for a monorepo root).

The checker sees form, not truth: whether the commands work and the opening line is accurate stays your judgment.

## Step 6.5. AI slop

Run `docs-humanizer` in README mode on both files: scan, fix only the flagged passages in a technical register, re-scan, gate at 1.5 tells per 100 words per file. Then re-run `check-readme.py` — a slop fix must not break EN/RU parity.

A draft that fails the gate is not shown; a finding kept on purpose is listed with its reason.

## Step 7. Show and ask

New files — show them. Updates — show `git diff -- README.md README_RUS.md`. Include the slop density per file. Ask:
1. Accept
2. Adjust (what)
3. Add (screenshot, configuration table, more examples)

## What not to do

- don't invent commands, flags, versions, features, numbers or badges — every fact traces to a file or to the user
- don't add sections for completeness; an empty or generic section is worse than none
- don't translate code, identifiers, flags or product names
- don't let RU drift: never finish with EN changed and RU not
- don't touch `LICENSE` — that's `docs-license`
- no "generated by" lines, no AI attribution, no emoji beyond the language switcher flag
- don't commit or push
