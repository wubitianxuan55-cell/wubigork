import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { SubagentsPanel } from "./SubagentsPanel";

describe("SubagentsPanel 多智能体分工可见（P2）", () => {
  it("展示分工列表：状态徽标 / 任务摘要 / 模型 / 工具范围", async () => {
    render(<SubagentsPanel sessionPath="s1.jsonl" />);
    // mock 返回 2 个子代理（running + completed）
    expect(await screen.findByText(/调研竞品表格 Agent 能力并总结可蒸馏点/)).toBeTruthy();
    expect(screen.getByText("进行中")).toBeTruthy();
    expect(screen.getByText("已完成")).toBeTruthy();
    expect(screen.getByText("deepseek-v4-flash")).toBeTruthy();
    expect(screen.getByText(/web_search/)).toBeTruthy();
  });

  it("点击展开查看子代理回答摘要", async () => {
    render(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText(/收集 2026 年办公 Agent 竞品更新信息/);
    // 展开 completed 子代理 → 显示回答摘要
    fireEvent.click(screen.getByText(/收集 2026 年办公 Agent 竞品更新信息/));
    expect(await screen.findByText(/WorkSwarm 蜂群智能体/)).toBeTruthy();
  });

  it("无 sessionPath 显示空状态（不请求）", async () => {
    render(<SubagentsPanel />);
    expect(await screen.findByText(/尚未派发子代理/)).toBeTruthy();
  });

  it("计数徽标显示总数与运行中数量", async () => {
    render(<SubagentsPanel sessionPath="s1.jsonl" />);
    await screen.findByText(/调研竞品表格 Agent 能力/);
    // 头部徽标：总数 2（runs.length）+ 运行中 1
    expect(screen.getByText("1 运行中")).toBeTruthy();
  });
});
