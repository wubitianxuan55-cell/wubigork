import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { FilePreview } from "./FilePreview";
import { ToastProvider } from "./Toast";
import type { PreviewResult } from "../lib/types";

const mocks = vi.hoisted(() => ({
  preview: null as PreviewResult | null,
  writeFile: vi.fn(async (_rel: string, _content: string) => {}),
  previewCall: vi.fn(async (_rel: string): Promise<PreviewResult> => mocks.preview ?? emptyPreview()),
}));

function emptyPreview(): PreviewResult {
  return { path: "", name: "", ext: "", size: 0, kind: "text", body: "", dataUrl: "", error: "" };
}

vi.mock("../lib/bridge", () => ({
  app: {
    Preview: (rel: string) => mocks.previewCall(rel),
    WriteFile: (rel: string, content: string) => mocks.writeFile(rel, content),
    RevealWorkspacePath: async () => {},
    OpenWorkspacePath: async () => {},
  },
  onEvent: () => () => {},
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

function markdownPreview(over: Partial<PreviewResult> = {}): PreviewResult {
  return {
    path: "notes/a.md",
    name: "a.md",
    ext: ".md",
    size: 12,
    kind: "markdown",
    body: "旧内容",
    dataUrl: "",
    error: "",
    ...over,
  };
}

beforeEach(() => {
  mocks.preview = markdownPreview();
  mocks.writeFile.mockClear();
  mocks.previewCall.mockClear();
  mocks.previewCall.mockImplementation(async (_rel: string) => mocks.preview ?? emptyPreview());
});

describe("FilePreview 工作区内联编辑（C5）", () => {
  it("文本/markdown 预览显示「编辑」按钮，图片预览不显示", async () => {
    const { rerender } = render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    expect(await screen.findByText("旧内容")).toBeTruthy();
    expect(screen.getByText("编辑")).toBeTruthy();

    mocks.preview = { ...markdownPreview(), kind: "image", dataUrl: "data:image/png;base64,x", body: "" };
    rerender(wrap(<FilePreview relPath="pic.png" onClose={() => {}} />));
    await waitFor(() => expect(screen.queryByText("编辑")).toBeNull());
  });

  it("截断的预览不显示「编辑」（内容不完整，写回会丢数据）", async () => {
    mocks.preview = { ...markdownPreview(), truncated: true, size: 5 * 1024 * 1024 };
    render(wrap(<FilePreview relPath="big.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.queryByText("编辑")).toBeNull();
  });

  it("进入编辑 → 修改显示脏标记 → 保存调用 WriteFile 并刷新预览", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "新内容\n第二行" } });
    expect(document.querySelector(".bg-amber-400")).toBeTruthy(); // 脏标记

    // 保存时后端会重读文件 → mock 预置保存后的内容
    mocks.preview = { ...markdownPreview(), body: "新内容\n第二行" };
    fireEvent.click(screen.getByText("保存"));
    await waitFor(() => expect(mocks.writeFile).toHaveBeenCalledWith("notes/a.md", "新内容\n第二行"));
    // 保存后脏标记消失（仍停留在编辑器），预览已重读
    await waitFor(() => expect(document.querySelector(".bg-amber-400")).toBeNull());
    expect(screen.getByLabelText("文本编辑")).toBeTruthy();
  });

  it("脏状态下取消编辑弹内联确认，确认后退出编辑", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "改了一半" } });

    fireEvent.click(screen.getByText("取消"));
    expect(await screen.findByText(/有未保存的修改/)).toBeTruthy();
    fireEvent.click(screen.getByText("放弃修改"));
    await waitFor(() => expect(screen.queryByLabelText("文本编辑")).toBeNull());
  });

  it("Ctrl+S 触发保存", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");

    fireEvent.click(screen.getByText("编辑"));
    const textarea = await screen.findByLabelText("文本编辑");
    fireEvent.change(textarea, { target: { value: "Ctrl+S 保存内容" } });

    fireEvent.keyDown(window, { key: "s", ctrlKey: true });
    await waitFor(() => expect(mocks.writeFile).toHaveBeenCalledWith("notes/a.md", "Ctrl+S 保存内容"));
  });
});

// v4.25 A3 右栏编辑器 tab 嵌入式渲染：embedded 只收窄头部（文件名由 tab 条
// 展示），预览/编辑能力原样保留；默认 false 行为完全不变（上方用例即回归）。
describe("FilePreview 嵌入式渲染（v4.25 A3 embedded）", () => {
  it("embedded：隐藏头部文件名，预览内容与编辑按钮保留", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} embedded />));
    expect(await screen.findByText("旧内容")).toBeTruthy();
    expect(screen.queryByText("a.md")).toBeNull(); // 头部文件名不重复展示
    expect(screen.getByText("编辑")).toBeTruthy(); // 编辑能力保留
    expect(screen.getByTitle("在外部程序中打开")).toBeTruthy(); // 操作区保留
  });

  it("默认（非 embedded）：头部文件名照常展示（向后兼容）", async () => {
    render(wrap(<FilePreview relPath="notes/a.md" onClose={() => {}} />));
    await screen.findByText("旧内容");
    expect(screen.getByText("a.md")).toBeTruthy();
  });
});
