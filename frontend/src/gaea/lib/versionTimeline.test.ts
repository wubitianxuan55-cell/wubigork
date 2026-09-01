// versionTimeline.test.ts — 版本时间线纯函数单测（聚合/排序/过滤/标签）。
import { describe, expect, it } from "vitest";
import {
  groupVersionsByPath,
  normalizeVersionPath,
  versionLabel,
  versionStatusText,
  versionTimeText,
} from "./versionTimeline";
import type { JournalChangeRecord } from "./types";

// 固定本地时间构造（2026-09-01 14:05），避免时区漂移。
const at = new Date(2026, 8, 1, 14, 5, 0).getTime();

const rec = (over: Partial<JournalChangeRecord> & { id: string }): JournalChangeRecord => ({
  sessionId: "s1",
  space: "work",
  turn: 1,
  tool: "edit_file",
  target: "docs/周报.docx",
  beforeSummary: "旧内容",
  afterSummary: "新内容",
  at,
  status: "pending_verify",
  baselinePath: "C:/ws/.gaea/work/rollback/xxx.snap",
  ...over,
});

describe("normalizeVersionPath 路径归一", () => {
  it("Windows 反斜杠归一为正斜杠", () => {
    expect(normalizeVersionPath("docs\\sub\\周报.docx")).toBe("docs/sub/周报.docx");
  });

  it("正斜杠路径原样返回", () => {
    expect(normalizeVersionPath("docs/周报.docx")).toBe("docs/周报.docx");
  });
});

describe("groupVersionsByPath 版本聚合", () => {
  it("按 target 聚合，同路径多条记录归为一组", () => {
    const grouped = groupVersionsByPath([
      rec({ id: "a" }),
      rec({ id: "b", target: "docs/成本测算.xlsx" }),
      rec({ id: "c" }),
    ]);
    expect(grouped.size).toBe(2);
    expect(grouped.get("docs/周报.docx")).toHaveLength(2);
    expect(grouped.get("docs/成本测算.xlsx")).toHaveLength(1);
  });

  it("组内按 at 倒序（最新在前）", () => {
    const older = rec({ id: "old", at: at - 60_000 });
    const newer = rec({ id: "new", at: at });
    const grouped = groupVersionsByPath([older, newer]);
    expect(grouped.get("docs/周报.docx")?.map((r) => r.id)).toEqual(["new", "old"]);
  });

  it("只保留有 baselinePath 的记录（无基线快照不能预览/恢复）", () => {
    const grouped = groupVersionsByPath([
      rec({ id: "has-snap" }),
      rec({ id: "no-snap", baselinePath: undefined }),
    ]);
    expect(grouped.get("docs/周报.docx")?.map((r) => r.id)).toEqual(["has-snap"]);
  });

  it("反斜杠 target 与正斜杠产物路径归入同一组", () => {
    const grouped = groupVersionsByPath([
      rec({ id: "slash" }),
      rec({ id: "backslash", target: "docs\\周报.docx", at: at + 60_000 }),
    ]);
    expect(grouped.size).toBe(1);
    expect(grouped.get("docs/周报.docx")?.map((r) => r.id)).toEqual(["backslash", "slash"]);
  });

  it("空输入 → 空 Map", () => {
    expect(groupVersionsByPath([]).size).toBe(0);
  });
});

describe("versionTimeText / versionLabel 版本标签", () => {
  it("输出 HH:MM（本地时区 24 小时制）", () => {
    expect(versionTimeText(rec({ id: "a" }))).toBe("14:05");
  });

  it("at 非法（NaN）→ 占位 --:--", () => {
    expect(versionTimeText(rec({ id: "a", at: NaN }))).toBe("--:--");
  });

  it("versionLabel = HH:MM · 工具名", () => {
    expect(versionLabel(rec({ id: "a", tool: "xlsx_apply" }))).toBe("14:05 · xlsx_apply");
  });
});

describe("versionStatusText 状态文案", () => {
  it("已知状态码翻成中文", () => {
    expect(versionStatusText("pending_verify")).toBe("待复核");
    expect(versionStatusText("verified")).toBe("复核通过");
    expect(versionStatusText("warned")).toBe("复核警告");
    expect(versionStatusText("failed")).toBe("复核未通过");
    expect(versionStatusText("applied")).toBe("已应用");
  });

  it("未知状态透传原文，空状态显示占位", () => {
    expect(versionStatusText("rolled_back")).toBe("rolled_back");
    expect(versionStatusText("")).toBe("—");
  });
});
