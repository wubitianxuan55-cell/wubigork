// 编队页集成测试：数据订阅、三块布局、Agent/事件选中态在页面内联动。
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { AgentNetwork } from "../../lib/types";
import type { SubagentRunView } from "../../lib/types/subagent";

vi.setConfig({ testTimeout: 20_000 });

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const NET: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "main",
    name: "主代理",
    task: "继续蒸馏斑马进度",
    kind: "root",
    status: "completed",
    toolCalls: 3,
    errors: 0,
    tokens: 120_000,
    firstTs: 1_760_000_000,
    lastTs: 1_760_000_300,
    children: [
      {
        id: "sa_1",
        name: "调研",
        task: "调研 dsh-context 功能面",
        kind: "subagent",
        status: "completed",
        toolCalls: 12,
        errors: 0,
        tokens: 30_000,
        firstTs: 1_760_000_010,
        lastTs: 1_760_000_310,
      },
      {
        id: "sa_2",
        name: "映射",
        task: "写映射表",
        kind: "subagent",
        status: "error",
        toolCalls: 2,
        errors: 1,
        tokens: 8_000,
      },
      {
        id: "sa_3",
        name: "探索",
        task: "长跑探索",
        kind: "subagent",
        status: "running",
        toolCalls: 5,
        errors: 0,
        tokens: 25_000,
      },
    ],
  },
};

const RUNS: SubagentRunView[] = [
  {
    ref: "sa_1",
    status: "completed",
    task: "调研 dsh-context 功能面",
    answer: "五入口+三投影，缺口四处",
    model: "deepseek-flash",
    toolCalls: 12,
    createdAt: "2026-11-12T10:00:00Z",
    updatedAt: "2026-11-12T10:05:00Z",
  },
  {
    ref: "sa_2",
    status: "failed",
    task: "写映射表",
    answer: "权限被拒",
    model: "deepseek-flash",
    toolCalls: 2,
    createdAt: "2026-11-12T10:02:00Z",
    updatedAt: "2026-11-12T10:03:00Z",
  },
  {
    ref: "sa_3",
    status: "running",
    task: "长跑探索",
    model: "deepseek-reasoner",
    toolCalls: 5,
    createdAt: "2026-11-12T10:06:00Z",
    updatedAt: "2026-11-12T10:08:00Z",
  },
];

vi.mock("../../lib/agentNetworkStore", () => ({
  subscribeAgentNetwork: (cb: (network: AgentNetwork) => void) => {
    cb(NET);
    return () => {};
  },
  reloadAgentNetwork: () => {},
}));

vi.mock("../../lib/subagentRunsStore", () => ({
  subscribeSubagentRuns: (
    _path: string,
    cb: (runs: SubagentRunView[], meta: unknown) => void,
  ) => {
    cb(RUNS, { status: "ready", total: RUNS.length, running: 1 });
    return () => {};
  },
  reloadSubagentRuns: () => {},
}));

const FAMILY = {
  ok: true,
  scanned: 4,
  refs: ["main", "sa_1", "sa_2", "sa_3"],
  agents: [
    { ref: "main", tokens: 120_000, cost: 0.42, dispatches: 3, messages: 3 },
    { ref: "sa_1", tokens: 30_000, cost: 0.11, dispatches: 1, messages: 4, reports: 1 },
    { ref: "sa_2", tokens: 8_000, dispatches: 1, messages: 2 },
    { ref: "sa_3", tokens: 25_000, cost: 0.08, dispatches: 1, messages: 1 },
  ],
  totals: {
    rootTokens: 120_000,
    rootCost: 0.42,
    subTokens: 63_000,
    subCost: 0.19,
    subCount: 3,
  },
  events: [
    { seq: 1, ts: 1_760_000_000, agent: "main", kind: "message", role: "user", text: "拆报表" },
    { seq: 2, ts: 1_760_000_001, agent: "main", kind: "dispatch", text: "调研 dsh-context 功能面" },
    { seq: 3, ts: 1_760_000_002, agent: "sa_1", kind: "dispatch", text: "调研 dsh-context 功能面" },
    { seq: 4, ts: 1_760_000_003, agent: "sa_1", kind: "report", ref: "sa_1", text: "拆完了" },
    { seq: 5, ts: 1_760_000_004, agent: "main", kind: "message", role: "assistant", text: "完成" },
  ],
};

vi.mock("../../lib/bridge", () => ({
  app: { FamilyEvents: vi.fn(() => Promise.resolve(FAMILY)) },
}));

import { Fleet } from "./Fleet";

describe("Fleet 编队页", () => {
  it("摘要条统计 Agent、运行中、工作项与真实消息数", async () => {
    renderT(<Fleet />);
    expect(screen.getByTestId("fleet-hero")).toBeTruthy();
    const summary = screen.getByTestId("fleet-summary");
    expect(summary.textContent).toContain("4");
    await waitFor(() => expect(summary.textContent).toContain("2"));
  });

  it("按参照图渲染左 Agent 网络、右时间线/任务与底部事件检查", async () => {
    renderT(<Fleet />);
    const main = screen.getByTestId("fleet-main");
    expect(main.children).toHaveLength(2);
    expect(main.children[0].querySelector('[data-testid="agent-topology"]')).toBeTruthy();
    await waitFor(() => expect(main.children[1].querySelector('[data-testid="fleet-lanes"]')).toBeTruthy());
    await waitFor(() => expect(main.children[1].querySelector('[data-testid="fleet-tasks"]')).toBeTruthy());
    expect(screen.getByTestId("fleet-events")).toBeTruthy();
  });

  it("事件检查默认可检查首条记录，并同步时间线节点选中态", async () => {
    renderT(<Fleet />);
    const events = await screen.findByTestId("fleet-events");
    expect(events.textContent).toContain("拆报表");

    const node = events.querySelector('[data-event-key="sa_1-report-4"]') as HTMLElement;
    fireEvent.click(node);
    expect(node.getAttribute("aria-pressed")).toBe("true");
    expect(events.textContent).toContain("拆完了");
  });

  it("网络节点选择会把筛选态同步给时间线、任务与事件检查", async () => {
    renderT(<Fleet />);
    const topology = screen.getByTestId("agent-topology");
    const hit = topology.querySelector('[data-agent="sa_1"] .agent-topology-hit') as HTMLElement;
    fireEvent.click(hit);

    expect(screen.getByTestId("fleet-hero").textContent).toContain("已筛选");
    const lanes = await screen.findByTestId("fleet-lanes");
    expect(lanes.textContent).toContain("调研 dsh-context 功能面");
    expect(lanes.textContent).not.toContain("写映射表");
    await waitFor(() => {
      expect(screen.getByTestId("fleet-events").textContent).toContain("调研 dsh-context 功能面");
    });
  });
});
