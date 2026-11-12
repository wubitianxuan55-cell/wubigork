import { describe, expect, it } from "vitest";
import { commitLanes, MAX_LANES } from "./commitGraph";

// G2 只读提交图：lane 拓扑纯函数拓扑钉死（线性/分支/合并/多根/孤儿尾/病理钳）。
describe("commitLanes", () => {
  it("线性历史恒在 lane 0 直线延续", () => {
    const rows = commitLanes([
      { hash: "c", parents: ["b"] },
      { hash: "b", parents: ["a"] },
      { hash: "a", parents: [] },
    ]);
    expect(rows.map((r) => r.lane)).toEqual([0, 0, 0]);
    expect(rows[0].edges).toEqual([{ from: 0, to: 0 }]);
    expect(rows[1].through).toEqual([0]); // a 待连接，竖线穿过 b 行
    expect(rows[2].edges).toEqual([]);
    expect(rows[2].through).toEqual([]);
  });

  it("分支外扩：非首父开新 lane 后折线并入", () => {
    // a → (main) b；a → (feat) c → d
    const rows = commitLanes([
      { hash: "d", parents: ["c"] },
      { hash: "c", parents: ["a"] },
      { hash: "b", parents: ["a"] },
      { hash: "a", parents: [] },
    ]);
    expect(rows[0].lane).toBe(0);
    expect(rows[0].edges).toEqual([{ from: 0, to: 0 }]);
    expect(rows[1].lane).toBe(0);
    expect(rows[1].edges).toEqual([{ from: 0, to: 0 }]);
    // b 未被任何父位等待（a 的槽位被 c→a 边占住）→ 新 lane 外扩，折线并入 0
    expect(rows[2].lane).toBe(1);
    expect(rows[2].edges).toEqual([{ from: 1, to: 0 }]);
    expect(rows[2].through).toEqual([0]); // a 仍在 lane 0 待其行
    expect(rows[3].lane).toBe(0);
  });

  it("合并：两父两 lane，次父外扩后并入", () => {
    // b (feat) 与 c (main) 同父 a，m 合并 c（首父）、b（次父）
    const rows = commitLanes([
      { hash: "m", parents: ["c", "b"] },
      { hash: "c", parents: ["a"] },
      { hash: "b", parents: ["a"] },
      { hash: "a", parents: [] },
    ]);
    // m 在 lane 0；c（首父）继承 lane 0；b（次父）外扩 lane 1
    expect(rows[0].edges).toEqual([
      { from: 0, to: 0 },
      { from: 0, to: 1 },
    ]);
    expect(rows[0].through).toEqual([0, 1]);
    expect(rows[1].lane).toBe(0); // c 在等待它的 lane 0
    expect(rows[2].lane).toBe(1); // b 在等待它的 lane 1
    // b 的首父 a：已在 lanes[0]（c 的父）→ 折线并入
    expect(rows[2].edges).toEqual([{ from: 1, to: 0 }]);
    expect(rows[3].lane).toBe(0);
    expect(rows[3].through).toEqual([]);
  });

  it("多根无连线：槽位复用同 lane 不打架（各自孤立圆点）", () => {
    const rows = commitLanes([
      { hash: "b", parents: [] },
      { hash: "a", parents: [] },
    ]);
    expect(rows[0].lane).toBe(0);
    expect(rows[0].through).toEqual([]); // b 无父无子，无穿线
    expect(rows[1].lane).toBe(0); // 空槽复用
    expect(rows[1].through).toEqual([]);
  });

  it("截断历史的父不进清单：through 画到图底由渲染层吃尾", () => {
    const rows = commitLanes([{ hash: "c", parents: ["missing"] }]);
    expect(rows[0].edges).toEqual([{ from: 0, to: 0 }]);
    expect(rows[0].through).toEqual([0]);
  });

  it("lane 数有界：病理链式多分支钳在 MAX_LANES", () => {
    const commits = Array.from({ length: 40 }, (_, i) => ({
      hash: `h${i}`,
      parents: [`h${i + 1}`, `x${i}`], // 每提交外扩一条孤儿支
    }));
    const rows = commitLanes(commits);
    const maxLane = Math.max(...rows.flatMap((r) => r.edges.map((e) => Math.max(e.from, e.to))));
    expect(maxLane).toBeLessThan(MAX_LANES);
  });
});
