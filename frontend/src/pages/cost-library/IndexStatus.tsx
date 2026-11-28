// IndexStatus.tsx — 造价数据库板块状态栏上的「语义索引」状态钮（2026-11-26 造价库接入本地
// 模型索引）。平时后台索引器自动维护（写入打脏 → 合并 → 增量对账），这里是「看着它补上」
// 的入口：一眼看出能不能按意思搜，点一下立即补齐。
import { useState } from "react";
import type { CostIndexStatusView } from "../../gaea/lib/types";

export type IndexTone = "ok" | "warn" | "partial" | "empty";

/** 状态钮的文案：tone 决定配色，text 是钮上的字，title 是悬停明细。 */
export function indexSummary(st: CostIndexStatusView): { tone: IndexTone; text: string; title: string; total: number } {
  const failing = st.kinds.filter((k) => !!k.lastError);
  // 检索时发现向量模型不可用（搜索框旁另有明示）：索引可能明明是齐的，语义检索却是坏的。
  const unreachable = st.kinds.filter((k) => !!k.searchError);
  // 「总数」取对账看到的条数与已建向量数的较大者：刚重启、这一轮还没对完时对账数可能落后。
  const count = st.kinds.reduce((n, k) => n + k.count, 0);
  const total = st.kinds.reduce((n, k) => n + Math.max(k.total, k.count), 0);
  const detail = st.kinds.map((k) => `${k.label} ${k.count}/${k.total}`).join(" · ");
  const lines = [`向量模型 ${st.model}`, detail];
  if (!st.running) lines.push("后台索引器未运行");
  for (const k of failing) lines.push(`${k.label}：${k.lastError}`);
  for (const k of unreachable) lines.push(`${k.label}：${k.searchError}`);
  lines.push("点击立即补齐（缺的补、正文变了的重嵌）");
  const title = lines.join("\n");
  if (unreachable.length > 0) return { tone: "warn", text: "语义模型 不可用", title, total };
  if (failing.length > 0) return { tone: "warn", text: "语义索引 待补齐", title, total };
  if (count === 0) return { tone: "empty", text: "语义索引 未建", title, total };
  if (count < total) return { tone: "partial", text: `语义索引 ${count}/${total}`, title, total };
  return { tone: "ok", text: "语义索引 就绪", title, total };
}

/** onBackfill 补齐动作（页面负责调绑定、重读状态、出提示）；这里只管钮的忙闲。 */
export function IndexStatus({ status, onBackfill }: {
  status: CostIndexStatusView | null;
  onBackfill: () => Promise<void>;
}) {
  const [busy, setBusy] = useState(false);
  if (!status) return null;
  const s = indexSummary(status);
  // 库里一条可索引的都没有（空库）：没有「索引」可言，不出这个钮。
  if (s.tone === "empty" && s.total === 0) return null;
  return (
    <button
      type="button"
      className="cl-index"
      data-tone={s.tone}
      data-testid="cost-index-status"
      title={s.title}
      disabled={busy}
      onClick={() => {
        setBusy(true);
        void onBackfill().finally(() => setBusy(false));
      }}
    >
      {busy ? "索引补齐中…" : s.text}
    </button>
  );
}
