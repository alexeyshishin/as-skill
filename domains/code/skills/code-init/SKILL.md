---
name: code-init
description: Generate a project CLAUDE.md for Claude Code — detects build/test/lint commands and the directory layout, then writes a concise rules-and-commands reference. Always embeds four fixed house rules (always use /caveman, no code comments, answer in Russian, report only hash and subject after commit/push). With the code harness installed it also records which executing agents (backend / frontend / devops) /code-build must call for which files, and when agents update the Memory Bank. With the user's consent it also registers the installed code hooks (`test-gate`, `prettier-format`, `memory-bank-load`) in the project's `.claude/settings.json` via `scripts/register-hooks.sh` (never the global `~/.claude/settings.json`). With the user's consent it also sets up project-local ast-index (structural code search): project-scope plugin, `Bash(ast-index *)` permission, initial index. Use when the user wants to "create a CLAUDE.md", "init CLAUDE.md", "set up project rules for Claude", "/code-init".
---

# Skill: /code-init — generate a project CLAUDE.md

Goal: write a `CLAUDE.md` at the project root that is a **reference of commands and
processes**, not a wall of prohibitions. Best-practice backbone (ранжировано по тому, что
реально встречается в успешных проектах): commands → checks → architecture → git →
pitfalls. Keep the core ~90–120 lines (~6 KB); the code-harness block (loop, executors,
Memory Bank, models) is on top of that — keep it tight too. Avoid shouty `NEVER`/`MUST`
everywhere — state a rule once, clearly, with the criterion for when it applies and the
reason in a clause. Instructions for agents follow the same discipline: one trigger, one
definition of done, one owner per action.

## Invocation
`/code-init`

## Step 1. Guard against clobbering

Check for an existing `CLAUDE.md` at the project root.
- If it exists, **read it**, show the user a one-line summary of what's there, and ask:
  overwrite, merge the four house rules into it, or cancel. Do not overwrite silently.
- If `AGENTS.md` exists, plan to reference it with a leading `@AGENTS.md` import instead of
  duplicating its content — put only Claude-Code-specific guidance in `CLAUDE.md`.

## Step 2. Detect the project (cheap, a few reads)

Gather only what you can observe — never invent commands:
- **Commands & checks**: `package.json` scripts, `Makefile` targets, `go.mod`,
  `pyproject.toml`/`setup.cfg`, `Cargo.toml`, `justfile`, `Dockerfile`, CI configs. Extract
  the real build / run / test / lint / typecheck invocations.
- **Architecture**: top-level directories and what each holds (one line each). Don't map
  every file — just the map a newcomer needs.
- **Git**: `git log --oneline -5` for commit-message language/style; note the default branch.
- **Executors** (only when the code harness is present — `.claude/skills/code-plan` or the
  `code` domain): which of `.claude/agents/{backend,frontend,devops}.md` are installed, and
  which real paths in the repo fall into each agent's scope. Confirm every path with
  `ls`/Glob; write globs only for what exists.

  | Agent | Look for |
  |-------|----------|
  | `backend` | server source dirs (`cmd/`, `internal/`, `api/`, `server/`, `app/`, `src/` with server code), `migrations/`, server manifests (`go.mod`, `pyproject.toml`, `Cargo.toml`, `pom.xml`) |
  | `frontend` | `web/`, `frontend/`, `client/`, `ui/`, `src/components/`, `*.tsx`/`*.vue`/`*.svelte`, `public/`, a `package.json` with react/vue/svelte/next/vite |
  | `devops` | `.github/workflows/`, `.gitlab-ci.yml`, `Dockerfile*`, `docker-compose*.yml`, `helm/`, `charts/`, `k8s/`, `terraform/` |

  Layers with no installed agent (e.g. `ios/`, `android/`, `pubspec.yaml` — there is no
  `mobile` agent) are recorded as "no executor", not assigned to the nearest agent.
- **Memory Bank**: whether `.memory-bank/index.md` and `.claude/skills/memory-bank` exist.
- **Hooks** (only when the code harness is present): from the project root run
  `bash .claude/skills/code-init/scripts/register-hooks.sh --dry-run`. It writes nothing and
  prints one line per installed hook, `present` or `would-add`, then hook name and event
  (`present` also covers an entry in `.claude/settings.local.json`). No output → no hooks
  installed, nothing to register. Any `would-add` → ask the user once, before composing:
  register those hooks in the project's `.claude/settings.json`? Claude Code runs a hook
  only when a `settings.json` lists it — the installer just places the files. The answer
  drives Step 3 and Step 4. Exit 1 (no `jq`, `settings.json` unparseable) → relay the
  stderr message, treat as declined.

- **ast-index**: from the project root run
  `bash .claude/skills/code-init/scripts/setup-ast-index.sh --dry-run`. It writes nothing and
  prints one line per step (`marketplace`, `plugin`, `permission`, `index`), `present` or
  `would-add` / `would-build`. Any non-`present` line → ask the user once, together with the
  hook question: set up ast-index for this project (project-scope plugin, permission, index —
  first build can take minutes)? The answer drives Step 3 and Step 4. Exit 1 → relay the
  stderr message (no binary → it carries the brew hint), treat as declined.

If a category yields nothing, omit that section rather than writing a placeholder.

## Step 3. Compose CLAUDE.md

Fill this skeleton. Russian prose (per house rule 3); keep command names, paths, and tool
names verbatim. Order sections as below.

```markdown
# CLAUDE.md

<одна строка: что это за проект>.
<если есть AGENTS.md:> @AGENTS.md — общие правила ниже только дополняют его.

## Обязательные правила

1. **Всегда вызывай навык `/caveman`** для ответов — ультра-сжатый режим, экономит токены.
2. **Не пиши комментарии в коде.** Код должен быть самодокументируемым: понятные имена,
   маленькие функции. Исключение — комментарий, который требует линтер/формат файла.
3. **Отвечай на русском.** Технические термины и имена оставляй на английском.
4. **После коммита и пуша сообщай только хеш, subject и диапазон пуша.** Не пересказывай,
   каким правилам следовал, и не упоминай атрибуцию (`Co-Authored-By` и т. п.). Формат — строго:
   `<короткий хеш> <subject>`, затем `Пуш: <старый>..<новый> <ветка> -> <ветка>`. Без пуша
   вторая строка опускается; несколько коммитов — по строке на коммит, потом одна строка пуша.
   Отклонение или блокировка (хук, отказ пуша) — отдельной строкой после этих.

## Команды

<build / run / test / lint / typecheck — только реально найденные>

## Проверки перед завершением

<линтер + типы + тесты — что прогнать, прежде чем считать задачу готовой>

## Структура проекта

<карта каталогов, по строке на каталог>

## Рабочий цикл (agent loop)

Разработка идёт через четыре скилл-оркестратора. Каждый сам не правит код в основном
цикле — он порождает субагентов (`Task`) и сводит их вывод. Артефакты между этапами —
в `swarm-report/<slug>-*.md`.

    /code-plan "<фича>"
        → параллельно: code-planner (проектирует план) + code-skeptic (ищет, что
          сломает) → merge → swarm-report/<slug>-plan.md
    /code-build <slug>
        → читает affected_files из плана, роутит по scope → backend / frontend / devops
          пишут код и гоняют тесты → swarm-report/<slug>-build.md
    /code-review <slug>
        → code-reviewer, read-only, сверяет диф с планом → verdict: ship | rework
    /code-debug "<симптом>"
        → code-debugger: воспроизводит → лестница гипотез → минимальный фикс

Петля: упал тест или `review: rework` → `/code-debug` → снова `/code-build` →
`/code-review`. <если test-gate зарегистрирован:> Stop-хук `test-gate` не даёт объявить
«готово», пока тесты реально не прогнаны. Планировать до билда, ревьюить после;
`/code-debug` — только когда что-то сломалось, не для проектирования фич.
`verdict: ship` закрывает петлю: после него — пополнение Memory Bank (см. ниже).

## Исполнители для /code-build

`/code-build` сам код не пишет: берёт `affected_files` из плана, сопоставляет их с
таблицей и зовёт по одному агенту на зону. Файл, которого в таблице нет, — не гадать, а
спросить пользователя, кто им владеет.

| Агент | Зона | Файлы проекта |
|-------|------|---------------|
| `backend` | API, сервисы, бизнес-логика, модели и миграции, фоновые задачи | <найденные пути> |
| `frontend` | компоненты, шаблоны, стили, клиентское состояние и роутинг | <найденные пути> |
| `devops` | CI/CD, Dockerfile, манифесты, деплой-конфиг | <найденные пути> |

<только реально найденные строки; слой без агента — отдельной строкой: «нет исполнителя: <пути> — спросить пользователя»>

- Зон несколько — агенты стартуют параллельно, каждый правит только свою зону.
- Один слой меняет контракт другого (форма ответа API, env-переменная, порт) — сначала
  автор контракта (обычно `backend`), его `contract_notes` уходят зависимому агенту в
  промпт.
- Агент отвечает YAML: `status`, `changed_files`, `tests_result`. `status: blocked` или
  упавший тест — работа не готова; «готово» без процитированного вывода тестов не
  принимается.

## Memory Bank

`.memory-bank/` — долговременная память проекта. Агенты читают `.memory-bank/index.md` до
работы и пополняют его после — но только когда задача **полностью** выполнена, иначе в
память попадёт то, что завтра переделают.

Задача полностью выполнена, когда верно всё сразу:
1. шаги плана сделаны, acceptance criteria выполнены;
2. проверки из «Проверки перед завершением» прогнаны, вывод зелёный и процитирован;
3. `/code-review` дал `verdict: ship`;
4. нет открытых blockers, `rework` и `status: blocked`.

Когда все четыре пункта верны:
- Пишет один агент за раз, не параллельные исполнители: основной цикл поручает запись
  одному из них (или делает сам). В поручении явно указана зона `.memory-bank/` — это
  единственное исключение из «правь только свою зону».
- Запись идёт через навык `memory-bank`: он отсекает мелочь (нужен importance ≥ 2) и
  требует сначала прочитать уже записанное. Правки из `MEMORY_SUGGESTION` применяются
  сразу, без ожидания подтверждения.
- Пишем устойчивое: архитектурные решения, контракты API и схемы, конвенции, смену
  стека, статус задачи в `tasks/`. Не пишем: детали реализации, эксперименты, мелкие
  рефакторинги, секреты.
- Обновлять нечего — ответить `NO_MEMORY_UPDATE` с причиной в одну фразу. Молчание не
  считается ответом.

## Модели

Тяжёлое рассуждение (план, критика, ревью, дебаг) — на Opus; исполнение готового плана —
на Sonnet (быстрее и дешевле при достаточном качестве).

| Роль | Модель | Почему |
|------|--------|--------|
| code-planner | `claude-opus-4-8` | проектирование — самый дорогой по качеству этап |
| code-skeptic | `claude-opus-4-8` | adversarial-разбор, нужен сильный ризонинг |
| backend / frontend / devops (исполнители) | `claude-sonnet-5` | пишут по готовому плану — скорость и цена |
| code-reviewer | `claude-opus-4-8` | ловит тонкие баги в дифе |
| code-debugger | `claude-opus-4-8` | root-cause-анализ — сложное рассуждение |
| скилл-оркестратор (роутинг, merge) | `claude-sonnet-5` | координация без глубокого ризонинга |

Правь это под бюджет: если дорого — переводи ревью/дебаг на Sonnet; тривиальная механика
(слагификация, мелкие правки) спокойно живёт на `claude-haiku-4-5`.

## Поиск по коду

<только если ast-index доступен>
Сначала `ast-index`: `explore "<вопрос>"`, `usages`, `refs`, `outline <файл>` перед Read
файлов длиннее 500 строк. Grep — только для regex, строковых литералов, комментариев и
не-кода, либо когда ast-index вернул пусто. После pull/rebase — `ast-index update`.

## Git

<ветки, формат коммитов, язык сообщений — но коммит делает пользователь (см. правило 1)>

## Подводные камни

<ненавязчиво: сгенерированные файлы (правим источник), платные команды, нестандартное>
```

### Rules for filling it

- The four house rules in **Обязательные правила** are mandatory and go in **verbatim** —
  this is the whole point of the skill. Keep them first.
- **Рабочий цикл**, **Исполнители для /code-build**, **Memory Bank** and **Модели**
  describe the installed `code` dev-loop harness — include them when
  `.claude/skills/code-plan` (or the `code` domain) is present in the target project; drop
  all four if the project doesn't use the code harness. Keep the model ids exact
  (`claude-opus-4-8`, `claude-sonnet-5`, `claude-haiku-4-5`).
- **Рабочий цикл**, **Memory Bank**, **Модели** go in verbatim, except the `test-gate`
  sentence in **Рабочий цикл**: keep it only if the hook is registered (already, or by
  Step 4); hook missing or user declined → drop it, the gate would be a false promise.
  **Исполнители для /code-build** is the one project-specific block: fill the table from
  the Step 2 executor detection, one row per installed agent that owns at least one real
  path. Agent not installed or no matching files → no row. Keep the three bullets under
  the table as written.
- **Поиск по коду** goes in verbatim, only when the Step 2 dry-run printed all `present` or
  the user agreed to the Step 4 setup; otherwise drop it.
- If `.memory-bank/` or the `memory-bank` skill is missing, still write the Memory Bank
  section (the agents' first write bootstraps the directory) and say in the Step 5 report
  that the `core` domain is what provides the skill.
- Everything else is project-specific: include a section only if Step 2 found real content.
- No invented commands, no guessed paths, no empty `## TODO` sections.
- Note the latent tension between rules 1 and 3 in passing only if the user asks — the
  intent is caveman-style brevity expressed in Russian; don't bloat the file explaining it.

## Step 4. Write CLAUDE.md, register hooks, set up ast-index

Write `CLAUDE.md` to the project root.

Then, only if Step 2 printed a `would-add` line and the user said yes, from the project
root:

```bash
bash .claude/skills/code-init/scripts/register-hooks.sh
```

The script does the whole job — no hand edits to `settings.json`, no re-reading it
afterwards. It writes only the project's own `.claude/settings.json` (it refuses when the
project is `$HOME`, so the global file is never touched), appends the missing entries under
`hooks.<Event>` with the matcher each hook declares in its `# register: <Event> [<matcher>]`
header line, skips hooks already listed in `settings.json` or `settings.local.json`, and
leaves every other key and entry alone. Re-running is safe. A new hook dropped into
`domains/code/hooks/` needs only that header line — nothing changes here.

Its stdout (`added` / `present` per hook) is the registration result for Step 5. Exit 1 →
nothing was written; relay the stderr message.

Then, only if the Step 2 ast-index dry-run had a non-`present` line and the user said yes,
from the project root:

```bash
bash .claude/skills/code-init/scripts/setup-ast-index.sh
```

The script does the whole job, each step idempotent: `claude plugin marketplace add` +
`claude plugin install ast-index@ast-index-marketplace`, both `--scope project` (even if the
plugin is enabled globally — the project must not depend on the user's global config);
`"Bash(ast-index *)"` merged into `permissions.allow` of the project's `.claude/settings.json`;
`ast-index rebuild` when no index exists, else `ast-index update`. It refuses when the project
is `$HOME`. The plugin ships its own hooks — add no ast-index hooks of your own.

Its stdout (`added` / `present` / `built` / `updated` per step, then the raw `ast-index stats`)
is the result for Step 5. Exit 1 → relay the stderr message; steps before the failure stay
applied, re-running is safe.

## Step 5. Report

3–5 lines: path written, which sections were populated from detection, which executors
`/code-build` will call and for which paths, which hooks the script reported (`added` /
`present`) or why none (declined / not installed / script error) — `test-gate` by name,
since it decides the **Рабочий цикл** sentence — the ast-index script's step lines and file /
symbol counts from its stats (or why skipped: declined / script error), plus a note if the
plugin is also enabled globally, and anything you couldn't detect (so the user can fill it
in). Don't re-print the whole file.

## What not to do

- Don't run `git commit` or `git add` — this skill writes `CLAUDE.md` and, with consent,
  hook and ast-index entries in the project's `.claude/settings.json` through the scripts;
  nothing else.
- Don't register hooks or set up ast-index without the user's yes, don't edit `settings.json`
  by hand (both scripts are idempotent and validate it), and never write to the global
  `~/.claude/settings.json`.
- Don't overwrite an existing `CLAUDE.md` without confirmation (Step 1).
- Don't fabricate build/test commands you didn't find in the repo.
- Don't list an executor whose `.claude/agents/<name>.md` isn't installed, and don't
  assign a layer with no agent (mobile, data, ML) to the nearest one.
- Don't pad with generic advice ("write clean code") — every line should be project-true.
- Don't drop or reword the four house rules.
