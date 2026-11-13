#!/usr/bin/env python3
import argparse
import re
import sys
from pathlib import Path

try:
    from docx import Document
    from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_LINE_SPACING
    from docx.oxml.ns import qn
except ImportError:
    sys.stderr.write("python-docx is required: pip install python-docx\n")
    sys.exit(1)

DASH = "–"
FONT = "Times New Roman"
CODE_FONTS = {"Courier New", "Consolas", "Courier"}
TOL_MM = 1.0
MAX_PER_RULE = 5

RE_FIG = re.compile(r"^Рисунок\s+([А-Я]?\.?\d+(?:\.\d+)?)\s*(.*)$")
RE_TBL = re.compile(r"^Таблица\s+([А-Я]?\.?\d+(?:\.\d+)?)\s*(.*)$")
RE_CITE = re.compile(r"\[(\d+(?:\s*[,–-]\s*(?:\d+|с\.\s*\d+(?:\s*[–-]\s*\d+)?))*)\]")
RE_STRUCT_REQUIRED = {
    "СОДЕРЖАНИЕ": re.compile(r"^СОДЕРЖАНИЕ$"),
    "ВВЕДЕНИЕ": re.compile(r"^ВВЕДЕНИЕ$"),
    "ЗАКЛЮЧЕНИЕ": re.compile(r"^ЗАКЛЮЧЕНИЕ$"),
    "СПИСОК ИСПОЛЬЗОВАННЫХ ИСТОЧНИКОВ": re.compile(r"^(СПИСОК.*|БИБЛИОГРАФИЧЕСКИЙ СПИСОК)$"),
}


class Report:
    def __init__(self):
        self.items = []
        self.counts = {}

    def add(self, severity, rule, where, message):
        n = self.counts.get(rule, 0) + 1
        self.counts[rule] = n
        if n <= MAX_PER_RULE:
            self.items.append((severity, rule, where, message))

    def finish(self):
        for rule, n in self.counts.items():
            if n > MAX_PER_RULE:
                sev = next(s for s, r, _, _ in self.items if r == rule)
                self.items.append((sev, rule, "…", f"ещё {n - MAX_PER_RULE} аналогичных"))


def mm(emu):
    return emu / 36000.0


def has_page_field(section):
    xml = section.footer._element.xml
    return "PAGE" in xml and "instrText" in xml


def effective_font(run, par):
    if run.font.name:
        return run.font.name
    if par.style is not None and par.style.font is not None and par.style.font.name:
        return par.style.font.name
    return None


def check_layout(doc, rep):
    for idx, sec in enumerate(doc.sections, 1):
        where = f"section {idx}"
        if abs(mm(sec.page_width) - 210) > TOL_MM or abs(mm(sec.page_height) - 297) > TOL_MM:
            rep.add("ERROR", "page-size", where,
                    f"формат {mm(sec.page_width):.0f}×{mm(sec.page_height):.0f} мм, нужен A4 210×297. Задать A4")
        expected = {"left": 30, "right": 15, "top": 20, "bottom": 20}
        actual = {
            "left": mm(sec.left_margin), "right": mm(sec.right_margin),
            "top": mm(sec.top_margin), "bottom": mm(sec.bottom_margin),
        }
        for side, want in expected.items():
            if abs(actual[side] - want) > TOL_MM:
                rep.add("ERROR", "margins", where,
                        f"поле {side} {actual[side]:.1f} мм, нужно {want} мм (30/15/20/20: лев/прав/верх/низ)")
    if not any(has_page_field(s) for s in doc.sections):
        rep.add("ERROR", "page-numbers", "footer", "нет номера страницы (поле PAGE) в нижнем колонтитуле")
    else:
        for idx, sec in enumerate(doc.sections, 1):
            if has_page_field(sec):
                align = sec.footer.paragraphs[0].alignment
                if align != WD_ALIGN_PARAGRAPH.CENTER:
                    rep.add("WARN", "page-numbers", f"section {idx} footer",
                            "номер страницы не по центру. Выровнять по центру внизу")
    if len(doc.sections) < 2:
        rep.add("WARN", "title-page", "document",
                "одна секция: титульный лист не отделён, номер на нём будет показан. Выделить титул в отдельную секцию")


def is_body(par):
    return par.style.name == "Normal" and par.text.strip() and not par.text.startswith("\t")


def check_body(doc, rep):
    normal = doc.styles["Normal"]
    if normal.font.name != FONT:
        rep.add("ERROR", "font", "style Normal", f"шрифт {normal.font.name!r}, нужен {FONT}")
    if normal.font.size is None or normal.font.size.pt not in (12.0, 13.0, 14.0):
        size = normal.font.size.pt if normal.font.size else None
        rep.add("ERROR", "font-size", "style Normal", f"размер {size}, нужен 12–14 пт (по умолчанию 14)")

    in_title = len(doc.sections) > 1
    for pidx, par in enumerate(doc.paragraphs, 1):
        if in_title and par._p.pPr is not None and par._p.pPr.find(qn("w:sectPr")) is not None:
            in_title = False
            continue
        if in_title:
            continue
        for run in par.runs:
            name = effective_font(run, par)
            if name and name != FONT and name not in CODE_FONTS:
                rep.add("ERROR", "font", f"paragraph {pidx}", f"шрифт {name!r} в «{par.text[:40]}». Заменить на {FONT}")
                break
        if not is_body(par):
            continue
        text = par.text.strip()
        if RE_FIG.match(text) or RE_TBL.match(text):
            continue
        pf = par.paragraph_format
        style_pf = par.style.paragraph_format
        line_rule = pf.line_spacing_rule or style_pf.line_spacing_rule
        spacing = pf.line_spacing or style_pf.line_spacing
        if text.startswith(("–", "-", "—")) or re.match(r"^\d+[.)]\s", text):
            continue
        if len(text) < 60:
            continue
        ok_spacing = line_rule == WD_LINE_SPACING.ONE_POINT_FIVE or (
            line_rule == WD_LINE_SPACING.MULTIPLE and spacing is not None and abs(float(spacing) - 1.5) < 0.05
        )
        if not ok_spacing:
            rep.add("ERROR", "line-spacing", f"paragraph {pidx}", f"межстрочный интервал не 1,5 в «{text[:40]}…»")
        indent = pf.first_line_indent if pf.first_line_indent is not None else style_pf.first_line_indent
        if indent is None or abs(mm(indent) - 12.5) > 1.0:
            got = f"{mm(indent):.1f}" if indent is not None else "нет"
            rep.add("ERROR", "indent", f"paragraph {pidx}", f"абзацный отступ {got} мм, нужно 12,5 мм в «{text[:40]}…»")
        align = par.alignment if par.alignment is not None else style_pf.alignment
        if align != WD_ALIGN_PARAGRAPH.JUSTIFY:
            rep.add("WARN", "alignment", f"paragraph {pidx}", f"не по ширине в «{text[:40]}…»")


def check_headings(doc, rep):
    headings = [(i, p) for i, p in enumerate(doc.paragraphs, 1) if p.style.name.startswith("Heading")]
    found = {key: False for key in RE_STRUCT_REQUIRED}
    chapter = 0
    for idx, par in headings:
        text = par.text.replace("\n", " ").strip()
        level = int(par.style.name.split()[-1]) if par.style.name.split()[-1].isdigit() else 1
        if text.endswith("."):
            rep.add("ERROR", "heading-period", f"paragraph {idx}", f"точка в конце заголовка «{text}»")
        upper = text.upper()
        for key, rx in RE_STRUCT_REQUIRED.items():
            if rx.match(upper):
                found[key] = True
                if text != text.upper():
                    rep.add("ERROR", "heading-case", f"paragraph {idx}", f"структурный заголовок «{text}» не прописными")
                if par.alignment != WD_ALIGN_PARAGRAPH.CENTER:
                    rep.add("ERROR", "heading-align", f"paragraph {idx}", f"«{text}» не по центру")
        if level == 1 and not re.match(r"^(ПРИЛОЖЕНИЕ|[А-ЯЁ\s]+$)", text):
            m = re.match(r"^(\d+)\s", text)
            if not m:
                rep.add("ERROR", "heading-number", f"paragraph {idx}", f"раздел без номера: «{text}»")
            else:
                chapter += 1
                if int(m.group(1)) != chapter:
                    rep.add("ERROR", "heading-number", f"paragraph {idx}",
                            f"номер раздела {m.group(1)}, ожидался {chapter}")
        pbb = par.paragraph_format.page_break_before
        if pbb is None:
            pbb = par.style.paragraph_format.page_break_before
        if level == 1 and not pbb:
            rep.add("WARN", "heading-newpage", f"paragraph {idx}", f"«{text}» не начинается с новой страницы")
    for key, ok in found.items():
        if not ok:
            rep.add("ERROR", "structure", "document", f"нет структурного элемента {key}")
    xml = doc.element.body.xml
    if "TOC" not in xml or "instrText" not in xml:
        rep.add("ERROR", "toc", "document", "нет поля оглавления (TOC)")
    return headings


def caption_paragraphs(doc):
    figs, tbls = [], []
    for idx, par in enumerate(doc.paragraphs, 1):
        text = par.text.strip()
        m = RE_FIG.match(text)
        if m:
            figs.append((idx, par, m.group(1), m.group(2)))
            continue
        m = RE_TBL.match(text)
        if m and par.style.name == "Caption":
            tbls.append((idx, par, m.group(1), m.group(2)))
    return figs, tbls


def check_caption_format(kind, entries, rep):
    seq = {}
    for idx, par, label, rest in entries:
        if not rest.startswith(DASH + " "):
            rep.add("ERROR", f"{kind}-caption", f"paragraph {idx}",
                    f"подпись «{par.text[:50]}»: после номера нужен « {DASH} » (тире с пробелами)")
        if par.text.strip().endswith("."):
            rep.add("ERROR", f"{kind}-caption", f"paragraph {idx}", "точка в конце подписи")
        scope, _, n = label.rpartition(".")
        seq.setdefault(scope, []).append((idx, int(n) if n else int(label)))
    for scope, items in seq.items():
        for pos, (idx, n) in enumerate(items, 1):
            if n != pos:
                rep.add("ERROR", f"{kind}-numbering", f"paragraph {idx}",
                        f"номер {n}, ожидался {pos}{' (раздел ' + scope + ')' if scope else ''}")


def check_figures(doc, rep, figs):
    pics = []
    for idx, par in enumerate(doc.paragraphs, 1):
        if par._p.findall(".//" + qn("w:drawing")):
            pics.append((idx, par))
    caption_idx = {i for i, _, _, _ in figs}
    for idx, par in pics:
        if idx + 1 not in caption_idx:
            rep.add("ERROR", "figure-caption", f"paragraph {idx}", "рисунок без подписи «Рисунок N – …» под ним")
        elif par.alignment != WD_ALIGN_PARAGRAPH.CENTER:
            rep.add("WARN", "figure-align", f"paragraph {idx}", "рисунок не по центру")
    pic_idx = {i for i, _ in pics}
    for idx, par, label, _ in figs:
        if idx - 1 not in pic_idx:
            rep.add("ERROR", "figure-caption", f"paragraph {idx}",
                    f"подпись «Рисунок {label}» без рисунка над ней (подпись под рисунком)")
        elif par.alignment != WD_ALIGN_PARAGRAPH.CENTER:
            rep.add("ERROR", "figure-caption", f"paragraph {idx}", "подпись рисунка не по центру")
    for idx, par in enumerate(doc.paragraphs, 1):
        if "[Рисунок не найден" in par.text:
            rep.add("ERROR", "figure-missing", f"paragraph {idx}", par.text.strip())


def check_tables(doc, rep, tbls):
    body = doc.element.body
    children = list(body.iterchildren())
    paragraphs = {p._p: i for i, p in enumerate(doc.paragraphs, 1)}
    caption_by_el = {p._p: (label, p) for _, p, label, _ in tbls}
    for pos, el in enumerate(children):
        if el.tag != qn("w:tbl"):
            continue
        prev = children[pos - 1] if pos else None
        if prev is None or prev not in caption_by_el:
            rep.add("ERROR", "table-caption", f"table {pos}", "таблица без подписи «Таблица N – …» над ней")
            continue
        label, par = caption_by_el[prev]
        if par.alignment not in (None, WD_ALIGN_PARAGRAPH.LEFT):
            rep.add("ERROR", "table-caption", f"paragraph {paragraphs[prev]}", "подпись таблицы не слева")
    for el, (label, par) in caption_by_el.items():
        nxt = el.getnext()
        if nxt is None or nxt.tag != qn("w:tbl"):
            rep.add("ERROR", "table-caption", f"paragraph {paragraphs[el]}",
                    f"подпись «Таблица {label}» без таблицы под ней")


def check_references(doc, rep, figs, tbls):
    text_pars = [p.text for p in doc.paragraphs if p.style.name != "Caption" and not p.style.name.startswith("Heading")]
    joined = "\n".join(text_pars)
    for _, _, label, _ in figs:
        pat = rf"рисун(?:ок|ке|ка|ку|ком|ки)\s+{re.escape(label)}(?!\d)"
        if not re.search(pat, joined, re.I):
            rep.add("ERROR", "figure-reference", f"Рисунок {label}", "нет ссылки в тексте («на рисунке N»)")
    for _, _, label, _ in tbls:
        pat = rf"таблиц(?:а|е|ы|у|ей)\s+{re.escape(label)}(?!\d)"
        if not re.search(pat, joined, re.I):
            rep.add("ERROR", "table-reference", f"Таблица {label}", "нет ссылки в тексте («в таблице N»)")
    letters = re.findall(r"^ПРИЛОЖЕНИЕ\s+([А-Я])", "\n".join(p.text for p in doc.paragraphs), re.M)
    for letter in letters:
        pat = rf"приложени(?:е|и|я|ю|ем)\s+{letter}(?![А-Яа-я])"
        count = len(re.findall(pat, joined, re.I))
        if count == 0:
            rep.add("ERROR", "appendix-reference", f"Приложение {letter}", "нет ссылки в тексте («в приложении А»)")


def check_sources(doc, rep, headings):
    start = None
    for idx, par in headings:
        if re.match(r"^(СПИСОК|БИБЛИОГРАФИЧЕСКИЙ)", par.text.strip().upper()):
            start = idx
            break
    if start is None:
        return
    entries = []
    for idx, par in enumerate(doc.paragraphs, 1):
        if idx <= start:
            continue
        if par.style.name.startswith("Heading"):
            break
        m = re.match(r"^(\d+)\.\s+\S", par.text.strip())
        if m:
            entries.append((idx, int(m.group(1))))
    if not entries:
        rep.add("ERROR", "sources", f"paragraph {start}", "в списке источников нет нумерованных записей «1. …»")
        return
    for pos, (idx, n) in enumerate(entries, 1):
        if n != pos:
            rep.add("ERROR", "sources-numbering", f"paragraph {idx}", f"номер {n}, ожидался {pos}")
    total = len(entries)
    cited = set()
    body_text = "\n".join(
        p.text for i, p in enumerate(doc.paragraphs, 1) if i < start and not p.style.name.startswith("Heading")
    )
    for m in RE_CITE.finditer(body_text):
        inner = m.group(1)
        inner = re.sub(r"с\.\s*\d+(?:\s*[–-]\s*\d+)?", "", inner)
        for part in re.split(r"\s*,\s*", inner):
            part = part.strip()
            if not part:
                continue
            rng = re.match(r"^(\d+)\s*[–-]\s*(\d+)$", part)
            if rng:
                cited.update(range(int(rng.group(1)), int(rng.group(2)) + 1))
            elif part.isdigit():
                cited.add(int(part))
    for n in sorted(cited):
        if n > total or n < 1:
            rep.add("ERROR", "citation", "text", f"ссылка [{n}] на несуществующий источник (в списке {total})")
    for pos, (idx, n) in enumerate(entries, 1):
        if n not in cited:
            rep.add("WARN", "citation", f"paragraph {idx}", f"источник {n} не процитирован в тексте")
    if total < 1:
        rep.add("ERROR", "sources", "document", "пустой список источников")


def check_placeholders(doc, rep):
    pattern = re.compile(r"\[(ДАННЫЕ|ТЕМА|проверить|TODO|ЗАПОЛНИТЬ)[^\]]*\]", re.I)
    for idx, par in enumerate(doc.paragraphs, 1):
        m = pattern.search(par.text)
        if m:
            rep.add("ERROR", "placeholder", f"paragraph {idx}", f"остался плейсхолдер {m.group(0)}")


def main():
    ap = argparse.ArgumentParser(description="Mechanical GOST 7.32 conformance check of a thesis .docx")
    ap.add_argument("docx")
    args = ap.parse_args()
    path = Path(args.docx)
    if not path.is_file():
        sys.stderr.write(f"not found: {path}\n")
        return 1
    try:
        doc = Document(str(path))
    except Exception as exc:
        sys.stderr.write(f"cannot open {path}: {exc}\n")
        return 1
    rep = Report()
    check_layout(doc, rep)
    check_body(doc, rep)
    headings = check_headings(doc, rep)
    figs, tbls = caption_paragraphs(doc)
    check_caption_format("figure", figs, rep)
    check_caption_format("table", tbls, rep)
    check_figures(doc, rep, figs)
    check_tables(doc, rep, tbls)
    check_references(doc, rep, figs, tbls)
    check_sources(doc, rep, headings)
    check_placeholders(doc, rep)
    rep.finish()
    order = {"ERROR": 0, "WARN": 1}
    for sev, rule, where, msg in sorted(rep.items, key=lambda x: order[x[0]]):
        sys.stdout.write(f"{path.name}:{where} — {sev}: [{rule}] {msg}\n")
    errors = sum(1 for s, *_ in rep.items if s == "ERROR")
    warns = sum(1 for s, *_ in rep.items if s == "WARN")
    sys.stdout.write(f"{errors} error(s), {warns} warning(s)\n")
    return 2 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
