// subagentRunsStore.test.ts — 共享单轮询聚合回归（对标 dsh #298：单轮询、
// 单在途、会话切换重建、0→N 新子代理检测）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { detectNewRunRefs, subscribeSubagentRuns } from "./subagentRunsStore";
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

describe("detectNewRunRefs 0→N 检测", () => {
  it("只返回 seen 中没有的新 ref，且不修改 seen", () => {
    const seen = new Set(["sa_1"]);
    const runs = [run("sa_1"), run("sa_2"), run("sa_3")];
    expect(detectNewRunRefs(seen, runs)).toEqual(["sa_2", "sa_3"]);
    expect(seen.has("sa_2")).toBe(false); // 不越权并入——去抖重评估由调用方做
  });
});
