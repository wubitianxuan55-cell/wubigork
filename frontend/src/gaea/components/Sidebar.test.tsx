import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";
import { Sidebar } from "./Sidebar";
import type { ProjectGroup, SessionMeta } from "../lib/types";

const group: ProjectGroup = {
  path: "/ws",
  name: "ws",
  current: true,
  modTime: 10,
  sessions: [
    { path: "/ws/cur.jsonl", preview: "当前预览", title: "当前标题", turns: 2, modTime: 10, current: true, pinned: false },
    { path: "/ws/other.jsonl", preview: "其他预览", turns: 1, modTime: 5, current: false, pinned: false },
  ],
  archived: [
    { path: "/ws/archive/a.jsonl", preview: "归档预览", turns: 1, modTime: 1, current: false, archived: true },
  ],
};

function renderSidebar(groups: ProjectGroup[] = [group], sidebarWidth = 264, collapsed = false) {
  // Sidebar 走 useT：钉住 zh 让既有中文文案断言（当前/置顶/已归档/未完成…）继续成立
  localStorage.setItem("gaea-lang", "zh");
  const callbacks = {
    toggleSidebar: vi.fn(),
    onNewSession: vi.fn(),
    onCreateProject: vi.fn().mockResolvedValue(undefined),
    onSearchChange: vi.fn(),
    onSwitchProject: vi.fn(),
    onNewSessionInProject: vi.fn(),
    onRemoveProject: vi.fn(),
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
  };
  const view = render(
    <LocaleProvider>
      <ToastProvider>
        <Sidebar
          collapsed={collapsed}
          toggleSidebar={callbacks.toggleSidebar}
          onNewSession={callbacks.onNewSession}
          running={false}
          jobs={[]}
          onCreateProject={callbacks.onCreateProject}
          projectGroups={groups}
          searchQuery=""
          onSearchChange={callbacks.onSearchChange}
          onSwitchProject={callbacks.onSwitchProject}
          onNewSessionInProject={callbacks.onNewSessionInProject}
          onRemoveProject={callbacks.onRemoveProject}
          onResumeSessionInProject={callbacks.onResumeSessionInProject}
          onArchiveSession={callbacks.onArchiveSession}
          onRestoreSession={callbacks.onRestoreSession}
          onPinSession={callbacks.onPinSession}
          onDeleteSession={callbacks.onDeleteSession}
          onRenameSession={callbacks.onRenameSession}
          onOpenSubagentThread={callbacks.onOpenSubagentThread}
          onOpenHistory={callbacks.onOpenHistory}
          onOpenCaps={callbacks.onOpenCaps}
          startResize={callbacks.startResize}
          resizeWithKeyboard={callbacks.resizeWithKeyboard}
          onDoubleClickResize={callbacks.onDoubleClickResize}
          sidebarWidth={sidebarWidth}
          SIDEBAR_MIN_WIDTH={200}
          SIDEBAR_MAX_WIDTH={480}
        />
      </ToastProvider>
    </LocaleProvider>,
  );
  return { ...callbacks, view };
}

describe("Sidebar 项目分组与会话操作", () => {
  it("v4.73：左侧不再渲染「记忆」入口（已迁主区记忆 tab）", () => {
    renderSidebar();
    expect(screen.queryByTitle("记忆")).toBeNull();
    expect(screen.queryByTitle("知识库")).toBeNull();
  });

  it("不再渲染过时的办公空间与娱乐空间切换", () => {
    renderSidebar();
    expect(screen.queryByText("办公空间")).toBeNull();
    expect(screen.queryByText("娱乐空间")).toBeNull();
  });

  it("不再渲染左侧事实底座", () => {
    renderSidebar();
    expect(screen.queryByText("事实底座")).toBeNull();
  });

  it("窄侧栏切换为单列快捷入口", () => {
    const { view } = renderSidebar([group], 240);
    expect(view.container.querySelector('aside')?.getAttribute("data-narrow")).toBe("true");
  });

  it("折叠态使用统一工作台窄轨：开合图标、新建、当前会话和底部图标", () => {
    const c = renderSidebar([group], 264, true);
    const toggle = screen.getByTestId("global-sidebar-toggle");
    expect(toggle.querySelector('[data-workspace-sidebar-toggle="expand"]')).not.toBeNull();
    expect(c.view.container.querySelector(".office-sidebar__brand-mark")).toBeNull();
    expect(c.view.container.querySelector(".office-sidebar__brand-name")).toBeNull();

    const current = screen.getByTestId("sidebar-current-session-collapsed");
    expect(current.querySelector(".office-sidebar__current-dot")).not.toBeNull();
    fireEvent.click(current);
    expect(c.toggleSidebar).toHaveBeenCalledTimes(1);

    expect(screen.getByRole("button", { name: "新建项目" })).toBeTruthy();
    fireEvent.click(screen.getByTestId("sidebar-new-session-collapsed"));
    expect(c.onNewSession).toHaveBeenCalledTimes(1);

    expect(screen.getByRole("button", { name: "MCP 与技能" })).toBeTruthy();
    expect(c.view.container.querySelector(".office-sidebar__collapsed-actions")).not.toBeNull();
  });

  it("展开态侧栏身份只保留文字，不再重复一级导航的 logo", () => {
    const { view } = renderSidebar();
    expect(view.container.querySelector(".office-sidebar__brand-mark")).toBeNull();
    expect(view.container.querySelector(".office-sidebar__brand-name")?.textContent).toContain("gaea");
    expect(screen.getByTestId("global-sidebar-toggle")).toBeTruthy();
  });

  it("渲染项目名称、当前会话与当前标记", () => {
    renderSidebar();
    expect(screen.getByText(/项目.*ws/)).toBeTruthy();
    expect(screen.getByText("ws")).toBeTruthy();
    expect(screen.getByText("当前标题")).toBeTruthy();
    // 「当前」出现两处：项目分组徽标 + 当前会话时间列（history.current），zh 钉住后均命中
    expect(screen.getAllByText("当前").length).toBeGreaterThanOrEqual(2);
  });

  it("点击置顶按钮触发 onPinSession", () => {
    const c = renderSidebar();
    fireEvent.click(screen.getByTitle("置顶"));
    expect(c.onPinSession).toHaveBeenCalledWith("/ws/other.jsonl", true);
  });

  it("展开「已归档」分组后可恢复归档会话", () => {
    const c = renderSidebar();
    fireEvent.click(screen.getByText("已归档"));
    const restore = screen.getByTitle("恢复");
    fireEvent.click(restore);
    expect(c.onRestoreSession).toHaveBeenCalledWith("/ws/archive/a.jsonl", "/ws");
  });

  it("2026-11-19 双击非当前项目切换工作区；当前项目双击不切换", () => {
    const other: ProjectGroup = {
      path: "/other", name: "other", current: false, modTime: 8,
      sessions: [{ path: "/other/s.jsonl", preview: "他项预览", turns: 1, modTime: 8, current: false }],
      archived: [],
    };
    const c = renderSidebar([group, other]);
    fireEvent.doubleClick(screen.getByText("other"));
    expect(c.onSwitchProject).toHaveBeenCalledWith("/other");
    fireEvent.doubleClick(screen.getByText("ws"));
    expect(c.onSwitchProject).not.toHaveBeenCalledWith("/ws");
  });

  it("2026-11-19 项目归档：非当前项目可归档隐藏，恢复后回到列表", () => {
    localStorage.removeItem("gaea.sidebar.archivedProjects");
    const other: ProjectGroup = {
      path: "/other", name: "other", current: false, modTime: 8,
      sessions: [{ path: "/other/s.jsonl", preview: "他项预览", turns: 1, modTime: 8, current: false }],
      archived: [],
    };
    renderSidebar([group, other]);
    fireEvent.click(screen.getByTestId("sidebar-archive-project-other"));
    // 归档后项目从主列表消失，尾部出现「已归档项目」区
    expect(screen.queryByText("other")).toBeNull();
    fireEvent.click(screen.getByText("已归档项目"));
    fireEvent.click(screen.getByTestId("sidebar-restore-project-other"));
    // 恢复后项目回到主列表，归档区收起
    expect(screen.getByText("other")).toBeTruthy();
    localStorage.removeItem("gaea.sidebar.archivedProjects");
  });

  it("2026-11-19 项目行操作组：+ 在项目中新建；删除调 onRemoveProject（当前项目只留新建）", () => {
    const other: ProjectGroup = {
      path: "/other", name: "other", current: false, modTime: 8,
      sessions: [{ path: "/other/s.jsonl", preview: "他项预览", turns: 1, modTime: 8, current: false }],
      archived: [],
    };
    const c = renderSidebar([group, other]);
    const otherActions = screen.getByTestId("sidebar-project-actions-other");
    const currentActions = screen.getByTestId("sidebar-project-actions-ws");
    expect(otherActions.className).toBe("office-project-actions");
    expect(currentActions.className).toBe("office-project-actions");
    expect(otherActions.querySelectorAll(".office-project-action")).toHaveLength(3);
    expect(currentActions.querySelectorAll(".office-project-action")).toHaveLength(1);
    fireEvent.click(screen.getByTestId("sidebar-new-in-project-other"));
    expect(c.onNewSessionInProject).toHaveBeenCalledWith("/other", false);
    fireEvent.click(screen.getByTestId("sidebar-new-in-project-ws"));
    expect(c.onNewSessionInProject).toHaveBeenCalledWith("/ws", true);
    fireEvent.click(screen.getByTestId("sidebar-remove-project-other"));
    expect(c.onRemoveProject).toHaveBeenCalledWith("/other");
    // 当前项目无删除/归档按钮
    expect(screen.queryByTestId("sidebar-remove-project-ws")).toBeNull();
    expect(screen.queryByTestId("sidebar-archive-project-ws")).toBeNull();
  });

  it("2026-11-19 顶部主按钮为「新建项目」：打开对话框提交 onCreateProject", async () => {
    const c = renderSidebar();
    fireEvent.click(screen.getByText("新建项目"));
    const nameInput = await screen.findByLabelText("项目名");
    fireEvent.change(nameInput, { target: { value: "造价审计" } });
    fireEvent.click(screen.getByText("创建并进入"));
    await waitFor(() => expect(c.onCreateProject).toHaveBeenCalledWith("D:\\新工作", "造价审计"));
  });

  it("interrupted 会话显示「未完成」徽标与提示", () => {
    // 过渡期：SessionMeta.interrupted 由契约层子代理补充，mock 数据用类型断言过渡
    const interruptedGroup: ProjectGroup = {
      path: "/ws",
      name: "ws",
      current: true,
      modTime: 20,
      sessions: [
        { path: "/ws/int.jsonl", preview: "中断预览", turns: 1, modTime: 20, current: false, interrupted: true } as SessionMeta,
        { path: "/ws/norm.jsonl", preview: "正常预览", turns: 1, modTime: 15, current: false },
      ],
      archived: [],
    };
    renderSidebar([interruptedGroup]);
    // 中断会话出现徽标，且 tooltip 文案正确
    expect(screen.getByText("未完成")).toBeTruthy();
    expect(screen.getByTitle("上次运行中断，恢复后会自动带上进度摘要")).toBeTruthy();
    // 正常会话不出现徽标（全页只有一处）
    expect(screen.getAllByText("未完成")).toHaveLength(1);
  });

  it("会话行展开后渲染子代理子行，点击子行打开独立子代理会话", async () => {
    const subGroup: ProjectGroup = {
      path: "/mock",
      name: "mock",
      current: true,
      modTime: 10,
      sessions: [
        { path: "/mock/sessions/c.jsonl", preview: "当前预览", title: "当前标题", turns: 2, modTime: 10, current: true, pinned: false },
        { path: "/mock/sessions/b.jsonl", preview: "其他预览", title: "其他标题", turns: 1, modTime: 5, current: false, pinned: false },
      ],
      archived: [],
    };
    const c = renderSidebar([subGroup]);

    // 展开当前会话（c.jsonl，mock 有 running + completed 两个子代理）
    const toggle = document.querySelector('[data-sidebar-subagent-toggle="/mock/sessions/c.jsonl"]') as HTMLElement;
    expect(toggle).toBeTruthy();
    fireEvent.click(toggle);
    const task = await screen.findByText("调研竞品表格 Agent 能力并总结可蒸馏点");
    expect(task).toBeTruthy();

    // 点击子行 → 复用既有「独立子代理会话 tab」入口（不替换主会话）
    const row = document.querySelector('[data-sidebar-subagent-row="/mock/sessions/c.jsonl:sa_20260817_110000_0000000002_b2b2b2b2"]') as HTMLElement;
    expect(row).toBeTruthy();
    fireEvent.click(row);
    expect(c.onOpenSubagentThread).toHaveBeenCalledWith({
      sessionPath: "/mock/sessions/c.jsonl",
      ref: "sa_20260817_110000_0000000002_b2b2b2b2",
      task: "调研竞品表格 Agent 能力并总结可蒸馏点",
      model: undefined,
      status: "running",
    });
  });

  it("无子代理的会话展开后显示空态并可收起", async () => {
    const subGroup: ProjectGroup = {
      path: "/mock",
      name: "mock",
      current: true,
      modTime: 10,
      sessions: [
        { path: "/mock/sessions/c.jsonl", preview: "当前预览", title: "当前标题", turns: 2, modTime: 10, current: true, pinned: false },
        { path: "/mock/sessions/b.jsonl", preview: "其他预览", title: "其他标题", turns: 1, modTime: 5, current: false, pinned: false },
      ],
      archived: [],
    };
    renderSidebar([subGroup]);

    const toggle = document.querySelector('[data-sidebar-subagent-toggle="/mock/sessions/b.jsonl"]') as HTMLElement;
    fireEvent.click(toggle);
    expect(await screen.findByText("该会话暂无子代理")).toBeTruthy();

    // 再次点击收起：空态行消失
    fireEvent.click(toggle);
    expect(screen.queryByText("该会话暂无子代理")).toBeNull();
  });

  it("虚拟化：1000 条会话只渲染可见窗口（<50 行），滚动后窗口移动", () => {
    const sessions: SessionMeta[] = Array.from({ length: 1000 }, (_, i) => ({
      path: `/ws/s${i}.jsonl`,
      preview: `预览 ${i}`,
      title: `会话 ${i}`,
      turns: 1,
      modTime: 1000 - i,
      current: false,
      pinned: false,
    }));
    const bigGroup: ProjectGroup = {
      path: "/big",
      name: "big",
      current: true,
      modTime: 1000,
      sessions,
      archived: [],
    };
    const { view } = renderSidebar([bigGroup]);

    // 默认每页折叠为 8 条 + 「显示更多」；点击后进入 1001 行虚拟列表
    fireEvent.click(screen.getByText(/显示更多/));

    // 渲染窗口化：DOM 中会话行远小于总条数（虚拟滚动生效）
    const rendered = screen.getAllByText(/^会话 \d+$/);
    expect(rendered.length).toBeGreaterThan(0);
    expect(rendered.length).toBeLessThan(50);
    expect(rendered.length).toBeLessThan(1000);
    // 首行可见、末行（会话 999）不在 DOM
    expect(screen.getByText("会话 0")).toBeTruthy();
    expect(screen.queryByText("会话 999")).toBeNull();

    // 滚动到中部：窗口移动，旧首行卸载、中部行出现
    const listEl = view.container.querySelector(".sidebar-session-scroll") as HTMLElement;
    expect(listEl).toBeTruthy();
    Object.defineProperty(listEl, "scrollTop", { value: 44 * 20, configurable: true, writable: true });
    fireEvent.scroll(listEl);
    expect(screen.queryByText("会话 0")).toBeNull();
    expect(screen.getByText("会话 20")).toBeTruthy();
  });
});
