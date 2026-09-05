import { describe, expect, it } from "vitest";
import {
  countGenuiNodes,
  repairGenuiSpec,
  repairSingleComponent,
  validateGenuiSpec,
} from "./guard";
import { GENUI_LIMITS } from "./spec";

describe("guard.repairGenuiSpec", () => {
  it("修复合法规格：保留已知节点并 clamp 数字", () => {
    const spec = repairGenuiSpec({
      title: "看板",
      items: [
        { type: "stat", label: "营收", value: "¥128k", delta: "+12%" },
        { type: "progress", label: "完成", value: 150, valueLabel: "150%" },
        { type: "button", label: "刷新", action: "refresh" },
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.title).toBe("看板");
    expect(spec?.items).toHaveLength(3);
    const progress = spec?.items[1];
    if (progress?.type !== "progress") throw new Error("progress expected");
    expect(progress.value).toBe(100);
  });

  it("丢弃未知 type 与缺必填字段的节点", () => {
    const spec = repairGenuiSpec({
      items: [
        { type: "evil", html: "<script>" },
        { type: "text" },
        { type: "stat", label: "ok", value: "1" },
      ],
    });
    expect(spec).not.toBeNull();
    expect(spec?.items).toHaveLength(1);
    expect(spec?.items[0]).toMatchObject({ type: "stat" });
  });

  it("根规格不是对象/无 items → null；单组件 root 自动包装", () => {
    expect(repairGenuiSpec("oops")).toBeNull();
    expect(repairGenuiSpec({ items: [] })).toBeNull();
    const wrapped = repairGenuiSpec({ type: "callout", content: "hi", tone: "info" });
    expect(wrapped?.items).toHaveLength(1);
    expect(wrapped?.items[0]).toMatchObject({ type: "callout" });
  });

  it("节点预算：超 200 节点时截断", () => {
    const items = Array.from({ length: 210 }, (_, i) => ({
      type: "text" as const,
      content: `n${i}`,
    }));
    const spec = repairGenuiSpec({ items });
    expect(spec?.items).toHaveLength(GENUI_LIMITS.maxNodes);
  });

  it("字符串按 cap 截断、颜色只收 hex/单词色", () => {
    const spec = repairGenuiSpec({
      items: [
        {
          type: "text",
          content: "x".repeat(3000),
        },
        {
          type: "avatar",
          name: "盖亚",
          color: "url(javascript:alert(1))",
        },
        { type: "avatar", name: "盖亚", color: "#2dd4bf" }, // hex-exempt 单测输入色值
      ],
    });
    expect(spec?.items[0]).toMatchObject({ type: "text" });
    const text = spec?.items[0];
    if (text?.type !== "text") throw new Error("text expected");
    expect(text.content).toHaveLength(GENUI_LIMITS.maxString);
    expect(spec?.items[1]).toMatchObject({ type: "avatar" });
    const bad = spec?.items[1];
    if (bad?.type !== "avatar") throw new Error("avatar expected");
    expect(bad.color).toBeUndefined();
    const good = spec?.items[2];
    if (good?.type !== "avatar") throw new Error("avatar expected");
    expect(good.color).toBe("#2dd4bf"); // hex-exempt 单测输入色值
  });

  it("表格行/列超限截断；quiz 选项不足丢弃", () => {
    const table = repairGenuiSpec({
      items: [
        {
          type: "table",
          columns: ["a", "b"],
          rows: Array.from({ length: 60 }, () => ["1", "2"]),
        },
        { type: "quiz", question: "q", options: [{ label: "only" }] },
      ],
    });
    const t = table?.items[0];
    if (t?.type !== "table") throw new Error("table expected");
    expect(t.rows).toHaveLength(GENUI_LIMITS.maxTableRows);
    expect(table?.items).toHaveLength(1);
  });
});

describe("guard.repairSingleComponent", () => {
  it("修复单个组件", () => {
    expect(repairSingleComponent({ type: "divider" })).toMatchObject({ type: "divider" });
    expect(repairSingleComponent({ type: "unknown" })).toBeNull();
  });
});

describe("guard.countGenuiNodes", () => {
  it("统计嵌套节点数", () => {
    const spec = {
      type: "card",
      items: [{ type: "stat", label: "a", value: "1" }, { type: "text", content: "b" }],
    };
    expect(countGenuiNodes(spec)).toBe(3);
  });
});

describe("guard.validateGenuiSpec", () => {
  it("报告未知 type 与缺必填字段", () => {
    const v = validateGenuiSpec({
      items: [
        { type: "evil" },
        { type: "stat", label: "x" },
        { type: "text", content: "ok" },
      ],
    });
    expect(v.ok).toBe(false);
    expect(v.errors.join("\n")).toContain("未知组件 type");
    expect(v.errors.join("\n")).toContain("stat 缺少必填字段 value");
  });

  it("合法规格 ok", () => {
    expect(validateGenuiSpec({ items: [{ type: "divider" }] }).ok).toBe(true);
  });
});
