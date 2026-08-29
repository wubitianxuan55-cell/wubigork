import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { ComposerQueueList } from "./ComposerQueueList";

describe("ComposerQueueList 排队列表", () => {
  const queue = ["第一问：整理季度数据", "第二问：生成成本测算表"];

  it("渲染排队项与计数，点击撤回编辑回调对应项", () => {
    const onEdit = vi.fn();
    const onCancel = vi.fn();
    render(<ComposerQueueList queueDisplay={queue} onEditItem={onEdit} onCancelItem={onCancel} />);
    expect(screen.getByText(/排队中 \(2\)/)).toBeTruthy();
    expect(screen.getByText("第一问：整理季度数据")).toBeTruthy();
    expect(screen.getByText("第二问：生成成本测算表")).toBeTruthy();

    // 整行可点 → 撤回编辑（第一项）
    fireEvent.click(screen.getByText("第一问：整理季度数据"));
    expect(onEdit).toHaveBeenCalledWith(0);
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("点击 X 取消对应排队项", () => {
    const onEdit = vi.fn();
    const onCancel = vi.fn();
    render(<ComposerQueueList queueDisplay={queue} onEditItem={onEdit} onCancelItem={onCancel} />);
    const cancelButtons = screen.getAllByTitle("取消排队");
    expect(cancelButtons).toHaveLength(2);
    fireEvent.click(cancelButtons[1]);
    expect(onCancel).toHaveBeenCalledWith(1);
    expect(onEdit).not.toHaveBeenCalled();
  });

  it("空队列不渲染", () => {
    const { container } = render(
      <ComposerQueueList queueDisplay={[]} onEditItem={() => {}} onCancelItem={() => {}} />,
    );
    expect(container.childNodes).toHaveLength(0);
  });
});
