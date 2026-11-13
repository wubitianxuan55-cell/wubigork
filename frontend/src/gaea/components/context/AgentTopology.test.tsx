import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { AgentNetwork, SubagentRunView } from "../../lib/types";
import { AgentTopology } from "./AgentTopology";

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const NETWORK: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "main",
    name: "主 Agent",
    kind: "root",
    status: "running",
    task: "整理季度报告",
    toolCalls: 7,
    errors: 0,
    tokens: 240_000,
    children: [
      {
        id: "sa_1",
        name: "调研 Agent",
        kind: "subagent",
        status: "running",
        task: "收集竞品信息",
        toolCalls: 3,
        errors: 0,
        tokens: 80_000,
        children: [],
      },
      {
        id: "sa_2",
        name: "写作 Agent",
        kind: "subagent",
        status: "completed",
        task: "撰写结论",
        toolCalls: 5,
        errors: 0,
        tokens: 120_000,
        children: [],
      },
    ],
  },
};

const RUNS: SubagentRunView[] = [
  {
    ref: "sa_1",
    status: "running",
    task: "收集竞品信息",
    toolCalls: 3,
    createdAt: "2026-11-13T10:00:00Z",
    updatedAt: "2026-11-13T10:02:00Z",
    model: "deepseek-v4-flash",
  },
];

describe("AgentTopology 编队卡片拓扑", () => {
  it("主 Agent 与子 Agent 渲染为卡片节点，不渲染径向 SVG 图", () => {
    const { container } = renderT(<AgentTopology network={NETWORK} runs={RUNS} />);
    expect(screen.getByTestId("agent-topology")).toBeTruthy();
    expect(container.querySelectorAll(".agent-topology-card")).toHaveLength(3);
    expect(container.querySelector(".agent-topology-branch.is-root > .agent-topology-card")).toBeTruthy();
    expect(container.querySelector('[data-agent="sa_1"]')?.getAttribute("data-status")).toBe("running");
    expect(container.querySelector('[data-testid="agent-radial-svg"]')).toBeNull();
  });

  it("子 Agent 卡片可键盘聚焦并打开对应上下文", () => {
    const onViewContext = vi.fn();
    const { container } = renderT(
      <AgentTopology network={NETWORK} runs={RUNS} onViewContext={onViewContext} />,
    );
    fireEvent.click(container.querySelector('[data-agent="sa_1"]') as HTMLElement);
    expect(onViewContext).toHaveBeenCalledWith("sa_1");
    expect((container.querySelector('[data-agent="sa_1"]') as HTMLElement).tagName).toBe("BUTTON");
  });
});
