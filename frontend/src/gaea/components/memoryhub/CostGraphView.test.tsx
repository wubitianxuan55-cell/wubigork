import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostGraphView } from "./CostGraphView";
import type { CostGraphView as CostGraphViewData } from "../../lib/types";

// 与后端契约同形：app.CostGraph 返回 JSON 串（CostGraphView），前端 JSON.parse。
const graphJson = vi.hoisted(() => ({ raw: "" }));
const costGraphSpy = vi.hoisted(() => vi.fn());

function treeView(): CostGraphViewData {
  return {
    nodes: [
      { id: "cat:综合单价", name: "综合单价", type: "category", desc: "3 条", val: 2230, meta: { path: "综合单价", entries: "3" } },
      { id: "cat:综合单价/土建", name: "土建", type: "category", desc: "2 条", val: 730, meta: { path: "综合单价/土建", entries: "2" } },
      { id: "proj:p1", name: "厂房 A", type: "project", desc: "2 条明细 · 1 版本", val: 970, meta: { projectId: "p1" } },
    ],
    edges: [],
    stats: { truncated: false, nodeCount: 3, edgeCount: 0, countsByType: { category: 2, project: 1 } },
  };
}

function entryView(truncated = false): CostGraphViewData {
  return {
    nodes: [
      { id: "cat:综合单价/土建", name: "土建", type: "category", desc: "综合单价/土建", val: 0, meta: { path: "综合单价/土建" } },
      { id: "entry:c30", name: "C30 商品混凝土", type: "entry", desc: "¥480/m³", val: 480, meta: { name: "c30", path: "综合单价/土建" } },
      { id: "item:p1:11", name: "C30 商品混凝土", type: "item", desc: "10/m³ × ¥500", val: 5000, meta: { projectId: "p1", unit: "m³", quantity: "10", price: "500", entryName: "" } },
      { id: "inq:1", name: "C30 商品混凝土（信息价）", type: "inquiry", desc: "信息价 · ¥470", val: 470, meta: { source: "信息价" } },
    ],
    edges: [
      { source: "cat:综合单价/土建", target: "entry:c30", type: "belongs_to", weight: 1 },
      { source: "item:p1:11", target: "entry:c30", type: "references", weight: 1, meta: { matchedBy: "entry_name" } },
      { source: "inq:1", target: "entry:c30", type: "suggests", weight: 1 },
    ],
    stats: {
      truncated,
      nodeCount: 4,
      edgeCount: 3,
      countsByType: { category: 1, entry: 1, item: 1, inquiry: 1 },
    },
  };
}

vi.mock("../../lib/bridge", () => ({
  app: {
    CostGraph: (...args: unknown[]) => costGraphSpy(...args) as Promise<string>,
    CostCategories: async () => [
      {
        id: 1, parentId: 0, name: "综合单价", sort: 0, count: 0,
        children: [{ id: 2, parentId: 1, name: "土建", sort: 0, count: 0, children: [] }],
      },
    ],
    CostProjectList: async () => [
      { id: "p1", name: "厂房 A", projectType: "房建", scale: "", craft: "", status: "已保存版本", note: "", createdAt: "", updatedAt: "", itemCount: 2, total: 970, versionCount: 1 },
    ],
  },
}));

describe("CostGraphView 成本知识图谱", () => {
  beforeEach(() => {
    costGraphSpy.mockReset();
    costGraphSpy.mockImplementation(async () => graphJson.raw);
  });

  it("默认分类总览：渲染聚合节点/规模统计/图例（tree scope 调用无 focus）", async () => {
    graphJson.raw = JSON.stringify(treeView());
    render(<CostGraphView />);
    expect(await screen.findByText("综合单价")).toBeTruthy();
    expect(screen.getByText("土建")).toBeTruthy();
    expect(screen.getByText("厂房 A")).toBeTruthy();
    expect(screen.getByText(/节点 3 · 边 0/)).toBeTruthy();
    expect(screen.getByText("分类 2")).toBeTruthy(); // 图例含数量
    expect(costGraphSpy).toHaveBeenCalledWith("tree", "", 300);
  });

  it("切换条目展开：未选 focus 给引导；选择分类后以 entry scope 加载并渲染边标签", async () => {
    graphJson.raw = JSON.stringify(treeView());
    render(<CostGraphView />);
    await screen.findByText("综合单价");

    fireEvent.click(screen.getByText("条目展开"));
    expect(screen.getByText("选择分类或项目后展开关联图谱")).toBeTruthy();
    expect(costGraphSpy).not.toHaveBeenCalledWith("entry", expect.anything(), expect.anything());

    graphJson.raw = JSON.stringify(entryView());
    fireEvent.change(screen.getByLabelText("展开范围"), { target: { value: "综合单价/土建" } });
    await waitFor(() => expect(costGraphSpy).toHaveBeenCalledWith("entry", "综合单价/土建", 300));
    // 边标签（≤40 条边时展示）与截断位未触发
    expect(await screen.findByText("归属")).toBeTruthy();
    expect(screen.getByText("引用")).toBeTruthy();
    expect(screen.queryByText("已达节点上限，仅展示部分子图")).toBeNull();
  });

  it("截断提示：stats.truncated=true 时展示部分子图提示", async () => {
    graphJson.raw = JSON.stringify(entryView(true));
    render(<CostGraphView />);
    fireEvent.click(screen.getByText("条目展开"));
    fireEvent.change(screen.getByLabelText("展开范围"), { target: { value: "综合单价/土建" } });
    expect(await screen.findByText("已达节点上限，仅展示部分子图")).toBeTruthy();
  });

  it("点击节点弹 antd Modal 展示 Meta 明细（type 标签 + 结构化键值）", async () => {
    graphJson.raw = JSON.stringify(entryView());
    render(<CostGraphView />);
    fireEvent.click(screen.getByText("条目展开"));
    fireEvent.change(screen.getByLabelText("展开范围"), { target: { value: "综合单价/土建" } });
    // 两个同名节点（条目/明细）都叫「C30 商品混凝土」，点击明细节点看项目明细
    const nodeTexts = await screen.findAllByText("C30 商品混凝土");
    fireEvent.click(nodeTexts[nodeTexts.length - 1]);
    expect(await screen.findByText("项目 ID")).toBeTruthy();
    expect(screen.getByText("数量")).toBeTruthy();
    expect(screen.getByText(/节点 ID item:p1:11/)).toBeTruthy();
  });
});
