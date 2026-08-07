import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Markdown } from "./Markdown";
import { usePreviewStore } from "../lib/store";

describe("Markdown 本地文件预览", () => {
  it("本地文件链接渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="报告见 [汇总报告](reports/汇总.md)。" />);
    const btn = screen.getByRole("button", { name: /汇总报告/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("reports/汇总.md");
  });

  it("外部 URL 仍渲染为普通链接", () => {
    render(<Markdown text="参考 [文档](https://example.com/doc)" />);
    expect(screen.queryByRole("button")).toBeNull();
    expect(screen.getByRole("link", { name: /文档/ })).toBeTruthy();
  });
});
