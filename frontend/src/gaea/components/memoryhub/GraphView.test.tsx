import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { GraphView } from "./GraphView";
import type { GraphLink, GraphNode, MemoryGraphView } from "../../lib/types";

// 3d-force-graph 在 jsdom 下无法真正初始化 WebGL 画布，这里用一个可链式调用
// 的 stub 替身：所有 setter 返回自身，graphData 记录每次构图调用，
// onNodeClick 捕获回调供测试直接触发（模拟画布上的节点点击）。
interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}
interface GraphStub {
  calls: GraphData[];
  nodeClickCb: ((d: GraphNode) => void) | null;
}
interface ChainableGraph {
  backgroundColor: (v: string) => ChainableGraph;
  nodeVal: (f: (d: GraphNode) => number) => ChainableGraph;
  nodeColor: (f: (d: GraphNode) => string) => ChainableGraph;
  nodeLabel: (f: (d: GraphNode) => string) => ChainableGraph;
  linkColor: (f: (l: GraphLink) => string) => ChainableGraph;
  linkWidth: (v: number) => ChainableGraph;
  d3AlphaDecay: (v: number) => ChainableGraph;
  warmupTicks: (v: number) => ChainableGraph;
  width: (v: number) => ChainableGraph;
  height: (v: number) => ChainableGraph;
  graphData: (data: GraphData) => ChainableGraph;
  onNodeClick: (cb: (d: GraphNode) => void) => ChainableGraph;
}

const { graphStub, forceGraphFactory, memoryGraphMock } = vi.hoisted(() => {
  const graphStub: GraphStub = { calls: [], nodeClickCb: null };
  const chainable = {} as ChainableGraph;
  for (const m of [
    "backgroundColor",
    "nodeVal",
    "nodeColor",
    "nodeLabel",
    "linkColor",
    "linkWidth",
    "d3AlphaDecay",
    "warmupTicks",
    "width",
    "height",
  ] as const) {
    (chainable as unknown as Record<string, unknown>)[m] = vi.fn(() => chainable);
  }
  chainable.graphData = vi.fn((data: GraphData) => {
    graphStub.calls.push(data);
    return chainable;
  });
  chainable.onNodeClick = vi.fn((cb: (d: GraphNode) => void) => {
    graphStub.nodeClickCb = cb;
    return chainable;
  });
  // 组件用法：ForceGraph3D()(containerRef.current) → chainable
  const forceGraphFactory = vi.fn(() => vi.fn(() => chainable));
  return { graphStub, forceGraphFactory, memoryGraphMock: vi.fn() };
});

vi.mock("3d-force-graph", () => ({ default: forceGraphFactory }));
vi.mock("../../lib/bridge", () => ({
  app: { MemoryGraph: memoryGraphMock },
  openExternal: vi.fn(),
}));

// 三种类型节点 + 两种边（同标签 / 引用），覆盖过滤逻辑。
const GRAPH: MemoryGraphView = {
  nodes: [
    { id: "n1", name: "项目策划案", type: "knowledge", desc: "关于项目策划案的知识描述", val: 3 },
    { id: "n2", name: "深夜闲聊", type: "whisper", desc: "与 gaea 深夜聊天的记忆片段", val: 2 },
    { id: "n3", name: "钢材成本", type: "cost", desc: "钢材市场价格条目", val: 1 },
  ],
  links: [
    { source: "n1", target: "n2", type: "reference" },
    { source: "n2", target: "n3", type: "same-tag" },
  ],
};

describe("GraphView 记忆 3D 图谱", () => {
  beforeEach(() => {
    memoryGraphMock.mockReset();
    graphStub.calls = [];
    graphStub.nodeClickCb = null;
  });

  it("加载后渲染工具条标题与节点/边计数，完整数据交给 graphData", async () => {
    memoryGraphMock.mockResolvedValue(GRAPH);
    render(<GraphView />);

    expect(await screen.findByText("记忆 3D 图谱")).toBeTruthy();
    expect(screen.getByText(/3 节点 · 2 边/)).toBeTruthy();

    await waitFor(() => expect(graphStub.calls.length).toBeGreaterThan(0));
    expect(graphStub.calls.at(-1).nodes.map((n) => n.type).sort()).toEqual(["cost", "knowledge", "whisper"]);
    expect(graphStub.calls.at(-1).links).toHaveLength(2);
  });

  it("类型过滤：点击「成本」后 graphData 收到过滤后的 nodes/links，计数更新；再点恢复全量", async () => {
    memoryGraphMock.mockResolvedValue(GRAPH);
    render(<GraphView />);
    await screen.findByText(/3 节点 · 2 边/);

    fireEvent.click(screen.getByRole("button", { name: /成本/ }));

    await waitFor(() => expect(graphStub.calls.at(-1).nodes).toHaveLength(2));
    const filtered = graphStub.calls.at(-1);
    expect(filtered.nodes.map((n) => n.type).sort()).toEqual(["knowledge", "whisper"]);
    // 与被过滤节点相连的边同步剔除（cost 节点相关边消失）
    expect(filtered.links).toHaveLength(1);
    expect(filtered.links[0]).toMatchObject({ source: "n1", target: "n2" });
    expect(await screen.findByText(/2 节点 · 1 边/)).toBeTruthy();

    // 重新点回「成本」→ 全量数据重新构图
    fireEvent.click(screen.getByRole("button", { name: /成本/ }));
    await waitFor(() => expect(graphStub.calls.at(-1).nodes).toHaveLength(3));
    expect(await screen.findByText(/3 节点 · 2 边/)).toBeTruthy();
  });

  it("点击节点（触发 stub 捕获的 onNodeClick）后详情 Modal 展示名称/类型/描述/ID", async () => {
    memoryGraphMock.mockResolvedValue(GRAPH);
    render(<GraphView />);
    await screen.findByText(/3 节点 · 2 边/);
    await waitFor(() => expect(graphStub.nodeClickCb).toBeTruthy());

    // 模拟 3D 画布上的节点点击 → 组件 setSelected(node) → Modal 打开
    graphStub.nodeClickCb(GRAPH.nodes[0]);

    // Modal 标题/正文由多个文本节点拼成，用 dialog 容器 textContent 断言
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("知识");
    expect(dialog.textContent).toContain("项目策划案");
    expect(dialog.textContent).toContain("关于项目策划案的知识描述");
    expect(dialog.textContent).toContain("ID: n1");
  });

  it("空数据（nodes:[]）渲染 0 节点 · 0 边且不崩溃", async () => {
    memoryGraphMock.mockResolvedValue({ nodes: [], links: [] });
    render(<GraphView />);

    expect(await screen.findByText(/0 节点 · 0 边/)).toBeTruthy();
    await waitFor(() => expect(graphStub.calls.at(-1)).toBeTruthy());
    expect(graphStub.calls.at(-1).nodes).toHaveLength(0);
    expect(graphStub.calls.at(-1).links).toHaveLength(0);
    // 工具条仍在
    expect(screen.getByText("记忆 3D 图谱")).toBeTruthy();
  });

  it("variant=\"home\" 隐藏工具条（纯净展示）", async () => {
    memoryGraphMock.mockResolvedValue(GRAPH);
    render(<GraphView variant="home" />);

    await waitFor(() => expect(graphStub.calls.length).toBeGreaterThan(0));
    expect(screen.queryByText("记忆 3D 图谱")).toBeNull();
  });
});
