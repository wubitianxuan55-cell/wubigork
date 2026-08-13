import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";
import { Sidebar } from "./Sidebar";
import type { ProjectGroup } from "../lib/types";

vi.mock("../../components/FeatureModelBar", () => ({
  default: () => <div data-testid="feature-model" />,
}));

const group: ProjectGroup = {
  path: "/ws",
  name: "ws",
  current: true,
  modTime: 10,
  sessions: [
    { path: "/ws/cur.jsonl", preview: "当前预览", title: "当前标题", turns: 2, modTime: 10, current: true, pinned: false, hasRequirement: true, requirementDone: false },
    { path: "/ws/other.jsonl", preview: "其他预览", turns: 1, modTime: 5, current: false, pinned: false },
  ],
  archived: [
    { path: "/ws/archive/a.jsonl", preview: "归档预览", turns: 1, modTime: 1, current: false, archived: true },
  ],
};

function renderSidebar() {
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
          projectGroups={[group]}
          searchQuery=""
          onSearchChange={callbacks.onSearchChange}
          onResumeSessionInProject={callbacks.onResumeSessionInProject}
          onArchiveSession={callbacks.onArchiveSession}
          onRestoreSession={callbacks.onRestoreSession}
          onPinSession={callbacks.onPinSession}
          onDeleteSession={callbacks.onDeleteSession}
          onRenameSession={callbacks.onRenameSession}
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
    expect(screen.getByText("当前")).toBeTruthy();
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
});
