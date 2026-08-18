import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoMarkdown } from "./MemoMarkdown";
import { usePreviewStore } from "../lib/store";

describe("MemoMarkdown 流式尾部文件链接", () => {
  it("流式未稳定尾部中的文件路径渲染为可点击预览 chip（文件名 + badge）", () => {
    usePreviewStore.setState({ previewFile: null });
    // 无空行分隔 → 全部视为不稳定尾部
    render(<MemoMarkdown text="正在输出文件：exports/方案.docx，请稍候…" streaming />);
    const btn = screen.getByRole("button", { name: /方案\.docx/ });
    expect(btn).toBeTruthy();
    expect(btn.textContent).toContain("方案.docx");
    expect(btn.textContent).toContain("docx");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("exports/方案.docx");
  });

  it("流式尾部代码块里的路径不渲染为链接", () => {
    render(<MemoMarkdown text="```\nC:\\AI\\内部.xlsx\n```" streaming />);
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });
});
