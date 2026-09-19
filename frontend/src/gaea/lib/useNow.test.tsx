// useNow(active) 条件订阅（v4.350）：此前所有已完成消息/过程卡无条件订阅全局
// 1s 时钟——长会话数百条消息每秒全体重渲染，O(N) 常驻开销随会话长度线性涨。
// active=false 时不订阅（返回冻结值）；全部订阅者转 false 后全局定时器停摆。
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useNow } from "./useNow";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("useNow(active)", () => {
  it("active=true：每秒刷新 Unix 秒值", () => {
    const start = 1_700_000_000;
    vi.setSystemTime(start * 1000);
    const { result } = renderHook(() => useNow(true));
    expect(result.current).toBe(start);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current).toBe(start + 1);
  });

  it("active=false：不随墙钟变化（冻结在订阅前的值）", () => {
    const start = 1_700_000_000;
    vi.setSystemTime(start * 1000);
    const { result, rerender } = renderHook(({ active }) => useNow(active), {
      initialProps: { active: true },
    });
    expect(result.current).toBe(start);
    rerender({ active: false });
    const frozen = result.current;
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(result.current).toBe(frozen);
  });

  it("active true→false→true：恢复订阅后继续走时", () => {
    const start = 1_700_000_000;
    vi.setSystemTime(start * 1000);
    const { result, rerender } = renderHook(({ active }) => useNow(active), {
      initialProps: { active: true },
    });
    rerender({ active: false });
    act(() => {
      vi.advanceTimersByTime(3000);
    });
    rerender({ active: true });
    const resumed = result.current;
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current).toBeGreaterThanOrEqual(resumed);
  });
});
