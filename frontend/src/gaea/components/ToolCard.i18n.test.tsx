// v4.57 i18n 收尾冒烟：ToolCard（diffstat title / 输出头部 / 展开收起 / TaskLiveRow 查看分工）
// 三语字典接线。钉住 zh 断言原硬编码文案。
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { ToolCard } from "./ToolCard";
import { LocaleProvider } from "../lib/i18n";
import { setTaskCardActivityProvider } from "../lib/taskActivity";
import type { Item } from "../lib/store";

const renderT = (ui: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const toolItem: Item = {
  kind: "tool",
  id: "t1",
  name: "edit_file",
  args: "{\"path\":\"docs/a.md\",\"old_string\":\"a\",\"new_string\":\"b\\nc\"}",
  readOnly: false,
  status: "done",
  output: Array.from({ length: 40 }, (_, i) => `line ${i}`).join("\n") + "\n" + Array.from({ length: 40 }, (_, i) => `tail ${i}`).join("\n"),
};

describe("ToolCard i18n 冒烟", () => {
  afterEach(() => {
    cleanup();
    // TaskLiveRow 契约：未注入 provider 时整行不渲染 → 用例内注入，用例后卸载
    setTaskCardActivityProvider(null);
  });
  it("diffstat 芯片 title 与输出头部「输出 · NL」走字典（zh）", () => {
    renderT(<ToolCard item={toolItem} />);
    expect(screen.getByTitle("行级增减")).toBeTruthy();
    fireEvent.click(screen.getByText("edit_file"));
    expect(screen.getByText(/输出 · \d+L/)).toBeTruthy();
  });

  it("有界输出折叠开关：展开全部 N 行 / 收起输出（zh）", () => {
    renderT(<ToolCard item={toolItem} />);
    fireEvent.click(screen.getByText("edit_file"));
    const expand = screen.getByText(/展开全部 \d+ 行/);
    fireEvent.click(expand);
    expect(screen.getByText("收起输出")).toBeTruthy();
  });

  it("task 卡运行中显示「查看分工」活动行（zh）", () => {
    setTaskCardActivityProvider(() => ({ lastText: "正在收集资料", state: "running" }));
    const taskItem: Item = {
      kind: "tool",
      id: "t2",
      name: "task",
      args: "{\"task\":\"调研\"}",
      readOnly: false,
      status: "running",
    };
    renderT(<ToolCard item={taskItem} />);
    expect(screen.getByTestId("task-live")).toBeTruthy();
    expect(screen.getByText("查看分工")).toBeTruthy();
  });
});
