#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""shell-report.py — 母版体检：字体清单×系统字体∩、16:9 复核、版式/占位符结构。
模板复刻流水线第 2 步。体检不过（非 16:9）即判败——失败即停纪律。

用法：python shell-report.py 模板.pptx
"""

from __future__ import annotations

import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pptxkit import check_16_9, effective_fonts, emit, Fail, open_presentation, placeholder_role  # noqa: E402


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("template", help="模板 pptx")
    ns = ap.parse_args()
    try:
        prs = open_presentation(ns.template)
        check_16_9(prs)  # 失败即停

        fonts: set = set()
        ph_struct = {}
        for slide in prs.slides:
            for sh in slide.shapes:
                if getattr(sh, "has_text_frame", False):
                    for para in sh.text_frame.paragraphs:
                        for run in para.runs:
                            if run.font.name:
                                fonts.add(run.font.name)
                role = placeholder_role(sh)
                if role != "none":
                    ph_struct.setdefault(slide.slide_layout.name, set()).add(role)
        for layout in prs.slide_masters[0].slide_layouts:
            for sh in layout.shapes:
                if getattr(sh, "has_text_frame", False):
                    for para in sh.text_frame.paragraphs:
                        for run in para.runs:
                            if run.font.name:
                                fonts.add(run.font.name)

        fonts_eff = effective_fonts(fonts)
        layouts = []
        for layout in prs.slide_masters[0].slide_layouts:
            roles = sorted(ph_struct.get(layout.name, set()))
            ph_in_layout = sorted({placeholder_role(sh) for sh in layout.shapes
                                   if placeholder_role(sh) != "none"})
            layouts.append({"layout": layout.name,
                            "placeholders_in_layout": ph_in_layout,
                            "used_by_slides": roles})
        data = {
            "template": ns.template,
            "slide_size_ok_16_9": True,
            "observed_fonts": sorted(fonts),
            "font_allow": fonts_eff["allow"],
            "font_fallback": fonts_eff["fallback"],
            "font_missing_from_whitelist": fonts_eff["missing"],
            "layout_count": len(layouts),
            "layouts": layouts,
        }
        msg = (f"体检通过：16:9，{len(layouts)} 版式；字体 {sorted(fonts) or '（全继承主题）'}；"
               f"允许字体 {len(fonts_eff['allow'])} 个，缺白名单字体 {len(fonts_eff['missing'])}")
        return emit(True, data, msg)
    except Fail as e:
        return emit(False, None, str(e))
    except Exception as e:  # noqa: BLE001
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
