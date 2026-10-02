import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { usePriceSources } from "./usePriceSources";
import { PriceSourcesRepository } from "./PriceSourcesRepository";
import { ToastProvider } from "../Toast";
import type { PriceSource } from "../../lib/types";

// ── bridge 桩：可控行为（resolve / reject / 永不 settle），不依赖 dev mock ──
const state = vi.hoisted(() => ({
  priceSources: vi.fn<() => Promise<PriceSource[]>>(),
  priceFetches: vi.fn<() => Promise<unknown[]>>(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    PriceSources: state.priceSources,
    PriceFetches: state.priceFetches,
    PriceSourceDelete: vi.fn(),
  },
  onTaskEvent: () => () => {},
  openExternal: vi.fn(),
}));

const SRC: PriceSource[] = [
  {
    id: "src-cq", name: "重庆施工造价信息网", parser: "sc_table", frequencyHours: 24, area: "重庆",
    url: "http://www.cqsgczjxx.org/Pages/CQZJW/priceInformation.aspx",
    enabled: true, lastFetchAt: "", createdAt: "2026-08-10T00:00:00Z",
  },
  {
    id: "src-sc", name: "四川造价信息网（期 758）", parser: "sc_table", frequencyHours: 24, area: "成都市区",
    url: "http://202.61.90.35:8032/pubpages/pricelist.aspx?period=758",
    enabled: true, lastFetchAt: "", createdAt: "2026-08-10T00:00:00Z",
  },
];

describe("usePriceSources 数据装载三态（超时兜底 / 失败可见化）", () => {
  beforeEach(() => {
    state.priceSources.mockReset();
    state.priceFetches.mockReset();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("成功装载：sources 就位、loadFailed=false，withFetches 时一并取抓取记录", async () => {
    state.priceSources.mockResolvedValue(SRC);
    state.priceFetches.mockResolvedValue([]);

    const { result } = renderHook(() => usePriceSources({ withFetches: true }));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.sources.map((s) => s.name)).toEqual(["重庆施工造价信息网", "四川造价信息网（期 758）"]);
    expect(result.current.fetches).toEqual([]);
    expect(result.current.loadFailed).toBe(false);
    expect(state.priceSources).toHaveBeenCalledTimes(1);
    expect(state.priceFetches).toHaveBeenCalledTimes(1);
  });

  it("阅览仓库（缺省只读）不发起 PriceFetches 调用", async () => {
    state.priceSources.mockResolvedValue(SRC);

    const { result } = renderHook(() => usePriceSources());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.sources).toHaveLength(2);
    expect(state.priceFetches).not.toHaveBeenCalled();
  });

  it("bridge 拒绝 → loadFailed=true（失败可见，不伪装空列表）", async () => {
    state.priceSources.mockRejectedValue(new Error("bridge unavailable"));

    const { result } = renderHook(() => usePriceSources());

    await waitFor(() => expect(result.current.loadFailed).toBe(true));
    expect(result.current.loading).toBe(false);
    expect(result.current.sources).toEqual([]);
  });

  it("超时兜底：8 秒无响应回退空列表，且不算失败态", async () => {
    vi.useFakeTimers();
    // 永不 settle 的挂起调用（后端卡住场景）。
    state.priceSources.mockReturnValue(new Promise<PriceSource[]>(() => {}));

    const { result } = renderHook(() => usePriceSources());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(7999);
    });
    expect(result.current.loading).toBe(true);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.sources).toEqual([]);
    expect(result.current.loadFailed).toBe(false);
  });

  it("阅览仓库失败可见化：bridge 拒绝渲染「读取失败 + 重试」，不再伪装空仓库；重试成功恢复列表", async () => {
    state.priceSources.mockRejectedValueOnce(new Error("disk io"));
    render(
      <ToastProvider>
        <PriceSourcesRepository />
      </ToastProvider>,
    );

    expect(await screen.findByTestId("price-sources-load-failed")).toBeTruthy();
    expect(screen.getByText("价格源列表读取失败")).toBeTruthy();
    expect(screen.queryByText(/仓库为空/)).toBeNull();

    state.priceSources.mockResolvedValue(SRC);
    fireEvent.click(screen.getByText("重试"));
    expect(await screen.findByText("重庆施工造价信息网")).toBeTruthy();
    expect(screen.queryByTestId("price-sources-load-failed")).toBeNull();
  });
});
