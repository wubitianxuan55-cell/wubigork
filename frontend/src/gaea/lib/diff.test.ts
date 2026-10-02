import { describe, expect, it } from "vitest";
import { diffLines, lcsDiff } from "./diff";

// lcsDiff 守恒性检查：ctx/del 按 ai 严格递增重建 a，ctx/add 按 bi 严格递增
// 重建 b——泛型实现的核心不变量（四个调用点都依赖此顺序语义）。
function rebuild<T>(a: T[], b: T[], eq: (x: T, y: T) => boolean): { ra: T[]; rb: T[] } {
  let ai = 0;
  let bi = 0;
  const ra: T[] = [];
  const rb: T[] = [];
  for (const op of lcsDiff(a, b, eq)) {
    if (op.type === "ctx") {
      expect(op.ai).toBe(ai);
      expect(op.bi).toBe(bi);
      ai += 1;
      bi += 1;
      ra.push(a[op.ai]!);
      rb.push(b[op.bi]!);
    } else if (op.type === "del") {
      expect(op.ai).toBe(ai);
      expect(op.bi).toBe(-1);
      ai += 1;
      ra.push(a[op.ai]!);
    } else {
      expect(op.ai).toBe(-1);
      expect(op.bi).toBe(bi);
      bi += 1;
      rb.push(b[op.bi]!);
    }
  }
  expect(ai).toBe(a.length);
  expect(bi).toBe(b.length);
  return { ra, rb };
}

describe("lcsDiff", () => {
  it("全等序列全 ctx（ai/bi 同步推进）", () => {
    const ops = lcsDiff([1, 2, 3], [1, 2, 3], (x, y) => x === y);
    expect(ops).toEqual([
      { type: "ctx", ai: 0, bi: 0 },
      { type: "ctx", ai: 1, bi: 1 },
      { type: "ctx", ai: 2, bi: 2 },
    ]);
  });

  it("空对非空 → 全 add；非空对空 → 全 del（尾部哨兵分支）", () => {
    expect(lcsDiff([], [1, 2], (x, y) => x === y)).toEqual([
      { type: "add", ai: -1, bi: 0 },
      { type: "add", ai: -1, bi: 1 },
    ]);
    expect(lcsDiff([1, 2], [], (x, y) => x === y)).toEqual([
      { type: "del", ai: 0, bi: -1 },
      { type: "del", ai: 1, bi: -1 },
    ]);
  });

  it("中间替换 → del+add 相邻对（同分先耗 a 侧取向）", () => {
    const ops = lcsDiff(["a", "x", "b"], ["a", "y", "b"], (p, q) => p === q);
    expect(ops).toEqual([
      { type: "ctx", ai: 0, bi: 0 },
      { type: "del", ai: 1, bi: -1 },
      { type: "add", ai: -1, bi: 1 },
      { type: "ctx", ai: 2, bi: 2 },
    ]);
  });

  it("多离散改动：重建守恒（ra===a、rb===b，下标单调）", () => {
    const a = ["a", "b", "x", "c", "d", "z"];
    const b = ["a", "y", "c", "w", "d"];
    const { ra, rb } = rebuild(a, b, (p, q) => p === q);
    expect(ra).toEqual(a);
    expect(rb).toEqual(b);
  });

  it("自定义 eq 生效（大小写不敏感比较命中 ctx）", () => {
    const ops = lcsDiff(["A"], ["a"], (p, q) => p.toLowerCase() === q.toLowerCase());
    expect(ops.map((o) => o.type)).toEqual(["ctx"]);
  });

  it("两侧全空 → 空操作流", () => {
    expect(lcsDiff([], [], (x, y) => x === y)).toEqual([]);
  });

  it("diffLines 行级输出与 lcsDiff 同源同取向", () => {
    expect(diffLines("a\nx\nb", "a\ny\nb")).toEqual([
      { type: "ctx", text: "a" },
      { type: "del", text: "x" },
      { type: "add", text: "y" },
      { type: "ctx", text: "b" },
    ]);
    expect(diffLines("a", "a\nb")).toEqual([
      { type: "ctx", text: "a" },
      { type: "add", text: "b" },
    ]);
  });
});
