import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { FamilyEventStreamView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { FleetEvents } from "./FleetEvents";

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const RUNS: SubagentRunView[] = [
  {
    ref: "sa_1",
    status: "completed",
    task: "双标尺与时标刻度内核",
    toolCalls: 8,
    createdAt: "2026-11-14T09:00:00Z",
    updatedAt: "2026-11-14T09:20:00Z",
  },
  {
    ref: "sa_2",
    status: "running",
    task: "就地编辑族与列级筛选",
    toolCalls: 6,
    createdAt: "2026-11-14T09:05:00Z",
    updatedAt: "2026-11-14T09:25:00Z",
  },
];

const FAMILY: FamilyEventStreamView = {
  ok: true,
  events: [
    { seq: 10, ts: 1763100000, agent: "main", kind: "dispatch", text: "双标尺与时标刻度内核" },
    { seq: 20, ts: 1763100060, agent: "sa_1", kind: "report", ref: "sa_1", text: "完成全部验收" },
    { seq: 30, ts: 1763100120, agent: "sa_2", kind: "message", role: "assistant", text: "正在处理列级筛选" },
  ],
  scanned: 2,
  refs: ["main", "sa_1", "sa_2"],
  totals: {
    rootTokens: 1000,
    subTokens: 500,
    subCount: 2,
  },
  agents: [
    { ref: "main", tokens: 1000, dispatches: 1, messages: 0 },
    { ref: "sa_1", tokens: 300, dispatches: 1, messages: 1, reports: 1 },
    { ref: "sa_2", tokens: 200, dispatches: 1, messages: 1 },
  ],
};

function Harness({
  selectedEventKey = null,
  onSelectEvent = vi.fn(),
  onViewContext = vi.fn(),
  onSelectAgent = vi.fn(),
  selectedAgent = null,
}: {
  selectedEventKey?: string | null;
  onSelectEvent?: (key: string | null) => void;
  onViewContext?: (ref: string) => void;
  onSelectAgent?: (ref: string | null) => void;
  selectedAgent?: string | null;
}) {
  return (
    <FleetEvents
      family={FAMILY}
      runs={RUNS}
      selectedAgent={selectedAgent}
      onSelectAgent={onSelectAgent}
      selectedEventKey={selectedEventKey}
      onSelectEvent={onSelectEvent}
      onViewContext={onViewContext}
    />
  );
}

describe("FleetEvents", () => {
  it("renders the first event in the inspector by default", () => {
    renderT(<Harness />);
    const panel = screen.getByTestId("fleet-events");
    expect(panel.textContent).toContain("事件检查");
    expect(panel.textContent).toContain("双标尺与时标刻度内核");
    expect(panel.textContent).toContain("序号 10");
    expect(panel.textContent).toContain("主代理");
  });

  it("filters by kind with counts and supports text search", () => {
    renderT(<Harness />);
    const panel = screen.getByTestId("fleet-events");
    fireEvent.click(panel.querySelector('[data-kind="report"]') as HTMLElement);
    expect(panel.querySelector('[data-event-key="sa_1-report-20"]')).toBeTruthy();
    expect(panel.querySelector('[data-event-key="main-dispatch-10"]')).toBeNull();

    const search = screen.getByPlaceholderText("搜索事件…");
    fireEvent.change(search, { target: { value: "没有匹配" } });
    expect(panel.textContent).toContain("当前筛选下无记录");
  });

  it("opens the selected report in context", () => {
    const onViewContext = vi.fn();
    renderT(
      <Harness
        selectedEventKey="sa_1-report-20"
        onViewContext={onViewContext}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /打开会话/ }));
    expect(onViewContext).toHaveBeenCalledWith("sa_1");
  });

  it("closes the inspector while preserving the event list", () => {
    const onSelectEvent = vi.fn();
    renderT(<Harness onSelectEvent={onSelectEvent} />);
    fireEvent.click(screen.getByRole("button", { name: "关闭" }));
    expect(onSelectEvent).toHaveBeenCalledWith(null);
    expect(screen.getByTestId("fleet-events").textContent).toContain(
      "选择时间线节点或事件，查看完整记录。",
    );
  });

  it("shows an agent scope pill and clears the scope", () => {
    const onSelectAgent = vi.fn();
    renderT(
      <Harness
        selectedAgent="sa_2"
        onSelectAgent={onSelectAgent}
      />,
    );
    expect(screen.getByTestId("fleet-events").textContent).toContain("就地编辑族与列级筛选");
    fireEvent.click(screen.getByLabelText(/清除.*篩選|清除.*筛选/));
    expect(onSelectAgent).toHaveBeenCalledWith(null);
  });
});
