import { describe, it, expect } from "vitest";
import { autoTopicTitle, filterChatTopics, sortByUpdatedAtDesc } from "./chatTopics";

describe("filterChatTopics", () => {
  const topics = [
    { title: "季度报告", preview: "帮我把三页数据整理成表", modeLabel: "" },
    { title: "倾诉心情", preview: "今天有点累", modeLabel: "角色" },
    { title: "方案草稿", preview: "按 Word 续写", modeLabel: "角色" },
  ];

  it("空查询返回原列表", () => {
    expect(filterChatTopics(topics, "  ")).toEqual(topics);
  });

  it("命中标题（大小写不敏感）", () => {
    expect(filterChatTopics(topics, "季度").map((t) => t.title)).toEqual(["季度报告"]);
  });

  it("命中预览", () => {
    expect(filterChatTopics(topics, "有点累").map((t) => t.title)).toEqual(["倾诉心情"]);
  });

  it("命中模式标签", () => {
    expect(filterChatTopics(topics, "角色").map((t) => t.title)).toEqual(["倾诉心情", "方案草稿"]);
  });

  it("无命中返回空数组", () => {
    expect(filterChatTopics(topics, "不存在的关键词")).toEqual([]);
  });
});

describe("sortByUpdatedAtDesc", () => {
  it("最近活跃排在前，且不修改原数组", () => {
    const a = { id: "a", updated_at: "2026-08-13 10:00:00" };
    const b = { id: "b", updated_at: "2026-08-13 10:03:00" };
    const c = { id: "c", updated_at: "2026-08-13 10:02:00" };
    const input = [a, b, c];
    const key = (t: (typeof input)[number]) => new Date(String(t.updated_at)).getTime();
    const out = sortByUpdatedAtDesc(input, key);
    expect(out.map((t) => t.id)).toEqual(["b", "c", "a"]);
    expect(input.map((t) => t.id)).toEqual(["a", "b", "c"]);
  });
});

describe("autoTopicTitle", () => {
  it("短文本原样返回", () => {
    expect(autoTopicTitle("你好")).toBe("你好");
  });

  it("超过 20 字截断并加省略号", () => {
    const long = "1234567890123456789012345";
    expect(autoTopicTitle(long)).toBe("12345678901234567890…");
  });
});
