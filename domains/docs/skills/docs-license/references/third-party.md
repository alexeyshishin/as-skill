# Third-party attribution

Code you didn't write keeps its license. This file covers what to keep, where to put it, and which upstream licenses fit which project license. A summary of OSI/FSF guidance, not legal advice.

## What counts

| counts | doesn't |
|--------|---------|
| files or snippets copied in | dependencies installed by a package manager and not redistributed |
| a vendored directory (`vendor/`, `third_party/`, `external/`) | tools used only at build time |
| code adapted, translated or ported from another project | ideas, algorithms, APIs reimplemented from scratch |
| bundled fonts, icons, images, datasets | |

Redistributing bundled binaries (a static Go binary, a JS bundle, a Docker image) carries every bundled dependency's notice too. Generating that list is a job for `go-licenses`, `license-checker`, `cargo-about`, `pip-licenses` — name the tool, don't hand-write the list.

## Collect, per upstream

1. Source: URL, version or commit.
2. License: SPDX id from the upstream `LICENSE`, not from memory or a README badge.
3. Copyright line(s): from the upstream `LICENSE` or file headers.
4. The license text itself, verbatim:

```bash
gh api repos/<owner>/<repo>/license --jq .content | base64 -d > LICENSE-<Upstream>
```

No license file upstream → the code is all rights reserved: stop, tell the user it can't be used without the author's permission.

## Where it goes

| case | placement |
|------|-----------|
| vendored directory | keep its `LICENSE` inside the directory, untouched |
| adapted collection inside one component | `LICENSE-<Upstream>` at the component root (as `domains/resume/LICENSE-ResumeSkills`) |
| single copied file or snippet | `LICENSE-<Upstream>` at the repo root, plus a `NOTICE` entry naming the file path |
| several upstreams | `NOTICE` (or `THIRD_PARTY_NOTICES.md`) at the root: one block per upstream — name, URL, SPDX, copyright, path(s), license file |

README gets one line per upstream in a Credits / Acknowledgements section or in the License section: `Adapted from [<name>](<url>) (<SPDX>), see [LICENSE-<Upstream>](LICENSE-<Upstream>).` RU mirrors it.

Apache-2.0 upstream:
- an upstream `NOTICE` file → its text goes into your `NOTICE` (§4(d))
- files you modified carry a prominent statement that you changed them (§4(b)) — a `NOTICE` entry listing the modified paths is the usual way when the project keeps code free of comments

MPL-2.0 upstream: the copied files stay MPL-2.0 and keep their header; your other files are unaffected.

## Compatibility: can upstream code go into your project?

| upstream ↓ \ your project → | MIT / BSD / ISC | Apache-2.0 | MPL-2.0 | LGPL-3.0 | GPL-2.0-only | GPL-3.0 | AGPL-3.0 | proprietary |
|---|---|---|---|---|---|---|---|---|
| MIT, BSD, ISC, Zlib, BSL-1.0 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Apache-2.0 | ✓ part stays Apache | ✓ | ✓ | ✓ | ✗ | ✓ | ✓ | ✓ |
| MPL-2.0 | ✓ files stay MPL | ✓ files stay MPL | ✓ | ✓ | ✓¹ | ✓¹ | ✓¹ | ✓ files stay MPL |
| LGPL-2.1 / LGPL-3.0 | link only² | link only² | link only² | ✓ | ✓ (2.1) | ✓ | ✓ | link only² |
| GPL-2.0-only | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ | ✗ | ✗ |
| GPL-2.0-or-later | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ | ✓ | ✗ |
| GPL-3.0 | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓³ | ✗ |
| AGPL-3.0 | ✗ | ✗ | ✗ | ✗ | ✗ | ✓³ | ✓ | ✗ |
| CC-BY-NC, "non-commercial", no license | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

¹ unless the MPL files are marked "Incompatible With Secondary Licenses".
² dynamic linking with the library replaceable by the user; copying LGPL code into your files makes those files LGPL.
³ GPL-3.0 §13 / AGPL-3.0 §13 allow combining; each part keeps its own license.

✗ means: the combined work can't be distributed under your license. Options to offer the user — replace the code, ask the upstream author for a different license, or change the project license (lawyer).
