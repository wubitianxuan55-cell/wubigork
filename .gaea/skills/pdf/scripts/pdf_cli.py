#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""pdf_cli.py — PDF 17 子命令统一 CLI（gaea 自研，非上游移植）。

契约（灵犀蒸馏，docs/gaea-lingxi-re-distill-2026-11.md §6 不变量 3）：
  - stdout 恒为单个 JSON：{"success": bool, "data": ..., "message": str}
  - success:false 即失败；退出码非 0 即失败；两者一致。
  - 所有子命令除全局 --help 外不打印任何额外文本到 stdout。

依赖：pypdf（结构操作）+ pdfplumber（文本/表格）+ pypdfium2（渲染）。
缺哪个库会在 message 里如实报「缺 <name>，pip install <name>」而不是崩栈。

用法示例（agent 经 bash 调用）：
  python pdf_cli.py info 报告.pdf
  python pdf_cli.py extract-text 报告.pdf --pages 1-3
  python pdf_cli.py merge -o 合并.pdf a.pdf b.pdf c.pdf
  python pdf_cli.py split 拆分.pdf -o 前缀
  python pdf_cli.py render 报告.pdf -o 页.png --dpi 150 --pages 1-4
"""

from __future__ import annotations

import argparse
import json
import sys

# Windows 控制台默认 GBK：JSON 与中文路径一律走 UTF-8。
try:
    sys.stdout.reconfigure(encoding="utf-8")
    sys.stderr.reconfigure(encoding="utf-8")
except Exception:
    pass


def emit(success: bool, data=None, message: str = "", code: int = 0) -> int:
    """单出口：打印契约 JSON 并返回退出码。"""
    print(json.dumps(
        {"success": success, "data": data, "message": message},
        ensure_ascii=False, default=str,
    ))
    return 0 if success else (code or 1)


class Fail(Exception):
    """业务失败：携带给用户看的一句话。"""


def open_reader(path: str, password: str = ""):
    """PdfReader 包装：空密码不传（pypdf 6.x 对未加密文件传 password="" 抛错）。"""
    pypdf = need("pypdf")
    kw = {"password": password} if password else {}
    return pypdf.PdfReader(path, **kw)


def need(mod: str, pip: str | None = None):
    """按需 import，缺库给可执行提示。"""
    try:
        return __import__(mod)
    except ImportError:
        raise Fail(f"缺依赖 {mod}：pip install {pip or mod}")


# ── 页码选择器："1-3,5,8-" → 有序不重复页号列表（1 基）────────────

def parse_pages(spec: str, total: int) -> list[int]:
    pages: list[int] = []
    for part in spec.split(","):
        part = part.strip()
        if not part:
            continue
        if part.endswith("-"):
            lo = int(part[:-1]) if part[:-1] else 1
            rng = range(lo, total + 1)
        elif "-" in part:
            lo, hi = part.split("-", 1)
            rng = range(int(lo), int(hi) + 1)
        else:
            rng = [int(part)]
        for n in rng:
            if 1 <= n <= total and n not in pages:
                pages.append(n)
    if not pages:
        raise Fail(f"页码选择 {spec!r} 没有选中任何页（文档共 {total} 页）")
    return pages


def selected_indices(spec: str, total: int) -> list[int]:
    """转 0 基下标。"""
    return [n - 1 for n in parse_pages(spec, total)]


# ── 各子命令实现：入参 namespace，返回 (data, message)────────────

def cmd_info(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    meta = reader.metadata or {}
    pages = []
    for i, page in enumerate(reader.pages):
        box = page.mediabox
        pages.append({"page": i + 1, "width": round(float(box.width), 1),
                      "height": round(float(box.height), 1)})
    data = {
        "path": ns.input,
        "page_count": len(reader.pages),
        "encrypted": reader.is_encrypted,
        "metadata": {k: str(v) for k, v in dict(meta).items()},
        "pages": pages if ns.verbose else pages[:5],
    }
    return data, f"共 {len(reader.pages)} 页"


def cmd_page_count(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    return {"path": ns.input, "page_count": len(reader.pages)}, f"{len(reader.pages)} 页"


def cmd_extract_text(ns) -> tuple[dict, str]:
    pdfplumber = need("pdfplumber", "pdfplumber")
    texts = []
    with pdfplumber.open(ns.input, password=ns.password) as pdf:
        idx = selected_indices(ns.pages, len(pdf.pages)) if ns.pages else range(len(pdf.pages))
        for i in idx:
            page = pdf.pages[i]
            texts.append({"page": i + 1, "text": page.extract_text() or ""})
    chars = sum(len(t["text"]) for t in texts)
    return {"pages": texts, "total_chars": chars}, f"提取 {len(texts)} 页共 {chars} 字"


def cmd_extract_table(ns) -> tuple[dict, str]:
    pdfplumber = need("pdfplumber", "pdfplumber")
    tables = []
    with pdfplumber.open(ns.input, password=ns.password) as pdf:
        idx = selected_indices(ns.pages, len(pdf.pages)) if ns.pages else range(len(pdf.pages))
        for i in idx:
            for tb in pdf.pages[i].extract_tables():
                tables.append({"page": i + 1, "rows": tb})
    return {"tables": tables, "count": len(tables)}, f"提取到 {len(tables)} 张表"


def cmd_to_md(ns) -> tuple[dict, str]:
    pdfplumber = need("pdfplumber", "pdfplumber")
    parts = []
    with pdfplumber.open(ns.input, password=ns.password) as pdf:
        idx = selected_indices(ns.pages, len(pdf.pages)) if ns.pages else range(len(pdf.pages))
        for i in idx:
            text = (pdf.pages[i].extract_text() or "").strip()
            parts.append(f"## 第 {i + 1} 页\n\n{text}" if text else f"## 第 {i + 1} 页\n\n（无文本层，可能是扫描件——OCR 走 scripts/ocr_local.py）")
    md = "\n\n".join(parts) + "\n"
    if ns.output:
        with open(ns.output, "w", encoding="utf-8") as f:
            f.write(md)
        return {"path": ns.output, "chars": len(md)}, f"Markdown 已写 {ns.output}"
    return {"markdown": md, "chars": len(md)}, f"提取 {len(md)} 字 Markdown"


def cmd_merge(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    if len(ns.inputs) < 2:
        raise Fail("merge 至少要两个输入文件")
    writer = pypdf.PdfWriter()
    total = 0
    for path in ns.inputs:
        reader = open_reader(path, ns.password)
        total += len(reader.pages)
        for page in reader.pages:
            writer.add_page(page)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output, "inputs": ns.inputs, "pages": total}, f"{len(ns.inputs)} 个文件合并为 {total} 页"


def cmd_split(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    stem = ns.output or _stem(ns.input) + "_页"
    outs = []
    for i, page in enumerate(reader.pages):
        writer = pypdf.PdfWriter()
        writer.add_page(page)
        out = f"{stem}{i + 1:02d}.pdf"
        with open(out, "wb") as fh:
            writer.write(fh)
        outs.append(out)
    return {"files": outs}, f"拆为 {len(outs)} 个单页文件"


def cmd_delete_pages(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    total = len(reader.pages)
    kill = set(parse_pages(ns.pages, total))
    if len(kill) >= total:
        raise Fail(f"不能删除全部 {total} 页")
    writer = pypdf.PdfWriter()
    kept = 0
    for i, page in enumerate(reader.pages):
        if (i + 1) not in kill:
            writer.add_page(page)
            kept += 1
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output, "deleted": sorted(kill), "pages": kept}, f"删 {len(kill)} 页余 {kept} 页"


def cmd_rotate(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    if ns.degrees % 90 != 0:
        raise Fail("degrees 必须是 90 的倍数")
    reader = open_reader(ns.input, ns.password)
    idx = selected_indices(ns.pages, len(reader.pages)) if ns.pages else range(len(reader.pages))
    writer = pypdf.PdfWriter()
    rotated = 0
    for i, page in enumerate(reader.pages):
        if i in idx:
            page.rotate(ns.degrees)
            rotated += 1
        writer.add_page(page)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output, "rotated": rotated}, f"旋转 {rotated} 页 {ns.degrees}°"


def cmd_crop(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    idx = selected_indices(ns.pages, len(reader.pages)) if ns.pages else range(len(reader.pages))
    writer = pypdf.PdfWriter()
    for i, page in enumerate(reader.pages):
        if i in idx:
            box = page.mediabox
            page.cropbox = pypdf.generic.RectangleObject([
                float(box.left) + ns.left, float(box.bottom) + ns.bottom,
                float(box.right) - ns.right, float(box.top) - ns.top,
            ])
        writer.add_page(page)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output, "cropped": len(idx), "margins": [ns.left, ns.top, ns.right, ns.bottom]}, f"裁边 {len(idx)} 页"


def cmd_watermark(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    mark = open_reader(ns.watermark, ns.password).pages[0]
    writer = pypdf.PdfWriter()
    for page in reader.pages:
        page.merge_page(mark)
        writer.add_page(page)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output, "pages": len(reader.pages)}, f"水印已叠 {len(reader.pages)} 页"


def cmd_compress(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    writer = pypdf.PdfWriter()
    for page in reader.pages:
        writer.add_page(page)
        try:
            page.compress_content_streams()  # 逐页流压缩（CPU 换体积）
        except Exception:
            pass
    writer.compress_identical_objects(remove_duplicates=True, remove_orphans=True)
    with open(ns.output, "wb") as f:
        writer.write(f)
    before, after = _size(ns.input), _size(ns.output)
    pct = round((1 - after / before) * 100, 1) if before else 0
    return {"path": ns.output, "before": before, "after": after, "saved_pct": pct}, f"压缩 {pct}%（图片为主的文件收益有限，深度压缩需 qpdf/ghostscript）"


def cmd_encrypt(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    if not ns.password:
        raise Fail("缺少参数 --password")
    reader = open_reader(ns.input)  # 源文件是明文：读时不带新密码
    writer = pypdf.PdfWriter()
    for page in reader.pages:
        writer.add_page(page)
    writer.encrypt(user_password=ns.password, owner_password=ns.owner_password or ns.password)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output}, "已加密（AES-128）"


def cmd_decrypt(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    if not ns.password:
        raise Fail("缺少参数 --password")
    reader = open_reader(ns.input, ns.password)
    if not reader.decrypt(ns.password):
        raise Fail("密码不正确或解密失败")
    writer = pypdf.PdfWriter()
    for page in reader.pages:
        writer.add_page(page)
    with open(ns.output, "wb") as f:
        writer.write(f)
    return {"path": ns.output}, "已解密"


def cmd_render(ns) -> tuple[dict, str]:
    pypdfium2 = need("pypdfium2")
    pdf = pypdfium2.PdfDocument(ns.input, password=ns.password or "")
    try:
        idx = selected_indices(ns.pages, len(pdf)) if ns.pages else range(len(pdf))
        outs = []
        stem = _stem(ns.output or ns.input)
        for i in idx:
            page = pdf[i]
            scale = ns.dpi / 72.0
            bitmap = page.render(scale=scale)
            out = f"{stem}_p{i + 1:02d}.png"
            pil = bitmap.to_pil()
            if pil.mode in ("RGBA", "P"):
                pil = pil.convert("RGB")
            pil.save(out)
            outs.append(out)
        return {"files": outs, "dpi": ns.dpi}, f"渲染 {len(outs)} 页 PNG（{ns.dpi} dpi）"
    finally:
        pdf.close()


def cmd_extract_images(ns) -> tuple[dict, str]:
    pypdf = need("pypdf")
    reader = open_reader(ns.input, ns.password)
    idx = selected_indices(ns.pages, len(reader.pages)) if ns.pages else range(len(reader.pages))
    stem = ns.output or (_stem(ns.input) + "_图")
    outs = []
    for i in idx:
        for j, img in enumerate(reader.pages[i].images):
            out = f"{stem}_p{i + 1:02d}_{j + 1:02d}{img.ext and ('.' + img.ext.lstrip('.')) or '.png'}"
            with open(out, "wb") as fh:
                fh.write(img.data)
            outs.append(out)
    return {"files": outs}, f"抽出 {len(outs)} 张图"


def cmd_batch(ns) -> tuple[dict, str]:
    """batch <subcommand> file... — 同一子命令跑多个文件，逐个给出契约结果。"""
    results = []
    ok = 0
    for path in ns.inputs:
        try:
            data, message = run_subcommand(ns.subcommand, path, ns)
            results.append({"input": path, "success": True, "data": data, "message": message})
            ok += 1
        except Fail as e:
            results.append({"input": path, "success": False, "message": str(e)})
        except Exception as e:  # noqa: BLE001 — batch 逐文件容错，错误如实入账
            results.append({"input": path, "success": False, "message": f"{type(e).__name__}: {e}"})
    return {"results": results, "ok": ok, "total": len(ns.inputs)}, f"batch {ns.subcommand}：{ok}/{len(ns.inputs)} 成功"


# ── 参数解析与入口 ────────────────────────────────────────────

def _stem(path: str) -> str:
    base = path.replace("\\", "/").rsplit("/", 1)[-1]
    return base.rsplit(".", 1)[0] if "." in base else base


def _size(path: str) -> int:
    import os
    return os.path.getsize(path)


def _add_common(p: argparse.ArgumentParser, need_input: bool = True, with_password: bool = True) -> None:
    if need_input:
        p.add_argument("input", help="PDF 路径")
    if with_password:
        p.add_argument("--password", default="", help="打开密码（加密文件用）")


def build_parser() -> argparse.ArgumentParser:
    ap = argparse.ArgumentParser(prog="pdf_cli", description=__doc__.split("\n")[0])
    sub = ap.add_subparsers(dest="command", required=True)

    p = sub.add_parser("info", help="元数据+页数+页面尺寸"); _add_common(p)
    p.add_argument("--verbose", action="store_true", help="列出全部页面尺寸（缺省前 5 页）")

    p = sub.add_parser("page-count", help="页数"); _add_common(p)

    p = sub.add_parser("extract-text", help="提取文本（pdfplumber）"); _add_common(p)
    p.add_argument("--pages", default="", help="页选择，如 1-3,5")

    p = sub.add_parser("extract-table", help="提取表格（pdfplumber，rows 二维数组）"); _add_common(p)
    p.add_argument("--pages", default="", help="页选择")

    p = sub.add_parser("to-md", help="转 Markdown（页标题分节；扫描件提示走 OCR）"); _add_common(p)
    p.add_argument("--pages", default="", help="页选择")
    p.add_argument("-o", "--output", default="", help="写文件（缺省打到 data.markdown）")

    p = sub.add_parser("merge", help="合并多个 PDF")
    p.add_argument("inputs", nargs="+", help="输入文件（≥2）")
    p.add_argument("-o", "--output", required=True, help="输出文件")
    p.add_argument("--password", default="")

    p = sub.add_parser("split", help="逐页拆分"); _add_common(p)
    p.add_argument("-o", "--output", default="", help="输出前缀（缺省 <名>_页NN.pdf）")

    p = sub.add_parser("delete-pages", help="删页"); _add_common(p)
    p.add_argument("--pages", required=True, help="要删的页，如 2-4,7")
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("rotate", help="旋转（90 的倍数）"); _add_common(p)
    p.add_argument("--degrees", type=int, required=True)
    p.add_argument("--pages", default="", help="页选择（缺省全部）")
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("crop", help="四边裁边（pt）"); _add_common(p)
    p.add_argument("--left", type=float, default=0)
    p.add_argument("--top", type=float, default=0)
    p.add_argument("--right", type=float, default=0)
    p.add_argument("--bottom", type=float, default=0)
    p.add_argument("--pages", default="", help="页选择（缺省全部）")
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("watermark", help="叠加水印（水印 PDF 首页）"); _add_common(p)
    p.add_argument("-w", "--watermark", required=True, help="水印 PDF")
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("compress", help="无损压缩（流压缩+去重对象）"); _add_common(p)
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("encrypt", help="加密"); _add_common(p, with_password=False)
    p.add_argument("--password", required=True)
    p.add_argument("--owner-password", default="")
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("decrypt", help="解密"); _add_common(p, with_password=False)
    p.add_argument("--password", required=True)
    p.add_argument("-o", "--output", required=True)

    p = sub.add_parser("render", help="渲页为 PNG（pypdfium2）"); _add_common(p)
    p.add_argument("--pages", default="", help="页选择（缺省全部）")
    p.add_argument("--dpi", type=int, default=150)
    p.add_argument("-o", "--output", default="", help="输出前缀（缺省 <名>_pNN.png）")

    p = sub.add_parser("extract-images", help="抽内嵌图"); _add_common(p)
    p.add_argument("--pages", default="", help="页选择（缺省全部）")
    p.add_argument("-o", "--output", default="", help="输出前缀")

    p = sub.add_parser("batch", help="同一子命令跑多个文件（容错逐文件入账）")
    p.add_argument("subcommand", help="要批量执行的子命令名")
    p.add_argument("inputs", nargs="+", help="输入文件")
    p.add_argument("--password", default="", help="打开密码（透传给子命令）")

    return ap


def run_subcommand(command: str, path: str, ns):
    """以 path 替换 input 后执行子命令（batch 复用入口）。"""
    ns.input = path
    handler = globals().get(f"cmd_{command.replace('-', '_')}")
    if handler is None:
        raise Fail(f"batch 不支持的子命令 {command}（无 cmd 处理器）")
    return handler(ns)


def main(argv: list[str] | None = None) -> int:
    ns = build_parser().parse_args(argv)
    try:
        if ns.command == "batch":
            data, message = cmd_batch(ns)
        else:
            handler = globals().get(f"cmd_{ns.command.replace('-', '_')}")
            data, message = handler(ns)
        return emit(True, data, message)
    except Fail as e:
        return emit(False, None, str(e))
    except FileNotFoundError:
        return emit(False, None, f"文件不存在：{getattr(ns, 'input', getattr(ns, 'inputs', ['?'])[0])}")
    except Exception as e:  # noqa: BLE001 — 契约要求任何异常都收进 JSON
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
