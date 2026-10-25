// ToolSheet — 工具面板壳层契约：
// 1) 打开时渲染 dialog 与标题；2) 关闭钮/遮罩/Escape 都回传 onClose；
// 3) 关闭后焦点归还触发元素（键盘用户不丢位置）。
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ToolSheet } from "./ToolSheet";

function Harness({ onClose }: { onClose: () => void }) {
  const [open, setOpen] = useState(false);
  return (
    <div>
      <button type="button" data-testid="trigger" onClick={() => setOpen(true)}>打开工具</button>
      <ToolSheet
        open={open}
        title="CAD 看图"
        subtitle="临时工具面板 · 关闭后结果留在产物"
        testId="tool-sheet-cad"
        onClose={() => { setOpen(false); onClose(); }}
      >
        <div data-testid="tool-body">图套 / 画布 / 检查器</div>
      </ToolSheet>
    </div>
  );
}

describe("ToolSheet 工具面板", () => {
  it("打开渲染 dialog、标题与临时面板说明", () => {
    render(<Harness onClose={() => {}} />);
    fireEvent.click(screen.getByTestId("trigger"));
    expect(screen.getByRole("dialog", { name: "CAD 看图" })).toBeTruthy();
    expect(screen.getByText("临时工具面板 · 关闭后结果留在产物")).toBeTruthy();
    expect(screen.getByTestId("tool-body")).toBeTruthy();
  });

  it("关闭钮 / 遮罩 / Escape 三条关闭路径", async () => {
    const onClose = vi.fn();
    render(<Harness onClose={onClose} />);
    fireEvent.click(screen.getByTestId("trigger"));
    fireEvent.click(screen.getByTestId("tool-sheet-cad-close"));
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("trigger"));
    fireEvent.click(screen.getByTestId("tool-sheet-cad-backdrop"));
    expect(onClose).toHaveBeenCalledTimes(2);

    fireEvent.click(screen.getByTestId("trigger"));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it("关闭后焦点归还触发元素", async () => {
    const onClose = vi.fn();
    render(<Harness onClose={onClose} />);
    const trigger = screen.getByTestId("trigger");
    trigger.focus();
    fireEvent.click(trigger);
    expect(screen.getByTestId("tool-sheet-cad-close")).toBeTruthy();
    fireEvent.click(screen.getByTestId("tool-sheet-cad-close"));
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });
});
