// agentNetworkStore.test.ts — GaeaAgentNetwork 共享单轮询聚合回归（模式对标
// subagentRunsStore.test.ts：单轮询单在途、迟到订阅补发、失败保留旧值自愈、
// reload、live:false 快照订阅、document.hidden 门控；单例特有：无订阅者
// reload no-op、注销弃册后重建重拉、快照 getter）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  getAgentNetworkSnapshot,
  reloadAgentNetwork,
  subscribeAgentNetwork,
  type AgentNetworkMeta,
  type AgentNetworkStatus,
} from "./agentNetworkStore";
import type { AgentNetwork } from "./types";

const mocks = vi.hoisted(() => ({ AgentNetwork: vi.fn() }));
vi.mock("./bridge", () => ({ app: mocks }));

const net = (tokens: number, status: AgentNetwork["root"]["status"] = "running"): AgentNetwork => ({
  ok: true,
  window: 0,
  root: { id: "root", name: "主 agent", kind: "root", status, toolCalls: 0, errors: 0, tokens },
});

beforeEach(() => {
  mocks.AgentNetwork.mockReset();
  mocks.AgentNetwork.mockResolvedValue(net(0));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("subscribeAgentNetwork 共享单轮询", () => {
  it("两个订阅者共享一个轮询器：一个 tick 只发一次请求（单轮询单在途）", async () => {
    vi.useFakeTimers();
    const seenA: number[] = [];
    const seenB: number[] = [];
    const offA = subscribeAgentNetwork((n) => seenA.push(n?.root.tokens ?? -1));
    const offB = subscribeAgentNetwork((n) => seenB.push(n?.root.tokens ?? -1));
    await vi.advanceTimersByTimeAsync(0); // 订阅即拉一次
    expect(seenA.at(-1)).toBe(0);
    expect(seenB.at(-1)).toBe(0);

    mocks.AgentNetwork.mockResolvedValue(net(420000));
    await vi.advanceTimersByTimeAsync(5000);
    expect(seenA.at(-1)).toBe(420000);
    expect(seenB.at(-1)).toBe(420000);
    // 两次 tick（订阅拉取 + 定时）只发 2 次请求
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(2);
    offA();
    offB();
  });

  it("在途去重：慢响应期间 tick 与显式 reload 不叠加请求", async () => {
    vi.useFakeTimers();
    let resolve!: (v: AgentNetwork) => void;
    mocks.AgentNetwork.mockImplementation(() => new Promise((res) => { resolve = res; }));
    const off = subscribeAgentNetwork(() => {});
    reloadAgentNetwork(); // 在途期间的显式重拉：合并
    await vi.advanceTimersByTimeAsync(15000); // 3 个 tick 窗口
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1); // 在途未回，后续全部跳过
    resolve(net(1));
    off();
  });

  it("迟到订阅者：立即补发当前快照与状态，不重复请求", async () => {
    vi.useFakeTimers();
    mocks.AgentNetwork.mockResolvedValue(net(65000, "completed"));
    const offA = subscribeAgentNetwork(() => {});
    await vi.advanceTimersByTimeAsync(0);
    mocks.AgentNetwork.mockClear();
    let lateNet: AgentNetwork | null | undefined;
    let lateMeta: AgentNetworkMeta | null = null as AgentNetworkMeta | null;
    const offB = subscribeAgentNetwork((n, m) => { lateNet = n; lateMeta = m; });
    expect(lateNet?.root.tokens).toBe(65000);
    expect(lateMeta?.status).toBe("ready");
    expect(mocks.AgentNetwork).not.toHaveBeenCalled(); // 补发走缓存，不发请求
    offA();
    offB();
  });
});

describe("状态广播 / 错误保留 / 显式重拉", () => {
  it("拉取失败：置 error 且保留上一份快照（宁旧勿断），下个 tick 自动重试自愈", async () => {
    vi.useFakeTimers();
    const events: Array<{ tokens: number | undefined; status?: AgentNetworkStatus }> = [];
    mocks.AgentNetwork.mockResolvedValueOnce(net(180000, "completed"));
    const off = subscribeAgentNetwork((n, m) => events.push({ tokens: n?.root.tokens, status: m.status }));
    await vi.advanceTimersByTimeAsync(0);
    mocks.AgentNetwork.mockRejectedValue(new Error("boom"));
    await vi.advanceTimersByTimeAsync(5000); // 失败 tick
    expect(events.at(-1)?.status).toBe("error");
    expect(events.at(-1)?.tokens).toBe(180000); // 上一份快照保留，不给白板
    mocks.AgentNetwork.mockResolvedValue(net(222222));
    await vi.advanceTimersByTimeAsync(5000); // 下个 tick 自愈
    expect(events.at(-1)?.status).toBe("ready");
    expect(events.at(-1)?.tokens).toBe(222222);
    off();
  });

  it("reloadAgentNetwork：置 loading 显式重拉；无订阅者 no-op", async () => {
    vi.useFakeTimers();
    reloadAgentNetwork(); // 无订阅者：不请求、不抛错
    expect(mocks.AgentNetwork).not.toHaveBeenCalled();
    const events: Array<{ tokens: number | undefined; status?: AgentNetworkStatus }> = [];
    mocks.AgentNetwork.mockResolvedValue(net(1000));
    const off = subscribeAgentNetwork((n, m) => events.push({ tokens: n?.root.tokens, status: m.status }));
    await vi.advanceTimersByTimeAsync(0);
    mocks.AgentNetwork.mockClear();
    mocks.AgentNetwork.mockResolvedValue(net(2000));
    reloadAgentNetwork();
    expect(events.at(-1)?.status).toBe("loading"); // 重试入口按下后立即有反馈
    await vi.advanceTimersByTimeAsync(0);
    expect(events.at(-1)?.status).toBe("ready");
    expect(events.at(-1)?.tokens).toBe(2000);
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1); // 重拉只发一次
    off();
  });

  it("live:false 快照订阅：不驱动 tick，只有订阅即拉与显式 reload", async () => {
    vi.useFakeTimers();
    const off = subscribeAgentNetwork(() => {}, { live: false });
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1); // 订阅即拉
    mocks.AgentNetwork.mockClear();
    await vi.advanceTimersByTimeAsync(15000);
    expect(mocks.AgentNetwork).not.toHaveBeenCalled(); // 静态数据不空转轮询
    reloadAgentNetwork();
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1); // 显式重拉仍可用
    off();
  });

  it("最后一个订阅者注销停表弃册：再订阅重建轮询并立即重拉（不陈货转售）", async () => {
    vi.useFakeTimers();
    const seen: Array<number | undefined> = [];
    mocks.AgentNetwork.mockResolvedValue(net(111));
    const off = subscribeAgentNetwork((n) => seen.push(n?.root.tokens));
    await vi.advanceTimersByTimeAsync(0);
    expect(seen.at(-1)).toBe(111);
    off(); // 全员注销：停表 + 弃册
    mocks.AgentNetwork.mockClear();
    mocks.AgentNetwork.mockResolvedValue(net(222));
    const off2 = subscribeAgentNetwork((n) => seen.push(n?.root.tokens));
    await vi.advanceTimersByTimeAsync(0);
    // 重建：立即重拉，拿到的是新数据而非旧册快照
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1);
    expect(seen.at(-1)).toBe(222);
    off2();
  });

  it("页面不可见：tick 跳过零请求，恢复可见后继续轮询（门控语义保持）", async () => {
    vi.useFakeTimers();
    const off = subscribeAgentNetwork(() => {});
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1);
    const doc = document as unknown as { hidden: boolean };
    const original = doc.hidden;
    Object.defineProperty(document, "hidden", { configurable: true, value: true });
    try {
      await vi.advanceTimersByTimeAsync(10000);
      expect(mocks.AgentNetwork).toHaveBeenCalledTimes(1); // 不可见：不轮询
    } finally {
      Object.defineProperty(document, "hidden", { configurable: true, value: original });
    }
    await vi.advanceTimersByTimeAsync(5000);
    expect(mocks.AgentNetwork).toHaveBeenCalledTimes(2); // 恢复可见：恢复轮询
    off();
  });

  it("getAgentNetworkSnapshot：非响应式读取最近成功快照；全员注销后归 null", async () => {
    vi.useFakeTimers();
    expect(getAgentNetworkSnapshot()).toBeNull(); // 从未订阅
    mocks.AgentNetwork.mockResolvedValue(net(31337));
    const off = subscribeAgentNetwork(() => {});
    await vi.advanceTimersByTimeAsync(0);
    expect(getAgentNetworkSnapshot()?.root.tokens).toBe(31337);
    off();
    expect(getAgentNetworkSnapshot()).toBeNull(); // 弃册即清
  });
});

// ── 路径声明（v4.181：绑定按会话读取）──
describe("subscribeAgentNetwork 路径声明", () => {
  it("订阅带 path → 拉取按单串传递（v4.236 wire 契约：Go 变参逐元素）；无 path → 内核会话（undefined）", () => {
    const off1 = subscribeAgentNetwork(() => {}, { path: "s1.jsonl" });
    expect(mocks.AgentNetwork).toHaveBeenCalledWith("s1.jsonl");
    off1();
    const off2 = subscribeAgentNetwork(() => {});
    expect(mocks.AgentNetwork).toHaveBeenCalledWith(""); // 无 path=内核当前会话（空串哨兵）
    off2();
  });

  it("同一 poller 跨会话切换：清空旧快照置 loading 并立即以新路径重拉，不串会话", async () => {
    const seen: string[] = [];
    let resolve2: (n: AgentNetwork) => void = () => {};
    mocks.AgentNetwork.mockImplementationOnce(() => Promise.resolve(net(111)));
    const off1 = subscribeAgentNetwork(
      (n, m) => seen.push(`${n?.root.tokens ?? -1}:${m.status}`),
      { path: "s1.jsonl" },
    );
    await new Promise((r) => setTimeout(r, 0));
    expect(seen).toEqual(["111:ready"]);
    // 订阅2 带新 path 加入同一 poller → declarePath 清快照 notify loading + 立即拉 s2
    mocks.AgentNetwork.mockImplementationOnce(
      () => new Promise<AgentNetwork>((r) => { resolve2 = r; }),
    );
    const off2 = subscribeAgentNetwork(
      (n, m) => seen.push(`${n?.root.tokens ?? -1}:${m.status}`),
      { path: "s2.jsonl" },
    );
    expect(seen.at(-1)).toBe("-1:loading"); // 旧会话树不转售
    expect(mocks.AgentNetwork).toHaveBeenLastCalledWith("s2.jsonl");
    resolve2(net(222));
    await new Promise((r) => setTimeout(r, 0));
    expect(seen.at(-1)).toBe("222:ready");
    off1();
    off2();
  });

  it("reloadAgentNetwork(path)：同路径走原重拉语义；跨路径立即切换", async () => {
    const off = subscribeAgentNetwork(() => {}, { path: "s1.jsonl" });
    await new Promise((r) => setTimeout(r, 0));
    mocks.AgentNetwork.mockClear();
    reloadAgentNetwork("s1.jsonl"); // 同路径：原语义重拉
    expect(mocks.AgentNetwork).toHaveBeenLastCalledWith("s1.jsonl");
    await new Promise((r) => setTimeout(r, 0));
    reloadAgentNetwork("s3.jsonl"); // 跨路径：立即以新路径拉取
    expect(mocks.AgentNetwork).toHaveBeenLastCalledWith("s3.jsonl");
    off();
  });
});
