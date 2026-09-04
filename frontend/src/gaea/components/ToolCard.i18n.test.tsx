// v4.57 i18n 收尾冒烟：ToolCard（diffstat title / 输出头部 / 展开收起 / TaskLiveRow 查看分工）
// 三语字典接线。钉住 zh 断言原硬编码文案。
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { ToolCard } from "./ToolCard";
import { LocaleProvider } from "../lib/i18n";
import { setTaskCardActivityProvider, setTaskCardOpenHandler, setTaskCardOpenTarget } from "../lib/taskActivity";
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

  it("task 卡运行中显示「点击打开子代理会话」活动行（zh）", () => {
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
    expect(screen.getByText("点击打开子代理会话")).toBeTruthy();
  });
});

// ── v4.63 子代理卡片整卡可点：task 卡点击派发「打开会话」跳转 ──
describe("ToolCard 子代理卡片点击跳转（v4.63）", () => {
  afterEach(() => {
    cleanup();
    setTaskCardOpenHandler(null);
    setTaskCardOpenTarget(null);
    setTaskCardActivityProvider(null);
  });

  const taskCard: Item = {
    kind: "tool",
    id: "task-1",
    name: "task",
    args: "{}",
    readOnly: false,
    status: "done",
    output: "调研完成。\nSubagent reference: sa_abc123",
  };

  it("解析到 ref：点击整卡派发跳转（不再走展开折叠）", () => {
    const opened: string[] = [];
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref) => ref);
    renderT(<ToolCard item={taskCard} />);
    fireEvent.click(screen.getByText("task"));
    expect(opened).toEqual(["sa_abc123"]);
  });

  it("无注入（解析不到 ref）：点击不跳转，维持展开折叠现状", () => {
    renderT(<ToolCard item={taskCard} />);
    fireEvent.click(screen.getByText("task"));
    // 展开行为：输出区可见（默认折叠，点击后展开）
    expect(screen.getByText(/输出 · /)).toBeTruthy();
  });

  it("空 ref 回退：唯一 running 命中时由 App 解析器给出目标 ref", () => {
    const opened: string[] = [];
    setTaskCardOpenHandler((ref) => opened.push(ref));
    setTaskCardOpenTarget((ref, args) => (ref === "" && String(args).includes("土壤修复") ? "sa_running9" : ""));
    renderT(
      <ToolCard
        item={{ ...taskCard, output: "", args: "{\"description\":\"土壤修复技术调研\"}", status: "running" }}
      />,
    );
    fireEvent.click(screen.getByText("task"));
    expect(opened).toEqual(["sa_running9"]);
  });
});
