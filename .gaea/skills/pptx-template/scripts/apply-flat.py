#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""apply-flat.py — 把结构化文案按骨架版式逐页填充（含 hybrid-manifest 职责：
页→版式→占位符映射在填充时即时解析并记录）。模板复刻流水线第 4 步。

spec.json 结构（UTF-8）：
{
  "font": "微软雅黑",              // 可省——省则用 shell-report 的 fallback
  "slides": [
    {"layout": "标题幻灯片", "title": "季度经营汇报", "subtitle": "2026 Q4",
     "notes": "开场 30 秒"},
    {"layout": "标题和内容", "title": "总体业绩", "bullets": ["要点一", "要点二"]}
  ]
}

字段约定：title/subtitle 按占位符角色匹配；bullets/body 都进 body 占位符
（一段一个要点）；页面 layout 名不存在即失败（失败即停）。
用法：python apply-flat.py skeleton.pptx spec.json -o final.pptx
"""

from __future__ import annotations

import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pptxkit import EMU_PER_PT, check_16_9, emit, est_overflow, Fail, open_presentation, placeholder_role, run_font  # noqa: E402


def find_ph(slide, roles: tuple) -> object | None:
    for sh in slide.shapes:
        if placeholder_role(sh) in roles:
            return sh
    return None


def fill_text(ph, text: str, fallback_font: str, warnings: list, page_no: int, name: str) -> None:
    tf = ph.text_frame
    tf.text = text
    for para in tf.paragraphs:
        for run in para.runs:
            if not run_font(run):
                run.font.name = fallback_font
    if est_overflow(text, ph.width or 0, ph.height or 0):
        warnings.append(f"第 {page_no} 页 {name} 疑似溢出（粗估）——assemble 后 office_render 复核")


def fill_bullets(ph, bullets: list, fallback_font: str, warnings: list, page_no: int) -> None:
    tf = ph.text_frame
    tf.text = str(bullets[0])
    for b in bullets[1:]:
        p = tf.add_paragraph()
        p.text = str(b)
    for para in tf.paragraphs:
        for run in para.runs:
            if not run_font(run):
                run.font.name = fallback_font
    joined = "\n".join(str(b) for b in bullets)
    if est_overflow(joined, ph.width or 0, ph.height or 0, font_pt=16.0):
        warnings.append(f"第 {page_no} 页 body 疑似溢出（{len(bullets)} 条，粗估）——精简要点或拆页")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("skeleton", help="make-skeleton 产物")
    ap.add_argument("spec", help="文案 spec.json（UTF-8）")
    ap.add_argument("-o", "--output", required=True)
    ns = ap.parse_args()
    try:
        prs = open_presentation(ns.skeleton)
        check_16_9(prs)
        with open(ns.spec, encoding="utf-8") as f:
            spec = json.load(f)
        slides_spec = spec.get("slides") or []
        if not slides_spec:
            raise Fail("spec.slides 为空——至少要一页")
        layout_map = {layout.name: layout for layout in prs.slide_masters[0].slide_layouts}
        fallback = spec.get("font") or ""
        manifest = []
        warnings: list = []
        for i, s in enumerate(slides_spec):
            lname = s.get("layout") or ""
            layout = layout_map.get(lname)
            if layout is None:
                raise Fail(f"第 {i + 1} 页 layout {lname!r} 不在模板版式里；可用：{sorted(layout_map)}")
            slide = prs.slides.add_slide(layout)
            page_no = i + 1
            page_manifest = {"page": page_no, "layout": lname, "filled": []}
            title = s.get("title")
            if title is not None:
                ph = find_ph(slide, ("title", "center_title")) or slide.shapes.title
                if ph is None:
                    raise Fail(f"第 {page_no} 页版式 {lname!r} 没有标题占位符，spec 却给了 title（失败即停）")
                fill_text(ph, str(title), fallback, warnings, page_no, "title")
                page_manifest["filled"].append("title")
            subtitle = s.get("subtitle")
            if subtitle is not None:
                ph = find_ph(slide, ("subtitle", "body"))
                if ph is None:
                    raise Fail(f"第 {page_no} 页版式 {lname!r} 没有副标题占位符，spec 却给了 subtitle")
                fill_text(ph, str(subtitle), fallback, warnings, page_no, "subtitle")
                page_manifest["filled"].append("subtitle")
            bullets = s.get("bullets") or s.get("body")
            if bullets:
                ph = find_ph(slide, ("body",))
                if ph is None:
                    raise Fail(f"第 {page_no} 页版式 {lname!r} 没有内容占位符，spec 却给了 bullets/body")
                fill_bullets(ph, bullets, fallback, warnings, page_no)
                page_manifest["filled"].append("body")
            if s.get("notes"):
                slide.notes_slide.notes_text_frame.text = str(s["notes"])
                page_manifest["filled"].append("notes")
            manifest.append(page_manifest)
        os.makedirs(os.path.dirname(os.path.abspath(ns.output)), exist_ok=True)
        prs.save(ns.output)
        data = {"path": ns.output, "pages": len(manifest), "manifest": manifest,
                "warnings": warnings}
        msg = f"填充 {len(manifest)} 页 → {ns.output}"
        if warnings:
            msg += f"；{len(warnings)} 条溢出警告"
        return emit(True, data, msg)
    except Fail as e:
        return emit(False, None, str(e))
    except Exception as e:  # noqa: BLE001
        return emit(False, None, f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
