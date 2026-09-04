import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
const openExternalMock = vi.hoisted(() => vi.fn());
vi.mock("../lib/bridge", () => ({
  app: new Proxy({}, { get: () => () => Promise.resolve({}) }),
  openExternal: openExternalMock,
  onEvent: () => () => {},
}));
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
    // P0-2 chip：文件名 + 扩展名 badge（不含句号后的中文尾巴）
    expect(btn.textContent).toContain("报告.docx");
    fireEvent.click(btn);
    expect(usePreviewStore.getState().previewFile).toBe("报告.docx");
  });

  it("代码块/内联代码里的路径不转成链接", () => {
    render(<Markdown text={"```\nC:\\AI\\bangong\\内部说明.xlsx\n```\n运行 `C:\\AI\\tools\\fix.bat` 即可。"} />);
    expect(screen.queryAllByRole("button").length).toBe(0);
  });
});

describe("Markdown 外链协议分流（1c）", () => {
  beforeEach(() => {
    openExternalMock.mockClear();
  });

  it("loopback 链接点击不交给系统（渲染文档不得探测本机服务）", () => {
    render(<Markdown text="探针 [内网](http://127.0.0.1:8080/api) 谢谢" />);
    const link = screen.getByRole("link", { name: /内网/ });
    expect(link).toBeTruthy();
    fireEvent.click(link);
    expect(openExternalMock).not.toHaveBeenCalled();
  });

  it("https 链接点击交给系统浏览器", () => {
    render(<Markdown text="参考 [文档](https://example.com/doc)" />);
    fireEvent.click(screen.getByRole("link", { name: /文档/ }));
    expect(openExternalMock).toHaveBeenCalledWith("https://example.com/doc");
  });
});
