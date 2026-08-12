import { describe, it, expect } from "vitest";
import { extractChangedPaths, WRITE_TOOL_NAMES } from "./changes";

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
