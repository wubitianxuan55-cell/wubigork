import { describe, it, expect } from "vitest";
import { filterProjectGroups } from "./projectGroups";
import type { ProjectGroup } from "./types";

function group(path: string, sessions: ProjectGroup["sessions"]): ProjectGroup {
  return { path, name: path.split("/").pop() || path, current: false, sessions, archived: [], modTime: 1 };
}

describe("filterProjectGroups", () => {
  const groups = [
    group("/ws/a", [
      { path: "/ws/a/s1.jsonl", preview: "季度报告", turns: 1, modTime: 1, current: false },
      { path: "/ws/a/s2.jsonl", preview: "市场方案", title: "方案草稿", turns: 1, modTime: 1, current: false },
    ]),
    group("/ws/b", [
      { path: "/ws/b/s3.jsonl", preview: "会议纪要", turns: 1, modTime: 1, current: false },
    ]),
  ];

  it("空查询返回原分组", () => {
    expect(filterProjectGroups(groups, "  ")).toEqual(groups);
  });

  it("按标题/预览/路径过滤，大小写不敏感", () => {
    const byTitle = filterProjectGroups(groups, "方案草稿");
    expect(byTitle).toHaveLength(1);
    expect(byTitle[0].sessions.map((s) => s.path)).toEqual(["/ws/a/s2.jsonl"]);

    const byPreview = filterProjectGroups(groups, "季度");
    expect(byPreview[0].sessions.map((s) => s.path)).toEqual(["/ws/a/s1.jsonl"]);

    const byPath = filterProjectGroups(groups, "s3");
    expect(byPath).toHaveLength(1);
    expect(byPath[0].path).toBe("/ws/b");
  });

  it("无命中分组被移除", () => {
    expect(filterProjectGroups(groups, "不存在的关键词")).toEqual([]);
  });

  it("也匹配已归档会话", () => {
    const withArchived: ProjectGroup[] = [
      {
        path: "/ws/a", name: "a", current: false, modTime: 1, sessions: [],
        archived: [
          { path: "/ws/a/archive/old.jsonl", preview: "旧版方案", turns: 1, modTime: 1, current: false, archived: true },
        ],
      },
    ];
    const result = filterProjectGroups(withArchived, "旧版");
    expect(result).toHaveLength(1);
    expect(result[0].archived.map((s) => s.path)).toEqual(["/ws/a/archive/old.jsonl"]);
    expect(result[0].sessions).toHaveLength(0);
  });
});
