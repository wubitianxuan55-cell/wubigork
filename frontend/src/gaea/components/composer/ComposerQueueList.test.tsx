import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { ComposerQueueList } from "./ComposerQueueList";
import { LocaleProvider } from "../../lib/i18n";
import type { ComposerQueueItem } from "./composerQueue";

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const item = (text: string, status: ComposerQueueItem["status"] = "pending", id = text): ComposerQueueItem => ({
  id, text, status,
});

describe("ComposerQueueList 排队列表", () => {
  const queue = [item("第一问：整理季度数据", "pending", "a"), item("第二问：生成成本测算表", "pending", "b")];

  it("渲染排队项与计数，点击撤回编辑回调对应项", () => {
    const onEdit = vi.fn();
    const onCancel = vi.fn();
    renderT(
      <ComposerQueueList
        items={queue}
        running
        onEditItem={onEdit}
        onCancelItem={onCancel}
        onSteerItem={() => {}}
        onSteerAll={() => {}}
        onCancelAll={() => {}}
        onReorder={() => {}}
      />,
    );
    expect(screen.getByText(/排队中 \(2\)/)).toBeTruthy();
    expect(screen.getByText("第一问：整理季度数据")).toBeTruthy();
    expect(screen.getByText("第二问：生成成本测算表")).toBeTruthy();

    fireEvent.click(screen.getByText("第一问：整理季度数据"));
    expect(onEdit).toHaveBeenCalledWith(0);
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("点击 X 取消对应排队项", () => {
    const onEdit = vi.fn();
    const onCancel = vi.fn();
    renderT(
      <ComposerQueueList
        items={queue}
        running
        onEditItem={onEdit}
        onCancelItem={onCancel}
        onSteerItem={() => {}}
        onSteerAll={() => {}}
        onCancelAll={() => {}}
        onReorder={() => {}}
      />,
    );
    const cancelButtons = screen.getAllByTitle("取消排队");
    expect(cancelButtons).toHaveLength(2);
    fireEvent.click(cancelButtons[1]);
    expect(onCancel).toHaveBeenCalledWith(1);
    expect(onEdit).not.toHaveBeenCalled();
  });

  it("空队列不渲染", () => {
    const { container } = renderT(
      <ComposerQueueList
        items={[]}
        running
        onEditItem={() => {}}
        onCancelItem={() => {}}
        onSteerItem={() => {}}
        onSteerAll={() => {}}
        onCancelAll={() => {}}
        onReorder={() => {}}
      />,
    );
    expect(container.childNodes).toHaveLength(0);
  });

  it("单条插话 / 全部插话 / 全部取消", () => {
    const onSteer = vi.fn();
    const onSteerAll = vi.fn();
    const onCancelAll = vi.fn();
    renderT(
      <ComposerQueueList
        items={queue}
        running
        onEditItem={() => {}}
        onCancelItem={() => {}}
        onSteerItem={onSteer}
        onSteerAll={onSteerAll}
        onCancelAll={onCancelAll}
        onReorder={() => {}}
      />,
    );
    fireEvent.click(screen.getByTestId("composer-queue-steer-0"));
    expect(onSteer).toHaveBeenCalledWith(0);
    fireEvent.click(screen.getByTestId("composer-queue-steer-all"));
    expect(onSteerAll).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId("composer-queue-cancel-all"));
    expect(onCancelAll).toHaveBeenCalledTimes(1);
  });

  it("发送中显示徽标，编辑/删除/插话禁用", () => {
    const onEdit = vi.fn();
    const onCancel = vi.fn();
    const onSteer = vi.fn();
    const onSteerAll = vi.fn();
    const onCancelAll = vi.fn();
    renderT(
      <ComposerQueueList
        items={[item("正在发出的问题", "sending", "s"), item("下一条", "pending", "p")]}
        running
        onEditItem={onEdit}
        onCancelItem={onCancel}
        onSteerItem={onSteer}
        onSteerAll={onSteerAll}
        onCancelAll={onCancelAll}
        onReorder={() => {}}
      />,
    );
    expect(screen.getByTestId("composer-queue-sending").textContent).toContain("发送中");
    expect(screen.getByTestId("composer-queue-steer-all")).toHaveProperty("disabled", true);
    expect(screen.getByTestId("composer-queue-cancel-all")).toHaveProperty("disabled", true);

    fireEvent.click(screen.getByText("正在发出的问题"));
    fireEvent.click(screen.getByText("下一条"));
    fireEvent.click(screen.getByTestId("composer-queue-steer-1"));
    fireEvent.click(screen.getByTestId("composer-queue-steer-all"));
    fireEvent.click(screen.getByTestId("composer-queue-cancel-all"));
    expect(onEdit).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
    expect(onSteer).not.toHaveBeenCalled();
    expect(onSteerAll).not.toHaveBeenCalled();
    expect(onCancelAll).not.toHaveBeenCalled();
  });

  it("拖拽 pending 项回调 onReorder；点握把不触发编辑", () => {
    const onReorder = vi.fn();
    const onEdit = vi.fn();
    const three = [
      item("甲", "pending", "a"),
      item("乙", "pending", "b"),
      item("丙", "pending", "c"),
    ];
    renderT(
      <ComposerQueueList
        items={three}
        running
        onEditItem={onEdit}
        onCancelItem={() => {}}
        onSteerItem={() => {}}
        onSteerAll={() => {}}
        onCancelAll={() => {}}
        onReorder={onReorder}
      />,
    );
    const dt = { setData: vi.fn(), getData: vi.fn(), effectAllowed: "move", dropEffect: "move" };
    fireEvent.click(screen.getByTestId("composer-queue-grip-2"));
    expect(onEdit).not.toHaveBeenCalled();
    fireEvent.dragStart(screen.getByTestId("composer-queue-grip-2"), { dataTransfer: dt });
    fireEvent.dragOver(screen.getByTestId("composer-queue-item-0"), { dataTransfer: dt });
    fireEvent.drop(screen.getByTestId("composer-queue-item-0"), { dataTransfer: dt });
    expect(onReorder).toHaveBeenCalledWith(2, 0);
  });

  it("sending 项没有握把，不可拖出", () => {
    renderT(
      <ComposerQueueList
        items={[item("正在发出的问题", "sending", "s"), item("下一条", "pending", "p")]}
        running
        onEditItem={() => {}}
        onCancelItem={() => {}}
        onSteerItem={() => {}}
        onSteerAll={() => {}}
        onCancelAll={() => {}}
        onReorder={() => {}}
      />,
    );
    expect(screen.queryByTestId("composer-queue-grip-0")).toBeNull();
    expect(screen.getByTestId("composer-queue-grip-1")).toBeTruthy();
  });
});
