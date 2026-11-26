#!/usr/bin/env python3
"""Local OCR for scanned PDFs and images — leverages gaea's local models.

This is the fallback of the fallback: since 2026-11-26 scanned PDFs go through WPS
first (`format_convert` → WPS online OCR, ~1 min for 174 pages); gaea's own local OCR
(Unlimited-OCR-2) is the fallback inside gaea. The OvisOCR2 and RapidOCR legs that used
to live here were retired on 2026-11-26 (OvisOCR2 superseded, RapidOCR venv not installed).

Two local channels (no network, no cloud, no external OCR binaries):
  1. Windows 原生 OCR (WinRT OcrEngine, offline, zh-Hans) — scripts/windows-ocr.ps1
  2. 本地视觉模型 (herdsman, OpenAI-compatible /v1/chat/completions, GAEA_VISION_*)

Usage:
    python ocr_local.py input.pdf [--dpi 300] [--mode auto|winrt|local] [--out out.txt] [--json]
    python ocr_local.py input.png [--mode auto|winrt|local] [--out out.txt] [--json]

Mode:
    auto   : Windows OCR first; empty/too-short pages fall back to local VLM
    winrt  : Windows OCR only (fastest, offline)
    local  : local VLM only (better layout/table understanding)
"""
import argparse
import base64
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path


def find_windows_ocr() -> str:
    here = Path(__file__).resolve().parent
    candidates = [
        here / "windows-ocr.ps1",
        Path(os.environ.get("USERPROFILE", "")) / ".codex" / "skills" / "ds-vision-skill" / "scripts" / "windows-ocr.ps1",
    ]
    for c in candidates:
        if c.is_file():
            return str(c)
    return ""


def winrt_ocr(image_path: str) -> str:
    script = find_windows_ocr()
    if not script:
        return ""
    try:
        r = subprocess.run(
            ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-ImagePath", image_path, "-Json"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=60,
        )
        data = json.loads(r.stdout)
        return str(data.get("result", "") or data.get("text", "")).strip()
    except Exception:
        return ""


def local_vlm_ocr(image_path: str, prompt: str) -> str:
    base_url = os.environ.get("GAEA_VISION_BASE_URL", "http://127.0.0.1:8080/v1").rstrip("/")
    model = os.environ.get("GAEA_VISION_MODEL", "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2")
    try:
        import urllib.request
        with open(image_path, "rb") as f:
            b64 = base64.b64encode(f.read()).decode()
        payload = {
            "model": model,
            "messages": [{
                "role": "user",
                "content": [
                    {"type": "text", "text": prompt},
                    {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{b64}"}},
                ],
            }],
            "max_tokens": 2048,
            "temperature": 0.1,
        }
        req = urllib.request.Request(
            base_url + "/chat/completions",
            data=json.dumps(payload).encode(),
            headers={"Content-Type": "application/json"},
        )
        with urllib.request.urlopen(req, timeout=120) as resp:
            out = json.loads(resp.read())
        return (out.get("choices") or [{}])[0].get("message", {}).get("content", "").strip()
    except Exception:
        return ""


def render_pdf_pages(pdf_path: str, dpi: int, out_dir: str):
    import fitz  # PyMuPDF
    doc = fitz.open(pdf_path)
    paths = []
    for i, page in enumerate(doc):
        pix = page.get_pixmap(dpi=dpi)
        p = os.path.join(out_dir, f"page_{i + 1:04d}.png")
        pix.save(p)
        paths.append(p)
    return paths


def ocr_image(image_path: str, mode: str, prompt: str) -> dict:
    text = ""
    tool = ""
    if mode in ("auto", "winrt"):
        text = winrt_ocr(image_path)
        if text:
            tool = "windows-ocr"
    if mode in ("auto", "local") and (not text or len(text) < 8):
        vlm = local_vlm_ocr(image_path, prompt)
        if vlm:
            text, tool = vlm, "local-vlm"
    return {"tool_used": tool, "text": text}


def main():
    ap = argparse.ArgumentParser(description="Local OCR for scanned PDFs/images")
    ap.add_argument("input", help="PDF or image path")
    ap.add_argument("--dpi", type=int, default=300)
    ap.add_argument("--mode", choices=["auto", "winrt", "local"], default="auto")
    ap.add_argument("--out", default="")
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()

    input_path = os.path.abspath(args.input)
    ext = Path(input_path).suffix.lower()
    prompt = "请完整提取这张图片中的所有文字，保留原有段落和表格结构。若是表格，用 Markdown 表格输出。"

    pages = []
    tmp = None
    if ext == ".pdf":
        tmp = tempfile.mkdtemp(prefix="gaea_ocr_")
        images = render_pdf_pages(input_path, args.dpi, tmp)
        for i, img in enumerate(images, 1):
            res = ocr_image(img, args.mode, prompt)
            pages.append({"page": i, "tool_used": res["tool_used"], "text": res["text"]})
    else:
        res = ocr_image(input_path, args.mode, prompt)
        pages.append({"page": 1, "tool_used": res["tool_used"], "text": res["text"]})

    full = "\n\n".join(f"## 第 {p['page']} 页\n{p['text']}" for p in pages if p["text"])
    result = {"pages": pages, "text": full}
    if args.out:
        Path(args.out).write_text(full, encoding="utf-8")
    print(json.dumps(result, ensure_ascii=False))
    if tmp:
        import shutil
        shutil.rmtree(tmp, ignore_errors=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
