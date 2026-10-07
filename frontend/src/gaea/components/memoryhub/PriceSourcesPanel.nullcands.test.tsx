// 价格源面板 nil 切片 JSON null 崩前端回归钉（绑定面 null 契约族）：
// 旧构建 Go 产物对空候选抓取记录序列化 candidates:null，卡片渲染
// f.candidates.slice/.length 即崩进 ErrorBoundary。修复后 Go 读侧归一，
// 此处用 null 桩钉前端兜底，保证对旧构建 Go 产物同样免疫。
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { PriceSourcesPanel } from "./PriceSourcesPanel";
import { ToastProvider } from "../Toast";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;
type G = { go?: { app?: Record<string, unknown> } };

const nullCandsFetch = {
  id: "f-null", sourceId: "s1", sourceName: "重庆站", url: "http://x", period: "758",
  fetchedAt: "2026-10-07T10:00:00Z", status: "pending", candidates: null,
};

describe("PriceSourcesPanel nil 切片 JSON null 兜底", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
  });

  it("candidates=null 的抓取记录正常渲染卡片，不进 ErrorBoundary", async () => {
    const f: Record<string, unknown> = {
      GaeaPriceSources: vi.fn().mockResolvedValue([]),
      GaeaPriceFetches: vi.fn().mockResolvedValue([nullCandsFetch]),
    };
    (window as unknown as G).go = { app: { CostB: f } };
    render(wrap(<PriceSourcesPanel />));

    expect(await screen.findByText("重庆站")).toBeTruthy();
    // 计数徽标按空候选渲染（↑0 新0 同0），而非崩溃。
    expect(screen.getByText("↑0 新0 同0")).toBeTruthy();
    expect(screen.getByText("发布 0 条")).toBeTruthy();
  });
});
