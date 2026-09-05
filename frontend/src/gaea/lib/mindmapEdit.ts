// mindmapEdit.ts — 导图画布编辑纯函数层（M2 自研回写，docs/
// gaea-office-mindmap-base-design-2026-09.md §3.4；选型=自研，与 M1 自研
// 交互树同构，不引入 mind-elixir——取道不取器/零依赖纪律）。
//
// 不可变树操作：所有 op 返回新树（未触及分支共享），非法操作返回 null。
// 序列化 = 规范大纲：root→H1、depth1→H2、depth≥2→嵌套列表（缩进 2 空格/档）。
// parse(serialize(tree)) 结构稳定（幂等），单测钉死。

import type { MindNode } from "./mindmap";

const MAX_NODE_TEXT = 120;

export function mindMaxId(root: MindNode): number {
  let max = 0;
  const walk = (n: MindNode): void => {
    const m = /^n(\d+)$/.exec(n.id);
    if (m) max = Math.max(max, Number(m[1]));
    for (const c of n.children) walk(c);
  };
  walk(root);
  return max;
}

function cleanText(text: string): string {
  return text.replace(/\r?\n/g, " ").trim().slice(0, MAX_NODE_TEXT);
}

export function findMindNode(root: MindNode, id: string): MindNode | null {
  if (root.id === id) return root;
  for (const c of root.children) {
    const hit = findMindNode(c, id);
    if (hit) return hit;
  }
  return null;
}

export function findMindParent(root: MindNode, id: string): MindNode | null {
  for (const c of root.children) {
    if (c.id === id) return root;
    const hit = findMindParent(c, id);
    if (hit) return hit;
  }
  return null;
}

function isMindDescendant(node: MindNode, id: string): boolean {
  return node.children.some((c) => c.id === id || isMindDescendant(c, id));
}

/** 在 parentId 下追加子节点（文本 trimmed/去换行/截 120）；非法返回 null。 */
export function mindAddChild(root: MindNode, parentId: string, text: string): MindNode | null {
  const t = cleanText(text);
  if (!t) return null;
  const nextId = `n${mindMaxId(root) + 1}`;
  let added = false;
  const walk = (n: MindNode): MindNode => {
    if (added) return n;
    if (n.id === parentId) {
      added = true;
      return { ...n, children: [...n.children, { id: nextId, text: t, depth: n.depth + 1, children: [] }] };
    }
    return { ...n, children: n.children.map(walk) };
  };
  const out = walk(root);
  return added ? out : null;
}

/** 在 nodeId 后追加同层兄弟节点（根不可加兄弟）。 */
export function mindAddSibling(root: MindNode, nodeId: string, text: string): MindNode | null {
  if (nodeId === root.id) return null;
  const t = cleanText(text);
  if (!t) return null;
  const nextId = `n${mindMaxId(root) + 1}`;
  let added = false;
  const walk = (n: MindNode): MindNode => {
    if (added) return n;
    const idx = n.children.findIndex((c) => c.id === nodeId);
    if (idx >= 0) {
      added = true;
      const children = [...n.children];
      children.splice(idx + 1, 0, { id: nextId, text: t, depth: n.children[idx]!.depth, children: [] });
      return { ...n, children };
    }
    return { ...n, children: n.children.map(walk) };
  };
  const out = walk(root);
  return added ? out : null;
}

/** 重命名节点（根亦可）。 */
export function mindRename(root: MindNode, nodeId: string, text: string): MindNode | null {
  const t = cleanText(text);
  if (!t) return null;
  let done = false;
  const walk = (n: MindNode): MindNode => {
    if (done) return n;
    if (n.id === nodeId) {
      done = true;
      return { ...n, text: t };
    }
    return { ...n, children: n.children.map(walk) };
  };
  const out = walk(root);
  return done ? out : null;
}

/** 删除节点及其子树（根不可删）。 */
export function mindRemove(root: MindNode, nodeId: string): MindNode | null {
  if (nodeId === root.id) return null;
  let done = false;
  const walk = (n: MindNode): MindNode => {
    if (done) return n;
    const next = n.children.filter((c) => {
      if (c.id === nodeId) {
        done = true;
        return false;
      }
      return true;
    });
    if (done) return { ...n, children: next };
    return { ...n, children: n.children.map(walk) };
  };
  const out = walk(root);
  return done ? out : null;
}

/** 拖拽挂载：把 dragId 子树移到 targetId 的最后一个孩子。守卫：不能动根、
 *  不能挂到自身、目标不能在被移动子树内（成环）。 */
export function mindMove(root: MindNode, dragId: string, targetId: string): MindNode | null {
  if (dragId === root.id || dragId === targetId) return null;
  const drag = findMindNode(root, dragId);
  const target = findMindNode(root, targetId);
  if (!drag || !target) return null;
  if (isMindDescendant(drag, targetId)) return null; // 目标在拖动子树内
  const without = mindRemove(root, dragId);
  if (!without) return null;
  const reTarget = findMindNode(without, targetId);
  if (!reTarget) return null;
  const moved: MindNode = { ...drag };
  let done = false;
  const walk = (n: MindNode): MindNode => {
    if (done) return n;
    if (n.id === targetId) {
      done = true;
      return { ...n, children: [...n.children, moved] };
    }
    return { ...n, children: n.children.map(walk) };
  };
  const out = walk(without);
  return done ? out : null;
}

/** 规范大纲序列化：root→`# `、depth1→`## `、depth≥2→`- `（缩进 (depth-2)×2 空格）。
 *  parse(serialize(tree)) 结构幂等（深度语义一致），M1 解析器「列表优先」口径下单测钉死。 */
export function serializeMindmapOutline(root: MindNode): string {
  const lines: string[] = [`# ${root.text}`];
  const emit = (n: MindNode, depth: number): void => {
    if (depth === 1) lines.push(`## ${n.text}`);
    else lines.push(`${"  ".repeat(depth - 2)}- ${n.text}`);
    for (const c of n.children) emit(c, depth + 1);
  };
  for (const c of root.children) emit(c, 1);
  return lines.join("\n") + "\n";
}
