import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { pushSpeedPt, rollingTokPerSec, RunStatus, SPEED_WINDOW_S, type SpeedPt } from "./AppStatus";

// 上下文占用已收敛到 ComposerToolbar 的 ContextPill；输入框上方 RunStatus
// 只保留运行时速度信息，不再重复显示百分比/压缩预警。
describe("RunStatus 上下文状态去重", () => {
  const base = { running: true, turnStartAt: 0, turnTokens: 0 };

  it("不再在输入框上方显示上下文百分比", () => {
    render(<RunStatus {...base} />);
    expect(screen.queryByText(/%/)).toBeNull();
    expect(screen.queryByText(/接近自动压缩/)).toBeNull();
    expect(screen.queryByText(/上下文窗口占用/)).toBeNull();
  });
});

// 流式速度计：4s 滚动窗口差商（turnTokens 由 store 随分片推进，useNow 每秒
// 走针采样）；停顿自然衰减到 0，同秒突发覆盖取最大。
describe("RunStatus 滚动 tok/s 速度计", () => {
  it("窗口内最早样本与当前值差商", () => {
    const pts: SpeedPt[] = [];
    pushSpeedPt(pts, 1000, 0);
    pushSpeedPt(pts, 1001, 40);
    pushSpeedPt(pts, 1002, 80);
    pushSpeedPt(pts, 1004, 160);
    // 基点 = 1000（≤ 1004-4）→ (160-0)/4 = 40 tok/s
    expect(rollingTokPerSec(pts, 1004, 160)).toBeCloseTo(40, 5);
  });

  it("不足窗口时长时退化为首样本平均速度", () => {
    const pts: SpeedPt[] = [];
    pushSpeedPt(pts, 1000, 0);
    pushSpeedPt(pts, 1002, 60);
    // 基点 = 首样本 1000 → (60-0)/2 = 30 tok/s
    expect(rollingTokPerSec(pts, 1002, 60)).toBeCloseTo(30, 5);
  });

  it("同秒多分片覆盖取最大，不堆重复点", () => {
    const pts: SpeedPt[] = [];
    pushSpeedPt(pts, 1000, 0);
    pushSpeedPt(pts, 1000, 30);
    pushSpeedPt(pts, 1000, 12);
    expect(pts.length).toBe(1);
    expect(pts[0][1]).toBe(30);
  });

  it("停顿（tokens 不涨）速度衰减到 0", () => {
    const pts: SpeedPt[] = [];
    pushSpeedPt(pts, 1000, 100);
    pushSpeedPt(pts, 1005, 100);
    expect(rollingTokPerSec(pts, 1005, 100)).toBe(0);
  });

  it("样本数裁剪到窗口+2", () => {
    const pts: SpeedPt[] = [];
    for (let i = 0; i < 20; i++) pushSpeedPt(pts, 1000 + i, i * 10);
    expect(pts.length).toBe(SPEED_WINDOW_S + 2);
  });

  it("空样本或零时长返回 0", () => {
    expect(rollingTokPerSec([], 1000, 50)).toBe(0);
    const pts: SpeedPt[] = [];
    pushSpeedPt(pts, 1000, 10);
    expect(rollingTokPerSec(pts, 1000, 50)).toBe(0);
  });

  it("速度 <0.5 tok/s 时底栏不显示速度段", () => {
    render(<RunStatus running turnStartAt={Date.now() - 5000} turnTokens={0} />);
    expect(screen.queryByText(/tok\/s/)).toBeNull();
  });

  it("停顿期（无实时流）显示回合均速（均 前缀）", () => {
    // turnStartAt 10s 前、已流 500 tokens：实时窗口无样本 → 退化为均速 ~50 tok/s
    render(
      <RunStatus running turnStartAt={Date.now() - 10000} turnTokens={500} />,
    );
    expect(screen.getByText(/均 \d+(\.\d+)? tok\/s/)).toBeTruthy();
  });
});
