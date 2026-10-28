"""Unlimited-OCR-2 local OCR CLI.

Usage:
  python ocr.py image.png [image2.png ...]      # single or multi page
  python ocr.py --pdf doc.pdf                   # PDF -> pages -> one request
Results print to stdout; per-run artifacts go to out\\.
"""
import os, sys, glob, re, subprocess, tempfile, time

os.environ.setdefault("HF_HUB_OFFLINE", "1")
os.environ.setdefault("TRANSFORMERS_OFFLINE", "1")

import torch
from transformers import AutoModel, AutoTokenizer

MODEL = os.path.join(os.path.dirname(os.path.abspath(__file__)), "model")
PDFTOPPM = r"C:\Users\wubi\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\poppler\Library\bin\pdftoppm.exe"
# 分窗批识别（与 gaea ocrPageLoopBatch 的 ocrBatchWindow 同口径）：跨页上下文 +
# 单请求时长可控；crop_mode=True 多页也逐页 gundam 切片（base 模式对 300dpi 密
# 排页有失真风险，勿用；infer_multi 即 base 兼容入口）。
WINDOW = 5


def pdf_to_images(pdf_path, dpi=150):
    tmp = tempfile.mkdtemp(prefix="uocr2_pdf_")
    subprocess.run([PDFTOPPM, "-png", "-r", str(dpi), pdf_path,
                    os.path.join(tmp, "page")], check=True)
    return sorted(glob.glob(os.path.join(tmp, "page-*.png")))


def _content_len(raw):
    """剥掉 <PAGE>/<BLOCK>…<FORK> 标记后的正文长度（近空页守卫用）。"""
    s = re.sub(r"<BLOCK>[^<]*<FORK>", "", raw)
    return len(s.replace("<PAGE>", "").strip())


def infer_window(model, tok, window, out_dir):
    """一窗批识别；页数对不上或出现近空页（生成退化实测形态：跳页幻觉致
    PageCountGuard 硬切/仅页眉无正文〔~28 字符〕）返回 None，调用方逐页回退。"""
    text = model.infer(tok, image_file=window, output_path=out_dir,
                       save_results=True, crop_mode=True, eval_mode=True,
                       max_new_tokens=8000 * len(window),
                       no_repeat_ngram_size=35,
                       ngram_window=1024 if len(window) > 1 else 128)
    parts = text.split("<PAGE>")
    pages = parts[1:] if not parts[0].strip() else parts
    if len(pages) != len(window):
        return None
    if any(_content_len(p) < 48 for p in pages):
        return None
    return text


def main():
    args = sys.argv[1:]
    dpi = 150
    if args and args[0] == "--dpi":
        dpi = int(args[1]); args = args[2:]
    files = []
    for a in args:
        if a.lower().endswith(".pdf"):
            files += pdf_to_images(a, dpi)
        else:
            files.append(a)
    if not files:
        print(__doc__); sys.exit(2)

    tok = AutoTokenizer.from_pretrained(MODEL, trust_remote_code=True)
    model = AutoModel.from_pretrained(MODEL, trust_remote_code=True, use_safetensors=True,
                                      torch_dtype=torch.bfloat16).eval()
    if torch.cuda.is_available():
        model = model.cuda()
    out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
    t0 = time.time()
    texts = []
    for i in range(0, len(files), WINDOW):
        window = files[i:i + WINDOW]
        text = infer_window(model, tok, window, out_dir)
        if text is None:  # 批退化：整窗回退逐页（gundam 单页保真最高）
            page_texts = []
            for f in window:
                page_texts.append(model.infer(tok, image_file=f, output_path=out_dir,
                                              save_results=True, eval_mode=True,
                                              max_new_tokens=8000,
                                              no_repeat_ngram_size=35, ngram_window=128))
            text = "\n\n".join(page_texts)
        texts.append(text)
    print("\n\n---\n\n".join(texts))
    print(f"\n[{time.time()-t0:.1f}s, {len(files)} page(s)]", file=sys.stderr)


if __name__ == "__main__":
    main()
