import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { createRef } from "react";
import { LocaleProvider } from "../../lib/i18n";
import { ComposerToolbar, type ComposerToolbarProps } from "./ComposerToolbar";

function renderToolbar(props: Partial<ComposerToolbarProps> = {}) {
  const base: ComposerToolbarProps = {
    workspaceName: "工作区",
    workspaceMenuOpen: false,
    onToggleWorkspaceMenu: () => {},
    workspaceAnchorRef: createRef<HTMLDivElement>(),
    running: false,
    pendingPaste: 0,
    captureBusy: false,
    onPickFiles: () => {},
    onScreenshot: () => {},
  };
  return render(<LocaleProvider><ComposerToolbar {...base} {...props} /></LocaleProvider>);
}

describe("ComposerToolbar 会话模式选择器", () => {
  it("默认/方案两档渲染，当前档高亮", () => {
    renderToolbar({ sessionMode: "plan" });
    const plan = screen.getByRole("button", { name: "方案" });
    expect(plan.getAttribute("aria-pressed")).toBe("true");
    const def = screen.getByRole("button", { name: "默认" });
    expect(def.getAttribute("aria-pressed")).toBe("false");
  });

  it("点击切换调用 onSetSessionMode", () => {
    const onSetPlan = vi.fn();
    const first = renderToolbar({ sessionMode: "default", onSetSessionMode: onSetPlan });
    fireEvent.click(screen.getByRole("button", { name: "方案" }));
    expect(onSetPlan).toHaveBeenCalledWith("plan");
    first.unmount();

    const onSetDefault = vi.fn();
    renderToolbar({ sessionMode: "plan", onSetSessionMode: onSetDefault });
    fireEvent.click(screen.getByRole("button", { name: "默认" }));
    expect(onSetDefault).toHaveBeenCalledWith("default");
  });

  it("当前档重复点击不重复回调", () => {
    const onSet = vi.fn();
    renderToolbar({ sessionMode: "plan", onSetSessionMode: onSet });
    fireEvent.click(screen.getByRole("button", { name: "方案" }));
    expect(onSet).not.toHaveBeenCalled();
  });
});
