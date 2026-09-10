// 发送队列：入队、回合结束后派出并显示「发送中」、期间锁编辑/删除/Steer。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { Composer } from "./Composer";
import { LocaleProvider } from "../lib/i18n";
import { ToastProvider } from "./Toast";

const appStub = vi.hoisted(() => new Proxy({}, { get: () => vi.fn().mockResolvedValue(null) }));

vi.mock("../lib/bridge", () => ({
  app: appStub,
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
  onReady: vi.fn(() => () => {}),
}));

function renderQueue(running: boolean, onSend = vi.fn(), onSteer = vi.fn(), onCancel = vi.fn(() => undefined)) {
  localStorage.setItem("gaea-lang", "zh");
  const ui = (run: boolean) => (
    <LocaleProvider>
      <ToastProvider>
        <Composer
          running={run}
          cwd="/ws"
          onSend={onSend}
          onSteer={onSteer}
          onCancel={onCancel}
          onPickFolder={async () => "/ws"}
        />
      </ToastProvider>
    </LocaleProvider>
  );
  const view = render(ui(running));
  return { ...view, onSend, onSteer, onCancel, rerun: (run: boolean) => view.rerender(ui(run)) };
}

describe("Composer 发送队列", () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { cleanup(); vi.useRealTimers(); });

  it("运行中入队；回合结束后显示发送中并派出；期间不可编辑/插话/取消", () => {
    const { onSend, onSteer, rerun } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "排队的问题" } });
    fireEvent.click(screen.getByTitle(/排队发送（当前回合结束后执行/));
    expect(screen.getByTestId("composer-queue")).toBeTruthy();
    expect(screen.getByText("排队的问题")).toBeTruthy();

    rerun(false);
    expect(screen.getByTestId("composer-queue-sending").textContent).toContain("发送中");
    fireEvent.click(screen.getByText("排队的问题"));
    expect(onSteer).not.toHaveBeenCalled();
    expect(screen.getByTestId("composer-queue-cancel-all")).toHaveProperty("disabled", true);

    act(() => { vi.advanceTimersByTime(50); });
    expect(onSend).toHaveBeenCalledWith("排队的问题", "排队的问题");
  });

  it("单条插话注入当前回合并从队列移除，不走 onSend", () => {
    const { onSend, onSteer } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "插这一条" } });
    fireEvent.click(screen.getByTitle(/排队发送（当前回合结束后执行/));
    fireEvent.click(screen.getByTestId("composer-queue-steer-0"));
    expect(onSteer).toHaveBeenCalledWith("插这一条");
    expect(onSend).not.toHaveBeenCalled();
    expect(screen.queryByTestId("composer-queue")).toBeNull();
  });

  it("全部取消只清队列，不调用 onCancel", () => {
    const { onCancel } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "不要了" } });
    fireEvent.click(screen.getByTitle(/排队发送（当前回合结束后执行/));
    fireEvent.click(screen.getByTestId("composer-queue-cancel-all"));
    expect(screen.queryByTestId("composer-queue")).toBeNull();
    expect(onCancel).not.toHaveBeenCalled();
  });
});
