#!/usr/bin/env bash
# register: PostToolUse
# platforms: codex,cursor
set -Eeuo pipefail
trap 'exit 0' ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LIB="$(dirname "$0")/test-gate-lib.sh"
[ -f "$LIB" ] || exit 0
. "$LIB"
STDIN="$(cat)"

PLATFORM="$(detect_platform)"
[ "$PLATFORM" != claude ] || exit 0

normalize_payload "$STDIN"
[ -n "$N_SID" ] || exit 0

SESS_DIR="$(sessions_dir)"
[ -n "$SESS_DIR" ] || exit 0
mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
STATE="$(state_file "$SESS_DIR" "$N_SID")"

if [ -n "$N_FILES" ]; then
  while IFS= read -r f; do
    [ -n "$f" ] && record_edit "$STATE" "$f"
  done <<EOT
$N_FILES
EOT
fi

if [ -n "$N_CMD" ]; then
  resolve_test_cmd "$ROOT"
  if [ -n "$TEST_PAT" ] && printf '%s\n' "$N_CMD" | grep -qiE "$TEST_PAT"; then
    record_test_run "$STATE"
  fi
fi
exit 0
