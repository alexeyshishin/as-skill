#!/bin/sh

as_skill_sourced=0
if [ -n "${ZSH_VERSION:-}" ]; then
    case "$ZSH_EVAL_CONTEXT" in
        *:file) as_skill_sourced=1 ;;
    esac
elif [ -n "${BASH_VERSION:-}" ]; then
    (return 0 2>/dev/null) && as_skill_sourced=1
else
    case "$0" in
        *install.sh) as_skill_sourced=0 ;;
        *) as_skill_sourced=1 ;;
    esac
fi

if [ ! -d domains ] || [ ! -d core/skills ]; then
    echo "install.sh: запустите из корня claude-harness (не нашёл domains/ и core/skills/ здесь)" >&2
    return 1 2>/dev/null || exit 1
fi

if ! command -v go >/dev/null 2>&1; then
    echo "install.sh: нужен go, не найден в PATH — https://go.dev/dl/ (на macOS: brew install go)" >&2
    return 1 2>/dev/null || exit 1
fi

echo "Собираю as-skill..."
if ! go build -o as-skill ./tools; then
    echo "install.sh: сборка не удалась" >&2
    return 1 2>/dev/null || exit 1
fi

as_skill_root="$(pwd)"
case ":$PATH:" in
    *":$as_skill_root:"*) ;;
    *) export PATH="$as_skill_root:$PATH" ;;
esac
unset as_skill_root

if [ "$as_skill_sourced" = 1 ]; then
    echo "Готово: as-skill доступен в этой оболочке ($(command -v as-skill))."
    echo
    as-skill list domains
else
    echo "Собрано: ./as-skill"
    echo "Это дочерний процесс — PATH не может передаться в родительскую оболочку."
    echo "Чтобы as-skill сразу заработал в ЭТОЙ оболочке — запустите: source install.sh"
fi
unset as_skill_sourced
