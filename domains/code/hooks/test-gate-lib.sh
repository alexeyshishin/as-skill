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
