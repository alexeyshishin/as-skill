#!/usr/bin/env python3
import argparse
import importlib.util
import math
import os
import random
import shutil
import subprocess
import sys
import tempfile
import textwrap
from html import escape
from pathlib import Path

HERE = Path(__file__).resolve().parent
BH = 64
STRIP = 130
SIDE_W = 260
M = 30
LANE_H = 40
PAD_T = 30
PAD_B = 30
PAD_S = 14
SIDE = ('LS', 'RS', 'Lt', 'Rt')
FONT = 'DejaVu Sans, Helvetica, Arial, sans-serif'
CHROME_ENV = 'CHROME_BIN'
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

STYLE = {
    'person': ('#FBEAF0', '#993556', '#4B1528'),
    'svc': ('#E6F1FB', '#185FA5', '#042C53'),
    'model': ('#FAEEDA', '#854F0B', '#412402'),
    'db': ('#E1F5EE', '#0F6E56', '#04342C'),
    'aux': ('#EEEDFE', '#534AB7', '#26215C'),
    'ext': ('#F1EFE8', '#5F5E5A', '#2C2C2A'),
}
CALLER_COLORS = ['#185FA5', '#534AB7', '#0F6E56', '#993C1D', '#993556', '#854F0B']
BOUNDARY_COLOR = '#2C2C2A'
OTHER_COLOR = '#5F5E5A'

TEXT = {
    'en': {
        'kinds': {'person': 'Person', 'svc': 'Our service', 'model': 'Model', 'db': 'Storage',
                  'aux': 'Supporting', 'ext': 'External system'},
        'callers': 'Arrow color — caller:',
        'others': 'others',
        'table': 'Relationships',
        'cols': ('#', 'Caller → callee', 'What it does', 'Protocol'),
        'down': 'Caller above, callee below: arrows point down',
        'except': ', except callbacks: {}',
        'label': '. Arrow label: what the call does and [format/transport].',
        'label_short': '. Arrow label: what the call does.',
    },
    'ru': {
        'kinds': {'person': 'Пользователь', 'svc': 'Наш сервис', 'model': 'Модель', 'db': 'Хранилище',
                  'aux': 'Вспомогательный', 'ext': 'Внешняя система'},
        'callers': 'Цвет стрелки — кто вызывает:',
        'others': 'остальные',
        'table': 'Связи',
        'cols': ('№', 'Кто вызывает → кого', 'Что делает', 'Протокол'),
        'down': 'Вызывающий сверху, вызываемый снизу: стрелки идут вниз',
        'except': ', кроме обратных вызовов: {}',
        'label': '. На стрелке: что делает вызов и [формат/транспорт].',
        'label_short': '. На стрелке: что делает вызов.',
    },
}


def fail(message, code=1):
    print(f'render.py: {message}', file=sys.stderr)
    sys.exit(code)


def load_checker():
    spec = importlib.util.spec_from_file_location('check_c4', HERE / 'check-c4.py')
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def tw(s, fs, bold=False):
    return len(s) * fs * (0.63 if bold else 0.57)


def wrap(s, width):
    return textwrap.wrap(s, width) or ['']


class Plan:
    def __init__(self, model, box_w, gap):
        self.lang = model.get('lang', 'en')
        self.layers = [list(l['nodes']) for l in model['layers']]
        self.labels = [l.get('label', '') for l in model['layers']]
        self.nodes = {k: {'name': v['name'], 'tech': v.get('tech', ''), 'kind': v.get('kind', 'svc'), 'col': v.get('col')}
                      for k, v in model['nodes'].items()}
        self.bounds = {b['id']: {'name': b.get('name', ''), 'nodes': list(b['nodes'])} for b in model.get('boundaries', [])}
        self.nrows = len(self.layers)
        self.rowof = {n: r for r, l in enumerate(self.layers) for n in l}
        for bid, b in self.bounds.items():
            self.rowof[bid] = self.rowof[b['nodes'][0]]
        self.panel_rows = {self.rowof[bid] for bid in self.bounds}
        self.box_w = box_w
        ncol = max(len(l) for l in self.layers)
        self.G0 = M + STRIP + SIDE_W
        self.x0, self.x1 = {}, {}
        for l in self.layers:
            off = (ncol - len(l)) / 2
            for i, n in enumerate(l):
                col = self.nodes[n]['col']
                c = off + i if col is None else col
                self.x0[n] = self.G0 + c * (box_w + gap)
                self.x1[n] = self.x0[n] + box_w
        for bid, b in self.bounds.items():
            self.x0[bid] = min(self.x0[n] for n in b['nodes']) - PAD_S
            self.x1[bid] = max(self.x1[n] for n in b['nodes']) + PAD_S
        self.GR = max(self.G0 + ncol * box_w + (ncol - 1) * gap, max(self.x1[n] for n in self.nodes))
        self.W = self.GR + SIDE_W + M
        self.corr = {'L': (M + STRIP + 10, self.G0 - PAD_S - 12, 6)}
        for g in range(ncol - 1):
            xa = self.G0 + (g + 1) * box_w + g * gap
            self.corr[f'g{g}'] = (xa + PAD_S + 4, xa + gap - PAD_S - 4, 3)
        self.corr['R'] = (self.GR + PAD_S + 12, self.W - M - 10, 6)
        self.edges = model.get('edges', [])
        self.E = []
        for i, e in enumerate(self.edges, 1):
            a, b = e['from'], e['to']
            ra, rb = self.rowof[a], self.rowof[b]
            rev = ra > rb
            if rev:
                a, b, ra, rb = b, a, rb, ra
            self.E.append({'n': i, 'a': a, 'b': b, 'ra': ra, 'rb': rb, 'long': rb - ra > 1, 'rev': rev})
        for e in self.E:
            e['cand'] = self.candidates(e)

    def ctr(self, n):
        return (self.x0[n] + self.x1[n]) / 2

    def band_free(self, row, xa, xb):
        for n in self.layers[row]:
            if not (xb < self.x0[n] - 6 or xa > self.x1[n] + 6):
                return False
        for bid in self.bounds:
            if self.rowof[bid] == row and not (xb < self.x0[bid] - 6 or xa > self.x1[bid] + 6):
                return False
        return True

    def edge_of_row(self, n, first):
        row = self.layers[self.rowof[n]]
        end = row[0] if first else row[-1]
        return n == end or (n in self.bounds and end in self.bounds[n]['nodes'])

    def candidates(self, e):
        if not e['long']:
            return ['-']
        mids = range(e['ra'] + 1, e['rb'])
        out = []
        b = e['b']
        if all(self.band_free(r, self.x0[b] + 16, self.x1[b] - 16) for r in mids):
            out.append('D')
        for c, (xa, xb, _) in self.corr.items():
            if all(self.band_free(r, xa, xb) for r in mids):
                out.append(c)
        sl, sr = self.edge_of_row(e['a'], True), self.edge_of_row(e['a'], False)
        tl, tr = self.edge_of_row(b, True), self.edge_of_row(b, False)
        if sl and tl:
            out.append('LS')
        if sr and tr:
            out.append('RS')
        if sl:
            out.append('Lt')
        if sr:
            out.append('Rt')
        return out

    def chan_segments(self, assign):
        segs = {k: [] for k in range(self.nrows - 1)}
        for e in self.E:
            ch = assign[e['n']]
            if ch in ('LS', 'RS'):
                continue
            if ch in ('Lt', 'Rt'):
                segs[e['rb'] - 1].append((e['n'], 'h2'))
                continue
            segs[e['ra']].append((e['n'], 'h1'))
            if e['long'] and ch != 'D':
                segs[e['rb'] - 1].append((e['n'], 'h2'))
        return segs

    def ports(self, n, fars):
        k = len(fars)
        if n in self.bounds:
            members = sorted(self.bounds[n]['nodes'], key=lambda m: self.x0[m])
            slots = [(self.x1[a] + self.x0[b]) / 2 for a, b in zip(members, members[1:])]
            if not slots:
                slots = [self.x0[n] + PAD_S / 2, self.x1[n] - PAD_S / 2]
            cands = sorted({s + d for s in slots for d in (-48, -24, 0, 24, 48)})
            return match(fars, cands)
        if k == 1:
            return [self.ctr(n)]
        step = min(40, (self.box_w - 32) / (k - 1))
        start = self.ctr(n) - step * (k - 1) / 2
        return [start + i * step for i in range(k)]

    def build(self, assign, corder, lord):
        x0, x1 = self.x0, self.x1
        lanex = {}
        for c, lst in corder.items():
            xa, xb, _ = self.corr[c]
            for i, n in enumerate(lst):
                lanex[n] = xa + (i + 1) * (xb - xa) / (len(lst) + 1)
        chh = {k: 46 + max(len(lord[k]) - 1, 0) * LANE_H + 34 for k in range(self.nrows - 1)}
        Y, rowtop, rowbot = {}, {}, {}
        y = 40
        for r in range(self.nrows):
            pt = PAD_T if r in self.panel_rows else 0
            pb = PAD_B if r in self.panel_rows else 0
            rowtop[r] = y
            y += pt
            Y[r] = y
            y += BH + pb
            rowbot[r] = y
            if r < self.nrows - 1:
                y += chh[r]

        far_out, far_in = {}, {}
        for e in self.E:
            n, c = e['n'], assign[e['n']]
            if c in ('LS', 'RS'):
                continue
            if c in ('Lt', 'Rt'):
                far_in[n] = lanex[n]
            elif not e['long'] or c == 'D':
                far_out[n], far_in[n] = self.ctr(e['b']), self.ctr(e['a'])
            else:
                far_out[n] = far_in[n] = lanex[n]
        outp, inp = {}, {}
        srcs, dsts = {}, {}
        for e in self.E:
            c = assign[e['n']]
            if c in ('LS', 'RS'):
                continue
            if c not in SIDE:
                srcs.setdefault(e['a'], []).append(e)
            dsts.setdefault(e['b'], []).append(e)
        for a, l in srcs.items():
            l.sort(key=lambda e: (far_out[e['n']], e['n']))
            for e, x in zip(l, self.ports(a, [far_out[e['n']] for e in l])):
                outp[e['n']] = x
        for b, l in dsts.items():
            l.sort(key=lambda e: (far_in[e['n']], e['n']))
            for e, x in zip(l, self.ports(b, [far_in[e['n']] for e in l])):
                inp[e['n']] = x

        for e in self.E:
            n = e['n']
            a, b = e['a'], e['b']
            if a in self.bounds or b in self.bounds or assign[n] in SIDE:
                continue
            if e['long'] and assign[n] != 'D':
                continue
            lo = max(x0[a], x0[b]) + 20
            hi = min(x1[a], x1[b]) - 20
            if lo > hi:
                continue
            oa = [outp[m['n']] for m in self.E if m['a'] == a and m['n'] != n and m['n'] in outp]
            ob = [inp[m['n']] for m in self.E if m['b'] == b and m['n'] != n and m['n'] in inp]
            ra = sum(1 for v in oa if v < outp[n])
            rb = sum(1 for v in ob if v < inp[n])
            best = None
            x = lo
            while x <= hi + 1e-6:
                if (all(abs(x - v) >= 38 for v in oa + ob)
                        and sum(1 for v in oa if v < x) == ra and sum(1 for v in ob if v < x) == rb):
                    d = abs(x - (outp[n] + inp[n]) / 2)
                    if best is None or d < best[0]:
                        best = (d, x)
                x += 2
            if best:
                outp[n] = inp[n] = best[1]

        sy, ty = {}, {}
        by_a, by_b = {}, {}
        for e in self.E:
            c = assign[e['n']]
            if c in SIDE:
                side = 'L' if c in ('LS', 'Lt') else 'R'
                by_a.setdefault((e['a'], side), []).append(e)
                if c in ('LS', 'RS'):
                    by_b.setdefault((e['b'], side), []).append(e)
        for (a, side), l in by_a.items():
            edge_x = x0[a] if side == 'L' else x1[a]
            l.sort(key=lambda e: (-abs(lanex[e['n']] - edge_x), e['n']))
            k = len(l)
            r = self.rowof[a]
            for i, e in enumerate(l):
                if a in self.bounds:
                    sy[e['n']] = Y[r] + BH + PAD_B / 2 + (i - (k - 1) / 2) * min(8, (PAD_B - 8) / max(k - 1, 1))
                else:
                    sy[e['n']] = Y[r] + BH / 2 + (i - (k - 1) / 2) * min(22, (BH - 16) / max(k - 1, 1))
        for (b, side), l in by_b.items():
            edge_x = x0[b] if side == 'L' else x1[b]
            l.sort(key=lambda e: (abs(lanex[e['n']] - edge_x), e['n']))
            k = len(l)
            r = self.rowof[b]
            for i, e in enumerate(l):
                if b in self.bounds:
                    ty[e['n']] = rowtop[r] + PAD_T / 2 + (i - (k - 1) / 2) * min(8, (PAD_T - 8) / max(k - 1, 1))
                else:
                    ty[e['n']] = Y[r] + BH / 2 + (i - (k - 1) / 2) * min(22, (BH - 16) / max(k - 1, 1))

        laney = {}
        for k in range(self.nrows - 1):
            for i, s in enumerate(lord[k]):
                laney[s] = rowbot[k] + 46 + i * LANE_H

        def ybot(a):
            return rowbot[self.rowof[a]] if a in self.bounds else Y[self.rowof[a]] + BH

        def ytop(b):
            return rowtop[self.rowof[b]] if b in self.bounds else Y[self.rowof[b]]

        R = {}
        for e in self.E:
            n, a, b = e['n'], e['a'], e['b']
            c = assign[n]
            if c in ('LS', 'RS'):
                xc = lanex[n]
                sx = x0[a] if c == 'LS' else x1[a]
                tx = x0[b] if c == 'LS' else x1[b]
                pts = [(sx, sy[n]), (xc, sy[n]), (xc, ty[n]), (tx, ty[n])]
            elif c in ('Lt', 'Rt'):
                xc, xt, y2 = lanex[n], inp[n], laney[(n, 'h2')]
                sx = x0[a] if c == 'Lt' else x1[a]
                pts = [(sx, sy[n]), (xc, sy[n]), (xc, y2), (xt, y2), (xt, ytop(b))]
            else:
                xs, xt = outp[n], inp[n]
                y1 = laney[(n, 'h1')]
                pts = [(xs, ybot(a)), (xs, y1)]
                if e['long'] and c != 'D':
                    xc, y2 = lanex[n], laney[(n, 'h2')]
                    pts += [(xc, y1), (xc, y2), (xt, y2)]
                else:
                    pts += [(xt, y1)]
                pts.append((xt, ytop(b)))
            R[n] = clean(pts)
        return {'R': R, 'Y': Y, 'rowtop': rowtop, 'rowbot': rowbot}


def match(fars, cands):
    k, p = len(fars), len(cands)
    if p < k:
        lo, hi = cands[0], cands[-1]
        return [lo + (hi - lo) * (i + 1) / (k + 1) for i in range(k)]
    inf = float('inf')
    dp = [[inf] * (p + 1) for _ in range(k + 1)]
    for j in range(p + 1):
        dp[0][j] = 0
    for i in range(1, k + 1):
        for j in range(i, p + 1):
            dp[i][j] = min(dp[i][j - 1], dp[i - 1][j - 1] + abs(fars[i - 1] - cands[j - 1]))
    out = []
    i, j = k, p
    while i > 0:
        if dp[i][j] == dp[i][j - 1] and j > i:
            j -= 1
        else:
            out.append(cands[j - 1])
            i -= 1
            j -= 1
    return out[::-1]


def clean(pts):
    out = [pts[0]]
    for p in pts[1:]:
        if abs(p[0] - out[-1][0]) > 1e-6 or abs(p[1] - out[-1][1]) > 1e-6:
            out.append(p)
    i = 1
    while i < len(out) - 1:
        a, b, c = out[i - 1], out[i], out[i + 1]
        if (abs(a[0] - b[0]) < 1e-6 and abs(b[0] - c[0]) < 1e-6) or (abs(a[1] - b[1]) < 1e-6 and abs(b[1] - c[1]) < 1e-6):
            out.pop(i)
        else:
            i += 1
    return out


def metrics(R):
    H, V = [], []
    length = 0
    for n, p in R.items():
        for a, b in zip(p, p[1:]):
            length += abs(b[0] - a[0]) + abs(b[1] - a[1])
            if abs(a[1] - b[1]) < 1e-6:
                H.append((n, min(a[0], b[0]), max(a[0], b[0]), a[1]))
            else:
                V.append((n, min(a[1], b[1]), max(a[1], b[1]), a[0]))
    cross = 0
    for hn, x1, x2, y in H:
        for vn, y1, y2, x in V:
            if hn != vn and x1 + 0.5 < x < x2 - 0.5 and y1 + 0.5 < y < y2 - 0.5:
                cross += 1
    overlap = 0
    for segs in (H, V):
        for i in range(len(segs)):
            for j in range(i + 1, len(segs)):
                s, t = segs[i], segs[j]
                if s[0] != t[0] and abs(s[3] - t[3]) < 4 and min(s[2], t[2]) - max(s[1], t[1]) > 1:
                    overlap += 1
    return cross, overlap, length


def anneal(plan, rng, restarts, iterations):
    E = plan.E
    if not E:
        return [({}, {}, {k: [] for k in range(plan.nrows - 1)})]

    def ck(c):
        return {'LS': 'L', 'Lt': 'L', 'RS': 'R', 'Rt': 'R'}.get(c, c)

    def cost(st):
        assign, corder, lord = st
        cross, overlap, length = metrics(plan.build(assign, corder, lord)['R'])
        pen = sum(1000 * (len(l) - plan.corr[c][2]) for c, l in corder.items() if len(l) > plan.corr[c][2])
        bonus = sum(1 for v in assign.values() if v in ('LS', 'RS'))
        return cross * 10 + overlap * 30 + length * 0.01 + pen - 15 * bonus

    def make(assign):
        corder = {}
        for n, c in assign.items():
            if c not in ('-', 'D'):
                corder.setdefault(ck(c), []).append(n)
        lord = {k: rng.sample(v, len(v)) for k, v in plan.chan_segments(assign).items()}
        return [dict(assign), corder, lord]

    def copy(st):
        return [dict(st[0]), {k: list(v) for k, v in st[1].items()}, {k: list(v) for k, v in st[2].items()}]

    movable = [e for e in E if len(e['cand']) > 1]
    results = []
    for _ in range(restarts):
        st = make({e['n']: rng.choice(e['cand']) for e in E})
        cur = cost(st)
        best = (cur, copy(st))
        for it in range(iterations):
            T = max(0.05, 3.0 * (1 - it / iterations))
            new = copy(st)
            mv = rng.random()
            lanes = [k for k in new[2] if len(new[2][k]) > 1]
            corrs = [c for c, l in new[1].items() if len(l) > 1]
            if mv < 0.55 and lanes:
                k = rng.choice(lanes)
                a, b = rng.sample(range(len(new[2][k])), 2)
                new[2][k][a], new[2][k][b] = new[2][k][b], new[2][k][a]
            elif mv < 0.8 and movable:
                e = rng.choice(movable)
                n, old = e['n'], st[0][e['n']]
                nc = rng.choice(e['cand'])
                if nc == old:
                    continue
                new[0][n] = nc
                if old not in ('-', 'D'):
                    new[1][ck(old)].remove(n)
                if nc not in ('-', 'D'):
                    new[1].setdefault(ck(nc), []).append(n)
                segs = plan.chan_segments(new[0])
                for k in segs:
                    keep = [s for s in new[2][k] if s in segs[k]]
                    new[2][k] = keep + [s for s in segs[k] if s not in keep]
            elif corrs:
                l = new[1][rng.choice(corrs)]
                a, b = rng.sample(range(len(l)), 2)
                l[a], l[b] = l[b], l[a]
            else:
                continue
            c2 = cost(new)
            if c2 <= cur or rng.random() < math.exp((cur - c2) / max(T, 1e-6)):
                st, cur = new, c2
                if cur < best[0]:
                    best = (cur, copy(st))
        results.append(best)
    results.sort(key=lambda r: r[0])
    return [r[1] for r in results]


def badge_pos(pts):
    p, q = pts[0], pts[1]
    d = math.hypot(q[0] - p[0], q[1] - p[1]) or 1
    return p[0] + 15 * (q[0] - p[0]) / d, p[1] + 15 * (q[1] - p[1]) / d


def drawn(plan, R, n):
    return R[n][::-1] if plan.E[n - 1]['rev'] else R[n]


def label_size(edge):
    label, tech = edge.get('label', ''), edge.get('tech', '')
    w = max(len(label) * 6.7, (len(tech) + 2) * 6.5 if tech else 0) + 10
    return w, 30 if tech else 16


def place_labels(plan, lay, rng, tries):
    R, Y = lay['R'], lay['Y']
    boxes = [(plan.x0[n] - 4, Y[plan.rowof[n]] - 4, plan.x1[n] + 4, Y[plan.rowof[n]] + BH + 4) for n in plan.nodes]
    badges = []
    for n in R:
        bx, by = badge_pos(drawn(plan, R, n))
        badges.append((bx - 11, by - 11, bx + 11, by + 11))
    segs = []
    for n, p in R.items():
        for i in range(len(p) - 1):
            segs.append((n, i, p[i], p[i + 1]))
    for bid in plan.bounds:
        r = plan.rowof[bid]
        a, b, t, u = plan.x0[bid], plan.x1[bid], lay['rowtop'][r], lay['rowbot'][r]
        for s, f in (((a, t), (b, t)), ((a, u), (b, u)), ((a, t), (a, u)), ((b, t), (b, u))):
            segs.append((-1, 0, s, f))
    xmin, xmax = M + STRIP + 4, plan.W - 20

    def hit(rect, s, pad=1.5):
        x0, y0, x1, y1 = rect
        a, b = s[2], s[3]
        if abs(a[1] - b[1]) < 1e-6:
            return y0 - pad < a[1] < y1 + pad and not (max(a[0], b[0]) < x0 - pad or min(a[0], b[0]) > x1 + pad)
        return x0 - pad < a[0] < x1 + pad and not (max(a[1], b[1]) < y0 - pad or min(a[1], b[1]) > y1 + pad)

    def ov(r1, r2, pad=0):
        return not (r1[2] + pad < r2[0] or r1[0] - pad > r2[2] or r1[3] + pad < r2[1] or r1[1] - pad > r2[3])

    cands = {}
    for n, p in R.items():
        w, h = label_size(plan.edges[n - 1])
        out = []
        for i in range(len(p) - 1):
            a, b = p[i], p[i + 1]
            if abs(a[1] - b[1]) < 1e-6:
                xa, xb = sorted((a[0], b[0]))
                y, ln = a[1], xb - xa
                if ln < w + 14:
                    continue
                x = xa + 7
                while x + w <= xb - 7:
                    c = (x + w / 2) - (xa + xb) / 2
                    out.append((abs(c) * 0.3 - ln * 0.02, (x, y - 3 - h, x + w, y - 3), i, 'h'))
                    out.append((abs(c) * 0.3 + 40 - ln * 0.02, (x, y + 3, x + w, y + 3 + h), i, 'h'))
                    x += 8
            else:
                ya, yb = sorted((a[1], b[1]))
                x, ln = a[0], yb - ya
                if ln < h + 24:
                    continue
                y = ya + 12
                while y + h <= yb - 12:
                    c = (y + h / 2) - (ya + yb) / 2
                    out.append((abs(c) * 0.3 + 90 - ln * 0.02, (x + 6, y, x + 6 + w, y + h), i, 'r'))
                    out.append((abs(c) * 0.3 + 95 - ln * 0.02, (x - 6 - w, y, x - 6, y + h), i, 'l'))
                    y += 8
        out.sort(key=lambda t: t[0])
        cands[n] = out[:1500]

    def ok(n, cand, placed):
        _, rect, si, _ = cand
        if rect[0] < xmin or rect[2] > xmax or rect[1] < 20:
            return False
        for s in segs:
            if not (s[0] == n and s[1] == si) and hit(rect, s):
                return False
        for bx in boxes:
            if ov(rect, bx):
                return False
        for bd in badges:
            if ov(rect, bd, 2):
                return False
        for r2, _ in placed.values():
            if ov(rect, r2, 5):
                return False
        return True

    order0 = sorted(R, key=lambda n: (len(cands[n]), n))
    best = None
    for t in range(tries):
        order = order0[:] if t == 0 else rng.sample(order0, len(order0))
        placed, missed = {}, []
        for n in order:
            for cand in cands[n]:
                if ok(n, cand, placed):
                    placed[n] = (cand[1], cand[3])
                    break
            else:
                missed.append(n)
        if best is None or len(missed) < len(best[1]):
            best = (placed, sorted(missed))
        if not missed:
            break
    return best


def pick_callers(plan, model):
    if model.get('callers') is not None:
        return list(model['callers'])
    deg = {}
    for e in plan.edges:
        if e['from'] in plan.nodes:
            deg[e['from']] = deg.get(e['from'], 0) + 1
    order = [n for l in plan.layers for n in l]
    ranked = sorted((n for n in deg if deg[n] >= 3), key=lambda n: (-deg[n], order.index(n)))
    return ranked[:4]


def render(plan, model, lay, labels, model_hash):
    T = TEXT[plan.lang]
    R, Y, rowtop, rowbot = lay['R'], lay['Y'], lay['rowtop'], lay['rowbot']
    edges = plan.edges
    callers = pick_callers(plan, model)
    colors = {c: CALLER_COLORS[i % len(CALLER_COLORS)] for i, c in enumerate(callers)}

    def col(a):
        if a in colors:
            return colors[a]
        return BOUNDARY_COLOR if a in plan.bounds else OTHER_COLOR

    def name(n):
        return plan.bounds[n]['name'] if n in plan.bounds else plan.nodes[n]['name']

    rev = [e['n'] for e in plan.E if e['rev']]
    has_tech = any(e.get('tech') for e in edges)
    head = T['down'] + (T['except'].format(', '.join(map(str, rev))) if rev else '')
    head += T['label'] if has_tech else T['label_short']
    note_lines = []
    for para in ([head] if edges else []) + ([model['note']] if model.get('note') else []):
        note_lines += wrap(para, max(60, int((plan.W - 60) / 6.6)))

    rows = [(i + 1, name(e['from']) + ' → ' + name(e['to']), e.get('description') or e.get('label', ''),
             e.get('protocol') or e.get('tech', '')) for i, e in enumerate(edges)]
    c1 = 70
    c2 = c1 + max([tw(r[1], 12, True) for r in rows] + [tw(T['cols'][1], 11)]) + 30
    c3 = c2 + max([tw(r[2], 12) for r in rows] + [tw(T['cols'][2], 11)]) + 30
    has_proto = any(r[3] for r in rows)
    table_w = c3 + (max([tw(r[3], 11) for r in rows] + [tw(T['cols'][3], 11)]) + 30 if has_proto else 0)

    kinds = [k for k in STYLE if any(v['kind'] == k for v in plan.nodes.values())]
    leg_kinds = [(k, T['kinds'][k]) for k in kinds]
    leg_callers = [(name(c), colors[c]) for c in callers if any(e['from'] == c for e in edges)]
    for bid in plan.bounds:
        if bid not in colors and any(e['from'] == bid for e in edges):
            leg_callers.append((name(bid), BOUNDARY_COLOR))
    if any(col(e['from']) == OTHER_COLOR for e in edges) and leg_callers:
        leg_callers.append((T['others'], OTHER_COLOR))
    kinds_w = sum(29 + tw(t, 11.5) + 24 for _, t in leg_kinds)
    callers_w = (tw(T['callers'], 11.5) + 14 + sum(28 + tw(t, 11.5) + 16 for t, _ in leg_callers)) if leg_callers else 0
    one_line = 30 + kinds_w + 10 + callers_w <= plan.W - 30

    W = int(max(plan.W, table_w if edges else 0, 30 + max(kinds_w, callers_w) + 30))
    TOP = 54 + 16 * len(note_lines)
    diag_h = max(rowbot.values()) + 30
    leg_y = TOP + diag_h + 12
    leg_h = 0 if one_line or not leg_callers else 24
    table_y = leg_y + leg_h + 48
    H = int(table_y + (60 + len(edges) * 24 + 30 if edges else 0))

    o = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="{W}" height="{H}" font-family="{FONT}">',
         f'<!-- c4-model sha256:{model_hash} -->',
         f'<title>{escape(model["title"])}</title>', '<defs>']
    for c in sorted({col(e['from']) for e in edges}):
        o.append(f'<marker id="m{c[1:]}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto">'
                 f'<path d="M0,1 L9,5 L0,9 z" fill="{c}"/></marker>')
    o.append('</defs>')
    o.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    o.append(f'<text x="30" y="30" font-size="18" font-weight="bold" fill="#2C2C2A">{escape(model["title"])}</text>')
    for i, line in enumerate(note_lines):
        o.append(f'<text x="30" y="{50 + 16 * i}" font-size="11.5" fill="#5F5E5A">{escape(line)}</text>')
    o.append(f'<g transform="translate(0,{TOP})">')
    for r, label in enumerate(plan.labels):
        t, b = rowtop[r], rowbot[r]
        if r % 2 == 0:
            o.append(f'<rect x="0" y="{t - 8}" width="{W}" height="{b - t + 16}" fill="#FAF9F5"/>')
        parts = wrap(label, 16)
        y0 = (t + b) / 2 + 4 - (len(parts) - 1) * 8
        for i, part in enumerate(parts):
            o.append(f'<text x="30" y="{y0 + i * 16}" font-size="12.5" font-weight="bold" fill="#5F5E5A">{escape(part)}</text>')
    for bid, bd in plan.bounds.items():
        r = plan.rowof[bid]
        x, w = plan.x0[bid], plan.x1[bid] - plan.x0[bid]
        o.append(f'<rect x="{x:.1f}" y="{rowtop[r]}" width="{w:.1f}" height="{rowbot[r] - rowtop[r]}" rx="12" '
                 f'fill="#F3F2EC" stroke="#B4B2A9" stroke-dasharray="6 4"/>')
        if bd['name']:
            o.append(f'<text x="{x + 12:.1f}" y="{rowtop[r] + 19}" font-size="11" font-weight="bold" fill="#5F5E5A">{escape(bd["name"])}</text>')
    for n, nd in plan.nodes.items():
        f, s, tc = STYLE[nd['kind']]
        x, w, y = plan.x0[n], plan.box_w, Y[plan.rowof[n]]
        if nd['kind'] == 'db':
            o.append(f'<path d="M{x:.1f},{y + 8} a{w / 2},8 0 0 1 {w},0 v{BH - 16} a{w / 2},8 0 0 1 -{w},0 z" '
                     f'fill="{f}" stroke="{s}" stroke-width="1.5"/>'
                     f'<path d="M{x:.1f},{y + 8} a{w / 2},8 0 0 0 {w},0" fill="none" stroke="{s}" stroke-width="1.5"/>')
            ty = y + BH / 2 + 6
        else:
            rx = BH / 2 if nd['kind'] == 'person' else 9
            dash = ' stroke-dasharray="6 4"' if nd['kind'] == 'ext' else ''
            o.append(f'<rect x="{x:.1f}" y="{y}" width="{w}" height="{BH}" rx="{rx}" fill="{f}" stroke="{s}" stroke-width="1.5"{dash}/>')
            ty = y + BH / 2
        if not nd['tech']:
            ty += 6
        fs = 13
        while fs > 10 and tw(nd['name'], fs, True) > w - 16:
            fs -= 0.5
        o.append(f'<text x="{x + w / 2:.1f}" y="{ty - 2}" text-anchor="middle" font-size="{fs}" font-weight="bold" fill="{tc}">{escape(nd["name"])}</text>')
        if nd['tech']:
            o.append(f'<text x="{x + w / 2:.1f}" y="{ty + 14}" text-anchor="middle" font-size="10" fill="{tc}" opacity="0.85">{escape(nd["tech"])}</text>')
    marks = []
    for n in R:
        pts = drawn(plan, R, n)
        c = col(edges[n - 1]['from'])
        d = 'M' + ' L'.join(f'{x:.1f},{y:.1f}' for x, y in pts)
        o.append(f'<path d="{d}" fill="none" stroke="{c}" stroke-width="1.6" marker-end="url(#m{c[1:]})"/>')
        bx, by = badge_pos(pts)
        marks.append((n, bx, by, c))
    for n, bx, by, c in marks:
        o.append(f'<circle cx="{bx:.1f}" cy="{by:.1f}" r="8.5" fill="{c}" stroke="#fff" stroke-width="1.5"/>'
                 f'<text x="{bx:.1f}" y="{by + 3.6:.1f}" text-anchor="middle" font-size="10" font-weight="bold" fill="#fff">{n}</text>')
    for n, (rect, kind) in sorted(labels.items()):
        e = edges[n - 1]
        x0, y0, x1, _ = rect
        anchor, tx = ('end', x1 - 2) if kind == 'l' else ('start', x0 + 2)
        o.append(f'<text x="{tx:.1f}" y="{y0 + 12:.1f}" font-size="11" fill="#2C2C2A" stroke="#fff" stroke-width="3" '
                 f'paint-order="stroke" text-anchor="{anchor}">{escape(e.get("label", ""))}</text>')
        if e.get('tech'):
            o.append(f'<text x="{tx:.1f}" y="{y0 + 25:.1f}" font-size="10" font-weight="bold" fill="{col(e["from"])}" stroke="#fff" '
                     f'stroke-width="3" paint-order="stroke" text-anchor="{anchor}">[{escape(e["tech"])}]</text>')
    o.append('</g>')

    x = 30
    for k, t in leg_kinds:
        f, s, _ = STYLE[k]
        rx = 7 if k == 'person' else 3
        dash = ' stroke-dasharray="4 3"' if k == 'ext' else ''
        o.append(f'<rect x="{x:.1f}" y="{leg_y - 11}" width="22" height="14" rx="{rx}" fill="{f}" stroke="{s}"{dash}/>'
                 f'<text x="{x + 29:.1f}" y="{leg_y}" font-size="11.5" fill="#444441">{escape(t)}</text>')
        x += 29 + tw(t, 11.5) + 24
    if leg_callers:
        if one_line:
            x += 10
        else:
            x, leg_y = 30, leg_y + 24
        o.append(f'<text x="{x:.1f}" y="{leg_y}" font-size="11.5" fill="#444441">{escape(T["callers"])}</text>')
        x += tw(T['callers'], 11.5) + 14
        for t, c in leg_callers:
            o.append(f'<line x1="{x:.1f}" y1="{leg_y - 4}" x2="{x + 22:.1f}" y2="{leg_y - 4}" stroke="{c}" stroke-width="2.5"/>'
                     f'<text x="{x + 28:.1f}" y="{leg_y}" font-size="11.5" fill="#444441">{escape(t)}</text>')
            x += 28 + tw(t, 11.5) + 16

    if edges:
        ty = table_y
        o.append(f'<text x="30" y="{ty}" font-size="14" font-weight="bold" fill="#2C2C2A">{escape(T["table"])}</text>'
                 f'<line x1="30" y1="{ty + 10}" x2="{W - 30}" y2="{ty + 10}" stroke="#B4B2A9"/>')
        for cx, t in zip((30, c1, c2, c3), T['cols'][:4 if has_proto else 3]):
            o.append(f'<text x="{cx:.0f}" y="{ty + 28}" font-size="11" fill="#5F5E5A">{escape(t)}</text>')
        y = ty + 50
        for (n, pair, what, proto), e in zip(rows, edges):
            c = col(e['from'])
            o.append(f'<circle cx="38" cy="{y - 4}" r="8.5" fill="{c}"/><text x="38" y="{y - 0.4}" text-anchor="middle" '
                     f'font-size="10" font-weight="bold" fill="#fff">{n}</text>')
            o.append(f'<text x="{c1}" y="{y}" font-size="12" font-weight="bold" fill="#2C2C2A">{escape(pair)}</text>'
                     f'<text x="{c2:.0f}" y="{y}" font-size="12" fill="#2C2C2A">{escape(what)}</text>'
                     f'<text x="{c3:.0f}" y="{y}" font-size="11" fill="#5F5E5A">{escape(proto)}</text>')
            y += 24
    o.append('</svg>')
    return '\n'.join(o) + '\n', W, H


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


def to_png(svg_path, png_path, W, H, scale):
    chrome = find_chrome()
    png_path = Path(png_path).resolve()
    png_path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='docs-c4-') as work:
        result = subprocess.run(
            [chrome, '--headless=new', '--disable-gpu', '--hide-scrollbars', '--no-first-run',
             '--no-default-browser-check', f'--force-device-scale-factor={scale}', f'--window-size={W},{H}',
             f'--user-data-dir={Path(work) / "profile"}', f'--screenshot={png_path}', Path(svg_path).resolve().as_uri()],
            capture_output=True, text=True, timeout=120,
        )
    if result.returncode != 0 or not png_path.exists():
        fail(f'chrome failed (exit {result.returncode}): {result.stderr.strip()[-400:]}')
    return png_path


def main():
    ap = argparse.ArgumentParser(prog='render.py', description='Lay out and render a C4 diagram model to SVG (and PNG via headless Chrome).')
    ap.add_argument('model', help='model JSON, e.g. docs/architecture/c4-container.json')
    ap.add_argument('-o', '--out', help='SVG path (default: <model>.svg)')
    ap.add_argument('--png', help='also write a PNG here (needs Chrome/Chromium)')
    ap.add_argument('--scale', type=float, default=1.0, help='PNG device scale factor (default 1)')
    ap.add_argument('--seed', type=int, help='layout seed (default: layout.seed in the model, else 1)')
    ap.add_argument('--restarts', type=int, default=4)
    ap.add_argument('--iterations', type=int, help='annealing steps per restart (default: max(2000, 80 × edges))')
    args = ap.parse_args()

    checker = load_checker()
    path = Path(args.model)
    model = checker.load(path)
    errors = [f for f in checker.validate(model, str(path)) if f.severity == 'ERROR']
    if errors:
        for f in errors:
            print(f, file=sys.stderr)
        fail(f'{len(errors)} model error(s); fix them (scripts/check-c4.py {path}) and re-run')

    layout = model.get('layout', {})
    seed = args.seed if args.seed is not None else layout.get('seed', 1)
    plan = Plan(model, layout.get('box_width', 240), layout.get('gap', 150))
    rng = random.Random(seed)
    iterations = args.iterations or max(2000, 80 * len(plan.E))
    states = anneal(plan, rng, max(1, args.restarts), iterations)

    chosen = None
    for st in states:
        lay = plan.build(*st)
        placed, missed = place_labels(plan, lay, rng, 200)
        if chosen is None or len(missed) < len(chosen[2]):
            chosen = (lay, placed, missed)
        if not missed:
            break
    lay, placed, missed = chosen
    svg, W, H = render(plan, model, lay, placed, checker.model_hash(model))
    out = Path(args.out) if args.out else path.with_suffix('.svg')
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(svg, encoding='utf-8')
    cross, overlap, _ = metrics(lay['R'])
    print(f'{out} {W}x{H} seed={seed} edges={len(plan.E)} crossings={cross} overlaps={overlap} '
          f'labels={len(placed)}/{len(plan.E)}')
    if args.png:
        print(to_png(out, args.png, W, H, args.scale))
    if missed:
        fail(f'labels not placed for edges {", ".join(map(str, missed))}: shorten them, reorder layer nodes or try another --seed', 2)


if __name__ == '__main__':
    main()
