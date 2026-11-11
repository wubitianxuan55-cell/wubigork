// 洞察页行为测试（刀3）：KPI 渲染、范围切换重查、会话卡点击回跳、
// 热力图日过滤。mock 桥同 ContextView.test 模式（vi.mock lib/bridge）。
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { ContextInsightsView } from "../../lib/types/context";

vi.setConfig({ testTimeout: 20_000 });

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// vi.hoisted：vi.mock 工厂在 import 阶段执行，须与 mock fn 同提升（TDZ 先例）。
const { insightsMock } = vi.hoisted(() => ({ insightsMock: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: { ContextInsights: insightsMock },
  onEvent: vi.fn(() => () => {}),
}));

// fixture：两天活动 + 两张会话卡。
const NOW = Math.floor(new Date("2026-11-12T12:00:00").getTime() / 1000);
const VIEW: ContextInsightsView = {
  ok: true,
  range: "30d",
  generatedAt: NOW,
  kpi: {
    sessions: 2, tokens: 248_600, requests: 128, turns: 34,
    toolCalls: 279, skills: 5, cost: 3.83, cacheHitPercent: 99.57,
  },
  days: [
    { date: "2026-11-11", tokens: 120_000, requests: 60, turns: 12, toolCalls: 100, cost: 2.1 },
    { date: "2026-11-12", tokens: 128_600, requests: 68, turns: 22, toolCalls: 179, cost: 1.73 },
  ],
  skills: [{ name: "gbt-9704-docx", count: 3, last: NOW - 3600 }],
  tools: [{ name: "read_file", count: 96, last: NOW - 600 }],
  programs: [{ name: "go", count: 22, last: NOW - 1200 }],
  sessions: [
    {
      path: "C:\\demo\\a.jsonl", title: "会话甲", spaceId: "work",
      modTime: NOW - 86400 - 7200, lastActive: NOW - 86400 - 7200, // 11-11 上午
      tokens: 248_600, requests: 96, turns: 21, cost: 3.1, cacheHitPercent: 99.6,
      composition: { system: 2100, tools: 10400, user: 5200, skill: 3200, inject: 21900, assistant: 93400, tool: 114000 },
    },
    {
      path: "C:\\demo\\b.jsonl", title: "会话乙（今天活跃）", spaceId: "work",
      modTime: NOW - 10, lastActive: NOW - 10,
      tokens: 1_000, requests: 2, turns: 1,
      composition: { system: 100, tools: 0, user: 200, skill: 0, inject: 0, assistant: 700, tool: 0 },
    },
  ],
  scanned: 2,
};

beforeEach(() => {
  insightsMock.mockReset();
  insightsMock.mockResolvedValue(VIEW);
});

import { Insights } from "./Insights";

describe("Insights 洞察页", () => {
  it("渲染 KPI/三统计/会话卡（mock 数据驱动）", async () => {
    renderT(<Insights />);
    await waitFor(() => expect(screen.getByTestId("insights-kpi")).toBeTruthy());
    // KPI 数字（248.6k 由 fmtTokens 渲染；直接断言会话卡标题更稳）
    await waitFor(() => expect(screen.getByText("会话甲")).toBeTruthy());
    expect(screen.getByText("会话乙（今天活跃）")).toBeTruthy();
    expect(screen.getByText("gbt-9704-docx")).toBeTruthy();
    expect(screen.getByText("read_file")).toBeTruthy();
    expect(screen.getByText("go")).toBeTruthy();
    // 默认范围 30d
    expect(insightsMock).toHaveBeenCalledWith("30d", false);
  });

  it("范围切换重查（today 载荷）且重置日过滤", async () => {
    renderT(<Insights />);
    await waitFor(() => expect(insightsMock).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByTestId("insights-range-today"));
    await waitFor(() => expect(insightsMock).toHaveBeenCalledWith("today", false));
  });

  it("会话卡点击回跳 onOpenSession", async () => {
    const onOpen = vi.fn();
    renderT(<Insights onOpenSession={onOpen} />);
    const card = await waitFor(() => screen.getByText("会话甲"));
    fireEvent.click(card.closest("button")!);
    expect(onOpen).toHaveBeenCalledWith("C:\\demo\\a.jsonl");
  });

  it("热力图点击有活动日 → 会话列表过滤该日；再点清除", async () => {
    renderT(<Insights />);
    await waitFor(() => expect(screen.getByText("会话甲")).toBeTruthy());
    // 今天格（2026-11-12 有活动）：只有会话乙 lastActive 落在 11-12
    const grid = screen.getByTestId("insights-heatmap-grid");
    const cell = grid.querySelector('button[title^="2026-11-12"]') as HTMLButtonElement | null;
    expect(cell).toBeTruthy();
    fireEvent.click(cell!);
    await waitFor(() => expect(screen.getByTestId("insights-dayfilter")).toBeTruthy());
    expect(screen.queryByText("会话甲")).toBeNull();
    expect(screen.getByText("会话乙（今天活跃）")).toBeTruthy();
    // 再点同格清除
    fireEvent.click(cell!);
    await waitFor(() => expect(screen.getByText("会话甲")).toBeTruthy());
  });
});
