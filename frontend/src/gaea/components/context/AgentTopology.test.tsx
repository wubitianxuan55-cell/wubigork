import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { AgentNetwork } from "../../lib/types";
import { AgentTopology } from "./AgentTopology";

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const NETWORK: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "main",
    name: "主 Agent",
    kind: "root",
    status: "running",
    task: "整理季度报告",
    toolCalls: 7,
    errors: 0,
    tokens: 240_000,
    children: [
      {
        id: "sa_1",
        name: "调研 Agent",
        kind: "subagent",
        status: "running",
        task: "收集竞品信息",
        toolCalls: 3,
        errors: 0,
        tokens: 80_000,
        // D1 会话骨架：活动行由后端直接落在节点上，卡片渲染文本尾巴。
        lastText: "正在读取竞品页面",
        lastTool: "web_fetch: competitor.io",
        children: [],
      },
      {
        id: "sa_2",
        name: "写作 Agent",
        kind: "subagent",
        status: "completed",
        task: "撰写结论",
        toolCalls: 5,
        errors: 0,
        tokens: 120_000,
        children: [],
      },
    ],
  },
};

describe("AgentTopology 编队卡片拓扑", () => {
  it("主 Agent 与子 Agent 渲染为 SVG 曲线谱系中的卡片节点", () => {
    const { container } = renderT(<AgentTopology network={NETWORK} />);
    expect(screen.getByTestId("agent-topology")).toBeTruthy();
    expect(container.querySelectorAll(".agent-topology-card")).toHaveLength(3);
    expect(container.querySelector(".agent-topology-canvas")).toBeTruthy();
    expect(container.querySelector("svg.agent-topology-links")).toBeTruthy();
    expect(container.querySelector('[data-agent="sa_1"]')?.getAttribute("data-status")).toBe("running");
  });

  it("子 Agent 主命中区可选中，独立按钮可打开对应上下文", () => {
    const onViewContext = vi.fn();
    const onSelectAgent = vi.fn();
    const { container } = renderT(
      <AgentTopology network={NETWORK} onSelectAgent={onSelectAgent} onViewContext={onViewContext} />,
    );
    const card = container.querySelector('[data-agent="sa_1"]') as HTMLElement;
    const select = card.querySelector(".agent-topology-hit") as HTMLButtonElement;
    const open = card.querySelector(".agent-topology-open") as HTMLButtonElement;
    expect(select.tagName).toBe("BUTTON");
    expect(open.tagName).toBe("BUTTON");
    fireEvent.click(select);
    expect(onSelectAgent).toHaveBeenCalledWith("sa_1");
    fireEvent.click(open);
    expect(onViewContext).toHaveBeenCalledWith("sa_1");
  });

  it("会话骨架节点直接渲染活动行尾巴（lastTool 单行 + lastText 两行截断）", () => {
    const { container } = renderT(<AgentTopology network={NETWORK} />);
    const card = container.querySelector('[data-agent="sa_1"]') as HTMLElement;
    expect(card.classList.contains("has-tail")).toBe(true);
    expect(card.querySelector(".agent-topology-tail-tool")?.textContent).toContain("web_fetch");
    expect(card.querySelector(".agent-topology-tail-text")?.textContent).toContain("竞品页面");
    const plain = container.querySelector('[data-agent="sa_2"]') as HTMLElement;
    expect(plain.classList.contains("has-tail")).toBe(false);
  });

  it("D8 双视图：默认图谱；切树视图复用同一模型且选中态保持", () => {
    const onSelectAgent = vi.fn();
    const { container } = renderT(<AgentTopology network={NETWORK} onSelectAgent={onSelectAgent} />);
    // 默认图谱：画布 + 卡片在场，树不在。
    expect(container.querySelector(".agent-topology-canvas")).toBeTruthy();
    expect(container.querySelector('[data-testid="fleet-tree"]')).toBeNull();
    // 切树视图：树行出现（root + sa_1 + sa_2），画布退场。
    const toggles = container.querySelectorAll(".agent-topology-view-toggle button");
    expect(toggles).toHaveLength(2);
    fireEvent.click(toggles[1]);
    const tree = container.querySelector('[data-testid="fleet-tree"]') as HTMLElement;
    expect(tree).toBeTruthy();
    expect(container.querySelector(".agent-topology-canvas")).toBeNull();
    expect(tree.querySelectorAll(".fleet-tree-row")).toHaveLength(3);
    // 树内点选 sa_1 → 归一 ref 回调。
    fireEvent.click(tree.querySelector('[data-agent="sa_1"] .fleet-tree-hit') as HTMLElement);
    expect(onSelectAgent).toHaveBeenLastCalledWith("sa_1");
  });

  it("D8 深度色带：卡片左缘按层着色（root 紫带/子代蓝带）", () => {
    const { container } = renderT(<AgentTopology network={NETWORK} />);
    const rootCard = container.querySelector('[data-agent="main"]') as HTMLElement;
    const subCard = container.querySelector('[data-agent="sa_1"]') as HTMLElement;
    expect(rootCard.style.getPropertyValue("--agent-depth-hue")).toContain("--fleet-violet");
    expect(subCard.style.getPropertyValue("--agent-depth-hue")).not.toContain("--fleet-violet");
    // 家系色保留在 --agent-hue（连线维度不变）。
    expect(subCard.style.getPropertyValue("--agent-hue")).not.toBe("");
  });

  it("ⓘ 详情浮层：节点行可开合，状态/工具/活动行在场", () => {
    const { container } = renderT(<AgentTopology network={NETWORK} />);
    const subCard = container.querySelector('[data-agent="sa_1"]') as HTMLElement;
    expect(container.querySelector('[data-testid="agent-topology-info"]')).toBeNull();
    fireEvent.click(subCard.querySelector(".agent-topology-info-btn") as HTMLElement);
    const info = container.querySelector('[data-testid="agent-topology-info"]') as HTMLElement;
    expect(info.textContent).toContain("进行中");
    expect(info.textContent).toContain("工具调用");
    expect(info.textContent).toContain("web_fetch");
    // 再点即收。
    fireEvent.click(subCard.querySelector(".agent-topology-info-btn") as HTMLElement);
    expect(container.querySelector('[data-testid="agent-topology-info"]')).toBeNull();
  });

  it("D3 折叠：≥4 张已结算卡收进聚合卡，点开只展开该组", () => {
    const foldedNetwork: AgentNetwork = {
      ...NETWORK,
      root: {
        ...NETWORK.root,
        status: "completed",
        children: [1, 2, 3, 4].map((i) => ({
          id: `sa_d${i}`,
          name: `结算 ${i}`,
          kind: "subagent" as const,
          status: "completed" as const,
          task: `任务 ${i}`,
          toolCalls: i,
          errors: 0,
          tokens: 1_000,
        })),
      },
    };
    const { container } = renderT(<AgentTopology network={foldedNetwork} />);
    expect(container.querySelectorAll('[data-agent^="sa_d"]')).toHaveLength(0);
    const foldCard = container.querySelector('[data-fold="done"]') as HTMLElement;
    expect(foldCard.getAttribute("data-count")).toBe("4");
    fireEvent.click(foldCard.querySelector(".agent-topology-hit") as HTMLElement);
    // 展开后成员可见，聚合卡保留作收起入口（aria-expanded）。
    expect(container.querySelectorAll('[data-agent^="sa_d"]')).toHaveLength(4);
    expect(container.querySelector('[data-fold="done"]')?.getAttribute("data-count")).toBe("4");
  });
});
