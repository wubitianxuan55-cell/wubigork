import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FileLinkText } from "./FileLinkText";
import { usePreviewStore } from "../lib/store";

describe("FileLinkText 工具输出文件链接", () => {
  it("纯文本中的文件路径渲染为可点击预览 chip（文件名 + 扩展名 badge）", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<FileLinkText text={"已转换：.gaea/exports/方案.docx\n行尾：报告.md"} />);
    const btn = screen.getByRole("button", { name: /预览 方案\.docx/ });
    expect(btn).toBeTruthy();
    // P0-2：chip 显示文件名（非完整路径）+ 扩展名 badge
    expect(btn.textContent).toContain("方案.docx");
    expect(btn.textContent).toContain("docx");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe(".gaea/exports/方案.docx");
  });

  it("无文件引用时原样输出文本", () => {
    const { container } = render(<FileLinkText text="转换完成，共 3 个文件" />);
    expect(container.querySelectorAll("button")).toHaveLength(0);
  });
});
