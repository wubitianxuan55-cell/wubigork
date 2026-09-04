import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Bot, Loader2, Users } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../lib/types";
import { usePollingGate } from "../../hooks/usePollingGate";
import { useLiveReload } from "../hooks/useLiveReload";
import {
  reloadSubagentRuns,
  subscribeSubagentRuns,
  type SubagentRunsMeta,
} from "../lib/subagentRunsStore";
import { loadSubagentAutoOpen, saveSubagentAutoOpen } from "../lib/subagentPrefs";
import { AgentTree } from "./AgentTree";
import { SubagentThread, type SubagentThreadStatus } from "./SubagentThread";

// SubagentsPanel — 子代理工作台（v4.24 A1「分工面板工作台化」，对标
// dsh-better-sidebar 任务页）。在 v3「谁在干什么」扁平卡片之上重造为三段式：
//  ①合并活动流（Devin 式单列 feed）：所有 running 子代理的 lastText/lastTool
//    按 updatedAt 倒序合并、上限 20 条、空态收起——面板顶部一眼看清"此刻"；
//  ②树形实时拓扑：整棵子代理树（AgentTree 组件，GaeaAgentNetwork 嵌套
//    Children，此前只渲染两层）+ 节点量化 + 新节点自动展开父链 + 下钻链
//    （节点 → 详情 → 完整 transcript → 工具行点击定位）；
//  ③新子代理自动展开（可关，默认开）：检测到新子代理 ref 出现时调用
//    props.onSubagentStarted 回调——面板只负责检测 + 回调，是否切换 tab/
//    亮出面板由 App 接线决定（偏好键 gaea.subagentAutoOpen 持久化）。
// 数据源两个（v4.64 起分轨）：
//  - GaeaSubagentRuns(sessionPath)：分工 meta + lastText/lastTool 实时预览
//    ——迁 subagentRunsStore 共享单轮询（与 App 会话 tab 同源去重，本组件
//    不再自管定时器），快照 + loading/ready/error 状态随订阅广播，失败给
//    重试入口（reloadSubagentRuns），不再静默白板；
//  - GaeaAgentNetwork()：树拓扑（节点/嵌套子树/token 富化）——store 只管
//    runs，本组件保留自己的 5s 轮询（不可见门控）。
// 两源按「ref 直等 → 任务摘要前缀双向」匹配（与后端 enrichAgentNetwork 同口径）。
// 刷新节奏：runs 随 store tick（不可见门控）；树 5s 轮询 + useLiveReload
// （turn_done 立即、运行中随事件节流；running 由数据派生：树根 running 或
// 存在运行中分工）。

// 活动流上限：超过后只保留最新 20 条（Devin feed 同款截断）。
const FEED_LIMIT = 20;

function safeTime(iso: string): number {
  const t = Date.parse(iso);
  return Number.isFinite(t) ? t : 0;
}

// 活动流行前缀：任务摘要截短（无摘要回退 ref 尾段），完整内容在 title。
function feedName(run: SubagentRunView): string {
  const base = run.task || run.ref;
  return base.length > 12 ? `${base.slice(0, 12)}…` : base;
}

function ActivityFeed({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  // 合并单列：只收 running 的最新动态行，updatedAt 倒序，上限 20 条；
  // 空（无运行中子代理）整体收起，不占面板空间。
  const items = useMemo(
    () =>
      runs
        .filter((r) => r.status === "running" && (r.lastText || r.lastTool))
        .slice()
        .sort((a, b) => safeTime(b.updatedAt) - safeTime(a.updatedAt))
        .slice(0, FEED_LIMIT),
    [runs],
  );
  if (items.length === 0) return null;
  return (
    <div className="flex flex-col gap-1" data-testid="agent-feed">
      <div className="flex items-center gap-1.5 px-0.5 text-[10px] font-medium" style={{ color: "var(--gaea-glow)" }}>
        <span className="inline-block h-1.5 w-1.5 rounded-full animate-pulse" style={{ background: "var(--gaea-glow)" }} aria-hidden />
        {t("subagent.feedTitle")}
        <span className="font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>{items.length}</span>
      </div>
      {items.map((r) => (
        <div
          key={r.ref}
          data-testid="agent-feed-row"
          className="flex flex-col gap-px rounded-md px-1.5 py-1 text-[10.5px] leading-relaxed"
          style={{
            background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)",
            border: "1px solid color-mix(in srgb, var(--gaea-glow) 16%, transparent)",
          }}
        >
          <span className="truncate" title={r.lastText ?? r.lastTool ?? r.task} style={{ color: "var(--md-sys-color-text)" }}>
            <span className="mr-1 inline-block h-1 w-1 rounded-full align-middle animate-pulse" style={{ background: "var(--gaea-glow)" }} aria-hidden />
            <span className="font-medium">{feedName(r)}</span>
            {r.lastText ? t("subagent.feedDoing", { text: r.lastText }) : t("subagent.feedTooling")}
          </span>
          {r.lastTool && (
            <span className="truncate pl-3 font-mono" title={r.lastTool} style={{ color: "var(--md-sys-color-text-secondary)" }}>
              ⚙ {r.lastTool}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

export function SubagentsPanel({ sessionPath, onSubagentStarted }: {
  sessionPath?: string;
  /** 检测到新子代理 ref 出现（且「自动展开」偏好为开）时回调；App 据此亮出分工面板。 */
  onSubagentStarted?: () => void;
}) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork | null>(null);
  const [netLoading, setNetLoading] = useState(true);
  const [netError, setNetError] = useState<string | null>(null);
  // v4.64：runs 迁共享单轮询 store——快照 + 状态（loading/ready/error）随订阅
  // 广播；本组件不再自管 GaeaSubagentRuns 定时器。meta 为 null 且有会话 = 首拉在途。
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [runsMeta, setRunsMeta] = useState<SubagentRunsMeta | null>(null);
  // v4.27 打开的子代理对话（Codex 式：点击子代理 → 右侧全面板实时对话）。
  // 只存身份快照；running/状态由 runs 轮询实时派生（liveThread）。
  const [thread, setThread] = useState<{
    ref: string;
    task?: string;
    model?: string;
  } | null>(null);
  // 新子代理自动展开偏好（默认开，localStorage 持久化，键 gaea.subagentAutoOpen）
  const [autoOpen, setAutoOpen] = useState(() => loadSubagentAutoOpen());
  const autoOpenRef = useRef(autoOpen);
  autoOpenRef.current = autoOpen;
  const onStartedRef = useRef(onSubagentStarted);
  onStartedRef.current = onSubagentStarted;
  // 已见过的子代理 ref 集合：null = 尚未建立基线（首次成功拉取只记基线不触发）
  const knownRefsRef = useRef<Set<string> | null>(null);
  // v4.5.2：树拓扑轮询接入系统级后台轮询门控（页面不可见时空转零成本）；
  // runs 的不可见门控由 store tick 自带（同语义）。
  const gate = usePollingGate();

  // 新子代理检测：本轮 refs 相对基线新出现的子代理视为「新」；偏好开启时回调
  // （回调无参，一次轮询出现多个新子代理合并为一次通知）。
  const detectNewSubagents = useCallback((nextRuns: SubagentRunView[]) => {
    const known = knownRefsRef.current;
    const refs = nextRuns.map((x) => x.ref);
    if (known) {
      const fresh = refs.filter((x) => !known.has(x));
      if (fresh.length > 0 && autoOpenRef.current) onStartedRef.current?.();
    }
    knownRefsRef.current = new Set(refs);
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

  // 树拓扑（GaeaAgentNetwork）：store 只收敛 runs，树仍由本组件拉取。
  // 单源失败置 error 态给重试入口，不再静默降级成「无树」。
  const loadNet = useCallback(() => {
    if (!sessionPath) {
      setNet(null);
      setNetLoading(false);
      return;
    }
    setNetLoading(true);
    void app
      .AgentNetwork()
      .then((n) => {
        setNet(n);
        setNetError(null);
      })
      .catch((e) => {
        setNetError(String(e?.message ?? e ?? "error"));
      })
      .finally(() => setNetLoading(false));
  }, [sessionPath]);

  // 会话切换重新拉取树；5s 轮询（不可见门控）+ 事件流刷新（turn_done 立即）。
  useEffect(() => {
    const tick = () => { if (gate) loadNet() };
    tick();
    if (!sessionPath) return;
    const timer = window.setInterval(tick, 5000);
    return () => window.clearInterval(timer);
  }, [loadNet, sessionPath, gate]);

  // 手动刷新 / 事件流刷新：树直拉 + runs 走 store 显式重拉（在途合并为一次）。
  const refresh = useCallback(() => {
    loadNet();
    if (sessionPath) reloadSubagentRuns(sessionPath);
  }, [loadNet, sessionPath]);

  // running 由数据派生：树根在跑或存在运行中分工 → 随事件节流刷新。
  const running = net?.root.status === "running" || (runsMeta?.running ?? 0) > 0;
  useLiveReload(running, refresh);

  const toggleAutoOpen = useCallback(() => {
    const next = !autoOpenRef.current;
    autoOpenRef.current = next;
    saveSubagentAutoOpen(next);
    setAutoOpen(next);
  }, []);

  const runsLoading = !!sessionPath && (runsMeta?.status ?? "loading") === "loading";
  const loading = netLoading || runsLoading;
  const runsError = runsMeta?.status === "error";
  const loadError = runsError || netError !== null;
  const retryError = useCallback(() => {
    if (runsError) reloadSubagentRuns(sessionPath ?? "");
    else loadNet();
  }, [runsError, sessionPath, loadNet]);
  const runningRuns = useMemo(() => runs.filter((r) => r.status === "running"), [runs]);
  const runningCount = runsMeta?.running || runningRuns.length;
  const hasRuns = runs.length > 0;
  const hasTree = net !== null && (net.root.children?.length ?? 0) > 0;
  const hasContent = hasRuns || hasTree;

  // 打开子代理对话：ref 直等命中 run 则带实时状态/模型；无 run（历史节点）用节点字段。
  const openThread = useCallback((node: AgentNode, run: SubagentRunView | null) => {
    const ref = run?.ref ?? (node.id.startsWith("sa_") ? node.id : null);
    if (!ref) return;
    setThread({
      ref,
      task: node.task ?? run?.task,
      model: node.model ?? run?.model,
    });
  }, []);

  // 对话视图实时派生：runs 轮询每 5s 刷新，ref 命中则状态/模型跟随最新。
  const liveThread = useMemo(() => {
    if (!thread) return null;
    const run = runs.find((r) => r.ref === thread.ref);
    const status: SubagentThreadStatus =
      run?.status === "running" ? "running"
        : run?.status === "failed" ? "failed"
          : run ? "completed" : "completed";
    return { ...thread, status, model: run?.model ?? thread.model };
  }, [thread, runs]);

  const iconBtn =
    "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {liveThread && sessionPath ? (
        <SubagentThread
          key={liveThread.ref}
          sessionPath={sessionPath}
          target={liveThread.ref}
          task={liveThread.task}
          status={liveThread.status}
          model={liveThread.model}
          onBack={() => setThread(null)}
        />
      ) : (
        <>
      {/* v3 细条头部：标题 + 计数徽标 + 自动展开胶囊开关 + 刷新 */}
      <div className="v3-panel-head">
        <Users size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">{t("subagent.title")}</span>
        {hasRuns && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {runs.length}
          </span>
        )}
        {runningCount > 0 && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
              color: "var(--md-sys-color-warning)",
              border: "1px solid color-mix(in srgb, var(--md-sys-color-warning) 32%, transparent)",
            }}
          >
            {t("subagent.runningCount", { n: runningCount })}
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button
          type="button"
          data-testid="subagent-auto-open-toggle"
          className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-full px-1.5 py-px text-[10px] leading-none transition-colors"
          aria-pressed={autoOpen}
          title={autoOpen
            ? t("subagent.autoOpenOnTitle")
            : t("subagent.autoOpenOffTitle")}
          onClick={toggleAutoOpen}
          style={autoOpen
            ? {
                background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)",
                color: "var(--gaea-glow)",
                border: "1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)",
              }
            : {
                background: "transparent",
                color: "var(--md-sys-color-text-secondary)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
        >
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{ background: autoOpen ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
            aria-hidden
          />
          {t("subagent.autoOpen", { state: autoOpen ? t("subagent.on") : t("subagent.off") })}
        </button>
        <button type="button" className={iconBtn} onClick={refresh} title={t("subagent.refreshTitle")} aria-label={t("subagent.refreshTitle")}>
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      {loading && !hasContent ? (
        <div className="flex items-center justify-center flex-1 gap-2 text-[11px]">
          <Loader2 size={14} className="animate-spin" />
          {t("subagent.loading")}
        </div>
      ) : !hasContent ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center">
          <Bot size={24} aria-hidden className="opacity-40" />
          {/* v4.64：拉取失败给「重试」入口，不用「暂无」空态冒充失败态（不得静默白板） */}
          {loadError ? (
            <>
              <span className="text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-error)" }}>
                {runsError ? t("sidebar.subagentsError") : t("subagent.netLoadFail", { msg: netError ?? "" })}
              </span>
              <button
                type="button"
                data-testid="subagent-retry"
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
            <span className="text-[11px] leading-relaxed">
              {runsMeta?.available === false || !sessionPath
                ? t("subagent.emptyNone")
                : t("subagent.emptyNoRecord")}
              <br />
              {t("subagent.emptyHint")}
            </span>
          )}
        </div>
      ) : (
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto min-h-0 p-2">
          {/* v4.64：有内容时失败以细条横幅呈现（runs 失败点重试走 store reload；树失败自动随 tick 自愈） */}
          {loadError && (
            <div
              className="flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10.5px]"
              style={{
                background: "color-mix(in srgb, var(--md-sys-color-error) 8%, transparent)",
                border: "1px solid color-mix(in srgb, var(--md-sys-color-error) 24%, transparent)",
                color: "var(--md-sys-color-error)",
              }}
            >
              <span className="min-w-0 flex-1 truncate">
                {runsError ? t("sidebar.subagentsError") : t("subagent.netLoadFail", { msg: netError ?? "" })}
              </span>
              <button
                type="button"
                data-testid="subagent-retry"
                className="shrink-0 cursor-pointer rounded-md border-0 bg-transparent text-[10.5px] font-medium underline-offset-2 hover:underline"
                onClick={retryError}
                style={{ color: "var(--md-sys-color-error)" }}
              >
                {t("subagent.retry")}
              </button>
            </div>
          )}
          {/* ① 合并活动流：running 子代理最新动态单列 feed（空态收起） */}
          <ActivityFeed runs={runningRuns} />
          {/* ② 树形实时拓扑：GaeaAgentNetwork 嵌套 Children 全量渲染 + 下钻链 */}
          {net && <AgentTree network={net} runs={runs} onOpenThread={openThread} />}
        </div>
      )}
        </>
      )}
    </div>
  );
}
