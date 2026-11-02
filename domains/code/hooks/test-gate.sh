#!/usr/bin/env bash
# register: Stop
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
[ -n "$SESS_DIR" ] || exit 0

[ "$(printf '%s' "$STDIN" | jq -r '.stop_hook_active // false')" = "true" ] && exit 0

if [ "$PLATFORM" = claude ]; then
  SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
  TRANSCRIPT="$(printf '%s' "$STDIN" | jq -r '.transcript_path // empty')"
  [ -n "$SESSION_ID" ] || exit 0
  [ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ] || exit 0
  EDITED="$(session_edited_files "$TRANSCRIPT")"
else
  normalize_payload "$STDIN"
  SESSION_ID="$N_SID"
  [ -n "$SESSION_ID" ] || exit 0
  if [ "$PLATFORM" = cursor ]; then [ "$N_STATUS" = completed ] || exit 0; fi
  STATE="$(state_file "$SESS_DIR" "$SESSION_ID")"
  [ -f "$STATE" ] || exit 0
  EDITED="$(state_edited_files "$STATE")"
fi

CODE_EDIT="$(printf '%s\n' "$EDITED" | grep -iE "$CODE_EXT_RE" | head -1 || true)"
[ -n "$CODE_EDIT" ] || exit 0

resolve_test_cmd "$ROOT"
[ -n "$TEST_CMD" ] || exit 0

if [ "$PLATFORM" = claude ]; then RAN="$(session_test_runs "$TRANSCRIPT" "$TEST_PAT")"
else RAN="$(state_test_runs "$STATE")"; fi
[ "$RAN" -gt 0 ] && exit 0

mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
COUNTER="$SESS_DIR/${SESSION_ID}.testcount"
HEAD="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo no-git)"
PREV_HEAD=""; PREV_COUNT=0
if [ -f "$COUNTER" ]; then
  PREV_HEAD="$(cut -d' ' -f1 "$COUNTER" 2>/dev/null || echo)"
  PREV_COUNT="$(cut -d' ' -f2 "$COUNTER" 2>/dev/null || echo 0)"
fi
case "$PREV_COUNT" in ''|*[!0-9]*) PREV_COUNT=0 ;; esac
if [ "$PREV_HEAD" = "$HEAD" ]; then
  [ "$PREV_COUNT" -ge 2 ] && exit 0
  NEW_COUNT=$((PREV_COUNT + 1))
else NEW_COUNT=1; fi
printf '%s %s\n' "$HEAD" "$NEW_COUNT" > "$COUNTER" 2>/dev/null || true

MSG="You edited code (\`$(basename "$CODE_EDIT")\`) but ran no tests this session. Run \`$TEST_CMD\` and cite the real output before declaring done. Build-green is not tested. If a test genuinely can't run here, say why."
emit_block "$PLATFORM" "$MSG" "$NEW_COUNT"
exit 0
