/* eslint-disable react-refresh/only-export-components -- countTurnSteps/latestTurnPhaseText 纯函数导出供测试复用 */
import { useRef } from "react";
import { CheckCircle, Loader2 } from "../icons";
import { useItems, useStore, useTurnStartAt } from "../lib/store";
import type { Item } from "../lib/store";
import { useNow } from "../lib/useNow";
import { formatElapsed } from "../lib/time";
import { useT } from "../lib/i18n";

// WorkHeader — 工作态头部行（v4.26「对话流式重造 · 对齐 Codex」）。
//
// Why：此前 turn_started 不产生 item、ProcessCard 需 seg.processItems.length>0
// 才渲染，首条 text/tool 之前对话窗唯一反馈是顶部细条 StreamingIndicator——
// 用户「发送后长时间静默，像模型崩了」（Codex 调研：首个 token 前必须有可见
// 反馈）。本组件在用户消息后渲染一条常驻头部：items 为空也渲染，消灭死寂
// 窗口；轮完成后转成 Codex 式完成态耗时行「已完成 · 用时 1m23s · 7 步」。
//
// How（数据依赖，全部来自 store 契约）：
//  - 阶段文本：items 里最新 { kind: "phase", id, text } 事件的 text（后端接线
//    「正在启动引擎/正在解析 @引用/思考中/正在重试 (n/m)」等）；只看本轮
//    （最后一条 user 之后），无 phase 回退「思考中…」；
//  - 已用时：turnStartAt（ms，reducer "user"/turn_started 写入）+ useNow 1s tick；
//  - 步数：本轮过程条目计数（与 ProcessCard 内容同口径：工具/阶段/通知/压缩
//    + 纯思考 assistant），见 countTurnSteps；
//  - 完成态：running true→false 时冻结耗时与步数（turnStartAt 会被下一轮
//    覆盖，不能实时算）；恢复历史会话（turnStartAt=0 且未经历运行）不渲染。
//
// 渲染位置：由 Transcript 锚定在「最后一轮的用户消息段」之后（TurnBlock
// workHeader prop）；items 为空且 running 时 Transcript 再兜底挂一个。

/** 已用时格式化挪至 lib/time.ts（formatElapsed），与 task 卡 live 行共用。 */

// countTurnSteps 统计最后一轮（最后一条 user 之后）的过程条目数。
// 口径对齐 alternatingSegments 的 curProcess 收集规则：tool / phase / notice /
// compaction 直接计入；assistant 只有「无正文」（纯思考或空消息）时计入——
// 有正文的 assistant 是给用户看的内容，不算一步。
export function countTurnSteps(items: Item[]): number {
  let lastUser = -1;
  for (let i = items.length - 1; i >= 0; i--) {
    if (items[i].kind === "user") { lastUser = i; break; }
  }
  let n = 0;
  for (let i = lastUser + 1; i < items.length; i++) {
    const it = items[i];
    if (it.kind === "tool" || it.kind === "phase" || it.kind === "notice" || it.kind === "compaction") n++;
    else if (it.kind === "assistant" && !it.text) n++;
  }
  return n;
}

// latestTurnPhaseText 返回本轮最新的 phase 文本（向前扫到 user 边界为止，
// 避免上一轮的「正在重试」泄进新一轮头部）；无则空串（调用方回退「思考中…」）。
export function latestTurnPhaseText(items: Item[]): string {
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i];
    if (it.kind === "user") break;
    if (it.kind === "phase") return it.text;
  }
  return "";
}

export function WorkHeader() {
  const running = useStore((s) => s.running);
  const turnStartAt = useTurnStartAt();
  const items = useItems();
  const now = useNow();
  const t = useT();

  // 完成态冻结：running true→false 的那次渲染里同步把耗时/步数拍进 ref
  // （turnStartAt 会被下一轮覆盖、items 完成后稳定，但「完成时刻」只能当场
  // 捕获）。turn_done/localCancel 在同一次 store 更新里同时写 running 与
  // 终态 items，因此捕获时步数已是终值；ref 渲染期写入是 React 认可的
  // 「前值比较」模式（每次转换只进分支一次，天然幂等）。
  const finalRef = useRef({ elapsed: 0, steps: 0, ran: false });
  const prevRunningRef = useRef(false);
  const wasRunning = prevRunningRef.current;
  prevRunningRef.current = running;
  if (running && !wasRunning) {
    finalRef.current = { elapsed: 0, steps: 0, ran: true };
  } else if (!running && wasRunning) {
    const doneAt = Math.floor(Date.now() / 1000);
    finalRef.current = {
      elapsed: turnStartAt > 0 ? Math.max(0, doneAt - Math.floor(turnStartAt / 1000)) : 0,
      steps: countTurnSteps(items),
      ran: true,
    };
  }

  // 恢复历史会话 / 与本组件无关的空闲态：从未运行过 → 不渲染（保持现状）。
  if (!running && !finalRef.current.ran && turnStartAt <= 0) return null;

  const phaseText = latestTurnPhaseText(items);
  const steps = running ? countTurnSteps(items) : finalRef.current.steps;
  const elapsed = running
    ? (turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0)
    : finalRef.current.elapsed;

  return (
    <div
      data-testid="work-header"
      data-state={running ? "running" : "done"}
      className="my-1.5 flex items-center gap-2 rounded-lg bg-accent/[0.03] px-2.5 py-1.5 text-[11.5px] leading-none"
      title={running && phaseText ? phaseText : undefined}
    >
      {running ? (
        <Loader2 size={12} className="shrink-0 animate-spin text-accent" />
      ) : (
        <CheckCircle size={12} className="shrink-0 text-ok" />
      )}
      <span className={`shrink-0 font-medium ${running ? "text-accent" : "text-fg-dim"}`}>
        {running ? (phaseText || t("work.thinking")) : t("work.done")}
      </span>
      <span className="min-w-0 truncate text-fg-faint/80 tabular-nums">
        · {t("work.elapsed", { t: formatElapsed(elapsed) })}
      </span>
      <span className="shrink-0 text-fg-faint/80 tabular-nums">· {t("work.steps", { n: steps })}</span>
    </div>
  );
}
