import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../lib/i18n";
import { ComposerToolbar } from "./ComposerToolbar";
import type { OfficeRunModeViewB } from "../../lib/bridge/model";

vi.setConfig({ testTimeout: 15_000 });

// 运行模式选择（2026-11-21 众神殿终案 · 会话定档语义）：
// 新会话未发消息=三态可选，选档即定档（本会话按新档重建生效）；
// 会话已有内容/回合运行中=锁定只显当前档徽标（切档需新建会话）；
// 触发钮摘要按当前会话实跑档（active）显形。

function renderT(props: Partial<Parameters<typeof ComposerToolbar>[0]> = {}) {
  localStorage.setItem("gaea-lang", "zh"); // jsdom navigator=en-US 漂英文词典（同款钉法）
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
        permLevel="ask"
        thinkLevel="normal"
        runMode={view("standard", "standard")}
        onSetRunMode={vi.fn()}
        {...props}
      />
    </LocaleProvider>,
  );
}

const view = (configured: string, active: string): OfficeRunModeViewB => ({
  configured,
  active,
  hide: ["cost_project", "cost_project_apply", "cost_audit", "longdoc_manifest", "longdoc_build"],
  extras: [],
});

afterEach(cleanup);

describe("ComposerToolbar 运行模式（会话定档）", () => {
  it("新会话（解锁）：独立模式钮三态可选，选档回调收目标档", () => {
    const onSetRunMode = vi.fn();
    renderT({ runMode: view("standard", "standard"), onSetRunMode });
    // 模式独立钮：触发面显当前档（标准），弹层三态
    const modeBtn = screen.getByRole("button", { name: "运行模式" });
    expect(modeBtn.textContent).toContain("标准");
    fireEvent.click(modeBtn);
    expect(screen.getByTestId("composer-runmode-menu")).toBeTruthy();
    for (const label of ["标准", "造价", "方案"]) {
      expect(screen.getAllByText(label).length).toBeGreaterThan(0);
    }
    fireEvent.click(screen.getAllByText("方案")[0].closest("button") as HTMLButtonElement);
    expect(onSetRunMode).toHaveBeenCalledWith("proposal");
  });

  it("运行后（锁定）：弹层只显当前档徽标，不出现三态选择", () => {
    renderT({ runMode: view("cost", "cost"), runModeLocked: true });
    fireEvent.click(screen.getByRole("button", { name: "运行模式" }));
    expect(screen.getByText("造价档")).toBeTruthy();
    expect(screen.getByText(/会话已定档/)).toBeTruthy();
    expect(screen.queryByText("方案")).toBeNull();
  });

  it("锁定态非标准档触发面显档名；无「新会话生效」后缀（选档即定档）", () => {
    renderT({ runMode: view("proposal", "proposal"), runModeLocked: true });
    expect(screen.getAllByText(/方案/).length).toBeGreaterThan(0);
    expect(screen.queryByText(/新会话生效/)).toBeNull();
  });
});
