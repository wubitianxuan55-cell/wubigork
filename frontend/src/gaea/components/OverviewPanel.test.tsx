import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { OverviewPanel } from "./OverviewPanel";
import type { StoredData } from "./StatsPanel";
import type { WireUsage } from "../lib/types";

// 与 StatsPanel.test.tsx 同形的假数据（概览 tab 冒烟钉住 props 透传不失真）
const DATA: StoredData = {
  turns: [
    { turn: 1, prompt: 1200, completion: 200, cacheHit: 800, cacheMiss: 400, cost: 0.05, totalTokens: 1400 },
  ],
  steps: [
    { step: 1, prompt: 1200, completion: 200, cacheHit: 800, cacheMiss: 400, cost: 0.05, source: "executor" },
  ],
};

const EXEC_USAGE: WireUsage = {
  promptTokens: 900,
  completionTokens: 150,
  totalTokens: 1050,
  cacheHitTokens: 600,
  cacheMissTokens: 300,
  sessionCacheHitTokens: 600,
  sessionCacheMissTokens: 300,
  costUsd: 0.03,
  source: "main",
};

function renderPanel(overrides?: Partial<Parameters<typeof OverviewPanel>[0]>) {
  return render(
    <OverviewPanel
      data={DATA}
      clearData={() => {}}
      turnSteps={[]}
      toolCounts={{}}
      skillCounts={{}}
      {...overrides}
    />,
  );
}

describe("OverviewPanel 主区概览 tab（v4.23 A4 统计迁移）", () => {
  it("容器撑满主区看板（h-full，滚动收敛在 StatsPanel 内部）", () => {
    const { container } = renderPanel();
    expect(container.firstElementChild?.className).toContain("h-full");
    expect(container.firstElementChild?.className).toContain("min-h-0");
  });

  it("透传 data：渲染会话统计（轮次表/Token/命中率）", () => {
    renderPanel();
    expect(screen.getByText(/会话 \(1轮·1步\)/)).toBeTruthy();
    expect(screen.getByText("1,200")).toBeTruthy();
    expect(screen.getByText("¥0.0500")).toBeTruthy();
    // 命中率 800/(800+400) = 66.67%（会话汇总 + 当前步均出现）
    expect(screen.getAllByText("66.67%").length).toBeGreaterThan(0);
  });

  it("空数据时显示 StatsPanel 空态占位", () => {
    renderPanel({ data: { turns: [], steps: [] } });
    expect(screen.getByText("暂无统计数据")).toBeTruthy();
  });

  it("透传 toolCounts/skillCounts/perTurnExecutorUsage：本轮表格与调用 chips", () => {
    renderPanel({
      toolCounts: { Bash: 2 },
      skillCounts: { report: 1 },
      perTurnExecutorUsage: EXEC_USAGE,
    });
    expect(screen.getByText(/工具调用/)).toBeTruthy();
    expect(screen.getByText("Bash")).toBeTruthy();
    expect(screen.getByText("report")).toBeTruthy();
    // 本轮级表格按 perTurnExecutorUsage 渲染
    expect(screen.getByText(/本轮 \(0步\)/)).toBeTruthy();
  });

  it("透传 clearData：点击「清空统计」触发回调", () => {
    const clearData = vi.fn();
    renderPanel({ clearData });
    fireEvent.click(screen.getByText("清空统计"));
    expect(clearData).toHaveBeenCalledTimes(1);
  });
});
