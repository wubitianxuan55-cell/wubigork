import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Trajectory } from "../lib/types";

const trajectoryMock = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: { Trajectory: trajectoryMock },
  openExternal: vi.fn(),
}));

const TRAJECTORY: Trajectory = {
  ok: true,
  turns: [
    {
      turn: 1,
      startedAt: 1750000000,
      end: { seq: 30, ts: 1750000100 },
      records: [
        { seq: 2, kind: "user", ts: 1750000001, user: { text: "调研 internal/gaea/config" } },
        {
          seq: 3, kind: "header", ts: 1750000002, step: 1,
          header: { system: "系统提示词预览", toolCount: 50, tokens: 12500, change: "initial" },
        },
        {
          seq: 4, kind: "assistant", ts: 1750000003, step: 1,
          assistant: { reasoning: "先搜索", text: "配置集中在 config.go", usage: { promptTokens: 350600, completionTokens: 93 } },
        },
        {
          seq: 8, kind: "tool", ts: 1750000004, step: 1, durationMs: 3200,
          tool: { id: "t1", name: "pwsh", args: "{\"command\":\"git status\"}", output: "M App.tsx", status: "ok" },
        },
        {
          seq: 12, kind: "tool", ts: 1750000008, step: 1, durationMs: 1500,
          tool: { id: "t2", name: "bash", args: "rm -rf /", output: "", err: "denied by policy", status: "error" },
        },
        { seq: 15, kind: "ask", ts: 1750000010, step: 1, ask: { question: "如何协调并行改动？" } },
      ],
    },
  ],
  betweenTurns: [
    { seq: 31, kind: "compact", ts: 1750000100, compact: { trigger: "manual", summary: "轮间压缩" } },
  ],
};

describe("TrajectoryView 轨迹事件账本", () => {
  // 全量套件高负载下 jsdom 调度会被饿死，RTL 默认 1s 超时出现过 flaky；
  // 显式放宽到 5s（仍有上界，不会掩盖真回归）。
  const LOAD = { timeout: 5000 };

  beforeEach(() => {
    trajectoryMock.mockReset();
    trajectoryMock.mockResolvedValue(TRAJECTORY);
  });

  it("渲染统计 chips、轮次与记录行", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/Turns 1/, undefined, LOAD)).toBeTruthy();
    expect(screen.getByText(/Calls 2/)).toBeTruthy();
    expect(screen.getByText(/Duration/)).toBeTruthy();
    expect(screen.getByText("第1轮")).toBeTruthy();
    expect(screen.getByText("ASSISTANT")).toBeTruthy();
    expect(screen.getAllByText("TOOL").length).toBe(2);
    expect(screen.getByText(/调研 internal\/gaea\/config/)).toBeTruthy();
  });

  it("展开工具记录显示参数/结果/错误", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    // 第一个 TOOL 行的按钮：文本含 pwsh
    const btn = await screen.findByRole("button", { name: /pwsh/ }, LOAD);
    fireEvent.click(btn);
    expect((await screen.findAllByText(/"command":"git status"/, undefined, LOAD)).length).toBeGreaterThan(0);
    expect((await screen.findAllByText("M App.tsx", undefined, LOAD)).length).toBeGreaterThan(0);
    expect(await screen.findByText(/denied by policy/, undefined, LOAD)).toBeTruthy();
  });

  it("渲染 ask 与 Between-turns 压缩记录", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/如何协调并行改动？/, undefined, LOAD)).toBeTruthy();
    expect(screen.getByText(/Between turns/)).toBeTruthy();
    expect(screen.getByText(/轮间压缩/)).toBeTruthy();
  });

  it("搜索过滤记录", async () => {
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    const input = await screen.findByPlaceholderText("搜索", undefined, LOAD);
    fireEvent.change(input, { target: { value: "git status" } });
    expect(screen.getByText(/pwsh/)).toBeTruthy();
    expect(screen.queryByText(/配置集中在 config.go/)).toBeNull();
  });

  it("空态提示", async () => {
    trajectoryMock.mockResolvedValue({ ok: true, turns: [] });
    const { TrajectoryView } = await import("./TrajectoryView");
    render(<TrajectoryView running={false} />);
    expect(await screen.findByText(/暂无轨迹记录/, undefined, LOAD)).toBeTruthy();
  });
});
