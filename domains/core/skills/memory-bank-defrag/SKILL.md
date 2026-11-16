---
name: memory-bank-defrag
description: >
  Defragment and re-actualize a project memory bank (`.memory-bank/` + the
  host's auto-memory, if any). Two phases every run: actualize — check what
  changed in the repo since the last defrag (baseline kept in
  `.memory-bank/.defrag.yaml`, works with or without git) and fix stale facts;
  compact — sweep the whole bank, fold amendment trails, collapse finished
  tasks and closed questions, cut narrative down to current-state facts. Then
  shows a diff for review. Use when the user says "defrag the memory bank",
  "привести memory bank в порядок", "дефрагментация памяти", "актуализируй
  memory bank", "причеши базу знаний", or runs /memory-bank-defrag.
allowed-tools: Bash, Read, Write, Edit, Glob, Grep
argument-hint: "[ref|date]  (optional: a commit or YYYY-MM-DD that overrides the baseline from .memory-bank/.defrag.yaml)"
---

# Memory Bank Defrag

Bring a project's memory bank back in sync with reality and cut it down to
current state. The memory bank is three things:

1. **`.memory-bank/`** in the repo — the project encyclopedia the `memory-bank`
   skill maintains: `index.md`, `product-overview/`, `steerings/`,
   `tech-details/`, `tasks/`. Operate on whatever files exist; don't create
   folders the project doesn't use.
2. **Auto-memory** — the host's cross-session memory (Claude Code: a
   `MEMORY.md` index plus one file per fact). Take its path from the session
   context; don't guess it. Skip silently if the context names none or the
   directory doesn't exist (cloud/headless runs, other hosts).
3. **Repo current-state docs** — files that assert *what is true now*:
   `README.md`, `AGENTS.md`/`CLAUDE.md`, design specs. Update them only when a
   change proves a statement stale. Do **not** touch raw source records that
   capture *what happened* — transcripts, chat dumps, extracted content — those
   are evidence; keep them verbatim.

Every run has two phases:

- **Actualize** — what changed in the repo since the last defrag → stale facts,
  dangling references, uncaptured decisions.
- **Compact** — the whole memory bank, regardless of what changed → same facts,
  less text.

## Non-negotiables

- **Accuracy > speed.** Memory is point-in-time and may be wrong. Before you
  assert a fact as current, **verify it against the live repo** (file exists,
  symbol not renamed, deadline not passed, stack matches code). If a memory
  claim contradicts current code, trust the code and fix the memory.
- **Compact the form, never the facts.** Every substantive fact survives: what
  is true now, where it lives, why it was decided (when the reason isn't
  obvious from the code). Narrative goes: how we got there, step-by-step logs,
  debugging history, "first we tried X". A reversal that carries
  decision-relevant context (why we moved off X) stays as one supersession
  line — that's history, not a patch.
- **Don't rephrase for style.** Text that already states a current fact tersely
  stays as is. Edit only to remove something (patch, narrative, duplicate) or
  fix something (stale, dangling). This keeps a repeat run on an unchanged
  bank a no-op.
- **Only substantive facts — no process noise.** The memory bank records the
  *project*, not your maintenance of it. Do **not** add "actualized on <date>"
  or "synced the docs" entries, and do not repeat the same standing rule across
  many files — state it once in its authoritative home and link to it. When a
  sync makes an old "needs updating" caveat stale, delete the caveat.
- **Don't invent.** No facts, paths, dates, or decisions that aren't in the
  changes, the code, or the existing memory. Unknown stays unknown.
- **Review before commit.** Make the edits, then show the diff and a summary.
  Do **not** auto-commit unless the user asks (interactive) or the unattended
  rules below apply. Auto-memory files live outside the repo and are written
  directly (not committed).
- **Convert relative dates to absolute** (use the current date from context).

## State: `.memory-bank/.defrag.yaml`

```yaml
date: <YYYY-MM-DD>
ref: <full commit SHA>
```

- `date` — the last defrag that changed something; `ref` — `git rev-parse HEAD`
  at that run, omitted when the project has no git.
- Machine state, not knowledge: never link it from `index.md`, never mention it
  in memory files. The `memory-bank-load` hook lists only `*.md` files, so it
  stays out of session context.
- Write it only on a run that changed content. A run with no edits leaves it
  untouched — a state-only diff is noise.
- It's committed together with the defrag edits when `.memory-bank/` is
  tracked; otherwise it stays local.

## Workflow

### 1. Resolve the baseline

First match wins:

1. The `[ref|date]` argument, if passed.
2. `ref` from `.defrag.yaml`, if git is present and the commit still exists:
   `git cat-file -e "<ref>^{commit}"`.
3. `date` from `.defrag.yaml` — the ref was rewritten away (rebase, squash) or
   never recorded. Turn it into a ref: `git rev-list -1 --before=<date> HEAD`.
4. No git — the mtime of `.defrag.yaml` marks the last defrag.
5. No state file (first run) — no baseline. Actualize verifies every claim
   against the current tree instead of a change set. Don't fall back to an
   arbitrary "last N commits".

### 2. Gather what changed

With a ref (baseline vs working tree, so uncommitted work counts too):

```bash
git log <ref>..HEAD --format='%h %cI %s'
git diff <ref> --stat -- . ':(exclude).memory-bank'
git status --short -- . ':(exclude).memory-bank'
```

Without git:

```bash
find . -type f -newer .memory-bank/.defrag.yaml \
  -not -path './.memory-bank/*' -not -path '*/node_modules/*' \
  -not -path '*/vendor/*' -not -path '*/dist/*' -not -path '*/build/*' \
  -not -path '*/.venv/*'
```

Read the actual diffs (or the changed files) for anything that changes a fact
the memory bank records: stack/dependency changes, renamed or deleted files,
new integrations, resolved decisions, changed config, new constraints. Ignore
pure formatting and lockfile churn. An empty or cosmetic-only change set means
actualize has nothing to do — compaction still runs.

### 3. Read the current memory bank

- `ls` and read every file under `.memory-bank/`.
- If auto-memory exists, read its `MEMORY.md` and the referenced files.

### 4. Actualize — detect divergences

Build a list, each tagged with evidence (commit SHA, `file:line`):

- **Stale facts** — memory says X, current code/docs say Y.
- **Dangling references** — memory cites a file/symbol/flag that no longer
  exists.
- **Resolved questions, finished tasks** — open questions the changes answered,
  tasks the changes completed.
- **Uncaptured facts** — durable decisions/constraints in the change set not yet
  in the memory bank.

Without a baseline, check stale facts and dangling references for every claim
and skip the uncaptured-facts sweep — capturing new facts is the `memory-bank`
skill's job.

### 5. Compact — sweep the whole bank

Over every file, whatever the baseline:

- **Amendment trails** — several "Amendment <date>" / "Update:" blocks on one
  decision, superseded-but-still-described content → one current-state
  statement, plus one supersession line where the reversal matters.
- **Narrative** — cut to the facts (see non-negotiables).
- **Finished tasks** in `tasks/` — move the outcome (what now exists, where,
  why) into its home file in `tech-details/`, `steerings/` or
  `product-overview/`. The task shrinks in place to one line, in whatever form
  `tasks/` already uses: what was done + a link to the home file
  (`- [x] <what was done> → [<topic>](../tech-details/<file>.md)`).
- **Closed questions** — the answer moves into the file it belongs to; the
  question is removed.
- **Duplicates** — one fact, one home; other files link to it instead of
  restating it.
- **Index** — `index.md` links every file; no orphan files, no dead links. Keep
  it under 16 KB (the `memory-bank-load` hook truncates it there): over budget,
  move detail out of `index.md` into the linked files.

If the edit list is large or ambiguous in an interactive run, surface it and
confirm direction before mass-editing. In an unattended run, proceed and let
the PR be the review surface.

### 6. Apply the edits

- Edit `.memory-bank/`, the repo current-state docs, and auto-memory per steps
  4–5.
- Auto-memory (skip if absent): update the relevant fact file (don't duplicate —
  edit the existing one), refresh its `MEMORY.md` index line, delete memories
  proven wrong. Link related memories with `[[name]]`.
- If anything changed, write `.defrag.yaml`: `date` = today, `ref` =
  `git rev-parse HEAD` (omit without git).

### 7. Verify

- `grep` the memory bank for leftovers of anything you retired (old vendor, old
  path, old decision) — catch half-updated mentions.
- Re-check internal consistency: a table row and the prose around it agree; a
  closed question or finished task isn't still referenced as open elsewhere;
  every link in `index.md` resolves.

### 8. Report + commit

Summary: files touched, patches folded, tasks collapsed, questions closed,
facts corrected, anything you deliberately left alone.

- **Interactive run:** show the diff (`git diff`, or the list of edited files
  without git); commit only when the user asks.
- **Unattended run:** see below.

The commit message follows the project's commit convention; by default a
one-line Conventional Commit, no body, no trailer:
`docs(memory-bank): <one-line theme>`. The baseline lives in `.defrag.yaml`, so
the message carries nothing machine-readable.

## Unattended / scheduled runs

When there is no human to review mid-run (a scheduled routine, CI, headless):

- **Open a pull request, never push to `main`.** Create a branch
  (`docs/memory-bank-defrag-<YYYY-MM-DD>`), commit the edits together with
  `.defrag.yaml`, push it, and open a PR whose body is the step-8 summary. A
  human merges.
- **No edits → nothing.** No branch, no commit, no PR. A quiet run is the
  correct outcome when the repo and the bank are already in sync.
- **Don't loop on yourself.** `.memory-bank/` is excluded from the change set,
  and a run without edits doesn't touch the state, so the next run on an
  unchanged repo is a no-op. Docs you edited outside `.memory-bank/` reappear in
  the next change set — they're already consistent and produce no edits.
- Auto-memory is local-only and absent here — operate on the repo alone.

## Notes

- This skill changes documentation/knowledge only — it never edits product code.
- If the project's `CLAUDE.md`/`AGENTS.md` defines a memory-update protocol, a
  working-language rule, or commit conventions, follow them.
- Keep the memory bank's existing tone and structure; match it, don't impose a
  new one.
