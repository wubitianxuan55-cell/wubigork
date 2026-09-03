// v4.26 子代理 task 卡 live 化：taskActivity 注入点契约 + ToolCard 活动行渲染。
// 覆盖：provider 注入/卸载/抛错兜底、ref 解析（continue_from / Subagent
// reference 行）、缺省（null）按现状渲染、完成卡结果摘要。
import { describe, expect, it, afterEach } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import {
  getTaskCardActivity, hasTaskCardActivityProvider, matchRunningRun,
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

// ── 并行多子代理派发瞬间 task 卡空 ref 关联增强 ──────────────────────────
const runOf = (patch: Partial<Parameters<typeof matchRunningRun>[1][number]> = {}) =>
  ({ ref: "sa_20260903_01", status: "running", task: "梳理项目配置项", ...patch });

describe("matchRunningRun：空 ref 并行多 running 的文本匹配", () => {
  it("唯一命中：args 描述包含 run.task（模型短摘要 ↔ 长首条消息）", () => {
    const runs = [runOf(), runOf({ ref: "sa_2", task: "抓取价格数据" })];
    const args = JSON.stringify({ description: "先梳理项目配置项再汇总", prompt: "x" });
    expect(matchRunningRun(args, runs)?.ref).toBe("sa_20260903_01");
  });

  it("唯一命中（反方向）：run.task 包含 args 描述", () => {
    const runs = [runOf({ ref: "sa_2", task: "把梳理项目配置项做完并输出表格" })];
    const args = { description: "梳理项目配置项" };
    expect(matchRunningRun(args, runs)?.ref).toBe("sa_2");
  });

  it("多命中（≥2）：歧义返回 null，绝不误绑", () => {
    const runs = [runOf(), runOf({ ref: "sa_2" })];
    const args = JSON.stringify({ description: "梳理项目配置项" });
    expect(matchRunningRun(args, runs)).toBeNull();
  });

  it("零命中：null", () => {
    const runs = [runOf(), runOf({ ref: "sa_2", task: "抓取价格数据" })];
    const args = JSON.stringify({ description: "写发布公告" });
    expect(matchRunningRun(args, runs)).toBeNull();
  });

  it("args 无任务文本（undefined / 非 JSON 串 / 缺 description+prompt）：null 不炸", () => {
    const runs = [runOf()];
    expect(matchRunningRun(undefined, runs)).toBeNull();
    expect(matchRunningRun("not-json", runs)).toBeNull();
    expect(matchRunningRun(JSON.stringify({ subagent_type: "checker" }), runs)).toBeNull();
    expect(matchRunningRun(null, runs)).toBeNull();
    expect(matchRunningRun(42, runs)).toBeNull();
  });

  it("run.task 为空 / 缺失：该 run 跳过不炸", () => {
    const runs = [runOf({ task: "" }), runOf({ ref: "sa_2", task: "  " }), runOf({ ref: "sa_3", task: undefined })];
    const args = JSON.stringify({ description: "梳理项目配置项" });
    expect(matchRunningRun(args, runs)).toBeNull();
    // 空 task 的 run 不参与后，剩余唯一命中仍可绑定
    const mixed = [...runs, runOf({ ref: "sa_4" })];
    expect(matchRunningRun(args, mixed)?.ref).toBe("sa_4");
  });

  it("非 running 的 run 不参与匹配", () => {
    const runs = [runOf({ status: "completed" }), runOf({ ref: "sa_2", status: "failed" })];
    const args = JSON.stringify({ description: "梳理项目配置项" });
    expect(matchRunningRun(args, runs)).toBeNull();
  });

  it("args 为 ToolCard 原始 JSON 字符串形态（trim 后前后空白不阻断匹配）", () => {
    const runs = [runOf({ task: "  梳理项目配置项  " })];
    const args = JSON.stringify({ prompt: "\n梳理项目配置项\n" });
    expect(matchRunningRun(args, runs)?.ref).toBe("sa_20260903_01");
  });
});

describe("provider 签名升级兼容（ref + 可选 args）", () => {
  it("旧签名 provider (ref) => …：透传 args 时不炸、行为不变", () => {
    setTaskCardActivityProvider((ref: string) =>
      ref === "sa_1" ? { lastText: "旧签名动态" } : undefined);
    expect(getTaskCardActivity("sa_1", JSON.stringify({ description: "x" }))?.lastText).toBe("旧签名动态");
    expect(getTaskCardActivity("sa_2", JSON.stringify({ description: "x" }))).toBeUndefined();
    expect(getTaskCardActivity("sa_1")).toEqual({ lastText: "旧签名动态" });
  });

  it("getTaskCardActivity 把 args 原样透传给 provider 第二参", () => {
    const seen: { ref?: string; args?: unknown } = {};
    setTaskCardActivityProvider((ref, args) => {
      seen.ref = ref;
      seen.args = args;
      return undefined;
    });
    const raw = JSON.stringify({ description: "梳理配置项" });
    getTaskCardActivity("", raw);
    expect(seen).toEqual({ ref: "", args: raw });
    getTaskCardActivity("sa_1");
    expect(seen).toEqual({ ref: "sa_1", args: undefined });
  });
});

describe("ToolCard 空 ref 并行匹配通路（item.args 透传）", () => {
  it("派发初期（无 ref、多 running）：provider 用 args 文本命中即渲染对应动态", () => {
    // 模拟 App 注入侧并行多 running 的唯一命中路径
    setTaskCardActivityProvider((ref, args) => {
      const s = typeof args === "string" ? args : "";
      return ref === "" && s.includes("梳理配置项")
        ? { lastText: "并行命中：正在核对 config.json", state: "running" }
        : undefined;
    });
    const view = render(wrap(<ToolCard item={taskTool()} />));
    const live = view.container.querySelector('[data-testid="task-live"]');
    expect(live).not.toBeNull();
    expect(live?.textContent).toContain("并行命中：正在核对 config.json");
  });

  it("args 不命中：live 行仍在但不渲染任何预览文本（维持现状）", () => {
    setTaskCardActivityProvider((ref, args) => {
      const s = typeof args === "string" ? args : "";
      return ref === "" && s.includes("别的任务") ? { lastText: "错卡动态" } : undefined;
    });
    const view = render(wrap(<ToolCard item={taskTool()} />));
    const live = view.container.querySelector('[data-testid="task-live"]');
    expect(live).not.toBeNull();
    expect(live?.textContent).not.toContain("错卡动态");
  });
});
