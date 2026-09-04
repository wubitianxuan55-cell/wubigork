import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { AgentRadial } from "./AgentRadial";
import type { AgentNetwork, AgentNode } from "../../lib/types";
import { LocaleProvider } from "../../lib/i18n";
import type { ReactElement } from "react";

const renderP = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

// AgentRadial 测试（jsdom）：SVG 渲染走 container.querySelector（TrendChart 同款
// 兼容写法），占比弧断言 stroke-dasharray 数值，点击断言回调两态（ref 可得/不可得）。
// 组件不走 useT（文案硬编码中文），无需 LocaleProvider。

function child(id: string, over: Partial<AgentNode> = {}): AgentNode {
  return {
    id,
    name: "task",
    kind: "subagent",
    status: "completed",
    toolCalls: 2,
    errors: 0,
    tokens: 1000,
    ...over,
  };
}

const NETWORK: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "root",
    name: "主 agent",
    kind: "root",
    status: "running",
    toolCalls: 3,
    errors: 0,
    tokens: 420_000,
    children: [
      child("sa_2_b2b2b2b2", {
        status: "running",
        task: "调研表格 Agent",
        model: "deepseek-v4-flash",
        toolCalls: 4,
        tokens: 180_000,
      }),
      child("taskB", { status: "error", task: "并行生成测试", errors: 1, toolCalls: 2, tokens: 65_000 }),
    ],
  },
};

// 与组件常量同源（组件内不导出）：中心环 r=34、子环 r=22。
const CIRC_CENTER = 2 * Math.PI * 34;
const CIRC_CHILD = 2 * Math.PI * 22;

function firstDash(el: Element | null): number {
  const raw = el?.getAttribute("stroke-dasharray") ?? "";
  return Number(raw.split(" ")[0]);
}

describe("AgentRadial Agent 网络径向树", () => {
  it("SVG 渲染：中心节点 + 子节点数量 + 标题统计（N Agent / M 运行中 / tokens 合计）", () => {
    const { container } = renderP(<AgentRadial network={NETWORK} running />);
    expect(container.querySelector('[data-testid="agent-radial-center-progress"]')).toBeTruthy();
    expect(container.querySelectorAll('[data-testid="agent-radial-node"]')).toHaveLength(2);
    // 主 agent + 2 子代理 = 3；运行中 = root + sa_2 = 2；tokens = 420k + 180k + 65k
    expect(container.textContent).toContain("3 个 Agent");
    expect(container.textContent).toContain("2 个运行中");
    expect(container.textContent).toContain("≈665k");
    // 连线数 = 子节点数，线色按兄弟序号（category 色板）
    // v4.71：卡头新增图标章（antd 内联 SVG path），连线断言收窄到网络图画布
    const netSvg = container.querySelector('[data-testid="agent-radial-svg"]');
    if (!netSvg) throw new Error("agent radial svg not found");
    expect(netSvg.querySelectorAll("path")).toHaveLength(2);
  });

  it("占比弧计算：中心 = root.tokens/窗口，子环 = 子 tokens/root tokens（stroke-dasharray 断言）", () => {
    const { container } = renderP(<AgentRadial network={NETWORK} running={false} />);
    // 中心：420000 / 1_000_000 = 42% → dash = 2π·34·0.42
    const centerDash = firstDash(container.querySelector('[data-testid="agent-radial-center-progress"]'));
    expect(centerDash).toBeCloseTo(CIRC_CENTER * 0.42, 0);
    // 子环①：180000 / 420000 ≈ 42.857% → dash = 2π·22·(180000/420000)
    const rings = container.querySelectorAll('[data-testid="agent-radial-node-ring"]');
    expect(firstDash(rings[0])).toBeCloseTo(CIRC_CHILD * (180_000 / 420_000), 0);
    expect(firstDash(rings[1])).toBeCloseTo(CIRC_CHILD * (65_000 / 420_000), 0);
  });

  it("点击子节点（ref 可得）→ onOpenSubagent 携带会话/ref/任务/模型/状态；点击中心不触发", () => {
    const onOpen = vi.fn();
    const { container } = renderP(
      <AgentRadial network={NETWORK} running={false} sessionPath="s1.jsonl" onOpenSubagent={onOpen} />,
    );
    fireEvent.click(container.querySelector('[data-node-id="sa_2_b2b2b2b2"]')!);
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(onOpen).toHaveBeenCalledWith({
      sessionPath: "s1.jsonl",
      ref: "sa_2_b2b2b2b2",
      task: "调研表格 Agent",
      model: "deepseek-v4-flash",
      status: "running",
    });
    // 中心节点点击无动作
    fireEvent.click(container.querySelector('[data-testid="agent-radial-center"]')!);
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("点击子节点（ref 不可得）→ 不触发回调，且节点 title 说明", () => {
    const onOpen = vi.fn();
    const { container } = renderP(
      <AgentRadial network={NETWORK} running={false} sessionPath="s1.jsonl" onOpenSubagent={onOpen} />,
    );
    const g = container.querySelector('[data-node-id="taskB"]')!;
    fireEvent.click(g);
    expect(onOpen).not.toHaveBeenCalled();
    expect(g.querySelector("title")?.textContent).toContain("ref 不可得，无法打开子代理对话");
  });

  it("error 子代理状态映射为 failed（回调用口径）", () => {
    const onOpen = vi.fn();
    // error 节点换上 sa_ 前缀以验证状态映射本身（ref 可得路径）
    const net: AgentNetwork = {
      ...NETWORK,
      root: { ...NETWORK.root, children: [child("sa_err_1", { status: "error", task: "失败任务" })] },
    };
    const { container } = renderP(
      <AgentRadial network={net} running={false} sessionPath="s1.jsonl" onOpenSubagent={onOpen} />,
    );
    fireEvent.click(container.querySelector('[data-node-id="sa_err_1"]')!);
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ ref: "sa_err_1", status: "failed" }));
  });

  it("空态：无子节点 → 仅中心节点 + 「暂无子代理」提示", () => {
    const empty: AgentNetwork = {
      ok: true,
      window: 0,
      root: { id: "root", name: "主 agent", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0 },
    };
    const { container } = renderP(<AgentRadial network={empty} running={false} />);
    expect(container.querySelectorAll('[data-testid="agent-radial-node"]')).toHaveLength(0);
    expect(screen.getByText("暂无子代理")).toBeTruthy();
    expect(container.textContent).toContain("1 个 Agent");
    // 窗口缺失 → 中心进度弧为 0（dash = 0）
    expect(firstDash(container.querySelector('[data-testid="agent-radial-center-progress"]'))).toBe(0);
  });

  it("running 节点：内盘带 animate-pulse 呼吸类，静止节点没有", () => {
    const { container } = renderP(<AgentRadial network={NETWORK} running={false} />);
    const runningNode = container.querySelector('[data-node-id="sa_2_b2b2b2b2"]')!;
    expect(runningNode.querySelector(".animate-pulse")).toBeTruthy();
    const doneNode = container.querySelector('[data-node-id="taskB"]')!;
    expect(doneNode.querySelector(".animate-pulse")).toBeNull();
  });

  it("布局规则：>6 个子代理 → 两圈嵌套（内圈 ceil(n/2) 个 @r84，外圈其余 @r148）", () => {
    const many: AgentNetwork = {
      ok: true,
      window: 1_000_000,
      root: {
        ...NETWORK.root,
        children: Array.from({ length: 7 }, (_, i) => child(`sa_many_${i}`, { tokens: 10_000 })),
      },
    };
    const { container } = renderP(<AgentRadial network={many} running={false} />);
    const rings = container.querySelectorAll('[data-testid="agent-radial-node-ring"]');
    expect(rings).toHaveLength(7);
    const radii = Array.from(rings).map((r) =>
      Math.hypot(Number(r.getAttribute("cx")) - 450, Number(r.getAttribute("cy")) - 175),
    );
    expect(radii.filter((r) => Math.abs(r - 84) < 1)).toHaveLength(4); // ceil(7/2)
    expect(radii.filter((r) => Math.abs(r - 148) < 1)).toHaveLength(3);
  });

  it("布局规则：1–6 个子代理 → 单圈下半圆（同一半径 r150）", () => {
    const { container } = renderP(<AgentRadial network={NETWORK} running={false} />);
    const rings = container.querySelectorAll('[data-testid="agent-radial-node-ring"]');
    const radii = Array.from(rings).map((r) =>
      Math.hypot(Number(r.getAttribute("cx")) - 450, Number(r.getAttribute("cy")) - 175),
    );
    expect(radii).toHaveLength(2);
    for (const r of radii) expect(Math.abs(r - 150)).toBeLessThan(1);
    // 下半圆：所有节点都在中心下方
    const ys = Array.from(container.querySelectorAll('[data-testid="agent-radial-node"] circle')).map((c) =>
      Number(c.getAttribute("cy")),
    );
    expect(Math.min(...ys)).toBeGreaterThan(175);
  });
});

describe("AgentRadial 2.5e 后半：查看上下文入口", () => {
  const saNode = {
    ok: true,
    window: 100,
    root: {
      id: "root",
      name: "主 agent",
      kind: "root" as const,
      status: "completed" as const,
      toolCalls: 0,
      errors: 0,
      tokens: 0,
      children: [
        { id: "sa_2_b2b2b2b2", name: "task", kind: "subagent" as const, status: "completed" as const, toolCalls: 3, errors: 0, tokens: 1200 },
      ],
    },
  };

  it("sa_ 节点 + onViewContext 在场 → 渲染「查看上下文」并回调 ref", async () => {
    const onViewContext = vi.fn();
    renderP(<AgentRadial network={saNode} running={false} sessionPath="s1.jsonl" onViewContext={onViewContext} />);
    const btn = await screen.findByTestId("agent-view-context-sa_2_b2b2b2b2");
    fireEvent.click(btn);
    expect(onViewContext).toHaveBeenCalledWith("sa_2_b2b2b2b2");
  });

  it("无回调时不渲染入口", async () => {
    renderP(<AgentRadial network={saNode} running={false} sessionPath="s1.jsonl" />);
    await screen.findByTestId("agent-radial-node");
    expect(screen.queryByTestId("agent-view-context-sa_2_b2b2b2b2")).toBeNull();
  });
});
