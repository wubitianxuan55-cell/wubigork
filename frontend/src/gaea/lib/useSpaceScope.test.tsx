// useSpaceScope 单测：检索 scope 的「当前空间」来源（S1.2-C）。
// 覆盖：挂载拉取 GaeaSpaceActive + 模块缓存、SpaceChip 激活后的广播同步、
// SCOPE_OPTIONS 三档选项（工位/乐园/全部，「全部」=scope ""）。
import { describe, expect, it } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { SCOPE_OPTIONS, noteSpaceActivated, useSpaceScope } from "./useSpaceScope";

describe("useSpaceScope 检索 scope 的当前空间来源", () => {
  it("挂载后拉取 GaeaSpaceActive 并返回当前生效空间（mock 默认 work）", async () => {
    const { result } = renderHook(() => useSpaceScope());
    await waitFor(() => expect(result.current.active).not.toBeNull());
    expect(result.current.active?.space).toBe("work");
    expect(result.current.active?.modeOn).toBe(true);
  });

  it("SpaceChip 激活后经 noteSpaceActivated 广播给已挂载的检索面", async () => {
    const { result } = renderHook(() => useSpaceScope());
    await waitFor(() => expect(result.current.active).not.toBeNull());
    act(() => {
      noteSpaceActivated({
        space: "play",
        modeOn: true,
        exportsDir: ".gaea/play/exports",
        workDir: ".gaea/play/work",
      });
    });
    expect(result.current.active?.space).toBe("play");
  });

  it("SCOPE_OPTIONS 提供书房/庭院/全部三档（「全部」=scope \"\"）", () => {
    expect(SCOPE_OPTIONS.map((o) => o.value)).toEqual(["work", "play", ""]);
    expect(SCOPE_OPTIONS.map((o) => o.label)).toEqual(["书房", "庭院", "全部"]);
  });
});
