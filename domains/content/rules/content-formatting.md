# content-formatting

Formatting for specific platforms. Skills in the `content` domain reference this as `~/.claude/rules/content-formatting.md`.

## Telegram

Telegram supports a limited markdown via "MarkdownV2" / HTML. Notes:

- **bold**: `*text*` (MarkdownV2) or `<b>text</b>` (HTML)
- *italic*: `_text_` or `<i>text</i>`
- `monospace`: `` `text` `` or `<code>text</code>`
- ```code blocks``` — triple backticks, can specify a language: ` ```go`
- ~~strikethrough~~: `~text~` or `<s>text</s>`
- Spoilers: `||text||` or `<tg-spoiler>text</tg-spoiler>`
- Links: `[text](https://...)` or `<a href="...">text</a>`

**Escaping in MarkdownV2:** the characters `_*[]()~`>#+-=|{}.!` must be escaped with a backslash. It's often easier to work around this with HTML formatting instead.

**Limits:**
- maximum 4096 characters per message
- if content is longer — split into several messages, don't try to squeeze it in
- ideal post length for a channel is 500-1500 characters

**Post structure:**
- **first sentence is the hook** — a thesis or a provocation
- **body** — short paragraphs (1-3 sentences), separated by a blank line
- **ending** — either a conclusion, a question to the reader, or just a stop
- **signature** (if needed) — on its own line, separated by a double line break

Don't use `#` headings — Telegram doesn't render them as markdown, they show up literally.

## Articles (Markdown, Habr / Medium / personal blog)

- **Title** — H1 (`#`), one per article.
- **Section subheadings** — H2 (`##`). Don't go deeper than H3 unless necessary.
- **Paragraphs** separated by a blank line.
- **Lists** — real `-` or `1.`, not "emoji bullets."
- **Code blocks** must specify a language.
- **Images and diagrams** — alt text every time. This matters both for accessibility and for SEO.
- **TL;DR** at the top is fine for articles longer than 5000 characters.

## Technical tutorials

- **State the tutorial's goal in the first line.** "After this tutorial you'll be able to deploy a K8s cluster with a TLS cert-manager on bare metal."
- **Prerequisites** as an explicit list: versions, access, environment.
- **Number the steps.** Each step = one action + the expected result.
- **Commands must be copy-pasteable** (see also `~/.claude/rules/sre-runbook-template.md`).
- **Verification commands** after every non-trivial step: "confirm the pod is Ready: `kubectl get pod ...`".
- **A Troubleshooting section** at the end with common errors.
- **Cleanup** — how to roll everything back if needed.

## General rules

- **English terms** stay in English, don't transliterate them (`pod`, not a transliterated form; conversationally a transliteration is fine, but in written text English is better).
- **Backticks for anything technical**: file names, commands, variables, flags.
- **Numbers > 999** — with separators: `1 000`, `5 000 000` (a space for Russian, a comma for English).
- **Time and dates** — in ISO format: `2026-05-23`, `14:30 UTC`. Not "May 23rd at half past two."
- **Hyphen vs. dash** — use the correct one: `-` hyphen within words, `—` em-dash within sentences.

## What to check before publishing

- open the preview on a phone — are long lines not getting mangled?
- do code blocks get syntax highlighting?
- are links clickable and do they lead where they should?
- do the title and first sentence work on their own (in case someone only sees those)?
