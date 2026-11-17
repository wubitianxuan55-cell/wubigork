import { describe, expect, it } from "vitest";
import type { AgentNetwork, AgentNode } from "../../lib/types";
import {
  FLEET_CARD_HEIGHT,
  FLEET_CARD_WIDTH_MIN,
  bandCardWidth,
  bandColsFor,
  cardHeightOf,
  flattenAgentTree,
  layoutAgentTopology,
  topologyLinkPath,
} from "./agentTopologyLayout";

const NETWORK: AgentNetwork = {
  ok: true,
  window: 1_000_000,
  root: {
    id: "root",
    name: "Lead",
    kind: "root",
    status: "completed",
    toolCalls: 3,
    errors: 0,
    tokens: 120_000,
    children: [
      {
        id: "sa_1",
        name: "Research",
        kind: "subagent",
        status: "completed",
        toolCalls: 2,
        errors: 0,
        tokens: 40_000,
        children: [
          { id: "sa_11", name: "Audit", kind: "subagent", status: "running", toolCalls: 1, errors: 0, tokens: 10_000 },
        ],
      },
      {
        id: "sa_2",
        name: "Build",
        kind: "subagent",
        status: "completed",
        toolCalls: 4,
        errors: 0,
        tokens: 30_000,
      },
    ],
  },
};

function fanoutNetwork(leafCount: number): AgentNetwork {
  const children: AgentNode[] = [];
  for (let i = 0; i < leafCount; i += 1) {
    children.push({
      id: `sa_${i}`,
      name: `Sub ${i}`,
      kind: "subagent",
      status: "completed",
      toolCalls: 1,
      errors: 0,
      tokens: 1_000,
    });
  }
  return {
    ok: true,
    window: 1_000_000,
    root: { id: "root", name: "Lead", kind: "root", status: "completed", toolCalls: 3, errors: 0, tokens: 120_000, children },
  };
}

describe("agentTopologyLayout", () => {
  it("flattens parent ids, depth and family ownership without duplicating nodes", () => {
    const nodes = flattenAgentTree(NETWORK.root);
    expect(nodes.map((entry) => entry.node.id)).toEqual(["root", "sa_1", "sa_11", "sa_2"]);
    expect(nodes.find((entry) => entry.node.id === "sa_11")).toMatchObject({
      parentId: "sa_1",
      depth: 2,
      family: 0,
    });
    expect(nodes.find((entry) => entry.node.id === "sa_2")?.family).toBe(1);
  });

  it("band cols follow the readable floor: 360px→2, 556px→3, 720px→4 cols", () => {
    expect(bandColsFor(360)).toBe(2);
    expect(bandColsFor(556)).toBe(3);
    expect(bandColsFor(720)).toBe(4);
    expect(bandColsFor(0)).toBe(4);
    const card = bandCardWidth(720);
    expect(card).toBeGreaterThanOrEqual(FLEET_CARD_WIDTH_MIN);
    expect(card).toBeLessThanOrEqual(232);
  });

  it("keeps levels aligned and emits one curved link per edge", () => {
    const layout = layoutAgentTopology(NETWORK, 720);
    expect(layout.links).toHaveLength(3);
    expect(layout.points.find((point) => point.id === "root")?.y).toBe(32);
    const sa1 = layout.points.find((point) => point.id === "sa_1");
    const sa2 = layout.points.find((point) => point.id === "sa_2");
    const sa11 = layout.points.find((point) => point.id === "sa_11");
    // 同层兄弟同行；嵌套子代在父下方且不早于父卡底。
    expect(sa1?.y).toBe(sa2?.y);
    expect(sa11!.y).toBeGreaterThan(sa1!.y);
    expect(sa11!.x).toBe(sa1!.x);
    expect(topologyLinkPath(layout.links[0])).toContain(" C ");
  });

  it("folds wide sibling bands instead of expanding canvas width", () => {
    for (const stageWidth of [360, 556, 720]) {
      const layout = layoutAgentTopology(fanoutNetwork(20), stageWidth);
      // 画布不横向溢出（此前 width = max(measured, content) 只会把画布撑宽；
      // 556px 4 列溢出走查案例并入 360/720 同一断言面）。
      expect(layout.width).toBeLessThanOrEqual(stageWidth);
      // 全部节点落在画布内。
      expect(
        layout.points.every((point) => point.x > 0 && point.x < layout.width),
      ).toBe(true);
      // 超出列数的兄弟折到下一行（行距按最高卡实测，不重叠）。
      const rows = new Set(layout.points.filter((p) => p.id !== "root").map((p) => p.y));
      expect(rows.size).toBeGreaterThan(1);
    }
  });

  it("scales row stride with the tallest card (text tail included)", () => {
    expect(cardHeightOf({})).toBe(FLEET_CARD_HEIGHT);
    expect(cardHeightOf({ lastText: "进行中" })).toBeGreaterThan(FLEET_CARD_HEIGHT);
    expect(cardHeightOf({ lastTool: "read_file: a.go", lastText: "进行中" })).toBeGreaterThan(
      cardHeightOf({ lastText: "进行中" }),
    );
    const plain = layoutAgentTopology(fanoutNetwork(6), 720);
    const tailed: AgentNetwork = fanoutNetwork(6);
    tailed.root.children = (tailed.root.children ?? []).map((child) => ({
      ...child,
      lastText: "正在读取文件",
      lastTool: "read_file: 报告.md",
    }));
    const withTails = layoutAgentTopology(tailed, 720);
    expect(withTails.height).toBeGreaterThan(plain.height);
  });
});
