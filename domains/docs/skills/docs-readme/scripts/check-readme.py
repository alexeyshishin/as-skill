#!/usr/bin/env python3
import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path
from urllib.parse import unquote

MAX_PER_CODE = 5
BADGE_MAX = 3
OPENER_MAX_WORDS = 50
UNTRANSLATED_MIN_WORDS = 6
SHELL_LANGS = {'bash', 'sh', 'shell', 'zsh'}

FENCE_OPEN_RE = re.compile(r'^(\s*)(`{3,}|~{3,})(.*)$')
FENCE_CLOSE_RE = re.compile(r'^\s*(`{3,}|~{3,})\s*$')
ATX_RE = re.compile(r'^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$')
SETEXT_RE = re.compile(r'^ {0,3}(=+|-+)\s*$')
TABLE_DELIM_RE = re.compile(r'^\s*\|?\s*:?-+:?\s*(?:\|\s*:?-+:?\s*)*\|?\s*$')
LIST_RE = re.compile(r'^\s*(?:[-*+]|\d+[.)])\s+')
CODE_SPAN_RE = re.compile(r'(`+)(.+?)\1')
IMG_MD_RE = re.compile(r'!\[([^\]]*)\]\(\s*<?([^)\s>]*)>?(?:\s+["\'(][^)]*)?\s*\)')
LINK_MD_RE = re.compile(r'(?<!!)\[([^\]]*)\]\(\s*<?([^)\s>]*)>?(?:\s+["\'(][^)]*)?\s*\)')
REF_DEF_RE = re.compile(r'^ {0,3}\[([^\]]+)\]:\s*<?(\S+?)>?(?:\s+.*)?$')
HTML_IMG_RE = re.compile(r'<img\b[^>]*>', re.I)
HTML_A_RE = re.compile(r'<a\b[^>]*?\bhref\s*=\s*["\']([^"\']*)["\']', re.I)
HTML_ID_RE = re.compile(r'<[a-z][^>]*?\b(?:id|name)\s*=\s*["\']([^"\']+)["\']', re.I)
HTML_H1_RE = re.compile(r'<h1\b', re.I)
SRC_RE = re.compile(r'\bsrc\s*=\s*["\']([^"\']*)["\']', re.I)
ALT_RE = re.compile(r'\balt\s*=\s*["\']([^"\']*)["\']', re.I)
SCHEME_RE = re.compile(r'^[a-zA-Z][a-zA-Z0-9+.-]*:|^//')
URL_RE = re.compile(r'https?://\S+')
BADGE_RE = re.compile(r'shields\.io|badgen\.net|badge', re.I)
CODE_COMMENT_RE = re.compile(r'^\s*(?:#|//)\s.*$|\s+(?:#|//)\s.*$')
CODE_PLACEHOLDER_RE = re.compile(r'<[^<>\s/]+>')
CYR_RE = re.compile(r'[А-Яа-яЁё]')
LATIN_WORD_RE = re.compile(r'\b[A-Za-z]{2,}\b')

GENERIC_ALT = {'image', 'img', 'screenshot', 'picture', 'photo', 'pic',
               'изображение', 'картинка', 'скриншот', 'скрин', 'фото', 'рисунок'}

GENERIC_OPENER_RE = re.compile(
    r'^\W*(?:welcome\b|hello\b|hi\b|this (?:is|repo|repository|project)\b'
    r'|добро пожаловать|привет\b|(?:это|данный|этот) (?:мой |наш )?(?:проект|репозиторий))',
    re.I)

MARKETING_RE = re.compile(
    r'\b(?:blazing(?:ly)?[- ]fast|lightning[- ]fast|seamless(?:ly)?|revolutionary'
    r'|cutting[- ]edge|state[- ]of[- ]the[- ]art|world[- ]class|next[- ]gen(?:eration)?'
    r'|game[- ]chang\w*|supercharg\w*|effortless(?:ly)?|powerful|best[- ]in[- ]class'
    r'|unparalleled)\b'
    r'|революционн\w*|молниеносн\w*|бесшовн\w*|мощн(?:ый|ая|ое|ые|ого|ым|ой)\b'
    r'|передов\w*|непревзойд\w*',
    re.I)

PLACEHOLDER_RE = re.compile(r'\b(?:TODO|TBD|FIXME)\b|(?i:lorem ipsum)')

INSTALL_RE = re.compile(
    r'\b(?:install(?:ation|ing)?|setup|set up|getting started|quick ?start'
    r'|prerequisites|requirements)\b|установк|быстрый старт|начало работы|требован',
    re.I)
USAGE_RE = re.compile(
    r'\b(?:usage|using|quick ?start|getting started|examples?|how to use|running)\b'
    r'|использ|быстрый старт|начало работы|пример|запуск',
    re.I)
LICENSE_RE = re.compile(r'\blicen[cs]e\b|лицензи', re.I)


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


class Heading:
    def __init__(self, line, level, text):
        self.line = line
        self.level = level
        self.text = text


class Fence:
    def __init__(self, line, char, length, lang):
        self.line = line
        self.end = None
        self.char = char
        self.length = length
        self.lang = lang
        self.body = []


class Doc:
    def __init__(self, path, text):
        self.path = path
        self.lines = text.splitlines()
        self.headings = []
        self.fences = []
        self.tables = []
        self.prose = []
        self.images = []
        self.links = []
        self.html_ids = set()
        self.skip_lines = set()
        self.has_html_h1 = False
        self._parse()
        self.anchors = self._anchors()

    def _parse(self):
        fence = None
        in_comment = False
        prev_kind = 'blank'
        table = None
        for n, raw in enumerate(self.lines, 1):
            if fence:
                self.skip_lines.add(n)
                m = FENCE_CLOSE_RE.match(raw)
                if m and m.group(1)[0] == fence.char and len(m.group(1)) >= fence.length:
                    fence.end = n
                    fence = None
                else:
                    fence.body.append(raw.rstrip())
                continue
            line = raw
            if in_comment:
                self.skip_lines.add(n)
                if '-->' in line:
                    in_comment = False
                continue
            line = re.sub(r'<!--.*?-->', '', line)
            if '<!--' in line:
                line = line.split('<!--', 1)[0]
                in_comment = True
            if table and ('|' not in line or not line.strip()):
                table = None
            m = FENCE_OPEN_RE.match(line)
            if m and not (m.group(2)[0] == '`' and '`' in m.group(3)):
                info = m.group(3).strip()
                fence = Fence(n, m.group(2)[0], len(m.group(2)), info.split()[0] if info else '')
                self.fences.append(fence)
                self.skip_lines.add(n)
                prev_kind = 'fence'
                continue
            if not line.strip():
                prev_kind = 'blank'
                continue
            if HTML_H1_RE.search(line):
                self.has_html_h1 = True
            m = ATX_RE.match(line)
            if m:
                text = re.sub(r'[ \t]+#+$', '', m.group(2) or '')
                self.headings.append(Heading(n, len(m.group(1)), text))
                self._inline(n, line, 'heading')
                prev_kind = 'heading'
                continue
            m = SETEXT_RE.match(line)
            if m and prev_kind == 'text':
                prev = self.lines[n - 2]
                self.headings.append(Heading(n - 1, 1 if m.group(1)[0] == '=' else 2, prev.strip()))
                self.prose = [p for p in self.prose if p[0] != n - 1]
                self.prose.append((n - 1, prev, 'heading'))
                self.skip_lines.add(n)
                prev_kind = 'heading'
                continue
            if '|' in line and TABLE_DELIM_RE.match(line) and prev_kind in ('text', 'table') \
                    and '|' in self.lines[n - 2]:
                cols = len([c for c in line.strip().strip('|').split('|')])
                table = [n - 1, 1, cols]
                self.tables.append(table)
                self.prose = [(p[0], p[1], 'table') if p[0] == n - 1 else p for p in self.prose]
                prev_kind = 'table'
                continue
            if table:
                table[1] += 1
                self._inline(n, line, 'table')
                prev_kind = 'table'
                continue
            m = REF_DEF_RE.match(line)
            if m:
                self.links.append((n, m.group(2)))
                prev_kind = 'refdef'
                continue
            kind = 'html' if line.lstrip().startswith('<') else 'text'
            self._inline(n, line, kind)
            prev_kind = 'list' if LIST_RE.match(line) else kind

    def _inline(self, n, line, kind):
        self.prose.append((n, line, kind))
        s = CODE_SPAN_RE.sub(lambda m: ' ' * len(m.group(0)), line)
        for m in IMG_MD_RE.finditer(s):
            self.images.append((n, m.group(1), m.group(2)))
        for tag in HTML_IMG_RE.findall(s):
            src = SRC_RE.search(tag)
            alt = ALT_RE.search(tag)
            self.images.append((n, alt.group(1) if alt else None, src.group(1) if src else ''))
        s2 = IMG_MD_RE.sub('IMG', s)
        for m in LINK_MD_RE.finditer(s2):
            self.links.append((n, m.group(2)))
        for url in HTML_A_RE.findall(s):
            self.links.append((n, url))
        for ident in HTML_ID_RE.findall(s):
            self.html_ids.add(ident.lower())

    def _anchors(self):
        seen = Counter()
        out = set()
        for h in self.headings:
            slug = slugify(h.text)
            out.add(f'{slug}-{seen[slug]}' if seen[slug] else slug)
            seen[slug] += 1
        return out | self.html_ids

    def heading_lines(self):
        return {h.line for h in self.headings}


def plain(text):
    text = CODE_SPAN_RE.sub(lambda m: m.group(2), text)
    text = IMG_MD_RE.sub('', text)
    text = LINK_MD_RE.sub(lambda m: m.group(1), text)
    text = re.sub(r'<[^>]+>', '', text)
    text = re.sub(r'[*~]+', '', text)
    return re.sub(r'(?<!\w)_+|_+(?!\w)', '', text)


def slugify(text):
    s = plain(text).strip().lower()
    s = re.sub(r'[^\w\- ]', '', s)
    return s.replace(' ', '-')


def repo_root(start):
    for d in [start, *start.parents]:
        if (d / '.git').exists():
            return d
    return start


def is_badge(src):
    return bool(BADGE_RE.search(src or ''))


def check_fences(doc, add):
    for f in doc.fences:
        if f.end is None:
            add(f.line, 'ERROR', 'fence-unclosed', 'code block is never closed',
                f'close it with {f.char * f.length}')
        if not f.lang:
            add(f.line, 'ERROR', 'fence-lang', 'code block has no language',
                'name it: bash, json, yaml, toml, go, text for output')
        if f.lang.lower() in SHELL_LANGS and any(re.match(r'^\s*\$\s', b) for b in f.body):
            add(f.line, 'WARN', 'shell-prompt', 'commands start with a `$` prompt',
                'drop the prompt so the block copy-pastes; move output to a `text` block')


def check_headings(doc, add):
    h1 = [h for h in doc.headings if h.level == 1]
    if not h1 and not doc.has_html_h1:
        add(1, 'WARN', 'no-title', 'no `#` title', 'start with `# <project name>`')
    for h in h1[1:]:
        add(h.line, 'WARN', 'multiple-h1', 'more than one `#` heading',
            'keep one title, make sections `##`')
    prev = None
    for h in doc.headings:
        if prev and h.level > prev + 1:
            add(h.line, 'WARN', 'heading-skip', f'heading jumps from level {prev} to {h.level}',
                f'use level {prev + 1}')
        prev = h.level
    content = {n for n, _, kind in doc.prose if kind != 'heading'} | {f.line for f in doc.fences}
    starts = [h.line for h in doc.headings] + [len(doc.lines) + 1]
    for h, nxt in zip(doc.headings, starts[1:]):
        if h.level > 1 and not any(h.line < n < nxt for n in content):
            add(h.line, 'WARN', 'empty-section', f'section "{plain(h.text).strip()}" is empty',
                'fill it or drop the heading')


def check_opener(doc, add):
    h1 = next((h for h in doc.headings if h.level == 1), None)
    start = h1.line if h1 else 0
    headings = doc.heading_lines()
    fence_starts = {f.line for f in doc.fences}
    for n in range(start + 1, len(doc.lines) + 1):
        if n in fence_starts or n in headings:
            add(n, 'ERROR', 'no-opener',
                'no opening line before the first section or code block',
                'add one sentence under the title: what it does and for whom')
            return
        if n in doc.skip_lines:
            continue
        text = doc.lines[n - 1].strip()
        if not text or text.startswith('<'):
            continue
        rest = LINK_MD_RE.sub('', IMG_MD_RE.sub('', text))
        if not re.search(r'\w', rest):
            continue
        para = []
        for m in range(n, len(doc.lines) + 1):
            t = doc.lines[m - 1].strip()
            if not t or m in headings or m in fence_starts:
                break
            para.append(t)
        body = plain(' '.join(para))
        if GENERIC_OPENER_RE.match(body):
            add(n, 'WARN', 'generic-opener', 'opening line is a generic greeting',
                'lead with what the project does and for whom, searchable nouns first')
        words = len(body.split())
        if words > OPENER_MAX_WORDS:
            add(n, 'WARN', 'long-opener', f'opening paragraph is {words} words',
                'cut it to one or two sentences, move the rest under Why')
        return


def check_sections(doc, add):
    found = {}
    for idx, h in enumerate(doc.headings):
        if h.level == 1:
            continue
        text = plain(h.text)
        for name, rx in (('Install', INSTALL_RE), ('Usage', USAGE_RE), ('License', LICENSE_RE)):
            if name not in found and rx.search(text):
                found[name] = (idx, h.line)
    for name in ('Install', 'Usage', 'License'):
        if name not in found:
            add(1, 'WARN', 'missing-section', f'no {name} section',
                f'add `## {name}` (see docs-conventions section order), or confirm it does not apply')
    if 'Install' in found and 'Usage' in found and found['Install'][0] > found['Usage'][0]:
        add(found['Install'][1], 'WARN', 'section-order', 'Install comes after Usage',
            'move Install before Usage')
    if 'License' in found:
        later = [found[k][0] for k in ('Install', 'Usage') if k in found]
        if later and found['License'][0] < max(later):
            add(found['License'][1], 'WARN', 'section-order', 'License comes before Install/Usage',
                'move License to the end')


def check_images(doc, add):
    badges = [(n, alt, src) for n, alt, src in doc.images if is_badge(src)]
    if len(badges) > BADGE_MAX:
        add(badges[BADGE_MAX][0], 'WARN', 'badges', f'{len(badges)} badges',
            f'keep at most {BADGE_MAX}: build status, version, license')
    for n, alt, src in doc.images:
        badge = is_badge(src)
        if alt is None or not alt.strip():
            add(n, 'WARN' if badge else 'ERROR', 'alt-missing', f'image `{src}` has no alt text',
                'describe what it shows')
        elif not badge and alt.strip().lower() in GENERIC_ALT:
            add(n, 'WARN', 'alt-generic', f'alt text "{alt}" says nothing',
                'describe what the image shows')


def check_links(doc, add, root):
    base = doc.path.parent
    for n, url in doc.links + [(n, src) for n, _, src in doc.images]:
        if not url or SCHEME_RE.match(url):
            continue
        path, _, frag = url.partition('#')
        path = unquote(path.split('?', 1)[0])
        frag = unquote(frag).lower()
        if not path:
            if frag and frag not in doc.anchors:
                add(n, 'ERROR', 'anchor', f'`#{frag}` matches no heading',
                    'fix the anchor to match a heading slug')
            continue
        target = (root / path.lstrip('/')) if path.startswith('/') else (base / path)
        target = target.resolve()
        try:
            target.relative_to(root.resolve())
        except ValueError:
            continue
        if not target.exists():
            add(n, 'ERROR', 'broken-link', f'`{url}` points to a missing file',
                'fix the path or drop the link')
            continue
        if frag and target.is_file() and target.suffix.lower() in ('.md', '.markdown'):
            other = Doc(target, target.read_text(encoding='utf-8', errors='replace'))
            if frag not in other.anchors:
                add(n, 'ERROR', 'anchor', f'`{url}`: no such heading in {target.name}',
                    'fix the anchor')


def check_prose(doc, add):
    for n, line, kind in doc.prose:
        s = CODE_SPAN_RE.sub(' ', line)
        s = URL_RE.sub(' ', s)
        m = PLACEHOLDER_RE.search(s)
        if m:
            add(n, 'ERROR', 'placeholder', f'leftover `{m.group(0)}`', 'fill it in or remove it')
        m = MARKETING_RE.search(s)
        if m:
            add(n, 'WARN', 'marketing', f'"{m.group(0)}" without a number',
                'replace with a measured fact or cut it')


def first_h2_line(doc):
    return next((h.line for h in doc.headings if h.level >= 2), len(doc.lines) + 1)


def check_switcher(doc, other, add):
    limit = first_h2_line(doc)
    for n, url in doc.links:
        if n < limit and Path(unquote(url.split('#', 1)[0])).name == other.path.name:
            return
    add(1, 'ERROR', 'switcher', f'no link to {other.path.name} above the first section',
        'add the language switcher right after the title (see docs-conventions)')


def code_of(fence):
    out = []
    for line in fence.body:
        line = CODE_PLACEHOLDER_RE.sub('<>', CODE_COMMENT_RE.sub('', line)).rstrip()
        if line:
            out.append(line)
    return out


def walk_pairs(a, b, key):
    for i in range(max(len(a), len(b))):
        if i >= len(a) or i >= len(b) or key(a[i]) != key(b[i]):
            return i
    return None


def check_pair(en, ru, add_en, add_ru):
    check_switcher(en, ru, add_en)
    check_switcher(ru, en, add_ru)

    i = walk_pairs(en.headings, ru.headings, lambda h: h.level)
    if i is not None:
        el =en.headings[i].line if i < len(en.headings) else len(en.lines)
        rl = ru.headings[i].line if i < len(ru.headings) else len(ru.lines)
        add_en(el, 'ERROR', 'heading-parity',
               f'headings diverge at #{i + 1} (EN {len(en.headings)}, RU {len(ru.headings)}; '
               f'RU line {rl})',
               'make RU follow EN heading for heading')

    ef, rf = en.fences, ru.fences
    if len(ef) != len(rf):
        add_en(1, 'ERROR', 'code-parity', f'EN has {len(ef)} code blocks, RU has {len(rf)}',
               'RU carries every EN code block, unchanged')
    for a, b in zip(ef, rf):
        if a.lang != b.lang or code_of(a) != code_of(b):
            add_ru(b.line, 'ERROR', 'code-parity',
                   f'code block differs from EN line {a.line}',
                   'copy the EN block; only comments and <placeholders> are translated')

    if len(en.tables) != len(ru.tables):
        add_en(1, 'ERROR', 'table-parity', f'EN has {len(en.tables)} tables, RU has {len(ru.tables)}',
               'RU carries every EN table')
    for a, b in zip(en.tables, ru.tables):
        if a[1:] != b[1:]:
            add_ru(b[0], 'ERROR', 'table-parity',
                   f'table is {b[1]}×{b[2]}, EN line {a[0]} is {a[1]}×{a[2]}',
                   'same rows and columns as EN')

    i = walk_pairs(en.images, ru.images, lambda x: x[2])
    if i is not None:
        line = ru.images[i][0] if i < len(ru.images) else len(ru.lines)
        add_ru(line, 'WARN', 'image-parity', f'images diverge from EN at #{i + 1}',
               'same images and badges in the same order')

    eu = Counter(u for _, u in en.links if SCHEME_RE.match(u or ''))
    ru_ = Counter(u for _, u in ru.links if SCHEME_RE.match(u or ''))
    missing = sorted(set(eu) - set(ru_))
    extra = sorted(set(ru_) - set(eu))
    if missing:
        add_ru(1, 'WARN', 'link-parity', f'EN links missing in RU: {", ".join(missing[:3])}',
               'carry every EN link')
    if extra:
        add_ru(1, 'WARN', 'link-parity', f'RU-only links: {", ".join(extra[:3])}',
               'add them to EN first, or drop them')

    for n, url in ru.links:
        if not url or SCHEME_RE.match(url) or url.startswith('#'):
            continue
        path = unquote(url.split('#', 1)[0].split('?', 1)[0])
        if not path.lower().endswith('.md') or Path(path).name == en.path.name:
            continue
        p = Path(path)
        alt = (ru.path.parent / p).with_name(p.stem + '_RUS' + p.suffix)
        if not p.stem.endswith('_RUS') and alt.exists():
            add_ru(n, 'WARN', 'link-ru-version', f'`{url}` has a Russian version',
                   f'link {p.stem}_RUS{p.suffix}')

    for n, line, kind in ru.prose:
        if kind not in ('text', 'table'):
            continue
        s = CODE_SPAN_RE.sub(' ', line)
        s = LINK_MD_RE.sub(lambda m: m.group(1), IMG_MD_RE.sub(' ', s))
        s = URL_RE.sub(' ', re.sub(r'<[^>]+>', ' ', s))
        if not CYR_RE.search(s) and len(LATIN_WORD_RE.findall(s)) >= UNTRANSLATED_MIN_WORDS:
            add_ru(n, 'WARN', 'untranslated', 'line has no Russian text', 'translate it')


def cap(findings):
    seen = Counter()
    out = []
    for f in findings:
        key = (f.where.rsplit(':', 1)[0], f.code)
        seen[key] += 1
        if seen[key] <= MAX_PER_CODE:
            out.append(f)
    for (path, code), n in seen.items():
        if n > MAX_PER_CODE:
            sev = next(f.severity for f in out if f.code == code and f.where.startswith(path))
            out.append(Finding(path, sev, code, f'{n - MAX_PER_CODE} more like this', 'same fix'))
    return out


def sort_key(f):
    path, _, line = f.where.rpartition(':')
    if not line.isdigit():
        return (f.severity != 'ERROR', f.where, 1 << 30)
    return (f.severity != 'ERROR', path, int(line))


def collector(doc, findings):
    def add(line, sev, code, problem, fix):
        findings.append(Finding(f'{doc.path}:{line}', sev, code, problem, fix))
    return add


def check_doc(doc, add, root):
    check_fences(doc, add)
    check_headings(doc, add)
    check_opener(doc, add)
    check_sections(doc, add)
    check_images(doc, add)
    check_links(doc, add, root)
    check_prose(doc, add)


def load(path):
    try:
        return Doc(path, path.read_text(encoding='utf-8'))
    except OSError as e:
        print(f'error: {e}', file=sys.stderr)
        sys.exit(1)


def main():
    ap = argparse.ArgumentParser(
        description='Check a README (and its Russian translation) against docs-conventions.')
    ap.add_argument('readme', nargs='?', default='README.md', help='README file (default README.md)')
    ap.add_argument('--ru', help='Russian translation (default: <name>_RUS.md next to it, if present)')
    ap.add_argument('--no-ru', action='store_true', help='check the README alone')
    ap.add_argument('--json', action='store_true')
    ap.add_argument('--strict', action='store_true', help='warnings fail too')
    args = ap.parse_args()

    en_path = Path(args.readme)
    en = load(en_path)
    root = repo_root(en_path.resolve().parent)
    findings = []
    add_en = collector(en, findings)
    check_doc(en, add_en, root)

    ru_path = None
    if not args.no_ru:
        if args.ru:
            ru_path = Path(args.ru)
        elif not en_path.stem.endswith('_RUS'):
            guess = en_path.with_name(en_path.stem + '_RUS' + en_path.suffix)
            ru_path = guess if guess.exists() else None
    if ru_path:
        ru = load(ru_path)
        add_ru = collector(ru, findings)
        check_doc(ru, add_ru, root)
        check_pair(en, ru, add_en, add_ru)

    findings = sorted(cap(findings), key=sort_key)
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
