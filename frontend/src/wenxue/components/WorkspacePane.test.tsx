import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { WorkspacePane } from "./WorkspacePane";
import { resetPaneTabsForTest } from "../lib/paneTabs";
import type { WorkspacePanelContext } from "../lib/sidebarRegistry";

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
});
