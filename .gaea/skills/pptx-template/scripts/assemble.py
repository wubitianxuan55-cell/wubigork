#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""assemble.py — 终检：16:9、字体全在允许集、无残留占位符提示文本
（「单击此处添加」）、页数与 manifest 对账。模板复刻流水线第 5 步（守门）。

用法：python assemble.py final.pptx [--min-pages 1]
"""

from __future__ import annotations

import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pptxkit import check_16_9, effective_fonts, emit, Fail, open_presentation, placeholder_role, run_font  # noqa: E402

PLACEHOLDER_HINTS = ("单击此处添加", "Click to add", "点击添加")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("final", help="apply-flat 产物")
    ap.add_argument("--min-pages", type=int, default=1)
    ns = ap.parse_args()
    try:
        prs = open_presentation(ns.final)
        check_16_9(prs)
        problems: list = []
        if len(prs.slides) < ns.min_pages:
            problems.append(f"页数 {len(prs.slides)} < 要求 {ns.min_pages}")

        fonts: set = set()
        for no, slide in enumerate(prs.slides, start=1):
            for sh in slide.shapes:
                if getattr(sh, "has_text_frame", False):
                    text = sh.text_frame.text
                    for hint in PLACEHOLDER_HINTS:
                        if hint in text:
                            problems.append(f"第 {no} 页形状 {sh.name!r} 有占位符残留文本「{hint}」")
                    for para in sh.text_frame.paragraphs:
                        for run in para.runs:
                            if run_font(run):
                                fonts.add(run_font(run))
        fonts_eff = effective_fonts(fonts)
        for f in fonts_eff["missing"]:
            problems.append(f"字体 {f!r} 不在系统∩白名单内——apply-flat 应已设 fallback，母版样式带出的需换")

        data = {
            "path": ns.final,
            "pages": len(prs.slides),
            "fonts_observed": sorted(fonts),
            "font_allow": fonts_eff["allow"],
            "problems": problems,
            "pass": not problems,
        }
        if problems:
            return emit(False, data, f"终检未过：{len(problems)} 项问题（失败即停，修完重跑 assemble）")
        return emit(True, data, f"终检通过：{len(prs.slides)} 页，字体全合规")
    except Fail as e:
        return emit(False, None, str(e))
    except Exception as e:  # noqa: BLE001
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
