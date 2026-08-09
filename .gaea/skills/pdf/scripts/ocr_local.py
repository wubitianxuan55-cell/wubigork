#!/usr/bin/env python3
"""Local OCR for scanned PDFs and images — leverages gaea's local models.

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
import time
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


_rapid_engine = None


def get_rapid_engine():
    """Lazy singleton RapidOCR engine (PP-OCR models via onnxruntime, offline)."""
    global _rapid_engine
    if _rapid_engine is None:
        from rapidocr_onnxruntime import RapidOCR

        _rapid_engine = RapidOCR()
    return _rapid_engine


def rapid_ocr(image_path: str) -> str:
    """RapidOCR 识别单张图片，返回逐行文本；未安装或失败返回空串。"""
    try:
        engine = get_rapid_engine()
        result, _ = engine(image_path)
        if not result:
            return ""
        lines = [str(item[1]).strip() for item in result if len(item) > 1 and str(item[1]).strip()]
        return "\n".join(lines)
    except Exception:
        return ""


_OVIS_PROMPT = (
    "<|im_start|>user\n"
    "Extract all readable content from the image in natural human reading order "
    "and output the result as a single Markdown document. Preserve the original "
    "text without translation.<|im_end|>\n<|im_start|>assistant\n"
)


def parse_ovis_output(raw: str) -> str:
    """从 llama-mtmd-cli 输出中提取 assistant 回答，去掉 think 块与模板尾巴。"""
    marker = "<|im_start|>assistant"
    idx = raw.rfind(marker)
    if idx >= 0:
        raw = raw[idx + len(marker):]
    t0 = raw.find("<think>")
    t1 = raw.find("</think>")
    if t0 >= 0 and t1 > t0:
        raw = raw[:t0] + raw[t1 + len("</think>"):]
    for stop in ("<|im_end|>", "You are a helpful assistant"):
        s = raw.find(stop)
        if s >= 0:
            raw = raw[:s]
    return raw.strip()


_OVIS_URL = os.environ.get("GAEA_OCR_URL", "http://127.0.0.1:8137")


def ovis_server_healthy() -> bool:
    try:
        import urllib.request
        with urllib.request.urlopen(_OVIS_URL + "/health", timeout=3) as resp:
            out = json.loads(resp.read())
        return bool(out.get("status") == "ok")
    except Exception:
        return False


def start_ovis_server() -> bool:
    """按需拉起常驻 llama-server（OvisOCR2，Vulkan），等待就绪（≤60s）。"""
    base = os.environ.get("GAEA_OCR_DIR", r"C:\AI\gaea-ocr")
    exe = os.environ.get("GAEA_OCR_LLAMA", os.path.join(base, "llama", "llama-server.exe"))
    model = os.environ.get("GAEA_OCR_MODEL", os.path.join(base, "models", "OvisOCR2-Q5_K_M.gguf"))
    mmproj = os.environ.get("GAEA_OCR_MMPROJ", os.path.join(base, "models", "mmproj-F16.gguf"))
    if not (os.path.isfile(exe) and os.path.isfile(model) and os.path.isfile(mmproj)):
        return False
    port = os.environ.get("GAEA_OCR_PORT", "8137")
    try:
        flags = getattr(subprocess, "CREATE_NO_WINDOW", 0) if os.name == "nt" else 0
        subprocess.Popen(
            [exe, "-m", model, "--mmproj", mmproj, "--port", port,
             "-c", "8192", "-ngl", "99", "--jinja", "--host", "127.0.0.1"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=flags,
        )
        for _ in range(60):
            time.sleep(1)
            if ovis_server_healthy():
                return True
    except Exception:
        pass
    return False


def ovis_server_ocr(image_path: str) -> str:
    """把图片发给常驻 OvisOCR2 服务（OpenAI 兼容接口），返回 Markdown 文本。"""
    try:
        import urllib.request
        with open(image_path, "rb") as f:
            b64 = base64.b64encode(f.read()).decode()
        payload = {
            "model": "ovis",
            "messages": [{
                "role": "user",
                "content": [
                    {"type": "text", "text": "Extract all readable content from the image in natural human reading order and output the result as a single Markdown document. Preserve the original text without translation."},
                    {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{b64}"}},
                ],
            }],
            "temperature": 0.0,
            "max_tokens": 1024,
        }
        req = urllib.request.Request(
            _OVIS_URL + "/v1/chat/completions",
            data=json.dumps(payload).encode(),
            headers={"Content-Type": "application/json"},
        )
        with urllib.request.urlopen(req, timeout=180) as resp:
            out = json.loads(resp.read())
        return (out.get("choices") or [{}])[0].get("message", {}).get("content", "").strip()
    except Exception:
        return ""


def ovis_cli_ocr(image_path: str) -> str:
    """OvisOCR2 CLI 兜底（服务不可用时逐页调用，模型每次加载较慢）。"""
    base = os.environ.get("GAEA_OCR_DIR", r"C:\AI\gaea-ocr")
    cli = os.environ.get("GAEA_OCR_LLAMA", os.path.join(base, "llama", "llama-mtmd-cli.exe"))
    model = os.environ.get("GAEA_OCR_MODEL", os.path.join(base, "models", "OvisOCR2-Q5_K_M.gguf"))
    mmproj = os.environ.get("GAEA_OCR_MMPROJ", os.path.join(base, "models", "mmproj-F16.gguf"))
    if not (os.path.isfile(cli) and os.path.isfile(model) and os.path.isfile(mmproj)):
        return ""
    try:
        r = subprocess.run(
            [cli, "-m", model, "--mmproj", mmproj, "--image", image_path,
             "-p", _OVIS_PROMPT, "-n", "1024", "--temp", "0.0"],
            capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=180,
        )
        return parse_ovis_output(r.stdout)
    except Exception:
        return ""


def ovis_ocr(image_path: str) -> str:
    """OvisOCR2 文档解析：优先常驻服务，缺失时按需拉起，再退回 CLI。"""
    if ovis_server_healthy() or start_ovis_server():
        text = ovis_server_ocr(image_path)
        if text:
            return text
    return ovis_cli_ocr(image_path)


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
    if mode in ("auto", "ovis"):
        text = ovis_ocr(image_path)
        if text:
            tool = "ovis-ocr2"
    if mode in ("auto", "rapid"):
        if not text or len(text) < 8:
            text = rapid_ocr(image_path)
        if text and tool == "":
            tool = "rapid-ocr"
    if mode in ("auto", "winrt"):
        if not text or len(text) < 8:
            text = winrt_ocr(image_path)
        if text and tool == "":
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
    ap.add_argument("--mode", choices=["auto", "ovis", "rapid", "winrt", "local"], default="auto")
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
