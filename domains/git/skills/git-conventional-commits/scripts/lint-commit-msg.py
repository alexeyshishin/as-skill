#!/usr/bin/env python3
"""Conventional Commits linter for this repo's git conventions.

Why a script when the convention is already written down: checking a header
against a grammar is mechanical. A model doing it by eye burns tokens on every
pass and still lets through the cases that actually recur in this history —
past-tense subjects, `chore:` on a docs-only change, two changes joined by
"and", a body nobody asked for, an AI-attribution trailer copied in by the
harness. Commits here are subject-only: anything after the header is an error.

Modes:
  lint-commit-msg.py <file>          lint one message (use as a commit-msg hook)
  lint-commit-msg.py -               lint a message on stdin
  lint-commit-msg.py --history [N]   audit the last N commits (default 50)
  ... --repo <path>                  run the audit against another checkout
  ... --json                         machine-readable output
  ... --strict                       warnings count as failures

Exit codes: 0 clean, 1 usage/IO error, 2 findings (errors; warnings too under
--strict). No dependencies, Python 3.8+.
"""

import json
import re
import subprocess
import sys
from pathlib import Path

# ─────────────────────────────────────────────────────────────────────────────
# Convention. Single source of truth is rules/git-conventions.md — these are
# the parts of it a regex can see.
# ─────────────────────────────────────────────────────────────────────────────

TYPES = ('feat', 'fix', 'refactor', 'perf', 'docs',
         'test', 'build', 'ci', 'chore', 'revert')

SUBJECT_MAX = 72        # hard limit from the convention

HEADER_RE = re.compile(r'^(?P<type>[a-zA-Z]+)'
                       r'(?:\((?P<scope>[^)]*)\))?'
                       r'(?P<bang>!)?'
                       r': (?P<subject>.+)$')

# Headers that aren't authored messages and shouldn't be graded.
SKIP_RE = re.compile(r'^(Merge\s|Revert\s"|fixup!\s|squash!\s|amend!\s)')

# Subjects that say nothing. Drawn from what actually sits in this history:
# `fix: some UI`, `chore: update`, `build: dashboard`.
VAGUE_RE = re.compile(r'^(?:'
                      r'(?:some|various|misc|minor|small|few)\b'
                      r'|(?:update|fix|fixes|changes|cleanup|refactor|'
                      r'improvements?|tweaks?|stuff|things)[\s.]*$'
                      r'|wip\b|tmp\b|temp\b|test commit\b|asdf'
                      r')', re.I)

# Past tense / third person / gerund where the convention wants an imperative.
NON_IMPERATIVE = {
    'added', 'fixed', 'filled', 'updated', 'removed', 'deleted', 'changed',
    'renamed', 'moved', 'created', 'translated', 'refactored', 'implemented',
    'bumped', 'wrote', 'made', 'documented', 'reverted',
    # 'split', 'drop' and friends are omitted on purpose: their past tense is
    # identical to the imperative, so flagging them only produces noise.
    'adds', 'fixes', 'updates', 'removes', 'deletes', 'changes', 'renames',
    'moves', 'creates', 'makes', 'bumps', 'implements', 'documents',
    'adding', 'fixing', 'updating', 'removing', 'changing', 'renaming',
    'moving', 'creating', 'refactoring', 'implementing', 'bumping',
}

# Docs-only signals in a subject. If the type isn't docs, that's a mismatch:
# `chore: update README.md` and `feat: update note` are both in this history.
DOCS_HINT_RE = re.compile(r'\b(readme(?:\.md)?|changelog|\.md\b|docs?\b|'
                          r'documentation|note|комментари)', re.I)
DOCS_HINT_TYPES = ('feat', 'fix', 'chore')

# Two changes in one commit, announced by the subject itself.
CONJUNCTION_RE = re.compile(r'(,\s+(?:add|fix|update|remove|rename|move|drop|'
                            r'delete|refactor)\b|\s+and\s+(?:add|fix|update|'
                            r'remove|rename|move|drop|delete|refactor)\b)', re.I)

# AI traces the convention bans outright. Matched as whole lines on purpose, so
# it can name the trace precisely (ai-attribution) instead of lumping it in with
# a generic body-present, and so prose that merely *mentions* a trailer
# ("...add a Co-Authored-By: X <y> footer") is not mistaken for one.
AI_TRAILER_PATTERNS = (
    re.compile(r'^co-authored-by:\s*[^<>]*'
               r'(?:claude|anthropic|gpt|copilot|codex)[^<>]*<[^>]+>$', re.I),
    re.compile(r'^co-authored-by:\s*[^<>]*<[^>]*'
               r'(?:anthropic|openai|copilot)[^>]*>$', re.I),
    re.compile(r'^claude-session:\s*\S+$', re.I),
    re.compile(r'^(?:🤖\s*)?generated with\s+\[?(?:claude|codex|copilot)', re.I),
    re.compile(r'^\S*\s*https?://claude\.(?:ai|com)/\S*$', re.I),
)


def is_ai_trailer(line):
    return any(p.match(line) for p in AI_TRAILER_PATTERNS)


SCOPE_RE = re.compile(r'^[a-z0-9][a-z0-9-]*$')


class Finding:
    def __init__(self, line, severity, code, problem, fix):
        self.line = line
        self.severity = severity
        self.code = code
        self.problem = problem
        self.fix = fix

    def as_dict(self):
        return {'line': self.line, 'severity': self.severity, 'code': self.code,
                'problem': self.problem, 'fix': self.fix}

    def __str__(self):
        return (f'L{self.line} {self.severity} [{self.code}] '
                f'{self.problem} → {self.fix}')


def err(line, code, problem, fix):
    return Finding(line, 'ERROR', code, problem, fix)


def warn(line, code, problem, fix):
    return Finding(line, 'WARN', code, problem, fix)


# ─────────────────────────────────────────────────────────────────────────────
# Header
# ─────────────────────────────────────────────────────────────────────────────

def lint_header(header):
    out = []
    m = HEADER_RE.match(header)
    if not m:
        hint = 'write `<type>(<scope>)?: <subject>`'
        if ':' in header:
            left = header.split(':', 1)[0]
            if left.lower().rstrip('!').split('(')[0] in TYPES:
                hint = 'a single space is required after the colon'
        return [err(1, 'no-conventional-header',
                    f'header is not a Conventional Commit: {header!r}', hint)]

    ctype, scope, subject = m.group('type'), m.group('scope'), m.group('subject')

    # type
    if ctype != ctype.lower():
        out.append(err(1, 'type-case', f'type {ctype!r} is not lowercase',
                       f'use {ctype.lower()!r}'))
    if ctype.lower() not in TYPES:
        out.append(err(1, 'unknown-type', f'type {ctype!r} is not allowed',
                       'pick one of: ' + ', '.join(TYPES)))

    # scope
    if scope is not None:
        if not scope:
            out.append(err(1, 'empty-scope', 'empty parentheses in the scope',
                           'drop the parentheses or name the module'))
        elif not SCOPE_RE.match(scope):
            reason = 'scope must be lowercase kebab-case'
            if '/' in scope:
                reason = 'scope is a module name, not a path'
            elif '.' in scope:
                reason = 'scope is a module name, not a filename'
            out.append(err(1, 'scope-format', f'{reason}: {scope!r}',
                           'e.g. `cli`, `sdk-go`, `auth`'))
        elif len(scope) > 15:
            out.append(warn(1, 'scope-long',
                            f'scope {scope!r} is {len(scope)} chars',
                            'shorten it to a module name'))

    # subject
    out += lint_subject(subject, ctype.lower())

    if len(header) > SUBJECT_MAX:
        out.append(err(1, 'header-too-long',
                       f'header is {len(header)} chars, limit is {SUBJECT_MAX}',
                       'shorten the subject — name the change, not its parts'))

    return out


def lint_subject(subject, ctype):
    out = []

    if subject.endswith('.'):
        out.append(err(1, 'subject-period', 'subject ends with a period',
                       'drop it'))
    if subject != subject.strip():
        out.append(err(1, 'subject-space', 'subject has leading/trailing space',
                       'trim it'))
    if '  ' in subject:
        out.append(warn(1, 'subject-double-space', 'double space in the subject',
                        'collapse it'))

    first = subject.split()[0] if subject.split() else ''
    # Capitalisation: an acronym or a proper noun is fine, a sentence start isn't.
    if first[:1].isupper() and not (first.isupper() or
                                    any(c.isupper() for c in first[1:])):
        out.append(err(1, 'subject-case',
                       f'subject starts with a capital: {first!r}',
                       f'use {first.lower()!r} — the subject is not a sentence'))

    if first.lower() in NON_IMPERATIVE:
        out.append(err(1, 'subject-mood',
                       f'{first!r} is not imperative',
                       'describe what changed: `add`, `fix`, `drop`, `rename`'))

    if VAGUE_RE.match(subject):
        out.append(err(1, 'subject-vague', f'subject says nothing: {subject!r}',
                       'name the component and the change'))

    if len(subject.split()) < 2:
        out.append(err(1, 'subject-too-short',
                       f'subject is one word: {subject!r}',
                       'a verb plus an object, at minimum'))

    if CONJUNCTION_RE.search(subject):
        out.append(warn(1, 'subject-non-atomic',
                        'subject announces two changes',
                        'split it into two commits (see the atomicity rule)'))

    if ctype in DOCS_HINT_TYPES and DOCS_HINT_RE.search(subject):
        out.append(warn(1, 'type-mismatch-docs',
                        f'{ctype}: on what looks like a docs-only change',
                        'use `docs:` if no code behaviour changed'))

    if ctype == 'revert' and not re.search(r'"|\b[0-9a-f]{7,40}\b', subject):
        out.append(warn(1, 'revert-no-ref',
                        'revert does not name the reverted commit',
                        'quote the original subject or its hash'))

    return out


# ─────────────────────────────────────────────────────────────────────────────
# Body and footers
# ─────────────────────────────────────────────────────────────────────────────

def lint_body(lines):
    """lines — everything after the header, numbered from line 2.

    The convention is subject-only: no body, no footer, nothing after the
    header. A breaking change is the `!` in the header and nothing more.
    """
    out = []
    prose = None

    for i, raw in enumerate(lines, start=2):
        line = raw.strip()
        if not line:
            continue
        if is_ai_trailer(line):
            out.append(err(i, 'ai-attribution',
                           f'AI-attribution trace: {line!r}',
                           'remove it — the author of a commit is the user '
                           'alone (git-conventions.md#no-ai-attribution)'))
        elif prose is None:
            prose = i

    if prose is not None:
        out.append(err(prose, 'body-present',
                       'commit has a body or footer',
                       'subject only — drop everything after the header; '
                       'the diff says what, the PR says why'))

    return out


def lint_message(text):
    """Returns (findings, skipped). A skipped message is machine-generated."""
    lines = text.replace('\r\n', '\n').split('\n')
    # Drop the comment block git appends to a commit-msg template.
    lines = [ln for ln in lines if not ln.startswith('#')]
    while lines and not lines[-1].strip():
        lines.pop()
    if not lines or not lines[0].strip():
        return [err(1, 'empty-message', 'commit message is empty',
                    'write a Conventional Commit header')], False

    header = lines[0].rstrip()
    if SKIP_RE.match(header):
        return [], True

    findings = lint_header(header) + lint_body(lines[1:])
    findings.sort(key=lambda f: (f.line, f.severity != 'ERROR'))
    return findings, False


# ─────────────────────────────────────────────────────────────────────────────
# Modes
# ─────────────────────────────────────────────────────────────────────────────

def read_message(arg):
    if arg == '-':
        return sys.stdin.read()
    path = Path(arg).expanduser()
    if not path.is_file():
        print(f'error: not a file: {path}', file=sys.stderr)
        sys.exit(1)
    return path.read_text(encoding='utf-8')


def git_log(repo, count):
    sep = '\x1e'
    try:
        out = subprocess.run(
            ['git', '-C', repo, 'log', f'-n{count}',
             f'--pretty=format:%h{sep}%B\x1d'],
            capture_output=True, text=True, check=True).stdout
    except FileNotFoundError:
        print('error: git not found', file=sys.stderr)
        sys.exit(1)
    except subprocess.CalledProcessError as e:
        print(f'error: {e.stderr.strip() or "git log failed"}', file=sys.stderr)
        sys.exit(1)
    commits = []
    for chunk in out.split('\x1d'):
        if sep not in chunk:
            continue
        sha, body = chunk.split(sep, 1)
        commits.append((sha.strip(), body))
    return commits


def report_one(findings, as_json):
    errors = sum(1 for f in findings if f.severity == 'ERROR')
    warns = len(findings) - errors
    if as_json:
        print(json.dumps({'errors': errors, 'warnings': warns,
                          'findings': [f.as_dict() for f in findings]},
                         ensure_ascii=False, indent=2))
        return
    if not findings:
        print('OK — message follows the convention')
        return
    for f in findings:
        print(f)
    print(f'\n{errors} error(s), {warns} warning(s)')


def report_history(repo, count, as_json, strict):
    commits = git_log(repo, count)
    rows, by_code = [], {}
    clean = skipped = 0
    for sha, body in commits:
        findings, was_skipped = lint_message(body)
        if was_skipped:
            skipped += 1
            continue
        subject = body.split('\n', 1)[0].strip()
        if not findings:
            clean += 1
            continue
        rows.append((sha, subject, findings))
        for f in findings:
            by_code.setdefault((f.severity, f.code), []).append(sha)

    # Same subject twice in a row is an amend/fixup that never happened.
    dupes = {}
    subjects = [(sha, b.split('\n', 1)[0].strip()) for sha, b in commits]
    for (_, a), (sha_b, b) in zip(subjects, subjects[1:]):
        if a and a == b:
            dupes.setdefault(b, set()).update({sha_b})

    graded = len(commits) - skipped
    if as_json:
        print(json.dumps({
            'repo': repo, 'graded': graded, 'skipped': skipped, 'clean': clean,
            'by_code': {f'{s}/{c}': shas for (s, c), shas in by_code.items()},
            'duplicate_subjects': {k: sorted(v) for k, v in dupes.items()},
            'commits': [{'sha': sha, 'subject': subj,
                         'findings': [f.as_dict() for f in fs]}
                        for sha, subj, fs in rows],
        }, ensure_ascii=False, indent=2))
    else:
        print(f'{repo}: {graded} commits graded, {clean} clean, '
              f'{skipped} merge/fixup skipped\n')
        for sha, subject, findings in rows:
            print(f'{sha}  {subject}')
            for f in findings:
                print(f'    {f.severity} [{f.code}] {f.problem} → {f.fix}')
            print()
        if by_code:
            print('Most common:')
            for (sev, code), shas in sorted(by_code.items(),
                                            key=lambda kv: -len(kv[1])):
                print(f'   {sev} [{code}] — {len(shas)}')
        if dupes:
            print('\nRepeated subjects (wanted `git commit --amend` or a fixup):')
            for subject, shas in dupes.items():
                print(f'   {subject!r} — {len(shas) + 1}×')

    errors = sum(1 for _, _, fs in rows for f in fs if f.severity == 'ERROR')
    warns = sum(1 for _, _, fs in rows for f in fs if f.severity == 'WARN')
    return errors, warns


def main():
    argv = sys.argv[1:]
    if not argv or '-h' in argv or '--help' in argv:
        print(__doc__)
        return
    as_json = '--json' in argv
    strict = '--strict' in argv

    repo = '.'
    if '--repo' in argv:
        i = argv.index('--repo')
        if i + 1 >= len(argv):
            print('error: --repo needs a path', file=sys.stderr)
            sys.exit(1)
        repo = argv[i + 1]
        del argv[i:i + 2]

    positional = [a for a in argv if not a.startswith('--')]

    if '--history' in argv:
        count = 50
        if positional:
            if not positional[0].isdigit():
                print('error: --history takes a commit count', file=sys.stderr)
                sys.exit(1)
            count = int(positional[0])
        errors, warns = report_history(repo, count, as_json, strict)
    else:
        if not positional:
            print('error: pass a message file, `-` for stdin, or --history',
                  file=sys.stderr)
            sys.exit(1)
        findings, skipped = lint_message(read_message(positional[0]))
        if skipped and not as_json:
            print('OK — merge/fixup message, not graded')
        else:
            report_one(findings, as_json)
        errors = sum(1 for f in findings if f.severity == 'ERROR')
        warns = len(findings) - errors

    sys.exit(2 if errors or (strict and warns) else 0)


if __name__ == '__main__':
    main()
