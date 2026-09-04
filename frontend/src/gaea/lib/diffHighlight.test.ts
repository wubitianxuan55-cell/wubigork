// diffHighlight 单测（v4.93：diff 行内语法着色）。
import { describe, expect, it } from "vitest";
import { highlightLine, parserForPath } from "./diffHighlight";

describe("highlightLine 行内语法着色", () => {
  it("ts 行切出关键字/数字/标识符片段并带 tok-* 类名", () => {
    const segs = highlightLine("const x = 1;", "src/a.ts");
    const all = segs.map((s) => s.text).join("");
    expect(all).toBe("const x = 1;"); // 片段拼接无损
    const kw = segs.find((s) => s.cls.includes("keyword"));
    expect(kw?.text).toBe("const");
    expect(segs.some((s) => s.cls.includes("number"))).toBe(true);
  });

  it("py 注行带注释类（斜体灰）", () => {
    const segs = highlightLine("# comment only", "a.py");
    expect(segs.some((s) => s.cls.includes("comment"))).toBe(true);
  });

  it("未知语言路径整行原样（单片段无着色）", () => {
    const segs = highlightLine("const x = 1;", "main.go");
    expect(segs).toEqual([{ text: "const x = 1;", cls: "" }]);
  });

  it("超长行不切 token（防退化），空行返回空片段", () => {
    const long = "x".repeat(700);
    expect(highlightLine(long, "a.ts")).toEqual([{ text: long, cls: "" }]);
    expect(highlightLine("", "a.ts")).toEqual([{ text: "", cls: "" }]);
  });
});

describe("parserForPath 缓存与映射", () => {
  it("映射正确且重复查询返回同一实例（缓存）", () => {
    const a = parserForPath("x/a.ts");
    expect(a).toBeTruthy();
    expect(parserForPath("x/b.ts")).toBe(a);
    expect(parserForPath("y/c.py")).toBeTruthy();
    expect(parserForPath("z/d.go")).toBeNull();
  });
});
