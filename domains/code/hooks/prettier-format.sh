#!/usr/bin/env bash
# PostToolUse hook — run prettier on code, but only once test-gate is satisfied.
# register: PostToolUse Edit|Write|MultiEdit|Bash
# Stop (test-gate) fires at the end of the turn, after every PostToolUse, so this hook can't wait for
# it: it re-checks the same condition via test-gate-lib.sh — a test command already ran this session,
# or the project has none (the gate then has nothing to enforce).
#   Edit/Write/MultiEdit -> format that one file
#   Bash running the test command -> format every code file edited this session
# No-ops without jq, prettier, or a prettier config in the project. Never blocks; fail-open on any error.
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

# Prettier needs a config of its own — never reformat a project with defaults it didn't ask for.
if ! compgen -G "$ROOT/.prettierrc*" >/dev/null && ! compgen -G "$ROOT/prettier.config.*" >/dev/null \
  && ! { [ -f "$ROOT/package.json" ] && jq -e '.prettier // empty' "$ROOT/package.json" >/dev/null 2>&1; }; then
  exit 0
fi
if [ -x "$ROOT/node_modules/.bin/prettier" ]; then PRETTIER="$ROOT/node_modules/.bin/prettier"
elif command -v prettier >/dev/null 2>&1; then PRETTIER="prettier"
else exit 0; fi

resolve_test_cmd "$ROOT"

# Which files, and is this tool call one we react to?
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

# test-gate satisfied? (the Bash branch above already proved a test command just ran)
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

# --write --list-different: rewrites what differs, prints only those paths.
CHANGED="$(cd "$ROOT" && "$PRETTIER" --write --list-different --ignore-unknown "${TARGETS[@]}" 2>/dev/null || true)"
[ -n "$CHANGED" ] || exit 0

MSG="prettier reformatted: $(printf '%s' "$CHANGED" | tr '\n' ' ')— re-read before editing them again."
jq -nc --arg c "$MSG" '{hookSpecificOutput:{hookEventName:"PostToolUse", additionalContext:$c}}'
exit 0
