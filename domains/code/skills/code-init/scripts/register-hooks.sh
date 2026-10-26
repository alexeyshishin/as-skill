#!/usr/bin/env bash
set -Eeuo pipefail

die() { echo "register-hooks: $*" >&2; exit 1; }

PROJECT="."; DRY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --project) [ $# -ge 2 ] || die "--project needs a directory"; PROJECT="$2"; shift 2 ;;
    --dry-run) DRY=1; shift ;;
    *) die "usage: register-hooks.sh [--project DIR] [--dry-run]" ;;
  esac
done

command -v jq >/dev/null 2>&1 || die "jq not found (the hooks need it too) — nothing registered"

ROOT="$(cd "$PROJECT" 2>/dev/null && pwd -P)" || die "no such directory: $PROJECT"
HOME_REAL="$(cd "$HOME" 2>/dev/null && pwd -P || printf '%s' "$HOME")"
[ "$ROOT" != "$HOME_REAL" ] || die "refusing: $ROOT/.claude/settings.json is the global settings file"

CLAUDE_DIR="$ROOT/.claude"
SETTINGS="$CLAUDE_DIR/settings.json"
LOCAL="$CLAUDE_DIR/settings.local.json"

JSON='{}'
if [ -s "$SETTINGS" ]; then
  jq -e 'type == "object"' "$SETTINGS" >/dev/null 2>&1 || die "$SETTINGS is not a valid JSON object — left untouched"
  JSON="$(cat "$SETTINGS")"
fi
LOCAL_JSON="$(jq -c . "$LOCAL" 2>/dev/null || true)"

has_entry() {
  printf '%s' "$1" | jq -e --arg ev "$2" --arg n "/.claude/hooks/$3" \
    '[(.hooks[$ev]? // []) | .[]? | .hooks[]? | .command? | strings | select(contains($n))] | length > 0' >/dev/null 2>&1
}

shopt -s nullglob
ADDED=0
for f in "$CLAUDE_DIR"/hooks/*.sh; do
  LINE="$(grep -m1 '^# register: ' "$f" || true)"
  [ -n "$LINE" ] || continue
  read -r EVENT MATCHER <<<"${LINE#\# register: }"
  [ -n "$EVENT" ] || continue
  NAME="$(basename "$f")"

  if has_entry "$JSON" "$EVENT" "$NAME" || has_entry "$LOCAL_JSON" "$EVENT" "$NAME"; then
    STATE=present
  else
    CMD="\"\$CLAUDE_PROJECT_DIR\"/.claude/hooks/$NAME"
    JSON="$(printf '%s' "$JSON" | jq --arg ev "$EVENT" --arg m "$MATCHER" --arg cmd "$CMD" \
      '.hooks[$ev] = ((.hooks[$ev] // []) + [(if $m != "" then {matcher: $m} else {} end) + {hooks: [{type: "command", command: $cmd}]}])')" \
      || die "can't add $NAME under hooks.$EVENT (unexpected shape of .hooks in $SETTINGS) — nothing written"
    ADDED=$((ADDED + 1))
    if [ "$DRY" -eq 1 ]; then STATE=would-add; else STATE=added; fi
  fi
  [ -x "$f" ] || echo "register-hooks: warning: $f is not executable — the hook will fail until chmod +x" >&2
  printf '%s\t%s\t%s\n' "$STATE" "$NAME" "$EVENT"
done

if [ "$ADDED" -gt 0 ] && [ "$DRY" -eq 0 ]; then
  mkdir -p "$CLAUDE_DIR"
  if printf '%s' "$JSON" | jq . > "$SETTINGS.tmp"; then
    mv "$SETTINGS.tmp" "$SETTINGS"
  else
    rm -f "$SETTINGS.tmp"; die "can't write $SETTINGS"
  fi
fi
exit 0
