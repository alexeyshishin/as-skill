#!/usr/bin/env bash
# register: PostToolUse Edit|Write|MultiEdit|Bash
# platforms: claude,codex,cursor
set -Eeuo pipefail
trap 'exit 0' ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LIB="$(dirname "$0")/test-gate-lib.sh"
[ -f "$LIB" ] || exit 0
. "$LIB"
STDIN="$(cat)"

PLATFORM="$(detect_platform)"
SESS_DIR="$(sessions_dir)"
if [ "$PLATFORM" = claude ]; then
  TRANSCRIPT="$(printf '%s' "$STDIN" | jq -r '.transcript_path // empty')"
  [ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ] || exit 0
else
  normalize_payload "$STDIN"
fi

if ! compgen -G "$ROOT/.prettierrc*" >/dev/null && ! compgen -G "$ROOT/prettier.config.*" >/dev/null \
  && ! { [ -f "$ROOT/package.json" ] && jq -e '.prettier // empty' "$ROOT/package.json" >/dev/null 2>&1; }; then
  exit 0
fi
if [ -x "$ROOT/node_modules/.bin/prettier" ]; then PRETTIER="$ROOT/node_modules/.bin/prettier"
elif command -v prettier >/dev/null 2>&1; then PRETTIER="prettier"
else exit 0; fi

resolve_test_cmd "$ROOT"

if [ "$PLATFORM" = claude ]; then
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
else
  FILES="$N_FILES"
  TESTED=0
  if [ -z "$FILES" ] && [ -n "$N_CMD" ]; then
    [ -n "$TEST_PAT" ] || exit 0
    printf '%s\n' "$N_CMD" | grep -qiE "$TEST_PAT" || exit 0
    STATE="$(state_file "$SESS_DIR" "$N_SID")"
    FILES="$(state_edited_files "$STATE" | sort -u)"
    TESTED=1
  fi
  [ -n "$FILES" ] || exit 0
  if [ "$TESTED" = 0 ] && [ -n "$TEST_CMD" ]; then
    [ -n "$N_SID" ] || exit 0
    STATE="$(state_file "$SESS_DIR" "$N_SID")"
    [ "$(state_test_runs "$STATE")" -gt 0 ] || exit 0
  fi
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
[ "$PLATFORM" != cursor ] || exit 0
jq -nc --arg c "$MSG" '{hookSpecificOutput:{hookEventName:"PostToolUse", additionalContext:$c}}'
exit 0
