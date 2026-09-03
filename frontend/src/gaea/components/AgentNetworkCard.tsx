import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import type { AgentNetwork, AgentNode, SubagentRunsView, SubagentRunView, SubagentTranscriptView } from "../lib/types";
import { fmtTokens } from "../lib/stats";
import { useLiveReload } from "../hooks/useLiveReload";

const EMPTY: AgentNetwork = { ok: true, window: 0, root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0 } };

// 每个一级子树一个色相（dsh-context agent network 同语义）。
const HUES = ["#a855f7", "#06b6d4", "#f59e0b", "#22c55e", "#3b82f6", "#ec4899"];

const RING_R = 24;
const NODE_R = 18;
const CIRC = 2 * Math.PI * RING_R;

function nodeStatusColor(status: AgentNode["status"]): string {
  switch (status) {
    case "running": return "#22c55e";
    case "error": return "#ef4444";
    default: return "#64748b";
  }
}

// ─── 单节点（环 = token 占比，中心 = 工具调用数，悬停显示详情） ───
function AgentCircle({ node, pct, x, y, hue, onHover, onLeave, onSelect }: {
  node: AgentNode;
  pct: number;
  x: number;
  y: number;
  hue: string;
  onHover: (n: AgentNode) => void;
  onLeave: () => void;
  onSelect: (n: AgentNode) => void;
}) {
  const t = useT();
  const statusColor = nodeStatusColor(node.status);
  return (
    <g
      className="cursor-pointer"
      onMouseEnter={() => onHover(node)}
      onMouseLeave={onLeave}
      onClick={() => onSelect(node)}
      transform={`translate(${x}, ${y})`}
    >
      {/* 背景盘 */}
      <circle r={RING_R + 3} fill="var(--bg-soft)" stroke="var(--border-soft)" strokeWidth={1} />
      {/* token 占比环 */}
      <circle
        r={RING_R}
        fill="none"
        stroke={hue}
        strokeWidth={4}
        strokeDasharray={`${(pct / 100) * CIRC} ${CIRC}`}
        transform="rotate(-90)"
        strokeLinecap="round"
        opacity={0.9}
      />
      {/* 节点主体：running 呼吸绿 */}
      <circle
        r={NODE_R}
        fill="var(--bg)"
        stroke={statusColor}
        strokeWidth={2}
        className={node.status === "running" ? "animate-pulse" : ""}
      />
      <text textAnchor="middle" dominantBaseline="central" fontSize={9} fontWeight={600} fill="var(--fg)">
        {node.kind === "root" ? t("subagent.rootGlyph") : String(node.toolCalls)}
      </text>
    </g>
  );
}

// ─── Agent 网络卡 ──────────────────────────────────────────
export function AgentNetworkCard({ running, sessionPath }: { running: boolean; sessionPath?: string }) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork>(EMPTY);
  const [hovered, setHovered] = useState<AgentNode | null>(null);
  const [detail, setDetail] = useState<AgentNode | null>(null);
  const [runDetail, setRunDetail] = useState<SubagentRunView | null>(null);
  const [transcript, setTranscript] = useState<SubagentTranscriptView | null>(null);
  const [transcriptOpen, setTranscriptOpen] = useState(false);
  const [transcriptQuery, setTranscriptQuery] = useState("");
  const transcriptListRef = useRef<HTMLDivElement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const runsCacheRef = useRef<SubagentRunsView | null>(null);

  const load = useCallback(() => {
    app.AgentNetwork()
      .then((n) => { setNet(n); setError(null); })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(() => { load(); }, [load]);
  // 运行中随事件流节流刷新 + 回合结束立即刷新（useLiveReload）。
  useLiveReload(running, load);

  // 点击子代理节点：拉取当前会话的分工 meta，按任务摘要前缀匹配（与后端
  // enrichAgentNetwork 同口径），固定展示该子代理详情（回答/活动行/统计）。
  const selectNode = useCallback((node: AgentNode) => {
    setDetail(node);
    setRunDetail(null);
    if (node.kind !== "subagent" || !sessionPath) return;
    const match = (runs: SubagentRunsView) => {
      const hit = runs.runs.find((r) =>
        node.task && r.task && (r.task.startsWith(node.task) || node.task.startsWith(r.task)),
      ) ?? null;
      setRunDetail(hit);
    };
    if (runsCacheRef.current) {
      match(runsCacheRef.current);
      return;
    }
    app.SubagentRuns(sessionPath)
      .then((v) => { runsCacheRef.current = v; match(v); })
      .catch(() => setRunDetail(null));
  }, [sessionPath]);

  // 查看完整 transcript：点击后拉取该子代理的 transcript JSONL 投影。
  const viewTranscript = useCallback((ref: string) => {
    if (!sessionPath) return;
    if (transcriptOpen && transcript?.ref === ref) {
      setTranscriptOpen(false);
      return;
    }
    app.SubagentTranscript(sessionPath, ref)
      .then((v) => { setTranscript(v); setTranscriptOpen(true); })
      .catch(() => setTranscriptOpen(false));
  }, [sessionPath, transcript, transcriptOpen]);

  const shownTranscriptMsgs = useMemo(() => {
    if (!transcript) return [];
    const q = transcriptQuery.trim().toLowerCase();
    const all = transcript.messages.map((m, idx) => ({ m, idx }));
    if (!q) return all;
    return all.filter(({ m }) =>
      [m.content, m.reasoning, m.name, m.toolCallId, ...(m.toolCalls ?? []).map((tc) => `${tc.name} ${tc.arguments}`)]
        .filter(Boolean).join("\n").toLowerCase().includes(q),
    );
  }, [transcript, transcriptQuery]);

  // 搜索命中自动定位：命中变化时把第一条命中消息滚进视口。
  useEffect(() => {
    if (!transcriptOpen || !transcriptQuery.trim()) return;
    transcriptListRef.current?.querySelector('[data-msg-hit="true"]')?.scrollIntoView?.({ behavior: "smooth", block: "start" });
  }, [transcriptOpen, transcriptQuery, shownTranscriptMsgs]);

  const root = net.root;
  const children = root.children ?? [];
  const rootTokens = root.tokens || 1;
  const W = Math.max(320, 120 + children.length * 100);
  const childY = 130;

  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">{t("subagent.netTitle")}</div>
        <div className="text-[9px] text-fg-faint">{t("subagent.netSubtitle", { n: children.length })}</div>
      </div>
      {error && <div className="mt-1 text-[10px] text-err">{t("subagent.netLoadFail", { msg: error })}</div>}
      <svg viewBox={`0 0 ${W} 170`} className="mt-1 w-full h-auto">
        {/* 边：root → 子代理（每个一级子树一个色相） */}
        {children.map((c, i) => {
          const x = W / 2 + (i - (children.length - 1) / 2) * 100;
          return (
            <line
              key={c.id}
              x1={W / 2}
              y1={58}
              x2={x}
              y2={childY - RING_R - 4}
              stroke={HUES[i % HUES.length]}
              strokeWidth={1.5}
              opacity={0.6}
              className={c.status === "running" ? "animate-pulse" : ""}
            />
          );
        })}
        <AgentCircle
          node={root}
          pct={100}
          x={W / 2}
          y={38}
          hue="#818cf8"
          onHover={setHovered}
          onLeave={() => setHovered(null)}
          onSelect={selectNode}
        />
        {children.map((c, i) => {
          const x = W / 2 + (i - (children.length - 1) / 2) * 100;
          return (
            <g key={c.id}>
              <AgentCircle node={c} pct={(c.tokens / rootTokens) * 100} x={x} y={childY} hue={HUES[i % HUES.length]} onHover={setHovered} onLeave={() => setHovered(null)} onSelect={selectNode} />
              <text x={x} y={childY + RING_R + 14} textAnchor="middle" fontSize={8.5} fill="var(--fg-dim)" className="truncate">
                {(c.task || c.name).slice(0, 12)}{(c.task || c.name).length > 12 ? "…" : ""}
              </text>
              <text x={x} y={childY + RING_R + 25} textAnchor="middle" fontSize={7.5} fill="var(--fg-faint)" className="uppercase">
                {c.status}
              </text>
            </g>
          );
        })}
      </svg>

      {/* 悬停详情条 */}
      <div className="mt-1 min-h-[38px] rounded-md border border-border-soft/60 bg-bg-soft/40 px-2 py-1.5 text-[10px]">
        {hovered ? (
          <div className="flex flex-wrap gap-x-3 gap-y-0.5">
            <span className="font-medium text-fg">{hovered.kind === "root" ? t("subagent.rootTitle") : (hovered.task || hovered.name)}</span>
            <span className={hovered.status === "running" ? "text-green-500" : hovered.status === "error" ? "text-err" : "text-fg-dim"}>
              {hovered.status === "running" ? t("subagent.statusActive") : hovered.status === "error" ? t("subagent.statusError") : t("subagent.statusDone")}
            </span>
            {hovered.model && <span className="text-fg-dim">{t("subagent.modelLabel", { model: hovered.model })}</span>}
            <span className="text-fg-dim tabular-nums">{t("subagent.toolCount", { n: hovered.toolCalls })}</span>
            {hovered.errors > 0 && <span className="text-err tabular-nums">{t("subagent.errCountLong", { n: hovered.errors })}</span>}
            <span className="text-fg-dim tabular-nums font-mono">≈{fmtTokens(hovered.tokens)}</span>
            {hovered.lastTs ? <span className="text-fg-faint tabular-nums font-mono">{new Date(hovered.lastTs * 1000).toLocaleTimeString()}</span> : null}
          </div>
        ) : (
          <div className="text-fg-faint">{t("subagent.hoverHint")}</div>
        )}
      </div>

      {/* 点击固定的详情（子代理 → 分工 meta；其余 → 节点统计） */}
      {detail && (
        <div className="mt-1 rounded-md border border-border-soft/60 bg-bg-soft/40 p-2 text-[10px]">
          <div className="flex items-center justify-between">
            <span className="font-medium text-fg">{detail.kind === "root" ? t("subagent.rootTitle") : (detail.task || detail.name)}</span>
            <button
              className="cursor-pointer border-0 bg-transparent text-fg-faint hover:text-fg"
              onClick={() => { setDetail(null); setRunDetail(null); }}
            >{t("subagent.close")}</button>
          </div>
          {runDetail ? (
            <>
              <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-fg-dim">
                <span className={runDetail.status === "running" ? "text-green-500" : runDetail.status === "failed" ? "text-err" : "text-fg-dim"}>
                  {runDetail.status === "running" ? t("subagent.statusActive") : runDetail.status === "failed" ? t("subagent.statusFailed") : t("subagent.statusDone")}
                </span>
                {runDetail.model && <span>{t("subagent.modelLabel", { model: runDetail.model })}</span>}
                <span className="tabular-nums">{t("subagent.toolCalls", { n: runDetail.toolCalls })}</span>
                <span className="text-fg-faint tabular-nums">{new Date(runDetail.updatedAt).toLocaleString()}</span>
              </div>
              {runDetail.lastText && <div className="mt-1 text-fg-dim">{runDetail.lastText}</div>}
              {runDetail.lastTool && <div className="mt-0.5 truncate font-mono text-fg-faint" title={runDetail.lastTool}>{runDetail.lastTool}</div>}
              {runDetail.answer && <div className="mt-1 rounded bg-bg p-1.5 text-fg-dim">{runDetail.answer}</div>}
              {runDetail.ref && (
                <button
                  className="mt-1.5 cursor-pointer rounded border border-border-soft bg-bg px-2 py-1 text-[10px] text-accent hover:bg-bg-soft"
                  onClick={() => viewTranscript(runDetail.ref)}
                >{transcriptOpen && transcript?.ref === runDetail.ref ? t("subagent.collapseTranscript") : t("subagent.viewTranscript")}</button>
              )}
            </>
          ) : (
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-fg-dim">
              <span>{t("subagent.stLabel", { status: detail.status })}</span>
              {detail.model && <span>{t("subagent.modelLabel", { model: detail.model })}</span>}
              <span className="tabular-nums">{t("subagent.toolCount", { n: detail.toolCalls })}</span>
              {detail.errors > 0 && <span className="text-err tabular-nums">{t("subagent.errCountLong", { n: detail.errors })}</span>}
              <span className="tabular-nums font-mono">≈{fmtTokens(detail.tokens)}</span>
            </div>
          )}
        </div>
      )}

      {/* 完整 transcript（点击「查看完整 transcript」展开） */}
      {transcriptOpen && transcript && (
        <div className="mt-1 rounded-md border border-border-soft/60 bg-bg p-2">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-[10px] font-medium text-fg">{transcript.task || transcript.ref}</span>
            <span className="flex items-center gap-1.5">
              <input
                className="w-32 rounded border border-border-soft bg-bg-soft/40 px-1.5 py-0.5 text-[9px] text-fg outline-none placeholder:text-fg-faint"
                placeholder={t("subagent.searchMsg")}
                value={transcriptQuery}
                onChange={(e) => setTranscriptQuery(e.target.value)}
              />
              <span className="text-[9px] text-fg-faint tabular-nums">{t("subagent.msgFilterCount", { shown: shownTranscriptMsgs.length, total: transcript.messages.length })}</span>
            </span>
          </div>
          <div ref={transcriptListRef} className="max-h-64 space-y-1 overflow-y-auto">
            {shownTranscriptMsgs.length === 0 && <div className="py-2 text-[10px] text-fg-faint">{t("subagent.noMatchMsg")}</div>}
            {shownTranscriptMsgs.map(({ m, idx }, i) => (
              <div
                key={idx}
                data-msg-hit={transcriptQuery.trim() && i === 0 ? "true" : undefined}
                className="rounded bg-bg-soft/40 px-1.5 py-1 text-[10px]"
              >
                <div className="flex items-center gap-1.5 text-fg-faint">
                  <span className="shrink-0 tabular-nums text-fg-faint/70">#{idx + 1}</span>
                  <span className={`shrink-0 rounded px-1 text-[9px] ${
                    m.role === "user" ? "bg-emerald-500/15 text-emerald-400"
                      : m.role === "assistant" ? "bg-purple-500/15 text-purple-400"
                        : m.role === "tool" ? "bg-orange-500/15 text-orange-400"
                          : "bg-cyan-500/15 text-cyan-400"
                  }`}>{m.role === "tool" ? "TOOL" : m.role.toUpperCase()}</span>
                  {m.name && <span className="font-mono">{m.name}</span>}
                  {m.toolCallId && <span className="font-mono text-fg-faint">{m.toolCallId}</span>}
                </div>
                {m.reasoning && <pre className="mt-0.5 whitespace-pre-wrap border-l-2 border-info/40 pl-1.5 font-mono text-fg-dim">{m.reasoning}</pre>}
                {m.toolCalls && m.toolCalls.map((tc, j) => (
                  <div key={j} className="mt-0.5 font-mono text-fg-dim">{tc.name} {tc.arguments}</div>
                ))}
                {m.content && <pre className="mt-0.5 whitespace-pre-wrap font-mono text-fg-dim">{m.content}</pre>}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
