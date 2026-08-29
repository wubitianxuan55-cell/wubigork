import { useEffect, useState } from "react";
import { Modal } from "antd";
import { Coins, FileText, Layers, Loader, TrendingUp, Wand2, X, Zap } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostComponent, CostComposeEvidence, CostComposeView, PriceBand } from "../../lib/types";
import { useToast } from "../Toast";

const fmtPrice = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const fmtPct = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 });

// 与 CostProjectsView / CostEntryModal 一致的内联样式（全部 Tailwind token 类，无 raw hex）。
const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";
const badgeCls = "px-1.5 py-px rounded text-[9.5px]";
const cellInputCls =
  "bg-bg border border-border-soft rounded-md text-fg text-[11px] px-1.5 py-1 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";

// 人材机类别（与 CostEntryModal 的 kind 契约一致）。
const KIND_OPTIONS = ["人工", "材料", "机械", "人工+机械"];

// 置信度徽标配色（高/中/低，其余降级为灰）。
const confidenceClass = (c: string) => {
  if (c === "高") return "text-emerald-400 bg-emerald-400/10";
  if (c === "中") return "text-amber-400 bg-amber-400/10";
  if (c === "低") return "text-red-400 bg-red-400/10";
  return "text-fg-faint bg-bg-elev";
};

// 离群判定：样本价 < P25-1.5IQR 或 > P75+1.5IQR（IQR = P75-P25），与后端 band.outliers 同口径。
const isOutlierPrice = (band: PriceBand, price: number) => {
  const iqr = band.p75 - band.p25;
  if (iqr <= 0) return false;
  return price < band.p25 - 1.5 * iqr || price > band.p75 + 1.5 * iqr;
};

/**
 * ComposeModal AI 组价弹窗（v4.2c）：输入清单描述/单位 → GaeaCostCompose →
 * 价格带推荐（P25/P50/P75/均值/离散度/离群数 + 推荐价与理由）+ 证据链相似条目表
 * + 人材机拆解（可增删改行，金额=含量×单价自动算）→ 「应用」回调父级（应用为
 * 明细行或沉淀成本库）。band=null 展示空态，失败持久展示错误可修改重试。
 * 组件行编辑不影响推荐价（推荐价来自价格带，组件是拆解明细）。
 */
export function ComposeModal({
  open,
  initialDesc = "",
  initialUnit = "",
  onClose,
  onApply,
}: {
  open: boolean;
  /** 预填清单描述（如来自明细行标题）。 */
  initialDesc?: string;
  /** 预填单位（如 ㎡/m³/台班，可空）。 */
  initialUnit?: string;
  onClose: () => void;
  /** 应用组价结果：desc/unit/推荐价/编辑后的 components/evidence，由父级决定落库方式。 */
  onApply: (r: {
    desc: string;
    unit: string;
    price: number;
    components: CostComponent[];
    evidence: CostComposeEvidence[];
  }) => void;
}) {
  const toast = useToast();
  const [desc, setDesc] = useState(initialDesc);
  const [unit, setUnit] = useState(initialUnit);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<CostComposeView | null>(null);
  const [components, setComponents] = useState<CostComponent[]>([]);

  // 打开/换预填时重置为初始输入，清空上一次结果与错误。
  useEffect(() => {
    if (!open) return;
    setDesc(initialDesc);
    setUnit(initialUnit);
    setView(null);
    setComponents([]);
    setError(null);
    setLoading(false);
  }, [open, initialDesc, initialUnit]);

  const handleCompose = async () => {
    if (!desc.trim() || loading) return;
    setLoading(true);
    setError(null);
    setView(null);
    try {
      const v = await app.CostCompose(desc.trim(), unit.trim());
      setView(v);
      setComponents(v.components ?? []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      toast.show(`组价失败:${msg}`, "warn");
    } finally {
      setLoading(false);
    }
  };

  const patchComponent = (i: number, patch: Partial<CostComponent>) => {
    setComponents((prev) =>
      prev.map((c, idx) => {
        if (idx !== i) return c;
        const next = { ...c, ...patch };
        // 金额=含量×单价 自动算。
        if (patch.quantity !== undefined || patch.price !== undefined) {
          next.amount = (next.quantity ?? 0) * (next.price ?? 0);
        }
        return next;
      }),
    );
  };

  const addComponent = () => {
    setComponents((prev) => [...prev, { kind: "人工", title: "", unit: "", quantity: 0, price: 0, amount: 0 }]);
  };

  const removeComponent = (i: number) => {
    setComponents((prev) => prev.filter((_, idx) => idx !== i));
  };

  const handleApply = () => {
    if (!view || !view.band) return;
    onApply({
      desc: desc.trim(),
      unit: unit.trim(),
      price: view.recommendedPrice,
      components,
      evidence: view.evidence,
    });
    onClose();
  };

  const band = view?.band ?? null;

  return (
    <Modal
      title={
        <span className="flex items-center gap-2">
          <Wand2 size={14} className="text-accent" />
          AI 组价
        </span>
      }
      open={open}
      onCancel={onClose}
      width={880}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      footer={
        <div className="flex justify-end gap-2">
          <button type="button" className={ghostBtn} onClick={onClose}>
            取消
          </button>
          {view && band && (
            <button type="button" className={solidBtn} onClick={handleApply}>
              应用
            </button>
          )}
        </div>
      }
    >
      {loading ? (
        <div className="flex items-center justify-center gap-2 py-14 text-fg-faint text-[12px]" role="status">
          <Loader size={14} className="animate-spin text-accent" />
          正在组价…
        </div>
      ) : error ? (
        <div className="space-y-3">
          <div className="px-3 py-2.5 rounded-lg border border-err/40 bg-err/10 text-[11.5px]" role="alert">
            <div className="text-err font-medium">组价失败：{error}</div>
            <div className="mt-0.5 text-fg-faint">可修改描述后重新组价，或先检查成本库数据。</div>
          </div>
          <ComposeForm
            desc={desc}
            unit={unit}
            disabled={loading}
            onDescChange={setDesc}
            onUnitChange={setUnit}
            onCompose={handleCompose}
          />
        </div>
      ) : !view ? (
        <ComposeForm
          desc={desc}
          unit={unit}
          disabled={loading}
          onDescChange={setDesc}
          onUnitChange={setUnit}
          onCompose={handleCompose}
        />
      ) : !band ? (
        <div className="py-12 text-center">
          <Coins size={24} className="mx-auto text-fg-faint" />
          <div className="mt-2 text-[12px] text-fg-faint leading-relaxed">成本库暂无相似条目,请先录入或导入成本数据</div>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-baseline gap-1.5 text-[11.5px] text-fg-dim">
            <span className="text-fg-faint shrink-0">组价对象</span>
            <span className="min-w-0 truncate font-medium text-fg">{view.description || desc}</span>
            {(view.unit || unit) && <span className="shrink-0 text-fg-faint">· {view.unit || unit}</span>}
          </div>

          {/* 价格带卡片 */}
          <div className="rounded-xl border border-border-soft bg-bg-soft/40 p-3">
            <div className="mb-2 flex items-center justify-between gap-2">
              <span className="flex items-center gap-1.5 text-[11.5px] font-semibold text-fg">
                <TrendingUp size={12} className="text-emerald-400" /> 价格带推荐
              </span>
              <span className="flex items-center gap-1.5">
                <span className={`${badgeCls} text-fg-dim bg-bg-elev`}>{band.samples} 个样本</span>
                <span className={`${badgeCls} ${confidenceClass(band.confidence)}`}>{band.confidence || "—"}</span>
              </span>
            </div>
            <div className="grid grid-cols-3 gap-1.5 text-[11px]">
              <BandStat label="P25" value={`¥${fmtPrice.format(band.p25)}`} />
              <BandStat label="中位数 P50" value={`¥${fmtPrice.format(band.median)}`} />
              <BandStat label="P75" value={`¥${fmtPrice.format(band.p75)}`} />
              <BandStat label="均值" value={`¥${fmtPrice.format(band.mean)}`} />
              <BandStat label="离散度" value={`${fmtPct.format(band.spreadPct)}%`} />
              <BandStat label="离群数" value={`${band.outliers} 个`} />
            </div>
            <div className="mt-2.5 flex items-baseline gap-2">
              <span className="text-[10.5px] text-fg-faint">推荐价</span>
              <span className="text-[22px] leading-none font-bold text-amber-300 tabular-nums">
                ¥{fmtPrice.format(view.recommendedPrice)}
              </span>
            </div>
            <div className="mt-1.5 text-[10.5px] text-fg-dim leading-relaxed">{view.reason || "—"}</div>
          </div>

          {/* 证据链 */}
          <div>
            <div className="mb-1.5 flex items-center gap-1.5 text-[11.5px] font-semibold text-fg">
              <FileText size={12} className="text-sky-400" /> 证据链（{band.sources.length} 条）
            </div>
            <div className="max-h-[36vh] overflow-auto rounded-lg border border-border-soft">
              <table className="w-full text-[11px]">
                <thead className="sticky top-0 bg-bg-elev text-fg-faint text-left">
                  <tr>
                    <th className="px-2 py-1.5 min-w-[150px]">标题</th>
                    <th className="px-2 py-1.5 min-w-[110px]">规格</th>
                    <th className="px-2 py-1.5 w-24 text-right">单价(元)</th>
                    <th className="px-2 py-1.5 w-14">单位</th>
                    <th className="px-2 py-1.5 min-w-[110px]">来源</th>
                    <th className="px-2 py-1.5 w-20">地区</th>
                    <th className="px-2 py-1.5 w-24">期数</th>
                    <th className="px-2 py-1.5 w-24">口径</th>
                  </tr>
                </thead>
                <tbody>
                  {band.sources.map((s, i) => {
                    const outlier = isOutlierPrice(band, s.price);
                    return (
                      <tr key={i} className={`border-t border-border-soft/60 ${outlier ? "bg-red-500/5" : ""}`}>
                        <td className="px-2 py-1.5 text-fg">
                          {s.title || "—"}
                          {outlier && (
                            <span className="ml-1.5 px-1 py-px rounded bg-red-500/10 text-red-400 text-[9px]">离群</span>
                          )}
                        </td>
                        <td className="px-2 py-1.5 text-fg-dim">{s.spec || "—"}</td>
                        <td className="px-2 py-1.5 text-right text-amber-300 font-semibold tabular-nums whitespace-nowrap">
                          ¥{fmtPrice.format(s.price)}
                        </td>
                        <td className="px-2 py-1.5 text-fg-dim">{s.unit || "—"}</td>
                        <td className="px-2 py-1.5 text-fg-dim">{s.source || "—"}</td>
                        <td className="px-2 py-1.5 text-fg-faint">{s.region || "—"}</td>
                        <td className="px-2 py-1.5 text-fg-faint">{s.priceDate || "—"}</td>
                        <td className="px-2 py-1.5 text-fg-faint">{s.priceType || "—"}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>

          {/* 人材机拆解 */}
          {view.components && (
            <div>
              <div className="mb-1.5 flex items-center justify-between gap-2">
                <span className="flex items-center gap-1.5 text-[11.5px] font-semibold text-fg">
                  <Layers size={12} className="text-violet-400" /> 人材机拆解（{components.length} 行）
                </span>
                {!view.llmUsed && <span className={`${badgeCls} text-amber-400 bg-amber-400/10`}>规则降级</span>}
              </div>
              <div className="space-y-1.5">
                {components.map((c, i) => (
                  <div key={i} className="flex items-center gap-1.5 rounded-lg border border-border/70 bg-bg-soft/40 p-1.5">
                    <select
                      className="w-24 shrink-0 bg-bg border border-border-soft rounded-md text-fg text-[11px] px-1.5 py-1 outline-none focus:border-accent"
                      value={c.kind}
                      onChange={(e) => patchComponent(i, { kind: e.target.value })}
                    >
                      {KIND_OPTIONS.map((k) => (
                        <option key={k} value={k}>
                          {k}
                        </option>
                      ))}
                    </select>
                    <input
                      className={`${cellInputCls} flex-1 min-w-0`}
                      value={c.title ?? ""}
                      placeholder="名称"
                      onChange={(e) => patchComponent(i, { title: e.target.value })}
                    />
                    <input
                      className={`${cellInputCls} w-16 shrink-0`}
                      value={c.unit ?? ""}
                      placeholder="单位"
                      onChange={(e) => patchComponent(i, { unit: e.target.value })}
                    />
                    <input
                      className={`${cellInputCls} w-16 shrink-0 text-right tabular-nums`}
                      type="number"
                      min={0}
                      value={c.quantity ?? ""}
                      placeholder="含量"
                      onChange={(e) => patchComponent(i, { quantity: e.target.value === "" ? 0 : Number(e.target.value) })}
                    />
                    <input
                      className={`${cellInputCls} w-20 shrink-0 text-right tabular-nums`}
                      type="number"
                      min={0}
                      value={c.price ?? ""}
                      placeholder="单价"
                      onChange={(e) => patchComponent(i, { price: e.target.value === "" ? 0 : Number(e.target.value) })}
                    />
                    <span className="w-20 shrink-0 text-right tabular-nums text-fg font-medium">
                      ¥{fmtPrice.format((c.quantity ?? 0) * (c.price ?? 0))}
                    </span>
                    <button
                      type="button"
                      className={iconBtn}
                      title={`删除第 ${i + 1} 行`}
                      onClick={() => removeComponent(i)}
                    >
                      <X size={11} />
                    </button>
                  </div>
                ))}
                <button
                  type="button"
                  className="w-full h-7 rounded-lg border border-dashed border-border text-fg-faint hover:text-accent hover:border-accent/50 transition-colors text-[11.5px]"
                  onClick={addComponent}
                >
                  ＋ 添加组成行
                </button>
              </div>
              {view.componentsNote && <div className="mt-1.5 text-[10.5px] text-fg-faint">备注：{view.componentsNote}</div>}
              {!view.llmUsed && (
                <div className="mt-1.5 flex items-center gap-1.5 rounded-md border border-amber-400/30 bg-amber-400/10 px-2 py-1 text-[10.5px] text-amber-400">
                  <Zap size={11} /> 规则降级：LLM 拆解不可用，以下为规则兜底结果，建议人工复核。
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </Modal>
  );
}

// ── 输入表单（描述 + 单位 + 开始组价）─────────────────────────
function ComposeForm({
  desc,
  unit,
  disabled,
  onDescChange,
  onUnitChange,
  onCompose,
}: {
  desc: string;
  unit: string;
  disabled?: boolean;
  onDescChange: (v: string) => void;
  onUnitChange: (v: string) => void;
  onCompose: () => void;
}) {
  return (
    <div className="space-y-2.5">
      <div>
        <span className="block text-[10.5px] text-fg-faint mb-1">清单描述</span>
        <textarea
          className={`${fieldCls} resize-none`}
          rows={3}
          value={desc}
          placeholder="清单描述，如：C30 泵送商品混凝土浇筑（含运输）"
          onChange={(e) => onDescChange(e.target.value)}
        />
      </div>
      <div>
        <span className="block text-[10.5px] text-fg-faint mb-1">单位</span>
        <input
          className={fieldCls}
          value={unit}
          placeholder="如 ㎡ / m³ / 台班，可空"
          onChange={(e) => onUnitChange(e.target.value)}
        />
      </div>
      <button
        type="button"
        className={`${solidBtn} w-full justify-center`}
        disabled={disabled || !desc.trim()}
        onClick={onCompose}
      >
        <Wand2 size={12} /> 开始组价
      </button>
    </div>
  );
}

// ── 价格带统计格 ─────────────────────────────────────────────
function BandStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-bg-elev/60 px-2 py-1">
      <div className="text-[9.5px] text-fg-faint">{label}</div>
      <div className="tabular-nums text-fg">{value}</div>
    </div>
  );
}
