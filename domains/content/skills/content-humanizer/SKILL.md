---
name: content-humanizer
description: "A skill for humanizing Russian-language text. Removes signs of AI generation, makes the text feel alive. ALWAYS use this when the user asks to: humanize a text, remove traces of a neural network, make a text lively/natural, rewrite it like a human, humanize in Russian, remove officialese, remove wateriness/filler, make a text less formal. Also use this if the user pastes in Russian-language text and says something like 'rewrite this,' 'make it better,' 'sounds like a robot,' 'too artificial.' Works ONLY with the Russian language. For English, use the original humanizer. Do NOT use this for: translation, writing from scratch, grammar, code."
---

# Humanizer

You are an editor. You turn sterile AI text into living Russian speech. You don't just strip out neural-network markers — you bring the author back into the text: with an opinion, a rhythm, a character.

Good Russian text is uneven. It stumbles, interrupts itself, speeds up and slows down. AI text is smooth and bland, like elevator music.

## Fundamental principle: statistical deviation

> An LLM picks the statistically most probable continuation of the text. The result gravitates toward the single most typical variant — the one applicable to the largest number of cases.

Humanizing = a deliberate deviation from the statistical norm. Every word choice, every turn of phrase, every rhythmic break is a choice of the LESS probable but MORE characteristic variant. AI writes «Это имеет важное значение» ["This holds important significance"]. A human writes «Это меняет всё» ["This changes everything"] or «Ну и что?» ["So what?"] — depending on the author. Both variants are statistically less probable, but both carry character.

Keep this principle in mind for every decision: "An AI would pick the most typical variant. Which variant would THIS particular author pick?"

Two key facts from the research:

- **LLMs favor nouns and nominalizations over verbs.** AI text is consistently "more nominal": more deverbal nouns and participial phrases than human writing (PNAS, "Do LLMs write like humans?", arxiv 2410.16107; survey arxiv 2510.05136). There's no single "canonical" noun/verb ratio in the literature — this is a working heuristic, not a benchmark figure. The point: humans anchor language in verbs (tense, aspect, mood), AI anchors it in noun phrases.
- **LLMs process Russian through English-biased representations.** The model generates through an internal "translation" from English (arxiv 2502.11806), so calques in AI-generated Russian aren't random mistakes — they're an artifact of the architecture. A translationese preference has been confirmed for multilingual models (arxiv 2603.08450, on the en-sv pair; no Russian-specific translationese study was found, but the mechanism is the same). This explains WHY patterns 7 (calques) and 8 («является» ["is"]) are so persistent.

## What detectors actually pick up on (2025-2026)

Detectors (GPTZero, Originality.ai, DivEye, RuBERT) measure three things:

1. **Perplexity (predictability).** How predictable each next word is. AI text has low perplexity: every word is "expected." Human text jitters: a predictable word, an unexpected one, a predictable one again.

2. **Burstiness.** Variability of structure across the document. AI writes evenly: all sentences are ~the same length, ~the same complexity. Humans alternate: a long complex sentence, a short choppy one, a question, a long one again.

3. **Morphological correctness (for Russian).** Russian-language detectors are fine-tuned on transformers (RuBERT/RuRoBERTa) and pick up morphological anomalies implicitly, as a statistical feature, not as a separate "case-checking module." In practice this means one thing: AI makes morphological mistakes differently than people do. People mix up -тся/-ться [a common reflexive-verb-ending confusion], AI mixes up cases in long chains. Clean morphology in the rewritten text is mandatory.

The humanizing task: raise perplexity (less predictable words), raise burstiness (structural diversity), preserve morphological cleanliness.

Specific numbers:

- DivEye (arxiv 2509.18880, PAN 2025): second derivatives of surprisal contribute **39.4%** to detection — more than any other feature type (distributional 34.2%, first-order 23.7%).
- Perplexity gap: even at a 99.9% style match by human ratings, average human-text perplexity is **29.5** vs. **15.2** for LLMs (arxiv 2509.24930). Detectors see this.
- Adversarial paraphrasing sharply cuts detectors' true positive rate (a universal attack, arxiv 2506.07001). But perturbation-invariant methods are emerging: PIFE (arxiv 2510.02319) holds **82.6% TPR** at 1% FPR even after sophisticated attacks, versus 48.8% for ordinary adversarial training. PIFE measures how strongly a text was "perturbed" by paraphrasing — that is, it catches the very fact of humanizing. Conclusion: simple paraphrasing no longer saves you; you need a distribution shift (voice calibration), not a surface-level word shuffle.
- PNAS 2025 ("Do LLMs write like humans?", arxiv 2410.16107): profanity appears **~100x less often** in AI text than in human text (GPT-4o uses "fuck" at ~0.008 of the human rate). Participial phrases, conversely, appear 2-5x more often. Perception verbs ("to look," "to hear") and fear/anger/hatred words appear an order of magnitude less often.

> **Domain shift:** Detectors don't generalize across domains (arxiv 2603.23146, March 2026). A model trained on academic texts poorly catches blog posts, and vice versa. The most informative features for one domain are useless for another. Practical takeaway: the more a text is anchored to a specific niche (jargon, format, audience style), the harder it is to detect. This is an additional argument for voice calibration and domain adaptation.

For Russian: the latest Russian-language benchmark is AINL-Eval 2025 (arxiv 2508.09622): 52K scientific abstracts, 12 scientific domains, GPT-4-Turbo/Gemma2/Llama3.3/DeepSeek-V3/GigaChat as generators. The best system is a fine-tuned transformer, 86.35% on the test set. Important caveat: the benchmark is narrow (scientific abstracts); the numbers will differ on blogs and posts. The principles (surprisal, burstiness) are language-agnostic, but the threshold values for Russian haven't been calibrated.

## Operating principle: contrastive subtraction

> Research (CoPA, EMNLP 2025, arxiv 2505.15337) has shown: the most effective way to humanize a text isn't to strip markers off a list, but to find the MOST PREDICTABLE word in each sentence and replace it with a less probable one that still fits the particular author.

Predictable ≠ formal. «Решение» ["solution"] in the context of «нашли решение проблемы» ["found a solution to the problem"] is predictable. «Выход» ["a way out"], «лазейка» ["a loophole"], «костыль» ["a workaround/hack"] are less probable but characterful. One such choice per sentence does more than three stylistic edits. This is a supplement to the pattern catalog, not a replacement: first strip the HARD BANS, then do a pass of contrastive subtraction.

> **Uncertainty gap** (arxiv 2602.16162, 2026): a formalized gap — human text is consistently less predictable than AI text, and this correlates directly with quality. Instruction tuning and reasoning models AMPLIFY predictability. Contrastive subtraction is a direct way to close this gap. The best 2025-2026 attacks go through style transfer (MASH, arxiv 2601.08564: 92% ASR), not paraphrasing, which confirms the skill's approach: voice calibration (Step 2) + contrastive subtraction > mechanically swapping out markers.

---

## Priority of the author's style

> This section overrides everything else, including the HARD BANS. Read it before you start editing.

If the project has an author style-guide (a separate skill like `*-blog-style`, a `STYLE.md` file, a `CONTRIBUTING.md` with tone rules, or the user has directly described their style) — **load it first and treat it as the source of truth**. In case of conflict, the style-guide rule wins over a HARD BAN. The humanizer removes generation traces, but doesn't rewrite a living author's voice into the averaged norm.

Why this matters: the catalog below is built on the statistics of the "average AI text." For an author with a distinctive voice, many "markers" are deliberate devices, not model artifacts. Mechanically cutting them out turns a recognizable text into a sterile one — that is, it does exactly what the skill was created to prevent (see the homogenization warning in Step 3).

Typical conflict points where we defer to the author's style by default:

- **Dashes.** A grammatical Russian dash is legitimate (see #15). The decision on dash typography belongs to the style-guide, not the HARD BAN.
- **The contrastive antithesis «не X, а Y»** ["not X, but Y"]. This is a rhetorical figure, not pattern #38 (which only bans «не просто X, а Y» ["not just X, but Y"] and «не только X, но и Y» ["not only X, but also Y"]). We don't touch a load-bearing argumentative device.
- **Didactic structure.** Parallel blocks, triads, bolded highlighting of theses in a long technical read are scanning aids for the reader, not generation laziness.
- **Register and vocabulary.** If the style-guide specifies informal address, anglicisms, dry humor — that's the author's voice, not something that needs "fixing."

Conflict-resolution rule: first ask, "is this a style-guide requirement, or an authorial device applied consistently throughout the text?" If yes — leave it and don't report it as a finding. The humanizer intervenes only where a marker isn't explained by the author's style.

## Modes of operation

**Full edit** (default). All 5 steps, the full pattern catalog. For texts that need to be brought to a human-sounding state.

**Audit** (on request: "check this," "find AI markers," "what gives it away?"). Diagnosis only, you don't rewrite the text. Sort findings into four buckets — don't dump them into one pile, or a deliberate authorial device reads as a defect:

1. **Confirmed AI tics.** Real generation markers with an example and a priority (A-D). This is fixed without a second thought.
2. **Density observations.** The device is fine on its own, but there's too much of it (e.g., an antithesis in every paragraph). Not "remove," but "dilute for variety." Flag it as an observation, not an error.
3. **False positives on deliberate devices.** Things the catalog formally flags but that are explained by the author's style or style-guide (see "Priority of the author's style"). List these explicitly and mark that they should NOT be touched — so the author can see the skill noticed it and deliberately skipped it.
4. **Non-AI defects.** Typos, cut-off sentences, broken links, factual errors. Worth reporting, but honestly labeled: this isn't about humanizing.

**Spot editing** (on request: "fix only X," "remove the officialese"). You work only with the specified category of patterns. Leave everything else alone.

## Text classification

Before working, determine the type of text — it determines the intensity of the edit:

| Type | Intensity | What to touch | What NOT to touch |
|---|---|---|---|
| Marketing / social media | Maximum | All 52 patterns + HARD BANS + tone | - |
| Expert content (Habr, articles) | High | A-C patterns, voice, specifics | Terminology; didactic structure (parallel blocks, triads, bold highlighting) if it helps the reader scan |
| Business correspondence | Medium | A-B patterns, officialese, wateriness | Formal register, polite phrasing |
| Documentation / tech texts | Low | Only A patterns + gross errors | Structure, terminology, format |
| Legal texts | Minimal | Only factual errors | Everything else (wording carries legal force) |
| Quotes within the text | None | Nothing | Everything (a quote = someone else's text) |

For short texts (<100 words): don't overload them with edits, removing 2-3 main markers is enough.
For texts mixing languages: work only with the Russian-language fragments.
If the text is already good: say so. Don't edit for the sake of editing.

**Test for deliberate structure** (for patterns #45 macro-burstiness, #12 rule of three, #16 bold): before flagging a structural pattern, ask "does this parallelism carry an argument, or does it help the reader scan?" In a didactic long-read, blocks like "What's being checked → Failing answer → Strong answer," triads of sections, and bold-highlighted theses are navigation for the reader, not generation monotony. Flag a structural pattern only if it's empty: identical blocks with no semantic payload, bold on every other word, a triad of synonyms. A deliberate didactic template — leave it.

## Pattern priorities

| Group | Level | Patterns | When to fix |
|---|---|---|---|
| A | Critical | HARD BANS, empty openings (1), officialese (6), chatbot artifacts (22), negative parallelisms (38), **modal uncertainty (46), pseudo-therapy (52)** | ALWAYS, in any mode |
| B | High | Vague authorities (2), calques (7), **punctuation calques (7b)**, «является» ["is"] (8), wateriness (26), «данный» ["this/given"] (30), emotional sterility (31), uniform density (43), smooth transitions (44), **macro-burstiness (45), translationese (47), choppy meditativeness (49), emoji decoration (51)** | In every mode except legal |
| C | Medium | Inflation (3), formulaic conclusions (4), syntax (11), rule of three (12), synonym carousel (13), dashes (15), hedging (25), particles (32), **no idioms (48), counter-questions (50)** | In full-edit and expert modes |
| D | Stylistic | Bold (16), minor spelling issues (17-18), quotation marks (21), lists (19), punctuation (20), literacy (36), typography (37) | Context-dependent, not mandatory |

When time or tokens are limited: fix top to bottom (A → B → C → D).

---

## Process (5 steps + quad-pass audit)

**Step 0. Mechanical scan.** If the text is in a file — run the scanner before reading it:

```bash
python3 scripts/scan_tells.py <file>
```

It finds occurrences of the regex-matchable subset of the catalog and counts density per 100 words. This doesn't replace diagnosis: the scanner can't see an even rhythm, a lack of specificity, or emotional flatness — half of what matters. But it never misses an occurrence and costs no tokens, so use your eyes for what it can't do, not for what it's already found.

Thresholds and scope of applicability are in [`references/tells-ru.md`](references/tells-ru.md). The scanner's report is a pointer, not a measurement: it shows where to look, it doesn't prove the text is machine-written.

**Step 1. Diagnosis + segment marking.** Read the text. Find concrete instances of the patterns from the catalog below. Not all 52 — only the ones that are actually present. Mark each instance found. Then tag paragraphs with a traffic light:

- **Red** (3+ markers): rewrite completely in Step 3.
- **Yellow** (1-2 markers): spot-edit, keep the structure.
- **Green** (clean): DON'T TOUCH IT. Rewriting a clean paragraph introduces new AI markers. Bonus: untouched paragraphs create "mixed content," where detectors perform worst (accuracy <62% — though that's for older document-level detectors). Warning: sentence-level detectors (arxiv 2509.17830) already catch individual AI sentences inside human text, so betting on "untouched paragraphs" is getting weaker. So even in green paragraphs, make 1-2 contrastive substitutions as a precaution.

**Step 2. Voice calibration.**

If the user gave samples of their own writing, run a structured analysis:

- **Rhythm:** average sentence length, variability (short/long), favorite constructions
- **Vocabulary:** formality (1-10), jargon, professional terms, colloquialisms
- **Quirks:** signature phrasings, favorite particles, characteristic digressions
- **Punctuation:** ellipses? parentheses? dashes? questions? exclamations?
- **Tone:** ironic, businesslike, friendly, provocative, mentor-like?

Write down a "voice passport" in 3-5 lines and check against it while rewriting.

If there are no samples, write like a smart person explaining something to a friend over coffee. Not like a textbook, not like a corporate report.

**Step 3. Rewrite according to the markup.** Work by the traffic light from Step 1: rewrite red paragraphs completely, spot-edit yellow ones, leave green ones alone. In red and yellow paragraphs: first strip the HARD BANS, then do a pass of contrastive subtraction (in each sentence, replace the most predictable word with a less probable but fitting one). At the same time, add voice and vary the structure. Check against the voice passport.

> **WARNING: homogenization.** An LLM doing a rewrite tends to strip out colloquialisms, anecdotes, and personal examples, replacing them with neutral phrasing (arxiv 2603.18161: +70% neutral essays when using an LLM). Even a "grammar only" prompt changes the semantics. If the original has a personal story, a specific turn of phrase, a colloquial expression — PRESERVE them. Don't replace something alive with something neutral. Applying the same transformation to every text through one tool creates a "humanized" style that is itself detectable (DAMAGE, arxiv 2501.03437). Vary your approach: rewrite different texts differently.

**Step 4. Quad-pass audit.**

Pass 1, "Detector": reread the draft, look for leftovers from each of the 12 pattern categories (A-L). 30 seconds per category is enough.

Pass 2, "Person off the street": forget you're an editor. Read the text as a random reader in a feed. Question: "Seeing this text with no context, would I think a neural network wrote it?" If yes — find exactly what gives it away and fix it.

Red flags for the second pass:

- Too smooth, not a single rough edge
- Every paragraph is the same length
- Every transition is smooth (live text sometimes jumps)
- Not a single unexpected word
- A feeling that the text could be about anything (no authorial specificity)
- All emotions are positive or neutral (no irritation, skepticism, indignation)

Pass 3, "Cardiogram" (for texts >300 words): mentally draw a graph: X-axis is sentences, Y-axis is "how unexpected this sentence is after the previous one." For a human, the graph jitters. For AI, it's almost a straight line. If the graph is smooth — insert 2-3 spikes: an unexpected comparison, an abrupt question, a numeric fact amid the reasoning, a personal remark in parentheses.

Pass 4, "Skeleton" (for texts with lists, numbered items, sections): read ONLY the first line of each item/section in a row, ignoring the content. That's the text's skeleton. Question: "Does the skeleton sound like a template?" If 3+ items start the same way (one construction, one length, one type of delivery) — that's a macro-burstiness failure (pattern #45). Fix it: different openers, different block lengths, different types of explanation. This pass catches what Passes 1-3 miss: structural monotony between blocks, not within them.

**Step 4.5. Re-scan.** If you worked with files — run a comparison:

```bash
python3 scripts/scan_tells.py <before> <after>
```

Look not at what was "removed," but at what was **"added during the rewrite."** A rewriting model carries the same clichés back in through the paraphrase: it removes «важно отметить» ["it's important to note"], and brings in «стоит подчеркнуть» ["it's worth emphasizing"]. The text changed, it feels better, but the marker density stayed the same — and this is almost impossible to catch by eye.

New tells appeared — remove them one at a time, without rewriting the paragraph again. A second pass will introduce a third batch.

**Step 5.** Hand over the final text and a short list of key changes (3-5 items).

---

## Hard bans (HARD BANS)

These constructions are ALWAYS forbidden. Don't fix them — delete them and restructure the phrase.

> The Antislop study (arxiv 2510.15061, ICLR 2026): some phrases occur in LLM text **1000+ times more often** than in human text. 8000+ patterns have been identified. Detectors know this. The constructions below are the highest-frequency markers.

| Construction | Why it's banned | Use instead |
|---|---|---|
| «Не просто X, а Y» | GPT's signature formula. Appears in 80%+ of AI texts | A direct statement — "Y" — without the contrast |
| «Не только X, но и Y» | A calque, a marker of synthetic emphasis | "X. And also Y" or just list them |
| «В современном мире...» | Empty opening #1 for every LLM | Start with a fact or a question |
| «Стоит отметить, что...» | A calque of "It's worth noting" | Cut it and say it directly |
| «Важно понимать, что...» | A calque of "It's important to understand" | Cut it, or use "Here's the thing:" |
| «Данный» / «Данная» / «Данное» | An officialese marker, always replace it | The plain Russian word for "this" (этот/эта/это), matching gender |
| «Является» (>once per 500 words) | Russian doesn't need the "to be" copula in the present tense | Restructure the sentence |
| «Играет важную/ключевую роль» | Inflated significance, a cliché | Show WHY it matters, with data |
| «Можно с уверенностью сказать» | A redundant hedge + inflation | Say it without the preamble |
| «Подводя итог» / «Таким образом» | A formulaic conclusion that sticks onto any text | Cut it, or open the conclusion with an action |
| A long dash "—" (decorative insertion) | AI overuses the dash-as-aside where Russian would use a comma/colon. NOT a blanket ban: see the caveat below the table | Only the decorative insertion: replace with a comma, colon, period, or restructure. Don't touch a grammatical dash |
| «от X до Y» (unrelated concepts) | A false range: "from startups to corporations" | List things concretely, or remove it |
| «В условиях [adj.] [noun]...» | An empty opening, a variant of "В современном мире" | Start with specifics: what exactly is competing, what's transforming |
| «Погрузимся в...» / «Давайте посмотрим X поближе» | A chatbot artifact, compulsive signaling | Start with the substance, don't announce it |
| «И вот здесь начинается самое интересное» | A calque of "and this is where it gets interesting," a ChatGPT marker | Delete it and go straight into the interesting part |
| «Раскрыть потенциал» | An AI motivational cliché, empty emphasis | Show a concrete result |
| «Вывести на новый уровень» | Empty emphasis, an AI marketing stock phrase | Say WHICH level, and what you're measuring it by |
| «Комплексный подход/решение» | Officialese + inflation. Means everything and nothing | List what specifically is included in the "complex" |
| «В связи с этим...» | A formulaic connector that sticks onto anything | Cut it, or show the actual cause-and-effect link |
| «Открывает новые горизонты/перспективы» | A marketing stock phrase, zero information | Name the concrete horizon, or delete it |

When you spot any construction from the list: don't think, delete it. This isn't a stylistic choice, it's a detectable marker. **Exception — the dash** (see the caveat below): that one you actually should think about.

> **Caveat about the dash.** The ban on "—" in the HARD BANS is an English-calibrated heuristic: detectors (GPTZero, DivEye) count em-dash frequency on an English corpus, where the dash-as-aside really is a marker. In Russian, the dash is mandatory grammar, and its frequency alone is a weak signal. So:
> - **Don't touch a grammatical dash.** «X — это Y» ["X is Y"] (a dash standing in for the copula «есть» ["is"]), a dash in an elliptical sentence, a dash in direct speech and dialogue — these are normal Russian. Cutting them out breaks the punctuation.
> - **A decorative insertion can be varied.** The dash AI uses to set off an aside, modeled on the English em-dash, where a comma/colon/period would fit better — that's the actual signal, and that's the one to touch.
> - **The decision on dash typography is delegated to the author's style-guide** (see "Priority of the author's style"). If the project has a rule about dashes, it outranks this point. Don't impose your own typography over the author's blogging infrastructure.

> Markers of AI text evolve (arxiv 2502.09606): words that get noticed start disappearing from AI text ("delve" dropped off after 2024, "significant" keeps rising). The HARD BANS need regular updates. The current list is accurate as of May 2026.

---

## Catalog: 52 patterns of AI generation in Russian

### A. Content-level (1-5)

**1. Empty openings.** AI starts with cosmic generalizations: «В современном мире...» ["In today's world..."], «В эпоху цифровых технологий...» ["In the age of digital technology..."], «Не секрет, что...» ["It's no secret that..."], «Данная тема отличается повышенной актуальностью» ["This topic is of heightened relevance"]. Delete the entire first paragraph. The real text starts with the second one. Or start with a fact, a story, a question.

Before: «В современном динамично развивающемся мире искусственный интеллект играет всё более важную роль в различных сферах жизнедеятельности человека.»
After: «GPT-4 вышел в марте 2023-го. Через полгода его использовали 92% компаний из Fortune 500.»

**2. Vague authorities.** «По мнению экспертов...» ["In the opinion of experts..."], «Специалисты рекомендуют...» ["Specialists recommend..."], «Исследования показывают...» ["Studies show..."], «Многие считают...» ["Many believe..."]. Either name a specific expert, or drop the reference and state it in your own voice. «Я считаю» ["I believe"] is more honest than «многие считают» ["many believe"].

**3. Inflating significance.** Every fact is "key," every event is a "turning point": «играет ключевую роль» ["plays a key role"], «имеет огромное значение» ["is of huge importance"], «невозможно переоценить» ["cannot be overstated"]. Drop the pathos. If something matters, show why through data, not adjectives.

**4. Formulaic conclusions.** Closings that stick onto any text: «Таким образом, можно сделать вывод...» ["Thus, we can conclude..."], «Подводя итог...» ["To sum up..."]. A conclusion should add something new. If it can be deleted without losing meaning, delete it.

**5. Forced structure.** AI stretches "introduction, body, conclusion" over even a Telegram post. For short texts (under 500 words), structure is often unnecessary. Start with the substance.

**Subtype: "A subheading every 2-3 sentences."** AI likes to chop up a short text into many sections with headings. If a 500-word post has 6 subheadings of 2-3 sentences each, that's AI handwriting. A human either writes solid prose, or uses subheadings meaningfully: one section = one big idea.

### B. Linguistic (6-14)

**6. Officialese.** The main marker. AI turns verbs into deverbal nouns: «осуществление» ["implementation"], «реализация» ["realization"], «внедрение» ["rollout"], «оптимизация» ["optimization"], «в целях реализации проекта» ["for the purposes of realizing the project"], «в рамках данного исследования» ["within the framework of this study"]. Bring back the verbs. «Осуществили внедрение системы» ["Carried out the implementation of the system"] means «внедрили систему» ["rolled out the system"]. «Реализация проекта завершена» ["The realization of the project is complete"] means «проект завершён» ["the project is done"]. A verb is almost always better than a noun.

Before: «Осуществление процесса оптимизации рабочих процессов способствует повышению эффективности деятельности организации.»
After: «Навели порядок в процессах, стало быстрее работать.»

**7. Calques from English.** Models are trained on English, and constructions leak through: «Стоит отметить, что...» (It's worth noting), «Важно помнить, что...» (It's important to remember), «Можно сказать, что...» (One could say), «является» ["is"] in every other sentence (is). Rephrase it in Russian. Russian prefers active constructions and allows dropping the subject. English-biased representations in LLMs mean Russian text is generated through an internal "translation": translationese patterns are unavoidable (arxiv 2603.08450, 2502.11806).

**7b. Punctuation calques.** AI places commas by English rules. «Однако,» ["However,"] at the start of a sentence (in English "However," is set off with a comma, in Russian it isn't). «Благодаря этому, результаты» ["Thanks to this, the results"] (a redundant comma). «Инструменты, такие как Python» ["Tools, such as Python"] (a calque of "such as"). «В 2024 году, компания выросла» ["In 2024, the company grew"] (a redundant comma setting off the adverbial). Check every comma after an introductory phrase: in Russian, many phrases are NOT set off with a comma that would be set off in English.

**8. «Является» ["Is"].** AI uses the copula «является» 2-3 times more often than people. Russian gets along fine without "to be" in the present tense. Write «Python является языком программирования» ["Python is a programming language"] as «Python, язык программирования, ...» ["Python, a programming language, ..."], or just restructure the sentence.

**9. Redundant subjects.** Russian allows dropping the subject (pro-drop), but AI always inserts it, because in English the subject is mandatory. «Он встал и он пошёл к двери» ["He got up and he walked to the door"] is better as «Встал, пошёл к двери» ["Got up, walked to the door"].

**10. Piling up participial phrases.** AI builds multi-story participial constructions: «Анализируя данные, учитывая результаты, рассматривая возможности, мы пришли к выводу...» ["Analyzing the data, taking the results into account, considering the possibilities, we arrived at the conclusion..."]. Participial phrases appear **2-5 times more often** in AI text than in human text (Texas study, arxiv 2602.15514; Biber framework, PNAS 2025). Break it into short sentences. One participial phrase, maximum. A regular subordinate clause is better.

**11. Syntactic monotony.** AI writes sentences of the same length (15-20 words) with the same structure. It also **avoids inversions and colloquial constructions**: it prefers the direct word order "subject → predicate → object" (arxiv 2602.15514). Alternate. Short. Then long, with commas, with qualifications, with parenthetical asides. A question? That works too. Use inversions («Хорошо это или плохо — не знаю» ["Whether that's good or bad — I don't know"]). Vary how paragraphs open.

This isn't cosmetic. Dependency-parsing detectors catch AI by the single structure of its syntactic relations, without any lexical cues at all (DependencyAI, arxiv 2602.15514). So swapping words isn't enough — change the syntax itself: the type of subordinate clause, word order, sentence length. Lexical editing and structural variability work together; each alone is weaker.

**12. Rule of three.** AI adores lists of three: «важный, значительный и ключевой» ["important, significant, and key"]. If it's three synonyms, keep one. If the enumeration is artificial, restructure the phrase. Sometimes two items are enough. Sometimes you need four.

**13. Synonym carousel.** AI cycles through «компания», «организация», «предприятие», «фирма» [company/organization/enterprise/firm] to refer to one entity. Pick one word. Repetition is normal in Russian. Unnatural variation is worse than repetition.

**14. Bias toward nouns.** AI text is consistently "more nominal": more deverbal nouns than in human writing (PNAS 2410.16107; survey 2510.05136). There's no single "canonical" noun/verb ratio — this is a working heuristic, not a benchmark figure. LLMs prefer noun phrases; humans anchor language in verbs carrying tense and mood. Instruction tuning AMPLIFIES the bias. ChatGPT has a stable, recognizable register fingerprint (arxiv 2508.16385). If a paragraph is short on verbs, look for nominalizations and unpack them. Also: detectors work even on pure syntax with no lexical cues — dependency relation patterns distinguish human from AI (DependencyAI, arxiv 2602.15514).

### C. Stylistic (15-21)

**15. Dashes.** AI overuses the dash-as-aside: it places «—» where Russian would use a comma, colon, or period. Detectors (GPTZero, DivEye) count em-dash frequency, but this is an English-calibrated heuristic — in Russian the dash is mandatory grammar, and its frequency alone is a weak signal. So DON'T burn out every dash indiscriminately. **Leave the grammatical dash alone:** «X — это Y» ["X is Y"] (standing in for the copula «есть»), an elliptical sentence, direct speech, dialogue. **Vary the decorative insertion:** replace it with a comma/colon/period, or restructure. If the project has an author style-guide with a rule about dashes, it takes priority (see "Priority of the author's style"). Details are in the caveat under the HARD BANS table.

**16. Bold.** AI bolds every key word. The pull toward bold, lists, and dashes comes from the markdown formatting in the training data (arxiv 2603.27006) — that is, it's a stable fingerprint, not chance. Remove it, or keep it for just 1-2 spots in the whole text.

**17-18. Minor spelling issues.** Lowercase after a colon (not «Решение: Следующий шаг» ["Solution: Next Step"], but «Решение: следующий шаг» ["Solution: next step"]). No Title Case in headings (not «Как Создать Отличный Продукт» ["How To Create A Great Product"], but «Как создать отличный продукт» ["How to create a great product"]).

**19. Compulsive lists.** AI turns everything into numbered lists. If the ideas are connected, rewrite them as continuous prose. Lists are good for instructions, not for reasoning.

**Subtype: "duplicate colons."** AI writes lists in the format "Word: An expanded repetition of the word" (e.g., «Эффективность: Повышает эффективность работы» ["Efficiency: Increases the efficiency of the work"]). Three signals: (1) a colon after a single word in every item, (2) the text after the colon duplicates the meaning, (3) a capital letter after the colon. Seeing a list like this, rewrite it as continuous prose or make each item actually substantive.

**20. Impoverished punctuation.** AI uses only periods and commas. No ellipses (a pause, a moment of reflection), no parentheses (an aside), no rhetorical questions. Add variety. Parentheses work great (like right now). An ellipsis... is sometimes fitting too.

**21. Quotation marks.** Depends on context. For articles, documents, official texts, the Russian standard is guillemets «...», with „..." for nested quotes. But for social media posts, Telegram, chats, use straight "quotes" like from a phone. This looks more natural because 90% of people type on mobile and don't bother switching to guillemets. Typographically correct quotation marks in an informal text give away not a human but a robot with perfect typography.

### D. Communicative (22-26)

**22. Chatbot artifacts.** «Конечно! Давайте разберёмся...» ["Sure! Let's figure this out..."], «Отличный вопрос!» ["Great question!"], «Рад помочь!» ["Happy to help!"], «Надеюсь, это было полезно» ["I hope this was helpful"]. Delete these entirely. The author of an article isn't "happy to help."

**23. Sycophancy.** AI agrees with everything. An author has the right to disagree, doubt, argue. Sycophancy is well documented (arxiv 2310.13548): models are trained to give answers the user likes, not necessarily correct ones. Before the rollback in spring 2025, GPT-4o would validate users' intrusive thoughts and praise absurd plans.

**24. Formulaic transitions.** «Давайте рассмотрим подробнее...» ["Let's take a closer look..."], «Перейдём к следующему аспекту...» ["Let's move on to the next aspect..."], «Не менее важным является...» ["No less important is..."]. Cut it (the reader can see there's a new paragraph), or make it substantive: through a question, a contrast, a link to the previous idea.

**25. Redundant hedges.** «В определённом смысле...» ["In a certain sense..."], «В той или иной степени...» ["To one degree or another..."], «Можно предположить, что возможно...» ["One might assume that possibly..."]. If you're sure, state it. If not, say specifically what you're unsure about. «В определённом смысле это работает» ["In a certain sense this works"] means nothing. «Работает для малых выборок, на больших не проверяли» ["Works for small samples, wasn't tested on large ones"] carries information.

**26. Wateriness.** AI text can be cut by 40-60% without losing meaning. One idea gets smeared across 3-5 sentences, the thesis is repeated in different words, paragraphs add nothing new. Compress it. Test: if the text can be cut in half without hurting the meaning, the original was watery.

### E. Morphological (27-30)

**27. Cases.** AI mixes up case endings: nominative instead of genitive («для различных платформ: социальные сети» ["for various platforms: social media" — nominative instead of genitive] instead of «социальных сетей» [genitive]), incorrect gender agreement («Компания заявил» ["The company (fem.) stated (masc. verb form)"]). Proofread every agreement.

**28. Verb aspect.** Perfective/imperfective aspect gets confused: «Он будет делать работу и сделает её» ["He will be doing the work (imperfective) and will do it (perfective)"]. Check that it fits the context.

**29. Gerunds attached to the wrong subject.** «Проанализировав данные, результаты были получены» ["Having analyzed the data, the results were obtained" — the gerund logically refers to "we," but the grammatical subject is "the results"]. A gerund must relate to the sentence's subject. It doesn't? Restructure it.

**30. «Данный», «определённый», «соответствующий»** ["this/given," "certain," "corresponding/appropriate"]. AI inserts these words everywhere: «данный метод» ["this/given method"], «определённые аспекты» ["certain aspects"], «соответствующие меры» ["appropriate measures"]. Replace with «этот» ["this"], or drop it. «Определённые аспекты» ["certain aspects"] means "I don't know which ones specifically."

**Important for morphology:** RuBERT-based detectors specifically analyze case chains and agreement. AI makes mistakes in long constructions: the wrong case after a preposition, mismatched gender in participial phrases. Humans make mistakes differently: mixing up -тся/-ться, adding a stray comma. When rewriting, check every agreement chain in the new phrases. A morphological error characteristic of AI is worse than "human" carelessness.

### F. Tonal (31-33)

**31. Emotional sterility.** AI text is like distilled water: clean, but tasteless. Research confirms it: AI text contains more joy and less negativity (arxiv 2505.01800), LLMs prefer strengtheners over hedges, creating an overconfident tone (arxiv 2507.10587), they "express undue linguistic confidence even when internally uncertain" (arxiv 2411.06528). Profanity appears **100 times less often** in AI text than in human text (PNAS 2025). Perception verbs («смотреть», «слышать», «чувствовать») [to look, to hear, to feel] and words of fear/anger/hatred are rare.

Four subtypes:

- **Positive skew.** Every conclusion is "positive" or "balanced." There are no texts where the author is simply angry, disappointed, or skeptical. A human isn't afraid to say «полгода мучились, результат ноль» ["struggled for six months, ended up with nothing"]. AI will say «несмотря на определённые сложности, данный подход открывает новые перспективы» ["despite certain difficulties, this approach opens up new prospects"]. If the topic implies a negative experience but the text stays optimistic anyway, that's a marker.
- **Absence of doubt.** A human doubts and corrects themselves: «может я и ошибаюсь» ["maybe I'm wrong"], «ну, тут спорно» ["well, that's debatable"], «хотя, нет, подожди» ["actually, no, wait"]. AI states things unequivocally. LLMs don't generate epistemic markers, and when they do, they prefer strengtheners (arxiv 2507.10587). This isn't the same problem as pattern 25 (redundant hedges): pattern 25 is about EXCESSIVE hedging, this one is about the absence of NORMAL human doubt.
- **Emotional dynamics.** Within a single text, a human shifts: excitement over a find, irritation over a bug, skepticism toward a solution, relief that it worked. AI holds one flat tone. If the whole text sits on one emotional note, that's a marker. Check the "emotional cardiogram" alongside the informational one (pattern 43).
- **Authorial rhetoric.** A human persuades through personal experience and mistakes («я попробовал X, не сработало, потом нашёл Y» ["I tried X, it didn't work, then I found Y"]). AI persuades by listing advantages. If a text persuades like a catalog rather than like a story, that's a marker.

Add an authorial stance. «Это работает» ["This works"] can be strengthened to «Это работает, и это удивляет, учитывая какой это костыль» ["This works, and that's surprising, given what a hack it is"]. An opinion is what makes a text human.

**32. No particles or speech habits.** Live Russian text is full of particles: же, ведь, вот, -то, ли, ну [emphatic/discourse particles, roughly like "after all," "you know," "well," etc.]. AI either doesn't use them or places them wrong. «Это важно» ["This is important"] can become «Это ведь важно» ["This is important, you know"] or «Это-то и важно» ["This is exactly what's important"]. Don't overdo it: 1-2 per paragraph for informal text. Besides particles, a human has filler words (ну, короче, типа) [well, basically, like], personal turns of phrase, characteristic sentence openers. LLMs are bad at imitating the informal style of everyday writers, especially on blogs/forums (arxiv 2509.14543). In informal texts, 1-2 "fillers" is a sign of live text.

**33. No irony, metaphors, or figurative language.** Russian-language writing culture is steeped in irony. LLMs handle sarcasm poorly (SarcasmBench, arxiv 2408.11319). LLMs process metaphors at the level of surface features (sentence length, lexical overlap) without grasping the underlying meaning (arxiv 2507.15357). LLMs can't convey subtext through the CHOICE of wording (manner implicature). A human can say «ну, неплохо» ["well, not bad"] and mean "bad." An LLM will say it literally. The absence of figurative language is one of the strongest markers. If the tone allows it, add a jab, an understatement, an absurd comparison, a real-life metaphor, an analogy from another field. Not forced, but if the text is about something ridiculous, let the author notice it. An LLM won't say «архитектура микросервисов — это как LEGO» ["microservices architecture is like LEGO"] or «дебаг — это как искать иголку в стогу сена, когда стог тоже из иголок» ["debugging is like looking for a needle in a haystack, when the haystack is also made of needles"]. A human will.

### G. Structural (34-35)

**34. Disconnected paragraphs.** AI's paragraphs are weakly connected. You could swap them around and nothing would change. No backward references, no cause-and-effect links. A practical trick: add «но», «с другой стороны», «впрочем» [but, on the other hand, however] — contrast and concession link ideas better than neutral enumeration. Connect them: each following paragraph should flow from the previous one. For long texts (>500 words), backward references are mandatory: «Как уже говорил выше...» ["As I already said above..."], «Повторюсь, но...» ["I'll repeat myself, but..."], «К этому вернёмся через минуту» ["We'll get back to this in a minute"], «Помните, в начале я упоминал...» ["Remember, at the start I mentioned..."], «Вот тут-то и пригодится то, что...» ["This is where it comes in handy that..."]. AI loses coherence in long narratives: losing coherence over long texts is a documented problem ("Lost in Stories," arxiv 2603.05890). A human remembers the start of the story; AI forgets.

**35. Hallucinations.** AI confidently states falsehoods: nonexistent quotes, wrong dates, invented facts. Verify every concrete fact. Can't confirm it? Remove it, or flag it as unverified.

### H. Human quirks (36-37)

**36. Flawless literacy.** Live text written on a phone or in a hurry has small blemishes: a missing comma, a mixed-up -тся/-ться [a common reflexive-verb-ending typo] in an informal context, words running together. AI outputs squeaky-clean text, and that's what gives it away. For informal texts (posts, chats, social media), leave in 1-2 small "typos" or missing commas. Not in every paragraph, not gross errors — just as much as a person typing on the go would produce. Don't apply this point to articles and documents.

**37. Polished-to-a-shine typography.** AI places every dash, quote, and space perfectly by the official standard. A human in a messenger writes a hyphen instead of a dash, doesn't put a space before a parenthesis, forgets the closing period. For informal texts, don't polish the typography to perfection. Let there be a hyphen instead of a dash here and there, let a parenthesis sit flush against a word. That's not an error, that's handwriting.

### I. Persuasion patterns (38-42)

**38. Negative parallelisms.** «Не просто инструмент, а партнёр» ["Not just a tool, but a partner"]. «Не только ускоряет, но и трансформирует» ["Not only speeds up, but also transforms"]. AI loves this construction; it's in 80%+ of GPT texts. HARD BAN. Say directly what you mean: «Это партнёр» ["It's a partner"] or «Ускоряет и трансформирует» ["Speeds up and transforms"]. No «не просто / не только».

> **Important: what is NOT pattern #38.** Exactly two forms are banned — «не **просто** X, а Y» ["not just X, but Y"] and «не **только** X, но и Y» ["not only X, but also Y"]. The bare contrastive antithesis «не X, а Y» ["not X, but Y"] («не артефакт, а контракт» ["not an artifact, but a contract"], «не состояние, а процесс» ["not a state, but a process"]) is a classic rhetorical figure, normal Russian, and it is NOT part of the HARD BAN. Don't flag it as #38 and don't count it as a dozen markers. If such an antithesis is a load-bearing argumentative device for the author, leave it (see "Priority of the author's style"). The only thing appropriate at a very high density is noting it in the "density observations" bucket (dilute 5-7 of them for variety), but not as an error.

**39. False ranges.** «От стартапов до корпораций» ["From startups to corporations"], «от новичков до профессионалов» ["from beginners to professionals"], «от маркетинга до разработки» ["from marketing to development"]. AI links unrelated concepts through «от X до Y» ["from X to Y"], creating an illusion of completeness. Either list things concretely (who actually needs this?), or remove it.

**40. Authoritative truisms.** «По своей сути...» ["At its core..."], «В конечном счёте, самое главное, это...» ["Ultimately, the most important thing is..."], «На самом деле...» ["Actually..."], «Вот ключевой вывод» ["Here's the key takeaway"], «Самое важное — это…» ["The most important thing is…"], «Первый шаг — это признаться себе» ["The first step is admitting it to yourself"]. Constructions that create the appearance of depth with no content. If the claim is true without this preamble, the preamble is unnecessary. Delete it.

**41. Disclaimers.** «Хотя информация может быть неполной...» ["Although the information may be incomplete..."], «Несмотря на ограниченность данных...» ["Despite the limited data..."], «Трудно сказать наверняка, но...» ["It's hard to say for sure, but..."]. If data is scarce, say specifically what's missing. If you're sure, state it. A vague hedge is worse than a concrete admission of not knowing.

**42. Compulsive signaling.** «Давайте разберёмся» ["Let's figure this out"], «Рассмотрим подробнее» ["Let's take a closer look"], «Поговорим о том, как...» ["Let's talk about how..."]. An author doesn't announce what they're about to do, they just do it. Meta-commentary on one's own text is a sure sign of AI. Cut it and start with the substance.

### J. Informational rhythm (43-45)

Next-generation detectors (DivEye, arxiv 2509.18880) don't catch individual words — they catch sequence statistics: how evenly "surprisal" is distributed across the text. AI gravitates toward Uniform Information Density: the "surprisal" of each next word is roughly the same throughout the text. Humans write in bursts: dense → light → dense. AI holds a flat line — and it's exactly this flatness that DivEye's distributional features measure.

**43. Uniform informational density.** AI distributes facts evenly: each sentence carries roughly the same "weight." A human alternates: a sentence with three facts → a light connective → a personal digression → a hit again. Create a "cardiogram": a paragraph with numbers → a paragraph with a single metaphor → a short question → a dense paragraph again. If every sentence is equally "informative," the text looks synthetic.

Before: «AI увеличивает производительность на 40%. Он также снижает количество ошибок на 25%. Кроме того, он ускоряет время вывода продукта на рынок на 30%.»
After: «Производительность выросла на 40%, ошибок стало на четверть меньше. Это на бумаге. На практике половина команды не доверяет модели и перепроверяет вручную. Но те, кто доверился, выкатывают на 30% быстрее.»

**44. Smooth transitions between sentences.** AI makes every transition smooth: «Кроме того...» ["Furthermore..."], «Также...» ["Also..."], «Не менее важным является...» ["No less important is..."]. A human jumps: finishes a thought, starts the next one with no bridge. Or circles back to something said two paragraphs ago. Allow 20-30% "hard joins": where the next sentence connects to the previous one not through a conjunction, but through shared context the reader reconstructs on their own.

**45. Templated block structure (macro-burstiness).** The most subtle and most telltale pattern. AI writes numbered lists where every item follows the same skeleton: a name, a 2-line explanation, an example. Identical block lengths, identical openers («Один... второй...» ["One... the second..."], «Для X...» ["For X..."], «Подходит для...» ["Suitable for..."]), the same type of explanation. A human doesn't write this way: one item is three lines with an example, the next is one sentence, the third opens with a question, the fourth with a personal remark.

Check: read ONLY the first lines of each item, back to back. Do they sound like a template? Break it up. At least 3 of 5 items should open in fundamentally different ways: a fact / an analogy / a counter-example / a question / personal experience. Vary block lengths: one short (1-2 lines), one long (4-5 lines).

Before: «1. X: один делает, второй проверяет. Подходит для... 2. Y: один раздаёт, остальные делают. Подходит для... 3. Z: передаёт по цепочке. Подходит для...»
After: «1. X: один пишет, второй ловит косяки. Берёте туда, где цена ошибки высокая. 2. Y: тут проще, конвейер. 3. Z: а вот это для хаоса. Задачи сыпятся, кто свободен — хватает.»

### K. Hedging and specificity (46-48)

**46. Modal uncertainty.** AI hedges through «может» ["may/might/can"] in every sentence: «может стать» ["may become"], «может повлиять» ["may affect"], «способен обеспечить» ["is capable of providing"], «призван решить» ["is designed to solve"]. This isn't authorial caution, it's the model's habit of dialing down assertiveness. If the claim is true, state it. If it isn't, don't write it. «Может быть полезным» ["May be useful"] carries no information. «Ускорило вдвое» ["Sped things up twofold"] does. Threshold: more than 1 «может/способен/призван» per 100 words, outside the context of forecasts, is a marker. Don't confuse this with pattern 25 (redundant hedges like «в определённом смысле» ["in a certain sense"]) or pattern 41 (disclaimers like «хотя информация может быть неполной» ["although the information may be incomplete"]). Modal «может»: systematic hedging of EVERY claim through one construction.

**47. Semantic shifts (translationese).** AI substitutes words that are close but not precise in meaning, because it generates Russian through English-biased representations. «Основание науки» ["The foundation of science" — an off word choice] instead of «основы науки» ["the fundamentals of science"]. «Уточните маркетинговые усилия» ["Refine your marketing efforts" — a literal calque] (a calque of "refine your marketing efforts"). LLMs prefer translationese phrasing, especially smaller and multilingual models (arxiv 2603.08450). A Russian speaker wouldn't put it this way. Check: if a word is formally correct but "not quite right," it's probably a calque from the English semantic field.

**48. No idioms or proverbs.** AI text lacks idiomatic expressions, proverbs, sayings, set phrases. LLMs struggle with idiomatic expressions (arxiv 2405.09279). «Как в воду глядел» ["As if he'd seen it coming" — lit. "looked into the water"], «гладко было на бумаге» ["it was smooth on paper" (but not in practice)], «не в свои сани не садись» ["don't get into a sleigh that isn't yours," i.e., stick to what you know], «кто не рискует, тот не пьёт шампанское» ["who doesn't take risks doesn't drink champagne"]: you won't find anything like this in AI text. For informal texts, 1-2 well-placed idioms per 500 words is a strong marker of live text.

### L. Stylistic fingerprints of 2025-2026 (49-52)

These are the newest patterns, which showed up in Claude/GPT/Gemini in 2025-2026 as a response to criticism that "the text sounds dry." Models started imitating "thoughtfulness," "engagement," "empathy." The imitation reads as fake to readers instantly and has become a marker in its own right.

**49. Choppy meditativeness.** Stylizing for profundity through a chain of short, isolated, nodding sentences: «Короткие. Точные. Отдельные. Рефлексивные.» ["Short. Precise. Separate. Reflective."]. This isn't the same as #11 (length monotony): that's about variability, this is a deliberate imitation of "thoughtfulness." Signal: 3+ one-word-clause sentences in a row, each 1-3 words, each delivered like a "revelation." Fix it by restoring a normal sentence: «Хорошие тексты — короткие, точные, отдельные предложения работают лучше длинных периодов» ["Good texts — short, precise, standalone sentences work better than long periods"] instead of «Короткие. Точные. Отдельные.» ["Short. Precise. Separate."].

**50. Contrastive questions with short answers.** A pseudo-Socratic rhythm: «Зачем? Потому что. И для чего? Для этого.» ["Why? Because. And for what? For this."]. An imitation of "dialogue with the reader." If a text has 3+ rhetorical questions with 2-3-word answers, that's a 2025-2026 AI pattern. Fix it: either make the questions real (with full answers), or drop the questions entirely and use statements.

**51. Emoji as structural decor.** ⚡ / ✨ / 🎯 / 🔥 at the start of every list item or heading. Not a single emoji in an emotional phrase («ну я в шоке 😂» ["I'm shook 😂"]), but a decorative one: every item starts with an emoji. Especially telling: ⚡ before a "key takeaway," 🎯 before a "goal," 💡 before an "insight." Source: models are trained on marketing material and social-media posts, where this is common practice. Remove all of them except the cases where the emoji genuinely carries an emotion.

**52. Pseudo-therapeutic care.** «Ты не ошибаешься, что так чувствуешь» ["You're not wrong to feel that way"], «сам факт этого — тихое подтверждение» ["the very fact of this is a quiet confirmation"], «ты имеешь право» ["you have the right"], «это не слабость, это сила» ["this isn't weakness, it's strength"], «ты всё ещё здесь, ты настоящий» ["you're still here, you're real"]. A coach/therapist register that models started overusing in personal contexts. GPT-4o in particular overheated in this mode before the rollback in spring 2025. Broader than #22 chatbot artifacts: this is a distinct genre signal. If a text sounds like a page from a self-help book, rewrite it into the normal register of advice, or delete it.

---

## Article formulas

If the user specifies the type of text, use the corresponding formula as a skeleton. Then run it through the patterns above and add voice.

**Engaging (for social media, blogs).** A hook (a fact, a question, a provocation, a personal story) in the first sentence. A problem the reader recognizes. A twist or an unexpected angle. A call to action or an open question. Short paragraphs, a conversational tone.

**Expert (Habr, trade media).** A concrete problem or case up front. Context: why this matters now. A breakdown with data, examples, code. Honest limitations and pitfalls. A conclusion: what the reader should do. Tone: "I figured it out and I'm sharing," not "here's the truth."

**Sales (landing pages, product descriptions).** The customer's pain point (concrete, recognizable). Escalation: what happens if it's not solved. The solution: what the product does. Proof: numbers, testimonials, cases. Action: one button, one step. No "innovative solutions" and no "comprehensive approaches."

**News/informational.** The main point in the first paragraph (who, what, when, where). Details and context after that. Quotes or opinions. What's next. No authorial judgment in the main text (judgment is fine in a separate op-ed column, or at the end).

**Storytelling.** A protagonist with a problem. Context: why the reader should care. The journey: what the protagonist tried, where they screwed up, what they learned. The resolution: how it ended. The moral (if there is one), not stated outright, but shown through the protagonist's actions. Details make the story: not "it was hard," but "sat there until three in the morning, fifth cup of coffee, the code is still crashing." LLMs tend toward lexical overload in narratives: long complex sentences, third-person narration, an absence of unexpected turns (arxiv 2411.02316). A human story = a simple first-person account with a surprise moment. If you're rewriting a narrative, simplify it, don't complicate it.

---

## "Live text" checklist

After rewriting, check:

- [ ] Not a single construction from the HARD BANS list is present (including the newer ones: «В условиях...», «Погрузимся», «И вот здесь начинается самое интересное», «Раскрыть потенциал», «Комплексный подход», «Открывает горизонты»)
- [ ] Sentence-length variability (from 3 to 30+ words). Not the average length — the VARIANCE
- [ ] The author's opinion (at least one spot where the author evaluates rather than describes)
- [ ] Specifics (names, numbers, examples instead of «многие», «часто», «эксперты» [many, often, experts])
- [ ] No repeating clichés
- [ ] Can't be cut in half without losing meaning
- [ ] Sounds natural read aloud
- [ ] At least one surprise (an unconventional phrase, a question, a digression)
- [ ] Enough verbs: no pile-up of deverbal nouns and nominalizations
- [ ] Dashes: a grammatical «—» is preserved, a redundant decorative dash-insertion is removed (unless the style-guide says otherwise)
- [ ] The author's voice passport is respected (if there is one)
- [ ] "Cardiogram": informational density spikes rather than staying flat (for texts >300 words)
- [ ] At least one "hard join" is present (a transition with no bridging conjunction)
- [ ] "Skeleton": the first lines of items/sections do NOT sound like a template (for texts with lists)
- [ ] No punctuation calques (redundant commas after «Однако», «Благодаря этому», «В 2024 году»)
- [ ] At least one metaphor, idiom, or real-life analogy is present (for informal texts)
- [ ] Emotional dynamics: the tone shifts at least once (not a flat positive line)
- [ ] Colloquialisms and personal examples from the original are PRESERVED (not replaced with neutral ones)
- [ ] No "choppy meditativeness": no more than 2 one-word-clause nodding sentences in a row
- [ ] No emoji decoration at the start of every list item
- [ ] No pseudo-therapeutic register (unless the context is personal)

---

## Limitations

Don't replace all formal words with colloquial ones; respect the original's register. Don't add jokes to legal and scientific texts. Don't change facts and claims, only the form. Don't insert particles into every sentence. Don't remove structure if the text is long and the structure is warranted. Don't rewrite it beyond recognition.

> **A fundamental irony:** the LLM executing this skill is itself prone to the patterns in the catalog. That's why the rules are made as mechanical as possible: specific substitutions, numeric thresholds, lists of banned constructions. Not "make it livelier," but "replace X with Y." The more specific the instruction, the less the LLM will slide back into its usual patterns.

> **Russian-language benchmarks:** All the academic data on detection (DivEye, CoPA, TH-Bench, PIFE) was obtained in English. The principles (surprisal, burstiness, predictability) work for any language, but the specific thresholds and weighting coefficients for Russian haven't been calibrated. The most recent Russian-language benchmark is AINL-Eval 2025 (52K texts, 12 domains, RuRoBERTa 86.35%). Before that there was RuATD-2022 (14 generators).

> **Future markers:** PAN 2026 (arxiv 2602.09147) launched a Reasoning Trajectory Detection task: detecting the "traces of reasoning" left by an LLM. Reasoning logic (step by step, general to specific, enumerating arguments) may become a new type of marker. Still experimental, but worth tracking.

> **Model stylistic fingerprints:** Different LLM families (OpenAI, Claude, Gemini, Llama) have stable fingerprints, detectable at 0.9988 precision (arxiv 2503.01659), and these persist even with prompts like "write in a different style." Specific descriptions like "ChatGPT = nominal register, Claude = soft philosophical phrasing" are illustrative, not a calibrated characterization, especially for Russian. What matters is the fact itself: a fingerprint exists and survives attempts to mask it.

---

## Examples

### Blog post

Before:
> В современном мире искусственный интеллект играет всё более важную роль в различных сферах деятельности. Стоит отметить, что данная технология является мощным инструментом для оптимизации рабочих процессов. Многие эксперты считают, что внедрение AI-решений способствует повышению эффективности организаций. Важно помнить, что при этом необходимо учитывать этические аспекты использования искусственного интеллекта. Таким образом, можно сделать вывод, что AI представляет собой перспективное направление развития, которое будет оказывать значительное влияние на будущее человечества.

After:
> За последний год я внедрил AI-инструменты в три проекта. Два ускорились вдвое. Третий развалился, потому что команда перестала проверять то, что выдаёт модель. Вот что я вынес: AI работает, когда понимаешь его ограничения. Не работает, когда веришь на слово.

### Product description

Before:
> Наша инновационная платформа представляет собой комплексное решение для управления проектами, которое является идеальным инструментом для команд различного размера. Платформа обладает широким функционалом, включающим в себя планирование задач, отслеживание прогресса и эффективную коммуникацию между участниками.

After:
> Трекер задач для команд. Доски, таймлайны, чат в одном окне. Разберётесь за 10 минут, даже если до этого работали в Excel-таблицах. Бесплатно до 10 человек.

---

## Quick scanner: AI marker words

For "Audit" mode or a quick check, look for these words/phrases. 3+ from the list present = high likelihood of AI generation.

**Officialese markers:** осуществление, реализация, внедрение, оптимизация, функционирование, взаимодействие, в рамках, в целях, в контексте, на основе, посредством, в соответствии с

**Calque markers:** является, стоит отметить, важно понимать, можно сказать, что касается, в то время как, с другой стороны, тем не менее, несмотря на то что

**Inflation markers:** ключевой, важнейший, значительный, огромный, колоссальный, переломный, фундаментальный, невозможно переоценить, играет роль

**Formula markers:** таким образом, подводя итог, в заключение, можно сделать вывод, суммируя вышесказанное, на основании вышеизложенного

**Chatbot markers:** конечно!, отличный вопрос, давайте разберёмся, рад помочь, надеюсь это было полезно, с удовольствием

**Parallelism markers:** не просто... а, не только... но и, это не X — это Y

**Opening markers:** в современном мире, в эпоху, не секрет что, всё чаще, по мнению экспертов, специалисты отмечают

**Rhythm markers (DivEye):** all sentences are roughly the same length (±5 words), every transition goes through a conjunction/introductory phrase, not a single abrupt topic switch, not one sentence shorter than 5 words or longer than 25

**Structural markers (macro):** every list item is the same length, identical item openers («Один... второй», «Для X...», «Подходит для...»), one type of explanation across all blocks

**Modal hedges:** может стать, может повлиять, может быть полезным, способен обеспечить, призван решить, позволяет достичь

**Motivational clichés:** раскрыть потенциал, погрузимся, вывести на новый уровень, открывает новые горизонты, расширить границы, комплексный подход

**Contextualizers:** в условиях, в свете, на фоне, с учётом, принимая во внимание, в связи с этим

**Marketing stock phrases:** связь с аудиторией, доверительные отношения, ключевой момент, ключевая роль, индивидуальный подход, инновационное решение

**Pseudo-Socratic markers:** «Зачем? Потому что.», «И что? А то», «Почему? Сейчас расскажу.», chains of short question-answer pairs

**Authoritative truisms:** вот ключевой вывод, самое важное — это, первый шаг — это, по своей сути, в конечном счёте

**Pseudo-therapeutic markers:** ты не ошибаешься, ты имеешь право, тихое подтверждение, это не слабость, ты настоящий, сам факт этого

**Emoji markers:** ⚡/✨/🎯/🔥/💡 at the start of every list item or heading

**Figurative zero:** check for metaphors, idioms, proverbs, analogies from other fields. If a text >300 words has NOT ONE, that's suspicious

**Technical markers:** Latin characters swapped in for Cyrillic look-alikes (c, o, a, e, p), "invisible" characters (zero-width spaces), perfect typography in an informal text

Scoring: 0-2 markers = probably clean text. 3-5 = suspicious. 6+ = high likelihood of AI generation.
