# Choosing a license

A recommendation, not legal advice. Ask the questions in one message, recommend one SPDX id plus at most one alternative, name the reason for each. The user decides.

## Questions

1. **May others ship your code inside closed-source products?**
   yes → permissive (Q2) · no → copyleft (Q3) · "I don't care at all" → Q4
2. **Permissive — does a patent grant matter?** (a company behind it, contributors who may hold patents)
   yes → `Apache-2.0` · no → `MIT` (`BSD-2-Clause` / `ISC` are equivalent in effect) · must forbid using your name to endorse forks → `BSD-3-Clause`
3. **Copyleft — how far should it reach?**
   - only the files you wrote, they may live inside a proprietary product → `MPL-2.0`
   - your library stays open, programs that link it may be closed → `LGPL-3.0` (`LGPL-2.1` for older ecosystems)
   - any program built on it stays open → `GPL-3.0`
   - also when it is only offered over a network (SaaS) → `AGPL-3.0`
4. **No conditions at all?**
   code → `Unlicense` or `0BSD` · data, docs, media → `CC0-1.0`
5. **What does the ecosystem expect?** It often settles a tie:

| ecosystem | convention |
|-----------|------------|
| Rust | `MIT OR Apache-2.0` (dual: `LICENSE-MIT` + `LICENSE-APACHE`) |
| Go, JavaScript/TypeScript, Ruby | `MIT` |
| Python | `MIT`, `Apache-2.0`, `BSD-3-Clause` |
| Java / Apache ecosystem, Kubernetes | `Apache-2.0` |
| C++ / Boost | `BSL-1.0` |
| Eclipse ecosystem | `EPL-2.0` |
| Linux kernel modules | `GPL-2.0` |
| WordPress plugins | `GPL-2.0-or-later` |

6. **Do you already include someone else's code?** Its license limits yours — see `third-party.md`. A known GPL dependency compiled in → the combined work is GPL.

## The bundled set

| SPDX | kind | key conditions | typical use |
|------|------|----------------|-------------|
| `MIT` | permissive | keep the notice | default for small libraries and tools |
| `ISC` | permissive | keep the notice | same as MIT, shorter text (npm default once) |
| `BSD-2-Clause` | permissive | keep the notice | same as MIT |
| `BSD-3-Clause` | permissive | + no endorsement using your name | academic, Python scientific stack |
| `Apache-2.0` | permissive | keep notice + NOTICE, mark changed files, explicit patent grant | company-backed projects, larger libraries |
| `BSL-1.0` | permissive | notice only in source, not in binaries | C++ header libraries |
| `MPL-2.0` | weak copyleft, file-level | modified MPL files stay MPL | libraries that want fixes back but allow proprietary use |
| `LGPL-2.1`, `LGPL-3.0` | weak copyleft, library-level | linking allowed, changes to the library stay open | libraries meant for wide reuse |
| `EPL-2.0` | weak copyleft, module-level | changed modules stay EPL | Eclipse / Java |
| `GPL-2.0`, `GPL-3.0` | strong copyleft | distributed derivatives stay GPL, source on distribution | applications, tools |
| `AGPL-3.0` | strong copyleft + network | same, and network users get the source | servers, SaaS |
| `Unlicense` | public domain dedication | none | throwaway code, snippets |
| `CC0-1.0` | public domain dedication | none | data, docs, media |

Not bundled but common — fetch them (SKILL.md, Step 4): `0BSD`, `Zlib`, `MIT-0`, `CC-BY-4.0` (docs/media, not code), `GPL-2.0-or-later` (the GPL-2.0 text plus an "or later" statement in README).

## When to stop and send the user to a lawyer

- changing the license of code already published under another one — every copyright holder must agree
- a contributor license agreement (CLA) or copyright assignment
- dual licensing for commercial sale (open + paid license)
- mixing GPL code with proprietary code
- anything about patents beyond "Apache-2.0 has an explicit grant"
- custom terms: "free for non-commercial use", "no use by company X" — that is not an open source license, none of the bundled texts can carry it
