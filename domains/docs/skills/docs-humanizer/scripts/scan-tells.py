#!/usr/bin/env python3
import argparse
import json
import re
import sys
from pathlib import Path


TELLS_RU = [
    ('A1', 'вводные-подушки',
     r'\b(важно (?:отметить|понимать|помнить)|стоит (?:отметить|подчеркнуть)|'
     r'следует (?:отметить|учитывать)|нельзя не (?:отметить|упомянуть))\b', 3),
    ('A2', 'обобщающие связки',
     r'\b(таким образом|в целом|в конечном (?:счёте|счете|итоге)|подводя итог|'
     r'резюмируя|итак,)\b', 2),
    ('A3', 'безусловно-конечно',
     r'\b(безусловно|несомненно|разумеется|конечно же)\b', 2),

    ('B1', 'является/представляет собой',
     r'\b(являет(?:ся|ются)|представля(?:ет|ют) собой)\b', 3),
    ('B2', 'осуществление действия',
     r'\b(осуществля(?:ет|ют|ется)|производит(?:ся)? (?:анализ|оценка|проверка)|'
     r'выполня(?:ет|ется) (?:функци|роль))', 3),
    ('B3', 'данный вместо этот',
     r'\b(данн(?:ый|ая|ое|ого|ой|ом|ую|ому))\b', 2),
    ('B4', 'в рамках/в целях/с целью',
     r'\b(в рамках|в целях|с целью|в качестве|при помощи)\b', 1),
    ('B5', 'наличие/отсутствие',
     r'\b(наличи(?:е|я|ем)|отсутстви(?:е|я|ем))\b', 1),

    ('C1', 'мощный-революционный',
     r'\b(мощн(?:ый|ая|ое|ые)|революционн(?:ый|ая|ое)|инновационн(?:ый|ая|ое)|'
     r'уникальн(?:ый|ая|ое)|передов(?:ой|ая|ое)|современн(?:ый|ая|ое) подход)\b', 3),
    ('C2', 'ключевой-важнейший',
     r'\b(ключев(?:ой|ая|ое|ые)|важнейш(?:ий|ая|ее)|критически важн)', 2),
    ('C3', 'погружение-мир',
     r'\b(давайте (?:погрузимся|разберёмся|рассмотрим)|погрузимся в|'
     r'в мире (?:технологий|разработки|бизнеса)|в современном мире)\b', 3),
    ('C4', 'эффективный-оптимальный',
     r'\b(эффективн(?:ый|ая|ое|ые|о)|оптимальн(?:ый|ая|ое|ые))\b', 1),

    ('D2', 'не просто X, а Y',
     r'\bне просто [^,.\n]{2,40}, а\b', 3),
    ('D3', 'не только X, но и Y',
     r'\bне только [^,.\n]{2,50}, но и\b', 2),
    ('D4', 'это не X, это Y',
     r'\bэто не [^,.\n]{2,40}[,.] это\b', 3),
    ('D5', 'дело не в X, а в Y',
     r'\bдело не в [^,.\n]{2,40}, а в\b', 2),

    ('E1', 'от X до Y (риторическое)',
     r'\bот [^.\n]{3,40} до [^.\n]{3,40}[,.]', 1),
    ('E2', 'десятки-сотни-тысячи',
     r'\b(десятки|сотни|тысячи) (?:различных|разных|способов|вариантов|причин)\b', 2),

    ('F2', 'с одной стороны',
     r'\bс одной стороны\b.{0,400}?\bс другой стороны\b', 2),
    ('F3', 'всё зависит от',
     r'\b(всё зависит от|зависит от (?:конкретной )?ситуации|нет однозначного ответа)\b', 2),

    ('H1', 'дорогой читатель',
     r'\b(дорог(?:ой|ие) (?:читател|друз)|уважаем(?:ый|ые) читател)', 3),
    ('H2', 'надеюсь, это поможет',
     r'\b(надеюсь,? (?:это|эта)? ?(?:помож|был)|буду рад, если|'
     r'если у вас (?:остались|есть) вопрос)', 3),

    ('I1', 'главное — начать',
     r'\b(главное[  ]—[  ]начать|не бойтесь экспериментировать|'
     r'выбор за вами|время покажет)\b', 3),
    ('I2', 'играет важную роль',
     r'\b(игра(?:ет|ют) (?:важную|ключевую) роль|имеет большое значение)\b', 3),

    ('R4', 'README: приветствие вместо описания',
     r'^\W*(добро пожаловать|(?:этот|данный) (?:проект|репозиторий) '
     r'(?:призван|создан для|предназначен для))', 3),
]

RATE_RU = [
    ('G1', 'плотность длинных тире', r'—', 2.5, 2),
    ('F1', 'скопление хеджирования',
     r'\b(возможно|вероятно|как правило|обычно|зачастую|порой|скорее всего)\b', 1.8, 2),
    ('D1', 'однообразные тройные перечисления',
     r'\b\w+, \w+ и \w+[.,]', 1.2, 1),
]

TELLS_EN = [
    ('A1', 'signposting',
     r"\b(it(?:'s| is) (?:worth (?:noting|mentioning)|important to (?:note|remember|understand)"
     r"|crucial to (?:note|understand))|it should be noted|needless to say)\b", 3),
    ('A2', 'summary connectives',
     r'\b(in summary|in conclusion|to sum up|in essence|all in all)\b', 2),
    ('A3', 'sentence-initial additive',
     r'(?:^|(?<=[.!?] ))(additionally|furthermore|moreover|notably),', 2),
    ('A4', 'certainty filler',
     r'\b(undoubtedly|without a doubt|rest assured)\b', 2),

    ('B1', 'copula avoidance',
     r'\b(serves as an?|stands as an?|acts as a testament)\b', 2),
    ('B2', 'corporate verbs',
     r'\b(leverag(?:e|es|ed|ing)|utiliz(?:e|es|ed|ing)|empower(?:s|ed|ing)?|'
     r'streamlin(?:e|es|ed|ing)|elevat(?:e|es|ed|ing) your)\b', 1),
    ('B3', 'trailing -ing analysis',
     r',\s+(highlighting|underscoring|emphasizing|showcasing|reflecting|fostering|'
     r'cementing|solidifying)\b', 2),

    ('C1', 'marketing adjectives',
     r'\b(seamless(?:ly)?|cutting[- ]edge|state[- ]of[- ]the[- ]art|revolutionary|'
     r'next[- ]gen(?:eration)?|world[- ]class|best[- ]in[- ]class|blazing(?:ly)?[- ]fast|'
     r'lightning[- ]fast|supercharg\w*|unleash\w*)\b', 3),
    ('C2', 'soft praise',
     r'\b(powerful|robust|intuitive|elegant|effortless(?:ly)?|comprehensive|feature[- ]rich|'
     r'versatile|hassle[- ]free)\b', 1),
    ('C3', 'AI vocabulary',
     r'\b(delv(?:e|es|ing)|tapestry|testament|pivotal|vibrant|intricate|intricacies|'
     r'meticulous(?:ly)?|underscor(?:e|es|ed)|showcas(?:e|es|ed)|multifaceted|realm|'
     r'ever[- ]evolving|interplay|garner(?:s|ed)?|bolster(?:s|ed)?)\b', 2),
    ('C4', 'grand openers',
     r"\b(in today's (?:fast[- ]paced |digital |modern )?world|in the (?:ever[- ]|rapidly )?"
     r"evolving (?:world|landscape)|let's dive (?:in|into)|embark(?:s|ed|ing)? on)\b", 3),
    ('C5', 'significance inflation',
     r'\b(plays? an? (?:vital|pivotal|crucial|key|important) role|is a testament to|'
     r'paves? the way|game[- ]chang(?:er|ing)|to the next level|'
     r'unlock(?:s|ing)? the (?:full )?(?:power|potential))\b', 3),

    ('D1', 'not just X, but Y',
     r'\bnot (?:just|only|merely) [^,.\n]{2,50},? but(?: also)?\b', 3),
    ('D2', "it's not X, it's Y",
     r"\b(?:it|this)(?:'s| is) not [^,.\n]{2,40}[,;—–-]\s*(?:it|this)(?:'s| is)\b", 3),
    ('D3', 'no X, no Y, just Z',
     r'\bno [^,.\n]{2,30}, no [^,.\n]{2,30}(?:,|\s[—–-])\s*just\b', 3),

    ('E1', 'audience range',
     r"\b(whether you(?:'re| are) an? [^,.\n]{2,40} or an?|and everything in between)\b", 2),

    ('F2', 'hedged non-answer',
     r'\b(generally speaking|to some extent|it depends on (?:your|the) '
     r'(?:specific )?(?:needs|use case|situation))\b', 2),

    ('H1', 'chatbot artifacts',
     r"\b(i hope this helps|hope this helps|happy coding|happy hacking|great question|"
     r"you're absolutely right|as an ai\b|certainly!)", 3),
    ('H2', 'reader nudges',
     r"\b(feel free to|don't hesitate to)\b", 1),
    ('H3', 'knowledge cutoff',
     r'\b(as of my (?:last|latest) (?:update|knowledge)|as of my training)\b', 3),

    ('R4', 'README: greeting instead of description',
     r'^\W*(welcome to\b|this (?:project|repository|repo) (?:aims|is designed|was created) to\b|'
     r'an? (?:simple|lightweight|powerful) yet (?:powerful|flexible|simple)\b)', 3),
]

RATE_EN = [
    ('G1', 'em dash density', r'—', 1.0, 2),
    ('F1', 'hedging cluster',
     r'\b(potentially|possibly|arguably|typically|generally|usually|often)\b', 1.5, 2),
    ('D4', 'monotone triads', r'\b\w+, \w+,? and \w+[.,]', 1.2, 1),
]

EMOJI = '[\U0001F300-\U0001FAFF☀-➿⭐⬆↔-⇿]'
EMOJI_OPT = '(?:' + EMOJI + '[️‍]*\\s*)?'

TELLS_MD = [
    ('R1', 'emoji in headings', r'^\s{0,3}#{1,6}\s*' + EMOJI, 2),
]

COUNT_MD = [
    ('R2', 'emoji bullets', r'^\s*(?:[-*+]|\d+[.)])\s+' + EMOJI, 3, 2),
    ('R3', 'bold inline-header bullets',
     r'^\s*(?:[-*+]|\d+[.)])\s+' + EMOJI_OPT
     + r'\*\*[^*\n]{1,40}(?:\*\*\s*[:—–-]|[:—–-]\s*\*\*)', 4, 2),
]


def compile_tells(tells):
    return [(c, d, re.compile(rx, re.IGNORECASE | re.MULTILINE), w) for c, d, rx, w in tells]


def compile_rates(rates):
    return [(c, d, re.compile(rx, re.IGNORECASE | re.MULTILINE), t, w) for c, d, rx, t, w in rates]


SETS = {
    'ru': (compile_tells(TELLS_RU), compile_rates(RATE_RU)),
    'en': (compile_tells(TELLS_EN), compile_rates(RATE_EN)),
}
MD_TELLS = compile_tells(TELLS_MD)
MD_COUNTS = compile_rates(COUNT_MD)

FENCE_RE = re.compile(r'^\s*(`{3,}|~{3,})')
CYR_RE = re.compile(r'[а-яё]', re.IGNORECASE)
LAT_RE = re.compile(r'[a-z]', re.IGNORECASE)

VERDICTS = [
    (1.5, 'clean'),
    (3.5, 'traces: spot edits'),
    (6.0, 'noticeable: rework the flagged passages'),
    (float('inf'), 'dense: rewrite'),
]


def split_markdown(text):
    structure, prose = [], []
    fence = None
    for line in text.splitlines():
        m = FENCE_RE.match(line)
        if fence:
            if m and m.group(1)[0] == fence[0] and len(m.group(1)) >= len(fence):
                fence = None
            structure.append('')
            prose.append('')
            continue
        if m:
            fence = m.group(1)
            structure.append('')
            prose.append('')
            continue
        structure.append(line)
        s = re.sub(r'<!--.*?-->', ' ', line)
        s = re.sub(r'(`+).+?\1', ' ', s)
        s = re.sub(r'!\[[^\]]*\]\([^)]*\)', ' ', s)
        s = re.sub(r'\[([^\]]*)\]\([^)]*\)', r'\1', s)
        s = re.sub(r'<[^>]+>', ' ', s)
        s = re.sub(r'https?://\S+', ' ', s)
        s = re.sub(r'^\s{0,3}#{1,6}\s+', '', s)
        s = re.sub(r'^\s*(?:[-*+]|\d+[.)])\s+', '', s)
        s = re.sub(r'\|\s*[—–-]\s*(?=\|)', '| ', s)
        s = re.sub(r'^\s*\|?[\s:|-]+\|?\s*$', '', s)
        s = re.sub(r'[*_]{1,3}', '', s)
        prose.append(s)
    return '\n'.join(structure), '\n'.join(prose)


def detect_lang(text):
    cyr = len(CYR_RE.findall(text))
    lat = len(LAT_RE.findall(text))
    return 'ru' if cyr and cyr >= 0.4 * (cyr + lat) else 'en'


def hit(code, desc, weight, line, match):
    return {'code': code, 'tell': desc, 'weight': weight, 'line': line, 'match': match.strip()[:80]}


def scan(text, lang='auto', md=False):
    structure, prose = split_markdown(text) if md else (text, text)
    lang = detect_lang(prose) if lang == 'auto' else lang
    tells, rates = SETS[lang]
    findings = []
    for code, desc, rx, weight in tells:
        for m in rx.finditer(prose):
            findings.append(hit(code, desc, weight, prose.count('\n', 0, m.start()) + 1, m.group(0)))
    if md:
        for code, desc, rx, weight in MD_TELLS:
            for m in rx.finditer(structure):
                findings.append(hit(code, desc, weight,
                                    structure.count('\n', 0, m.start()) + 1, m.group(0)))
        for code, desc, rx, minimum, weight in MD_COUNTS:
            ms = list(rx.finditer(structure))
            if len(ms) >= minimum:
                findings.append(hit(code, desc, weight,
                                    structure.count('\n', 0, ms[0].start()) + 1,
                                    f'{len(ms)} items, threshold {minimum}'))

    words = len(re.findall(r'\b[а-яёa-z]+\b', prose, re.IGNORECASE))
    for code, desc, rx, threshold, weight in rates:
        hits = len(rx.findall(prose))
        if not words or not hits:
            continue
        rate = hits / words * 100
        if rate > threshold:
            findings.append(hit(code, desc, weight, 0,
                                f'{hits} hits, {rate:.1f} per 100 words, threshold {threshold}'))

    findings.sort(key=lambda f: (-f['weight'], f['line']))
    score = sum(f['weight'] for f in findings)
    return {
        'lang': lang,
        'words': words,
        'findings': findings,
        'total': len(findings),
        'score': score,
        'density': round(score / words * 100, 2) if words else 0.0,
    }


def verdict(density):
    return next(label for limit, label in VERDICTS if density < limit)


def load(path, lang, md):
    p = Path(path).expanduser()
    if not p.is_file():
        print(f'error: not a file: {p}', file=sys.stderr)
        sys.exit(1)
    use_md = p.suffix.lower() in ('.md', '.markdown') if md == 'auto' else md == 'on'
    return scan(p.read_text(encoding='utf-8'), lang, use_md)


def report(path, data, as_json):
    if as_json:
        print(json.dumps({'path': str(path), 'verdict': verdict(data['density']), **data},
                         ensure_ascii=False, indent=2))
        return
    print(f'{path} [{data["lang"]}]')
    print(f'   words: {data["words"]}, findings: {data["total"]}, '
          f'density: {data["density"]} per 100 words')
    print(f'   verdict: {verdict(data["density"])}\n')
    if not data['findings']:
        print('   No tells found.')
        return
    by_code = {}
    for f in data['findings']:
        by_code.setdefault((f['code'], f['tell']), []).append(f)
    for (code, tell), items in sorted(by_code.items(), key=lambda kv: -len(kv[1])):
        print(f'   [{code}] {tell} — {len(items)}')
        for f in items[:3]:
            print(f'        line {f["line"]}: {f["match"]}')
        if len(items) > 3:
            print(f'        … {len(items) - 3} more')


def compare(before_path, after_path, lang, md):
    before = load(before_path, lang, md)
    after = load(after_path, lang, md)

    def key_set(d):
        return {(f['code'], f['match']) for f in d['findings']}

    b, a = key_set(before), key_set(after)
    removed, added, kept = b - a, a - b, b & a

    print(f'Density: {before["density"]} → {after["density"]} per 100 words')
    print(f'Verdict: {verdict(before["density"])} → {verdict(after["density"])}\n')
    print(f'  removed: {len(removed)}')
    print(f'  kept:    {len(kept)}')
    print(f'  added by the rewrite: {len(added)}')
    if added:
        print('\n  New tells (appeared where there were none):')
        for code, match in sorted(added)[:10]:
            print(f'    [{code}] {match}')
    sys.exit(0 if after['density'] <= before['density'] else 2)


def main():
    ap = argparse.ArgumentParser(
        description='Regex scanner for AI-writing tells in Russian and English text. '
                    'One file: report. Two files: compare before → after.')
    ap.add_argument('files', nargs='+', metavar='FILE', help='FILE, or BEFORE AFTER to compare')
    ap.add_argument('--lang', choices=('auto', 'ru', 'en'), default='auto',
                    help='tell catalog (default: by the share of Cyrillic letters)')
    ap.add_argument('--md', choices=('auto', 'on', 'off'), default='auto',
                    help='skip code, links, HTML and add README tells (default: on for .md)')
    ap.add_argument('--max-density', type=float, metavar='N',
                    help='exit 2 when density per 100 words is above N')
    ap.add_argument('--json', action='store_true')
    args = ap.parse_args()

    if len(args.files) >= 2:
        compare(args.files[0], args.files[1], args.lang, args.md)
        return
    data = load(args.files[0], args.lang, args.md)
    report(args.files[0], data, args.json)
    if args.max_density is not None and data['density'] > args.max_density:
        sys.exit(2)


if __name__ == '__main__':
    main()
