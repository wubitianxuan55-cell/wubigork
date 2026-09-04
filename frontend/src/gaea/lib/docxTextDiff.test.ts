// docxTextDiff.test.ts — docx 段级结构化对比纯函数测试（A2 收 unsupported 欠账）。
import { describe, expect, it } from "vitest";
import { diffDocxParagraphs, docxDiffStat } from "./docxTextDiff";

describe("diffDocxParagraphs 段级 LCS diff", () => {
  it("完全一致：全 ctx 行，index 取当前文档序号（1 起）", () => {
    const rows = diffDocxParagraphs(["一", "二", "三"], ["一", "二", "三"]);
    expect(rows).toEqual([
      { type: "ctx", index: 1, text: "一" },
      { type: "ctx", index: 2, text: "二" },
      { type: "ctx", index: 3, text: "三" },
    ]);
  });

  it("纯新增：尾部追加两段", () => {
    const rows = diffDocxParagraphs(["一"], ["一", "新1", "新2"]);
    expect(rows).toEqual([
      { type: "ctx", index: 1, text: "一" },
      { type: "add", index: 2, text: "新1" },
      { type: "add", index: 3, text: "新2" },
    ]);
  });

  it("纯删除：中间段落移除，del index 取基线序号", () => {
    const rows = diffDocxParagraphs(["一", "删", "三"], ["一", "三"]);
    expect(rows).toEqual([
      { type: "ctx", index: 1, text: "一" },
      { type: "del", index: 2, text: "删" },
      { type: "ctx", index: 2, text: "三" },
    ]);
  });

  it("段内改写 = 相邻 del+add 对（与行级 diff 同语义，不做字词对齐）", () => {
    const rows = diffDocxParagraphs(["一", "旧内容", "三"], ["一", "新内容", "三"]);
    expect(rows).toEqual([
      { type: "ctx", index: 1, text: "一" },
      { type: "del", index: 2, text: "旧内容" },
      { type: "add", index: 2, text: "新内容" },
      { type: "ctx", index: 3, text: "三" },
    ]);
  });

  it("空文档 ↔ 有内容：整篇 add/del", () => {
    expect(diffDocxParagraphs([], ["一", "二"])).toEqual([
      { type: "add", index: 1, text: "一" },
      { type: "add", index: 2, text: "二" },
    ]);
    expect(diffDocxParagraphs(["一"], [])).toEqual([{ type: "del", index: 1, text: "一" }]);
  });

  it("空段落参与 diff（版面结构不被吞）", () => {
    const base = ["一", "", "三"];
    const cur = ["一", "三"];
    const rows = diffDocxParagraphs(base, cur);
    expect(rows).toContainEqual({ type: "del", index: 2, text: "" });
  });
});

describe("docxDiffStat 计数", () => {
  it("增删计数对齐 diffStatOf 口径", () => {
    const rows = diffDocxParagraphs(["一", "旧", "三"], ["一", "新", "三", "四"]);
    expect(docxDiffStat(rows)).toEqual({ add: 2, del: 1 });
  });
});
