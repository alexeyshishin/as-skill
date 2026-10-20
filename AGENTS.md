# AGENTS.md

This file is the entry point for any AI agent working with the skills in this repository. Read it in full before your first action.

## What this repository is

A multi-domain collection of rules, skills, and agents for Claude Code. Each domain is a self-contained vertical (Obsidian PKB, Git workflow, DevOps/Kubernetes, content, a code dev-loop harness). The layout is uniform; the working contract is shared across domains.

## Always-on skill

Always run with the `caveman` core skill (`core/skills/caveman/`) active — ultra-compressed output, full technical accuracy kept. Don't wait for the user to ask for brevity; it's the default in this repo.

## Architecture

```
claude-harness/
└── domains/
    ├── <domain-name>/
    │   ├── manifest.yaml      ← name, description, requires_env, targets
    │   ├── rules/             ← durable domain rules (optional — not every domain has one)
    │   ├── skills/<name>/     ← atomic operations: SKILL.md + README.md
    │   ├── agents/            ← orchestrators
    │   └── hooks/             ← optional; only `code` has one today
    ├── code/
    ├── content/
    ├── devops/
    ├── git/
    └── obsidian/
```

Each `domains/<name>/manifest.yaml` declares:
- `name`, `description` — what the domain is
- `requires_env` — environment variables the domain won't deploy without (e.g. `OBSIDIAN_VAULT` for `obsidian`)
- `targets` — where rules/skills/agents land after install

Dependencies run top-down: an agent knows about a skill, a skill knows about a rule. Not the reverse — **except in `code`** (see below).

**Exception — `code`:** the `/plan`, `/build`, `/review`, `/debug` skills are orchestrators that spawn subagents (`code-planner`, `code-skeptic`, `code-reviewer`, `code-debugger`) and merge their output — skill → agent, inverted from every other domain. `code` also has no `rules/` directory; instead its skills/agents read `.memory-bank/index.md` and `swarm-report/<slug>-*.md` inside whichever project they're running in.

## Current domains

| Domain | requires_env | What it does |
|--------|--------------|--------------|
| `code` | — | Dev-loop harness — plan → build → review → debug, installable into any target project |
| `content` | — | Content — Telegram posts, articles, technical documentation, a content factory, Russian-text humanizing |
| `devops` | — | Kubernetes, Helm, GitLab CI, ArgoCD — advisory (Consilium) and executing agents |
| `git` | — | Git workflow — Conventional Commits, PR descriptions |
| `obsidian` | `OBSIDIAN_VAULT` | Obsidian PKB on PARA — ingest, inbox triage, note critique, splitting, hub untangling |

Details live in each `domains/<name>/manifest.yaml` and the files inside it.

## Where things deploy

The installer is `tools/cli.go`, CLI name `as-skill` (see `tools/README.md`). It
installs into a target *project's* `.claude/` — not the user's global
`~/.claude/` — per each manifest's declared `targets`.

One verb, two modes: `as-skill install ...` defaults to **symlink** — edits under
`domains/`/`core/skills/` show up in the target immediately, no reinstall; this is
how this repo's own root `.claude/` stays live (`rm -rf .claude && ./as-skill
install domains code git --project .`). `as-skill install ... --copy` instead
writes a static snapshot, independent of this checkout's lifetime — for sharing,
or for projects that shouldn't depend on a harness checkout sticking around.
`as-skill uninstall ...` mirrors `install`'s positional shapes and removes only
what a prior `install`/`install --copy` placed (symlink or files, matching the
mode it was recorded under), never foreign content. Both modes write to the same
per-project `<project>/.claude/skills-lock.json`, so `status`/`doctor` can tell
owned-vs-foreign and mode apart even when a project mixes the two across runs.

- `${CLAUDE_HOME}` in every manifest resolves to `<project>/.claude/`, where
  `<project>` is whatever `--project` you pass `as-skill` (default: the
  current directory)
- rules, skills, agents for every domain → `${CLAUDE_HOME}/{rules,skills,agents}/`
- `core/skills/*` (domain-independent — `caveman`, `memory-bank`, `memory-bank-defrag`, `swarm-report`) → `${CLAUDE_HOME}/skills/` always; no manifest, no gating

`obsidian` is the one domain gated on `requires_env: OBSIDIAN_VAULT` — an
opt-in check (install it only for users who actually keep a vault), not a
path source. Its skills still reference their rules by a path relative to
the vault root (`.claude/rules/<file>.md`), so install it with `--project`
pointed at the vault root — otherwise the project's `.claude/` and the vault
are different directories and that reference won't resolve.

If a domain's `requires_env` isn't satisfied, an explicit `as-skill install
domain`/`domains`/`skill` request fails outright; `as-skill install all`
instead skips it with a warning and installs the rest. Right now only
`obsidian` declares `requires_env`; no domain currently declares a
`requires_bin` gate.

`code` is a separate case: it also has its own `/code-setup` skill that installs the
`code` domain (agents, skills, the `test-gate` hook) plus core skills into a *target
project* directory, run from a clone of this repo. Under the hood it's the same
`as-skill install domain code --project <target> --copy --with-core` as above —
`code-setup` just always passes `--copy` explicitly, on purpose: the target is an
unrelated project elsewhere on the user's machine, and coupling it to this
checkout's lifetime via a symlink (the installer's default) would be wrong. See
`domains/code/skills/code-setup/SKILL.md` for the full recipe.

## What to do before your first action

1. **Identify which domain the task belongs to.** Skill/rule names carry the domain prefix (`obsidian-ingest`, `git-conventional-commits`, `content-tg-post`, `code-plan`); `devops` agent files do too (`devops-sre.md`), though a few agents' internal `name:` is shorter (`sre`, `architect`, `ci-agent`) — go by the file path when unsure.
2. **Read your domain's rules.** Skills and agents name which rules they need at the top of the file. `code` and `devops` currently have none — for `code`, read `.memory-bank/index.md` and the relevant `swarm-report/` file instead.
3. **If the task spans multiple domains** (e.g. write up an incident and publish it as a Telegram post), chain skills from each domain in sequence, respecting each one's rules.
4. **If the task fits neither a skill nor an agent**, act manually on the general principles below. Afterward, ask the user whether it's worth turning into a skill.

## General principles (apply across all domains)

### Working contract with the user

**Plan → confirmation → action → report.** For a batch of changes or anything hard to reverse (push, tag, delete, publish, a direct change to production) — show the plan first, wait for confirmation, then act.

### Language and tone

Not a single blanket rule — it's domain-dependent:
- **`content`, `obsidian`**: output is in Russian, technical terms stay in English (`SRE`, `Kubernetes`, `goroutine`, `Pod`); informal "you" or impersonal phrasing; personal, authorial voice — not Wikipedia, not corporate copy. See `content-voice.md` / `content-style.md`.
- **`code`, `git`, `devops`**: output is in English, terse and technical (commit subjects, PR descriptions, YAML, agent findings as `path:line — SEVERITY: problem. fix.`). See each domain's own rule/agent files.

### Don't, in any domain

- **Don't delete notes / commits / resources** without an explicit request
- **Don't invent numbers, names, commands** — ask if the data is missing
- **Don't write "human error" as a root cause** — it's always a systemic problem (an SRE principle, applies everywhere)
- **Don't publish / push / tag / deploy** without an explicit go-ahead
- **In `devops`, don't touch production outside GitOps** — prod changes land only through ArgoCD, never a direct `kubectl apply`

### When a domain overlaps with another

Examples:
- **Incident root-cause (devops's `sre`/`diagnostics` agents)** → write it up and ingest it into the **knowledge base (obsidian)** via `obsidian-ingest`.
- **A PR or release (`git-flow`)** → can become a **Telegram post (`content-tg-post`)** announcing it.
- **A tutorial (`content-documentation`)** → may embed runbook logic reviewed by devops's `sre` Consilium agent.
- **An article (`content-article-draft`)** → can expand into a full package via **`content-factory`**.

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

**Agents:**
- `git-flow` — orchestrates branch → commits → PR → merge, calling the two skills above at the right moments

### domain: content

**Rules:** `content-voice` (authorial voice, Russian + English terms) · `content-formatting` (per-platform formatting: Telegram, articles, tutorials)

**Skills:**
- `content-tg-post` — idea/raw text → Telegram post, 500–1500 characters, thesis-first
- `content-article-draft` — idea/notes → article draft (Habr / Medium / personal blog)
- `content-documentation` — technical-tutorial skeleton: prerequisites, numbered steps, troubleshooting, cleanup
- `content-humanizer` — de-AI-ifies Russian-language text (Russian only)
- `content-factory` (trigger `/content-zavod`) — one source → 6-file content package (article, threads, Reels scripts, posts, carousels, plan)

**Agents:**
- `content-editor` — content pipeline from idea to publication; picks the format and calls the matching skill

### domain: devops

No `rules/` or `skills/` yet — agents only. Two roles:
- **Consilium** (`architect`, `security`, `sre`, `diagnostics`) — advisory, read-only, never touches code or configs
- **Executing** (`ci-agent`, `helm-agent`, `k8s-agent`) — writes pipelines, charts, and manifests within its own file scope

GitOps rule: production changes go through ArgoCD only.

**Agents:**
- `devops-architecture` (`architect`) — platform/service topology, K8s workload design, IaC structure
- `devops-security` (`security`) — RBAC, network policies, secrets, container/supply-chain security
- `devops-sre` (`sre`) — SLOs/error budgets, observability, alerting, runbooks, postmortems
- `devops-diagnostic` (`diagnostics`) — root-cause analysis: crashes, OOMKills, network, ArgoCD sync, pipeline failures
- `devops-ci` (`ci-agent`) — GitLab CI/CD pipelines and shared templates
- `devops-helm` (`helm-agent`) — Helm charts (`Chart.yaml`, `templates/`, `values.yaml`)
- `devops-k8s` (`k8s-agent`) — Kubernetes manifests, Kustomize overlays, ArgoCD ApplicationSets

### domain: code

A dev-loop harness meant to be installed into other projects (see `/code-setup` above), not a set of personal-productivity skills. No `rules/`; context comes from `.memory-bank/index.md` and `swarm-report/`.

**Skills:**
- `code-plan` (`/plan`) — spawns `code-planner` + `code-skeptic`, writes a plan to `swarm-report/`; use before `/build`
- `code-build` (`/build`) — implements an approved plan, routes tasks by file scope, runs tests, writes a build report
- `code-review` (`/review`) — spawns `code-reviewer`, read-only, ship/rework with severity-tagged findings
- `code-debug` (`/debug`) — spawns `code-debugger`, reproduces → ladders hypotheses → minimal fix
- `code-setup` — installs this harness (agents, skills, hooks, settings, `AGENTS.md`) into a target project

**Agents:** `code-planner`, `code-skeptic`, `code-reviewer`, `code-debugger` — all TERSE-output subagents, spawned only by the skills above, never invoked directly.

**Hooks:** `test-gate.sh` — Stop hook; blocks declaring a task done if code files were edited but no test command ran this session (fail-open, capped at 2 blocks per commit).

## Core skills (no domain)

In `core/skills/` — not tied to any domain, always installed, no `manifest.yaml`:

- `caveman` — ultra-compressed communication mode (lite/full/ultra, plus wenyan variants)
- `memory-bank` — maintains a lightweight project encyclopedia in `.memory-bank/`
- `memory-bank-defrag` — folds accumulated `.memory-bank/` patches into clean current-state docs
- `swarm-report` — placeholder; content not written yet
