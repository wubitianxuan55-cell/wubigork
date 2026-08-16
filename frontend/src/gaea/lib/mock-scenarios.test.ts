// 评审 03-office-frontend.md 缺陷 8：审批卡/提问卡/压缩卡此前无浏览器 mock
// 场景。本测试锁定 ?mock=approval|ask|compaction 三条事件流的事件序列契约，
// 防止 mock 场景与真实 Go 事件流（gaea_handler.go gaeaEventMap）漂移。
import { describe, expect, it, beforeEach } from "vitest";
import { makeMockApp, mockListeners } from "./mock";
import type { WireEvent } from "./types";
import { mockScenario } from "./mock/shared";

// 场景分流：mockScenario 读 window.location.search，测试内用 URL 覆盖。
function withScenario(scenario: string, fn: () => Promise<void>) {
  const url = new URL(window.location.href);
  url.searchParams.set("mock", scenario);
  window.history.replaceState({}, "", url.toString());
  return fn().finally(() => window.history.replaceState({}, "", window.location.pathname));
}

// 订阅 emitMock 并收集事件 kind 序列（异步事件需等待）。
function collectKinds(): { kinds: string[]; done: Promise<void> } {
  const kinds: string[] = [];
  let resolve: () => void;
  const done = new Promise<void>((r) => { resolve = r; });
  const listener = (e: WireEvent) => {
    kinds.push(e.kind);
    if (e.kind === "turn_done") { mockListeners.delete(listener); resolve(); }
  };
  mockListeners.add(listener);
  return { kinds, done };
}

// initBridge 的 ?mock= 场景优先：浏览器 dev 显式带 ?mock= 时跳过 RPC 代理
// （保持 window.go 为空 → realApp() undefined → app 代理走 getMock()）。
// 此前 initBridge 非 Wails 环境无条件创建 RPC 代理，?mock= 从未生效。
describe("mock 场景 · initBridge 优先", () => {
  it("非 Wails + ?mock= 时不创建 RPC 代理（window.go 保持为空）", async () => {
    await withScenario("approval", async () => {
      // 模拟非 Wails 浏览器（无 window.go）
      delete (window as unknown as { go?: unknown }).go;
      delete (window as unknown as Record<string, unknown>).__bridge_initialized;
      const { initBridge } = await import("./bridge");
      initBridge();
      const goApp = (window as unknown as { go?: { app?: unknown } }).go?.app;
      expect(goApp).toBeUndefined();
      // app 代理落到 mock：Submit 发射 approval_request
      const { app } = await import("./bridge");
      const { kinds, done } = collectKinds();
      await app.Submit("写入审批测试文件");
      expect(kinds).toContain("approval_request");
      await app.Approve("appr-1", true, false);
      await done;
      expect(kinds[kinds.length - 1]).toBe("turn_done");
    });
  });

  it("无 ?mock= 时保持既有行为：非 Wails 创建 RPC 代理", async () => {
    await withScenario("", async () => {
      delete (window as unknown as { go?: unknown }).go;
      delete (window as unknown as Record<string, unknown>).__bridge_initialized;
      const { initBridge } = await import("./bridge");
      initBridge();
      const goApp = (window as unknown as { go?: { app?: unknown } }).go?.app;
      expect(goApp).toBeDefined();
    });
  });
});

describe("mock 场景 · 审批卡（?mock=approval）", () => {
  beforeEach(() => { mockListeners.clear(); });

  it("mockScenario 识别 approval 别名", async () => {
    await withScenario("approval", async () => {
      expect(mockScenario()).toBe("approval");
    });
    await withScenario("approve", async () => {
      expect(mockScenario()).toBe("approval");
    });
  });

  it("Submit 发射 approval_request 并挂起；Approve 后收尾 turn_done", async () => {
    await withScenario("approval", async () => {
      const app = makeMockApp();
      const { kinds, done } = collectKinds();
      await app.Submit("写入审批测试文件");
      // 挂起点：审批请求已发出，turn_done 尚未出现
      expect(kinds).toContain("approval_request");
      expect(kinds).not.toContain("turn_done");
      // 审批通过 → 工具结果 + 正文 + turn_done
      await app.Approve("appr-1", true, false);
      await done;
      expect(kinds).toContain("tool_result");
      expect(kinds[kinds.length - 1]).toBe("turn_done");
    });
  });
});

describe("mock 场景 · 提问卡（?mock=ask）", () => {
  beforeEach(() => { mockListeners.clear(); });

  it("Submit 发射带开工计划的 ask_request；AnswerQuestion 后收尾", async () => {
    await withScenario("ask", async () => {
      const app = makeMockApp();
      const kinds: string[] = [];
      let askPayload: WireEvent["ask"] | undefined;
      let resolve: () => void;
      const done = new Promise<void>((r) => { resolve = r; });
      const listener = (e: WireEvent) => {
        kinds.push(e.kind);
        if (e.kind === "ask_request") askPayload = e.ask;
        if (e.kind === "turn_done") { mockListeners.delete(listener); resolve(); }
      };
      mockListeners.add(listener);
      await app.Submit("生成季度成本测算");
      expect(kinds).toContain("ask_request");
      expect(askPayload?.plan).toBeTruthy();
      expect(askPayload?.plan?.steps.length).toBeGreaterThan(0);
      expect(askPayload?.questions[0]?.id).toBe("plan");
      expect(kinds).not.toContain("turn_done");
      await app.AnswerQuestion("ask-1", [{ questionId: "plan", selected: ["按计划开工"] }]);
      await done;
      expect(kinds[kinds.length - 1]).toBe("turn_done");
    });
  });
});

describe("mock 场景 · 压缩卡（?mock=compaction）", () => {
  beforeEach(() => { mockListeners.clear(); });

  it("Submit 发射 compaction_started → compaction_done → turn_done", async () => {
    await withScenario("compaction", async () => {
      const app = makeMockApp();
      const { kinds, done } = collectKinds();
      await app.Submit("继续之前的任务");
      await done;
      expect(kinds).toContain("compaction_started");
      expect(kinds).toContain("compaction_done");
      expect(kinds[kinds.length - 1]).toBe("turn_done");
    });
  });

  it("demo 场景不发射审批/提问/压缩事件（回归基线）", async () => {
    await withScenario("demo", async () => {
      const app = makeMockApp();
      const { kinds, done } = collectKinds();
      await app.Submit("写一份周报");
      await done;
      expect(kinds).not.toContain("approval_request");
      expect(kinds).not.toContain("ask_request");
      expect(kinds).not.toContain("compaction_started");
      expect(kinds).toContain("tool_dispatch");
      expect(kinds[kinds.length - 1]).toBe("turn_done");
    });
  });
});
