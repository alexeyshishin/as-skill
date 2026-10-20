---
name: code-init
description: Generate a project CLAUDE.md for Claude Code — detects build/test/lint commands and the directory layout, then writes a concise rules-and-commands reference. Always embeds four fixed house rules (no git commit, always use /caveman, no code comments, answer in Russian). Use when the user wants to "create a CLAUDE.md", "init CLAUDE.md", "set up project rules for Claude", "/code-init".
---

# Skill: /code-init — generate a project CLAUDE.md

Goal: write a `CLAUDE.md` at the project root that is a **reference of commands and
processes**, not a wall of prohibitions. Best-practice backbone (ранжировано по тому, что
реально встречается в успешных проектах): commands → checks → architecture → git →
pitfalls. Keep it ~90–120 lines (~6 KB). Avoid shouty `NEVER`/`MUST` everywhere — state a
rule once, clearly, with the criterion for when it applies.

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

If a category yields nothing, omit that section rather than writing a placeholder.

## Step 3. Compose CLAUDE.md

Fill this skeleton. Russian prose (per house rule 4); keep command names, paths, and tool
names verbatim. Order sections as below.

```markdown
# CLAUDE.md

<одна строка: что это за проект>.
<если есть AGENTS.md:> @AGENTS.md — общие правила ниже только дополняют его.

## Обязательные правила

1. **Не выполняй `git commit`.** Готовь изменения и формулируй сообщение, но сам коммит
   оставляй пользователю — он коммитит вручную.
2. **Всегда вызывай навык `/caveman`** для ответов — ультра-сжатый режим, экономит токены.
3. **Не пиши комментарии в коде.** Код должен быть самодокументируемым: понятные имена,
   маленькие функции. Исключение — комментарий, который требует линтер/формат файла.
4. **Отвечай на русском.** Технические термины и имена оставляй на английском.

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
`/code-review`. Stop-хук `test-gate` не даёт объявить «готово», пока тесты реально не
прогнаны. Планировать до билда, ревьюить после; `/code-debug` — только когда что-то
сломалось, не для проектирования фич.

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

## Git

<ветки, формат коммитов, язык сообщений — но коммит делает пользователь (см. правило 1)>

## Подводные камни

<ненавязчиво: сгенерированные файлы (правим источник), платные команды, нестандартное>
```

### Rules for filling it

- The four house rules in **Обязательные правила** are mandatory and go in **verbatim** —
  this is the whole point of the skill. Keep them first.
- **Рабочий цикл** and **Модели** describe the installed `code` dev-loop harness — include
  them verbatim when `.claude/skills/code-plan` (or the `code` domain) is present in the
  target project; drop both sections if the project doesn't use the code harness. Keep the
  model ids exact (`claude-opus-4-8`, `claude-sonnet-5`, `claude-haiku-4-5`).
- Everything else is project-specific: include a section only if Step 2 found real content.
- No invented commands, no guessed paths, no empty `## TODO` sections.
- Note the latent tension between rules 2 and 4 in passing only if the user asks — the
  intent is caveman-style brevity expressed in Russian; don't bloat the file explaining it.

## Step 4. Write and report

Write `CLAUDE.md` to the project root. Then report in 3–4 lines: path written, which
sections were populated from detection, and anything you couldn't detect (so the user can
fill it in). Don't re-print the whole file.

## What not to do

- Don't run `git commit` or `git add` — this skill only writes a file.
- Don't overwrite an existing `CLAUDE.md` without confirmation (Step 1).
- Don't fabricate build/test commands you didn't find in the repo.
- Don't pad with generic advice ("write clean code") — every line should be project-true.
- Don't drop or reword the four house rules.
