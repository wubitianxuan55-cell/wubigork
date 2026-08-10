import { beforeEach, describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PriceSourcesPanel } from "./PriceSourcesPanel";
import { ToastProvider } from "../Toast";
import { __resetPriceMocksForTest } from "../../lib/mock";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("PriceSourcesPanel 价格源", () => {
  beforeEach(() => {
    __resetPriceMocksForTest();
  });

  it("展示订阅源与待确认抓取结果，确认发布价格更新", async () => {
    render(wrap(<PriceSourcesPanel />));

    // 订阅源列表（含用户提供的四川源 URL）。
    expect(await screen.findByText("http://202.61.90.35:8032/pubpages/pricelist.aspx?period=758")).toBeTruthy();
    expect(screen.getByText("重庆施工造价信息网")).toBeTruthy();

    // 待确认抓取：更新 + 新增 各 1 条，默认勾选 → 发布 2 条。
    expect(screen.getByText(/热轧光圆钢筋/)).toBeTruthy();
    expect(screen.getByText(/螺纹钢/)).toBeTruthy();
    expect(screen.getByText(/↑1 新1 同0/)).toBeTruthy();
    // 价格异常徽标：跳幅 +25% 的更新条目标「异常」并带原因。
    const badge = screen.getByText("异常");
    expect(badge.getAttribute("title")).toContain("单期跳幅 +25.0%");

    fireEvent.click(screen.getByText("发布 2 条"));
    await waitFor(() => expect(screen.getByText("已发布 2 条价格更新")).toBeTruthy());
    expect(screen.getByText("已发布")).toBeTruthy();
  });

  it("忽略抓取结果标记为已忽略", async () => {
    render(wrap(<PriceSourcesPanel />));
    await screen.findByText("重庆施工造价信息网");

    fireEvent.click(screen.getByText("忽略"));
    await waitFor(() => expect(screen.getByText("已忽略")).toBeTruthy());
  });
});
