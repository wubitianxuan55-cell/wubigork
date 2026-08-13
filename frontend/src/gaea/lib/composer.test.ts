import { describe, it, expect } from "vitest";
import { slashQueryOf, atMentionOf } from "./composer";

describe("slashQueryOf", () => {
  it("提取斜杠命令查询", () => {
    expect(slashQueryOf("/doc")).toBe("doc");
    expect(slashQueryOf("/Doc")).toBe("doc");
  });

  it("非斜杠开头或含空格返回 null", () => {
    expect(slashQueryOf("你好")).toBeNull();
    expect(slashQueryOf("/help me")).toBeNull();
  });
});

describe("atMentionOf", () => {
  it("解析末尾 @ 引用为目录与片段", () => {
    expect(atMentionOf("参考 @docs/report.md")).toEqual({ raw: "docs/report.md", dir: "docs/", frag: "report.md" });
    expect(atMentionOf("@src/a.ts")).toEqual({ raw: "src/a.ts", dir: "src/", frag: "a.ts" });
  });

  it("无斜杠时目录为空", () => {
    expect(atMentionOf("@report")).toEqual({ raw: "report", dir: "", frag: "report" });
  });

  it("非 @ 结尾返回 null", () => {
    expect(atMentionOf("没有引用")).toBeNull();
  });
});
