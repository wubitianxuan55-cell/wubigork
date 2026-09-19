// ResizableDrawer 可访问性收口（v4.353）：手写弹层此前无 role=dialog、无 Esc、
// 开不聚焦关不还焦——对照 ApprovalModal 先例补齐。Esc 在输入控件内不劫持（可
// 编辑目标除外，与 ApprovalModal 同纪律）。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { ResizableDrawer } from "./ResizableDrawer";

vi.mock("../lib/i18n", () => ({
  useT: () => (k: string) => k,
}));

describe("ResizableDrawer 可访问性（v4.353）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("Esc 关闭：document keydown 触发 onClose（经退出动画延迟）", () => {
    vi.useFakeTimers();
    try {
      const onClose = vi.fn();
      render(<ResizableDrawer onClose={onClose} label="测试面板"><div>内容</div></ResizableDrawer>);
      fireEvent.keyDown(document, { key: "Escape" });
      vi.advanceTimersByTime(200);
      expect(onClose).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("普通按键不触发关闭", () => {
    const onClose = vi.fn();
    render(<ResizableDrawer onClose={onClose} label="测试面板"><div>内容</div></ResizableDrawer>);
    fireEvent.keyDown(document, { key: "Enter" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("dialog 语义：role=dialog + aria-modal + aria-label", () => {
    const { container } = render(
      <ResizableDrawer onClose={vi.fn()} label="历史"><div>内容</div></ResizableDrawer>,
    );
    const dialog = container.querySelector('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog?.getAttribute("aria-modal")).toBe("true");
    expect(dialog?.getAttribute("aria-label")).toBe("历史");
  });

  it("打开即聚焦抽屉容器；卸载后焦点还原到打开者", () => {
    const { unmount } = render(<ResizableDrawer onClose={vi.fn()} label="测试面板"><div>内容</div></ResizableDrawer>);
    const dialog = document.querySelector<HTMLElement>('[role="dialog"]');
    expect(document.activeElement).toBe(dialog);
    // 记录还原目标：打开前聚焦 body（render 前 activeElement=body），卸载后应还原
    unmount();
    expect(document.activeElement).toBe(document.body);
  });
});
