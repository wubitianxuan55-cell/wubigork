import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useDebouncedValue } from "./useDebouncedValue";

describe("useDebouncedValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("初始值立即返回，延迟结束后才更新为新值", () => {
    const { result, rerender } = renderHook(({ v }) => useDebouncedValue(v, 250), {
      initialProps: { v: "a" },
    });
    expect(result.current).toBe("a");

    rerender({ v: "b" });
    // 未到 delay：仍是旧值
    expect(result.current).toBe("a");
    act(() => vi.advanceTimersByTime(249));
    expect(result.current).toBe("a");
    act(() => vi.advanceTimersByTime(1));
    expect(result.current).toBe("b");
  });

  it("快速连续输入只取最后一次", () => {
    const { result, rerender } = renderHook(({ v }) => useDebouncedValue(v, 250), {
      initialProps: { v: "" },
    });
    rerender({ v: "h" });
    act(() => vi.advanceTimersByTime(100));
    rerender({ v: "he" });
    act(() => vi.advanceTimersByTime(100));
    rerender({ v: "hel" });
    act(() => vi.advanceTimersByTime(100));
    // 每次输入都重置了 timer，250ms 内最后一次输入前不会更新
    expect(result.current).toBe("");
    act(() => vi.advanceTimersByTime(150));
    expect(result.current).toBe("hel");
  });

  it("卸载时清理未触发的 timer", () => {
    const clearSpy = vi.spyOn(globalThis, "clearTimeout");
    const { rerender, unmount } = renderHook(({ v }) => useDebouncedValue(v, 250), {
      initialProps: { v: "" },
    });
    rerender({ v: "pending" });
    unmount();
    expect(clearSpy).toHaveBeenCalled();
    clearSpy.mockRestore();
  });

  it("清空输入（空串）即时同步，不等待 delay", () => {
    const { result, rerender } = renderHook(({ v }) => useDebouncedValue(v, 250), {
      initialProps: { v: "abc" },
    });
    act(() => vi.advanceTimersByTime(250));
    expect(result.current).toBe("abc");

    rerender({ v: "" });
    // 空串不经过 delay，立即生效
    expect(result.current).toBe("");
  });
});
