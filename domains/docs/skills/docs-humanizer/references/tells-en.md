# English AI tells (scanner catalog)

What [`scripts/scan-tells.py`](../scripts/scan-tells.py) looks for in English text, and why. English is covered in README mode only; the full humanizer in `SKILL.md` is Russian.

Source: [Wikipedia: Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing) (read 2026-11-17), plus README-specific patterns. The page itself warns that these are observations, not proof, and that the vocabulary drifts by model generation — "delve" faded after 2024, "showcasing", "highlighting", "enhance" persist. Update the lists when the page does.

```bash
python3 scripts/scan-tells.py README.md                    # report; language and Markdown mode detected
python3 scripts/scan-tells.py README.md --max-density 1.5  # gate: exit 2 above the threshold
python3 scripts/scan-tells.py before.md after.md           # removed / kept / added by the rewrite
```

## Point tells

The occurrence itself is the signal. Weight 1–3 is signal strength.

| code | group | examples | weight |
|------|-------|----------|--------|
| `A1` | signposting | it's worth noting, it's important to note, it should be noted, needless to say | 3 |
| `A2` | summary connectives | in summary, in conclusion, to sum up, in essence | 2 |
| `A3` | sentence-initial additive | Additionally, Furthermore, Moreover, Notably, | 2 |
| `A4` | certainty filler | undoubtedly, without a doubt, rest assured | 2 |
| `B1` | copula avoidance | serves as a, stands as a | 2 |
| `B2` | corporate verbs | leverage, utilize, empower, streamline, elevate your | 1 |
| `B3` | trailing -ing analysis | ", highlighting / underscoring / showcasing / fostering …" | 2 |
| `C1` | marketing adjectives | seamless, cutting-edge, state-of-the-art, revolutionary, blazing fast, supercharge, unleash | 3 |
| `C2` | soft praise | powerful, robust, intuitive, elegant, effortless, comprehensive, feature-rich, versatile | 1 |
| `C3` | AI vocabulary | delve, tapestry, testament, pivotal, vibrant, intricate, meticulous, underscore, showcase, multifaceted, realm, ever-evolving | 2 |
| `C4` | grand openers | in today's fast-paced world, in the ever-evolving landscape, let's dive in, embark on | 3 |
| `C5` | significance inflation | plays a pivotal role, is a testament to, paves the way, game-changer, to the next level, unlock the full potential | 3 |
| `D1` | not just X, but Y | not only fast, but also safe | 3 |
| `D2` | it's not X, it's Y | it's not a library — it's a platform | 3 |
| `D3` | no X, no Y, just Z | no config, no boilerplate, just code | 3 |
| `E1` | audience range | whether you're a beginner or an expert; and everything in between | 2 |
| `F2` | hedged non-answer | generally speaking, to some extent, it depends on your use case | 2 |
| `H1` | chatbot artifacts | I hope this helps, happy coding, great question, Certainly! | 3 |
| `H2` | reader nudges | feel free to, don't hesitate to | 1 |
| `H3` | knowledge cutoff | as of my last update | 3 |
| `R4` | README greeting | Welcome to X, This project aims to, a simple yet powerful | 3 |

`C2` and `H2` are weak on purpose: "robust" is a statistics term, "feel free to open an issue" is ordinary README prose. They count only when they pile up.

## Aggregate tells

The rate is the signal, not the occurrence.

| code | what | threshold |
|------|------|-----------|
| `G1` | em dash | > 1.0 per 100 words |
| `F1` | hedging: potentially, possibly, arguably, typically, generally, usually, often | > 1.5 per 100 words |
| `D4` | triads "X, Y, and Z" | > 1.2 per 100 words |

The em dash threshold is lower than Russian's 2.5: in Russian the dash is grammar, in English the dash-as-aside is the signal.

## Markdown tells (both languages)

On for `.md` files (`--md on|off` overrides).

| code | what | rule |
|------|------|------|
| `R1` | emoji at the start of a heading | each one, weight 2 |
| `R2` | emoji at the start of list items | 3+ items, weight 2 |
| `R3` | bold inline-header bullets `- **Fast:** …` | 4+ items, weight 2 |

Markdown mode skips fenced code, inline code, link targets, image syntax, HTML, URLs, and dash-only table cells, so a comment inside a code block or a `—` placeholder in a table never counts.

## Thresholds

Same scale as the Russian catalog (`tells-ru.md`): < 1.5 clean · 1.5–3.5 spot edits · 3.5–6.0 rework the flagged passages · > 6.0 rewrite.

Measured on 2026-11-17: this repo's `README.md` 0.24, `tools/README.md` 0.30, `README_RUS.md` 0.30, `tools/README_RUS.md` 0.38; a deliberately sloppy README (emoji headings, "Welcome to", "seamlessly", "It's not just a parser — it's a toolkit") 48.65 in English and 54.39 in its Russian version. Two repos are not a calibration set: treat the thresholds as a guide.

## What the scanner can't see

- claims without evidence: "fast" with no benchmark, "secure" with no threat model
- a Features list of adjectives instead of capabilities
- a Why section that restates the title
- summary paragraphs at the end of sections
- sections that could describe any project

That is the reading pass in README mode (`SKILL.md`).
