import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { ChevronDown, ChevronRight, MessageSquare } from "../icons";
import { useT, type Translator } from "../lib/i18n";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../lib/types";
import { fmtTokens } from "../lib/stats";

// AgentTree — 子代理树实时拓扑（v4.24 A1「分工面板工作台化」核心①）。
// 以 GaeaAgentNetwork 的嵌套 Children 渲染整棵子代理树（此前 AgentNetworkCard
// 只画两层，深层子树数据被丢弃）：
//  - root 折叠为「主 agent」行；一级子代理始终可见，更深层默认收起、可展开/收起；
//  - 新节点出现自动展开其父链（本次挂载生命周期内未见过的节点 id 即「新」，
//    首轮轮询只记基线不展开）；
//  - 节点行量化：状态色点（running 呼吸动画）/任务摘要/工具调用数/模型徽标/
//    耗时（running 显示实时已用时——组件内 1s tick；已完成显示总耗时）/
//    token 数（有则显）/错误数；
//  - 运行节点用 SubagentRuns 的 lastText/lastTool 富化为实时预览行
//    （按 ref 直等或任务摘要前缀双向匹配，与后端 enrichAgentNetwork 同口径；
//    匹配失败降级纯节点统计）；
//  - 下钻链（v4.27 对齐 Codex）：子代理节点点击 → onOpenThread 回调，由面板
//    层切换到「子代理对话」全面板视图（SubagentThread：完整 transcript +
//    实时刷新），不再在树内塞窄小内嵌卡。

// 树展平条目：ancestors 供「新节点自动展开父链」用（含 root id，不含自身）。
interface FlatNode {
  node: AgentNode;
  depth: number;
  ancestors: string[];
}

function flattenTree(node: AgentNode, depth: number, ancestors: string[], out: FlatNode[]): void {
  out.push({ node, depth, ancestors });
  for (const child of node.children ?? []) {
    flattenTree(child, depth + 1, [...ancestors, node.id], out);
  }
}

// 节点 → 分工 meta 匹配：先 ref 直等（后端若以 ref 作节点 id），再任务摘要
// 前缀双向（与 AgentNetworkCard.selectNode / 后端富化同口径）。
function matchRunForNode(node: AgentNode, runs: SubagentRunView[]): SubagentRunView | null {
  for (const r of runs) {
    if (r.ref && node.id && r.ref === node.id) return r;
  }
  for (const r of runs) {
    if (node.task && r.task && (r.task.startsWith(node.task) || node.task.startsWith(r.task))) return r;
  }
  return null;
}

function statusColor(status: AgentNode["status"]): string {
  switch (status) {
    case "running":
      return "var(--gaea-glow)";
    case "error":
      return "var(--md-sys-color-destructive)";
    default:
      return "var(--md-sys-color-success)";
  }
}

function statusText(status: AgentNode["status"], t: Translator): string {
  switch (status) {
    case "running":
      return t("subagent.statusRunning");
    case "error":
      return t("subagent.statusError");
    default:
      return t("subagent.statusDone");
  }
}

// 耗时格式化（与 SubagentsPanel.fmtDuration 同语义：秒/分/小时三档）。
function fmtDur(ms: number, t: Translator): string {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  const s = Math.max(1, Math.round(ms / 1000));
  if (s < 60) return t("subagent.durSec", { n: s });
  const m = Math.floor(s / 60);
  if (m < 60) return t("subagent.durMin", { n: m });
  return t("subagent.durHour", { n: Math.floor(m / 60) });
}

// 节点耗时：优先节点 firstTs/lastTs（后端权威），缺省回落匹配 run 的
// createdAt/updatedAt（ISO 串前端可算）。running 显示实时已用时（live=true
// 走 1s tick），已完成/出错显示总耗时。
function nodeDuration(node: AgentNode, run: SubagentRunView | null, nowMs: number, t: Translator): { label: string; live: boolean } {
  const nodeStart = node.firstTs !== undefined ? node.firstTs * 1000 : NaN;
  const runStart = run ? Date.parse(run.createdAt) : NaN;
  const start = Number.isFinite(nodeStart) ? nodeStart : runStart;
  if (node.status === "running") {
    if (!Number.isFinite(start)) return { label: "", live: false };
    return { label: t("subagent.elapsed", { dur: fmtDur(nowMs - start, t) }), live: true };
  }
  const nodeEnd = node.lastTs !== undefined ? node.lastTs * 1000 : NaN;
  const runEnd = run ? Date.parse(run.updatedAt) : NaN;
  const end = Number.isFinite(nodeEnd) ? nodeEnd : runEnd;
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return { label: "", live: false };
  return { label: fmtDur(end - start, t), live: false };
}

const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// ─── 子代理树 ──────────────────────────────────────────────
export function AgentTree({ network, runs, onOpenThread }: {
  network: AgentNetwork;
  runs: SubagentRunView[];
  /** 子代理节点点击 → 面板层切换「子代理对话」全面板视图（v4.27 Codex 式）。 */
  onOpenThread: (node: AgentNode, run: SubagentRunView | null) => void;
}) {
  const t = useT();
  const root = network.root;

  // 展平一次供逻辑用（匹配表/新节点检测/hasRunning），渲染走递归保结构。
  const flat = useMemo(() => {
    const out: FlatNode[] = [];
    flattenTree(root, 0, [], out);
    return out;
  }, [root]);

  // 节点 → 分工 meta 匹配表：单轮轮询算全表，避免逐节点重复扫描。
  const runByNodeId = useMemo(() => {
    const map = new Map<string, SubagentRunView | null>();
    for (const f of flat) map.set(f.node.id, matchRunForNode(f.node, runs));
    return map;
  }, [flat, runs]);

  // 展开态：root 默认展开（直接子级可见）；点根卡/箭头可整树收起。
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set([root.id]));
  const toggleNode = useCallback((id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  // 新节点出现自动展开其父链：本次挂载未见过的节点 id 即「新」；
  // 首轮轮询只记基线不展开（挂载时已存在的子代理不算新）。
  const seenIdsRef = useRef<Set<string> | null>(null);
  useEffect(() => {
    const ids = new Set(flat.map((f) => f.node.id));
    const seen = seenIdsRef.current;
    seenIdsRef.current = ids;
    if (!seen) return;
    const fresh = flat.filter((f) => !seen.has(f.node.id));
    if (fresh.length === 0) return;
    setExpanded((prev) => {
      const next = new Set(prev);
      for (const f of fresh) {
        for (const a of f.ancestors) next.add(a);
      }
      return next;
    });
  }, [flat]);

  // running 节点存在时启动 1s tick：实时已用时走秒（无运行节点零开销）。
  const hasRunning = useMemo(() => flat.some((f) => f.node.status === "running"), [flat]);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!hasRunning) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [hasRunning]);

  const renderNode = (f: FlatNode): ReactNode => {
    const node = f.node;
    const run = runByNodeId.get(node.id) ?? null;
    const dur = nodeDuration(node, run, now, t);
    const title = node.kind === "root" ? t("subagent.rootTitle") : node.task || node.name;
    const childEntries = node.children ?? [];
    const hasChildren = childEntries.length > 0;
    const isExpanded = expanded.has(node.id);
    // v4.75 卡片感：表面抬到 surface-container-high，运行/出错用语义描边
    const cardSurface =
      node.kind === "root"
        ? "color-mix(in srgb, var(--md-sys-color-primary) 8%, var(--md-sys-color-surface-container-high))"
        : "var(--md-sys-color-surface-container-high)";
    const cardBorder =
      node.status === "running"
        ? "1px solid color-mix(in srgb, var(--gaea-glow) 38%, transparent)"
        : node.status === "error"
          ? "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 45%, transparent)"
          : "1px solid var(--md-sys-color-outline-variant)";
    // transcript 引用：优先匹配 run 的 ref；后端若以 ref 作节点 id 也可直用。
    const transcriptRef = run?.ref ?? (node.id.startsWith("sa_") ? node.id : null);
    // 子代理节点可打开对话（root 无 transcript，不响应）
    const canOpenThread = node.kind === "subagent" && transcriptRef !== null;
    // v4.76：点卡片头部 = 折叠/展开（有子节点时优先）；打开对话收敛到独立按钮
    const onHeaderClick = () => {
      if (hasChildren) toggleNode(node.id);
      else if (canOpenThread) onOpenThread(node, run);
    };
    const dotCls = `inline-block h-2 w-2 shrink-0 rounded-full${node.status === "running" ? " animate-pulse" : ""}`;
    return (
      <div
        key={node.id}
        data-testid="agent-node"
        data-node-id={node.id}
        className="flex flex-col gap-1 rounded-[8px] p-1.5 transition-all duration-200"
        style={{
          background: cardSurface,
          border: cardBorder,
        }}
      >
        <div className="flex items-start gap-1">
          {hasChildren ? (
            <button
              type="button"
              className={iconBtn}
              data-testid="agent-node-toggle"
              aria-label={`${isExpanded ? t("common.collapse") : t("common.expand")} ${title}`}
              aria-expanded={isExpanded}
              onClick={() => toggleNode(node.id)}
            >
              {isExpanded ? <ChevronDown size={12} aria-hidden /> : <ChevronRight size={12} aria-hidden />}
            </button>
          ) : (
            <span className="w-6 shrink-0" aria-hidden />
          )}
          <button
            type="button"
            data-testid="agent-node-row"
            onClick={onHeaderClick}
            className={`flex min-w-0 flex-1 items-start gap-1.5 border-0 bg-transparent p-0 text-left ${canOpenThread || hasChildren ? "cursor-pointer" : "cursor-default"}`}
            title={hasChildren ? `${isExpanded ? t("common.collapse") : t("common.expand")} ${title}` : `${statusText(node.status, t)} · ${title}`}
          >
            <span className={`${dotCls} mt-[5px]`} aria-hidden style={{ background: statusColor(node.status) }} />
            <div className="min-w-0 flex-1">
              {/* 卡片标题行：任务名 + 状态胶囊 */}
              <div className="flex items-center gap-1.5">
                <span className="min-w-0 flex-1 truncate text-[12.5px] font-medium leading-snug" style={{ color: "var(--md-sys-color-text)" }}>
                  {title}
                </span>
                <span
                  className="shrink-0 rounded-full px-1.5 py-px text-[9.5px] font-medium leading-relaxed"
                  style={
                    node.status === "running"
                      ? { color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)", border: "1px solid color-mix(in srgb, var(--gaea-glow) 22%, transparent)" }
                      : node.status === "error"
                        ? { color: "var(--md-sys-color-destructive)", background: "color-mix(in srgb, var(--md-sys-color-destructive) 10%, transparent)" }
                        : { color: "var(--md-sys-color-success)", background: "color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)" }
                  }
                >
                  {statusText(node.status, t)}
                </span>
              </div>
              {/* 指标行：模型 / 工具调用 / 耗时 / tokens / 错误 */}
              <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[10px] leading-none" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {(node.model || run?.model) && (
                  <span className="rounded bg-(color:--md-sys-color-surface-container-high) px-1 py-px">{node.model || run?.model}</span>
                )}
                <span>{`⚙${node.toolCalls}`}</span>
                {dur.label && (
                  <span style={{ color: dur.live ? "var(--gaea-glow)" : "var(--md-sys-color-text-secondary)" }}>
                    {dur.label}
                  </span>
                )}
                {node.tokens > 0 && <span>≈{fmtTokens(node.tokens)}</span>}
                {node.errors > 0 && (
                  <span style={{ color: "var(--md-sys-color-destructive)" }}>{t("subagent.errCount", { n: node.errors })}</span>
                )}
              </div>
              {/* 非运行态：完成/失败展示分工回答摘要（无则不占行） */}
              {node.status !== "running" && run?.answer && (
                <div
                  className="mt-1 truncate rounded-md px-1.5 py-1 text-[10px] leading-snug"
                  style={{ color: "var(--md-sys-color-text-secondary)", background: "var(--md-sys-color-surface-container-high)" }}
                  title={run.answer}
                >
                  {run.answer}
                </div>
              )}
            </div>
          </button>
          {canOpenThread && (
            <button
              type="button"
              data-testid="agent-node-open"
              aria-label={t("subagent.openThread", { title })}
              title={t("subagent.threadHint")}
              className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md border-0 bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
              onClick={() => onOpenThread(node, run)}
            >
              <MessageSquare size={12} aria-hidden />
            </button>
          )}
        </div>

        {/* C2 活动行迁入树内：运行节点的实时预览（匹配失败时整块省略 = 纯节点统计降级） */}
        {node.status === "running" && run && (run.lastText || run.lastTool) && (
          <div
            className="flex flex-col gap-0.5 rounded-md px-2 py-1.5 text-[10.5px] leading-relaxed"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 16%, transparent)",
            }}
          >
            {run.lastText && (
              <span className="truncate" title={run.lastText} style={{ color: "var(--md-sys-color-text)" }}>
                <span className="mr-1 inline-block h-1 w-1 rounded-full align-middle animate-pulse" style={{ background: "var(--gaea-glow)" }} aria-hidden />
                {t("subagent.doing", { text: run.lastText })}
              </span>
            )}
            {run.lastTool && (
              <span className="truncate font-mono" title={run.lastTool} style={{ color: "var(--md-sys-color-text-secondary)" }}>
                ⚙ {run.lastTool}
              </span>
            )}
          </div>
        )}

        {/* 子树：按展开态渲染；左侧树形线 */}
        {isExpanded && childEntries.length > 0 && (
          <div
            className="ml-1.5 flex flex-col gap-2 border-l pl-2.5"
            style={{ borderColor: "color-mix(in srgb, var(--md-sys-color-text-secondary) 45%, transparent)" }}
          >
            {childEntries.map((child) =>
              renderNode({ node: child, depth: f.depth + 1, ancestors: [...f.ancestors, node.id] }),
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="flex flex-col gap-2" data-testid="agent-tree">
      {renderNode({ node: root, depth: 0, ancestors: [] })}
    </div>
  );
}
