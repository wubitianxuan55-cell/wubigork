// v4.26 子代理 task 卡 live 化：taskActivity 注入点契约 + ToolCard 活动行渲染。
// 覆盖：provider 注入/卸载/抛错兜底、ref 解析（continue_from / Subagent
// reference 行）、缺省（null）按现状渲染、完成卡结果摘要。
import { describe, expect, it, afterEach } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import {
  getTaskCardActivity, hasTaskCardActivityProvider,
  resolveTaskRef, setTaskCardActivityProvider, taskResultSummary,
} from "./taskActivity";
import { ToolCard } from "../components/ToolCard";
import { ToolGroup } from "../components/ToolGroup";
import { LocaleProvider } from "./i18n";
import type { Item } from "./store";

type ToolItem = Extract<Item, { kind: "tool" }>;

afterEach(() => {
  // 卸载注入，避免串测
  setTaskCardActivityProvider(null);
});

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

const taskTool = (patch: Partial<ToolItem> = {}): ToolItem =>
  ({ kind: "tool", id: "t1", name: "task", args: JSON.stringify({ description: "梳理配置项", prompt: "把配置梳理成表" }), readOnly: false, status: "running", ...patch }) as ToolItem;

describe("resolveTaskRef / taskResultSummary 纯函数", () => {
  it("args.continue_from 优先（续跑子代理运行前即可得 ref）", () => {
    expect(resolveTaskRef(JSON.stringify({ continue_from: "sa_20260901_01", prompt: "x" }))).toBe("sa_20260901_01");
  });

  it("output 的 Subagent reference 行（tool_result 到达后可得）", () => {
    expect(resolveTaskRef("{}", "梳理完成。\nSubagent reference: sa_20260901_02")).toBe("sa_20260901_02");
    expect(resolveTaskRef("", "Subagent reference: sa_a.b_c-d")).toBe("sa_a.b_c-d");
  });

  it("解析不到 → 空串（provider 收到空串自行回退）", () => {
    expect(resolveTaskRef(JSON.stringify({ prompt: "x" }))).toBe("");
    expect(resolveTaskRef("", "")).toBe("");
    expect(resolveTaskRef("not-json", "没有引用行")).toBe("");
  });

  it("taskResultSummary：首条非空行、跳过引用行、超 80 字截断、error 为空", () => {
    expect(taskResultSummary("结果 OK\nSubagent reference: sa_1")).toBe("结果 OK");
    expect(taskResultSummary("\n\n  缩进行也认  \nSubagent reference: sa_1")).toBe("缩进行也认");
    const long = "x".repeat(90);
    expect(taskResultSummary(long)).toBe(`${"x".repeat(80)}…`);
    expect(taskResultSummary("结果", "出错了")).toBe("");
    expect(taskResultSummary(undefined)).toBe("");
  });
});

describe("setTaskCardActivityProvider 注入点契约", () => {
  it("未注入：取数 undefined、hasProvider false", () => {
    expect(hasTaskCardActivityProvider()).toBe(false);
    expect(getTaskCardActivity("sa_1")).toBeUndefined();
  });

  it("注入后按 ref 取数；卸载（null）恢复缺省", () => {
    setTaskCardActivityProvider((ref) =>
      ref === "sa_1" ? { lastText: "正在读取配置", lastTool: "read_file", state: "running" } : undefined);
    expect(hasTaskCardActivityProvider()).toBe(true);
    expect(getTaskCardActivity("sa_1")?.lastText).toBe("正在读取配置");
    expect(getTaskCardActivity("sa_2")).toBeUndefined();
    setTaskCardActivityProvider(null);
    expect(getTaskCardActivity("sa_1")).toBeUndefined();
  });

  it("provider 抛错不炸：吞掉返回 undefined", () => {
    setTaskCardActivityProvider(() => { throw new Error("boom"); });
    expect(getTaskCardActivity("sa_1")).toBeUndefined();
  });
});

describe("ToolCard task 卡 live 行", () => {
  it("未注入 provider（null）：按现状渲染，无 live 行", () => {
    const view = render(wrap(<ToolCard item={taskTool()} />));
    expect(view.container.querySelector('[data-testid="task-live"]')).toBeNull();
  });

  it("注入后运行中显示活动预览 / 已用时 / 查看分工", () => {
    setTaskCardActivityProvider((ref) =>
      ref === "sa_9"
        ? { lastText: "正在核对 config.json 的键位", lastTool: "read_file config.json", state: "running" }
        : undefined);
    const item = taskTool({ args: JSON.stringify({ description: "梳理配置", continue_from: "sa_9" }) });
    const view = render(wrap(<ToolCard item={item} />));
    const live = view.container.querySelector('[data-testid="task-live"]');
    expect(live).not.toBeNull();
    expect(live?.textContent).toContain("正在核对 config.json 的键位");
    expect(live?.textContent).toContain("read_file config.json");
    expect(live?.textContent).toContain("查看分工");
    expect(live?.textContent).toMatch(/\d+s/);
  });

  it("provider 查不到该 ref：live 行仍在（已用时+提示），但不渲染预览文本", () => {
    setTaskCardActivityProvider(() => undefined);
    const view = render(wrap(<ToolCard item={taskTool()} />));
    const live = view.container.querySelector('[data-testid="task-live"]');
    expect(live).not.toBeNull();
    expect(live?.textContent).not.toContain("正在核对");
  });

  it("完成后（tool_result 到达）：live 行消失，显示结果摘要", () => {
    setTaskCardActivityProvider(() => ({ lastText: "旧动态" }));
    const item = taskTool({
      status: "done",
      output: "已梳理 12 项配置。\nSubagent reference: sa_1",
    });
    const view = render(wrap(<ToolCard item={item} />));
    expect(view.container.querySelector('[data-testid="task-live"]')).toBeNull();
    expect(view.container.textContent).toContain("已梳理 12 项配置。");
  });
});

describe("ToolGroup 重复调用折叠（已调用 X · N 次）", () => {
  const tools: ToolItem[] = ["a", "b", "c"].map((id) =>
    ({ kind: "tool", id, name: "bash", args: JSON.stringify({ command: `echo ${id}` }), readOnly: false, status: "done" }));

  it("折叠行文案为「已调用 X · N 次」，点开展开明细", () => {
    const view = render(wrap(<ToolGroup tools={tools} />));
    expect(view.container.textContent).toContain("已调用 bash · 3 次");
    // 展开后明细可见（GSAP 折叠只改样式，DOM 常驻；点击验证交互通路）
    fireEvent.click(view.container.firstElementChild as Element);
    expect(view.container.textContent).toContain("echo a");
    expect(view.container.textContent).toContain("echo b");
    expect(view.container.textContent).toContain("echo c");
  });
});
