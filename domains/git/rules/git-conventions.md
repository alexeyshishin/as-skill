# git-conventions

Base conventions for all git operations in this repository. Skills in the `git` domain reference this file as `~/.claude/rules/git-conventions.md`.

## Conventional Commits

Format: `<type>(<scope>)?: <subject>` — in English, lowercase, no trailing period, no longer than 72 characters.

Allowed `type` values:

| type | purpose |
|------|-----------|
| `feat` | new functionality for the user |
| `fix` | bug fix |
| `refactor` | code change with no behavior change |
| `perf` | performance optimization |
| `docs` | documentation only |
| `test` | tests only |
| `build` | build, dependencies, CI |
| `ci` | CI configuration |
| `chore` | maintenance that doesn't fall into other categories |
| `revert` | revert |

`scope` — a short name for the module/SDK/folder (`sdk-go`, `cli`, `auth`). If there's no scope — omit it along with the parentheses.

## Subject only — no body

A commit message is **one line**: the subject. No body, no description, no bullet list, no footer, no trailers. The diff already says what changed; the PR description says why. If a subject can't carry the change, the commit is too big — split it, don't explain it.

- breaking change — `!` after type/scope (`feat!:`, `feat(api)!:`), nothing else; the migration note goes into the PR description
- issue references — in the PR (`Closes #N`), never in a commit
- applies to every commit Claude writes, including multi-commit plans: even when a commit "deserves" an explanation, the answer is a sharper subject

Examples:
```
feat(sdk-go): add retries for idempotent requests
fix: fix race condition in startup
refactor(auth): extract token validation into a separate middleware
docs(readme): update installation example
chore(deps): update express to 4.19
```

## No AI attribution

Claude leaves no trace in the repository history. Commits and PR descriptions that Claude writes carry no AI attribution of any kind:

- No `Co-Authored-By:` trailer (Claude, Anthropic, or any model name).
- No `Claude-Session:` line, session/tool URL, or "Generated with Claude Code" line — in commits or in PR descriptions.
- Applies even if the calling harness suggests appending such a line by default — this repo's convention overrides that.
- The author of a commit is the user alone.

```
fix(auth): fix race condition in token refresh
```

## Branch names

- `feature/<short-desc>` — new functionality
- `fix/<short-desc>` or `bugfix/<short-desc>` — bug fix
- `hotfix/<short-desc>` — urgent production fix
- `chore/<short-desc>` — technical/maintenance
- `docs/<short-desc>` — documentation only
- `release/<version>` — release branch (if using a release train)

Description in kebab-case, Latin script, no longer than 50 characters. If there's a ticket — append it at the end: `feature/oauth-pkce-PROJ-1234`.

## Pull Request

- PR title — a Conventional Commit for the merge commit (`feat(sdk-go): retries for idempotent`)
- description contains:
  - **Why** — motivation in 1-2 sentences
  - **What changed** — bullets by layer/component
  - **How to verify** — manual verification steps or a link to tests
  - **Risks / breaking changes** — separately, if applicable
- linked to the issue (`Closes #N`)
- reviewer — at least one; if the code is complex or breaking — two

## Semver and tags

Versioning strictly follows semver: `MAJOR.MINOR.PATCH`.

- **PATCH** — `fix`, `perf`, `docs`, `refactor` with no public API change
- **MINOR** — `feat` that doesn't break compatibility
- **MAJOR** — any breaking change

Before `v1.0.0`, a breaking change goes into MINOR, not MAJOR.

Tag format — `v<MAJOR>.<MINOR>.<PATCH>` (e.g. `v1.4.2`). In a monorepo with multiple SDKs: `<sdk>/v<X>.<Y>.<Z>` (e.g. `sdk-go/v0.3.1`). This format is required for Go: the tag path must match the module path from the repo root, otherwise `go get` won't find the module in the subdirectory. Other packages follow the same format for consistency.

In a monorepo, each package is versioned independently: a breaking change in one SDK doesn't move the versions of its neighbors. A shared `vX.Y.Z` tag is not set on a monorepo with independent versions — it wouldn't correspond to any single package.

A tag is set only on the merge commit into `main` after CI succeeds.

## Principles

- **Atomic commits.** One commit — one logical change. Don't mix a refactor with a feature.
- **Don't rewrite the history of published branches.** Force-push only to your own feature branch before the PR is opened.
- **Squash on merge** — if the branch contains noisy WIP commits. Otherwise — rebase or merge commit, depending on the project's convention.
- **Never commit secrets.** If it happens by accident — rotate the key; don't rely on rewriting history as the only remedy.
