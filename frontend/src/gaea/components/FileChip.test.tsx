import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FileChip } from "./FileChip";
import { fileChipHtml } from "../lib/fileLinks";

describe("FileChip 行内文件引用 chip（P0-2 视觉统一）", () => {
  it("渲染文件名 + 扩展名 badge", () => {
    render(<FileChip path=".gaea/exports/方案.docx" onOpen={() => {}} />);
    const btn = screen.getByRole("button", { name: /预览 方案\.docx/ });
    expect(btn.textContent).toContain("方案.docx");
    expect(btn.textContent).toContain("docx");
  });

  it("点击触发 onOpen(path)", () => {
    let opened = "";
    render(<FileChip path="docs/数据.xlsx" onOpen={(p) => { opened = p; }} />);
    fireEvent.click(screen.getByRole("button"));
    expect(opened).toBe("docs/数据.xlsx");
  });

  it("紧凑形态保留可点击语义", () => {
    render(<FileChip path="a.md" compact onOpen={() => {}} />);
    expect(screen.getByRole("button", { name: /预览 a\.md/ })).toBeTruthy();
  });

  it("路径含 Windows 反斜杠时取末段文件名", () => {
    render(<FileChip path="C:\\AI\\wubigrok\\报告.docx" onOpen={() => {}} />);
    expect(screen.getByText("报告.docx")).toBeTruthy();
  });
});

describe("fileChipHtml 字符串形态（流式尾部同构）", () => {
  it("生成含 data-file-preview 与 badge 的 button", () => {
    const html = fileChipHtml("exports/方案.docx", "方案.docx");
    expect(html).toContain('data-file-preview="exports/方案.docx"');
    expect(html).toContain("方案.docx");
    expect(html).toContain("docx");
  });

  it("label 缺省时取路径末段", () => {
    const html = fileChipHtml("exports/报告.pdf");
    expect(html).toContain("报告.pdf");
    expect(html).toContain("pdf");
  });

  it("HTML 特殊字符转义", () => {
    const html = fileChipHtml('a<b&"c>.md');
    expect(html).not.toContain('<b&"c>');
    expect(html).toContain("&lt;b&amp;&quot;c&gt;");
  });
});
