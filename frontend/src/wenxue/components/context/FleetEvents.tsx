import { useCallback, useMemo, useState } from "react";
import {
  ArrowUpRight,
  Bot,
  List,
  MessageSquare,
  User,
  X,
} from "../../icons";
import { useT } from "../../lib/i18n";
import type {
  FamilyAgentKind,
  FamilyEventStreamView,
  FamilyEventView,
} from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import "./fleet-events.css";

type KindFilter = "all" | FamilyAgentKind;

export interface FleetEventsProps {
  family: FamilyEventStreamView | null;
  runs: SubagentRunView[];
  selectedAgent: string | null;
  onSelectAgent: (ref: string | null) => void;
  selectedEventKey: string | null;
  onSelectEvent: (eventKey: string | null) => void;
  onViewContext?: (ref: string) => void;
}

interface EventRowView {
  event: FamilyEventView;
  key: string;
  sender: string;
  recipient: string;
  senderRef: string | null;
  recipientRef: string | null;
  senderSub: string;
  recipientSub: string;
  summary: string;
}

function familyEventKey(event: FamilyEventView): string {
  return `${event.agent}-${event.kind}-${event.seq}`;
}

export function FleetEvents({
  family,
  runs,
  selectedAgent,
  onSelectAgent,
  selectedEventKey,
  onSelectEvent,
  onViewContext,
}: FleetEventsProps) {
  const t = useT();
  const [kind, setKind] = useState<KindFilter>("all");
  const [query, setQuery] = useState("");
  const [closedView, setClosedView] = useState<string | null>(null);
  const needle = query.trim().toLowerCase();

  const taskByRef = useMemo(
    () => new Map(runs.map((run) => [run.ref, run.task])),
    [runs],
  );
  const labelOf = useCallback((ref: string): string => (
    ref === "main"
      ? t("fleet.agentMain")
      : shortTask(taskByRef.get(ref) ?? ref, 18)
  ), [t, taskByRef]);
  const scopedRows = useMemo(() => {
    const rowOf = (event: FamilyEventView): EventRowView => {
      const participants = participantsOf(event, runs, labelOf);
      return {
        event,
        key: familyEventKey(event),
        ...participants,
        summary: event.text || "—",
      };
    };
    const rows = (family?.events ?? []).map(rowOf);
    if (selectedAgent === null) return rows;
    return rows.filter((row) => (
      row.event.agent === selectedAgent ||
      row.senderRef === selectedAgent ||
      row.recipientRef === selectedAgent
    ));
  }, [family, labelOf, runs, selectedAgent]);

  const counts = useMemo(() => ({
    all: scopedRows.length,
    message: scopedRows.filter((row) => row.event.kind === "message").length,
    dispatch: scopedRows.filter((row) => row.event.kind === "dispatch").length,
    report: scopedRows.filter((row) => row.event.kind === "report").length,
  }), [scopedRows]);

  const filteredRows = useMemo(() => {
    return scopedRows.filter((row) => {
      if (kind !== "all" && row.event.kind !== kind) return false;
      if (needle === "") return true;
      return [
        row.summary,
        row.sender,
        row.recipient,
        row.event.agent,
        kindLabel(row.event, t),
      ].some((value) => value.toLowerCase().includes(needle));
    });
  }, [kind, needle, scopedRows, t]);

  const partFingerprint = `${selectedAgent ?? "all"}|${kind}|${needle}`;
  const explicit = filteredRows.find((row) => row.key === selectedEventKey);
  const fallback = closedView === partFingerprint ? undefined : filteredRows[0];
  const shown = explicit ?? fallback;

  const selectRow = (row: EventRowView) => {
    if (shown?.key === row.key) {
      setClosedView(partFingerprint);
      onSelectEvent(null);
      return;
    }
    setClosedView(null);
    onSelectEvent(row.key);
  };

  const close = () => {
    setClosedView(partFingerprint);
    onSelectEvent(null);
  };

  return (
    <section className="ctx-card fleet-events-panel" data-testid="fleet-events">
      <div className="fleet-events-head">
        <span className="fleet-events-title">
          <List size={14} aria-hidden />
          {t("fleet.eventsTitle")}
        </span>
        <span className="fleet-events-meta">
          {t("fleet.eventSummary", {
            shown: filteredRows.length,
            total: scopedRows.length,
          })}
        </span>
        {family?.truncated ? (
          <span className="fleet-events-note">
            {t("fleet.truncated", { count: family.truncated })}
          </span>
        ) : null}
      </div>

      <div className="fleet-events-toolbar">
        <div className="fleet-events-kinds" role="group" aria-label={t("fleet.eventsTitle")}>
          {(["all", "message", "dispatch", "report"] as const).map((entry) => (
            <button
              key={entry}
              type="button"
              aria-pressed={kind === entry}
              data-kind={entry}
              onClick={() => setKind(entry)}
            >
              {kindName(entry, t)}
              <b>{counts[entry]}</b>
            </button>
          ))}
        </div>
        <label className="fleet-events-search">
          <span className="sr-only">{t("fleet.eventSearch")}</span>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("fleet.eventSearch")}
          />
        </label>
        {selectedAgent !== null ? (
          <button
            type="button"
            className="fleet-events-agent-pill"
            aria-label={t("fleet.clearFilter", { name: labelOf(selectedAgent) })}
            title={t("fleet.clearFilter", { name: labelOf(selectedAgent) })}
            onClick={() => onSelectAgent(null)}
          >
            <span>{labelOf(selectedAgent)}</span>
            <X size={10} aria-hidden />
          </button>
        ) : null}
      </div>

      <div className="fleet-events-layout">
        <div className="fleet-events-stream">
          {filteredRows.length === 0 ? (
            <div className="fleet-events-empty">
              <MessageSquare size={19} aria-hidden />
              <span>{t((family?.events.length ?? 0) === 0 ? "fleet.noEvents" : "fleet.none")}</span>
            </div>
          ) : (
            <ul className="fleet-events-list">
              {filteredRows.map((row) => (
                <li key={row.key}>
                  <button
                    type="button"
                    className={`fleet-event-card is-${row.event.kind}${shown?.key === row.key ? " is-open" : ""}`}
                    data-event-key={row.key}
                    aria-pressed={shown?.key === row.key}
                    onClick={() => selectRow(row)}
                  >
                    <span className="fleet-event-meta">
                      <time>{fmtClock(row.event.ts)}</time>
                      <span>{kindLabel(row.event, t)}</span>
                      <b>#{row.event.seq}</b>
                    </span>
                    <span className="fleet-event-route">
                      <span>{row.sender}</span>
                      <i aria-hidden>→</i>
                      <span>{row.recipient}</span>
                    </span>
                    <span className="fleet-event-summary">{row.summary}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="fleet-events-inspector">
          {shown ? (
            <>
              <div className="fleet-events-detail-head">
                <span className={`fleet-event-kind is-${shown.event.kind}`}>
                  {kindLabel(shown.event, t)}
                </span>
                <time>{fmtDateTime(shown.event.ts)}</time>
                <span className="fleet-events-seq">{t("fleet.recordSeq", { n: shown.event.seq })}</span>
                <button
                  type="button"
                  className="fleet-events-close"
                  aria-label={t("common.close")}
                  title={t("common.close")}
                  onClick={close}
                >
                  <X size={14} aria-hidden />
                </button>
              </div>

              <div className="fleet-events-participants-head">{t("fleet.participants")}</div>
              <div className={`fleet-events-route is-${shown.event.kind}`}>
                <Participant name={shown.sender} sub={shown.senderSub} side="sender" />
                <span className="fleet-events-route-arrow" aria-hidden>→</span>
                <Participant name={shown.recipient} sub={shown.recipientSub} side="recipient" />
              </div>

              <div className="fleet-events-detail-body">
                <p>{shown.summary}</p>
              </div>

              <div className="fleet-events-detail-foot">
                <span>{kindLabel(shown.event, t)}</span>
                {shown.event.kind === "report" && onViewContext ? (
                  <button
                    type="button"
                    className="fleet-events-open"
                    onClick={() => onViewContext(shown.event.ref || shown.event.agent)}
                  >
                    <ArrowUpRight size={12} aria-hidden />
                    {t("fleet.open")}
                  </button>
                ) : null}
              </div>
            </>
          ) : (
            <div className="fleet-events-empty is-inspector">
              <List size={19} aria-hidden />
              <span>{t("fleet.selectEvent")}</span>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function Participant({
  name,
  sub,
  side,
}: {
  name: string;
  sub: string;
  side: "sender" | "recipient";
}) {
  const t = useT();
  const Icon = sub === "user" || sub === "assistant"
    ? User
    : side === "sender"
      ? MessageSquare
      : Bot;
  return (
    <span className="fleet-events-participant">
      <i aria-hidden><Icon size={14} /></i>
      <span>
        <small className="fleet-events-participant-side">
          {t(side === "sender" ? "fleet.sender" : "fleet.recipient")}
        </small>
        <b>{name}</b>
        {sub !== "" ? <small>{sub}</small> : null}
      </span>
    </span>
  );
}

function participantsOf(
  event: FamilyEventView,
  runs: SubagentRunView[],
  labelOf: (ref: string) => string,
): Pick<EventRowView, "sender" | "recipient" | "senderRef" | "recipientRef" | "senderSub" | "recipientSub"> {
  const main = labelOf("main");
  if (event.kind === "report") {
    const ref = event.ref || event.agent;
    return {
      sender: labelOf(ref),
      recipient: main,
      senderRef: ref,
      recipientRef: "main",
      senderSub: ref,
      recipientSub: "main",
    };
  }
  if (event.kind === "dispatch") {
    const ref = event.agent === "main"
      ? matchRunRef(event.text ?? "", runs)
      : event.agent;
    if (event.agent === "main") {
      return {
        sender: main,
        recipient: ref ? labelOf(ref) : "—",
        senderRef: "main",
        recipientRef: ref,
        senderSub: "main",
        recipientSub: ref ?? "",
      };
    }
    return {
      sender: main,
      recipient: labelOf(ref || event.agent),
      senderRef: "main",
      recipientRef: ref || event.agent,
      senderSub: "main",
      recipientSub: ref || event.agent,
    };
  }
  if (event.agent === "main") {
    return event.role === "user"
      ? {
        sender: labelOf("main"),
        recipient: labelOf("main"),
        senderRef: "main",
        recipientRef: "main",
        senderSub: "user",
        recipientSub: "assistant",
      }
      : {
        sender: labelOf("main"),
        recipient: labelOf("main"),
        senderRef: "main",
        recipientRef: "main",
        senderSub: "assistant",
        recipientSub: "user",
      };
  }
  return event.role === "user"
    ? {
      sender: main,
      recipient: labelOf(event.agent),
      senderRef: "main",
      recipientRef: event.agent,
      senderSub: "main",
      recipientSub: event.agent,
    }
    : {
      sender: labelOf(event.agent),
      recipient: main,
      senderRef: event.agent,
      recipientRef: "main",
      senderSub: event.agent,
      recipientSub: "main",
    };
}

function matchRunRef(text: string, runs: SubagentRunView[]): string | null {
  const normalized = normalize(text);
  if (normalized === "") return null;
  const match = runs.find((run) => {
    const task = normalize(run.task);
    if (task === "") return false;
    return normalized.includes(task) || task.includes(normalized);
  });
  return match?.ref ?? null;
}

function normalize(value: string): string {
  return value.replace(/\s+/g, " ").trim().toLowerCase();
}

function shortTask(task: string, max: number): string {
  const first = task.split("\n")[0]?.trim() ?? task;
  return first.length > max ? `${first.slice(0, max)}…` : first;
}

function kindName(kind: KindFilter, t: ReturnType<typeof useT>): string {
  if (kind === "all") return t("fleet.filterAll");
  if (kind === "message") return t("fleet.evMessage");
  if (kind === "dispatch") return t("fleet.evDispatch");
  return t("fleet.evReport");
}

function kindLabel(event: FamilyEventView, t: ReturnType<typeof useT>): string {
  if (event.kind === "message") {
    return event.role === "user" ? t("fleet.roleUser") : t("fleet.roleAssistant");
  }
  return event.kind === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport");
}

function fmtClock(ts: number): string {
  const date = new Date(ts * 1000);
  return Number.isFinite(date.getTime())
    ? date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : "—";
}

function fmtDateTime(ts: number): string {
  const date = new Date(ts * 1000);
  return Number.isFinite(date.getTime())
    ? date.toLocaleString([], {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
    : "—";
}
