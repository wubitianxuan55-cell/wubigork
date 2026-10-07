// mock/cost.ts — 成本库/价格域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
import type { CostCategory, CostEntry, CostEstimateItem, CostEstimateVersion, CostGraphView, CostIndicator, CostProject, CostProjectSummary, CostReviewNote, CostInquiryRecord, CostAdjustSuggestion, CostInquiryScanFinding, CostStageValue, CostStageCompareRow, CostStageDeviation, PriceFetchRecord, PriceSource, CostSummary } from "../types";
import {
  costCategoriesMock,
  costMock,
  priceFetchMock,
  priceSourcesMock,
  setCostCategoriesMock,
  setPriceFetchMock,
  setPriceSourcesMock,
  taskView,
} from "./shared";
import type { MakeMockState } from "./state";
import type {
  WorkcostBillItem,
  WorkcostBillProject,
  WorkcostCompose,
  WorkcostComposeOverride,
  WorkcostComposeLineView,
  WorkcostFeeResult,
  WorkcostKind,
  WorkcostQuota,
  WorkcostQuotaItem,
  WorkcostRateSet,
  WorkcostResource,
  WorkcostResourcePrice,
} from "../types";

type CostMethods = Pick<
  AppBindings,
  | "CostList" | "CostSearch" | "CostSearchPage" | "CostCategories" | "CostCategorySave" | "CostCategoryDelete"
  | "SemanticIndexStatus" | "SemanticIndexBackfill"
  | "CostGet" | "CostSave" | "CostDelete"
  | "CostImportPreview" | "CostImportAIParse" | "CostImportVisionPreview" | "CostImportApply"
  | "PriceSources" | "PriceSourceSave" | "PriceSourceDelete"
  | "PriceFetch" | "PriceFetchAll" | "PriceFetches" | "PriceFetchApply" | "PriceFetchIgnore"
  | "PriceHistory" | "CostCompare"
  | "CostProjectSave" | "CostProjectList" | "CostProjectGet" | "CostProjectDelete"
  | "CostEstimateItemSave" | "CostEstimateItemDelete" | "CostEstimateItems"
  | "CostEstimateVersionSave" | "CostEstimateVersions" | "CostEstimateSediment"
  | "CostIndicators" | "CostAttribution" | "CostNoteSave" | "CostNoteList" | "CostNoteDelete" | "CostNoteBumpRef"
  | "CostGraph"
  // v4.158 组价复核闭环：确认记录回看（条目详情「组价依据」折叠区取数）。
  | "CostComposeRecords"
  // v4.50 询价飞轮 + 五算对比域补 mock（此前缺失，询价库视图在浏览器 dev 直接崩）
  | "CostInquirySave" | "CostInquiryList" | "CostInquiryDelete" | "CostInquiryExpiring" | "CostInquiryAdjust"
  | "CostInquiryScan"
  | "CostStageSave" | "CostStages" | "CostStageCompare" | "CostStageDeviations"
  // AI 组价双件（批 46 NOT_MOCKED 收尾刀；诚实拒=dev 无 LLM 内核）。
  | "CostCompose" | "CostComposeApply"
  // 工料法成本数据库（工料机资源库 + 消耗定额库 + 综合单价核算）：
  // 浏览器 dev 态给可交互的内存实现（资源增删改/调价/按定额核算），
  // 使「工料机」模块在无 Go 后端时仍可用。
  | "WorkcostResourceSave" | "WorkcostResourceGet" | "WorkcostResourceList" | "WorkcostResourceDelete"
  | "WorkcostResourceSetPrice" | "WorkcostResourcePrices"
  | "WorkcostQuotaSave" | "WorkcostQuotaGet" | "WorkcostQuotaList" | "WorkcostQuotaDelete"
  | "WorkcostQuotaCompose" | "WorkcostProjectFees"
  | "WorkcostSeedPreview" | "WorkcostSeedApply" | "WorkcostRecompose"
  | "WorkcostProjectParse" | "WorkcostProjectApply" | "WorkcostProjectExport" | "WorkcostProjectExportToWorkspace"
  | "WorkcostBillProjects" | "WorkcostBillItems" | "WorkcostBillItemSave" | "WorkcostBillItemDelete" | "WorkcostBillProjectRatesSave"
  | "WorkcostBillProjectDelete"
>;

// ── 工料法 mock 状态（工料机资源库 + 消耗定额库，浏览器内存态）──
//
// 演示数据刻意取自实测模版口径（旺平矿业/什邡五表产物）：
// 资源编码用助记码（L01/L02/EXC/C20/HDPE），综合单价只含人材机。
let mockResources: WorkcostResource[] = [];
let mockResourceSeq = 1;
let mockResourcePrices: WorkcostResourcePrice[] = [];
let mockResourcePriceSeq = 1;
let mockQuotas: WorkcostQuota[] = [];
// 工料法④清单层：项目与清单项（内存态演示）。
let mockBillProjects: WorkcostBillProject[] = [];
let mockBillItems: WorkcostBillItem[] = [];
let mockBillSeq = 1;

// ensureWorkcostSeed 首次访问时播种演示数据（幂等）。
function ensureWorkcostSeed(): void {
  if (mockResources.length > 0) return;
  const add = (code: string, kind: WorkcostKind, title: string, spec: string, unit: string, price: number, source: string): void => {
    mockResources.push({
      id: mockResourceSeq++, code, kind, title, spec, unit,
      basePrice: price, currentPrice: price, categoryPath: `工料机/${kind}`,
      source, supplier: "", region: "乐山", priceDate: "2026年第7期", priceType: "到场价",
      validUntil: "", lossRate: 0, note: "", tags: [], status: "现行",
    });
  };
  add("L01", "人工", "普通工", "清底砌筑铺装巡井", "工日", 300, "成本库");
  add("L02", "人工", "技术工/带班", "测量焊接指挥", "工日", 350, "成本库");
  add("EXC", "机械", "挖掘机1.0m³台班", "折旧900+柴油×90L+司机", "台班", 1773.3, "台班公式");
  // 清单层演示种子：一个项目 + 两条清单项（工程量是录入数据）。
  mockBillProjects = [{
    id: 1, name: "旺平矿业修复（演示）", fileName: "旺平矿业修复项目成本测算表.xlsx",
    source: "", location: "乐山", duration: "180 天", pricing: "综合单价只含人材机，管理费/利润/税金不进综合单价",
    managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09,
    profitIncludesRegulatory: false, controlPrice: 0, itemCount: 2,
    createdAt: "2026-10-07T10:00:00Z", updatedAt: "2026-10-07T10:00:00Z",
  }];
  mockBillItems = [
    { id: mockBillSeq++, projectId: 1, code: "WP01", title: "施工便道（铺筑压实）", unit: "m", division: "A临建", quantity: 850, quantityExpr: "长850×宽1", quotaCode: "WP01", feature: "泥结碎石", priceOverride: 0, sort: 1 },
    { id: mockBillSeq++, projectId: 1, code: "WP02", title: "场地平整", unit: "m²", division: "A临建", quantity: 12000, quantityExpr: "", quotaCode: "WP02", feature: "", priceOverride: 0, sort: 2 },
  ];
  add("LDR", "机械", "装载机50型台班", "折旧700+柴油×70L+司机", "台班", 1445.9, "台班公式");
  add("C20", "材料", "C20商品混凝土", "泵送到场", "m³", 345, "商品混凝土C30减20");
  add("AGG", "材料", "级配碎石", "便道基层到场", "m³", 78, "估价");
  add("HDPE", "材料", "HDPE防渗膜1.5mm", "双光面", "m²", 12, "成本库");
  add("GT", "材料", "土工布400g", "非织造", "m²", 5, "成本库");
  add("KILN", "外委", "水泥窑协同处置", "不含运（用户确认价）", "t", 190, "外委");

  const byCode = (c: string): WorkcostResource | undefined => mockResources.find((r) => r.code === c);
  const item = (code: string, qty: number): WorkcostQuotaItem => {
    const r = byCode(code);
    return {
      quotaCode: "", resourceCode: code, kind: r?.kind ?? "材料",
      title: r?.title ?? code, unit: r?.unit ?? "", quantity: qty, resourcePrice: 0, lossRate: 0,
    };
  };
  mockQuotas = [
    {
      id: 1, code: "WP02", title: "施工便道", specialty: "土壤修复", chapter: "A临建", unit: "m",
      categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
      source: "项目导入（演示）", region: "", priceDate: "", note: "3m宽，20cm碎石+15cm混凝土",
      status: "现行",
      items: [item("L01", 0.03), item("AGG", 0.75), item("C20", 0.45), item("LDR", 0.005)],
    },
    {
      id: 2, code: "WP01", title: "测量放线", specialty: "土壤修复", chapter: "A临建", unit: "项",
      categoryPath: "综合单价/A临建", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
      source: "项目导入（演示）", region: "", priceDate: "", note: "拐点、分层标高",
      status: "现行",
      items: [item("L02", 20), item("L01", 20)],
    },
    {
      id: 3, code: "WP19", title: "水泥窑协同处置", specialty: "土壤修复", chapter: "D外运处置", unit: "t",
      categoryPath: "综合单价/D外运处置", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
      source: "项目导入（演示）", region: "", priceDate: "", note: "不含运 190 元/t",
      status: "现行",
      items: [item("KILN", 1)],
    },
  ];
}

// composeMock 按定额核算综合单价（只含人材机；外委并入材料桶）。
function composeMock(code: string): WorkcostCompose {
  ensureWorkcostSeed();
  const q = mockQuotas.find((x) => x.code === code);
  const lines: WorkcostComposeLineView[] = [];
  let labor = 0;
  let material = 0;
  let machine = 0;
  let outsourced = 0;
  for (const it of q?.items ?? []) {
    const r = mockResources.find((x) => x.code === it.resourceCode);
    const price = r ? (r.currentPrice > 0 ? r.currentPrice : r.basePrice) : (it.resourcePrice ?? 0);
    const amount = it.quantity > 0 && price > 0 ? it.quantity * price * (1 + (it.lossRate ?? 0)) : 0;
    switch (it.kind) {
      case "人工":
        labor += amount;
        break;
      case "机械":
        machine += amount;
        break;
      case "外委":
        outsourced += amount;
        material += amount;
        break;
      default:
        material += amount;
    }
    lines.push({
      kind: it.kind, title: it.title, unit: it.unit,
      quantity: it.quantity, price, lossRate: it.lossRate ?? 0, amount, sharePct: 0,
    });
  }
  const subtotal = labor + material + machine;
  for (const l of lines) l.sharePct = subtotal > 0 ? (l.amount / subtotal) * 100 : 0;
  const r2 = (v: number) => Math.round(v * 100) / 100;
  return {
    laborFee: r2(labor), materialFee: r2(material), machineFee: r2(machine),
    outsourcedFee: r2(outsourced), otherFee: 0, subtotal: r2(subtotal),
    compositePrice: r2(subtotal), zeroLines: lines.filter((l) => l.quantity === 0 || l.price === 0).length,
    lines, warnings: [],
  };
}
// ── 测算项目 mock 状态（浏览器开发环境内存态，无持久化）──
let mockProjects: CostProject[] = [];
let mockProjectSeq = 1;
let mockItems: CostEstimateItem[] = [];
let mockItemSeq = 1;
let mockVersions: CostEstimateVersion[] = [];
let mockVersionSeq = 1;
let mockNotes: CostReviewNote[] = [];
let mockNoteSeq = 1;

// ── 询价库 mock 状态（v4.50 补域：内存态，种子覆盖到期预警 + 调差两场景）──
let mockInquiries: CostInquiryRecord[] = [];
let mockInquirySeq = 1;
let mockStageValues: CostStageValue[] = [];
let mockStageSeq = 1;

function isoDaysFromNow(days: number): string {
  const d = new Date(Date.now() + days * 86400000);
  return d.toISOString().slice(0, 10);
}

// 成本库检索过滤（CostSearch 与 CostSearchPage 共用，v4.386 抽出）。
function filterCostMock(query: string, category: string, status: string): CostSummary[] {
  const q = (query ?? "").toLowerCase();
  return costMock.filter((e) => {
    const path = e.categoryPath || e.category || "";
    if (category && category !== "all" && path !== category && !path.startsWith(category + "/")) return false;
    if (status && status !== "all" && e.status !== status) return false;
    if (!q) return true;
    return [e.name, e.title, e.code, e.spec, e.source].some((s) => (s ?? "").toLowerCase().includes(q));
  });
}
function seedInquiry(r: Partial<CostInquiryRecord>): CostInquiryRecord {
  const now = new Date().toISOString();
  return {
    id: mockInquirySeq++, title: "", spec: "", unit: "", price: 0, source: "手动询价", supplier: "",
    region: "", priceDate: isoDaysFromNow(-7), validUntil: "", note: "", status: "现行",
    createdAt: now, updatedAt: now, ...r,
  };
}
function seedInquiries() {
  if (mockInquiries.length > 0) return;
  mockInquiries = [
    seedInquiry({ title: "P.O 42.5 水泥", spec: "散装", unit: "吨", price: 520, source: "供应商比价", supplier: "华新水泥", region: "重庆", validUntil: isoDaysFromNow(15) }),
    seedInquiry({ title: "HP300 高频液压振动锤", unit: "台班", price: 3100, source: "信息价", supplier: "2026-08 信息价", region: "重庆", validUntil: isoDaysFromNow(200) }),
    seedInquiry({ title: "中砂", spec: "细度模数 2.3-3.0", unit: "吨", price: 98, source: "手动询价", supplier: "本地砂场", region: "重庆", validUntil: "" }),
  ];
}

function mockProjectSummaries(): CostProjectSummary[] {
  return mockProjects.map((p) => {
    const items = mockItems.filter((i) => i.projectId === p.id);
    const versions = mockVersions.filter((v) => v.projectId === p.id);
    return {
      ...p,
      itemCount: items.length,
      total: items.reduce((s, i) => s + (i.quantity ?? 0) * (i.price ?? 0), 0),
      versionCount: versions.length,
    };
  });
}

function mockIndicators(group: string): CostIndicator[] {
  const caseIds = new Set(mockVersions.map((v) => v.projectId));
  const rows = mockItems.filter((i) => caseIds.has(i.projectId) && (i.price ?? 0) > 0);
  const buckets = new Map<string, { prices: number[]; units: Map<string, number> }>();
  for (const r of rows) {
    const key = group === "category" ? (r.categoryPath?.split("/")[0] || "未分类") : r.title;
    if (!key) continue;
    if (!buckets.has(key)) buckets.set(key, { prices: [], units: new Map() });
    const b = buckets.get(key)!;
    b.prices.push(r.price ?? 0);
    if (group !== "category" && r.unit) b.units.set(r.unit, (b.units.get(r.unit) ?? 0) + 1);
  }
  const out: CostIndicator[] = [];
  const pct = (sorted: number[], p: number) => {
    if (sorted.length === 1) return sorted[0];
    const rank = p * (sorted.length - 1);
    const lo = Math.floor(rank);
    const hi = Math.min(lo + 1, sorted.length - 1);
    return sorted[lo] + (rank - lo) * (sorted[hi] - sorted[lo]);
  };
  for (const [key, b] of buckets) {
    const sorted = [...b.prices].sort((x, y) => x - y);
    let unit = "";
    let bestN = 0;
    for (const [u, n] of b.units) {
      if (n > bestN) {
        unit = u;
        bestN = n;
      }
    }
    out.push({
      key,
      unit,
      samples: sorted.length,
      min: sorted[0],
      max: sorted[sorted.length - 1],
      mean: sorted.reduce((s, x) => s + x, 0) / sorted.length,
      median: pct(sorted, 0.5),
      p25: pct(sorted, 0.25),
      p75: pct(sorted, 0.75),
    });
  }
  return out.sort((a, b) => a.key.localeCompare(b.key, "zh-CN"));
}

export function buildCost(_s: MakeMockState): CostMethods {
  return {
    async CostList() {
      return costMock;
    },
    async SemanticIndexStatus() {
      // 浏览器演示态：模型不可用（无本地引擎），诚实提示。
      return { total: costMock.length, indexed: costMock.length, modelOk: false, modelNote: "浏览器演示态无本地语义模型——真机启用 Herdsman bge-m3 后可用" };
    },
    async SemanticIndexBackfill() {
      return { total: costMock.length, updated: 0, indexed: costMock.length, missing: 0 };
    },
    async CostSearch(query: string, category: string, status: string) {
      return filterCostMock(query, category, status);
    },
    // v4.386 分页检索：与 Go GaeaCostSearchPage 同口径——过滤后排序
    // （title/price/updatedAt，name tie-break）再切片，total=过滤后总数。
    // GA6-09 契约：error 仅在读取失败时非空；mock 恒可读，故不设该字段
    //（omitempty 语义等价于空串），前端消费面按「缺省=无错误」处理。
    async CostSearchPage(query: string, category: string, status: string, sortKey: string, sortDir: number, limit: number, offset: number) {
      const all = filterCostMock(query, category, status);
      if (sortKey === "title" || sortKey === "price" || sortKey === "updatedAt") {
        const dir = sortDir < 0 ? -1 : 1;
        const key = sortKey;
        all.sort((a, b) => {
          const av = a[key] ?? (key === "price" ? 0 : "");
          const bv = b[key] ?? (key === "price" ? 0 : "");
          const d = typeof av === "number" && typeof bv === "number"
            ? av - bv
            : String(av).localeCompare(String(bv), "zh-CN");
          if (d !== 0) return d * dir;
          return a.name.localeCompare(b.name) * dir;
        });
      }
      if (limit <= 0) limit = 100;
      if (limit > 200) limit = 200;
      if (offset < 0) offset = 0;
      return { items: all.slice(offset, offset + limit), total: all.length };
    },
    async CostCategories() {
      return costCategoriesMock;
    },
    async CostCategorySave(parentId: number, name: string, sort: number, id: number) {
      if (id > 0) {
        const walk = (nodes: CostCategory[]): CostCategory | null => {
          for (const n of nodes) {
            if (n.id === id) {
              n.name = name;
              n.sort = sort;
              return n;
            }
            const hit = walk(n.children ?? []);
            if (hit) return hit;
          }
          return null;
        };
        walk(costCategoriesMock);
        return id;
      }
      const nextId = Math.max(0, ...costCategoriesMock.flatMap((n) => [n.id, ...(n.children ?? []).map((c) => c.id)])) + 1;
      const node: CostCategory = { id: nextId, parentId, name, sort, count: 0, children: [] };
      if (parentId === 0) {
        costCategoriesMock.push(node);
      } else {
        const walk = (nodes: CostCategory[]): boolean => {
          for (const n of nodes) {
            if (n.id === parentId) {
              n.children = [...(n.children ?? []), node];
              return true;
            }
            if (walk(n.children ?? [])) return true;
          }
          return false;
        };
        walk(costCategoriesMock);
      }
      return nextId;
    },
    async CostCategoryDelete(id: number) {
      const walk = (nodes: CostCategory[]): CostCategory[] =>
        nodes.filter((n) => {
          if (n.id === id) return false;
          n.children = walk(n.children ?? []);
          return true;
        });
      setCostCategoriesMock(walk(costCategoriesMock));
    },
    async CostGet(name: string) {
      const e = costMock.find((c) => c.name === name);
      return e ? { ...e, body: "", createdAt: "" } : null;
    },
    // 形参对齐 Go 门面签名（X1-11 契约测试）：实参只收不用，行为零变化。
    async CostSave(_e: CostEntry) {
      // mock: no-op——浏览器开发环境无持久化成本库（真实实现写 CostLibrary 表）。
    },
    async CostDelete(_name: string) {
      // mock: no-op——同上，无库可删。
    },
    // ── 成本库导入（mock：对已知文件返回样例候选）──
    async CostImportPreview(path: string) {
      return {
        path,
        fileName: path.split(/[\\/]/).pop() ?? path,
        columns: ["材料名称", "规格型号", "单位", "单价(元)", "供应商"],
        unmapped: ["备注"],
        rows: [
          {
            name: "hp300", title: "HP300 高频液压振动锤", code: "A1-12", category: "机械", unit: "台班",
            price: 3200, spec: "300kW", source: "XX租赁", status: "现行",
            existingName: "hp300", existingPrice: 3000, matchNote: "将覆盖更新（现价 ¥3,000）",
            raw: "HP300 高频液压振动锤 | 300kW | 台班 | 3200 | XX租赁", skip: false, skipReason: "",
          },
          {
            name: "", title: "P.O 42.5 水泥", category: "材料", unit: "吨",
            price: 480, spec: "", source: "海螺", status: "现行",
            existingName: "", existingPrice: 0, matchNote: "新增",
            raw: "P.O 42.5 水泥 | | 吨 | 480 | 海螺", skip: false, skipReason: "",
          },
        ],
        message: "",
        aiUsed: false,
      };
    },
    async CostImportAIParse(path: string) {
      const pv = await this.CostImportPreview(path);
      pv.aiUsed = true;
      pv.message = "AI 智能解析完成，请核对后确认导入。";
      return pv;
    },
    // ── PDF/图片报价单导入（mock：source=pdf_text 的样例候选）──
    async CostImportVisionPreview(path: string) {
      return {
        path,
        fileName: path.split(/[/]/).pop() ?? path,
        columns: ["材料名称", "规格型号", "单位", "单价(元)", "备注"],
        unmapped: [],
        rows: [
          {
            name: "rebar", title: "热轧光圆钢筋", category: "材料", unit: "t",
            price: 3750, spec: "HPB300 Φ12", source: "供应商报价单.pdf", status: "现行",
            existingName: "rebar", existingPrice: 3000, matchNote: "将覆盖更新（现价 ¥3,000）",
            raw: "热轧光圆钢筋 HPB300 Φ12 | t | 3750", skip: false, skipReason: "",
          },
          {
            name: "", title: "螺纹钢", category: "材料", unit: "t",
            price: 3420, spec: "HRB400 Φ20", source: "供应商报价单.pdf", status: "现行",
            existingName: "", existingPrice: 0, matchNote: "新增",
            raw: "螺纹钢 HRB400 Φ20 | t | 3420", skip: false, skipReason: "",
          },
        ],
        message: "PDF 文字提取解析完成，请核对后确认导入。",
        aiUsed: true,
        source: "pdf_text",
      };
    },
    async CostImportApply(_rows?: CostEntry[], _inquirySource?: string) {
      return 0;
    },
    // ── 价格源（mock）──
    async PriceSources() {
      return priceSourcesMock;
    },
    async PriceSourceSave(src: PriceSource) {
      const i = priceSourcesMock.findIndex((s) => s.id === src.id);
      if (i >= 0) priceSourcesMock[i] = src;
      else priceSourcesMock.push(src);
    },
    async PriceSourceDelete(id: string) {
      setPriceSourcesMock(priceSourcesMock.filter((s) => s.id !== id));
    },
    async PriceFetch(id: string) {
      const src = priceSourcesMock.find((s) => s.id === id);
      const rec: PriceFetchRecord = {
        id: "fetch-" + Date.now(), sourceId: id, sourceName: src?.name ?? id,
        url: src?.url ?? "", period: "758", fetchedAt: new Date().toISOString(), status: "pending",
        candidates: priceFetchMock[0]?.candidates ?? [],
      };
      setPriceFetchMock([rec, ...priceFetchMock.filter((f) => f.id !== rec.id)]);
      if (src) src.lastFetchAt = rec.fetchedAt;
      return taskView("price_fetch", "抓取 " + (src?.name ?? id), { count: rec.candidates.length, fetchId: rec.id });
    },
    async PriceFetchAll() {
      const enabled = priceSourcesMock.filter((s) => s.enabled);
      for (const src of enabled) {
        await this.PriceFetch(src.id);
      }
      return taskView("price_fetch_all", "一键抓取全部价格源", { fetched: enabled.length, failed: 0 });
    },
    async PriceFetches() {
      return priceFetchMock;
    },
    async PriceFetchApply(fetchId: string, titles: string[]) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "applied";
      return titles.length;
    },
    async PriceFetchIgnore(fetchId: string) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "ignored";
    },
    async PriceHistory(name: string) {
      return [
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3181, source: "四川造价信息网",
          period: "758", fetchedAt: new Date().toISOString(), note: "价格源更新",
        },
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3000, source: "手动录入",
          period: "", fetchedAt: "", note: "",
        },
      ];
    },
    // ── 比价（mock：现价/历史/价格源抓取 3 行多源对比；字段对齐 Go CostCompareRow）──
    async CostCompare(name: string) {
      if (!name.trim()) return [];
      const now = new Date().toISOString();
      return [
        {
          source: "成本库", period: "", price: 3000, diffPct: 0,
          fetchedAt: "", kind: "current",
        },
        {
          source: "四川造价信息网", period: "757", price: 2980, diffPct: -0.7,
          fetchedAt: now, kind: "history",
        },
        {
          source: "供应商报价单", period: "758", price: 3750, diffPct: 25,
          fetchedAt: now, kind: "fetch",
        },
      ];
    },
    // ── 测算项目与沉淀闭环（mock：内存态）──
    async CostProjectSave(p: CostProject) {
      if (!p.name?.trim()) throw new Error("测算项目需要名称");
      if (!p.id) {
        p = { ...p, id: `proj-mock-${mockProjectSeq++}`, createdAt: new Date().toISOString() };
        mockProjects.push(p);
      } else {
        mockProjects = mockProjects.map((x) => (x.id === p.id ? { ...x, ...p, updatedAt: new Date().toISOString() } : x));
      }
      return p.id;
    },
    async CostProjectList() {
      return mockProjectSummaries();
    },
    async CostProjectGet(id: string) {
      return mockProjects.find((p) => p.id === id) ?? null;
    },
    async CostProjectDelete(id: string) {
      mockProjects = mockProjects.filter((p) => p.id !== id);
      mockItems = mockItems.filter((i) => i.projectId !== id);
      mockVersions = mockVersions.filter((v) => v.projectId !== id);
    },
    async CostEstimateItemSave(i: CostEstimateItem) {
      if (!i.title?.trim()) throw new Error("明细行需要标题");
      const item: CostEstimateItem = {
        ...i,
        amount: (i.quantity ?? 0) * (i.price ?? 0),
        updatedAt: new Date().toISOString(),
      };
      if (!item.id) {
        item.id = mockItemSeq++;
        item.createdAt = new Date().toISOString();
        mockItems.push(item);
      } else {
        mockItems = mockItems.map((x) => (x.id === item.id ? { ...x, ...item } : x));
      }
      return item.id;
    },
    async CostEstimateItemDelete(id: number) {
      mockItems = mockItems.filter((i) => i.id !== id);
    },
    async CostEstimateItems(projectId: string) {
      return mockItems.filter((i) => i.projectId === projectId);
    },
    async CostEstimateVersionSave(projectId: string, note: string) {
      const items = mockItems.filter((i) => i.projectId === projectId);
      if (items.length === 0) throw new Error("项目没有明细行，无法保存版本");
      const versions = mockVersions.filter((v) => v.projectId === projectId);
      const v: CostEstimateVersion = {
        id: mockVersionSeq++,
        projectId,
        version: versions.length + 1,
        total: items.reduce((s, i) => s + (i.amount ?? 0), 0),
        snapshot: JSON.stringify(items),
        note,
        createdAt: new Date().toISOString(),
      };
      mockVersions.push(v);
      return v;
    },
    async CostEstimateVersions(projectId: string) {
      return mockVersions.filter((v) => v.projectId === projectId).sort((a, b) => b.version - a.version);
    },
    async CostEstimateSediment(projectId: string, itemIds: number[]) {
      // mock：与 CostSave 同口径 no-op，仅按缺单价过滤后返回条数。
      return mockItems.filter((i) => i.projectId === projectId && itemIds.includes(i.id ?? -1) && (i.price ?? 0) > 0).length;
    },
    // ── 造价参考与复盘笔记（mock）──
    async CostIndicators(group: string) {
      return mockIndicators(group);
    },
    // v4.6.1 归因对标 mock：无参考样本时的空报告（dev 演示不编造数据）。
    async CostAttribution(projectId: string) {
      const p = mockProjects.find((x) => x.id === projectId);
      return {
        projectId,
        projectName: p?.name ?? projectId,
        totalAmount: 0,
        refTotal: 0,
        totalDiff: 0,
        totalDiffPct: 0,
        items: [],
        topDrivers: [],
        summary: "暂无归因数据（需要参考项目与明细行）",
      };
    },
    async CostNoteSave(n: CostReviewNote) {
      if (!n.title?.trim()) throw new Error("复盘笔记需要标题");
      const now = new Date().toISOString();
      if (!n.id) {
        n = { ...n, id: mockNoteSeq++, status: n.status || "草稿", confidence: n.confidence || "中", createdAt: now, updatedAt: now };
        mockNotes.push(n);
      } else {
        mockNotes = mockNotes.map((x) => (x.id === n.id ? { ...x, ...n, updatedAt: now } : x));
      }
      return n.id!;
    },
    async CostNoteList(query: string, status: string) {
      const q = (query ?? "").toLowerCase();
      return mockNotes
        .filter((n) => (status && status !== "all" ? n.status === status : true))
        .filter((n) => !q || [n.title, n.conclusion, n.boundary, n.risk, n.evidence].some((s) => (s ?? "").toLowerCase().includes(q)))
        .sort((a, b) => (b.updatedAt ?? "").localeCompare(a.updatedAt ?? ""));
    },
    async CostNoteDelete(id: number) {
      mockNotes = mockNotes.filter((n) => n.id !== id);
    },
    async CostNoteBumpRef(id: number) {
      mockNotes = mockNotes.map((n) => (n.id === id ? { ...n, refCount: (n.refCount ?? 0) + 1 } : n));
    },
    // ── v4.50 询价飞轮 mock（四源归一数据点：种子含 15 天内到期 + 水泥调差场景）──
    async CostInquiryList(query: string, limit: number): Promise<CostInquiryRecord[]> {
      seedInquiries();
      const q = (query ?? "").trim().toLowerCase();
      const rows = mockInquiries
        .filter((r) => !q || [r.title, r.spec, r.supplier, r.region].some((s) => (s ?? "").toLowerCase().includes(q)))
        .sort((a, b) => (b.priceDate ?? "").localeCompare(a.priceDate ?? ""));
      return !limit || limit <= 0 ? rows : rows.slice(0, limit);
    },
    async CostInquirySave(r: CostInquiryRecord): Promise<number> {
      seedInquiries();
      const now = new Date().toISOString();
      if (r.id && mockInquiries.some((x) => x.id === r.id)) {
        mockInquiries = mockInquiries.map((x) => (x.id === r.id ? { ...x, ...r, updatedAt: now } : x));
        return r.id;
      }
      const id = mockInquirySeq++;
      mockInquiries = [...mockInquiries, { ...r, id, createdAt: now, updatedAt: now }];
      return id;
    },
    async CostInquiryDelete(id: number) {
      mockInquiries = mockInquiries.filter((x) => x.id !== id);
    },
    async CostInquiryExpiring(days: number): Promise<CostInquiryRecord[]> {
      seedInquiries();
      const today = new Date(); today.setHours(0, 0, 0, 0);
      const horizon = new Date(today.getTime() + Math.max(0, days) * 86400000);
      return mockInquiries.filter((r) => {
        if (!r.validUntil) return false; // 空 = 长期有效
        const until = new Date(r.validUntil + "T00:00:00");
        if (Number.isNaN(until.getTime())) return false;
        return until >= today && until <= horizon;
      });
    },
    async CostInquiryScan(): Promise<CostInquiryScanFinding[]> {
      // 浏览器演示态：样例数据点亮两条发现（离散 + 过期），其余检查静默。
      seedInquiries();
      const rows: CostInquiryScanFinding[] = [];
      const byTitle = new Map<string, CostInquiryRecord[]>();
      for (const r of mockInquiries) {
        if ((r.price ?? 0) <= 0) continue;
        const list = byTitle.get(r.title) ?? [];
        list.push(r);
        byTitle.set(r.title, list);
      }
      for (const [title, recs] of byTitle) {
        if (recs.length < 2) continue;
        const prices = recs.map((r) => r.price ?? 0);
        const min = Math.min(...prices);
        const max = Math.max(...prices);
        if (min > 0 && max / min >= 1.5) {
          rows.push({
            kind: "离散", severity: max / min >= 2 ? "异常" : "关注", title,
            detail: `${recs.length} 个数据点价差 ${(max / min).toFixed(1)} 倍——同名可能不同规格或录入有误`,
            refIds: recs.map((r) => r.id),
          });
        }
      }
      const today = new Date().toISOString().slice(0, 10);
      for (const r of mockInquiries) {
        if (r.validUntil && r.validUntil < today && (r.status ?? "现行") === "现行") {
          rows.push({ kind: "过期", severity: "关注", title: r.title, detail: `有效期至 ${r.validUntil} 已过，状态仍为「现行」`, refIds: [r.id] });
        }
      }
      return rows;
    },
    async CostInquiryAdjust(): Promise<CostAdjustSuggestion[]> {
      seedInquiries();
      const rows: CostAdjustSuggestion[] = [];
      for (const e of costMock) {
        const points = mockInquiries
          .filter((r) => r.title === e.title && (r.price ?? 0) > 0)
          .sort((a, b) => (b.priceDate ?? "").localeCompare(a.priceDate ?? ""));
        const latest = points[0];
        if (!latest || !e.price) continue;
        const diff = latest.price - e.price;
        const diffPct = (diff / e.price) * 100;
        if (Math.abs(diffPct) <= 2) continue;
        rows.push({
          entryName: e.name, entryTitle: e.title, entryPrice: e.price,
          latestPrice: latest.price, latestDate: latest.priceDate, latestSource: latest.source,
          diff, diffPct: Math.round(diffPct * 10) / 10, unit: e.unit,
        });
      }
      return rows;
    },
    // ── v4.50 五算对比 mock（浏览器 dev 空态：阶段值经 CostStageSave 内存累积）──
    async CostStageSave(v: CostStageValue) {
      if (!v.projectId || !v.stage) return;
      mockStageValues = [
        ...mockStageValues.filter((s) => !(s.projectId === v.projectId && s.stage === v.stage)),
        { ...v, id: mockStageSeq++, updatedAt: new Date().toISOString() },
      ];
    },
    async CostStages(projectId: string): Promise<CostStageValue[]> {
      return mockStageValues.filter((s) => s.projectId === projectId);
    },
    async CostStageCompare(projectId: string): Promise<CostStageCompareRow[]> {
      const stages = ["估算", "概算", "预算", "结算", "决算"];
      const amountOf = (st: string) => mockStageValues.find((s) => s.projectId === projectId && s.stage === st)?.amount ?? 0;
      let base = 0;
      return stages.map((stage, i) => {
        const amount = amountOf(stage);
        if (i === 0) base = amount;
        const prevStage = i > 0 ? stages[i - 1] : "";
        const prevAmount = i > 0 ? amountOf(prevStage) : 0;
        const chainDiff = i > 0 && prevAmount ? amount - prevAmount : 0;
        const baseDiff = i > 0 && base ? amount - base : 0;
        return {
          stage, amount,
          hasValue: mockStageValues.some((s) => s.projectId === projectId && s.stage === stage),
          prevStage, hasPrev: i > 0,
          chainDiff, chainDiffPct: i > 0 && prevAmount ? Math.round((chainDiff / prevAmount) * 1000) / 10 : 0,
          baseDiff, baseDiffPct: i > 0 && base ? Math.round((baseDiff / base) * 1000) / 10 : 0,
        };
      });
    },
    async CostStageDeviations(projectId: string): Promise<CostStageDeviation[]> {
      void projectId;
      return [];
    },
    // ── v4.158 组价确认记录（mock）：Compose/ComposeApply 在 mock 域本无桩
    // （dev 浏览器模式无 AI 组价演示数据），Records 只给空数组桩 = 条目详情
    // 「组价依据」折叠区零噪音，不编造历史记录（诚实纪律）。
    async CostComposeRecords(_entryName: string) {
      return [];
    },
    // ── v4.8 成本知识图谱（mock：与后端同形 JSON 串，tree/entry 两种 scope 演示）──
    async CostGraph(scope: string, focus: string, limit: number): Promise<string> {
      const lim = !limit || limit <= 0 || limit > 600 ? 600 : limit;
      const full = mockGraphView(scope, focus);
      // 与后端同口径：节点超过 limit 截断并置 Truncated（边过滤悬挂端点）。
      const truncated = full.nodes.length > lim;
      const nodes = full.nodes.slice(0, lim);
      const ids = new Set(nodes.map((n) => n.id));
      const edges = truncated ? full.edges.filter((e) => ids.has(e.source) && ids.has(e.target)) : full.edges;
      return JSON.stringify({
        nodes,
        edges,
        stats: { truncated, nodeCount: nodes.length, edgeCount: edges.length, countsByType: countByType(nodes) },
      } satisfies CostGraphView);
    },
    // ── AI 组价双件（批 46 NOT_MOCKED 收尾刀）：浏览器 dev 无 LLM 内核，诚实拒。──
    async CostCompose(_desc: string, _unit: string) {
      throw new Error("dev mock：AI 组价需真实 LLM 内核");
    },
    async CostComposeApply(_v: unknown) {
      throw new Error("dev mock：AI 组价应用需真实 LLM 内核（先组价成功才有可应用结果）");
    },

    // ── 工料法：工料机资源库 ──────────────────────────────────────
    async WorkcostResourceSave(r: WorkcostResource): Promise<WorkcostResource> {
      ensureWorkcostSeed();
      const idx = mockResources.findIndex((x) => x.id > 0 && x.id === r.id);
      if (idx >= 0) {
        // 同身份（类别+名称+规格+单位）改价复用编码，对齐 Go 侧身份唯一语义。
        const saved = { ...mockResources[idx], ...r, id: mockResources[idx].id };
        mockResources[idx] = saved;
        return saved;
      }
      const code = r.code || `P${String(mockResourceSeq).padStart(3, "0")}`;
      const saved: WorkcostResource = { ...r, id: mockResourceSeq++, code };
      mockResources.push(saved);
      return saved;
    },
    async WorkcostResourceGet(id: number): Promise<WorkcostResource | null> {
      ensureWorkcostSeed();
      return mockResources.find((x) => x.id === id) ?? null;
    },
    async WorkcostResourceList(kind: string, keyword: string): Promise<WorkcostResource[]> {
      ensureWorkcostSeed();
      const kw = keyword.trim().toLowerCase();
      return mockResources.filter((r) => {
        if (kind && kind !== "全部" && r.kind !== kind) return false;
        if (!kw) return true;
        return [r.title, r.spec, r.code, r.categoryPath].some((v) => (v ?? "").toLowerCase().includes(kw));
      });
    },
    async WorkcostResourceDelete(id: number, force: boolean): Promise<void> {
      ensureWorkcostSeed();
      const r = mockResources.find((x) => x.id === id);
      if (!r) return;
      // 引用完整性：被定额引用时拒绝（与 Go 侧同语义），force=true 才删。
      const refs = mockQuotas.filter((q) => (q.items ?? []).some((i) => i.resourceCode === r.code));
      if (refs.length > 0 && !force) {
        throw new Error(`资源「${r.title}」仍被 ${refs.length} 条消耗定额引用，不能删除（可改为归档）`);
      }
      mockResources = mockResources.filter((x) => x.id !== id);
      mockResourcePrices = mockResourcePrices.filter((p) => p.resourceId !== id);
    },
    async WorkcostResourceSetPrice(id: number, price: number, period: string, region: string, priceType: string, source: string, note: string): Promise<WorkcostResource> {
      ensureWorkcostSeed();
      const r = mockResources.find((x) => x.id === id);
      if (!r) throw new Error("资源不存在");
      if (price <= 0) throw new Error("调价须为正数");
      const now = new Date().toISOString();
      mockResourcePrices.push({
        id: mockResourcePriceSeq++, resourceId: id, price, period, region, priceType,
        source: source || "手动调价", fetchedAt: now, note,
      });
      r.currentPrice = price;
      r.updatedAt = now;
      return r;
    },
    async WorkcostResourcePrices(id: number): Promise<WorkcostResourcePrice[]> {
      ensureWorkcostSeed();
      return mockResourcePrices.filter((p) => p.resourceId === id).reverse();
    },

    // ── 工料法：消耗定额库与核算 ──────────────────────────────────
    async WorkcostQuotaSave(q: WorkcostQuota): Promise<WorkcostQuota> {
      ensureWorkcostSeed();
      const idx = mockQuotas.findIndex((x) => x.code === q.code);
      const saved: WorkcostQuota = { ...q, id: idx >= 0 ? mockQuotas[idx].id : mockQuotas.length + 1 };
      if (idx >= 0) mockQuotas[idx] = saved;
      else mockQuotas.push(saved);
      return saved;
    },
    async WorkcostQuotaGet(code: string): Promise<WorkcostQuota | null> {
      ensureWorkcostSeed();
      return mockQuotas.find((x) => x.code === code) ?? null;
    },
    async WorkcostQuotaList(specialty: string, keyword: string): Promise<WorkcostQuota[]> {
      ensureWorkcostSeed();
      const kw = keyword.trim().toLowerCase();
      return mockQuotas.filter((q) => {
        if (specialty && specialty !== "全部" && q.specialty !== specialty) return false;
        if (!kw) return true;
        return [q.title, q.code, q.chapter].some((v) => (v ?? "").toLowerCase().includes(kw));
      });
    },
    async WorkcostQuotaDelete(code: string): Promise<void> {
      ensureWorkcostSeed();
      mockQuotas = mockQuotas.filter((q) => q.code !== code);
    },
    async WorkcostQuotaCompose(code: string, _overrides: Record<string, WorkcostComposeOverride> | null): Promise<WorkcostCompose> {
      // dev mock 不实现项目级覆盖（真机由 Go 侧 ComposeOverride 生效）。
      return composeMock(code);
    },
    async WorkcostProjectFees(directFee: number, rates: WorkcostRateSet, profitIncludesRegulatory: boolean, measures: number, contingency: number, controlPrice: number): Promise<WorkcostFeeResult> {
      // 与 Go 侧 ComposeProjectFees 同口径：取费只在项目合计层跑一次。
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

    // ── 工料法：存量资源化与核算缓存化 ────────────────────────────
    // dev mock 无真实成本库（costMock 是静态演示数据），资源化与核算缓存化
    // 需要真机数据库，故诚实拒绝——不用假数据假装成功。
    async WorkcostSeedPreview(_includeComposite: boolean) {
      throw new Error("dev mock：存量资源化需真实成本库（Go 后端）");
    },
    async WorkcostSeedApply(_includeComposite: boolean) {
      throw new Error("dev mock：存量资源化需真实成本库（Go 后端）");
    },
    async WorkcostRecompose() {
      throw new Error("dev mock：核算缓存化需真实成本库（Go 后端）");
    },
    // 项目工作簿解析/落库/导出均依赖真实 xlsx 与文件系统 → dev mock 诚实拒。
    async WorkcostProjectParse(_path: string) {
      throw new Error("dev mock：项目工作簿解析需真实文件系统与 Go 后端");
    },
    async WorkcostProjectApply(_path: string) {
      throw new Error("dev mock：项目工作簿落库需真实数据库与 Go 后端");
    },
    async WorkcostProjectExport(_srcPath: string, _outPath: string, _project: string, _location: string, _duration: string) {
      throw new Error("dev mock：五表导出需真实文件系统与 Go 后端");
    },
    async WorkcostProjectExportToWorkspace(_srcPath: string, _fileName: string) {
      throw new Error("dev mock：五表导出需真实文件系统与 Go 后端");
    },

    // ── 工料法④：分部分项清单（内存态，刷新即清——演示录入形态）────
    async WorkcostBillProjects(): Promise<WorkcostBillProject[]> {
      ensureWorkcostSeed();
      return mockBillProjects.map((p) => ({
        ...p,
        itemCount: mockBillItems.filter((b) => b.projectId === p.id).length,
      }));
    },
    async WorkcostBillItems(projectId: number): Promise<WorkcostBillItem[]> {
      ensureWorkcostSeed();
      return mockBillItems.filter((b) => b.projectId === projectId);
    },
    async WorkcostBillItemSave(item: WorkcostBillItem): Promise<WorkcostBillItem> {
      ensureWorkcostSeed();
      const idx = mockBillItems.findIndex(
        (b) => b.projectId === item.projectId && b.code === item.code && item.code !== "",
      );
      if (idx >= 0) {
        mockBillItems[idx] = { ...mockBillItems[idx], ...item, id: mockBillItems[idx].id };
        return mockBillItems[idx];
      }
      const saved: WorkcostBillItem = { ...item, id: mockBillSeq++ };
      if (!saved.code) saved.code = `M${String(mockBillSeq).padStart(3, "0")}`;
      mockBillItems.push(saved);
      return saved;
    },
    async WorkcostBillItemDelete(id: number): Promise<void> {
      ensureWorkcostSeed();
      mockBillItems = mockBillItems.filter((b) => b.id !== id);
    },
    async WorkcostBillProjectDelete(id: number): Promise<number> {
      ensureWorkcostSeed();
      const before = mockBillProjects.length;
      mockBillProjects = mockBillProjects.filter((p) => p.id !== id);
      mockBillItems = mockBillItems.filter((b) => b.projectId !== id);
      return before - mockBillProjects.length >= 0 ? 0 : 0;
    },
    async WorkcostBillProjectRatesSave(
      id: number,
      rates: WorkcostRateSet,
      profitIncludesRegulatory: boolean,
      controlPrice: number,
    ): Promise<void> {
      ensureWorkcostSeed();
      const idx = mockBillProjects.findIndex((p) => p.id === id);
      if (idx < 0) throw new Error("dev mock：项目不存在");
      mockBillProjects[idx] = {
        ...mockBillProjects[idx],
        ...rates,
        profitIncludesRegulatory,
        controlPrice,
      };
    },
  };
}

// mockGraphView 知识图谱演示数据：tree=分类聚合（环上分类+项目节点）；
// entry=围绕 focus 的条目/明细/指标/询价/笔记展开（focus 为项目 ID 或分类路径）。
function mockGraphView(scope: string, focus: string): CostGraphView {
  const nodes: CostGraphView["nodes"] = [];
  const edges: CostGraphView["edges"] = [];
  const addNode = (n: CostGraphView["nodes"][number]) => {
    if (!nodes.some((x) => x.id === n.id)) nodes.push(n);
  };
  const addEdge = (e: CostGraphView["edges"][number]) => {
    if (nodes.some((x) => x.id === e.source) && nodes.some((x) => x.id === e.target)) {
      if (!edges.some((x) => x.type === e.type && x.source === e.source && x.target === e.target)) edges.push(e);
    }
  };
  const category = (path: string, val: number, count: number) => {
    addNode({
      id: "cat:" + path, name: path.split("/").pop() ?? path, type: "category",
      desc: `${count} 条`, val,
      meta: { path, entries: String(count), amount: String(val) },
    });
  };
  const project = (id: string, name: string, total: number, items: number) => {
    addNode({
      id: "proj:" + id, name, type: "project", desc: `${items} 条明细 · 1 版本`, val: total,
      meta: { projectId: id, projectType: "房建", status: "已保存版本", items: String(items), versions: "1" },
    });
  };
  const entry = (name: string, title: string, price: number, path: string, unit: string) => {
    addNode({
      id: "entry:" + name, name: title, type: "entry", desc: `¥${price}/${unit}`, val: price,
      meta: { name, path, unit, source: "手动录入", status: "现行" },
    });
  };
  const item = (projId: string, key: string, title: string, qty: number, price: number, unit: string) => {
    addNode({
      id: `item:${projId}:${key}`, name: title, type: "item", desc: `${qty}${unit ? "/" + unit : ""} × ¥${price}`,
      val: qty * price,
      meta: { projectId: projId, unit, quantity: String(qty), price: String(price), entryName: "" },
    });
  };
  const indicator = (key: string, median: number, samples: number) => {
    addNode({
      id: "ind:" + key, name: key, type: "indicator", desc: `样本 ${samples} · 中位 ¥${median}`, val: median,
      meta: { samples: String(samples), min: String(median * 0.9), max: String(median * 1.1), mean: String(median), p25: String(median * 0.95), p75: String(median * 1.05), unit: "m³" },
    });
  };
  const inquiry = (id: number, title: string, price: number) => {
    addNode({
      id: "inq:" + id, name: title, type: "inquiry", desc: `信息价 · ¥${price} · 2026-08`, val: price,
      meta: { source: "信息价", supplier: "造价信息网", region: "成都", priceDate: "2026-08", validUntil: "", unit: "m³", spec: "", status: "现行" },
    });
  };
  const note = (id: number, title: string) => {
    addNode({
      id: "note:" + id, name: title, type: "note", desc: "高 · 已确认 · 引用 2", val: 2,
      meta: { confidence: "高", status: "已确认", category: "土建", boundary: "泵送 C30", risk: "期数波动", evidence: "厂房 A V1" },
    });
  };

  if (scope === "entry") {
    const isProject = focus.startsWith("proj-");
    // 分类/条目骨架（演示数据恒含一套条目，focus 决定中心）
    category("综合单价", 2780, 3);
    category("综合单价/土建", 1460, 2);
    category("综合单价/机械", 1320, 1);
    entry("c30", "C30 商品混凝土", 480, "综合单价/土建", "m³");
    entry("rebar", "HRB400 钢筋", 250, "综合单价/土建", "t");
    entry("excavator", "挖掘机台班", 1320, "综合单价/机械", "台班");
    addEdge({ source: "cat:综合单价/土建", target: "entry:c30", type: "belongs_to", weight: 1 });
    addEdge({ source: "cat:综合单价/土建", target: "entry:rebar", type: "belongs_to", weight: 1 });
    addEdge({ source: "cat:综合单价/机械", target: "entry:excavator", type: "belongs_to", weight: 1 });
    if (isProject) {
      project(focus, "厂房 A（演示）", 2230, 3);
      item(focus, "i1", "C30 商品混凝土", 10, 500, "m³");
      item(focus, "i2", "HRB400 钢筋", 2, 245, "t");
      addEdge({ source: "proj:" + focus, target: `item:${focus}:i1`, type: "contains", weight: 1 });
      addEdge({ source: "proj:" + focus, target: `item:${focus}:i2`, type: "contains", weight: 1 });
      addEdge({ source: `item:${focus}:i1`, target: "entry:c30", type: "references", weight: 1, meta: { matchedBy: "entry_name" } });
      addEdge({ source: `item:${focus}:i2`, target: "entry:rebar", type: "references", weight: 1, meta: { matchedBy: "title" } });
      addEdge({ source: `item:${focus}:i1`, target: "ind:C30 商品混凝土", type: "benchmarks", weight: 1 });
    } else {
      project("proj-demo-a", "厂房 A（演示）", 2230, 3);
      item("proj-demo-a", "i1", "C30 商品混凝土", 10, 500, "m³");
      addEdge({ source: "proj:proj-demo-a", target: "item:proj-demo-a:i1", type: "contains", weight: 1 });
      addEdge({ source: "item:proj-demo-a:i1", target: "entry:c30", type: "references", weight: 1, meta: { matchedBy: "entry_name" } });
      addEdge({ source: "item:proj-demo-a:i1", target: "ind:C30 商品混凝土", type: "benchmarks", weight: 1 });
    }
    indicator("C30 商品混凝土", 490, 5);
    inquiry(1, "C30 商品混凝土", 470);
    note(1, "C30 泵送价区间");
    addEdge({ source: "inq:1", target: "entry:c30", type: "suggests", weight: 1, meta: { matchedBy: "title" } });
    addEdge({ source: "note:1", target: "cat:综合单价/土建", type: "notes", weight: 1 });
    return { nodes, edges, stats: { truncated: false, nodeCount: nodes.length, edgeCount: edges.length, countsByType: countByType(nodes) } };
  }
  // tree：分类树聚合 + 项目节点（无边）。
  category("综合单价", 2780, 3);
  category("综合单价/土建", 1460, 2);
  category("综合单价/机械", 1320, 1);
  project("proj-demo-a", "厂房 A（演示）", 2230, 3);
  project("proj-demo-b", "厂房 B（演示）", 550, 1);
  return { nodes, edges, stats: { truncated: false, nodeCount: nodes.length, edgeCount: 0, countsByType: countByType(nodes) } };
}

function countByType(nodes: CostGraphView["nodes"]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const n of nodes) out[n.type] = (out[n.type] ?? 0) + 1;
  return out;
}
