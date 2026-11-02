#!/usr/bin/env python3
import argparse
import json
import re
import subprocess
import sys

TYPES = ('feat', 'fix', 'refactor', 'perf', 'docs',
         'test', 'build', 'ci', 'chore', 'revert')

SUBJECT_MAX = 72
WHY_MAX_SENTENCES = 3
BODY_MAX_LINES = 60

SECTIONS = ('Why', 'What changed', 'How to verify', 'Risks / breaking changes')

HEADER_RE = re.compile(r'^(?P<type>[a-zA-Z]+)'
                       r'(?:\((?P<scope>[^)]*)\))?'
                       r'(?P<bang>!)?'
                       r': (?P<subject>.+)$')

HEADING_RE = re.compile(r'^(#{1,6})\s+(.*?)\s*#*\s*$')

VAGUE_RE = re.compile(r'^(?:'
                      r'(?:some|various|misc|minor|small|few)\b'
                      r'|(?:update|fix|fixes|changes|cleanup|refactor|'
                      r'improvements?|tweaks?|stuff|things)[\s.]*$'
                      r'|wip\b|tmp\b|temp\b|asdf'
                      r')', re.I)

VAGUE_BULLET_RE = re.compile(r'^(?:misc|various|minor|small|some|other|general)\b'
                             r'|^(?:fixes|improvements|updates|changes|cleanup|'
                             r'tweaks)[\s.]*$', re.I)

NON_IMPERATIVE = {
    'added', 'fixed', 'filled', 'updated', 'removed', 'deleted', 'changed',
    'renamed', 'moved', 'created', 'translated', 'refactored', 'implemented',
    'bumped', 'wrote', 'made', 'documented', 'reverted',
    'adds', 'fixes', 'updates', 'removes', 'deletes', 'changes', 'renames',
    'moves', 'creates', 'makes', 'bumps', 'implements', 'documents',
    'adding', 'fixing', 'updating', 'removing', 'changing', 'renaming',
    'moving', 'creating', 'refactoring', 'implementing', 'bumping',
}

AI_TRACE_RE = re.compile(r'^\W*(?:generated with\s+\[?(?:claude|codex|copilot)'
                         r'|co-authored-by:|claude-session:)'
                         r'|https?://claude\.(?:ai|com)/', re.I)

PLACEHOLDER_RE = re.compile(r'<[^<>\n]*(?:sentence|summary|todo|bullet|'
                            r'change|either|another)[^<>\n]*>'
                            r'|\bTODO\b|\bTBD\b', re.I)

CHATTER_RE = re.compile(r'^(?:want any changes|let me know|should i\b|'
                        r'hope this helps|feel free)', re.I)

NONE_RE = re.compile(r'^(?:none|n/?a|no|нет|-)[\s.]*$', re.I)

EMOJI_RE = re.compile('[\U0001F000-\U0001FAFF☀-➿⭐⬆↔-⇿️]')

PATH_BULLET_RE = re.compile(r'^`?[\w./-]+\.[a-z]{1,5}`?\s*(?:[—:-]|$)')

CODE_RE = re.compile(r'`[^`\n]*`')


class Finding:
    def __init__(self, where, severity, code, problem, fix):
        self.where = where
        self.severity = severity
        self.code = code
        self.problem = problem
        self.fix = fix

    def as_dict(self):
        return {'where': self.where, 'severity': self.severity,
                'code': self.code, 'problem': self.problem, 'fix': self.fix}

    def __str__(self):
        return (f'{self.where} {self.severity} [{self.code}] '
                f'{self.problem} → {self.fix}')


def normalize(text):
    return re.sub(r'\W+', ' ', CODE_RE.sub(lambda m: m.group(0).strip('`'),
                                          text)).strip().lower()


def lint_title(title):
    out = []

    def add(sev, code, problem, fix):
        out.append(Finding('title', sev, code, problem, fix))

    title = title.strip()
    if not title:
        add('ERROR', 'title-empty', 'no title', 'pass --title')
        return out

    m = HEADER_RE.match(title)
    if not m:
        add('ERROR', 'title-format', f'not a Conventional Commit: {title!r}',
            'use `type(scope): subject`')
        return out

    typ, subject = m.group('type'), m.group('subject')

    if typ not in TYPES:
        add('ERROR', 'title-type', f'unknown type {typ!r}',
            'use one of ' + ', '.join(TYPES))
    if m.group('scope') is not None and not m.group('scope').strip():
        add('ERROR', 'title-scope-empty', 'empty scope `()`',
            'drop the parentheses or name the scope')
    if len(title) > SUBJECT_MAX:
        add('WARN', 'title-long', f'{len(title)} chars, limit {SUBJECT_MAX}',
            'shorten the subject')
    if subject.endswith('.'):
        add('WARN', 'title-period', 'trailing period', 'drop it')
    if subject[:1].isupper() and not subject.split()[0].isupper():
        add('WARN', 'title-capital', 'subject starts with a capital letter',
            'lowercase it')
    first = subject.split()[0].lower().strip('`')
    if first in NON_IMPERATIVE:
        add('WARN', 'title-tense', f'{first!r} is not imperative',
            'use the bare verb: add, fix, drop')
    if VAGUE_RE.match(subject):
        add('ERROR', 'title-vague', f'subject says nothing: {subject!r}',
            'name what changed')
    if AI_TRACE_RE.match(title) or EMOJI_RE.search(title):
        add('ERROR', 'title-trace', 'emoji or AI attribution in title',
            'remove it')
    return out


def split_sections(lines):
    sections = []
    current = None
    in_fence = False
    for n, line in enumerate(lines, 1):
        if line.lstrip().startswith('```'):
            in_fence = not in_fence
        m = None if in_fence else HEADING_RE.match(line)
        if m:
            current = {'level': len(m.group(1)), 'name': m.group(2),
                       'line': n, 'lines': []}
            sections.append(current)
        elif current is not None:
            current['lines'].append((n, line))
    return sections


def bullets_of(section):
    return [(n, re.sub(r'^\s*(?:[-*+]|\d+[.)])\s+', '', line).strip())
            for n, line in section['lines']
            if re.match(r'^\s*(?:[-*+]|\d+[.)])\s+\S', line)]


def prose_of(section):
    return ' '.join(line.strip() for _, line in section['lines']
                    if line.strip() and not line.lstrip().startswith(('```', '|')))


def count_sentences(text):
    text = CODE_RE.sub('x', text)
    return len([s for s in re.split(r'(?<=[.!?])\s+(?=[A-ZА-Я])', text.strip()) if s])


def lint_body(body, title, subjects):
    out = []

    def add(line, sev, code, problem, fix):
        out.append(Finding(f'L{line}' if line else 'body', sev, code, problem, fix))

    lines = body.splitlines()
    if not any(l.strip() for l in lines):
        add(0, 'ERROR', 'body-empty', 'empty description',
            'write Why / What changed / How to verify / Risks')
        return out

    if len(lines) > BODY_MAX_LINES:
        add(0, 'WARN', 'body-long', f'{len(lines)} lines, limit {BODY_MAX_LINES}',
            'cut to reasons, move detail into the diff or docs')

    for n, line in enumerate(lines, 1):
        if AI_TRACE_RE.match(line.strip()):
            add(n, 'ERROR', 'ai-attribution', f'AI-attribution trace: {line.strip()!r}',
                'delete the line')
        elif EMOJI_RE.search(line):
            add(n, 'WARN', 'emoji', 'emoji in description', 'remove it')
        if PLACEHOLDER_RE.search(line):
            add(n, 'ERROR', 'placeholder', f'template placeholder left in: {line.strip()!r}',
                'replace with real content')
        if CHATTER_RE.match(line.strip()):
            add(n, 'ERROR', 'chatter', f'assistant chatter in body: {line.strip()!r}',
                'delete it — talk to the user outside the PR text')

    sections = split_sections(lines)
    top = [s for s in sections if s['level'] == 2]
    by_name = {}
    for s in top:
        by_name.setdefault(s['name'].lower(), s)

    for s in sections:
        if s['level'] != 2:
            add(s['line'], 'WARN', 'heading-level',
                f'heading {s["name"]!r} is level {s["level"]}', 'use `## ` for sections')

    missing = [name for name in SECTIONS if name.lower() not in by_name]
    for name in missing:
        plain = any(l.strip().lower().rstrip(':') == name.lower() for l in lines)
        hint = 'make it a `## ` heading' if plain else f'add `## {name}`'
        add(0, 'ERROR', 'section-missing', f'no `## {name}` section', hint)

    present = [by_name[n.lower()] for n in SECTIONS if n.lower() in by_name]
    if [s['line'] for s in present] != sorted(s['line'] for s in present):
        add(0, 'WARN', 'section-order',
            'sections are out of order', 'order: ' + ' → '.join(SECTIONS))

    for s in top:
        known = s['name'].lower() in [n.lower() for n in SECTIONS]
        if known and not any(l.strip() for _, l in s['lines']):
            add(s['line'], 'ERROR', 'section-empty', f'`## {s["name"]}` is empty',
                'fill it or write "none" for Risks')

    why = by_name.get('why')
    if why:
        text = prose_of(why)
        if text and count_sentences(text) > WHY_MAX_SENTENCES:
            add(why['line'], 'WARN', 'why-long',
                f'Why runs past {WHY_MAX_SENTENCES} sentences',
                'keep the motivation to 1-2 sentences, detail goes to What changed')

    what = by_name.get('what changed')
    if what:
        bullets = bullets_of(what)
        if not bullets and any(l.strip() for _, l in what['lines']):
            add(what['line'], 'WARN', 'what-not-bullets',
                'What changed is prose, not bullets', 'one bullet per logical change')
        path_bullets = [b for b in bullets if PATH_BULLET_RE.match(b[1])]
        if len(path_bullets) >= 3 and len(path_bullets) * 2 >= len(bullets):
            add(path_bullets[0][0], 'WARN', 'bullet-per-file',
                f'{len(path_bullets)} bullets lead with a file path',
                'group by logical change, not by file')
        for n, b in bullets:
            if VAGUE_BULLET_RE.match(b):
                add(n, 'WARN', 'bullet-vague', f'vague bullet: {b!r}', 'say what changed')
            if normalize(b) in subjects:
                add(n, 'WARN', 'bullet-is-commit', f'bullet repeats a commit subject: {b!r}',
                    'summarize at the level of reasons, not the log')

    verify = by_name.get('how to verify')
    if verify:
        text = prose_of(verify)
        if text and len(text) < 12:
            add(verify['line'], 'WARN', 'verify-thin', f'verification says only {text!r}',
                'give steps, a command, or the test path')

    risks = by_name.get('risks / breaking changes')
    header = HEADER_RE.match(title.strip())
    bang = bool(header and header.group('bang'))
    if risks:
        text = prose_of(risks)
        is_none = bool(NONE_RE.match(text.strip())) if text else False
        if bang and (is_none or not text):
            add(risks['line'], 'ERROR', 'breaking-undocumented',
                'title has `!` but Risks says none', 'list the break and the migration')
        if not bang and text and not is_none and re.search(r'\bbreaking\b', text, re.I) \
                and not re.search(r'\b(?:no|not|without)\b[^.]{0,20}\bbreaking\b', text, re.I):
            add(risks['line'], 'WARN', 'breaking-no-bang',
                'Risks mentions a breaking change but the title has no `!`',
                'add `!` after type/scope')
    elif bang:
        add(0, 'ERROR', 'breaking-undocumented',
            'title has `!` but there is no Risks section', 'add `## Risks / breaking changes`')

    return out


def lint_pr(title, body, subjects=()):
    return lint_title(title) + lint_body(body, title, set(subjects))


def branch_subjects(base, repo):
    try:
        res = subprocess.run(['git', '-C', repo, 'log', f'{base}..HEAD', '--format=%s'],
                             capture_output=True, text=True, check=True)
    except (OSError, subprocess.CalledProcessError):
        return []
    subs = []
    for s in res.stdout.splitlines():
        m = HEADER_RE.match(s)
        subs.append(normalize(m.group('subject') if m else s))
    return subs


def read_text(path):
    if path == '-':
        return sys.stdin.read()
    try:
        with open(path, encoding='utf-8') as fh:
            return fh.read()
    except OSError as e:
        print(f'error: {e}', file=sys.stderr)
        sys.exit(1)


def report_one(findings, as_json):
    if as_json:
        print(json.dumps([f.as_dict() for f in findings], ensure_ascii=False, indent=2))
    elif not findings:
        print('OK')
    else:
        for f in findings:
            print(f)


def report_history(count, repo, as_json, strict):
    try:
        res = subprocess.run(
            ['gh', 'pr', 'list', '--state', 'all', '--limit', str(count),
             '--json', 'number,title,body'],
            cwd=repo, capture_output=True, text=True, check=True)
    except (OSError, subprocess.CalledProcessError) as e:
        print(f'error: gh pr list failed: {e}', file=sys.stderr)
        sys.exit(1)
    prs = json.loads(res.stdout)
    rows = [(p['number'], p['title'], lint_pr(p['title'], p['body'] or '')) for p in prs]
    clean = sum(1 for _, _, fs in rows if not fs)
    by_code = {}
    for _, _, fs in rows:
        for f in fs:
            by_code.setdefault((f.severity, f.code), 0)
            by_code[(f.severity, f.code)] += 1

    if as_json:
        print(json.dumps({
            'graded': len(rows), 'clean': clean,
            'by_code': {f'{s}/{c}': n for (s, c), n in by_code.items()},
            'prs': [{'number': n, 'title': t, 'findings': [f.as_dict() for f in fs]}
                    for n, t, fs in rows],
        }, ensure_ascii=False, indent=2))
    else:
        print(f'{len(rows)} PRs graded, {clean} clean\n')
        for n, t, fs in rows:
            print(f'#{n}  {t}')
            for f in fs:
                print(f'    {f.where} {f.severity} [{f.code}] {f.problem} → {f.fix}')
            print()
        if by_code:
            print('Most common:')
            for (sev, code), n in sorted(by_code.items(), key=lambda kv: -kv[1]):
                print(f'   {sev} [{code}] — {n}')

    errors = sum(1 for _, _, fs in rows for f in fs if f.severity == 'ERROR')
    warns = sum(1 for _, _, fs in rows for f in fs if f.severity == 'WARN')
    return errors, warns


def main():
    ap = argparse.ArgumentParser(
        description='Lint a PR title and description against the git-conventions PR rules.')
    ap.add_argument('body', nargs='?', help='description file, or - for stdin')
    ap.add_argument('--title', help='PR title')
    ap.add_argument('--base', help='base branch; flags bullets that repeat commit subjects')
    ap.add_argument('--history', type=int, metavar='N',
                    help='audit the last N PRs via gh')
    ap.add_argument('--repo', default='.', help='repository path (default .)')
    ap.add_argument('--json', action='store_true')
    ap.add_argument('--strict', action='store_true', help='warnings fail too')
    args = ap.parse_args()

    if args.history is not None:
        errors, warns = report_history(args.history, args.repo, args.json, args.strict)
    else:
        if args.body is None or args.title is None:
            print('error: pass a description file (or -) and --title, or --history N',
                  file=sys.stderr)
            sys.exit(1)
        subjects = branch_subjects(args.base, args.repo) if args.base else []
        findings = lint_pr(args.title, read_text(args.body), subjects)
        report_one(findings, args.json)
        errors = sum(1 for f in findings if f.severity == 'ERROR')
        warns = len(findings) - errors

    sys.exit(2 if errors or (args.strict and warns) else 0)


if __name__ == '__main__':
    main()
