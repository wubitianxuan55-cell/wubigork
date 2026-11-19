import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../lib/i18n";
import type { ProjectGroup, WorkspaceView } from "../lib/types";
import { Sidebar, type SidebarProps } from "./Sidebar";

const currentNovel: ProjectGroup = {
  path: "C:\\AI\\tuwen\\长夜灯火",
  name: "长夜灯火",
  current: true,
  sessions: [],
  archived: [],
  modTime: 2,
};

function renderSidebar(overrides: Partial<SidebarProps> = {}) {
  localStorage.setItem("gaea-lang", "zh");
  const props: SidebarProps = {
    collapsed: false,
    toggleSidebar: vi.fn(),
    running: false,
    jobs: [],
    newSessionAndReset: vi.fn(),
    onCreateNovel: vi.fn(async (name: string, _infoJson: string): Promise<WorkspaceView> => ({
      path: `C:\\AI\\tuwen\\${name}`,
      name,
      current: true,
    })),
    onSelectNovelWorkspace: vi.fn(async (path: string) => path),
    onDeleteNovel: vi.fn(async () => {}),
    projectGroups: [currentNovel],
    onResumeSessionInProject: vi.fn(),
    onArchiveSession: vi.fn(),
    onRestoreSession: vi.fn(),
    onPinSession: vi.fn(),
    onDeleteSession: vi.fn(),
    onRenameSession: vi.fn(),
    onOpenSubagentThread: vi.fn(),
    onOpenHistory: vi.fn(),
    onOpenCaps: vi.fn(),
    startResize: vi.fn(),
    resizeWithKeyboard: vi.fn(),
    onDoubleClickResize: vi.fn(),
    sidebarWidth: 280,
    SIDEBAR_MIN_WIDTH: 220,
    SIDEBAR_MAX_WIDTH: 420,
    searchQuery: "",
    onSearchChange: vi.fn(),
    ...overrides,
  };
  render(
    <LocaleProvider>
      <Sidebar {...props} />
    </LocaleProvider>,
  );
  return props;
}

describe("Wenxue Sidebar novel workspaces", () => {
  it("does not expose office knowledge or planning entry points", () => {
    renderSidebar();

    expect(screen.queryByText("知识库")).toBeNull();
    expect(screen.queryByText("CAD 图纸")).toBeNull();
    expect(screen.queryByText("进度计划")).toBeNull();
    expect(screen.queryByText("工具：CAD 图纸 / 进度计划")).toBeNull();
  });

  it("creates a novel from the left-panel modal", async () => {
    const props = renderSidebar();

    fireEvent.click(screen.getByRole("button", { name: "新建小说" }));
    fireEvent.change(screen.getByLabelText("小说名字"), { target: { value: "星河旧梦" } });
    fireEvent.click(screen.getByRole("button", { name: "创建并进入" }));

    await waitFor(() => expect(props.onCreateNovel).toHaveBeenCalledWith("星河旧梦", expect.any(String)));
  });

  it("switches to another novel workspace from the left list", async () => {
    const otherNovel: ProjectGroup = {
      path: "C:\\AI\\tuwen\\旧城来信",
      name: "旧城来信",
      current: false,
      sessions: [],
      archived: [],
      modTime: 1,
    };
    const props = renderSidebar({ projectGroups: [currentNovel, otherNovel] });

    fireEvent.click(screen.getByTitle(otherNovel.path));

    await waitFor(() => expect(props.onSelectNovelWorkspace).toHaveBeenCalledWith(otherNovel.path));
  });

  it("keeps a visible new-session action on the novel node", () => {
    const props = renderSidebar();

    fireEvent.click(screen.getByRole("button", { name: /在《长夜灯火》中新建会话/ }));

    expect(props.newSessionAndReset).toHaveBeenCalledTimes(1);
  });

  it("switches to another novel before creating its session", async () => {
    const otherNovel: ProjectGroup = {
      path: "C:\\AI\\tuwen\\旧城来信",
      name: "旧城来信",
      current: false,
      sessions: [],
      archived: [],
      modTime: 1,
    };
    const props = renderSidebar({ projectGroups: [currentNovel, otherNovel] });

    fireEvent.click(screen.getByRole("button", { name: /在《旧城来信》中新建会话/ }));

    await waitFor(() => expect(props.onSelectNovelWorkspace).toHaveBeenCalledWith(otherNovel.path));
    await waitFor(() => expect(props.newSessionAndReset).toHaveBeenCalledTimes(1));
  });

  it("confirms before deleting a novel workspace", async () => {
    const props = renderSidebar();

    fireEvent.click(screen.getByRole("button", { name: "删除《长夜灯火》" }));
    expect(screen.getByText(/将删除《长夜灯火》及其中的全部会话、稿件和资料/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "永久删除" }));

    await waitFor(() => expect(props.onDeleteNovel).toHaveBeenCalledWith(currentNovel.path));
  });

  it("隐私模式遮罩作品名、会话名和路径提示", () => {
    const privateNovel: ProjectGroup = {
      ...currentNovel,
      sessions: [{
        path: "C:\\AI\\tuwen\\长夜灯火\\.gaea\\sessions\\one.jsonl",
        preview: "必须保密的情节",
        turns: 3,
        modTime: Date.now() - 60_000,
        current: true,
      }],
    };
    renderSidebar({ projectGroups: [privateNovel], privacyMode: true });

    expect(screen.getAllByText("私密作品").length).toBeGreaterThan(0);
    expect(screen.getByText("私密会话 1")).toBeTruthy();
    expect(screen.queryByText("长夜灯火")).toBeNull();
    expect(screen.queryByText("必须保密的情节")).toBeNull();
    expect(screen.queryByTitle(privateNovel.path)).toBeNull();
  });

  it("折叠态保留统一开合图标、当前会话活动点和底部能力入口", () => {
    const novelWithSession: ProjectGroup = {
      ...currentNovel,
      sessions: [{
        path: "C:\\AI\\tuwen\\长夜灯火\\.gaea\\sessions\\current.jsonl",
        preview: "当前章节",
        title: "长夜灯火 第一章",
        turns: 1,
        modTime: Date.now(),
        current: true,
      }],
    };
    const props = renderSidebar({ collapsed: true, projectGroups: [novelWithSession] });

    const toggle = screen.getByTestId("global-sidebar-toggle");
    expect(toggle.querySelector('[data-workspace-sidebar-toggle="expand"]')).not.toBeNull();

    const current = screen.getByTestId("sidebar-current-session-collapsed");
    expect(current.querySelector(".office-sidebar__current-dot")).not.toBeNull();
    fireEvent.click(current);
    expect(props.toggleSidebar).toHaveBeenCalledTimes(1);

    expect(screen.getByRole("button", { name: "新建小说" })).toBeTruthy();
    fireEvent.click(screen.getByTestId("sidebar-new-session-collapsed"));
    expect(props.newSessionAndReset).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "MCP 与技能" })).toBeTruthy();
    expect(document.querySelector(".office-sidebar__collapsed-actions")).not.toBeNull();
  });
});
