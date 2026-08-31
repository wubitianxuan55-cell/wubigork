import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ContextTimeline } from "../lib/types";

const contextViewMock = vi.fn();
const agentNetworkMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { ContextView: contextViewMock, AgentNetwork: agentNetworkMock },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
}));

const TIMELINE: ContextTimeline = {
  ok: true,
  window: 1_000_000,
  current: { system: 2100, tools: 10400, user: 20, inject: 21900, assistant: 93400, tool: 114000 },
  stats: { turns: 2, steps: 60, injects: 6, compacts: 1, prunes: 0, toolCalls: 279, images: 0, cacheHitPercent: 99.57, costEstimate: 3.83 },
  requests: [
    {
      seq: 8, ts: 1750000000, turn: 1, step: 1,
      category: { system: 2100, tools: 10400, user: 20, inject: 21000, assistant: 1200, tool: 8000 },
      briefUser: "grep internal/gaea/config",
      briefResp: "read internal/gaea/config/config.go",
      promptTokens: 350600, outputTokens: 93, cacheHitTokens: 350200, cacheMissTokens: 400,
    },
  ],
  events: [
    { kind: "inject", seq: 2, delta: 10500, source: "指令注入 · .gaea\\AGENTS.md", turn: 1, step: 4, ts: 1750000000 },
    { kind: "compact", seq: 30, delta: -535500, source: "ratio", turn: 2, step: 20, ts: 1750000100 },
  ],
  nodes: [
    { seq: 1, cat: "system", tokens: 2100, text: "你是 gaea，土壤修复工程办公专用 AI 助手……" },
    { seq: 4, cat: "tool", tokens: 8000, text: "package main 这是一段足够长的工具输出内容，用于验证上下文浏览器节点的展开与收起交互，全文预览在这里展示。" },
  ],
  archive: [
    { seq: 30, cat: "user", tokens: 12000, text: "旧的一轮用户输入内容", gone: 30 },
  ],
  files: [
    { seq: 5, ts: 1750000005, turn: 1, step: 1, tool: "read_file", action: "read", path: "internal/gaea/config/config.go" },
    { seq: 9, ts: 1750000009, turn: 1, step: 2, tool: "write_file", action: "write", path: "docs/结论.md" },
  ],
};

describe("ContextView 上下文看板", () => {
  beforeEach(() => {
    contextViewMock.mockReset();
    contextViewMock.mockResolvedValue(TIMELINE);
    agentNetworkMock.mockReset();
    agentNetworkMock.mockResolvedValue({
      ok: true,
      window: 1_000_000,
      root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0, children: [] },
    });
  });

  it("渲染统计卡与六分类组成", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText("工具调用")).toBeTruthy();
    expect(screen.getByText("279")).toBeTruthy();
    expect(screen.getByText("99.57%")).toBeTruthy();
    expect(screen.getByText("系统提示词")).toBeTruthy();
    expect(screen.getAllByText("工具结果").length).toBeGreaterThan(0);
  });

  it("渲染趋势图、步骤详情与事件流", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText("上下文趋势")).toBeTruthy();
    expect(await screen.findByText(/点击趋势图中的柱查看该请求的输入、回复与上下文构成/)).toBeTruthy();
    expect(await screen.findByText(/指令注入 · \.gaea/)).toBeTruthy();
    expect((await screen.findAllByText("压缩")).length).toBeGreaterThan(0);
    expect(await screen.findByText(/-535\.5k/)).toBeTruthy();
  });

  it("点击柱后展示步骤详情", async () => {
    const { ContextView } = await import("./ContextView");
    // 图表首根柱：趋势 SVG 里的第一个 rect（jsdom 下 SVG title 不可被 byTitle 命中）
    const { container } = render(<ContextView running={false} />);
    await screen.findByText("上下文趋势");
    const rect = container.querySelector("svg rect");
    if (!rect) throw new Error("trend bar rect not found");
    fireEvent.click(rect);
    expect(await screen.findByText(/grep internal\/gaea\/config/)).toBeTruthy();
    expect(screen.getByText(/实际 prompt 350.6k/)).toBeTruthy();
  });

  it("渲染文件活动时间线（读/写徽标 + 路径 + 次数）", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText("文件活动")).toBeTruthy();
    expect(screen.getByText("2 次文件接触")).toBeTruthy();
    expect(screen.getByText("internal/gaea/config/config.go")).toBeTruthy();
    expect(screen.getByText("docs/结论.md")).toBeTruthy();
    expect(screen.getAllByText("read_file").length).toBeGreaterThan(0);
    expect(screen.getAllByText("write_file").length).toBeGreaterThan(0);
    expect(screen.getAllByText("读").length).toBeGreaterThan(0);
    expect(screen.getAllByText("写").length).toBeGreaterThan(0);
    // 文件活动已接入 → 页脚不再声称「后续阶段」
    expect(screen.queryByText(/文件活动将在后续阶段接入/)).toBeNull();
  });

  it("增量模式：点击「增量」展示净增减图例", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    await screen.findByText("上下文趋势");
    fireEvent.click(screen.getByText("增量"));
    expect(await screen.findByText(/净增/)).toBeTruthy();
    expect(screen.getByText(/净减/)).toBeTruthy();
  });

  it("渲染上下文浏览器（活跃节点 + 展开）", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText("上下文浏览器")).toBeTruthy();
    expect(screen.getByText(/活跃 2/)).toBeTruthy();
    expect(screen.getByText(/你是 gaea/)).toBeTruthy();
    // tool 节点文本超长 → 展开按钮出现，点击后展示全文并变为「收起」
    fireEvent.click(screen.getByText("展开"));
    expect(screen.getByText("收起")).toBeTruthy();
    expect(screen.getByText(/全文预览在这里展示/)).toBeTruthy();
    fireEvent.click(screen.getByText("收起"));
    expect(screen.queryByText("收起")).toBeNull();
  });

  it("上下文浏览器归档页展示被压缩节点", async () => {
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    fireEvent.click(await screen.findByText(/归档 1/));
    expect(screen.getByText(/旧的一轮用户输入内容/)).toBeTruthy();
    expect(screen.getAllByText("已压缩").length).toBeGreaterThan(0);
    // 页脚占位已移除
    expect(screen.queryByText(/上下文浏览器将在后续阶段接入/)).toBeNull();
  });

  it("加载失败显示错误", async () => {
    contextViewMock.mockRejectedValue(new Error("boom"));
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText(/上下文视图加载失败/)).toBeTruthy();
  });
});
