import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Modal } from "antd";
import { WorkspacePane } from "./WorkspacePane";
import { ToastProvider } from "./Toast";
import { resetPaneTabsForTest } from "../lib/paneTabs";
import type { PreviewResult } from "../lib/types";
import { registerEditorDraft, unregisterEditorDraft } from "../lib/editorDrafts";

const mocks = vi.hoisted(() => ({
  listDir: vi.fn(async (_dir: string) => [{ name: "README.md", isDir: false }]),
  preview: vi.fn(async (rel: string): Promise<PreviewResult> => ({
    path: rel,
    name: rel.split("/").pop() ?? rel,
    ext: "",
    size: 3,
    kind: "text",
    body: `内容 of ${rel}`,
    dataUrl: "",
    error: "",
  })),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    ListDir: (dir: string) => mocks.listDir(dir),
    Preview: (rel: string) => mocks.preview(rel),
    WriteFile: vi.fn(async () => {}),
    OpenWorkspacePath: vi.fn(async () => {}),
    RevealWorkspacePath: vi.fn(async () => {}),
    FileSearch: vi.fn(async () => []),
  },
  onEvent: () => () => {},
}));

beforeEach(() => {
  resetPaneTabsForTest();
  mocks.listDir.mockClear();
  mocks.preview.mockClear();
});
afterEach(() => {
  resetPaneTabsForTest();
  localStorage.clear();
  vi.restoreAllMocks();
});

// 最小渲染上下文（Files 视图需要的字段；其余视图本用例不打开）
const ctx = {
  cwd: "C:/proj",
  refreshKey: 0,
  revealRequest: null,
  sessionDeliverables: [],
  sessionChanges: [],
  freshDeliverablePaths: [],
  onOpenFile: vi.fn(),
  onClosePanel: vi.fn(),
  onRefreshPanel: vi.fn(),
  onLocateSource: vi.fn(),
  onSubagentStarted: vi.fn(),
  onRevealInTree: vi.fn(),
  onAutoWidenPanel: vi.fn(),
} as unknown as Parameters<typeof WorkspacePane>[0]["context"];

describe("WorkspacePane pane 工作台（对标 better-sidebar）", () => {
  it("空态 = 欢迎卡片；点「文件」卡片开资源管理器视图 tab", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    render(
      <ToastProvider>
        <WorkspacePane context={ctx} />
      </ToastProvider>,
    );
    expect(screen.getByText("选择要打开的工作台页面")).toBeTruthy();
    fireEvent.click(screen.getByText("文件", { exact: true }));
    expect(await screen.findByText("README.md")).toBeTruthy();
  });

  it("资源管理器内点文件 → 新增文件 tab 浏览；关闭文件 tab 回资源管理器", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    render(
      <ToastProvider>
        <WorkspacePane context={ctx} />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText("文件"));
    await screen.findByText("README.md");
    fireEvent.click(screen.getAllByText("README.md").at(-1)!);
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /README\.md/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of README.md")).toBeTruthy();
    // 关闭文件 tab → 回资源管理器视图（README 行仍在）
    fireEvent.click(screen.getByRole("button", { name: "关闭 README.md" }));
    await waitFor(() => expect(screen.getByText("README.md")).toBeTruthy());
  });

  it("关闭有未保存草稿的文件页先确认，取消时保留 tab", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    const owner = Symbol("dirty");
    registerEditorDraft("README.md", owner);
    let confirmConfig: Parameters<typeof Modal.confirm>[0] | null = null;
    vi.spyOn(Modal, "confirm").mockImplementation((config) => {
      confirmConfig = config;
      return { destroy: vi.fn(), update: vi.fn() } as never;
    });
    render(
      <ToastProvider>
        <WorkspacePane context={ctx} />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText("文件"));
    await screen.findByText("README.md");
    fireEvent.click(screen.getAllByText("README.md").at(-1)!);
    await waitFor(() => expect(screen.getByRole("tab", { name: /README\.md/ })).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "关闭 README.md" }));
    expect(confirmConfig).not.toBeNull();
    expect((confirmConfig as unknown as { title: string }).title).toBe("有未保存的修改");
    expect(screen.getByRole("tab", { name: /README\.md/ })).toBeTruthy();
    unregisterEditorDraft("README.md", owner);
  });
});


