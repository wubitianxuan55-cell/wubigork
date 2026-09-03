import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { ChatTabs } from "./ChatTabs";

describe("ChatTabs 对话标签栏（v4.23 新增「概览」tab）", () => {
  it("渲染 4 个 tab：对话/轨迹/上下文/概览，顺序一致", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const labels = Array.from(document.querySelectorAll("button")).map((b) => b.textContent);
    expect(labels).toEqual(["对话", "轨迹", "上下文", "概览"]);
  });

  it("点击「概览」触发 onChange 并携带 overview", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    fireEvent.click(screen.getByText("概览").closest("button") as HTMLElement);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith("overview");
  });

  it("active=overview 时概览高亮、其余 tab 不高亮", () => {
    render(<ChatTabs active="overview" onChange={() => {}} />);
    const overviewBtn = screen.getByText("概览").closest("button") as HTMLElement;
    const chatBtn = screen.getByText("对话").closest("button") as HTMLElement;
    expect(overviewBtn.className).toContain("text-accent");
    expect(chatBtn.className).not.toContain("text-accent");
  });

  it("概览 tab 带统计提示 tooltip（与轨迹 tab 的 title 同风格）", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const overviewBtn = screen.getByText("概览").closest("button") as HTMLElement;
    expect(overviewBtn.getAttribute("title")).toBe("Token/成本/命中率统计");
  });

  it("子代理会话 tab：状态点随运行态着色、title 显示完整详情（同主代理口径）", () => {
    render(
      <ChatTabs
        active="sub:sa_1"
        onChange={() => {}}
        extraTabs={[
          {
            id: "sub:sa_1",
            label: "调研竞品表格 Agent…",
            status: "running",
            detail: "调研竞品表格 Agent 能力 ｜ 进行中 · deepseek-v4-flash",
          },
          {
            id: "sub:sa_2",
            label: "收集竞品更新",
            status: "completed",
            detail: "收集竞品更新 ｜ 已完成",
          },
        ]}
        onCloseExtra={() => {}}
      />,
    );
    const runningTab = screen.getByTitle(/调研竞品表格 Agent 能力 ｜ 进行中/);
    expect(runningTab).toBeTruthy();
    const runningDot = screen.getByTestId("chat-tab-status-sub:sa_1");
    expect(runningDot.style.background).toBe("var(--gaea-glow)");
    const doneDot = screen.getByTestId("chat-tab-status-sub:sa_2");
    expect(doneDot.style.background).toBe("var(--md-sys-color-success)");
    // 选中态与关闭入口保留（与主对话 tab 同区展示）
    expect(screen.getByTitle("收集竞品更新 ｜ 已完成").getAttribute("aria-selected")).toBe("false");
    const runningTabEl = document.querySelector('[data-chat-session-tab="sub:sa_1"]');
    expect(runningTabEl?.querySelector('button[aria-label^="关闭"]')).toBeTruthy();
  });
});
