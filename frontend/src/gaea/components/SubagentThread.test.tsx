// SubagentThread — 子代理对话全面板视图（v4.27 Codex 式点击下钻）测试：
// 消息流渲染（system/user/assistant 思考+正文/tool）、返回回调、运行中
// 3s 轮询实时刷新（fake timers 确定性）。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { SubagentThread } from "./SubagentThread";
import type { SubagentTranscriptView } from "../lib/types";

const transcript: SubagentTranscriptView = {
  ref: "sa_2_b2b2b2b2",
  task: "调研竞品表格 Agent 能力并总结可蒸馏点",
  messages: [
    { role: "system", content: "你是子代理，专注完成派发任务。" },
    { role: "user", content: "调研竞品表格 Agent 能力并总结可蒸馏点" },
    { role: "assistant", reasoning: "先检索竞品资料", content: "开始检索公开信息。" },
    { role: "assistant", toolCalls: [{ id: "call_1", name: "web_fetch", arguments: "{\"url\":\"https://example.com\"}" }] },
    { role: "tool", name: "web_fetch", toolCallId: "call_1", content: "三家竞品表格交互结论：A 支持选区即图表。" },
    { role: "assistant", content: "正在比对三家竞品的表格选中→图表链路…" },
  ],
};

const mocks = vi.hoisted(() => ({
  SubagentTranscript: vi.fn(),
  onEvent: vi.fn(() => () => {}),
}));

vi.mock("../lib/bridge", () => ({ app: mocks, onEvent: mocks.onEvent }));

const wrap = (node: React.ReactNode) => node;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.SubagentTranscript.mockResolvedValue(transcript);
});

afterEach(() => {
  vi.useRealTimers();
});

describe("SubagentThread 子代理对话全面板（v4.27）", () => {
  it("渲染消息流：system 弱化 / user 右对齐 / assistant 思考+正文 / tool 结果", async () => {
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="调研竞品表格 Agent 能力并总结可蒸馏点" status="running" onBack={() => {}} />));

    expect(await screen.findByText("你是子代理，专注完成派发任务。")).toBeTruthy();
    // 任务标题（头部）与 user 消息正文同文案 → getAllByText
    expect(screen.getAllByText("调研竞品表格 Agent 能力并总结可蒸馏点").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("开始检索公开信息。")).toBeTruthy();
    // 工具调用行与 tool 结果消息都可能含 web_fetch → 用 getAllByText
    expect(screen.getAllByText(/web_fetch/).length).toBeGreaterThan(0);
    expect(screen.getByText(/三家竞品表格交互结论/)).toBeTruthy();
    // 头部：状态徽标 + 消息计数 + 刷新按钮
    expect(screen.getByText("进行中")).toBeTruthy();
    expect(screen.getByText("6 条")).toBeTruthy();
    // 思考块默认折叠，点开显示推理
    expect(screen.queryByText("先检索竞品资料")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /思考/ }));
    expect(screen.getByText("先检索竞品资料")).toBeTruthy();
  });

  it("头部返回按钮触发 onBack", async () => {
    const onBack = vi.fn();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={onBack} />));
    await screen.findByText("任务");
    fireEvent.click(screen.getByRole("button", { name: /分工/ }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("已完成：只拉取一次，不轮询", async () => {
    vi.useFakeTimers();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="completed" onBack={() => {}} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);
    mocks.SubagentTranscript.mockClear();
    act(() => { vi.advanceTimersByTime(9000); });
    expect(mocks.SubagentTranscript).not.toHaveBeenCalled();
  });

  it("运行中：每 3s 轮询刷新对话（实时显示）", async () => {
    vi.useFakeTimers();
    render(wrap(<SubagentThread sessionPath="s1.jsonl" target="sa_2_b2b2b2b2" task="任务" status="running" onBack={() => {}} />));
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);
    mocks.SubagentTranscript.mockClear();

    act(() => { vi.advanceTimersByTime(3000); });
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(1);

    act(() => { vi.advanceTimersByTime(6000); });
    await act(async () => { await Promise.resolve(); });
    expect(mocks.SubagentTranscript).toHaveBeenCalledTimes(3);
  });
});
