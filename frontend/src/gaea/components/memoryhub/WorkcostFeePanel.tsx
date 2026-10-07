// WorkcostFeePanel.tsx — 项目层取费汇总（工料法第③层的项目合计面）。
//
// 口径（实测产物「费用汇总」表，与 Go 侧 ComposeProjectFees 一致）：
//   综合单价只含人材机 → 直接费 = Σ(工程量 × 综合单价)；
//   管理费/利润/规费/税金**不进综合单价**，只在项目合计层跑一次：
//     直接费 → 企管(直接费×率) → 利润((直接费+企管[+规费])×率) → 规费
//     → 税前合计(+措施+暂列) → 增值税 → 含税总造价；可对照招标控制价。
//   利润基数含不含规费是两份实测产物的真实差异（百锦路不含 / 市政道路含），
//   做成开关而非写死。
//
// 定位：**试算面板**——清单行与费率是草稿（localStorage 持久化），不落数据库；
// 定稿走「导出五表」（WorkcostComposeView，活公式工作簿）。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Calculator, List, Plus, RefreshCw, Search, Trash2, X } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostFeeResult, WorkcostQuota, WorkcostRateSet } from "../../lib/types";

const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const fmtPrice = (p: number) => "¥" + fmt.format(p);
const round2 = (v: number) => Math.round(v * 100) / 100;

const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-err hover:bg-bg-soft transition-colors";

// DRAFT_KEY 草稿持久化键。试算不落数据库（无确认不落库红线），但刷新不能丢
// ——localStorage 是浏览器态，存储禁用时静默降级为不持久化。
const DRAFT_KEY = "workcost-fee-draft";

// DEFAULT_RATES 与 Go 侧 DefaultRateSet 同值（市政道路模版实测：企管 10%、
// 规费 2%、利润 7%、增值税 9%）。仅作初始值，改后以草稿为准。
const DEFAULT_RATES: WorkcostRateSet = { managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09 };

interface FeeLine {
  key: string;
  quotaCode: string; // 空 = 手填行
  title: string;
  unit: string;
  quantity: number;
  unitPrice: number;
}

interface FeeDraft {
  lines: FeeLine[];
  rates: WorkcostRateSet;
  profitIncludesRegulatory: boolean;
  measures: number;
  contingency: number;
  controlPrice: number;
}

const DEFAULT_DRAFT: FeeDraft = {
  lines: [],
  rates: { ...DEFAULT_RATES },
  profitIncludesRegulatory: false,
  measures: 0,
  contingency: 0,
  controlPrice: 0,
};

let lineSeq = 0;
const nextKey = () => `fee-line-${Date.now()}-${lineSeq++}`;

function loadDraft(): FeeDraft {
  try {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) return { ...DEFAULT_DRAFT, rates: { ...DEFAULT_RATES } };
    const d = JSON.parse(raw) as FeeDraft;
    if (!Array.isArray(d.lines) || !d.rates) return { ...DEFAULT_DRAFT, rates: { ...DEFAULT_RATES } };
    return { ...DEFAULT_DRAFT, ...d };
  } catch {
    return { ...DEFAULT_DRAFT, rates: { ...DEFAULT_RATES } };
  }
}

/**
 * WorkcostFeePanel — 取费汇总（试算）。
 *
 * 左：分部分项清单（从定额库选行现算综合单价，或手填）→ 直接费；
 * 右：取费参数（四费率 + 利润基数策略 + 措施/暂列/控制价）→ 费用链。
 */
export function WorkcostFeePanel() {
  const [draft, setDraft] = useState<FeeDraft>(loadDraft);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [toast, setToast] = useState("");
  const [fee, setFee] = useState<WorkcostFeeResult | null>(null);
  const [feeError, setFeeError] = useState("");

  // 草稿持久化（存储禁用静默降级）。
  useEffect(() => {
    try {
      localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
    } catch {
      /* 存储不可用：仅本会话内保留 */
    }
  }, [draft]);

  // 直接费 = Σ(round2(工程量 × 综合单价))——每行先收敛到分，与 Go 侧 Round2
  // 导出口径一致，避免明细合计与费用链首行差一分。
  const directFee = useMemo(
    () => round2(draft.lines.reduce((s, l) => s + round2(l.quantity * l.unitPrice), 0)),
    [draft.lines],
  );

  // 取费链：参数一动即重算（后端是纯函数，无副作用，调用便宜）。alive 防竞态。
  useEffect(() => {
    let alive = true;
    setFeeError("");
    app
      .WorkcostProjectFees(
        directFee,
        draft.rates,
        draft.profitIncludesRegulatory,
        draft.measures,
        draft.contingency,
        draft.controlPrice,
      )
      .then((r) => {
        if (alive) setFee(r);
      })
      .catch((e) => {
        if (alive) {
          setFee(null);
          setFeeError(e instanceof Error ? e.message : String(e));
        }
      });
    return () => {
      alive = false;
    };
  }, [directFee, draft.rates, draft.profitIncludesRegulatory, draft.measures, draft.contingency, draft.controlPrice]);

  const setLines = useCallback((fn: (lines: FeeLine[]) => FeeLine[]) => {
    setDraft((d) => ({ ...d, lines: fn(d.lines) }));
  }, []);

  const patchLine = useCallback((key: string, patch: Partial<FeeLine>) => {
    setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)));
  }, [setLines]);

  // addQuotaLine 从定额库选行：加行后现算综合单价（核算输出不落库——资源价
  // 一变这里自动跟新；失败置 0 显形「缺价」，不阻塞试算）。
  const addQuotaLine = useCallback(
    (q: WorkcostQuota) => {
      const line: FeeLine = {
        key: nextKey(),
        quotaCode: q.code,
        title: q.title,
        unit: q.unit,
        quantity: 1,
        unitPrice: 0,
      };
      setLines((ls) => [...ls, line]);
      setPickerOpen(false);
      app
        .WorkcostQuotaCompose(q.code, null)
        .then((c) => {
          setLines((ls) => ls.map((l) => (l.key === line.key ? { ...l, unitPrice: c.compositePrice } : l)));
        })
        .catch((e) => {
          setToast(`「${q.title}」核算失败（单价 0，可手填）：${e instanceof Error ? e.message : String(e)}`);
        });
    },
    [setLines],
  );

  const addManualLine = useCallback(() => {
    setLines((ls) => [...ls, { key: nextKey(), quotaCode: "", title: "", unit: "", quantity: 1, unitPrice: 0 }]);
  }, [setLines]);

  const resetDraft = useCallback(() => {
    setDraft({ ...DEFAULT_DRAFT, rates: { ...DEFAULT_RATES } });
    setToast("已清空草稿（恢复默认费率 10% / 2% / 7% / 9%）");
  }, []);

  const r = draft.rates;
  const setRate = (k: keyof WorkcostRateSet, pct: number) =>
    setDraft((d) => ({ ...d, rates: { ...d.rates, [k]: pct / 100 } }));

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 工具条 */}
      <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 h-auto min-h-12 py-2 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <Calculator size={14} className="text-accent" /> 取费汇总
        </span>
        <span className="text-[11px] text-fg-faint hidden lg:inline">
          费用汇总口径：直接费=Σ(工程量×综合单价) → 企管/利润/规费 → 税前 → 增值税 · 试算不落库
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <button type="button" className={ghostBtn} onClick={addManualLine} title="添加一行手填清单项（外委包干等）">
            <Plus size={12} /> 手填行
          </button>
          <button type="button" className={solidBtn} onClick={() => setPickerOpen(true)} title="从消耗定额库选行，综合单价现算">
            <List size={12} /> 从定额库添加
          </button>
          <button type="button" className={ghostBtn} onClick={resetDraft} title="清空清单与费率草稿，恢复默认费率">
            <RefreshCw size={12} /> 清空草稿
          </button>
        </div>
      </div>

      {toast && (
        <div className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-border-soft/60 bg-accent/5 text-[11px] text-fg-dim">
          <span className="flex-1 min-w-0 break-all">{toast}</span>
          <button type="button" className={iconBtn} onClick={() => setToast("")} title="关闭提示">
            <X size={10} />
          </button>
        </div>
      )}

      {/* 主体：左清单 + 右取费 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-4">
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-3 items-start">
          {/* 左：分部分项清单（直接费来源） */}
          <section className="v3-panel rounded-2xl p-4 xl:col-span-2" data-testid="workcost-fee-lines">
            <div className="flex items-center justify-between mb-2">
              <span className="text-fg text-[12.5px] font-semibold">分部分项清单</span>
              <span className="text-[10.5px] text-fg-faint">
                定额行的综合单价现算，可手改（= 项目级调价）
              </span>
            </div>
            {draft.lines.length === 0 ? (
              <div className="py-10 text-center text-[11.5px] text-fg-faint leading-relaxed">
                还没有清单行。「从定额库添加」选一条消耗定额（综合单价按当前资源价现算），
                或「手填行」直接录入外委包干类项目。
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-[11.5px]">
                  <thead>
                    <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                      <th className="py-2 px-2 w-20">编码</th>
                      <th className="py-2 px-2">名称</th>
                      <th className="py-2 px-2 w-14">单位</th>
                      <th className="py-2 px-2 w-24 text-right">工程量</th>
                      <th className="py-2 px-2 w-28 text-right">综合单价</th>
                      <th className="py-2 px-2 w-28 text-right">合价</th>
                      <th className="py-2 px-2 w-8" />
                    </tr>
                  </thead>
                  <tbody>
                    {draft.lines.map((l) => (
                      <tr key={l.key} className="border-b border-border-soft/25 last:border-0">
                        <td className="py-1.5 px-2 font-mono text-[10px] text-fg-faint">{l.quotaCode || "—"}</td>
                        <td className="py-1.5 px-2">
                          <input
                            className="w-full bg-transparent border-0 outline-none focus:border-b focus:border-accent text-fg"
                            value={l.title}
                            placeholder="清单项名称"
                            aria-label="清单项名称"
                            onChange={(e) => patchLine(l.key, { title: e.target.value })}
                          />
                        </td>
                        <td className="py-1.5 px-2">
                          <input
                            className="w-full bg-transparent border-0 outline-none focus:border-b focus:border-accent text-fg-faint text-right"
                            value={l.unit}
                            aria-label="单位"
                            onChange={(e) => patchLine(l.key, { unit: e.target.value })}
                          />
                        </td>
                        <td className="py-1.5 px-2 text-right">
                          <input
                            className="w-20 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent"
                            type="number"
                            min={0}
                            value={l.quantity || ""}
                            aria-label="工程量"
                            onChange={(e) => patchLine(l.key, { quantity: Number(e.target.value) })}
                          />
                        </td>
                        <td className="py-1.5 px-2 text-right">
                          <input
                            className="w-24 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent placeholder:text-amber-300/70"
                            type="number"
                            min={0}
                            step="0.01"
                            value={l.unitPrice || ""}
                            aria-label="综合单价"
                            title="定额行现算的核算输出；手改=项目级调价。空=未取到价（核算失败或手填未录），可直接输入"
                            onChange={(e) => patchLine(l.key, { unitPrice: Number(e.target.value) })}
                          />
                        </td>
                        <td className="py-1.5 px-2 text-right tabular-nums text-fg font-medium">
                          {fmtPrice(round2(l.quantity * l.unitPrice))}
                        </td>
                        <td className="py-1.5 px-2">
                          <button
                            type="button"
                            className={iconBtn}
                            title="删除本行"
                            onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}
                          >
                            <Trash2 size={11} />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr className="border-t border-border-soft/50 bg-bg-soft/30">
                      <td className="py-2 px-2 text-fg-faint" colSpan={5}>
                        直接费（分部分项人材机合计 = Σ 工程量 × 综合单价）
                      </td>
                      <td className="py-2 px-2 text-right tabular-nums font-semibold text-fg" data-testid="workcost-fee-direct">
                        {fmtPrice(directFee)}
                      </td>
                      <td />
                    </tr>
                  </tfoot>
                </table>
              </div>
            )}
          </section>

          {/* 右：取费参数 + 费用链 */}
          <section className="v3-panel rounded-2xl p-4 space-y-3" data-testid="workcost-fee-params">
            <div className="flex items-center justify-between">
              <span className="text-fg text-[12.5px] font-semibold">取费参数</span>
              <span className="text-[10.5px] text-fg-faint">费率 0 = 不计取该段</span>
            </div>
            <div className="grid grid-cols-2 gap-2.5">
              <RateField label="企业管理费 %" value={r.managementRate} onChange={(v) => setRate("managementRate", v)} />
              <RateField label="规费 %" value={r.regulatoryRate} onChange={(v) => setRate("regulatoryRate", v)} />
              <RateField label="利润 %" value={r.profitRate} onChange={(v) => setRate("profitRate", v)} />
              <RateField label="增值税 %" value={r.taxRate} onChange={(v) => setRate("taxRate", v)} />
            </div>
            <label className="flex items-center gap-2 text-[11.5px] text-fg-dim cursor-pointer select-none" title="两份实测产物的真实差异：百锦路=不含规费，市政道路模版=含规费">
              <input
                type="checkbox"
                checked={draft.profitIncludesRegulatory}
                onChange={(e) => setDraft((d) => ({ ...d, profitIncludesRegulatory: e.target.checked }))}
              />
              利润基数含规费（直接费+企管+规费）
            </label>
            <div className="grid grid-cols-1 gap-2.5">
              <AmountField label="总价措施费（元）" value={draft.measures} onChange={(v) => setDraft((d) => ({ ...d, measures: v }))} note="已列入分部分项时填 0，不重复计取" />
              <AmountField label="暂列金额（元）" value={draft.contingency} onChange={(v) => setDraft((d) => ({ ...d, contingency: v }))} />
              <AmountField label="招标控制价（元）" value={draft.controlPrice} onChange={(v) => setDraft((d) => ({ ...d, controlPrice: v }))} note="填 0 = 不对照" />
            </div>

            {/* 费用链 */}
            <div className="pt-1">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-fg text-[12.5px] font-semibold">费用汇总</span>
              </div>
              {feeError ? (
                <div role="alert" className="text-[11px] text-amber-300 border border-amber-400/30 bg-amber-400/10 rounded-lg px-2.5 py-2">
                  取费核算失败：{feeError}
                </div>
              ) : fee ? (
                <table className="w-full text-[11.5px]" data-testid="workcost-fee-result">
                  <tbody>
                    <FeeRow label="直接费（人材机）" expr="Σ 工程量×综合单价" amount={fee.directFee} />
                    <FeeRow label="企业管理费" expr={`直接费×${pct(r.managementRate)}`} amount={fee.managementFee} />
                    <FeeRow label="规费" expr={`直接费×${pct(r.regulatoryRate)}`} amount={fee.regulatoryFee} />
                    <FeeRow
                      label="利润"
                      expr={`(${draft.profitIncludesRegulatory ? "直接费+企管+规费" : "直接费+企管"})×${pct(r.profitRate)}`}
                      amount={fee.profitFee}
                    />
                    {fee.measuresFee > 0 && <FeeRow label="总价措施费" expr="计入税前合计" amount={fee.measuresFee} />}
                    {fee.contingency > 0 && <FeeRow label="暂列金额" expr="计入税前合计" amount={fee.contingency} />}
                    <FeeRow label="税前合计（不含税造价）" expr="直接费+企管+利润+规费+措施+暂列" amount={fee.preTaxTotal} strong />
                    <FeeRow label="增值税" expr={`税前合计×${pct(r.taxRate)}`} amount={fee.taxFee} />
                    <FeeRow label="含税总造价" expr="税前合计+增值税" amount={fee.total} strong tone="accent" />
                    {fee.controlPrice > 0 && (
                      <>
                        <FeeRow label="招标控制价" expr="对照基准" amount={fee.controlPrice} />
                        <FeeRow
                          label="差额（造价−控制价）"
                          expr="正数 = 超控制价"
                          amount={fee.controlDiff}
                          tone={fee.controlDiff > 0 ? "warn" : undefined}
                        />
                        <FeeRow
                          label="控制价利用率"
                          expr="含税总造价 ÷ 控制价"
                          amount={fee.controlUtilPct}
                          unit="%"
                          tone={fee.controlUtilPct > 100 ? "warn" : undefined}
                        />
                      </>
                    )}
                  </tbody>
                </table>
              ) : (
                <div className="py-6 text-center text-[11px] text-fg-faint">核算中…</div>
              )}
            </div>

            <p className="text-[10.5px] text-fg-faint leading-relaxed border-t border-border-soft/40 pt-2">
              试算面板：清单与费率存浏览器草稿，不落数据库。定稿请在「单价分析」导出五表工作簿
              （活公式，改价后 Excel 全表重算）。
            </p>
          </section>
        </div>
      </div>

      {pickerOpen && <QuotaPickerModal onCancel={() => setPickerOpen(false)} onPick={addQuotaLine} />}
    </div>
  );
}

function pct(v: number): string {
  return fmt.format(v * 100) + "%";
}

function RateField({ label, value, onChange }: { label: string; value: number; onChange: (pct: number) => void }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">{label}</span>
      <input
        className={fieldCls}
        type="number"
        min={0}
        step="0.1"
        aria-label={label}
        value={value > 0 ? round2(value * 100) : 0}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    </label>
  );
}

function AmountField({
  label,
  value,
  onChange,
  note,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  note?: string;
}) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">
        {label}
        {note && <span className="ml-1.5 text-fg-faint/70">{note}</span>}
      </span>
      <input
        className={fieldCls}
        type="number"
        min={0}
        aria-label={label}
        value={value || ""}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    </label>
  );
}

function FeeRow({
  label,
  expr,
  amount,
  unit = "",
  strong,
  tone,
}: {
  label: string;
  expr: string;
  amount: number;
  unit?: string;
  strong?: boolean;
  tone?: "accent" | "warn";
}) {
  return (
    <tr className="border-b border-border-soft/25 last:border-0">
      <td className="py-1.5 pr-2">
        <div className={strong ? "text-fg font-medium" : "text-fg-dim"}>{label}</div>
        <div className="text-[9.5px] text-fg-faint">{expr}</div>
      </td>
      <td
        className={`py-1.5 pl-2 text-right tabular-nums ${
          tone === "warn" ? "text-amber-300 font-semibold" : tone === "accent" ? "text-accent font-semibold" : "text-fg"
        } ${strong ? "font-semibold" : ""}`}
      >
        {unit === "%" ? fmt.format(amount) + "%" : fmtPrice(amount)}
      </td>
    </tr>
  );
}

// ── 定额选择弹窗（加清单行用）──────────────────────────────────────

function QuotaPickerModal({ onCancel, onPick }: { onCancel: () => void; onPick: (q: WorkcostQuota) => void }) {
  const [keyword, setKeyword] = useState("");
  const [rows, setRows] = useState<WorkcostQuota[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const reqSeq = useRef(0);

  useEffect(() => {
    const seq = ++reqSeq.current;
    setLoading(true);
    setError("");
    app
      .WorkcostQuotaList("", keyword)
      .then((r) => {
        if (seq === reqSeq.current) setRows(r ?? []);
      })
      .catch((e) => {
        if (seq === reqSeq.current) {
          setRows([]);
          setError(e instanceof Error ? e.message : String(e));
        }
      })
      .finally(() => {
        if (seq === reqSeq.current) setLoading(false);
      });
  }, [keyword]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <div className="v3-panel rounded-2xl p-5 space-y-3 w-[40rem] max-h-[80vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-fg font-semibold flex-1">从消耗定额库选行</span>
          <button type="button" className={iconBtn} onClick={onCancel} title="关闭">
            <X size={11} />
          </button>
        </div>
        <div className="relative">
          <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input
            className={`${fieldCls} !pl-6`}
            value={keyword}
            placeholder="搜索定额名称/编码/章节"
            autoFocus
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        {error ? (
          <div className="text-[11px] text-amber-300">定额列表读取失败：{error}</div>
        ) : loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-lg h-9" />
            <div className="v3-panel rounded-lg h-9" />
            <div className="v3-panel rounded-lg h-9" />
          </div>
        ) : rows.length === 0 ? (
          <div className="py-8 text-center text-[11.5px] text-fg-faint">没有匹配的定额。先在「单价分析」导入或新建定额。</div>
        ) : (
          <ul className="v3-panel rounded-lg divide-y divide-border-soft/25" data-testid="workcost-fee-picker-list">
            {rows.slice(0, 50).map((q) => (
              <li key={q.code} className="flex items-center gap-2 px-3 py-2">
                <span className="font-mono text-[10px] text-fg-faint w-14 shrink-0">{q.code}</span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[11.5px] text-fg">{q.title}</span>
                  <span className="block truncate text-[10px] text-fg-faint">
                    {q.unit || "-"} · {q.items?.length ?? 0} 条工料机{q.specialty ? ` · ${q.specialty}` : ""}
                  </span>
                </span>
                <button type="button" className={ghostBtn} onClick={() => onPick(q)}>
                  添加
                </button>
              </li>
            ))}
          </ul>
        )}
        {rows.length > 50 && <p className="text-[11px] text-fg-faint">仅显示前 50 条，输入关键词缩小范围。</p>}
      </div>
    </div>
  );
}
