// trend-focus.test.tsx — v4.96 三级下钻收尾「Token 卡 → 趋势」定位/强调定向测试。
// TokenCard 侧（入口渲染/回调透传，无 mock）在 cards.test.tsx；本文件覆盖趋势
// 卡侧锚点：focus {tick} → rAF scrollIntoView({block:"nearest"}) 定位 + accent
// 外圈强调 3s 自动解除——与 v4.82 brief→浏览器（ContextBrowserTree focus）同款
// 机制与令牌；只定位不触发 onPick（趋势卡默认最新请求的选中语义不变）。
import { act, fireEvent, render, waitFor } from "@testing-library/react";
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
  category: { system: seq * 100, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 },
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
  localStorage.removeItem("gaea.context.prefs");
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


// ─── v4.542 刀4：总览带（overview band）──────────────────────────
describe("ContextTrendChart 总览带（刀4）", () => {
  it("开关关闭不渲染；打开渲染分段带并响应点击定位", () => {
    const { getByTestId, queryByTestId } = renderT(el(null));
    expect(queryByTestId("trend-overview-band")).toBeNull();
    act(() => {
      fireEvent.click(getByTestId("trend-overview"));
    });
    const band = getByTestId("trend-overview-band");
    expect(band).toBeTruthy();
    // 两根请求 → 两个 <g> 分段（等宽 100/2）
    expect(band.querySelectorAll("g[data-overview-step]").length).toBe(2);
    expect(band.querySelectorAll('[data-testid="trend-overview-quartile"]').length).toBe(3);
  });

  it("悬停柱回调请求，离开清空；总览分类点击回传步与分类", () => {
    const onHover = vi.fn();
    const onPickCategory = vi.fn();
    const { getByTestId, rerender } = renderT(
      <ContextTrendChart
        requests={[REQ(1), REQ(2)]}
        events={EVENTS}
        onPick={vi.fn()}
        onHover={onHover}
        onPickCategory={onPickCategory}
      />,
    );
    fireEvent.mouseEnter(getByTestId("trend-bar-1"));
    expect(onHover).toHaveBeenLastCalledWith(expect.objectContaining({ seq: 1 }));
    fireEvent.mouseLeave(getByTestId("trend-bar-1"));
    expect(onHover).toHaveBeenLastCalledWith(null);

    rerender(
      <LocaleProvider>
        <ContextTrendChart
          requests={[REQ(1), REQ(2)]}
          events={EVENTS}
          onPick={vi.fn()}
          onPickCategory={onPickCategory}
        />
      </LocaleProvider>,
    );
    fireEvent.click(getByTestId("trend-overview"));
    const systemBand = getByTestId("trend-overview-band").querySelector('[data-overview-cat="system"]');
    if (!systemBand) throw new Error("overview system band missing");
    fireEvent.click(systemBand);
    expect(onPickCategory).toHaveBeenCalledWith("system", expect.objectContaining({ seq: 1 }));
  });

  it("达到自动总览阈值时切换总览带，手动关闭后本会话保持关闭", () => {
    const many = Array.from({ length: 100 }, (_, i) => REQ(i + 1));
    const { getByTestId, queryByTestId, rerender } = renderT(
      <ContextTrendChart requests={many} events={EVENTS} onPick={vi.fn()} sessionKey="/s/a" />,
    );
    expect(getByTestId("trend-overview-band")).toBeTruthy();
    fireEvent.click(getByTestId("trend-overview"));
    expect(queryByTestId("trend-overview-band")).toBeNull();
    rerender(
      <LocaleProvider>
        <ContextTrendChart requests={[...many, REQ(101)]} events={EVENTS} onPick={vi.fn()} sessionKey="/s/a" />
      </LocaleProvider>,
    );
    expect(queryByTestId("trend-overview-band")).toBeNull();
  });

  it("总览支持方向键、Home/End 与 Enter 键盘下钻", () => {
    const onPick = vi.fn();
    const { getByTestId } = renderT(
      <ContextTrendChart requests={[REQ(1), REQ(2)]} events={EVENTS} onPick={onPick} />,
    );
    fireEvent.click(getByTestId("trend-overview"));
    const band = getByTestId("trend-overview-band");
    fireEvent.keyDown(band, { key: "Home" });
    fireEvent.keyDown(band, { key: "Enter" });
    expect(onPick).toHaveBeenLastCalledWith(expect.objectContaining({ seq: 1 }));
    fireEvent.keyDown(band, { key: "End" });
    fireEvent.keyDown(band, { key: " " });
    expect(onPick).toHaveBeenLastCalledWith(expect.objectContaining({ seq: 2 }));
  });
});

describe("ContextTrendChart 增量与自适应（上游补齐）", () => {
  it("增量模式按分类堆叠净增/净减，带零轴与键盘选择", () => {
    const first = { ...REQ(1), category: { ...REQ(1).category, system: 100 } };
    const second = {
      ...REQ(2),
      category: { ...REQ(2).category, system: 150, tools: 50 },
    };
    const onPick = vi.fn();
    const { getByTestId } = renderT(
      <ContextTrendChart requests={[first, second]} events={EVENTS} onPick={onPick} />,
    );
    fireEvent.click(getByTestId("trend-mode-delta"));
    expect(getByTestId("trend-delta-baseline")).toBeTruthy();
    expect(getByTestId("trend-adaptive").getAttribute("aria-pressed")).toBe("true");
    const secondBar = getByTestId("trend-bar-2");
    expect(secondBar.querySelectorAll("rect")).toHaveLength(2);
    fireEvent.keyDown(secondBar, { key: "Enter" });
    expect(onPick).toHaveBeenLastCalledWith(expect.objectContaining({ seq: 2 }));
  });

  it("总览激活时自适应禁用", () => {
    const { getByTestId } = renderT(el(null));
    fireEvent.click(getByTestId("trend-overview"));
    const adaptive = getByTestId("trend-adaptive") as HTMLButtonElement;
    expect(adaptive.disabled).toBe(true);
    expect(adaptive.getAttribute("aria-pressed")).toBe("false");
  });

  it("DNA 模式批量读取可见步骤并按模型读取序绘制；总览激活时退出 DNA", async () => {
    const fetchDna = vi.fn(async (seqs: number[]) => seqs.map((seq) => ({
      ok: true,
      seq,
      bands: [
        { key: "sys", cat: "system" as const, tokens: 100, nodeSeq: 1 },
        { key: "n2", cat: "user" as const, tokens: 50, nodeSeq: 2 },
        { key: "n3", cat: "tool" as const, tokens: 20 + seq, nodeSeq: 3, label: "read_file" },
      ],
    })));
    const { getByTestId } = renderT(
      <ContextTrendChart
        requests={[REQ(1), REQ(2)]}
        events={EVENTS}
        onPick={vi.fn()}
        fetchDna={fetchDna}
      />,
    );
    fireEvent.click(getByTestId("trend-dna"));
    await waitFor(() => expect(fetchDna).toHaveBeenCalledWith([1, 2]));
    await waitFor(() => expect(getByTestId("trend-bar-2").getAttribute("data-dna")).toBe("true"));
    expect(getByTestId("trend-bar-2").querySelectorAll("[data-dna-key]")).toHaveLength(3);
    fireEvent.click(getByTestId("trend-overview"));
    expect(getByTestId("trend-dna").getAttribute("aria-pressed")).toBe("false");
    expect(getByTestId("trend-overview-band")).toBeTruthy();
  });
});
