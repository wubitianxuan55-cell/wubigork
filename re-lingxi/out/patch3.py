import re

# ---- 1) 契约 join：通用名禁止全局回退 ----
p = "extract_contracts.py"
s = open(p, encoding="utf-8").read()
s = s.replace('''GLOBAL = {}
for _, _, d in clusters:
    for k, v in d.items():
        if len(v) > len(GLOBAL.get(k, "")):
            GLOBAL[k] = v''',
'''GENERIC = {"action", "brief", "description", "path", "paths", "content", "query", "type", "name",
           "value", "id", "enabled", "timeout", "timeout_ms", "options", "header", "headers",
           "format", "message", "question", "questions", "code", "text", "reason", "status"}
GLOBAL, GLOBAL_N = {}, {}
for _, _, d in clusters:
    for k, v in d.items():
        GLOBAL_N[k] = GLOBAL_N.get(k, 0) + 1
        if len(v) > len(GLOBAL.get(k, "")):
            GLOBAL[k] = v
GLOBAL_UNIQUE = {k for k, n in GLOBAL_N.items() if n == 1}''')
s = s.replace('''        if not body:                     # 簇内没有 → 退回全局并标注
            body = GLOBAL.get(f"{sn}_arg") or GLOBAL.get(sn) or ""
            if body:
                src = "全局*"''',
'''        if not body:                     # 簇内没有 → 仅在「非通用名 + 全局唯一」时回退
            cand = f"{sn}_arg" if f"{sn}_arg" in GLOBAL else sn
            if cand in GLOBAL_UNIQUE and sn not in GENERIC:
                body = GLOBAL[cand]
                src = "全局唯一"
            elif cand in GLOBAL:
                src = "歧义"''')
open(p, "w", encoding="utf-8").write(s)

# ---- 2) JS 分组：补关键词 ----
q = "extract_all.py"
t = open(q, encoding="utf-8").read()
t = t.replace('''def klass(s):
    if re.search(r"Presentations|\\.Slides|Slide\\b|pptx", s):
        return "ppt"
    if re.search(r"Workbooks|Worksheets|\\.Range\\(|Sheets\\(", s):
        return "sheet"
    if re.search(r"PDF|ExportTo|Exports\\.", s, re.I):
        return "pdf"
    if re.search(r"Application\\.Documents|Paragraphs|Sections", s):
        return "doc"
    if re.search(r"OCR|Image|Picture", s):
        return "image"
    return "other"''',
'''def klass(s):
    if re.search(r"Presentations|Slides|SlideIndex|pptx|Slide\\b", s):
        return "ppt"
    if re.search(r"Workbooks|Worksheets|\\.Range\\(|Sheets\\(|Cells\\(|RangeBoundaries|UsedRange", s):
        return "sheet"
    if re.search(r"\\bPDF\\b|ExportTo|Exports\\.|pdfDoc|Pages\\.", s, re.I):
        return "pdf"
    if re.search(r"ActiveDocument|Application\\.Documents|Paragraphs|Sections|Bookmarks|"
                 r"ContentControls|Fields|InlineShapes|Footnotes|Endnotes", s):
        return "doc"
    if re.search(r"OCR|InlineShapes|Picture|Image", s):
        return "image"
    return "other"''')
open(q, "w", encoding="utf-8").write(t)
print("patched both")
