import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bot, ClipboardList, Loader2, Users } from "../icons";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../lib/types";
import { useLiveReload } from "../hooks/useLiveReload";
import {
  reloadAgentNetwork,
  subscribeAgentNetwork,
  type AgentNetworkMeta,
} from "../lib/agentNetworkStore";
import {
  reloadSubagentRuns,
  subscribeSubagentRuns,
  type SubagentRunsMeta,
} from "../lib/subagentRunsStore";
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
//  - 数据源两个（v4.66 起全部入共享 store，本组件零自管定时器），与
//    better-sidebar subagents.live「一次枚举整树」同思路：
//     · GaeaSubagentRuns(sessionPath) → subagentRunsStore（按路径建册，与
//       SubagentsPanel / 左栏子行同源去重），快照 + loading/ready/error 随订阅
//       广播，失败给重试入口（reloadSubagentRuns），不再静默白板；
//     · GaeaAgentNetwork() → agentNetworkStore（绑定无参 → 全局单例轮询器，
//       会话切换显式 reload 补即时性），失败保留旧树 + 重试入口。
//    不可见门控（document.hidden 跳过 tick）与单在途收敛由 store 自带，本组件
//    不再自管 usePollingGate/interval。任务清单仍由 TaskCenter 自管理
//    （GaeaTaskList + 事件）。
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
  const [netMeta, setNetMeta] = useState<AgentNetworkMeta | null>(null);
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [runsMeta, setRunsMeta] = useState<SubagentRunsMeta | null>(null);
  const autoOpen = loadSubagentAutoOpen();
  const autoOpenRef = useRef(autoOpen);
  autoOpenRef.current = autoOpen;
  const onStartedRef = useRef(onSubagentStarted);
  onStartedRef.current = onSubagentStarted;
  // 已见过的子代理 ref 集合：null = 尚未建立基线（首次成功拉取只记基线不触发）
  const knownRefsRef = useRef<Set<string> | null>(null);
  // 双源的不可见门控均由各自 store tick 自带（document.hidden 跳过，与
  // usePollingGate 同语义），本组件不再自管门控。

  // 新子代理检测：本轮 refs（过滤空 ref）相对基线新出现 → 偏好开启时回调
  // （回调无参，一次快照出现多个新子代理合并为一次通知）。
  const detectNewSubagents = useCallback((nextRuns: SubagentRunView[]) => {
    const known = knownRefsRef.current;
    const refs = new Set(nextRuns.map((x) => x.ref).filter((x): x is string => !!x));
    if (known) {
      const fresh = [...refs].filter((x) => !known.has(x));
      if (fresh.length > 0 && autoOpenRef.current) onStartedRef.current?.();
    }
    knownRefsRef.current = refs;
  }, []);

  // runs：订阅共享 store（live 订阅——随 store 5s tick 实时刷新，不可见跳过）。
  // 新子代理检测只对成功快照（ready）做，与既有「成功拉取才检测」口径一致。
  useEffect(() => {
    if (!sessionPath) {
      setRuns([]);
      setRunsMeta(null);
      return;
    }
    return subscribeSubagentRuns(sessionPath, (nextRuns, meta) => {
      setRuns(nextRuns);
      setRunsMeta(meta);
      if (meta.status === "ready") detectNewSubagents(nextRuns);
    });
  }, [sessionPath, detectNewSubagents]);

  // 树拓扑（GaeaAgentNetwork）：订阅 agentNetworkStore 共享单轮询（绑定无参 →
  // 全局单例轮询器，不按路径建册；理由见 store 头注）。失败置 error 态给重试
  // 入口，快照保留旧树，不再静默降级成「无树」。无 sessionPath 不订阅（保持
  // 「空状态不请求」口径）。
  useEffect(() => {
    if (!sessionPath) {
      setNet(null);
      setNetMeta(null);
      return;
    }
    return subscribeAgentNetwork((nextNet, meta) => {
      setNet(nextNet);
      setNetMeta(meta);
    });
  }, [sessionPath]);

  // 会话切换立即重拉树：单例轮询器不随路径重建（无参绑定），显式 reload 补
  // 即时性；与订阅重建触发的重拉在途合并，至多一次请求。runs 按路径建册天然
  // 重建，无需此处接线。
  const prevPathRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    const prev = prevPathRef.current;
    prevPathRef.current = sessionPath;
    if (sessionPath && prev && prev !== sessionPath) reloadAgentNetwork();
  }, [sessionPath]);

  // 手动刷新 / 事件流刷新：双 store 显式重拉（各自在途合并为一次）。
  const refresh = useCallback(() => {
    reloadAgentNetwork();
    if (sessionPath) reloadSubagentRuns(sessionPath);
  }, [sessionPath]);

  // running 由数据派生：树根在跑或存在运行中分工 → useLiveReload 随事件节流
  // 刷新（turn_done 立即），内部走双 store reload。
  const running = net?.root.status === "running" || (runsMeta?.running ?? 0) > 0;
  useLiveReload(running, refresh);

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
  const runsLoading = !!sessionPath && (runsMeta?.status ?? "loading") === "loading";
  const netLoading = !!sessionPath && (netMeta?.status ?? "loading") === "loading";
  const loading = netLoading || runsLoading;
  const runsError = runsMeta?.status === "error";
  const netError = netMeta?.status === "error";
  const loadError = runsError || netError;
  const retryError = useCallback(() => {
    if (runsError) reloadSubagentRuns(sessionPath ?? "");
    else reloadAgentNetwork();
  }, [runsError, sessionPath]);
  const hasContent =
    (net !== null && (net.root.children?.length ?? 0) > 0) || runs.length > 0;
  const runningCount = runsMeta?.running || runs.filter((r) => r.status === "running").length;

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
          onClick={refresh}
        >
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      <div className="tasks-scroll flex-1 min-h-0 overflow-y-auto">
        {/* v4.66：有内容时失败以细条横幅呈现（重试走 store reload；树失败随 tick 自愈） */}
        {hasContent && loadError && (
          <div
            className="flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10.5px] mx-2 mt-2"
            style={{
              background: "color-mix(in srgb, var(--md-sys-color-error) 8%, transparent)",
              border: "1px solid color-mix(in srgb, var(--md-sys-color-error) 24%, transparent)",
              color: "var(--md-sys-color-error)",
            }}
          >
            <span className="min-w-0 flex-1 truncate">
              {t("subagent.runsLoadFail")}
            </span>
            <button
              type="button"
              data-testid="tasks-workbench-retry"
              className="shrink-0 cursor-pointer rounded-md border-0 bg-transparent text-[10.5px] font-medium underline-offset-2 hover:underline"
              onClick={retryError}
              style={{ color: "var(--md-sys-color-error)" }}
            >
              {t("subagent.retry")}
            </button>
          </div>
        )}
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
            {/* v4.66：拉取失败给「重试」入口，不用「暂无」空态冒充失败态（不得静默白板） */}
            {loadError ? (
              <>
                <span className="text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-error)" }}>
                  {t("subagent.runsLoadFail")}
                </span>
                <button
                  type="button"
                  data-testid="tasks-workbench-retry"
                  className="cursor-pointer rounded-md border px-2 py-0.5 text-[11px] transition-colors"
                  onClick={retryError}
                  style={{
                    color: "var(--md-sys-color-error)",
                    border: "1px solid color-mix(in srgb, var(--md-sys-color-error) 40%, transparent)",
                    background: "color-mix(in srgb, var(--md-sys-color-error) 8%, transparent)",
                  }}
                >
                  {t("subagent.retry")}
                </button>
              </>
            ) : (
              <span className="text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                暂无子代理
                <br />
                当前主代理派生的子代理将显示在这里
              </span>
            )}
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
