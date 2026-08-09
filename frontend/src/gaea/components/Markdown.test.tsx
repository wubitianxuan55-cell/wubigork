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

  it("纯文本里的绝对文件路径渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="输出文件：C:\\AI\\bangong\\福达利_成本测算准备资料_v8.3.docx" />);
    const btn = screen.getByRole("button", { name: /福达利_成本测算准备资料_v8\.3\.docx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("C:\\AI\\bangong\\福达利_成本测算准备资料_v8.3.docx");
  });

  it("纯文本里的相对文件路径渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="已生成方案文档：exports/方案.docx，同时更新 .gaea/exports/成本测算.xlsx。" />);
    const btn = screen.getByRole("button", { name: /\.gaea\/exports\/成本测算\.xlsx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe(".gaea/exports/成本测算.xlsx");
  });

  it("关键词引导的裸文件名渲染为可点击预览按钮", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="输出文件：成本测算.xlsx，已生成：报告.docx" />);
    const btn = screen.getByRole("button", { name: /成本测算\.xlsx/ });
    expect(btn).toBeTruthy();
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("成本测算.xlsx");
  });

  it("URL 与域名式文本不误判为本地文件", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="参考 https://example.com/reports/a.pdf 与 docs.example.com/a.pdf。" />);
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("文件路径带中文句号/括号等标点时只链接路径本身", () => {
    usePreviewStore.setState({ previewFile: null });
    render(<Markdown text="保存到：报告.docx。（详见附件）" />);
    const btn = screen.getByRole("button", { name: /报告\.docx/ });
    expect(btn.textContent).toBe("报告.docx");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("报告.docx");
  });

  it("代码块/内联代码里的路径不转成链接", () => {
    render(<Markdown text={"```\nC:\\AI\\bangong\\内部说明.xlsx\n```\n运行 `C:\\AI\\tools\\fix.bat` 即可。"} />);
    expect(screen.queryAllByRole("button").length).toBe(0);
  });
});
