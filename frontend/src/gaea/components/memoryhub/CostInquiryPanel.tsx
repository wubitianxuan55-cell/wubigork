import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Modal } from "antd";
import {
  AlertCircle, ChevronDown, ChevronRight, Pencil, Plus, Save, Search, Trash2, TrendingUp,
} from "../../icons";
import { app } from "../../lib/bridge";
import type { CostAdjustSuggestion, CostInquiryRecord } from "../../lib/types";
import { useToast } from "../Toast";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { EmptyState } from "../EmptyState";

// ── 四源归一（信息价 / OCR报价 / 供应商比价 / 手动询价）─────────────
const SOURCES = ["信息价", "OCR报价", "供应商比价", "手动询价"] as const;

// 四源徽标配色（不同颜色区分来源）。
const SOURCE_STYLES: Record<string, string> = {
  信息价: "bg-sky-400/15 text-sky-400",
  OCR报价: "bg-purple-400/15 text-purple-400",
  供应商比价: "bg-emerald-400/15 text-emerald-400",
  手动询价: "bg-amber-400/15 text-amber-400",
};

const fmtPrice = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const fmtPct = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 });

// 调差差幅着色：|diffPct|≥10 红（大幅波动）、>2 琥珀（关注），
// 其余按方向红升绿降（价格升为红系、降为绿系）。
const diffClass = (p: number) => {
  const a = Math.abs(p);
  if (a >= 10) return "text-red-400";
  if (a > 2) return "text-amber-400";
  return p >= 0 ? "text-red-400/80" : "text-emerald-400";
};

// 与 CostLibraryView / CostProjectsView 一致的 Tailwind class 常量。
const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity";
const iconMini =
  "inline-flex items-center justify-center w-6 h-6 rounded-md text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

function blankRecord(): CostInquiryRecord {
  return {
    id: 0,
    title: "",
    spec: "",
    unit: "",
    price: 0,
    source: "手动询价",
    supplier: "",
    region: "",
    priceDate: "",
    validUntil: "",
    note: "",
    status: "",
    createdAt: "",
    updatedAt: "",
  };
}

/**
 * CostInquiryPanel — v4.2 询价飞轮前端面板（四源归一数据点：信息价 /
 * OCR报价 / 供应商比价 / 手动询价）。
 *
 * 自上而下：到期预警横幅（30 天内到期）→ 搜索 + 新增询价 → 数据点列表 →
 * 调差建议（成本库条目 vs 最新询价，|差幅|>2%）。
 */
export function CostInquiryPanel({ compact = false }: { compact?: boolean }) {
  const toast = useToast();
  const [records, setRecords] = useState<CostInquiryRecord[]>([]);
  const [expiring, setExpiring] = useState<CostInquiryRecord[]>([]);
  const [adjust, setAdjust] = useState<CostAdjustSuggestion[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<CostInquiryRecord | null>(null);
  const [form, setForm] = useState<CostInquiryRecord>(blankRecord);
  const [adjustOpen, setAdjustOpen] = useState(true);

  // 搜索防抖 250ms（与成本库一致：清空即时生效）。
  const debouncedQuery = useDebouncedValue(query, 250);

  const reloadRecords = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const list = (await app.CostInquiryList(q, 100)) ?? [];
      setRecords(list);
    } catch {
      setRecords([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // 到期预警 + 调差建议一并刷新（保存/删除/更新成本库后调用）。
  const reloadMeta = useCallback(() => {
    app
      .CostInquiryExpiring(30)
      .then((r) => setExpiring(r ?? []))
      .catch(() => setExpiring([]));
    app
      .CostInquiryAdjust()
      .then((r) => setAdjust(r ?? []))
      .catch(() => setAdjust([]));
  }, []);

  const reloadAll = useCallback(() => {
    void reloadRecords(debouncedQuery);
    reloadMeta();
  }, [debouncedQuery, reloadRecords, reloadMeta]);

  useEffect(() => {
    void reloadRecords(debouncedQuery);
  }, [debouncedQuery, reloadRecords]);

  useEffect(() => {
    reloadMeta();
  }, [reloadMeta]);

  const setField = useCallback((patch: Partial<CostInquiryRecord>) => {
    setForm((f) => ({ ...f, ...patch }));
  }, []);

  const openCreate = useCallback(() => {
    setEditing(null);
    setForm(blankRecord());
    setModalOpen(true);
  }, []);

  const openEdit = useCallback((r: CostInquiryRecord) => {
    setEditing(r);
    setForm({ ...r });
    setModalOpen(true);
  }, []);

  const save = async () => {
    if (!form.title.trim()) {
      toast.show("请输入品名", "warn");
      return;
    }
    if (!(form.price > 0)) {
      toast.show("请输入有效单价", "warn");
      return;
    }
    setSaving(true);
    try {
      await app.CostInquirySave(form);
      toast.show(editing ? "询价已更新" : "询价已保存", "info");
      setModalOpen(false);
      reloadAll();
    } catch (e) {
      toast.show(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  const remove = useCallback(
    (r: CostInquiryRecord) => {
      Modal.confirm({
        title: "删除询价数据",
        content: `删除「${r.title}」的询价记录？`,
        okText: "删除",
        okButtonProps: { danger: true },
        cancelText: "取消",
        onOk: async () => {
          try {
            await app.CostInquiryDelete(r.id);
            toast.show("询价已删除", "info");
          } catch (e) {
            toast.show(String(e), "error");
          }
          reloadAll();
        },
      });
    },
    [reloadAll, toast],
  );

  // 调差建议「更新成本库」：CostGet 取原条目（保留 unit/分类等信息），仅改 price 后 CostSave。
  const applyAdjust = useCallback(
    async (s: CostAdjustSuggestion) => {
      try {
        const entry = await app.CostGet(s.entryName);
        if (!entry) {
          toast.show(`成本条目「${s.entryTitle}」不存在`, "warn");
          return;
        }
        await app.CostSave({ ...entry, price: s.latestPrice });
        toast.show(`已更新成本库：${s.entryTitle} ¥${fmtPrice.format(s.latestPrice)}`, "info");
        reloadMeta();
      } catch (e) {
        toast.show(String(e), "error");
      }
    },
    [reloadMeta, toast],
  );

  return (
    <div className={`h-full flex flex-col min-h-0 ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}>
      {/* ① 到期预警横幅：valid_until 在 30 天内的数据点 */}
      {expiring.length > 0 && (
        <div className="shrink-0 mx-3 mt-2 rounded-lg border border-err/40 bg-err/10 px-3 py-2">
          <div className="flex items-center gap-1.5">
            <AlertCircle size={13} className="text-err shrink-0" />
            <span className="text-err text-[12px] font-semibold">⚠ {expiring.length} 条询价到期预警</span>
            <span className="text-err/80 text-[10.5px]">有效期 30 天内，请及时复核询价</span>
          </div>
          <div className="mt-1.5 space-y-1">
            {expiring.map((r) => (
              <div key={r.id} className="flex items-center gap-2 text-[11px] text-fg-dim min-w-0">
                <span className="min-w-0 truncate font-medium text-fg">{r.title}</span>
                {r.spec && <span className="shrink-0 truncate max-w-[120px] text-fg-faint">{r.spec}</span>}
                <span className="ml-auto shrink-0 tabular-nums text-amber-300">¥{fmtPrice.format(r.price)}</span>
                <span className="shrink-0 text-err/90">至 {r.validUntil || "—"}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ② 工具栏：搜索（防抖 250ms）+ 新增询价 */}
      <div className="shrink-0 flex items-center gap-2 px-3 py-2">
        <div className="flex items-center gap-1.5 px-2 h-7 rounded-lg border border-border bg-bg">
          <Search size={11} className="text-fg-faint shrink-0" />
          <input
            className="w-36 sm:w-48 bg-transparent text-[11.5px] text-fg outline-none placeholder:text-fg-faint/50"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索品名/规格/供应商…"
          />
        </div>
        <span className="ml-auto text-fg-faint text-[10.5px] tabular-nums">{records.length} 条</span>
        <button type="button" className={solidBtn} onClick={openCreate} title="新增询价">
          <Plus size={12} /> 新增询价
        </button>
      </div>

      {/* ③ 数据点列表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-3 pb-2">
        {loading && records.length === 0 ? (
          <div className="space-y-2 animate-pulse">
            {Array.from({ length: compact ? 3 : 5 }).map((_, i) => (
              <div key={i} className="h-12 rounded-lg bg-bg-elev/60" />
            ))}
          </div>
        ) : records.length === 0 ? (
          <div className="h-full flex items-center justify-center">
            <EmptyState message="暂无询价数据——导入报价单（成本库导入）或手动录入，询价越多调差越准" />
          </div>
        ) : (
          <div className="space-y-1.5">
            {records.map((r) => (
              <div
                key={r.id}
                className="rounded-lg border border-border/70 bg-bg-soft/40 px-2.5 py-2 transition-colors hover:border-accent/30 hover:bg-bg-soft/70"
              >
                <div className="flex items-center gap-2 min-w-0">
                  <span className="text-fg font-medium text-[12.5px] truncate">{r.title}</span>
                  <SourceBadge source={r.source} />
                  {r.spec && <span className="text-fg-faint text-[11px] truncate max-w-[130px]">{r.spec}</span>}
                  <span className="ml-auto shrink-0 text-amber-300 font-semibold tabular-nums text-[12px]">
                    ¥{fmtPrice.format(r.price)}
                    {r.unit && <span className="text-fg-faint font-normal"> /{r.unit}</span>}
                  </span>
                  <span className="flex items-center gap-0.5 shrink-0">
                    <button type="button" className={iconMini} onClick={() => openEdit(r)} title="编辑">
                      <Pencil size={12} />
                    </button>
                    <button type="button" className={`${iconMini} hover:text-err`} onClick={() => remove(r)} title="删除">
                      <Trash2 size={12} />
                    </button>
                  </span>
                </div>
                <div className="mt-0.5 flex flex-wrap items-center gap-x-1.5 gap-y-0.5 pl-1 text-fg-faint text-[10.5px] min-w-0">
                  {r.supplier && <span className="truncate max-w-[160px]">供应商：{r.supplier}</span>}
                  {r.region && <span>· {r.region}</span>}
                  {r.priceDate && <span>· 期数 {r.priceDate}</span>}
                  {r.validUntil && <span>· 至 {r.validUntil}</span>}
                  {r.note && (
                    <span className="truncate max-w-[200px]" title={r.note}>
                      · {r.note}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* ⑤ 调差建议区（折叠面板） */}
      <div className="shrink-0 border-t border-border-soft/70">
        <button
          type="button"
          className="flex w-full items-center gap-1.5 px-3 py-2 text-left hover:bg-bg-soft/50 transition-colors"
          onClick={() => setAdjustOpen((o) => !o)}
          aria-expanded={adjustOpen}
        >
          {adjustOpen ? (
            <ChevronDown size={12} className="text-fg-faint shrink-0" />
          ) : (
            <ChevronRight size={12} className="text-fg-faint shrink-0" />
          )}
          <TrendingUp size={12} className="text-amber-400 shrink-0" />
          <span className="text-fg font-medium text-[12px]">调差建议</span>
          <span className="text-fg-faint text-[10.5px]">成本库 vs 最新询价（|差幅|&gt;2%）</span>
          {adjust.length > 0 && (
            <span className="ml-auto shrink-0 px-1.5 py-px rounded-full bg-amber-400/15 text-amber-300 text-[10px] tabular-nums">
              {adjust.length}
            </span>
          )}
        </button>
        {adjustOpen && (
          <div className="px-3 pb-2 space-y-1.5 max-h-[32vh] overflow-y-auto">
            {adjust.length === 0 ? (
              <div className="py-2 text-center text-fg-faint text-[11px]">暂无调差建议——成本库与最新询价差幅都在 2% 以内</div>
            ) : (
              adjust.map((s) => (
                <div key={s.entryName} className="flex items-center gap-2 rounded-lg border border-border/70 bg-bg-soft/40 px-2.5 py-1.5">
                  {s.level && (
                    <span
                      className={`shrink-0 px-1.5 py-px rounded text-[9.5px] font-medium ${
                        s.level === "异常"
                          ? "bg-red-500/15 text-red-400"
                          : s.level === "关注"
                            ? "bg-amber-400/15 text-amber-300"
                            : "bg-emerald-500/15 text-emerald-400"
                      }`}
                    >
                      {s.level}
                    </span>
                  )}
                  <span className="min-w-0 truncate text-fg text-[11.5px] font-medium">{s.entryTitle}</span>
                  <span className="shrink-0 text-fg-faint text-[10.5px] tabular-nums">
                    {`¥${fmtPrice.format(s.entryPrice)} → ¥${fmtPrice.format(s.latestPrice)}`}
                  </span>
                  <span className={`shrink-0 text-[11px] font-semibold tabular-nums ${diffClass(s.diffPct)}`}>
                    {s.diffPct > 0 ? "+" : ""}
                    {fmtPct.format(s.diffPct)}%
                  </span>
                  {(s.latestDate || s.latestSource) && (
                    <span className="hidden lg:inline shrink-0 text-fg-faint text-[10px]">
                      {[s.latestDate, s.latestSource].filter(Boolean).join(" · ")}
                    </span>
                  )}
                  {!!s.predictedNext && s.predictedNext > 0 && (
                    <span
                      className="hidden xl:inline shrink-0 text-[10px] tabular-nums"
                      style={{ color: "var(--color-info)" }}
                      title={s.predictionNote ?? "询价序列线性回归"}
                    >
                      预测下期 ¥{fmtPrice.format(s.predictedNext)}
                    </span>
                  )}
                  <button
                    type="button"
                    className="ml-auto shrink-0 inline-flex items-center gap-1 px-2 h-6 rounded-md bg-accent text-white text-[10.5px] hover:opacity-90 transition-opacity"
                    onClick={() => void applyAdjust(s)}
                  >
                    <Save size={11} /> 更新成本库
                  </button>
                </div>
              ))
            )}
          </div>
        )}
      </div>

      {/* ④ 新增/编辑询价弹窗 */}
      <Modal
        title={editing ? "编辑询价" : "新增询价"}
        open={modalOpen}
        onOk={() => void save()}
        onCancel={() => setModalOpen(false)}
        okText="保存"
        cancelText="取消"
        confirmLoading={saving}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={600}
      >
        <div className="space-y-2.5 py-1">
          <div className="grid grid-cols-2 gap-2.5">
            <Field label="品名">
              <input
                className={fieldCls}
                aria-label="品名"
                value={form.title}
                onChange={(e) => setField({ title: e.target.value })}
                placeholder="如：P.O 42.5 水泥"
                autoFocus
              />
            </Field>
            <Field label="规格">
              <input
                className={fieldCls}
                aria-label="规格"
                value={form.spec}
                onChange={(e) => setField({ spec: e.target.value })}
                placeholder="如：散装 / Φ16"
              />
            </Field>
          </div>
          <div className="grid grid-cols-3 gap-2.5">
            <Field label="来源">
              <select
                className={fieldCls}
                aria-label="来源"
                value={form.source}
                onChange={(e) => setField({ source: e.target.value })}
              >
                {SOURCES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="单价（元）">
              <input
                className={fieldCls}
                type="number"
                min={0}
                step="any"
                aria-label="单价"
                value={form.price || ""}
                onChange={(e) => setField({ price: Number(e.target.value) || 0 })}
                placeholder="380"
              />
            </Field>
            <Field label="单位">
              <input
                className={fieldCls}
                aria-label="单位"
                value={form.unit}
                onChange={(e) => setField({ unit: e.target.value })}
                placeholder="吨 / m³ / 台班"
              />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <Field label="供应商">
              <input
                className={fieldCls}
                aria-label="供应商"
                value={form.supplier}
                onChange={(e) => setField({ supplier: e.target.value })}
                placeholder="报价方名称"
              />
            </Field>
            <Field label="地区">
              <input
                className={fieldCls}
                aria-label="地区"
                value={form.region}
                onChange={(e) => setField({ region: e.target.value })}
                placeholder="如：成都市区"
              />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <Field label="期数 / 价格时间">
              <input
                className={fieldCls}
                aria-label="期数"
                value={form.priceDate}
                onChange={(e) => setField({ priceDate: e.target.value })}
                placeholder="如：2026-08"
              />
            </Field>
            <Field label="有效期至（可空）">
              <input
                className={fieldCls}
                aria-label="有效期至"
                value={form.validUntil}
                onChange={(e) => setField({ validUntil: e.target.value })}
                placeholder="YYYY-MM-DD，留空=长期"
              />
            </Field>
          </div>
          <Field label="备注">
            <input
              className={fieldCls}
              aria-label="备注"
              value={form.note}
              onChange={(e) => setField({ note: e.target.value })}
              placeholder="口径说明（含税/运杂/到场等）"
            />
          </Field>
        </div>
      </Modal>
    </div>
  );
}

// ── 来源徽标（四源不同颜色）─────────────────────────────────────
function SourceBadge({ source }: { source: string }) {
  const cls = SOURCE_STYLES[source] ?? "bg-bg-elev text-fg-faint";
  return <span className={`shrink-0 px-1.5 py-0.5 rounded text-[10px] ${cls}`}>{source || "—"}</span>;
}

// ── 表单字段（label 包 input，测试可用 getByLabelText）────────────
function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block min-w-0">
      <span className="block text-[10.5px] text-fg-faint mb-1">{label}</span>
      {children}
    </label>
  );
}
