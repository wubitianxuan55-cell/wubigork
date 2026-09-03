import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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

function renderSidebar(groups: ProjectGroup[] = [group]) {
  // Sidebar 走 useT：钉住 zh 让既有中文文案断言（当前/置顶/已归档/未完成…）继续成立
  localStorage.setItem("gaea-lang", "zh");
  const callbacks = {
    toggleSidebar: vi.fn(),
    onClearFactBase: vi.fn(),
    onPromoteFactBase: vi.fn().mockResolvedValue(0),
    newSessionAndReset: vi.fn(),
    onSearchChange: vi.fn(),
    onResumeSessionInProject: vi.fn(),
    onArchiveSession: vi.fn(),
    onRestoreSession: vi.fn(),
    onPinSession: vi.fn(),
    onDeleteSession: vi.fn(),
    onRenameSession: vi.fn(),
    onOpenSubagentThread: vi.fn(),
    onOpenHistory: vi.fn(),
    onOpenMemory: vi.fn(),
    onOpenCaps: vi.fn(),
    onOpenKnowledge: vi.fn(),
    startResize: vi.fn(),
    resizeWithKeyboard: vi.fn(),
    onDoubleClickResize: vi.fn(),
  };
  const view = render(
    <LocaleProvider>
      <ToastProvider>
        <Sidebar
          collapsed={false}
          toggleSidebar={callbacks.toggleSidebar}
          running={false}
          jobs={[]}
          factBase={{ facts: [], markdown: "", count: 0, path: "" }}
          onClearFactBase={callbacks.onClearFactBase}
          onPromoteFactBase={callbacks.onPromoteFactBase}
          newSessionAndReset={callbacks.newSessionAndReset}
          projectGroups={groups}
          searchQuery=""
          onSearchChange={callbacks.onSearchChange}
          onResumeSessionInProject={callbacks.onResumeSessionInProject}
          onArchiveSession={callbacks.onArchiveSession}
          onRestoreSession={callbacks.onRestoreSession}
          onPinSession={callbacks.onPinSession}
          onDeleteSession={callbacks.onDeleteSession}
          onRenameSession={callbacks.onRenameSession}
          onOpenSubagentThread={callbacks.onOpenSubagentThread}
          onOpenHistory={callbacks.onOpenHistory}
          onOpenMemory={callbacks.onOpenMemory}
          onOpenCaps={callbacks.onOpenCaps}
          onOpenKnowledge={callbacks.onOpenKnowledge}
          startResize={callbacks.startResize}
          resizeWithKeyboard={callbacks.resizeWithKeyboard}
          onDoubleClickResize={callbacks.onDoubleClickResize}
          sidebarWidth={264}
          SIDEBAR_MIN_WIDTH={200}
          SIDEBAR_MAX_WIDTH={480}
        />
      </ToastProvider>
    </LocaleProvider>,
  );
  return { ...callbacks, view };
}

describe("Sidebar 项目分组与会话操作", () => {
  it("渲染项目名称、当前会话与当前标记", () => {
    renderSidebar();
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
