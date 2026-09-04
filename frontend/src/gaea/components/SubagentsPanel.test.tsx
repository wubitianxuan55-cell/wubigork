import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act, waitFor, within } from "@testing-library/react";
import type { ReactElement } from "react";
import { SubagentsPanel } from "./SubagentsPanel";
import { LocaleProvider } from "../lib/i18n";
import type { AgentNetwork, SubagentRunsView, SubagentTranscriptView } from "../lib/types";

// 面板/树/对话组件走 useT：钉住 zh 让既有中文文案断言继续成立
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// ── v4.24 A1「分工面板工作台化」测试 ─────────────────────────────
// 三段式结构（合并活动流 + 树形实时拓扑 + 自动展开偏好）在 SubagentsPanel
// 层钉住：树渲染/运行富化/计数徽标/嵌套子树展开/详情下钻/开关持久化/
// 新子代理回调（跨会话路径触发，确定性无假定时器）。

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
      {
        id: "sa_1_a1a1a1a1",
        name: "task",
        kind: "subagent",
        status: "completed",
        task: "收集 2026 年办公 Agent 竞品更新信息",
        model: "deepseek-v4-flash",
        toolCalls: 3,
        errors: 0,
        tokens: 180000,
        firstTs: 1750000100,
        lastTs: 1750000150,
      },
      {
        id: "sa_2_b2b2b2b2",
        name: "task",
        kind: "subagent",
        status: "running",
        task: "调研竞品表格 Agent 能力并总结可蒸馏点",
        toolCalls: 1,
        errors: 0,
        tokens: 65000,
        firstTs: 1750000160,
        children: [
          {
            id: "sa_3_c3c3c3c3",
            name: "task",
            kind: "subagent",
            status: "running",
            task: "子任务：对比三家竞品表格交互",
            toolCalls: 2,
            errors: 0,
            tokens: 22000,
            firstTs: 1750000170,
          },
        ],
      },
    ],
  },
};

const runsA: SubagentRunsView = {
  available: true,
  total: 2,
  running: 1,
  runs: [
    {
      ref: "sa_2_b2b2b2b2",
      status: "running",
      task: "调研竞品表格 Agent 能力并总结可蒸馏点",
      lastText: "正在比对三家竞品的表格选中→图表链路…",
      lastTool: "web_fetch: https://example.com/table-agent",
      toolCalls: 1,
      createdAt: "2026-08-17T11:00:00+08:00",
      updatedAt: "2026-08-17T11:01:00+08:00",
    },
    {
      ref: "sa_1_a1a1a1a1",
      status: "completed",
      model: "deepseek-v4-flash",
      task: "收集 2026 年办公 Agent 竞品更新信息",
      answer: "千问办公公测、WorkSwarm 蜂群智能体、QClaw V2 多 Agent。",
      lastText: "千问办公公测、WorkSwarm 蜂群智能体、QClaw V2 多 Agent。",
      lastTool: "web_search: 办公 Agent 竞品 2026",
      toolCalls: 3,
      createdAt: "2026-08-17T10:00:00+08:00",
      updatedAt: "2026-08-17T10:30:00+08:00",
    },
  ],
};

// 会话 B：多一个运行中的新子代理（自动展开回调触发用）。
const runsB: SubagentRunsView = {
  available: true,
  total: 3,
  running: 2,
  runs: [
    { ...runsA.runs[0] },
    { ...runsA.runs[1] },
    {
      ref: "sa_4_d4d4d4d4",
      status: "running",
      task: "新增并行任务：整理竞品报价单",
      toolCalls: 0,
      createdAt: "2026-08-17T12:00:00+08:00",
      updatedAt: "2026-08-17T12:00:30+08:00",
    },
  ],
};

const transcript: SubagentTranscriptView = {
  ref: "sa_2_b2b2b2b2",
  task: "调研竞品表格 Agent 能力并总结可蒸馏点",
  messages: [
    { role: "user", content: "调研竞品表格 Agent 能力并总结可蒸馏点" },
    { role: "assistant", reasoning: "先检索竞品资料", content: "开始检索公开信息。" },
    {
      role: "assistant",
      toolCalls: [{ id: "call_1", name: "web_fetch", arguments: "{\"url\":\"https://example.com/table-agent\"}" }],
    },
    { role: "tool", name: "web_fetch", toolCallId: "call_1", content: "三家竞品表格交互结论：A 支持选区即图表，B 仅整表，C 无。" },
    { role: "assistant", content: "正在比对三家竞品的表格选中→图表链路…" },
  ],
};

const mocks = vi.hoisted(() => ({
  AgentNetwork: vi.fn(),
  SubagentRuns: vi.fn(),
  SubagentTranscript: vi.fn(),
  onEvent: vi.fn(() => () => {}),
  onSubagentText: vi.fn(() => () => {}),
}));

vi.mock("../lib/bridge", () => ({ app: mocks, onEvent: mocks.onEvent, onSubagentText: mocks.onSubagentText }));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.AgentNetwork.mockResolvedValue(network);
  mocks.SubagentRuns.mockResolvedValue(runsA);
  mocks.SubagentTranscript.mockResolvedValue(transcript);
  window.localStorage.removeItem("gaea.subagentAutoOpen");
});

describe("SubagentsPanel 子代理工作台（v4.24 A1 三段式）", () => {
  it("树形实时拓扑：主 agent + 子代理全量渲染，计数徽标显示总数与运行中", async () => {
    renderT(<SubagentsPanel sessionPath="s1.jsonl" />);
    expect(await screen.findByText("主 agent")).toBeTruthy();
    expect(screen.getByText("收集 2026 年办公 Agent 竞品更新信息")).toBeTruthy();
    expect(screen.getByText("调研竞品表格 Agent 能力并总结可蒸馏点")).toBeTruthy();
    // 头部徽标：总数 2（runs.length）+ 运行中 1
    expect(screen.getByText("1 运行中")).toBeTruthy();
    expect(screen.getByText("2")).toBeTruthy();
    // 运行节点富化：模型徽标（ref 直等匹配命中 completed run）
    expect(screen.getByText("deepseek-v4-flash")).toBeTruthy();
  });

  it("合并活动流：运行中动态单列 feed（最新在前，空态收起）", async () => {
    renderT(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText("主 agent");
    // 活动流在 feed 容器内（运行预览同时在树内行出现 → 限定容器避免多命中）
    const feed = screen.getByTestId("agent-feed");
    expect(feed).toBeTruthy();
    expect(within(feed).getByText(/正在：正在比对三家竞品的表格选中→图表链路/)).toBeTruthy();
    expect(within(feed).getByText(/web_fetch: https:\/\/example\.com\/table-agent/)).toBeTruthy();
  });

  it("嵌套子树默认收起，点击展开/收起孙节点", async () => {
    renderT(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText("主 agent");
    // 孙节点（depth 2）初始不可见
    expect(screen.queryByText("子任务：对比三家竞品表格交互")).toBeNull();
    // 展开运行中子代理（有子树的节点显示折叠按钮）
    fireEvent.click(screen.getByLabelText("展开 调研竞品表格 Agent 能力并总结可蒸馏点"));
    expect(screen.getByText("子任务：对比三家竞品表格交互")).toBeTruthy();
    // 收起后孙节点消失
    fireEvent.click(screen.getByLabelText("收起 调研竞品表格 Agent 能力并总结可蒸馏点"));
    expect(screen.queryByText("子任务：对比三家竞品表格交互")).toBeNull();
  });

  it("下钻链（v4.27）：节点点击 → 右侧全面板子代理对话（实时视图），返回回分工树", async () => {
    renderT(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText("主 agent");
    // 点击运行中的子代理行 → 切换为全面板对话视图（替代旧的内嵌窄卡）
    fireEvent.click(screen.getByText("调研竞品表格 Agent 能力并总结可蒸馏点"));
    const thread = await screen.findByTestId("agent-thread");
    expect(thread).toBeTruthy();
    // 头部：任务标题 + 实时状态（运行中）+ 返回按钮
    // 任务标题与 user 消息正文同文案 → getAllByText
    expect(screen.getAllByText("调研竞品表格 Agent 能力并总结可蒸馏点").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("进行中")).toBeTruthy();
    // 消息流渲染：user / assistant 正文 / tool 结果
    expect(screen.getByText(/开始检索公开信息/)).toBeTruthy();
    expect(screen.getByText(/三家竞品表格交互结论/)).toBeTruthy();
    // 返回 → 分工树恢复
    fireEvent.click(screen.getByRole("button", { name: /分工/ }));
    await waitFor(() => expect(screen.queryByTestId("agent-thread")).toBeNull());
    expect(screen.getByTestId("agent-tree")).toBeTruthy();
  });

  it("自动展开偏好开关：切换持久化到 localStorage", async () => {
    renderT(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText("主 agent");
    const toggle = screen.getByTestId("subagent-auto-open-toggle");
    expect(toggle.textContent).toContain("自动展开 开");
    fireEvent.click(toggle);
    expect(screen.getByText("自动展开 关")).toBeTruthy();
    expect(window.localStorage.getItem("gaea.subagentAutoOpen")).toBe("0");
  });

  it("新子代理出现（跨会话路径触发）→ onSubagentStarted 回调", async () => {
    const onStarted = vi.fn();
    const { rerender } = renderT(<SubagentsPanel sessionPath="s1.jsonl" onSubagentStarted={onStarted} />);
    await screen.findByText("主 agent");
    // 切到会话 B：mock 返回带新 ref 的分工列表 → 检测到新子代理（面板只负责
    // 检测 + 回调，亮出面板由 App 接线决定——此处只钉回调触发）。
    mocks.SubagentRuns.mockResolvedValue(runsB);
    rerender(<LocaleProvider><SubagentsPanel sessionPath="s2.jsonl" onSubagentStarted={onStarted} /></LocaleProvider>);
    await waitFor(() => expect(onStarted).toHaveBeenCalledTimes(1));
  });

  it("偏好关闭：新子代理出现只更新数据，不触发 onSubagentStarted", async () => {
    window.localStorage.setItem("gaea.subagentAutoOpen", "0");
    const onStarted = vi.fn();
    const { rerender } = renderT(<SubagentsPanel sessionPath="s1.jsonl" onSubagentStarted={onStarted} />);
    await screen.findByText("主 agent");
    mocks.SubagentRuns.mockResolvedValue(runsB);
    rerender(<LocaleProvider><SubagentsPanel sessionPath="s2.jsonl" onSubagentStarted={onStarted} /></LocaleProvider>);
    await act(async () => { await Promise.resolve(); });
    expect(onStarted).not.toHaveBeenCalled();
    window.localStorage.removeItem("gaea.subagentAutoOpen");
  });

  it("无 sessionPath 显示空状态（不请求）", async () => {
    renderT(<SubagentsPanel />);
    expect(await screen.findByText(/尚未派发子代理/)).toBeTruthy();
    expect(mocks.AgentNetwork).not.toHaveBeenCalled();
  });
});
