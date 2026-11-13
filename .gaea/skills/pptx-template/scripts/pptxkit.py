#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""pptxkit.py — pptx-template 技能公共库（gaea 自研）。

契约同 pdf_cli：stdout 恒单 JSON {success, data, message}；success:false 即失败。
纪律（灵犀蒸馏）：失败即停（模板非 16:9 直接判败）；字体白名单 ∩ 系统字体，
空交集回退微软雅黑；入口互斥——本技能只做「模板复刻」，新建走 pptx 技能，
局部编辑走 office_edit。
"""

from __future__ import annotations

import json
import sys

try:
    sys.stdout.reconfigure(encoding="utf-8")
except Exception:
    pass

# 16:9 硬约束（EMU）：13.333in × 7.5in。
EMU_16_9 = (12192000, 6858000)
EMU_PER_PT = 12700
FALLBACK_FONT = "微软雅黑"
# 白名单：办公交付常用字体（∩ 系统字体后生效）。
FONT_WHITELIST = [
    "微软雅黑", "Microsoft YaHei", "等线", "DengXian", "黑体", "SimHei",
    "宋体", "SimSun", "楷体", "KaiTi", "仿宋", "FangSong",
    "Arial", "Calibri", "Segoe UI", "Times New Roman", "Georgia",
    "Consolas", "Cambria", "Candara", "Corbel",
]


def emit(success: bool, data=None, message: str = "") -> int:
    print(json.dumps({"success": success, "data": data, "message": message},
                     ensure_ascii=False, default=str))
    return 0 if success else 1


class Fail(Exception):
    """业务失败：给用户看的一句话，走 success:false。"""


def need_pptx():
    try:
        import pptx  # noqa: F401
    except ImportError:
        raise Fail("缺依赖 python-pptx：pip install python-pptx")
    from pptx import Presentation
    return Presentation


def open_presentation(path: str):
    Presentation = need_pptx()
    try:
        return Presentation(path)
    except Exception as e:  # noqa: BLE001
        raise Fail(f"打不开 {path}（不是有效 pptx？）：{e}")


def check_16_9(prs) -> None:
    """失败即停：模板不是 16:9 直接判败（灵犀 pptx 纪律）。"""
    got = (prs.slide_width, prs.slide_height)
    if got != EMU_16_9:
        raise Fail(
            f"模板不是 16:9（{got[0]/914400:.2f}in×{got[1]/914400:.2f}in）。"
            f"模板复刻仅支持 16:9——先把母版画布改成 13.33×7.5in 再跑")


def system_font_names() -> set:
    """GDI EnumFontFamiliesExW 枚举本机字族名（含中文别名），失败返回空集。"""
    names: set = set()
    try:
        import ctypes
        gdi32 = ctypes.windll.gdi32
        user32 = ctypes.windll.user32
        hdc = user32.GetDC(0)

        class LOGFONTW(ctypes.Structure):
            _fields_ = [
                ("lfHeight", ctypes.c_long), ("lfWidth", ctypes.c_long),
                ("lfEscapement", ctypes.c_long), ("lfOrientation", ctypes.c_long),
                ("lfWeight", ctypes.c_long), ("lfItalic", ctypes.c_byte),
                ("lfUnderline", ctypes.c_byte), ("lfStrikeOut", ctypes.c_byte),
                ("lfCharSet", ctypes.c_byte), ("lfOutPrecision", ctypes.c_byte),
                ("lfClipPrecision", ctypes.c_byte), ("lfQuality", ctypes.c_byte),
                ("lfPitchAndFamily", ctypes.c_byte),
                ("lfFaceName", ctypes.c_wchar * 32),
            ]

        ENUMPROC = ctypes.WINFUNCTYPE(
            ctypes.c_int, ctypes.POINTER(LOGFONTW), ctypes.c_void_p,
            ctypes.c_uint, ctypes.c_void_p)

        def _cb(lp, lptm, font_type, lpv):
            try:
                name = lp.contents.lfFaceName
                if name:
                    names.add(name)
            except Exception:
                pass
            return 1

        lf = LOGFONTW()
        lf.lfCharSet = 1  # DEFAULT_CHARSET
        cb = ENUMPROC(_cb)
        gdi32.EnumFontFamiliesExW(hdc, ctypes.byref(lf), cb, 0, 0)
    except Exception:
        pass  # 非 Windows/枚举失败：返回空集，调用方走全白名单回退逻辑
    return names


def effective_fonts(observed: set) -> dict:
    """白名单 ∩ 系统字体；空交集回退微软雅黑。返回 {allow, fallback, missing}。"""
    sysfonts = system_font_names()
    if sysfonts:
        allow = sorted(f for f in FONT_WHITELIST if f in sysfonts)
    else:
        allow = sorted(FONT_WHITELIST)  # 枚举失败：全白名单放行（诚实降级）
    missing = sorted(f for f in observed
                     if f and f not in allow
                     and not any(f in a or a in f for a in allow))
    fallback = FALLBACK_FONT if FALLBACK_FONT in allow else (allow[0] if allow else FALLBACK_FONT)
    return {"allow": allow, "fallback": fallback, "missing": missing}


def shape_texts(shape) -> list:
    """取 shape 全部 run 文本（表格递归单元格）。"""
    out = []
    if getattr(shape, "has_text_frame", False):
        for para in shape.text_frame.paragraphs:
            for run in para.runs:
                out.append(run)
    if getattr(shape, "has_table", False):
        for row in shape.table.rows:
            for cell in row.cells:
                for para in cell.text_frame.paragraphs:
                    for run in para.runs:
                        out.append(run)
    return out


def run_font(run) -> str:
    """run 级字体名（继承链的显式层；空=继承主题）。"""
    try:
        return run.font.name or ""
    except Exception:
        return ""


def placeholder_role(shape) -> str:
    """占位符角色归一：title / body / subtitle / other（非占位符=none）。"""
    if not shape.is_placeholder:
        return "none"
    name = str(shape.placeholder_format.type)
    if "TITLE" in name and "SUBTITLE" not in name:
        return "title"
    if "SUBTITLE" in name:
        return "subtitle"
    if "BODY" in name or "OBJECT" in name or "TEXT" in name:
        return "body"
    return "other"


def est_overflow(text: str, shape_width_emu: int, shape_height_emu: int, font_pt: float = 18.0) -> bool:
    """溢出粗估：中文 1 字宽≈1em、ASCII≈0.55em，行高≈1.25em（近似，给 warning 不阻断）。"""
    if not text:
        return False
    width_pt = shape_width_emu / EMU_PER_PT
    height_pt = shape_height_emu / EMU_PER_PT
    per_line = max(1, int(width_pt / (font_pt * 0.9)))
    lines = 0
    for seg in text.split("\n"):
        weight = sum(1.0 if ord(ch) > 0x2E7F else 0.55 for ch in seg)
        lines += max(1, int(weight / per_line) + (1 if weight % per_line else 0))
    return lines * font_pt * 1.25 > height_pt
