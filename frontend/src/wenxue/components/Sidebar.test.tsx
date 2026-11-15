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
  const props: SidebarProps = {
    collapsed: false,
    toggleSidebar: vi.fn(),
    running: false,
    jobs: [],
    newSessionAndReset: vi.fn(),
    onCreateNovel: vi.fn(async (name: string): Promise<WorkspaceView> => ({
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

    await waitFor(() => expect(props.onCreateNovel).toHaveBeenCalledWith("星河旧梦"));
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
});
