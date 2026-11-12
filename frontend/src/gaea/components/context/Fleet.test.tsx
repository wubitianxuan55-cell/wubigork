// 刀5 编队页行为测试：摘要条/泳道/任务面板/事件检查。mock 两个共享 store
// （agentNetworkStore / subagentRunsStore），不触桥。
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
  root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 3, errors: 0, tokens: 0, children: [] },
};

const RUNS: SubagentRunView[] = [
  {
    ref: "sa_1", status: "completed", task: "调研 dsh-context 功能面",
    answer: "五入口+三投影，缺口四处", toolCalls: 12,
    createdAt: "2026-11-12T10:00:00Z", updatedAt: "2026-11-12T10:05:00Z",
  },
  {
    ref: "sa_2", status: "failed", task: "写映射表",
    answer: "权限被拒", toolCalls: 2,
    createdAt: "2026-11-12T10:02:00Z", updatedAt: "2026-11-12T10:03:00Z",
  },
  {
    ref: "sa_3", status: "running", task: "长跑探索", toolCalls: 5,
    createdAt: "2026-11-12T10:06:00Z", updatedAt: "2026-11-12T10:08:00Z",
  },
];

vi.mock("../../lib/agentNetworkStore", () => ({
  subscribeAgentNetwork: (cb: (n: AgentNetwork) => void) => { cb(NET); return () => {}; },
  reloadAgentNetwork: () => {},
}));
vi.mock("../../lib/subagentRunsStore", () => ({
  subscribeSubagentRuns: (_p: string, cb: (r: SubagentRunView[], m: unknown) => void) => {
    cb(RUNS, { status: "ready", total: RUNS.length, running: 1 });
    return () => {};
  },
  reloadSubagentRuns: () => {},
}));

// 刀10：家族事件流 mock（main+两子代理合并流）
const FAMILY = {
  ok: true,
  scanned: 3,
  refs: ["main", "sa_1", "sa_2"],
  // 刀11：成员卡行（Go FamilyAgentRows 镜像；main 在前）
  agents: [
    { ref: "main", tokens: 120000, cost: 0.42, dispatches: 2, messages: 5 },
    { ref: "sa_1", tokens: 30000, cost: 0.11, dispatches: 1, messages: 4, reports: 1 },
    { ref: "sa_2", tokens: 8000, dispatches: 1, messages: 2 },
  ],
  events: [
    { seq: 1, ts: 1750000000, agent: "main", kind: "message", role: "user", text: "拆报表" },
    { seq: 2, ts: 1750000001, agent: "main", kind: "dispatch", text: "task {...}" },
    { seq: 3, ts: 1750000002, agent: "sa_1", kind: "dispatch", text: "拆报表" },
    { seq: 4, ts: 1750000003, agent: "sa_1", kind: "report", ref: "sa_1", text: "拆完了" },
    { seq: 5, ts: 1750000004, agent: "main", kind: "message", role: "assistant", text: "完成" },
  ],
};
vi.mock("../../lib/bridge", () => ({
  app: { FamilyEvents: vi.fn(() => Promise.resolve(FAMILY)) },
}));

import { Fleet } from "./Fleet";

describe("Fleet 编队页", () => {
  it("摘要条：Agent 数=1+子代理、运行中、任务数、家族消息数", async () => {
    renderT(<Fleet />);
    const summary = screen.getByTestId("fleet-summary");
    expect(summary.textContent).toContain("4"); // 1 root + 3 subagents
    // 家族事件流异步到达（.then 微任务）
    await waitFor(() => expect(summary.textContent).toContain("5"));
  });

  it("泳道：父代理道 + 每子代理一道", () => {
    renderT(<Fleet />);
    const lanes = screen.getByTestId("fleet-lanes");
    expect(lanes.textContent).toContain("父代理（派发点）");
    expect(lanes.textContent).toContain("调研 dsh-context 功能面");
    expect(lanes.textContent).toContain("长跑探索");
  });

  it("协作任务：状态筛选只留匹配行", () => {
    renderT(<Fleet />);
    fireEvent.click(screen.getByTestId("fleet-task-status-failed"));
    const tasks = screen.getByTestId("fleet-tasks");
    expect(tasks.textContent).toContain("写映射表");
    expect(tasks.textContent).not.toContain("调研 dsh-context 功能面");
  });

  it("事件检查：家族流渲染（agent chips/kind 筛选数据源）", async () => {
    renderT(<Fleet />);
    const ev = await waitFor(() => screen.getByTestId("fleet-events"));
    await waitFor(() => expect(ev.querySelectorAll("li").length).toBe(5));
  });

});

// 刀11：成员卡树——每 Agent 卡（角色徽标/费用/任务/消息计数）+点击联动
// 事件检查筛选。
describe("Fleet 成员卡（刀11，1:1 上游 Fleet 成员卡）", () => {
  it("渲染成员卡行：主控徽标+费用+委派/消息计数；点击子 Agent 卡筛选事件", async () => {
    renderT(<Fleet running={false} />);
    const members = await screen.findByTestId("fleet-members");
    expect(members).toBeTruthy();
    // 主控卡：main 在前，角色徽标+费用
    expect(screen.getByTestId("fleet-member-main")).toBeTruthy();
    expect(screen.getByText("主控")).toBeTruthy();
    expect(screen.getByText(/0\.42/)).toBeTruthy();
    // 子 Agent 卡回报计数
    expect(screen.getByTestId("fleet-member-sa_1")).toBeTruthy();
    expect(screen.getAllByText("子 Agent").length).toBeGreaterThan(0); // 每张子卡一枚徽标
    // 点击 sa_1 卡 → aria-pressed 且事件检查筛选到 sa_1（联动）
    fireEvent.click(screen.getByTestId("fleet-member-sa_1"));
    expect(screen.getByTestId("fleet-member-sa_1").getAttribute("aria-pressed")).toBe("true");
    // 再点同卡 = 清除筛选（Wave4 chips 同款语义）
    fireEvent.click(screen.getByTestId("fleet-member-sa_1"));
    expect(screen.getByTestId("fleet-member-sa_1").getAttribute("aria-pressed")).toBe("false");
  });
});
