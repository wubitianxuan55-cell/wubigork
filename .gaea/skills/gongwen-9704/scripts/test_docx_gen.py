#!/usr/bin/env python3
"""docx_gen.py 的自洽单测:生成 → 用 docx_layout.extract 复检 → 断言生成物
逐项命中 SKILL.md 排版细则表(dogfood,校验器即验收器)。stdlib unittest。

运行: python test_docx_gen.py
"""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import docx_gen
import docx_layout

BODY = "\n".join([
    "各区、县人民政府，市政府各部门：",
    "为做好××工作，现将有关事项通知如下，请认真贯彻执行。",
    "一、总体要求",
    "（一）坚持高标准推进，确保各项任务落到实处。",
    "二、保障措施",
    "请各单位结合实际抓好落实。",
])


def gen(tmpdir, **kw):
    out = os.path.join(tmpdir, "out.docx")
    defaults = dict(
        org="××市人民政府", title="关于加强××管理工作的通知",
        docnum="×政发〔2026〕12号", to="各区、县人民政府，市政府各部门：",
        date="2026年9月12日", body_lines=BODY.splitlines(),
        cc="抄送：××。", print_org="××市人民政府办公室",
    )
    defaults.update(kw)
    docx_gen.build_docx(out, **defaults)
    with open(out, "rb") as f:
        data = f.read()
    facts = docx_layout.extract(out)
    return facts, data


class TestDocxGen(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()

    def test_margins_gb9704(self):
        facts, _ = gen(self.tmp)
        self.assertTrue(facts["margins_found"])
        m = facts["margins"]
        self.assertAlmostEqual(m["top_mm"], 37.0, delta=2.0)
        self.assertAlmostEqual(m["bottom_mm"], 35.0, delta=2.0)
        self.assertAlmostEqual(m["left_mm"], 28.0, delta=2.0)
        self.assertAlmostEqual(m["right_mm"], 26.0, delta=2.0)

    def test_header_red_and_large(self):
        facts, _ = gen(self.tmp)
        red = {c["val"]: c for c in facts["colors"]}
        self.assertIn("FF0000", red)
        self.assertGreaterEqual(red["FF0000"]["max_size_half_pt"], 44)
        self.assertIn("××市人民政府", red["FF0000"]["sample"])

    def test_title_font_size_center(self):
        facts, _ = gen(self.tmp)
        titles = [p for p in facts["paragraphs"]
                  if p["text"].startswith("关于") and p["text"].endswith("通知")]
        self.assertTrue(titles)
        t = titles[0]
        self.assertEqual(t["size_half_pt"], 44)
        self.assertTrue(t["center"])

    def test_body_paragraphs_compliant(self):
        """正文段(排除红头/文号/标题/主送/日期/版记)逐段合规:仿宋三号+
        28磅固定行距+首行缩进 2 字符——与排版细则表的多数票口径同向。"""
        facts, _ = gen(self.tmp)
        keys = ("为做好", "落到实处", "抓好落实")
        body = [p for p in facts["paragraphs"] if any(k in p["text"] for k in keys)]
        self.assertTrue(body)
        for p in body:
            self.assertEqual(p["size_half_pt"], 32, p["text"])
            # 「（一）」前缀=二级标题按规范落楷体,其余正文段仿宋
            expected_font = "楷体" if p["text"].startswith("（一）") else "仿宋_GB2312"
            self.assertEqual(p["font"], expected_font, p["text"])
            self.assertEqual((p["line_twips"], p["line_rule"]), (560, "exact"))
            self.assertEqual(p["first_line_chars"], 200)

    def test_level_heading_fonts(self):
        facts, _ = gen(self.tmp)
        by_prefix = {}
        for p in facts["paragraphs"]:
            if p["text"].startswith("一、"):
                by_prefix["hei"] = p["font"]
            if p["text"].startswith("（一）"):
                by_prefix["kai"] = p["font"]
        self.assertEqual(by_prefix.get("hei"), "黑体")
        self.assertEqual(by_prefix.get("kai"), "楷体")

    def test_footer_page_number(self):
        facts, _ = gen(self.tmp)
        self.assertTrue(facts["footers"])
        f = facts["footers"][0]
        self.assertTrue(f["has_page_field"])
        self.assertEqual(f["size_half_pt"], 28)
        self.assertIn("—", f["text"])

    def test_docnum_brackets(self):
        """发文字号年份用六角括号〔〕不用方括号[]（红头要素表口径）。"""
        facts, _ = gen(self.tmp)
        nums = [p["text"] for p in facts["paragraphs"] if "政发〔" in p["text"]]
        self.assertTrue(nums)
        self.assertNotIn("[2026]", nums[0])
        self.assertIn("〔2026〕", nums[0])

    def test_empty_body_placeholder(self):
        facts, _ = gen(self.tmp, body_lines=[])
        texts = [p["text"] for p in facts["paragraphs"]]
        self.assertTrue(any("（正文内容）" in t for t in texts))

    def test_output_opens_as_zip(self):
        import zipfile
        _, data = gen(self.tmp)
        with zipfile.ZipFile(io := __import__("io").BytesIO(data)) as z:
            names = z.namelist()
        for need in ("word/document.xml", "word/footer1.xml", "word/styles.xml"):
            self.assertIn(need, names)


if __name__ == "__main__":
    unittest.main()
