// 编队页：对齐 dsh-context 0.84.1 MissionPulse。
// 左 Agent 网络、右跨 Agent 时间线与协作任务、底部事件检查共享同一
// Agent 筛选与事件选中态；数据继续使用 GaeaAgentNetwork / SubagentRuns /
// GaeaFamilyEvents 三处既有投影，不伪造后端没有的团队任务字段。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Users, X } from "../../icons";
import { app } from "../../lib/bridge";
import { useT } from "../../lib/i18n";
import { subscribeAgentNetwork } from "../../lib/agentNetworkStore";
import { subscribeSubagentRuns, type SubagentRunsMeta } from "../../lib/subagentRunsStore";
import type { AgentNetwork } from "../../lib/types";
import type { FamilyEventStreamView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { AgentTopology } from "./AgentTopology";
import { flattenAgentTree } from "./agentTopologyLayout";
import { FleetEvents } from "./FleetEvents";
import { FleetTasks } from "./FleetTasks";
import { FleetTimeline } from "./FleetTimeline";
import "./fleet-view.css";

function shortTask(task: string, max = 20): string {
  const one = task.split("\n")[0] ?? task;
  return one.length > max ? `${one.slice(0, max)}…` : one;
}

export function Fleet({ sessionPath, onViewContext, onOpenAside }: {
  running?: boolean;
  sessionPath?: string;
  onViewContext?: (ref: string) => void;
  /** 「在侧边栏打开」（2026-11-18 分工卡/拓扑卡→右栏子会话窗）。 */
  onOpenAside?: (ref: string, title?: string) => void;
}) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork | null>(null);
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [meta, setMeta] = useState<SubagentRunsMeta>({
    status: "loading",
    total: 0,
    running: 0,
  });
  const [family, setFamily] = useState<FamilyEventStreamView | null>(null);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [selectedEventKey, setSelectedEventKey] = useState<string | null>(null);
  const eventsRef = useRef<HTMLDivElement | null>(null);

  useEffect(
    () => subscribeAgentNetwork((network) => setNet(network), { path: sessionPath }),
    [sessionPath],
  );
  useEffect(
    () => subscribeSubagentRuns(sessionPath ?? "", (nextRuns, nextMeta) => {
      setRuns(nextRuns);
      setMeta(nextMeta);
    }),
    [sessionPath],
  );

  const loadFamily = useCallback(() => {
    app.FamilyEvents(sessionPath ?? "")
      .then(setFamily)
      .catch(() => setFamily(null));
  }, [sessionPath]);
  useEffect(() => {
    loadFamily();
  }, [loadFamily]);

  const subagentRuns = useMemo(
    () => runs.filter((run) => (run.kind ?? "subagent") === "subagent"),
    [runs],
  );
  const familyAgents = family?.agents ?? [];
  const topologyCount = net === null ? 0 : flattenAgentTree(net.root).length;
  const agentCount = Math.max(1 + subagentRuns.length, familyAgents.length, topologyCount);
  const messageCount = family?.events.filter((event) => event.kind === "message").length ?? 0;
  const knownRefs = useMemo(() => {
    const refs = new Set<string>(["main", ...(family?.refs ?? [])]);
    for (const run of subagentRuns) refs.add(run.ref);
    return refs;
  }, [family?.refs, subagentRuns]);

  useEffect(() => {
    if (selectedAgent !== null && !knownRefs.has(selectedAgent)) {
      setSelectedAgent(null);
      setSelectedEventKey(null);
    }
  }, [knownRefs, selectedAgent]);

  const chooseAgent = useCallback((ref: string | null) => {
    setSelectedAgent(ref);
    setSelectedEventKey(null);
  }, []);

  const openEvent = useCallback((eventKey: string | null) => {
    setSelectedEventKey(eventKey);
    if (eventKey !== null) {
      eventsRef.current?.scrollIntoView?.({ block: "nearest", behavior: "smooth" });
    }
  }, []);

  const selectedAgentName = selectedAgent === null
    ? ""
    : selectedAgent === "main"
      ? t("fleet.agentMain")
      : shortTask(subagentRuns.find((run) => run.ref === selectedAgent)?.task ?? selectedAgent, 18);

  return (
    <div className="fleet-view" data-testid="ctx-fleet">
      <header className="fleet-hero" data-testid="fleet-hero">
        <div className="fleet-hero-copy">
          <span className="fleet-eyebrow">{t("fleet.heroEyebrow")}</span>
          <div className="fleet-title-row">
            <span className="fleet-beacon" aria-hidden><i /><i /><i /></span>
            <h2>{t("fleet.heroTitle")}</h2>
            {selectedAgent !== null && (
              <button
                type="button"
                className="fleet-selection-pill"
                aria-label={t("fleet.clearFilter", { name: selectedAgentName })}
                title={t("fleet.clearFilter", { name: selectedAgentName })}
                onClick={() => chooseAgent(null)}
              >
                <span>{t("fleet.filteredTo", { name: selectedAgentName })}</span>
                <X size={11} aria-hidden />
              </button>
            )}
          </div>
          <p>{t("fleet.heroSubtitle")}</p>
        </div>
        <dl className="fleet-summary" data-testid="fleet-summary">
          <Summary label={t("fleet.kpiAgents")} value={String(agentCount)} />
          <Summary label={t("fleet.kpiRunning")} value={String(meta.running)} tone="running" />
          <Summary label={t("fleet.kpiTasks")} value={String(subagentRuns.length)} />
          <Summary label={t("fleet.kpiMessages")} value={String(messageCount)} tone="messages" />
        </dl>
      </header>

      <div className="fleet-main-grid" data-testid="fleet-main">
        <div className="fleet-network-col">
          {net ? (
            <AgentTopology
              network={net}
              agents={familyAgents}
              selectedAgent={selectedAgent}
              onSelectAgent={chooseAgent}
              onViewContext={onViewContext}
              onOpenAside={onOpenAside}
            />
          ) : (
            <section className="ctx-card fleet-network-empty">
              <div className="fleet-card-head">
                <span className="fleet-card-title">
                  <Users size={14} aria-hidden />
                  {t("fleet.networkTitle")}
                </span>
              </div>
              <div className="fleet-empty-state">
                <Users size={20} aria-hidden />
                <span>{t("fleet.empty")}</span>
              </div>
            </section>
          )}
        </div>

        <div className="fleet-side-stack">
          <FleetTimeline
            runs={subagentRuns}
            family={family}
            agents={familyAgents}
            selectedAgent={selectedAgent}
            selectedEventKey={selectedEventKey}
            onSelectAgent={chooseAgent}
            onSelectEvent={openEvent}
          />
          <FleetTasks
            runs={subagentRuns}
            agents={familyAgents}
            selectedAgent={selectedAgent}
            onSelectAgent={chooseAgent}
            onViewContext={onViewContext}
            onOpenAside={onOpenAside}
          />
        </div>
      </div>

      <div ref={eventsRef} className="fleet-events-wrap">
        <FleetEvents
          family={family}
          runs={subagentRuns}
          selectedAgent={selectedAgent}
          onSelectAgent={chooseAgent}
          selectedEventKey={selectedEventKey}
          onSelectEvent={setSelectedEventKey}
          onViewContext={onViewContext}
        />
      </div>
    </div>
  );
}

function Summary({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "running" | "messages";
}) {
  return (
    <div className="fleet-summary-stat">
      <dt>{label}</dt>
      <dd data-tone={tone}>{value}</dd>
    </div>
  );
}
