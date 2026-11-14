import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { ArrowUpRight, GitBranch } from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../../lib/types";
import type { FamilyAgentRowView } from "../../lib/types/context";
import {
  FLEET_CARD_HEIGHT,
  layoutAgentTopology,
  topologyLinkPath,
  type FleetTopologyNode,
} from "./agentTopologyLayout";
import "./agent-topology.css";

function matchRun(node: AgentNode, runs: SubagentRunView[]): SubagentRunView | null {
  const exact = runs.find((run) => run.ref === node.id);
  if (exact) return exact;
  return runs.find((run) => (
    !!run.task && !!node.task &&
    (run.task.startsWith(node.task) || node.task.startsWith(run.task))
  )) ?? null;
}

function statusLabel(node: AgentNode, t: ReturnType<typeof useT>): string {
  if (node.status === "running") return t("subagent.statusRunning");
  if (node.status === "error") return t("subagent.statusError");
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
  runs,
  agents,
  selectedAgent = null,
  onSelectAgent,
  onViewContext,
}: {
  network: AgentNetwork;
  runs: SubagentRunView[];
  agents?: FamilyAgentRowView[];
  selectedAgent?: string | null;
  onSelectAgent?: (ref: string | null) => void;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const stageRef = useRef<HTMLDivElement | null>(null);
  const [stageWidth, setStageWidth] = useState(720);
  const [hovered, setHovered] = useState<string | null>(null);
  const rowByRef = useMemo(
    () => new Map((agents ?? []).map((agent) => [agent.ref, agent])),
    [agents],
  );
  const layout = useMemo(
    () => layoutAgentTopology(network, stageWidth),
    [network, stageWidth],
  );
  const allNodes = layout.nodes.map((entry) => entry.node);
  const runningCount = allNodes.filter((node) => node.status === "running").length;
  const totalTokens = allNodes.reduce((sum, node) => sum + node.tokens, 0);
  const selectedNode = selectedAgent === null
    ? null
    : allNodes.find((node) => node.id === selectedAgent) ?? null;
  const selectedRun = selectedNode === null ? null : matchRun(selectedNode, runs);
  const selectedLabel = selectedNode === null
    ? null
    : selectedRun?.task || selectedNode.task || selectedNode.name;
  const selectedPoint = selectedAgent === null
    ? null
    : layout.points.find((point) => point.id === selectedAgent) ?? null;
  const focusId = hovered ?? selectedAgent;
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
    onSelectAgent(selectedAgent === id ? null : id);
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
            return (
              <TopologyCard
                key={entry.node.id}
                entry={entry}
                x={point.x}
                y={point.y}
                width={layout.cardWidth}
                hue={point.hue}
                index={index}
                windowSize={network.window}
                run={matchRun(entry.node, runs)}
                row={rowByRef.get(entry.node.id)}
                selected={selectedAgent === entry.node.id}
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

function TopologyCard({
  entry,
  x,
  y,
  width,
  hue,
  index,
  windowSize,
  run,
  row,
  selected,
  hovered,
  focused,
  related,
  onChoose,
  onHover,
  onViewContext,
}: {
  entry: FleetTopologyNode;
  x: number;
  y: number;
  width: number;
  hue: string;
  index: number;
  windowSize: number;
  run: SubagentRunView | null;
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
  const model = run?.model || node.model;
  const meta = [
    duration,
    node.toolCalls > 0 ? `${node.toolCalls} ${t("fleet.kpiToolCalls")}` : null,
    row?.messages ? `${row.messages} ${t("fleet.kpiMessages")}` : null,
  ].filter((part): part is string => !!part);

  return (
    <div
      className={`agent-topology-card${root ? " is-root" : ""}${selected ? " is-selected" : ""}${hovered ? " is-hover" : ""}${focused && !related ? " is-muted" : ""}`}
      style={{
        left: x - width / 2,
        top: y,
        width,
        height: FLEET_CARD_HEIGHT,
        "--agent-hue": hue,
        "--agent-index": Math.min(index, 8),
      } as CSSProperties}
      data-agent={node.id}
      data-status={node.status}
      title={`${label} · ${statusLabel(node, t)}`}
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
          {statusLabel(node, t)}
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
