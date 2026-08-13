import { describe, it, expect } from "vitest";
import { slashQueryOf } from "./composer";

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
