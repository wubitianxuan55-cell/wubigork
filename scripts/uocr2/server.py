"""Unlimited-OCR-2 常驻 OCR 服务（gaea docmd OCR seam 的 uocr2 后端）。

gaea（internal/docmd/ocr_uocr2.go）按需拉起：
    venv\\Scripts\\python.exe server.py --port 8138
模型加载完成后才开监听——端口能应答即服务就绪。

协议：
    GET  /health    → 200 "ok"
    POST /ocr       {"image": "<base64 png>", "max_tokens": 8000}
                 → 200 {"text": "<模型原始输出>"}
                 → 422 {"error": "..."} 输出被 max_tokens 截断（finish_reason=length，
                    fail-closed：不把残缺文本当完整结果，由调用方按单页失败处理）
    POST /ocr_batch {"images": ["<base64 png>"...], "max_tokens_per_page": 8000}
                 → 200 {"texts": ["<每页原始输出>"...]}（与 images 等长、同序；
                    一次请求多页共享跨页上下文，SR-SWA 卖点：跨页表格合并）
                 → 422 输出截断 或 <PAGE> 页数与输入对不上（fail-closed）
模型原始输出含 <PAGE>/<BLOCK>…<FORK> 结构标记，由 Go 侧 uocr2CleanText 清洗。
"""
import argparse
import base64
import json
import os
import sys
import threading
from io import BytesIO

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

os.environ.setdefault("HF_HUB_OFFLINE", "1")
os.environ.setdefault("TRANSFORMERS_OFFLINE", "1")

import torch  # noqa: E402
from PIL import Image  # noqa: E402
from transformers import AutoModel, AutoTokenizer  # noqa: E402

# 与官方单页评测口径一致：NoRepeatNGram 防复读（window 128）。
NO_REPEAT_NGRAM_SIZE = 35
NGRAM_WINDOW = 128
DEFAULT_MAX_TOKENS = 8000

_lock = threading.Lock()
_model = None
_tok = None


def load_model(model_dir):
    global _model, _tok
    _tok = AutoTokenizer.from_pretrained(model_dir, trust_remote_code=True)
    _model = AutoModel.from_pretrained(model_dir, trust_remote_code=True,
                                       use_safetensors=True,
                                       torch_dtype=torch.bfloat16).eval()
    if torch.cuda.is_available():
        _model = _model.cuda()
        print(f"[uocr2-server] cuda: {torch.cuda.get_device_name(0)}", flush=True)


def infer_image(png_bytes, max_tokens):
    img = Image.open(BytesIO(png_bytes)).convert("RGB")
    with _lock:  # 模型推理非线程安全：逐页串行
        res = _model.infer(_tok, image_file=img, save_results=False,
                           eval_mode=True, max_new_tokens=max_tokens,
                           no_repeat_ngram_size=NO_REPEAT_NGRAM_SIZE,
                           ngram_window=NGRAM_WINDOW, return_dict=True)
    return res


def infer_images(png_bytes_list, max_tokens_per_page):
    """多页一次请求（gundam 逐页切片，与官方多页评测口径一致：window 1024）。"""
    imgs = [Image.open(BytesIO(b)).convert("RGB") for b in png_bytes_list]
    with _lock:  # 模型推理非线程安全：逐批串行
        res = _model.infer(_tok, image_file=imgs, crop_mode=True, save_results=False,
                           eval_mode=True, max_new_tokens=max_tokens_per_page * len(imgs),
                           no_repeat_ngram_size=NO_REPEAT_NGRAM_SIZE,
                           ngram_window=1024, return_dict=True)
    return res


def split_pages(raw, n_expect):
    """按 <PAGE> 标记拆分多页输出；页数与输入对不上返回 None（fail-closed）。"""
    parts = raw.split("<PAGE>")
    if parts[0].strip():  # 首个 <PAGE> 之前有正文 = 结构异常
        return None
    pages = [p for p in parts[1:]]
    if len(pages) != n_expect:
        return None
    return pages


class Handler(BaseHTTPRequestHandler):
    def _send(self, code, obj):
        body = json.dumps(obj, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            self._send(200, {"status": "ok"})
            return
        self._send(404, {"error": "not found"})

    def do_POST(self):
        if self.path not in ("/ocr", "/ocr_batch"):
            self._send(404, {"error": "not found"})
            return
        try:
            n = int(self.headers.get("Content-Length", "0"))
            req = json.loads(self.rfile.read(n).decode("utf-8"))
            if self.path == "/ocr":
                pngs = [base64.b64decode(req["image"])]
                max_tokens = max(1, int(req.get("max_tokens", DEFAULT_MAX_TOKENS)))
            else:
                pngs = [base64.b64decode(x) for x in req["images"]]
                if not 1 <= len(pngs) <= 64:
                    raise ValueError(f"images count {len(pngs)} out of range")
                max_tokens = max(1, int(req.get("max_tokens_per_page", DEFAULT_MAX_TOKENS)))
        except Exception as e:
            self._send(400, {"error": f"bad request: {e}"})
            return
        try:
            if self.path == "/ocr":
                res = infer_image(pngs[0], max_tokens)
                if res.get("finish_reason") == "length":
                    self._send(422, {"error": "输出被 max_tokens 截断（finish_reason=length）"})
                    return
                self._send(200, {"text": res["text"]})
            else:
                res = infer_images(pngs, max_tokens)
                if res.get("finish_reason") == "length":
                    self._send(422, {"error": "输出被 max_tokens 截断（finish_reason=length）"})
                    return
                pages = split_pages(res["text"], len(pngs))
                if pages is None:
                    self._send(422, {"error": "多页输出 <PAGE> 数与输入对不上（fail-closed）"})
                    return
                self._send(200, {"texts": pages})
        except Exception as e:
            self._send(500, {"error": f"inference failed: {e}"})
            return

    def log_message(self, fmt, *args):
        print("[uocr2-server]", fmt % args, file=sys.stderr, flush=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=8138)
    ap.add_argument("--model", default=os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "model"))
    args = ap.parse_args()
    load_model(args.model)
    print(f"[uocr2-server] listening on {args.host}:{args.port}", flush=True)
    ThreadingHTTPServer((args.host, args.port), Handler).serve_forever()


if __name__ == "__main__":
    main()
