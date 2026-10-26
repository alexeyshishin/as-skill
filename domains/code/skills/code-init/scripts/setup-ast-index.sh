#!/usr/bin/env bash
# Sets up ast-index (structural code search) for one project, project-scope only.
# Usage: setup-ast-index.sh [--project DIR] [--dry-run]     (DIR defaults to the current directory)
# Steps, each idempotent: marketplace + plugin via `claude plugin ... --scope project`,
# "Bash(ast-index *)" in permissions.allow of <project>/.claude/settings.json, then the index
# (`ast-index rebuild` when none exists, else incremental `ast-index update`).
# Installs project-scope even if the plugin is enabled globally. Adds no hooks — the plugin
# brings its own. Never writes the global ~/.claude/settings.json.
# stdout, one line per step: <added|would-add|present|built|would-build|updated> <TAB> <step>,
# then the raw `ast-index stats` output. `claude` / `ast-index` progress goes to stderr.
# Exit 1, message on stderr: no ast-index / claude / jq, settings.json not a JSON object,
# project is $HOME, or a step failed (earlier steps stay applied; re-run is safe).
set -Eeuo pipefail

die() { echo "setup-ast-index: $*" >&2; exit 1; }

PROJECT="."; DRY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --project) [ $# -ge 2 ] || die "--project needs a directory"; PROJECT="$2"; shift 2 ;;
    --dry-run) DRY=1; shift ;;
    *) die "usage: setup-ast-index.sh [--project DIR] [--dry-run]" ;;
  esac
done

command -v ast-index >/dev/null 2>&1 || die "ast-index not found — install: brew tap defendend/ast-index && brew install ast-index"
command -v claude >/dev/null 2>&1 || die "claude CLI not found — nothing set up"
command -v jq >/dev/null 2>&1 || die "jq not found — nothing set up"

ROOT="$(cd "$PROJECT" 2>/dev/null && pwd -P)" || die "no such directory: $PROJECT"
HOME_REAL="$(cd "$HOME" 2>/dev/null && pwd -P || printf '%s' "$HOME")"
[ "$ROOT" != "$HOME_REAL" ] || die "refusing: $ROOT/.claude/settings.json is the global settings file"

REPO="defendend/Claude-ast-index-search"
MARKETPLACE="ast-index-marketplace"
PLUGIN="ast-index@$MARKETPLACE"
PERM="Bash(ast-index *)"
SETTINGS="$ROOT/.claude/settings.json"

settings() {
  if [ -s "$SETTINGS" ]; then
    jq -e 'type == "object"' "$SETTINGS" >/dev/null 2>&1 || die "$SETTINGS is not a valid JSON object — left untouched"
    cat "$SETTINGS"
  else
    echo '{}'
  fi
}

has() { settings | jq -e --arg m "$MARKETPLACE" --arg p "$PLUGIN" --arg perm "$PERM" "$1" >/dev/null 2>&1; }

report() { if [ "$DRY" -eq 1 ]; then printf '%s\t%s\n' "$2" "$3"; else printf '%s\t%s\n' "$1" "$3"; fi; }

in_root() { (cd "$ROOT" && "$@") >&2; }

settings >/dev/null

if has '.extraKnownMarketplaces[$m] != null'; then
  report present present marketplace
else
  [ "$DRY" -eq 1 ] || in_root claude plugin marketplace add "$REPO" --scope project || die "marketplace add failed"
  report added would-add marketplace
fi

if has '.enabledPlugins[$p] == true'; then
  report present present plugin
else
  [ "$DRY" -eq 1 ] || in_root claude plugin install "$PLUGIN" --scope project || die "plugin install failed"
  report added would-add plugin
fi

if has '(.permissions.allow // []) | index($perm) != null'; then
  report present present permission
else
  if [ "$DRY" -eq 0 ]; then
    mkdir -p "$ROOT/.claude"
    JSON="$(settings | jq --arg perm "$PERM" '.permissions.allow = ((.permissions.allow // []) + [$perm])')" \
      || die "can't add $PERM (unexpected shape of .permissions in $SETTINGS) — nothing written"
    if printf '%s' "$JSON" | jq . > "$SETTINGS.tmp"; then
      mv "$SETTINGS.tmp" "$SETTINGS"
    else
      rm -f "$SETTINGS.tmp"; die "can't write $SETTINGS"
    fi
  fi
  report added would-add permission
fi

DB="$(cd "$ROOT" && ast-index db-path 2>/dev/null)" || die "ast-index db-path failed"
if [ -f "$DB" ]; then
  [ "$DRY" -eq 1 ] || in_root ast-index update || die "ast-index update failed"
  report updated present index
else
  [ "$DRY" -eq 1 ] || in_root ast-index rebuild || die "ast-index rebuild failed"
  report built would-build index
fi

[ "$DRY" -eq 1 ] || (cd "$ROOT" && ast-index stats)
exit 0
