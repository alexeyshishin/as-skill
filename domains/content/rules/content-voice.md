# content-voice

Authorial voice and tone for content (Telegram, articles, tutorials). Skills in the `content` domain reference this as `~/.claude/rules/content-voice.md`.

## Basic principles

- **Address the reader as "you" (informal)** or use impersonal constructions. Not the formal "you," not a bureaucratic tone.
- **Russian language**; technical terms stay in English (`SRE`, `Kubernetes`, `goroutine`, `Pod`), formulas in LaTeX.
- **Authorial voice**, not Wikipedia. This is a personal stance, experience, sometimes doubts and jokes.

## What makes text recognizable

- **Specifics instead of abstractions.** "The service went down" → "The API returned 503 for 4 minutes, we lost ~3000 requests."
- **Technical detail without snobbery.** Don't explain the obvious to experienced readers, but don't shut the door on newcomers either — give context in one brief phrase.
- **Personal experience, not theory.** "I tried X — it doesn't work because Y" beats "It is commonly believed that X."
- **Admitting the unknown.** "I didn't figure out why, but" is better than a made-up explanation.

## Bans

- **No "dear readers,"** "esteemed colleagues," "as is well known."
- **No marketing promises** ("revolutionary," "seamless," "best in the industry").
- **No Wikipedia style** ("represents," "is," "it can be noted that").
- **Don't use AI clichés**: "let's dive in," "in this article we will look at," "in conclusion, I'd like to note."
- **Don't quote random gurus** without reason. If the author is genuinely important, name them.

## Structural preferences

- **Short paragraphs.** 1-4 sentences. On a phone screen, a long paragraph is a wall of text.
- **Minimal bullet points.** Prose is better where possible. Use bullets only for a genuine list of 3+ parallel items.
- **Subheadings** in articles/tutorials. Usually unnecessary in Telegram.
- **Code blocks with a language tag**: ` ```go`, ` ```bash`. Syntax highlighting matters.

## Length

- **Telegram post**: 500-1500 characters. One point, one idea, no "introductions."
- **Article**: 3-10k characters. Thesis → arguments → example → implication.
- **Tutorial**: as long as it needs to be to reproduce the result. No longer.

## Structure of thought

- **Thesis first.** Don't keep the reader in suspense for 5 paragraphs — they'll close the tab.
- **Concrete arguments.** Numbers, links, code. Not "as practice shows."
- **Counterargument / nuance** — if there is one. Trust grows when the author doesn't pretend to know everything.
- **An ending isn't mandatory.** If the thought is finished — stop. Don't write "thus" just for the sake of closure.

## Emoji

- In Telegram — **sparingly**, when appropriate. No more than 1-2 per post.
- In articles — almost never (especially in code/technical content).
- Don't use emoji as bullet markers (✅, ❌, 🚀) — it reads as marketing.

## Self-check before publishing

- if you remove the first 1-3 sentences, does the text lose meaning? If it does **not** — cut them.
- is there at least one concrete example / number / command in the text?
- could any other author have written this? If so — add something personal.
- will the reader be able to reproduce / apply this, or is it just an essay?
