import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkspacePane } from "./WorkspacePane";
import { ToastProvider } from "./Toast";
import { resetPaneTabsForTest } from "../lib/paneTabs";
import { resetPaneEnabledForTest } from "../lib/paneEnabled";
import { LocaleProvider } from "../lib/i18n";
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

beforeEach(() => {
  resetPaneTabsForTest();
  mocks.listDir.mockClear();
  mocks.preview.mockClear();
});
afterEach(() => {
  resetPaneTabsForTest();
  localStorage.clear();
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
    fireEvent.click(screen.getByText("README.md"));
    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /README\.md/ }).getAttribute("aria-selected")).toBe("true"),
    );
    expect(await screen.findByText("内容 of README.md")).toBeTruthy();
    // 关闭文件 tab → 回资源管理器视图（README 行仍在）
    fireEvent.click(screen.getByRole("button", { name: "关闭 README.md" }));
    await waitFor(() => expect(screen.getByText("README.md")).toBeTruthy());
  });
});

// v4.524 面板启用集（可选卡红线）：欢迎网格/＋下拉按启用集收敛，管理 Popover
// 为开关常驻入口；cad（defaultEnabled:false）是首个可选面板，用它作实弹。
describe("WorkspacePane 面板启用集（v4.524 可选卡）", () => {
  beforeEach(() => {
    resetPaneTabsForTest();
    resetPaneEnabledForTest();
  });
  afterEach(() => {
    resetPaneTabsForTest();
    resetPaneEnabledForTest();
    localStorage.clear();
  });

  const renderPane = () =>
    render(
      <ToastProvider>
        <LocaleProvider>
          <WorkspacePane context={ctx} />
        </LocaleProvider>
      </ToastProvider>,
    );

  it("默认启用集：欢迎网格无 CAD 卡；管理 Popover 列出全部清单且 CAD 开关关", () => {
    renderPane();
    expect(screen.queryByText("CAD 图纸")).toBeFalsy(); // 可选卡默认不进网格
    expect(screen.getByText("文件", { exact: true })).toBeTruthy(); // 默认启用面照常
    fireEvent.click(screen.getByTestId("tabs-manage"));
    const sw = screen.getByRole("switch", { name: "启用 CAD 图纸" });
    expect(sw.getAttribute("aria-checked")).toBe("false");
    expect(screen.getByRole("switch", { name: "启用 文件" }).getAttribute("aria-checked")).toBe("true");
  });

  it("管理开关切 CAD → 网格即出卡，启用集持久化全局键", async () => {
    renderPane();
    fireEvent.click(screen.getByTestId("tabs-manage"));
    fireEvent.click(screen.getByRole("switch", { name: "启用 CAD 图纸" }));
    // Popover 仍开着（弹层行文案同名），断言网格卡用 welcome-card 锚点
    await waitFor(() => expect(document.querySelector('[data-welcome-card="cad"]')).toBeTruthy());
    expect(
      screen.getByRole("switch", { name: "启用 CAD 图纸" }).getAttribute("aria-checked"),
    ).toBe("true");
    // 写路径走 workbenchStorage 空间分键（gaea.work.*）
    const saved = JSON.parse(localStorage.getItem("gaea.work.rightPanel.v1:tabsEnabled") ?? "{}");
    expect(saved).toMatchObject({ cad: true });
  });

  it("＋下拉未启用组「CAD 图纸（未启用）」点击即启用并开卡", async () => {
    renderPane();
    // ＋按钮在 tab 条上（欢迎态无 tab 条）：先开一个视图 tab
    fireEvent.click(screen.getByText("文件", { exact: true }));
    await waitFor(() => expect(screen.getByRole("button", { name: "新建标签页" })).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "新建标签页" }));
    const item = await waitFor(() => {
      const el = screen
        .getAllByRole("menuitem")
        .find((m) => m.textContent?.includes("CAD 图纸（未启用）"));
      expect(el).toBeTruthy();
      return el as HTMLElement;
    });
    fireEvent.click(item);
    await waitFor(() => expect(screen.getByTestId("cad-panel")).toBeTruthy());
    // 点击即启用：启用集落盘（workbenchStorage 空间分键），欢迎网格此后常驻 CAD 卡
    const saved = JSON.parse(localStorage.getItem("gaea.work.rightPanel.v1:tabsEnabled") ?? "{}");
    expect(saved).toMatchObject({ cad: true });
  });
});
