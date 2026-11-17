# CLAUDE.md

This file is the entry point for any AI agent working with the skills in this repository. Read it in full before your first action.

## What this repository is

A multi-domain collection of rules, skills, and agents for Claude Code. Each domain is a self-contained vertical (Obsidian PKB, Git workflow, DevOps/Kubernetes, content, a code dev-loop harness). The layout is uniform; the working contract is shared across domains.

## Always-on skill

Always run with the `caveman` core skill (`domains/core/skills/caveman/`) active — ultra-compressed output, full technical accuracy kept. Don't wait for the user to ask for brevity; it's the default in this repo.

## Architecture

```
as-skill/
└── domains/
    ├── <domain-name>/
    │   ├── manifest.yaml      ← name, description, requires_env, targets
    │   ├── rules/             ← durable domain rules (optional — not every domain has one)
    │   ├── skills/<name>/     ← atomic operations: SKILL.md + README.md
    │   ├── agents/            ← orchestrators
    │   └── hooks/             ← optional; only `code` and `core` have one today
    ├── code/
    ├── content/
    ├── core/
    ├── devops/
    ├── docs/
    ├── git/
    ├── obsidian/
    ├── resume/
    └── university/
```

Each `domains/<name>/manifest.yaml` declares:
- `name`, `description` — what the domain is
- `requires_env` — environment variables the domain won't deploy without (e.g. `OBSIDIAN_VAULT` for `obsidian`)
- `targets` — where rules/skills/agents land after install

Dependencies run top-down: an agent knows about a skill, a skill knows about a rule. Not the reverse — **except in `code`** (see below).

**Exception — `code`:** the `/code-plan`, `/code-build`, `/code-review`, `/code-debug` skills are orchestrators that spawn subagents (`code-planner`, `code-skeptic`, `code-reviewer`, `code-debugger`) and merge their output — skill → agent, inverted from every other domain. `code` also has no `rules/` directory; instead its skills/agents read `.memory-bank/index.md` and `swarm-report/<slug>-*.md` inside whichever project they're running in.

## Current domains

| Domain | requires_env | What it does |
|--------|--------------|--------------|
| `code` | — | Dev-loop harness — plan → build → review → debug, installable into any target project |
| `content` | — | Content — a workspace with status folders for posts, Telegram posts, articles, post images, Russian-text humanizing |
| `core` | — | Domain-independent utility skills, always installable: ultra-compressed output mode, project memory bank, memory bank defrag, fact-checked web search |
| `devops` | — | Kubernetes, Helm, GitLab CI, ArgoCD — advisory (Consilium) and executing agents |
| `docs` | — | README.md + README_RUS.md and licenses — writer and reviewer agents, skills with linters, third-party attribution, AI-slop check of both READMEs, Russian-text humanizing; C4 diagrams stubbed |
| `git` | — | Git workflow — Conventional Commits, PR descriptions, semver releases |
| `obsidian` | `OBSIDIAN_VAULT` | Obsidian PKB on PARA — ingest, inbox triage, note critique, splitting, hub untangling |
| `resume` | — | Resume review and job search — ATS optimization, bullets, tailoring, cover letters, interview prep (22 skills adapted from Paramchoudhary/ResumeSkills, MIT) |
| `university` | — | Graduation thesis (ВКР) — writer and reviewer agents, Markdown → GOST 7.32 `.docx` skill |

Details live in each `domains/<name>/manifest.yaml` and the files inside it.

## Where things deploy

The installer entrypoint is `tools/main.go` (dispatcher and `spf13/pflag`-based flag parsing in `tools/internal/cli`; engine in `tools/internal/{harness,install,render,hookcfg,doctor,lint,platform,lockfile,fsys,frontmatter,iostreams,prompt}`),
CLI name `as-skill` (see `tools/README.md`). It
installs into a target *project's* `.claude/` — not the user's global
`~/.claude/` — per each manifest's declared `targets`.

One verb, two modes: `as-skill add ...` defaults to **symlink** — edits under
`domains/` show up in the target immediately, no reinstall; this is
how this repo's own root `.claude/` stays live (`rm -rf .claude && ./as-skill
add code git core -p . -y`). `as-skill add ... --copy` instead
writes a static snapshot, independent of this checkout's lifetime — for sharing,
or for projects that shouldn't depend on a harness checkout sticking around.
In a terminal, bare `as-skill add` (or `remove`) opens an interactive dialog (project → domains → customize skills → platforms → mode → conflicts → hooks → summary + confirm); names/flags pre-answer their question and `-y` takes all defaults with no prompts — the dialog needs stdin and stderr both terminals (`add git | cat` still opens it); `</dev/null`, `2>file`, `CI`, `TERM=dumb` are non-interactive and still need names and `-y` (or `--dry-run`). `as-skill remove ...` is lockfile-driven and removes only
what a prior `add`/`add --copy` placed (symlink or files, matching the
mode it was recorded under) plus the directories that leaves empty, never foreign content. Uninstalling a domain's `hooks`
also drops their registrations from the project's `.claude/settings.json` (only
entries pointing at the hook files it just removed; `settings.local.json` is
never touched). Both modes write to the same
per-project `<project>/.as-skill/lock.json` (platform-neutral; a legacy
`.claude/skills-lock.json` moves there on the next `add`/`remove`), so `status`/`doctor` can tell
owned-vs-foreign and mode apart even when a project mixes the two across runs.

- `${CLAUDE_HOME}` in every manifest resolves to `<project>/.claude/`, where
  `<project>` is whatever `--project` you pass `as-skill` (default: the
  current directory)
- rules, skills, agents for every domain → `${CLAUDE_HOME}/{rules,skills,agents}/`
- `domains/core/skills/*` (domain-independent — `caveman`, `memory-bank`, `memory-bank-defrag`, `advanced-search`) → `${CLAUDE_HOME}/skills/`, installed the same way as any other domain's skills

`obsidian` is the one domain gated on `requires_env: OBSIDIAN_VAULT` — an
opt-in check (install it only for users who actually keep a vault), not a
path source. Its skills still reference their rules by a path relative to
the vault root (`.claude/rules/<file>.md`), so install it with `--project`
pointed at the vault root — otherwise the project's `.claude/` and the vault
are different directories and that reference won't resolve.

If a domain's `requires_env` isn't satisfied, an explicit `as-skill add
<name>` request fails outright; `as-skill add all`
instead skips it with a warning and installs the rest. Right now only
`obsidian` declares `requires_env`; no domain currently declares a
`requires_bin` gate.

Other verbs: `as-skill list [domains|skills|platforms]` (marks installed), `as-skill doctor` (status table + checks). Legacy `install|uninstall|status` and `--platform`/`--force`/`--harness-root` still work, hidden from help (the write verbs `install`/`uninstall`/`hooks enable|disable` refuse a `$HOME` project like `add`/`remove`; `status` is unrestricted). Harness checkout is auto-resolved (`--harness-root` → cwd walk-up → binary's real path), so a binary built here works from any project.

`code` is a separate case: it's a portable dev-loop harness meant to be installed
into *other* projects, not just this repo's own `.claude/`. From a clone of this
repo, run `as-skill add code core -p <target> --copy -y` to
place it (agents, skills, hooks, plus core skills) into a target
project directory. `--copy` here is deliberate, not the default: the target is an
unrelated project elsewhere on the user's machine, and coupling it to this
checkout's lifetime via a symlink (the installer's default) would be wrong.

`-a/--agent claude|codex|cursor|opencode[,…]|all` picks target hosts (default: autodetected from existing `.codex/ .cursor/ .opencode/ .claude/` dirs and the lockfile, else `claude`; `add -y` fails before writing if a detected link root isn't git-ignored — use `-a claude` or `--copy`). Skills are placed once per location by set cover; agents are symlinked on claude and rendered natively for the others; rules go via `.agents/rules` plus an `AGENTS.md` block, or a Cursor `.mdc` stub. Hook adapters for codex/cursor are Phase 3; opencode hooks are unsupported. claude+codex together warns about duplicate skills. Hook platform support: `test-gate`, `prettier-format`, `memory-bank-load` — claude/codex/cursor; `memory-bank-logger` — claude only; opencode none. Non-claude adapters (hooks, rendered agents, rules) are doc-derived; claude is verified, codex skills and agents are verified on a real Codex CLI (2026-11-16, link install), codex rules/hooks and all of cursor/opencode are unverified.

## What to do before your first action

1. **Identify which domain the task belongs to.** Skill/rule names carry the domain prefix (`obsidian-ingest`, `git-conventional-commits`, `content-tg-post`, `code-plan`); `devops` agent files do too (`devops-sre.md`), each agent's `name:` equals its filename stem.
2. **Read your domain's rules.** Skills and agents name which rules they need at the top of the file. `code` and `devops` currently have none — for `code`, read `.memory-bank/index.md` and the relevant `swarm-report/` file instead.
3. **If the task spans multiple domains** (e.g. write up an incident and publish it as a Telegram post), chain skills from each domain in sequence, respecting each one's rules.
4. **If the task fits neither a skill nor an agent**, act manually on the general principles below. Afterward, ask the user whether it's worth turning into a skill.

## General principles (apply across all domains)

### Working contract with the user

**Plan → confirmation → action → report.** For a batch of changes or anything hard to reverse (push, tag, delete, publish, a direct change to production) — show the plan first, wait for confirmation, then act.

### Language and tone

Not a single blanket rule — it's domain-dependent:
- **`content`, `obsidian`**: output is in Russian, technical terms stay in English (`SRE`, `Kubernetes`, `goroutine`, `Pod`); informal "you" or impersonal phrasing; personal, authorial voice — not Wikipedia, not corporate copy. See `content-voice.md` / `content-style.md`.
- **`university`**: thesis text and review findings are in Russian, technical terms stay in English; formal impersonal academic register, not an authorial voice. See `university-gost.md`.
- **`code`, `git`, `devops`**: output is in English, terse and technical (commit subjects, PR descriptions, YAML, agent findings as `path:line — SEVERITY: problem. fix.`). See each domain's own rule/agent files.
- **`docs`**: `README.md` in English, `README_RUS.md` its Russian translation with technical terms in English; review findings as `path:line — SEVERITY: problem. fix.`. See `docs-conventions.md`.

### Don't, in any domain

- **Don't delete notes / commits / resources** without an explicit request
- **Don't invent numbers, names, commands** — ask if the data is missing
- **Don't write "human error" as a root cause** — it's always a systemic problem (an SRE principle, applies everywhere)
- **Don't publish / push / tag / deploy** without an explicit go-ahead
- **In `devops`, don't touch production outside GitOps** — prod changes land only through ArgoCD, never a direct `kubectl apply`

### When a domain overlaps with another

Examples:
- **Incident root-cause (devops's `devops-sre`/`devops-diagnostic` agents)** → write it up and ingest it into the **knowledge base (obsidian)** via `obsidian-ingest`.
- **A PR or release (`git-flow`)** → can become a **Telegram post (`content-tg-post`)** announcing it.

In these cases follow both domains' rules — they complement, not contradict, each other.

## Component map

### domain: obsidian

**Rules:** `vault-struct` (PARA layout, vault root) · `tags` · `note-types-frontmatter` · `file-naming` · `content-style` (voice, language, links) · `knowledge-structures` (MOC, atomic notes) · `template-usage` · `workflows` (plan → confirm → act → report)

**Skills:**
- `obsidian-inbox-review` — inspect `00. Входящие/`, report only, no changes
- `obsidian-refactor-inbox` — full inbox cycle: typing → frontmatter → links → move into PARA
- `obsidian-enrich-note` — targeted frontmatter update (`aliases`/`up`/`down`/`other`) without moving the note
- `obsidian-ingest` — external source → literature note + atomic insight notes
- `obsidian-refactor-lecture` — lecture notes → table-of-contents note + extracted concept notes (up to 3 sets/run)
- `obsidian-split-note` — one large note → several smaller ones, links preserved
- `obsidian-untangle-knot` — overloaded hub (high in-degree) → sub-MOCs + re-linked incoming notes
- `obsidian-note-critic` — critique one note: similar/opposing notes, contradictions, argumentation gaps

**Agents:**
- `obsidian-inbox-cleaner` — full triage cycle for `00. Входящие/`
- `obsidian-source-ingester` — external source (article, book, video, transcript) → connected notes
- `obsidian-knowledge-cartographer` — graph health: hub untangling, note splitting
- `obsidian-note-doctor` — one note: enrichment, critique, or both

### domain: git

**Rules:** `git-conventions` — Conventional Commits format, branch names, PR structure, semver/tags

**Skills:**
- `git-conventional-commits` — Conventional Commits message from the current (staged, else unstaged) diff
- `git-pr-description` — PR description from the branch diff against the base branch
- `git-release-tag` — next semver tag from commits since the last one, slop-linted release notes, annotated tag + GitHub release via `gh`

**Agents:**
- `git-flow` — orchestrates branch → commits → PR → merge, calling the three skills above at the right moments

### domain: docs

**Rules:** `docs-conventions` — README section order and checklist, EN ↔ RU sync (EN is the source), license text verbatim only, license file layout, third-party attribution

**Skills:**
- `docs-readme` — README.md + README_RUS.md from the code: project type → sections, commands only from manifests/scripts/CI, EN → RU sync; lints via `scripts/check-readme.py`, clears AI slop via `docs-humanizer`
- `docs-license` — license questionnaire → SPDX, canonical text copied from `references/licenses/` (15 bundled, else `gh api /licenses` / SPDX), placeholders only; third-party attribution + compatibility table; checks via `scripts/check-license.py`
- `docs-humanizer` — README mode: AI-slop scan of README.md + README_RUS.md (EN + RU catalogs, `scripts/scan-tells.py`, gate 1.5 per 100 words), spot fixes in a technical register; full mode: de-AI-ifies Russian-language text (based on `content-humanizer`, which stays in `content` for posts and articles)
- `docs-c4` — stub, not implemented (C4 model diagrams)

**Agents:**
- `docs-readme-writer` — write/update/sync via `docs-readme`; fix mode applies approved reviewer findings, EN and RU together
- `docs-readme-reviewer` — read-only; runs the checker, then checks README against the code, EN ↔ RU meaning and AI slop; ship/rework with severity-tagged findings
- `docs-license-writer` — choose/write/attribute via `docs-license`; fix mode applies approved reviewer findings, recopies text instead of patching
- `docs-license-reviewer` — read-only; runs the checker, audits borrowed code, attribution, compatibility; ship/rework with severity-tagged findings
- `docs-c4-writer`, `docs-c4-reviewer` — stubs, not implemented

### domain: university

**Rules:** `university-gost` — GOST 7.32 structure, page layout, headings, figures/tables/formulas, sources, appendices, writing rules (no fabricated data or sources)

**Skills:**
- `university-md-to-docx` — Markdown thesis → `.docx` per GOST 7.32 (title page, TOC field, page numbers, captions, appendices) via `scripts/md2docx.py`, then a mechanical check via `scripts/check-docx.py`; needs `python3` + `python-docx` at run time

**Agents:**
- `university-writer` — outline → chapter-by-chapter Markdown draft → converts via the skill; missing data stays a `[ДАННЫЕ: …]` placeholder
- `university-reviewer` — read-only; runs the `.docx` check, then reviews goal–tasks–conclusions, citations, register; reports ship/rework with severity-tagged findings

### domain: resume

No `rules/` or `agents/` — skills only, prefixed `resume-`. Adapted from [Paramchoudhary/ResumeSkills](https://github.com/Paramchoudhary/ResumeSkills) (MIT, `LICENSE-ResumeSkills`). English-language resumes.

**Skills:** review/optimize — `resume-ats-optimizer`, `resume-bullet-writer`, `resume-quantifier`, `resume-formatter`, `resume-section-builder`, `resume-tech-resume-optimizer`, `resume-executive-resume-writer`, `resume-academic-cv-builder`, `resume-creative-portfolio-resume`, `resume-career-changer-translator` · job match — `resume-tailor`, `resume-job-description-analyzer`, `resume-version-manager`, `resume-application-form-filler` · outreach/prep — `resume-cover-letter-generator`, `resume-cold-email-writer`, `resume-linkedin-profile-optimizer`, `resume-interview-prep-generator`, `resume-salary-negotiation-prep`, `resume-offer-comparison-analyzer`, `resume-reference-list-builder`, `resume-portfolio-case-study-writer`

### domain: content

**Rules:** `content-voice` (authorial voice, Russian + English terms) · `content-formatting` (per-platform formatting: Telegram, articles) · `content-workspace` (workspace layout: `CONTENT.md` passport, `voice/`, status = folder `ideas/` → `drafts/` → `ready/` → `published/<YYYY>/` → `archive/`, `images/<slug>/`, slug naming, post frontmatter)

**Skills:**
- `content-init` — creates a content workspace per `content-workspace`: `CONTENT.md` channel passport, `voice/` samples, status folders, `images/`; idempotent, moves legacy drafts/images only on the author's say-so
- `content-tg-post` — idea/raw text → Telegram post, 500–1500 characters, thesis-first; in a workspace saves to `drafts/`
- `content-article-draft` — idea/notes → article draft (Habr / Medium / personal blog); in a workspace saves to `drafts/`
- `content-humanizer` — de-AI-ifies Russian-language text (Russian only), takes voice samples from the workspace's `voice/`; README/docs slop check is `docs-humanizer`
- `content-image` — heading/text → 1920×1920 PNG in the author's Figma-derived style (title card with an always-English lowercase heading, always 3 variants whose emoji never repeat another post's; text page in the author's language) via `scripts/render.py` + headless Chrome; tokens in `references/tokens.css`, Roboto bundled; Figma is not needed at run time; in a workspace saves to `images/<slug>/`

**Agents:**
- `content-editor` — content pipeline from idea to publication; picks the format, calls the matching skill, and `content-image` when a picture is needed; in a workspace moves approved posts to `ready/` and published ones to `published/<YYYY>/`

### domain: devops

No `rules/` or `skills/` yet — agents only. Two roles:
- **Consilium** (`devops-architecture`, `devops-security`, `devops-sre`, `devops-diagnostic`) — advisory, read-only, never touches code or configs
- **Executing** (`devops-ci`, `devops-helm`, `devops-k8s`) — writes pipelines, charts, and manifests within its own file scope

GitOps rule: production changes go through ArgoCD only.

**Agents:**
- `devops-architecture` — platform/service topology, K8s workload design, IaC structure
- `devops-security` — RBAC, network policies, secrets, container/supply-chain security
- `devops-sre` — SLOs/error budgets, observability, alerting, runbooks, postmortems
- `devops-diagnostic` — root-cause analysis: crashes, OOMKills, network, ArgoCD sync, pipeline failures
- `devops-ci` — GitLab CI/CD pipelines and shared templates
- `devops-helm` — Helm charts (`Chart.yaml`, `templates/`, `values.yaml`)
- `devops-k8s` — Kubernetes manifests, Kustomize overlays, ArgoCD ApplicationSets

### domain: code

A dev-loop harness meant to be installed into other projects (see the install command above), not a set of personal-productivity skills. No `rules/`; context comes from `.memory-bank/index.md` and `swarm-report/`.

**Skills:**
- `code-init` (`/code-init`) — generates the target project's `CLAUDE.md` (build/test/lint commands, layout, house rules, executing-agents table), registers installed hooks in the project's `.claude/settings.json`, and (with consent, if the `ast-index` binary exists) runs `scripts/setup-ast-index.sh`: `ast-index` plugin project-scope, `Bash(ast-index *)` allowed, index built
- `code-plan` (`/code-plan`) — spawns `code-planner` + `code-skeptic`, writes a plan to `swarm-report/`; use before `/code-build`
- `code-build` (`/code-build`) — implements an approved plan, routes tasks by file scope, runs tests, writes a build report
- `code-review` (`/code-review`) — spawns `code-reviewer`, read-only, ship/rework with severity-tagged findings
- `code-debug` (`/code-debug`) — spawns `code-debugger`, reproduces → ladders hypotheses → minimal fix

**Agents:** `code-planner`, `code-skeptic`, `code-reviewer`, `code-debugger` — orchestrator-side TERSE subagents, spawned only by the skills above, never invoked directly. `backend`, `frontend`, `devops` — executing agents `/code-build` routes `affected_files` to by scope (per the `## Исполнители для /code-build` table that `/code-init` writes into the target project's `CLAUDE.md`); they write code/config, not plans. All seven carry a `## Code navigation` block: `ast-index` first when `test -f "$(ast-index db-path)"` succeeds (index exists; `ast-index stats` exits 0 even without one), Grep otherwise — a soft dependency, no `requires_bin` gate; the harness ships no ast-index hooks (the plugin brings its own). (A `mobile` exec agent doesn't exist yet — add it the same way if a target project needs it.)

**Hooks:** `test-gate.sh` — Stop hook; blocks declaring a task done if code files were edited but no test command ran this session (fail-open, capped at 2 blocks per commit). `prettier-format.sh` — PostToolUse hook (`Edit|Write|MultiEdit|Bash`); runs prettier only once the test-gate condition is met (test command already ran, or project has none); no-op without a prettier config. Both source `test-gate-lib.sh`, the single definition of that condition. (`memory-bank-load.sh` lives in `core`.) Hooks are copied by the installer, then registered in the project's `.claude/settings.json` by `/code-init` via `skills/code-init/scripts/register-hooks.sh` (the `add` dialog can also register explicitly selected hooks: default none, `-y` never) — each hook declares its own event/matcher in a `# register: <Event> [<matcher>]` header line, so a new hook needs no script change. On codex/cursor use `as-skill hooks enable`; adapters are doc-derived, unverified.

### domain: core

Domain-independent utility skills — not tied to any other domain, but installed and gated the same way as any other domain (`domains/core/manifest.yaml`, `requires_env: []`). No `rules/` or `agents/`.

**Hooks:** `memory-bank-load.sh` — SessionStart hook; injects `.memory-bank/index.md` (capped at 16 KB) plus the list of the other Memory Bank files into context, no-op without `jq` or an index. `memory-bank-logger.sh` — SessionEnd hook; detached headless `claude -p` reads the session transcript and updates `.memory-bank/` via the memory-bank skill (skips short sessions, recursion-guarded via `MEMORY_BANK_LOGGER`). Registered the same way as `code` hooks (`# register:` header, `/code-init`).

**Skills:**
- `caveman` — ultra-compressed communication mode (lite/full/ultra, plus wenyan variants)
- `memory-bank` — maintains a lightweight project encyclopedia in `.memory-bank/`; first bootstrap seeds from `ast-index map`/`conventions` when ast-index is available
- `memory-bank-defrag` — actualizes `.memory-bank/` against changes since the last run (baseline in `.memory-bank/.defrag.yaml`, git optional) and compacts the whole bank: folds amendment trails, collapses finished tasks and closed questions, cuts narrative to current-state facts
- `advanced-search` (`/advanced-search`) — OSINT-style fact-checked web research: fresh authoritative sources, bold one-line verdict, tables, links to primary sources
