import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { CHAT_VIEW_TAB_IDS, ChatTabs } from "./ChatTabs";

// 2026-11-17 审查批4 自 gaea 树移植（tab 白名单事故族唯一自动化防线——
// characters/reading/memory 三次真机定罪的回归锚）。

describe("ChatTabs 对话标签栏（图文八 tab）", () => {
  it("渲染 8 个 tab：对话/轨迹/上下文/洞察/角色/阅读/设定/记忆，顺序一致（插图移右栏 2026-11-18）", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const labels = Array.from(document.querySelectorAll("button")).map((b) => b.textContent);
    expect(labels).toEqual(["对话", "轨迹", "上下文", "洞察", "角色", "阅读", "设定", "记忆"]);
  });

  it("CHAT_VIEW_TAB_IDS 单源与 TABS 渲染序一致（白名单漏名=tab 被当子代理 tab 吃掉）", () => {
    expect(CHAT_VIEW_TAB_IDS).toHaveLength(8);
    expect(CHAT_VIEW_TAB_IDS).toEqual([
      "chat", "trajectory", "context", "insights", "characters", "reading", "settings", "memory",
    ]);
  });

  it("点击「记忆」触发 onChange 并携带 memory", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    fireEvent.click(screen.getByText("记忆").closest("button") as HTMLElement);
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith("memory");
  });

  it("点击「角色」触发 onChange 并携带 characters（三犯定罪位回归锚）", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    fireEvent.click(screen.getByText("角色").closest("button") as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("characters");
  });

  it("active=memory 时记忆高亮、其余 tab 不高亮", () => {
    render(<ChatTabs active="memory" onChange={() => {}} />);
    const memoryBtn = screen.getByText("记忆").closest("button") as HTMLElement;
    const chatBtn = screen.getByText("对话").closest("button") as HTMLElement;
    expect(memoryBtn.className).toContain("text-accent");
    expect(chatBtn.className).not.toContain("text-accent");
  });

  it("v4.412 子集渲染：缺省全量，传 tabs 只挂子集", () => {
    const { unmount } = render(
      <ChatTabs active="chat" onChange={() => {}} tabs={["chat", "trajectory", "context"]} />,
    );
    const labels = Array.from(document.querySelectorAll("button")).map((b) => b.textContent);
    expect(labels).toEqual(["对话", "轨迹", "上下文"]);
    unmount();
    render(<ChatTabs active="chat" onChange={() => {}} />);
    expect(Array.from(document.querySelectorAll("button")).map((b) => b.textContent)).toEqual(
      ["对话", "轨迹", "上下文", "洞察", "角色", "阅读", "设定", "记忆"],
    );
  });

  it("v4.412 标签覆盖：主 tab 显示「故事」，其余回落内置文案", () => {
    render(
      <ChatTabs
        active="chat"
        onChange={() => {}}
        tabs={["chat", "trajectory", "context"]}
        labelOverrides={{ chat: "故事" }}
      />,
    );
    const labels = Array.from(document.querySelectorAll("button")).map((b) => b.textContent);
    expect(labels).toEqual(["故事", "轨迹", "上下文"]);
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
    expect(screen.getByTitle("收集竞品更新 ｜ 已完成").getAttribute("aria-selected")).toBe("false");
    const runningTabEl = document.querySelector('[data-chat-session-tab="sub:sa_1"]');
    expect(runningTabEl?.querySelector('button[aria-label^="关闭"]')).toBeTruthy();
  });
});

describe("ChatTabs tab 语义（a11y）", () => {
  it("容器 role=tablist；主 tab role=tab + aria-selected；仅选中项在 tab 序", () => {
    render(<ChatTabs active="chat" onChange={() => {}} />);
    const list = screen.getByRole("tablist");
    const tabs = within(list).getAllByRole("tab");
    expect(tabs).toHaveLength(8);
    expect(tabs[0].getAttribute("aria-selected")).toBe("true");
    expect(tabs.slice(1).every((t) => t.getAttribute("aria-selected") === "false")).toBe(true);
    expect(tabs[0].tabIndex).toBe(0);
    expect(tabs.slice(1).every((t) => t.tabIndex === -1)).toBe(true);
  });

  it("方向键导航：在 chat 上按 → 聚焦并选中轨迹", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    const list = screen.getByRole("tablist");
    fireEvent.keyDown(list, { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith("trajectory");
  });

  it("Home/End 导航到首尾 tab（图文末位=记忆）", () => {
    const onChange = vi.fn();
    render(<ChatTabs active="chat" onChange={onChange} />);
    fireEvent.keyDown(screen.getByRole("tablist"), { key: "End" });
    expect(onChange).toHaveBeenCalledWith("memory");
    fireEvent.keyDown(screen.getByRole("tablist"), { key: "Home" });
    expect(onChange).toHaveBeenCalledWith("chat");
  });
});
