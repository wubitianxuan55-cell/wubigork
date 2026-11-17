import type { AgentNetwork, AgentNode } from "../../lib/types";
import { agentHue } from "./fleetVisuals";

export const FLEET_CARD_WIDTH_MIN = 160;
export const FLEET_CARD_WIDTH_MAX = 232;
export const FLEET_CARD_GUTTER = 10;
export const FLEET_CARD_HEIGHT = 92;
export const FLEET_LEVEL_HEIGHT = 136;
export const FLEET_STAGE_PAD_Y = 32;
// D2 折行（dsh Tasks tasks-graph-layout 蒸馏）：折行上限 4 列（上游
// maxBandCols 默认 4）；卡宽钳在 [MIN, MAX] 摊开容器宽——列数按**实际卡宽**
// 可行性判（行宽必须不超容器）：判列槽位若固定用 MIN 槽，窄容器会选出放
// 不下的列数（556px 选 4 列=680 溢出，实弹走查逮住后改为逐列迭代）。
// 上游 0.78 可读下限在本算法里由「卡宽钳不下探 + 画布恒 1:1」隐式满足
// （卡宽下限 160 / 上限 232 ≈ 0.69 最差比值出现在单列窄容器，宁溢一张卡
// 的宽度也不把卡片压到不可读——该极端仅在容器 <2×MIN 槽时出现）。
export const FLEET_BAND_MAX_COLS = 4;
// D2 活动行尾巴的高度估算（与 AgentTopology 卡片 tail 区 CSS 一致）：
// lastText 两行（line-clamp 2）+ lastTool 单行省略。行距按模型里最高卡片
// 实测（上游 rowStrideFor 语义），不再写死单层 136。
export const FLEET_TEXT_TAIL_HEIGHT = 34;
export const FLEET_TOOL_TAIL_HEIGHT = 18;
// D3 折叠聚合卡高度（紧凑：计数 + 组名 + 两条预览）。
export const FLEET_FOLD_HEIGHT = 56;
const FLEET_LEVEL_GAP = FLEET_LEVEL_HEIGHT - FLEET_CARD_HEIGHT;

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

export interface FleetTopologyLayout<T extends FleetTopologyNode = FleetTopologyNode> {
  width: number;
  height: number;
  cardWidth: number;
  nodes: T[];
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

// cardHeightOf 估算单卡高度（活动行尾巴按有无计价；fold 聚合卡紧凑高）：
// 布局与卡片渲染共用同一常量，行距「按当前模型最高卡片实测」才不会压住
// 下一行。
export function cardHeightOf(node: { kind?: AgentNode["kind"]; lastText?: string; lastTool?: string }): number {
  if (node.kind === "fold") return FLEET_FOLD_HEIGHT;
  return (
    FLEET_CARD_HEIGHT +
    (node.lastText ? FLEET_TEXT_TAIL_HEIGHT : 0) +
    (node.lastTool ? FLEET_TOOL_TAIL_HEIGHT : 0)
  );
}

// bandColsFor 折行列数：从最宽列数逐级收窄，取「按钳制后卡宽摊开的行宽
// 不超容器」的最大列数——360px→2 列、556px→3 列、720px→4 列，封顶
// FLEET_BAND_MAX_COLS。
export function bandColsFor(containerWidth: number): number {
  if (!Number.isFinite(containerWidth) || containerWidth <= 0) return FLEET_BAND_MAX_COLS;
  for (let cols = FLEET_BAND_MAX_COLS; cols > 1; cols -= 1) {
    const card = Math.max(
      FLEET_CARD_WIDTH_MIN,
      Math.min(FLEET_CARD_WIDTH_MAX, Math.floor(containerWidth / cols) - FLEET_CARD_GUTTER),
    );
    if (cols * (card + FLEET_CARD_GUTTER) <= containerWidth) return cols;
  }
  return 1;
}

// bandCardWidth 折行后的实际卡宽：按所选列数摊开容器宽，夹在 [MIN, MAX]。
// 列数与判列槽位同源，保证行宽永不超容器（宽度不足时 MIN 兜底，宁溢出
// 一张卡的宽度也不把卡片压到不可读）。
export function bandCardWidth(containerWidth: number): number {
  const fit = Math.floor(containerWidth / bandColsFor(containerWidth)) - FLEET_CARD_GUTTER;
  return Math.max(FLEET_CARD_WIDTH_MIN, Math.min(FLEET_CARD_WIDTH_MAX, fit));
}

interface BandBlock {
  id: string;
  // width 是本子树的横向预留（layout slot 的整倍数，≥ slot）；rows 是子带
  // 按 cols 折行后的块矩阵（叶子为 null）。
  width: number;
  headHeight: number;
  totalHeight: number;
  rows: BandBlock[][] | null;
}

// 泛型保全入参扩展字段（FleetNode 的 fold 信息随 nodes 原样带回渲染层）。
export function layoutFleetTopology<T extends FleetTopologyNode>(entries: T[], stageWidth: number): FleetTopologyLayout<T> {
  const entryOf = new Map(entries.map((entry) => [entry.node.id, entry]));
  const kidsOf = new Map<string, FleetTopologyNode[]>();
  let rootEntry: FleetTopologyNode | undefined;
  for (const entry of entries) {
    if (entry.parentId === undefined) {
      rootEntry = entry;
      continue;
    }
    const siblings = kidsOf.get(entry.parentId) ?? [];
    siblings.push(entry);
    kidsOf.set(entry.parentId, siblings);
  }
  if (rootEntry === undefined) {
    return { width: 0, height: 0, cardWidth: FLEET_CARD_WIDTH_MIN, nodes: entries, points: [], links: [] };
  }

  const cardWidth = bandCardWidth(stageWidth);
  // layout slot = 卡宽 + 右侧空隙（gutter 已含在块宽里，行宽不再另加）。
  const slot = cardWidth + FLEET_CARD_GUTTER;
  const cols = bandColsFor(stageWidth);
  const rowWidthOf = (row: BandBlock[]): number => row.reduce((sum, block) => sum + block.width, 0);

  const measure = (entry: FleetTopologyNode): BandBlock => {
    const kids = kidsOf.get(entry.node.id) ?? [];
    const headHeight = cardHeightOf(entry.node);
    if (kids.length === 0) {
      return { id: entry.node.id, width: slot, headHeight, totalHeight: headHeight, rows: null };
    }
    const blocks = kids.map(measure);
    const rows: BandBlock[][] = [];
    for (let i = 0; i < blocks.length; i += cols) rows.push(blocks.slice(i, i + cols));
    // 宽度=最宽行；高度=头卡 + 各行（行内取最高子树）+ 行间空隙。
    let width = slot;
    let rowsTotal = 0;
    for (let r = 0; r < rows.length; r += 1) {
      width = Math.max(width, rowWidthOf(rows[r]));
      const rowMax = rows[r].reduce((max, block) => Math.max(max, block.totalHeight), 0);
      rowsTotal += rowMax + (r < rows.length - 1 ? FLEET_LEVEL_GAP : 0);
    }
    return {
      id: entry.node.id,
      width,
      headHeight,
      totalHeight: headHeight + FLEET_LEVEL_GAP + rowsTotal,
      rows,
    };
  };

  const placements = new Map<string, { x: number; y: number }>();
  let maxBottom = 0;
  const place = (block: BandBlock, xLeft: number, yTop: number): void => {
    placements.set(block.id, { x: xLeft + block.width / 2, y: yTop });
    maxBottom = Math.max(maxBottom, yTop + block.totalHeight);
    if (block.rows === null) return;
    let rowY = yTop + block.headHeight + FLEET_LEVEL_GAP;
    for (const row of block.rows) {
      let x = xLeft + (block.width - rowWidthOf(row)) / 2;
      for (const child of row) {
        place(child, x, rowY);
        // slot（块宽）已含右侧 gutter，步进不再另加——与 rowWidthOf 同口径
        //（步进另加会把行尾卡推出画布，实弹走查逮住：done 聚合卡右缘出界）。
        x += child.width;
      }
      rowY += row.reduce((max, child) => Math.max(max, child.totalHeight), 0) + FLEET_LEVEL_GAP;
    }
  };

  const rootBlock = measure(rootEntry);
  const width = Math.max(Number.isFinite(stageWidth) && stageWidth > 0 ? stageWidth : 720, rootBlock.width);
  const offsetX = Math.max(0, (width - rootBlock.width) / 2);
  place(rootBlock, offsetX, FLEET_STAGE_PAD_Y);

  const points = entries
    .filter((entry) => placements.has(entry.node.id))
    .map((entry): FleetTopologyPoint => {
      const at = placements.get(entry.node.id)!;
      return {
        id: entry.node.id,
        x: at.x,
        y: at.y,
        depth: entry.depth,
        hue: agentHue(
          entry.node.kind === "root" ? "main" : entry.node.id,
          entry.family >= 0 ? entry.family : undefined,
        ),
      };
    });
  const pointAt = new Map(points.map((point) => [point.id, point]));
  const links: FleetTopologyLink[] = [];
  for (const entry of entries) {
    if (entry.parentId === undefined) continue;
    const from = pointAt.get(entry.parentId);
    const to = pointAt.get(entry.node.id);
    if (from === undefined || to === undefined) continue;
    const parentEntry = entryOf.get(entry.parentId);
    links.push({
      to: entry.node.id,
      running: entry.node.status === "running",
      hue: to.hue,
      x1: from.x,
      y1: from.y + cardHeightOf(parentEntry?.node ?? entry.node),
      x2: to.x,
      y2: to.y,
    });
  }

  return {
    width,
    height: maxBottom + 24,
    cardWidth,
    nodes: entries.filter((entry) => placements.has(entry.node.id)),
    points,
    links,
  };
}

export function layoutAgentTopology(network: AgentNetwork, stageWidth: number): FleetTopologyLayout {
  return layoutFleetTopology(flattenAgentTree(network.root), stageWidth);
}

export function topologyLinkPath(link: FleetTopologyLink): string {
  const curve = Math.min(96, Math.max(24, (link.y2 - link.y1) * 0.5));
  return `M ${link.x1} ${link.y1} C ${link.x1} ${link.y1 + curve}, ${link.x2} ${link.y2 - curve}, ${link.x2} ${link.y2}`;
}
