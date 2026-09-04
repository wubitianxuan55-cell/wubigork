import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { ChatTabs } from "./ChatTabs";

describe("ChatTabs 对话标签栏（v4.73 记忆 tab 迁入主区）", () => {
  it("渲染 4 个 tab：对话/轨迹/上下文/记忆，顺序一致", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const labels = Array.from(document.querySelectorAll("button")).map((b) => b.textContent);
    expect(labels).toEqual(["对话", "轨迹", "上下文", "记忆"]);
  });

  it("点击「记忆」触发 onChange 并携带 memory", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    fireEvent.click(screen.getByText("记忆").closest("button") as HTMLElement);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith("memory");
  });

  it("active=memory 时记忆高亮、其余 tab 不高亮", () => {
    render(<ChatTabs active="memory" onChange={() => {}} />);
    const memoryBtn = screen.getByText("记忆").closest("button") as HTMLElement;
    const chatBtn = screen.getByText("对话").closest("button") as HTMLElement;
    expect(memoryBtn.className).toContain("text-accent");
    expect(chatBtn.className).not.toContain("text-accent");
  });

  it("轨迹 tab 保留 tooltip（工具调用/步骤时间线）", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const trajectoryBtn = screen.getByText("轨迹").closest("button") as HTMLElement;
    expect(trajectoryBtn.getAttribute("title")).toBe("工具调用/步骤时间线");
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
