// v4.26「对话流式重造 · 对齐 Codex」WorkHeader 工作态头部行测试。
// 覆盖：零 items 也渲染（消灭死寂窗口）/ phase 更新与上轮 phase 不泄入 /
// 轮完成转完成态耗时行 / 恢复历史会话不渲染 / 步数与耗时纯函数。
import { describe, expect, it, beforeEach } from "vitest";
import { act, render } from "@testing-library/react";
import { WorkHeader, countTurnSteps, latestTurnPhaseText } from "./WorkHeader";
import { LocaleProvider } from "../lib/i18n";
import { useStore } from "../lib/store";
import type { ControllerState, Item } from "../lib/store";
import { formatElapsed } from "../lib/time";

function setStore(patch: Partial<ControllerState>) {
  useStore.setState(patch);
}

const user = (id: string, text: string): Item => ({ kind: "user", id, text });
const phase = (id: string, text: string): Item => ({ kind: "phase", id, text });
const tool = (id: string, status: "running" | "done"): Item =>
  ({ kind: "tool", id, name: "bash", args: "", readOnly: false, status });

beforeEach(() => {
  setStore({ running: false, turnStartAt: 0, items: [] });
  // WorkHeader 走 useT；钉住 zh 让断言用中文文案
  localStorage.setItem("gaea-lang", "zh");
});

const mount = (node: React.ReactElement) => render(<LocaleProvider>{node}</LocaleProvider>);

function headerOf(container: HTMLElement): HTMLElement | null {
  return container.querySelector<HTMLElement>('[data-testid="work-header"]');
}

describe("WorkHeader 工作态头部行", () => {
  it("零 items 也渲染（turn_started 后无任何 item 的死寂窗口有反馈）", () => {
    setStore({ running: true, turnStartAt: Date.now() - 5_000, items: [] });
    const view = mount(<WorkHeader />);
    const header = headerOf(view.container);
    expect(header).not.toBeNull();
    expect(header?.getAttribute("data-state")).toBe("running");
    // 无 phase 回退「思考中…」；无过程条目 → 0 步
    expect(header?.textContent).toContain("思考中…");
    expect(header?.textContent).toContain("0 步");
    expect(header?.textContent).toContain("用时");
    // 运行态有 spinner
    expect(view.container.querySelector(".animate-spin")).not.toBeNull();
  });

  it("phase 更新：最新 phase 文本上头部，历史 phase 不重复", () => {
    setStore({
      running: true,
      turnStartAt: Date.now() - 10_000,
      items: [user("u1", "跑个任务"), phase("p1", "正在启动引擎"), phase("p2", "正在重试 (2/3)")],
    });
    const view = mount(<WorkHeader />);
    const header = headerOf(view.container);
    expect(header?.textContent).toContain("正在重试 (2/3)");
    expect(header?.textContent).not.toContain("正在启动引擎");
    // 步数 = 两条 phase
    expect(header?.textContent).toContain("2 步");
  });

  it("上一轮的 phase 不泄入新一轮头部（user 边界截断）", () => {
    setStore({
      running: true,
      turnStartAt: Date.now() - 3_000,
      items: [user("u1", "第一问"), phase("p1", "上一轮正在重试"), user("u2", "第二问")],
    });
    const view = mount(<WorkHeader />);
    expect(headerOf(view.container)?.textContent).toContain("思考中…");
    expect(headerOf(view.container)?.textContent).not.toContain("上一轮正在重试");
  });

  it("轮完成后转完成态耗时行：不再吃 spinner，冻结步数", () => {
    setStore({
      running: true,
      turnStartAt: Date.now() - 83_000,
      items: [user("u1", "干个活"), tool("t1", "done"), phase("p1", "思考中")],
    });
    const view = mount(<WorkHeader />);
    expect(headerOf(view.container)?.getAttribute("data-state")).toBe("running");

    // turn_done：running → false（同一次 store 更新已带终态 items）
    act(() => { setStore({ running: false }); });
    const header = headerOf(view.container);
    expect(header?.getAttribute("data-state")).toBe("done");
    expect(header?.textContent).toContain("已完成");
    expect(header?.textContent).not.toContain("思考中…");
    expect(header?.textContent).toContain("用时 1m");
    expect(header?.textContent).toContain("2 步");
    // 完成态不吃 spinner
    expect(view.container.querySelector(".animate-spin")).toBeNull();
  });

  it("恢复历史会话（turnStartAt=0 且未经历运行）不渲染", () => {
    setStore({
      running: false,
      turnStartAt: 0,
      items: [user("u1", "历史问题"), { kind: "assistant", id: "a1", text: "历史回答", reasoning: "", streaming: false }],
    });
    const view = mount(<WorkHeader />);
    expect(headerOf(view.container)).toBeNull();
  });
});

describe("countTurnSteps / latestTurnPhaseText / formatElapsed 纯函数", () => {
  it("countTurnSteps：与过程卡同口径（有正文 assistant 不算步）", () => {
    const items: Item[] = [
      // 上一轮（在最后一轮 user 之前，不计入）
      user("u0", "上一问"),
      tool("t0", "done"),
      // 最后一轮
      user("u1", "问"),
      { kind: "assistant", id: "a1", text: "", reasoning: "思考", streaming: false },
      tool("t1", "done"),
      { kind: "assistant", id: "a2", text: "正文答案", reasoning: "", streaming: false },
      { kind: "phase", id: "p1", text: "重试" },
      { kind: "notice", id: "n1", level: "info", text: "提示" },
    ];
    expect(countTurnSteps(items)).toBe(4);
    expect(countTurnSteps([])).toBe(0);
  });

  it("latestTurnPhaseText：取本轮最新 phase，跨 user 边界截断", () => {
    expect(latestTurnPhaseText([user("u1", "问"), phase("p1", "A"), phase("p2", "B")])).toBe("B");
    expect(latestTurnPhaseText([phase("p0", "旧"), user("u1", "问")])).toBe("");
    expect(latestTurnPhaseText([])).toBe("");
  });

  it("formatElapsed：<60s 秒显示，≥60s 分秒显示", () => {
    expect(formatElapsed(0)).toBe("0s");
    expect(formatElapsed(42)).toBe("42s");
    expect(formatElapsed(83)).toBe("1m23s");
    expect(formatElapsed(600)).toBe("10m0s");
  });
});
