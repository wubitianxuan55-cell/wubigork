import { describe, expect, it } from "vitest";
import { layoutMindmap, MINDMAP_MAX_NODES, MM_NODE_H, parseMindmapOutline } from "./mindmap";

describe("parseMindmapOutline（M1 大纲解析口径）", () => {
  it("首个 H1 命名根，H2/H3 按级别嵌套", () => {
    const r = parseMindmapOutline("# 项目\n## 目标\n### 短期\n## 风险\n", "fallback");
    expect(r.root.text).toBe("项目");
    expect(r.root.children.map((c) => c.text)).toEqual(["目标", "风险"]);
    const goal = r.root.children[0]!;
    expect(goal.children[0]!.text).toBe("短期");
    expect(goal.children[0]!.depth).toBe(2);
  });

  it("无 H1 时用 fallbackTitle，列表挂在根下", () => {
    const r = parseMindmapOutline("- A\n  - A1\n- B\n", "我的导图");
    expect(r.root.text).toBe("我的导图");
    expect(r.root.children.map((c) => c.text)).toEqual(["A", "B"]);
    expect(r.root.children[0]!.children[0]!.text).toBe("A1");
  });

  it("标题下的列表嵌在标题节点下（列表优先口径）", () => {
    const r = parseMindmapOutline("# 根\n## 章\n- 要点1\n  - 细节\n- 要点2\n", "f");
    const chapter = r.root.children[0]!;
    expect(chapter.children.map((c) => c.text)).toEqual(["要点1", "要点2"]);
    expect(chapter.children[0]!.children[0]!.text).toBe("细节");
  });

  it("同级列表项去重弹栈，不重复嵌套", () => {
    const r = parseMindmapOutline("- A\n  - A1\n  - A2\n- B\n  - B1\n", "f");
    const a = r.root.children[0]!;
    const b = r.root.children[1]!;
    expect(a.children.map((c) => c.text)).toEqual(["A1", "A2"]);
    expect(b.children.map((c) => c.text)).toEqual(["B1"]);
  });

  it("代码围栏内的 # 与 - 不参与解析", () => {
    const r = parseMindmapOutline("# 根\n```\n# 不是标题\n- 不是列表\n```\n- 真条目\n", "f");
    expect(r.root.children.map((c) => c.text)).toEqual(["真条目"]);
  });

  it("勾选框转符号；无序/点号有序/中文顿号有序均可解析", () => {
    const r = parseMindmapOutline("- [ ] 待办\n- [x] 完成\n1. 第一\n2、第二\n", "f");
    expect(r.root.children.map((c) => c.text)).toEqual(["☐ 待办", "☑ 完成", "第一", "第二"]);
  });

  it("非列表数字文本（3.14）不误判为列表", () => {
    const r = parseMindmapOutline("# 根\n3.14 是圆周率\n", "f");
    expect(r.root.children).toHaveLength(0);
  });

  it("超过节点上限标记 truncated 且不再建节点", () => {
    const lines = ["# 根", ...Array.from({ length: MINDMAP_MAX_NODES + 10 }, (_, i) => `- 项${i}`)];
    const r = parseMindmapOutline(lines.join("\n"), "f");
    expect(r.truncated).toBe(true);
    expect(r.count).toBe(MINDMAP_MAX_NODES);
  });

  it("空文档 → 单节点根，不截断；空列表项跳过", () => {
    expect(parseMindmapOutline("", "空")).toMatchObject({ count: 1, truncated: false });
    const r = parseMindmapOutline("- \n- 有内容\n", "f");
    expect(r.root.children.map((c) => c.text)).toEqual(["有内容"]);
  });
});

describe("layoutMindmap（右向逻辑树布局）", () => {
  // 节点 id：root=n0，A=n1，A1=n2，B=n3
  const tree = parseMindmapOutline("# R\n- A\n  - A1\n- B\n", "f");

  it("可见节点全量、边数 = 节点数-1、几何右向递增", () => {
    const l = layoutMindmap(tree.root, new Set());
    expect(l.nodes).toHaveLength(4);
    expect(l.edges).toHaveLength(3);
    const root = l.nodes.find((n) => n.depth === 0)!;
    for (const n of l.nodes) expect(n.x).toBeGreaterThanOrEqual(root.x);
    const a = l.nodes.find((n) => n.text === "A")!;
    const b = l.nodes.find((n) => n.text === "B")!;
    expect(b.y).toBeGreaterThan(a.y); // 兄弟行不重叠
    expect(a.h).toBe(MM_NODE_H);
    const a1 = l.nodes.find((n) => n.text === "A1")!;
    expect(a1.x).toBeGreaterThan(a.x);
  });

  it("折叠父节点：子树消失、badge 计直接子节点数、整体高度收缩", () => {
    // A 挂两个子节点：折叠后叶子行 3→2，总高必然收缩
    const wide = parseMindmapOutline("# R\n- A\n  - A1\n  - A2\n- B\n", "f");
    const full = layoutMindmap(wide.root, new Set());
    const c = layoutMindmap(wide.root, new Set(["n1"]));
    expect(c.nodes.map((n) => n.text)).not.toContain("A1");
    expect(c.nodes.map((n) => n.text)).not.toContain("A2");
    const a = c.nodes.find((n) => n.text === "A")!;
    expect(a.collapsed).toBe(true);
    expect(a.hiddenChildren).toBe(2);
    expect(c.height).toBeLessThan(full.height);
    expect(c.edges).toHaveLength(c.nodes.length - 1);
    expect(c.edges.every((d) => d.startsWith("M "))).toBe(true);
  });
});
