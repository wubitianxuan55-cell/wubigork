import { useCallback, useEffect, useRef, useState } from "react";
import { app } from "../lib/bridge";
import type { AgentNetwork, AgentNode } from "../lib/types";
import { fmtTokens } from "../lib/stats";

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
function AgentCircle({ node, pct, x, y, hue, onHover, onLeave }: {
  node: AgentNode;
  pct: number;
  x: number;
  y: number;
  hue: string;
  onHover: (n: AgentNode) => void;
  onLeave: () => void;
}) {
  const statusColor = nodeStatusColor(node.status);
  return (
    <g
      className="cursor-pointer"
      onMouseEnter={() => onHover(node)}
      onMouseLeave={onLeave}
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
        {node.kind === "root" ? "主" : String(node.toolCalls)}
      </text>
    </g>
  );
}

// ─── Agent 网络卡 ──────────────────────────────────────────
export function AgentNetworkCard({ running }: { running: boolean }) {
  const [net, setNet] = useState<AgentNetwork>(EMPTY);
  const [hovered, setHovered] = useState<AgentNode | null>(null);
  const [error, setError] = useState<string | null>(null);
  const runningRef = useRef(running);

  const load = useCallback(() => {
    app.AgentNetwork()
      .then((n) => { setNet(n); setError(null); })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    if (runningRef.current && !running) load();
    runningRef.current = running;
  }, [running, load]);

  const root = net.root;
  const children = root.children ?? [];
  const rootTokens = root.tokens || 1;
  const W = Math.max(320, 120 + children.length * 100);
  const childY = 130;

  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">Agent 网络</div>
        <div className="text-[9px] text-fg-faint">{children.length} 个子代理 · 环 = 上下文 token 占比</div>
      </div>
      {error && <div className="mt-1 text-[10px] text-err">加载失败：{error}</div>}
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
        />
        {children.map((c, i) => {
          const x = W / 2 + (i - (children.length - 1) / 2) * 100;
          return (
            <g key={c.id}>
              <AgentCircle node={c} pct={(c.tokens / rootTokens) * 100} x={x} y={childY} hue={HUES[i % HUES.length]} onHover={setHovered} onLeave={() => setHovered(null)} />
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
            <span className="font-medium text-fg">{hovered.kind === "root" ? "主 agent" : (hovered.task || hovered.name)}</span>
            <span className={hovered.status === "running" ? "text-green-500" : hovered.status === "error" ? "text-err" : "text-fg-dim"}>
              {hovered.status === "running" ? "运行中" : hovered.status === "error" ? "出错" : "已完成"}
            </span>
            {hovered.model && <span className="text-fg-dim">模型 {hovered.model}</span>}
            <span className="text-fg-dim tabular-nums">工具 {hovered.toolCalls}</span>
            {hovered.errors > 0 && <span className="text-err tabular-nums">错误 {hovered.errors}</span>}
            <span className="text-fg-dim tabular-nums font-mono">≈{fmtTokens(hovered.tokens)}</span>
            {hovered.lastTs ? <span className="text-fg-faint tabular-nums font-mono">{new Date(hovered.lastTs * 1000).toLocaleTimeString()}</span> : null}
          </div>
        ) : (
          <div className="text-fg-faint">悬停节点查看子代理详情；点击跳转会话将在后续阶段接入</div>
        )}
      </div>
    </div>
  );
}
