// WorkcostQuotaModal.tsx — 消耗定额编辑弹窗（新建/编辑 + 工料机含量行）。
//
// 定额是「每单位子目消耗多少工料机」的全局标准（gf_quotas/gf_quota_items）。
// 编辑落库走 WorkcostQuotaSave（整组替换含量行）；金额刻意不入列——综合单价
// 由 WorkcostQuotaCompose 按当前资源价现算，改含量/改资源价都自动跟新。
//
// 含量行的资源引用两种来路：手填（编码/类别/名称/单位）或从资源库搜索回填
// （推荐——resourceCode 对上资源库锚点，资源价变动才能传导进来）。
import { useEffect, useRef, useState } from "react";
import { Plus, Search, Trash2, X } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostKind, WorkcostQuota, WorkcostQuotaItem, WorkcostResource } from "../../lib/types";

const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

const KINDS: WorkcostKind[] = ["人工", "材料", "机械", "外委"];

// emptyItem 一条空白含量行（类别默认材料——最常见；行类可改）。
function emptyItem(quotaCode: string): WorkcostQuotaItem {
  return { quotaCode, resourceCode: "", kind: "材料", title: "", unit: "", quantity: 0, lossRate: 0 };
}

/**
 * QuotaModal — 新建/编辑消耗定额。
 *
 * value.id===0 视为新建（编码必填，落库由后端校验唯一）；
 * 保存成功后父组件刷新列表并保持选中。
 */
export function QuotaModal({
  value,
  onCancel,
  onSaved,
}: {
  value: WorkcostQuota;
  onCancel: () => void;
  onSaved: (saved: WorkcostQuota) => void;
}) {
  const [draft, setDraft] = useState<WorkcostQuota>({ ...value, items: (value.items ?? []).map((it) => ({ ...it })) });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [pickerFor, setPickerFor] = useState<number | null>(null); // 正在选资源的行下标

  const set = (patch: Partial<WorkcostQuota>) => setDraft((d) => ({ ...d, ...patch }));
  const setItem = (idx: number, patch: Partial<WorkcostQuotaItem>) =>
    setDraft((d) => {
      const items = (d.items ?? []).map((it, i) => (i === idx ? { ...it, ...patch } : it));
      return { ...d, items };
    });
  const addItem = () =>
    setDraft((d) => ({ ...d, items: [...(d.items ?? []), emptyItem(d.code)] }));
  const removeItem = (idx: number) =>
    setDraft((d) => ({ ...d, items: (d.items ?? []).filter((_, i) => i !== idx) }));

  // 编码改动时同步含量行的 quotaCode（行级引用与父编码一致）。
  useEffect(() => {
    setDraft((d) => ({
      ...d,
      items: (d.items ?? []).map((it) => ({ ...it, quotaCode: d.code })),
    }));
  }, [draft.code]);

  const save = async () => {
    setBusy(true);
    setError("");
    try {
      const saved = await app.WorkcostQuotaSave(draft);
      onSaved(saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const items = draft.items ?? [];
  const hasEmptyTitle = items.some((it) => it.title.trim() === "");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <div
        className="v3-panel rounded-2xl p-5 space-y-3 w-[52rem] max-h-[86vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-fg font-semibold flex-1 min-w-0 truncate">
            {value.id ? `编辑消耗定额：${value.title}` : "新建消耗定额"}
          </span>
          <button type="button" className={iconBtn} onClick={onCancel} title="关闭">
            <X size={11} />
          </button>
        </div>

        {/* 定额头 */}
        <div className="grid grid-cols-5 gap-2.5">
          <label className="block">
            <span className="block text-[10.5px] text-fg-faint mb-1">编码 *</span>
            <input className={fieldCls} value={draft.code} placeholder="如 WP03" aria-label="定额编码" onChange={(e) => set({ code: e.target.value })} disabled={!!value.id} />
          </label>
          <label className="block col-span-2">
            <span className="block text-[10.5px] text-fg-faint mb-1">名称 *</span>
            <input className={fieldCls} value={draft.title} placeholder="如：场地平整" aria-label="定额名称" onChange={(e) => set({ title: e.target.value })} autoFocus={!value.id} />
          </label>
          <label className="block">
            <span className="block text-[10.5px] text-fg-faint mb-1">专业</span>
            <input className={fieldCls} value={draft.specialty} placeholder="土壤修复" aria-label="专业" onChange={(e) => set({ specialty: e.target.value })} />
          </label>
          <label className="block">
            <span className="block text-[10.5px] text-fg-faint mb-1">计量单位</span>
            <input className={fieldCls} value={draft.unit} placeholder="m³ / t / 项" aria-label="计量单位" onChange={(e) => set({ unit: e.target.value })} />
          </label>
        </div>
        {value.id && (
          <p className="text-[10.5px] text-fg-faint">编码是定额的稳定锚点（改址不改引用），编辑态不可改。</p>
        )}

        {/* 含量行 */}
        <div className="flex items-center justify-between">
          <span className="text-[12px] text-fg font-semibold">工料机含量行（每{draft.unit || "单位"}消耗量）</span>
          <button type="button" className={ghostBtn} onClick={addItem}>
            <Plus size={11} /> 添加含量行
          </button>
        </div>
        {items.length === 0 ? (
          <div className="v3-panel rounded-lg py-6 text-center text-[11.5px] text-fg-faint">
            还没有含量行——没有消耗量的定额综合单价为 0。
          </div>
        ) : (
          <div className="v3-panel rounded-lg overflow-x-auto">
            <table className="w-full text-[11.5px]">
              <thead>
                <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50 bg-bg-soft/30">
                  <th className="py-1.5 px-2 w-20">类别</th>
                  <th className="py-1.5 px-2 w-24">资源编码</th>
                  <th className="py-1.5 px-2">名称</th>
                  <th className="py-1.5 px-2 w-14">单位</th>
                  <th className="py-1.5 px-2 w-24 text-right">含量</th>
                  <th className="py-1.5 px-2 w-20 text-right">损耗 %</th>
                  <th className="py-1.5 px-2 w-24">备注</th>
                  <th className="py-1.5 px-2 w-16" />
                </tr>
              </thead>
              <tbody>
                {items.map((it, i) => (
                  <tr key={i} className="border-b border-border-soft/20 last:border-0">
                    <td className="py-1 px-2">
                      <select
                        className="bg-transparent border-0 outline-none text-fg text-[11.5px]"
                        value={it.kind}
                        aria-label="行类别"
                        onChange={(e) => setItem(i, { kind: e.target.value as WorkcostKind })}
                      >
                        {KINDS.map((k) => (
                          <option key={k} value={k}>{k}</option>
                        ))}
                      </select>
                    </td>
                    <td className="py-1 px-2">
                      <div className="flex items-center gap-1">
                        <input
                          className="w-16 bg-transparent border-0 outline-none font-mono text-[10.5px] text-fg-faint focus:border-b focus:border-accent"
                          value={it.resourceCode}
                          aria-label="资源编码"
                          onChange={(e) => setItem(i, { resourceCode: e.target.value })}
                        />
                        <button
                          type="button"
                          className="text-fg-faint hover:text-accent transition-colors"
                          title="从资源库搜索回填（编码对上锚点，资源调价才能传导）"
                          onClick={() => setPickerFor(i)}
                        >
                          <Search size={11} />
                        </button>
                      </div>
                    </td>
                    <td className="py-1 px-2">
                      <input
                        className="w-full bg-transparent border-0 outline-none text-fg focus:border-b focus:border-accent"
                        value={it.title}
                        aria-label="资源名称"
                        onChange={(e) => setItem(i, { title: e.target.value })}
                      />
                    </td>
                    <td className="py-1 px-2">
                      <input
                        className="w-12 bg-transparent border-0 outline-none text-fg-faint focus:border-b focus:border-accent"
                        value={it.unit}
                        aria-label="单位"
                        onChange={(e) => setItem(i, { unit: e.target.value })}
                      />
                    </td>
                    <td className="py-1 px-2 text-right">
                      <input
                        className="w-20 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent"
                        type="number"
                        min={0}
                        value={it.quantity || ""}
                        aria-label="含量"
                        title="0 = 合法占位（工序模板列了但本项未用）"
                        onChange={(e) => setItem(i, { quantity: Number(e.target.value) })}
                      />
                    </td>
                    <td className="py-1 px-2 text-right">
                      <input
                        className="w-16 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent"
                        type="number"
                        min={0}
                        value={it.lossRate ? Math.round(it.lossRate * 1000) / 10 : ""}
                        aria-label="损耗率百分比"
                        onChange={(e) => setItem(i, { lossRate: Number(e.target.value) / 100 })}
                      />
                    </td>
                    <td className="py-1 px-2">
                      <input
                        className="w-full bg-transparent border-0 outline-none text-fg-faint text-[10.5px] focus:border-b focus:border-accent"
                        value={it.note ?? ""}
                        aria-label="备注"
                        onChange={(e) => setItem(i, { note: e.target.value })}
                      />
                    </td>
                    <td className="py-1 px-2">
                      <button type="button" className={`${iconBtn} hover:text-err`} title="删除本行" onClick={() => removeItem(i)}>
                        <Trash2 size={11} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="text-[10.5px] text-fg-faint leading-relaxed">
          含量 0 是合法占位（「技术工/带班 ×0」是实测模版真实存在的行）；负值会被核算拒绝。
          金额不入库——综合单价 = Σ(含量 × 资源现行价)，保存后在右侧分析表现算。
        </p>

        {error && (
          <div role="alert" className="text-[11px] text-amber-300 border border-amber-400/30 bg-amber-400/10 rounded-lg px-2.5 py-2">
            {error}
          </div>
        )}

        <div className="flex items-center justify-end gap-2 pt-1">
          <button type="button" className={ghostBtn} onClick={onCancel} disabled={busy}>
            取消
          </button>
          <button
            type="button"
            className={solidBtn}
            onClick={() => void save()}
            disabled={busy || draft.code.trim() === "" || draft.title.trim() === "" || hasEmptyTitle}
            title={hasEmptyTitle ? "有含量行未填名称" : undefined}
          >
            {busy ? "保存中…" : "保存定额"}
          </button>
        </div>
      </div>

      {pickerFor !== null && (
        <ResourcePickerModal
          onCancel={() => setPickerFor(null)}
          onPick={(r) => {
            setItem(pickerFor, { resourceCode: r.code, kind: r.kind, title: r.title, unit: r.unit });
            setPickerFor(null);
          }}
        />
      )}
    </div>
  );
}

// ResourcePickerModal 资源搜索回填（含量行引用资源库的推荐来路）。
function ResourcePickerModal({
  onCancel,
  onPick,
}: {
  onCancel: () => void;
  onPick: (r: WorkcostResource) => void;
}) {
  const [keyword, setKeyword] = useState("");
  const [rows, setRows] = useState<WorkcostResource[]>([]);
  const [loading, setLoading] = useState(true);
  const reqSeq = useRef(0);

  useEffect(() => {
    const seq = ++reqSeq.current;
    setLoading(true);
    app
      .WorkcostResourceList("", keyword)
      .then((r) => {
        if (seq === reqSeq.current) setRows(r ?? []);
      })
      .catch(() => {
        if (seq === reqSeq.current) setRows([]);
      })
      .finally(() => {
        if (seq === reqSeq.current) setLoading(false);
      });
  }, [keyword]);

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <div className="v3-panel rounded-2xl p-4 space-y-2.5 w-[32rem] max-h-[70vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2">
          <span className="text-[12.5px] text-fg font-semibold flex-1">从资源库选（回填编码/类别/名称/单位）</span>
          <button type="button" className={iconBtn} onClick={onCancel} title="关闭">
            <X size={11} />
          </button>
        </div>
        <div className="relative">
          <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input className={`${fieldCls} !pl-6`} value={keyword} placeholder="搜索名称/规格/编码" autoFocus onChange={(e) => setKeyword(e.target.value)} />
        </div>
        {loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-lg h-8" />
            <div className="v3-panel rounded-lg h-8" />
          </div>
        ) : rows.length === 0 ? (
          <div className="py-6 text-center text-[11px] text-fg-faint">没有匹配资源。</div>
        ) : (
          <ul className="v3-panel rounded-lg divide-y divide-border-soft/25 max-h-72 overflow-y-auto">
            {rows.slice(0, 50).map((r) => (
              <li key={r.id}>
                <button type="button" className="w-full text-left px-3 py-1.5 hover:bg-bg-elev/40 transition-colors" onClick={() => onPick(r)}>
                  <span className="flex items-center gap-2">
                    <span className="text-[10px] px-1 py-px rounded bg-bg-soft text-fg-faint shrink-0">{r.kind}</span>
                    <span className="truncate text-[11.5px] text-fg flex-1">{r.title}</span>
                    <span className="font-mono text-[10px] text-fg-faint shrink-0">{r.code}</span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
