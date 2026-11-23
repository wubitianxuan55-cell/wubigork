import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useComposerMenus } from "./useComposerMenus";

const bridgeMocks = vi.hoisted(() => ({
  Commands: vi.fn(),
  SlashArgs: vi.fn(),
  FileSearch: vi.fn(),
  ListDir: vi.fn(),
  GaeaListDirBatch: vi.fn(),
}));

vi.mock("../lib/bridge", () => ({ app: bridgeMocks }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("useComposerMenus 异步查询", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    bridgeMocks.Commands.mockResolvedValue([]);
    bridgeMocks.FileSearch.mockResolvedValue([]);
    bridgeMocks.ListDir.mockResolvedValue([]);
    bridgeMocks.GaeaListDirBatch.mockResolvedValue([]);
  });

  it("等待期给 waiting 状态，过期参数结果不能覆盖当前参数", async () => {
    const first = deferred<{ items: { label: string; insert: string }[]; from: number }>();
    const second = deferred<{ items: { label: string; insert: string }[]; from: number }>();
    bridgeMocks.SlashArgs
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);

    const { result, rerender } = renderHook(
      ({ text }) => useComposerMenus({
        text,
        debouncedText: text,
        setTextCaretEnd: vi.fn(),
      }),
      { initialProps: { text: "/plan " } },
    );

    await waitFor(() => expect(result.current.menuWaiting).toBe(true));

    rerender({ text: "/goal " });
    await waitFor(() => expect(bridgeMocks.SlashArgs).toHaveBeenCalledTimes(2));

    await act(async () => {
      first.resolve({ items: [{ label: "旧结果", insert: "old" }], from: 0 });
      await first.promise;
    });
    expect(result.current.argRes).toBeNull();
    expect(result.current.menuWaiting).toBe(true);

    await act(async () => {
      second.resolve({ items: [{ label: "新结果", insert: "new" }], from: 0 });
      await second.promise;
    });
    await waitFor(() => expect(result.current.argRes?.items[0].label).toBe("新结果"));
    expect(result.current.menuWaiting).toBe(false);
  });

  it("命令过滤在数据入口生效", async () => {
    bridgeMocks.Commands.mockResolvedValue([
      { name: "plan", kind: "builtin", description: "plan" },
      { name: "memory_search", kind: "builtin", description: "memory_search" },
    ]);

    const { result } = renderHook(() => useComposerMenus({
      text: "/",
      debouncedText: "/",
      setTextCaretEnd: vi.fn(),
      filterCommand: (name) => name !== "memory_search",
    }));

    await waitFor(() => expect(result.current.slashMatches.map((c) => c.name)).toEqual(["plan"]));
  });
});
