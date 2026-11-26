import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { WorkspacePane } from "./WorkspacePane";
import { resetPaneTabsForTest, usePaneTabsStore } from "../lib/paneTabs";
import type { WorkspacePanelContext } from "../lib/sidebarRegistry";

// 子代理会话 pane 的宿主壳（2026-11-26 文学同步批）：本体拉 runs/Transcript 轮询，
// 这里只验证「页签 → 特例渲染 → 透传取数键」，壳换成纯展示桩。
vi.mock("./SubagentPaneChat", () => ({
  SubagentPaneChat: (p: { sessionPath: string; target: string; fallbackTitle: string }) => (
    <div data-testid="subagent-pane-chat">{`${p.sessionPath}|${p.target}|${p.fallbackTitle}`}</div>
  ),
}));

const context = {
  cwd: "C:/books/wugang",
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
} as unknown as WorkspacePanelContext;

beforeEach(() => {
  resetPaneTabsForTest();
  localStorage.clear();
});

afterEach(() => {
  cleanup();
  resetPaneTabsForTest();
  localStorage.clear();
});

describe("文学 WorkspacePane", () => {
  it("启动页使用统一工作台壳，并保留插图快捷入口", () => {
    const { container } = render(<WorkspacePane context={context} />);

    expect(screen.getByText("选择要打开的工作台页面")).toBeTruthy();
    expect(container.querySelector(".workspace-illustration")).toBeTruthy();
    const gallery = container.querySelector('[data-welcome-card="gallery"]');
    expect(gallery).toBeTruthy();
    expect(gallery?.textContent).toContain("插图");
    expect(gallery?.textContent).toContain("封面、场景插图与角色立绘");
  });

  it("子代理会话页签由 SubagentPaneChat 特例渲染，取数键原样透传", () => {
    usePaneTabsStore.getState().openSubagentPane("sa_1", "核对第三章时间线");
    render(
      <WorkspacePane
        context={{ ...context, currentSessionPath: "C:/books/wugang/.gaea/sessions/s1.jsonl" } as WorkspacePanelContext}
      />,
    );
    expect(screen.getByTestId("subagent-pane-chat").textContent).toBe(
      "C:/books/wugang/.gaea/sessions/s1.jsonl|sa_1|核对第三章时间线",
    );
  });
});
