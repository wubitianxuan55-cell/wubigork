import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../lib/i18n";
import { ComposerToolbar } from "./ComposerToolbar";

vi.setConfig({ testTimeout: 15_000 });

// 工具下拉按板块派生（2026-11-21「不同面板工具不一样」）：office=CAD/进度计划；
// wenxue=右栏 pane 全集（插图/文件/产物/浏览器/Git/GenUI）。antd Dropdown 点击
// 触发后菜单挂 portal，直接断言菜单项文本。

function renderT(variant?: "office" | "wenxue") {
  localStorage.setItem("gaea-lang", "zh"); // jsdom navigator=en-US 漂英文词典（BootSplash 同款钉法）
  return render(
    <LocaleProvider>
    <ComposerToolbar
      workspaceName="ws"
      workspaceMenuOpen={false}
      onToggleWorkspaceMenu={() => {}}
      workspaceAnchorRef={{ current: null } as React.RefObject<HTMLDivElement>}
      running={false}
      pendingPaste={0}
      captureBusy={false}
      onPickFiles={() => {}}
      onScreenshot={() => {}}
      onOpenTool={vi.fn()}
      variant={variant}
    />
    </LocaleProvider>,
  );
}

afterEach(cleanup);

describe("ComposerToolbar 工具下拉按板块派生", () => {
  it("office（缺省）：工具钮已撤（2026-11-21 用户拍板：CAD/进度入口归运行模式）", () => {
    renderT();
    expect(screen.queryByTestId("composer-tools")).toBeNull();
    expect(screen.queryByText("CAD 图纸")).toBeNull();
    expect(screen.queryByText("进度计划")).toBeNull();
  });

  it("wenxue：插图/文件/产物/浏览器/Git/GenUI，无 CAD", () => {
    renderT("wenxue");
    const btn = screen.getByTestId("composer-tools");
    expect((btn as HTMLButtonElement).title).toContain("插图");
    fireEvent.click(btn);
    for (const label of ["插图", "文件", "产物", "浏览器", "GenUI"]) {
      expect(screen.getByText(label)).toBeTruthy();
    }
    expect(screen.queryByText("CAD 图纸")).toBeNull();
    expect(screen.queryByText("进度计划")).toBeNull();
  });
});
