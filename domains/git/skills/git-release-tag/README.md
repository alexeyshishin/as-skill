# git-release-tag

Выпускает semver-релиз: считает следующую версию по коммитам, пишет release notes без нейрослопа, ставит annotated tag и создаёт GitHub release.

## Когда срабатывает

«Сделай релиз», «поставь тег», «подними версию», «опубликуй release на GitHub», «напиши release notes».

## Что делает

Проверяет, что HEAD на `main`, совпадает с `origin/main` и CI зелёный. Находит последний валидный `vX.Y.Z`, достижимый из HEAD, и по Conventional Commits с того момента выбирает bump: `!`/`BREAKING CHANGE` → MAJOR (MINOR до `v1.0.0`), `feat` → MINOR, `fix`/`perf`/`docs`/`refactor` → PATCH. Строит каркас notes: одна строка сути, потом секции Breaking changes / Features / Fixes / Performance / Other. Прогоняет notes через линтер, показывает план и ждёт подтверждения. Только после этого — `git tag -a`, `git push origin <tag>`, `gh release create`.

## Границы

Только обычный репо с одной версией. Монорепо (`<pkg>/vX.Y.Z`) и pre-release (`-rc.N`) не поддерживаются. Мусорные теги (`v.1.0.0`, `1.0.0`) игнорируются с предупреждением, но не удаляются. Ничего не пушит и не публикует без явного «да». Не двигает существующие теги. Не использует `--generate-notes`. Никаких эмодзи, восклицательных знаков и следов Claude в notes.

## Чем укомплектован

[`scripts/release.py`](scripts/release.py) — три режима:

```bash
python3 scripts/release.py next               # последний тег, следующая версия, коммиты по группам
python3 scripts/release.py next --bump minor  # ручной override уровня
python3 scripts/release.py notes > notes.md   # каркас release notes
python3 scripts/release.py lint notes.md      # линт на слоп и структуру
```

Коды выхода: 0 — ок, 1 — ошибка вызова/git, 2 — линтер нашёл ошибки (с `--strict` ещё и предупреждения), 3 — с последнего тега нечего релизить. `--json` — машиночитаемый вывод, `--repo <path>` — другой checkout. Без зависимостей, Python 3.8+.

Линтер ловит: продающие слова (`exciting`, `seamless`, `unlock`, `рады представить`), пустые фразы (`bug fixes and improvements`, `various fixes`), вступление «This release brings…», эмодзи, `!`, AI-атрибуцию, пустые секции, нестандартные заголовки, дубли и незаполненный placeholder. Текст в backticks не проверяет. Проверяет форму, а не правду: назвала ли первая строка главное изменение — решает модель.

[`references/release-notes.md`](references/release-notes.md) — таблица bump'ов и когда её переопределять, анатомия notes, примеры строки сути (плохо → хорошо), полный стоп-лист с объяснением.

Конвенция semver и тегов — в правиле [`git-conventions`](../../rules/git-conventions.md#semver-and-tags).
