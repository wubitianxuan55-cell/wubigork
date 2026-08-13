import { describe, it, expect } from "vitest";
import { applyEvent, reducer, initialState, rebuildHistoryItems } from "./store";
import type { ControllerState, Item } from "./store";

function base(overrides: Partial<ControllerState> = {}): ControllerState {
  return { ...initialState, seq: 1, ...overrides };
}

const todoArgs = JSON.stringify({
  todos: [
    { content: "A", status: "pending" },
    { content: "B", status: "pending" },
  ],
});

describe("applyEvent", () => {
  it("turn_started 重置本轮状态", () => {
    const s = applyEvent(base({ running: false, turnActive: false }), { kind: "turn_started" });
    expect(s.running).toBe(true);
    expect(s.turnActive).toBe(true);
    expect(s.lastAssistantIdx).toBe(-1);
    expect(s.turnTokens).toBe(0);
  });

  it("text 流式追加到当前 assistant", () => {
    const s0 = base({
      items: [{ kind: "assistant", id: "a0", text: "你好", reasoning: "", streaming: true }],
      lastAssistantIdx: 0,
    });
    const s = applyEvent(s0, { kind: "text", text: "世界" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.text).toBe("你好世界");
    expect(it.streaming).toBe(true);
  });

  it("跨轮 text 不覆盖上一轮已终结的 assistant", () => {
    const s0 = base({
      items: [{ kind: "assistant", id: "a0", text: "上一轮", reasoning: "", streaming: false }],
      lastAssistantIdx: 0,
      turnActive: true,
    });
    const s = applyEvent(s0, { kind: "text", text: "这一轮" });
    expect(s.items.length).toBe(2);
    expect((s.items[0] as Extract<Item, { kind: "assistant" }>).text).toBe("上一轮");
    expect((s.items[1] as Extract<Item, { kind: "assistant" }>).text).toBe("这一轮");
  });

  it("message 将流式 assistant 收尾", () => {
    const s0 = base({
      items: [{ kind: "assistant", id: "a0", text: "正文", reasoning: "", streaming: true }],
      lastAssistantIdx: 0,
    });
    const s = applyEvent(s0, { kind: "message", text: "最终正文" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.text).toBe("最终正文");
    expect(it.streaming).toBe(false);
  });

  it("tool_dispatch + tool_result 完成工具卡片", () => {
    let s = applyEvent(base(), { kind: "tool_dispatch", tool: { name: "edit_file", id: "c1", args: '{"path":"a.md"}', readOnly: false } });
    expect((s.items[0] as Extract<Item, { kind: "tool" }>).status).toBe("running");
    s = applyEvent(s, { kind: "tool_result", tool: { name: "edit_file", id: "c1", output: "done", readOnly: false } });
    const it = s.items[0] as Extract<Item, { kind: "tool" }>;
    expect(it.status).toBe("done");
    expect(it.output).toBe("done");
  });

  it("turn_done 将待办收尾并停止运行中的工具", () => {
    const s0 = base({
      running: true,
      turnActive: true,
      items: [{ kind: "tool", id: "t1", name: "todo_write", args: todoArgs, readOnly: false, status: "running" }],
    });
    const s = applyEvent(s0, { kind: "turn_done" });
    const it = s.items[0] as Extract<Item, { kind: "tool" }>;
    expect(it.status).toBe("stopped");
    const todos = JSON.parse(it.args).todos as { status: string }[];
    expect(todos.every((t) => t.status === "completed")).toBe(true);
  });
});

describe("reducer history", () => {
  it("恢复历史时重建工具卡并收尾未推进的待办", () => {
    const messages = [
      { role: "user", content: "开始" },
      { role: "tool", toolName: "todo_write", toolId: "c1", toolArgs: todoArgs, content: "" },
      { role: "tool_result", toolId: "c1", toolName: "todo_write", content: "ok" },
    ];
    const s = reducer(base(), { type: "history", messages });
    const tool = s.items.find((it) => it.kind === "tool") as Extract<Item, { kind: "tool" }>;
    expect(tool).toBeDefined();
    const todos = JSON.parse(tool.args).todos as { status: string }[];
    expect(todos.every((t) => t.status === "completed")).toBe(true);
    expect(s.lastAssistantIdx).toBe(-1);
  });

  it("rebuildHistoryItems 计算最后一个 assistant 索引", () => {
    const messages = [
      { role: "user", content: "hi" },
      { role: "assistant", content: "answer" },
    ];
    const { items, lastAssistantIdx } = rebuildHistoryItems(messages);
    expect(items.filter((it) => it.kind === "assistant")).toHaveLength(1);
    expect(lastAssistantIdx).toBe(1);
  });
});
