#!/usr/bin/env python3
import argparse
import base64
import html
import mimetypes
import os
import re
import shutil
import struct
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REFERENCES = ROOT / 'references'
CANVAS = 1920
HANDLE_ENV = 'CONTENT_IMAGE_HANDLE'
CHROME_ENV = 'CHROME_BIN'
CYRILLIC_RE = re.compile(r'[\u0400-\u052f]')

CHROME_CANDIDATES = [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
    'google-chrome',
    'google-chrome-stable',
    'chromium',
    'chromium-browser',
    'chrome',
    'microsoft-edge',
]

EMOJI_PRESENTATION = (
    r'\u231a\u231b\u23e9-\u23ec\u23f0\u23f3\u25fd\u25fe\u2614\u2615\u2648-\u2653'
    r'\u267f\u2693\u26a1\u26aa\u26ab\u26bd\u26be\u26c4\u26c5\u26ce\u26d4\u26ea'
    r'\u26f2\u26f3\u26f5\u26fa\u26fd\u2705\u270a\u270b\u2728\u274c\u274e'
    r'\u2753-\u2755\u2757\u2795-\u2797\u27b0\u27bf\u2b1b\u2b1c\u2b50\u2b55'
)
EMOJI_TEXT_DEFAULT = (
    r'\u00a9\u00ae\u203c\u2049\u2122\u2139\u2194-\u2199\u21a9\u21aa\u2328\u23cf'
    r'\u23ed-\u23ef\u23f1\u23f2\u23f8-\u23fa\u24c2\u25aa\u25ab\u25b6\u25c0'
    r'\u25fb-\u25fe\u2600-\u27bf\u2934\u2935\u2b05-\u2b07\u2b1b\u2b1c\u2b50\u2b55'
    r'\u3030\u303d\u3297\u3299'
)
EMOJI_ELEMENT = (
    rf'(?:[{EMOJI_PRESENTATION}\U0001F000-\U0001FAFF]'
    rf'|[{EMOJI_TEXT_DEFAULT}]\ufe0f)'
    r'[\U0001F3FB-\U0001F3FF]?\ufe0f?'
)
EMOJI_RE = re.compile(
    r'[\U0001F1E6-\U0001F1FF]{2}'
    r'|[0-9#*]\ufe0f?\u20e3'
    rf'|{EMOJI_ELEMENT}(?:\u200d{EMOJI_ELEMENT})*'
)
INLINE_EMOJI_RE = re.compile(rf'[ \t]*({EMOJI_RE.pattern})')
BOLD_RE = re.compile(r'\*\*(.+?)\*\*', re.DOTALL)
EMOJI_MODIFIERS_RE = re.compile(r'[\ufe0f\U0001F3FB-\U0001F3FF]')
POST_FOLDERS = ('drafts', 'ready', 'published', 'archive')


FIT_SCRIPT = '''<script>
window.addEventListener('load', () => document.fonts.ready.then(() => {
  const root = getComputedStyle(document.documentElement);
  const px = name => parseFloat(root.getPropertyValue(name));
  const problems = [];
  const title = document.querySelector('.title');
  if (title) {
    const width = title.getBoundingClientRect().width;
    const limit = px('--canvas') - 2 * px('--margin');
    if (width > limit) problems.push('heading block is ' + Math.round(width) + 'px wide, limit ' + limit + 'px');
  }
  const text = document.querySelector('.text');
  if (text) {
    const height = text.getBoundingClientRect().height;
    const limit = px('--handle-top') - px('--margin');
    if (height > limit) problems.push('text is ' + Math.round(height) + 'px tall, limit ' + limit + 'px');
    if (text.scrollWidth > text.clientWidth) problems.push('a word is wider than the line');
  }
  document.title = 'fit:' + (problems.join('; ') || 'ok');
}));
</script>'''


def fail(message):
    print(f'render.py: {message}', file=sys.stderr)
    sys.exit(1)


def find_chrome():
    override = os.environ.get(CHROME_ENV)
    if override:
        if Path(override).exists() or shutil.which(override):
            return override
        fail(f'{CHROME_ENV}={override} not found')
    for candidate in CHROME_CANDIDATES:
        if Path(candidate).exists():
            return candidate
        found = shutil.which(candidate)
        if found:
            return found
    fail(f'Chrome/Chromium not found; install it or set {CHROME_ENV}')


def data_uri(path):
    mime = mimetypes.guess_type(path.name)[0] or 'application/octet-stream'
    payload = base64.b64encode(path.read_bytes()).decode('ascii')
    return f'data:{mime};base64,{payload}'


def embedded_fonts():
    fonts_dir = REFERENCES / 'fonts'
    css = (fonts_dir / 'fonts.css').read_text(encoding='utf-8')

    def inline(match):
        return f'url({data_uri(fonts_dir / match.group(1))})'

    return re.sub(r'url\(([^)]+\.woff2)\)', inline, css)


def emoji_span(glyph):
    return f'<span class="emoji"><span>{glyph}</span></span>'


def wrap_emoji(text):
    return INLINE_EMOJI_RE.sub(lambda m: f'<span class="emoji"><span>{m.group(1)}</span></span>', text)


def render_inline(text):
    escaped = html.escape(text, quote=False)
    bolded = BOLD_RE.sub(r'<strong>\1</strong>', escaped)
    return wrap_emoji(bolded.replace('\n', '<br>'))


def render_paragraphs(text):
    blocks = [block.strip('\n') for block in re.split(r'\n\s*\n', text.strip())]
    return '\n'.join(f'<p>{render_inline(block)}</p>' for block in blocks if block.strip())


def fill(template_name, values):
    page = (REFERENCES / 'templates' / template_name).read_text(encoding='utf-8')
    values = {
        'fonts': embedded_fonts(),
        'tokens': (REFERENCES / 'tokens.css').read_text(encoding='utf-8'),
        **values,
    }
    for key, value in values.items():
        page = page.replace('{{' + key + '}}', value)
    return page


def normalize_emoji(glyph):
    return EMOJI_MODIFIERS_RE.sub('', glyph)


def post_slug(path):
    return re.sub(r'^\d{4}-\d{2}-\d{2}-', '', path.stem)


def post_body(text):
    parts = text.split('---', 2)
    return parts[2] if text.startswith('---') and len(parts) == 3 else text


def used_emoji(root, slug=None):
    found = set()
    for sidecar in (root / 'images').glob('*/emoji.txt'):
        if sidecar.parent.name != slug:
            lines = sidecar.read_text(encoding='utf-8').splitlines()
            found.update(normalize_emoji(line.strip()) for line in lines if line.strip())
    for folder in POST_FOLDERS:
        for post in (root / folder).rglob('*.md'):
            if post_slug(post) != slug:
                body = post_body(post.read_text(encoding='utf-8'))
                found.update(normalize_emoji(match.group(0)) for match in EMOJI_RE.finditer(body))
    return found


def require_english(label, value):
    if CYRILLIC_RE.search(value):
        fail(f'{label} must be in English, translate it first (got {value!r})')


def build_title(args):
    heading = args.heading.strip()
    if not heading:
        fail('--heading is empty')
    require_english('heading', heading)
    if args.emoji and args.icon:
        fail('--emoji and --icon are mutually exclusive')
    if args.emoji and args.workspace:
        glyph = args.emoji.strip()
        if normalize_emoji(glyph) in used_emoji(Path(args.workspace), args.slug):
            fail(f'emoji {glyph} is already used in another post, pick a different one')
    mark = ''
    line_class = ''
    if args.emoji:
        mark = emoji_span(html.escape(args.emoji.strip(), quote=False))
    elif args.icon:
        icon = Path(args.icon)
        if not icon.is_file():
            fail(f'icon not found: {icon}')
        mark = f'<img class="icon" src="{data_uri(icon)}" alt="">'
        line_class = 'has-icon'
    subtitle = ''
    if args.subtitle:
        require_english('subtitle', args.subtitle)
        subtitle = f'<div class="subtitle">{html.escape(args.subtitle.strip(), quote=False)}</div>'
    return fill('title-card.html', {
        'heading': html.escape(heading, quote=False),
        'mark': mark,
        'line_class': line_class,
        'subtitle': subtitle,
    })


def build_text(args):
    if args.file:
        source = sys.stdin.read() if args.file == '-' else Path(args.file).read_text(encoding='utf-8')
    else:
        source = args.text or ''
    if not source.strip():
        fail('text is empty')
    handle = args.handle if args.handle is not None else os.environ.get(HANDLE_ENV, '')
    handle_html = ''
    if handle.strip():
        handle_html = f'<div class="handle">{html.escape(handle.strip(), quote=False)}</div>'
    return fill('text-page.html', {
        'body': render_paragraphs(source),
        'handle': handle_html,
    })


def png_size(path):
    header = path.read_bytes()[:24]
    if header[:8] != b'\x89PNG\r\n\x1a\n':
        return None
    return struct.unpack('>II', header[16:24])


def chrome_command(chrome, work, *extra):
    return [
        chrome,
        '--headless=new',
        '--disable-gpu',
        '--hide-scrollbars',
        '--no-first-run',
        '--no-default-browser-check',
        '--force-device-scale-factor=1',
        f'--window-size={CANVAS},{CANVAS}',
        '--virtual-time-budget=3000',
        f'--user-data-dir={Path(work) / "profile"}',
        *extra,
    ]


def check_fit(chrome, work, page):
    probe = Path(work) / 'probe.html'
    probe.write_text(page.replace('</body>', FIT_SCRIPT + '</body>'), encoding='utf-8')
    result = subprocess.run(
        chrome_command(chrome, work, '--dump-dom', probe.as_uri()),
        capture_output=True, text=True, timeout=120,
    )
    verdict = re.search(r'<title>fit:(.*?)</title>', result.stdout, re.DOTALL)
    if result.returncode != 0 or not verdict:
        fail(f'chrome fit check failed (exit {result.returncode}): {result.stderr.strip()[-400:]}')
    return verdict.group(1).strip()


def screenshot(page, out):
    chrome = find_chrome()
    out = out.resolve()
    out.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='content-image-') as work:
        verdict = check_fit(chrome, work, page)
        if verdict != 'ok':
            fail(f'content does not fit the canvas: {verdict}; shorten it or split it into several images')
        source = Path(work) / 'page.html'
        source.write_text(page, encoding='utf-8')
        result = subprocess.run(
            chrome_command(chrome, work, f'--screenshot={out}', source.as_uri()),
            capture_output=True, text=True, timeout=120,
        )
    if result.returncode != 0 or not out.exists():
        fail(f'chrome failed (exit {result.returncode}): {result.stderr.strip()[-400:]}')
    size = png_size(out)
    if size != (CANVAS, CANVAS):
        fail(f'unexpected output size {size}, expected {(CANVAS, CANVAS)}')
    return out


def parse_args(argv):
    parser = argparse.ArgumentParser(
        prog='render.py',
        description='Render a post image from the content-image templates to PNG via headless Chrome.',
    )
    sub = parser.add_subparsers(dest='kind', required=True)

    title = sub.add_parser('title', help='heading card with an emoji or icon')
    title.add_argument('--heading', required=True)
    title.add_argument('--subtitle')
    title.add_argument('--emoji')
    title.add_argument('--icon')
    title.add_argument('--workspace', help='content workspace root; refuses an emoji already used in another post')
    title.add_argument('--slug', help='the current post, left out of the emoji check')
    title.add_argument('-o', '--out', required=True)

    text = sub.add_parser('text', help='text page with bold spans, inline emoji and a handle')
    source = text.add_mutually_exclusive_group(required=True)
    source.add_argument('--text')
    source.add_argument('--file', help="path to a text file, or '-' for stdin")
    text.add_argument('--handle', help=f'overrides ${HANDLE_ENV}; empty string disables it')
    text.add_argument('-o', '--out', required=True)

    used = sub.add_parser('used', help='list emoji already used in a content workspace')
    used.add_argument('root', help='content workspace root')
    used.add_argument('--slug', help='the current post, left out of the list')

    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if args.kind == 'used':
        print('\n'.join(sorted(used_emoji(Path(args.root), args.slug))))
        return
    page = build_title(args) if args.kind == 'title' else build_text(args)
    print(screenshot(page, Path(args.out)))


if __name__ == '__main__':
    main()
