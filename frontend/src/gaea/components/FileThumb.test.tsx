import { describe, expect, it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { FileThumb, FileTypeIcon, IMAGE_EXT_RE } from "./FileThumb";

// FileThumb 通过 bridge app 代理调用：真实 Wails 绑定或浏览器 mock（getMock）。
// 本测试运行于 vitest（无 window.go），走 lib/mock 的 makeMockApp：
//   - Preview(rel) 对 .xlsx 返回 kind=xlsx + MOCK_XLSX_BODY（含结构化单元格）
//   - Preview 对 .md 返回 kind=markdown + body
//   - AttachmentDataURL 对图片返回 dataURL
// 因此直接渲染即可验证缩略图渲染路径，无需 mock 注入。

describe("FileThumb（P1 产物缩略图增强）", () => {
  it("图片走 dataURL 缩略图", async () => {
    const { container } = render(<FileThumb path="exports/趋势.png" ext=".png" />);
    await waitFor(() => expect(container.querySelector("img")).toBeTruthy());
    expect(container.querySelector("img")?.getAttribute("src")).toMatch(/^data:image\//);
  });

  it("xlsx 渲染迷你表格缩略图（复用 GaeaPreview 结构化数据）", async () => {
    const { container } = render(<FileThumb path="exports/成本测算.xlsx" ext=".xlsx" />);
    await waitFor(() => expect(container.querySelector('[aria-label="表格内容缩略图"]')).toBeTruthy());
    const grid = container.querySelector('[aria-label="表格内容缩略图"]');
    expect(grid?.textContent).toContain("项目"); // MOCK_XLSX_BODY 表头
  });

  it("markdown 渲染文本摘要缩略图", async () => {
    const { container } = render(<FileThumb path="README.md" ext=".md" />);
    await waitFor(() => expect(container.querySelector('[aria-label="文本内容缩略图"]')).toBeTruthy());
    expect(container.querySelector('[aria-label="文本内容缩略图"]')?.textContent).toContain("gaea");
  });

  it("docx 渲染文本摘要缩略图（后端 GaeaPreview 附带 Markdown 文本）", async () => {
    const { container } = render(<FileThumb path="exports/方案.docx" ext=".docx" />);
    await waitFor(() => expect(container.querySelector('[aria-label="文本内容缩略图"]')).toBeTruthy());
    expect(container.querySelector('[aria-label="文本内容缩略图"]')?.textContent).toContain("季度经营");
  });

  it("pdf 渲染文本摘要缩略图（转换后的 Markdown 正文）", async () => {
    const { container } = render(<FileThumb path="exports/报告.pdf" ext=".pdf" />);
    await waitFor(() => expect(container.querySelector('[aria-label="文本内容缩略图"]')).toBeTruthy());
  });

  it("不支持的类型回退类型图标（不抛错）", async () => {
    render(<FileThumb path="exports/素材.psd" ext=".psd" />);
    await new Promise((r) => setTimeout(r, 60));
    expect(screen.getByRole("img")).toBeTruthy();
  });
});

describe("FileTypeIcon / IMAGE_EXT_RE", () => {
  it("扩展名分类正确", () => {
    expect(IMAGE_EXT_RE.test("a.png")).toBe(true);
    expect(IMAGE_EXT_RE.test("a.docx")).toBe(false);
    expect(FileTypeIcon({ ext: ".xlsx", size: 14 }).type.name).toBeTruthy();
  });
});
