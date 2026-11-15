import { useMemo, useRef, useState, type CSSProperties, type UIEvent } from "react";
import { GitBranch } from "../../icons";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import type {
  FamilyAgentRowView,
  FamilyEventStreamView,
  FamilyEventView,
} from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { agentHue } from "./fleetVisuals";
import "./fleet-timeline.css";

const LANE_HEIGHT = 40;
const EVENT_SLOT_MIN = 46;
const TRACK_MIN_WIDTH = 520;
export interface FleetTimelineProps {
  runs: SubagentRunView[];
  family: FamilyEventStreamView | null;
  agents: FamilyAgentRowView[];
  selectedAgent: string | null;
  selectedEventKey: string | null;
  onSelectAgent: (ref: string | null) => void;
  onSelectEvent: (eventKey: string) => void;
}

function familyEventKey(event: FamilyEventView): string {
  return `${event.agent}-${event.kind}-${event.seq}`;
}

function shortText(text: string | undefined, max = 42): string {
  const first = (text ?? "").split("\n")[0] ?? "";
  return first.length > max ? `${first.slice(0, max)}…` : first;
}

function fmtCost(value: number | undefined): string | null {
  if (value === undefined || value <= 0) return null;
  return `¥${value < 0.01 ? "<0.01" : value.toFixed(2)}`;
}

interface LaneDescriptor {
  ref: string;
  name: string;
  role: string;
  model?: string;
  usage: string;
  counts: string;
  hue: string;
  run?: SubagentRunView;
  stats?: FamilyAgentRowView;
}

export function FleetTimeline({
  runs,
  family,
  agents,
  selectedAgent,
  selectedEventKey,
  onSelectAgent,
  onSelectEvent,
}: FleetTimelineProps) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [latest, setLatest] = useState(false);
  const [hoveredAgent, setHoveredAgent] = useState<string | null>(null);
  const labelsRef = useRef<HTMLDivElement | null>(null);
  const trackRef = useRef<HTMLDivElement | null>(null);

  const orderedRuns = useMemo(
    () => [...runs].sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt)),
    [runs],
  );
  const statsByRef = useMemo(
    () => new Map(agents.map((agent) => [agent.ref, agent])),
    [agents],
  );
  const allRefs = useMemo(() => {
    const refs = ["main"];
    for (const run of orderedRuns) {
      if (!refs.includes(run.ref)) refs.push(run.ref);
    }
    return refs;
  }, [orderedRuns]);
  const laneDescriptors = useMemo(() => {
    const x = family?.totals ?? {
      rootTokens: 0,
      rootCost: undefined,
      subTokens: 0,
      subCost: undefined,
      subCount: 0,
      outputTokens: undefined,
    };
    return allRefs.map((ref, index): LaneDescriptor => {
      const run = orderedRuns.find((entry) => entry.ref === ref);
      const stats = statsByRef.get(ref);
      const isMain = ref === "main";
      const tokens = stats?.tokens ?? (isMain ? x.rootTokens : 0);
      const cost = stats?.cost ?? (isMain ? x.rootCost : undefined);
      const messages = stats?.messages ?? 0;
      const task = shortText(run?.task || (isMain ? t("fleet.agentMain") : ref), isMain ? 32 : 46);
      const usage = [
        tokens > 0 ? `${fmtTokens(tokens)} tok` : null,
        fmtCost(cost),
      ].filter((part): part is string => part !== null).join(" · ");
      return {
        ref,
        name: task,
        role: isMain ? t("fleet.roleLead") : t("fleet.roleSub"),
        model: run?.model,
        usage: usage || `0 tok`,
        counts: isMain
          ? `${(family?.events ?? []).filter((event) => event.agent === "main").length} ${t("fleet.evMessage")}`
          : `${messages} ${t("fleet.evMessage")} · ${run?.toolCalls ?? 0} ${t("fleet.kpiToolCalls")}`,
        hue: agentHue(ref, index - 1),
        run,
        stats,
      };
    });
  }, [allRefs, family?.events, family?.totals, orderedRuns, statsByRef, t]);

  const needle = query.trim().toLowerCase();
  const scopedLanes = useMemo(() => {
    const selected = selectedAgent === null
      ? laneDescriptors
      : laneDescriptors.filter((lane) => lane.ref === "main" || lane.ref === selectedAgent);
    if (needle === "") return selected;
    return selected.filter((lane) => (
      `${lane.name} ${lane.ref} ${lane.model ?? ""} ${lane.run?.task ?? ""}`
        .toLowerCase()
        .includes(needle)
    ));
  }, [laneDescriptors, needle, selectedAgent]);
  const visibleRefs = useMemo(() => new Set(scopedLanes.map((lane) => lane.ref)), [scopedLanes]);
  const visibleEvents = useMemo(
    () => (family?.events ?? []).filter((event) => visibleRefs.has(event.agent)),
    [family?.events, visibleRefs],
  );
  const laneIndex = useMemo(
    () => new Map(scopedLanes.map((lane, index) => [lane.ref, index])),
    [scopedLanes],
  );
  const eventCount = Math.max(1, visibleEvents.length);
  const trackWidth = Math.max(TRACK_MIN_WIDTH, eventCount * EVENT_SLOT_MIN);
  const eventX = (index: number): number => ((index + 0.5) / eventCount) * trackWidth;
  const trackHeight = Math.max(scopedLanes.length * LANE_HEIGHT, 300);
  const totalRecords = family?.events.length ?? 0;

  const syncVerticalScroll = (event: UIEvent<HTMLDivElement>) => {
    if (labelsRef.current !== null) labelsRef.current.scrollTop = event.currentTarget.scrollTop;
    const max = event.currentTarget.scrollWidth - event.currentTarget.clientWidth;
    if (max <= 1) return;
    setLatest(event.currentTarget.scrollLeft >= max - 1 && event.currentTarget.scrollLeft > 0);
  };

  const jumpTo = (target: "first" | "latest") => {
    setLatest(target === "latest");
    const track = trackRef.current;
    if (track === null) return;
    track.scrollLeft = target === "latest" ? track.scrollWidth : 0;
  };

  return (
    <section className="ctx-card fleet-timeline-card" data-testid="fleet-lanes">
      <div className="fleet-timeline-head">
        <span className="fleet-timeline-title">
          <GitBranch size={14} aria-hidden />
          {t("fleet.lanesTitle")}
        </span>
        <span className="fleet-timeline-meta">
          {t("fleet.lanesSummary", {
            shown: visibleEvents.length,
            total: totalRecords,
            lanes: scopedLanes.length,
          })}
        </span>
        <div className="fleet-timeline-tools">
          <label className="fleet-timeline-search">
            <span className="sr-only">{t("fleet.lanesSearch")}</span>
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("fleet.lanesSearch")}
            />
          </label>
          <span className="fleet-timeline-segmented" role="group">
            <button type="button" aria-pressed={!latest} onClick={() => jumpTo("first")}>
              {t("fleet.first")}
            </button>
            <button type="button" aria-pressed={latest} onClick={() => jumpTo("latest")}>
              {t("fleet.latest")}
            </button>
          </span>
        </div>
      </div>

      {scopedLanes.length === 0 ? (
        <div className="fleet-timeline-empty">{t("fleet.none")}</div>
      ) : (
        <div className="fleet-timeline-waterfall">
          <div className="fleet-timeline-labels" ref={labelsRef} aria-label={t("fleet.lanesTitle")}>
            {scopedLanes.map((lane) => {
              const active = selectedAgent === lane.ref;
              const dim = hoveredAgent !== null && hoveredAgent !== lane.ref;
              return (
                <div
                  key={lane.ref}
                  className={`fleet-timeline-label${active ? " is-selected" : ""}${dim ? " is-dim" : ""}`}
                  style={{ "--fleet-lane-hue": lane.hue } as CSSProperties}
                  onMouseEnter={() => setHoveredAgent(lane.ref)}
                  onMouseLeave={() => setHoveredAgent(null)}
                >
                  <button
                    type="button"
                    className="fleet-timeline-agent"
                    aria-pressed={active}
                    onClick={() => onSelectAgent(active ? null : lane.ref)}
                  >
                    <span className="fleet-timeline-avatar" aria-hidden>
                      {lane.name.slice(0, 2)}
                    </span>
                    <span className="fleet-timeline-lane-copy">
                      <span className="fleet-timeline-lane-head">
                        <b title={lane.name}>{lane.name}</b>
                        <em>{lane.role}</em>
                      </span>
                      <span className="fleet-timeline-lane-model">
                        {lane.model || (lane.ref === "main" ? t("fleet.agentMain") : lane.ref)}
                      </span>
                      <span className="fleet-timeline-lane-stats">
                        <span>{lane.usage}</span>
                        <span>{lane.counts}</span>
                      </span>
                    </span>
                  </button>
                </div>
              );
            })}
          </div>

          <div
            className="fleet-timeline-track-scroll"
            ref={trackRef}
            tabIndex={0}
            aria-label={t("fleet.lanesTitle")}
            onScroll={syncVerticalScroll}
          >
            <div
              className="fleet-timeline-track"
              style={{ width: trackWidth, height: trackHeight }}
            >
              <div className="fleet-timeline-columns" aria-hidden>
                {visibleEvents.map((event, index) => (
                  <i key={`column-${familyEventKey(event)}`} style={{ left: eventX(index) }} />
                ))}
              </div>
              {scopedLanes.map((lane, index) => (
                <div
                  key={`row-${lane.ref}`}
                  className="fleet-timeline-row"
                  style={{
                    top: index * LANE_HEIGHT,
                    height: LANE_HEIGHT,
                    "--fleet-lane-hue": lane.hue,
                  } as CSSProperties}
                />
              ))}
              <svg
                className="fleet-timeline-links"
                width={trackWidth}
                height={trackHeight}
                viewBox={`0 0 ${trackWidth} ${trackHeight}`}
                aria-hidden
              >
                {visibleEvents.map((event, index) => {
                  const targetIndex = laneIndex.get(event.agent);
                  const mainIndex = laneIndex.get("main");
                  if (
                    targetIndex === undefined ||
                    mainIndex === undefined ||
                    targetIndex === mainIndex ||
                    (event.kind !== "dispatch" && event.kind !== "report")
                  ) {
                    return null;
                  }
                  const x = eventX(index);
                  const fromY = mainIndex * LANE_HEIGHT + LANE_HEIGHT / 2;
                  const toY = targetIndex * LANE_HEIGHT + LANE_HEIGHT / 2;
                  const lane = scopedLanes[targetIndex];
                  const bend = Math.min(48, Math.max(20, Math.abs(toY - fromY) * 0.42));
                  const direction = toY >= fromY ? 1 : -1;
                  return (
                    <path
                      key={`link-${familyEventKey(event)}`}
                      d={`M ${x} ${fromY} C ${x} ${fromY + bend * direction}, ${x} ${toY - bend * direction}, ${x} ${toY}`}
                      stroke={lane?.hue ?? "var(--fleet-cyan, var(--info))"}
                      className={`fleet-timeline-link is-${event.kind}`}
                    />
                  );
                })}
              </svg>
              {visibleEvents.map((event, index) => {
                const row = laneIndex.get(event.agent);
                if (row === undefined) return null;
                const lane = scopedLanes[row];
                const key = familyEventKey(event);
                const selected = selectedEventKey === key;
                const kindLabel = event.kind === "dispatch"
                  ? t("fleet.evDispatch")
                  : event.kind === "report"
                    ? t("fleet.evReport")
                    : event.role === "user"
                      ? t("fleet.roleUser")
                      : t("fleet.evMessage");
                return (
                  <button
                    key={key}
                    type="button"
                    className={`fleet-timeline-event is-${event.kind}${selected ? " is-selected" : ""}`}
                    style={{
                      left: eventX(index),
                      top: row * LANE_HEIGHT + LANE_HEIGHT / 2,
                      "--fleet-lane-hue": lane?.hue ?? "var(--fleet-cyan, var(--info))",
                    } as CSSProperties}
                    data-event-key={key}
                    aria-pressed={selected}
                    aria-label={`${kindLabel}: ${shortText(event.text, 70)}`}
                    title={`${kindLabel} · ${shortText(event.text, 70)}`}
                    onClick={() => onSelectEvent(key)}
                  >
                    <span aria-hidden>
                      {event.kind === "dispatch" ? "+" : event.kind === "report" ? "✓" : ""}
                    </span>
                  </button>
                );
              })}
              {(family?.events.length ?? 0) === 0 && (
                <div className="fleet-timeline-track-empty">{t("fleet.noEvents")}</div>
              )}
            </div>
          </div>
        </div>
      )}
      {family?.truncated ? (
        <div className="fleet-timeline-note">{t("fleet.truncated", { count: family.truncated })}</div>
      ) : null}
    </section>
  );
}
