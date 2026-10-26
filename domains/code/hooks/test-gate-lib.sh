#!/usr/bin/env bash
# Sourced by test-gate.sh and prettier-format.sh — one definition of "test-gate is satisfied".
# Not a hook itself. Every function must stay failure-proof: callers fail open via an ERR trap.

# Sets TEST_CMD (for messages) and TEST_PAT (grep -E) from what's present in project root $1.
# Both stay empty when no known test command exists.
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

# Paths the agent wrote via Edit/Write/MultiEdit this session ($1 = transcript), one per line.
session_edited_files() {
  jq -r 'select(.type=="assistant") | .message.content[]?
    | select(.type=="tool_use")
    | select(.name=="Edit" or .name=="Write" or .name=="MultiEdit")
    | .input.file_path // empty' "$1" 2>/dev/null || true
  return 0
}

# How many Bash tool calls this session ($1 = transcript) matched the test pattern $2.
session_test_runs() {
  local n
  n="$(jq -r 'select(.type=="assistant") | .message.content[]?
    | select(.type=="tool_use") | select(.name=="Bash")
    | .input.command // empty' "$1" 2>/dev/null | grep -ciE "$2" || true)"
  printf '%s\n' "${n:-0}"
  return 0
}
