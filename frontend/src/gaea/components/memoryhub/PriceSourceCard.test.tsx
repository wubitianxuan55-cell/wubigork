import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PriceSourceCard } from "./PriceSourceCard";
import { ToastProvider } from "../Toast";
import type { PriceSource } from "../../lib/types";

const baseSrc: PriceSource = {
  id: "src-1",
  name: "重庆施工造价信息网",
  url: "http://www.cqsgczjxx.org/Pages/CQZJW/priceInformation.aspx",
  parser: "sc_table",
  frequencyHours: 24,
  area: "重庆",
  enabled: true,
  lastFetchAt: "",
  createdAt: "2026-08-10T00:00:00Z",
};

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("PriceSourceCard 单源卡片（两容器共用）", () => {
  beforeEach(() => {
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
    });
  });

  it("repository 变体：名称/频率/地区/启用态徽标/最近抓取前缀/完整地址与四操作", () => {
    render(wrap(<PriceSourceCard src={baseSrc} variant="repository" onEdit={() => {}} onDelete={() => {}} />));

    expect(screen.getByText("重庆施工造价信息网")).toBeTruthy();
    expect(screen.getByText("每天")).toBeTruthy(); // 频率徽标（frequencyHours=24）
    expect(screen.getByText("重庆")).toBeTruthy(); // 地区徽标仅阅览仓库显示
    expect(screen.getByText("启用")).toBeTruthy(); // 启用态徽标常显
    expect(screen.getByText("最近抓取：从未抓取")).toBeTruthy();
    expect(screen.getByText(/抓取地址：http:\/\/www\.cqsgczjxx\.org\/Pages\/CQZJW\/priceInformation\.aspx/)).toBeTruthy();

    expect(screen.getByTitle("编辑价格源")).toBeTruthy();
    expect(screen.getByTitle("删除价格源")).toBeTruthy();
    expect(screen.getByTitle("复制抓取地址")).toBeTruthy();
    expect(screen.getByTitle("在浏览器打开抓取地址")).toBeTruthy();
    // 阅览仓库无「抓取」按钮（管理在价格源页）。
    expect(screen.queryByTitle("立即抓取该价格源")).toBeNull();
  });

  it("repository 变体停用源：徽标变「停用」且不出现「启用」", () => {
    render(wrap(<PriceSourceCard src={{ ...baseSrc, enabled: false }} variant="repository" onEdit={() => {}} onDelete={() => {}} />));

    expect(screen.getByText("停用")).toBeTruthy();
    expect(screen.queryByText("启用")).toBeNull();
  });

  it("panel 变体：无地区/启用徽标（启用时）、时间无前缀、抓取按钮触发 onFetch，编辑/删除回调携带源", () => {
    const onFetch = vi.fn();
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    render(
      wrap(
        <PriceSourceCard src={baseSrc} variant="panel" onFetch={onFetch} onEdit={onEdit} onDelete={onDelete} />,
      ),
    );

    expect(screen.getByText("从未抓取")).toBeTruthy(); // 面板时间无「最近抓取：」前缀
    expect(screen.queryByText("重庆")).toBeNull();
    expect(screen.queryByText("启用")).toBeNull(); // 面板仅停用时出徽标
    expect(screen.queryByText("最近抓取：从未抓取")).toBeNull();

    fireEvent.click(screen.getByTitle("立即抓取该价格源"));
    expect(onFetch).toHaveBeenCalledWith(baseSrc);
    fireEvent.click(screen.getByTitle("编辑价格源"));
    expect(onEdit).toHaveBeenCalledWith(baseSrc);
    fireEvent.click(screen.getByTitle("删除价格源"));
    expect(onDelete).toHaveBeenCalledWith(baseSrc);
  });

  it("panel 变体停用源出灰色徽标；fetching 时按钮文案「抓取中…」且禁用", () => {
    render(
      wrap(
        <PriceSourceCard
          src={{ ...baseSrc, enabled: false }}
          variant="panel"
          fetching
          onFetch={() => {}}
          onEdit={() => {}}
          onDelete={() => {}}
        />,
      ),
    );

    expect(screen.getByText("停用")).toBeTruthy();
    expect(screen.getByText("抓取中…")).toBeTruthy();
    expect((screen.getByTitle("立即抓取该价格源") as HTMLButtonElement).disabled).toBe(true);
  });

  it("复制抓取地址写入剪贴板并提示", async () => {
    const writeText = navigator.clipboard.writeText as ReturnType<typeof vi.fn>;
    render(wrap(<PriceSourceCard src={baseSrc} variant="repository" onEdit={() => {}} onDelete={() => {}} />));

    fireEvent.click(screen.getByTitle("复制抓取地址"));
    await waitFor(() => expect(screen.getByText("已复制抓取地址")).toBeTruthy());
    expect(writeText).toHaveBeenCalledWith(baseSrc.url);
  });
});
