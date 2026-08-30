import { useCallback, useEffect, useState } from "react";
import { Bot, CheckCircle, ChevronDown, ChevronRight, Loader2, Users, XCircle } from "../icons";
import { app } from "../lib/bridge";
import type { SubagentRunView } from "../lib/types";
import { usePollingGate } from "../../hooks/usePollingGate";

// SubagentsPanel — 多智能体分工可见（P2，对标 WorkSwarm 蜂群 / QClaw V2 多 Agent）：
// 展示当前会话派发的全部子代理「谁在干什么」——状态徽标、任务摘要（transcript
// 首条 user 消息）、模型、工具范围、时间，点击展开最后回答。
// 数据源：GaeaSubagentRuns(sessionPath)（meta + transcript 派生）。
// v3「星枢」面板语言：v3-panel-head 细条头部；状态徽标 = 语义色 + 图标 + 文字三重传达。

function statusMeta(status: string): { icon: React.ReactNode; color: string; text: string } {
  switch (status) {
    case "running":
      return { icon: <Loader2 size={10} className="animate-spin" aria-hidden />, color: "var(--gaea-glow)", text: "进行中" };
    case "completed":
      return { icon: <CheckCircle size={10} aria-hidden />, color: "var(--md-sys-color-success)", text: "已完成" };
    default:
      return { icon: <XCircle size={10} aria-hidden />, color: "var(--md-sys-color-destructive)", text: "失败" };
  }
}

function fmtTime(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString("zh-CN", { hour12: false });
}

function fmtDuration(created: string, updated: string): string {
  const c = new Date(created).getTime();
  const u = new Date(updated).getTime();
  if (!Number.isFinite(c) || !Number.isFinite(u) || u < c) return "";
  const s = Math.max(1, Math.round((u - c) / 1000));
  if (s < 60) return `${s} 秒`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} 分`;
  return `${Math.floor(m / 60)} 小时`;
}

const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

export function SubagentsPanel({ sessionPath }: { sessionPath?: string }) {
  const [view, setView] = useState<{ available: boolean; runs: SubagentRunView[]; running: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  // v4.5.2：子代理运行轮询接入系统级后台轮询门控（页面不可见时空转零成本）
  const gate = usePollingGate();

  const load = useCallback(() => {
    if (!sessionPath) {
      setView(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    app
      .SubagentRuns(sessionPath)
      .then((v) => setView(v))
      .catch(() => setView({ available: false, runs: [], running: 0 }))
      .finally(() => setLoading(false));
  }, [sessionPath]);

  // 会话切换重新拉取；运行中的子代理每 5 秒轮询刷新（轻量，仅面板打开时）
  useEffect(() => {
    const tick = () => { if (gate) load() };
    tick();
    if (!sessionPath) return;
    const timer = window.setInterval(tick, 5000);
    return () => window.clearInterval(timer);
  }, [load, sessionPath, gate]);

  const toggle = useCallback((ref: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(ref)) next.delete(ref);
      else next.add(ref);
      return next;
    });
  }, []);

  const hasRuns = (view?.runs.length ?? 0) > 0;

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 计数徽标 + 刷新 */}
      <div className="v3-panel-head">
        <Users size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">分工</span>
        {hasRuns && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {view?.runs.length}
          </span>
        )}
        {view && view.running > 0 && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
              color: "var(--md-sys-color-warning)",
              border: "1px solid color-mix(in srgb, var(--md-sys-color-warning) 32%, transparent)",
            }}
          >
            {view.running} 运行中
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button type="button" className={iconBtn} onClick={() => void load()} title="刷新分工列表" aria-label="刷新分工列表">
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      {loading && !hasRuns ? (
        <div className="flex items-center justify-center flex-1 gap-2 text-[11px]">
          <Loader2 size={14} className="animate-spin" />
          读取子代理分工…
        </div>
      ) : !hasRuns ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center">
          <Bot size={24} aria-hidden className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            {view?.available === false || !sessionPath
              ? "本会话尚未派发子代理"
              : "暂无子代理分工记录"}
            <br />
            Agent 用 task 工具派发子任务后会出现在这里
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1.5">
          {view?.runs.map((r) => {
            const st = statusMeta(r.status);
            const isOpen = expanded.has(r.ref);
            return (
              <div
                key={r.ref}
                className="flex flex-col gap-1 px-2 py-1.5 rounded-[var(--radius-md)] transition-all duration-200"
                style={{
                  background: "var(--md-sys-color-surface-container)",
                  border: "1px solid var(--md-sys-color-outline-variant)",
                }}
              >
                <button
                  type="button"
                  className="flex items-start gap-2 min-w-0 text-left cursor-pointer"
                  onClick={() => toggle(r.ref)}
                  aria-expanded={isOpen}
                >
                  <span
                    className="shrink-0 inline-flex items-center gap-1 rounded-full px-1.5 py-px text-[9px] leading-none font-medium"
                    style={{ color: st.color, background: `color-mix(in srgb, ${st.color} 12%, transparent)`, border: `1px solid color-mix(in srgb, ${st.color} 32%, transparent)` }}
                  >
                    {st.icon}
                    {st.text}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block text-[12px] font-medium leading-snug" style={{ color: "var(--md-sys-color-text)" }}>
                      {r.task || "（无任务摘要）"}
                    </span>
                    <span className="block mt-0.5 text-[10px] font-mono leading-tight">
                      {r.ref.slice(0, 24)}…{r.toolCalls > 0 ? ` · ${r.toolCalls} 次工具调用` : ""}
                    </span>
                  </span>
                  <span className="shrink-0 mt-0.5 text-(color:--md-sys-color-text-secondary)">
                    {isOpen ? <ChevronDown size={12} aria-hidden /> : <ChevronRight size={12} aria-hidden />}
                  </span>
                </button>

                <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[10px] px-0.5">
                  {r.model && (
                    <span className="rounded bg-(color:--md-sys-color-surface-container-high) px-1 py-px font-mono">{r.model}</span>
                  )}
                  <span>{fmtTime(r.createdAt)}</span>
                  {fmtDuration(r.createdAt, r.updatedAt) && <span>· {fmtDuration(r.createdAt, r.updatedAt)}</span>}
                  {r.toolScope && r.toolScope.length > 0 && (
                    <span className="truncate max-w-[180px]" title={r.toolScope.join(", ")}>
                      工具：{r.toolScope.join(" / ")}
                    </span>
                  )}
                </div>

                {/* C2 活动行：运行中的子代理「此刻正在干什么」（transcript 尾部派生） */}
                {r.status === "running" && (r.lastText || r.lastTool) && (
                  <div className="flex flex-col gap-0.5 px-1.5 py-1 rounded-md text-[10.5px] leading-relaxed"
                    style={{
                      background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)",
                      border: "1px solid color-mix(in srgb, var(--gaea-glow) 18%, transparent)",
                    }}
                  >
                    {r.lastText && (
                      <span className="truncate" title={r.lastText} style={{ color: "var(--md-sys-color-text)" }}>
                        <span className="inline-block w-1 h-1 rounded-full mr-1.5 align-middle animate-pulse" style={{ background: "var(--gaea-glow)" }} />
                        正在：{r.lastText}
                      </span>
                    )}
                    {r.lastTool && (
                      <span className="truncate font-mono" title={r.lastTool} style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        ⚙ {r.lastTool}
                      </span>
                    )}
                  </div>
                )}

                {isOpen && r.answer && (
                  <div
                    className="mt-1 px-2 py-1.5 rounded-md text-[11px] leading-relaxed whitespace-pre-wrap break-words"
                    style={{ background: "color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)", color: "var(--md-sys-color-text)" }}
                  >
                    {r.answer}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
