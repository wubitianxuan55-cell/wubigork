import { describe, expect, it } from "vitest";
import JSZip from "jszip";
import { bytesToDocxDataUrl } from "./docxText";
import {
  extractDocxOutline,
  linkDocxOutlineAnchors,
  normalizeDocxOutlineText,
} from "./docxOutline";
import type { DocxOutlineItem } from "./docxOutline";

const WXML =
  'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"';

async function docxDataUrl(documentXml: string, stylesXml?: string): Promise<string> {
  const zip = new JSZip();
  zip.file(
    "word/document.xml",
    `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document ${WXML}><w:body>${documentXml}</w:body></w:document>`,
  );
  if (stylesXml !== undefined) {
    zip.file(
      "word/styles.xml",
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles ${WXML}>${stylesXml}</w:styles>`,
    );
  }
  const bytes = await zip.generateAsync({ type: "uint8array" });
  return bytesToDocxDataUrl(bytes);
}

function para(styleId: string | undefined, text: string, outlineRaw?: string): string {
  const pPr =
    styleId !== undefined || outlineRaw !== undefined
      ? `<w:pPr>${styleId !== undefined ? `<w:pStyle w:val="${styleId}"/>` : ""}${
          outlineRaw !== undefined ? `<w:outlineLvl w:val="${outlineRaw}"/>` : ""
        }</w:pPr>`
      : "";
  return `<w:p>${pPr}<w:r><w:t>${text}</w:t></w:r></w:p>`;
}

const STYLES = `
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style>
  <w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/></w:style>
  <w:style w:type="paragraph" w:styleId="CT1"><w:basedOn w:val="Heading1"/></w:style>
  <w:style w:type="paragraph" w:styleId="zh1"><w:name w:val="标题 1"/><w:pPr><w:outlineLvl w:val="0"/></w:pPr></w:style>
  <w:style w:type="paragraph" w:styleId="Bodyish"><w:name w:val="标题 1"/><w:pPr><w:outlineLvl w:val="9"/></w:pPr></w:style>
  <w:style w:type="paragraph" w:styleId="TOC1"><w:name w:val="toc 1"/></w:style>
`;

describe("docxOutline 标题结构提取", () => {
  it("按样式名/样式继承/outlineLvl/中文标题名识别标题并忽略 TOC1 与正文", async () => {
    const documentXml = [
      para("Heading1", "第一章 项目概述"),
      para(undefined, "引言正文……"),
      para("TOC1", "第一章 项目概述"),
      para("CT1", "第二章 现状分析"),
      para(undefined, "2.1 竞品扫描", "1"),
      para("zh1", "第三章 方案设计"),
      para("Bodyish", "第四章 不应出现"),
    ].join("");
    const items = await extractDocxOutline(await docxDataUrl(documentXml, STYLES));
    expect(items).toEqual([
      { level: 1, text: "第一章 项目概述", paraIndex: 0 },
      { level: 1, text: "第二章 现状分析", paraIndex: 3 },
      { level: 2, text: "2.1 竞品扫描", paraIndex: 4 },
      { level: 1, text: "第三章 方案设计", paraIndex: 5 },
    ]);
  });

  it("缺 styles.xml 时仍支持段落自身 outlineLvl（Word 大纲级别直标）", async () => {
    const documentXml = [
      para(undefined, "第一级", "0"),
      para(undefined, "正文"),
      para(undefined, "第三级", "2"),
    ].join("");
    const items = await extractDocxOutline(await docxDataUrl(documentXml));
    expect(items.map((i) => [i.level, i.text])).toEqual([
      [1, "第一级"],
      [3, "第三级"],
    ]);
  });

  it("包损坏或非 docx 内容时抛错（由调用方降级为目录不可用）", async () => {
    await expect(
      extractDocxOutline(bytesToDocxDataUrl(new TextEncoder().encode("plain text"))),
    ).rejects.toThrow();
  });
});

describe("docxOutline 渲染锚点链接", () => {
  function containerWith(html: string): HTMLElement {
    const container = document.createElement("div");
    container.innerHTML = html;
    return container;
  }

  const items: DocxOutlineItem[] = [
    { level: 1, text: "第一章 项目概述", paraIndex: 0 },
    { level: 2, text: "1.1 目标", paraIndex: 1 },
  ];

  it("跳过页眉/目录域（TOC1）重复文本，锚点落到正文标题段落", () => {
    const container = containerWith(`
      <section class="docx"><header><p>第一章 项目概述</p></header>
      <p class="docx_toc1">第一章 项目概述</p>
      <p class="docx_heading1">第一章 项目概述</p>
      <p>正文内容</p>
      <p class="docx_heading2">1.1 目标</p></section>
    `);
    const anchors = linkDocxOutlineAnchors(container, items);
    expect(anchors).toHaveLength(2);
    expect(anchors[0]?.className).toContain("docx_heading1");
    expect(anchors[1]?.className).toContain("docx_heading2");
  });

  it("找不到精确文本的条目返回 null 且不推进游标（后续条目仍可匹配）", () => {
    const container = containerWith(`
      <p class="docx_heading1">第一章 项目概述</p>
      <p>正文</p>
      <p class="docx_heading2">1.1 目标</p>
    `);
    const withMissing: DocxOutlineItem[] = [
      items[0],
      { level: 2, text: "这个标题在版式里不存在", paraIndex: 99 },
      items[1],
    ];
    const anchors = linkDocxOutlineAnchors(container, withMissing);
    expect(anchors[0]?.className).toContain("docx_heading1");
    expect(anchors[1]).toBeNull();
    expect(anchors[2]?.className).toContain("docx_heading2");
  });

  it("文本比较收敛空白（换行/连续空格按 Word 渲染口径）", () => {
    const container = containerWith(`
      <p class="docx_heading1">第一章
        项目概述</p>
    `);
    const anchors = linkDocxOutlineAnchors(container, items.slice(0, 1));
    expect(anchors[0]?.className).toContain("docx_heading1");
    expect(normalizeDocxOutlineText(" 第一章\n项目概述  ")).toBe("第一章 项目概述");
  });
});
