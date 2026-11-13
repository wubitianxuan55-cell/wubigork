import { useMemo } from "react";
import { GitBranch } from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type { AgentNetwork, AgentNode, SubagentRunView } from "../../lib/types";
import type { FamilyAgentRowView } from "../../lib/types/context";
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

export function AgentTopology({
  network,
  runs,
  agents,
  onViewContext,
}: {
  network: AgentNetwork;
  runs: SubagentRunView[];
  agents?: FamilyAgentRowView[];
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const rowByRef = useMemo(
    () => new Map((agents ?? []).map((agent) => [agent.ref, agent])),
    [agents],
  );
  const root = network.root;

  return (
    <section className="ctx-card agent-topology" data-testid="agent-topology">
      <div className="agent-topology-head">
        <span className="agent-topology-title">
          <GitBranch size={14} aria-hidden />
          {t("fleet.networkTitle")}
        </span>
        <span className="agent-topology-sub">{t("fleet.networkSub")}</span>
        <span className="agent-topology-meta">
          {t("contextview.radialAgents", { n: flatten(root).length })}
          {" · "}
          {t("contextview.radialRunning", { n: flatten(root).filter((node) => node.status === "running").length })}
          {" · "}
          {t("contextview.radialTotal", { tokens: fmtTokens(flatten(root).reduce((sum, node) => sum + node.tokens, 0)) })}
        </span>
      </div>
      <div className="agent-topology-scroll">
        <TopologyNode
          node={root}
          root
          window={network.window}
          runs={runs}
          rowByRef={rowByRef}
          onViewContext={onViewContext}
        />
      </div>
    </section>
  );
}

function flatten(node: AgentNode): AgentNode[] {
  const nodes: AgentNode[] = [node];
  for (const child of node.children ?? []) nodes.push(...flatten(child));
  return nodes;
}

function TopologyNode({
  node,
  root = false,
  window,
  runs,
  rowByRef,
  onViewContext,
}: {
  node: AgentNode;
  root?: boolean;
  window: number;
  runs: SubagentRunView[];
  rowByRef: ReadonlyMap<string, FamilyAgentRowView>;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const run = matchRun(node, runs);
  const row = rowByRef.get(node.id);
  const children = node.children ?? [];
  const pct = window > 0 ? Math.min(100, Math.round((node.tokens / window) * 100)) : null;
  const interactive = !root && !!onViewContext;
  const Card = interactive ? "button" : "div";

  return (
    <div className={`agent-topology-branch${root ? " is-root" : ""}`}>
      <Card
        type={interactive ? "button" : undefined}
        className={`agent-topology-card${interactive ? " is-interactive" : ""}`}
        data-agent={node.id}
        data-status={node.status}
        onClick={interactive ? () => onViewContext?.(node.id) : undefined}
        title={interactive ? `${node.task || node.name} · ${statusLabel(node, t)}` : undefined}
      >
        <span className="agent-topology-card-top">
          <span className="agent-topology-role">{root ? t("fleet.roleLead") : t("fleet.roleSub")}</span>
          <span className="agent-topology-status" data-status={node.status}>
            <i aria-hidden />
            {statusLabel(node, t)}
          </span>
        </span>
        <span className="agent-topology-name" title={node.task || node.name}>
          {node.task || node.name}
        </span>
        <span className="agent-topology-metrics">
          {run?.model || node.model ? <span>{run?.model || node.model}</span> : null}
          <span>{fmtTokens(node.tokens)} tok</span>
          <span>工具×{node.toolCalls}</span>
          {row?.cost != null && <span>¥{row.cost.toFixed(2)}</span>}
          {row?.messages ? <span>消息×{row.messages}</span> : null}
        </span>
        {pct !== null && (
          <span className="agent-topology-meter">
            <i style={{ width: `${pct}%` }} />
            <b>{pct}%</b>
          </span>
        )}
      </Card>

      {children.length > 0 && (
        <div className="agent-topology-descend">
          <span className="agent-topology-stem" aria-hidden />
          <div className="agent-topology-children">
            {children.map((child) => (
              <div className="agent-topology-child" key={child.id}>
                <TopologyNode
                  node={child}
                  window={window}
                  runs={runs}
                  rowByRef={rowByRef}
                  onViewContext={onViewContext}
                />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
