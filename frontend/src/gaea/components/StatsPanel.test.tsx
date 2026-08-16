import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatsPanel, type StoredData } from "./StatsPanel";

const REAL_DATA: StoredData = {
  turns: [
    { turn: 1, prompt: 1200, completion: 200, cacheHit: 800, cacheMiss: 400, cost: 0.05, totalTokens: 1400 },
  ],
  steps: [
    { step: 1, prompt: 1200, completion: 200, cacheHit: 800, cacheMiss: 400, cost: 0.05, source: "executor" },
  ],
};

describe("StatsPanel 统计面板", () => {
  it("会话级统计按真实用量聚合展示（Token/成本/命中率不为 0）", () => {
    render(
      <StatsPanel
        data={REAL_DATA}
        clearData={() => {}}
        turnSteps={[]}
        toolCounts={{}}
        skillCounts={{}}
      />,
    );

    // 会话 (1轮·1步)：prompt 1,200 / completion 200 / 成本 ¥0.0500
    expect(screen.getByText(/会话 \(1轮·1步\)/)).toBeTruthy();
    expect(screen.getByText("1,200")).toBeTruthy();
    expect(screen.getByText("200")).toBeTruthy();
    expect(screen.getByText("¥0.0500")).toBeTruthy();
    // 命中率 800/(800+400) = 66.67%（会话汇总 + 当前步均显示）
    expect(screen.getAllByText("66.67%").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/800 命中 \/ 400 未命中/).length).toBeGreaterThan(0);

    // 当前步明细
    expect(screen.getByText(/当前步 #1/)).toBeTruthy();
    expect(screen.getByText(/Prompt 1,200/)).toBeTruthy();
    expect(screen.getByText(/Compl 200/)).toBeTruthy();
  });

  it("空数据时显示占位提示而非 0 表格", () => {
    render(
      <StatsPanel
        data={{ turns: [], steps: [] }}
        clearData={() => {}}
        turnSteps={[]}
        toolCounts={{}}
        skillCounts={{}}
      />,
    );
    expect(screen.getByText("暂无统计数据")).toBeTruthy();
  });

  it("有派生统计（available）时展示历史累计块，且空数据不再显示占位", () => {
    render(
      <StatsPanel
        data={{ turns: [], steps: [] }}
        clearData={() => {}}
        turnSteps={[]}
        toolCounts={{}}
        skillCounts={{}}
        sessionStats={{
          available: true,
          stats: {
            promptTokens: 128000,
            completionTokens: 32000,
            totalTokens: 160000,
            cacheHitTokens: 96000,
            cacheMissTokens: 32000,
            reasoningTokens: 0,
            usageCount: 12,
            cost: 0.284,
            currency: "usd",
            mainCost: 0.211,
            subagentCost: 0.073,
          },
        }}
      />,
    );
    // 历史累计块
    expect(screen.getByText(/会话历史累计（日志派生）/)).toBeTruthy();
    expect(screen.getByText("12 次调用")).toBeTruthy();
    expect(screen.getByText("160,000")).toBeTruthy();
    expect(screen.getByText("¥0.2840")).toBeTruthy();
    // 命中率 96000/(96000+32000) = 75%
    expect(screen.getByText("75.0%")).toBeTruthy();
    // 不再显示空态占位
    expect(screen.queryByText("暂无统计数据")).toBeNull();
  });

  it("available=false 时不展示历史累计块，空数据仍显示占位", () => {
    render(
      <StatsPanel
        data={{ turns: [], steps: [] }}
        clearData={() => {}}
        turnSteps={[]}
        toolCounts={{}}
        skillCounts={{}}
        sessionStats={{ available: false, stats: { promptTokens: 0, completionTokens: 0, totalTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, reasoningTokens: 0, usageCount: 0, cost: 0, mainCost: 0, subagentCost: 0 } }}
      />,
    );
    expect(screen.queryByText(/会话历史累计（日志派生）/)).toBeNull();
    expect(screen.getByText("暂无统计数据")).toBeTruthy();
  });
});
