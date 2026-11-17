import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { ArrowUpRight, ChevronDown, ChevronRight, GitBranch } from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type { AgentNetwork, AgentNode } from "../../lib/types";
import type { FamilyAgentRowView } from "../../lib/types/context";
import {
  cardHeightOf,
  layoutFleetTopology,
  topologyLinkPath,
} from "./agentTopologyLayout";
import { buildFleetModel, type FleetNode } from "./fleetModel";
import "./agent-topology.css";

// D1 会话骨架（dsh Tasks 页节点语义蒸馏）：network.root 的节点 id 已是子代理
// 会话 ref（buildSessionNetwork 以 runs 父子链建树），状态/模型/任务摘要/活动
// 行由后端直接落在节点上——此前按「任务前缀互含」猜 run 的 matchRun 启发式
// 随之退役。

function statusLabel(status: AgentNode["status"], t: ReturnType<typeof useT>): string {
  if (status === "running") return t("subagent.statusRunning");
  if (status === "queued") return t("subagent.statusQueued");
  if (status === "aborted") return t("subagent.statusAborted");
  if (status === "error" || status === "failed") return t("subagent.statusError");
  return t("subagent.statusDone");
}

function avatarLabelOf(label: string): string {
  const compact = label.replace(/\s+/g, "");
  return compact.slice(0, 2).toUpperCase() || "A";
}

function durationOf(node: AgentNode): string | null {
  if (node.firstTs === undefined || node.lastTs === undefined || node.lastTs <= node.firstTs) return null;
  const totalSeconds = Math.max(1, Math.round(node.lastTs - node.firstTs));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

export function AgentTopology({
  network,
  agents,
  selectedAgent = null,
  onSelectAgent,
  onViewContext,
}: {
  network: AgentNetwork;
  agents?: FamilyAgentRowView[];
  selectedAgent?: string | null;
  onSelectAgent?: (ref: string | null) => void;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const stageRef = useRef<HTMLDivElement | null>(null);
  const [stageWidth, setStageWidth] = useState(720);
  const [hovered, setHovered] = useState<string | null>(null);
  // D3 折叠态：expandedGroups（点聚合卡只展开该组）+ foldedIds（手折优先）。
  const [expandedGroups, setExpandedGroups] = useState<ReadonlySet<string>>(new Set());
  const [foldedIds, setFoldedIds] = useState<ReadonlySet<string>>(new Set());
  const rowByRef = useMemo(
    () => new Map((agents ?? []).map((agent) => [agent.ref, agent])),
    [agents],
  );
  const model = useMemo(
    () => buildFleetModel(network.root, { expandedGroups, foldedIds }),
    [network.root, expandedGroups, foldedIds],
  );
  const layout = useMemo(
    () => layoutFleetTopology(model, stageWidth),
    [model, stageWidth],
  );
  const toggleFold = (group: string, expanded: boolean) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (expanded) next.delete(group);
      else next.add(group);
      return next;
    });
    setFoldedIds((prev) => {
      const next = new Set(prev);
      if (expanded) next.add(group); // 手折优先：展开态再点 = 手折收起
      else next.delete(group);
      return next;
    });
  };
  const rootId = network.root.id;
  const allNodes = layout.nodes.map((entry) => entry.node);
  // 聚合卡不入计数（fold 节点的 queued/completed 是组语义，不是真实运行态）。
  const runningCount = allNodes.filter(
    (node) => node.kind !== "fold" && (node.status === "running" || node.status === "queued"),
  ).length;
  const totalTokens = allNodes.reduce((sum, node) => sum + node.tokens, 0);
  // Fleet 侧选中态用 "main" 指主 agent（knownRefs 语义）；拓扑节点 id 是
  // "root"——双向归一，root 卡片选中不再被 Fleet 的 knownRefs 校验清掉。
  const selectedNodeId = selectedAgent === "main" ? rootId : selectedAgent;
  const selectedNode = selectedNodeId === null
    ? null
    : allNodes.find((node) => node.id === selectedNodeId) ?? null;
  const selectedLabel = selectedNode === null ? null : selectedNode.task || selectedNode.name;
  const selectedPoint = selectedNodeId === null
    ? null
    : layout.points.find((point) => point.id === selectedNodeId) ?? null;
  const focusId = hovered ?? selectedNodeId;
  const relatedIds = useMemo(() => {
    if (focusId === null) return null;
    const byId = new Map(layout.nodes.map((entry) => [entry.node.id, entry]));
    const related = new Set<string>();
    let current = byId.get(focusId);
    while (current !== undefined) {
      related.add(current.node.id);
      current = current.parentId === undefined ? undefined : byId.get(current.parentId);
    }
    const addDescendants = (id: string) => {
      for (const entry of layout.nodes) {
        if (entry.parentId !== id || related.has(entry.node.id)) continue;
        related.add(entry.node.id);
        addDescendants(entry.node.id);
      }
    };
    addDescendants(focusId);
    return related;
  }, [focusId, layout.nodes]);

  useEffect(() => {
    const element = stageRef.current;
    if (element === null) return undefined;
    const measure = () => {
      const width = element.getBoundingClientRect().width;
      if (width > 0) setStageWidth(width);
    };
    measure();
    if (typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const choose = (id: string) => {
    if (onSelectAgent === undefined) return;
    const ref = id === rootId ? "main" : id;
    onSelectAgent(selectedAgent === ref ? null : ref);
  };

  return (
    <section className="ctx-card agent-topology" data-testid="agent-topology">
      <div className="agent-topology-head">
        <span className="agent-topology-title">
          <GitBranch size={14} aria-hidden />
          {t("fleet.networkTitle")}
        </span>
        <span className="agent-topology-sub">{t("fleet.networkSub")}</span>
        <span className="agent-topology-meta">
          {t("contextview.radialAgents", { n: allNodes.length })}
        </span>
        <span className="agent-topology-meta agent-topology-meta-running" data-running={runningCount > 0}>
          {t("contextview.radialRunning", { n: runningCount })}
        </span>
        <span className="agent-topology-meta">
          {t("contextview.radialTotal", { tokens: fmtTokens(totalTokens) })}
        </span>
      </div>

      <div className="agent-topology-legend" aria-label={t("fleet.networkTitle")}>
        <span><i className="is-lead" />{t("fleet.roleLead")}</span>
        <span><i className="is-sub" />{t("fleet.roleSub")}</span>
        <span><i className="is-running" />{t("fleet.stRunning")}</span>
        {selectedPoint !== null && (
          <span className="agent-topology-legend-selected">
            <i style={{ background: selectedPoint.hue }} />
            {selectedLabel}
          </span>
        )}
      </div>

      <div className="agent-topology-stage" ref={stageRef}>
        <div
          className={`agent-topology-canvas${focusId !== null ? " is-focused" : ""}`}
          style={{ width: layout.width, height: layout.height }}
        >
          <svg
            className="agent-topology-links"
            width={layout.width}
            height={layout.height}
            viewBox={`0 0 ${layout.width} ${layout.height}`}
            aria-hidden
          >
            {layout.links.map((link) => {
              const child = layout.nodes.find((entry) => entry.node.id === link.to);
              const parent = child?.parentId === undefined
                ? undefined
                : layout.points.find((point) => point.id === child.parentId);
              const childPoint = layout.points.find((point) => point.id === link.to);
              if (childPoint === undefined) return null;
              const lit = relatedIds !== null
                ? relatedIds.has(link.to) && parent !== undefined && relatedIds.has(parent.id)
                : false;
              return (
                <g key={link.to} className={lit ? "is-lit" : undefined}>
                  <path
                    className={`agent-topology-link${link.running ? " is-running" : ""}`}
                    d={topologyLinkPath(link)}
                    stroke={link.hue}
                  />
                  {link.running && (
                    <path
                      className="agent-topology-flow"
                      d={topologyLinkPath(link)}
                      stroke={link.hue}
                    />
                  )}
                  <circle className="agent-topology-joint" cx={link.x1} cy={link.y1} r="2.5" fill={link.hue} />
                  <circle className="agent-topology-joint" cx={link.x2} cy={link.y2} r="3" fill={link.hue} />
                </g>
              );
            })}
          </svg>

          {layout.nodes.map((entry, index) => {
            const point = layout.points.find((candidate) => candidate.id === entry.node.id);
            if (point === undefined) return null;
            if (entry.fold !== undefined) {
              return (
                <FoldCard
                  key={entry.node.id}
                  fold={entry.fold}
                  x={point.x}
                  y={point.y}
                  width={layout.cardWidth}
                  height={cardHeightOf(entry.node)}
                  hue={point.hue}
                  onToggle={() => toggleFold(entry.node.id, entry.fold!.expanded)}
                />
              );
            }
            return (
              <TopologyCard
                key={entry.node.id}
                entry={entry}
                x={point.x}
                y={point.y}
                width={layout.cardWidth}
                height={cardHeightOf(entry.node)}
                hue={point.hue}
                index={index}
                windowSize={network.window}
                row={rowByRef.get(entry.node.id)}
                selected={selectedNodeId === entry.node.id}
                hovered={hovered === entry.node.id}
                focused={focusId !== null}
                related={relatedIds?.has(entry.node.id) ?? false}
                onChoose={() => choose(entry.node.id)}
                onHover={setHovered}
                onViewContext={onViewContext}
              />
            );
          })}
        </div>
      </div>
    </section>
  );
}

// FoldCard 是 D3 折叠聚合卡（done/idle 两类）：折叠态点开只展开该组；展开态
// 卡片保留作收起入口（手折优先，chevron 不失效）。
function FoldCard({
  fold,
  x,
  y,
  width,
  height,
  hue,
  onToggle,
}: {
  fold: NonNullable<FleetNode["fold"]>;
  x: number;
  y: number;
  width: number;
  height: number;
  hue: string;
  onToggle: () => void;
}) {
  const t = useT();
  const label = fold.kind === "done" ? t("fleet.foldDone", { n: fold.count }) : t("fleet.foldIdle", { n: fold.count });
  return (
    <div
      className={`agent-topology-card is-fold${fold.expanded ? " is-expanded" : ""}${fold.errorCount > 0 ? " has-errors" : ""}`}
      style={{
        left: x - width / 2,
        top: y,
        width,
        height,
        "--agent-hue": hue,
      } as CSSProperties}
      data-fold={fold.kind}
      data-count={fold.count}
      title={`${label}${fold.previews.length > 0 ? ` · ${fold.previews.join(" / ")}` : ""}`}
    >
      <button
        type="button"
        className="agent-topology-hit"
        aria-label={label}
        aria-expanded={fold.expanded}
        onClick={onToggle}
      />
      <span className="agent-topology-fold-head">
        <span className="agent-topology-fold-chevron" aria-hidden>
          {fold.expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        </span>
        <span className="agent-topology-fold-label">{label}</span>
        {fold.errorCount > 0 && (
          <span className="agent-topology-fold-errors">{t("contextview.radialErrors", { n: fold.errorCount })}</span>
        )}
      </span>
      {fold.previews.length > 0 && (
        <span className="agent-topology-fold-previews">{fold.previews.join(" / ")}</span>
      )}
    </div>
  );
}

function TopologyCard({
  entry,
  x,
  y,
  width,
  height,
  hue,
  index,
  windowSize,
  row,
  selected,
  hovered,
  focused,
  related,
  onChoose,
  onHover,
  onViewContext,
}: {
  entry: FleetNode;
  x: number;
  y: number;
  width: number;
  height: number;
  hue: string;
  index: number;
  windowSize: number;
  row?: FamilyAgentRowView;
  selected: boolean;
  hovered: boolean;
  focused: boolean;
  related: boolean;
  onChoose: () => void;
  onHover: (ref: string | null) => void;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const node = entry.node;
  const root = node.kind === "root";
  const label = node.task || node.name || node.id;
  const pct = windowSize > 0 ? Math.min(100, Math.round((node.tokens / windowSize) * 100)) : null;
  const cost = row?.cost;
  const duration = durationOf(node);
  const role = root ? t("fleet.roleLead") : t("fleet.roleSub");
  const model = node.model;
  const meta = [
    duration,
    node.toolCalls > 0 ? `${node.toolCalls} ${t("fleet.kpiToolCalls")}` : null,
    row?.messages ? `${row.messages} ${t("fleet.kpiMessages")}` : null,
  ].filter((part): part is string => !!part);
  const tail = !root && (node.lastTool !== undefined || node.lastText !== undefined);

  return (
    <div
      className={`agent-topology-card${root ? " is-root" : ""}${tail ? " has-tail" : ""}${selected ? " is-selected" : ""}${hovered ? " is-hover" : ""}${focused && !related ? " is-muted" : ""}`}
      style={{
        left: x - width / 2,
        top: y,
        width,
        height,
        "--agent-hue": hue,
        "--agent-index": Math.min(index, 8),
      } as CSSProperties}
      data-agent={node.id}
      data-status={node.status}
      title={`${label} · ${statusLabel(node.status, t)}`}
      onMouseEnter={() => onHover(node.id)}
      onMouseLeave={() => onHover(null)}
      onFocus={() => onHover(node.id)}
      onBlur={() => onHover(null)}
    >
      <button
        type="button"
        className="agent-topology-hit"
        aria-label={label}
        aria-pressed={selected}
        onClick={onChoose}
      />
      <span className="agent-topology-card-top">
        <span className="agent-topology-avatar" aria-hidden>{avatarLabelOf(label)}</span>
        <span className="agent-topology-name" title={label}>{label}</span>
        <span className="agent-topology-status" data-status={node.status}>
          <i aria-hidden />
          {statusLabel(node.status, t)}
        </span>
      </span>
      <span className="agent-topology-role">{role}{model ? ` · ${model}` : ""}</span>
      <span className="agent-topology-metrics">
        <b>{fmtTokens(node.tokens)} tok</b>
        {cost !== undefined && <span>¥{cost.toFixed(2)}</span>}
      </span>
      <span className="agent-topology-meter" aria-hidden>
        <i style={{ width: `${pct ?? 0}%` }} />
      </span>
      {node.lastTool !== undefined && (
        <span className="agent-topology-tail-tool" title={node.lastTool}>{node.lastTool}</span>
      )}
      {node.lastText !== undefined && (
        <span className="agent-topology-tail-text" title={node.lastText}>{node.lastText}</span>
      )}
      <span className="agent-topology-card-foot">
        <span title={meta.join(" · ")}>{meta.length > 0 ? meta.join(" · ") : "—"}</span>
        {pct !== null && <b>{pct}%</b>}
      </span>
      {!root && onViewContext !== undefined && (
        <button
          type="button"
          className="agent-topology-open"
          aria-label={t("fleet.open")}
          title={t("fleet.open")}
          onClick={(event) => {
            event.stopPropagation();
            onViewContext(node.id);
          }}
        >
          <ArrowUpRight size={11} aria-hidden />
        </button>
      )}
    </div>
  );
}
