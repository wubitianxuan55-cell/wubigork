import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostIndicatorsView } from "./CostIndicatorsView";
import type { CostIndicator } from "../../lib/types";

const indicators = vi.hoisted(() => ({ rows: [] as CostIndicator[] }));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostIndicators: async (group: string): Promise<CostIndicator[]> => {
      // 与后端同口径：title 按科目聚合
      if (group !== "category") {
        return indicators.rows;
      }
      return indicators.rows.map((r) => ({ ...r, key: "综合单价/" + r.key }));
    },
  },
}));

describe("CostIndicatorsView 造价参考", () => {
  it("无案例时给出引导提示（保存版本/沉淀后自动成为样本）", async () => {
    indicators.rows = [];
    render(<CostIndicatorsView />);
    expect(await screen.findByText("暂无对标案例")).toBeTruthy();
    expect(screen.getByText(/保存版本或沉淀明细后/)).toBeTruthy();
  });

  it("有案例时渲染分位数表格（中位数/均值/P25/P75）", async () => {
    indicators.rows = [
      {
        key: "机械挖土方",
        unit: "m³",
        samples: 3,
        min: 10,
        max: 15,
        mean: 12.5,
        median: 12,
        p25: 11,
        p75: 13.5,
      },
    ];
    render(<CostIndicatorsView />);
    expect(await screen.findByText("机械挖土方")).toBeTruthy();
    expect(screen.getByText("¥12")).toBeTruthy(); // 中位数
    expect(screen.getByText("¥12.5")).toBeTruthy(); // 均值
    expect(screen.getByText("¥11")).toBeTruthy(); // P25
    expect(screen.getByText("3")).toBeTruthy(); // 样本数
  });

  it("按科目/按分类切换分组重新加载", async () => {
    indicators.rows = [
      { key: "机械挖土方", unit: "m³", samples: 2, min: 10, max: 14, mean: 12, median: 12, p25: 11, p75: 13 },
    ];
    render(<CostIndicatorsView />);
    await screen.findByText("机械挖土方");

    fireEvent.click(screen.getByText("按分类"));
    await waitFor(() => expect(screen.getByText("综合单价/机械挖土方")).toBeTruthy());
  });
});
