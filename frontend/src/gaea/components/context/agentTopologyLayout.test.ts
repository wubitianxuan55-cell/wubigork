import { describe, expect, it } from "vitest";
import type { AgentNetwork } from "../../lib/types";
import {
  FLEET_CARD_HEIGHT,
  FLEET_CARD_WIDTH_MIN,
  FLEET_LEVEL_HEIGHT,
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

  it("uses leaf slots, keeps levels aligned and emits one curved link per edge", () => {
    const layout = layoutAgentTopology(NETWORK, 720);
    expect(layout.cardWidth).toBeGreaterThanOrEqual(FLEET_CARD_WIDTH_MIN - 10);
    expect(layout.points.find((point) => point.id === "root")?.y).toBe(32);
    expect(layout.points.find((point) => point.id === "sa_1")?.y).toBe(32 + FLEET_LEVEL_HEIGHT);
    expect(layout.points.find((point) => point.id === "sa_11")?.y).toBe(32 + FLEET_LEVEL_HEIGHT * 2);
    expect(layout.links).toHaveLength(3);
    expect(layout.height).toBeGreaterThan(32 + FLEET_LEVEL_HEIGHT * 2 + FLEET_CARD_HEIGHT);
    expect(topologyLinkPath(layout.links[0])).toContain(" C ");
  });

  it("expands canvas width instead of squashing dense leaf rows", () => {
    const wide = layoutAgentTopology(NETWORK, 220);
    expect(wide.width).toBeGreaterThan(220);
    expect(wide.points.every((point) => point.x > 0 && point.x < wide.width)).toBe(true);
  });
});
