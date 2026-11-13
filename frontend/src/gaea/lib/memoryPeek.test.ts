import { describe, expect, it } from "vitest";
import { parseMemoryPeek } from "./memoryPeek";

describe("parseMemoryPeek（提炼记忆弹窗解析）", () => {
  it("remember：title/type/kind/description/body 全取", () => {
    const p = parseMemoryPeek(
      "remember",
      JSON.stringify({
        name: "prefers-tabs",
        title: "Prefers tabs",
        description: "用户偏好制表符缩进",
        type: "user",
        kind: "semantic",
        body: "正文第一行\n正文第二行",
      }),
    );
    expect(p).not.toBeNull();
    expect(p!.title).toBe("Prefers tabs");
    expect(p!.typeLabel).toBe("remember · user · semantic");
    expect(p!.hook).toBe("用户偏好制表符缩进");
    expect(p!.body).toBe("正文第一行\n正文第二行");
  });

  it("remember：title 缺省回退 name", () => {
    const p = parseMemoryPeek("remember", JSON.stringify({ name: "slug-x", body: "b" }));
    expect(p!.title).toBe("slug-x");
  });

  it("fact_add：key/value/category/source", () => {
    const p = parseMemoryPeek(
      "fact_add",
      JSON.stringify({ key: "工期", value: "90 日历天", source: "招标 P3", category: "project" }),
    );
    expect(p!.title).toBe("工期");
    expect(p!.typeLabel).toBe("fact_add · project");
    expect(p!.hook).toBe("招标 P3");
    expect(p!.body).toBe("90 日历天");
    expect(p!.bodyLabel).toBe("value");
  });

  it("坏 JSON/空 args/未知工具 → null（诚实缺省）", () => {
    expect(parseMemoryPeek("remember", "{bad")).toBeNull();
    expect(parseMemoryPeek("remember", undefined)).toBeNull();
    expect(parseMemoryPeek("bash", JSON.stringify({ command: "ls" }))).toBeNull();
    expect(parseMemoryPeek("remember", JSON.stringify({ tags: ["x"] }))).toBeNull();
  });
});
