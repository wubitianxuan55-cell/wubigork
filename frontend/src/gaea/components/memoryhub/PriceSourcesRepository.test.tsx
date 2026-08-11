import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PriceSourcesRepository } from "./PriceSourcesRepository";
import { ToastProvider } from "../Toast";
import { __resetPriceMocksForTest } from "../../lib/mock";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("PriceSourcesRepository 价格源阅览仓库", () => {
  beforeEach(() => {
    __resetPriceMocksForTest();
  });

  it("集中展示所有已添加价格源的名称与完整抓取地址", async () => {
    render(wrap(<PriceSourcesRepository />));

    expect(await screen.findByText("价格源阅览仓库")).toBeTruthy();
    expect(screen.getByText("共 2 个（启用 2）")).toBeTruthy();
    expect(screen.getByText("重庆施工造价信息网")).toBeTruthy();
    expect(screen.getByText("四川造价信息网（期 758）")).toBeTruthy();
    // 完整抓取地址可见（不截断）。
    expect(screen.getByText(/抓取地址：http:\/\/202\.61\.90\.35:8032\/pubpages\/pricelist\.aspx\?period=758/)).toBeTruthy();
    expect(screen.getByText(/抓取地址：http:\/\/www\.cqsgczjxx\.org\/Pages\/CQZJW\/priceInformation\.aspx/)).toBeTruthy();
    expect(screen.getAllByText("启用").length).toBe(2);
  });

  it("复制抓取地址到剪贴板", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    render(wrap(<PriceSourcesRepository />));
    await screen.findAllByText(/抓取地址：/);

    fireEvent.click(screen.getAllByTitle("复制抓取地址")[0]);
    await waitFor(() => expect(screen.getByText("已复制抓取地址")).toBeTruthy());
    expect(writeText).toHaveBeenCalledWith(expect.stringMatching(/^https?:/));
  });

  it("可打开编辑弹窗修改价格源", async () => {
    render(wrap(<PriceSourcesRepository />));
    await screen.findAllByText(/抓取地址：/);

    fireEvent.click(screen.getAllByTitle("编辑价格源")[0]);
    expect(await screen.findByText("编辑价格源")).toBeTruthy();
    expect(screen.getByDisplayValue("重庆施工造价信息网")).toBeTruthy();
    expect(screen.getByDisplayValue("http://www.cqsgczjxx.org/Pages/CQZJW/priceInformation.aspx")).toBeTruthy();
  });

  it("可确认删除价格源并从仓库移除", async () => {
    render(wrap(<PriceSourcesRepository />));
    await screen.findAllByText(/抓取地址：/);

    fireEvent.click(screen.getAllByTitle("删除价格源")[0]);
    // antd 按钮对两个汉字自动插空格（删 除），用 role + 正则匹配。
    fireEvent.click(await screen.findByRole("button", { name: /删\s*除/ }));
    await waitFor(() => expect(screen.getByText(/已删除价格源「重庆施工造价信息网」/)).toBeTruthy());
    expect(screen.queryByText("重庆施工造价信息网")).toBeNull();
    expect(screen.getByText("四川造价信息网（期 758）")).toBeTruthy();
  });
});
