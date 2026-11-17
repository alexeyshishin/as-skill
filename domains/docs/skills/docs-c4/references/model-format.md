# C4 model format

One JSON file per diagram — the source of truth. `scripts/render.py` lays it out and writes the SVG; `scripts/check-c4.py` checks it. A complete example: [`example-container.json`](example-container.json).

## Top level

| key | required | meaning |
|-----|----------|---------|
| `title` | yes | system + level, e.g. `Bookshop — containers` |
| `level` | no, `container` | `context`, `container` or `component` |
| `lang` | no, `en` | `en` or `ru` — language of the legend, the table headers and the reading note |
| `layers` | yes | rows top to bottom: `[{"label": "Storage", "nodes": ["pg", "s3"]}]`; node order = left to right |
| `nodes` | yes | `{"id": {"name", "tech", "kind", "col"}}` |
| `boundaries` | no | `[{"id", "name", "nodes": [...]}]` — a dashed panel around adjacent nodes of one layer |
| `edges` | yes | `[{"from", "to", "label", "tech", "description", "protocol"}]`; numbered in list order |
| `callers` | no | node or boundary ids whose arrows get their own color, up to 6; default: up to 4 nodes with 3+ outgoing edges |
| `note` | no | one extra line under the title for what arrows can't show (`Temporal long-polls its workers; 7–9 show who starts whom`) |
| `layout` | no | `{"seed": 1, "box_width": 240, "gap": 150}` — layout seed, box width, gap between grid columns |

## Nodes

| key | meaning |
|-----|---------|
| `name` | box title, up to ~26 characters |
| `tech` | second line: language, product, runtime (`Go`, `PostgreSQL 16`, `Stripe`); required on container and component level except `person` and `ext` |
| `kind` | shape and color, see below; default `svc` |
| `col` | grid column, 0-based, halves allowed — pins a node over the ones it calls; default: the layer is centered |

| kind | drawn as | for |
|------|----------|-----|
| `person` | pill | a human user or role |
| `svc` | blue box | our own deployable / component / system |
| `model` | amber box | an ML model server or LLM gateway we run |
| `db` | green cylinder | database, cache, queue, bucket |
| `aux` | violet box | supporting infra we run: tracing, metrics, scheduler UI |
| `ext` | dashed grey box | a system someone else runs |

## Edges

| key | meaning |
|-----|---------|
| `from`, `to` | caller and callee ids — the one who starts the call is `from`, whichever way data flows |
| `label` | on the arrow: what the call does, verb first, from the caller's side, up to ~34 characters |
| `tech` | on the arrow in brackets: `format/transport` — `JSON/HTTPS`, `gRPC/Protobuf`, `SQL/TCP`, `S3/HTTPS`, `RESP/TCP`, `AMQP/TCP`; required on container and component level |
| `description` | full sentence for the relationships table; default: `label` |
| `protocol` | full protocol for the table (`HTTPS, GitHub App API`); default: `tech` |

An edge from a boundary means "every member makes this call" and replaces identical edges from each member.

## Layering

Callers above callees, so arrows point down. A typical container stack:

| layer | holds |
|-------|-------|
| people and triggering externals | users, chat platforms, webhook senders that start flows |
| entry | API gateway, BFF, webhook receiver, frontend |
| orchestration | workflow engine, scheduler |
| services and workers | business services, background workers |
| models and stores | databases, queues, buckets, model servers |
| external dependencies | SaaS APIs, providers the system calls |

An external system that both starts calls and receives them (git hosting: sends webhooks, receives PRs) sits where most of its edges point down; the rest become upward callbacks. Keep callbacks to a few — `too-many-reverse` fires above 20% of edges.

Two nodes of one layer can't be linked: move the callee one layer down.

## Layout knobs

`render.py` places nodes on a grid, routes every arrow orthogonally through horizontal lanes between layers and vertical corridors (left, right, between grid columns), and picks routes and lane order by simulated annealing to cut crossings; then it places labels where they hit no box, arrow or other label. When the result is poor, in this order:

1. reorder nodes inside a layer: put a node under the caller that talks to it most
2. pin a node with `col`
3. shorten labels — long ones don't fit next to short segments
4. widen `layout.gap` (more room for vertical lanes and labels) or `layout.box_width`
5. try another `--seed`; keep the winner in `layout.seed`, so re-renders are byte-stable

`render.py` exits 2 when some labels don't fit — the SVG is written, those arrows show only their number.

## Check codes

| code | severity | meaning |
|------|----------|---------|
| `schema` | ERROR | missing or mistyped field |
| `level`, `lang`, `kind` | ERROR | value outside the allowed set |
| `unknown-node`, `duplicate-node`, `unplaced-node`, `duplicate-id` | ERROR | ids don't line up between layers, nodes, boundaries, edges |
| `boundary-layers`, `boundary-gap`, `boundary-overlap`, `boundary-member` | ERROR | a boundary spans layers, has a non-member inside, shares a member, or is linked to its own member |
| `same-layer`, `self-edge`, `duplicate-edge` | ERROR | edge can't be drawn top-down or repeats another |
| `missing-label`, `missing-tech` | ERROR | arrow or node lacks its text |
| `too-many-reverse` | ERROR | layer order is off |
| `stale-svg`, `svg-missing` | ERROR | SVG doesn't match the model — re-render |
| `pair-structure`, `pair-lang` | ERROR | RU model differs from EN in more than text |
| `reverse-edge` | WARN | arrow points up — fine for a callback |
| `orphan-node`, `person-not-top`, `level-kind` | WARN | node placement or level mix-up |
| `long-name`, `long-label`, `long-tech`, `tech-format` | WARN | text too long or not `format/transport` |
| `wide-layer`, `dense`, `many-callers` | WARN | diagram too busy — split it |
| `layer-label`, `boundary-name`, `unknown-key` | WARN | missing name or a typo |
| `untranslated` | WARN | RU text equals EN |
