import { useEffect, useRef, useState } from "react";
import { Cpu } from "../icons";
import { fmtTokens } from "../lib/stats";
import { useNow } from "../lib/useNow";
import type { JobView } from "../lib/types";
import { useToast } from "./Toast";

// ── 流式速度计（办公底栏 tok/s）────────────────────────────
// turnTokens 由 store 随分片推进；useNow 每秒走针时取 ~4s 前样本算差商。
// 工具执行/思考停顿期 tokens 不涨，速度自然衰减到 0（显示阈值 0.5 隐藏）。

export type SpeedPt = [number, number]; // [unix 秒, 该秒已流式 tokens]

export const SPEED_WINDOW_S = 4;

// 每秒至多一个样本：同秒重复渲染（分片突发）覆盖取最大，防突发把基点抬高。
export function pushSpeedPt(pts: SpeedPt[], now: number, tokens: number): void {
  const last = pts[pts.length - 1];
  if (last && last[0] === now) {
    last[1] = Math.max(last[1], tokens);
    return;
  }
  pts.push([now, tokens]);
  while (pts.length > SPEED_WINDOW_S + 2) pts.shift();
}

// 差商基点 = 窗口内最早样本（不足 4s 时退化为首样本即平均速度）。
export function rollingTokPerSec(pts: SpeedPt[], now: number, tokens: number): number {
  if (!pts.length) return 0;
  const base = pts.find(([t]) => t <= now - SPEED_WINDOW_S) ?? pts[0];
  const dt = now - base[0];
  return dt > 0 ? Math.max(0, (tokens - base[1]) / dt) : 0;
}

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
  // v4.350：仅运行中订阅全局时钟（!running 时组件渲染 null，白订阅纯浪费）
  const now = useNow(running);
  // 流式速度：秒级采样 → 4s 滚动窗口差商；新回合（turnStartAt 变化）清样本。
  const [tokSpeed, setTokSpeed] = useState(0);
  const speedPtsRef = useRef<SpeedPt[]>([]);
  const speedTurnRef = useRef(0);
  useEffect(() => {
    if (!running) {
      setTokSpeed(0);
      return;
    }
    if (speedTurnRef.current !== turnStartAt) {
      speedTurnRef.current = turnStartAt;
      speedPtsRef.current = [];
    }
    pushSpeedPt(speedPtsRef.current, now, turnTokens);
    setTokSpeed(rollingTokPerSec(speedPtsRef.current, now, turnTokens));
  }, [running, turnStartAt, turnTokens, now]);
  if (!running) return null;
  const elapsed = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
  const elapsedStr = elapsed < 60 ? `${elapsed}s` : `${Math.floor(elapsed / 60)}m${elapsed % 60}s`;
  const tokStr = turnTokens > 0 ? `↓${fmtTokens(turnTokens)}` : "";
  // 实时速度（4s 窗口）→ 停顿期退化为回合均速（暗色「均」前缀），保证整个
  // 回合期间速度始终可见：工具执行/检索阶段 tokens 不涨，实时窗口会归零。
  const avgSpeed = elapsed > 0 && turnTokens > 0 ? turnTokens / elapsed : 0;
  const fmtSpd = (v: number) => (v >= 10 ? Math.round(v) : Math.round(v * 10) / 10);
  const spdStr =
    tokSpeed >= 0.5
      ? `${fmtSpd(tokSpeed)} tok/s`
      : avgSpeed >= 0.5
        ? `均 ${fmtSpd(avgSpeed)} tok/s`
        : "";
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
        {spdStr && (
          <span
            className={tokSpeed >= 0.5 ? "text-info/80" : "text-fg-faint"}
            title={tokSpeed >= 0.5 ? "流式输出速度（4s 滚动窗口）" : "回合平均速度（当前处于工具/思考停顿，无实时流）"}
          >
            {spdStr}
          </span>
        )}
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
