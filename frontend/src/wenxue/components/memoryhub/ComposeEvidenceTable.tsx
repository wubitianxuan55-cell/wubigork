// ComposeEvidenceTable — AI 组价证据链表（v4.158 自 ComposeModal 抽出共用）：
// ComposeModal（组价建议）与 CostEntryModal（组价依据回看）两处渲染同一套
// 溯源列（标题/规格/单价/单位/来源/地区/期数/口径），抽小组件避免双份维护。
// 行数据按结构类型接收（PriceBandSource / CostComposeEvidence 均天然兼容），
// 传入 band 时按 IQR 口径标注离群样本（与价格带卡片同一口径）。
import type { PriceBand } from "../../lib/types";

// 证据行最小结构（不导出——调用方按结构传入，类型名无需感知）。
interface ComposeEvidenceRow {
  title: string;
  spec: string;
  price: number;
  unit: string;
  source: string;
  region: string;
  priceDate: string;
  priceType: string;
}

const fmtPrice = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });

// 离群判定：样本价 < P25-1.5IQR 或 > P75+1.5IQR（IQR = P75-P25），与后端 band.outliers 同口径。
const isOutlierPrice = (band: PriceBand, price: number) => {
  const iqr = band.p75 - band.p25;
  if (iqr <= 0) return false;
  return price < band.p25 - 1.5 * iqr || price > band.p75 + 1.5 * iqr;
};

export function ComposeEvidenceTable({
  rows,
  band = null,
  maxCls = "max-h-[36vh]",
}: {
  rows: ComposeEvidenceRow[];
  /** 传入价格带时对越界样本标注「离群」（组价建议与回看快照同口径）。 */
  band?: PriceBand | null;
  /** 滚动容器最大高度类（回看小表可用更矮的高度）。 */
  maxCls?: string;
}) {
  return (
    <div className={`${maxCls} overflow-auto rounded-lg border border-border-soft`}>
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
          {rows.map((s, i) => {
            const outlier = band ? isOutlierPrice(band, s.price) : false;
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
  );
}
