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

describe("buildSegments", () => {
  it("完成后折叠成一张大过程卡：中间文本也收进卡内，只留最终正文", () => {
    const items: Item[] = [
      u("u1", "帮我写方案"),
      a("a1", "", "先分析需求"),
      tool("t1", "read_file"),
      a("a2", "这是第一段正文", ""),
      tool("t2", "write_file"),
      a("a3", "这是最终正文", "补充收尾"),
    ];
    const segs = buildSegments(items, false);

    expect(segs).toHaveLength(2);
    // 第一段：用户问题（过程卡渲染在其之后）
    expect(segs[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(segs[0].processItems).toHaveLength(0);
    // 第二段：过程卡（思考 + 工具 + 中间正文）+ 最终正文
    expect(segs[1].outsideItems.map((x) => x.id)).toEqual(["a3"]);
    // 卡内：思考 + 工具 + 中间正文
    const procIds = segs[1].processItems.map((x) => x.id);
    expect(procIds).toEqual(["a1", "t1", "a2", "t2", "a3"]);
    const procA2 = segs[1].processItems.find((x) => x.id === "a2");
    const procA3 = segs[1].processItems.find((x) => x.id === "a3");
    const outA3 = segs[1].outsideItems.find((x) => x.id === "a3");
    if (procA2?.kind !== "assistant" || procA3?.kind !== "assistant" || outA3?.kind !== "assistant") throw new Error("bad kind");
    expect(procA2.reasoning).toBe("");
    expect(procA2.text).toBe("这是第一段正文");
    expect(procA3.text).toBe("");
    expect(procA3.reasoning).toContain("补充收尾");
    expect(outA3.reasoning).toBe("");
    expect(outA3.text).toBe("这是最终正文");
  });

  it("多轮对话：已完成的轮全部折叠，最后仍在流式的轮保持交替", () => {
    const items: Item[] = [
      u("u1", "第一问"),
      a("a1", "答案一", "思考一"),
      u("u2", "第二问"),
      a("a2", "答案二", "思考二"),
      tool("t1", "read_file"),
      a("a3", "补充", ""),
    ];
    // running=true：第一轮已折叠，第二轮（最后一轮）交替出现
    const segs = buildSegments(items, true);
    expect(segs).toHaveLength(5);
    // 第一轮（已折叠）：用户 → 过程卡 + 最终正文
    expect(segs[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(segs[1].processItems.map((x) => x.id)).toEqual(["a1"]);
    expect(segs[1].outsideItems.map((x) => x.id)).toEqual(["a1"]);
    // 第二轮（流式中）：用户 → 过程卡(思考+正文) → 过程卡(工具) → 正文
    expect(segs[2].outsideItems.map((x) => x.id)).toEqual(["u2"]);
    expect(segs[3].processItems.map((x) => x.id)).toEqual(["a2"]);
    expect(segs[3].outsideItems.map((x) => x.id)).toEqual(["a2"]);
    expect(segs[4].processItems.map((x) => x.id)).toEqual(["t1"]);
    expect(segs[4].outsideItems.map((x) => x.id)).toEqual(["a3"]);

    // running=false：两轮都折叠
    const done = buildSegments(items, false);
    expect(done).toHaveLength(4);
    expect(done[0].outsideItems.map((x) => x.id)).toEqual(["u1"]);
    expect(done[1].processItems.map((x) => x.id)).toEqual(["a1"]);
    expect(done[1].outsideItems.map((x) => x.id)).toEqual(["a1"]);
    expect(done[2].outsideItems.map((x) => x.id)).toEqual(["u2"]);
    // a2 同时有思考与中间正文，折叠后各占一条
    expect(done[3].processItems.map((x) => x.id)).toEqual(["a2", "a2", "t1"]);
    expect(done[3].outsideItems.map((x) => x.id)).toEqual(["a3"]);
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
