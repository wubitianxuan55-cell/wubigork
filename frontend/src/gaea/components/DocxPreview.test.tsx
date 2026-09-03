import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import JSZip from "jszip";
import { DocxPreview } from "./DocxPreview";
import { extractDocxParagraphs, bytesToDocxDataUrl } from "../lib/docxText";
import { useComposerInsertStore } from "../lib/store";

// docx-preview 渲染行为由用例控制：降级用例让 renderAsync 抛错，
// 正常用例替换为 no-op（jsdom 下完整版式渲染不可行也不必要）。
const renderAsyncMock = vi.hoisted(() => vi.fn());
vi.mock("docx-preview", () => ({ renderAsync: renderAsyncMock }));

// 用 jszip 现场构造一个最小 docx（zip 包 + word/document.xml），段落文本可控。
async function sampleDocxDataUrl(paragraphs: string[]): Promise<string> {
  const zip = new JSZip();
  const body = paragraphs.map((t) => (t ? `<w:p><w:r><w:t>${t}</w:t></w:r></w:p>` : "<w:p/>")).join("");
  zip.file(
    "word/document.xml",
    `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>${body}</w:body></w:document>`,
  );
  const bytes = await zip.generateAsync({ type: "uint8array" });
  return bytesToDocxDataUrl(bytes);
}

// 在预览根节点内放置一个真实文本节点，stub window.getSelection 指向它
// （DocxPreview 的框选检测要求 anchorNode 落在根节点内）。
function stubSelectionInRoot(text: string) {
  const root = document.querySelector(".docx-preview-root")!;
  const span = document.createElement("span");
  root.appendChild(span);
  const node = document.createTextNode("x");
  span.appendChild(node);
  const range = {
    commonAncestorContainer: node,
    getBoundingClientRect: () => ({ left: 0, top: 0, width: 0, height: 0 }),
  };
  const sel = {
    toString: () => text,
    anchorNode: node,
    rangeCount: 1,
    getRangeAt: () => range,
  };
  vi.spyOn(window, "getSelection").mockReturnValue(sel as unknown as Selection);
}

afterEach(() => {
  vi.restoreAllMocks();
  renderAsyncMock.mockReset();
  useComposerInsertStore.setState({ pendingText: null });
});

describe("docxText 纯文本提取", () => {
  it("从 docx 包内 word/document.xml 按序提取段落（含空段落占位）", async () => {
    const dataUrl = await sampleDocxDataUrl(["第一段：项目背景说明", "第二段：预算合计 120.5", ""]);
    expect(await extractDocxParagraphs(dataUrl)).toEqual(["第一段：项目背景说明", "第二段：预算合计 120.5", ""]);
  });

  it("包损坏（非 docx 内容）时抛错，由调用方降级", async () => {
    const dataUrl = bytesToDocxDataUrl(new TextEncoder().encode("plain text, not a zip"));
    await expect(extractDocxParagraphs(dataUrl)).rejects.toThrow();
  });
});

describe("DocxPreview 渲染失败降级（B3）", () => {
  it("docx-preview 抛异常 → 降级为纯文本视图 + 提示条，不落死错误页", async () => {
    renderAsyncMock.mockRejectedValue(new Error("版式引擎不支持该文档"));
    const dataUrl = await sampleDocxDataUrl(["第一段：项目背景说明", "第二段：预算合计 120.5"]);
    render(<DocxPreview dataUrl={dataUrl} fileName="报告.docx" relPath="报告.docx" />);

    const fallback = await screen.findByTestId("docx-fallback");
    expect(fallback.textContent).toContain("已降级为纯文本视图");
    expect(fallback.textContent).toContain("不含图片/文本框与版式信息");
    expect(screen.getByText("第一段：项目背景说明")).toBeTruthy();
    expect(screen.getByText("第二段：预算合计 120.5")).toBeTruthy();
    // 渲染容器让位给降级视图（隐藏而非卸载，重渲染依赖 ref 常驻）
    await waitFor(() => {
      const container = document.querySelector(".docx-preview-body") as HTMLElement;
      expect(container.style.display).toBe("none");
    });
  });

  it("提取也失败 → 死错误页如实展示渲染与提取两个错误", async () => {
    renderAsyncMock.mockRejectedValue(new Error("渲染 boom"));
    // 合法 base64 但非 zip：渲染错误来自 mock（渲染 boom），提取错误来自 jszip。
    const dataUrl = bytesToDocxDataUrl(new TextEncoder().encode("plain text, not a zip"));
    render(
      <DocxPreview
        dataUrl={dataUrl}
        fileName="坏.docx"
        relPath="坏.docx"
      />,
    );
    expect(await screen.findByText(/该 Word 文档渲染失败：渲染 boom/)).toBeTruthy();
    expect(screen.getByText(/纯文本降级也失败/)).toBeTruthy();
    expect(screen.queryByTestId("docx-fallback")).toBeNull();
  });
});

describe("DocxPreview B3 次级「引用到对话」入口", () => {
  it("框选 → 工具栏出现 → 点击引用到对话，回调收到引用块且工具栏收起（框选即改不受影响）", async () => {
    renderAsyncMock.mockResolvedValue(undefined);
    const onQuoteSelection = vi.fn();
    render(
      <DocxPreview dataUrl="data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64," fileName="报告.docx" relPath="报告.docx" onQuoteSelection={onQuoteSelection} />,
    );
    await screen.findByText(/版式保真预览/);

    stubSelectionInRoot("这段选中的文字需要继续处理");
    fireEvent.mouseUp(document.querySelector(".docx-preview-root")!);
    expect(await screen.findByText("AI 编辑选中内容")).toBeTruthy();

    fireEvent.click(screen.getByTestId("docx-quote-btn"));
    expect(onQuoteSelection).toHaveBeenCalledTimes(1);
    const quote = onQuoteSelection.mock.calls[0][0] as string;
    expect(quote).toContain("> 这段选中的文字需要继续处理");
    expect(quote).toContain("请基于以上内容继续处理");
    // 回调优先时不走默认 composer 通道
    expect(useComposerInsertStore.getState().pendingText).toBeNull();
    // 工具栏收起，主流程（AI 编辑/修订）状态清空
    await waitFor(() => expect(screen.queryByText("AI 编辑选中内容")).toBeNull());
  });

  it("缺省回调时走 composer 插入通道（requestText）", async () => {
    renderAsyncMock.mockResolvedValue(undefined);
    render(
      <DocxPreview dataUrl="data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64," fileName="报告.docx" relPath="报告.docx" />,
    );
    await screen.findByText(/版式保真预览/);

    stubSelectionInRoot("引用我");
    fireEvent.mouseUp(document.querySelector(".docx-preview-root")!);
    fireEvent.click(await screen.findByTestId("docx-quote-btn"));
    const text = useComposerInsertStore.getState().pendingText ?? "";
    expect(text).toContain("> 引用我");
  });
});

describe("DocxPreview Word 目录（大纲导航）", () => {
  async function headingDocxDataUrl(): Promise<string> {
    const WXML = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"';
    const zip = new JSZip();
    zip.file(
      "word/document.xml",
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document ${WXML}><w:body>` +
        `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>第一章 项目概述</w:t></w:r></w:p>` +
        `<w:p><w:r><w:t>正文内容</w:t></w:r></w:p>` +
        `<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>1.1 目标</w:t></w:r></w:p>` +
        `</w:body></w:document>`,
    );
    zip.file(
      "word/styles.xml",
      `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles ${WXML}>` +
        `<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style>` +
        `<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/></w:style>` +
        `</w:styles>`,
    );
    const bytes = await zip.generateAsync({ type: "uint8array" });
    return bytesToDocxDataUrl(bytes);
  }

  it("打开目录 → 标题条目可见 → 点击定位到对应标题段落（高亮）", async () => {
    // 假 renderAsync：把标题版式段落写入渲染容器（与 docx-preview 产物同构：
    // 样式类 docx_heading1/2 由 styleName 派生）。
    renderAsyncMock.mockImplementation(async () => {
      const body = document.querySelector<HTMLElement>(".docx-preview-body");
      if (body) {
        body.innerHTML =
          '<p class="docx_heading1">第一章 项目概述</p><p>正文内容</p><p class="docx_heading2">1.1 目标</p>';
      }
    });
    const dataUrl = await headingDocxDataUrl();
    render(<DocxPreview dataUrl={dataUrl} fileName="报告.docx" relPath="报告.docx" />);

    fireEvent.click(await screen.findByTestId("docx-outline-toggle"));
    await waitFor(() => expect(screen.getByTestId("docx-outline-item-0")).toBeTruthy());
    // 目录条目 + 版式正文各出现一次（正文是渲染容器里的真实标题段落）
    expect(screen.getAllByText("第一章 项目概述").length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("1.1 目标").length).toBeGreaterThanOrEqual(2);

    fireEvent.click(screen.getByTestId("docx-outline-item-1"));
    const headingEl = document.querySelector(".docx_heading2");
    expect(headingEl?.classList.contains("docx-outline-flash")).toBe(true);
    // 点击后仍停留在版式预览，AI 编辑流程不受影响
    expect(screen.queryByText("AI 编辑选中内容")).toBeNull();
  });

  it("目录条目的「定位章节修改」把模板插入 composer（不自动发送）", async () => {
    renderAsyncMock.mockImplementation(async () => {
      const body = document.querySelector<HTMLElement>(".docx-preview-body");
      if (body) {
        body.innerHTML = '<p class="docx_heading1">第一章 项目概述</p>';
      }
    });
    const dataUrl = await headingDocxDataUrl();
    render(<DocxPreview dataUrl={dataUrl} fileName="立项报告.docx" relPath="立项报告.docx" />);

    fireEvent.click(await screen.findByTestId("docx-outline-toggle"));
    const editBtn = await screen.findByTestId("docx-outline-edit-0");
    fireEvent.click(editBtn);
    const pending = useComposerInsertStore.getState().pendingText ?? "";
    expect(pending).toContain("请修改 立项报告.docx 中「第一章 项目概述」这一节：");
  });
});
