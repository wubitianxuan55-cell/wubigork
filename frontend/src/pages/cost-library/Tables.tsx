// Tables — 三级列表（清单 / 定额 / 工料机）。只读浏览面：行点击看详情，复选框
// 批量删除，行尾删除。定额码是下钻入口（清单 → 定额）。状态全在页面，纯展示。
import { memo } from "react";
import { ChevronDown, ChevronRight, Trash2 } from "../../gaea/icons";
import type { WorkcostQuota, WorkcostResource } from "../../gaea/lib/types";
import {
  effectivePrice, fmtMoney, groupBill, linkedQuotas, selKey, unitPriceOf,
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
  const allChecked = rows.length > 0 && rows.every((it) => checked.has(selKey("bill", it.id)));
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
    const groups = groupBill(rows);
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
          {props.showProject && <th className="cl-pri-2">项目</th>}
          <th>编码</th><th>名称</th><th className="cl-pri-1">项目特征</th><th className="cl-pri-1">单位</th><th>挂接定额</th>
          <th className="cl-num">原表单价</th><th className="cl-num">现算单价</th>
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>{rows.map((it) => rowBody(it, props.showProject))}</tbody>
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
  const allChecked = rows.length > 0 && rows.every((q) => checked.has(selKey("quota", q.code)));
  return (
    <table className="cl-table" data-testid="cost-quota-rows">
      <thead>
        <tr>
          <th className="cl-check"><input type="checkbox" aria-label="全选当前定额" checked={allChecked} onChange={(e) => props.onToggleAll(e.target.checked)} /></th>
          <th>编码</th><th>名称</th><th className="cl-pri-1">单位</th><th className="cl-pri-1">专业</th><th className="cl-num cl-pri-2">被引用</th><th className="cl-num">综合单价</th>
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>
        {rows.map((q) => {
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
              </td>
              <td className="cl-pri-1">{q.unit || "—"}</td>
              <td className="cl-dim cl-pri-1">{q.specialty || "—"}</td>
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
  const allChecked = rows.length > 0 && rows.every((r) => checked.has(selKey("resource", r.code)));
  return (
    <table className="cl-table" data-testid="cost-resource-rows">
      <thead>
        <tr>
          <th className="cl-check"><input type="checkbox" aria-label="全选当前工料机" checked={allChecked} onChange={(e) => props.onToggleAll(e.target.checked)} /></th>
          <th>编码</th><th>类别</th><th>名称</th><th className="cl-pri-1">规格</th><th className="cl-pri-1">单位</th>
          <th className="cl-num">取用价</th><th className="cl-num cl-pri-2">基准价</th>
          <th className="cl-act"><span className="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => {
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
