import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { Modal } from "antd";
import { WorkspacePane } from "./WorkspacePane";
import { ToastProvider } from "./Toast";
import { resetPaneTabsForTest, usePaneTabsStore } from "../lib/paneTabs";
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
  onRevealInTree: vi.fn(),
  onAutoWidenPanel: vi.fn(),
} as unknown as Parameters<typeof WorkspacePane>[0]["context"];

describe("WorkspacePane pane 工作台（对标 better-sidebar）", () => {
  it("空态 = 欢迎卡片；点「文件」卡片开资源管理器视图 tab", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    const { container } = render(
      <ToastProvider>
        <WorkspacePane context={ctx} />
      </ToastProvider>,
    );
    expect(screen.getByText("选择要打开的工作台页面")).toBeTruthy();
    expect(container.querySelector(".workspace-illustration")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "任务" })).toBeNull();
    fireEvent.click(screen.getByText("文件", { exact: true }));
    expect(await screen.findByText("README.md")).toBeTruthy();
    expect(container.querySelector(".workspace-view-context")).toBeNull();
    const fileTab = screen.getByRole("tab", { name: "文件" });
    expect(fileTab.closest("[data-tab-kind]")?.getAttribute("data-tab-kind")).toBe("view");
    expect(fileTab.closest("[data-view-id]")?.getAttribute("data-view-id")).toBe("files");
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

  it("页签使用 roving focus，方向键/End 切换且关闭钮不与 tab 嵌套", async () => {
    const store = usePaneTabsStore.getState();
    store.openFile("README.md", "README.md");
    store.openFile("notes.md", "notes.md");
    store.activate("file:README.md");
    render(
      <ToastProvider>
        <WorkspacePane context={ctx} />
      </ToastProvider>,
    );

    const first = screen.getByRole("tab", { name: /README\.md/ });
    const second = screen.getByRole("tab", { name: /notes\.md/ });
    expect(first.tabIndex).toBe(0);
    expect(second.tabIndex).toBe(-1);
    expect(first.querySelector("button")).toBeNull();
    const tablist = screen.getByRole("tablist", { name: "工作台标签页" });
    expect(within(tablist).queryByRole("button", { name: "新建标签页" })).toBeNull();
    expect(screen.getByRole("button", { name: "新建标签页" })).toBeTruthy();

    fireEvent.keyDown(first, { key: "ArrowRight" });
    await waitFor(() => expect(second.getAttribute("aria-selected")).toBe("true"));
    expect(document.activeElement).toBe(second);

    fireEvent.keyDown(second, { key: "End" });
    expect(second.getAttribute("aria-selected")).toBe("true");
    expect(document.activeElement).toBe(second);
  });
});


