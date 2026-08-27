import { describe, expect, it } from "vitest";
import { buildSegments } from "./Transcript";
import type { Item } from "../lib/store";

function u(id: string, text: string): Item {
  return { kind: "user", id, text };
}

function a(id: string, text: string, reasoning: string): Item {
  return { kind: "assistant", id, text, reasoning, streaming: false };
}

function tool(id: string, name: string): Item {
  return { kind: "tool", id, name, args: "", readOnly: false, status: "done" };
}

// 2026-08-26（用户决策）：删除大过程卡——已完成轮不再把整轮合并成一张
// 展开的大卡，所有轮次统一交替：正文独立显示、过程（思考/工具）单独成卡。
describe("buildSegments", () => {
  it("完成轮统一交替：过程卡与正文交替，文本不进卡", () => {
    const items: Item[] = [
      u("u1", "帮我写方案"),
      a("a1", "", "先分析需求"),
      tool("t1", "read_file"),
      a("a2", "这是第一段正文", ""),
      tool("t2", "write_file"),
      a("a3", "这是最终正文", "补充收尾"),
    ];
    const segs = buildSegments(items, false);

    // 用户问题独立一段
    expect(segs[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(segs[0].processItems).toHaveLength(0);
    // 思考 a1 + 工具 t1 累积成过程段，正文 a2 同段独立显示（不进卡）
    expect(segs[1].processItems.map((x) => x.id)).toEqual(["a1", "t1"]);
    expect(segs[1].outsideItems.map((x) => x.id)).toEqual(["a2"]);
    // 工具 t2 + 最终思考 a3 累积成过程段，最终正文 a3 同段独立显示
    expect(segs[2].processItems.map((x) => x.id)).toEqual(["t2", "a3"]);
    expect(segs[2].outsideItems.map((x) => x.id)).toEqual(["a3"]);
    const procA3 = segs[2].processItems.find((x) => x.id === "a3");
    const outA3 = segs[2].outsideItems.find((x) => x.id === "a3");
    if (procA3?.kind !== "assistant" || outA3?.kind !== "assistant") throw new Error("bad kind");
    expect(procA3.text).toBe("");
    expect(procA3.reasoning).toContain("补充收尾");
    expect(outA3.reasoning).toBe("");
    expect(outA3.text).toBe("这是最终正文");
  });

  it("多轮对话：running 与否都统一交替，无大过程卡", () => {
    const items: Item[] = [
      u("u1", "第一问"),
      a("a1", "答案一", "思考一"),
      u("u2", "第二问"),
      a("a2", "答案二", "思考二"),
      tool("t1", "read_file"),
      a("a3", "补充", ""),
    ];
    // running=true：两轮都交替（第二轮最后一个段带正文）
    const segs = buildSegments(items, true);
    // 第一轮：用户 → 思考卡 + 正文（同段：思考进卡、正文在外）
    expect(segs[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(segs[1].processItems.map((x) => x.id)).toEqual(["a1"]);
    expect(segs[1].outsideItems.map((x) => x.id)).toEqual(["a1"]);
    // 第二轮：用户 → 思考卡+正文 → 工具卡 → 正文
    expect(segs[2].outsideItems.map((x) => x.id)).toEqual(["u2"]);
    expect(segs[3].processItems.map((x) => x.id)).toEqual(["a2"]);
    expect(segs[3].outsideItems.map((x) => x.id)).toEqual(["a2"]);
    expect(segs[4].processItems.map((x) => x.id)).toEqual(["t1"]);
    expect(segs[4].outsideItems.map((x) => x.id)).toEqual(["a3"]);

    // running=false：结果与 running=true 完全一致（无大过程卡合并）
    const done = buildSegments(items, false);
    expect(done.map((s) => ({ p: s.processItems.map((x) => x.id), o: s.outsideItems.map((x) => x.id) }))).toEqual(
      segs.map((s) => ({ p: s.processItems.map((x) => x.id), o: s.outsideItems.map((x) => x.id) })),
    );
  });

  it("纯文本回复不产生过程卡", () => {
    const items: Item[] = [u("u1", "你好"), a("a1", "你好，有什么可以帮你？", "")];
    const segs = buildSegments(items, false);
    expect(segs).toHaveLength(2);
    expect(segs[0].processItems).toHaveLength(0);
    expect(segs[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(segs[1].processItems).toHaveLength(0);
    expect(segs[1].outsideItems.map((x) => x.id)).toEqual(["a1"]);
  });
});
