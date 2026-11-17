// D3 折叠聚合（dsh-better-sidebar Tasks 页 tasks-model 蒸馏，计划 §4·D3）：
// 每个父级两类独立聚合组——done（已结算叶子）与 idle（排队中）——把散叶子
// 收进聚合卡，宽树不再横向溢出。统一模型层：buildFleetModel 产出 pre-order
// 节点表，拓扑渲染（GRAPH）与未来的 TREE 视图共用同一模型与同一折叠态。
import type { AgentNode } from "../../lib/types";
import type { FleetTopologyNode } from "./agentTopologyLayout";

// FOLD_IDLE_MIN（上游常量逐字）：待命成员 ≥3 才扫进聚合；1–2 个是工作集，
// 任务行就显示在它们卡上。
export const FOLD_IDLE_MIN = 3;
// gaea 偏差（留档）：上游 done 组无下限（单卡也折）；gaea 典型会话只有 2–5
// 个子代理，卡片本身承载状态/活动行/统计——done ≥4 才折（720px 4 列一行
// 放不下、聚合才产生真实的空间收益），小会话的已结算卡保持直读。
export const FOLD_DONE_MIN = 4;

export type FleetFoldKind = "done" | "idle";

export interface FleetFoldInfo {
  kind: FleetFoldKind;
  parentId: string;
  count: number;
  errorCount: number;
  // 展开态的聚合卡仍渲染（收起入口）；折叠态成员隐藏。
  expanded: boolean;
  // 两条 label 预览（上游聚合卡同款）。
  previews: string[];
}

export interface FleetNode extends FleetTopologyNode {
  fold?: FleetFoldInfo;
}

// foldGroupId 只由父级+种类派生（上游 §5 同款）：成员变动时稳定，展开状态
// 不丢。
export function foldGroupId(kind: FleetFoldKind, parentId: string): string {
  return `${kind}:${parentId}`;
}

function isSettled(status: AgentNode["status"]): boolean {
  return status === "completed" || status === "error" || status === "failed";
}

// 折叠资格：settled 叶子与完全 settled 的已知分支；running 从不折叠（上游
// §5 折叠候选资格——分支里有运行中/排队后代就不折，聚合卡不吞活卡）。
function hasActiveDescendant(node: AgentNode): boolean {
  for (const child of node.children ?? []) {
    if (child.status === "running" || child.status === "queued") return true;
    if (hasActiveDescendant(child)) return true;
  }
  return false;
}

function labelOf(node: AgentNode): string {
  return node.task || node.name || node.id;
}

// statusLabelOf 是拓扑/树两视图共用的状态词汇（queued/aborted 为 D1/D4 新
// 词汇；failed 是 run 侧别名，工具调用骨架产出 error）。labels 由调用方从
// i18n 注入——模型层不持字典。
export function statusLabelOf(
  status: AgentNode["status"],
  labels: { running: string; queued: string; aborted: string; error: string; done: string },
): string {
  if (status === "running") return labels.running;
  if (status === "queued") return labels.queued;
  if (status === "aborted") return labels.aborted;
  if (status === "error" || status === "failed") return labels.error;
  return labels.done;
}

export interface FleetModelOptions {
  // 点聚合卡展开的组（只展开该组，其它组保持折叠）。
  expandedGroups?: ReadonlySet<string>;
  // 手折（读者最新的显式意图）优先于组展开：手折后组展开不能让 chevron 变死键。
  foldedIds?: ReadonlySet<string>;
}

export function buildFleetModel(root: AgentNode, opts: FleetModelOptions = {}): FleetNode[] {
  const out: FleetNode[] = [];
  let family = -1;

  const visit = (node: AgentNode, depth: number, parentId: string | undefined, ownerFamily: number) => {
    out.push({ node, parentId, depth, family: ownerFamily });
    emitChildren(node, depth, ownerFamily);
  };

  const emitChildren = (parent: AgentNode, depth: number, ownerFamily: number) => {
    const children = parent.children ?? [];
    if (children.length === 0) return;
    const done = children.filter((child) => isSettled(child.status) && !hasActiveDescendant(child));
    const idle = children.filter((child) => child.status === "queued");

    // 组折叠态三分：手折 > 组展开 > 默认规则（done≥FOLD_DONE_MIN / idle≥FOLD_IDLE_MIN）。
    const stateOf = (kind: FleetFoldKind, members: AgentNode[], min: number) => {
      const group = foldGroupId(kind, parent.id);
      const manual = opts.foldedIds?.has(group) ?? false;
      const expanded = !manual && (opts.expandedGroups?.has(group) ?? false);
      return {
        group,
        expanded,
        // 折叠态（成员隐藏）= 手折，或默认规则命中且未被展开。
        collapsed: manual || (members.length >= min && !expanded),
        // 聚合卡可见 = 折叠，或已展开（收起入口）。
        showCard: manual || expanded || members.length >= min,
      };
    };
    const doneState = stateOf("done", done, FOLD_DONE_MIN);
    const idleState = stateOf("idle", idle, FOLD_IDLE_MIN);
    const doneSet = new Set(done);
    const idleSet = new Set(idle);

    const emitItem = (child: AgentNode) => {
      if (depth === 0) family += 1;
      visit(child, depth + 1, parent.id, depth === 0 ? family : ownerFamily);
    };
    for (const child of children) {
      if ((doneSet.has(child) && doneState.collapsed) || (idleSet.has(child) && idleState.collapsed)) {
        continue; // 折叠成员不渲染（子树一并收起）
      }
      emitItem(child);
    }
    const emitFoldNode = (kind: FleetFoldKind, members: AgentNode[], state: { group: string; expanded: boolean; showCard: boolean }) => {
      if (!state.showCard || members.length === 0) return;
      if (depth === 0) family += 1;
      const errorCount = members.filter((m) => m.status === "error" || m.status === "failed").length;
      out.push({
        node: {
          id: state.group,
          name: "",
          kind: "fold",
          status: kind === "done" ? "completed" : "queued",
          toolCalls: 0,
          errors: errorCount,
          tokens: 0,
          children: [],
        },
        parentId: parent.id,
        depth: depth + 1,
        family: depth === 0 ? family : ownerFamily,
        fold: {
          kind,
          parentId: parent.id,
          count: members.length,
          errorCount,
          expanded: state.expanded,
          previews: members.slice(0, 2).map(labelOf),
        },
      });
    };
    emitFoldNode("done", done, doneState);
    emitFoldNode("idle", idle, idleState);
  };

  visit(root, 0, undefined, -1);
  return out;
}
