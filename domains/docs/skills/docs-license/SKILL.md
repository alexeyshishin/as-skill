---
name: docs-license
description: >
  Picks and writes a project license: a short questionnaire → an SPDX
  recommendation with reasons, then the LICENSE file copied verbatim from
  bundled canonical texts (or fetched via gh / SPDX for others) with only the
  year and holder filled in; also third-party attribution for copied, vendored
  or adapted code (LICENSE-<upstream>, NOTICE, README credit). Use when the user
  wants to "add a license", "choose a license", "which license should I use",
  "create LICENSE", "attribute third-party code", "выбери лицензию", "добавь
  лицензию", "оформи атрибуцию". A review of existing license files is
  docs-license-reviewer.
---

# docs-license — license choice, LICENSE file, third-party attribution

Goal: a license file that is exactly the canonical text, the right holder and year in it, a README that names it, and every borrowed piece of code credited under its own license.

Before starting, read `references/rules/docs-conventions.md` — section Licenses.

Assets:
- [`references/licenses/`](references/licenses/) — 15 canonical texts, verbatim from GitHub's license API (`<SPDX>.txt`; list: `python3 scripts/check-license.py --list`).
- [`references/choose-license.md`](references/choose-license.md) — the questionnaire, the bundled set, ecosystem conventions, when to send the user to a lawyer.
- [`references/third-party.md`](references/third-party.md) — what counts as third-party, where its license goes, compatibility table.
- [`scripts/check-license.py`](scripts/check-license.py) — verbatim check, placeholders, README and manifest consistency (Step 6).

**The model never types license text.** Texts are copied with `cp` or downloaded; only the placeholders are edited.

## Step 1. Inspect what exists

```bash
ls | grep -iE '^(licen[cs]e|copying|notice)'
python3 scripts/check-license.py --no-readme <license files>
```

Also: the `license` field in manifests (`package.json`, `pyproject.toml`, `Cargo.toml`, `*.gemspec`, `composer.json`), the License section of `README.md` / `README_RUS.md`, vendored directories (`vendor/`, `third_party/`, `external/`).

A license already exists and the user asks for another → that's relicensing: say that code already published under the old license stays under it, and changing it needs every copyright holder's consent (`choose-license.md`, lawyer list). Continue only on explicit confirmation.

## Step 2. Pick the mode

| request | mode |
|---------|------|
| "which license", no license yet and no choice made | **choose** → Step 3, then Step 4 |
| the license is named | **write** → Step 4 |
| borrowed code to credit | **third-party** → Step 5 |

## Step 3. Choose

Ask the questions from `references/choose-license.md` in one message, skipping what the repo already answers (ecosystem from manifests, a company in `git log` authors, a server → AGPL question matters). Recommend one SPDX id and at most one alternative, each with its reason in one line. The user decides.

## Step 4. Write

Ask for the copyright holder (suggest `git config user.name`, or the organization) and confirm the year (default: the current one; for an existing project — the year of the first commit, `git log --reverse --format=%cs | head -1`). Never invent a holder.

Bundled id:

```bash
cp references/licenses/<SPDX>.txt <project>/LICENSE
```

Then replace only `[year]` and `[fullname]` (MIT, BSD-2-Clause, BSD-3-Clause, ISC). Other bundled texts have nothing to fill — Apache-2.0's `[yyyy]` appendix stays as is.

Not bundled — download to a scratch file, copy, keep the scratch copy as the canonical for the check:

```bash
gh api /licenses/<key> --jq .body > "$CANON"
curl -fsSL https://raw.githubusercontent.com/spdx/license-list-data/main/text/<SPDX>.txt -o "$CANON"
```

`<key>` is the lowercase GitHub key (`0bsd`, `zlib`, `cc-by-4.0`); the SPDX source is the fallback when `gh` is missing or the key is unknown. SPDX texts use their own placeholders (`<year>`, `<copyright holders>`) — read the copyright line and pass them to the checker with `--placeholder`. Neither source works → stop and say so; don't reconstruct the text.

File layout per `docs-conventions` → Files: dual → `LICENSE-MIT` + `LICENSE-APACHE`; LGPL-3.0 → `COPYING.LESSER` + `COPYING` (GPL-3.0); keep an existing file name.

README: set the License section of `README.md` — `## License` + `[<SPDX>](LICENSE)` — and `## Лицензия` in `README_RUS.md` if it exists. Touch nothing else in the README; anything more is `docs-readme`.

A manifest declaring a different license → report it, don't edit the manifest unless the user asks.

## Step 5. Third-party

Follow `references/third-party.md`: collect source, license, copyright per upstream; fetch the upstream license verbatim; place it; add the `NOTICE` entry and the README credit line (EN and RU). Check the compatibility table against the project license — a ✗ is reported with the options listed there, never papered over.

## Step 6. Check

```bash
python3 scripts/check-license.py
python3 scripts/check-license.py LICENSE --canonical "$CANON" --placeholder "<year>" --placeholder "<copyright holders>"
python3 scripts/check-license.py vendor/<lib>/LICENSE --no-readme --no-manifest
```

The first form finds `LICENSE*` / `COPYING*` in the current directory, detects each against the bundled texts, checks README EN + RU and manifests. Exit 0 — proceed. Exit 2 — fix and re-run; `text-diff` means the text was altered: recopy, don't patch words. `--strict` makes warnings fail too.

## Step 7. Report

Files written, SPDX id, holder and year, README lines changed, third-party entries, anything reported but not changed (manifest mismatch, incompatible code). If the case touched the lawyer list in `choose-license.md` — say so once, plainly.

## What not to do

- don't type, paraphrase, shorten or "clean up" a license text
- don't invent the holder or backdate the year
- don't add or remove terms; a custom clause is a different license — stop and say so
- don't relicense without explicit confirmation and the relicensing warning
- don't edit manifests or add SPDX headers to source files unless asked
- don't treat this as legal advice — recommend, the user decides
- don't commit or push
