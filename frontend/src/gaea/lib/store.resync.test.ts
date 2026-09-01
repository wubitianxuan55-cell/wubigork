// v4.26「对话流式重造」前端状态线测试：
//  - reducer case "resync"（好快照替换 / 坏快照保底 / running 保持 / streaming 续接 / pendingUser 去重）
//  - message 事件透传 subagentRef（补充契约：带/不带、新建/更新、替换语义）
//  - onEvent seq 缺口 → 经注入 fetcher 触发一次补拉（集成：含无 seq 旁路、冷却、失败保底、会话切换归零）
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { applyEvent, initialState, parseResyncItems, reducer, useController, useStore } from "./store";
import { emitMock } from "./mock";
import { resetEventSync, setEventSyncFetcher } from "./eventSync";
import type { EventSyncSnapshot } from "./eventSync";
import type { ControllerState, Item } from "./store";
import type { WireEvent } from "./types";

function base(overrides: Partial<ControllerState> = {}): ControllerState {
  return { ...initialState, seq: 1, ...overrides };
}

async function flush(): Promise<void> {
  await act(async () => { await new Promise((r) => setTimeout(r, 0)); });
}

// 后端 GaeaResyncEvents 折叠快照的合法样例（六种 kind 齐全）。
const goodSnapshot: unknown[] = [
  { kind: "user", id: "h0", text: "帮我做表" },
  { kind: "assistant", id: "h1", text: "好的", reasoning: "", streaming: false },
  { kind: "tool", id: "c9", name: "read_file", args: "{}", readOnly: true, status: "done", output: "ok" },
  { kind: "notice", id: "h3", level: "warn", text: "提醒" },
  { kind: "phase", id: "h4", text: "规划" },
  { kind: "compaction", id: "h5", pending: false, trigger: "auto", messages: 3, summary: "摘要", archive: "p" },
];

describe("reducer case \"resync\"", () => {
  it("好快照整体替换 items 并重算 lastAssistantIdx / seq", () => {
    const s0 = base({ seq: 5, items: [{ kind: "notice", id: "n0", level: "info", text: "旧" }] });
    const s = reducer(s0, { type: "resync", items: goodSnapshot });
    expect(s.items.map((it) => it.kind)).toEqual(["user", "assistant", "tool", "notice", "phase", "compaction"]);
    expect(s.lastAssistantIdx).toBe(1); // 最后一个 assistant 在快照 idx 1
    expect(s.seq).toBe(11); // 5 + 6：id 计数器前移，防后续本地 id 撞快照 id
  });

  it("running/turnActive 保持：resync 只补历史，不撤销进行中的回合", () => {
    const s0 = base({ running: true, turnActive: true, items: [], lastAssistantIdx: -1 });
    const s = reducer(s0, { type: "resync", items: goodSnapshot });
    expect(s.running).toBe(true);
    expect(s.turnActive).toBe(true);
  });

  it("缺口期间本地仍在流式输出：快照最后一个 assistant 续上 streaming，后续 text 不劈新气泡", () => {
    const s0 = base({
      running: true,
      turnActive: true,
      items: [{ kind: "assistant", id: "a0", text: "正在", reasoning: "", streaming: true }],
      lastAssistantIdx: 0,
    });
    const snap = [{ kind: "assistant", id: "h0", text: "正在输出更多", reasoning: "", streaming: false }];
    const s = reducer(s0, { type: "resync", items: snap });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.streaming).toBe(true);
    const s2 = applyEvent(s, { kind: "text", text: "…" });
    const it2 = s2.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it2.text).toBe("正在输出更多…"); // 追加而非新建
    expect(s2.items).toHaveLength(1);
  });

  it("本地不在流式（无活跃回合）：快照保持非流式，不打补丁", () => {
    const s0 = base({ items: [], lastAssistantIdx: -1, turnActive: false });
    const snap = [{ kind: "assistant", id: "h0", text: "历史", reasoning: "", streaming: false }];
    const s = reducer(s0, { type: "resync", items: snap });
    expect((s.items[0] as Extract<Item, { kind: "assistant" }>).streaming).toBe(false);
  });

  it.each([
    ["非数组", "not-array"],
    ["空数组（后端读日志失败）", []],
    ["未知 kind", [{ kind: "mystery", id: "x" }]],
    ["缺 id", [{ kind: "user", text: "没有 id" }]],
    ["非法 tool status", [{ kind: "tool", id: "t", name: "n", readOnly: false, status: "weird" }]],
    ["null 元素", [null]],
  ])("坏快照静默忽略并保底：%s", (_name, raw) => {
    const s0 = base({ items: [{ kind: "notice", id: "n0", level: "info", text: "旧" }] });
    const s = reducer(s0, { type: "resync", items: raw as unknown[] });
    expect(s).toBe(s0); // 原状态原样保留（引用相等）
  });

  it("pendingUser 已进快照：只清标记，不重复追加用户气泡", () => {
    const s0 = base({ pendingUser: "帮我做表", seq: 3 });
    const s = reducer(s0, { type: "resync", items: [{ kind: "user", id: "h0", text: "帮我做表" }] });
    expect(s.pendingUser).toBeUndefined();
    expect(s.items).toHaveLength(1);
  });

  it("pendingUser 未进快照：保留标记，等后续事件 flush", () => {
    const s0 = base({ pendingUser: "帮我做表", seq: 3 });
    const s = reducer(s0, { type: "resync", items: [{ kind: "assistant", id: "h0", text: "历史", reasoning: "", streaming: false }] });
    expect(s.pendingUser).toBe("帮我做表");
  });

  it("discardTurn（unsend 未决）时不替换（快照可能含已撤回消息）", () => {
    const s0 = base({ discardTurn: true });
    expect(reducer(s0, { type: "resync", items: goodSnapshot })).toBe(s0);
  });
});

describe("parseResyncItems", () => {
  it("归一化：streaming 强制 false、subagentRef 保留、可选字段补默认", () => {
    const items = parseResyncItems([
      { kind: "assistant", id: "h0", text: "t", reasoning: "r", streaming: true, subagentRef: "sa_1" },
      { kind: "assistant", id: "h1", text: "t2", reasoning: "" },
      { kind: "compaction", id: "h2", pending: true },
      { kind: "tool", id: "c1", name: "edit_file", readOnly: false, status: "error", error: 1 },
    ]);
    expect(items).not.toBeNull();
    const a0 = items![0] as Extract<Item, { kind: "assistant" }>;
    expect(a0.streaming).toBe(false); // 磁盘折叠没有「正在输出」概念
    expect(a0.subagentRef).toBe("sa_1"); // 折叠快照不丢子代理徽标
    const a1 = items![1] as Extract<Item, { kind: "assistant" }>;
    expect(a1.subagentRef).toBeUndefined();
    const comp = items![2] as Extract<Item, { kind: "compaction" }>;
    expect(comp).toEqual({ kind: "compaction", id: "h2", pending: true, trigger: "", messages: 0, summary: "", archive: "" });
    const tool = items![3] as Extract<Item, { kind: "tool" }>;
    expect(tool.status).toBe("error");
    expect(tool.error).toBeUndefined(); // 非字符串不透传
  });

  it("空数组判坏（不能清空对话窗）", () => {
    expect(parseResyncItems([])).toBeNull();
    expect(parseResyncItems(null)).toBeNull();
  });
});

describe("message 透传 subagentRef（v4.26 补充契约）", () => {
  it("message 带 subagentRef：落到新建 assistant item", () => {
    const s = applyEvent(base(), { kind: "message", text: "子代理答复", subagentRef: "sa_x" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.text).toBe("子代理答复");
    expect(it.subagentRef).toBe("sa_x");
  });

  it("message 不带 subagentRef：旧行为不变（字段 undefined）", () => {
    const s = applyEvent(base(), { kind: "message", text: "主回答" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.subagentRef).toBeUndefined();
  });

  it("message 更新既有流式 assistant：带 ref 落到该条", () => {
    const s0 = base({
      items: [{ kind: "assistant", id: "a0", text: "部分", reasoning: "", streaming: true }],
      lastAssistantIdx: 0,
    });
    const s = applyEvent(s0, { kind: "message", text: "完整", subagentRef: "sa_y" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.text).toBe("完整");
    expect(it.subagentRef).toBe("sa_y");
  });

  it("message 更新时不带 ref：整体替换为 undefined（徽标不黏到后续主回答）", () => {
    const s0 = base({
      items: [{ kind: "assistant", id: "a0", text: "子代理答复", reasoning: "", streaming: false, subagentRef: "sa_x" }],
      lastAssistantIdx: 0,
    });
    const s = applyEvent(s0, { kind: "message", text: "主回答最终版" });
    const it = s.items[0] as Extract<Item, { kind: "assistant" }>;
    expect(it.text).toBe("主回答最终版");
    expect(it.subagentRef).toBeUndefined();
  });
});

// 集成：gaea-event（mock 通道）带 seq → onEvent 入口防线 → 注入 fetcher 补拉 →
// resync 落库。fetcher 经 setEventSyncFetcher 注入 mock，不 import bridge 真名。
describe("onEvent seq 缺口 → 补拉（集成）", () => {
  beforeEach(() => {
    resetEventSync();
    setEventSyncFetcher(null);
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });
  afterEach(() => {
    setEventSyncFetcher(null);
  });

  it("缺口经注入 fetcher 触发一次补拉，快照整体替换 items", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({
      seq: afterSeq,
      items: [
        { kind: "user", id: "h0", text: "快照用户" },
        { kind: "assistant", id: "h1", text: "快照回答", reasoning: "", streaming: false },
      ],
    }));
    setEventSyncFetcher(fetcher);
    renderHook(() => useController());
    await flush();
    act(() => { emitMock({ kind: "notice", text: "a", seq: 1 } as WireEvent); }); // 基线
    act(() => { emitMock({ kind: "notice", text: "b", seq: 5 } as WireEvent); }); // 缺 2/3/4
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledWith(5);
    const st = useStore.getState();
    expect(st.items).toEqual([
      { kind: "user", id: "h0", text: "快照用户" },
      { kind: "assistant", id: "h1", text: "快照回答", reasoning: "", streaming: false },
    ]);
    expect(st.lastAssistantIdx).toBe(1);
  });

  it("密集缺口 5s 冷却内只补一次（不形成补拉风暴）", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({
      seq: afterSeq,
      items: [{ kind: "user", id: "h0", text: "快照" }],
    }));
    setEventSyncFetcher(fetcher);
    renderHook(() => useController());
    await flush();
    act(() => {
      emitMock({ kind: "notice", text: "a", seq: 1 } as WireEvent);
      emitMock({ kind: "notice", text: "b", seq: 5 } as WireEvent); // 缺口1 → 触发（在途）
      emitMock({ kind: "notice", text: "c", seq: 9 } as WireEvent); // 在途期间 → 跳过
    });
    await flush();
    act(() => { emitMock({ kind: "notice", text: "d", seq: 13 } as WireEvent); }); // 冷却内 → 跳过
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledWith(5);
    // 快照落地后新到的事件照常追加：notice d（seq 13）在补拉冷却内不再触发
    // 第二次补拉，其本身正常上屏（id n4：resync 后 id 计数器从快照长度前移）。
    expect(useStore.getState().items).toEqual([
      { kind: "user", id: "h0", text: "快照" },
      { kind: "notice", id: "n4", level: "info", text: "d" },
    ]);
  });

  it("旧后端无 seq：防线静默旁路，事件照常渲染", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: [] }));
    setEventSyncFetcher(fetcher);
    renderHook(() => useController());
    await flush();
    act(() => {
      emitMock({ kind: "notice", text: "a" } as WireEvent);
      emitMock({ kind: "notice", text: "b" } as WireEvent);
    });
    await flush();
    expect(fetcher).not.toHaveBeenCalled();
    const notices = useStore.getState().items.filter((it) => it.kind === "notice");
    expect(notices).toHaveLength(2);
  });

  it("补拉失败：items 保底不变，不崩溃", async () => {
    const fetcher = vi.fn(async (): Promise<EventSyncSnapshot> => { throw new Error("resync boom"); });
    setEventSyncFetcher(fetcher);
    renderHook(() => useController());
    await flush();
    act(() => {
      emitMock({ kind: "notice", text: "a", seq: 1 } as WireEvent);
      emitMock({ kind: "notice", text: "b", seq: 5 } as WireEvent);
    });
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
    const notices = useStore.getState().items.filter((it) => it.kind === "notice");
    expect(notices).toHaveLength(2); // 快照未落地，原 items 保留
  });

  it("newSession 归零 seq 基线：新会话首件不与旧基线比较、缺口照常检出", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({
      seq: afterSeq,
      items: [{ kind: "user", id: "h0", text: "新会话快照" }],
    }));
    setEventSyncFetcher(fetcher);
    const { result } = renderHook(() => useController());
    await flush();
    act(() => { emitMock({ kind: "notice", text: "旧会话", seq: 100 } as WireEvent); }); // 旧基线
    await act(async () => { await result.current.newSession(); });
    act(() => {
      emitMock({ kind: "notice", text: "n1", seq: 50 } as WireEvent); // 新会话首件：建基线
      emitMock({ kind: "notice", text: "n2", seq: 60 } as WireEvent); // 缺 51..59
    });
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledWith(60);
    expect(useStore.getState().items).toEqual([{ kind: "user", id: "h0", text: "新会话快照" }]);
  });
});
