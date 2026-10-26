#!/usr/bin/env python3
"""Next semver tag and release notes for a single-package repo.

Why a script: picking the version is mechanical — find the last valid tag,
read the Conventional Commit headers since it, take the highest bump. A model
doing it by eye miscounts, trusts a `v.1.0.0` lying around, or forgets the
pre-1.0 rule. Same for the notes: the skeleton comes straight from the
commits, and the slop check is a stop-list, not taste.

Modes:
  release.py next  [--bump LEVEL]     last tag, next version, commits by group
  release.py notes [--bump LEVEL]     release-notes skeleton (markdown)
  release.py lint  <file|->           lint drafted notes for slop and structure
  ... --repo <path>                   run against another checkout
  ... --json                          machine-readable output (next, lint)
  ... --strict                        lint: warnings count as failures

Tags: only `vMAJOR.MINOR.PATCH`. Anything else (`v.1.0.0`, `1.0.0`,
`v1.0.0-rc.1`, `sdk-go/v0.3.1`) is ignored with a warning.

Exit codes: 0 ok, 1 usage/IO/git error, 2 lint findings (errors; warnings too
under --strict), 3 nothing release-worthy since the last tag.
No dependencies, Python 3.8+.
"""

import json
import re
import subprocess
import sys
from pathlib import Path


TAG_RE = re.compile(r'^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$')

HEADER_RE = re.compile(r'^(?P<type>[a-zA-Z]+)'
                       r'(?:\((?P<scope>[^)]*)\))?'
                       r'(?P<bang>!)?'
                       r': (?P<subject>.+)$')

BREAKING_FOOTER_RE = re.compile(r'^BREAKING[ -]CHANGE:', re.M)

PR_RE = re.compile(r'\s*\(#(\d+)\)$')

LEVELS = ('none', 'patch', 'minor', 'major')

TYPE_LEVEL = {
    'feat': 'minor',
    'fix': 'patch', 'perf': 'patch', 'docs': 'patch',
    'refactor': 'patch', 'revert': 'patch',
    'build': 'none', 'ci': 'none', 'chore': 'none',
    'test': 'none', 'style': 'none',
}

SECTIONS = (
    ('breaking', 'Breaking changes'),
    ('feat', 'Features'),
    ('fix', 'Fixes'),
    ('perf', 'Performance'),
    ('other', 'Other'),
)
OTHER_TYPES = ('docs', 'refactor', 'revert')
SECTION_TITLES = {title for _, title in SECTIONS}

SUMMARY_PLACEHOLDER = '<one sentence: the main user-visible change>'
SUMMARY_MAX = 120


def die(msg):
    print(f'error: {msg}', file=sys.stderr)
    sys.exit(1)


def git(repo, *args):
    try:
        return subprocess.run(['git', '-C', repo, *args], capture_output=True,
                              text=True, check=True).stdout
    except FileNotFoundError:
        die('git not found')
    except subprocess.CalledProcessError as e:
        die(e.stderr.strip() or f'git {args[0]} failed')



def parse_tag(name):
    m = TAG_RE.match(name)
    return tuple(int(x) for x in m.groups()) if m else None


def scan_tags(repo):
    names = git(repo, 'tag', '--list').split()
    merged = set(git(repo, 'tag', '--list', '--merged', 'HEAD').split())

    valid, ignored = [], []
    for name in names:
        ver = parse_tag(name)
        if ver is None:
            ignored.append(name)
        else:
            valid.append((ver, name))
    valid.sort()

    reachable = [(v, n) for v, n in valid if n in merged]
    latest = reachable[-1] if reachable else None
    newer = [n for v, n in valid
             if n not in merged and (latest is None or v > latest[0])]
    return latest, ignored, newer



def read_commits(repo, since):
    rng = f'{since}..HEAD' if since else 'HEAD'
    out = git(repo, 'log', '--no-merges', '--format=%h%x1f%s%x1f%b%x1e', rng)
    commits = []
    for chunk in out.split('\x1e'):
        chunk = chunk.strip('\n')
        if not chunk:
            continue
        sha, subject, body = (chunk.split('\x1f') + ['', ''])[:3]
        commits.append(classify(sha, subject.strip(), body))
    return commits


def classify(sha, header, body):
    c = {'sha': sha, 'header': header, 'type': None, 'scope': None,
         'subject': header, 'pr': None, 'breaking': False, 'level': 'none'}
    m = HEADER_RE.match(header)
    if not m:
        return c
    subject = m.group('subject')
    pr = PR_RE.search(subject)
    if pr:
        c['pr'] = int(pr.group(1))
        subject = subject[:pr.start()]
    c.update(type=m.group('type').lower(), scope=m.group('scope') or None,
             subject=subject,
             breaking=bool(m.group('bang') or BREAKING_FOOTER_RE.search(body)))
    c['level'] = 'major' if c['breaking'] else TYPE_LEVEL.get(c['type'], 'none')
    return c


def section_of(c):
    if c['type'] is None:
        return None
    if c['breaking']:
        return 'breaking'
    if c['type'] in ('feat', 'fix', 'perf'):
        return c['type']
    if c['type'] in OTHER_TYPES:
        return 'other'
    return None



def bump(base, level):
    major, minor, patch = base
    if level == 'major':
        return major + 1, 0, 0
    if level == 'minor':
        return major, minor + 1, 0
    if level == 'patch':
        return major, minor, patch + 1
    return base


def plan(repo, forced):
    latest, ignored, newer = scan_tags(repo)
    base = latest[0] if latest else (0, 0, 0)
    commits = read_commits(repo, latest[1] if latest else None)

    level = max((c['level'] for c in commits), key=LEVELS.index, default='none')
    drivers = [c for c in commits if c['level'] == level and level != 'none']
    notes = []

    if level == 'major' and base[0] == 0:
        level = 'minor'
        notes.append('pre-1.0: breaking change bumps MINOR, not MAJOR')
    if latest is None and level in ('none', 'patch') and commits and not forced:
        level = 'minor'
        notes.append('no previous tag: first release is v0.1.0')
    if forced:
        notes.append(f'bump forced to {forced} (computed: {level})')
        level = forced

    nxt = bump(base, level)
    return {
        'current': latest[1] if latest else None,
        'next': None if level == 'none' else 'v%d.%d.%d' % nxt,
        'level': level,
        'drivers': [c['sha'] for c in drivers],
        'notes': notes,
        'commits': commits,
        'ignored_tags': ignored,
        'unreachable_newer_tags': newer,
        'compare_url': compare_url(repo, latest[1] if latest else None,
                                   'v%d.%d.%d' % nxt),
    }


def compare_url(repo, prev, nxt):
    if not prev:
        return None
    try:
        url = subprocess.run(['git', '-C', repo, 'remote', 'get-url', 'origin'],
                             capture_output=True, text=True,
                             check=True).stdout.strip()
    except (FileNotFoundError, subprocess.CalledProcessError):
        return None
    m = re.match(r'^(?:git@github\.com:|https://github\.com/)'
                 r'([^/]+/[^/]+?)(?:\.git)?/?$', url)
    return f'https://github.com/{m.group(1)}/compare/{prev}...{nxt}' if m else None



def bullet(c):
    text = f"{c['scope']}: {c['subject']}" if c['scope'] else c['subject']
    return f"- {text} (#{c['pr']})" if c['pr'] else f'- {text}'


def render_notes(p):
    lines = [SUMMARY_PLACEHOLDER]
    for key, title in SECTIONS:
        items = [c for c in p['commits'] if section_of(c) == key]
        if items:
            lines += ['', f'## {title}'] + [bullet(c) for c in items]
    if p['compare_url']:
        lines += ['', f"Full diff: {p['compare_url']}"]
    return '\n'.join(lines) + '\n'


def print_next(p):
    since = p['current'] or '(no tag)'
    print(f"current: {since}")
    if p['next']:
        print(f"next:    {p['next']}  ({p['level']})")
    else:
        print('next:    — nothing release-worthy')
    for n in p['notes']:
        print(f'         {n}')
    print(f"commits: {len(p['commits'])} since {since}\n")

    groups = [(title, [c for c in p['commits'] if section_of(c) == key])
              for key, title in SECTIONS]
    omitted = [c for c in p['commits']
               if c['type'] is not None and section_of(c) is None]
    unparsed = [c for c in p['commits'] if c['type'] is None]
    groups += [('Omitted from notes (build/ci/chore/test)', omitted),
               ('Not conventional — read the diff and decide', unparsed)]
    for title, items in groups:
        if not items:
            continue
        print(f'{title} ({len(items)})')
        for c in items:
            mark = ' *' if c['sha'] in p['drivers'] else ''
            print(f"   {c['sha']}  {c['header']}{mark}")
        print()
    if p['drivers']:
        print('* drives the bump\n')

    if p['unreachable_newer_tags']:
        print('WARN tags newer than the base are not ancestors of HEAD: '
              + ', '.join(p['unreachable_newer_tags'])
              + ' — wrong branch, or main rewritten after a release')
    if p['ignored_tags']:
        print('WARN ignored non-semver tags: ' + ', '.join(p['ignored_tags']))



SLOP_ERRORS = [
    r'excit(?:ed|ing)', r'thrilled', r'delighted',
    r'(?:happy|pleased|proud) to', r'seamless(?:ly)?', r'cutting[- ]edge',
    r'game[- ]chang\w*', r'revolutionar\w*', r'elevat\w*', r'empower\w*',
    r'unlock\w*', r'supercharg\w*', r'leverag\w*', r'delv\w*', r'journey',
    r'landscape', r'ever[- ]evolving', r'stay tuned', r'and (?:much )?more',
    r'as always', r'under the hood', r'a host of', r'wide range',
    r'next level', r'best[- ]in[- ]class', r'blazing(?:ly)?[- ]fast',
    r'lightning[- ]fast', r'major milestone', r'bug fixes and (?:\w+ )?'
    r'improvements', r'(?:various|several|minor|small|general) '
    r'(?:fixes|improvements|enhancements|tweaks|changes)',
    r'рады (?:представить|сообщить|анонсировать)', r'с радостью',
    r'долгожданн\w*', r'революционн\w*', r'бесшовн\w*',
    r'и многое другое', r'(?:различные|мелкие|небольшие) '
    r'(?:улучшения|исправления|доработки)',
]
SLOP_WARNINGS = [
    r'robust', r'comprehensive', r'powerful', r'significantly', r'greatly',
    r'enhance(?:d|s|ments?)?', r'streamlin\w*', r'various', r'numerous',
    r"we(?:'ve| have| are)", r'our', r'мощн\w*', r'значительно',
    r'существенно',
]
SLOP_ERR_RE = re.compile(r'\b(?:' + '|'.join(SLOP_ERRORS) + r')\b', re.I)
SLOP_WARN_RE = re.compile(r'\b(?:' + '|'.join(SLOP_WARNINGS) + r')\b', re.I)

SUMMARY_FILLER_RE = re.compile(r'^(?:this|the|in this) (?:release|version|'
                               r'update)\b|^(?:этот|данный|в этом) релиз', re.I)

VAGUE_BULLET_RE = re.compile(r'^(?:[\w-]+: )?(?:misc|various|minor|small|some|'
                             r'other|general)\b|^(?:[\w-]+: )?(?:fixes|'
                             r'improvements|updates|changes|cleanup|tweaks)'
                             r'[\s.]*$', re.I)

EMOJI_RE = re.compile('[\U0001F000-\U0001FAFF☀-➿⭐⬆'
                      '↔-⇿️]')

EXCLAIM_RE = re.compile(r'\w!(?=\s|$)')

AI_TRACE_RE = re.compile(r'generated with\s+\[?(?:claude|codex|copilot)'
                         r'|co-authored-by:|claude-session:'
                         r'|https?://claude\.(?:ai|com)/', re.I)

PLACEHOLDER_RE = re.compile(r'<[^<>\n]*(?:sentence|summary|todo|change)[^<>\n]*>'
                            r'|\bTODO\b|\bTBD\b', re.I)

CODE_RE = re.compile(r'`[^`\n]*`')


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


def lint_notes(text):
    lines = text.replace('\r\n', '\n').split('\n')
    out = []

    first = next((i for i, ln in enumerate(lines) if ln.strip()), None)
    if first is None:
        return [err(1, 'empty-notes', 'release notes are empty',
                    'run `release.py notes` for the skeleton')]
    out += lint_summary(first + 1, lines[first].strip())

    section, section_line, bullets, seen = None, None, 0, {}
    for i, raw in enumerate(lines, start=1):
        line = raw.strip()
        plain = CODE_RE.sub('', line)

        heading = re.match(r'^(#+)\s*(.*)$', line)
        if heading:
            if section is not None and bullets == 0:
                out.append(empty_section(section_line, section))
            section, section_line, bullets = heading.group(2), i, 0
            if heading.group(1) != '##' or section not in SECTION_TITLES:
                out.append(warn(i, 'unknown-section',
                                f'heading {line!r} is not a standard section',
                                '`## ` + one of: '
                                + ', '.join(sorted(SECTION_TITLES))))
        elif re.match(r'^[-*]\s+', line):
            bullets += 1
            item = re.sub(r'^[-*]\s+', '', line)
            if VAGUE_BULLET_RE.match(item):
                out.append(err(i, 'bullet-vague', f'bullet says nothing: {item!r}',
                               'name the component and the change'))
            key = item.lower()
            if key in seen:
                out.append(warn(i, 'bullet-duplicate',
                                f'same bullet as line {seen[key]}',
                                'keep one'))
            seen.setdefault(key, i)

        hard = []
        for m in SLOP_ERR_RE.finditer(plain):
            hard.append(m.span())
            out.append(err(i, 'slop', f'{m.group(0)!r} sells instead of describing',
                           'say what changed and for whom, or cut it'))
        for m in SLOP_WARN_RE.finditer(plain):
            if any(a <= m.start() < b for a, b in hard):
                continue
            out.append(warn(i, 'slop-soft', f'{m.group(0)!r} is vague',
                            'replace with the concrete effect, or cut it'))
        if EMOJI_RE.search(plain):
            out.append(err(i, 'emoji', 'emoji in release notes',
                           'remove it — headings carry the structure'))
        if EXCLAIM_RE.search(plain):
            out.append(err(i, 'exclamation', 'exclamation mark',
                           'state the fact; the change speaks for itself'))
        if AI_TRACE_RE.search(line):
            out.append(err(i, 'ai-attribution', f'AI-attribution trace: {line!r}',
                           'remove it (git-conventions.md#no-ai-attribution)'))
        if PLACEHOLDER_RE.search(line):
            out.append(err(i, 'placeholder', f'unfilled placeholder: {line!r}',
                           'write the actual text'))

    if section is not None and bullets == 0:
        out.append(empty_section(section_line, section))

    out.sort(key=lambda f: (f.line, f.severity != 'ERROR'))
    return out


def empty_section(line, title):
    return err(line, 'empty-section', f'section {title!r} has no bullets',
               'drop the heading')


def lint_summary(n, line):
    if re.match(r'^(?:#|[-*]\s)', line):
        return [err(n, 'summary-missing',
                    'notes start with a heading or bullet, not a summary line',
                    'open with one sentence: the main user-visible change')]
    out = []
    if len(line) > SUMMARY_MAX:
        out.append(err(n, 'summary-too-long',
                       f'summary is {len(line)} chars, limit is {SUMMARY_MAX}',
                       'one change, one sentence — details go in the bullets'))
    if SUMMARY_FILLER_RE.match(line):
        out.append(err(n, 'summary-filler',
                       'summary opens with "this release…"',
                       'start with the change itself: '
                       '"`install` now removes hooks on uninstall."'))
    if len(re.findall(r'[.!?](?:\s|$)', CODE_RE.sub('', line))) > 1:
        out.append(warn(n, 'summary-multi-sentence',
                        'summary has more than one sentence',
                        'keep one; move the rest into bullets'))
    return out


def report_lint(findings, as_json):
    errors = sum(1 for f in findings if f.severity == 'ERROR')
    warns = len(findings) - errors
    if as_json:
        print(json.dumps({'errors': errors, 'warnings': warns,
                          'findings': [f.as_dict() for f in findings]},
                         ensure_ascii=False, indent=2))
    elif not findings:
        print('OK — release notes are clean')
    else:
        for f in findings:
            print(f)
        print(f'\n{errors} error(s), {warns} warning(s)')
    return errors, warns



def take(argv, flag):
    if flag not in argv:
        return None
    i = argv.index(flag)
    if i + 1 >= len(argv):
        die(f'{flag} needs a value')
    value = argv[i + 1]
    del argv[i:i + 2]
    return value


def main():
    argv = sys.argv[1:]
    if not argv or argv[0] in ('-h', '--help'):
        print(__doc__.strip())
        sys.exit(0 if argv else 1)

    mode = argv.pop(0)
    as_json = '--json' in argv
    strict = '--strict' in argv
    argv = [a for a in argv if a not in ('--json', '--strict')]
    repo = take(argv, '--repo') or '.'
    forced = take(argv, '--bump')
    if forced is not None and forced not in LEVELS[1:]:
        die('--bump must be one of: major, minor, patch')

    if mode == 'lint':
        if len(argv) != 1:
            die('lint takes one argument: a file or -')
        if argv[0] == '-':
            text = sys.stdin.read()
        else:
            path = Path(argv[0]).expanduser()
            if not path.is_file():
                die(f'not a file: {path}')
            text = path.read_text(encoding='utf-8')
        errors, warns = report_lint(lint_notes(text), as_json)
        sys.exit(2 if errors or (strict and warns) else 0)

    if mode not in ('next', 'notes') or argv:
        die(f'unknown arguments: {" ".join([mode] + argv)} (see --help)')

    p = plan(repo, forced)
    if mode == 'notes':
        if p['next'] is None:
            die('nothing release-worthy since the last tag; pass --bump to force')
        print(render_notes(p), end='')
    elif as_json:
        print(json.dumps(p, ensure_ascii=False, indent=2))
    else:
        print_next(p)
    sys.exit(3 if p['next'] is None else 0)


if __name__ == '__main__':
    main()
