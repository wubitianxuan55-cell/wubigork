import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ScheduleApplyRollback } from "./ScheduleApplyRollback";
import { ToastProvider } from "./Toast";
import type { JournalChangeRecord } from "../lib/types";

// 桥接 mock：回滚链路（GaeaJournalList → RollbackRecord）是本组件唯一下行绑定
// （与 ChangesPanel.test 同款范式；diff 确认闭环刀A v4.146）。
const mocks = vi.hoisted(() => ({
  journalList: vi.fn(async (_limit: number): Promise<JournalChangeRecord[]> => []),
  rollback: vi.fn(async (_id: string) => {}),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    GaeaJournalList: (limit: number) => mocks.journalList(limit),
    RollbackRecord: (id: string) => mocks.rollback(id),
  },
  onEvent: () => () => {},
  onReady: (cb: () => void) => {
    cb();
    return () => {};
  },
}));

function rec(partial: Partial<JournalChangeRecord>): JournalChangeRecord {
  return {
    id: "rec-1", sessionId: "s", space: "work", turn: 3,
    tool: "schedule_apply", target: "进度计划/当前计划.gsched.json",
    beforeSummary: "", afterSummary: "", baselinePath: "/ws/.journal/b1.json",
    ...partial,
  } as JournalChangeRecord;
}

function renderRow(args: unknown) {
  return render(
    <ToastProvider>
      <ScheduleApplyRollback args={JSON.stringify(args)} />
    </ToastProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ScheduleApplyRollback（schedule_apply 回执卡回滚行，v4.146 刀A）", () => {
  it("Journal 有带基线的 schedule_apply 记录 → 常驻回滚按钮，点击调 RollbackRecord", async () => {
    mocks.journalList.mockResolvedValue([
      rec({ id: "rec-old", turn: 1, target: "进度计划/办公楼二期.gsched.json" }),
      rec({ id: "rec-1", turn: 3 }),
    ]);
    renderRow({ ops: [] }); // 缺省 path → 当前计划
    const btn = await screen.findByTestId("sched-apply-rollback");
    await waitFor(() => expect(mocks.journalList).toHaveBeenCalled());
    expect(btn.textContent).toContain("回滚本次");
    fireEvent.click(btn);
    await waitFor(() => expect(mocks.rollback).toHaveBeenCalledWith("rec-1"));
    expect(screen.getByText(/已回滚 进度计划\/当前计划.gsched.json/)).toBeTruthy();
  });

  it("取该路径最新一条记录；无基线/无匹配 → 整行不渲染", async () => {
    mocks.journalList.mockResolvedValue([
      rec({ id: "rec-old", turn: 1 }),
      rec({ id: "rec-new", turn: 7 }),
      rec({ id: "rec-nobase", turn: 9, baselinePath: "" }),
    ]);
    const { container } = renderRow({ path: "进度计划/当前计划.gsched.json" });
    const btn = await screen.findByTestId("sched-apply-rollback");
    await waitFor(() => fireEvent.click(btn));
    await waitFor(() => expect(mocks.rollback).toHaveBeenCalledWith("rec-new"));
    expect(mocks.rollback).not.toHaveBeenCalledWith("rec-old");

    mocks.journalList.mockResolvedValue([rec({ id: "other", target: "进度计划/别的工程.gsched.json" })]);
    const r2 = renderRow({ path: "进度计划/当前计划.gsched.json" });
    await waitFor(() => expect(mocks.journalList).toHaveBeenCalledTimes(2));
    expect(r2.container.querySelector('[data-testid="sched-apply-rollback"]')).toBeNull();
    expect(container).toBeTruthy();
  });
});
