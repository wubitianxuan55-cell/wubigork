---
name: pdf
description: Use this skill whenever the user wants to do anything with PDF files. This includes reading or extracting text/tables from PDFs, combining or merging multiple PDFs into one, splitting PDFs apart, rotating pages, adding watermarks, creating new PDFs, filling PDF forms, encrypting/decrypting PDFs, extracting images, and OCR on scanned PDFs to make them searchable. If the user mentions a .pdf file or asks to produce one, use this skill.
license: Proprietary. LICENSE.txt has complete terms
---

# PDF Processing Guide

## Overview

This guide covers essential PDF processing operations using Python libraries and command-line tools. For advanced features, JavaScript libraries, and detailed examples, see REFERENCE.md. If you need to fill out a PDF form, read FORMS.md and follow its instructions.

## Quick Start

```python
from pypdf import PdfReader, PdfWriter

# Read a PDF
reader = PdfReader("document.pdf")
print(f"Pages: {len(reader.pages)}")

# Extract text
text = ""
for page in reader.pages:
    text += page.extract_text()
```

## Python Libraries

### pypdf - Basic Operations

#### Merge PDFs
```python
from pypdf import PdfWriter, PdfReader

writer = PdfWriter()
for pdf_file in ["doc1.pdf", "doc2.pdf", "doc3.pdf"]:
    reader = PdfReader(pdf_file)
    for page in reader.pages:
        writer.add_page(page)

with open("merged.pdf", "wb") as output:
    writer.write(output)
```

#### Split PDF
```python
reader = PdfReader("input.pdf")
for i, page in enumerate(reader.pages):
    writer = PdfWriter()
    writer.add_page(page)
    with open(f"page_{i+1}.pdf", "wb") as output:
        writer.write(output)
```

#### Extract Metadata
```python
reader = PdfReader("document.pdf")
meta = reader.metadata
print(f"Title: {meta.title}")
print(f"Author: {meta.author}")
print(f"Subject: {meta.subject}")
print(f"Creator: {meta.creator}")
```

#### Rotate Pages
```python
reader = PdfReader("input.pdf")
writer = PdfWriter()

page = reader.pages[0]
page.rotate(90)  # Rotate 90 degrees clockwise
writer.add_page(page)

with open("rotated.pdf", "wb") as output:
    writer.write(output)
```

### pdfplumber - Text and Table Extraction

#### Extract Text with Layout
```python
import pdfplumber

with pdfplumber.open("document.pdf") as pdf:
    for page in pdf.pages:
        text = page.extract_text()
        print(text)
```

#### Extract Tables
```python
with pdfplumber.open("document.pdf") as pdf:
    for i, page in enumerate(pdf.pages):
        tables = page.extract_tables()
        for j, table in enumerate(tables):
            print(f"Table {j+1} on page {i+1}:")
            for row in table:
                print(row)
```

#### Advanced Table Extraction
```python
import pandas as pd

with pdfplumber.open("document.pdf") as pdf:
    all_tables = []
    for page in pdf.pages:
        tables = page.extract_tables()
        for table in tables:
            if table:  # Check if table is not empty
                df = pd.DataFrame(table[1:], columns=table[0])
                all_tables.append(df)

# Combine all tables
if all_tables:
    combined_df = pd.concat(all_tables, ignore_index=True)
    combined_df.to_excel("extracted_tables.xlsx", index=False)
```

### reportlab - Create PDFs

#### Basic PDF Creation
```python
from reportlab.lib.pagesizes import letter
from reportlab.pdfgen import canvas

c = canvas.Canvas("hello.pdf", pagesize=letter)
width, height = letter

# Add text
c.drawString(100, height - 100, "Hello World!")
c.drawString(100, height - 120, "This is a PDF created with reportlab")

# Add a line
c.line(100, height - 140, 400, height - 140)

# Save
c.save()
```

#### Create PDF with Multiple Pages
```python
from reportlab.lib.pagesizes import letter
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
from reportlab.lib.styles import getSampleStyleSheet

doc = SimpleDocTemplate("report.pdf", pagesize=letter)
styles = getSampleStyleSheet()
story = []

# Add content
title = Paragraph("Report Title", styles['Title'])
story.append(title)
story.append(Spacer(1, 12))

body = Paragraph("This is the body of the report. " * 20, styles['Normal'])
story.append(body)
story.append(PageBreak())

# Page 2
story.append(Paragraph("Page 2", styles['Heading1']))
story.append(Paragraph("Content for page 2", styles['Normal']))

# Build PDF
doc.build(story)
```

#### Subscripts and Superscripts

**IMPORTANT**: Never use Unicode subscript/superscript characters (₀₁₂₃₄₅₆₇₈₉, ⁰¹²³⁴⁵⁶⁷⁸⁹) in ReportLab PDFs. The built-in fonts do not include these glyphs, causing them to render as solid black boxes.

Instead, use ReportLab's XML markup tags in Paragraph objects:
```python
from reportlab.platypus import Paragraph
from reportlab.lib.styles import getSampleStyleSheet

styles = getSampleStyleSheet()

# Subscripts: use <sub> tag
chemical = Paragraph("H<sub>2</sub>O", styles['Normal'])

# Superscripts: use <super> tag
squared = Paragraph("x<super>2</super> + y<super>2</super>", styles['Normal'])
```

For canvas-drawn text (not Paragraph objects), manually adjust font the size and position rather than using Unicode subscripts/superscripts.

## Command-Line Tools

### pdftotext (poppler-utils)
```bash
# Extract text
pdftotext input.pdf output.txt

# Extract text preserving layout
pdftotext -layout input.pdf output.txt

# Extract specific pages
pdftotext -f 1 -l 5 input.pdf output.txt  # Pages 1-5
```

### qpdf
```bash
# Merge PDFs
qpdf --empty --pages file1.pdf file2.pdf -- merged.pdf

# Split pages
qpdf input.pdf --pages . 1-5 -- pages1-5.pdf
qpdf input.pdf --pages . 6-10 -- pages6-10.pdf

# Rotate pages
qpdf input.pdf output.pdf --rotate=+90:1  # Rotate page 1 by 90 degrees

# Remove password
qpdf --password=mypassword --decrypt encrypted.pdf decrypted.pdf
```

### pdftk (if available)
```bash
# Merge
pdftk file1.pdf file2.pdf cat output merged.pdf

# Split
pdftk input.pdf burst

# Rotate
pdftk input.pdf rotate 1east output rotated.pdf
```

## Common Tasks

### Extract Text from Scanned PDFs

 **首选 WPS（2026-11-26 拍板：转换解析优先用 WPS）**：直接用 `format_convert` 工具转整份
PDF——文字型出真表格，扫描件自动走 WPS 在线 OCR（174 页实测约 1 分钟），照片版 PDF 也能
出真表格。WPS 不可用时工具自动退回本地提取 / 本地 OCR（一次最多 30 页，超限用 `pages` 分段）。

```bash
# 扫描件 / 照片版 PDF → Markdown（含表格）；要图片落盘就加 output 与 preserve_images=true
format_convert path=扫描件.pdf output=扫描件.md preserve_images=true
```

 **要 Excel 真表格（清单 / 定额表 / 报价单取数）**：`to=xlsx` 走 WPS 的转表引擎——金额是真数值，
定额编码带前导零存成文本。扫描件 / 照片走 WPS 在线识别，默认开 `ai_fix`（数字零丢改，约 14 秒/页，
一次最多 20 页，超出用 `pages` 分一段连续页）；回执列出的疑似缺格行须对照原件核对。

```bash
format_convert path=工程量清单.pdf to=xlsx output=工程量清单.xlsx
format_convert path=报价单照片/ to=xlsx output=报价单.xlsx   # 目录：多张照片合成一个工作簿
```

 **本地 OCR（仅 WPS 不可用时的兜底；离线）**：gaea 内建的本地 OCR 只保留 Unlimited-OCR-2
（2026-11-26 精简；`format_convert` 的本地兜底会自动用它）。技能自带脚本只剩两条不依赖
引擎工件的通道——Windows 原生 OCR 与本地视觉模型：

```bash
# 自动：Windows 原生 OCR 先行，文本过短/为空的页回退本地视觉模型
python scripts/ocr_local.py 扫描件.pdf --mode auto --out 识别结果.txt
```

链路：PyMuPDF 渲染 PDF 页（默认 300 DPI）→ WinRT OcrEngine（离线、zh-Hans，
`scripts/windows-ocr.ps1`）→ 本地视觉模型（herdsman，`GAEA_VISION_BASE_URL` /
`GAEA_VISION_MODEL` 可覆盖）。输出 UTF-8 文本或 JSON（每页 + 工具来源）。

> 本机未装 tesseract；OvisOCR2 / RapidOCR 两腿已于 2026-11-26 退役，不要再按旧文档去找。
> 扫描件请优先走上面的 WPS 路径。

### Add Watermark
```python
from pypdf import PdfReader, PdfWriter

# Create watermark (or load existing)
watermark = PdfReader("watermark.pdf").pages[0]

# Apply to all pages
reader = PdfReader("document.pdf")
writer = PdfWriter()

for page in reader.pages:
    page.merge_page(watermark)
    writer.add_page(page)

with open("watermarked.pdf", "wb") as output:
    writer.write(output)
```

### Extract Images
```bash
# Using pdfimages (poppler-utils)
pdfimages -j input.pdf output_prefix

# This extracts all images as output_prefix-000.jpg, output_prefix-001.jpg, etc.
```

### Password Protection
```python
from pypdf import PdfReader, PdfWriter

reader = PdfReader("input.pdf")
writer = PdfWriter()

for page in reader.pages:
    writer.add_page(page)

# Add password
writer.encrypt("userpassword", "ownerpassword")

with open("encrypted.pdf", "wb") as output:
    writer.write(output)
```

### PDF → Word（可编辑 .docx）

用 LibreOffice 转换（Windows 上 soffice.exe 常见于 `C:\Program Files\LibreOffice\program\soffice.exe`，
不在 PATH 时用完整路径）：

```bash
python - <<'PY'
import os, subprocess, glob
soffice = r"C:\Program Files\LibreOffice\program\soffice.exe"
if not os.path.isfile(soffice):
    soffice = "soffice"  # 已加入 PATH 时
for pdf in glob.glob("*.pdf"):
    subprocess.run([soffice, "--headless", "--convert-to", "docx", pdf], check=True)
PY
```

转换后必须验证：用 python-docx 打开 .docx 抽查标题与段落是否完整；扫描件（无文本层）转换结果为空时，
先 OCR（见"Extract Text from Scanned PDFs"）再转换。

### 表格提取 → Excel（闭环）

```python
import pdfplumber, pandas as pd

with pdfplumber.open("报表.pdf") as pdf:
    tables = [t for page in pdf.pages for t in page.extract_tables() if t]
df = pd.concat([pd.DataFrame(t[1:], columns=t[0]) for t in tables], ignore_index=True)
df.to_excel("报表.xlsx", index=False)
```

表格提取后必须人工核对列对齐与合并单元格（pdfplumber 对复杂表格可能拆错列）。

## Quick Reference

| Task | Best Tool | Command/Code |
|------|-----------|--------------|
| Merge PDFs | pypdf | `writer.add_page(page)` |
| Split PDFs | pypdf | One page per file |
| Extract text | pdfplumber | `page.extract_text()` |
| Extract tables | pdfplumber | `page.extract_tables()` |
| PDF → Word | LibreOffice | `soffice --headless --convert-to docx` |
| 表格 → Excel | pdfplumber + pandas | `page.extract_tables()` → `to_excel()` |
| Create PDFs | reportlab | Canvas or Platypus |
| Command line merge | qpdf | `qpdf --empty --pages ...` |
| OCR scanned PDFs | `format_convert`（WPS 在线 OCR，首选）；兜底 `python scripts/ocr_local.py --mode auto` | 见「Extract Text from Scanned PDFs」 |
| Fill PDF forms | pdf-lib or pypdf (see FORMS.md) | See FORMS.md |

## Next Steps

- For advanced pypdfium2 usage, see REFERENCE.md
- For JavaScript libraries (pdf-lib), see REFERENCE.md
- If you need to fill out a PDF form, follow the instructions in FORMS.md
- For troubleshooting guides, see REFERENCE.md

## pdf_cli.py 统一命令面（gaea 自研，优先用这个）

17 个子命令一条命令完成，stdout 恒单 JSON `{success, data, message}`（`success:false` 即失败，退出码非 0 即失败）。多数操作能用它就**不要手写 python**——只有它没覆盖的组合才落回上面的代码片段。

```bash
S=scripts/pdf_cli.py   # 相对技能根（run_skill 后 SKILL_PATH 已就位）
python $S info 文件.pdf                          # 元数据+页数+页面尺寸
python $S page-count 文件.pdf
python $S extract-text 文件.pdf --pages 1-3      # pdfplumber 文本
python $S extract-table 文件.pdf --pages 2       # 表格→rows 二维数组
python $S to-md 文件.pdf -o 文件.md              # 扫描件会提示走 OCR
python $S merge -o 合并.pdf a.pdf b.pdf          # ≥2 个输入
python $S split 文件.pdf -o 前缀                 # 逐页拆 前缀NN.pdf
python $S delete-pages 文件.pdf --pages 2-4 -o out.pdf
python $S rotate 文件.pdf --degrees 90 -o out.pdf [--pages 1-2]
python $S crop 文件.pdf --left 20 --top 0 --right 20 --bottom 20 -o out.pdf
python $S watermark 文件.pdf -w 水印.pdf -o out.pdf
python $S compress 文件.pdf -o out.pdf           # 无损；深度压缩需 qpdf/gs
python $S encrypt 文件.pdf --password 密码 -o out.pdf
python $S decrypt 文件.pdf --password 密码 -o out.pdf
python $S render 文件.pdf --dpi 150 --pages 1-4 -o 前缀   # → 前缀_pNN.png
python $S extract-images 文件.pdf -o 前缀
python $S batch page-count a.pdf b.pdf c.pdf     # 同命令多文件，逐文件容错入账
```

- 页选择统一语法 `1-3,5,8-`（开区间到末页）。
- 加密文件加 `--password`；`encrypt` 的 `--password` 是**新**密码（源文件可为明文）。
- 依赖：pypdf + pdfplumber + pypdfium2；缺库时 message 会给出 `pip install <name>`。
- 交付纪律与表格门禁（公式优先/零错误）见 xlsx 技能；本文只管 PDF 本体。
