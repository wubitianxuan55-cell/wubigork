import type { AgentNetwork, AgentNode } from "../../lib/types";
import { agentHue } from "./fleetVisuals";

export const FLEET_CARD_WIDTH_MIN = 160;
export const FLEET_CARD_WIDTH_MAX = 232;
export const FLEET_CARD_GUTTER = 10;
export const FLEET_CARD_HEIGHT = 92;
export const FLEET_LEVEL_HEIGHT = 136;
export const FLEET_STAGE_PAD_Y = 32;

export interface FleetTopologyPoint {
  id: string;
  x: number;
  y: number;
  depth: number;
  hue: string;
}

export interface FleetTopologyLink {
  to: string;
  running: boolean;
  hue: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

export interface FleetTopologyNode {
  node: AgentNode;
  parentId?: string;
  depth: number;
  family: number;
}

export interface FleetTopologyLayout {
  width: number;
  height: number;
  cardWidth: number;
  nodes: FleetTopologyNode[];
  points: FleetTopologyPoint[];
  links: FleetTopologyLink[];
}

export function flattenAgentTree(root: AgentNode): FleetTopologyNode[] {
  const out: FleetTopologyNode[] = [];
  let family = -1;

  const visit = (node: AgentNode, depth: number, parentId: string | undefined, ownerFamily: number) => {
    out.push({ node, parentId, depth, family: ownerFamily });
    for (const child of node.children ?? []) {
      if (depth === 0) family += 1;
      visit(child, depth + 1, node.id, depth === 0 ? family : ownerFamily);
    }
  };

  visit(root, 0, undefined, -1);
  return out;
}

export function layoutAgentTopology(network: AgentNetwork, stageWidth: number): FleetTopologyLayout {
  const nodes = flattenAgentTree(network.root);
  const childrenOf = new Map<string, FleetTopologyNode[]>();
  for (const entry of nodes) {
    if (entry.parentId === undefined) continue;
    const siblings = childrenOf.get(entry.parentId) ?? [];
    siblings.push(entry);
    childrenOf.set(entry.parentId, siblings);
  }

  const slotOf = new Map<string, number>();
  let leafSlots = 0;
  const place = (entry: FleetTopologyNode): number => {
    const children = childrenOf.get(entry.node.id) ?? [];
    if (children.length === 0) {
      const slot = leafSlots++;
      slotOf.set(entry.node.id, slot);
      return slot;
    }
    let first = 0;
    let last = 0;
    children.forEach((child, index) => {
      const slot = place(child);
      if (index === 0) first = slot;
      last = slot;
    });
    const slot = (first + last) / 2;
    slotOf.set(entry.node.id, slot);
    return slot;
  };
  place(nodes[0]);

  const measuredWidth = Number.isFinite(stageWidth) && stageWidth > 0 ? stageWidth : 720;
  const preferredSlot = leafSlots > 1
    ? Math.min(FLEET_CARD_WIDTH_MAX, Math.max(FLEET_CARD_WIDTH_MIN, measuredWidth / leafSlots))
    : FLEET_CARD_WIDTH_MAX;
  const contentWidth = leafSlots * preferredSlot;
  const width = Math.max(measuredWidth, contentWidth);
  const offsetX = Math.max(0, (width - contentWidth) / 2);
  const cardWidth = preferredSlot - FLEET_CARD_GUTTER;
  const points = nodes.map((entry): FleetTopologyPoint => ({
    id: entry.node.id,
    x: offsetX + ((slotOf.get(entry.node.id) ?? 0) + 0.5) * preferredSlot,
    y: FLEET_STAGE_PAD_Y + entry.depth * FLEET_LEVEL_HEIGHT,
    depth: entry.depth,
    hue: agentHue(
      entry.node.kind === "root" ? "main" : entry.node.id,
      entry.family >= 0 ? entry.family : undefined,
    ),
  }));
  const pointAt = new Map(points.map((point) => [point.id, point]));
  const nodeAt = new Map(nodes.map((entry) => [entry.node.id, entry]));
  const links: FleetTopologyLink[] = [];

  for (const child of nodes) {
    if (child.parentId === undefined) continue;
    const from = pointAt.get(child.parentId);
    const to = pointAt.get(child.node.id);
    if (from === undefined || to === undefined) continue;
    links.push({
      to: child.node.id,
      running: child.node.status === "running",
      hue: pointAt.get(child.node.id)?.hue ?? agentHue(child.node.id),
      x1: from.x,
      y1: from.y + FLEET_CARD_HEIGHT,
      x2: to.x,
      y2: to.y,
    });
  }

  const maxDepth = points.reduce((max, point) => Math.max(max, point.depth), 0);
  return {
    width,
    height: FLEET_STAGE_PAD_Y + maxDepth * FLEET_LEVEL_HEIGHT + FLEET_CARD_HEIGHT + 24,
    cardWidth,
    nodes: nodes.filter((entry) => nodeAt.has(entry.node.id)),
    points,
    links,
  };
}

export function topologyLinkPath(link: FleetTopologyLink): string {
  const curve = Math.min(96, Math.max(24, (link.y2 - link.y1) * 0.5));
  return `M ${link.x1} ${link.y1} C ${link.x1} ${link.y1 + curve}, ${link.x2} ${link.y2 - curve}, ${link.x2} ${link.y2}`;
}
