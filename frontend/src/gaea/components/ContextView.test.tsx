import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ContextTimeline } from "../lib/types";

const contextViewMock = vi.fn();
const agentNetworkMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { ContextView: contextViewMock, AgentNetwork: agentNetworkMock },
  openExternal: vi.fn(),
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
  nodes: [],
  archive: [],
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
    expect(screen.getByText("工具结果")).toBeTruthy();
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

  it("加载失败显示错误", async () => {
    contextViewMock.mockRejectedValue(new Error("boom"));
    const { ContextView } = await import("./ContextView");
    render(<ContextView running={false} />);
    expect(await screen.findByText(/上下文视图加载失败/)).toBeTruthy();
  });
});
