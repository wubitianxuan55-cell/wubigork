import { describe, expect, it } from "vitest";
import { buildSessionChanges, deliverableDedupKey } from "./changes";
import type { Item } from "./store";

// 2026-11-16「4 个相同交互文件」根修行为锁：交付路径去重键归一化——同一
// 文件以绝对/相对、正/反斜杠、大小写、./ 前缀等不同写法出现时必须收敛为
// 一个键、一条面板条目、计数累加；展示 path 保留首次原始写法。
describe("deliverableDedupKey", () => {
  const cwd = "C:\\AI\\tuwen\\东北之光";

  it("同一文件的四种写法收敛为一个键", () => {
    const keys = new Set([
      deliverableDedupKey("交互文件/大纲.md", cwd),
      deliverableDedupKey("C:\\AI\\tuwen\\东北之光\\交互文件\\大纲.md", cwd),
      deliverableDedupKey("C:/AI/tuwen/东北之光/交互文件/大纲.md", cwd),
      deliverableDedupKey("./交互文件/大纲.md", cwd),
      deliverableDedupKey(".\\交互文件\\大纲.md", cwd),
      deliverableDedupKey("交互文件\\大纲.md", cwd),
    ]);
    expect(keys.size).toBe(1);
  });

  it("大小写不敏感（Windows 文件系统）", () => {
    expect(deliverableDedupKey("C:/AI/Tuwen/书/交互文件/X.md", cwd)).toBe(
      deliverableDedupKey("c:/ai/tuwen/书/交互文件/x.md", cwd),
    );
  });

  it("工作区外的绝对路径保持独立且仍归一化", () => {
    const a = deliverableDedupKey("D:\\其他\\a.md", "C:/AI/tuwen/书");
    const b = deliverableDedupKey("d:/其他/a.md", "C:\\AI\\tuwen\\书");
    expect(a).toBe(b);
    expect(a).not.toBe(deliverableDedupKey("a.md", "C:/AI/tuwen/书"));
  });

  it("无 cwd 时退化为分隔符+大小写归一", () => {
    expect(deliverableDedupKey("交互文件\\X.md")).toBe(deliverableDedupKey("交互文件/x.md"));
    // 相对与绝对在无 cwd 时无法合流（诚实降级），键不同但各自稳定。
    expect(deliverableDedupKey("交互文件/x.md")).not.toBe(
      deliverableDedupKey("C:/AI/tuwen/书/交互文件/x.md"),
    );
  });

  it("工作区根本身与空路径", () => {
    expect(deliverableDedupKey("C:\\AI\\tuwen\\东北之光", cwd)).toBe("");
    expect(deliverableDedupKey("  交互文件/x.md  ", cwd)).toBe(deliverableDedupKey("交互文件/x.md", cwd));
  });
});

describe("buildSessionChanges 归一化合并", () => {
  const cwd = "C:\\AI\\tuwen\\东北之光";
  const tool = (id: string, args: Record<string, unknown>): Item => ({
    kind: "tool", id, name: "write_file", args: JSON.stringify(args), readOnly: false, status: "done",
  });

  it("同一文件多种写法合并且计数累加，展示保留首次写法", () => {
    const items: Item[] = [
      tool("t1", { path: "交互文件/大纲.md", content: "v1" }),
      { kind: "assistant", id: "a1", text: "已写入 C:\\AI\\tuwen\\东北之光\\交互文件\\大纲.md", reasoning: "", streaming: false },
      tool("t2", { path: "C:/AI/tuwen/东北之光/交互文件/大纲.md", content: "v2" }),
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
