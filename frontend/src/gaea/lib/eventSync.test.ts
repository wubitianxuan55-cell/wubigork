// v4.26 事件序号防线测试：tracker 簿记（正常递增/首件/乱序/跳号）、冷却门、
// noteEventSeq 编排（无 seq 旁路 / 缺口触发 / 5s 冷却 / 在途去重 / 失败重试）。
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createEventSyncTracker,
  extractSeq,
  getEventSyncFetcher,
  noteEventSeq,
  resetEventSync,
  RESYNC_COOLDOWN_MS,
  setEventSyncFetcher,
  shouldResync,
} from "./eventSync";
import type { EventSyncSnapshot, EventSyncFetcher } from "./eventSync";

// 微任务排空：noteEventSeq 的补拉是异步 then 链，测试内统一等一个宏任务。
async function tick(): Promise<void> {
  await new Promise((r) => setTimeout(r, 0));
}

describe("createEventSyncTracker", () => {
  it("正常递增无缺口", () => {
    const t = createEventSyncTracker();
    expect(t.feed(1)).toBe(0);
    expect(t.feed(2)).toBe(0);
    expect(t.feed(3)).toBe(0);
    expect(t.lastSeq()).toBe(3);
  });

  it("首件不判缺：以首件为基线", () => {
    const t = createEventSyncTracker();
    expect(t.feed(7)).toBe(0); // 订阅前丢了几件无从知晓，不误报
    expect(t.lastSeq()).toBe(7);
    expect(t.feed(8)).toBe(0);
  });

  it("乱序回退忽略：不回退基线、不误报缺口", () => {
    const t = createEventSyncTracker();
    t.feed(5);
    expect(t.feed(3)).toBe(0); // 迟到的旧事件
    expect(t.lastSeq()).toBe(5); // 基线不动
    expect(t.feed(6)).toBe(0); // 5→6 相邻，无缺口
  });

  it("重复序号按乱序处理", () => {
    const t = createEventSyncTracker();
    t.feed(5);
    expect(t.feed(5)).toBe(0);
    expect(t.lastSeq()).toBe(5);
  });

  it("跳号返回缺失数", () => {
    const t = createEventSyncTracker();
    t.feed(5);
    expect(t.feed(9)).toBe(3); // 6/7/8 丢失
    expect(t.lastSeq()).toBe(9);
  });

  it("reset 后重新建基线（会话切换语义）", () => {
    const t = createEventSyncTracker();
    t.feed(10);
    t.reset();
    expect(t.lastSeq()).toBeNull();
    expect(t.feed(1)).toBe(0); // 新会话从 1 起算，不再与旧基线比较
    expect(t.lastSeq()).toBe(1);
  });

  it("脏序号（NaN/Infinity）不建基线不误报", () => {
    const t = createEventSyncTracker();
    expect(t.feed(Number.NaN)).toBe(0);
    expect(t.feed(Number.POSITIVE_INFINITY)).toBe(0);
    expect(t.lastSeq()).toBeNull();
  });
});

describe("shouldResync 冷却门（纯函数）", () => {
  it("无缺口不补", () => {
    expect(shouldResync(0, 1000, 0)).toBe(false);
  });

  it("gap≥1 且冷却窗口外 → 补", () => {
    expect(shouldResync(1, 10000, 0)).toBe(true);
    // 恰好到达冷却窗口（>=）也算窗外
    expect(shouldResync(2, RESYNC_COOLDOWN_MS, 0)).toBe(true);
  });

  it("冷却窗口内即使缺口更大也不补", () => {
    expect(shouldResync(5, RESYNC_COOLDOWN_MS - 1, 0)).toBe(false);
    expect(shouldResync(1, 100, 50)).toBe(false);
  });
});

describe("extractSeq", () => {
  it("合法 seq 透传（负数/非有限数判为无 seq）", () => {
    expect(extractSeq({ seq: 3 })).toBe(3);
    expect(extractSeq({ seq: 3.9 })).toBe(3);
    expect(extractSeq({ seq: 0 })).toBe(0);
    expect(extractSeq({ seq: -1 })).toBeNull();
    expect(extractSeq({ seq: "5" })).toBeNull();
    expect(extractSeq({})).toBeNull();
    expect(extractSeq(null)).toBeNull();
    expect(extractSeq("str")).toBeNull();
  });
});

describe("noteEventSeq 编排（模块级单例）", () => {
  beforeEach(() => {
    resetEventSync();
    setEventSyncFetcher(null);
  });

  it("payload 无 seq（旧后端）：整条防线静默旁路", async () => {
    const fetcher = vi.fn(async (_afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: 0, items: [] }));
    setEventSyncFetcher(fetcher);
    const onSnapshot = vi.fn();
    expect(noteEventSeq({ kind: "text", text: "x" }, { onSnapshot })).toEqual({ gap: 0, resync: false });
    await tick();
    expect(fetcher).not.toHaveBeenCalled();
    expect(onSnapshot).not.toHaveBeenCalled();
  });

  it("首件只建基线，不触发补拉", () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: [] }));
    setEventSyncFetcher(fetcher);
    expect(noteEventSeq({ seq: 1 }, { nowMs: 1000, onSnapshot: vi.fn() })).toEqual({ gap: 0, resync: false });
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("缺口触发一次补拉：afterSeq=已见最新序号，快照交 onSnapshot", async () => {
    const snap: EventSyncSnapshot = { seq: 5, items: [{ kind: "user", id: "h0", text: "补" }] };
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: snap.items }));
    setEventSyncFetcher(fetcher);
    const onSnapshot = vi.fn();
    expect(noteEventSeq({ seq: 1 }, { nowMs: 1000, onSnapshot })).toEqual({ gap: 0, resync: false });
    expect(noteEventSeq({ seq: 5 }, { nowMs: 1000, onSnapshot })).toEqual({ gap: 3, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledWith(5);
    expect(onSnapshot).toHaveBeenCalledWith(snap);
  });

  it("5s 内只触发一次：连续缺口不形成补拉风暴", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: [] }));
    setEventSyncFetcher(fetcher);
    const onSnapshot = vi.fn();
    const t0 = 1000;
    noteEventSeq({ seq: 1 }, { nowMs: t0, onSnapshot });
    expect(noteEventSeq({ seq: 5 }, { nowMs: t0, onSnapshot })).toEqual({ gap: 3, resync: true });
    await tick(); // 等在途补拉落地，隔离在途去重变量
    // 冷却内第二个缺口：只记不拉
    expect(noteEventSeq({ seq: 9 }, { nowMs: t0 + 100, onSnapshot })).toEqual({ gap: 3, resync: false });
    expect(fetcher).toHaveBeenCalledTimes(1);
    // 冷却外第三个缺口：可再次触发
    expect(noteEventSeq({ seq: 13 }, { nowMs: t0 + RESYNC_COOLDOWN_MS, onSnapshot })).toEqual({ gap: 3, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(fetcher).toHaveBeenLastCalledWith(13);
  });

  it("未挂 fetcher（null）：缺口只记不拉，且冷却不被预支", async () => {
    const onSnapshot = vi.fn();
    noteEventSeq({ seq: 1 }, { nowMs: 100, onSnapshot });
    expect(noteEventSeq({ seq: 5 }, { nowMs: 100, onSnapshot })).toEqual({ gap: 3, resync: false });
    // 挂上 fetcher 后同一时刻即可触发（冷却未在旁路期间被消耗）
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: [] }));
    setEventSyncFetcher(fetcher);
    expect(noteEventSeq({ seq: 9 }, { nowMs: 100, onSnapshot })).toEqual({ gap: 3, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledWith(9);
    expect(onSnapshot).toHaveBeenCalledTimes(1);
  });

  it("在途补拉期间再遇缺口不重复触发，落地后可再补", async () => {
    let release!: (v: EventSyncSnapshot) => void;
    const gate = new Promise<EventSyncSnapshot>((res) => { release = res; });
    const fetcher: EventSyncFetcher = vi.fn(() => gate);
    setEventSyncFetcher(fetcher);
    const onSnapshot = vi.fn();
    const t0 = 1000;
    noteEventSeq({ seq: 1 }, { nowMs: t0, onSnapshot });
    expect(noteEventSeq({ seq: 5 }, { nowMs: t0, onSnapshot })).toEqual({ gap: 3, resync: true });
    // 在途：冷却窗口外的第二个缺口也不重复触发
    expect(noteEventSeq({ seq: 9 }, { nowMs: t0 + RESYNC_COOLDOWN_MS + 1, onSnapshot })).toEqual({ gap: 3, resync: false });
    release({ seq: 9, items: [] });
    await tick(); // 等 then 链跑完（onSnapshot 落地 + 在途标记复位）
    expect(onSnapshot).toHaveBeenCalledTimes(1);
    // 落地后下一缺口（冷却外）可再次触发
    expect(noteEventSeq({ seq: 20 }, { nowMs: t0 + 2 * RESYNC_COOLDOWN_MS, onSnapshot })).toEqual({ gap: 10, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("fetcher 失败：错误交 onError、防线不崩，冷却外仍可重试", async () => {
    const fetcher = vi.fn(async (): Promise<EventSyncSnapshot> => { throw new Error("resync boom"); });
    setEventSyncFetcher(fetcher);
    const onSnapshot = vi.fn();
    const onError = vi.fn();
    const t0 = 500;
    noteEventSeq({ seq: 1 }, { nowMs: t0, onSnapshot, onError });
    expect(noteEventSeq({ seq: 4 }, { nowMs: t0, onSnapshot, onError })).toEqual({ gap: 2, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
    expect(onSnapshot).not.toHaveBeenCalled();
    expect(noteEventSeq({ seq: 7 }, { nowMs: t0 + RESYNC_COOLDOWN_MS, onSnapshot, onError })).toEqual({ gap: 2, resync: true });
    await tick();
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("resetEventSync 归零基线与冷却：新会话首件不误报", async () => {
    const fetcher = vi.fn(async (afterSeq: number): Promise<EventSyncSnapshot> => ({ seq: afterSeq, items: [] }));
    setEventSyncFetcher(fetcher);
    noteEventSeq({ seq: 100 }, { nowMs: 1000, onSnapshot: vi.fn() });
    noteEventSeq({ seq: 105 }, { nowMs: 1000, onSnapshot: vi.fn() }); // 触发过一次补拉（异步发出）
    await tick(); // 等在途补拉真正发出，再归零
    resetEventSync();
    // 新会话序号重新从小值起算：先建基线再相邻递增，全程无缺口
    expect(noteEventSeq({ seq: 1 }, { nowMs: 1001, onSnapshot: vi.fn() })).toEqual({ gap: 0, resync: false });
    expect(noteEventSeq({ seq: 2 }, { nowMs: 1001, onSnapshot: vi.fn() })).toEqual({ gap: 0, resync: false });
    expect(fetcher).toHaveBeenCalledTimes(1); // 只有 reset 前那一次
  });

  it("getEventSyncFetcher 反映注入态", () => {
    expect(getEventSyncFetcher()).toBeNull();
    const fn = (afterSeq: number): Promise<EventSyncSnapshot> => Promise.resolve({ seq: afterSeq, items: [] });
    setEventSyncFetcher(fn);
    expect(getEventSyncFetcher()).toBe(fn);
    setEventSyncFetcher(null);
    expect(getEventSyncFetcher()).toBeNull();
  });
});
