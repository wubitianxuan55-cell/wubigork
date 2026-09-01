// v4.25 A3 EditorTabs 渲染测试：tab 条渲染与激活切换、关闭收敛、空态、
// 内容区复用 FilePreview（编辑能力保留——「换壳不换芯」红线回归）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { EditorTabs } from "./EditorTabs";
import { ToastProvider } from "./Toast";
import { resetEditorTabsForTest, useEditorTabsStore } from "../lib/editorTabs";
import type { PreviewResult } from "../lib/types";

const mocks = vi.hoisted(() => ({
  previewByPath: {} as Record<string, PreviewResult>,
  previewCall: vi.fn(async (rel: string) => mocks.previewByPath[rel] ?? textPreview(rel)),
}));

function textPreview(rel: string, body = `内容 of ${rel}`): PreviewResult {
  return {
    path: rel,
    name: rel.split("/").pop() ?? rel,
    ext: "",
    size: body.length,
    kind: "text",
    body,
    dataUrl: "",
    error: "",
  };
}

vi.mock("../lib/bridge", () => ({
  app: {
    Preview: (rel: string) => mocks.previewCall(rel),
    WriteFile: vi.fn(async () => {}),
    RevealWorkspacePath: async () => {},
    OpenWorkspacePath: async () => {},
  },
  onEvent: () => () => {},
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

beforeEach(() => {
  resetEditorTabsForTest();
  mocks.previewByPath = {};
  mocks.previewCall.mockClear();
});

afterEach(() => {
  resetEditorTabsForTest();
  localStorage.clear();
});

describe("EditorTabs 右栏编辑器 tab（v4.25 A3）", () => {
  it("空态：提示从左侧文件树打开文件", () => {
    render(wrap(<EditorTabs />));
    expect(screen.getByText("从左侧文件树打开文件")).toBeTruthy();
    expect(screen.queryByRole("tablist")).toBeNull();
  });

  it("store 打开多个文件：tab 条渲染、激活态在最后打开的文件上", async () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("docs/b.md");
    render(wrap(<EditorTabs />));

    const tabA = screen.getByRole("tab", { name: /a\.md/ });
    const tabB = screen.getByRole("tab", { name: /b\.md/ });
    expect(tabA.getAttribute("aria-selected")).toBe("false");
    expect(tabB.getAttribute("aria-selected")).toBe("true");
    // 内容区渲染激活文件（docs/b.md）的预览
    expect(await screen.findByText("内容 of docs/b.md")).toBeTruthy();
    expect(screen.queryByText("内容 of a.md")).toBeNull();
  });

  it("点击 tab 切换激活，内容区跟随切换", async () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("docs/b.md");
    render(wrap(<EditorTabs />));
    await screen.findByText("内容 of docs/b.md");

    fireEvent.click(screen.getByRole("tab", { name: /a\.md/ }));
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /a\.md/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of a.md")).toBeTruthy();
  });

  it("tab 条 X 关闭激活 tab：激活相邻；关闭唯一 tab 回空态", async () => {
    useEditorTabsStore.getState().open("a.md");
    useEditorTabsStore.getState().open("b.md");
    useEditorTabsStore.getState().open("c.md");
    render(wrap(<EditorTabs />));
    await screen.findByText("内容 of c.md");

    // 关激活 c.md → 激活右邻不存在 → 左邻 b.md（close 语义：末位取左邻）
    fireEvent.click(screen.getByLabelText("关闭 c.md"));
    expect(screen.getByRole("tab", { name: /b\.md/ }).getAttribute("aria-selected")).toBe("true");
    expect(screen.queryByRole("tab", { name: /c\.md/ })).toBeNull();

    // 关到只剩一个再关 → 空态
    fireEvent.click(screen.getByLabelText("关闭 b.md"));
    fireEvent.click(screen.getByLabelText("关闭 a.md"));
    expect(screen.getByText("从左侧文件树打开文件")).toBeTruthy();
  });

  it("内容区复用 FilePreview：文本预览 + 编辑按钮保留（换壳不换芯）", async () => {
    mocks.previewByPath["notes/a.md"] = {
      path: "notes/a.md",
      name: "a.md",
      ext: ".md",
      size: 12,
      kind: "markdown",
      body: "旧内容",
      dataUrl: "",
      error: "",
    };
    useEditorTabsStore.getState().open("notes/a.md");
    render(wrap(<EditorTabs />));
    expect(await screen.findByText("旧内容")).toBeTruthy();
    expect(screen.getByText("编辑")).toBeTruthy();
  });
});
