---
name: memory-bank-defrag
description: >
  Defragment and re-actualize a project memory bank (the .assistant/ knowledge
  base + the project's auto-memory). Reads everything that changed in the repo
  since the last defrag, finds where the memory bank has gone stale or
  accumulated amendment-on-amendment "patches", folds the patches into clean
  current-state docs, closes resolved questions, and updates the auto-memory —
  then shows a diff for review. Use when the user says "defrag the memory bank",
  "привести memory bank в порядок", "дефрагментация памяти", "актуализируй
  .assistant", "причеши базу знаний", or runs /memory-bank-defrag.
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, AskUserQuestion
argument-hint: "[baseline-ref]  (optional: e.g. HEAD~20 or a commit SHA to override auto-detected baseline)"
---

# Memory Bank Defrag

Bring a project's memory bank back in sync with reality and remove cruft. The
memory bank is three things:

1. **`.assistant/`** in the repo — the durable, version-controlled knowledge
   base (project context, requirements, decisions/ADRs, scope, open questions,
   stakeholders, glossary). Structure varies per project; operate on whatever
   files exist, don't assume a fixed layout.
2. **Auto-memory** — the cross-session memory at
   `~/.claude/projects/<slugified-cwd>/memory/` (an `MEMORY.md` index plus one
   file per fact). Local only — **absent on cloud/headless runs**; skip silently
   if the directory doesn't exist.
3. **Repo current-state docs** — files that assert *what is true now* and drift
   the same way: `README.md`, `AGENTS.md`/`CLAUDE.md`, design specs, proposals.
   Update these when a recent commit makes them stale. Do **not** touch raw
   source records that capture *what happened* — meeting transcripts, chat
   dumps, extracted board content — those are evidence; keep them verbatim
   (translate only by adding a summary alongside, never by overwriting).

The job is **defragmentation**, not rewriting: same facts, fewer patches.
Amendment-on-amendment trails collapse into one current-state statement;
resolved questions get closed; stale facts get corrected against the actual
code/docs; new durable facts from recent commits get captured.

## Non-negotiables

- **Accuracy > speed.** Memory is point-in-time and may be wrong. Before you
  assert a fact as current, **verify it against the live repo** (file exists,
  symbol not renamed, deadline not passed, stack matches code). If a memory
  claim contradicts current code, trust the code and fix the memory.
- **Fold, don't lose.** When collapsing patches, preserve every substantive
  fact. A reversal that carries decision-relevant context (why we moved off X)
  stays as a short supersession note — that's history, not a patch.
- **Only substantive facts — no process noise.** The memory bank records the
  *project*, not your maintenance of it. Do **not** add "actualized on <date>",
  "switched to English", "synced the docs" timeline/history entries, and do not
  repeat the same standing rule (e.g. "working language is English") across many
  files — state it once in its authoritative home and link to it. When a sync
  you just did makes an old "needs updating" caveat stale, delete the caveat;
  don't leave a trail.
- **Don't invent.** No facts, paths, dates, or decisions that aren't in the
  commits, the code, or the existing memory. Unknown stays unknown.
- **Review before commit.** Make the edits, then show the diff and a summary. Do
  **not** auto-commit unless the user asks (interactive) or the unattended rules
  below apply. Auto-memory files live outside the repo and are written directly
  (not committed).
- **Convert relative dates to absolute** (use the current date from context).

## Workflow

### 1. Locate the baseline (last defrag)

Find the commit of the previous defrag so you only process what changed since.

```bash
# Preferred: explicit trailer this skill writes on its own commits
git log -1 --grep='^Memory-Defrag:' --format='%H %cI %s'
# Fallback: the conventional subject line
git log -1 --grep='actualize memory bank' --format='%H %cI %s'
```

- If the user passed a `baseline-ref` argument, use it instead.
- If neither is found (first ever run), say so and propose a sensible range
  (e.g. last 20 commits, or since the memory bank dir was created:
  `git log --diff-filter=A -1 --format=%H -- .assistant`). Confirm with the user
  before processing a large history (interactive) or just use it (unattended).

### 2. Gather what changed since baseline — and bail early if nothing did

```bash
git log <baseline>..HEAD --format='%h %cI %s'           # commit subjects
git diff <baseline>..HEAD --stat -- . ':(exclude).assistant'  # what code/docs moved
```

**No-op guard:** if `<baseline>..HEAD` is empty (no commits since the last
defrag), there is nothing to do. Stop here — no edits, no commit, no PR. Report
"nothing changed since the last defrag" and exit. (This is what keeps a daily
schedule cheap and quiet.)

Otherwise read the actual diffs (or the changed files) for anything that
changes a fact the memory bank records: stack/dependency changes, renamed or
deleted files, new integrations, resolved decisions, changed config, new
constraints. Ignore pure formatting / lockfile churn — if the only changes are
cosmetic, treat it as a no-op too.

### 3. Read the current memory bank

- `ls` and read every file under `.assistant/`.
- If auto-memory exists, read its `MEMORY.md` and the referenced files. (On
  cloud/headless runs this directory won't exist — skip it.)

### 4. Detect divergences

Build a list, each tagged with evidence (commit SHA, file:line):

- **Stale facts** — memory says X, current code/docs say Y.
- **Patches to fold** — multiple "Amendment <date>" blocks on one decision,
  superseded-but-still-described content, the same fact stated three ways across
  files. Collapse to current state + one supersession note where a reversal
  happened.
- **Resolvable questions** — open questions the recent commits actually answered.
- **Dangling references** — memory cites a file/symbol/flag that no longer exists.
- **Uncaptured facts** — durable decisions/constraints in recent commits not yet
  in the memory bank.

If divergences are large or ambiguous in an interactive run, surface the list
and confirm direction before mass-editing. In an unattended run, proceed and let
the PR be the review surface.

### 5. Apply the edits

Edit `.assistant/`, the repo current-state docs (README and the like), and
auto-memory to current state:

- Rewrite patched sections as clean current-state; keep a one-line
  "superseded by … because …" only where the reversal matters.
- Close resolved questions in place (mark resolved + the answer + the commit),
  matching the file's existing convention.
- Correct stale facts; fix or drop dangling references.
- Capture new durable facts in the right file.
- Auto-memory (skip if absent): update the relevant fact file (don't duplicate —
  edit the existing one), refresh its `MEMORY.md` index line, delete memories
  proven wrong. Link related memories with `[[name]]`.

### 6. Verify

- `grep` the memory bank for leftovers of anything you retired (old vendor,
  old path, old decision) — catch half-updated mentions.
- Re-check internal consistency: a table row and the prose around it agree; a
  closed question isn't still referenced as open elsewhere.

### 7. Report + commit

Summary: files touched, patches folded, questions closed, facts corrected,
anything you deliberately left alone.

- **Interactive run:** show the diff (`git diff`); commit only when the user asks.
- **Unattended run:** see below.

Commit subject + trailer (the trailer is what the next run's baseline detection
finds — always include it):

```
docs(assistant): actualize memory bank — <one-line theme>

<what was folded/closed/corrected>

Memory-Defrag: <YYYY-MM-DD>
```

## Unattended / scheduled runs (cloud routine)

When there is no human to review mid-run (a scheduled routine, CI, headless):

- **Open a pull request, never push to `main`.** Create a branch
  (`memory-defrag/<YYYY-MM-DD>`), commit with the trailer above, push it, and
  open a PR whose body is the step-7 summary. A human merges. This keeps the
  baseline-trailer in history once merged, and keeps the knowledge base reviewed.
- **If the no-op guard fired (step 2), do nothing** — no branch, no PR. A quiet
  run is the correct outcome on a day with no relevant commits.
- **Don't loop on yourself.** Your own defrag commit carries the `Memory-Defrag:`
  trailer and becomes the next baseline, so the following run sees no new
  substantive commits and no-ops. Never treat a prior defrag commit as a change
  to process.
- Auto-memory is local-only and absent here — operate on the repo alone.
- **Announce the PR in Mattermost** (next section) once it's open, so a human
  knows there's a knowledge-base diff to review. Skip entirely on a no-op run.

### Notify Mattermost (after the PR is opened)

When a PR was opened, post a short report to Mattermost via incoming webhook so a
reviewer can open the PR and leave comments there. **Only after a successful PR —
never on a no-op run** (a quiet day stays quiet).

- Webhook: read it from env `MM_DEFRAG_WEBHOOK` — never hardcode a hook URL in
  the skill or the repo (it's a write credential). Skip the whole notify step if
  the var is unset.
- Channel: whichever channel the run instructions name, else env `MM_DEFRAG_CHANNEL`
  (each project targets its own channel). Use the channel **name/slug** (the URL
  handle), not the 26-char channel ID — the webhook override rejects IDs. If neither
  is set, omit `channel` and the post lands in the webhook's default.

**Language** — write the message in the **project's working language** (the one
its `CLAUDE.md`/`AGENTS.md` mandates). Don't default to English on a project that
works in another language.

**Message content** — write it for a human skimming chat, not a changelog dump.
Keep it short and scannable:

1. **Title** — the one-line run theme; links to the PR (`title_link`).
2. **Folded / corrected** — 2–4 grouped bullets of what you tidied (patches
   folded, facts corrected, questions closed). Summarize, don't list every file.
3. **Heads-up** — gaps you noticed that a human or the client should check or
   confirm. Include only items that genuinely need someone — if there are none
   this run, say so in one line rather than padding.
4. **Call to action** (always last) — make clear you don't read replies in the
   channel; you act on PR review comments and edit reactively, so they should
   comment inline on the PR and keep the thread there.

**Formatting** — use a Mattermost attachment (colored bar + clickable PR title)
with markdown in the body. Build the JSON with `jq` so the summary text can't
break the payload — never hand-concatenate summary text into JSON:

```bash
PR_URL="<from gh pr create>"
TITLE="Memory bank defrag — <one-line theme>"
BODY=$(cat <<'EOF'
Tidied up the `.assistant/` knowledge base after <N> commits — docs only.

**Folded / corrected**
- <grouped bullet>
- <grouped bullet>

**Heads-up**
- <gap to check / confirm with client — or: "nothing needs input this run">

:point_right: **I don't read replies in this channel.** I act on the PR review
comments and make edits reactively — drop your notes inline on the PR and let's
keep the thread there.
EOF
)

curl -fsS -X POST -H 'Content-Type: application/json' \
  --data "$(jq -n \
    --arg ch "${MM_DEFRAG_CHANNEL:-}" --arg title "$TITLE" \
    --arg link "$PR_URL" --arg body "$BODY" \
    '{username:"Memory Bank",
      attachments:[{color:"#2eb886", title:$title, title_link:$link, text:$body,
                    fallback:("Memory bank defrag — review the PR: "+$link)}]}
     + (if $ch=="" then {} else {channel:$ch} end)')" \
  "$MM_DEFRAG_WEBHOOK"
```

The webhook post is best-effort: if it fails (non-2xx, network), log it in the
run summary but don't fail the run — the PR is the source of truth, the ping is
convenience.

## Notes

- This skill changes documentation/knowledge only — it never edits product code.
- If the project's `CLAUDE.md`/`AGENTS.md` defines a memory-update protocol or a
  working-language rule, follow it.
- Keep the memory bank's existing tone and structure; match it, don't impose a
  new one.