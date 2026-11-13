#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""make-skeleton.py — 从模板生成空骨架 deck：保母版/版式/主题，删内容页，
后续 apply-flat 按 spec.json 逐页从版式建页填充。模板复刻流水线第 3 步。

用法：python make-skeleton.py 模板.pptx -o <任务目录>/skeleton.pptx
"""

from __future__ import annotations

import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pptxkit import check_16_9, emit, Fail, need_pptx, open_presentation  # noqa: E402


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("template", help="模板 pptx")
    ap.add_argument("-o", "--output", required=True, help="骨架输出路径")
    ns = ap.parse_args()
    try:
        prs = open_presentation(ns.template)
        check_16_9(prs)
        Presentation = need_pptx()
        xml_slides = prs.slides._sldIdLst  # noqa: SLF001 — 官方无 API，删页惯用式
        for sld_id in list(xml_slides):
            xml_slides.remove(sld_id)
        os.makedirs(os.path.dirname(os.path.abspath(ns.output)), exist_ok=True)
        prs.save(ns.output)
        layouts = [layout.name for layout in prs.slide_masters[0].slide_layouts]
        data = {"path": ns.output, "slides": 0, "layouts_available": layouts}
        return emit(True, data, f"骨架就绪：0 页，可用版式 {layouts}")
    except Fail as e:
        return emit(False, None, str(e))
    except Exception as e:  # noqa: BLE001
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
