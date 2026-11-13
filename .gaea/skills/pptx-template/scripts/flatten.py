#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""flatten.py — 模板摊平：pptx → flat.json（每页版式/占位符/字体/颜色/尺寸）
+ assets/（内嵌图导出）。模板复刻流水线第 1 步（flatten → shell-report →
make-skeleton → apply-flat → assemble）。

用法：python flatten.py 模板.pptx -o <任务目录>/flat.json [--assets <任务目录>/assets]
"""

from __future__ import annotations

import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pptxkit import check_16_9, emit, Fail, open_presentation, placeholder_role, run_font, shape_texts  # noqa: E402


def flatten_shape(shape) -> dict:
    d = {
        "name": shape.name,
        "role": placeholder_role(shape),
        "ph_idx": shape.placeholder_format.idx if shape.is_placeholder else None,
        "left": shape.left, "top": shape.top,
        "width": shape.width, "height": shape.height,
    }
    if shape.shape_type is not None:
        d["shape_type"] = str(shape.shape_type)
    if getattr(shape, "has_text_frame", False):
        d["text"] = shape.text_frame.text
        runs = shape_texts(shape)
        fonts = sorted({run_font(r) for r in runs if run_font(r)})
        sizes = sorted({r.font.size.pt for r in runs if r.font.size is not None})
        d["fonts"] = fonts
        d["sizes_pt"] = sizes
    if getattr(shape, "has_table", False):
        d["table_rows"] = len(shape.table.rows)
        d["table_cols"] = len(shape.table.columns)
    if shape.shape_type == 13 or "Picture" in str(getattr(shape, "shape_type", "")):  # PICTURE
        try:
            d["image_ext"] = shape.image.ext
        except Exception:
            pass
    return d


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("template", help="模板 pptx")
    ap.add_argument("-o", "--output", required=True, help="flat.json 输出路径")
    ap.add_argument("--assets", default="", help="图片导出目录（缺省不导出）")
    ns = ap.parse_args()
    try:
        prs = open_presentation(ns.template)
        check_16_9(prs)
        slides = []
        for i, slide in enumerate(prs.slides):
            entry = {
                "slide": i + 1,
                "layout": slide.slide_layout.name,
                "shapes": [flatten_shape(sh) for sh in slide.shapes],
            }
            if slide.has_notes_slide and slide.notes_slide.notes_text_frame is not None:
                entry["notes"] = slide.notes_slide.notes_text_frame.text
            slides.append(entry)
        layouts = sorted({s["layout"] for s in slides})
        data = {
            "template": ns.template,
            "slide_size_emu": [prs.slide_width, prs.slide_height],
            "slide_count": len(slides),
            "layouts": layouts,
            "slides": slides,
        }
        exported = 0
        if ns.assets:
            os.makedirs(ns.assets, exist_ok=True)
            for s in slides:
                for sh in s["shapes"]:
                    if "image_ext" in sh:
                        src = None
                        for pic in prs.slides[s["slide"] - 1].shapes:
                            if pic.name == sh["name"] and pic.shape_type == 13:
                                src = pic
                                break
                        if src is None:
                            continue
                        out = os.path.join(ns.assets, f"p{s['slide']:02d}_{sh['name']}.{sh['image_ext']}")
                        with open(out, "wb") as f:
                            f.write(src.image.blob)
                        sh["asset"] = out
                        exported += 1
            data["assets_dir"] = ns.assets
            data["assets_exported"] = exported
        os.makedirs(os.path.dirname(os.path.abspath(ns.output)), exist_ok=True)
        with open(ns.output, "w", encoding="utf-8") as f:
            import json
            json.dump(data, f, ensure_ascii=False, indent=1)
        msg = f"摊平 {len(slides)} 页 {len(layouts)} 版式"
        if ns.assets:
            msg += f"，导出 {exported} 图"
        return emit(True, data, msg + f" → {ns.output}")
    except Fail as e:
        return emit(False, None, str(e))
    except Exception as e:  # noqa: BLE001
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
