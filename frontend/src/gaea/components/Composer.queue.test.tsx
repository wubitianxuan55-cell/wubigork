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

  it("拖拽握把调整发送顺序", () => {
    const { onSend } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "先发这条" } });
    fireEvent.click(screen.getByTitle(/排队发送（当前回合结束后执行/));
    fireEvent.change(screen.getByPlaceholderText(/排队中/), { target: { value: "后发这条" } });
    fireEvent.click(screen.getByTitle(/排队发送（当前回合结束后执行/));
    expect(screen.getByTestId("composer-queue-item-0").textContent).toContain("先发这条");
    expect(screen.getByTestId("composer-queue-item-1").textContent).toContain("后发这条");
    const dt = { setData: vi.fn(), getData: vi.fn(), effectAllowed: "move", dropEffect: "move" };
    fireEvent.dragStart(screen.getByTestId("composer-queue-grip-1"), { dataTransfer: dt });
    fireEvent.dragOver(screen.getByTestId("composer-queue-item-0"), { dataTransfer: dt });
    fireEvent.drop(screen.getByTestId("composer-queue-item-0"), { dataTransfer: dt });
    expect(screen.getByTestId("composer-queue-item-0").textContent).toContain("后发这条");
    expect(screen.getByTestId("composer-queue-item-1").textContent).toContain("先发这条");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("运行中 Enter 默认排队（2026-09-17 对齐 DSH/Codex），不直插当前回合", () => {
    const { onSend, onSteer } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "Enter 排队" } });
    fireEvent.keyDown(ta, { key: "Enter" });
    expect(screen.getByTestId("composer-queue")).toBeTruthy();
    expect(screen.getByText("Enter 排队")).toBeTruthy();
    expect(onSteer).not.toHaveBeenCalled();
    expect(onSend).not.toHaveBeenCalled();
  });

  it("运行中 Alt+Enter 直插当前回合（显式插话，不入队）", () => {
    const { onSend, onSteer } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "插话调整" } });
    fireEvent.keyDown(ta, { key: "Enter", altKey: true });
    expect(onSteer).toHaveBeenCalledWith("插话调整");
    expect(onSend).not.toHaveBeenCalled();
    expect(screen.queryByTestId("composer-queue")).toBeNull();
  });

  it("Shift+Enter 纠正：清队列+取消当前回合，回合结束后补发纠正文本", () => {
    const { onSend, onCancel, rerun } = renderQueue(true);
    const ta = screen.getByPlaceholderText(/任务执行中/);
    fireEvent.change(ta, { target: { value: "纠正后的文本" } });
    fireEvent.keyDown(ta, { key: "Enter", shiftKey: true });
    expect(onCancel).toHaveBeenCalled();
    expect(screen.queryByTestId("composer-queue")).toBeNull();
    rerun(false);
    expect(onSend).toHaveBeenCalledWith("纠正后的文本", "纠正后的文本");
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
