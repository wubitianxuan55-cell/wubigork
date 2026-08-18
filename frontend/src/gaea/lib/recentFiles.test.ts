import { describe, expect, it, beforeEach } from "vitest";
import { clearRecentFilesForTest, loadRecentFiles, recordRecentFile } from "./recentFiles";

describe("recentFiles 最近文件（P0-3 单源）", () => {
  beforeEach(() => {
    clearRecentFilesForTest();
  });

  it("空态返回空数组", () => {
    expect(loadRecentFiles()).toEqual([]);
  });

  it("recordRecentFile 记录并去重置顶", () => {
    recordRecentFile("docs/a.docx");
    recordRecentFile("docs/b.xlsx");
    recordRecentFile("docs/a.docx"); // 重复 → 置顶
    const list = loadRecentFiles();
    expect(list).toHaveLength(2);
    expect(list[0].path).toBe("docs/a.docx");
    expect(list[0].name).toBe("a.docx");
    expect(list[1].path).toBe("docs/b.xlsx");
  });

  it("文件名默认取路径末段", () => {
    recordRecentFile("exports/报告.pdf");
    expect(loadRecentFiles()[0].name).toBe("报告.pdf");
  });

  it("空路径不记录", () => {
    recordRecentFile("");
    expect(loadRecentFiles()).toEqual([]);
  });

  it("limit 20 条", () => {
    for (let i = 0; i < 25; i++) recordRecentFile(`f${i}.md`);
    expect(loadRecentFiles()).toHaveLength(20);
    expect(loadRecentFiles()[0].path).toBe("f24.md");
  });

  it("损坏数据返回空数组", () => {
    localStorage.setItem("gaea.atRecentFiles", "{{{not json");
    expect(loadRecentFiles()).toEqual([]);
  });
});
