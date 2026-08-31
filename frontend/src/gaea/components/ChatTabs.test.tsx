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
});
