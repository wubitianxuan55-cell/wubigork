import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { AgentTree } from "./AgentTree";
import type { AgentNetwork, AgentNode, SubagentRunsView, SubagentTranscriptView } from "../lib/types";

// ── v4.24 A1「子代理树实时拓扑」测试 ────────────────────────────
// AgentTree 直接以 props 驱动（无轮询/事件依赖）：钉住嵌套子树渲染与
// 展开/收起、新节点自动展开父链、运行富化与耗时、详情→transcript→
// 工具调用定位下钻链。

function node(id: string, over: Partial<AgentNode> = {}): AgentNode {
  return {
    id,
    name: "task",
    kind: "subagent",
    status: "completed",
    toolCalls: 1,
    errors: 0,
    tokens: 1000,
    ...over,
  };
}

const network: AgentNetwork = {
  ok: true,
  window: 0,
  root: {
    id: "root",
    name: "主 agent",
    kind: "root",
    status: "running",
    toolCalls: 3,
    errors: 0,
    tokens: 420000,
    firstTs: 1750000000,
    lastTs: 1750000200,
    children: [
      node("sa_1_a1a1a1a1", { status: "completed", task: "收集竞品信息", model: "deepseek-v4-flash", toolCalls: 3, firstTs: 1750000100, lastTs: 1750000150 }),
      node("sa_2_b2b2b2b2", {
        status: "running",
        task: "调研表格 Agent",
        toolCalls: 1,
        firstTs: 1750000160,
        children: [node("sa_3_c3c3c3c3", { status: "running", task: "子任务：对比表格交互", toolCalls: 2, firstTs: 1750000170 })],
      }),
    ],
  },
};

const runs: SubagentRunsView = {
  available: true,
  total: 2,
  running: 1,
  runs: [
    { ref: "sa_2_b2b2b2b2", status: "running", task: "调研表格 Agent", lastText: "正在比对三家竞品的表格选中→图表链路…", lastTool: "web_fetch: https://example.com/table-agent", toolCalls: 1, createdAt: "2026-08-17T11:00:00+08:00", updatedAt: "2026-08-17T11:01:00+08:00" },
    { ref: "sa_1_a1a1a1a1", status: "completed", model: "deepseek-v4-flash", task: "收集竞品信息", answer: "已汇总三条可蒸馏点。", toolCalls: 3, createdAt: "2026-08-17T10:00:00+08:00", updatedAt: "2026-08-17T10:30:00+08:00" },
  ],
};

const transcript: SubagentTranscriptView = {
  ref: "sa_2_b2b2b2b2",
  task: "调研表格 Agent",
  messages: [
    { role: "user", content: "调研表格 Agent" },
    { role: "assistant", toolCalls: [{ id: "call_1", name: "web_fetch", arguments: "{\"url\":\"https://example.com\"}" }] },
    { role: "tool", name: "web_fetch", toolCallId: "call_1", content: "三家竞品结论…" },
    { role: "assistant", content: "调研完成。" },
  ],
};

const mocks = vi.hoisted(() => ({ SubagentTranscript: vi.fn() }));
vi.mock("../lib/bridge", () => ({ app: mocks }));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.SubagentTranscript.mockResolvedValue(transcript);
});

describe("AgentTree 子代理树实时拓扑（v4.24 A1）", () => {
  it("嵌套渲染：主 agent + 一级子代理恒可见，孙节点默认收起", () => {
    render(<AgentTree network={network} runs={runs.runs} />);
    expect(screen.getByText("主 agent")).toBeTruthy();
    expect(screen.getByText("收集竞品信息")).toBeTruthy();
    expect(screen.getByText("调研表格 Agent")).toBeTruthy();
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
  });

  it("展开/收起嵌套子树：孙节点随父节点开关显隐", () => {
    render(<AgentTree network={network} runs={runs.runs} />);
    fireEvent.click(screen.getByLabelText("展开 调研表格 Agent"));
    expect(screen.getByText("子任务：对比表格交互")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("收起 调研表格 Agent"));
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
  });

  it("新节点出现自动展开其父链（本轮挂载后的新节点）", async () => {
    const { rerender } = render(<AgentTree network={network} runs={runs.runs} />);
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
    // 挂载后出现新孙节点：父链（调研表格 Agent）应自动展开，孙节点可见
    const v2: AgentNetwork = {
      ...network,
      root: {
        ...network.root,
        children: [
          network.root.children![0],
          {
            ...network.root.children![1],
            children: [
              node("sa_3_c3c3c3c3", { status: "running", task: "子任务：对比表格交互", toolCalls: 2, firstTs: 1750000170 }),
              node("sa_9_new_new", { status: "running", task: "新出现的孙任务", toolCalls: 1, firstTs: 1750000180 }),
            ],
          },
        ],
      },
    };
    rerender(<AgentTree network={v2} runs={runs.runs} />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText("新出现的孙任务")).toBeTruthy();
    expect(screen.getByText("子任务：对比表格交互")).toBeTruthy();
  });

  it("运行节点富化：匹配分工 meta 显示实时预览与模型徽标；无匹配降级纯统计", () => {
    render(<AgentTree network={network} runs={runs.runs} />);
    // ref 直等匹配命中：运行节点行内嵌「正在…」实时预览
    expect(screen.getByText(/正在：正在比对三家竞品的表格选中→图表链路/)).toBeTruthy();
    expect(screen.getByText(/web_fetch: https:\/\/example\.com\/table-agent/)).toBeTruthy();
    // completed 节点模型徽标（run.model）
    expect(screen.getByText("deepseek-v4-flash")).toBeTruthy();
  });

  it("运行节点显示实时已用耗时（1s tick 驱动）", () => {
    render(<AgentTree network={network} runs={runs.runs} />);
    // running 节点（root + 运行中子树）有 firstTs → 「已用 …」文案存在
    // （不锁具体数值，避免真实时间漂移；多节点命中用 getAllByText）
    expect(screen.getAllByText(/已用 /).length).toBeGreaterThan(0);
  });

  it("下钻链：节点点击 → 详情卡 → 完整 transcript → 工具调用行定位结果消息", async () => {
    render(<AgentTree network={network} runs={runs.runs} sessionPath="s1.jsonl" />);
    fireEvent.click(screen.getByText("调研表格 Agent"));
    expect(await screen.findByTestId("agent-detail")).toBeTruthy();
    fireEvent.click(screen.getByText("查看完整 transcript"));
    expect(await screen.findByTestId("agent-transcript")).toBeTruthy();
    expect(screen.getByText(/三家竞品结论/)).toBeTruthy();
    fireEvent.click(screen.getByTestId("agent-transcript-toolcall"));
    const resultMsg = document.querySelector('[data-msg-idx="2"]');
    expect(resultMsg?.getAttribute("data-located")).toBe("true");
  });

  it("节点点击再点收起详情卡（切换节点清空 transcript）", async () => {
    render(<AgentTree network={network} runs={runs.runs} sessionPath="s1.jsonl" />);
    // 详情卡打开后标题重复出现 → 用节点行按钮（第一个命中=树内行）切换
    const row = screen.getAllByText("调研表格 Agent")[0];
    fireEvent.click(row);
    expect(await screen.findByTestId("agent-detail")).toBeTruthy();
    fireEvent.click(screen.getAllByText("调研表格 Agent")[0]);
    expect(screen.queryByTestId("agent-detail")).toBeNull();
  });
});
