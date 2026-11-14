import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../lib/i18n";
import type { FamilyEventStreamView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { FleetTimeline } from "./FleetTimeline";

const runs: SubagentRunView[] = [
  {
    ref: "sa_1",
    status: "completed",
    task: "数据模型与接口",
    model: "deepseek-chat",
    toolCalls: 12,
    createdAt: "2026-11-14T09:00:00Z",
    updatedAt: "2026-11-14T09:20:00Z",
  },
  {
    ref: "sa_2",
    status: "running",
    task: "事件面板",
    model: "deepseek-reasoner",
    toolCalls: 4,
    createdAt: "2026-11-14T09:03:00Z",
    updatedAt: "2026-11-14T09:18:00Z",
  },
];

const family: FamilyEventStreamView = {
  ok: true,
  events: [
    { seq: 1, ts: 1_760_000_000, agent: "main", kind: "message", role: "user", text: "请并行完成编队页" },
    { seq: 2, ts: 1_760_000_010, agent: "main", kind: "dispatch", text: "task 数据模型与接口" },
    { seq: 3, ts: 1_760_000_020, agent: "sa_1", kind: "dispatch", text: "数据模型与接口" },
    { seq: 4, ts: 1_760_000_030, agent: "sa_2", kind: "dispatch", text: "事件面板" },
    { seq: 5, ts: 1_760_000_040, agent: "sa_1", kind: "report", ref: "sa_1", text: "已完成并在等待验收" },
  ],
  scanned: 3,
  refs: ["main", "sa_1", "sa_2"],
  totals: {
    rootTokens: 120_000,
    rootCost: 0.42,
    subTokens: 42_000,
    subCost: 0.18,
    subCount: 2,
  },
  agents: [
    { ref: "main", tokens: 120_000, cost: 0.42, dispatches: 2, messages: 3 },
    { ref: "sa_1", tokens: 30_000, cost: 0.12, dispatches: 1, messages: 4, reports: 1 },
    { ref: "sa_2", tokens: 12_000, cost: 0.06, dispatches: 1, messages: 2 },
  ],
};

function renderTimeline(overrides: Partial<React.ComponentProps<typeof FleetTimeline>> = {}) {
  const props: React.ComponentProps<typeof FleetTimeline> = {
    runs,
    family,
    agents: family.agents ?? [],
    selectedAgent: null,
    selectedEventKey: null,
    onSelectAgent: vi.fn(),
    onSelectEvent: vi.fn(),
    ...overrides,
  };
  const view = render(
    <LocaleProvider>
      <FleetTimeline {...props} />
    </LocaleProvider>,
  );
  return { ...view, props };
}

describe("FleetTimeline", () => {
  it("渲染标题、泳道摘要、代理标签与事件节点", () => {
    renderTimeline();
    expect(screen.getByText("跨 Agent 协作")).toBeTruthy();
    expect(screen.getByText(/5\/5 条记录 · 3 条泳道/)).toBeTruthy();
    expect(screen.getByText("数据模型与接口")).toBeTruthy();
    expect(screen.getByText("事件面板")).toBeTruthy();
    expect(document.querySelectorAll(".fleet-timeline-event")).toHaveLength(5);
  });

  it("搜索无匹配代理时显示空态", () => {
    renderTimeline();
    fireEvent.change(screen.getByPlaceholderText("搜索 Agent…"), { target: { value: "不存在的代理" } });
    expect(screen.getByText("当前筛选下无记录。")).toBeTruthy();
  });

  it("点击泳道标签和事件节点触发对应回调", () => {
    const { props } = renderTimeline();
    fireEvent.click(screen.getByText("数据模型与接口").closest("button") as HTMLElement);
    expect(props.onSelectAgent).toHaveBeenCalledWith("sa_1");

    fireEvent.click(document.querySelector('[data-event-key="sa_1-dispatch-3"]') as HTMLElement);
    expect(props.onSelectEvent).toHaveBeenCalledWith("sa_1-dispatch-3");
  });

  it("选中代理时只保留主代理与该代理泳道，并标记选中态", () => {
    renderTimeline({ selectedAgent: "sa_1" });
    expect(screen.getByText("数据模型与接口")).toBeTruthy();
    expect(screen.queryByText("事件面板")).toBeNull();
    expect(screen.getByText("数据模型与接口").closest("button")?.getAttribute("aria-pressed")).toBe("true");
  });
});
