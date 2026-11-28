// mock/cost.ts — 造价数据库域（浏览器 dev 态的内存实现）。
//
// 2026-11-25 造价数据库推倒重建：板块只剩三级浏览面（清单 / 定额 / 工料机），
// 这里只实现它真实消费的绑定——读三级数据、现算综合单价、删除、导出、体检。
// 旧板块的资料条目 / 价格源 / 询价 / 导入 / 编辑类 mock 随界面一并退役。
//
// 演示数据刻意取自实测模版口径（旺平矿业/什邡五表产物）：资源编码用助记码
// （L01/L02/EXC/C20/HDPE），综合单价只含人材机。写文件类动作（导出 xlsx）
// 在浏览器里没有文件系统，诚实拒绝——不假装成功。
import type { AppBindings } from "../bridge";
import type { MakeMockState } from "./state";
import type {
  CostIndexStatusView,
  WorkcostBillItem,
  WorkcostBillProject,
  WorkcostBillQuotaLink,
  WorkcostCompose,
  WorkcostComposeLineView,
  WorkcostComposeOverride,
  WorkcostDictItem,
  WorkcostKind,
  WorkcostLibraryFilter,
  WorkcostQuota,
  WorkcostQuotaItem,
  WorkcostQuotaRegionFix,
  WorkcostRateSet,
  WorkcostResource,
  WorkcostResourcePrice,
  WorkcostUsageRef,
} from "../types";

type CostMethods = Pick<
  AppBindings,
  | "CostHealthScan" | "CostLibraryExport" | "CostIndexStatus" | "CostIndexBackfill"
  | "WorkcostResourceList" | "WorkcostResourceGet" | "WorkcostResourceDelete" | "WorkcostResourcePrices" | "WorkcostResourceUsage"
  | "WorkcostResourceSave" | "WorkcostResourceSetPrice"
  | "WorkcostQuotaGet" | "WorkcostQuotaList" | "WorkcostQuotaDelete" | "WorkcostQuotaSave"
  | "WorkcostQuotaCompose" | "WorkcostQuotaComposeMany"
  | "WorkcostBillProjects" | "WorkcostBillItems" | "WorkcostBillItemDelete" | "WorkcostBillProjectDelete"
  | "WorkcostBillItemSave" | "WorkcostBillQuotaLinks" | "WorkcostBillQuotaAttach" | "WorkcostBillQuotaDetach"
  | "WorkcostBillProjectRatesSave" | "WorkcostBillProjectProfileSave"
  | "WorkcostBillSearch"
  // 库维护（2026-11-28）：字典读写、定额解除全部挂接、定额地区回填。
  | "WorkcostDictList" | "WorkcostDictSet" | "WorkcostQuotaDetachEverywhere" | "WorkcostQuotaRegionBackfill"
>;

// ── 工料法 mock 状态（工料机资源库 + 消耗定额库 + 清单，浏览器内存态）──
let mockResources: WorkcostResource[] = [];
let mockResourceSeq = 1;
let mockResourcePrices: WorkcostResourcePrice[] = [];
let mockBillProjects: WorkcostBillProject[] = [];
let mockBillItems: WorkcostBillItem[] = [];
let mockQuotas: WorkcostQuota[] = [];
let mockBillSeq = 1;
// 清单↔定额挂接（编辑面用；由种子清单行的 quotaCodes 播种，Attach/Detach 同步维护）。
let mockBillLinks: WorkcostBillQuotaLink[] = [];
let mockLinkSeq = 1;
// 字典（规范值）与检索别名的演示态：首次访问时从已有数据机械收一遍（真实库升级时的做法）。
const mockDictItems: WorkcostDictItem[] = [];
let mockDictSeq = 1;
function ensureDictSeed(): void {
  if (mockDictItems.length > 0) return;
  const push = (kind: string, name: string, parent = ""): void => {
    if (name && !mockDictItems.some((d) => d.kind === kind && d.name === name)) {
      mockDictItems.push({ id: mockDictSeq++, kind, name, parent });
    }
  };
  for (const p of mockBillProjects) {
    push("specialty", p.specialty);
    push("region", p.region);
  }
  for (const q of mockQuotas) push("specialty", q.specialty);
  for (const r of mockResources) {
    push("region", r.region);
    push("category", r.categoryPath);
  }
}

// ensureWorkcostSeed 首次访问时播种演示数据（幂等）。
function ensureWorkcostSeed(): void {
  if (mockResources.length > 0) return;
  const kindSeg: Record<string, string> = { "人工": "R", "材料": "C", "机械": "J", "外委": "W" };
  const gseq: Record<string, number> = {};
  const add = (code: string, kind: WorkcostKind, title: string, spec: string, unit: string, price: number, source: string): void => {
    const slot = kindSeg[kind] ?? "X";
    gseq[slot] = (gseq[slot] ?? 0) + 1;
    mockResources.push({
      id: mockResourceSeq++, code, gcode: `R${slot}01${String(gseq[slot]).padStart(4, "0")}`,
      kind, title, spec, unit,
      basePrice: price, currentPrice: price, categoryPath: `工料机/${kind}`,
      source, supplier: "", region: "四川省", priceDate: "2026-07", priceType: "到场价",
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
    profitIncludesRegulatory: false, controlPrice: 0, specialty: "土壤修复", region: "四川省", pricePeriod: "2026-07", itemCount: 2,
    createdAt: "2026-10-07T10:00:00Z", updatedAt: "2026-10-07T10:00:00Z",
  }];
  mockBillItems = [
    { id: mockBillSeq++, projectId: 1, code: "WP01", gcode: "Q0001001", title: "施工便道（铺筑压实）", unit: "m", division: "A临建", quantity: 850, quantityExpr: "长850×宽1", quotaCode: "WP01", feature: "泥结碎石", priceOverride: 0, listPrice: 420, sort: 1, quotaCodes: ["WP01"] },
    { id: mockBillSeq++, projectId: 1, code: "WP02", gcode: "Q0001002", title: "场地平整", unit: "m²", division: "A临建", quantity: 12000, quantityExpr: "", quotaCode: "WP02", feature: "", priceOverride: 0, sort: 2, quotaCodes: ["WP02"] },
    { id: mockBillSeq++, projectId: 1, code: "M001", gcode: "Q0002001", title: "外委检测（手填）", unit: "项", division: "D外运处置", quantity: 1, quantityExpr: "", quotaCode: "", feature: "", priceOverride: 5000, sort: 3, quotaCodes: [] },
  ];
  add("LDR", "机械", "装载机50型台班", "折旧700+柴油×70L+司机", "台班", 1445.9, "台班公式");
  add("C20", "材料", "C20商品混凝土", "泵送到场", "m³", 345, "商品混凝土C30减20");
  add("AGG", "材料", "级配碎石", "便道基层到场", "m³", 78, "估价");
  add("HDPE", "材料", "HDPE防渗膜1.5mm", "双光面", "m²", 12, "成本库");
  add("KILN", "外委", "水泥窑协同处置", "不含运（用户确认价）", "t", 190, "外委");
  // 零价资源：体检「零价资源」分区的演示样本。
  add("ZERO", "材料", "待询价材料", "尚未取得报价", "t", 0, "—");

  const byCode = (c: string): WorkcostResource | undefined => mockResources.find((r) => r.code === c);
  const item = (code: string, qty: number): WorkcostQuotaItem => {
    const r = byCode(code);
    // 演示数据的类别都在四类之内（读侧类型放宽是为了存量行，见 WorkcostKindRead）。
    const kind = (r?.kind ?? "材料") as WorkcostKind;
    return {
      quotaCode: "", resourceCode: code, kind,
      title: r?.title ?? code, unit: r?.unit ?? "", quantity: qty, resourcePrice: 0, lossRate: 0,
    };
  };
  const quota = (id: number, code: string, title: string, chapter: string, unit: string, note: string, items: WorkcostQuotaItem[]): WorkcostQuota => ({
    id, code, gcode: `D01${String(id).padStart(2, "0")}${String(id).padStart(3, "0")}`, title, specialty: "土壤修复", chapter, unit,
    categoryPath: `综合单价/${chapter}`, baseLabor: 0, baseMaterial: 0, baseMachine: 0,
    source: "项目导入（演示）", region: "", priceDate: "", note, status: "现行", items,
  });
  mockQuotas = [
    quota(1, "WP01", "施工便道", "A临建", "m", "3m宽，20cm碎石+15cm混凝土", [item("L01", 0.03), item("AGG", 0.75), item("C20", 0.45), item("LDR", 0.005)]),
    quota(2, "WP02", "场地平整", "A临建", "m²", "推土机平整", [item("L02", 0.002), item("EXC", 0.001)]),
    quota(3, "WP19", "水泥窑协同处置", "D外运处置", "t", "不含运 190 元/t", [item("KILN", 1)]),
  ];
  // 挂接播种：清单行的 quotaCodes → 引用行（quantity=0 跟随清单工程量）。
  for (const b of mockBillItems) {
    for (const code of b.quotaCodes ?? []) {
      mockBillLinks.push({ id: mockLinkSeq++, billItemId: b.id, quotaCode: code, quantity: 0, sort: mockLinkSeq });
    }
  }
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

// quotaUsedBy 定额被哪些清单行引用（与页面同口径：挂接码集合，旧数据回退单值列）。
function quotaUsedBy(code: string): WorkcostBillItem[] {
  return mockBillItems.filter((b) => {
    const linked = b.quotaCodes ?? [];
    const codes = linked.length > 0 ? linked : [b.quotaCode];
    return codes.includes(code);
  });
}

export function buildCost(_s: MakeMockState): CostMethods {
  return {
    // ── 体检（只读）──
    async CostHealthScan() {
      ensureWorkcostSeed();
      const zeroPrice = mockResources
        .filter((r) => r.currentPrice <= 0 && r.basePrice <= 0)
        .map((r) => ({ id: r.id, code: r.code, title: r.title, kind: r.kind, unit: r.unit, detail: "现行价与基准价均为 0" }));
      return { zeroPrice, quotaNoPrice: [], brokenRef: [], duplicates: [], codeMismatch: [], totalFindings: zeroPrice.length };
    },
    async CostLibraryExport(
      _kind: string, _filter: WorkcostLibraryFilter, _outPath: string,
    ): Promise<string> {
      // 浏览器 dev 态没有文件系统：诚实拒，不假装导出成功（桌面端由 Go 侧写 xlsx）。
      throw new Error("浏览器 dev 态不支持导出 xlsx，请在桌面端使用");
    },

    // ── 工料机 ──
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
      if (!r) throw new Error("资源不存在");
      // 引用完整性：被定额引用时拒绝（与 Go 侧同语义），force=true 才删。
      const refs = mockQuotas.filter((q) => (q.items ?? []).some((i) => i.resourceCode === r.code));
      if (refs.length > 0 && !force) {
        throw new Error(`资源「${r.title}」仍被 ${refs.length} 条消耗定额引用，不能删除`);
      }
      mockResources = mockResources.filter((x) => x.id !== id);
      mockResourcePrices = mockResourcePrices.filter((p) => p.resourceId !== id);
    },
    async WorkcostResourcePrices(id: number): Promise<WorkcostResourcePrice[]> {
      ensureWorkcostSeed();
      return mockResourcePrices.filter((p) => p.resourceId === id).reverse();
    },
    async WorkcostResourceUsage(code: string): Promise<WorkcostUsageRef[]> {
      ensureWorkcostSeed();
      // 按定额含量行的资源编码反查（谁在用这条工料机）。
      const refs: WorkcostUsageRef[] = [];
      for (const q of mockQuotas) {
        for (const it of q.items ?? []) {
          if (it.resourceCode === code) {
            refs.push({ id: q.id ?? 0, code: q.code, title: q.title, unit: q.unit, quantity: it.quantity });
          }
        }
      }
      return refs;
    },
    async WorkcostResourceGet(id: number): Promise<WorkcostResource | null> {
      ensureWorkcostSeed();
      return mockResources.find((x) => x.id === id) ?? null;
    },
    async WorkcostResourceSave(r: WorkcostResource): Promise<WorkcostResource> {
      ensureWorkcostSeed();
      const title = r.title.trim();
      if (!title) throw new Error("工料机资源需要名称");
      const at = mockResources.findIndex((x) => x.id === r.id);
      if (at >= 0) {
        // 同行更新（Go 侧另按身份防撞别行；mock 简化为按 id 定点改）。
        mockResources[at] = { ...mockResources[at], ...r, title };
        return mockResources[at];
      }
      throw new Error("浏览器演示态不支持新建工料机，请在桌面端操作");
    },
    async WorkcostResourceSetPrice(id, price, period, region, priceType, source, note): Promise<WorkcostResource> {
      ensureWorkcostSeed();
      const r = mockResources.find((x) => x.id === id);
      if (!r) throw new Error("资源不存在");
      if (price <= 0) throw new Error("调价需要正的价格");
      mockResourcePrices.push({
        id: mockResourcePrices.length + 1, resourceId: id, price,
        period: period.trim(), region: region.trim(), priceType: priceType.trim(),
        source: source.trim(), fetchedAt: new Date().toISOString().slice(0, 16).replace("T", " "), note: note.trim(),
      });
      r.currentPrice = price;
      return r;
    },

    // ── 定额 ──
    async WorkcostQuotaGet(code: string): Promise<WorkcostQuota | null> {
      ensureWorkcostSeed();
      const q = mockQuotas.find((x) => x.code === code);
      if (!q) return null;
      return { ...q, items: q.items ?? [], usageCount: quotaUsedBy(code).length };
    },
    async WorkcostQuotaList(specialty: string, keyword: string): Promise<WorkcostQuota[]> {
      ensureWorkcostSeed();
      const kw = keyword.trim().toLowerCase();
      return mockQuotas
        .filter((q) => {
          if (specialty && specialty !== "全部" && q.specialty !== specialty) return false;
          if (!kw) return true;
          return [q.title, q.code, q.chapter].some((v) => (v ?? "").toLowerCase().includes(kw));
        })
        .map((q) => ({ ...q, items: [], usageCount: quotaUsedBy(q.code).length }));
    },
    async WorkcostQuotaDelete(code: string): Promise<void> {
      ensureWorkcostSeed();
      // 与 Go 绑定同语义：仍被清单项引用时拒绝，不留悬空引用。
      const used = quotaUsedBy(code).length;
      if (used > 0) throw new Error(`该定额仍被 ${used} 条清单项引用，不能删除`);
      mockQuotas = mockQuotas.filter((q) => q.code !== code);
    },
    async WorkcostQuotaSave(q: WorkcostQuota): Promise<WorkcostQuota> {
      ensureWorkcostSeed();
      const code = q.code.trim();
      const title = q.title.trim();
      if (!code || !title) throw new Error("定额需要编码与名称");
      const at = mockQuotas.findIndex((x) => x.code === code);
      const items = (q.items ?? []).map((it, i) => ({
        ...it, quotaCode: code, sort: i,
        resourcePrice: it.resourcePrice ?? 0, lossRate: it.lossRate ?? 0,
      }));
      if (at >= 0) {
        mockQuotas[at] = { ...mockQuotas[at], ...q, code, title, items };
        return { ...mockQuotas[at], usageCount: quotaUsedBy(code).length };
      }
      throw new Error("浏览器演示态不支持新建定额，请在桌面端操作");
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

    // ── 清单 ──
    async WorkcostBillProjects(): Promise<WorkcostBillProject[]> {
      ensureWorkcostSeed();
      return mockBillProjects.map((p) => ({
        ...p,
        itemCount: mockBillItems.filter((b) => b.projectId === p.id).length,
      }));
    },
    // ── 语义索引（浏览器 dev 态没有本地模型：如实报未建，不假装就绪）──
    async CostIndexStatus(): Promise<CostIndexStatusView> {
      ensureWorkcostSeed();
      const missing = "浏览器演示态无本地语义模型";
      return {
        model: "bge-m3",
        running: false,
        kinds: [
          { kind: "wc_quota", label: "定额", total: mockQuotas.length, count: 0, updatedAt: "", lastError: missing },
          { kind: "wc_res", label: "工料机", total: mockResources.length, count: 0, updatedAt: "", lastError: missing },
          { kind: "wc_bill", label: "清单", total: mockBillItems.length, count: 0, updatedAt: "", lastError: missing },
        ],
      };
    },
    async CostIndexBackfill(_kind: string): Promise<CostIndexStatusView> {
      throw new Error("浏览器演示态无本地语义模型，补齐不可用");
    },
    // WorkcostBillSearch 清单行跨项目检索（演示态只有子串匹配，没有语义路）。
    async WorkcostBillSearch(keyword: string, _specialty: string, projectId: number): Promise<WorkcostBillItem[]> {
      ensureWorkcostSeed();
      const kw = keyword.trim();
      return mockBillItems.filter((b) =>
        (projectId === 0 || b.projectId === projectId)
        && (kw === "" || b.title.includes(kw) || b.code.includes(kw) || (b.feature ?? "").includes(kw)));
    },
    // ── 库维护（2026-11-28）：字典 / 解除全部挂接 / 定额地区回填 ──
    async WorkcostDictList(kind: string): Promise<WorkcostDictItem[]> {
      ensureWorkcostSeed();
      ensureDictSeed();
      return mockDictItems.filter((d) => kind === "" || d.kind === kind);
    },
    async WorkcostDictSet(item: WorkcostDictItem, remove: boolean): Promise<void> {
      ensureWorkcostSeed();
      ensureDictSeed();
      const name = item.name.trim();
      const at = mockDictItems.findIndex((d) => d.kind === item.kind && d.name === name);
      if (remove) {
        if (at < 0) throw new Error(`字典里没有「${name}」`);
        mockDictItems.splice(at, 1);
        return;
      }
      if (!name) throw new Error("新增字典项需要名称");
      if (item.kind === "alias" && !item.parent?.trim()) throw new Error("新增检索别名需要写明它对应的标准说法");
      if (at < 0) mockDictItems.push({ id: mockDictSeq++, kind: item.kind, name, parent: item.parent ?? "", note: item.note ?? "" });
    },
    async WorkcostQuotaDetachEverywhere(code: string): Promise<number> {
      ensureWorkcostSeed();
      let n = 0;
      for (const b of mockBillItems) {
        if (b.quotaCode !== code && !(b.quotaCodes ?? []).includes(code)) continue;
        n++;
        b.quotaCode = "";
        b.quotaCodes = (b.quotaCodes ?? []).filter((c) => c !== code);
      }
      mockBillLinks = mockBillLinks.filter((l) => l.quotaCode !== code);
      return n;
    },
    async WorkcostQuotaRegionBackfill(dryRun: boolean): Promise<WorkcostQuotaRegionFix[]> {
      ensureWorkcostSeed();
      const out: WorkcostQuotaRegionFix[] = [];
      for (const q of mockQuotas) {
        if (q.region) continue;
        const used = mockBillItems.filter((b) => b.quotaCode === q.code || (b.quotaCodes ?? []).includes(q.code));
        const regions = [...new Set(used.map((b) => mockBillProjects.find((p) => p.id === b.projectId)?.region ?? "").filter(Boolean))];
        const fix: WorkcostQuotaRegionFix = { code: q.code, title: q.title, links: used.length, applied: false };
        if (used.length === 0) fix.reason = "没有被清单引用，没有上游项目可依据";
        else if (regions.length === 0) fix.reason = "引用它的清单项目没标地区";
        else if (regions.length > 1) fix.reason = `被 ${regions.length} 个省的项目引用（${regions.join("、")}），不猜`;
        else {
          fix.region = regions[0];
          if (!dryRun) {
            q.region = regions[0];
            fix.applied = true;
          }
        }
        out.push(fix);
      }
      return out;
    },
    async WorkcostBillItems(projectId: number): Promise<WorkcostBillItem[]> {
      ensureWorkcostSeed();
      return mockBillItems.filter((b) => b.projectId === projectId);
    },
    async WorkcostBillItemDelete(id: number): Promise<void> {
      mockBillItems = mockBillItems.filter((b) => b.id !== id);
      mockBillLinks = mockBillLinks.filter((l) => l.billItemId !== id);
    },
    async WorkcostBillItemSave(item: WorkcostBillItem): Promise<WorkcostBillItem> {
      ensureWorkcostSeed();
      const title = item.title.trim();
      if (!title) throw new Error("清单项需要名称");
      const at = mockBillItems.findIndex((b) => b.id === item.id || (b.projectId === item.projectId && b.code === item.code));
      if (at < 0) throw new Error("浏览器演示态不支持新建清单项，请在桌面端操作");
      mockBillItems[at] = { ...mockBillItems[at], ...item, id: mockBillItems[at].id, title };
      return mockBillItems[at];
    },
    async WorkcostBillQuotaLinks(billItemId: number): Promise<WorkcostBillQuotaLink[]> {
      ensureWorkcostSeed();
      return mockBillLinks.filter((l) => l.billItemId === billItemId);
    },
    async WorkcostBillQuotaAttach(billItemId: number, quotaCode: string, quantity: number): Promise<WorkcostBillQuotaLink> {
      ensureWorkcostSeed();
      const code = quotaCode.trim();
      if (!code) throw new Error("挂接需要定额编码");
      if (!mockQuotas.some((x) => x.code === code)) throw new Error(`定额 ${code} 不存在，无法挂接`);
      const at = mockBillLinks.findIndex((l) => l.billItemId === billItemId && l.quotaCode === code);
      if (at >= 0) {
        mockBillLinks[at] = { ...mockBillLinks[at], quantity: Math.max(0, quantity) };
      } else {
        mockBillLinks.push({ id: mockLinkSeq++, billItemId, quotaCode: code, quantity: Math.max(0, quantity), sort: mockLinkSeq });
      }
      // 清单行的 quotaCodes 聚合同步（读取侧口径）。
      const b = mockBillItems.find((x) => x.id === billItemId);
      if (b && !(b.quotaCodes ?? []).includes(code)) b.quotaCodes = [...(b.quotaCodes ?? []), code];
      return mockBillLinks.find((l) => l.billItemId === billItemId && l.quotaCode === code)!;
    },
    async WorkcostBillQuotaDetach(billItemId: number, linkId: number): Promise<void> {
      ensureWorkcostSeed();
      const link = mockBillLinks.find((l) => l.id === linkId && l.billItemId === billItemId);
      if (!link) throw new Error("引用不存在或不属于该清单项");
      mockBillLinks = mockBillLinks.filter((l) => l.id !== linkId);
      const b = mockBillItems.find((x) => x.id === billItemId);
      if (b) b.quotaCodes = (b.quotaCodes ?? []).filter((c) => c !== link.quotaCode);
    },
    async WorkcostBillProjectRatesSave(id: number, rates: WorkcostRateSet, profitIncludesRegulatory: boolean, controlPrice: number): Promise<void> {
      ensureWorkcostSeed();
      const p = mockBillProjects.find((x) => x.id === id);
      if (!p) throw new Error("项目不存在");
      p.managementRate = rates.managementRate;
      p.regulatoryRate = rates.regulatoryRate;
      p.profitRate = rates.profitRate;
      p.taxRate = rates.taxRate;
      p.profitIncludesRegulatory = profitIncludesRegulatory;
      p.controlPrice = controlPrice;
    },
    async WorkcostBillProjectProfileSave(id: number, specialty: string, region: string, pricePeriod: string): Promise<void> {
      ensureWorkcostSeed();
      const p = mockBillProjects.find((x) => x.id === id);
      if (!p) throw new Error("项目不存在");
      if (specialty.trim()) p.specialty = specialty.trim();
      if (region.trim()) p.region = region.trim();
      if (pricePeriod.trim()) p.pricePeriod = pricePeriod.trim();
    },
    async WorkcostBillProjectDelete(id: number): Promise<number> {
      ensureWorkcostSeed();
      if (!mockBillProjects.some((p) => p.id === id)) throw new Error("项目不存在");
      const gone = new Set(mockBillItems.filter((b) => b.projectId === id).map((b) => b.id));
      mockBillProjects = mockBillProjects.filter((p) => p.id !== id);
      mockBillItems = mockBillItems.filter((b) => b.projectId !== id);
      mockBillLinks = mockBillLinks.filter((l) => !gone.has(l.billItemId));
      // 演示定额全部是项目导入的独占定额：随项目删除。
      const removed = mockQuotas.length;
      mockQuotas = [];
      return removed;
    },
  };
}
