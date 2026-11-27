// mock/cost.ts — 造价库（workcost）域 mock（T6-10.1 拆分自 lib/mock.ts）。旧成本库 /
// 价格源 / 组价 / 询价 / 五算对比的 mock 已随旧成本库退役拆除（2026-11-26）。
import type { AppBindings } from "../bridge";
import type { MakeMockState } from "./state";
import type {
  WorkcostBillItem,
  WorkcostBillProject,
  WorkcostBillQuotaLink,
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
  | "SemanticIndexStatus"
  | "CostBulkPricePreview" | "CostBulkPriceApply"
  | "CostQuotaFlywheelPreview" | "CostQuotaFlywheelApply"
  | "CostAuditScan"
  | "CostHealthScan" | "CostDedupeResources"
  // 工料法成本数据库（工料机资源库 + 消耗定额库 + 综合单价核算）：
  // 浏览器 dev 态给可交互的内存实现（资源增删改/调价/按定额核算），
  // 使「工料机」模块在无 Go 后端时仍可用。
  | "WorkcostResourceSave" | "WorkcostResourceGet" | "WorkcostResourceList" | "WorkcostResourceDelete"
  | "WorkcostResourceSetPrice" | "WorkcostResourcePrices"
  | "WorkcostQuotaSave" | "WorkcostQuotaGet" | "WorkcostQuotaList" | "WorkcostQuotaDelete"
  | "WorkcostQuotaCompose" | "WorkcostQuotaComposeMany" | "WorkcostProjectFees"
  | "WorkcostProjectParse" | "WorkcostProjectApply"
  | "WorkcostBillProjects" | "WorkcostBillItems" | "WorkcostBillItemSave" | "WorkcostBillItemDelete"
  | "WorkcostBillProjectRatesSave"
  | "WorkcostBillProjectDelete"
  | "WorkcostBillQuotaLinks" | "WorkcostBillQuotaAttach" | "WorkcostBillQuotaDetach"
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
// 清单↔定额 1:N 引用（内存态；导入种子走单值列兜底，这里只存显式挂接）。
const mockBillLinks: Record<number, WorkcostBillQuotaLink[]> = {};
let mockBillSeq = 1;

// ensureWorkcostSeed 首次访问时播种演示数据（幂等）。
function ensureWorkcostSeed(): void {
  if (mockResources.length > 0) return;
  const add = (code: string, kind: WorkcostKind, title: string, spec: string, unit: string, price: number, source: string): void => {
    mockResources.push({
      id: mockResourceSeq++, code, kind, title, spec, unit,
      basePrice: price, currentPrice: price, categoryPath: `工料机/${kind}`,
      source, supplier: "", region: "四川省", priceDate: "2026年第7期", priceType: "到场价",
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

export function buildCost(_s: MakeMockState): CostMethods {
  return {
    async SemanticIndexStatus() {
      // 浏览器演示态：模型不可用（无本地引擎），诚实提示。
      return { available: false, counts: {}, error: "浏览器演示态无本地语义模型——真机启用 Herdsman bge-m3 后可用" };
    },
    // 资源价批量维护（三期 3.4；演示口径=空报告）。
    async CostBulkPricePreview(_xlsxBase64: string) {
      return { total: 0, matched: [], skipped: [], rowErrors: [] };
    },
    async CostHealthScan() {
      return { zeroPrice: [], quotaNoPrice: [], brokenRef: [], duplicates: [], codeMismatch: [], totalFindings: 0 };
    },
    async CostDedupeResources(_keepId: number, _mergeIds: number[]) {
      return { kept: "", merged: 0, redirectedItems: 0, redirectedPrices: 0 };
    },
    async CostBulkPriceApply(_xlsxBase64: string) {
      return { updated: 0, skipped: [], total: 0, applied: [] };
    },
    // 企业定额自生成（11.6；演示口径=无跨项目数据，零候选）。
    async CostQuotaFlywheelPreview() {
      return { clusters: 0, candidates: [], skipped: [] };
    },
    async CostQuotaFlywheelApply(_origCodes: string[]) {
      return { saved: [], skipped: [] };
    },
    // 跨项目对标查漏（11.3；演示口径=空报告）。
    async CostAuditScan(_projectID: number) {
      return { billGaps: [], quotaGaps: [], auditedProj: "", projectHits: 0 };
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
      const q = mockQuotas.find((x) => x.code === code);
      if (!q) return null;
      return { ...q, usageCount: mockBillItems.filter((b) => b.quotaCode === code).length };
    },
    async WorkcostQuotaList(specialty: string, keyword: string): Promise<WorkcostQuota[]> {
      ensureWorkcostSeed();
      const kw = keyword.trim().toLowerCase();
      const usage = (code: string) => mockBillItems.filter((b) => b.quotaCode === code).length;
      return mockQuotas
        .filter((q) => {
          if (specialty && specialty !== "全部" && q.specialty !== specialty) return false;
          if (!kw) return true;
          return [q.title, q.code, q.chapter].some((v) => (v ?? "").toLowerCase().includes(kw));
        })
        .map((q) => ({ ...q, usageCount: usage(q.code) }));
    },
    async WorkcostQuotaDelete(code: string): Promise<void> {
      ensureWorkcostSeed();
      mockQuotas = mockQuotas.filter((q) => q.code !== code);
    },
    async WorkcostQuotaCompose(code: string, _overrides: Record<string, WorkcostComposeOverride> | null): Promise<WorkcostCompose> {
      // dev mock 不实现项目级覆盖（真机由 Go 侧 ComposeOverride 生效）。
      return composeMock(code);
    },
    async WorkcostQuotaComposeMany(codes: string[]): Promise<Record<string, WorkcostCompose>> {
      ensureWorkcostSeed();
      const out: Record<string, WorkcostCompose> = {};
      for (const code of codes) {
        if (!mockQuotas.some((x) => x.code === code)) continue; // 与 Go 同：缺失不写键
        out[code] = composeMock(code);
      }
      return out;
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

    // 项目工作簿解析/落库依赖真实 xlsx 与数据库 → dev mock 诚实拒。
    // （重新导出五表的 WorkcostProjectExport* 已随旧版渲染器删除，2026-11-27。）
    async WorkcostProjectParse(_path: string) {
      throw new Error("dev mock：项目工作簿解析需真实文件系统与 Go 后端");
    },
    async WorkcostProjectApply(_path: string) {
      throw new Error("dev mock：项目工作簿落库需真实数据库与 Go 后端");
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
      const title = item.title.trim();
      if (!title) throw new Error("清单项需要名称");
      if (item.projectId <= 0) throw new Error("清单项需要归属项目");
      let code = item.code.trim();
      if (!code) {
        let maxSeq = 0;
        for (const b of mockBillItems) {
          if (b.projectId !== item.projectId) continue;
          const m = /^M(\d{3})$/.exec(b.code);
          if (m) maxSeq = Math.max(maxSeq, Number(m[1]));
        }
        code = `M${String(maxSeq + 1).padStart(3, "0")}`;
      }
      const existing = mockBillItems.find((b) => b.projectId === item.projectId && b.code === code);
      if (existing) {
        Object.assign(existing, { ...item, id: existing.id, code, title });
        return existing;
      }
      const saved: WorkcostBillItem = {
        ...item,
        id: mockBillSeq++,
        code,
        title,
        quotaCodes: item.quotaCodes ?? [],
      };
      mockBillItems.push(saved);
      return saved;
    },
    async WorkcostBillItemDelete(id: number): Promise<void> {
      mockBillItems = mockBillItems.filter((b) => b.id !== id);
      delete mockBillLinks[id];
    },
    async WorkcostBillQuotaLinks(billItemId: number): Promise<WorkcostBillQuotaLink[]> {
      return (mockBillLinks[billItemId] ?? []).slice().sort((a, b) => a.sort - b.sort);
    },
    async WorkcostBillQuotaAttach(billItemId: number, quotaCode: string, quantity: number): Promise<WorkcostBillQuotaLink> {
      const links = (mockBillLinks[billItemId] ??= []);
      const found = links.find((l) => l.quotaCode === quotaCode);
      if (found) {
        found.quantity = quantity;
        found.updatedAt = new Date().toISOString();
        return found;
      }
      const link: WorkcostBillQuotaLink = {
        id: Math.max(0, ...links.map((l) => l.id)) + 1,
        billItemId, quotaCode, quantity,
        sort: links.length + 1,
        createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
      };
      links.push(link);
      return link;
    },
    async WorkcostBillQuotaDetach(billItemId: number, linkId: number): Promise<void> {
      mockBillLinks[billItemId] = (mockBillLinks[billItemId] ?? []).filter((l) => l.id !== linkId);
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
