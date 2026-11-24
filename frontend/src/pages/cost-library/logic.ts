// cost-library/logic.ts — 造价数据库（三级浏览面）的纯函数与行模型。
//
// 板块只管看：清单 → 定额 → 工料机 三级数据的筛选、归并与引用索引都在这里，
// 不碰桥接、不带 React，页面组件只负责取数与渲染。
import type {
  WorkcostBillItem, WorkcostQuota, WorkcostResource,
} from "../../gaea/lib/types";

export type Tier = "bill" | "quota" | "resource";

export interface TierMeta {
  key: Tier;
  label: string;
  hint: string;
  searchPlaceholder: string;
}

export const TIERS: TierMeta[] = [
  { key: "bill", label: "清单", hint: "所有项目累计的分部分项清单", searchPlaceholder: "搜索清单：名称 / 编码 / 特征 / 定额码" },
  { key: "quota", label: "定额", hint: "企业消耗定额 · 每单位消耗多少工料机", searchPlaceholder: "搜索定额：名称 / 编码 / 章节" },
  { key: "resource", label: "工料机", hint: "人工 / 材料 / 机械 / 外委 主数据与取用价", searchPlaceholder: "搜索工料机：名称 / 规格 / 编码 / 分类" },
];

export interface BillFilters {
  projectId: number;
  division: string;
  keyword: string;
  /** false=按明细逐条陈列；true=同名+同特征+同单位归并成库项。 */
  grouped: boolean;
}
export interface QuotaFilters {
  specialty: string;
  keyword: string;
}
export interface ResourceFilters {
  kind: string;
  keyword: string;
}

export const emptyBillFilters: BillFilters = { projectId: 0, division: "", keyword: "", grouped: false };
export const emptyQuotaFilters: QuotaFilters = { specialty: "", keyword: "" };
export const emptyResourceFilters: ResourceFilters = { kind: "", keyword: "" };

/** 带项目名的清单行（累计视图的行模型）。 */
export interface BillRow extends WorkcostBillItem {
  projectName: string;
}

/** 归并视图的一条库项：同名+同特征+同单位的清单项聚合。 */
export interface BillGroup {
  key: string;
  title: string;
  feature: string;
  unit: string;
  projects: string[];
  quotaCodes: string[];
  items: BillRow[];
}

// ── 关键词 ───────────────────────────────────────────────────────────

/** 关键词包含匹配（忽略大小写）；空关键词恒命中。 */
export function matchKeyword(kw: string, ...fields: (string | undefined)[]): boolean {
  const k = kw.trim().toLowerCase();
  if (!k) return true;
  return fields.some((f) => (f ?? "").toLowerCase().includes(k));
}

/** 把挂接定额码归一成集合（引用表聚合优先；旧数据只有单值列时兜底）。 */
export function linkedQuotas(item: Pick<WorkcostBillItem, "quotaCode" | "quotaCodes">): string[] {
  const codes = (item.quotaCodes ?? []).map((c) => c.trim()).filter(Boolean);
  if (codes.length > 0) return codes;
  return item.quotaCode ? [item.quotaCode] : [];
}

// ── 清单 ─────────────────────────────────────────────────────────────

export function filterBill(rows: BillRow[], f: BillFilters): BillRow[] {
  return rows.filter(
    (it) =>
    (f.projectId === 0 || it.projectId === f.projectId) &&
    (!f.division || it.division === f.division) &&
    matchKeyword(f.keyword, it.title, it.code, it.feature, it.quotaCode, ...(it.quotaCodes ?? [])),
  );
}

export function divisionsOf(rows: BillRow[]): string[] {
  const set = new Set<string>();
  for (const it of rows) if (it.division) set.add(it.division);
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN"));
}

export function groupBill(rows: BillRow[]): BillGroup[] {
  const map = new Map<string, BillRow[]>();
  for (const it of rows) {
    const key = `${it.title}|${it.unit}|${it.feature}`;
    const arr = map.get(key);
    if (arr) arr.push(it);
    else map.set(key, [it]);
  }
  return [...map.entries()]
    .map(([key, items]) => ({
      key,
      title: items[0]?.title ?? "",
      feature: items[0]?.feature ?? "",
      unit: items[0]?.unit ?? "",
      projects: [...new Set(items.map((x) => x.projectName))],
      quotaCodes: [...new Set(items.flatMap(linkedQuotas))],
      items,
    }))
    .sort((a, b) => b.projects.length - a.projects.length || a.title.localeCompare(b.title, "zh-CN"));
}

/**
 * 清单行的现算单价（人材机）：挂接定额的综合单价之和。
 * 手填价优先；一条定额都算不出时返回 null（界面显示「—」，不冒充 0）。
 */
export function unitPriceOf(item: WorkcostBillItem, refPrices: Record<string, number>): number | null {
  if (item.priceOverride > 0) return item.priceOverride;
  let sum = 0;
  let got = false;
  for (const code of linkedQuotas(item)) {
    const p = refPrices[code];
    if (typeof p === "number") {
      sum += p;
      got = true;
    }
  }
  return got ? sum : null;
}

/** 定额码 → 引用它的清单行（三级下钻的回链：定额被谁用）。 */
export function quotaUsageIndex(rows: BillRow[]): Map<string, BillRow[]> {
  const index = new Map<string, BillRow[]>();
  for (const it of rows) {
    for (const code of new Set(linkedQuotas(it))) {
      const arr = index.get(code);
      if (arr) arr.push(it);
      else index.set(code, [it]);
    }
  }
  return index;
}

// ── 定额 ─────────────────────────────────────────────────────────────

export function filterQuotas(quotas: WorkcostQuota[], f: QuotaFilters): WorkcostQuota[] {
  return quotas.filter(
    (q) => (!f.specialty || (q.specialty ?? "") === f.specialty) && matchKeyword(f.keyword, q.title, q.code, q.origCode, q.chapter),
  );
}

export function specialtiesOf(quotas: WorkcostQuota[]): string[] {
  const set = new Set<string>();
  for (const q of quotas) if (q.specialty) set.add(q.specialty);
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN"));
}

// ── 工料机 ───────────────────────────────────────────────────────────

/** 核算取用价：现行价优先，为 0 回退基准价（与 Go 侧 EffectivePrice 同口径）。 */
export function effectivePrice(r: Pick<WorkcostResource, "currentPrice" | "basePrice">): number {
  return r.currentPrice > 0 ? r.currentPrice : r.basePrice;
}

export const RESOURCE_KINDS = ["人工", "材料", "机械", "外委"] as const;

export function filterResources(resources: WorkcostResource[], f: ResourceFilters): WorkcostResource[] {
  return resources.filter(
    (r) => (!f.kind || r.kind === f.kind) && matchKeyword(f.keyword, r.title, r.code, r.spec, r.categoryPath),
  );
}

export function kindCounts(resources: WorkcostResource[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const r of resources) counts[r.kind] = (counts[r.kind] ?? 0) + 1;
  return counts;
}

// ── 展示 ─────────────────────────────────────────────────────────────

const money = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const qty = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 4 });

export function fmtMoney(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return "—";
  return "¥" + money.format(v);
}

export function fmtQty(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v) || v <= 0) return "—";
  return qty.format(v);
}

/** 费率：0=显式不计取。 */
export function fmtRate(v: number): string {
  return v > 0 ? `${Math.round(v * 10000) / 100}%` : "不计取";
}

/** 长列表渐进披露：先给前 cap 行，其余等用户点。 */
export function capRows<T>(rows: T[], cap: number): { shown: T[]; hidden: number } {
  return rows.length > cap ? { shown: rows.slice(0, cap), hidden: rows.length - cap } : { shown: rows, hidden: 0 };
}

/** 空库与空态里给用户的示例提示词（去造价模式录入）。 */
export const SAMPLE_PROMPTS: Record<Tier, string> = {
  bill: "把这份五表成本测算工作簿导入造价库：D:\\新工作\\项目目录\\成本测算表.xlsx",
  quota: "在造价库里新建一条消耗定额「垫层 C20」：每 m³ 消耗 C20商品混凝土 1.02、普工 0.3 工日",
  resource: "把工料机库里「中砂」的现行价调到 120 元/m³，期数 2026年第11期",
};

/** 三级选择键：`tier:标识`，详情面板与表格行共用。 */
export type SelKey = string;

export function selKey(tier: Tier, id: string | number): SelKey {
  return `${tier}:${id}`;
}

export function parseSelKey(key: string): { tier: Tier; id: string } | null {
  const i = key.indexOf(":");
  if (i < 0) return null;
  const tier = key.slice(0, i);
  if (tier !== "bill" && tier !== "quota" && tier !== "resource") return null;
  return { tier, id: key.slice(i + 1) };
}
