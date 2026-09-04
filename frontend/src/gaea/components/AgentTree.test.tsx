import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import type { ReactElement } from "react";
import { AgentTree } from "./AgentTree";
import { LocaleProvider } from "../lib/i18n";
import type { AgentNetwork, AgentNode, SubagentRunsView } from "../lib/types";

// AgentTree 走 useT：钉住 zh 让「主 agent/展开 …/正在：…」等中文断言继续成立
const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// ── v4.24 A1「子代理树实时拓扑」测试 ────────────────────────────
// AgentTree 直接以 props 驱动（无轮询/事件依赖）：钉住嵌套子树渲染与
// 展开/收起、新节点自动展开父链、运行富化与耗时、子代理节点点击 →
// onOpenThread 回调（v4.27 由面板层打开全面板对话）。

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

describe("AgentTree 子代理树实时拓扑（v4.24 A1）", () => {
  it("嵌套渲染：主 agent + 一级子代理恒可见，孙节点默认收起", () => {
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
    expect(screen.getByText("主 agent")).toBeTruthy();
    expect(screen.getByText("收集竞品信息")).toBeTruthy();
    expect(screen.getByText("调研表格 Agent")).toBeTruthy();
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
  });

  it("展开/收起嵌套子树：孙节点随父节点开关显隐", () => {
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
    fireEvent.click(screen.getByLabelText("展开 调研表格 Agent"));
    expect(screen.getByText("子任务：对比表格交互")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("收起 调研表格 Agent"));
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
  });

  it("点击父卡片标题同样可折叠/展开（v4.76 点卡片即折叠）", () => {
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
    fireEvent.click(screen.getByText("调研表格 Agent"));
    expect(screen.getByText("子任务：对比表格交互")).toBeTruthy();
    fireEvent.click(screen.getByText("调研表格 Agent"));
    expect(screen.queryByText("子任务：对比表格交互")).toBeNull();
  });

  it("新节点出现自动展开其父链（本轮挂载后的新节点）", async () => {
    const { rerender } = renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
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
    rerender(<LocaleProvider><AgentTree network={v2} runs={runs.runs} onOpenThread={() => {}} /></LocaleProvider>);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText("新出现的孙任务")).toBeTruthy();
    expect(screen.getByText("子任务：对比表格交互")).toBeTruthy();
  });

  it("运行节点富化：匹配分工 meta 显示实时预览与模型徽标；无匹配降级纯统计", () => {
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
    // ref 直等匹配命中：运行节点行内嵌「正在…」实时预览
    expect(screen.getByText(/正在：正在比对三家竞品的表格选中→图表链路/)).toBeTruthy();
    expect(screen.getByText(/web_fetch: https:\/\/example\.com\/table-agent/)).toBeTruthy();
    // completed 节点模型徽标（run.model）
    expect(screen.getByText("deepseek-v4-flash")).toBeTruthy();
  });

  it("运行节点显示实时已用耗时（1s tick 驱动）", () => {
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={() => {}} />);
    // running 节点（root + 运行中子树）有 firstTs → 「已用 …」文案存在
    // （不锁具体数值，避免真实时间漂移；多节点命中用 getAllByText）
    expect(screen.getAllByText(/已用 /).length).toBeGreaterThan(0);
  });

  it("子代理节点点击 → onOpenThread(node, run)（v4.27 打开全面板对话）", () => {
    const onOpenThread = vi.fn();
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={onOpenThread} />);
    // v4.76：点卡片标题改为折叠/展开；打开对话走独立按钮（aria-label）
    fireEvent.click(screen.getByLabelText("打开对话 调研表格 Agent"));
    expect(onOpenThread).toHaveBeenCalledTimes(1);
    const [node, run] = onOpenThread.mock.calls[0] as [AgentNode, SubagentRunsView["runs"][number] | null];
    expect(node.id).toBe("sa_2_b2b2b2b2");
    // run 匹配：ref 直等命中运行中的分工 meta（供对话视图实时派生状态）
    expect(run?.ref).toBe("sa_2_b2b2b2b2");
  });

  it("主 agent 根节点点击不触发 onOpenThread（无独立 transcript）", () => {
    const onOpenThread = vi.fn();
    renderT(<AgentTree network={network} runs={runs.runs} onOpenThread={onOpenThread} />);
    fireEvent.click(screen.getByText("主 agent"));
    expect(onOpenThread).not.toHaveBeenCalled();
  });
});
