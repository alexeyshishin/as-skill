#!/usr/bin/env python3
import argparse
import hashlib
import json
import re
import sys
from pathlib import Path

LEVELS = ('context', 'container', 'component')
LANGS = ('en', 'ru')
KINDS = ('person', 'svc', 'model', 'db', 'aux', 'ext')
TECH_FREE_KINDS = ('person', 'ext')
TOP_KEYS = {'title', 'note', 'level', 'lang', 'layers', 'nodes', 'boundaries', 'edges', 'callers', 'layout'}
NODE_KEYS = {'name', 'tech', 'kind', 'col'}
EDGE_KEYS = {'from', 'to', 'label', 'tech', 'description', 'protocol'}
LAYOUT_KEYS = {'seed', 'box_width', 'gap'}
MAX_LABEL = 34
MAX_TECH = 18
MAX_NAME = 26
MAX_EDGES = 30
MAX_PER_LAYER = 6
MAX_CALLERS = 6
SVG_HASH_RE = re.compile(r'<!-- c4-model sha256:([0-9a-f]{64}) -->')
CYR_RE = re.compile(r'[А-Яа-яЁё]')
LETTER_RE = re.compile(r'[A-Za-zА-Яа-яЁё]')


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


def model_hash(model):
    canon = json.dumps(model, sort_keys=True, ensure_ascii=False, separators=(',', ':'))
    return hashlib.sha256(canon.encode('utf-8')).hexdigest()


def load(path):
    try:
        return json.loads(Path(path).read_text(encoding='utf-8'))
    except OSError as e:
        raise SystemExit(f'error: {e}')
    except json.JSONDecodeError as e:
        raise SystemExit(f'error: {path}: invalid JSON: {e}')


def text(v):
    return v.strip() if isinstance(v, str) else ''


def validate(model, src='model'):
    out = []

    def add(where, severity, code, problem, fix):
        out.append(Finding(f'{src}:{where}' if where else src, severity, code, problem, fix))

    if not isinstance(model, dict):
        add('', 'ERROR', 'schema', 'top level is not an object', 'start from references/example-container.json')
        return out

    for k in sorted(set(model) - TOP_KEYS):
        add(k, 'WARN', 'unknown-key', f'unknown key "{k}"', 'remove it or fix the spelling')

    if not text(model.get('title')):
        add('title', 'ERROR', 'schema', 'no title', 'name the system and the level, e.g. "Shop — containers"')
    level = model.get('level', 'container')
    if level not in LEVELS:
        add('level', 'ERROR', 'level', f'level "{level}"', f'use one of {", ".join(LEVELS)}')
    lang = model.get('lang', 'en')
    if lang not in LANGS:
        add('lang', 'ERROR', 'lang', f'lang "{lang}"', f'use one of {", ".join(LANGS)}')
    detailed = level in ('container', 'component')

    layout = model.get('layout', {})
    if not isinstance(layout, dict):
        add('layout', 'ERROR', 'schema', 'layout is not an object', 'use {"seed": 1, "box_width": 240, "gap": 150}')
    else:
        for k in sorted(set(layout) - LAYOUT_KEYS):
            add(f'layout.{k}', 'WARN', 'unknown-key', f'unknown key "{k}"', f'use {", ".join(sorted(LAYOUT_KEYS))}')
        for k in sorted(set(layout) & LAYOUT_KEYS):
            if not isinstance(layout[k], int) or isinstance(layout[k], bool) or layout[k] < 0:
                add(f'layout.{k}', 'ERROR', 'schema', f'{k} is not a non-negative integer', 'set an integer')
        if isinstance(layout.get('box_width'), int) and layout['box_width'] < 160:
            add('layout.box_width', 'ERROR', 'schema', 'box_width under 160', 'use 160 or more')
        if isinstance(layout.get('gap'), int) and layout['gap'] < 80:
            add('layout.gap', 'ERROR', 'schema', 'gap under 80 leaves no room for vertical lanes', 'use 80 or more')

    nodes = model.get('nodes')
    if not isinstance(nodes, dict) or not nodes:
        add('nodes', 'ERROR', 'schema', 'nodes missing or not an object', 'map node ids to {name, tech, kind}')
        nodes = {}
    for nid, n in nodes.items():
        where = f'nodes.{nid}'
        if not isinstance(n, dict):
            add(where, 'ERROR', 'schema', 'node is not an object', 'use {name, tech, kind}')
            continue
        for k in sorted(set(n) - NODE_KEYS):
            add(f'{where}.{k}', 'WARN', 'unknown-key', f'unknown key "{k}"', f'use {", ".join(sorted(NODE_KEYS))}')
        if not text(n.get('name')):
            add(where, 'ERROR', 'schema', 'no name', 'give the node a display name')
        elif len(n['name']) > MAX_NAME:
            add(where, 'WARN', 'long-name', f'name is {len(n["name"])} chars, box fits ~{MAX_NAME}', 'shorten it; details go to tech')
        kind = n.get('kind', 'svc')
        if kind not in KINDS:
            add(where, 'ERROR', 'kind', f'kind "{kind}"', f'use one of {", ".join(KINDS)}')
        if detailed and kind not in TECH_FREE_KINDS and not text(n.get('tech')):
            add(where, 'ERROR', 'missing-tech', f'{level}-level node without technology', 'set tech: language, product or runtime')
        if level == 'context' and kind in ('db', 'model'):
            add(where, 'WARN', 'level-kind', f'{kind} node on a context diagram', 'stores and models are containers; fold them into the system')
        if 'col' in n and (not isinstance(n['col'], (int, float)) or isinstance(n['col'], bool) or n['col'] < 0):
            add(f'{where}.col', 'ERROR', 'schema', 'col is not a non-negative number', 'use a grid column, 0-based, halves allowed')

    layers = model.get('layers')
    if not isinstance(layers, list) or not layers:
        add('layers', 'ERROR', 'schema', 'layers missing or empty', 'list layers top to bottom: callers above callees')
        layers = []
    rowof = {}
    for r, layer in enumerate(layers):
        where = f'layers[{r}]'
        if not isinstance(layer, dict) or not isinstance(layer.get('nodes'), list) or not layer['nodes']:
            add(where, 'ERROR', 'schema', 'layer is not {label, nodes: [...]} or has no nodes', 'give it a label and node ids')
            continue
        if not text(layer.get('label')):
            add(where, 'WARN', 'layer-label', 'layer without a label', 'name what the layer holds, e.g. "Storage"')
        if len(layer['nodes']) > MAX_PER_LAYER:
            add(where, 'WARN', 'wide-layer', f'{len(layer["nodes"])} nodes in one layer', f'keep {MAX_PER_LAYER} or fewer; split the layer or the diagram')
        for nid in layer['nodes']:
            if nid not in nodes:
                add(where, 'ERROR', 'unknown-node', f'"{nid}" is not in nodes', 'define it under nodes or remove it')
            elif nid in rowof:
                add(where, 'ERROR', 'duplicate-node', f'"{nid}" already in layers[{rowof[nid]}]', 'a node sits in exactly one layer')
            else:
                rowof[nid] = r
    for nid in nodes:
        if nid not in rowof and isinstance(nodes[nid], dict):
            add(f'nodes.{nid}', 'ERROR', 'unplaced-node', 'node is in no layer', 'add it to a layer or delete it')
    for nid, n in nodes.items():
        if isinstance(n, dict) and n.get('kind') == 'person' and rowof.get(nid, 0) != 0:
            add(f'nodes.{nid}', 'WARN', 'person-not-top', 'person below the first layer', 'people start calls; put them in the top layer')

    bounds = model.get('boundaries', [])
    if not isinstance(bounds, list):
        add('boundaries', 'ERROR', 'schema', 'boundaries is not a list', 'use [{id, name, nodes}]')
        bounds = []
    bound_of = {}
    bids = set()
    for i, b in enumerate(bounds):
        where = f'boundaries[{i}]'
        if not isinstance(b, dict) or not text(b.get('id')) or not isinstance(b.get('nodes'), list) or not b['nodes']:
            add(where, 'ERROR', 'schema', 'boundary needs id, name and nodes', 'use {id, name, nodes: [...]}')
            continue
        bid = b['id']
        if bid in nodes or bid in bids:
            add(where, 'ERROR', 'duplicate-id', f'id "{bid}" already used', 'boundary ids differ from node ids and each other')
        bids.add(bid)
        if not text(b.get('name')):
            add(where, 'WARN', 'boundary-name', 'boundary without a name', 'name the group, e.g. "Workers"')
        members = [m for m in b['nodes'] if m in rowof]
        for m in b['nodes']:
            if m not in rowof:
                add(where, 'ERROR', 'unknown-node', f'"{m}" is not placed in a layer', 'add it to a layer first')
            elif m in bound_of:
                add(where, 'ERROR', 'boundary-overlap', f'"{m}" already in boundary "{bound_of[m]}"', 'a node belongs to one boundary')
            else:
                bound_of[m] = bid
        rows = {rowof[m] for m in members}
        if len(rows) > 1:
            add(where, 'ERROR', 'boundary-layers', 'members span several layers', 'keep a boundary inside one layer')
        elif rows:
            r = rows.pop()
            idx = sorted(layers[r]['nodes'].index(m) for m in members)
            if idx != list(range(idx[0], idx[0] + len(idx))):
                add(where, 'ERROR', 'boundary-gap', 'members are not adjacent in their layer', 'reorder the layer so members sit side by side')
            rowof[bid] = r

    edges = model.get('edges', [])
    if not isinstance(edges, list):
        add('edges', 'ERROR', 'schema', 'edges is not a list', 'use [{from, to, label, tech}]')
        edges = []
    seen = {}
    degree = {}
    reverse = []
    for i, e in enumerate(edges, 1):
        where = f'edges[{i - 1}]'
        if not isinstance(e, dict):
            add(where, 'ERROR', 'schema', 'edge is not an object', 'use {from, to, label, tech}')
            continue
        for k in sorted(set(e) - EDGE_KEYS):
            add(f'{where}.{k}', 'WARN', 'unknown-key', f'unknown key "{k}"', f'use {", ".join(sorted(EDGE_KEYS))}')
        a, b = e.get('from'), e.get('to')
        ok = True
        for side, v in (('from', a), ('to', b)):
            if v not in rowof:
                add(where, 'ERROR', 'unknown-node', f'{side} "{v}" is not a placed node or boundary', 'fix the id')
                ok = False
        label = text(e.get('label'))
        if not label:
            add(where, 'ERROR', 'missing-label', f'#{i} has no label', 'say what the call does, verb first')
        elif len(label) > MAX_LABEL:
            add(where, 'WARN', 'long-label', f'#{i} label is {len(label)} chars', f'keep it under {MAX_LABEL}; the long text goes to description')
        tech = text(e.get('tech'))
        if not tech:
            if detailed:
                add(where, 'ERROR', 'missing-tech', f'#{i} has no tech', 'set format/transport, e.g. JSON/HTTPS, SQL/TCP')
        else:
            if len(tech) > MAX_TECH:
                add(where, 'WARN', 'long-tech', f'#{i} tech is {len(tech)} chars', 'keep format/transport short; details go to protocol')
            if detailed and '/' not in tech:
                add(where, 'WARN', 'tech-format', f'#{i} tech "{tech}" is not format/transport', 'write it as JSON/HTTPS, gRPC/Protobuf, SQL/TCP')
        if not ok:
            continue
        if a == b:
            add(where, 'ERROR', 'self-edge', f'#{i} points to itself', 'drop it; internal calls are not on this level')
            continue
        if (a, b) in seen:
            add(where, 'ERROR', 'duplicate-edge', f'#{i} repeats #{seen[(a, b)]}', 'merge both into one label')
        seen[(a, b)] = i
        if a in bound_of and bound_of[a] == b or b in bound_of and bound_of[b] == a:
            add(where, 'ERROR', 'boundary-member', f'#{i} links a node to its own boundary', 'link the node to what it calls instead')
            continue
        if rowof[a] == rowof[b]:
            add(where, 'ERROR', 'same-layer', f'#{i} links two nodes of one layer', 'move the callee to a lower layer')
            continue
        if rowof[a] > rowof[b]:
            reverse.append(i)
            add(where, 'WARN', 'reverse-edge', f'#{i} points up', 'fine for a callback the callee starts (webhook, push); otherwise move the caller up')
        degree[a] = degree.get(a, 0) + 1
        degree[b] = degree.get(b, 0) + 1

    limit = max(2, len(edges) // 5)
    if len(reverse) > limit:
        add('edges', 'ERROR', 'too-many-reverse', f'{len(reverse)} of {len(edges)} edges point up', 'the layer order is off: callers go above callees')
    if len(edges) > MAX_EDGES:
        add('edges', 'WARN', 'dense', f'{len(edges)} edges', f'split the diagram by scope above {MAX_EDGES}')
    for nid in nodes:
        if nid in rowof and edges and nid not in degree and bound_of.get(nid) not in degree:
            add(f'nodes.{nid}', 'WARN', 'orphan-node', 'node has no edges', 'connect it or drop it')

    callers = model.get('callers')
    if callers is not None:
        if not isinstance(callers, list):
            add('callers', 'ERROR', 'schema', 'callers is not a list', 'list node or boundary ids')
        else:
            for c in callers:
                if c not in rowof:
                    add('callers', 'ERROR', 'unknown-node', f'"{c}" is not a placed node or boundary', 'fix the id')
            if len(callers) > MAX_CALLERS:
                add('callers', 'WARN', 'many-callers', f'{len(callers)} caller colors', f'keep {MAX_CALLERS} or fewer')
    return out


def check_svg(model, svg_path, src):
    where = str(svg_path)
    if not Path(svg_path).exists():
        return [Finding(where, 'ERROR', 'svg-missing', 'rendered SVG not found', f'run scripts/render.py {src}')]
    m = SVG_HASH_RE.search(Path(svg_path).read_text(encoding='utf-8', errors='replace'))
    if not m:
        return [Finding(where, 'ERROR', 'stale-svg', 'SVG was not rendered by render.py', f'render it from {src}; never edit the SVG by hand')]
    if m.group(1) != model_hash(model):
        return [Finding(where, 'ERROR', 'stale-svg', f'SVG does not match {src}', f'run scripts/render.py {src}')]
    return []


def shape(model):
    nodes = model.get('nodes') or {}
    return {
        'layers': [list(l.get('nodes', [])) for l in model.get('layers', []) if isinstance(l, dict)],
        'kinds': {k: v.get('kind', 'svc') for k, v in nodes.items() if isinstance(v, dict)},
        'boundaries': [(b.get('id'), list(b.get('nodes', []))) for b in model.get('boundaries', []) if isinstance(b, dict)],
        'edges': [(e.get('from'), e.get('to')) for e in model.get('edges', []) if isinstance(e, dict)],
        'callers': model.get('callers'),
    }


def untranslated(en, ru):
    return bool(en) and en == ru and LETTER_RE.search(en) and not CYR_RE.search(ru) and len(en.split()) > 1


def check_pair(en, ru, en_src, ru_src):
    out = []
    if ru.get('lang') != 'ru':
        out.append(Finding(f'{ru_src}:lang', 'ERROR', 'pair-lang', 'Russian model without "lang": "ru"', 'set lang to ru'))
    a, b = shape(en), shape(ru)
    for key in a:
        if a[key] != b[key]:
            out.append(Finding(f'{ru_src}:{key}', 'ERROR', 'pair-structure', f'{key} differ from {en_src}',
                               'ids, order, kinds and edges match EN; only text is translated'))
    texts = [('title', en.get('title'), ru.get('title'))]
    texts += [(f'layers[{i}].label', x.get('label'), y.get('label'))
              for i, (x, y) in enumerate(zip(en.get('layers', []), ru.get('layers', []))) if isinstance(x, dict) and isinstance(y, dict)]
    texts += [(f'edges[{i}].label', x.get('label'), y.get('label'))
              for i, (x, y) in enumerate(zip(en.get('edges', []), ru.get('edges', []))) if isinstance(x, dict) and isinstance(y, dict)]
    for where, x, y in texts:
        if untranslated(x, y):
            out.append(Finding(f'{ru_src}:{where}', 'WARN', 'untranslated', f'"{y}" is the EN text', 'translate it; keep terms in English'))
    return out


def main():
    ap = argparse.ArgumentParser(description='Check a C4 diagram model (and its rendered SVG, its Russian pair) against docs-conventions.')
    ap.add_argument('model', help='model JSON, e.g. docs/architecture/c4-container.json')
    ap.add_argument('--svg', help='rendered SVG to check for staleness (default: <model>.svg, if present)')
    ap.add_argument('--no-svg', action='store_true', help='skip the SVG check')
    ap.add_argument('--ru', help='Russian model (default: <model>_RUS.json next to it, if present)')
    ap.add_argument('--no-ru', action='store_true', help='check the model alone')
    ap.add_argument('--json', action='store_true')
    ap.add_argument('--strict', action='store_true', help='warnings fail too')
    args = ap.parse_args()

    path = Path(args.model)
    model = load(path)
    findings = validate(model, str(path))
    svg = None if args.no_svg else Path(args.svg) if args.svg else path.with_suffix('.svg')
    if svg and (args.svg or svg.exists()):
        findings += check_svg(model, svg, str(path))

    ru_path = None
    if not args.no_ru:
        if args.ru:
            ru_path = Path(args.ru)
        elif not path.stem.endswith('_RUS'):
            guess = path.with_name(path.stem + '_RUS' + path.suffix)
            ru_path = guess if guess.exists() else None
    if ru_path:
        ru = load(ru_path)
        findings += validate(ru, str(ru_path))
        ru_svg = ru_path.with_suffix('.svg')
        if not args.no_svg and ru_svg.exists():
            findings += check_svg(ru, ru_svg, str(ru_path))
        findings += check_pair(model, ru, str(path), str(ru_path))

    findings.sort(key=lambda f: (f.severity != 'ERROR', f.where))
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
