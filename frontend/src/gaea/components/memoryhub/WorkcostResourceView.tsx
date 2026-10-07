// WorkcostResourceView.tsx — 工料机资源库（工料法第①层）。
//
// 用户定调：工料机（人材机）是**基本数据**，综合单价分析表是靠「工料机 × 消耗
// 定额」核算出来的。本视图是资源主数据的唯一维护面：新增/编辑、调价历史（信息价
// 推进现行价）、停用、删除（被定额引用时拒绝），以及存量成本条目的一键资源化。
//
// 关键语义（与 Go 侧 workcost 包一致）：
//   核算取用价 = 现行价优先，为 0 回退基准价 —— 所以「现行价空着」不等于免费，
//   界面上必须把两者都显示出来，不能让用户误判。
import { useCallback, useEffect, useMemo, useState } from "react";
import { Box, Coins, Layers, Pencil, Plus, RefreshCw, Search, Trash2, TrendingUp, X } from "../../icons";
import { app } from "../../lib/bridge";
import type {
  WorkcostResource,
  WorkcostResourcePrice,
  WorkcostSeedPreview,
  WorkcostKind,
} from "../../lib/types";

const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);
const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

const KINDS: WorkcostKind[] = ["人工", "材料", "机械", "外委"];

// kindTone 四类配色（人工/材料/机械/外委），仅用于标签底色。
function kindTone(kind: string): string {
  switch (kind) {
    case "人工":
      return "text-sky-400 bg-sky-400/10";
    case "机械":
      return "text-violet-400 bg-violet-400/10";
    case "外委":
      return "text-amber-400 bg-amber-400/10";
    default:
      return "text-ok bg-ok/10";
  }
}

// emptyResource 新建资源的初值（现行价与基准价都留 0，由用户填）。
function emptyResource(): WorkcostResource {
  return {
    id: 0, code: "", kind: "材料", title: "", spec: "", unit: "",
    basePrice: 0, currentPrice: 0, categoryPath: "", source: "", supplier: "",
    region: "", priceDate: "", priceType: "", validUntil: "", lossRate: 0,
    note: "", tags: [], status: "现行",
  };
}

/**
 * WorkcostResourceView — 工料机资源库管理面。
 *
 * 布局：工具条（类别筛选 + 搜索 + 资源化 + 新增 + 刷新）→ 资源表 → 编辑/调价弹窗。
 */
export function WorkcostResourceView() {
  const [kind, setKind] = useState<string>("全部");
  const [keyword, setKeyword] = useState("");
  const [rows, setRows] = useState<WorkcostResource[]>([]);
  const [loading, setLoading] = useState(true);
  const [readError, setReadError] = useState("");
  const [editing, setEditing] = useState<WorkcostResource | null>(null);
  const [pricing, setPricing] = useState<WorkcostResource | null>(null);
  const [preview, setPreview] = useState<WorkcostSeedPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setReadError("");
    try {
      const r = (await app.WorkcostResourceList(kind === "全部" ? "" : kind, keyword)) ?? [];
      setRows(r);
    } catch (e) {
      // 读失败必须显形为「读取失败」，不得装成「没有资源」——两者对用户
      // 意味着完全不同的处置（重试 vs 去新建）。
      setRows([]);
      setReadError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [kind, keyword]);

  useEffect(() => {
    load();
  }, [load]);

  const counts = useMemo(() => {
    const c: Record<string, number> = { 人工: 0, 材料: 0, 机械: 0, 外委: 0 };
    for (const r of rows) c[r.kind] = (c[r.kind] ?? 0) + 1;
    return c;
  }, [rows]);

  const save = useCallback(
    async (r: WorkcostResource) => {
      setBusy(true);
      try {
        await app.WorkcostResourceSave(r);
        setEditing(null);
        await load();
        setToast(`已保存「${r.title}」`);
      } catch (e) {
        setToast(`保存失败：${e instanceof Error ? e.message : String(e)}`);
      } finally {
        setBusy(false);
      }
    },
    [load],
  );

  const remove = useCallback(
    async (r: WorkcostResource, force: boolean) => {
      try {
        await app.WorkcostResourceDelete(r.id, force);
        await load();
        setToast(`已删除「${r.title}」`);
      } catch (e) {
        // 被定额引用的资源会被后端拒绝——把原因如实告诉用户（可改归档）。
        setToast(e instanceof Error ? e.message : String(e));
      }
    },
    [load],
  );

  const setPrice = useCallback(
    async (r: WorkcostResource, price: number, period: string, region: string, note: string) => {
      setBusy(true);
      try {
        await app.WorkcostResourceSetPrice(r.id, price, period, region, "到场价", "手动调价", note);
        setPricing(null);
        await load();
        setToast(`「${r.title}」现行价已更新为 ${fmtPrice(price)}——引用它的综合单价将同步重算`);
      } catch (e) {
        setToast(`调价失败：${e instanceof Error ? e.message : String(e)}`);
      } finally {
        setBusy(false);
      }
    },
    [load],
  );

  const openSeedPreview = useCallback(async () => {
    setBusy(true);
    try {
      // 干跑预览：先让用户看清归类分布与跳过原因，再决定是否写库。
      setPreview(await app.WorkcostSeedPreview(false));
    } catch (e) {
      setToast(`预览失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, []);

  const applySeed = useCallback(async () => {
    setBusy(true);
    try {
      const res = await app.WorkcostSeedApply(false);
      setPreview(null);
      await load();
      setToast(
        `资源化完成：新建 ${res.created} / 更新 ${res.updated}` +
          (res.errors.length ? ` / 失败 ${res.errors.length}` : ""),
      );
    } catch (e) {
      setToast(`资源化失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [load]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 工具条 */}
      <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 h-auto min-h-12 py-2 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <Layers size={14} className="text-accent" /> 工料机资源库
        </span>
        <span className="text-[11px] text-fg-faint hidden lg:inline">
          工料法第①层：独立主数据 · 核算取用价 = 现行价优先、为 0 回退基准价
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <div className="flex items-center rounded-lg border border-border bg-bg p-0.5 text-[11px]">
            {["全部", ...KINDS].map((k) => (
              <button
                key={k}
                type="button"
                className={`px-2 h-6 rounded-md transition-colors ${
                  kind === k ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"
                }`}
                onClick={() => setKind(k)}
                title={k === "全部" ? "不筛选类别" : `只看「${k}」`}
              >
                {k}
              </button>
            ))}
          </div>
          <div className="relative">
            <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
            <input
              className={`${fieldCls} !pl-6 !w-44`}
              value={keyword}
              placeholder="搜索名称/规格/编码"
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <button type="button" className={ghostBtn} onClick={openSeedPreview} disabled={busy} title="把存量成本条目按类别归入资源库">
            <TrendingUp size={12} /> 存量资源化
          </button>
          <button type="button" className={solidBtn} onClick={() => setEditing(emptyResource())} title="新增工料机资源">
            <Plus size={12} /> 新增资源
          </button>
          <button type="button" className={ghostBtn} onClick={load} title="刷新">
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      {toast && (
        <div className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-border-soft/60 bg-accent/5 text-[11px] text-fg-dim">
          <span className="flex-1 min-w-0">{toast}</span>
          <button type="button" className={iconBtn} onClick={() => setToast("")} title="关闭提示">
            <X size={10} />
          </button>
        </div>
      )}

      {readError && (
        <div
          role="alert"
          data-testid="workcost-resource-read-error"
          className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-amber-400/30 bg-amber-400/10 text-amber-200 text-[11px]"
        >
          <span className="flex-1 min-w-0">资源库读取失败：{readError}——结果可能不完整。</span>
          <button type="button" className={ghostBtn} onClick={load}>
            重试
          </button>
        </div>
      )}

      {/* 资源表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-4">
        {loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-xl h-10" />
            <div className="v3-panel rounded-xl h-64" />
          </div>
        ) : rows.length === 0 ? (
          <div className="v3-panel rounded-2xl py-16 text-center">
            <Box size={26} className="mx-auto text-fg-faint" />
            <div className="mt-3 text-[13px] text-fg font-medium">
              {readError ? "资源库读取失败，未取到资源" : "还没有工料机资源"}
            </div>
            <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">
              {readError
                ? "点上方「重试」重新读取；若持续失败请检查数据库可用性。"
                : "两种入库方式：①「存量资源化」把成本库里的纯资源价（材料/机械/人工）按类别一次归入；②「新增资源」手工录入，或导入项目工作簿的「工料机价格」表。"}
            </p>
          </div>
        ) : (
          <>
            <div className="mb-2 flex flex-wrap items-center gap-2 text-[11px] text-fg-faint">
              <span>共 {rows.length} 条</span>
              {KINDS.map((k) =>
                counts[k] ? (
                  <span key={k} className={`px-1.5 py-px rounded ${kindTone(k)}`}>
                    {k} {counts[k]}
                  </span>
                ) : null,
              )}
            </div>
            <div className="v3-panel rounded-xl overflow-x-auto">
              <table className="w-full text-[11.5px]">
                <thead>
                  <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50 bg-bg-soft/30">
                    <th className="py-2 px-3 w-20">编码</th>
                    <th className="py-2 px-2 w-14">类别</th>
                    <th className="py-2 px-2">名称</th>
                    <th className="py-2 px-2">规格</th>
                    <th className="py-2 px-2 w-14">单位</th>
                    <th className="py-2 px-2 w-24 text-right">现行价</th>
                    <th className="py-2 px-2 w-24 text-right">基准价</th>
                    <th className="py-2 px-2 w-16 text-right">损耗</th>
                    <th className="py-2 px-2 w-40">来源</th>
                    <th className="py-2 px-2 w-20 text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.id} className="border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30">
                      <td className="py-1.5 px-3 font-mono text-fg-faint">{r.code || "-"}</td>
                      <td className="py-1.5 px-2">
                        <span className={`px-1.5 py-px rounded text-[10px] ${kindTone(r.kind)}`}>{r.kind}</span>
                      </td>
                      <td className="py-1.5 px-2 text-fg font-medium">{r.title}</td>
                      <td className="py-1.5 px-2 text-fg-faint">{r.spec || "-"}</td>
                      <td className="py-1.5 px-2 text-fg-faint">{r.unit || "-"}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg font-semibold">
                        {fmtPrice(r.currentPrice > 0 ? r.currentPrice : r.basePrice)}
                        {r.currentPrice <= 0 && r.basePrice > 0 && (
                          <span className="ml-1 text-[9.5px] text-fg-faint" title="现行价未录入，核算回退基准价">
                            基准
                          </span>
                        )}
                      </td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg-faint">{fmtPrice(r.basePrice)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg-faint">
                        {r.lossRate > 0 ? `${(r.lossRate * 100).toFixed(1)}%` : "-"}
                      </td>
                      <td className="py-1.5 px-2 text-fg-faint truncate max-w-[10rem]" title={r.source}>
                        {r.source || "-"}
                      </td>
                      <td className="py-1.5 px-2">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            type="button"
                            className={iconBtn}
                            title="调价（写调价历史并推进现行价）"
                            onClick={() => setPricing(r)}
                          >
                            <Coins size={11} />
                          </button>
                          <button
                            type="button"
                            className={iconBtn}
                            title="编辑"
                            onClick={() => setEditing(r)}
                          >
                            <Pencil size={11} />
                          </button>
                          <button
                            type="button"
                            className={`${iconBtn} hover:text-err`}
                            title="删除（被定额引用时会拒绝）"
                            onClick={() => remove(r, false)}
                          >
                            <Trash2 size={11} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>

      {editing && (
        <ResourceModal
          value={editing}
          busy={busy}
          onCancel={() => setEditing(null)}
          onSave={save}
        />
      )}
      {pricing && (
        <PriceModal value={pricing} busy={busy} onCancel={() => setPricing(null)} onSave={setPrice} />
      )}
      {preview && (
        <SeedPreviewModal
          preview={preview}
          busy={busy}
          onCancel={() => setPreview(null)}
          onApply={applySeed}
        />
      )}
    </div>
  );
}

// ── 资源编辑弹窗 ────────────────────────────────────────────────────

function ResourceModal({
  value,
  busy,
  onCancel,
  onSave,
}: {
  value: WorkcostResource;
  busy: boolean;
  onCancel: () => void;
  onSave: (r: WorkcostResource) => void;
}) {
  const [draft, setDraft] = useState<WorkcostResource>(value);
  const set = (patch: Partial<WorkcostResource>) => setDraft((d) => ({ ...d, ...patch }));
  return (
    <Modal title={value.id ? "编辑工料机资源" : "新增工料机资源"} onCancel={onCancel}>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="名称" required>
          <input className={fieldCls} value={draft.title} placeholder="如：C20商品混凝土" onChange={(e) => set({ title: e.target.value })} autoFocus />
        </Field>
        <Field label="类别">
          <select className={fieldCls} value={draft.kind} onChange={(e) => set({ kind: e.target.value as WorkcostKind })}>
            {KINDS.map((k) => (
              <option key={k} value={k}>{k}</option>
            ))}
          </select>
        </Field>
        <Field label="规格">
          <input className={fieldCls} value={draft.spec} placeholder="如：泵送到场 / DN110" onChange={(e) => set({ spec: e.target.value })} />
        </Field>
        <Field label="单位">
          <input className={fieldCls} value={draft.unit} placeholder="工日 / t / m³ / 台班" onChange={(e) => set({ unit: e.target.value })} />
        </Field>
        <Field label="基准价（元）">
          <input className={fieldCls} type="number" min={0} value={draft.basePrice || ""} onChange={(e) => set({ basePrice: Number(e.target.value) })} />
        </Field>
        <Field label="现行价（元）">
          <input className={fieldCls} type="number" min={0} value={draft.currentPrice || ""} onChange={(e) => set({ currentPrice: Number(e.target.value) })} />
        </Field>
        <Field label="损耗率（%）">
          <input
            className={fieldCls}
            type="number"
            min={0}
            value={draft.lossRate ? (draft.lossRate * 100).toString() : ""}
            placeholder="如 3 表示 3%"
            onChange={(e) => set({ lossRate: Number(e.target.value) / 100 })}
          />
        </Field>
        <Field label="来源">
          <input className={fieldCls} value={draft.source} placeholder="信息价 / 市场询价 / 成本库" onChange={(e) => set({ source: e.target.value })} />
        </Field>
        <Field label="地区">
          <input className={fieldCls} value={draft.region} placeholder="如：乐山 / 德阳" onChange={(e) => set({ region: e.target.value })} />
        </Field>
        <Field label="价格期数">
          <input className={fieldCls} value={draft.priceDate} placeholder="如：2026年第7期" onChange={(e) => set({ priceDate: e.target.value })} />
        </Field>
      </div>
      <Field label="备注">
        <textarea className={fieldCls} rows={2} value={draft.note} placeholder="口径说明 / 计算式（如：折旧900+柴油×90L+司机）" onChange={(e) => set({ note: e.target.value })} />
      </Field>
      {draft.currentPrice <= 0 && draft.basePrice > 0 && (
        <p className="text-[11px] text-amber-300">
          现行价为空 → 核算将回退基准价 {fmtPrice(draft.basePrice)}。若两者应不同（信息价调整），请单独填现行价。
        </p>
      )}
      <ModalActions
        busy={busy}
        canSave={draft.title.trim() !== ""}
        onCancel={onCancel}
        onSave={() => onSave(draft)}
        saveLabel="保存资源"
      />
    </Modal>
  );
}

// ── 调价弹窗 ────────────────────────────────────────────────────────

function PriceModal({
  value,
  busy,
  onCancel,
  onSave,
}: {
  value: WorkcostResource;
  busy: boolean;
  onCancel: () => void;
  onSave: (r: WorkcostResource, price: number, period: string, region: string, note: string) => void;
}) {
  const [price, setPrice] = useState<number>(value.currentPrice > 0 ? value.currentPrice : value.basePrice);
  const [period, setPeriod] = useState("");
  const [region, setRegion] = useState(value.region);
  const [note, setNote] = useState("");
  return (
    <Modal title={`调价：${value.title}`} onCancel={onCancel}>
      <p className="text-[11.5px] text-fg-faint leading-relaxed">
        写入一条调价历史并推进**现行价**。引用该资源的消耗定额在下次核算时按新价重算
        ——不必逐条改综合单价。
      </p>
      <div className="grid grid-cols-2 gap-2.5">
        <Field label="新单价（元）" required>
          <input className={fieldCls} type="number" min={0} value={price || ""} onChange={(e) => setPrice(Number(e.target.value))} autoFocus />
        </Field>
        <Field label="期数">
          <input className={fieldCls} value={period} placeholder="如：2026年第7期" onChange={(e) => setPeriod(e.target.value)} />
        </Field>
        <Field label="地区">
          <input className={fieldCls} value={region} placeholder="如：乐山" onChange={(e) => setRegion(e.target.value)} />
        </Field>
        <Field label="备注">
          <input className={fieldCls} value={note} placeholder="调价依据" onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>
      <ModalActions
        busy={busy}
        canSave={price > 0}
        onCancel={onCancel}
        onSave={() => onSave(value, price, period, region, note)}
        saveLabel="确认调价"
      />
    </Modal>
  );
}

// ── 资源化预览弹窗 ──────────────────────────────────────────────────

function SeedPreviewModal({
  preview,
  busy,
  onCancel,
  onApply,
}: {
  preview: WorkcostSeedPreview;
  busy: boolean;
  onCancel: () => void;
  onApply: () => void;
}) {
  const skipEntries = Object.entries(preview.skipped ?? {});
  return (
    <Modal title="存量成本条目 → 工料机资源库（干跑预览）" onCancel={onCancel} wide>
      <p className="text-[11.5px] text-fg-faint leading-relaxed">
        把成本库里的**纯资源价**按类别归入资源库（材料/机械/人工/外委）。综合单价条目
        （分类含「综合单价」）默认保留在综合单价层，不降级为资源。同身份资源走更新，
        重复执行不会产生重复数据。
      </p>
      <div className="grid grid-cols-4 gap-2 text-center">
        <Stat label="人工" value={preview.counts.labor} tone="text-sky-400" />
        <Stat label="材料" value={preview.counts.material} tone="text-ok" />
        <Stat label="机械" value={preview.counts.machine} tone="text-violet-400" />
        <Stat label="外委" value={preview.counts.outsourced} tone="text-amber-400" />
      </div>
      <p className="text-[11.5px] text-fg-dim">
        共扫描 {preview.total} 条 → 可资源化 <span className="text-fg font-semibold">{preview.candidates.length}</span> 条
        {skipEntries.length > 0 && (
          <>
            ，跳过 {skipEntries.reduce((s, [, n]) => s + n, 0)} 条（
            {skipEntries.map(([reason, n]) => `${reason} ${n}`).join("；")}）
          </>
        )}
      </p>
      <div className="max-h-56 overflow-y-auto v3-panel rounded-lg">
        <table className="w-full text-[11px]">
          <thead>
            <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
              <th className="py-1.5 px-2 w-14">类别</th>
              <th className="py-1.5 px-2">名称</th>
              <th className="py-1.5 px-2 w-16">单位</th>
              <th className="py-1.5 px-2 w-24 text-right">价格</th>
              <th className="py-1.5 px-2 w-40">分类</th>
            </tr>
          </thead>
          <tbody>
            {preview.candidates.slice(0, 200).map((c, i) => (
              <tr key={`${c.title}-${i}`} className="border-b border-border-soft/20 last:border-0">
                <td className="py-1 px-2">
                  <span className={`px-1.5 py-px rounded text-[10px] ${kindTone(c.kind)}`}>{c.kind}</span>
                </td>
                <td className="py-1 px-2 text-fg-dim">{c.title}</td>
                <td className="py-1 px-2 text-fg-faint">{c.unit || "-"}</td>
                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice(c.price)}</td>
                <td className="py-1 px-2 text-fg-faint truncate max-w-[10rem]" title={c.categoryPath}>{c.categoryPath || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {preview.candidates.length > 200 && (
        <p className="text-[11px] text-fg-faint">仅预览前 200 条，入库会处理全部 {preview.candidates.length} 条。</p>
      )}
      <ModalActions
        busy={busy}
        canSave={preview.candidates.length > 0}
        onCancel={onCancel}
        onSave={onApply}
        saveLabel={`确认入库 ${preview.candidates.length} 条`}
      />
    </Modal>
  );
}

// ── 共用弹窗零件 ────────────────────────────────────────────────────

function Modal({
  title,
  children,
  onCancel,
  wide,
}: {
  title: string;
  children: React.ReactNode;
  onCancel: () => void;
  wide?: boolean;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <div
        className={`v3-panel rounded-2xl p-5 space-y-3 max-h-[86vh] overflow-y-auto ${wide ? "w-[46rem]" : "w-[34rem]"}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-fg font-semibold flex-1 min-w-0 truncate">{title}</span>
          <button type="button" className={iconBtn} onClick={onCancel} title="关闭">
            <X size={11} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

function ModalActions({
  busy,
  canSave,
  onCancel,
  onSave,
  saveLabel,
}: {
  busy: boolean;
  canSave: boolean;
  onCancel: () => void;
  onSave: () => void;
  saveLabel: string;
}) {
  return (
    <div className="flex items-center justify-end gap-2 pt-1">
      <button type="button" className={ghostBtn} onClick={onCancel} disabled={busy}>
        取消
      </button>
      <button type="button" className={solidBtn} onClick={onSave} disabled={busy || !canSave}>
        {busy ? "处理中…" : saveLabel}
      </button>
    </div>
  );
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
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

function Stat({ label, value, tone }: { label: string; value: number; tone: string }) {
  return (
    <div className="v3-panel rounded-lg py-2">
      <div className={`text-[16px] font-semibold tabular-nums ${tone}`}>{value}</div>
      <div className="text-[10.5px] text-fg-faint">{label}</div>
    </div>
  );
}

// 供测试导入（资源价历史视图用）。
export { PriceHistoryInline };

function PriceHistoryInline({ resourceId }: { resourceId: number }) {
  const [rows, setRows] = useState<WorkcostResourcePrice[]>([]);
  useEffect(() => {
    let alive = true;
    app
      .WorkcostResourcePrices(resourceId)
      .then((r) => {
        if (alive) setRows(r ?? []);
      })
      .catch(() => {
        if (alive) setRows([]);
      });
    return () => {
      alive = false;
    };
  }, [resourceId]);
  if (rows.length === 0) return <div className="text-[11px] text-fg-faint">暂无调价记录</div>;
  return (
    <ul className="text-[11px] space-y-0.5">
      {rows.map((p) => (
        <li key={p.id} className="flex items-center gap-2">
          <span className="tabular-nums text-fg">{fmtPrice(p.price)}</span>
          <span className="text-fg-faint">{p.period || "-"}</span>
          <span className="text-fg-faint">{p.region}</span>
          <span className="text-fg-faint ml-auto">{p.fetchedAt}</span>
        </li>
      ))}
    </ul>
  );
}
