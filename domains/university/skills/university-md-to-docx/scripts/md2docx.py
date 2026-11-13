#!/usr/bin/env python3
import argparse
import re
import sys
from pathlib import Path

try:
    from docx import Document
    from docx.enum.section import WD_SECTION
    from docx.enum.table import WD_TABLE_ALIGNMENT
    from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_BREAK, WD_LINE_SPACING, WD_TAB_ALIGNMENT
    from docx.oxml import OxmlElement
    from docx.oxml.ns import qn
    from docx.shared import Cm, Pt, RGBColor
except ImportError:
    sys.stderr.write("python-docx is required: pip install python-docx\n")
    sys.exit(1)

FONT = "Times New Roman"
CODE_FONT = "Courier New"
BODY_PT = 14
TABLE_PT = 12
CODE_PT = 12
INDENT_CM = 1.25
TEXT_WIDTH_CM = 16.5
DASH = "–"
APPENDIX_LETTERS = "АБВГДЕЖИКЛМНПРСТУФХЦШЩЭЮЯ"

STRUCTURAL = re.compile(
    r"^(РЕФЕРАТ|АННОТАЦИЯ|СОДЕРЖАНИЕ|ТЕРМИНЫ.*|ОПРЕДЕЛЕНИЯ|ОБОЗНАЧЕНИЯ.*|СОКРАЩЕНИЯ.*|ПЕРЕЧЕНЬ.*"
    r"|ВВЕДЕНИЕ|ЗАКЛЮЧЕНИЕ|СПИСОК.*|БИБЛИОГРАФИЧЕСКИЙ СПИСОК)$",
    re.I,
)
SOURCES = re.compile(r"^(СПИСОК.*|БИБЛИОГРАФИЧЕСКИЙ СПИСОК)$", re.I)
APPENDIX = re.compile(r"^приложение\b[\s:.–-]*(.*)$", re.I)
LEADING_NUMBER = re.compile(r"^\d+(\.\d+)*\.?\s+")

RE_HEADING = re.compile(r"^(#{1,3})\s+(.+?)\s*#*\s*$")
RE_IMAGE = re.compile(r"^!\[(.*?)\]\((.*?)\)\s*(\{#fig:([\w-]+)\})?\s*$")
RE_TBL_CAPTION = re.compile(r"^Таблица:\s*(.*?)\s*(\{#tbl:([\w-]+)\})?\s*$")
RE_EQUATION = re.compile(r"^\$\$(.+?)\$\$\s*(\{#eq:([\w-]+)\})?\s*$")
RE_BULLET = re.compile(r"^\s*[-*+]\s+(.*)$")
RE_ORDERED = re.compile(r"^\s*\d+[.)]\s+(.*)$")
RE_REF = re.compile(r"@(fig|tbl|eq):([\w-]+)")
RE_INLINE = re.compile(r"(\*\*.+?\*\*|\*.+?\*|`.+?`)")


def parse_frontmatter(lines):
    meta = {}
    if not lines or lines[0].strip() != "---":
        return meta, lines
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            for raw in lines[1:i]:
                if ":" in raw and not raw.lstrip().startswith("#"):
                    key, value = raw.split(":", 1)
                    meta[key.strip()] = value.strip().strip("\"'")
            return meta, lines[i + 1:]
    return meta, lines


def parse_blocks(lines):
    blocks = []
    i = 0
    n = len(lines)
    pending_caption = None
    while i < n:
        line = lines[i].rstrip()
        stripped = line.strip()
        if not stripped:
            i += 1
            continue
        if stripped == "\\newpage":
            blocks.append({"t": "pagebreak"})
            i += 1
            continue
        if stripped.startswith("```"):
            code = []
            i += 1
            while i < n and not lines[i].strip().startswith("```"):
                code.append(lines[i].rstrip("\n"))
                i += 1
            i += 1
            blocks.append({"t": "code", "lines": code})
            continue
        m = RE_HEADING.match(stripped)
        if m:
            blocks.append({"t": "heading", "level": len(m.group(1)), "text": m.group(2)})
            i += 1
            continue
        m = RE_IMAGE.match(stripped)
        if m:
            blocks.append({"t": "image", "caption": m.group(1), "path": m.group(2), "id": m.group(4)})
            i += 1
            continue
        m = RE_EQUATION.match(stripped)
        if m:
            blocks.append({"t": "equation", "text": m.group(1).strip(), "id": m.group(3)})
            i += 1
            continue
        m = RE_TBL_CAPTION.match(stripped)
        if m:
            j = i + 1
            while j < n and not lines[j].strip():
                j += 1
            if j < n and lines[j].lstrip().startswith("|"):
                pending_caption = (m.group(1), m.group(3))
                i = j
                continue
        if stripped.startswith("|"):
            rows = []
            while i < n and lines[i].lstrip().startswith("|"):
                rows.append(lines[i].strip())
                i += 1
            caption, ident = pending_caption if pending_caption else (None, None)
            pending_caption = None
            blocks.append({"t": "table", "rows": rows, "caption": caption, "id": ident})
            continue
        if RE_BULLET.match(line) or RE_ORDERED.match(line):
            items = []
            ordered = bool(RE_ORDERED.match(line))
            while i < n:
                cur = lines[i].rstrip()
                mb = RE_BULLET.match(cur)
                mo = RE_ORDERED.match(cur)
                if mb or mo:
                    items.append((mb or mo).group(1).strip())
                    i += 1
                elif cur.strip() and cur.startswith((" ", "\t")) and items:
                    items[-1] += " " + cur.strip()
                    i += 1
                else:
                    break
            blocks.append({"t": "list", "ordered": ordered, "items": items})
            continue
        para = [stripped]
        i += 1
        while i < n:
            nxt = lines[i].rstrip()
            if (
                not nxt.strip()
                or nxt.strip().startswith(("```", "|", "#", "$$", "![", "\\newpage"))
                or RE_BULLET.match(nxt)
                or RE_ORDERED.match(nxt)
                or RE_TBL_CAPTION.match(nxt.strip())
            ):
                break
            para.append(nxt.strip())
            i += 1
        blocks.append({"t": "para", "text": " ".join(para)})
    return blocks


def classify_headings(blocks):
    appendix_index = 0
    for b in blocks:
        if b["t"] != "heading":
            continue
        text = b["text"].strip()
        b["kind"] = "section"
        if b["level"] == 1:
            ma = APPENDIX.match(text)
            if ma:
                if appendix_index >= len(APPENDIX_LETTERS):
                    raise SystemExit("too many appendices")
                b["kind"] = "appendix"
                b["letter"] = APPENDIX_LETTERS[appendix_index]
                b["title"] = ma.group(1).strip()
                appendix_index += 1
                continue
            bare = LEADING_NUMBER.sub("", text).strip()
            if STRUCTURAL.match(bare) and not LEADING_NUMBER.match(text):
                b["kind"] = "structural"
                b["text"] = bare.upper()
                b["sources"] = bool(SOURCES.match(bare))
                b["toc"] = bare.upper() == "СОДЕРЖАНИЕ"
                continue
        b["text"] = LEADING_NUMBER.sub("", text).strip()


def assign_numbers(blocks, mode):
    refs = {}
    counters = {"chapter": 0, "sub": 0, "subsub": 0}
    seq = {"fig": {}, "tbl": {}, "eq": {}}
    scope = ""
    warnings = []

    def next_label(kind):
        counters_for = seq[kind]
        counters_for[scope] = counters_for.get(scope, 0) + 1
        n = counters_for[scope]
        return f"{scope}.{n}" if scope else str(n)

    for b in blocks:
        t = b["t"]
        if t == "heading":
            lvl = b["level"]
            if b["kind"] == "appendix":
                scope = b["letter"]
                counters["sub"] = counters["subsub"] = 0
                b["number"] = b["letter"]
            elif b["kind"] == "structural":
                scope = ""
                b["number"] = ""
            elif scope and scope in APPENDIX_LETTERS:
                if lvl == 2:
                    counters["sub"] += 1
                    counters["subsub"] = 0
                    b["number"] = f"{scope}.{counters['sub']}"
                elif lvl == 3:
                    counters["subsub"] += 1
                    b["number"] = f"{scope}.{counters['sub']}.{counters['subsub']}"
                else:
                    b["number"] = ""
            else:
                if lvl == 1:
                    counters["chapter"] += 1
                    counters["sub"] = counters["subsub"] = 0
                    b["number"] = str(counters["chapter"])
                    scope = str(counters["chapter"]) if mode == "section" else ""
                elif lvl == 2:
                    counters["sub"] += 1
                    counters["subsub"] = 0
                    b["number"] = f"{counters['chapter']}.{counters['sub']}"
                else:
                    counters["subsub"] += 1
                    b["number"] = f"{counters['chapter']}.{counters['sub']}.{counters['subsub']}"
        elif t == "image":
            b["label"] = next_label("fig")
            if b["id"]:
                refs[("fig", b["id"])] = b["label"]
        elif t == "table":
            b["label"] = next_label("tbl")
            if b["id"]:
                refs[("tbl", b["id"])] = b["label"]
            if not b["caption"]:
                warnings.append(f"table {b['label']} has no caption line 'Таблица: ...'")
        elif t == "equation":
            b["label"] = next_label("eq")
            if b["id"]:
                refs[("eq", b["id"])] = b["label"]
    return refs, warnings


def resolve_refs(text, refs, warnings):
    def sub(m):
        key = (m.group(1), m.group(2))
        if key not in refs:
            warnings.append(f"unresolved reference {m.group(0)}")
            return "??"
        return refs[key]

    return RE_REF.sub(sub, text)


def set_run_font(run, size=None, name=FONT, bold=None, italic=None):
    run.font.name = name
    rpr = run._element.get_or_add_rPr()
    rfonts = rpr.find(qn("w:rFonts"))
    if rfonts is None:
        rfonts = OxmlElement("w:rFonts")
        rpr.insert(0, rfonts)
    for attr in ("w:ascii", "w:hAnsi", "w:eastAsia", "w:cs"):
        rfonts.set(qn(attr), name)
    if size is not None:
        run.font.size = Pt(size)
    if bold is not None:
        run.font.bold = bold
    if italic is not None:
        run.font.italic = italic
    run.font.color.rgb = RGBColor(0, 0, 0)


def style_font(style, size, bold=False):
    style.font.name = FONT
    style.font.size = Pt(size)
    style.font.bold = bold
    style.font.italic = False
    style.font.color.rgb = RGBColor(0, 0, 0)
    rpr = style.element.get_or_add_rPr()
    rfonts = rpr.find(qn("w:rFonts"))
    if rfonts is None:
        rfonts = OxmlElement("w:rFonts")
        rpr.insert(0, rfonts)
    for attr in list(rfonts.attrib):
        del rfonts.attrib[attr]
    for attr in ("w:ascii", "w:hAnsi", "w:eastAsia", "w:cs"):
        rfonts.set(qn(attr), FONT)


def configure_styles(doc):
    normal = doc.styles["Normal"]
    style_font(normal, BODY_PT)
    pf = normal.paragraph_format
    pf.line_spacing_rule = WD_LINE_SPACING.ONE_POINT_FIVE
    pf.first_line_indent = Cm(INDENT_CM)
    pf.space_before = Pt(0)
    pf.space_after = Pt(0)
    pf.alignment = WD_ALIGN_PARAGRAPH.JUSTIFY
    pf.widow_control = True

    for name in ("Heading 1", "Heading 2", "Heading 3"):
        st = doc.styles[name]
        style_font(st, BODY_PT, bold=True)
        hp = st.paragraph_format
        hp.line_spacing_rule = WD_LINE_SPACING.ONE_POINT_FIVE
        hp.first_line_indent = Cm(INDENT_CM)
        hp.alignment = WD_ALIGN_PARAGRAPH.LEFT
        hp.space_before = Pt(0)
        hp.space_after = Pt(12)
        hp.keep_with_next = True
    doc.styles["Heading 1"].paragraph_format.page_break_before = True

    cap = doc.styles["Caption"]
    style_font(cap, BODY_PT)
    cap.paragraph_format.first_line_indent = Cm(0)
    cap.paragraph_format.line_spacing_rule = WD_LINE_SPACING.ONE_POINT_FIVE

    for name in ("TOC 1", "TOC 2", "TOC 3"):
        try:
            st = doc.styles[name]
        except KeyError:
            st = doc.styles.add_style(name, 1)
        style_font(st, BODY_PT)
        st.paragraph_format.first_line_indent = Cm(0)
        st.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.LEFT
        st.paragraph_format.line_spacing_rule = WD_LINE_SPACING.SINGLE
        st.paragraph_format.space_after = Pt(6)
    doc.styles["TOC 2"].paragraph_format.left_indent = Cm(0.75)
    doc.styles["TOC 3"].paragraph_format.left_indent = Cm(1.5)


def configure_section(section):
    section.page_width = Cm(21.0)
    section.page_height = Cm(29.7)
    section.left_margin = Cm(3.0)
    section.right_margin = Cm(1.5)
    section.top_margin = Cm(2.0)
    section.bottom_margin = Cm(2.0)
    section.footer_distance = Cm(1.0)


def add_field(paragraph, instruction, placeholder="", size=BODY_PT):
    def fld(kind, dirty=False):
        run = paragraph.add_run()
        set_run_font(run, size)
        el = OxmlElement("w:fldChar")
        el.set(qn("w:fldCharType"), kind)
        if dirty:
            el.set(qn("w:dirty"), "true")
        run._element.append(el)

    fld("begin", dirty=True)
    run = paragraph.add_run()
    set_run_font(run, size)
    instr = OxmlElement("w:instrText")
    instr.set(qn("xml:space"), "preserve")
    instr.text = f" {instruction} "
    run._element.append(instr)
    fld("separate")
    r = paragraph.add_run(placeholder)
    set_run_font(r, size)
    fld("end")


def add_page_number_footer(section):
    section.footer.is_linked_to_previous = False
    p = section.footer.paragraphs[0]
    for r in list(p.runs):
        r._element.getparent().remove(r._element)
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.first_line_indent = Cm(0)
    add_field(p, "PAGE", "1")


def clear_footer(section):
    section.footer.is_linked_to_previous = False
    p = section.footer.paragraphs[0]
    for r in list(p.runs):
        r._element.getparent().remove(r._element)


def enable_update_fields(doc):
    settings = doc.settings.element
    el = OxmlElement("w:updateFields")
    el.set(qn("w:val"), "true")
    settings.append(el)


def add_runs(paragraph, text, size=None, bold=None):
    pos = 0
    for m in RE_INLINE.finditer(text):
        if m.start() > pos:
            r = paragraph.add_run(text[pos:m.start()])
            set_run_font(r, size, bold=bold)
        token = m.group(0)
        if token.startswith("**"):
            r = paragraph.add_run(token[2:-2])
            set_run_font(r, size, bold=True)
        elif token.startswith("`"):
            r = paragraph.add_run(token[1:-1])
            set_run_font(r, CODE_PT if size is None or size >= CODE_PT else size, name=CODE_FONT, bold=bold)
        else:
            r = paragraph.add_run(token[1:-1])
            set_run_font(r, size, bold=bold, italic=True)
        pos = m.end()
    if pos < len(text):
        r = paragraph.add_run(text[pos:])
        set_run_font(r, size, bold=bold)


def plain_paragraph(doc, text="", align=WD_ALIGN_PARAGRAPH.CENTER, bold=False, size=BODY_PT, indent=0.0,
                    left=0.0, spacing=WD_LINE_SPACING.SINGLE, after=0):
    p = doc.add_paragraph()
    p.alignment = align
    pf = p.paragraph_format
    pf.first_line_indent = Cm(indent)
    pf.left_indent = Cm(left)
    pf.line_spacing_rule = spacing
    pf.space_after = Pt(after)
    if text:
        r = p.add_run(text)
        set_run_font(r, size, bold=bold)
    return p


def build_title_page(doc, meta):
    def centered(key, bold=False, after=0, default=None):
        value = meta.get(key, default)
        if value:
            plain_paragraph(doc, value, bold=bold, after=after)

    centered("ministry", after=6)
    centered("university", bold=True, after=6)
    centered("faculty")
    centered("department", after=0)
    plain_paragraph(doc, "", after=0)
    plain_paragraph(doc, "", after=0)
    plain_paragraph(doc, "", after=0)
    work_type = meta.get("work_type", "ВЫПУСКНАЯ КВАЛИФИКАЦИОННАЯ РАБОТА")
    plain_paragraph(doc, work_type, bold=True, after=12)
    if meta.get("degree"):
        plain_paragraph(doc, meta["degree"], after=6)
    if meta.get("direction"):
        plain_paragraph(doc, f"Направление подготовки: {meta['direction']}", after=0)
    if meta.get("profile"):
        plain_paragraph(doc, f"Профиль: {meta['profile']}", after=0)
    plain_paragraph(doc, "", after=0)
    title = meta.get("title", "")
    plain_paragraph(doc, f"Тема: {title}" if title else "Тема: [ТЕМА]", bold=True, after=18)
    plain_paragraph(doc, "", after=0)

    def signature(name):
        p = plain_paragraph(doc, "", align=WD_ALIGN_PARAGRAPH.LEFT, after=0)
        p.paragraph_format.tab_stops.add_tab_stop(Cm(8.5), WD_TAB_ALIGNMENT.LEFT)
        p.paragraph_format.tab_stops.add_tab_stop(Cm(TEXT_WIDTH_CM), WD_TAB_ALIGNMENT.RIGHT)
        r = p.add_run(f"\t______________\t{name}")
        set_run_font(r, BODY_PT)
        plain_paragraph(doc, "", after=6)

    if meta.get("author"):
        group = f" группы {meta['group']}" if meta.get("group") else ""
        plain_paragraph(doc, f"Выполнил: студент{group}", align=WD_ALIGN_PARAGRAPH.LEFT, left=8.5, after=0)
        signature(meta["author"])
    if meta.get("supervisor"):
        pos = meta.get("supervisor_position", "")
        label = f"Научный руководитель: {pos}".strip().rstrip(",") if pos else "Научный руководитель:"
        plain_paragraph(doc, label, align=WD_ALIGN_PARAGRAPH.LEFT, left=8.5, after=0)
        signature(meta["supervisor"])
    if meta.get("advisor"):
        plain_paragraph(doc, "Консультант:", align=WD_ALIGN_PARAGRAPH.LEFT, left=8.5, after=0)
        signature(meta["advisor"])
    if meta.get("head"):
        plain_paragraph(doc, "Допустить к защите. Заведующий кафедрой:", align=WD_ALIGN_PARAGRAPH.LEFT, left=8.5,
                        after=0)
        signature(meta["head"])
    plain_paragraph(doc, "", after=0)
    city = meta.get("city", "")
    year = meta.get("year", "")
    footer = f"{city} {year}".strip()
    if footer:
        plain_paragraph(doc, footer, after=0)


def add_toc(doc):
    p = doc.add_paragraph()
    p.paragraph_format.first_line_indent = Cm(0)
    p.paragraph_format.line_spacing_rule = WD_LINE_SPACING.SINGLE
    add_field(p, 'TOC \\o "1-3" \\h \\z \\u', "Обновите поле: Ctrl+A, F9")


def add_heading(doc, b):
    kind = b["kind"]
    lvl = b["level"]
    if kind == "structural":
        h = doc.add_heading(level=1)
        h.alignment = WD_ALIGN_PARAGRAPH.CENTER
        h.paragraph_format.first_line_indent = Cm(0)
        r = h.add_run(b["text"])
        set_run_font(r, BODY_PT, bold=True)
        return h
    if kind == "appendix":
        h = doc.add_heading(level=1)
        h.alignment = WD_ALIGN_PARAGRAPH.CENTER
        h.paragraph_format.first_line_indent = Cm(0)
        r = h.add_run(f"ПРИЛОЖЕНИЕ {b['letter']}")
        set_run_font(r, BODY_PT, bold=True)
        if b["title"]:
            r.add_break(WD_BREAK.LINE)
            r2 = h.add_run(b["title"])
            set_run_font(r2, BODY_PT, bold=True)
        return h
    h = doc.add_heading(level=lvl)
    label = b["number"]
    r = h.add_run(f"{label} {b['text']}" if label else b["text"])
    set_run_font(r, BODY_PT, bold=True)
    return h


def add_paragraph(doc, text, refs, warnings):
    p = doc.add_paragraph()
    add_runs(p, resolve_refs(text, refs, warnings))
    return p


def add_list(doc, b, refs, warnings, in_sources):
    items = b["items"]
    count = len(items)
    for idx, item in enumerate(items, 1):
        p = doc.add_paragraph()
        p.paragraph_format.first_line_indent = Cm(INDENT_CM)
        text = resolve_refs(item, refs, warnings)
        if b["ordered"]:
            marker = f"{idx}." if in_sources else f"{idx})"
        else:
            marker = DASH
        r = p.add_run(f"{marker} ")
        set_run_font(r, BODY_PT)
        add_runs(p, text)


def add_image(doc, b, base_dir, refs, warnings):
    path = Path(b["path"])
    if not path.is_absolute():
        path = base_dir / path
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.first_line_indent = Cm(0)
    p.paragraph_format.line_spacing_rule = WD_LINE_SPACING.SINGLE
    p.paragraph_format.keep_with_next = True
    p.paragraph_format.space_before = Pt(6)
    if path.is_file():
        run = p.add_run()
        run.add_picture(str(path))
        shape = doc.inline_shapes[-1]
        max_w = Cm(TEXT_WIDTH_CM)
        if shape.width > max_w:
            ratio = max_w / shape.width
            shape.width = int(shape.width * ratio)
            shape.height = int(shape.height * ratio)
    else:
        warnings.append(f"image not found: {b['path']}")
        r = p.add_run(f"[Рисунок не найден: {b['path']}]")
        set_run_font(r, BODY_PT)
    cap = doc.add_paragraph(style="Caption")
    cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
    cap.paragraph_format.space_after = Pt(12)
    r = cap.add_run(f"Рисунок {b['label']} {DASH} ")
    set_run_font(r, BODY_PT)
    add_runs(cap, resolve_refs(b["caption"], refs, warnings))


def split_row(row):
    cells = row.strip().strip("|").split("|")
    return [c.strip() for c in cells]


def add_table(doc, b, refs, warnings):
    rows = [split_row(r) for r in b["rows"]]
    aligns = None
    body = []
    header = None
    for idx, row in enumerate(rows):
        if all(re.fullmatch(r":?-+:?", c) for c in row if c) and any(row):
            aligns = [
                WD_ALIGN_PARAGRAPH.CENTER if c.startswith(":") and c.endswith(":")
                else WD_ALIGN_PARAGRAPH.RIGHT if c.endswith(":")
                else WD_ALIGN_PARAGRAPH.LEFT
                for c in row
            ]
            continue
        if header is None and aligns is None:
            header = row
        else:
            body.append(row)
    if header is None:
        header = body.pop(0)
    ncols = max(len(header), *(len(r) for r in body)) if body else len(header)
    if b["caption"]:
        cap = doc.add_paragraph(style="Caption")
        cap.alignment = WD_ALIGN_PARAGRAPH.LEFT
        cap.paragraph_format.first_line_indent = Cm(0)
        cap.paragraph_format.keep_with_next = True
        cap.paragraph_format.space_before = Pt(6)
        r = cap.add_run(f"Таблица {b['label']} {DASH} ")
        set_run_font(r, BODY_PT)
        add_runs(cap, resolve_refs(b["caption"], refs, warnings))
    table = doc.add_table(rows=0, cols=ncols)
    table.style = "Table Grid"
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    for ridx, row in enumerate([header] + body):
        cells = table.add_row().cells
        for cidx in range(ncols):
            text = row[cidx] if cidx < len(row) else ""
            if not text and ridx > 0:
                text = DASH
            cp = cells[cidx].paragraphs[0]
            cp.paragraph_format.first_line_indent = Cm(0)
            cp.paragraph_format.line_spacing_rule = WD_LINE_SPACING.SINGLE
            if ridx == 0:
                cp.alignment = WD_ALIGN_PARAGRAPH.CENTER
            elif aligns and cidx < len(aligns):
                cp.alignment = aligns[cidx]
            else:
                cp.alignment = WD_ALIGN_PARAGRAPH.LEFT
            add_runs(cp, resolve_refs(text, refs, warnings), size=TABLE_PT, bold=True if ridx == 0 else None)
        if ridx == 0:
            trpr = table.rows[0]._tr.get_or_add_trPr()
            hdr = OxmlElement("w:tblHeader")
            hdr.set(qn("w:val"), "true")
            trpr.append(hdr)
        cant = OxmlElement("w:cantSplit")
        cant.set(qn("w:val"), "true")
        table.rows[-1]._tr.get_or_add_trPr().append(cant)
    spacer = doc.add_paragraph()
    spacer.paragraph_format.first_line_indent = Cm(0)
    spacer.paragraph_format.line_spacing_rule = WD_LINE_SPACING.SINGLE


def add_equation(doc, b, refs, warnings):
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.LEFT
    pf = p.paragraph_format
    pf.first_line_indent = Cm(0)
    pf.tab_stops.add_tab_stop(Cm(TEXT_WIDTH_CM / 2), WD_TAB_ALIGNMENT.CENTER)
    pf.tab_stops.add_tab_stop(Cm(TEXT_WIDTH_CM), WD_TAB_ALIGNMENT.RIGHT)
    text = resolve_refs(b["text"], refs, warnings)
    r = p.add_run(f"\t{text}\t({b['label']})")
    set_run_font(r, BODY_PT)


def add_code(doc, b):
    for line in b["lines"] or [""]:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.LEFT
        pf = p.paragraph_format
        pf.first_line_indent = Cm(0)
        pf.line_spacing_rule = WD_LINE_SPACING.SINGLE
        r = p.add_run(line.replace("\t", "    "))
        set_run_font(r, CODE_PT, name=CODE_FONT)


def convert(md_path, out_path, strict=False):
    text = Path(md_path).read_text(encoding="utf-8")
    meta, body_lines = parse_frontmatter(text.splitlines())
    blocks = parse_blocks(body_lines)
    classify_headings(blocks)
    mode = meta.get("numbering", "continuous")
    if mode not in ("continuous", "section"):
        raise SystemExit("numbering must be 'continuous' or 'section'")
    refs, warnings = assign_numbers(blocks, mode)

    doc = Document()
    configure_styles(doc)
    configure_section(doc.sections[0])
    clear_footer(doc.sections[0])

    has_title = meta.get("title_page", "true").lower() != "false"
    if has_title:
        build_title_page(doc, meta)
        body_section = doc.add_section(WD_SECTION.NEW_PAGE)
        configure_section(body_section)
    else:
        body_section = doc.sections[0]
    add_page_number_footer(body_section)

    has_author_toc = any(b["t"] == "heading" and b.get("toc") for b in blocks)
    toc_before = None
    if not has_author_toc:
        for idx, b in enumerate(blocks):
            if b["t"] == "heading" and not (b["kind"] == "structural" and b["text"] in ("РЕФЕРАТ", "АННОТАЦИЯ")):
                toc_before = idx
                break
    in_sources = False
    base_dir = Path(md_path).resolve().parent

    for idx, b in enumerate(blocks):
        t = b["t"]
        if idx == toc_before:
            h = doc.add_heading(level=1)
            h.alignment = WD_ALIGN_PARAGRAPH.CENTER
            h.paragraph_format.first_line_indent = Cm(0)
            set_run_font(h.add_run("СОДЕРЖАНИЕ"), BODY_PT, bold=True)
            add_toc(doc)
        if t == "heading":
            in_sources = bool(b.get("sources"))
            add_heading(doc, b)
            if b.get("toc"):
                add_toc(doc)
        elif t == "para":
            add_paragraph(doc, b["text"], refs, warnings)
        elif t == "list":
            add_list(doc, b, refs, warnings, in_sources)
        elif t == "image":
            add_image(doc, b, base_dir, refs, warnings)
        elif t == "table":
            add_table(doc, b, refs, warnings)
        elif t == "equation":
            add_equation(doc, b, refs, warnings)
        elif t == "code":
            add_code(doc, b)
        elif t == "pagebreak":
            doc.add_paragraph().add_run().add_break(WD_BREAK.PAGE)

    if not has_author_toc and toc_before is None:
        warnings.append("no headings found: table of contents not generated")

    enable_update_fields(doc)
    doc.core_properties.title = meta.get("title", "")
    doc.core_properties.author = meta.get("author", "")
    doc.save(out_path)
    for w in warnings:
        sys.stderr.write(f"warning: {w}\n")
    if strict and warnings:
        return 2
    return 0


def main():
    ap = argparse.ArgumentParser(description="Markdown to GOST 7.32 .docx for a graduation thesis")
    ap.add_argument("input")
    ap.add_argument("-o", "--output")
    ap.add_argument("--strict", action="store_true", help="exit 2 on any warning")
    args = ap.parse_args()
    src = Path(args.input)
    if not src.is_file():
        sys.stderr.write(f"not found: {src}\n")
        return 1
    out = Path(args.output) if args.output else src.with_suffix(".docx")
    code = convert(src, out, args.strict)
    sys.stdout.write(f"{out}\n")
    return code


if __name__ == "__main__":
    sys.exit(main())
