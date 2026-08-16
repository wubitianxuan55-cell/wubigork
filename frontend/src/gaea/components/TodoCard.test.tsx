import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { Todo } from "../lib/tools";
import { TodoCard } from "./TodoCard";

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

const todos: Todo[] = [
  { content: "收集季度经营数据", status: "completed", level: 0 },
  { content: "生成图表并嵌入报告", status: "in_progress", level: 1, activeForm: "正在生成折线图…" },
  { content: "导出 docx 交付物", status: "pending", level: 1 },
];

describe("TodoCard 待办卡", () => {
  it("默认折叠：只显示头部与进度摘要，不渲染列表", () => {
    const { container } = render(wrap(<TodoCard todos={todos} onDismiss={() => {}} />));
    expect(screen.getByRole("button", { name: /expand|展开/i })).toBeTruthy();
    expect(screen.getByText("1/3 · 33%")).toBeTruthy();
    expect(screen.getByText(/进行中：正在生成折线图/)).toBeTruthy();
    expect(container.querySelector("ul")).toBeNull();
  });

  it("展开后渲染列表，已完成项收尾分组", () => {
    const { container } = render(wrap(<TodoCard todos={todos} onDismiss={() => {}} />));
    fireEvent.click(screen.getByRole("button", { name: /expand|展开/i }));
    expect(screen.getByRole("button", { name: /collapse|收起/i })).toBeTruthy();
    const rows = Array.from(container.querySelectorAll("ul li"));
    const texts = rows.map((r) => r.textContent?.trim() ?? "");
    expect(texts[0]).toContain("正在生成折线图");
    expect(texts[texts.length - 1]).toContain("收集季度经营数据"); // 已完成收尾
    expect(screen.getByText(/已完成 1/)).toBeTruthy();
  });

  it("全部完成时折叠态显示全部完成", () => {
    const allDone = todos.map((td) => ({ ...td, status: "completed" as const }));
    render(wrap(<TodoCard todos={allDone} onDismiss={() => {}} />));
    expect(screen.getByText("3/3")).toBeTruthy();
    expect(screen.getByText("全部完成")).toBeTruthy();
  });

  it("关闭按钮回调 onDismiss", () => {
    const onDismiss = vi.fn();
    render(wrap(<TodoCard todos={todos} onDismiss={onDismiss} />));
    fireEvent.click(screen.getByLabelText("关闭待办"));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});
