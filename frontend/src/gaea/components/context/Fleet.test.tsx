// 刀5 编队页行为测试：摘要条/泳道/任务面板/事件检查。mock 两个共享 store
// （agentNetworkStore / subagentRunsStore），不触桥。
import { fireEvent, render, screen } from "@testing-library/react";
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

import { Fleet } from "./Fleet";

describe("Fleet 编队页", () => {
  it("摘要条：Agent 数=1+子代理、运行中、任务数、工具调用合计", () => {
    renderT(<Fleet />);
    const summary = screen.getByTestId("fleet-summary");
    expect(summary.textContent).toContain("4"); // 1 root + 3 subagents
    expect(summary.textContent).toContain("19"); // 工具调用 12+2+5
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

  it("事件检查：派发+回报两类、running 无回报事件", () => {
    renderT(<Fleet />);
    const ev = screen.getByTestId("fleet-events");
    // 2 完成态 run 各有派发+回报，1 running 只有派发 → 5 行
    expect(ev.querySelectorAll("li").length).toBe(5);
  });

});
