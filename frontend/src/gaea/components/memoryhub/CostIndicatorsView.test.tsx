import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostIndicatorsView } from "./CostIndicatorsView";
import type { CostIndicator, WorkcostBillProject } from "../../lib/types";

const mocks = vi.hoisted(() => ({
  indicatorRows: [] as CostIndicator[],
  billProjects: [] as WorkcostBillProject[],
  projectDeleted: 0,
  ratesSaved: null as { id: number; rates: WorkcostRateSet } | null,
}));

import type { WorkcostRateSet } from "../../lib/types";

vi.mock("../../lib/bridge", () => ({
  app: {
    CostIndicators: async (group: string): Promise<CostIndicator[]> => {
      // 与后端同口径：title 按科目聚合
      if (group !== "category") {
        return mocks.indicatorRows;
      }
      return mocks.indicatorRows.map((r) => ({ ...r, key: "综合单价/" + r.key }));
    },
    // 项目案例（v4.463 移入造价参考）
    WorkcostBillProjects: async (): Promise<WorkcostBillProject[]> => mocks.billProjects,
    WorkcostBillProjectDelete: async (id: number): Promise<number> => {
      mocks.projectDeleted = id;
      mocks.billProjects = mocks.billProjects.filter((p) => p.id !== id);
      return 2;
    },
    WorkcostBillProjectRatesSave: async (id: number, rates: WorkcostRateSet): Promise<void> => {
      mocks.ratesSaved = { id, rates };
    },
    PickFiles: async (): Promise<{ path: string; name: string }[]> => [],
  },
}));

const proj: WorkcostBillProject = {
  id: 1, name: "宝兴县旺平矿业土壤修复", fileName: "旺平.xlsx", source: "", location: "乐山",
  duration: "180 天", pricing: "综合单价只含人材机",
  managementRate: 0.05, regulatoryRate: 0.03, profitRate: 0.07, taxRate: 0.09,
  profitIncludesRegulatory: false, controlPrice: 0, itemCount: 32,
};

beforeEach(() => {
  mocks.indicatorRows = [];
  mocks.billProjects = [];
  mocks.projectDeleted = 0;
  mocks.ratesSaved = null;
});

describe("CostIndicatorsView 造价参考", () => {
  it("无对标样本时给出引导（样本来自资料条目沉淀）", async () => {
    render(<CostIndicatorsView />);
    expect(await screen.findByText("暂无对标样本")).toBeTruthy();
    expect(screen.getByText(/样本来自「价格数据 → 资料条目」/)).toBeTruthy();
  });

  it("有样本时渲染分位数表格（中位数/均值/P25/P75）", async () => {
    mocks.indicatorRows = [
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
    mocks.indicatorRows = [
      { key: "机械挖土方", unit: "m³", samples: 2, min: 10, max: 14, mean: 12, median: 12, p25: 11, p75: 13 },
    ];
    render(<CostIndicatorsView />);
    await screen.findByText("机械挖土方");

    fireEvent.click(screen.getByText("按分类"));
    await waitFor(() => expect(screen.getByText("综合单价/机械挖土方")).toBeTruthy());
  });

  it("项目案例区：导入的项目卡片（清单数/费率摘要）显形", async () => {
    mocks.billProjects = [proj];
    render(<CostIndicatorsView />);
    expect(await screen.findByText("宝兴县旺平矿业土壤修复")).toBeTruthy();
    expect(screen.getByText(/32 条清单/)).toBeTruthy();
    // 费率摘要（录入数据展示）。
    expect(screen.getByText(/企管 5% · 规费 3% · 利润 7% · 税 9%/)).toBeTruthy();
  });

  it("展开项目卡片 → 费率编辑（录入数据，保存走 RatesSave）", async () => {
    mocks.billProjects = [proj];
    render(<CostIndicatorsView />);
    fireEvent.click(await screen.findByText("宝兴县旺平矿业土壤修复"));
    const mgmt = await screen.findByLabelText("企管 %");
    expect((mgmt as HTMLInputElement).value).toBe("5");
    fireEvent.change(mgmt, { target: { value: "8" } });
    fireEvent.click(screen.getByText("保存费率"));
    await waitFor(() => expect(mocks.ratesSaved).toBeTruthy());
    expect(mocks.ratesSaved!.rates.managementRate).toBe(0.08);
  });

  it("删除项目案例：二次确认 → WorkcostBillProjectDelete（保留语义显形）", async () => {
    mocks.billProjects = [proj];
    render(<CostIndicatorsView />);
    fireEvent.click(await screen.findByTitle(/整体删除该项目/));
    expect(screen.getByText(/共享定额与工料机资源/)).toBeTruthy();
    expect(mocks.projectDeleted).toBe(0);
    fireEvent.click(screen.getByText("确认删除"));
    await waitFor(() => expect(mocks.projectDeleted).toBe(1));
    expect(await screen.findByText(/连带独占定额 2 条/)).toBeTruthy();
  });
});
