#!/usr/bin/env python3
import argparse
import datetime
import difflib
import json
import re
import sys
from pathlib import Path

LICENSES_DIR = Path(__file__).resolve().parent.parent / 'references' / 'licenses'
BUILTIN_PLACEHOLDERS = ('[year]', '[fullname]')
OPTIONAL_TAIL = {'Apache-2.0': 'appendix'}
DETECT_MIN_RATIO = 0.85
EXCERPT_WORDS = 10
MAX_DIFFS = 5

LICENSE_FILE_RE = re.compile(r'^(?:licen[cs]e|copying)(?:[.-].*)?$', re.I)
WORD_RE = re.compile(r'\w+|[^\w\s]')
DROP_TOKEN_RE = re.compile(r'^[#*>_=`|-]$')
COPYRIGHT_SIGN_RE = re.compile(r'\(c\)|©', re.I)
QUOTES = str.maketrans({'“': '"', '”': '"', '„': '"', '«': '"', '»': '"',
                        '‘': "'", '’': "'", '–': '-', '—': '-'})
HEADING_RE = re.compile(r'^ {0,3}(#{1,6})[ \t]+(.*?)[ \t#]*$')
LICENSE_HEADING_RE = re.compile(r'\blicen[cs]e\b|лицензи', re.I)
LINK_TARGET_RE = re.compile(r'\]\(\s*<?([^)\s>]+)|href\s*=\s*["\']([^"\']+)', re.I)

ALIASES = {
    'MIT': [r'MIT(?!-0)'],
    'Apache-2.0': [r'Apache[- ](?:License[, ]*)?(?:Version |v)?2(?:\.0)?'],
    'BSD-2-Clause': [r'BSD[- ]2[- ]Clause', r'2[- ]clause BSD', r'Simplified BSD', r'FreeBSD License'],
    'BSD-3-Clause': [r'BSD[- ]3[- ]Clause', r'3[- ]clause BSD', r'(?:New|Modified|Revised) BSD'],
    'ISC': [r'ISC'],
    'MPL-2.0': [r'MPL[- ]?v?2(?:\.0)?', r'Mozilla Public License,? (?:Version |v\.? ?)?2(?:\.0)?'],
    'LGPL-2.1': [r'LGPL[- ]?v?2\.1', r'(?:GNU )?Lesser General Public License,? (?:Version |v)?2\.1'],
    'LGPL-3.0': [r'LGPL[- ]?v?3(?:\.0)?', r'(?:GNU )?Lesser General Public License,? (?:Version |v)?3(?:\.0)?'],
    'GPL-2.0': [r'GPL[- ]?v?2(?:\.0)?', r'GNU General Public License,? (?:Version |v)?2(?:\.0)?'],
    'GPL-3.0': [r'GPL[- ]?v?3(?:\.0)?', r'GNU General Public License,? (?:Version |v)?3(?:\.0)?'],
    'AGPL-3.0': [r'AGPL[- ]?v?3(?:\.0)?', r'(?:GNU )?Affero General Public License,? (?:Version |v)?3(?:\.0)?'],
    'Unlicense': [r'(?:The )?Unlicense'],
    'CC0-1.0': [r'CC0(?:[- ]1\.0)?'],
    'BSL-1.0': [r'BSL[- ]1\.0', r'Boost Software License'],
    'EPL-2.0': [r'EPL[- ]?v?2(?:\.0)?', r'Eclipse Public License,? (?:- )?(?:Version |v)?2(?:\.0)?'],
}
ALIAS_RE = {k: re.compile(r'(?<![\w-])(?:' + '|'.join(v) + r')(?!\w|\.\d)', re.I)
            for k, v in ALIASES.items()}

MANIFESTS = (
    ('package.json', re.compile(r'"license"\s*:\s*"([^"]+)"')),
    ('composer.json', re.compile(r'"license"\s*:\s*(?:"([^"]+)"|\[([^\]]+)\])')),
    ('pyproject.toml', re.compile(r'^\s*license(?:-expression)?\s*=\s*(?:"([^"]+)"|\{\s*text\s*=\s*"([^"]+)")', re.M)),
    ('Cargo.toml', re.compile(r'^\s*license\s*=\s*"([^"]+)"', re.M)),
    ('*.gemspec', re.compile(r'\.licen[cs]es?\s*=\s*(?:\[)?\s*["\']([^"\']+)')),
)


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


def tokenize(text, placeholders=()):
    marks = {}
    for i, ph in enumerate(placeholders):
        mark = f'\x00{i}\x00'
        if ph in text:
            marks[mark] = ph
            text = text.replace(ph, f' {mark} ')
    text = COPYRIGHT_SIGN_RE.sub(' ', text.translate(QUOTES))
    out = []
    for n, line in enumerate(text.splitlines(), 1):
        for word in line.split():
            if word in marks:
                out.append((marks[word], n, True))
                continue
            for w in WORD_RE.findall(word.lower()):
                if not DROP_TOKEN_RE.match(w):
                    out.append((w, n, False))
    return out


def bundled():
    return {p.stem: p for p in sorted(LICENSES_DIR.glob('*.txt'))}


def find_bundled(spdx):
    key = re.sub(r'-(?:only|or-later)$', '', spdx, flags=re.I).rstrip('+').lower()
    return next(((k, p) for k, p in bundled().items() if k.lower() == key), (None, None))


def words(tokens):
    return [t[0] for t in tokens]


def detect(file_tokens):
    fw = words(file_tokens)
    best = (0.0, None)
    for spdx, path in bundled().items():
        tw = words(tokenize(path.read_text(encoding='utf-8'), BUILTIN_PLACEHOLDERS))
        sm = difflib.SequenceMatcher(None, tw, fw, autojunk=False)
        if sm.real_quick_ratio() < DETECT_MIN_RATIO or sm.quick_ratio() < DETECT_MIN_RATIO:
            continue
        r = sm.ratio()
        if r > best[0]:
            best = (r, spdx)
    return best


def excerpt(tokens, i, j):
    ws = [t[0] for t in tokens[i:j]]
    s = ' '.join(ws[:EXCERPT_WORDS])
    return s + (' …' if len(ws) > EXCERPT_WORDS else '') if ws else '∅'


def compare(path, spdx, tmpl_tokens, file_tokens, add):
    tw, fw = words(tmpl_tokens), words(file_tokens)
    sm = difflib.SequenceMatcher(None, tw, fw, autojunk=False)
    tail = OPTIONAL_TAIL.get(spdx)
    diffs = 0
    year_seen = False

    def line_at(j):
        if not file_tokens:
            return 1
        return file_tokens[min(j, len(file_tokens) - 1)][1]

    for op, i1, i2, j1, j2 in sm.get_opcodes():
        span = tmpl_tokens[i1:i2]
        only_ph = bool(span) and all(t[2] for t in span)
        if op == 'equal':
            for k in range(i2 - i1):
                if tmpl_tokens[i1 + k][2]:
                    add(f'{path}:{file_tokens[j1 + k][1]}', 'ERROR', 'placeholder',
                        f'{tmpl_tokens[i1 + k][0]} is not filled',
                        'replace it with the real value (ask the user for the holder)')
            continue
        if op == 'replace' and only_ph:
            value = ' '.join(t[0] for t in file_tokens[j1:j2])
            raw = value.replace(' ', '')
            left = [t[0] for t in span if t[0].lower() in raw]
            if left:
                add(f'{path}:{line_at(j1)}', 'ERROR', 'placeholder',
                    f'{" ".join(left)} is not filled',
                    'replace it with the real value (ask the user for the holder)')
                continue
            if '[year]' in [t[0] for t in span]:
                year_seen = True
                check_year(f'{path}:{line_at(j1)}', value, add)
            continue
        if op == 'delete' and only_ph:
            add(f'{path}:{line_at(j1)}', 'ERROR', 'placeholder',
                f'{" ".join(t[0] for t in span)} is missing', 'fill in the copyright line')
            continue
        if op == 'delete' and tail and i2 == len(tw) and tail in tw[i1:i1 + 3]:
            continue
        diffs += 1
        if diffs <= MAX_DIFFS:
            add(f'{path}:{line_at(j1)}', 'ERROR', 'text-diff',
                f'expected «{excerpt(tmpl_tokens, i1, i2)}», found «{excerpt(file_tokens, j1, j2)}»',
                f'restore the canonical {spdx} text; terms are never edited')
    if diffs > MAX_DIFFS:
        add(path, 'ERROR', 'text-diff', f'{diffs - MAX_DIFFS} more differences', 'same fix')
    return year_seen


def check_year(where, value, add):
    years = [int(y) for y in re.findall(r'\b(1[89]\d\d|2\d\d\d)\b', value)]
    if not years:
        add(where, 'ERROR', 'year', f'no year in «{value}»', 'put the year of first publication')
    elif max(years) > datetime.date.today().year:
        add(where, 'WARN', 'year', f'year {max(years)} is in the future', 'use the publication year')


def mentioned(text):
    return {k for k, rx in ALIAS_RE.items() if rx.search(text)}


def license_section(text):
    lines = text.splitlines()
    for i, line in enumerate(lines):
        m = HEADING_RE.match(line)
        if m and LICENSE_HEADING_RE.search(m.group(2)):
            level = len(m.group(1))
            body = []
            for j in range(i + 1, len(lines)):
                m2 = HEADING_RE.match(lines[j])
                if m2 and len(m2.group(1)) <= level:
                    break
                body.append(lines[j])
            return i + 1, '\n'.join(body)
    return None, None


def check_readme(readme, ids, files, add):
    try:
        text = readme.read_text(encoding='utf-8')
    except OSError as e:
        print(f'error: {e}', file=sys.stderr)
        sys.exit(1)
    line, body = license_section(text)
    if line is None:
        add(f'{readme}:1', 'WARN', 'readme-section', 'no License section',
            'add `## License` with the SPDX id linked to the license file')
        return
    where = f'{readme}:{line}'
    found = mentioned(body)
    if found and not found & ids:
        add(where, 'ERROR', 'readme-mismatch',
            f'README names {", ".join(sorted(found))}, license file is {", ".join(sorted(ids))}',
            'make them agree; never guess which one is right — ask')
    else:
        for spdx in sorted(ids - found):
            add(where, 'WARN', 'readme-unnamed', f'License section does not name {spdx}',
                f'state the SPDX id: [{spdx}]({files[0].name})')
        for spdx in sorted(found - ids):
            add(where, 'WARN', 'readme-extra', f'License section also names {spdx}',
                'dual licensing needs a license file per license; otherwise drop it')
    targets = {Path(a or b).name for a, b in LINK_TARGET_RE.findall(body)}
    if not targets & {f.name for f in files}:
        add(where, 'WARN', 'readme-link', 'License section does not link the license file',
            f'link it: [{sorted(ids)[0] if ids else "License"}]({files[0].name})')


def check_manifests(root, ids, add):
    for pattern, rx in MANIFESTS:
        for path in sorted(root.glob(pattern)):
            try:
                text = path.read_text(encoding='utf-8')
            except OSError:
                continue
            m = rx.search(text)
            if not m:
                continue
            declared = next(g for g in m.groups() if g)
            found = mentioned(declared)
            if found and not found & ids:
                add(f'{path}', 'ERROR', 'manifest-mismatch',
                    f'declares {declared}, license file is {", ".join(sorted(ids))}',
                    'report it to the user; do not pick one silently')
            elif found and ids - found:
                add(f'{path}', 'WARN', 'manifest-mismatch',
                    f'declares {declared}, license files also carry {", ".join(sorted(ids - found))}',
                    'confirm the expression covers every license file')


def discover(root):
    return sorted(p for p in root.iterdir() if p.is_file() and LICENSE_FILE_RE.match(p.name))


def check_file(path, spdx, canonical, placeholders, add):
    try:
        text = path.read_text(encoding='utf-8')
    except OSError as e:
        print(f'error: {e}', file=sys.stderr)
        sys.exit(1)
    file_tokens = tokenize(text)
    if canonical:
        tmpl_text = canonical.read_text(encoding='utf-8')
        spdx = spdx or 'custom'
    elif spdx:
        key, tmpl_path = find_bundled(spdx)
        if not tmpl_path:
            print(f'error: {spdx} is not bundled; fetch the canonical text and pass --canonical FILE',
                  file=sys.stderr)
            sys.exit(1)
        spdx, tmpl_text = key, tmpl_path.read_text(encoding='utf-8')
    else:
        ratio, spdx = detect(file_tokens)
        if not spdx:
            add(str(path), 'ERROR', 'unknown-license',
                'text matches no bundled license',
                'pass --spdx ID, or fetch the canonical text and pass --canonical FILE')
            return None
        tmpl_text = bundled()[spdx].read_text(encoding='utf-8')
    tmpl_tokens = tokenize(tmpl_text, BUILTIN_PLACEHOLDERS + tuple(placeholders))
    compare(path, spdx, tmpl_tokens, file_tokens, add)
    return spdx


def main():
    ap = argparse.ArgumentParser(
        description='Check license files against canonical texts and README/manifests against docs-conventions.')
    ap.add_argument('files', nargs='*',
                    help='license files (default: LICENSE*, COPYING* in the current directory)')
    ap.add_argument('--spdx', help='expected SPDX id (default: detect from the bundled texts)')
    ap.add_argument('--canonical', type=Path, help='canonical text for a license that is not bundled')
    ap.add_argument('--placeholder', action='append', default=[],
                    help='extra placeholder in --canonical to be filled, e.g. "<year>"')
    ap.add_argument('--readme', action='append', type=Path,
                    help='README to check (default: README.md and README_RUS.md next to the license)')
    ap.add_argument('--no-readme', action='store_true')
    ap.add_argument('--no-manifest', action='store_true')
    ap.add_argument('--list', action='store_true', help='list bundled SPDX ids and exit')
    ap.add_argument('--json', action='store_true')
    ap.add_argument('--strict', action='store_true', help='warnings fail too')
    args = ap.parse_args()

    if args.list:
        print('\n'.join(bundled()))
        return

    findings = []

    def add(where, sev, code, problem, fix):
        findings.append(Finding(where, sev, code, problem, fix))

    files = [Path(f) for f in args.files] or discover(Path('.'))
    if (args.spdx or args.canonical) and len(files) > 1:
        print('error: --spdx/--canonical apply to one license file at a time', file=sys.stderr)
        sys.exit(1)
    ids = set()
    if not files:
        add('.', 'ERROR', 'no-license', 'no LICENSE or COPYING file',
            'add one with docs-license; without it the code is all rights reserved')
    for f in files:
        spdx = check_file(f, args.spdx, args.canonical, args.placeholder, add)
        if spdx:
            ids.add(spdx)
            print(f'{f}: {spdx}', file=sys.stderr)
    if 'LGPL-3.0' in ids and 'GPL-3.0' not in ids:
        add(str(files[0]), 'WARN', 'lgpl-without-gpl', 'LGPL-3.0 without the GPL-3.0 text',
            'add COPYING with GPL-3.0; LGPL-3.0 only adds permissions on top of it')

    root = files[0].parent if files else Path('.')
    if ids and not args.no_readme:
        readmes = args.readme or [p for p in (root / 'README.md', root / 'README_RUS.md') if p.exists()]
        for r in readmes:
            check_readme(r, ids, files, add)
    if ids and not args.no_manifest:
        check_manifests(root, ids, add)

    findings.sort(key=lambda f: f.severity != 'ERROR')
    if args.json:
        print(json.dumps([f.as_dict() for f in findings], ensure_ascii=False, indent=2))
    elif not findings:
        print('OK')
    else:
        for f in findings:
            print(f)

    errors = sum(1 for f in findings if f.severity == 'ERROR')
    warns = len(findings) - errors
    sys.exit(2 if errors or (args.strict and warns) else 0)


if __name__ == '__main__':
    main()
