// subagentRunsStore.test.ts — 共享单轮询聚合回归（对标 dsh #298：单轮询、
// 单在途、会话切换重建、0→N 新子代理检测；v4.64 补 loading/error/reload、
// 多会话并存、live:false 快照订阅、document.hidden 门控）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  detectNewRunRefs,
  reloadSubagentRuns,
  subscribeSubagentRuns,
  type SubagentRunsMeta,
  type SubagentRunsStatus,
} from "./subagentRunsStore";
import type { SubagentRunView } from "./types";

const mocks = vi.hoisted(() => ({ SubagentRuns: vi.fn() }));
vi.mock("./bridge", () => ({ app: mocks }));

const run = (ref: string, status: SubagentRunView["status"] = "running"): SubagentRunView => ({
  ref,
  status,
  task: `任务 ${ref}`,
  toolCalls: 0,
  createdAt: "",
  updatedAt: "",
});

beforeEach(() => {
  mocks.SubagentRuns.mockReset();
  mocks.SubagentRuns.mockResolvedValue({ runs: [] });
});

afterEach(() => {
  vi.useRealTimers();
});

describe("subscribeSubagentRuns 共享单轮询", () => {
  it("两个订阅者共享一个定时器：一个 tick 只发一次请求（单轮询单在途）", async () => {
    vi.useFakeTimers();
    const seenA: string[][] = [];
    const seenB: string[][] = [];
    const offA = subscribeSubagentRuns("s.jsonl", (r) => seenA.push(r.map((x) => x.ref)));
    const offB = subscribeSubagentRuns("s.jsonl", (r) => seenB.push(r.map((x) => x.ref)));
    await vi.advanceTimersByTimeAsync(0); // 订阅即拉一次
    expect(seenA.at(-1)).toEqual([]);
    expect(seenB.at(-1)).toEqual([]);

    mocks.SubagentRuns.mockResolvedValue({ runs: [run("sa_1")] });
    await vi.advanceTimersByTimeAsync(5000);
    expect(seenA.at(-1)).toEqual(["sa_1"]);
    expect(seenB.at(-1)).toEqual(["sa_1"]);
    // 两次 tick（订阅拉取 + 定时）只发 2 次请求
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(2);
    offA();
    offB();
  });

  it("会话切换：立即重建轮询并清空旧快照", async () => {
    vi.useFakeTimers();
    const seen: string[][] = [];
    const off = subscribeSubagentRuns("a.jsonl", (r) => seen.push(r.map((x) => x.ref)));
    await vi.advanceTimersByTimeAsync(0);
    mocks.SubagentRuns.mockResolvedValue({ runs: [run("sa_old")] });
    await vi.advanceTimersByTimeAsync(5000);
    expect(seen.at(-1)).toEqual(["sa_old"]);
    mocks.SubagentRuns.mockClear();
    mocks.SubagentRuns.mockResolvedValue({ runs: [] });
    off();
    const off2 = subscribeSubagentRuns("b.jsonl", (r) => seen.push(r.map((x) => x.ref)));
    await vi.advanceTimersByTimeAsync(0);
    // 新会话立即拉取且快照从零开始（旧会话数据不串台）
    expect(mocks.SubagentRuns).toHaveBeenCalledWith("b.jsonl");
    expect(seen.at(-1)).toEqual([]);
    off2();
  });

  it("在途去重：慢响应期间 tick 不叠加请求", async () => {
    vi.useFakeTimers();
    let resolve!: (v: unknown) => void;
    mocks.SubagentRuns.mockImplementation(() => new Promise((res) => { resolve = res; }));
    const off = subscribeSubagentRuns("s.jsonl", () => {});
    await vi.advanceTimersByTimeAsync(15000); // 3 个 tick 窗口
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1); // 在途未回，后续 tick 全部跳过
    resolve({ runs: [] });
    off();
  });
});

describe("store 扩展：状态广播 / 错误保留 / 显式重拉（v4.64）", () => {
  it("meta 随快照广播：首拉成功转 ready 并带 available/total/running", async () => {
    vi.useFakeTimers();
    const metas: SubagentRunsMeta[] = [];
    mocks.SubagentRuns.mockResolvedValue({ available: true, total: 2, running: 1, runs: [run("sa_1")] });
    const off = subscribeSubagentRuns("s.jsonl", (_r, meta) => metas.push(meta));
    await vi.advanceTimersByTimeAsync(0);
    expect(metas).toHaveLength(1);
    expect(metas[0].status).toBe("ready");
    expect(metas[0].available).toBe(true);
    expect(metas[0].total).toBe(2);
    expect(metas[0].running).toBe(1);
    off();
  });

  it("迟到订阅者：立即补发当前快照与状态，不重复请求", async () => {
    vi.useFakeTimers();
    mocks.SubagentRuns.mockResolvedValue({ available: true, total: 1, running: 0, runs: [run("sa_1", "completed")] });
    const offA = subscribeSubagentRuns("s.jsonl", () => {});
    await vi.advanceTimersByTimeAsync(0);
    mocks.SubagentRuns.mockClear();
    let lateRuns: string[] | null = null as string[] | null;
    let lateMeta: SubagentRunsMeta | null = null as SubagentRunsMeta | null;
    const offB = subscribeSubagentRuns("s.jsonl", (r, m) => { lateRuns = r.map((x) => x.ref); lateMeta = m; });
    expect(lateRuns).toEqual(["sa_1"]);
    expect(lateMeta?.status).toBe("ready");
    expect(mocks.SubagentRuns).not.toHaveBeenCalled(); // 补发走缓存，不发请求
    offA();
    offB();
  });

  it("拉取失败：置 error 且保留上一份快照（宁旧勿断），下个 tick 自动重试自愈", async () => {
    vi.useFakeTimers();
    const events: Array<{ runs: string[]; status?: SubagentRunsStatus }> = [];
    mocks.SubagentRuns.mockResolvedValueOnce({ available: true, total: 1, running: 0, runs: [run("sa_1", "completed")] });
    const off = subscribeSubagentRuns("s.jsonl", (r, m) => events.push({ runs: r.map((x) => x.ref), status: m.status }));
    await vi.advanceTimersByTimeAsync(0);
    mocks.SubagentRuns.mockRejectedValue(new Error("boom"));
    await vi.advanceTimersByTimeAsync(5000); // 失败 tick
    expect(events.at(-1)?.status).toBe("error");
    expect(events.at(-1)?.runs).toEqual(["sa_1"]); // 上一份快照保留，不给白板
    mocks.SubagentRuns.mockResolvedValue({ available: true, total: 1, running: 0, runs: [run("sa_2", "completed")] });
    await vi.advanceTimersByTimeAsync(5000); // 下个 tick 自愈
    expect(events.at(-1)?.status).toBe("ready");
    expect(events.at(-1)?.runs).toEqual(["sa_2"]);
    off();
  });

  it("reloadSubagentRuns：置 loading 显式重拉；未订阅路径 no-op", async () => {
    vi.useFakeTimers();
    reloadSubagentRuns("ghost.jsonl"); // 未订阅：不请求、不抛错
    expect(mocks.SubagentRuns).not.toHaveBeenCalled();
    const events: Array<{ runs: string[]; status?: SubagentRunsStatus }> = [];
    mocks.SubagentRuns.mockResolvedValue({ available: false, total: 0, running: 0, runs: [] });
    const off = subscribeSubagentRuns("s.jsonl", (r, m) => events.push({ runs: r.map((x) => x.ref), status: m.status }));
    await vi.advanceTimersByTimeAsync(0);
    mocks.SubagentRuns.mockClear();
    mocks.SubagentRuns.mockResolvedValue({ available: true, total: 1, running: 0, runs: [run("sa_9")] });
    reloadSubagentRuns("s.jsonl");
    expect(events.at(-1)?.status).toBe("loading"); // 重试入口按下后立即有反馈
    await vi.advanceTimersByTimeAsync(0);
    expect(events.at(-1)?.status).toBe("ready");
    expect(events.at(-1)?.runs).toEqual(["sa_9"]);
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1); // 重拉只发一次
    off();
  });

  it("live:false 快照订阅：不驱动 tick，只有订阅即拉与显式 reload", async () => {
    vi.useFakeTimers();
    const off = subscribeSubagentRuns("s.jsonl", () => {}, { live: false });
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1); // 订阅即拉
    mocks.SubagentRuns.mockClear();
    await vi.advanceTimersByTimeAsync(15000);
    expect(mocks.SubagentRuns).not.toHaveBeenCalled(); // 静态数据不空转轮询
    reloadSubagentRuns("s.jsonl");
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1); // 显式重拉仍可用
    off();
  });

  it("多会话并存：各路径独立轮询器互不串台；一方注销停表不影响另一方", async () => {
    vi.useFakeTimers();
    mocks.SubagentRuns.mockImplementation((p: string) =>
      Promise.resolve({ available: true, total: 1, running: 0, runs: [run(p === "a.jsonl" ? "sa_a" : "sa_b")] }),
    );
    const seen: Record<string, string[]> = {};
    const offA = subscribeSubagentRuns("a.jsonl", (r) => { seen.a = r.map((x) => x.ref); });
    const offB = subscribeSubagentRuns("b.jsonl", (r) => { seen.b = r.map((x) => x.ref); });
    await vi.advanceTimersByTimeAsync(0);
    expect(seen.a).toEqual(["sa_a"]);
    expect(seen.b).toEqual(["sa_b"]);
    offA();
    mocks.SubagentRuns.mockClear();
    await vi.advanceTimersByTimeAsync(5000);
    // a 注销后停表，b 照常单轮询（且只请求 b）
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1);
    expect(mocks.SubagentRuns).toHaveBeenCalledWith("b.jsonl");
    offB();
  });

  it("页面不可见：tick 跳过零请求，恢复可见后继续轮询（门控语义保持）", async () => {
    vi.useFakeTimers();
    const off = subscribeSubagentRuns("s.jsonl", () => {});
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1);
    const doc = document as unknown as { hidden: boolean };
    const original = doc.hidden;
    Object.defineProperty(document, "hidden", { configurable: true, value: true });
    try {
      await vi.advanceTimersByTimeAsync(10000);
      expect(mocks.SubagentRuns).toHaveBeenCalledTimes(1); // 不可见：不轮询
    } finally {
      Object.defineProperty(document, "hidden", { configurable: true, value: original });
    }
    await vi.advanceTimersByTimeAsync(5000);
    expect(mocks.SubagentRuns).toHaveBeenCalledTimes(2); // 恢复可见：恢复轮询
    off();
  });
});

describe("detectNewRunRefs 0→N 检测", () => {
  it("只返回 seen 中没有的新 ref，且不修改 seen", () => {
    const seen = new Set(["sa_1"]);
    const runs = [run("sa_1"), run("sa_2"), run("sa_3")];
    expect(detectNewRunRefs(seen, runs)).toEqual(["sa_2", "sa_3"]);
    expect(seen.has("sa_2")).toBe(false); // 不越权并入——去抖重评估由调用方做
  });
});
