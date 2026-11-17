import { describe, expect, it } from "vitest";
import { canBacktrack } from "./backtrack";
import type { Item } from "./store/controller_state";

function user(id: string, text: string): Item {
  return { kind: "user", id, text };
}

function assistant(id: string, text: string): Item {
  return { kind: "assistant", id, text, reasoning: "", streaming: false };
}

describe("canBacktrack", () => {
  it("完整历史中的正常回合可回退并带回原提示词", () => {
    const result = canBacktrack({
      items: [user("u1", "第一轮"), assistant("a1", "回答一"), user("u2", "第二轮"), assistant("a2", "回答二")],
      turn: 1,
      running: false,
      turnActive: false,
    });
    expect(result).toEqual({ ok: true, prompt: "第二轮", userId: "u2" });
  });

  it("在飞回合或待发送回合拒绝回退", () => {
    const items = [user("u1", "正在执行")];
    expect(canBacktrack({ items, turn: 0, running: true })).toMatchObject({ ok: false, reason: "in-flight" });
    expect(canBacktrack({ items, turn: 0, turnActive: true })).toMatchObject({ ok: false, reason: "in-flight" });
    expect(canBacktrack({ items, turn: 0, pendingUser: "排队" })).toMatchObject({ ok: false, reason: "in-flight" });
  });

  it("缺失、空提示、隐藏项与半截历史均拒绝", () => {
    const items = [user("u1", "第一轮"), user("u2", " "), user("u3", "第三轮")];
    expect(canBacktrack({ items, turn: 9 })).toMatchObject({ ok: false, reason: "missing-prompt" });
    expect(canBacktrack({ items, turn: 1 })).toMatchObject({ ok: false, reason: "missing-prompt" });
    expect(canBacktrack({ items, turn: 2, hiddenUserIds: new Set(["u3"]) })).toMatchObject({ ok: false, reason: "hidden-prompt" });
    expect(canBacktrack({ items, turn: 2, historyComplete: false })).toMatchObject({ ok: false, reason: "partial-history" });
  });
});
