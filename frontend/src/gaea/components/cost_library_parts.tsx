// cost_library_parts.tsx — 成本库展示件（批 34 FE2-02/FE1-05 拆分自
// CostLibraryView.tsx，原位搬移零逻辑改动）：分类树递归节点 CategoryNode、
// 列表/表格两视图（ListView/TableRow/TableView，React.memo props 稳定跳渲染）
// + CostRow 行件 + 人材机 MiniCompositionBar。SortKey 排序键类型两段共用，
// 落此单源。主文件经命名导入回接；CostRow/ListView/TableRow/TableView 由
// CostLibraryView.tsx re-export 保测试导入路径不变。
import { memo } from "react";
import {
  BarChart3, ChevronDown, ChevronRight, Clock, Coins, Pencil, Plus, Trash2,
} from "../icons";
import type { CostCategory, CostSummary } from "../lib/types";

export type SortKey = "title" | "price" | "updatedAt";

// ── 分类树节点（递归）──
export function CategoryNode({
  node,
  depth,
  selectedPath,
  pathById,
  expanded,
  onToggle,
  onSelect,
  onAddChild,
  onRename,
  onDelete,
}: {
  node: CostCategory;
  depth: number;
  selectedPath: string;
  pathById: Map<number, string>;
  expanded: Set<number>;
  onToggle: (id: number) => void;
  onSelect: (path: string) => void;
  onAddChild: (node: CostCategory) => void;
  onRename: (node: CostCategory) => void;
  onDelete: (node: CostCategory) => void;
}) {
  const path = pathById.get(node.id) ?? "";
  const hasChildren = !!node.children?.length;
  const isOpen = expanded.has(node.id);
  const active = selectedPath === path;
  return (
    <div>
      <div
        className={`group flex items-center gap-1 h-7 pr-1 rounded-md cursor-pointer transition-colors ${
          active ? "bg-accent/15 text-accent" : "text-fg-dim hover:bg-bg-soft"
        }`}
        style={{ paddingLeft: 6 + depth * 14 }}
        onClick={() => onSelect(path)}
        title={path}
      >
        <button
          className={`w-3.5 h-3.5 inline-flex items-center justify-center shrink-0 text-fg-faint ${hasChildren ? "" : "invisible"}`}
          onClick={(e) => {
            e.stopPropagation();
            onToggle(node.id);
          }}
        >
          {isOpen ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        </button>
        <span className="truncate text-[12px]">{node.name}</span>
        {node.count > 0 && (
          <span className="ml-auto shrink-0 text-[10px] text-fg-faint tabular-nums bg-bg-elev rounded-full px-1.5">{node.count}</span>
        )}
        <span
          className="hidden group-hover:flex items-center gap-0.5 shrink-0"
          onClick={(e) => e.stopPropagation()}
        >
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-accent hover:bg-bg-elev" title="添加子分类" onClick={() => onAddChild(node)}>
            <Plus size={11} />
          </button>
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" title="重命名" onClick={() => onRename(node)}>
            <Pencil size={11} />
          </button>
          <button className="w-5 h-5 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" title="删除" onClick={() => onDelete(node)}>
            <Trash2 size={11} />
          </button>
        </span>
      </div>
      {hasChildren && isOpen && (
        <div>
          {node.children!.map((child) => (
            <CategoryNode
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedPath={selectedPath}
              pathById={pathById}
              expanded={expanded}
              onToggle={onToggle}
              onSelect={onSelect}
              onAddChild={onAddChild}
              onRename={onRename}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface CostRowProps {
  row: CostSummary;
  selected: boolean;
  priceText: (p: number) => string;
  onToggleSelect: (name: string) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
}

interface ListViewProps {
  rows: CostSummary[];
  selected: Set<string>;
  toggleSelect: (name: string) => void;
  priceText: (p: number) => string;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
}

interface TableRowProps {
  row: CostSummary;
  selected: boolean;
  priceText: (p: number) => string;
  onToggleSelect: (name: string) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
}

interface TableViewProps {
  rows: CostSummary[];
  selected: Set<string>;
  toggleSelect: (name: string) => void;
  sortKey: SortKey | null;
  sortDir: 1 | -1;
  toggleSort: (k: SortKey) => void;
  priceText: (p: number) => string;
  onSelectAll: (next: Set<string>) => void;
  onEdit: (e: CostSummary) => void;
  onDelete: (name: string) => void;
  onHistory: (name: string) => void;
  onCompare: (e: CostSummary) => void;
}

// ── 列表项（React.memo：props 稳定（useCallback 回调）时跳过重渲染）──
export const CostRow = memo(function CostRow({
  row: e,
  selected,
  priceText,
  onToggleSelect,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
}: CostRowProps) {
  return (
    <div
      key={e.name}
      className={`group rounded-lg border p-2.5 transition-colors ${
        selected
          ? "border-accent/50 bg-accent/10"
          : "border-border/70 bg-bg-soft/40 hover:border-accent/30 hover:bg-bg-soft/70"
      }`}
    >
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={selected}
          onChange={() => onToggleSelect(e.name)}
          className="accent-[var(--accent)] shrink-0"
          title="多选"
        />
        <Coins size={14} className="text-amber-400 shrink-0" />
        <span className="text-fg font-medium truncate text-[13px]">{e.title}</span>
        <button
          className="w-5 h-5 shrink-0 inline-flex items-center justify-center rounded text-fg-faint opacity-0 group-hover:opacity-100 hover:text-sky-400 hover:bg-bg-elev transition-opacity"
          onClick={() => onCompare(e)}
          title="比价"
        >
          <BarChart3 size={11} />
        </button>
        {e.spec && <span className="text-fg-faint text-[11px] shrink-0 truncate max-w-[140px]">{e.spec}</span>}
        {e.code && (
          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px] font-mono shrink-0" title="定额/清单编码">
            {e.code}
          </span>
        )}
        {e.categoryPath && (
          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px] shrink-0 truncate max-w-[160px]" title={e.categoryPath}>
            {e.categoryPath}
          </span>
        )}
        {e.status !== "现行" && (
          <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px] shrink-0">{e.status}</span>
        )}
        <span className="ml-auto shrink-0 text-fg font-semibold text-amber-300 tabular-nums">
          {priceText(e.price)}
          {e.unit && <span className="text-fg-faint font-normal"> /{e.unit}</span>}
        </span>
        <span className="flex items-center gap-0.5 shrink-0">
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-sky-400 hover:bg-bg-elev" onClick={() => onCompare(e)} title="比价">
            <BarChart3 size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onHistory(e.name)} title="价格历史">
            <Clock size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onEdit(e)} title="编辑">
            <Pencil size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => onDelete(e.name)} title="删除">
            <Trash2 size={12} />
          </button>
        </span>
      </div>
      {(e.source || e.region || e.priceType || e.priceDate || e.validUntil) && (
        <div className="mt-0.5 pl-6 flex items-center gap-1.5 text-fg-faint text-[10.5px] min-w-0">
          {e.source && <span className="truncate">来源：{e.source}</span>}
          {e.region && <span className="shrink-0 truncate max-w-[90px]">· {e.region}</span>}
          {e.priceType && <span className="shrink-0 px-1 py-px rounded bg-bg-elev">{e.priceType}</span>}
          {e.priceDate && <span className="shrink-0 truncate max-w-[110px]">· {e.priceDate}</span>}
          {e.validUntil && <span className="shrink-0">· 至 {e.validUntil}</span>}
        </div>
      )}
      {((e.laborFee ?? 0) > 0 || (e.materialFee ?? 0) > 0 || (e.machineFee ?? 0) > 0) && (
        <div className="mt-1 pl-6 flex items-center gap-2 text-[10.5px] text-fg-faint min-w-0">
          <span className="shrink-0 font-medium">人材机</span>
          <div className="shrink-0 w-24">
            <MiniCompositionBar labor={e.laborFee ?? 0} material={e.materialFee ?? 0} machine={e.machineFee ?? 0} />
          </div>
          <span className="shrink-0 tabular-nums text-sky-400/90">人工 {priceText(e.laborFee ?? 0)}</span>
          <span className="shrink-0 tabular-nums text-emerald-400/90">材料 {priceText(e.materialFee ?? 0)}</span>
          <span className="shrink-0 tabular-nums text-amber-400/90">机械 {priceText(e.machineFee ?? 0)}</span>
        </div>
      )}
    </div>
  );
});

// ── 人材机组成 mini 条（列表行内：人工/材料/机械 占比）─────────────
function MiniCompositionBar({ labor, material, machine }: { labor: number; material: number; machine: number }) {
  const total = labor + material + machine;
  if (total <= 0) return null;
  const seg = (v: number, cls: string, label: string) =>
    v > 0 ? (
      <div
        className={`h-full ${cls}`}
        style={{ width: `${Math.max(2, (v / total) * 100)}%` }}
        title={`${label} ${v.toFixed(2)}`}
      />
    ) : null;
  return (
    <div
      className="flex h-1 rounded-full overflow-hidden bg-bg-elev"
      role="img"
      aria-label={`人材机：人工 ${labor.toFixed(2)}，材料 ${material.toFixed(2)}，机械 ${machine.toFixed(2)}`}
    >
      {seg(labor, "bg-sky-400/80", "人工")}
      {seg(material, "bg-emerald-400/80", "材料")}
      {seg(machine, "bg-amber-400/80", "机械")}
    </div>
  );
}

// ── 列表视图 ──
export const ListView = memo(function ListView({
  rows,
  selected,
  toggleSelect,
  priceText,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
}: ListViewProps) {
  return (
    <div className="px-4 pb-4 space-y-1.5">
      {rows.map((e) => (
        <CostRow
          key={e.name}
          row={e}
          selected={selected.has(e.name)}
          priceText={priceText}
          onToggleSelect={toggleSelect}
          onEdit={onEdit}
          onDelete={onDelete}
          onHistory={onHistory}
          onCompare={onCompare}
        />
      ))}
    </div>
  );
});

// ── 表格行（React.memo：props 稳定时跳过重渲染）──
export const TableRow = memo(function TableRow({
  row: e,
  selected,
  priceText,
  onToggleSelect,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
}: TableRowProps) {
  return (
    <tr key={e.name} className={`border-b border-border-soft/50 hover:bg-bg-soft/50 transition-colors ${selected ? "bg-accent/10" : ""}`}>
      <td className="px-3 py-1.5">
        <input type="checkbox" checked={selected} onChange={() => onToggleSelect(e.name)} className="accent-[var(--accent)]" />
      </td>
      <td className="px-3 py-1.5">
        <div className="text-fg font-medium truncate max-w-[220px]">{e.title}</div>
        <div className="text-fg-faint text-[10px] font-mono truncate max-w-[220px]">{e.code ? `${e.code} · ` : ""}{e.name}</div>
      </td>
      <td className="px-3 py-1.5 text-fg-dim whitespace-nowrap max-w-[180px] truncate" title={e.categoryPath}>
        {e.categoryPath || e.category || "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-dim">{e.spec || "—"}</td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap max-w-[150px] truncate" title={[e.region, e.priceDate].filter(Boolean).join(" · ")}>
        {[e.region, e.priceDate].filter(Boolean).join(" · ") || "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-dim">{e.unit || "—"}</td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap">
        {(e.laborFee ?? 0) > 0 || (e.materialFee ?? 0) > 0 || (e.machineFee ?? 0) > 0 ? (
          <span className="tabular-nums">
            人 {priceText(e.laborFee ?? 0)} · 材 {priceText(e.materialFee ?? 0)} · 机 {priceText(e.machineFee ?? 0)}
          </span>
        ) : (
          "—"
        )}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] text-center tabular-nums">
        {(e.componentCount ?? 0) > 0 ? `${e.componentCount} 行` : "—"}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] whitespace-nowrap">{e.priceType || "—"}</td>
      <td className="px-3 py-1.5 text-right text-amber-300 font-semibold tabular-nums whitespace-nowrap">
        {priceText(e.price)}
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[11px] max-w-[140px] truncate" title={e.source}>{e.source || "—"}</td>
      <td className="px-3 py-1.5">
        <span className={`px-1.5 py-0.5 rounded text-[10px] ${e.status === "现行" ? "bg-emerald-500/15 text-emerald-400" : e.status === "草稿" ? "bg-amber-500/15 text-amber-400" : "bg-bg-elev text-fg-faint"}`}>
          {e.status || "现行"}
        </span>
      </td>
      <td className="px-3 py-1.5 text-fg-faint text-[10.5px] tabular-nums whitespace-nowrap">
        {e.updatedAt ? new Date(e.updatedAt).toLocaleDateString("zh-CN") : "—"}
      </td>
      <td className="px-3 py-1.5">
        <div className="flex items-center justify-end gap-0.5">
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-sky-400 hover:bg-bg-elev" onClick={() => onCompare(e)} title="比价">
            <BarChart3 size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onHistory(e.name)} title="价格历史">
            <Clock size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-fg hover:bg-bg-elev" onClick={() => onEdit(e)} title="编辑">
            <Pencil size={12} />
          </button>
          <button className="w-6 h-6 inline-flex items-center justify-center rounded text-fg-faint hover:text-red-400 hover:bg-bg-elev" onClick={() => onDelete(e.name)} title="删除">
            <Trash2 size={12} />
          </button>
        </div>
      </td>
    </tr>
  );
});

// ── 表格视图 ──
export const TableView = memo(function TableView({
  rows,
  selected,
  toggleSelect,
  sortKey,
  sortDir,
  toggleSort,
  priceText,
  onSelectAll,
  onEdit,
  onDelete,
  onHistory,
  onCompare,
}: TableViewProps) {
  const sortArrow = (k: SortKey) => (sortKey === k ? (sortDir === 1 ? " ↑" : " ↓") : "");
  const th = (label: string, k?: SortKey, align: "left" | "right" = "left") => (
    <th
      className={`px-3 py-2 font-medium text-fg-faint text-[11px] whitespace-nowrap select-none ${align === "right" ? "text-right" : "text-left"} ${k ? "cursor-pointer hover:text-fg" : ""}`}
      onClick={k ? () => toggleSort(k) : undefined}
    >
      {label}
      {k ? sortArrow(k) : ""}
    </th>
  );
  return (
    <div className="px-3 pb-4">
      <table className="w-full text-[12px] border-collapse">
        <thead>
          <tr className="border-b border-border-soft/80">
            <th className="px-3 py-2 w-8">
              <input
                type="checkbox"
                checked={rows.length > 0 && selected.size === rows.length}
                onChange={(e) => {
                  if (e.target.checked) setAllSelected();
                  else clearSelection();
                }}
                className="accent-[var(--accent)]"
              />
            </th>
            {th("标题", "title")}
            {th("分类")}
            {th("规格")}
            {th("地区 · 期数")}
            {th("单位")}
            {th("人材机")}
            {th("组成")}
            {th("口径")}
            {th("单价（元）", "price", "right")}
            {th("来源")}
            {th("状态")}
            {th("更新", "updatedAt")}
            <th className="px-3 py-2 text-right font-medium text-fg-faint text-[11px] whitespace-nowrap">操作</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => (
            <TableRow
              key={e.name}
              row={e}
              selected={selected.has(e.name)}
              priceText={priceText}
              onToggleSelect={toggleSelect}
              onEdit={onEdit}
              onDelete={onDelete}
              onHistory={onHistory}
              onCompare={onCompare}
            />
          ))}
        </tbody>
      </table>
    </div>
  );

  function setAllSelected() {
    onSelectAll(new Set(rows.map((e) => e.name)));
  }
  function clearSelection() {
    onSelectAll(new Set());
  }
});
