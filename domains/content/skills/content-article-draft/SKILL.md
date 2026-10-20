---
name: content-article-draft
description: >
  Turns an idea, a series of notes, or raw material into an article draft
  (Habr / Medium / personal blog) — with a TL;DR, a thesis → arguments →
  example → implication structure, and an authorial voice. Use when "write an
  article," "shape these thoughts into a long-read," "make a Habr post,"
  "article draft."
---

# content-article-draft — Article draft

Goal: turn material into a **first working draft** of a 3-10k character article. Not a final version — the user polishes that themselves. The draft needs the right skeleton and tone.

Before starting, read:

- `~/.claude/rules/content-voice.md`
- `~/.claude/rules/content-formatting.md` — the articles section

## Step 1. Context

Ask:

- **the topic** in one sentence
- **the audience**: technical specialists at a certain level? Managers? Beginners?
- **the platform**: Habr / Medium / dev.to / personal blog / Telegraph?
- **the source material**: notes, experience, a project, an existing draft?
- **the article's goal**: teach something / share experience / break down a case / a provocative essay?

## Step 2. Formulate the thesis

An article's thesis is **a single claim** that the author defends or illustrates. Not a topic, but a claim.

| Topic (bad) | Thesis (good) |
|--------------|----------------|
| "Microservices" | "Microservices are only worth the complexity when you genuinely have different release cycles or languages" |
| "Postgres vs Mongo" | "For most startups, Postgres covers 95% of the needs, and the remaining 5% don't justify a second database" |
| "My experience with k8s" | "After a year running k8s in production, I realized the main risk isn't complexity, but the blurring of SRE vs. dev responsibility" |

Show the thesis to the user and **ask them to confirm or correct it**. The thesis is the foundation.

## Step 3. Build the skeleton

Standard structure:

```markdown
# <Title>

> **TL;DR:** <thesis in 1-2 phrases + a brief summary>

## <Subheading 1: context / problem>

<Why this is interesting. A concrete real-life example.>

## <Subheading 2: argument / breakdown / solution>

<The main point. Code, diagrams, numbers.>

## <Subheading 3: nuances / counterarguments>

<Where the thesis doesn't hold. Where there are caveats.>

## <Subheading 4: application / summary>

<What the reader can do with all of this.>

---

<signature / author / links to resources>
```

Show the skeleton (title + subheadings + a one-phrase description of each section) and ask if it works. Don't move on to writing the full text until the skeleton is agreed on.

## Step 4. Write the draft section by section

Once the skeleton is confirmed — write section by section.

Applying `content-voice.md`:

- informal "you" / impersonal address
- authorial voice, not Wikipedia
- specifics, numbers, code
- no AI clichés

Applying `content-formatting.md`:

- code blocks with a language tag
- images/diagrams — alt text
- backticks for technical terms
- paragraphs — 1-4 sentences

## Step 5. TL;DR at the top

For articles longer than 5000 characters — a TL;DR block at the top is mandatory. 2-3 lines: thesis + main takeaway + who will find it useful.

## Step 6. Self-check

- **the thesis carries through** the whole article, it doesn't get diluted
- **there are concrete examples** (not just theory)
- **there's code / diagrams / numbers** where possible
- **a counterargument** is mentioned — otherwise the text reads like propaganda
- **the reader can apply something** after reading
- does the article still hold together if you remove the intro?
- does the ending make sense? (or is it just "thus")

## Step 7. Show it and ask

Show the draft. Below it:

- length: X characters
- checks: ✓ thesis / ✓ specifics / ✓ counterargument / ✓ code

Ask:

1. Accept as a first draft
2. Fix a specific section
3. Strengthen the thesis / change the angle
4. Cut it down (if it ran to 12k+)
5. Go deeper on a particular argument

## Step 8. Save

Ask where to save it. If there's an Obsidian-vault or blog-repo convention, ask for the path. Otherwise suggest `drafts/YYYY-MM-DD-<short-slug>.md` next to the current working context.

## What not to do

- don't produce an article without an explicit thesis — that's an essay, not an article
- don't quote random gurus for authority
- don't write "in this article we will look at" — cut it and get to the point
- don't glue a series of loosely related thoughts into one article — a series of 3 short pieces is better
- don't finalize the article without the author confirming the thesis — it's their text, not yours
