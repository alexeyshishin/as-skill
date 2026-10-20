#!/usr/bin/env bash
# Stop hook — "test after every feature" enforcement.
# If code files were edited this session and no test command was run, block the
# agent from declaring done until it runs the project's tests and cites output.
# Fail-open on any error. Loop-safe: caps at 2 blocks per git HEAD, then gives up.
set -euo pipefail
trap 'exit 0' ERR

command -v jq >/dev/null 2>&1 || exit 0

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STDIN="$(cat)"

# Never re-block once already inside a Stop-hook continuation.
[ "$(printf '%s' "$STDIN" | jq -r '.stop_hook_active // false')" = "true" ] && exit 0

SESSION_ID="$(printf '%s' "$STDIN" | jq -r '.session_id // empty')"
TRANSCRIPT="$(printf '%s' "$STDIN" | jq -r '.transcript_path // empty')"
[ -n "$SESSION_ID" ] || exit 0
[ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ] || exit 0

# Did the agent edit any code file this session?
EDITED="$(jq -r 'select(.type=="assistant") | .message.content[]?
  | select(.type=="tool_use")
  | select(.name=="Edit" or .name=="Write" or .name=="MultiEdit")
  | .input.file_path // empty' "$TRANSCRIPT" 2>/dev/null || echo)"
CODE_EDIT="$(printf '%s\n' "$EDITED" | grep -iE '\.(ts|tsx|js|jsx|py|dart|swift|kt|go|rb|java|tf)$' | head -1 || true)"
[ -n "$CODE_EDIT" ] || exit 0

# Resolve this project's test command from what's actually present.
TEST_CMD=""; TEST_PAT=""
if [ -f "$ROOT/pubspec.yaml" ]; then TEST_CMD="flutter test"; TEST_PAT="flutter test"
elif [ -f "$ROOT/pyproject.toml" ]; then TEST_CMD="pytest (or uv run pytest)"; TEST_PAT="pytest"
elif [ -f "$ROOT/go.mod" ]; then TEST_CMD="go test ./..."; TEST_PAT="go test"
elif [ -f "$ROOT/build.gradle.kts" ] || [ -f "$ROOT/build.gradle" ]; then TEST_CMD="./gradlew test"; TEST_PAT="gradlew test"
elif [ -f "$ROOT/package.json" ] && jq -e '.scripts.test // empty' "$ROOT/package.json" >/dev/null 2>&1; then TEST_CMD="npm test"; TEST_PAT="npm test|vitest|jest|node --test"
elif [ -f "$ROOT/Makefile" ] && grep -qE '^test:' "$ROOT/Makefile" 2>/dev/null; then TEST_CMD="make test"; TEST_PAT="make test"
fi
[ -n "$TEST_CMD" ] || exit 0   # no known test command -> nothing to enforce

# Was a test command actually run in a Bash tool call this session?
RAN="$(jq -r 'select(.type=="assistant") | .message.content[]?
  | select(.type=="tool_use") | select(.name=="Bash")
  | .input.command // empty' "$TRANSCRIPT" 2>/dev/null | grep -ciE "$TEST_PAT" || true)"
RAN="${RAN:-0}"
[ "$RAN" -gt 0 ] && exit 0

# Loop-safe cap: <=2 blocks at the same HEAD.
SESS_DIR="$ROOT/.claude/sessions"; mkdir -p "$SESS_DIR" 2>/dev/null || exit 0
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
jq -nc --arg r "$MSG" --arg c "$MSG  (test-gate ${NEW_COUNT}/2 at this commit)" \
  '{decision:"block", reason:$r, hookSpecificOutput:{hookEventName:"Stop", additionalContext:$c}}'
exit 0