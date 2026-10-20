# git-conventional-commits

Пишет commit-сообщение по Conventional Commits для текущих изменений.

## Когда срабатывает

«Сделать коммит», «закоммитить изменения», «сформулируй commit-message», «commit по конвенции».

## Что делает

Читает staged-diff (а если staged пуст — unstaged), определяет type и scope по содержимому изменений, формулирует subject в императиве. Тело добавляет только когда «зачем» не очевидно из subject. Показывает готовое сообщение и ждёт подтверждения перед `git commit`.

## Границы

Не коммитит без подтверждения. Не пишет `chore: update` — пустое сообщение хуже отсутствующего. В футер каждого коммита добавляет `Co-Authored-By: Claude <noreply@anthropic.com>` — формат и обоснование в [`git-conventions`](../../rules/git-conventions.md#ai-attribution).

Конвенция целиком — в правиле [`git-conventions`](../../rules/git-conventions.md).