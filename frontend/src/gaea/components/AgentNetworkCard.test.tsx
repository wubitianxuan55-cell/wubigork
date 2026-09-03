import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { AgentNetwork } from "../lib/types";

// AgentNetworkCard 走 useT：钉住 zh 让「Agent 网络/主 agent/查看完整 transcript」等中文断言继续成立
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const agentNetworkMock = vi.fn();
const subagentRunsMock = vi.fn();
const subagentTranscriptMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { AgentNetwork: agentNetworkMock, SubagentRuns: subagentRunsMock, SubagentTranscript: subagentTranscriptMock },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
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
    subagentRunsMock.mockReset();
    subagentRunsMock.mockResolvedValue({
      available: true,
      total: 1,
      running: 0,
      runs: [
        {
          ref: "sa_20260831_100000_0000000001_a1a1a1a1",
          status: "completed",
          model: "deepseek-v4-flash",
          task: "调研模块A的现状",
          answer: "结论：模块 A 现状已梳理，三条可蒸馏点。",
          lastText: "模块 A 现状已梳理。",
          lastTool: "read_file: internal/a.go",
          toolCalls: 4,
          createdAt: "2026-08-31T10:00:00+08:00",
          updatedAt: "2026-08-31T10:30:00+08:00",
        },
      ],
    });
    subagentTranscriptMock.mockReset();
    subagentTranscriptMock.mockResolvedValue({
      ref: "sa_20260831_100000_0000000001_a1a1a1a1",
      task: "调研模块A的现状",
      messages: [
        { role: "system", content: "你是子代理。" },
        { role: "user", content: "调研模块A的现状" },
        { role: "assistant", reasoning: "先读代码", content: "开始调研。" },
        { role: "tool", name: "read_file", content: "internal/a.go 内容" },
        { role: "assistant", content: "调研完成。" },
      ],
    });
  });

  it("渲染标题、子代理标签与统计", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    renderT(<AgentNetworkCard running={false} />);
    expect(await screen.findByText("Agent 网络")).toBeTruthy();
    expect(screen.getByText(/2 个子代理/)).toBeTruthy();
    expect(screen.getByText(/调研模块A的现状/)).toBeTruthy();
    expect(screen.getAllByText("completed").length).toBeGreaterThan(0);
  });

  it("悬停节点显示详情", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    renderT(<AgentNetworkCard running={false} />);
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
    renderT(<AgentNetworkCard running={false} />);
    expect(await screen.findByText(/0 个子代理/)).toBeTruthy();
    expect(screen.getByText(/悬停查看节点详情/)).toBeTruthy();
  });

  it("点击子代理节点固定分工详情（任务/回答/活动行）", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    renderT(<AgentNetworkCard running={false} sessionPath="s1.jsonl" />);
    await screen.findByText("Agent 网络");
    // 子代理 A 的中心数字 = 工具调用数 4（circle 内 text）
    fireEvent.click(screen.getByText("4"));
    expect(await screen.findByText(/结论：模块 A 现状已梳理/)).toBeTruthy();
    expect(screen.getByText(/已完成/)).toBeTruthy();
    expect(screen.getByText(/deepseek-v4-flash/)).toBeTruthy();
    expect(screen.getByText(/read_file: internal\/a.go/)).toBeTruthy();
    expect(subagentRunsMock).toHaveBeenCalledWith("s1.jsonl");
  });

  it("查看完整 transcript（消息流渲染 + 收起）", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    renderT(<AgentNetworkCard running={false} sessionPath="s1.jsonl" />);
    await screen.findByText("Agent 网络");
    fireEvent.click(screen.getByText("4"));
    fireEvent.click(await screen.findByText("查看完整 transcript"));
    expect(await screen.findByText(/开始调研/)).toBeTruthy();
    expect(screen.getByText(/internal\/a.go 内容/)).toBeTruthy();
    expect(screen.getByText(/5\/5 条/)).toBeTruthy();
    expect(screen.getByText("#1")).toBeTruthy();
    expect(screen.getByText("#5")).toBeTruthy();
    expect(subagentTranscriptMock).toHaveBeenCalledWith("s1.jsonl", "sa_20260831_100000_0000000001_a1a1a1a1");
    fireEvent.click(screen.getByText("收起完整 transcript"));
    expect(screen.queryByText(/internal\/a.go 内容/)).toBeNull();
  });

  it("transcript 搜索过滤消息", async () => {
    const { AgentNetworkCard } = await import("./AgentNetworkCard");
    renderT(<AgentNetworkCard running={false} sessionPath="s1.jsonl" />);
    await screen.findByText("Agent 网络");
    fireEvent.click(screen.getByText("4"));
    fireEvent.click(await screen.findByText("查看完整 transcript"));
    await screen.findByText(/5\/5 条/);
    const input = screen.getByPlaceholderText("搜索消息");
    fireEvent.change(input, { target: { value: "调研完成" } });
    expect(screen.getByText(/1\/5 条/)).toBeTruthy();
    expect(screen.getByText("调研完成。")).toBeTruthy();
    expect(screen.queryByText(/internal\/a.go 内容/)).toBeNull();
  });
});
