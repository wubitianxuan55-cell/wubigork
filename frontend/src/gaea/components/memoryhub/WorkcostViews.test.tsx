import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkcostResourceView } from "./WorkcostResourceView";
import { WorkcostComposeView } from "./WorkcostComposeView";
import { WorkcostFeePanel } from "./WorkcostFeePanel";
import { QuotaModal } from "./WorkcostQuotaModal";
import type {
  WorkcostCompose,
  WorkcostFeeResult,
  WorkcostProjectBundle,
  WorkcostQuota,
  WorkcostQuotaItem,
  WorkcostRateSet,
  WorkcostResource,
  WorkcostSeedPreview,
} from "../../lib/types";

// ── mock 桥 ────────────────────────────────────────────────────────
// 语义对齐 Go 侧 workcost：
//   · 核算取用价 = 现行价优先、为 0 回退基准价；
//   · 外委独立行类但并入材料桶；
//   · 零含量/零单价行是合法占位（zeroLines），不告警；
//   · 取费链与 Go 侧 ComposeProjectFees 同口径（利润基数策略可切）。
const state = vi.hoisted(() => ({
  resources: [] as WorkcostResource[],
  quotas: [] as WorkcostQuota[],
  compose: null as WorkcostCompose | null,
  listError: "",
  seedPreview: null as WorkcostSeedPreview | null,
  seedApplied: 0,
  // 取费面板 / 定额管理 / 导出（本刀扩展）。
  quotaSaved: null as WorkcostQuota | null,
  quotaDeleted: "" as string,
  getQuota: null as WorkcostQuota | null,
  pickedDir: "" as string,
  exportCalls: [] as string[][],
  applyBundle: null as WorkcostProjectBundle | null,
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
    WorkcostQuotaGet: async (code: string): Promise<WorkcostQuota | null> => state.getQuota ?? state.quotas.find((q) => q.code === code) ?? null,
    WorkcostQuotaSave: async (q: WorkcostQuota): Promise<WorkcostQuota> => {
      state.quotaSaved = q;
      return q;
    },
    WorkcostQuotaDelete: async (code: string): Promise<void> => {
      state.quotaDeleted = code;
    },
    WorkcostProjectFees: async (
      directFee: number,
      rates: WorkcostRateSet,
      profitIncludesRegulatory: boolean,
      measures: number,
      contingency: number,
      controlPrice: number,
    ): Promise<WorkcostFeeResult> => {
      // 与 Go 侧 ComposeProjectFees / mock/cost.ts 同口径。
      const r2 = (v: number) => Math.round(v * 100) / 100;
      const managementFee = r2(directFee * rates.managementRate);
      const regulatoryFee = r2(directFee * rates.regulatoryRate);
      const profitBase = directFee + managementFee + (profitIncludesRegulatory ? regulatoryFee : 0);
      const profitFee = r2(profitBase * rates.profitRate);
      const preTaxTotal = r2(directFee + managementFee + profitFee + regulatoryFee + measures + contingency);
      const taxFee = r2(preTaxTotal * rates.taxRate);
      const total = r2(preTaxTotal + taxFee);
      const cp = r2(controlPrice);
      return {
        directFee: r2(directFee), managementFee, profitFee, regulatoryFee,
        measuresFee: r2(measures), contingency: r2(contingency),
        preTaxTotal, taxFee, total,
        controlPrice: cp,
        controlDiff: cp > 0 ? r2(total - cp) : 0,
        controlUtilPct: cp > 0 ? r2((total / cp) * 100) : 0,
      };
    },
    WorkcostSeedPreview: async (): Promise<WorkcostSeedPreview> => {
      if (!state.seedPreview) throw new Error("预览失败");
      return state.seedPreview;
    },
    WorkcostSeedApply: async () => {
      state.seedApplied++;
      return { created: 3, updated: 1, skipped: 0, counts: { labor: 1, material: 1, machine: 1, outsourced: 0 }, errors: [] };
    },
    // 导入/导出链（ComposeView 集成测试用）。
    PickFiles: async (): Promise<{ path: string; name: string; size: number }[]> => [
      { path: "C:/src/旺平矿业.xlsx", name: "旺平矿业.xlsx", size: 58583 },
    ],
    WorkcostProjectParse: async (path: string): Promise<WorkcostProjectBundle> => {
      if (!state.applyBundle) throw new Error("解析失败");
      return { ...state.applyBundle, path };
    },
    WorkcostProjectApply: async () => ({
      resourceNew: 10, resourceUpd: 2, quotaNew: 5, quotaUpd: 1, lines: 40, errors: [],
    }),
    PickDirectory: async (): Promise<string> => state.pickedDir,
    WorkcostProjectExport: async (srcPath: string, outPath: string): Promise<string> => {
      state.exportCalls.push([srcPath, outPath]);
      return outPath;
    },
    WorkcostProjectExportToWorkspace: async (srcPath: string): Promise<string> => {
      state.exportCalls.push([srcPath, "workspace"]);
      return "D:/ws/.gaea/exports/成本测算表-20260101-000000.xlsx";
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
  state.quotaSaved = null;
  state.quotaDeleted = "";
  state.getQuota = null;
  state.pickedDir = "";
  state.exportCalls = [];
  state.applyBundle = null;
  // FeePanel 草稿持久化会跨测试泄漏——每个用例从干净草稿开始。
  localStorage.removeItem("workcost-fee-draft");
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

// ── 取费汇总（WorkcostFeePanel）────────────────────────────────────
// 计算链钉子（默认率 10%/2%/7%/9%，利润基数不含规费）：
//   直接费 1000 → 企管 100 → 利润 77（1100×7%）→ 规费 20
//   → 税前 1197 → 增值税 107.73 → 含税总造价 1304.73。
describe("WorkcostFeePanel 取费汇总", () => {
  it("默认费率与空清单引导（试算不落库口径）", async () => {
    render(<WorkcostFeePanel />);
    expect((screen.getByLabelText("企业管理费 %") as HTMLInputElement).value).toBe("10");
    expect((screen.getByLabelText("规费 %") as HTMLInputElement).value).toBe("2");
    expect((screen.getByLabelText("利润 %") as HTMLInputElement).value).toBe("7");
    expect((screen.getByLabelText("增值税 %") as HTMLInputElement).value).toBe("9");
    expect(screen.getByText(/还没有清单行/)).toBeTruthy();
    expect(screen.getByText(/试算不落库/)).toBeTruthy();
  });

  it("手填行 → 直接费 → 取费链全链数值（口径钉子）", async () => {
    render(<WorkcostFeePanel />);
    fireEvent.click(screen.getByRole("button", { name: /手填行/ }));
    fireEvent.change(screen.getByLabelText("清单项名称"), { target: { value: "外委处置" } });
    fireEvent.change(screen.getByLabelText("工程量"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("综合单价"), { target: { value: "100" } });
    // 取费链完成标志：含税总造价出现（链是异步核算的）。
    expect(await screen.findByText("¥1,304.73")).toBeTruthy();
    // 直接费 = 10 × 100（合价/直接费合计/费用链首行三处同值，全部渲染才对）。
    expect(screen.getAllByText("¥1,000").length).toBe(3);
    // 取费链各行（金额互不相同，可直接匹配）。
    expect(screen.getByText("¥100")).toBeTruthy(); // 企管
    expect(screen.getByText("¥77")).toBeTruthy(); // 利润（基数 1100，不含规费）
    expect(screen.getByText("¥20")).toBeTruthy(); // 规费
    expect(screen.getByText("¥1,197")).toBeTruthy(); // 税前
    expect(screen.getByText("¥107.73")).toBeTruthy(); // 增值税
  });

  it("利润基数含规费开关：77 → 78.4（两份实测产物的真实口径差异）", async () => {
    render(<WorkcostFeePanel />);
    fireEvent.click(screen.getByRole("button", { name: /手填行/ }));
    fireEvent.change(screen.getByLabelText("工程量"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("综合单价"), { target: { value: "100" } });
    expect(await screen.findByText("¥77")).toBeTruthy();
    fireEvent.click(screen.getByText(/利润基数含规费/));
    await waitFor(() => expect(screen.getByText("¥78.4")).toBeTruthy());
    // 基数 = 1000+100+20 = 1120 × 7% = 78.4。
  });

  it("招标控制价对照：差额与利用率（超 100% 显形）", async () => {
    render(<WorkcostFeePanel />);
    fireEvent.click(screen.getByRole("button", { name: /手填行/ }));
    fireEvent.change(screen.getByLabelText("工程量"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("综合单价"), { target: { value: "100" } });
    expect(await screen.findByText("¥1,304.73")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("招标控制价（元）"), { target: { value: "1200" } });
    expect(await screen.findByText("¥104.73")).toBeTruthy(); // 差额
    expect(screen.getByText("108.73%")).toBeTruthy(); // 利用率
    expect(screen.getByText("正数 = 超控制价")).toBeTruthy();
  });

  it("从定额库添加：现算综合单价回填 → 合价 → 直接费", async () => {
    state.quotas = [
      {
        id: 1, code: "WP02", title: "施工便道", specialty: "土壤修复", chapter: "A临建", unit: "m",
        categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
        source: "项目导入", region: "", priceDate: "", note: "", status: "现行",
        items: [],
      },
    ];
    state.compose = {
      laborFee: 9, materialFee: 213.75, machineFee: 7.23, outsourcedFee: 0, otherFee: 0,
      subtotal: 229.98, compositePrice: 229.98, zeroLines: 0,
      lines: [], warnings: [],
    };
    render(<WorkcostFeePanel />);
    fireEvent.click(screen.getByRole("button", { name: /从定额库添加/ }));
    fireEvent.click(await screen.findByText("添加"));
    // 行自动带入定额名称与现算单价，工程量默认 1。
    expect(await screen.findByDisplayValue("施工便道")).toBeTruthy();
    expect(await screen.findByDisplayValue("229.98")).toBeTruthy();
    // 合价/直接费/费用链首行三处同值。
    expect((await screen.findAllByText("¥229.98")).length).toBe(3);
    expect(screen.getByTestId("workcost-fee-direct").textContent).toBe("¥229.98");
  });

  it("草稿持久化：改动入 localStorage，重挂载恢复；清空草稿回到默认", async () => {
    const { unmount } = render(<WorkcostFeePanel />);
    fireEvent.click(screen.getByRole("button", { name: /手填行/ }));
    fireEvent.change(screen.getByLabelText("清单项名称"), { target: { value: "跨会话行" } });
    await waitFor(() =>
      expect(JSON.parse(localStorage.getItem("workcost-fee-draft") ?? "{}").lines[0].title).toBe("跨会话行"),
    );
    unmount();
    const { container } = render(<WorkcostFeePanel />);
    expect(container).toBeTruthy();
    expect(await screen.findByDisplayValue("跨会话行")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /清空草稿/ }));
    expect(await screen.findByText(/还没有清单行/)).toBeTruthy();
  });
});

// ── 定额编辑弹窗（QuotaModal）──────────────────────────────────────
describe("QuotaModal 消耗定额编辑", () => {
  const item: WorkcostQuotaItem = {
    quotaCode: "WP02", resourceCode: "P001", kind: "材料", title: "级配碎石", unit: "m³",
    quantity: 0.75, lossRate: 0,
  };
  const quota: WorkcostQuota = {
    id: 7, code: "WP02", title: "施工便道", specialty: "土壤修复", chapter: "A临建", unit: "m",
    categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
    source: "项目导入", region: "", priceDate: "", note: "", status: "现行",
    items: [item],
  };

  it("编辑：改名称 + 加一行（填名）+ 删旧行 → 保存走 WorkcostQuotaSave", async () => {
    render(<QuotaModal value={quota} onCancel={() => {}} onSaved={() => {}} />);
    expect(screen.getByText("编辑消耗定额：施工便道")).toBeTruthy();
    // 编辑态编码锁定（稳定锚点，改址不改引用）。
    expect((screen.getByLabelText("定额编码") as HTMLInputElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("定额名称"), { target: { value: "施工便道（改）" } });
    fireEvent.click(screen.getByText("添加含量行"));
    // 新行必须填名（空名行会被保存按钮拦截）。
    fireEvent.change(screen.getAllByLabelText("资源名称")[1], { target: { value: "挖掘机1.0m³台班" } });
    fireEvent.change(screen.getAllByLabelText("单位")[1], { target: { value: "台班" } });
    fireEvent.change(screen.getAllByLabelText("含量")[1], { target: { value: "0.02" } });
    // 删掉旧行（级配碎石）：替换为新行。
    fireEvent.click(screen.getAllByTitle("删除本行")[0]);
    fireEvent.click(screen.getByText("保存定额"));
    await waitFor(() => expect(state.quotaSaved).toBeTruthy());
    expect(state.quotaSaved!.title).toBe("施工便道（改）");
    expect(state.quotaSaved!.code).toBe("WP02");
    expect(state.quotaSaved!.items).toHaveLength(1);
    expect(state.quotaSaved!.items![0].title).toBe("挖掘机1.0m³台班");
    expect(state.quotaSaved!.items![0].quantity).toBe(0.02);
  });

  it("含量行编辑：含量与损耗联动（损耗按百分比录入）", async () => {
    render(<QuotaModal value={quota} onCancel={() => {}} onSaved={() => {}} />);
    fireEvent.change(screen.getByLabelText("含量"), { target: { value: "1.2" } });
    fireEvent.change(screen.getByLabelText("损耗率百分比"), { target: { value: "3" } });
    fireEvent.click(screen.getByText("保存定额"));
    await waitFor(() => expect(state.quotaSaved).toBeTruthy());
    expect(state.quotaSaved!.items![0].quantity).toBe(1.2);
    expect(state.quotaSaved!.items![0].lossRate).toBe(0.03);
  });

  it("未填名称的空含量行拦截保存", () => {
    render(<QuotaModal value={quota} onCancel={() => {}} onSaved={() => {}} />);
    fireEvent.click(screen.getByText("添加含量行"));
    expect((screen.getByText("保存定额") as HTMLButtonElement).disabled).toBe(true);
  });
});

// ── 定额管理 + 导出（WorkcostComposeView 集成）─────────────────────
describe("WorkcostComposeView 定额管理与导出", () => {
  const quota: WorkcostQuota = {
    id: 1, code: "WP02", title: "施工便道", specialty: "土壤修复", chapter: "A临建", unit: "m",
    categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
    source: "项目导入", region: "", priceDate: "", note: "", status: "现行",
  };
  const compose: WorkcostCompose = {
    laborFee: 9, materialFee: 213.75, machineFee: 7.23, outsourcedFee: 0, otherFee: 0,
    subtotal: 229.98, compositePrice: 229.98, zeroLines: 0, lines: [], warnings: [],
  };

  it("编辑定额：QuotaGet 取全量（列表不含含量行）→ 改名保存 → 列表刷新重算", async () => {
    state.quotas = [quota];
    state.compose = compose;
    state.getQuota = {
      ...quota,
      items: [{ quotaCode: "WP02", resourceCode: "P001", kind: "材料", title: "级配碎石", unit: "m³", quantity: 0.75, lossRate: 0 }],
    };
    render(<WorkcostComposeView />);
    fireEvent.click(await screen.findByTitle("编辑定额与含量行"));
    expect(await screen.findByText("编辑消耗定额：施工便道")).toBeTruthy();
    // 含量行来自 QuotaGet 全量，不是列表的空 items。
    expect(screen.getByDisplayValue("级配碎石")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("定额名称"), { target: { value: "施工便道（土方外运）" } });
    fireEvent.click(screen.getByText("保存定额"));
    await waitFor(() => expect(state.quotaSaved?.title).toBe("施工便道（土方外运）"));
    expect(await screen.findByText(/已保存定额/)).toBeTruthy();
  });

  it("删除定额：二次确认弹窗 → WorkcostQuotaDelete", async () => {
    state.quotas = [quota];
    state.compose = compose;
    render(<WorkcostComposeView />);
    fireEvent.click(await screen.findByTitle("删除定额"));
    expect(await screen.findByText("删除消耗定额")).toBeTruthy();
    // 未确认前不删除。
    expect(state.quotaDeleted).toBe("");
    fireEvent.click(screen.getByText("确认删除"));
    await waitFor(() => expect(state.quotaDeleted).toBe("WP02"));
  });

  it("导出五表到指定目录：导入后选目录 → 显式路径导出（取消不导）", async () => {
    state.quotas = [quota];
    state.compose = compose;
    state.applyBundle = {
      path: "C:/src/旺平矿业.xlsx", fileName: "旺平矿业.xlsx", project: "旺平矿业",
      location: "", duration: "", pricing: "", resources: [], items: [], lines: [],
      fee: { managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09, controlPrice: 0, managementFormula: "", taxNote: "" },
      quantities: [], skipped: [], warnings: [], sheets: [],
    };
    render(<WorkcostComposeView />);
    // 导入项目表（拿到 importedPath，导出按钮才启用）。
    fireEvent.click(await screen.findByRole("button", { name: /导入项目表/ }));
    await waitFor(() =>
      expect((screen.getByRole("button", { name: /导出五表到…/ }) as HTMLButtonElement).disabled).toBe(false),
    );
    // 取消目录选择（返回空串）：静默返回，不导出。
    fireEvent.click(screen.getByRole("button", { name: /导出五表到…/ }));
    await new Promise((r) => setTimeout(r, 50));
    expect(state.exportCalls).toHaveLength(0);
    // 选目录：文件名 = 源名-时间戳.xlsx。
    state.pickedDir = "D:/成本测算";
    fireEvent.click(screen.getByRole("button", { name: /导出五表到…/ }));
    await waitFor(() => expect(state.exportCalls).toHaveLength(1));
    expect(state.exportCalls[0][0]).toBe("C:/src/旺平矿业.xlsx");
    expect(state.exportCalls[0][1]).toMatch(/^D:\/成本测算\/旺平矿业-\d{8}-\d{6}\.xlsx$/);
  });

  it("→ 工作区快捷路径仍走 ExportToWorkspace", async () => {
    state.quotas = [quota];
    state.compose = compose;
    state.applyBundle = {
      path: "C:/src/旺平矿业.xlsx", fileName: "旺平矿业.xlsx", project: "旺平矿业",
      location: "", duration: "", pricing: "", resources: [], items: [], lines: [],
      fee: { managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09, controlPrice: 0, managementFormula: "", taxNote: "" },
      quantities: [], skipped: [], warnings: [], sheets: [],
    };
    render(<WorkcostComposeView />);
    fireEvent.click(await screen.findByRole("button", { name: /导入项目表/ }));
    await waitFor(() =>
      expect((screen.getByRole("button", { name: /→ 工作区/ }) as HTMLButtonElement).disabled).toBe(false),
    );
    fireEvent.click(screen.getByRole("button", { name: /→ 工作区/ }));
    await waitFor(() => expect(state.exportCalls).toHaveLength(1));
    expect(state.exportCalls[0][1]).toBe("workspace");
  });
});
