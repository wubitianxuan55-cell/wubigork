#!/usr/bin/env python3
"""docx_layout.py 的最小 docx 夹具单测(zipfile 现场构造,stdlib unittest)。

运行: python test_docx_layout.py
"""
import json
import os
import sys
import tempfile
import unittest
import zipfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import docx_layout

W_NS = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"'
R_NS = 'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"'

BODY_PLAIN = (
    '<w:p><w:pPr><w:spacing w:line="560" w:lineRule="exact"/>'
    '<w:ind w:firstLineChars="200" w:firstLine="560"/><w:jc w:val="center"/></w:pPr>'
    '<w:r><w:rPr><w:rFonts w:eastAsia="方正小标宋简体" w:ascii="方正小标宋简体"/>'
    '<w:sz w:val="44"/></w:rPr><w:t>关于测试工作的通知</w:t></w:r></w:p>'
    '<w:p><w:pPr><w:ind w:firstLineChars="200"/></w:pPr>'
    '<w:r><w:rPr><w:rFonts w:eastAsia="仿宋_GB2312"/><w:sz w:val="32"/></w:rPr>'
    '<w:t>各市人民政府：</w:t></w:r></w:p>'
)

SECTPR_PLAIN = ('<w:sectPr><w:pgMar w:top="2098" w:right="1474" '
                'w:bottom="1984" w:left="1588"/></w:sectPr>')

SECTPR_REFS = ('<w:sectPr><w:pgMar w:top="2098" w:right="1474" '
               'w:bottom="1984" w:left="1588"/>'
               '<w:footerReference w:type="default" r:id="rId3"/>'
               '<w:footerReference w:type="even" r:id="rId4"/></w:sectPr>')

BODY_COLORS = (
    '<w:p><w:pPr><w:jc w:val="center"/></w:pPr>'
    '<w:r><w:rPr><w:rFonts w:eastAsia="方正小标宋简体"/><w:color w:val="FF0000"/>'
    '<w:sz w:val="44"/></w:rPr><w:t>××市大数据局文件</w:t></w:r></w:p>'
    '<w:p>'
    '<w:r><w:rPr><w:color w:val="FF0000"/><w:sz w:val="28"/></w:rPr><w:t>密件</w:t></w:r>'
    '<w:r><w:rPr><w:color w:val="auto"/><w:sz w:val="32"/></w:rPr><w:t>正文黑白</w:t></w:r>'
    '</w:p>' + SECTPR_PLAIN
)

FTR_OPEN = f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr {W_NS}>'

# 一字线 + PAGE 字段(fldChar/instrText 写法) + 缓存结果文本
FOOTER1 = (FTR_OPEN
           + '<w:p><w:pPr><w:jc w:val="center"/></w:pPr>'
           '<w:r><w:rPr><w:rFonts w:eastAsia="宋体"/><w:sz w:val="28"/></w:rPr>'
           '<w:t>— </w:t></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:fldChar w:fldCharType="begin"/></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr>'
           '<w:instrText xml:space="preserve"> PAGE </w:instrText></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:fldChar w:fldCharType="separate"/></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:t>1</w:t></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:fldChar w:fldCharType="end"/></w:r>'
           '<w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:t> —</w:t></w:r>'
           '</w:p></w:ftr>')

# NUMPAGES 含 PAGE 子串但非页码字段,且走 fldSimple 写法
FOOTER2 = (FTR_OPEN
           + '<w:p><w:r><w:rPr><w:sz w:val="28"/></w:rPr>'
           '<w:fldSimple w:instr=" NUMPAGES "><w:r><w:t>2</w:t></w:r></w:fldSimple>'
           '</w:r></w:p></w:ftr>')

FOOTER3 = (FTR_OPEN
           + '<w:p><w:r><w:rPr><w:sz w:val="28"/></w:rPr><w:t>第 3 页</w:t></w:r></w:p></w:ftr>')

RELS = ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument'
        '/2006/relationships/styles" Target="styles.xml"/>'
        '<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument'
        '/2006/relationships/footer" Target="footer1.xml"/>'
        '<Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument'
        '/2006/relationships/footer" Target="footer2.xml"/>'
        '</Relationships>')


def document_xml(body):
    return ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            f'<w:document {W_NS} {R_NS}><w:body>{body}</w:body></w:document>')


def make_docx(path, document, extras=None):
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("word/document.xml", document)
        for name, data in (extras or {}).items():
            z.writestr(name, data)


class DocxLayoutTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.dir = self._tmp.name

    def tearDown(self):
        self._tmp.cleanup()

    def _docx(self, name, document, extras=None):
        path = os.path.join(self.dir, name)
        make_docx(path, document, extras)
        return path

    def test_plain_doc_no_footer_no_color(self):
        """素文档:旧键取值不回归,新键 footers/colors = 空数组。"""
        path = self._docx("plain.docx", document_xml(BODY_PLAIN + SECTPR_PLAIN))
        out = docx_layout.extract(path)
        self.assertTrue(out["margins_found"])
        self.assertEqual(out["margins"], {"top_mm": 37.0, "bottom_mm": 35.0,
                                          "left_mm": 28.0, "right_mm": 26.0})
        self.assertEqual(len(out["paragraphs"]), 2)
        p0, p1 = out["paragraphs"]
        self.assertEqual(p0["text"], "关于测试工作的通知")
        self.assertEqual(p0["font"], "方正小标宋简体")
        self.assertEqual(p0["size_half_pt"], 44)
        self.assertEqual(p0["line_twips"], 560)
        self.assertEqual(p0["line_rule"], "exact")
        self.assertEqual(p0["first_line_chars"], 200)
        self.assertTrue(p0["center"])
        self.assertEqual(p1["font"], "仿宋_GB2312")
        self.assertEqual(p1["size_half_pt"], 32)
        self.assertEqual(p1["first_line_chars"], 200)
        self.assertFalse(p1["center"])
        self.assertEqual(out["footers"], [])
        self.assertEqual(out["colors"], [])

    def test_footer_page_field(self):
        """页脚层:PAGE 字段/一字线文本/字号,NUMPAGES 不误认,未被引用的 footer 照报。"""
        path = self._docx("footer.docx", document_xml(BODY_PLAIN + SECTPR_REFS),
                          {"word/footer1.xml": FOOTER1,
                           "word/footer2.xml": FOOTER2,
                           "word/footer3.xml": FOOTER3,
                           "word/_rels/document.xml.rels": RELS})
        out = docx_layout.extract(path)
        self.assertEqual(len(out["footers"]), 3)
        f1, f2, f3 = out["footers"]
        self.assertEqual(f1["part"], "word/footer1.xml")
        self.assertEqual(f1["ref_types"], ["default"])
        self.assertTrue(f1["has_page_field"])
        self.assertEqual(f1["size_half_pt"], 28)
        self.assertEqual(f1["text"], "— 1 —")
        self.assertEqual(f2["part"], "word/footer2.xml")
        self.assertEqual(f2["ref_types"], ["even"])
        self.assertFalse(f2["has_page_field"])
        self.assertEqual(f2["size_half_pt"], 28)
        self.assertEqual(f2["text"], "2")
        self.assertEqual(f3["part"], "word/footer3.xml")
        self.assertEqual(f3["ref_types"], [])
        self.assertFalse(f3["has_page_field"])
        self.assertEqual(out["colors"], [])
        # 旧键不回归
        self.assertEqual(out["margins"]["bottom_mm"], 35.0)
        self.assertEqual(out["paragraphs"][0]["size_half_pt"], 44)

    def test_color_red_head(self):
        """字色层:红色 run 聚合计数/最大字号/样例拼接,auto 色不采。"""
        path = self._docx("colors.docx", document_xml(BODY_COLORS))
        out = docx_layout.extract(path)
        self.assertEqual(out["colors"], [
            {"val": "FF0000", "count": 2, "max_size_half_pt": 44,
             "sample": "××市大数据局文件密件"},
        ])
        # auto 色不进 colors,但其文本/字号仍进段落事实
        self.assertEqual(out["paragraphs"][1]["text"], "密件正文黑白")
        self.assertEqual(out["paragraphs"][1]["size_half_pt"], 28)
        # 旧键不回归
        self.assertTrue(out["margins_found"])
        self.assertEqual(out["paragraphs"][0]["size_half_pt"], 44)

    def test_output_keys_stable(self):
        """顶层键序稳定,旧键一字不动(向后兼容口)。"""
        path = self._docx("keys.docx", document_xml(BODY_PLAIN + SECTPR_PLAIN))
        out = json.loads(json.dumps(docx_layout.extract(path)))
        self.assertEqual(list(out.keys()),
                         ["margins", "margins_found", "paragraphs", "footers", "colors"])


if __name__ == "__main__":
    unittest.main()
