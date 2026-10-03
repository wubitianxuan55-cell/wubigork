// cost_projects_parts.tsx — 成本项目明细件（批 35 FE2-04 拆分自
// CostProjectsView.tsx，原位搬移零逻辑改动）：可编辑明细行 ItemRow、成本库
// 单价搜索下拉 EntryPicker（自带 debounce）、项目表单 ProjectForm/Field、
// 快照表格 SnapshotTable、StatusBadge/fmtTime/slug/emptyProject 小工具。
// 主组件经命名导入回接（八件全部为主段消费）；无外部消费者，测试只导主件。
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { message } from "antd";
import { Search, X } from "../../icons";
import { app } from "../../lib/bridge";
import { costReadErrorText } from "../../lib/bridge/cost";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import type { CostEstimateItem, CostEstimateVersion, CostProject, CostSummary } from "../../lib/types";

export const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);

// 与既有组件一致的内联样式（项目无自定义 .field/.btn-* 类，全部 Tailwind）
export const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
export const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
export const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
export const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

export { ItemRow, EntryPicker, ProjectForm, SnapshotTable, StatusBadge, fmtTime, slug, emptyProject };

function ItemRow({
  item,
  selected,
  onToggle,
  onPatch,
  onSave,
  onDelete,
}: {
  item: CostEstimateItem;
  selected: boolean;
  onToggle: (c: boolean) => void;
  onPatch: (p: Partial<CostEstimateItem>) => void;
  onSave: () => void;
  onDelete: () => void;
}) {
  const [pickOpen, setPickOpen] = useState(false);
  const amount = (item.quantity || 0) * (item.price || 0);
  return (
    <tr className="border-b border-border-soft/30 last:border-0 hover:bg-bg-elev/30 align-top">
      <td className="py-1 px-2">
        <input
          type="checkbox"
          className="accent-accent"
          checked={selected}
          disabled={item.id === undefined}
          title={item.id === undefined ? "先保存该行再沉淀" : "选择以沉淀回成本库"}
          onChange={(e) => onToggle(e.target.checked)}
        />
      </td>
      <td className="py-1 px-2">
        <input
          className={fieldCls}
          value={item.title}
          placeholder="名称（必填）"
          onChange={(e) => onPatch({ title: e.target.value, name: slug(e.target.value) })}
          onBlur={onSave}
        />
        <div className="relative mt-1">
          <div className="flex items-center gap-1">
            <Search size={10} className="text-fg-faint shrink-0" />
            <input
              className={`${fieldCls} !text-[10.5px]`}
              value={item.entryName || ""}
              placeholder="引用成本库单价（搜索）或留空手动估价"
              onChange={(e) => onPatch({ entryName: e.target.value })}
              onFocus={() => setPickOpen(true)}
              onBlur={() => setTimeout(() => setPickOpen(false), 200)}
            />
          </div>
          {pickOpen && (
            <EntryPicker
              query={item.entryName || ""}
              onPick={(e) => {
                onPatch({
                  title: e.title,
                  name: e.name,
                  code: e.code,
                  unit: e.unit,
                  price: e.price,
                  categoryPath: e.categoryPath || e.category || "",
                  entryName: e.name,
                  source: e.source || "成本库引用",
                });
                setPickOpen(false);
                onSave();
              }}
            />
          )}
        </div>
        <input
          className={`${fieldCls} !text-[10.5px] mt-1 font-mono`}
          value={item.code || ""}
          placeholder="定额/清单编码（可选）"
          onChange={(e) => onPatch({ code: e.target.value })}
          onBlur={onSave}
        />
      </td>
      <td className="py-1 px-2">
        <input className={fieldCls} value={item.unit} placeholder="单位" onChange={(e) => onPatch({ unit: e.target.value })} onBlur={onSave} />
      </td>
      <td className="py-1 px-2">
        <input
          className={`${fieldCls} text-right tabular-nums`}
          type="number"
          min={0}
          value={item.quantity || ""}
          placeholder="数量"
          onChange={(e) => onPatch({ quantity: Number(e.target.value) })}
          onBlur={onSave}
        />
      </td>
      <td className="py-1 px-2">
        <input
          className={`${fieldCls} text-right tabular-nums`}
          type="number"
          min={0}
          value={item.price || ""}
          placeholder="单价"
          onChange={(e) => onPatch({ price: Number(e.target.value) })}
          onBlur={onSave}
        />
      </td>
      <td className="py-1 px-2 text-right tabular-nums text-fg font-medium">{fmtPrice(amount)}</td>
      <td className="py-1 px-2">
        <button type="button" className={`${iconBtn} text-fg-faint hover:text-err`} title="删除行" onClick={onDelete}>
          <X size={12} />
        </button>
      </td>
    </tr>
  );
}

// ── 成本库单价搜索下拉 ────────────────────────────────────────
function EntryPicker({ query, onPick }: { query: string; onPick: (e: CostSummary) => void }) {
  const [q] = useState(query);
  const [results, setResults] = useState<CostSummary[]>([]);
  // GA6-09：Go GaeaCostSearchPage 读取失败时 error 非空、items 仍是已读到的部分
  // ——非空即显形提示条（含重试），绝不把「读坏了」装成「无匹配条目」。
  const [readError, setReadError] = useState("");
  const seqRef = useRef(0);
  const debounced = useDebouncedValue(q, 250);
  const runSearch = useCallback(() => {
    if (!debounced.trim()) {
      setResults([]);
      setReadError("");
      return () => {};
    }
    const seq = ++seqRef.current;
    let alive = true;
    // v4.386 切分页绑定：只要 top 8，不再全表拉回后客户端截断。
    app
      .CostSearchPage(debounced, "", "", "", 1, 8, 0)
      .then((r) => {
        if (!alive || seq !== seqRef.current) return;
        setResults(r?.items ?? []);
        setReadError(costReadErrorText(r?.error));
      })
      .catch((e: unknown) => {
        if (!alive || seq !== seqRef.current) return;
        setResults([]);
        setReadError(costReadErrorText(e instanceof Error ? e.message : String(e)));
      });
    return () => {
      alive = false;
    };
  }, [debounced]);
  useEffect(() => runSearch(), [runSearch]);
  return (
    <div className="absolute z-20 left-0 right-0 top-full mt-0.5 rounded-lg border border-border bg-bg-elev shadow-xl overflow-hidden">
      {readError && (
        <div
          role="alert"
          data-testid="cost-picker-read-error"
          className="flex items-center gap-2 px-2.5 py-1.5 border-b border-amber-400/30 bg-amber-400/10 text-amber-200 text-[10.5px]"
        >
          <span className="flex-1 min-w-0">{readError}——结果可能不完整。</span>
          <button
            type="button"
            data-testid="cost-picker-read-error-retry"
            onClick={() => runSearch()}
            className="shrink-0 px-1.5 h-5 rounded border border-amber-400/40 hover:bg-amber-400/20 transition-colors"
          >
            重试
          </button>
        </div>
      )}
      {results.length === 0 ? (
        <div className="px-2.5 py-2 text-[10.5px] text-fg-faint">
          {readError ? "成本库读取失败，未取到条目——点上方「重试」重新读取" : "输入关键词搜索成本库单价…"}
        </div>
      ) : (
        results.map((e) => (
          <button
            key={e.name}
            type="button"
            className="w-full text-left px-2.5 py-1.5 hover:bg-accent/10 transition-colors"
            onMouseDown={(ev) => ev.preventDefault()}
            onClick={() => onPick(e)}
          >
            <span className="block truncate text-[11px] text-fg">{e.title}</span>
            <span className="block truncate text-[9.5px] text-fg-faint">
              {e.categoryPath || e.category || "未分类"} · {e.unit || "-"} · <span className="tabular-nums">{fmtPrice(e.price)}</span>
            </span>
          </button>
        ))
      )}
    </div>
  );
}

// ── 项目表单 ──────────────────────────────────────────────────
function ProjectForm({ value, onChange }: { value: CostProject; onChange: (p: CostProject) => void }) {
  const set = (patch: Partial<CostProject>) => onChange({ ...value, ...patch });
  return (
    <div className="space-y-2.5">
      <Field label="项目名称" required>
        <input className={fieldCls} value={value.name} onChange={(e) => set({ name: e.target.value })} placeholder="如：XX 市政道路土方测算" autoFocus />
      </Field>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="项目类型">
          <input className={fieldCls} value={value.projectType} onChange={(e) => set({ projectType: e.target.value })} placeholder="房建 / 市政 / 安装…" />
        </Field>
        <Field label="规模">
          <input className={fieldCls} value={value.scale} onChange={(e) => set({ scale: e.target.value })} placeholder="如：5 万 m²" />
        </Field>
      </div>
      <Field label="工艺 / 说明">
        <input className={fieldCls} value={value.craft} onChange={(e) => set({ craft: e.target.value })} placeholder="施工工艺或备注" />
      </Field>
      <Field label="备注">
        <textarea className={fieldCls} rows={2} value={value.note} onChange={(e) => set({ note: e.target.value })} placeholder="可选" />
      </Field>
    </div>
  );
}

function Field({ label, required, children }: { label: string; required?: boolean; children: ReactNode }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">
        {label}
        {required && <span className="text-err"> *</span>}
      </span>
      {children}
    </label>
  );
}

// ── 快照表格 ──────────────────────────────────────────────────
function SnapshotTable({ v }: { v: CostEstimateVersion }) {
  let rows: CostEstimateItem[] = [];
  try {
    rows = JSON.parse(v.snapshot);
  } catch {
    rows = [];
  }
  return (
    <div>
      <p className="text-[11px] text-fg-faint mb-2">
        {v.note || "（无备注）"} · 保存于 {fmtTime(v.createdAt)} · 合计 {fmtPrice(v.total)}
      </p>
      {rows.length === 0 ? (
        <div className="text-[11.5px] text-fg-faint py-6 text-center">快照为空或已损坏</div>
      ) : (
        <table className="w-full text-[11.5px]">
          <thead>
            <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
              <th className="py-1.5 px-2">名称</th>
              <th className="py-1.5 px-2 w-14">单位</th>
              <th className="py-1.5 px-2 w-16 text-right">数量</th>
              <th className="py-1.5 px-2 w-24 text-right">单价</th>
              <th className="py-1.5 px-2 w-24 text-right">金额</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i} className="border-b border-border-soft/20 last:border-0">
                <td className="py-1 px-2 text-fg-dim">{r.title}</td>
                <td className="py-1 px-2 text-fg-faint">{r.unit}</td>
                <td className="py-1 px-2 text-right tabular-nums">{r.quantity}</td>
                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice(r.price)}</td>
                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice((r.quantity || 0) * (r.price || 0))}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// ── 小工具 ────────────────────────────────────────────────────
function StatusBadge({ status }: { status: string }) {
  const tone =
    status === "已沉淀"
      ? "text-ok bg-ok/10"
      : status === "已保存版本"
        ? "text-accent bg-accent/10"
        : "text-amber-400 bg-amber-400/10";
  return <span className={`shrink-0 px-1.5 py-px rounded text-[9.5px] ${tone}`}>{status || "编制中"}</span>;
}

function fmtTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

function slug(s: string): string {
  return s
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}

function emptyProject(): CostProject {
  return { id: "", name: "", projectType: "", scale: "", craft: "", status: "编制中", note: "" };
}
