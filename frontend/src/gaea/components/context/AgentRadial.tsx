import { useCallback, useMemo, useState } from "react";
import type { AgentNetwork, AgentNode } from "../../lib/types";
import { fmtTokens } from "../../lib/stats";
import { DOMAIN_COLORS, DOMAIN_KEYS } from "../../lib/domainColors";
import { useT } from "../../lib/i18n";
import "./context-view.css";
import { Users } from "../../icons";

// AgentRadial — 「Agent 网络」径向树卡（对齐 dsh-context 的 Agent network 卡）。
// 替换 AgentNetworkCard 在上下文页 inspector 的角色（旧组件本体不动）：
//  - 中心 = 主 agent：双环（底环 surface-container-high 全圆 + 进度弧 --gaea-glow），
//    进度 = root.tokens / network.window（上下文占用%），环心两行 = 会话任务名（截断）+ tokens；
//  - 子代理 = 放射一圈：每节点圆环 = 子 tokens / 主 agent tokens 占比，环心 = 工具调用数，
//    节点下方两行 = 任务摘要（截断 12 字）+ tokens；
//  - 连线 = 中心到子节点的轻弧（二次贝塞尔），线色取子代理在兄弟间的序号色
//    （色板与 domainColors 六分类同源；本文件不在 no-raw-hex 豁免名单，色值统一走 lib 单源）；
//  - 交互：hover 高亮连线 + 原生 <title>（完整任务/模型/状态/工具数）；running 节点呼吸
//    （Tailwind animate-pulse，SVG 元素同样生效）；点击子节点 = ref 可得（id 以 sa_ 开头）
//    时回调 onOpenSubagent（与 TasksWorkbench.openThread 同口径），否则不触发、title 说明；
//    点击中心节点无动作。
//
// 文案硬编码中文——i18n 待主代理收口。

// ─── 画布/布局常量（固定 viewBox 等比缩放 = 自适应宽度；不引 ResizeObserver：
// jsdom 无原生实现，且 w-full + viewBox 已满足自适应，与 TrendChart 同思路） ───
const W = 900;
const H = 380;
const CX = 450;
const CY = 175;
const CENTER_R = 34; // 中心进度环半径
const CHILD_R = 22; // 子节点占比环半径
const R_SINGLE = 150; // 单圈半径（子节点 1–6 个）
const R_INNER = 84; // 双圈内圈半径（子节点 >6 个）
const R_OUTER = 148; // 双圈外圈半径（子节点 >6 个）
const CIRC_CENTER = 2 * Math.PI * CENTER_R;
const CIRC_CHILD = 2 * Math.PI * CHILD_R;

// 兄弟序号色（分类色板单源复用，按序号循环取色）。
function siblingHue(index: number): string {
  return DOMAIN_COLORS[DOMAIN_KEYS[index % DOMAIN_KEYS.length]];
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

function statusLabel(status: AgentNode["status"], t: (k: "subagent.statusRunning" | "subagent.statusFailed" | "subagent.statusDone") => string): string {
  switch (status) {
    case "running":
      return t("subagent.statusRunning");
    case "error":
      return t("subagent.statusFailed");
    default:
      return t("subagent.statusDone");
  }
}

// SVG 文本无 CSS truncate，手动截断（与 AgentNetworkCard 同款省略号）。
function trunc(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + "…" : s;
}

function clampPct(p: number): number {
  return Math.min(100, Math.max(0, p));
}

// ─── 全网统计：节点数（含主 agent）/ 运行中数 / tokens 合计 ───
function collectStats(node: AgentNode): { total: number; running: number; tokens: number } {
  let s = { total: 1, running: node.status === "running" ? 1 : 0, tokens: Math.max(0, node.tokens) };
  for (const c of node.children ?? []) {
    const cs = collectStats(c);
    s = { total: s.total + cs.total, running: s.running + cs.running, tokens: s.tokens + cs.tokens };
  }
  return s;
}

// ─── 布局 ───
// 布局规则（本组件唯一布点逻辑）：
//  - 1–6 个子代理：单圈、下半圆均匀分布（角 = π·(i+1)/(n+1)，正弦为正即圆心下方，
//    标签统一放节点下方不会与连线/圆环交叠），半径 R_SINGLE；
//  - >6 个：两圈嵌套——内圈 ceil(n/2) 个全圆分布（半步相位偏移，避免与外圈径向对齐），
//    外圈放其余，内圈 R_INNER / 外圈 R_OUTER。
interface RadialChild {
  node: AgentNode;
  index: number;
  hue: string;
  ringPct: number; // 子 tokens / 主 agent tokens 占比（%）
  x: number;
  y: number;
}

function layoutChildren(children: AgentNode[], rootTokens: number): RadialChild[] {
  const n = children.length;
  if (n === 0) return [];
  const pct = (c: AgentNode) => (rootTokens > 0 ? clampPct((c.tokens / rootTokens) * 100) : 0);
  const entry = (c: AgentNode, i: number, x: number, y: number): RadialChild => ({
    node: c,
    index: i,
    hue: siblingHue(i),
    ringPct: pct(c),
    x,
    y,
  });
  const out: RadialChild[] = [];
  if (n <= 6) {
    for (let i = 0; i < n; i++) {
      const a = (Math.PI * (i + 1)) / (n + 1); // (0, π)：下半圆
      out.push(entry(children[i], i, CX + R_SINGLE * Math.cos(a), CY + R_SINGLE * Math.sin(a)));
    }
    return out;
  }
  const innerCount = Math.ceil(n / 2);
  for (let i = 0; i < n; i++) {
    const inner = i < innerCount;
    const count = inner ? innerCount : n - innerCount;
    const k = inner ? i : i - innerCount;
    const a = -Math.PI / 2 + ((k + (inner ? 0.5 : 0)) * 2 * Math.PI) / count; // 全圆；内圈偏移半步
    const r = inner ? R_INNER : R_OUTER;
    out.push(entry(children[i], i, CX + r * Math.cos(a), CY + r * Math.sin(a)));
  }
  return out;
}

// 中心 → 子节点的轻弧（二次贝塞尔，中点沿法线偏移 fixed 曲率）。
function edgePath(x1: number, y1: number, x2: number, y2: number): string {
  const mx = (x1 + x2) / 2;
  const my = (y1 + y2) / 2;
  const dx = x2 - x1;
  const dy = y2 - y1;
  const len = Math.hypot(dx, dy) || 1;
  const k = 16;
  const cx = mx + (-dy / len) * k;
  const cy = my + (dx / len) * k;
  return `M ${x1.toFixed(1)} ${y1.toFixed(1)} Q ${cx.toFixed(1)} ${cy.toFixed(1)} ${x2.toFixed(1)} ${y2.toFixed(1)}`;
}

// ─── 组件 ───
export function AgentRadial({
  network,
  running,
  sessionPath,
  onOpenSubagent,
}: {
  network: AgentNetwork;
  running: boolean;
  sessionPath?: string;
  onOpenSubagent?: (p: {
    sessionPath: string;
    ref: string;
    task?: string;
    model?: string;
    status: "running" | "completed" | "failed";
  }) => void;
}) {
  const t = useT();
  const root = network.root;
  const children = useMemo(() => root.children ?? [], [root]);
  const laidOut = useMemo(() => layoutChildren(children, root.tokens), [children, root.tokens]);
  const stats = useMemo(() => collectStats(root), [root]);
  const [hoverId, setHoverId] = useState<string | null>(null);

  // 中心进度 = 主 agent 上下文占用%（root.tokens / 上下文窗口；窗口缺失按 0）。
  const centerPct = network.window > 0 ? clampPct((root.tokens / network.window) * 100) : 0;
  const centerActive = running || root.status === "running";

  // 点击子节点 → 打开子代理对话：ref 口径与 AgentTree/TasksWorkbench 相同
  // （id 以 sa_ 开头可直用；本组件拿不到 runs，无 sa_ 前缀则 ref 不可得 → 不触发）。
  const openNode = useCallback(
    (node: AgentNode) => {
      if (!onOpenSubagent || !sessionPath) return;
      if (!node.id.startsWith("sa_")) return;
      onOpenSubagent({
        sessionPath,
        ref: node.id,
        task: node.task,
        model: node.model,
        status: node.status === "error" ? "failed" : node.status,
      });
    },
    [onOpenSubagent, sessionPath],
  );

  const centerTitle = [
    root.task || root.name,
    `状态：${statusLabel(root.status, t)}`,
    root.model ? `模型：${root.model}` : null,
    `工具调用：${root.toolCalls}`,
    root.errors > 0 ? `错误：${root.errors}` : null,
    network.window > 0 ? `上下文窗口：≈${fmtTokens(network.window)}` : null,
  ]
    .filter(Boolean)
    .join("\n");

  return (
    <div className="ctx-card p-3" data-testid="agent-radial">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <Users size={12} />
          </span>
          <span className="truncate text-[12.5px] font-semibold text-fg">{t("contextview.radialTitle")}</span>
        </div>
        <div className="shrink-0 text-[9px] text-fg-faint tabular-nums">
          {t("contextview.radialAgents", { n: stats.total })} · {t("contextview.radialRunning", { n: stats.running })} · {t("contextview.radialTotal", { tokens: fmtTokens(stats.tokens) })}
        </div>
      </div>

      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="mt-1 w-full h-auto"
        data-testid="agent-radial-svg"
        role="img"
        aria-label={t("contextview.radialTitle")}
      >
        {/* 连线（先画，节点/标签盖在上面）：hover 对应节点时高亮 */}
        {laidOut.map((c) => {
          const dx = c.x - CX;
          const dy = c.y - CY;
          const len = Math.hypot(dx, dy) || 1;
          const sx = CX + (dx / len) * (CENTER_R + 6);
          const sy = CY + (dy / len) * (CENTER_R + 6);
          const ex = c.x - (dx / len) * (CHILD_R + 5);
          const ey = c.y - (dy / len) * (CHILD_R + 5);
          const hovered = hoverId === c.node.id;
          return (
            <path
              key={`edge-${c.node.id}`}
              d={edgePath(sx, sy, ex, ey)}
              fill="none"
              stroke={c.hue}
              strokeWidth={hovered ? 2.5 : 1.5}
              opacity={hovered ? 1 : 0.55}
              className={c.node.status === "running" ? "animate-pulse" : ""}
            />
          );
        })}

        {/* 中心节点：主 agent（点击无动作） */}
        <g data-testid="agent-radial-center">
          <title>{centerTitle}</title>
          <circle cx={CX} cy={CY} r={CENTER_R + 6} fill="var(--bg-soft)" stroke="var(--border-soft)" strokeWidth={1} />
          {/* 底环：全圆 */}
          <circle cx={CX} cy={CY} r={CENTER_R} fill="none" stroke="var(--md-sys-color-surface-container-high)" strokeWidth={6} />
          {/* 进度弧：主 agent 上下文占用% */}
          <circle
            data-testid="agent-radial-center-progress"
            cx={CX}
            cy={CY}
            r={CENTER_R}
            fill="none"
            stroke="var(--gaea-glow)"
            strokeWidth={6}
            strokeLinecap="round"
            strokeDasharray={`${(centerPct / 100) * CIRC_CENTER} ${CIRC_CENTER}`}
            transform={`rotate(-90 ${CX} ${CY})`}
          />
          {/* 内盘：running 呼吸 */}
          <circle
            cx={CX}
            cy={CY}
            r={CENTER_R - 8}
            fill="var(--bg)"
            className={centerActive ? "animate-pulse" : ""}
          />
          <text x={CX} y={CY - 5} textAnchor="middle" fontSize={9.5} fontWeight={600} fill="var(--fg)">
            {trunc(root.task || root.name, 8)}
          </text>
          <text x={CX} y={CY + 11} textAnchor="middle" fontSize={9} fill="var(--fg-dim)" className="tabular-nums">
            ≈{fmtTokens(root.tokens)}
          </text>
        </g>

        {/* 子代理节点 */}
        {laidOut.map((c) => {
          const hasRef = c.node.id.startsWith("sa_");
          const clickable = hasRef && !!sessionPath && !!onOpenSubagent;
          const title = [
            c.node.task || c.node.name,
            `状态：${statusLabel(c.node.status, t)}`,
            c.node.model ? `模型：${c.node.model}` : null,
            `工具调用：${c.node.toolCalls}`,
            c.node.errors > 0 ? `错误：${c.node.errors}` : null,
            `tokens：≈${fmtTokens(c.node.tokens)}`,
            !hasRef
              ? "ref 不可得，无法打开子代理对话"
              : clickable
                ? "点击打开子代理对话"
                : "未接线打开回调，点击不触发",
          ]
            .filter(Boolean)
            .join("\n");
          return (
            <g
              key={c.node.id}
              data-testid="agent-radial-node"
              data-node-id={c.node.id}
              className={clickable ? "cursor-pointer" : "cursor-default"}
              onMouseEnter={() => setHoverId(c.node.id)}
              onMouseLeave={() => setHoverId((cur) => (cur === c.node.id ? null : cur))}
              onClick={() => openNode(c.node)}
            >
              <title>{title}</title>
              <circle cx={c.x} cy={c.y} r={CHILD_R + 3} fill="var(--bg-soft)" stroke="var(--border-soft)" strokeWidth={1} />
              {/* 占比环：子 tokens / 主 agent tokens */}
              <circle
                data-testid="agent-radial-node-ring"
                cx={c.x}
                cy={c.y}
                r={CHILD_R}
                fill="none"
                stroke={c.hue}
                strokeWidth={4}
                strokeLinecap="round"
                opacity={0.9}
                strokeDasharray={`${(c.ringPct / 100) * CIRC_CHILD} ${CIRC_CHILD}`}
                transform={`rotate(-90 ${c.x} ${c.y})`}
              />
              {/* 内盘：状态描边，running 呼吸 */}
              <circle
                cx={c.x}
                cy={c.y}
                r={CHILD_R - 5}
                fill="var(--bg)"
                stroke={statusColor(c.node.status)}
                strokeWidth={1.5}
                className={c.node.status === "running" ? "animate-pulse" : ""}
              />
              <text x={c.x} y={c.y + 3} textAnchor="middle" fontSize={8.5} fontWeight={600} fill="var(--fg)" className="tabular-nums">
                {c.node.toolCalls}
              </text>
              <text x={c.x} y={c.y + CHILD_R + 14} textAnchor="middle" fontSize={8.5} fill="var(--fg-dim)">
                {trunc(c.node.task || c.node.name, 12)}
              </text>
              <text x={c.x} y={c.y + CHILD_R + 25} textAnchor="middle" fontSize={7.5} fill="var(--fg-faint)" className="tabular-nums">
                ≈{fmtTokens(c.node.tokens)}
              </text>
            </g>
          );
        })}

        {/* 空态：无子代理 */}
        {laidOut.length === 0 && (
          <text x={CX} y={CY + CENTER_R + 26} textAnchor="middle" fontSize={10} fill="var(--fg-faint)" data-testid="agent-radial-empty">
            {t("contextview.radialNoChildren")}
          </text>
        )}
      </svg>
    </div>
  );
}
