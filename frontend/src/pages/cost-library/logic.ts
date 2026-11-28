// cost-library/logic.ts — 造价数据库（三级浏览面）的纯函数与行模型。
//
// 板块只管看：清单 → 定额 → 工料机 三级数据的筛选、归并与引用索引都在这里，
// 不碰桥接、不带 React，页面组件只负责取数与渲染。
//
// 维度（2026-11-25 库定规矩；2026-11-27 定额「章节」改叫「分部」，字段仍 chapter）：
// 清单与定额按专业分，定额在专业内按分部分；工料机按种类（类别 → 分类）、地区、
// 期数分。规范值由库说了算——这里的筛选只认库里的取值，空值归「未标注」一桶。
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
  { key: "bill", label: "清单", hint: "各专业的分部分项清单 · 工程量 × 定额", searchPlaceholder: "搜索清单：名称 / 编码 / 特征 / 定额码" },
  { key: "quota", label: "定额", hint: "按专业分的消耗定额 · 每单位消耗多少工料机", searchPlaceholder: "搜索定额：名称 / 编码 / 分部" },
  { key: "resource", label: "工料机", hint: "人工 / 材料 / 机械 / 外委 · 按种类、地区、期数分", searchPlaceholder: "搜索工料机：名称 / 规格 / 编码 / 分类" },
];

/**
 * 「未标注」哨兵：筛选值等于它=只要该字段为空的行。与 Go 侧 workcost.FilterNone
 * 同值——导出按同一口径走。
 */
export const NONE = "__none__";

/**
 * 「类别待整理」哨兵：类别不在四类（人工/材料/机械/外委）之内的工料机单列一桶。
 * 多为存量里把说明行等错记成工料机的行——库定了规矩，它们要能被找出来处理。
 * 与 Go 侧 workcost.FilterOutside 同值。
 */
export const OUTSIDE = "__outside__";

/** 维度取值的显示名：空值显示「未标注」。 */
export function dimLabel(value: string | undefined, empty = "未标注"): string {
  const v = (value ?? "").trim();
  return v === NONE ? empty : v || empty;
}

/**
 * 状态筛选（2026-11-28）：把「该处理的那一类行」一键筛出来，就地修。
 * 清单：manual=未套定额、cross=挂了外专业定额；定额：unused=没有被清单引用；
 * 工料机：unused=没有被定额含量行引用、noprice=缺价（现行价与基准价都是 0）。
 */
export type RowState = "" | "manual" | "cross" | "unused" | "noprice";

export const STATE_LABELS: Record<Tier, { value: RowState; label: string }[]> = {
  bill: [{ value: "manual", label: "未套定额（手填）" }, { value: "cross", label: "挂了外专业定额" }],
  quota: [{ value: "unused", label: "没有被清单引用" }],
  resource: [{ value: "unused", label: "没有被定额引用" }, { value: "noprice", label: "缺价" }],
};

export interface BillFilters {
  projectId: number;
  division: string;
  /** 专业（清单项已归一后的专业）；"" = 全部；NONE = 未标注。 */
  specialty: string;
  keyword: string;
  /** false=按明细逐条陈列；true=同名+同特征+同单位归并成库项。 */
  grouped: boolean;
  /** 状态筛选；缺省 / "" = 不限。 */
  state?: RowState;
}
export interface QuotaFilters {
  specialty: string;
  /** 分部（专业内，字段沿用 chapter）；"" = 全部；NONE = 未归分部。 */
  chapter: string;
  /** 地区（省级）；缺省 / "" = 全部；NONE = 未标地区。 */
  region?: string;
  keyword: string;
  /** 状态筛选；缺省 / "" = 不限。 */
  state?: RowState;
}
export interface ResourceFilters {
  kind: string;
  /** 分类路径（前缀匹配：选上级连带下级）；"" = 全部；NONE = 未细分。 */
  category: string;
  /** 地区；"" = 全部；NONE = 未标注。 */
  region: string;
  /** 期数（现行价所属期，YYYY-MM）；"" = 全部；NONE = 无期。 */
  period: string;
  keyword: string;
  /** 状态筛选；缺省 / "" = 不限。 */
  state?: RowState;
}

export const emptyBillFilters: BillFilters = { projectId: 0, division: "", specialty: "", keyword: "", grouped: false, state: "" };
export const emptyQuotaFilters: QuotaFilters = { specialty: "", chapter: "", region: "", keyword: "", state: "" };
export const emptyResourceFilters: ResourceFilters = { kind: "", category: "", region: "", period: "", keyword: "", state: "" };

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

/** 左栏一个可点的维度取值。name 为 "" 表示「全部」；NONE 表示「未标注」。 */
export interface Facet {
  name: string;
  count: number;
}

// ── 关键词 ───────────────────────────────────────────────────────────

/** 把搜索框的内容拆成词条（去首尾空白、忽略大小写、去重、保序）。 */
export function keywordTerms(kw: string): string[] {
  return [...new Set(kw.trim().toLowerCase().split(/\s+/).filter(Boolean))];
}

/**
 * 关键词包含匹配（忽略大小写）；空关键词恒命中。多个词用空白分开时要求每个词都命中
 * （「C25 混凝土」= 名称里既有 C25 又有混凝土），一个词落在任一字段里算命中。
 */
export function matchKeyword(kw: string, ...fields: (string | undefined)[]): boolean {
  const terms = keywordTerms(kw);
  if (terms.length === 0) return true;
  return terms.every((t) => fields.some((f) => (f ?? "").toLowerCase().includes(t)));
}

/** 关键词在一段文本里命中的片段（高亮用）：返回 [原文片段, 是否命中…] 交替的数组；没命中返回单元素数组。 */
export function splitByKeyword(kw: string, text: string | undefined): { text: string; hit: boolean }[] {
  const s = text ?? "";
  const terms = keywordTerms(kw);
  if (terms.length === 0 || s === "") return [{ text: s, hit: false }];
  // 各词条在文本里的命中区间（忽略大小写），合并重叠后按顺序切分。
  const lower = s.toLowerCase();
  const spans: [number, number][] = [];
  for (const t of terms) {
    for (let at = lower.indexOf(t); at >= 0; at = lower.indexOf(t, at + t.length)) spans.push([at, at + t.length]);
  }
  if (spans.length === 0) return [{ text: s, hit: false }];
  spans.sort((a, b) => a[0] - b[0]);
  const out: { text: string; hit: boolean }[] = [];
  let pos = 0;
  for (const [from, to] of spans) {
    if (to <= pos) continue;
    if (from > pos) out.push({ text: s.slice(pos, from), hit: false });
    pos = Math.max(pos, from);
    out.push({ text: s.slice(pos, to), hit: true });
    pos = to;
  }
  if (pos < s.length) out.push({ text: s.slice(pos), hit: false });
  return out;
}

/** 把挂接定额码归一成集合（引用表聚合优先；旧数据只有单值列时兜底）。 */
export function linkedQuotas(item: Pick<WorkcostBillItem, "quotaCode" | "quotaCodes">): string[] {
  const codes = (item.quotaCodes ?? []).map((c) => c.trim()).filter(Boolean);
  if (codes.length > 0) return codes;
  return item.quotaCode ? [item.quotaCode] : [];
}

/** 规范值筛选命中：筛选为空（或缺省）=不过滤；NONE=只要空值；否则精确相等。 */
export function facetMatch(filter: string | undefined, value: string | undefined): boolean {
  const f = (filter ?? "").trim();
  if (!f) return true;
  const v = (value ?? "").trim();
  return f === NONE ? v === "" : v === f;
}

// ── 清单 ─────────────────────────────────────────────────────────────

/** 各级「关键词落在哪些字段」——筛选与分面共用同一口径，计数才跟得上搜索。 */
const billKeywordFields = (it: BillRow): (string | undefined)[] => [it.title, it.code, it.feature, it.quotaCode, ...(it.quotaCodes ?? [])];
const quotaKeywordFields = (q: WorkcostQuota): (string | undefined)[] => [q.title, q.code, q.origCode, q.chapter];
const resourceKeywordFields = (r: WorkcostResource): (string | undefined)[] => [r.title, r.code, r.spec, r.categoryPath];

/** 筛选时页面顺带给出的引用关系（状态筛选要用）：定额的专业、定额被清单引用的次数、被含量行引用的工料机编码。 */
export interface FilterContext {
  quotaSpecialty?: Map<string, string>;
  quotaUsage?: Map<string, number>;
  usedResourceCodes?: Set<string>;
}

function stateBill(it: BillRow, state: RowState, ctx?: FilterContext): boolean {
  switch (state) {
    case "manual":
      return linkedQuotas(it).length === 0;
    case "cross":
      return linkedQuotas(it).some((c) => isCrossSpecialty(it.specialty ?? "", ctx?.quotaSpecialty?.get(c)));
    default:
      return true;
  }
}

export function filterBill(rows: BillRow[], f: BillFilters, ctx?: FilterContext): BillRow[] {
  return rows.filter(
    (it) =>
      (f.projectId === 0 || it.projectId === f.projectId) &&
      (!f.division || it.division === f.division) &&
      facetMatch(f.specialty, it.specialty) &&
      stateBill(it, f.state ?? "", ctx) &&
      matchKeyword(f.keyword, ...billKeywordFields(it)),
  );
}

/** 清单行的专业（后端已归一：行自身为空则取项目默认专业）。 */
export function billSpecialty(it: Pick<WorkcostBillItem, "specialty">): string {
  return (it.specialty ?? "").trim();
}

type BillFacetScope = Partial<Pick<BillFilters, "projectId" | "division" | "keyword" | "state">>;

/** 一个清单行在「项目 / 分部 / 关键词 / 状态」下算不算进分面计数（没给的维度不限）。 */
function inBillScope(it: BillRow, f: BillFacetScope, ctx?: FilterContext): boolean {
  if ((f.projectId ?? 0) !== 0 && it.projectId !== f.projectId) return false;
  if (f.division && it.division !== f.division) return false;
  return matchKeyword(f.keyword ?? "", ...billKeywordFields(it)) && stateBill(it, f.state ?? "", ctx);
}

/**
 * 某个筛选下的专业分布（按条数降序；「未标注」单列一桶，排在最后）。只套项目 / 分部 /
 * 关键词 / 状态——这一维自己的筛选不套，否则选了一个专业就只剩它一个可选项。搜索时
 * 计数跟着关键词走（分面联动）。
 */
export function billSpecialties(rows: BillRow[], f: BillFacetScope = {}, ctx?: FilterContext): Facet[] {
  const counts = new Map<string, number>();
  for (const it of rows) {
    if (!inBillScope(it, f, ctx)) continue;
    const s = billSpecialty(it) || NONE;
    counts.set(s, (counts.get(s) ?? 0) + 1);
  }
  return sortFacets(counts);
}

export function divisionsOf(rows: BillRow[], f: BillFacetScope = {}, ctx?: FilterContext): string[] {
  const set = new Set<string>();
  for (const it of rows) {
    if (!inBillScope(it, f, ctx)) continue;
    if (it.division) set.add(it.division);
  }
  return [...set].sort((a, b) => a.localeCompare(b, "zh-CN"));
}

/** 每个项目的清单行数在某组条件下各剩多少（左栏项目列表随搜索联动；没有符合条件行的项目不出）。 */
export function projectCounts(rows: BillRow[], f: BillFacetScope = {}, ctx?: FilterContext): Map<number, number> {
  const counts = new Map<number, number>();
  for (const it of rows) {
    if (!inBillScope(it, { ...f, projectId: 0 }, ctx)) continue;
    counts.set(it.projectId, (counts.get(it.projectId) ?? 0) + 1);
  }
  return counts;
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

/**
 * 一条清单行上某条定额的工程量：挂接明细里给了就用它（>0），否则跟随清单工程量。
 * 详情里「消耗量 = 该定额工程量 × 定额含量」用它。
 */
export function quotaFactor(item: Pick<WorkcostBillItem, "quantity" | "linkQuantities">, quotaCode: string): number {
  const q = item.linkQuantities?.[quotaCode] ?? 0;
  return q > 0 ? q : item.quantity;
}

/** 消耗量 = 工程量 × 含量；任一非正返回 null（显示「—」）。 */
export function consumptionOf(factor: number, content: number): number | null {
  if (!(factor > 0) || !(content > 0) || !Number.isFinite(factor * content)) return null;
  return factor * content;
}

/**
 * 把核算行和定额含量行一一对上，返回每条核算行对应的含量行（对不上的位置为
 * undefined）——详情里点哪一行下钻到哪个工料机靠它。
 *
 * 不能按下标硬对：后端把核算行按 人工→材料→机械→外委 重排过，含量行却是保存时的
 * 原序；两边顺序一不同，下标对齐就会钻到别的工料机。优先按「类别 + 名称」配对，
 * 配不上的位置再退回同下标。
 */
export function alignItems<T extends { kind: string; title: string }>(
  lines: { kind: string; title: string }[], items: T[],
): (T | undefined)[] {
  const used = new Set<number>();
  const same = (a: string, b: string) => a.trim() === b.trim();
  const out: (T | undefined)[] = lines.map((l) => {
    const hit = items.findIndex((it, i) => !used.has(i) && same(it.kind, l.kind) && same(it.title, l.title));
    if (hit >= 0) {
      used.add(hit);
      return items[hit];
    }
    return undefined;
  });
  lines.forEach((l, i) => {
    if (out[i] || used.has(i) || !items[i]) return;
    if (same(items[i]!.kind, l.kind)) {
      out[i] = items[i]!;
      used.add(i);
    }
  });
  return out;
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

/** 清单专业与定额专业是否对不上（清单套定额同专业优先）。任一方未标注不算。 */
export function isCrossSpecialty(billSpecialty: string, quotaSpecialty: string | undefined): boolean {
  const b = billSpecialty.trim();
  const q = (quotaSpecialty ?? "").trim();
  return !!b && !!q && b !== q;
}

// ── 定额 ─────────────────────────────────────────────────────────────

export function filterQuotas(quotas: WorkcostQuota[], f: QuotaFilters, ctx?: FilterContext): WorkcostQuota[] {
  return quotas.filter(
    (q) =>
      facetMatch(f.specialty, q.specialty) && facetMatch(f.chapter, q.chapter) && facetMatch(f.region ?? "", q.region) &&
      (f.state !== "unused" || (ctx?.quotaUsage?.get(q.code) ?? 0) === 0) &&
      matchKeyword(f.keyword, ...quotaKeywordFields(q)),
  );
}

type QuotaFacetScope = Partial<Pick<QuotaFilters, "keyword" | "state">>;

/** 一个定额在「关键词 + 状态」下算不算进分面计数（没给的维度不限）。 */
function inQuotaScope(q: WorkcostQuota, f: QuotaFacetScope, ctx?: FilterContext): boolean {
  if (!matchKeyword(f.keyword ?? "", ...quotaKeywordFields(q))) return false;
  return f.state !== "unused" || (ctx?.quotaUsage?.get(q.code) ?? 0) === 0;
}

/** 定额的专业分布（随关键词与状态联动；专业这一维自己的筛选不套）。 */
export function specialtiesOf(quotas: WorkcostQuota[], f: QuotaFacetScope = {}, ctx?: FilterContext): Facet[] {
  const counts = new Map<string, number>();
  for (const q of quotas) {
    if (!inQuotaScope(q, f, ctx)) continue;
    const s = q.specialty?.trim() || NONE;
    counts.set(s, (counts.get(s) ?? 0) + 1);
  }
  return sortFacets(counts);
}

/**
 * 某个专业内的分部分布（按条数降序）；specialty 为空时按全部专业算。
 * kw 给了就只数命中关键词的定额（分部这一维自己的筛选不套）。
 */
export function chaptersOf(quotas: WorkcostQuota[], specialty = "", kw = ""): Facet[] {
  const counts = new Map<string, number>();
  for (const q of quotas) {
    if (!facetMatch(specialty, q.specialty)) continue;
    if (kw && !matchKeyword(kw, ...quotaKeywordFields(q))) continue;
    const c = q.chapter?.trim() || NONE;
    counts.set(c, (counts.get(c) ?? 0) + 1);
  }
  return sortFacets(counts);
}

/** 定额的地区分布（地区与上游清单项目挂钩；没标的单列一桶）。 */
export function quotaRegionsOf(quotas: WorkcostQuota[], f: QuotaFacetScope = {}, ctx?: FilterContext): Facet[] {
  const counts = new Map<string, number>();
  for (const q of quotas) {
    if (!inQuotaScope(q, f, ctx)) continue;
    const r = q.region?.trim() || NONE;
    counts.set(r, (counts.get(r) ?? 0) + 1);
  }
  return sortFacets(counts);
}

// ── 工料机 ───────────────────────────────────────────────────────────

/** 核算取用价：现行价优先，为 0 回退基准价（与 Go 侧 EffectivePrice 同口径）。 */
export function effectivePrice(r: Pick<WorkcostResource, "currentPrice" | "basePrice">): number {
  return r.currentPrice > 0 ? r.currentPrice : r.basePrice;
}

export const RESOURCE_KINDS = ["人工", "材料", "机械", "外委"] as const;

/** 类别是否在库认可的四类之内。 */
export function isKnownKind(kind: string | undefined): boolean {
  return (RESOURCE_KINDS as readonly string[]).includes((kind ?? "").trim());
}

/** 类别在界面上的叫法。 */
export function kindLabel(kind: string): string {
  return kind === OUTSIDE ? "类别待整理" : kind;
}

export function kindCounts(resources: WorkcostResource[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const r of resources) counts[r.kind] = (counts[r.kind] ?? 0) + 1;
  return counts;
}

/** 左栏的类别列表：四类里有行的按固定顺序；四类外有行时末尾加一桶「类别待整理」。f 给了就只数命中关键词 / 状态的行。 */
export function kindsOf(resources: WorkcostResource[], f: ResourceFacetScope = {}, ctx?: FilterContext): { kind: string; count: number }[] {
  const scoped = resources.filter((r) => inResourceScope(r, f, ctx));
  const counts = kindCounts(scoped);
  const list = RESOURCE_KINDS.filter((k) => (counts[k] ?? 0) > 0).map((k) => ({ kind: k, count: counts[k] ?? 0 }));
  const outside = scoped.filter((r) => !isKnownKind(r.kind)).length;
  return outside > 0 ? [...list, { kind: OUTSIDE, count: outside }] : list;
}

function matchKind(filter: string, kind: string | undefined): boolean {
  if (!filter) return true;
  if (filter === OUTSIDE) return !isKnownKind(kind);
  return (kind ?? "").trim() === filter;
}

export type ResourceFacetScope = Partial<Pick<ResourceFilters, "keyword" | "state">>;

function stateResource(r: WorkcostResource, state: RowState, ctx?: FilterContext): boolean {
  switch (state) {
    case "unused":
      return !ctx?.usedResourceCodes?.has(r.code);
    case "noprice":
      return effectivePrice(r) <= 0;
    default:
      return true;
  }
}

/** 一个工料机在「关键词 + 状态」下算不算进分面计数（没给的维度不限）。 */
function inResourceScope(r: WorkcostResource, f: ResourceFacetScope, ctx?: FilterContext): boolean {
  return matchKeyword(f.keyword ?? "", ...resourceKeywordFields(r)) && stateResource(r, f.state ?? "", ctx);
}

export function filterResources(resources: WorkcostResource[], f: ResourceFilters, ctx?: FilterContext): WorkcostResource[] {
  return resources.filter(
    (r) =>
      matchKind(f.kind, r.kind) &&
      pickCategory(f.category, r.categoryPath) &&
      facetMatch(f.region, r.region) &&
      facetMatch(f.period, r.priceDate) &&
      stateResource(r, f.state ?? "", ctx) &&
      matchKeyword(f.keyword, ...resourceKeywordFields(r)),
  );
}

/** 分类筛选：选上级路径连带下级（「材料/土建材料」命中「材料/土建材料/钢材」）。 */
function pickCategory(filter: string, path: string | undefined): boolean {
  const f = filter.trim();
  if (!f) return true;
  const p = (path ?? "").trim();
  if (f === NONE) return p === "";
  return p === f || p.startsWith(f + "/");
}

/** 分类树的一个节点：path 是完整分类路径（作筛选值），children 是它的下一级。 */
export interface CategoryNode {
  path: string;
  label: string;
  count: number;
  children: CategoryNode[];
}

/**
 * 某个类别下的分类树：按分类路径聚合（路径的每一段是一级），条数含下级。
 * 没有分类路径的行不计入（由未细分计数表达）。f 给了就只数命中关键词 / 状态的行
 * （分类这一维自己的筛选不套）。
 */
export function categoryTreeOf(resources: WorkcostResource[], kind: string, f: ResourceFacetScope = {}, ctx?: FilterContext): { tree: CategoryNode[]; uncategorized: number } {
  const counts = new Map<string, number>();
  let uncategorized = 0;
  for (const r of resources) {
    if (!matchKind(kind, r.kind) || !inResourceScope(r, f, ctx)) continue;
    const path = (r.categoryPath ?? "").trim();
    if (!path) {
      uncategorized++;
      continue;
    }
    const segs = path.split("/").map((s) => s.trim()).filter(Boolean);
    for (let i = 1; i <= segs.length; i++) {
      const prefix = segs.slice(0, i).join("/");
      counts.set(prefix, (counts.get(prefix) ?? 0) + 1);
    }
  }
  const tree: CategoryNode[] = [];
  const depth = (p: string) => p.split("/").length;
  // 先建上级再建下级（按层级排）；同级按条数降序。
  const paths = [...counts.keys()].sort((a, b) => depth(a) - depth(b) || (counts.get(b) ?? 0) - (counts.get(a) ?? 0) || a.localeCompare(b, "zh-CN"));
  for (const p of paths) {
    const segs = p.split("/");
    let level = tree;
    let cur = "";
    for (const seg of segs) {
      cur = cur ? `${cur}/${seg}` : seg;
      let node = level.find((n) => n.path === cur);
      if (!node) {
        node = { path: cur, label: seg, count: counts.get(cur) ?? 0, children: [] };
        level.push(node);
      }
      level = node.children;
    }
  }
  return { tree, uncategorized };
}

export function regionsOf(resources: WorkcostResource[], f: ResourceFacetScope = {}, ctx?: FilterContext): Facet[] {
  const counts = new Map<string, number>();
  for (const r of resources) {
    if (!inResourceScope(r, f, ctx)) continue;
    const key = r.region?.trim() || NONE;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return sortFacets(counts);
}

/** 期数的可排序键：YYYY-MM / YYYY年M月 → 年月数值；认不出的排最后（-1）。 */
export function periodKey(period: string | undefined): number {
  const m = (period ?? "").trim().match(/^(\d{4})\s*(?:-|\.|\/|年)\s*(\d{1,2})/);
  if (!m) return -1;
  return Number(m[1]) * 100 + Number(m[2]);
}

/** 期数分布：新→旧；无期的排最后。 */
export function periodsOf(resources: WorkcostResource[], f: ResourceFacetScope = {}, ctx?: FilterContext): Facet[] {
  const counts = new Map<string, number>();
  for (const r of resources) {
    if (!inResourceScope(r, f, ctx)) continue;
    const p = r.priceDate?.trim() || NONE;
    counts.set(p, (counts.get(p) ?? 0) + 1);
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => {
      if (a.name === NONE) return 1;
      if (b.name === NONE) return -1;
      return periodKey(b.name) - periodKey(a.name);
    });
}

/** 期数的显示：库里统一存 YYYY-MM，界面照常显示「2026年6月」；其他写法原样显示。 */
export function fmtPeriod(period: string | undefined): string {
  const p = (period ?? "").trim();
  const m = p.match(/^(\d{4})-(\d{2})$/);
  return m ? `${m[1]}年${Number(m[2])}月` : p;
}

/** 维度分布排序：条数降序、同数按名称；「未标注」恒排最后。 */
function sortFacets(counts: Map<string, number>): Facet[] {
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => {
      if (a.name === NONE) return 1;
      if (b.name === NONE) return -1;
      return b.count - a.count || a.name.localeCompare(b.name, "zh-CN");
    });
}

/** 待整理：不合库规矩的条数（文件不一定带这些信息，或存量取值在规矩外，由造价模式的模型归类）。 */
export interface PendingCounts {
  billSpecialty: number;
  quotaSpecialty: number;
  quotaChapter: number;
  resourceKind: number;
  resourceRegion: number;
  resourceCategory: number;
  resourcePeriod: number;
}

export function pendingOf(rows: BillRow[], quotas: WorkcostQuota[], resources: WorkcostResource[]): PendingCounts & { total: number } {
  const p: PendingCounts = { billSpecialty: 0, quotaSpecialty: 0, quotaChapter: 0, resourceKind: 0, resourceRegion: 0, resourceCategory: 0, resourcePeriod: 0 };
  for (const it of rows) if (!billSpecialty(it)) p.billSpecialty++;
  for (const q of quotas) {
    if (!q.specialty?.trim()) p.quotaSpecialty++;
    if (!q.chapter?.trim()) p.quotaChapter++;
  }
  for (const r of resources) {
    if (!isKnownKind(r.kind)) p.resourceKind++;
    if (!r.region?.trim()) p.resourceRegion++;
    if (!r.categoryPath?.trim()) p.resourceCategory++;
    if (!r.priceDate?.trim()) p.resourcePeriod++;
  }
  // 总数 = 下面逐项列出之和（入口上的数要和它自己的说明对得上）。
  const total = p.billSpecialty + p.quotaSpecialty + p.quotaChapter + p.resourceKind + p.resourceRegion + p.resourceCategory + p.resourcePeriod;
  return { ...p, total };
}

/** 「待整理」入口的说明（逐项列出，0 的不列）。 */
export function pendingSummary(p: PendingCounts): string {
  const parts: [string, number][] = [
    ["清单无专业", p.billSpecialty], ["定额无专业", p.quotaSpecialty], ["定额未归分部", p.quotaChapter],
    ["工料机类别待整理", p.resourceKind],
    ["工料机未标地区", p.resourceRegion], ["工料机未细分", p.resourceCategory], ["工料机无期数", p.resourcePeriod],
  ];
  return parts.filter(([, n]) => n > 0).map(([label, n]) => `${label} ${n}`).join(" · ");
}

/** 「待整理」入口的逐项（2026-11-28）：点一项就筛出对应的那批行，在板块里直接修或批量改。 */
export const PENDING_ITEMS: { key: keyof PendingCounts; tier: Tier; label: string }[] = [
  { key: "billSpecialty", tier: "bill", label: "清单无专业" },
  { key: "quotaSpecialty", tier: "quota", label: "定额无专业" },
  { key: "quotaChapter", tier: "quota", label: "定额未归分部" },
  { key: "resourceKind", tier: "resource", label: "工料机类别待整理" },
  { key: "resourceRegion", tier: "resource", label: "工料机未标地区" },
  { key: "resourceCategory", tier: "resource", label: "工料机未细分" },
  { key: "resourcePeriod", tier: "resource", label: "工料机无期数" },
];

/** 「待整理」某一项点开后的落点：切到对应那一级，只留这一项的筛选（别的筛选清空）。 */
export function pendingFilter(key: keyof PendingCounts): { tier: Tier; filters: BillFilters | QuotaFilters | ResourceFilters } {
  switch (key) {
    case "billSpecialty":
      return { tier: "bill", filters: { ...emptyBillFilters, specialty: NONE } };
    case "quotaSpecialty":
      return { tier: "quota", filters: { ...emptyQuotaFilters, specialty: NONE } };
    case "quotaChapter":
      return { tier: "quota", filters: { ...emptyQuotaFilters, chapter: NONE } };
    case "resourceKind":
      return { tier: "resource", filters: { ...emptyResourceFilters, kind: OUTSIDE } };
    case "resourceRegion":
      return { tier: "resource", filters: { ...emptyResourceFilters, region: NONE } };
    case "resourceCategory":
      return { tier: "resource", filters: { ...emptyResourceFilters, category: NONE } };
    default:
      return { tier: "resource", filters: { ...emptyResourceFilters, period: NONE } };
  }
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

/** 「待整理」入口复制的提示词：文件不一定带专业/地区/分类，由造价模式的模型归类。 */
export const PENDING_PROMPT = "整理造价库：先 cost_library action=dims 盘点现状，再把还没标注专业的清单与定额、还没标地区 / 分类 / 期数的工料机归类（action=classify，先 dry_run 给我看方案，我确认后再落库）。专业、地区、分类只用字典里已有的叫法，拿不准的先问我。";

/** 空库与空态里给用户的示例提示词（去造价模式录入）。录入由模型读资料，不限定格式。 */
export const SAMPLE_PROMPTS: Record<Tier, string> = {
  bill: "把这份造价资料录入造价库：D:\\新工作\\项目目录\\成本测算表.xlsx（先告诉我你读到了什么、口径怎么定）",
  quota: "在造价库里新建一条消耗定额「垫层 C20」：每 m³ 消耗 C20商品混凝土 1.02、普工 0.3 工日",
  resource: "把工料机库里「中砂」的现行价调到 120 元/m³，期数 2026-11",
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
