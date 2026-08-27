import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentNetwork } from "../lib/types";

const agentNetworkMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { AgentNetwork: agentNetworkMock },
  openExternal: vi.fn(),
}));

const NETWORK: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "root",
    name: "主 agent",
    kind: "root",
    status: "running",
    toolCalls: 3,
    errors: 0,
    tokens: 420000,
    children: [
      {
        id: "taskA",
        name: "task",
        kind: "subagent",
        status: "completed",
        model: "deepseek-v4-flash",
        task: "调研模块A的现状",
        toolCalls: 4,
        errors: 0,
        tokens: 180000,
      },
      {
        id: "taskB",
        name: "task",
        kind: "subagent",
        status: "error",
        task: "并行生成测试",
        toolCalls: 2,
        errors: 1,
        tokens: 65000,
      },
    ],
  },
};

describe("AgentNetworkCard Agent 网络", () => {
  beforeEach(() => {
    agentNetworkMock.mockReset();
    agentNetworkMock.mockResolvedValue(NETWORK);
  });

  it("渲染标题、子代理标签与统计", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    render(<AgentNetworkCard running={false} />);
    expect(await screen.findByText("Agent 网络")).toBeTruthy();
    expect(screen.getByText(/2 个子代理/)).toBeTruthy();
    expect(screen.getByText(/调研模块A的现状/)).toBeTruthy();
    expect(screen.getAllByText("completed").length).toBeGreaterThan(0);
  });

  it("悬停节点显示详情", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    render(<AgentNetworkCard running={false} />);
    // 悬停根节点（文字「主」）
    const rootNode = await screen.findByText("主");
    fireEvent.mouseOver(rootNode);
    expect(screen.getByText("主 agent")).toBeTruthy();
    expect(screen.getByText(/≈420k/)).toBeTruthy();
    expect(screen.getByText(/工具 3/)).toBeTruthy();
  });

  it("无子代理时渲染空提示", async () => {
    agentNetworkMock.mockResolvedValue({
      ok: true,
      window: 1_000_000,
      root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0, children: [] },
    });
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    render(<AgentNetworkCard running={false} />);
    expect(await screen.findByText(/0 个子代理/)).toBeTruthy();
    expect(screen.getByText(/悬停节点查看子代理详情/)).toBeTruthy();
  });
});
