import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../lib/i18n";
import type { FamilyAgentRowView } from "../../lib/types/context";
import type { FleetNode } from "./fleetModel";
import { FleetTree } from "./FleetTree";

const renderT = (ui: React.ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

function agentNode(id: string, status: "running" | "queued" | "completed", children?: never[]): FleetNode["node"] {
  return { id, name: id, kind: "subagent", status, toolCalls: 2, errors: 0, tokens: 1_000, children };
}

const NODES: FleetNode[] = [
  { node: { id: "root", name: "主 Agent", kind: "root", status: "completed", toolCalls: 1, errors: 0, tokens: 900, children: [] }, depth: 0, family: -1 },
  { node: agentNode("sa_1", "running"), parentId: "root", depth: 1, family: 0 },
  { node: agentNode("sa_2", "completed"), parentId: "root", depth: 1, family: 0 },
  {
    node: { id: "done:root", name: "", kind: "fold", status: "completed", toolCalls: 0, errors: 1, tokens: 0, children: [] },
    parentId: "root",
    depth: 1,
    family: 0,
    fold: { kind: "done", parentId: "root", count: 4, errorCount: 1, expanded: false, previews: ["任务 A", "任务 B"] },
  },
];

const ROWS: Map<string, FamilyAgentRowView> = new Map();

describe("FleetTree D8 缩进树视图", () => {
  it("按 pre-order 渲染行：root/成员/聚合卡同序，缩进随深度", () => {
    const { container } = renderT(<FleetTree nodes={NODES} rowByRef={ROWS} windowSize={100_000} onToggleFold={() => {}} />);
    const rows = container.querySelectorAll(".fleet-tree-row");
    expect(rows).toHaveLength(4);
    expect(rows[0].getAttribute("data-agent")).toBe("root");
    expect(rows[1].getAttribute("data-agent")).toBe("sa_1");
    expect(rows[3].getAttribute("data-fold")).toBe("done");
    // 缩进随深度递增（root 0 / 成员 1 × 18px）。
    expect((rows[0] as HTMLElement).style.paddingLeft).toBe("0px");
    expect((rows[1] as HTMLElement).style.paddingLeft).toBe("18px");
  });

  it("成员行点击选择（root 归一 main），↗ 打开上下文", () => {
    const onSelectAgent = vi.fn();
    const onViewContext = vi.fn();
    const { container } = renderT(
      <FleetTree nodes={NODES} selectedAgent={null} onSelectAgent={onSelectAgent} onViewContext={onViewContext} rowByRef={ROWS} windowSize={100_000} onToggleFold={() => {}} />,
    );
    const rows = container.querySelectorAll(".fleet-tree-row");
    fireEvent.click((rows[0].querySelector(".fleet-tree-hit") as HTMLElement));
    expect(onSelectAgent).toHaveBeenLastCalledWith("main");
    fireEvent.click((rows[1].querySelector(".fleet-tree-hit") as HTMLElement));
    expect(onSelectAgent).toHaveBeenLastCalledWith("sa_1");
    fireEvent.click((rows[1].querySelector(".fleet-tree-open") as HTMLElement));
    expect(onViewContext).toHaveBeenCalledWith("sa_1");
  });

  it("选中态落行（selectedAgent=main 高亮 root；sa_1 高亮对应行）", () => {
    const { container } = renderT(
      <FleetTree nodes={NODES} selectedAgent="main" rowByRef={ROWS} windowSize={100_000} onToggleFold={() => {}} />,
    );
    expect(container.querySelector('[data-agent="root"]')?.classList.contains("is-selected")).toBe(true);
    const { container: c2 } = renderT(
      <FleetTree nodes={NODES} selectedAgent="sa_1" rowByRef={ROWS} windowSize={100_000} onToggleFold={() => {}} />,
    );
    expect(c2.querySelector('[data-agent="sa_1"]')?.classList.contains("is-selected")).toBe(true);
  });

  it("聚合卡行点击回传组 id 与展开态（收起入口语义）", () => {
    const onToggleFold = vi.fn();
    const { container } = renderT(<FleetTree nodes={NODES} rowByRef={ROWS} windowSize={100_000} onToggleFold={onToggleFold} />);
    fireEvent.click(container.querySelector('[data-fold="done"] .fleet-tree-hit') as HTMLElement);
    expect(onToggleFold).toHaveBeenCalledWith("done:root", false);
  });
});
