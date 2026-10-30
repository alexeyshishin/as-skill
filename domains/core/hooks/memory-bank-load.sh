#!/usr/bin/env bash
# register: SessionStart startup|clear|compact
# platforms: claude,codex,cursor
set -Eeuo pipefail
trap 'exit 0' ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MB="$ROOT/.memory-bank"
INDEX="$MB/index.md"
[ -f "$INDEX" ] || exit 0

MAX_BYTES=16000
MAX_FILES=100

BODY="$(head -c "$MAX_BYTES" "$INDEX")"
if [ "$(wc -c < "$INDEX" | tr -d ' ')" -gt "$MAX_BYTES" ]; then
  BODY="$BODY"$'\n'"[index.md truncated at $MAX_BYTES bytes — read the file for the rest]"
fi

LIST="$(cd "$MB" && find . -type f -name '*.md' ! -path ./index.md | sed 's|^\./||' | sort)"
TOTAL="$(printf '%s' "$LIST" | grep -c . || true)"

MSG="Memory Bank (.memory-bank/index.md), loaded at session start:

$BODY"

if [ "$TOTAL" -gt 0 ]; then
  SHOWN="$(printf '%s\n' "$LIST" | sed -n "1,${MAX_FILES}p" | sed 's/^/- /')"
  MSG="$MSG

Other Memory Bank files (paths relative to .memory-bank/) — read the ones relevant to the task:
$SHOWN"
  if [ "$TOTAL" -gt "$MAX_FILES" ]; then
    MSG="$MSG
- ... and $((TOTAL - MAX_FILES)) more"
  fi
fi

PLATFORM_DIR="$(basename "$(cd "$(dirname "$0")/.." && pwd)")"
if [ "$PLATFORM_DIR" = ".cursor" ]; then
  jq -nc --arg c "$MSG" '{additional_context:$c}'
else
  jq -nc --arg c "$MSG" '{hookSpecificOutput:{hookEventName:"SessionStart", additionalContext:$c}}'
fi
exit 0
