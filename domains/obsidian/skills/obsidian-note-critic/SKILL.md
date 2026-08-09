---
name: obsidian-note-critic
description: "Critical analysis of Obsidian notes — hybrid search for similar and opposing topics in the knowledge base, identifying contradictions with existing knowledge, reasoned critique of argumentation and coverage completeness. Use this skill whenever the user asks to: critique a note, break it down, find contradictions, check the argumentation, find similar or alternative notes, do a peer review, \"what did I miss\", \"what's missing\", \"what's wrong\", \"find the gaps\". Also apply it after creating a new note if the user wants to make sure it doesn't conflict with existing knowledge."
---

## Overview

The skill analyzes a note along four axes:

1. **Similar knowledge** — what already exists in the base on this topic
2. **Opposing views** — alternative approaches, competing ideas
3. **Contradictions** — where the note diverges from existing knowledge
4. **Reasoned critique** — weak points in logic, wording, coverage completeness

**Vault root folder (`<vault>/`):** set at install time (`--vault` or the `OBSIDIAN_VAULT` env var). See `.agents/rules/vault-struct.md`.
**Output language:** Russian, technical terms in English.

---

## Algorithm

### 1. Read the note

Read the full content. Note down for yourself:

- **Main thesis** — what's being claimed or described (one phrase)
- **Key claims** — 3–7 specific statements from the text
- **Domain** — field of knowledge (SRE, DevOps, career, management, etc.)
- **Sources** — is there a `sources` field, `links`, quotes, or is this purely personal opinion
- **Confidence** — the `confidence` field, if present

### 2. Search for similar topics

Use `mcp__obsidian-hybrid-search__myvault_search` with several phrasings to catch different angles on the same topic. Good practice is to give three queries: one in Russian, one in English, and one "by contrast" (what this is NOT, or how it differs).

Parameters:

- `queries`: [main thesis, key concept, English equivalent]
- `limit`: 8
- `threshold`: 0.3
- `rerank`: true

Skip the note being analyzed itself (usually rank 1). Collect the 4–7 most relevant results.

### 3. Search for opposing views

Formulate queries that catch alternative approaches or criticism:

- `queries`: ["alternative to [thesis]", "criticism of [approach]", "against [concept]", "downsides of [method]"]
- `limit`: 6
- `threshold`: 0.25

Also: if the note is about practice X — search for "when X doesn't work", "limitations of X", "X vs Y".

### 4. Contradiction analysis

Compare the note's content against the similar notes found. Look for:

- **Direct contradictions** — note A claims X, note B claims not-X
- **Terminological mismatches** — the same concept is called different things, causing confusion
- **Contextual conflicts** — a claim is true in one context, wrong in another
- **Outdated knowledge** — a later note updates or overrides this one

For each contradiction: quote the exact wording from both notes. Without specifics it's not a contradiction, just a guess.

### 5. Reasoned critique

Assess the note along three dimensions. Each critical point is one sentence stating the claim + one sentence justifying it. Optionally: a recommendation.

**Tone:** specific, well-argued, respectful. The critique targets the note, not the author.

#### Argumentation

- A claim with no backing → point out that it needs a source or a personal-experience example
- A single source with `confidence: high` → risk of confirmation bias
- Circular logic (A because B, B because A)
- Generalization without examples ("always", "never", "all" — without evidence)
- Causation substituted for correlation

#### Wording

- Vague concepts with no definition ("efficiency", "quality" — of what exactly, how is it measured?)
- Passive voice hiding the subject ("it is believed", "it is common practice", "it is known" — by whom?)
- Mixing factual description and personal opinion in the same sentence

#### Coverage completeness

- Obvious aspects of the topic that aren't mentioned
- Edge cases or exceptions that might refute the thesis
- Related topics that would be worth considering or mentioning

---

## Output format

### When to do a short review → callout in the note

Choose this format if all of the following hold at once:

- ≤ 3 critical points
- Each fits in 1–2 sentences
- No deep contradictions requiring a breakdown with quotes

Add **at the end** of the analyzed note:

```markdown
> [!note] Review
> **Similar notes:** [[Note 1]], [[Note 2]]
> **Contradiction:** [[Another note]] claims X, this one claims Y; possibly different contexts.
> **To improve:** The thesis isn't backed by sources — add `sources`.
```

The "Similar notes" and "Contradiction" lines are optional — add them only if there's something to say.

### When to do a long review → separate note in the inbox

Choose this format if at least one of the following holds:

- ≥ 4 critical points
- There's at least one contradiction requiring a breakdown with quotes
- Several alternative positions need to be compared

Create the file `00. Входящие/Ревью — [Note title].md`:

```markdown
---
aliases: []
tags:
  - inbox/review
up:
  - "[[Title of the analyzed note]]"
down: []
links: []
other: []
---

## Similar notes

- [[Note 1]] — how it's relevant, what it adds to the topic
- [[Note 2]] — how it's relevant

## Opposing views

- [[Note 3]] — exactly how it's an alternative

## Contradictions

> [!warning] [[Note X]] vs this note
> **There:** "exact quote"
> **Here:** "exact quote"
> Possible resolution: different context / one is outdated / needs a synthesis

## Critique

### Argumentation

- Claim. Justification. Recommendation.

### Wording

- Claim. Justification.

### Coverage completeness

- What's not covered and why it matters.

## Next steps

- [ ] Add a source for thesis X
- [ ] Cross-check against [[Conflicting note]]
- [ ] Consider aspect Y
```

After creating it — tell the user the file's path.

---

## Validation mode: close out a review

Use this mode when the user says "check what's been done", "close the review", "validate the changes", "what from the review has already been applied", "update the checklist".

### Validation algorithm

#### 1. Find the review note

The review note lives at `00. Входящие/Ревью — [Note title].md`. If the user didn't specify a title — find it via backlink or ask.

If there's no review note — say so and suggest running the critique first.

#### 2. Read both notes

Read the current content of the original note and the review note. Note down the list of items from the `## Next steps` section.

#### 3. Check each item

For each checklist item, determine its status:

| Status | Criterion |
|--------|----------|
| ✅ Done | The change is clearly present in the original note |
| ⚠️ Partial | The change is there but incomplete (e.g., `confidence` was added but not the source) |
| ❌ Not done | The original note hasn't changed in this respect |

Be specific: if the item is "add `confidence: medium`" — check the frontmatter field. If it's "add a link to [[X]]" — check whether that link exists in `other` or in the text.

#### 4. Update the checklist in the review note

Replace the `## Next steps` lines in the review note with the updated version:

```markdown
## Next steps

- [x] Completed item — applied ✅
- [-] Partially completed item — clarify: X done, Y not yet ⚠️
- [ ] Item not done ❌
```

#### 5. Decide whether to close it

**Everything done** (no `[ ]` or `[-]` left):
- Delete the review note from `00. Входящие/`
- Report: "All items closed, the review note has been deleted."

**Partially done** (some done, some not):
- Leave the review note in the inbox with the updated checklist
- Report a summary: how many closed, how many remain, exactly what remains

**Nothing done**:
- Don't change the review note
- Say so directly and offer to help apply the changes

---

## Constraints

- Don't edit the note's main body except to add the callout at the end
- Don't delete or change existing tags, links, or frontmatter
- Don't add wikilinks to nonexistent notes
- If there are no similar notes (everything below the 0.25 threshold) — say so honestly, don't invent links
- If there are no contradictions — don't manufacture them
- In validation mode, don't rewrite the critique — only update the checklist and decide whether to close it
