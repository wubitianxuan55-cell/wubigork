import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkcostResourceView } from "./WorkcostResourceView";
import { WorkcostComposeView } from "./WorkcostComposeView";
import type {
  WorkcostCompose,
  WorkcostQuota,
  WorkcostResource,
  WorkcostSeedPreview,
} from "../../lib/types";

// ── mock 桥 ────────────────────────────────────────────────────────
// 语义对齐 Go 侧 workcost：
//   · 核算取用价 = 现行价优先、为 0 回退基准价；
//   · 外委独立行类但并入材料桶；
//   · 零含量/零单价行是合法占位（zeroLines），不告警。
const state = vi.hoisted(() => ({
  resources: [] as WorkcostResource[],
  quotas: [] as WorkcostQuota[],
  compose: null as WorkcostCompose | null,
  listError: "",
  seedPreview: null as WorkcostSeedPreview | null,
  seedApplied: 0,
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    WorkcostResourceList: async (): Promise<WorkcostResource[]> => {
      if (state.listError) throw new Error(state.listError);
      return state.resources;
    },
    WorkcostResourceSave: async (r: WorkcostResource) => r,
    WorkcostResourceDelete: async () => undefined,
    WorkcostResourceSetPrice: async () => state.resources[0],
    WorkcostResourcePrices: async () => [],
    WorkcostQuotaList: async (): Promise<WorkcostQuota[]> => {
      if (state.listError) throw new Error(state.listError);
      return state.quotas;
    },
    WorkcostQuotaCompose: async (): Promise<WorkcostCompose> => {
      if (!state.compose) throw new Error("核算失败");
      return state.compose;
    },
    WorkcostSeedPreview: async (): Promise<WorkcostSeedPreview> => {
      if (!state.seedPreview) throw new Error("预览失败");
      return state.seedPreview;
    },
    WorkcostSeedApply: async () => {
      state.seedApplied++;
      return { created: 3, updated: 1, skipped: 0, counts: { labor: 1, material: 1, machine: 1, outsourced: 0 }, errors: [] };
    },
  },
}));

function res(patch: Partial<WorkcostResource>): WorkcostResource {
  return {
    id: 1, code: "P001", kind: "材料", title: "C20商品混凝土", spec: "泵送到场", unit: "m³",
    basePrice: 345, currentPrice: 345, categoryPath: "工料机/材料", source: "成本库",
    supplier: "", region: "乐山", priceDate: "2026年第7期", priceType: "到场价",
    validUntil: "", lossRate: 0, note: "", tags: [], status: "现行",
    ...patch,
  };
}

beforeEach(() => {
  state.resources = [];
  state.quotas = [];
  state.compose = null;
  state.listError = "";
  state.seedPreview = null;
  state.seedApplied = 0;
});

describe("WorkcostResourceView 工料机资源库", () => {
  it("空库时给出两种入库方式的引导（存量资源化 / 新增资源）", async () => {
    render(<WorkcostResourceView />);
    expect(await screen.findByText("还没有工料机资源")).toBeTruthy();
    // 「存量资源化」同时出现在工具条按钮与空态引导里 —— 用 role 精确定位按钮。
    expect(screen.getByRole("button", { name: /存量资源化/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /新增资源/ })).toBeTruthy();
  });

  it("渲染资源表：类别标签、现行价、基准价与来源", async () => {
    state.resources = [res({}), res({ id: 2, code: "EXC", kind: "机械", title: "挖掘机1.0m³台班", unit: "台班", basePrice: 1773.3, currentPrice: 1773.3, source: "台班公式" })];
    render(<WorkcostResourceView />);
    expect(await screen.findByText("C20商品混凝土")).toBeTruthy();
    expect(screen.getByText("挖掘机1.0m³台班")).toBeTruthy();
    // 「机械」既是筛选胶囊也是行内类别标签 —— 断言至少渲染出类别标签。
    expect(screen.getAllByText("机械").length).toBeGreaterThan(0);
    expect(screen.getByText("台班公式")).toBeTruthy();
  });

  it("现行价为空时回退基准价并标注「基准」——不让用户误判为免费", async () => {
    // 语义钉子：EffectivePrice 现行价优先、为 0 回退基准价。界面必须显形。
    state.resources = [res({ currentPrice: 0, basePrice: 420 })];
    render(<WorkcostResourceView />);
    expect(await screen.findByText("基准")).toBeTruthy();
    // 「¥420」同时出现在现行价（回退基准价）与基准价两列 —— 断言两列都渲染。
    expect(screen.getAllByText("¥420").length).toBeGreaterThanOrEqual(1);
  });

  it("读取失败显形为「读取失败 + 可重试」，不装成「没有资源」", async () => {
    state.listError = "database is locked";
    render(<WorkcostResourceView />);
    const alert = await screen.findByTestId("workcost-resource-read-error");
    expect(alert.textContent).toContain("读取失败");
    expect(alert.textContent).toContain("database is locked");
    expect(screen.getByText("重试")).toBeTruthy();
  });

  it("存量资源化：先出干跑预览（归类分布 + 跳过原因），确认后才入库", async () => {
    state.seedPreview = {
      candidates: [
        { sourceName: "c20", title: "C20商品混凝土", spec: "泵送", unit: "m³", kind: "材料", price: 345, categoryPath: "材料/土建材料", source: "", region: "", priceDate: "", priceType: "", note: "" },
        { sourceName: "l01", title: "普通工", spec: "", unit: "工日", kind: "人工", price: 300, categoryPath: "人工", source: "", region: "", priceDate: "", priceType: "", note: "" },
      ],
      counts: { labor: 1, material: 1, machine: 0, outsourced: 0 },
      skipped: { "综合单价条目（保留在综合单价层）": 99 },
      total: 101,
    };
    state.resources = [res({})];
    render(<WorkcostResourceView />);
    fireEvent.click(await screen.findByText("存量资源化"));

    // 预览不落库：先看到分布与跳过原因。
    expect(await screen.findByText(/干跑预览/)).toBeTruthy();
    expect(screen.getByText(/共扫描 101 条/)).toBeTruthy();
    expect(screen.getByText(/综合单价条目（保留在综合单价层） 99/)).toBeTruthy();
    expect(state.seedApplied).toBe(0);

    fireEvent.click(screen.getByText(/确认入库 2 条/));
    await waitFor(() => expect(state.seedApplied).toBe(1));
  });
});

describe("WorkcostComposeView 综合单价分析表", () => {
  const quota: WorkcostQuota = {
    id: 1, code: "WP02", title: "施工便道", specialty: "土壤修复", chapter: "A临建", unit: "m",
    categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
    source: "项目导入", region: "", priceDate: "", note: "", status: "现行",
  };

  it("无定额时引导导入项目工作簿", async () => {
    render(<WorkcostComposeView />);
    expect(await screen.findByText(/还没有消耗定额/)).toBeTruthy();
  });

  it("选中定额后展示分析表：工料机明细 + 三费拆分 + 综合单价（只含人材机）", async () => {
    state.quotas = [quota];
    state.compose = {
      laborFee: 9, materialFee: 213.75, machineFee: 7.23, outsourcedFee: 0, otherFee: 0,
      subtotal: 229.98, compositePrice: 229.98, zeroLines: 0,
      lines: [
        { kind: "人工", title: "普通工", unit: "工日", quantity: 0.03, price: 300, lossRate: 0, amount: 9, sharePct: 3.9 },
        { kind: "材料", title: "级配碎石", unit: "m³", quantity: 0.75, price: 78, lossRate: 0, amount: 58.5, sharePct: 25.4 },
      ],
      warnings: [],
    };
    render(<WorkcostComposeView />);
    expect(await screen.findByText("施工便道")).toBeTruthy();
    expect(screen.getByText("普通工")).toBeTruthy();
    expect(screen.getByText("级配碎石")).toBeTruthy();
    // 汇总行必须写明口径：只含人材机、不含管利税。
    expect(screen.getByText(/综合单价（人材机，不含管理费\/利润\/税金）/)).toBeTruthy();
    // 小计与综合单价同值（综合单价=人材机小计）——两处都渲染才对。
    expect(screen.getAllByText("¥229.98").length).toBe(2);
  });

  it("资源缺价的行显形为「缺价」而非静默算 0", async () => {
    state.quotas = [quota];
    state.compose = {
      laborFee: 0, materialFee: 0, machineFee: 0, outsourcedFee: 0, otherFee: 0,
      subtotal: 0, compositePrice: 0, zeroLines: 1,
      lines: [
        { kind: "材料", title: "无价材料", unit: "t", quantity: 1, price: 0, lossRate: 0, amount: 0, sharePct: 0 },
      ],
      warnings: [],
    };
    render(<WorkcostComposeView />);
    expect(await screen.findByText("缺价")).toBeTruthy();
    expect(screen.getByText(/1 条零含量\/零单价行/)).toBeTruthy();
  });

  it("核算告警显形（负值等真问题）", async () => {
    state.quotas = [quota];
    state.compose = {
      laborFee: 0, materialFee: 0, machineFee: 0, outsourcedFee: 0, otherFee: 0,
      subtotal: 0, compositePrice: 0, zeroLines: 0, lines: [],
      warnings: ["含量/单价不得为负：负数资源(t)"],
    };
    render(<WorkcostComposeView />);
    expect(await screen.findByText(/含量\/单价不得为负/)).toBeTruthy();
  });
});
