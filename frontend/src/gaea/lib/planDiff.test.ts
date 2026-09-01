import { describe, expect, it } from "vitest";
import { buildChangeCalls, buildChangeDiff, pathsMatch } from "./planDiff";
import type { Item } from "./store";

const editArgs = JSON.stringify({
  path: "/ws/报告.md",
  old_string: "旧标题\n共有的行",
  new_string: "新标题\n共有的行",
});

describe("planDiff buildChangeDiff（变更 tab 数据源诚实评估）", () => {
  it("edit_file 有 old_string/new_string → 行级红绿 diff", () => {
    const d = buildChangeDiff("edit_file", editArgs);
    expect(d.kind).toBe("diff");
    expect(d.hunks).toHaveLength(1);
    expect(d.hunks[0].rows.map((r) => `${r.type}:${r.text}`)).toEqual([
      "del:旧标题",
      "add:新标题",
      "ctx:共有的行",
    ]);
  });

  it("multi_edit 逐片段构造 diff，多片段时带「编辑 N」标注", () => {
    const args = JSON.stringify({
      path: "/ws/a.md",
      edits: [
        { old_string: "one", new_string: "ONE" },
        { old_string: "two\nthree", new_string: "two\nTHREE" },
      ],
    });
    const d = buildChangeDiff("multi_edit", args);
    expect(d.kind).toBe("diff");
    expect(d.hunks).toHaveLength(2);
    expect(d.hunks[0].label).toBe("编辑 1");
    expect(d.hunks[1].label).toBe("编辑 2");
    expect(d.hunks[1].rows.map((r) => r.text)).toEqual(["two", "three", "THREE"]);
  });

  it("edit_lines 只有新内容 → 诚实降级为写入内容预览，不伪造 diff", () => {
    const d = buildChangeDiff(
      "edit_lines",
      JSON.stringify({ path: "/ws/a.md", start_line: 3, end_line: 5, new_content: "新行A\n新行B" }),
    );
    expect(d.kind).toBe("content");
    expect(d.content).toBe("新行A\n新行B");
    expect(d.note).toContain("第 3–5 行");
    expect(d.note).toContain("未随事件记录");
    expect(d.hunks).toHaveLength(0);
  });

  it("write_file 覆盖写入 → 降级为写入内容预览", () => {
    const d = buildChangeDiff("write_file", JSON.stringify({ path: "/ws/new.md", content: "全文内容" }));
    expect(d.kind).toBe("content");
    expect(d.content).toBe("全文内容");
    expect(d.note).toContain("覆盖写入");
  });

  it("move_file / 未知工具 / 参数缺失 → kind=none 并说明原因", () => {
    expect(buildChangeDiff("move_file", JSON.stringify({ source: "a", destination: "b" })).kind).toBe("none");
    expect(buildChangeDiff("delete_range", "{}").note).toContain("无法构造行级 diff");
    expect(buildChangeDiff("edit_file", "不是JSON").kind).toBe("none");
    expect(buildChangeDiff("write_file", "{}").kind).toBe("none");
  });

  it("CRLF 文本归一为 LF 再做行 diff（行内容不残留 \\r）", () => {
    const d = buildChangeDiff(
      "edit_file",
      JSON.stringify({ path: "a.txt", old_string: "a\r\nb", new_string: "a\r\nc" }),
    );
    expect(d.kind).toBe("diff");
    expect(d.hunks[0].rows.map((r) => r.text)).toEqual(["a", "b", "c"]);
  });
});

function toolItem(id: string, name: string, args: string, status: "done" | "error" | "running" | "stopped" = "done"): Item {
  return { kind: "tool", id, name, args, readOnly: false, status } as Item;
}

describe("planDiff buildChangeCalls（路径 → 写类调用明细）", () => {
  it("按路径聚合写类调用，非写类工具忽略", () => {
    const items: Item[] = [
      toolItem("t1", "read_file", JSON.stringify({ path: "/ws/a.md" })),
      toolItem("t2", "edit_file", editArgs),
      toolItem("t3", "write_file", JSON.stringify({ path: "b.md", content: "hi" })),
    ];
    const map = buildChangeCalls(items);
    expect([...map.keys()].sort()).toEqual(["/ws/报告.md", "b.md"].sort());
    expect(map.get("/ws/报告.md")).toHaveLength(1);
    expect(map.get("/ws/报告.md")![0].diff.kind).toBe("diff");
    expect(map.get("/ws/报告.md")![0].applied).toBe(true);
  });

  it("失败调用保留但 applied=false（UI 显式标注不构成变更）", () => {
    const items: Item[] = [
      toolItem("t1", "edit_file", editArgs, "error"),
    ];
    const map = buildChangeCalls(items);
    expect(map.get("/ws/报告.md")![0].applied).toBe(false);
  });
});

describe("planDiff pathsMatch（变更路径 ↔ Journal target）", () => {
  it("一致（含斜杠方向差异）匹配", () => {
    expect(pathsMatch("/ws/a/b.md", "\\ws\\a\\b.md")).toBe(true);
    expect(pathsMatch("docs/报告.md", "docs/报告.md")).toBe(true);
  });
  it("绝对 vs 相对：带边界的后缀匹配（较短路径至少两段）", () => {
    expect(pathsMatch("/home/u/ws/docs/报告.md", "docs/报告.md")).toBe(true);
    expect(pathsMatch("docs/报告.md", "/home/u/ws/docs/报告.md")).toBe(true);
  });
  it("单段文件名不做后缀匹配（避免同名误配）", () => {
    expect(pathsMatch("/other/目录/b.md", "b.md")).toBe(false);
  });
  it("无关路径不匹配", () => {
    expect(pathsMatch("docs/a.md", "docs/b.md")).toBe(false);
  });
});
