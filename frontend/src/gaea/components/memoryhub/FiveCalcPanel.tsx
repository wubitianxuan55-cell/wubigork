import { useCallback, useEffect, useState } from "react";
import { AlertCircle, BarChart3, Calculator, Layers, RefreshCw, Save } from "../../icons";
import { app } from "../../lib/bridge";
import { useToast } from "../Toast";
import type { CostStageCompareRow, CostStageDeviation, CostStageValue } from "../../lib/types";

// 五算阶段固定顺序（对齐后端 coststage.StageOrder：投资估算/设计概算/
// 施工图预算/竣工结算/竣工决算）。
const STAGES = ["估算", "概算", "预算", "结算", "决算"] as const;
const STAGE_FULL: Record<string, string> = {
  估算: "投资估算",
  概算: "设计概算",
  预算: "施工图预算",
  结算: "竣工结算",
  决算: "竣工决算",
};

// 差幅阈值（对齐后端 coststage：|pct|<5 正常 / 5<=|pct|<=15 关注 / >15 异常）。
const WATCH_PCT = 5;
const DEEP_PCT = 15;

const panelCls = "h-full flex flex-col min-h-0 text-[12.5px]";
const sectionTitleCls = "flex items-center gap-1.5 text-[12px] font-semibold text-fg";
const amountInputCls =
  "w-40 shrink-0 bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50 text-right tabular-nums";
const dateInputCls =
  "w-36 shrink-0 bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-white text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const chipCls = "px-1.5 h-[18px] rounded text-[9.5px] font-semibold leading-[18px] shrink-0";

const fmtPrice = (v: number) =>
  "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(v);
// 带符号金额：符号在币符前（-¥40,000 / +¥180,000 / ¥0），避免 "¥-40,000"。
const fmtSignedPrice = (v: number) => {
  const s = fmtPrice(Math.abs(v));
  return v < 0 ? `-${s}` : v > 0 ? `+${s}` : s;
};
const fmtPct = (v: number) => {
  const n = v === 0 ? 0 : v; // 归一 -0，避免 "-0%"
  return `${n > 0 ? "+" : ""}${new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 }).format(n)}%`;
};
const parseAmount = (s: string) => {
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
};

// 差幅着色：上升红 / 下降绿；|pct|>=15 深色、5<=|pct|<15 琥珀、否则默认。
function diffCls(pct: number): string {
  const ap = Math.abs(pct);
  if (ap >= DEEP_PCT) return pct > 0 ? "text-red-500" : "text-emerald-500";
  if (ap >= WATCH_PCT) return "text-amber-400";
  return pct > 0 ? "text-red-400" : pct < 0 ? "text-emerald-400" : "text-fg-dim";
}

// 偏差档位徽标：正常绿 / 关注琥珀 / 异常红。
function levelCls(level: string): string {
  if (level === "正常") return "text-ok bg-ok/10";
  if (level === "关注") return "text-amber-400 bg-amber-400/10";
  if (level === "异常") return "text-red-500 bg-red-500/10";
  return "text-fg-dim bg-bg-elev";
}

/**
 * FiveCalcPanel — 五算对比（估/概/预/结/决）。
 *
 * 交互：五阶段金额/日期输入 + 单阶段保存（CostStageSave，成功 toast + 刷新对比）+
 * 对比表（环比 chainDiff/chainDiffPct、累计差 baseDiff/baseDiffPct，差幅着色）+
 * 偏差卡片（level 徽标 + 后端规则 suggestion 文案）。无任何阶段值时展示空态提示。
 * 父级可传 onChanged：保存成功后回调（刷新项目列表合计等）。
 */
export function FiveCalcPanel({ projectId, onChanged }: { projectId: string; onChanged?: () => void }) {
  const toast = useToast();
  const [stages, setStages] = useState<CostStageValue[]>([]);
  const [compare, setCompare] = useState<CostStageCompareRow[]>([]);
  const [deviations, setDeviations] = useState<CostStageDeviation[]>([]);
  const [loading, setLoading] = useState(true);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [dates, setDates] = useState<Record<string, string>>({});
  const [savingStage, setSavingStage] = useState<string | null>(null);

  const loadAll = useCallback(async () => {
    const [sv, cmp, dev] = await Promise.all([
      app.CostStages(projectId).catch(() => [] as CostStageValue[]),
      app.CostStageCompare(projectId).catch(() => [] as CostStageCompareRow[]),
      app.CostStageDeviations(projectId).catch(() => [] as CostStageDeviation[]),
    ]);
    const list = sv ?? [];
    setStages(list);
    setCompare(cmp ?? []);
    setDeviations(dev ?? []);
    // 已存值回填到输入框（金额/日期），未录入阶段保持空。
    const d: Record<string, string> = {};
    const dt: Record<string, string> = {};
    for (const v of list) {
      d[v.stage] = v.amount > 0 ? String(v.amount) : "";
      dt[v.stage] = v.date ?? "";
    }
    setDrafts(d);
    setDates(dt);
    setLoading(false);
  }, [projectId]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const saveStage = useCallback(
    async (stage: string) => {
      const raw = (drafts[stage] ?? "").trim();
      if (raw === "") {
        toast.show("请输入金额后再保存", "warn");
        return;
      }
      setSavingStage(stage);
      try {
        const payload: CostStageValue = {
          id: 0, // 后端 SaveStage 按 (project_id, stage) UPSERT，id/时间戳忽略
          projectId,
          stage,
          amount: parseAmount(raw),
          date: dates[stage] ?? "",
          note: "",
          createdAt: "",
          updatedAt: "",
        };
        await app.CostStageSave(payload);
        toast.show(`${stage}阶段已保存`, "info");
        await loadAll();
        onChanged?.();
      } catch (e) {
        toast.show(String(e), "error");
      } finally {
        setSavingStage(null);
      }
    },
    [projectId, drafts, dates, loadAll, toast, onChanged],
  );

  const stageById = new Map(stages.map((v) => [v.stage, v]));
  const empty = stages.length === 0;

  return (
    <div className={panelCls}>
      <div className="shrink-0 flex items-center gap-2 px-4 h-10 border-b border-border-soft/50">
        <span className="flex items-center gap-1.5 text-fg font-semibold text-[12px]">
          <Calculator size={13} className="text-accent" /> 五算对比
        </span>
        <span className="text-[10.5px] text-fg-faint">估 / 概 / 预 / 结 / 决</span>
        <button type="button" className={`${ghostBtn} ml-auto`} title="刷新对比" onClick={() => void loadAll()}>
          <RefreshCw size={12} /> 刷新
        </button>
      </div>

      {loading ? (
        <div className="flex-1 flex items-center justify-center text-[11.5px] text-fg-faint">加载中…</div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3 space-y-3">
          {/* 五阶段录入 */}
          <div className="flex items-center justify-between">
            <span className={sectionTitleCls}>
              <Layers size={12} className="text-sky-400" /> 五阶段录入
            </span>
            <span className="text-[10.5px] text-fg-faint">金额单位：元</span>
          </div>
          <div className="v3-panel rounded-xl divide-y divide-border-soft/40">
            {STAGES.map((stage) => {
              const saved = stageById.get(stage);
              return (
                <div key={stage} data-testid={`stage-input-${stage}`} className="flex items-center gap-2 px-3 py-2">
                  <span className="w-9 shrink-0 font-medium text-fg" title={STAGE_FULL[stage]}>
                    {stage}
                  </span>
                  {saved ? (
                    <span className={`${chipCls} bg-ok/10 text-ok`} title={saved.date ? `录入日期 ${saved.date}` : undefined}>
                      {saved.date ? saved.date.slice(5) : "已录入"}
                    </span>
                  ) : (
                    <span className={`${chipCls} bg-bg-elev text-fg-faint font-normal`}>未录入</span>
                  )}
                  <input
                    className={amountInputCls}
                    type="number"
                    min={0}
                    step="0.01"
                    value={drafts[stage] ?? ""}
                    placeholder="金额（元）"
                    onChange={(e) => setDrafts((p) => ({ ...p, [stage]: e.target.value }))}
                  />
                  <input
                    className={dateInputCls}
                    type="date"
                    value={dates[stage] ?? ""}
                    title="阶段日期（可选）"
                    onChange={(e) => setDates((p) => ({ ...p, [stage]: e.target.value }))}
                  />
                  <button
                    type="button"
                    className={solidBtn}
                    disabled={savingStage === stage}
                    aria-label={`保存${stage}`}
                    title={`保存${STAGE_FULL[stage]}金额`}
                    onClick={() => void saveStage(stage)}
                  >
                    <Save size={12} /> 保存
                  </button>
                </div>
              );
            })}
          </div>

          {empty ? (
            <div className="v3-panel rounded-xl py-10 text-center text-[11.5px] text-fg-faint">
              录入五个阶段的金额后自动生成对比与偏差诊断
            </div>
          ) : (
            <>
              {/* 对比表 */}
              <div className="flex items-center justify-between">
                <span className={sectionTitleCls}>
                  <BarChart3 size={12} className="text-sky-400" /> 对比表
                </span>
              </div>
              <div className="v3-panel rounded-xl overflow-x-auto">
                {compare.length === 0 ? (
                  <div className="py-8 text-center text-[11.5px] text-fg-faint">
                    至少录入两个阶段的金额后自动生成对比
                  </div>
                ) : (
                  <table className="w-full text-[11.5px]">
                    <thead>
                      <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                        <th className="py-1.5 px-2 w-16">阶段</th>
                        <th className="py-1.5 px-2 text-right">金额</th>
                        <th className="py-1.5 px-2 text-right">
                          环比
                          <span className="block text-[9px] text-fg-faint/70 font-normal">较上一有值阶段</span>
                        </th>
                        <th className="py-1.5 px-2 text-right">
                          累计差
                          <span className="block text-[9px] text-fg-faint/70 font-normal">较首个有值阶段</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {compare.map((r) => (
                        <CompareRowView key={r.stage} r={r} />
                      ))}
                    </tbody>
                  </table>
                )}
              </div>

              {/* 偏差卡片 */}
              <div className="flex items-center justify-between">
                <span className={sectionTitleCls}>
                  <AlertCircle size={12} className="text-amber-400" /> 偏差诊断
                </span>
              </div>
              {deviations.length === 0 ? (
                <div className="text-[11px] text-fg-faint">暂无偏差诊断——相邻阶段差幅数据不足或处于正常范围。</div>
              ) : (
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-2">
                  {deviations.map((d) => (
                    <DeviationCard key={`${d.fromStage}-${d.toStage}`} d={d} />
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}

// 对比表行：hasValue=false 的缺失阶段整行显示「—」。
function CompareRowView({ r }: { r: CostStageCompareRow }) {
  return (
    <tr data-testid={`compare-row-${r.stage}`} className="border-b border-border-soft/30 last:border-0 align-top">
      <td className="py-1.5 px-2 text-fg font-medium">{r.stage}</td>
      {!r.hasValue ? (
        <>
          <td className="py-1.5 px-2 text-right text-fg-faint">—</td>
          <td className="py-1.5 px-2 text-right text-fg-faint">—</td>
          <td className="py-1.5 px-2 text-right text-fg-faint">—</td>
        </>
      ) : (
        <>
          <td className="py-1.5 px-2 text-right tabular-nums text-fg">{fmtPrice(r.amount)}</td>
          <td className="py-1.5 px-2 text-right tabular-nums">
            {r.hasPrev ? (
              <>
                <div className="text-fg-dim">{fmtSignedPrice(r.chainDiff)}</div>
                <div className={diffCls(r.chainDiffPct)}>{fmtPct(r.chainDiffPct)}</div>
              </>
            ) : (
              <span className="text-fg-faint">—</span>
            )}
          </td>
          <td className="py-1.5 px-2 text-right tabular-nums">
            <div className="text-fg-dim">{fmtSignedPrice(r.baseDiff)}</div>
            <div className={diffCls(r.baseDiffPct)}>{fmtPct(r.baseDiffPct)}</div>
          </td>
        </>
      )}
    </tr>
  );
}

// 偏差卡片：标题「预算较概算 +18.2%」+ direction + level 徽标 + 后端规则文案。
function DeviationCard({ d }: { d: CostStageDeviation }) {
  const down = d.direction === "下降";
  return (
    <div data-testid={`deviation-${d.fromStage}-${d.toStage}`} className="v3-panel rounded-xl p-3">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-[12px] font-semibold text-fg">
          {d.toStage}较{d.fromStage} {fmtPct(d.diffPct)}
        </span>
        <span className={`${chipCls} ${levelCls(d.level)}`}>{d.level}</span>
      </div>
      <div className="mt-1 flex items-center gap-1.5 text-[10.5px]">
        <span className={down ? "text-emerald-400" : "text-red-400"}>{down ? "↓ 下降" : "↑ 上升"}</span>
        <span className="text-fg-faint">
          {d.fromStage} {fmtPrice(d.fromAmount)} → {d.toStage} {fmtPrice(d.toAmount)}
        </span>
      </div>
      <p className="mt-1.5 text-[11px] text-fg-dim leading-relaxed">{d.suggestion}</p>
    </div>
  );
}
