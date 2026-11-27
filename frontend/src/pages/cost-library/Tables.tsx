// Tables — 三级列表（清单 / 定额 / 工料机）。只读浏览面：行点击看详情，复选框
// 批量删除，行尾删除。定额码是下钻入口（清单 → 定额）。状态全在页面，纯展示。
// 列按宽度渐进披露（cl-pri-1 先收、cl-pri-2 最先收）：窄窗只留编码、名称与价。
import { memo, useState } from "react";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Trash2 } from "../../gaea/icons";
import type { WorkcostQuota, WorkcostResource } from "../../gaea/lib/types";
import {
  dimLabel, effectivePrice, fmtMoney, fmtPeriod, fmtQty, groupBill, linkedQuotas, selKey, unitPriceOf,
  type BillRow,
} from "./logic";

const toneOf = (kind: string): string => {
  switch (kind) {
    case "人工": return " is-labor";
    case "机械": return " is-machine";
    case "外委": return " is-outsourced";
    default: return " is-material";
  }
};

type SortDirection = "asc" | "desc";
interface SortState<K extends string> {
  key: K;
  dir: SortDirection;
}
type BillSortKey =
  | "project" | "specialty" | "code" | "title" | "feature" | "unit"
  | "quantity" | "quotas" | "listPrice" | "price";
type QuotaSortKey = "code" | "title" | "unit" | "specialty" | "chapter" | "region" | "usage" | "price";
type ResourceSortKey = "code" | "kind" | "title" | "spec" | "unit" | "region" | "period" | "price" | "basePrice";

const zhCollator = new Intl.Collator("zh-Hans-CN", { numeric: true, sensitivity: "base" });

/** 空值始终沉底；值为空时不随升/降序翻到顶部。 */
function orderRows<T, K extends string>(
  rows: T[],
  sort: SortState<K> | null,
  valueOf: (row: T, key: K) => string | number | null | undefined,
): T[] {
  if (!sort) return rows;
  const factor = sort.dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    const av = valueOf(a, sort.key);
    const bv = valueOf(b, sort.key);
    const aMissing = av === null || av === undefined || av === "";
    const bMissing = bv === null || bv === undefined || bv === "";
    if (aMissing || bMissing) return aMissing === bMissing ? 0 : aMissing ? 1 : -1;
    const cmp = typeof av === "number" && typeof bv === "number"
      ? av - bv
      : zhCollator.compare(String(av), String(bv));
    return cmp * factor;
  });
}

function nextSort<K extends string>(current: SortState<K> | null, key: K): SortState<K> | null {
  if (!current || current.key !== key) return { key, dir: "asc" };
  if (current.dir === "asc") return { key, dir: "desc" };
  return null;
}

function SortHeader<K extends string>({ sort, onSort, sortKey, label, numeric, className }: {
  sort: SortState<K> | null;
  onSort: (key: K) => void;
  sortKey: K;
  label: string;
  numeric?: boolean;
  className?: string;
}) {
  const direction: SortDirection | null = sort?.key === sortKey ? sort.dir : null;
  const active = direction !== null;
  const ariaSort: "none" | "ascending" | "descending" =
    direction === "asc" ? "ascending" : direction === "desc" ? "descending" : "none";
  return (
    <th className={className} aria-sort={ariaSort}>
      <button
        type="button"
        className={`cl-sort-head${numeric ? " is-num" : ""}${active ? " is-active" : ""}`}
        aria-label={`按${label}排序`}
        title={active ? `当前${direction === "asc" ? "升序" : "降序"}，再点切换` : "点击排序"}
        onClick={() => onSort(sortKey)}
      >
        <span>{label}</span>
        {active
          ? direction === "asc"
            ? <ArrowUp size={10} aria-hidden />
            : <ArrowDown size={10} aria-hidden />
          : <span className="cl-sort-pair" aria-hidden><ArrowUp size={8} /><ArrowDown size={8} /></span>}
      </button>
    </th>
  );
}

export function KindBadge({ kind }: { kind: string }) {
  return <span className={`cl-kind${toneOf(kind)}`}>{kind}</span>;
}

function CheckCell({ checked, label, onChange }: { checked: boolean; label: string; onChange: (next: boolean) => void }) {
  return (
    <td className="cl-check" onClick={(e) => e.stopPropagation()}>
      <input type="checkbox" checked={checked} aria-label={label} onChange={(e) => onChange(e.target.checked)} />
    </td>
  );
}

function DeleteCell({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <td className="cl-act" onClick={(e) => e.stopPropagation()}>
      <button type="button" className="cl-icon-btn cl-row-delete" aria-label={label} title="删除" onClick={onClick}>
        <Trash2 size={12} />
      </button>
    </td>
  );
}

// ── 清单 ─────────────────────────────────────────────────────────────

export interface BillTableProps {
  rows: BillRow[];
  grouped: boolean;
  showProject: boolean;
  refPrices: Record<string, number>;
  checked: Set<string>;
  active: string | null;
  expanded: Set<string>;
  onToggleGroup: (key: string) => void;
  onToggle: (key: string, next: boolean) => void;
  onToggleAll: (next: boolean) => void;
  onOpen: (row: BillRow) => void;
  onOpenQuota: (code: string) => void;
  onDelete: (row: BillRow) => void;
}

export const BillTable = memo(function BillTable(props: BillTableProps) {
  const { rows, refPrices, checked, active } = props;
  const [sort, setSort] = useState<SortState<BillSortKey> | null>(null);
  const allChecked = rows.length > 0 && rows.every((it) => checked.has(selKey("bill", it.id)));
  const orderedRows = orderRows(rows, sort, (row, key) => {
    switch (key) {
      case "project": return row.projectName;
      case "specialty": return row.specialty;
      case "code": return row.code;
      case "title": return row.title;
      case "feature": return row.feature;
      case "unit": return row.unit;
      case "quantity": return row.quantity;
      case "quotas": return linkedQuotas(row).join(" ");
      case "listPrice": return (row.listPrice ?? 0) > 0 ? row.listPrice : null;
      case "price": return unitPriceOf(row, refPrices);
    }
  });
  const onSort = (key: BillSortKey) => setSort((current) => nextSort(current, key));
  const rowBody = (it: BillRow, withProject: boolean) => {
    const key = selKey("bill", it.id);
    const quotas = linkedQuotas(it);
    return (
      <tr
        key={key}
        data-row-key={key}
        className={`cl-tr${active === key ? " is-active" : ""}`}
        onClick={() => props.onOpen(it)}
        title="点击查看综合单价分析"
      >
        <CheckCell checked={checked.has(key)} label={`选择 ${it.title}`} onChange={(next) => props.onToggle(key, next)} />
        {withProject && (
          <td className="cl-cell-project cl-pri-2" title={it.projectName}>{it.projectName}</td>
        )}
        <td className={`cl-pri-1${it.specialty ? "" : " cl-dim"}`} title={it.specialty ? "专业" : "还没标注专业（待整理）"}>
          {dimLabel(it.specialty, "未标注")}
        </td>
        <td className="cl-cell-code">{it.code}</td>
        <td className="cl-cell-title">
          <button
            type="button"
            className="cl-cell-link"
            aria-label={`查看「${it.title}」的综合单价分析`}
            onClick={(e) => { e.stopPropagation(); props.onOpen(it); }}
          >
            {it.title}
          </button>
        </td>
        <td className="cl-cell-feature cl-pri-1" title={it.quantityExpr ? `${it.feature}｜计算式：${it.quantityExpr}` : it.feature}>
          {it.feature || it.quantityExpr || <span className="cl-dim">—</span>}
        </td>
        <td className="cl-pri-1">{it.unit || "—"}</td>
        <td className="cl-num cl-pri-2" title={it.quantityExpr ? `计算式：${it.quantityExpr}` : "工程量"}>{fmtQty(it.quantity)}</td>
        <td className="cl-cell-quotas">
          {quotas.length > 0 ? quotas.map((c) => (
            <button
              key={c}
              type="button"
              className="cl-chip"
              title={`查看定额 ${c}`}
              aria-label={`下钻到定额 ${c}`}
              onClick={(e) => { e.stopPropagation(); props.onOpenQuota(c); }}
            >
              {c}
            </button>
          )) : <span className="cl-chip is-manual" title="未套定额（手填价）">手填</span>}
        </td>
        <td className="cl-num" title="原表综合单价快照（导入工作簿留痕）">{fmtMoney((it.listPrice ?? 0) > 0 ? it.listPrice : null)}</td>
        <td className="cl-num cl-strong" title={it.priceOverride > 0 ? "手填单价" : "按当前资源价现算（人材机），挂接定额的综合单价之和"}>
          {fmtMoney(unitPriceOf(it, refPrices))}
        </td>
        <DeleteCell label={`删除「${it.title}」`} onClick={() => props.onDelete(it)} />
      </tr>
    );
  };

  if (props.grouped) {
    const groups = groupBill(orderedRows);
    return (
      <div className="cl-groups" data-testid="cost-bill-grouped">
        {groups.map((g) => {
          const open = props.expanded.has(g.key);
          const groupChecked = g.items.length > 0 && g.items.every((it) => checked.has(selKey("bill", it.id)));
          return (
            <section key={g.key} className="cl-group" aria-expanded={open}>
              <div className="cl-group-head">
                <input
                  type="checkbox"
                  aria-label={`选择库项「${g.title}」的全部 ${g.items.length} 条`}
                  checked={groupChecked}
                  onChange={(e) => g.items.forEach((it) => props.onToggle(selKey("bill", it.id), e.target.checked))}
                />
                <button type="button" className="cl-group-toggle" aria-expanded={open} onClick={() => props.onToggleGroup(g.key)}>
                  {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                  <span className="cl-group-title">{g.title}</span>
                  {g.feature && <span className="cl-group-feature">{g.feature}</span>}
                </button>
                <span className="cl-meta">
                  <span>{g.unit || "—"}</span>
                  <span>被 {g.projects.length} 个项目使用</span>
                  <span>{g.items.length} 条</span>
                  <span>{g.quotaCodes.length > 0 ? `定额 ${g.quotaCodes.join(" / ")}` : "未套定额"}</span>
                </span>
              </div>
              {open && (
                <table className="cl-table cl-table-inner">
                  <tbody>
                    {g.items.map((it) => (
                      <tr key={it.id} data-row-key={selKey("bill", it.id)} className={`cl-tr${active === selKey("bill", it.id) ? " is-active" : ""}`}
                        onClick={() => props.onOpen(it)}>
                        <CheckCell checked={checked.has(selKey("bill", it.id))} label={`选择 ${it.projectName} 的 ${it.title}`} onChange={(next) => props.onToggle(selKey("bill", it.id), next)} />
                        <td className="cl-cell-project cl-pri-2" title={it.projectName}>{it.projectName}</td>
                        <td className="cl-cell-code">{it.code}</td>
                        <td className="cl-cell-title">{it.title}</td>
                        <td className="cl-pri-1">{it.unit || "—"}</td>
                        <td className="cl-num cl-pri-2">{fmtQty(it.quantity)}</td>
                        <td className="cl-cell-quotas">
                          {linkedQuotas(it).map((c) => (
                            <button key={c} type="button" className="cl-chip" title={`查看定额 ${c}`}
                              onClick={(e) => { e.stopPropagation(); props.onOpenQuota(c); }}>{c}</button>
                          ))}
                        </td>
                        <td className="cl-num">{fmtMoney((it.listPrice ?? 0) > 0 ? it.listPrice : null)}</td>
                        <td className="cl-num cl-strong">{fmtMoney(unitPriceOf(it, refPrices))}</td>
                        <DeleteCell label={`删除「${it.title}」`} onClick={() => props.onDelete(it)} />
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          );
        })}
      </div>
    );
  }

  return (
    <table className="cl-table" data-testid="cost-bill-rows">
      <thead>
        <tr>
          <th className="cl-check"><input type="checkbox" aria-label="全选当前清单" checked={allChecked} onChange={(e) => props.onToggleAll(e.target.checked)} /></th>
          {props.showProject && <SortHeader sort={sort} onSort={onSort} sortKey="project" label="项目" className="cl-pri-2" />}
          <SortHeader sort={sort} onSort={onSort} sortKey="specialty" label="专业" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="code" label="编码" />
          <SortHeader sort={sort} onSort={onSort} sortKey="title" label="名称" />
          <SortHeader sort={sort} onSort={onSort} sortKey="feature" label="项目特征" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="unit" label="单位" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="quantity" label="工程量" numeric className="cl-num cl-pri-2" />
          <SortHeader sort={sort} onSort={onSort} sortKey="quotas" label="挂接定额" />
          <SortHeader sort={sort} onSort={onSort} sortKey="listPrice" label="原表单价" numeric className="cl-num" />
          <SortHeader sort={sort} onSort={onSort} sortKey="price" label="现算单价" numeric className="cl-num" />
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>{orderedRows.map((it) => rowBody(it, props.showProject))}</tbody>
    </table>
  );
});

// ── 定额 ─────────────────────────────────────────────────────────────

export interface QuotaTableProps {
  rows: WorkcostQuota[];
  usage: Map<string, number>;
  refPrices: Record<string, number>;
  checked: Set<string>;
  active: string | null;
  onToggle: (key: string, next: boolean) => void;
  onToggleAll: (next: boolean) => void;
  onOpen: (quota: WorkcostQuota) => void;
  onDelete: (quota: WorkcostQuota) => void;
}

export const QuotaTable = memo(function QuotaTable(props: QuotaTableProps) {
  const { rows, usage, refPrices, checked, active } = props;
  const [sort, setSort] = useState<SortState<QuotaSortKey> | null>(null);
  const allChecked = rows.length > 0 && rows.every((q) => checked.has(selKey("quota", q.code)));
  const orderedRows = orderRows(rows, sort, (row, key) => {
    switch (key) {
      case "code": return row.code;
      case "title": return row.title;
      case "unit": return row.unit;
      case "specialty": return row.specialty;
      case "chapter": return row.chapter;
      case "region": return row.region;
      case "usage": return usage.get(row.code) ?? 0;
      case "price": return typeof refPrices[row.code] === "number" ? refPrices[row.code] : null;
    }
  });
  const onSort = (key: QuotaSortKey) => setSort((current) => nextSort(current, key));
  return (
    <table className="cl-table" data-testid="cost-quota-rows">
      <thead>
        <tr>
          <th className="cl-check"><input type="checkbox" aria-label="全选当前定额" checked={allChecked} onChange={(e) => props.onToggleAll(e.target.checked)} /></th>
          <SortHeader sort={sort} onSort={onSort} sortKey="code" label="编码" />
          <SortHeader sort={sort} onSort={onSort} sortKey="title" label="名称" />
          <SortHeader sort={sort} onSort={onSort} sortKey="unit" label="单位" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="specialty" label="专业" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="chapter" label="分部" className="cl-pri-2" />
          <SortHeader sort={sort} onSort={onSort} sortKey="region" label="地区" className="cl-pri-2" />
          <SortHeader sort={sort} onSort={onSort} sortKey="usage" label="被引用" numeric className="cl-num cl-pri-2" />
          <SortHeader sort={sort} onSort={onSort} sortKey="price" label="综合单价" numeric className="cl-num" />
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>
        {orderedRows.map((q) => {
          const key = selKey("quota", q.code);
          const used = usage.get(q.code) ?? 0;
          const price = refPrices[q.code];
          return (
            <tr key={key} data-row-key={key} className={`cl-tr${active === key ? " is-active" : ""}`}
              onClick={() => props.onOpen(q)} title="点击查看含量行分析表">
              <CheckCell checked={checked.has(key)} label={`选择定额 ${q.title}`} onChange={(next) => props.onToggle(key, next)} />
              <td className="cl-cell-code">{q.code}</td>
              <td className="cl-cell-title">
                <button type="button" className="cl-cell-link" aria-label={`查看定额「${q.title}」`}
                  onClick={(e) => { e.stopPropagation(); props.onOpen(q); }}>{q.title}</button>
                {q.origCode && q.origCode !== q.code && <span className="cl-tag" title="工作簿原始编码">原码 {q.origCode}</span>}
                {q.derivedFrom && <span className="cl-tag is-derived" title="借鉴生成的溯源：本专业没有可用定额，以外专业定额为底生成">{q.derivedFrom}</span>}
              </td>
              <td className="cl-pri-1">{q.unit || "—"}</td>
              <td className={q.specialty ? "cl-pri-1" : "cl-pri-1 cl-dim"}>{dimLabel(q.specialty, "未标注")}</td>
              <td className={`cl-pri-2${q.chapter ? "" : " cl-dim"}`} title={q.chapter ? `分部 ${q.chapter}` : "还没归分部（待整理）"}>
                {dimLabel(q.chapter, "未归分部")}
              </td>
              {/* 地区与上游项目挂钩：随来源清单项目落库，空=还没从项目带回（待整理）。 */}
              <td className={`cl-pri-2${q.region ? "" : " cl-dim"}`} title={q.region ? `地区 ${q.region}` : "还没标注地区（随上游项目导入带上）"}>
                {dimLabel(q.region, "未标地区")}
              </td>
              <td className="cl-num cl-pri-2">{used > 0 ? used : <span className="cl-dim">—</span>}</td>
              <td className="cl-num cl-strong" title="按当前资源价现算（人材机，不含管理费/利润/税金）">{fmtMoney(typeof price === "number" ? price : null)}</td>
              <DeleteCell label={`删除定额「${q.title}」`} onClick={() => props.onDelete(q)} />
            </tr>
          );
        })}
      </tbody>
    </table>
  );
});

// ── 工料机 ───────────────────────────────────────────────────────────

export interface ResourceTableProps {
  rows: WorkcostResource[];
  checked: Set<string>;
  active: string | null;
  onToggle: (key: string, next: boolean) => void;
  onToggleAll: (next: boolean) => void;
  onOpen: (resource: WorkcostResource) => void;
  onDelete: (resource: WorkcostResource) => void;
}

export const ResourceTable = memo(function ResourceTable(props: ResourceTableProps) {
  const { rows, checked, active } = props;
  const [sort, setSort] = useState<SortState<ResourceSortKey> | null>(null);
  const allChecked = rows.length > 0 && rows.every((r) => checked.has(selKey("resource", r.code)));
  const orderedRows = orderRows(rows, sort, (row, key) => {
    switch (key) {
      case "code": return row.code;
      case "kind": return row.kind;
      case "title": return row.title;
      case "spec": return row.spec;
      case "unit": return row.unit;
      case "region": return row.region;
      case "period": return row.priceDate;
      case "price": {
        const value = effectivePrice(row);
        return value > 0 ? value : null;
      }
      case "basePrice": return row.basePrice > 0 ? row.basePrice : null;
    }
  });
  const onSort = (key: ResourceSortKey) => setSort((current) => nextSort(current, key));
  return (
    <table className="cl-table" data-testid="cost-resource-rows">
      <thead>
        <tr>
          <th className="cl-check"><input type="checkbox" aria-label="全选当前工料机" checked={allChecked} onChange={(e) => props.onToggleAll(e.target.checked)} /></th>
          <SortHeader sort={sort} onSort={onSort} sortKey="code" label="编码" />
          <SortHeader sort={sort} onSort={onSort} sortKey="kind" label="类别" />
          <SortHeader sort={sort} onSort={onSort} sortKey="title" label="名称" />
          <SortHeader sort={sort} onSort={onSort} sortKey="spec" label="规格" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="unit" label="单位" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="region" label="地区" className="cl-pri-1" />
          <SortHeader sort={sort} onSort={onSort} sortKey="period" label="期" className="cl-pri-2" />
          <SortHeader sort={sort} onSort={onSort} sortKey="price" label="取用价" numeric className="cl-num" />
          <SortHeader sort={sort} onSort={onSort} sortKey="basePrice" label="基准价" numeric className="cl-num cl-pri-2" />
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>
        {orderedRows.map((r) => {
          const key = selKey("resource", r.code);
          const price = effectivePrice(r);
          return (
            <tr key={key} data-row-key={key} className={`cl-tr${active === key ? " is-active" : ""}`}
              onClick={() => props.onOpen(r)} title="点击查看价格卡与调价历史">
              <CheckCell checked={checked.has(key)} label={`选择工料机 ${r.title}`} onChange={(next) => props.onToggle(key, next)} />
              <td className="cl-cell-code">{r.code}</td>
              <td><KindBadge kind={r.kind} /></td>
              <td className="cl-cell-title">
                <button type="button" className="cl-cell-link" aria-label={`查看工料机「${r.title}」`}
                  onClick={(e) => { e.stopPropagation(); props.onOpen(r); }}>{r.title}</button>
              </td>
              <td className="cl-cell-feature cl-pri-1" title={r.spec}>{r.spec || <span className="cl-dim">—</span>}</td>
              <td className="cl-pri-1">{r.unit || "—"}</td>
              {/* 工料机按地区分行：同一种材料每个地区一行；期是现行价所属的期。 */}
              <td className={`cl-pri-1${r.region ? "" : " cl-dim"}`} title={r.region ? `地区 ${r.region}` : "还没标注地区（待整理）"}>
                {dimLabel(r.region, "未标注")}
              </td>
              <td className={`cl-pri-2${r.priceDate ? "" : " cl-dim"}`} title={r.priceDate ? `现行价的期 ${fmtPeriod(r.priceDate)}` : "现行价还没有期（待整理）"}>
                {r.priceDate ? fmtPeriod(r.priceDate) : "无"}
              </td>
              <td className={`cl-num cl-strong${price <= 0 ? " is-zero" : ""}`}
                title={price <= 0 ? "现行价与基准价均为 0——引用它的定额子目按 0 计入" : r.currentPrice > 0 ? "现行价" : "现行价未录入，核算回退基准价"}>
                {fmtMoney(price > 0 ? price : null)}
              </td>
              <td className="cl-num cl-pri-2">{fmtMoney(r.basePrice > 0 ? r.basePrice : null)}</td>
              <DeleteCell label={`删除工料机「${r.title}」`} onClick={() => props.onDelete(r)} />
            </tr>
          );
        })}
      </tbody>
    </table>
  );
});
