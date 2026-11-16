import { describe, it, expect } from "vitest";
import {
  extractChangedPaths, extractDeliverablePaths, buildSessionChanges, deliverableDedupKey,
  WRITE_TOOL_NAMES, EDIT_TOOL_NAMES, WRITE_ONLY_TOOL_NAMES, buildSessionReads, categoryOf,
} from "./changes";
import type { Item } from "./store";

// 2026-11-17 审查批4 行为锁（自 wenxue 回种 0b1f4c62）：交付路径去重键归一化
// ——同一文件以绝对/相对、正/反斜杠、大小写、./ 前缀等不同写法出现时必须
// 收敛为一个键、一条面板条目、计数累加；展示 path 保留首次原始写法。
describe("deliverableDedupKey（gaea 回种）", () => {
  const cwd = "C:\\新工作\\2630合并沧州";

  it("同一文件的多种写法收敛为一个键", () => {
    const keys = new Set([
      deliverableDedupKey("交互文件/大纲.md", cwd),
      deliverableDedupKey("C:\\新工作\\2630合并沧州\\交互文件\\大纲.md", cwd),
      deliverableDedupKey("C:/新工作/2630合并沧州/交互文件/大纲.md", cwd),
      deliverableDedupKey("./交互文件/大纲.md", cwd),
      deliverableDedupKey(".\\交互文件\\大纲.md", cwd),
      deliverableDedupKey("交互文件\\大纲.md", cwd),
    ]);
    expect(keys.size).toBe(1);
  });

  it("大小写不敏感（Windows 文件系统）", () => {
    expect(deliverableDedupKey("C:/新工作/书/交互文件/X.md", cwd)).toBe(
      deliverableDedupKey("c:/新工作/书/交互文件/x.md", cwd),
    );
  });

  it("工作区外的绝对路径保持独立且仍归一化", () => {
    const a = deliverableDedupKey("D:\\其他\\a.md", "C:/新工作/书");
    const b = deliverableDedupKey("d:/其他/a.md", "C:\\新工作\\书");
    expect(a).toBe(b);
    expect(a).not.toBe(deliverableDedupKey("a.md", "C:/新工作/书"));
  });

  it("无 cwd 时退化为分隔符+大小写归一", () => {
    expect(deliverableDedupKey("交互文件\\X.md")).toBe(deliverableDedupKey("交互文件/x.md"));
    expect(deliverableDedupKey("交互文件/x.md")).not.toBe(
      deliverableDedupKey("C:/新工作/书/交互文件/x.md"),
    );
  });

  it("工作区根本身与空路径", () => {
    expect(deliverableDedupKey("C:\\新工作\\2630合并沧州", cwd)).toBe("");
    expect(deliverableDedupKey("  交互文件/x.md  ", cwd)).toBe(deliverableDedupKey("交互文件/x.md", cwd));
  });
});

describe("buildSessionChanges 归一化合并（gaea 回种）", () => {
  const cwd = "C:\\新工作\\2630合并沧州";
  const tool = (id: string, args: Record<string, unknown>): Item => ({
    kind: "tool", id, name: "write_file", args: JSON.stringify(args), readOnly: false, status: "done",
  });

  it("同一文件多种写法合并且计数累加，展示保留首次写法", () => {
    const items: Item[] = [
      tool("t1", { path: "交互文件/大纲.md", content: "v1" }),
      { kind: "assistant", id: "a1", text: "已写入 C:\\新工作\\2630合并沧州\\交互文件\\大纲.md", reasoning: "", streaming: false },
      tool("t2", { path: "C:/新工作/2630合并沧州/交互文件/大纲.md", content: "v2" }),
      { kind: "user", id: "u1", text: "再改一版" },
      tool("t3", { path: "./交互文件/大纲.md", content: "v3" }),
    ];
    const changes = buildSessionChanges(items, undefined, cwd);
    expect(changes).toHaveLength(1);
    expect(changes[0].count).toBe(3);
    expect(changes[0].path).toBe("交互文件/大纲.md"); // 首次原始写法
  });

  it("不同文件不误合并", () => {
    const items: Item[] = [
      tool("t1", { path: "交互文件/大纲.md", content: "v1" }),
      tool("t2", { path: "交互文件/细纲.md", content: "v2" }),
    ];
    expect(buildSessionChanges(items, undefined, cwd)).toHaveLength(2);
  });
});

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
    for (const name of ["write_file", "edit_file", "edit_lines", "multi_edit", "move_file", "replace"]) {
      expect(WRITE_TOOL_NAMES.has(name)).toBe(true);
    }
    // v4.519：幽灵名摘出名单后不再是写类（工具从未注册，变更面板不应计入）
    expect(WRITE_TOOL_NAMES.has("delete_range")).toBe(false);
  });

  // v4.146 diff 确认闭环刀A：schedule_apply 纳入写类白名单 + 缺省路径回填
  it("schedule_apply 纳入白名单；显式 path 直取", () => {
    expect(WRITE_TOOL_NAMES.has("schedule_apply")).toBe(true);
    expect(extractChangedPaths('{"path":"进度计划/办公楼二期.gsched.json"}', "schedule_apply"))
      .toEqual(["进度计划/办公楼二期.gsched.json"]);
  });

  it("schedule_apply 缺省 path 回填当前计划（其他工具不回填）", () => {
    expect(extractChangedPaths('{"ops":[]}', "schedule_apply")).toEqual(["进度计划/当前计划.gsched.json"]);
    expect(extractChangedPaths("{}", "schedule_apply")).toEqual(["进度计划/当前计划.gsched.json"]);
    expect(extractChangedPaths("{}", "write_file")).toEqual([]);
    expect(extractChangedPaths('{"path":"a.md"}')).toEqual(["a.md"]); // 不带 tool 名=旧行为不变
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

  it("schedule_apply：显式 path 与缺省回填都进聚合（v4.146 刀A）", () => {
    const items: Item[] = [
      { kind: "tool", id: "s1", name: "schedule_apply", args: '{"path":"进度计划/办公楼二期.gsched.json","project":{}}', readOnly: false, status: "done" },
      { kind: "tool", id: "s2", name: "schedule_apply", args: '{"ops":[{"op":"add_task"}]}', readOnly: false, status: "done" },
    ];
    expect(buildSessionChanges(items)).toEqual([
      { path: "进度计划/当前计划.gsched.json", count: 1, lastTouched: 1 },
      { path: "进度计划/办公楼二期.gsched.json", count: 1, lastTouched: 0 },
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
