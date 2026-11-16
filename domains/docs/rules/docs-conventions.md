# docs-conventions

Conventions for README files and licenses. Skills and agents in the `docs` domain reference this file as `docs-conventions`.

## README

### Files

- `README.md` — English, the source of truth. At the repository root; in a monorepo also at each package root.
- `README_RUS.md` — Russian, a full translation of `README.md`, next to it.
- The root README of a monorepo says what the repo is and lists packages in a table (name, one line, link to the package README); each package README follows the rules below.

### Section order

| # | section | content | required |
|---|---------|---------|----------|
| 1 | title + opening line | one sentence: what the project does and for whom, the nouns people search for up front — GitHub search and AI assistants quote it | yes |
| 2 | Why | the problem it solves, 2–3 sentences, not a manifesto | if the title line doesn't already say it |
| 3 | Install | prerequisites with versions, then the shortest working path — one command where possible | yes |
| 4 | Usage | a minimal example that produces visible output, with that output shown | yes |
| 5 | Examples | more commands, one screenshot or GIF for anything with a visible interface | optional |
| 6 | Documentation | links to deeper material: configuration reference, architecture, FAQ, benchmarks | if such material exists |
| 7 | Contributing | how to report issues and send changes; link `CONTRIBUTING.md` if present | optional |
| 8 | License | the SPDX id linked to the license file | yes |

Every section after Usage loses readers. Architecture, benchmarks and FAQs go to linked docs, not inline. A short, clear README beats a long unfocused one.

Project type adds sections in their natural place: configuration table (service, CLI), API overview (library), package table (monorepo), supported versions table (library, SDK).

### Format

- One `#` title, `##` sections, `###` subsections.
- Every fenced code block names its language: `bash`, `json`, `yaml`, `toml`, `go`, `text` for output.
- Commands are copy-paste ready: no `$` prompt, output in its own `text` block.
- Every command is either run during writing or taken verbatim from the project's own scripts, `Makefile`, CI or manifests. Never invented.
- Configuration options, CLI flags, environment variables, supported versions — tables. The *why* — prose. No concepts crammed into table cells.
- Badges: at most 3 (build status, package version, license), each useful to a stranger deciding whether to use the project. Under the title, never pushing the opening line down.
- Images: sparingly, with meaningful alt text ("Dashboard showing live query latency", not "screenshot").
- Relative links to repo files must resolve; anchor links must match a heading.
- Plain language. Define jargon or cut it. Don't assume the reader knows the environment.
- No marketing without a measured number: "blazing fast", "seamless", "revolutionary", "powerful" are cut.
- A table of contents only when there are more than ~8 `##` sections.

### Anti-patterns

- opening with build instructions or a generic "Welcome to my project!"
- a wall of badges above the description
- fenced code without a language
- options buried in paragraphs
- long code blocks without a sentence of explanation
- long sections after Usage
- commands that don't work or describe an old version
- images without alt text

### EN ↔ RU

- `README.md` is edited first; `README_RUS.md` is synced in the same change. RU never lags behind EN.
- Same structure: the same headings in the same order and levels, the same tables (rows, columns), the same images and badges.
- Code blocks carry the same code — commands, output, config, identifiers are not translated. Only comments inside them (`# …`, `// …`) and placeholder names (`<name>` → `<имя>`) may be.
- Technical terms, product names, CLI flags, identifiers stay in English (`Pod`, `goroutine`, `--dry-run`, `GitHub Actions`). The rest is natural Russian, not a word-for-word calque. Impersonal phrasing or «вы», consistent within the file.
- Alt text and table prose are translated; link targets are not, except internal anchors: GitHub derives anchors from heading text, so RU links point to RU headings (`#установка`).
- Links to other docs point to their Russian version (`X_RUS.md`) when one exists.
- Language switcher right after the title (or banner + title):
  - `README.md`: `🇷🇺 [Русская версия](README_RUS.md)`
  - `README_RUS.md`: `🇬🇧 [English version](README.md)`

## Licenses

### Text

- License text is copied verbatim from `references/licenses/<SPDX>.txt` of `docs-license`, or fetched from a canonical source (`gh api /licenses/<key>`, then SPDX license-list-data). Never typed or reconstructed from memory — a "nearly MIT" text is a different, unvetted license.
- Fill only the template's own placeholders `[year]` and `[fullname]` (MIT, BSD-2-Clause, BSD-3-Clause, ISC). Apache-2.0's appendix (`[yyyy] [name of copyright owner]`) is instructions for file headers — keep it as is.
- No edits to the terms. A custom clause ("non-commercial", "attribution in the UI") makes a different license: say so and stop, don't write it.
- Year — the year of first publication; a range `2023-2026` is fine. Don't silently change an existing year.
- Copyright holder — ask. Suggest `git config user.name`, or the organization if the work belongs to one. Never invent.

### Files

- One license → `LICENSE` at the root, no extension. Keep an existing `LICENSE.md` / `LICENSE.txt` / `COPYING` name.
- Dual licensing (`MIT OR Apache-2.0`) → `LICENSE-MIT` + `LICENSE-APACHE`; README states the expression.
- LGPL-3.0 → `COPYING.LESSER` (LGPL-3.0 text) + `COPYING` (GPL-3.0 text): LGPL-3.0 is a set of extra permissions on top of GPL-3.0 and is incomplete alone. LGPL-2.1 is self-contained.
- README `## License` / `## Лицензия`: the SPDX id linked to the file, e.g. `[MIT](LICENSE)`.
- A license already declared in a manifest (`package.json`, `pyproject.toml`, `Cargo.toml`, `*.gemspec`, `composer.json`) that differs from `LICENSE` is reported, never silently resolved.

### Third-party code

- Copied, vendored or adapted code keeps its original license and copyright notice.
- Its license text goes next to it: `LICENSE` inside the vendored directory, or `LICENSE-<upstream>` at the root of the component that contains it (as `domains/resume/LICENSE-ResumeSkills`). README names the source and its license.
- Apache-2.0 upstream with a `NOTICE` file: its notices are carried into the project's `NOTICE` (Apache-2.0 §4(d)).
- Before adopting copyleft code, check compatibility with the project license — see `references/third-party.md` of `docs-license`.

### Not legal advice

Skills recommend, the user decides. Every recommendation names its reason. Relicensing, CLAs, mixing copyleft with proprietary code, patent questions — say a lawyer should check.
