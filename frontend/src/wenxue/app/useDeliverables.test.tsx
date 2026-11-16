import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { useDeliverables } from "./useDeliverables";
import type { ControllerState, Item } from "../lib/store";

/**
 * useDeliverables.test — 交付登记去重主战场（2026-11-17 审查批15 补 hook 级
 * 锁；0b1f4c62 根修=键恒归一化绝对形态，批10 回种 gaea 前先在此钉死 hook 层）。
 * 「4 个相同交互文件」事故的原始形态：同一文件以相对/绝对/./ 前缀/反斜杠
 * 等写法出现在工具参数与正文提及里，必须收敛为一条且 versions 累加。
 */
const cwd = "C:/book";
const tool = (id: string, path: string): Item => ({
  kind: "tool", id, name: "write_file",
  args: JSON.stringify({ path }), readOnly: false, status: "done",
});
const say = (id: string, text: string): Item => ({
  kind: "assistant", id, text, reasoning: "", streaming: false,
});

function buildState(items: Item[]): ControllerState {
  return { items, meta: { cwd } } as unknown as ControllerState;
}

describe("useDeliverables 交付登记去重（hook 级）", () => {
  it("同一文件的四种写法收敛一条且 versions 累加（工具参数+正文提及混合源）", () => {
    const items = [
      tool("t1", "交互文件/大纲.md"),
      say("a1", "已写入 C:\\book\\交互文件\\大纲.md，请过目。"),
      tool("t2", "C:/book/交互文件/大纲.md"),
      tool("t3", "./交互文件/大纲.md"),
    ];
    const { result } = renderHook(() =>
      useDeliverables({ state: buildState(items), currentSessionKey: "s1", rightTab: null, openPaneView: () => {} }),
    );
    const d = result.current.sessionDeliverables;
    expect(d).toHaveLength(1);
    expect(d[0].versions).toBe(4);
    expect(d[0].path).toBe("交互文件/大纲.md"); // 首次原始写法
  });

  it("不同文件不误合并；sessionChanges 同键归一", () => {
    const items = [
      tool("t1", "交互文件/大纲.md"),
      tool("t2", "交互文件\\细纲.md"),
      tool("t3", "C:/book/交互文件/大纲.md"),
    ];
    const { result } = renderHook(() =>
      useDeliverables({ state: buildState(items), currentSessionKey: "s1", rightTab: null, openPaneView: () => {} }),
    );
    expect(result.current.sessionDeliverables).toHaveLength(2);
    const changes = result.current.sessionChanges;
    const dagang = changes.find((c) => c.path.includes("大纲"));
    expect(dagang?.count).toBe(2); // 相对+绝对同键合并
    expect(changes.find((c) => c.path.includes("细纲"))).toBeTruthy();
  });
});
