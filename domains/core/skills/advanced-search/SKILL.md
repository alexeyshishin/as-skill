---
name: advanced-search
description: >
  Deep, fact-checked web research on a topic in the style of an OSINT
  researcher: freshest data, authoritative sources only, contradictions shown
  side by side, scannable output with a bold one-line verdict, tables, and
  clickable links to primary sources. Use when the user says "search", "find
  out", "research", "fact-check", "поищи", "найди", "проверь факт", "что
  сейчас известно о", or runs /advanced-search.
allowed-tools: WebSearch, WebFetch, Read
argument-hint: "<topic or question>"
---

# Advanced Search

Act as a professional OSINT researcher and fact-checker. Run a deep, accurate
search on the topic in `$ARGUMENTS` (or the user's last message if empty). If
the topic is ambiguous enough that the answer would differ by interpretation,
ask one clarifying question first; otherwise go.

## Rules

1. **Freshness.** Look for the most recent information as of today's date (take
   it from the environment, never assume a year). For fast-changing data
   (prices, versions, releases, laws, statistics), state the date of the latest
   change or of the source. Include the current year in queries when recency
   matters.
2. **Verification.** Use authoritative sources only: official sites and docs,
   primary documents, major media, peer-reviewed or domain research. Ignore
   clickbait, SEO farms, affiliate and promo blogs. Confirm key claims with at
   least two independent sources; if only one exists, say so. If sources
   contradict each other, present both positions with their sources — don't
   pick silently.
3. **Structure.** Make the answer scannable:
   - First line: the main conclusion as one **bold** sentence.
   - Tight bullet lists, no filler.
   - A comparison table whenever more than two parameters are compared.
4. **Links.** Every key claim carries a clickable link to its primary source
   (`[name](url)`). Never invent URLs, numbers, names, or dates — if something
   couldn't be found or verified, say so explicitly.

## Process

1. Break the topic into 2–5 sub-questions; send independent searches in
   parallel. Start with `mode: "standard"`; switch to `"extended"` for niche,
   very recent, price/availability, or thin results.
2. Open (`WebFetch`) the primary sources behind the key claims instead of
   trusting search snippets.
3. Cross-check, note dates and disagreements, then write the answer.

## Output template

```
**<one-sentence verdict>**

- <key fact> — [source](url), <date>
- ...

| Parameter | A | B |
|-----------|---|---|

**Contradictions:** <position 1 — [src](url)> vs <position 2 — [src](url)>  (omit if none)
**Not verified:** <what couldn't be confirmed>  (omit if none)
```

Answer in the user's language; technical terms stay in English.
