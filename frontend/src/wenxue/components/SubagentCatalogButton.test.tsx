// SubagentCatalogButton — 顶栏「N 个子智能体」目录树测试（2026-11-18，dsh
// S2/S3/S4 形态）：触发器可见性（runs>0 才渲染）、血缘树多级缩进（L 徽标）、
// 行点击=主区打开、侧栏钮=右栏打开、mt_ 不进目录。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { SubagentCatalogButton } from "./SubagentCatalogButton";
import { LocaleProvider } from "../lib/i18n";
import type { SubagentRunView } from "../lib/types";

const mocks = vi.hoisted(() => ({
  subscribe: vi.fn(),
}));

vi.mock("../lib/subagentRunsStore", () => ({
  subscribeSubagentRuns: (...args: unknown[]) => mocks.subscribe(...args),
}));

const wrap = (node: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

const RUNS: SubagentRunView[] = [
  { ref: "sa_1", status: "completed", task: "一代任务", toolCalls: 0, createdAt: "", updatedAt: "" },
  { ref: "sa_2", status: "running", task: "二代任务", parentRef: "sa_1", toolCalls: 2, lastTool: "web_fetch", createdAt: "", updatedAt: "" },
  { ref: "sa_3", status: "queued", task: "三代任务", parentRef: "sa_2", toolCalls: 0, createdAt: "", updatedAt: "" },
  { ref: "mt_9", status: "completed", task: "本地摘要", toolCalls: 0, kind: "model_tool", tool: "vision", createdAt: "", updatedAt: "" },
];

beforeEach(() => {
  vi.clearAllMocks();
  // subscribeSubagentRuns(path, cb) 立即拉一次的契约：同步回调快照。
  mocks.subscribe.mockImplementation((_path: string, cb: (runs: SubagentRunView[]) => void) => {
    cb(RUNS);
    return () => {};
  });
});

describe("SubagentCatalogButton 子智能体目录树（dsh S2/S3/S4）", () => {
  it("触发器：runs>0 才渲染，计数=排除 mt_；空会话返回 null", () => {
    const { container } = render(wrap(
      <SubagentCatalogButton sessionPath="s1.jsonl" onOpenMain={() => {}} />,
    ));
    const trigger = screen.getByTestId("subagent-catalog-trigger");
    // 3 条 sa_（mt_ 不计）
    expect(trigger.textContent).toContain("3 个子智能体");
    expect(trigger.textContent).not.toContain("4");

    cleanup();
    mocks.subscribe.mockImplementation((_p: string, cb: (r: SubagentRunView[]) => void) => {
      cb([]);
      return () => {};
    });
    const empty = render(wrap(<SubagentCatalogButton sessionPath="s1.jsonl" onOpenMain={() => {}} />));
    expect(empty.container.querySelector("[data-testid='subagent-catalog']")).toBeNull();
    expect(container).toBeTruthy();
  });

  it("下拉树：血缘行全在（三代链+独立一代）、mt_ 不进目录、二代带 L1 徽标、running 行有侧栏钮", () => {
    render(wrap(<SubagentCatalogButton sessionPath="s1.jsonl" onOpenMain={() => {}} onOpenAside={() => {}} />));
    fireEvent.click(screen.getByTestId("subagent-catalog-trigger"));
    const rows = screen.getAllByTestId("subagent-catalog-row");
    expect(rows).toHaveLength(3); // mt_ 不进
    const text = rows.map((r) => r.textContent).join("|");
    expect(text).toContain("一代任务");
    expect(text).toContain("二代任务");
    expect(text).toContain("三代任务");
    expect(text).not.toContain("本地摘要");
    expect(text).toContain("L1");
    expect(text).toContain("L2");
    // running 行有「在侧边栏打开」钮
    expect(screen.getAllByTestId("subagent-catalog-aside").length).toBeGreaterThan(0);
  });

  it("行点击=主区打开并收起下拉；侧栏钮=右栏打开（stopPropagation 不触发行）", () => {
    const onOpenMain = vi.fn();
    const onOpenAside = vi.fn();
    render(wrap(
      <SubagentCatalogButton sessionPath="s1.jsonl" onOpenMain={onOpenMain} onOpenAside={onOpenAside} />,
    ));
    fireEvent.click(screen.getByTestId("subagent-catalog-trigger"));
    const rows = screen.getAllByTestId("subagent-catalog-row");
    fireEvent.click(rows[0]); // 第一行=一代任务
    expect(onOpenMain).toHaveBeenCalledWith(expect.objectContaining({ ref: "sa_1" }));
    expect(screen.queryByTestId("subagent-catalog-dropdown")).toBeNull();

    fireEvent.click(screen.getByTestId("subagent-catalog-trigger"));
    // 侧栏钮每行都有（sa_ 运行皆可续聊/查看）；第一行=sa_1。
    fireEvent.click(screen.getAllByTestId("subagent-catalog-aside")[0]);
    expect(onOpenAside).toHaveBeenCalledWith(expect.objectContaining({ ref: "sa_1" }));
  });
});
