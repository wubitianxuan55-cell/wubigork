import { describe, it, expect } from "vitest";
import {
  extractChangedPaths, extractDeliverablePaths, buildSessionChanges, WRITE_TOOL_NAMES,
  EDIT_TOOL_NAMES, WRITE_ONLY_TOOL_NAMES, buildSessionReads, categoryOf,
} from "./changes";
import type { Item } from "./store";

describe("extractChangedPaths", () => {
  it("提取顶层单路径字段", () => {
    expect(extractChangedPaths('{"path":"a.md"}')).toEqual(["a.md"]);
    expect(extractChangedPaths('{"file_path":"src/b.ts"}')).toEqual(["src/b.ts"]);
    expect(extractChangedPaths('{"notebook_path":"nb.ipynb"}')).toEqual(["nb.ipynb"]);
  });

  it("move_file 使用 source/destination 两个路径", () => {
    expect(extractChangedPaths('{"source":"old.md","destination":"new.md"}')).toEqual(["old.md", "new.md"]);
  });

  it("提取数组字段与 edits 片段", () => {
    expect(extractChangedPaths('{"paths":["a.md","b.md"]}')).toEqual(["a.md", "b.md"]);
    expect(extractChangedPaths('{"edits":[{"path":"a.md"},{"file_path":"b.md"}]}')).toEqual(["a.md", "b.md"]);
  });

  it("非法 JSON 返回空数组", () => {
    expect(extractChangedPaths("{not json")).toEqual([]);
  });

  it("覆盖常见写类工具", () => {
    for (const name of ["write_file", "edit_file", "edit_lines", "multi_edit", "move_file", "notebook_edit", "delete_range", "delete_symbol"]) {
      expect(WRITE_TOOL_NAMES.has(name)).toBe(true);
    }
  });
});

describe("extractDeliverablePaths", () => {
  it("提取目标路径，move_file 不计 source（源不是交付物）", () => {
    expect(extractDeliverablePaths('{"path":"报告.docx"}')).toEqual(["报告.docx"]);
    expect(extractDeliverablePaths('{"source":"old.md","destination":"new.md"}')).toEqual(["new.md"]);
  });

  it("提取数组与 edits 片段并去重保持顺序", () => {
    expect(extractDeliverablePaths('{"paths":["a.md","b.md","a.md"]}')).toEqual(["a.md", "b.md"]);
    expect(extractDeliverablePaths('{"edits":[{"path":"x.docx"},{"file_path":"y.xlsx"}]}')).toEqual(["x.docx", "y.xlsx"]);
  });

  it("非法 JSON 返回空数组", () => {
    expect(extractDeliverablePaths("{not json")).toEqual([]);
  });
});

describe("buildSessionChanges", () => {
  it("按最近改动倒序汇总文件与次数，忽略非写工具", () => {
    const items: Item[] = [
      { kind: "tool", id: "t1", name: "write_file", args: '{"path":"a.md"}', readOnly: false, status: "done" },
      { kind: "tool", id: "t2", name: "edit_file", args: '{"path":"b.md"}', readOnly: false, status: "done" },
      { kind: "tool", id: "t3", name: "read_file", args: '{"path":"c.md"}', readOnly: true, status: "done" },
      { kind: "tool", id: "t4", name: "write_file", args: '{"path":"a.md"}', readOnly: false, status: "done" },
    ];
    expect(buildSessionChanges(items)).toEqual([
      { path: "a.md", count: 2, lastTouched: 3 },
      { path: "b.md", count: 1, lastTouched: 1 },
    ]);
  });
});

describe("changes 2a 三态：buildSessionReads + categoryOf", () => {
  const mk = (id: string, name: string, args: Record<string, unknown>): Item => ({
    kind: "tool",
    id,
    name,
    args: JSON.stringify(args),
    readOnly: true,
    status: "done",
  });

  it("buildSessionReads 聚合读类工具（含 vision 的 image_path），排除写类", () => {
    const items: Item[] = [
      mk("1", "read_file", { path: "a.md" }),
      mk("2", "grep", { path: "src" }),
      mk("3", "vision", { image_path: "图表.png" }),
      mk("4", "write_file", { path: "b.md", content: "x" }),
      mk("5", "read_file", { path: "a.md" }),
    ];
    const reads = buildSessionReads(items);
    const paths = reads.map((r) => r.path);
    expect(paths).toContain("a.md");
    expect(paths).toContain("src");
    expect(paths).toContain("图表.png");
    expect(paths).not.toContain("b.md");
    const a = reads.find((r) => r.path === "a.md")!;
    expect(a.count).toBe(2);
  });

  it("categoryOf 扩展名分桶（含全角标点黏连路径不受影响场景外的正确分桶）", () => {
    expect(categoryOf("报告.docx")).toBe("doc");
    expect(categoryOf("数据.xlsx")).toBe("sheet");
    expect(categoryOf("图.png")).toBe("image");
    expect(categoryOf("main.go")).toBe("code");
    expect(categoryOf("说明.xyz")).toBe("other");
  });

  it("EDIT/WRITE_ONLY 工具集与 WRITE_TOOL_NAMES 无交并完整", () => {
    for (const t of EDIT_TOOL_NAMES) expect(WRITE_TOOL_NAMES.has(t)).toBe(true);
    for (const t of WRITE_ONLY_TOOL_NAMES) expect(WRITE_TOOL_NAMES.has(t)).toBe(true);
    for (const t of WRITE_TOOL_NAMES) {
      expect(EDIT_TOOL_NAMES.has(t) || WRITE_ONLY_TOOL_NAMES.has(t)).toBe(true);
    }
  });
});
