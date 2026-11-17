// D8 双视图之 TREE（dsh Tasks 页蒸馏：一个模型喂两种视图，折叠态与视图切换
// 不分叉）：缩进树渲染器复用 buildFleetModel 的 pre-order 节点表——选中态/
// 折叠态/过滤联动全部作用于同一模型，切视图不丢状态（计划 §4·D8 验收）。
import { ArrowUpRight, ChevronDown, ChevronRight } from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type { AgentNode } from "../../lib/types";
import type { FamilyAgentRowView } from "../../lib/types/context";
import { statusLabelOf, type FleetNode } from "./fleetModel";
import { depthHue } from "./fleetVisuals";
import "./fleet-tree.css";

export function FleetTree({
  nodes,
  selectedAgent = null,
  onSelectAgent,
  onViewContext,
  rowByRef,
  windowSize,
  onToggleFold,
}: {
  nodes: FleetNode[];
  selectedAgent?: string | null;
  onSelectAgent?: (ref: string | null) => void;
  onViewContext?: (ref: string) => void;
  rowByRef: Map<string, FamilyAgentRowView>;
  windowSize: number;
  onToggleFold: (group: string, expanded: boolean) => void;
}) {
  const t = useT();
  const statusText = (status: AgentNode["status"]) =>
    statusLabelOf(status, {
      running: t("subagent.statusRunning"),
      queued: t("subagent.statusQueued"),
      aborted: t("subagent.statusAborted"),
      error: t("subagent.statusError"),
      done: t("subagent.statusDone"),
    });

  return (
    <div className="fleet-tree" data-testid="fleet-tree">
      {nodes.map((entry) => {
        const node = entry.node;
        if (entry.fold !== undefined) {
          const fold = entry.fold;
          const text =
            fold.kind === "done"
              ? t("fleet.foldDone", { n: fold.count })
              : t("fleet.foldIdle", { n: fold.count });
          return (
            <div
              key={node.id}
              className={`fleet-tree-row is-fold${fold.expanded ? " is-expanded" : ""}${fold.errorCount > 0 ? " has-errors" : ""}`}
              style={{ paddingLeft: entry.depth * 18 }}
              data-fold={fold.kind}
              data-count={fold.count}
            >
              <button
                type="button"
                className="fleet-tree-hit"
                aria-expanded={fold.expanded}
                aria-label={text}
                title={`${text}${fold.previews.length > 0 ? ` · ${fold.previews.join(" / ")}` : ""}`}
                onClick={() => onToggleFold(node.id, fold.expanded)}
              >
                <span className="fleet-tree-chevron" aria-hidden>
                  {fold.expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
                </span>
                <span className="fleet-tree-label">{text}</span>
                {fold.errorCount > 0 && (
                  <span className="fleet-tree-errors">{t("contextview.radialErrors", { n: fold.errorCount })}</span>
                )}
                {fold.previews.length > 0 && (
                  <span className="fleet-tree-previews">{fold.previews.join(" / ")}</span>
                )}
              </button>
            </div>
          );
        }
        const root = node.kind === "root";
        const selected = selectedAgent !== null && (selectedAgent === "main" ? root : selectedAgent === node.id);
        const labelText = node.task || node.name || node.id;
        const row = rowByRef.get(node.id);
        const pct = windowSize > 0 ? Math.min(100, Math.round((node.tokens / windowSize) * 100)) : null;
        return (
          <div
            key={node.id}
            className={`fleet-tree-row${root ? " is-root" : ""}${selected ? " is-selected" : ""}`}
            style={{ paddingLeft: entry.depth * 18, "--agent-depth-hue": depthHue(entry.depth) } as React.CSSProperties}
            data-agent={node.id}
            data-status={node.status}
          >
            <button
              type="button"
              className="fleet-tree-hit"
              aria-pressed={selected}
              aria-label={labelText}
              title={labelText}
              onClick={() => {
                if (onSelectAgent === undefined) return;
                const ref = root ? "main" : node.id;
                onSelectAgent(selectedAgent === ref ? null : ref);
              }}
            >
              <span className="fleet-tree-dot" data-status={node.status} aria-hidden />
              <span className="fleet-tree-label" title={labelText}>{labelText}</span>
              <span className="fleet-tree-status" data-status={node.status}>{statusText(node.status)}</span>
              {node.model && <span className="fleet-tree-model">{node.model}</span>}
              <span className="fleet-tree-metrics">
                {fmtTokens(node.tokens)} tok
                {node.toolCalls > 0 ? ` · ${node.toolCalls} ${t("fleet.kpiToolCalls")}` : ""}
                {row?.messages ? ` · ${row.messages} ${t("fleet.kpiMessages")}` : ""}
              </span>
            </button>
            {!root && onViewContext !== undefined && (
              <button
                type="button"
                className="fleet-tree-open"
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
            {pct !== null && pct > 0 && <span className="fleet-tree-pct">{pct}%</span>}
          </div>
        );
      })}
    </div>
  );
}
