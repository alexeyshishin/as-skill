#!/usr/bin/env bash

resolve_test_cmd() {
  local root="$1"
  TEST_CMD=""; TEST_PAT=""
  if [ -f "$root/pubspec.yaml" ]; then TEST_CMD="flutter test"; TEST_PAT="flutter test"
  elif [ -f "$root/pyproject.toml" ]; then TEST_CMD="pytest (or uv run pytest)"; TEST_PAT="pytest"
  elif [ -f "$root/go.mod" ]; then TEST_CMD="go test ./..."; TEST_PAT="go test"
  elif [ -f "$root/build.gradle.kts" ] || [ -f "$root/build.gradle" ]; then TEST_CMD="./gradlew test"; TEST_PAT="gradlew test"
  elif [ -f "$root/package.json" ] && jq -e '.scripts.test // empty' "$root/package.json" >/dev/null 2>&1; then TEST_CMD="npm test"; TEST_PAT="npm test|vitest|jest|node --test"
  elif [ -f "$root/Makefile" ] && grep -qE '^test:' "$root/Makefile" 2>/dev/null; then TEST_CMD="make test"; TEST_PAT="make test"
  fi
  return 0
}

session_edited_files() {
  jq -r 'select(.type=="assistant") | .message.content[]?
    | select(.type=="tool_use")
    | select(.name=="Edit" or .name=="Write" or .name=="MultiEdit")
    | .input.file_path // empty' "$1" 2>/dev/null || true
  return 0
}

session_test_runs() {
  local n
  n="$(jq -r 'select(.type=="assistant") | .message.content[]?
    | select(.type=="tool_use") | select(.name=="Bash")
    | .input.command // empty' "$1" 2>/dev/null | grep -ciE "$2" || true)"
  printf '%s\n' "${n:-0}"
  return 0
}

CODE_EXT_RE='\.(ts|tsx|js|jsx|py|dart|swift|kt|go|rb|java|tf)$'

detect_platform() {
  local d
  d="$(cd "$(dirname "${1:-$0}")/.." 2>/dev/null && basename "$PWD")" || d=""
  case "$d" in
    .codex) echo codex ;;
    .cursor) echo cursor ;;
    *) echo claude ;;
  esac
  return 0
}

sessions_dir() {
  local d
  d="$(cd "$(dirname "${1:-$0}")/.." 2>/dev/null && pwd)" || return 0
  printf '%s/sessions\n' "$d"
  return 0
}

state_file() {
  local sid
  sid="$(printf '%s' "$2" | tr -c 'A-Za-z0-9._-' '_')"
  printf '%s/%s.state\n' "$1" "$sid"
  return 0
}

record_edit() {
  printf 'E %s\n' "$2" >> "$1" 2>/dev/null || true
  return 0
}

record_test_run() {
  printf 'T\n' >> "$1" 2>/dev/null || true
  return 0
}

state_edited_files() {
  [ -f "$1" ] || return 0
  sed -n 's/^E //p' "$1" 2>/dev/null || true
  return 0
}

state_test_runs() {
  local n=0
  [ -f "$1" ] && n="$(grep -c '^T$' "$1" 2>/dev/null || true)"
  printf '%s\n' "${n:-0}"
  return 0
}

normalize_payload() {
  local p="$1" f base
  N_SID="$(printf '%s' "$p" | jq -r '.session_id // .conversation_id // empty' 2>/dev/null || true)"
  N_EVENT="$(printf '%s' "$p" | jq -r '.hook_event_name // empty' 2>/dev/null || true)"
  N_STATUS="$(printf '%s' "$p" | jq -r '.status // empty' 2>/dev/null || true)"
  N_ACTIVE="$(printf '%s' "$p" | jq -r '.stop_hook_active // false' 2>/dev/null || true)"
  local raw patch
  raw="$(printf '%s' "$p" | jq -r '[.tool_input.file_path?, .file_path?] | .[] | strings' 2>/dev/null || true)"
  patch="$(printf '%s' "$p" | jq -r '[.tool_input?, .patch?] | .. | strings' 2>/dev/null \
    | sed -nE 's/^\*\*\* (Update|Add) File: (.*)$/\2/p' || true)"
  raw="$(printf '%s\n%s\n' "$raw" "$patch" | sed '/^$/d')"
  N_FILES=""
  base="${ROOT:-$PWD}"
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    case "$f" in /*) ;; *) f="$base/$f" ;; esac
    N_FILES="${N_FILES:+$N_FILES
}$f"
  done <<EOT
$raw
EOT
  N_CMD=""
  if [ -z "$patch" ]; then
    N_CMD="$(printf '%s' "$p" | jq -r '(.tool_input.command? // .tool_input.cmd? // .command? // empty)
      | if type=="array" then map(tostring) | join(" ") else tostring end' 2>/dev/null || true)"
  fi
  return 0
}

emit_block() {
  case "$1" in
    codex) jq -nc --arg r "$2" '{decision:"block", reason:$r}' ;;
    cursor) jq -nc --arg r "$2" '{followup_message:$r}' ;;
    *) jq -nc --arg r "$2" --arg c "$2  (test-gate ${3}/2 at this commit)" \
      '{decision:"block", reason:$r, hookSpecificOutput:{hookEventName:"Stop", additionalContext:$c}}' ;;
  esac
  return 0
}
