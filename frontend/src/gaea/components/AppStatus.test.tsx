import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { RunStatus } from "./AppStatus";

// C5：运行状态行显示上下文窗口占用百分比 + 压缩前预警（75%/90% 两档，
// 对齐 gaea 压缩触发线 80% / 强制线 90%）；窗口未知时隐藏。
describe("RunStatus 上下文占用状态行（C5）", () => {
  const base = { running: true, turnStartAt: 0, turnTokens: 0 };

  it("窗口未知（window=0）时隐藏占用段", () => {
    render(<RunStatus {...base} used={123000} window={0} />);
    expect(screen.queryByText(/%/)).toBeNull();
    expect(screen.queryByText(/接近自动压缩/)).toBeNull();
  });

  it("占用 60% 仅显示百分比，不预警", () => {
    render(<RunStatus {...base} used={60000} window={100000} />);
    expect(screen.getByText("60%")).toBeTruthy();
    expect(screen.queryByText(/接近自动压缩/)).toBeNull();
  });

  it("占用 80% 显示「接近自动压缩」预警", () => {
    render(<RunStatus {...base} used={80000} window={100000} />);
    expect(screen.getByText("80%")).toBeTruthy();
    expect(screen.getByText(/接近自动压缩/)).toBeTruthy();
  });

  it("占用 95% 升级为「即将强制压缩」", () => {
    render(<RunStatus {...base} used={95000} window={100000} />);
    expect(screen.getByText(/即将强制压缩/)).toBeTruthy();
    expect(screen.queryByText(/接近自动压缩/)).toBeNull();
  });
});
