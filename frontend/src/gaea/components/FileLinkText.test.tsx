import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FileLinkText } from "./FileLinkText";
import { usePreviewStore } from "../lib/store";

describe("FileLinkText 工具输出文件链接", () => {
  it("纯文本中的文件路径渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<FileLinkText text={"已转换：.gaea/exports/方案.docx\n行尾：报告.md"} />);
    const btn = screen.getByRole("button", { name: /\.gaea\/exports\/方案\.docx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe(".gaea/exports/方案.docx");
  });

  it("无文件引用时原样输出文本", () => {
    const { container } = render(<FileLinkText text="转换完成，共 3 个文件" />);
    expect(container.querySelectorAll("button")).toHaveLength(0);
  });
});
