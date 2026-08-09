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

 **首选本地 OCR（离线、零外部依赖，发挥 gaea 本地模型优势）**：

```bash
# 扫描件 PDF / 图片 → 本地 OCR
# 自动链路：OvisOCR2 文档解析（推荐，Markdown/表格/公式）→ RapidOCR → Windows 原生 OCR → 本地视觉模型
python scripts/ocr_local.py 扫描件.pdf --mode auto --out 识别结果.txt

# 强制只用 OvisOCR2（0.8B 端到端文档解析，GGUF + Vulkan，已装于 C:\AI\gaea-ocr）
python scripts/ocr_local.py 扫描件.pdf --mode ovis --out 识别结果.txt

# 强制只用 RapidOCR（PP-OCR 转 ONNX，需 venv：C:\AI\gaea-ocr-env\Scripts\python.exe）
C:\AI\gaea-ocr-env\Scripts\python.exe scripts/ocr_local.py 扫描件.pdf --mode rapid --out 识别结果.txt

# 强制只用本地视觉模型（版式/表格理解更强，需要 herdsman 视觉服务在 127.0.0.1:8080 运行）
python scripts/ocr_local.py 扫描件.pdf --mode local --out 识别结果.txt
```

链路：PyMuPDF 渲染 PDF 页（默认 300 DPI）→ **OvisOCR2**（llama.cpp Vulkan，
`C:\AI\gaea-ocr`，路径可用 `GAEA_OCR_DIR` / `GAEA_OCR_LLAMA` / `GAEA_OCR_MODEL` / `GAEA_OCR_MMPROJ`
覆盖）→ RapidOCR（PP-OCR，onnxruntime）→ WinRT OcrEngine（离线、zh-Hans）→
本地视觉模型（herdsman，`GAEA_VISION_BASE_URL` / `GAEA_VISION_MODEL` 可覆盖）。
文本过短/为空时逐级降级。输出 UTF-8 文本或 JSON（每页 + 工具来源）。

可选外部方案（仅当本地通道都不可用时）：
```python
# Requires: pip install pytesseract pdf2image + tesseract.exe
import pytesseract
from pdf2image import convert_from_path
images = convert_from_path('scanned.pdf')
for i, image in enumerate(images):
    print(f"Page {i+1}:\n{pytesseract.image_to_string(image)}")
```

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
| OCR scanned PDFs | pytesseract | Convert to image first |
| Fill PDF forms | pdf-lib or pypdf (see FORMS.md) | See FORMS.md |

## Next Steps

- For advanced pypdfium2 usage, see REFERENCE.md
- For JavaScript libraries (pdf-lib), see REFERENCE.md
- If you need to fill out a PDF form, follow the instructions in FORMS.md
- For troubleshooting guides, see REFERENCE.md
