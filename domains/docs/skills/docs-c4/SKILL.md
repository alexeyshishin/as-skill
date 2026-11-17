---
name: docs-c4
description: >
  Draws C4 architecture diagrams (context, container, component) from the
  actual code: finds services, stores, external systems and who calls whom in
  compose files, manifests, dependencies and client code, writes a JSON model
  with layers top to bottom (caller above callee), checks it, and renders an
  SVG with numbered arrows, [format/transport] labels, a legend and a
  relationships table; optional Russian pair. Use when the user wants to "draw
  a C4 diagram", "make an architecture diagram", "container diagram", "context
  diagram", "update the architecture diagram", "нарисуй C4", "схема
  архитектуры", "диаграмма контейнеров", "обнови схему". Not for sequence
  diagrams or flowcharts; a review of an existing diagram is docs-c4-reviewer.
---

# docs-c4 — C4 diagrams from the code

Goal: a diagram where a newcomer sees in a minute which parts exist, who calls whom and over what — and every box and arrow traces to the code.

Before starting, read `references/rules/docs-conventions.md` — section C4 diagrams.

Assets:
- [`references/model-format.md`](references/model-format.md) — model fields, node kinds, layering, layout knobs, check codes.
- [`references/example-container.json`](references/example-container.json) — a complete container model to start from.
- [`scripts/check-c4.py`](scripts/check-c4.py) — model checker; also flags a stale SVG and checks an EN/RU pair (Step 5).
- [`scripts/render.py`](scripts/render.py) — layout, label placement, SVG; `--png` through headless Chrome (Step 6).

Both scripts need only `python3` 3.8+; `--png` needs Chrome or Chromium (`CHROME_BIN` to override).

## Step 1. Pick level, scope and mode

| level | shows | scope |
|-------|-------|-------|
| `context` | the system as one box, the people who use it, the external systems around it | the whole system |
| `container` (default) | deployable units and stores inside the system, plus the people and external systems they talk to | the whole system |
| `component` | parts inside one container and what they call | one container |

| situation | mode |
|-----------|------|
| no model for this level and scope | **new** |
| model exists, the code moved on | **update** — edit the model, re-render |
| model fine, SVG stale or layout poor | **re-render** — Step 6 only |
| the doc that embeds the diagram has a `_RUS` version | **pair** — also Step 6.5 |

Files: `docs/architecture/c4-<level>.json` + `.svg` next to it; component level adds the scope: `c4-component-api.json`. An existing location in the project wins. Ask only when the user's words don't settle level or scope.

For **update** read the model and `git log -3 --format='%h %cs %s' -- <model>` first.

## Step 2. Gather facts from the code

Everything on the diagram must trace to a file. Read:

| what | where |
|------|-------|
| our deployables | `docker-compose*.yml` services, Kubernetes manifests and Helm `templates/` (Deployment, StatefulSet, CronJob), `Dockerfile`s, `Procfile`, `cmd/*`, `apps/*`, `services/*` |
| stores | images in compose (`postgres`, `redis`, `minio`, `rabbitmq`), drivers in manifests (`pgx`, `psycopg`, `redis`, `kafka`, `boto3`, `aws-sdk`), migrations directories |
| external systems | SDKs in manifests (`stripe`, `openai`, `slack`), base URLs in config, env vars the code reads named `*_URL`, `*_HOST`, `*_DSN`, `*_ENDPOINT`, `*_API_KEY` |
| who calls whom | client construction and calls: HTTP clients and their base URLs, gRPC stubs from `.proto`, producers and consumers with their topics or queues, SQL per service, S3 clients |
| callbacks | webhook routes (`/webhook`, signature checks) — an upward edge from the sender |
| tech | the client kind: `JSON/HTTPS`, `gRPC/Protobuf`, `SQL/TCP`, `RESP/TCP`, `AMQP/TCP`, `S3/HTTPS` |

Keep an evidence list as you go: each node and edge → `file:line`. It's shown at Step 7, not drawn.

Only read. Don't start services, port-forward or call endpoints to find out.

## Step 3. Ask what the code can't tell

One message, only the gaps: the system's name, which product hides behind a config value, whether a box is ours or someone else's, what a call is for when nothing in the code says. Don't fill a gap with a guess.

## Step 4. Write the model

Start from `references/example-container.json`; fields in `references/model-format.md`. Then:

- `from` is whoever starts the call: a read from a database is service → database; a consumer pulling from a queue is consumer → queue
- layers top to bottom by who starts calls: people and triggering externals → entry → orchestration → services and workers → models and stores → external dependencies; every arrow points down
- an upward arrow only for a callback the callee starts (webhook, push into chat) — a few at most
- two nodes of one layer never link: move the callee one layer down
- order nodes in a layer so each sits under the caller that talks to it most; pin with `col` when centering puts it elsewhere
- similar nodes of one layer (workers) → a boundary; an edge from the boundary replaces the same edge from every member
- `label` — what the call does from the caller's side, verb first, short; `tech` — `format/transport`; the long version goes to `description` and `protocol` for the table
- `callers` — the busiest callers, so their arrows get their own color
- `note` — one line for what arrows can't show (long-poll, fan-out)
- one level per diagram: a context diagram has no databases, a container diagram no classes

## Step 5. Check

```bash
python3 scripts/check-c4.py docs/architecture/c4-container.json --no-svg
```

Exit 0 — proceed. Exit 2 — fix and re-run. A remaining WARN is fixed or kept with a reason you'll state at Step 7 (a `reverse-edge` for a real webhook is a reason).

## Step 6. Render and look

```bash
python3 scripts/render.py docs/architecture/c4-container.json --png "${TMPDIR:-/tmp}/c4-preview.png"
```

It writes `c4-container.svg` next to the model and prints `crossings`, `overlaps` and `labels placed/total`. Open the PNG and look at it: every label readable and next to its arrow, no arrow through a box, the main flow readable top to bottom.

Exit 2 or a tangle: work through the knobs in `references/model-format.md` → Layout knobs — reorder layer nodes, pin `col`, shorten labels, widen `layout.gap`, try a few `--seed` values. Keep the winning seed in `layout.seed`. Done when every label is placed, overlaps are 0, and reordering no longer cuts crossings.

## Step 6.5. Russian pair

When the doc embedding the diagram has a `_RUS` version: copy the model to `c4-<level>_RUS.json`, set `"lang": "ru"`, translate `title`, `note`, layer labels, node names (product names stay), edge labels and descriptions; terms stay English per `docs-conventions` → EN ↔ RU. Ids, order, kinds, boundaries, edges, callers don't change. Render it, then check both:

```bash
python3 scripts/check-c4.py docs/architecture/c4-container.json
```

It pairs `<name>_RUS.json` and both SVGs automatically (`--ru PATH` for another name).

## Step 7. Embed, show, ask

Embed the SVG in the architecture doc — the one the project has, else `docs/architecture.md` (and its `_RUS` pair) — with alt text that says what it shows:

```markdown
![C4 container diagram: services, stores and external systems of Bookshop](architecture/c4-container.svg)
```

The README links that doc from its Documentation section; it doesn't inline the diagram. Before overwriting a file with uncommitted changes — ask.

Show: the PNG, the render line (crossings, labels), the evidence list, the WARNs kept and why. Ask:
1. Accept
2. Adjust (what)
3. Add another level or scope

## What not to do

- don't draw a box or an arrow the code doesn't show; don't invent names, products or protocols
- don't edit the SVG by hand — every render overwrites it from the model; change the model
- don't flip an arrow to follow data instead of the call
- don't mix levels on one diagram
- no AI attribution or "generated by" text in the model, the SVG or the doc
- don't commit or push
