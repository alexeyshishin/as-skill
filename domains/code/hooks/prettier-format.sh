#!/usr/bin/env bash
# register: PostToolUse Edit|Write|MultiEdit|Bash
set -Eeuo pipefail
trap 'exit 0' ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LIB="$(dirname "$0")/test-gate-lib.sh"
[ -f "$LIB" ] || exit 0
. "$LIB"
STDIN="$(cat)"

TRANSCRIPT="$(printf '%s' "$STDIN" | jq -r '.transcript_path // empty')"
[ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ] || exit 0

if ! compgen -G "$ROOT/.prettierrc*" >/dev/null && ! compgen -G "$ROOT/prettier.config.*" >/dev/null \
  && ! { [ -f "$ROOT/package.json" ] && jq -e '.prettier // empty' "$ROOT/package.json" >/dev/null 2>&1; }; then
  exit 0
fi
if [ -x "$ROOT/node_modules/.bin/prettier" ]; then PRETTIER="$ROOT/node_modules/.bin/prettier"
elif command -v prettier >/dev/null 2>&1; then PRETTIER="prettier"
else exit 0; fi

resolve_test_cmd "$ROOT"

TOOL="$(printf '%s' "$STDIN" | jq -r '.tool_name // empty')"
case "$TOOL" in
  Edit|Write|MultiEdit)
    FILES="$(printf '%s' "$STDIN" | jq -r '.tool_input.file_path // empty')" ;;
  Bash)
    [ -n "$TEST_PAT" ] || exit 0
    printf '%s' "$STDIN" | jq -r '.tool_input.command // empty' | grep -qiE "$TEST_PAT" || exit 0
    FILES="$(session_edited_files "$TRANSCRIPT" | sort -u)" ;;
  *) exit 0 ;;
esac

if [ "$TOOL" != "Bash" ] && [ -n "$TEST_CMD" ]; then
  [ "$(session_test_runs "$TRANSCRIPT" "$TEST_PAT")" -gt 0 ] || exit 0
fi

TARGETS=()
while IFS= read -r f; do
  [ -n "$f" ] && [ -f "$f" ] || continue
  case "$f" in "$ROOT"/*) ;; *) continue ;; esac
  printf '%s\n' "$f" | grep -qiE '\.(ts|tsx|js|jsx|mjs|cjs|json|css|scss|html|vue)$' || continue
  TARGETS+=("$f")
done <<EOF
$FILES
EOF
[ "${#TARGETS[@]}" -gt 0 ] || exit 0

CHANGED="$(cd "$ROOT" && "$PRETTIER" --write --list-different --ignore-unknown "${TARGETS[@]}" 2>/dev/null || true)"
[ -n "$CHANGED" ] || exit 0

MSG="prettier reformatted: $(printf '%s' "$CHANGED" | tr '\n' ' ')— re-read before editing them again."
jq -nc --arg c "$MSG" '{hookSpecificOutput:{hookEventName:"PostToolUse", additionalContext:$c}}'
exit 0
