import { describe, expect, it } from "vitest";
import { boundedOutput, diffStatFor, TOOL_OUTPUT_MAX_PREVIEW_LINES } from "./tools";

describe("diffStatFor 编辑类工具 diffstat（v4.27 Codex 式 +N−M 芯片）", () => {
  it("edit_file：按 old/new 行差异输出 +N/−N", () => {
    expect(
      diffStatFor("edit_file", JSON.stringify({ path: "a.md", old_string: "a\nb", new_string: "a\nb\nc" })),
    ).toEqual({ add: 1, del: 0 });
    expect(
      diffStatFor("edit_file", JSON.stringify({ path: "a.md", old_string: "a\nb\nc", new_string: "a\nc" })),
    ).toEqual({ add: 0, del: 1 });
  });

  it("multi_edit：逐组累加行级增减", () => {
    expect(
      diffStatFor("multi_edit", JSON.stringify({
        path: "a.md",
        edits: [
          { old_string: "x", new_string: "x\ny" },
          { old_string: "a\nb", new_string: "a" },
        ],
      })),
    ).toEqual({ add: 1, del: 1 });
  });

  it("非编辑工具 / 缺参返回 null", () => {
    expect(diffStatFor("read_file", JSON.stringify({ path: "a.md" }))).toBeNull();
    expect(diffStatFor("write_file", JSON.stringify({ path: "a.md", content: "x" }))).toBeNull();
    expect(diffStatFor("edit_file", JSON.stringify({ path: "a.md" }))).toBeNull();
  });
});

describe("boundedOutput 大工具输出有界预览（P2-2）", () => {
  it("空输出原样返回且不折叠", () => {
    const r = boundedOutput("");
    expect(r.full).toBe("");
    expect(r.collapsed).toBe(false);
  });

  it("undefined 视为空", () => {
    const r = boundedOutput(undefined);
    expect(r.full).toBe("");
    expect(r.collapsed).toBe(false);
  });

  it("行数未超阈值不折叠", () => {
    const out = Array.from({ length: 10 }, (_, i) => `line${i}`).join("\n");
    const r = boundedOutput(out, 60);
    expect(r.collapsed).toBe(false);
    expect(r.preview).toBe(out);
    expect(r.totalLines).toBe(10);
  });

  it("超长输出折叠为头部 + 折叠提示行", () => {
    const out = Array.from({ length: 100 }, (_, i) => `line${i}`).join("\n");
    const r = boundedOutput(out, 60);
    expect(r.collapsed).toBe(true);
    expect(r.totalLines).toBe(100);
    expect(r.hiddenLines).toBe(40);
    expect(r.preview).toContain("line0");
    expect(r.preview).toContain("line59");
    expect(r.preview).not.toContain("line60");
    expect(r.preview).toContain("… 已折叠 40 行");
    expect(r.full).toBe(out);
  });

  it("默认阈值 60 行生效", () => {
    expect(TOOL_OUTPUT_MAX_PREVIEW_LINES).toBe(60);
    const out = Array.from({ length: 61 }, (_, i) => `l${i}`).join("\n");
    expect(boundedOutput(out).collapsed).toBe(true);
  });
});
