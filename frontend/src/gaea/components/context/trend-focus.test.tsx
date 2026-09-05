// trend-focus.test.tsx — v4.96 三级下钻收尾「Token 卡 → 趋势」定位/强调定向测试。
// TokenCard 侧（入口渲染/回调透传，无 mock）在 cards.test.tsx；本文件覆盖趋势
// 卡侧锚点：focus {tick} → rAF scrollIntoView({block:"nearest"}) 定位 + accent
// 外圈强调 3s 自动解除——与 v4.82 brief→浏览器（ContextBrowserTree focus）同款
// 机制与令牌；只定位不触发 onPick（趋势卡默认最新请求的选中语义不变）。
import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { ContextEvent, ContextRequestRecord } from "../../lib/types";
import { ContextTrendChart } from "../ContextView";

// 组件链经由 ContextView 触达 bridge；jsdom 无 Wails 全局，按 inspector.test
// 惯例 mock 掉 seam（ContextTrendChart 自身不调用 bridge）。
vi.mock("../../lib/bridge", () => ({
  app: new Proxy({}, { get: () => () => Promise.reject(new Error("unused in trend-focus tests")) }),
  openExternal: () => {},
  onEvent: () => () => {},
  onReady: () => () => {},
}));

// 走 useT：钉住 zh（断言基于 data-testid/类名，不依赖文案）。
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const REQ = (seq: number): ContextRequestRecord => ({
  seq,
  ts: 1750000000 + seq,
  turn: 1,
  step: seq,
  category: { system: seq * 100, tools: 0, user: 0, inject: 0, assistant: 0, tool: 0 },
});
const EVENTS: ContextEvent[] = [];

const el = (focus?: { tick: number } | null, onPick: (r: ContextRequestRecord) => void = vi.fn()) => (
  <ContextTrendChart requests={[REQ(1), REQ(2)]} events={EVENTS} onPick={onPick} focus={focus} />
);

// setup.ts 已给 Element.prototype.scrollIntoView 空实现（jsdom 缺原生）→ spy 记账。
const scrollMock = vi
  .spyOn(Element.prototype, "scrollIntoView")
  .mockImplementation(() => {});

// rAF（定位）与 3s 强调解除计时都收进假时钟，逐段推进做确定性断言。
const useFakeClock = () =>
  vi.useFakeTimers({
    toFake: ["setTimeout", "clearTimeout", "requestAnimationFrame", "cancelAnimationFrame"],
  });

beforeEach(() => {
  scrollMock.mockClear();
});

describe("ContextTrendChart focus 锚点（Token 卡→趋势 定位+强调）", () => {
  it("无 focus：不定位不强调（缺省语义不变）", () => {
    useFakeClock();
    try {
      const { getByTestId } = renderT(el(null));
      const card = getByTestId("trend-card");
      expect(card.getAttribute("data-flash")).toBe("false");
      expect(card.className).not.toContain("outline-accent/50");
      act(() => {
        vi.advanceTimersByTime(32);
      });
      expect(scrollMock).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("focus 触发：定位 scrollIntoView(block:nearest) + 强调类出现，3s 后自动解除；不触发 onPick", () => {
    const onPick = vi.fn();
    useFakeClock();
    try {
      const { getByTestId } = renderT(el({ tick: 1 }, onPick));
      const card = getByTestId("trend-card");
      // 强调立即出现（复用浏览器锚点同款 accent 外圈令牌）
      expect(card.getAttribute("data-flash")).toBe("true");
      expect(card.className).toContain("outline-accent/50");
      act(() => {
        vi.advanceTimersByTime(16); // flush rAF → 定位
      });
      expect(scrollMock).toHaveBeenCalledTimes(1);
      expect(scrollMock).toHaveBeenCalledWith({ block: "nearest" });
      act(() => {
        vi.advanceTimersByTime(3000); // 强调满 3s → 解除
      });
      expect(card.getAttribute("data-flash")).toBe("false");
      expect(card.className).not.toContain("outline-accent/50");
      expect(onPick).not.toHaveBeenCalled(); // 只定位，不改选中
    } finally {
      vi.useRealTimers();
    }
  });

  it("同 focus 对象重渲染不重复触发；新 tick 重新定位+强调（tick 去重语义）", () => {
    useFakeClock();
    try {
      const focus = { tick: 7 };
      const { getByTestId, rerender } = renderT(el(focus));
      act(() => {
        vi.advanceTimersByTime(16);
      });
      expect(scrollMock).toHaveBeenCalledTimes(1);
      // 同 tick（同对象）重渲染 → 不重复定位/强调
      rerender(<LocaleProvider>{el(focus)}</LocaleProvider>);
      act(() => {
        vi.advanceTimersByTime(16);
      });
      expect(scrollMock).toHaveBeenCalledTimes(1);
      // 新 tick → 重新定位 + 重新强调（连续点击 Token 卡入口可重触发）
      rerender(<LocaleProvider>{el({ tick: 8 })}</LocaleProvider>);
      expect(getByTestId("trend-card").getAttribute("data-flash")).toBe("true");
      act(() => {
        vi.advanceTimersByTime(16);
      });
      expect(scrollMock).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });
});
