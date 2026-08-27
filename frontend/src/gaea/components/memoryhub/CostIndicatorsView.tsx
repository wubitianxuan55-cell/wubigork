import { useCallback, useEffect, useState } from "react";
import { BarChart3, RefreshCw, TrendingUp } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostIndicator } from "../../lib/types";

const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";

/**
 * CostIndicatorsView — 造价参考（zaojia-database 蒸馏：案例指标 → 报价对标）。
 * 对「已保存版本/已沉淀」测算项目的明细行实时聚合（不落表），按科目标题或
 * 一级分类给出 样本数/极值/P25/P75/中位数/均值。
 */
export function CostIndicatorsView() {
  const [group, setGroup] = useState<"title" | "category">("title");
  const [rows, setRows] = useState<CostIndicator[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = (await app.CostIndicators(group)) ?? [];
      setRows(r);
    } catch {
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [group]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <BarChart3 size={14} className="text-accent" /> 造价参考
        </span>
        <span className="text-[11px] text-fg-faint hidden md:inline">
          对已保存版本/已沉淀的测算明细实时聚合 · 供下次报价对标
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <div className="flex items-center rounded-lg border border-border bg-bg p-0.5 text-[11px]">
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${group === "title" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setGroup("title")}
              title="按科目标题聚合"
            >
              按科目
            </button>
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${group === "category" ? "bg-accent text-white" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setGroup("category")}
              title="按一级分类聚合"
            >
              按分类
            </button>
          </div>
          <button type="button" className={ghostBtn} onClick={load} title="刷新">
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-4">
        {loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-xl h-10" />
            <div className="v3-panel rounded-xl h-64" />
          </div>
        ) : rows.length === 0 ? (
          <div className="v3-panel rounded-2xl py-16 text-center">
            <TrendingUp size={26} className="mx-auto text-fg-faint" />
            <div className="mt-3 text-[13px] text-fg font-medium">暂无对标案例</div>
            <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">
              到「测算项目」里保存版本或沉淀明细后，这些明细行会自动成为对标样本——
              样本越多，分位数越可信。
            </p>
          </div>
        ) : (
          <div className="v3-panel rounded-xl overflow-x-auto">
            <table className="w-full text-[11.5px]">
              <thead>
                <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50 bg-bg-soft/30">
                  <th className="py-2 px-3">{group === "title" ? "科目" : "一级分类"}</th>
                  <th className="py-2 px-2 w-16 text-right">样本数</th>
                  <th className="py-2 px-2 w-22 text-right">最小值</th>
                  <th className="py-2 px-2 w-22 text-right">P25</th>
                  <th className="py-2 px-2 w-24 text-right">中位数</th>
                  <th className="py-2 px-2 w-24 text-right">均值</th>
                  <th className="py-2 px-2 w-22 text-right">P75</th>
                  <th className="py-2 px-2 w-22 text-right">最大值</th>
                  <th className="py-2 px-2 w-12 text-right">单位</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.key} className="border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30">
                    <td className="py-1.5 px-3 text-fg-dim font-medium">{r.key}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums text-fg-faint">{r.samples}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums">{fmtPrice(r.min)}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums text-fg-dim">{fmtPrice(r.p25)}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums text-fg font-semibold">{fmtPrice(r.median)}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums text-accent">{fmtPrice(r.mean)}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums text-fg-dim">{fmtPrice(r.p75)}</td>
                    <td className="py-1.5 px-2 text-right tabular-nums">{fmtPrice(r.max)}</td>
                    <td className="py-1.5 px-2 text-right text-fg-faint">{r.unit || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default CostIndicatorsView;
