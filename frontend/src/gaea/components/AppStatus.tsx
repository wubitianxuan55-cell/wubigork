import { useEffect, useRef } from "react";
import { Cpu } from "../icons";
import { fmtTokens } from "../lib/stats";
import { useNow } from "../lib/useNow";
import type { JobView } from "../lib/types";
import { useToast } from "./Toast";

export function NewSessionToast({ done }: { done: boolean }) {
  const toast = useToast();
  useEffect(() => { if (done) toast.show("新会话已创建", "info"); }, [done, toast]);
  return null;
}

// 后台任务从运行列表消失即视为结束，弹 toast 提示。
export function JobDoneNotifier({ jobs }: { jobs: JobView[] }) {
  const toast = useToast();
  const prevRef = useRef<Map<string, string>>(new Map()); // id -> label
  useEffect(() => {
    const prev = prevRef.current;
    const current = new Map(jobs.map((j) => [j.id, j.label] as const));
    for (const [id, label] of prev) {
      if (!current.has(id)) toast.show(`后台任务已完成：${label}`, "info");
    }
    prevRef.current = current;
  }, [jobs, toast]);
  return null;
}

// 输入框上方的运行时状态行。
// C5（蒸馏 codex thread_usage/chatwidget 上下文窗口百分比 + 压缩前预警）：
// 窗口占用 ≥75% 提示「接近自动压缩」（gaea 压缩触发线 80%），≥90% 升级为
// 「即将强制压缩」（ForceRatio 90%）；窗口未知（window=0）时隐藏。
export function RunStatus({ running, turnStartAt, turnTokens, used, window: win }: {
  running: boolean;
  turnStartAt: number;
  turnTokens: number;
  used: number;
  window: number;
}) {
  const now = useNow();
  if (!running) return null;
  const elapsed = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
  const elapsedStr = elapsed < 60 ? `${elapsed}s` : `${Math.floor(elapsed / 60)}m${elapsed % 60}s`;
  const tokStr = turnTokens > 0 ? `↓${fmtTokens(turnTokens)}` : "";
  const slowHint =
    elapsed >= 20 && used >= 40000
      ? `处理大上下文中 · ${fmtTokens(used)}`
      : "";
  const pct = win > 0 ? Math.min(100, Math.round((used / win) * 100)) : 0;
  const ctxHint =
    pct >= 90 ? "即将强制压缩" : pct >= 75 ? "接近自动压缩" : "";
  const ctxColor = pct >= 90 ? "text-err" : pct >= 75 ? "text-warning" : "text-fg-faint";
  return (
    <div className="flex items-center justify-between px-4 py-1.5 text-[11px] select-none border-b border-border-soft/50 bg-bg-soft/30">
      <div className="flex items-center gap-2 text-fg-dim tabular-nums font-mono">
        <span className="font-medium">{elapsedStr}</span>
        {tokStr && <span className="text-fg-faint">{tokStr}</span>}
        {slowHint && <span className="text-warning/90">{slowHint}</span>}
        {pct > 0 && (
          <span className={`inline-flex items-center gap-1 ${ctxColor}`} title={`上下文窗口占用 ${fmtTokens(used)}/${fmtTokens(win)}`}>
            <span className="inline-block w-10 h-1 rounded-full bg-bg-soft overflow-hidden align-middle">
              <span
                className={`block h-full rounded-full transition-all duration-500 ${pct >= 90 ? "bg-err" : pct >= 75 ? "bg-warning" : "bg-info/70"}`}
                style={{ width: `${pct}%` }}
              />
            </span>
            <span className="font-medium">{pct}%</span>
            {ctxHint && <span>{ctxHint}</span>}
          </span>
        )}
      </div>
      <div className="flex items-center gap-3">
        <span className="flex items-center gap-1.5 text-fg">
          <Cpu size={12} className="text-info" />
          <span className="font-medium">执行中</span>
          <span className="inline-flex items-center gap-1 ml-0.5">
            <span className="w-1.5 h-1.5 rounded-full bg-info animate-pulse" />
            <span className="text-[10px] text-info/70">中</span>
          </span>
        </span>
      </div>
    </div>
  );
}
