import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bot, ClipboardList, Loader2, Users } from "../icons";
import { app } from "../lib/bridge";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../lib/types";
import { usePollingGate } from "../../hooks/usePollingGate";
import { useLiveReload } from "../hooks/useLiveReload";
import { loadSubagentAutoOpen } from "../lib/subagentPrefs";
import { AgentTree } from "./AgentTree";
import { TaskCenter } from "./TaskCenter";
import { useT } from "../lib/i18n";

// TasksWorkbench — 任务视图（对标 dsh-better-sidebar 的 SubagentView 同构页）。
//
// 与 better-sidebar 的差异刻意处理：
//  - 拓扑在前、后台任务在后，同一个滚动页；
//  - 子代理节点点击**不**在任务页内再开 transcript/轨迹式全面板（那与主区
//    「轨迹」重合）——AgentTree 只承担拓扑 + 实时预览；下钻跳转交由后续
//    「主会话切换」接线；
//  - 数据源两个单轮询并行（GaeaAgentNetwork + GaeaSubagentRuns），与
//    better-sidebar subagents.live「一次枚举整棵树」同思路；任务清单由
//    TaskCenter 自管理（GaeaTaskList + 事件）。
export function TasksWorkbench({
  sessionPath,
  onSubagentStarted,
  onOpenSubagent,
}: {
  sessionPath?: string;
  onSubagentStarted?: () => void;
  onOpenSubagent?: (p: {
    sessionPath: string;
    ref: string;
    task?: string;
    model?: string;
    status: "running" | "completed" | "failed";
  }) => void;
}) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork | null>(null);
  const [runsView, setRunsView] = useState<{ runs: SubagentRunView[]; running: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const gate = usePollingGate();
  const autoOpen = loadSubagentAutoOpen();
  const autoOpenRef = useRef(autoOpen);
  autoOpenRef.current = autoOpen;
  const onStartedRef = useRef(onSubagentStarted);
  onStartedRef.current = onSubagentStarted;
  const knownRefsRef = useRef<Set<string> | null>(null);

  const detectNewSubagents = useCallback((r: { runs: SubagentRunView[] }) => {
    const known = knownRefsRef.current;
    const refs = new Set(r.runs.map((x) => x.ref).filter((x): x is string => !!x));
    if (known) {
      const fresh = [...refs].filter((x) => !known.has(x));
      if (fresh.length > 0 && autoOpenRef.current) onStartedRef.current?.();
    }
    knownRefsRef.current = refs;
  }, []);

  const load = useCallback(() => {
    if (!sessionPath) {
      setNet(null);
      setRunsView(null);
      setLoading(false);
      return;
    }
    void Promise.all([
      app.AgentNetwork().catch(() => null),
      app.SubagentRuns(sessionPath).catch(() => null),
    ]).then(([n, r]) => {
      setNet(n);
      if (r) {
        const view = { runs: r.runs ?? [], running: r.running ?? 0 };
        setRunsView(view);
        detectNewSubagents(view);
      }
      setLoading(false);
    });
  }, [sessionPath, detectNewSubagents]);

  useEffect(() => {
    const tick = () => { if (gate) load(); };
    tick();
    if (!sessionPath) return;
    const timer = window.setInterval(tick, 5000);
    return () => window.clearInterval(timer);
  }, [load, sessionPath, gate]);

  const running = net?.root.status === "running" || (runsView?.running ?? 0) > 0;
  useLiveReload(running, load);

  const runs = useMemo(() => runsView?.runs ?? [], [runsView]);
  // 本地模型工具（vision/summarize_file 等）：与子代理同构的 mt_ 运行不在
  // AgentNetwork 树里（树只含派生子代理），这里以扁平区块展示同一数据源。
  const modelToolRuns = useMemo(
    () =>
      runs
        .filter((r) => r.kind === "model_tool")
        .slice()
        .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt)),
    [runs],
  );
  const hasContent =
    (net !== null && (net.root.children?.length ?? 0) > 0) || runs.length > 0;
  const runningCount = runsView?.running || runs.filter((r) => r.status === "running").length;

  // 点击节点 → 主区打开该子代理转录（better-sidebar openSubagent 语义；
  // 不在右栏任务页内开全面板，避免与主区轨迹内容重合）。
  const openThread = useCallback(
    (node: AgentNode, run: SubagentRunView | null) => {
      if (!sessionPath || !onOpenSubagent) return;
      const ref = run?.ref ?? (node.id.startsWith("sa_") ? node.id : null);
      if (!ref) return;
      onOpenSubagent({
        sessionPath,
        ref,
        task: node.task ?? run?.task,
        model: node.model ?? run?.model,
        status: run?.status === "running" ? "running" : node.status === "error" ? "failed" : run?.status === "failed" ? "failed" : "completed",
      });
    },
    [sessionPath, onOpenSubagent],
  );

  return (
    <div className="tasks-workbench flex flex-col h-full min-h-0 text-xs">
      <style>{`
        .tasks-workbench > .tasks-scroll > div [class*="h-full"] { height: auto !important; }
        .tasks-workbench > .tasks-scroll > div [class*="flex-1"] { flex: 0 0 auto !important; }
        .tasks-workbench > .tasks-scroll > div [class*="min-h-0"] { min-height: 0 !important; }
        .tasks-workbench > .tasks-scroll > div div[class*="overflow-y-auto"] { overflow: visible !important; }
        .tasks-workbench > .tasks-scroll > div > div { overflow: visible !important; }
      `}</style>
      <div className="v3-panel-head">
        <ClipboardList size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">任务管理</span>
        {runningCount > 0 && (
          <span className="rounded-full px-1.5 py-px text-[10px]" style={{ color: "var(--gaea-glow)" }}>
            {t("subagent.runningCount", { n: runningCount })}
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button
          type="button"
          aria-label="刷新"
          className="p-1 rounded bg-transparent cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={() => void load()}
        >
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      <div className="tasks-scroll flex-1 min-h-0 overflow-y-auto">
        {/* ① 子代理拓扑（better-sidebar：整树挂主 agent，running 实时预览） */}
        <div className="px-2 pt-2 pb-1 flex items-center gap-1.5 text-[10px] uppercase tracking-wider" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          <Users size={10} aria-hidden />
          子代理
        </div>
        {loading && !hasContent ? (
          <div className="flex items-center gap-2 px-4 py-6 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <Loader2 size={13} className="animate-spin" />
            {t("subagent.loading")}
          </div>
        ) : !hasContent ? (
          <div className="flex flex-col items-center gap-2 px-6 py-8 text-center">
            <Bot size={22} aria-hidden className="opacity-30" />
            <span className="text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              暂无子代理
              <br />
              当前主代理派生的子代理将显示在这里
            </span>
          </div>
        ) : net ? (
          <div className="px-1">
            <AgentTree network={net} runs={runs} onOpenThread={openThread} />
          </div>
        ) : null}

        {/* ①b 本地模型工具：单轮模型调用 = 变相子代理，同 UI 行 + 可开对话 tab */}
        {modelToolRuns.length > 0 && (
          <>
            <div
              className="px-2 pt-2.5 pb-1 flex items-center gap-1.5 text-[10px] uppercase tracking-wider"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
            >
              <Bot size={10} aria-hidden />
              {t("subagent.modelToolSection")}
              <span className="rounded-full px-1 font-mono" style={{ background: "var(--md-sys-color-surface-container-high)" }}>
                {modelToolRuns.length}
              </span>
            </div>
            <div className="flex flex-col gap-1 px-1 pb-1">
              {modelToolRuns.map((r) => {
                const statusLabel =
                  r.status === "running"
                    ? t("subagent.statusRunning")
                    : r.status === "failed"
                      ? t("subagent.statusFailed")
                      : t("subagent.statusDone");
                return (
                  <button
                    key={r.ref}
                    type="button"
                    data-model-tool-row={`${sessionPath}:${r.ref}`}
                    className="flex w-full items-start gap-2 rounded-md px-1.5 py-1.5 text-left bg-transparent border-0 cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                    onClick={() =>
                      sessionPath &&
                      onOpenSubagent?.({
                        sessionPath,
                        ref: r.ref,
                        task: r.task || r.tool || r.ref,
                        status: r.status === "running" ? "running" : r.status === "failed" ? "failed" : "completed",
                      })
                    }
                  >
                    <span
                      className={`mt-[4px] h-2 w-2 shrink-0 rounded-full ${
                        r.status === "running"
                          ? "bg-accent animate-pulse"
                          : r.status === "failed"
                            ? "bg-err"
                            : "bg-ok"
                      }`}
                    />
                    <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="truncate text-[12px] leading-snug" style={{ color: "var(--md-sys-color-text)" }}>
                        {r.task || r.tool || r.ref}
                      </span>
                      <span className="truncate text-[10px] leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        {statusLabel}
                        {" · "}
                        {t("subagent.modelToolLabel")}
                      </span>
                    </span>
                  </button>
                );
              })}
            </div>
          </>
        )}

        {/* ② 后台任务（同页滚动；TaskCenter 自带输出 dock/取消/重试） */}
        <div className="mt-1 border-t border-border-soft" style={{ paddingTop: 2 }}>
          <TaskCenter />
        </div>
      </div>
    </div>
  );
}
