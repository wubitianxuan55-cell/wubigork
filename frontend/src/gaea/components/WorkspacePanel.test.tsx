// v4.25 A3 WorkspacePanel「文件工作台」测试：
// - 行点击 / 最近文件 chip → 右栏内 EditorTabs 打开（不再走 onSelectFile 收面板）；
// - 双入口保留：文件树右键菜单「预览」仍开主区（onSelectFile）；
// - revealRequest 透传 FileTree（树中定位闪烁）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkspacePanel } from "./WorkspacePanel";
import { ToastProvider } from "./Toast";
import { resetEditorTabsForTest } from "../lib/editorTabs";
import { clearRecentFilesForTest, recordRecentFile } from "../lib/recentFiles";
import type { PreviewResult } from "../lib/types";

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

const wrap = (ui: React.ReactNode) => <ToastProvider>{ui}</ToastProvider>;

beforeEach(() => {
  resetEditorTabsForTest();
  clearRecentFilesForTest();
  mocks.listDir.mockClear();
  mocks.preview.mockClear();
});

afterEach(() => {
  resetEditorTabsForTest();
  localStorage.clear();
});

describe("WorkspacePanel 文件工作台（v4.25 A3 编辑器 tab 化）", () => {
  it("点击文件树行：右栏内 EditorTabs 打开预览，不触发 onSelectFile（不收面板开主区）", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    const onSelectFile = vi.fn();
    const onAutoWiden = vi.fn();
    render(wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={onSelectFile} onAutoWiden={onAutoWiden} />));

    fireEvent.click(await screen.findByText("README.md"));

    // 右栏内 tab 打开：tab 条渲染该文件且为激活态，内容区出现预览正文
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /README\.md/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of README.md")).toBeTruthy();
    // 双入口分工：行点击不再走主区预览回调
    expect(onSelectFile).not.toHaveBeenCalled();
    // v4.27 首次打开文件触发自动加宽（App 侧把右栏抬升到舒适阅读宽度）
    expect(onAutoWiden).toHaveBeenCalledTimes(1);
  });

  it("双入口保留：文件树右键菜单「预览」仍走 onSelectFile 开主区（不开右栏内 tab）", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    const onSelectFile = vi.fn();
    render(wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={onSelectFile} />));
    await screen.findByText("README.md");

    fireEvent.contextMenu(screen.getByText("README.md"));
    fireEvent.click(await screen.findByText("预览"));

    expect(onSelectFile).toHaveBeenCalledWith("README.md");
    expect(mocks.preview).not.toHaveBeenCalled(); // 右栏内编辑器未打开
    expect(screen.queryByRole("tablist")).toBeNull();
  });

  it("RecentFilesBar chip 点击：右栏内 EditorTabs 打开（不触发 onSelectFile）", async () => {
    recordRecentFile("docs/方案.docx");
    mocks.listDir.mockResolvedValue([]);
    const onSelectFile = vi.fn();
    render(wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={onSelectFile} />));

    fireEvent.click(await screen.findByRole("button", { name: /预览 方案\.docx/ }));

    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /方案\.docx/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of docs/方案.docx")).toBeTruthy();
    expect(onSelectFile).not.toHaveBeenCalled();
  });

  it("revealRequest 透传 FileTree：nonce 变化后目标行闪烁高亮", async () => {
    mocks.listDir.mockImplementation(async (dir: string) =>
      dir === "docs" ? [{ name: "plan.md", isDir: false }] : [{ name: "docs", isDir: true }],
    );
    Element.prototype.scrollIntoView = vi.fn();
    const { rerender, unmount } = render(
      wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={() => {}} revealRequest={null} />),
    );
    await screen.findByText("docs");

    rerender(
      wrap(
        <WorkspacePanel
          cwd="C:/proj"
          onClose={() => {}}
          onSelectFile={() => {}}
          revealRequest={{ rel: "docs/plan.md", nonce: 1 }}
        />,
      ),
    );
    const row = await screen.findByText("plan.md");
    await waitFor(() => expect(row.closest("[data-path]")?.getAttribute("data-flash")).toBe("true"));
    unmount();
    delete (Element.prototype as { scrollIntoView?: unknown }).scrollIntoView;
  });

  it("无打开文件时右栏为文件树全屏（编辑器空态提示不占位）", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    render(wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={() => {}} />));
    expect(await screen.findByText("README.md")).toBeTruthy();
    expect(screen.queryByRole("tablist")).toBeNull();
    expect(screen.queryByText("从左侧文件树打开文件")).toBeNull();
  });

  it("v4.27 打开文件后编辑器占满右栏（树自动收起），「文件」按钮可展开树侧栏", async () => {
    mocks.listDir.mockResolvedValue([{ name: "README.md", isDir: false }]);
    render(wrap(<WorkspacePanel cwd="C:/proj" onClose={() => {}} onSelectFile={() => {}} />));

    fireEvent.click(await screen.findByText("README.md"));

    // 编辑器占满右栏：tab 条 + 预览正文在，文件树行已卸载
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /README\.md/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of README.md")).toBeTruthy();
    expect(document.querySelector('[data-path="README.md"]')).toBeNull();

    // 头部「文件」按钮展开树侧栏：文件树行回归，编辑器内容仍保留
    fireEvent.click(screen.getByRole("button", { name: "文件" }));
    await waitFor(() => expect(document.querySelector('[data-path="README.md"]')).not.toBeNull());
    expect(screen.getByText("内容 of README.md")).toBeTruthy();
  });
});
