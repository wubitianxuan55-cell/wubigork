import { useEffect, useState } from "react";
import { Modal } from "antd";
import { BarChart3, Loader } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostCompareRow } from "../../lib/types";
import { useToast } from "../Toast";

const fmtPrice = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const fmtPct = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 });

// 比价来源类型 → 中文标注（kind 契约：current=现价 / history=历史快照 / fetch=价格源抓取）。
const KIND_LABELS: Record<CostCompareRow["kind"], string> = {
  current: "现价",
  history: "历史快照",
  fetch: "价格源抓取",
};

// 相对现价跳幅着色：|diffPct|>=20 红、>5 琥珀、否则绿。
const diffClass = (p: number) => {
  const a = Math.abs(p);
  if (a >= 20) return "text-red-400";
  if (a > 5) return "text-amber-400";
  return "text-emerald-400";
};

/**
 * CostCompareModal 供应商比价弹层：以成本条目现价为基准，跨来源对比
 * 历史快照 / 价格源抓取 / 市场参考等可比价格，展示相对现价跳幅（diffPct）。
 * 无其他来源时给出空态提示，查询失败持久展示错误（可关闭后重开重试）。
 */
export function CostCompareModal({
  open,
  name,
  title,
  currentPrice,
  onClose,
}: {
  open: boolean;
  name: string;
  title: string;
  /** 成本条目现价（比价基准，用于展示参考行）。 */
  currentPrice?: number;
  onClose: () => void;
}) {
  const [rows, setRows] = useState<CostCompareRow[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  useEffect(() => {
    if (!open || !name) return;
    setRows(null);
    setError(null);
    setLoading(true);
    app
      .CostCompare(name)
      .then((rs) => setRows(rs ?? []))
      .catch((e) => {
        setError(String(e));
        toast.show(`比价失败：${String(e)}`, "warn");
      })
      .finally(() => setLoading(false));
  }, [open, name, toast]);

  return (
    <Modal
      title={
        <span className="flex items-center gap-2">
          <BarChart3 size={14} className="text-sky-400" />
          供应商比价：{title}
        </span>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={680}
      // WebView2 在特定状态下会冻结 rAF/CSS 动画：退出动画永远不结束，
      // 遮罩残留在窗口上导致整个软件点不了。这里禁用弹层动画，关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
    >
      {typeof currentPrice === "number" && (
        <div className="mb-2 flex items-center gap-2 text-[11.5px] text-fg-dim">
          <span className="text-fg-faint">现价基准</span>
          <span className="font-semibold text-amber-300 tabular-nums">¥{fmtPrice.format(currentPrice)}</span>
          <span className="text-fg-faint">· 各来源价格相对现价的跳幅：≥20% 红 / &gt;5% 琥珀 / 其余绿</span>
        </div>
      )}
      {loading ? (
        <div className="flex items-center justify-center gap-2 py-10 text-fg-faint text-[12px]" role="status">
          <Loader size={14} className="animate-spin text-accent" />
          正在查询比价数据…
        </div>
      ) : error ? (
        <div className="px-3 py-3 rounded-lg border border-err/40 bg-err/10 text-fg-dim text-[11.5px]" role="alert">
          <span className="text-err font-medium">比价失败：{error}</span>
          <div className="mt-0.5">可关闭弹层后重新打开重试，或检查价格源/历史快照数据。</div>
        </div>
      ) : !rows || rows.length === 0 ? (
        <div className="py-10 text-center text-fg-faint text-[12px]">暂无其他来源</div>
      ) : (
        <div className="max-h-[52vh] overflow-auto rounded-lg border border-border-soft">
          <table className="w-full text-[11.5px]">
            <thead className="sticky top-0 bg-bg-elev text-fg-faint text-left">
              <tr>
                <th className="px-2 py-1.5 min-w-[150px]">来源</th>
                <th className="px-2 py-1.5 w-24">期数</th>
                <th className="px-2 py-1.5 w-28 text-right">价格(元)</th>
                <th className="px-2 py-1.5 w-28 text-right">相对现价</th>
                <th className="px-2 py-1.5 min-w-[120px]">抓取时间</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={i} className="border-t border-border-soft/60">
                  <td className="px-2 py-1.5">
                    <span className="text-fg">{r.source || "—"}</span>
                    <span className="ml-1.5 px-1 py-0.5 rounded bg-bg-elev text-fg-faint text-[9.5px] align-middle">
                      {KIND_LABELS[r.kind] ?? r.kind}
                    </span>
                  </td>
                  <td className="px-2 py-1.5 text-fg-dim">{r.period || "—"}</td>
                  <td className="px-2 py-1.5 text-right text-amber-300 font-semibold tabular-nums whitespace-nowrap">
                    ¥{fmtPrice.format(r.price)}
                  </td>
                  <td className={`px-2 py-1.5 text-right font-medium tabular-nums whitespace-nowrap ${diffClass(r.diffPct)}`}>
                    {r.diffPct > 0 ? "+" : ""}
                    {fmtPct.format(r.diffPct)}%
                  </td>
                  <td className="px-2 py-1.5 text-fg-faint text-[10.5px] whitespace-nowrap">
                    {r.fetchedAt ? new Date(r.fetchedAt).toLocaleString("zh-CN", { hour12: false }) : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Modal>
  );
}
