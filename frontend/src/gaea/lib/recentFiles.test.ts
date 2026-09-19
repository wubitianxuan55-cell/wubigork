import { describe, expect, it, beforeEach, vi } from "vitest";
import { clearRecentFilesForTest, loadRecentFiles, recordRecentFile, subscribeRecentFiles } from "./recentFiles";

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

  // v4.349 实时化：写入广播 + 快照引用稳定（首页/办公文件面改用
  // useSyncExternalStore 订阅；getSnapshot 必须缓存，否则每次读取都算变化）。
  it("写入即广播：订阅者收到一次通知，取消订阅后不再收到", () => {
    const cb = vi.fn();
    const unsub = subscribeRecentFiles(cb);

    recordRecentFile("docs/x.md");
    expect(cb).toHaveBeenCalledTimes(1);

    recordRecentFile("docs/y.md");
    expect(cb).toHaveBeenCalledTimes(2);

    recordRecentFile(""); // 空路径不写不广播（与改动前一致）
    expect(cb).toHaveBeenCalledTimes(2);

    unsub();
    recordRecentFile("docs/z.md");
    expect(cb).toHaveBeenCalledTimes(2);
  });

  it("快照引用稳定：未写入时多次读取返回同一引用，写入后换新引用", () => {
    recordRecentFile("docs/a.md");
    const first = loadRecentFiles();
    expect(loadRecentFiles()).toBe(first);
    recordRecentFile("docs/b.md");
    const second = loadRecentFiles();
    expect(second).not.toBe(first);
    expect(second).toHaveLength(2);
  });
});
