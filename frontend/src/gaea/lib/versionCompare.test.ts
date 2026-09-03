import { describe, expect, it } from "vitest";
import {
  acceptanceOf,
  setAcceptance,
  statusKeyOf,
  type DeliverableStatusMap,
} from "./deliverableStatus";
import type { DiffRow } from "./diff";
import {
  buildTextDiff,
  clampDiffRows,
  diffStatOf,
  extOfPath,
  isTextComparable,
} from "./versionCompare";

describe("deliverableStatus 验收状态机", () => {
  it("缺省 open；标记后读回同状态", () => {
    let map: DeliverableStatusMap = {};
    const key = statusKeyOf("/mock/sessions/a.jsonl", "out/报告.md");
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md")).toBe("open");
    map = setAcceptance(map, "/mock/sessions/a.jsonl", "out/报告.md", "confirmed", 1000, 1754438400);
    expect(map[key].status).toBe("confirmed");
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md", 1754438400)).toBe("confirmed");
  });

  it("路径归一：反斜杠/大小写同键", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "/Mock/Sessions/A.jsonl", "OUT\\报告.MD", "redo", 1000, 5);
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md", 5)).toBe("redo");
  });

  it("新版本落盘（updatedAt 前进）→ 重置 open；versionAt=0 不误重置", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s", "a.md", "confirmed", 1000, 100);
    expect(acceptanceOf(map, "s", "a.md", 101)).toBe("open");
    map = setAcceptance(map, "s", "a.md", "confirmed", 1000, 0);
    expect(acceptanceOf(map, "s", "a.md", 999)).toBe("confirmed");
  });

  it("open 等价清除记录", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s", "a.md", "redo", 1000, 1);
    map = setAcceptance(map, "s", "a.md", "open", 2000, 1);
    expect(acceptanceOf(map, "s", "a.md", 1)).toBe("open");
    expect(Object.keys(map)).toHaveLength(0);
  });

  it("不同会话同路径互不影响", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s1", "a.md", "confirmed", 1000, 1);
    expect(acceptanceOf(map, "s2", "a.md", 1)).toBe("open");
  });
});

describe("versionCompare 版本对比", () => {
  it("isTextComparable 按扩展名判定", () => {
    expect(isTextComparable("a.md")).toBe(true);
    expect(isTextComparable("b.XLSX")).toBe(false);
    expect(isTextComparable("docs/c.md")).toBe(true);
    expect(isTextComparable("noext")).toBe(false);
    expect(extOfPath("a.DOCX")).toBe(".docx");
  });

  it("buildTextDiff 行级差异与增删计数", () => {
    const r = buildTextDiff("a\nb\nc", "a\nc\nd");
    expect(r.add).toBe(1);
    expect(r.del).toBe(1);
    expect(r.rows.some((x) => x.type === "add" && x.text === "d")).toBe(true);
    expect(r.rows.some((x) => x.type === "del" && x.text === "b")).toBe(true);
  });

  it("diffStatOf 与 contentMissing 透传", () => {
    const rows = buildTextDiff("x", "x\ny").rows;
    expect(diffStatOf(rows)).toEqual({ add: 1, del: 0 });
    expect(buildTextDiff("", "", true).contentMissing).toBe(true);
  });

  it("clampDiffRows 未超限：原样返回且不标记截断", () => {
    const rows: DiffRow[] = [
      { type: "ctx", text: "a" },
      { type: "add", text: "b" },
    ];
    expect(clampDiffRows(rows, 200)).toEqual({ shown: rows, total: 2, truncated: false });
    expect(clampDiffRows([], 200)).toEqual({ shown: [], total: 0, truncated: false });
  });

  it("clampDiffRows 超限：只保留前 max 行并标记截断（长 diff 折叠）", () => {
    const long: DiffRow[] = Array.from({ length: 250 }, (_, i) => ({ type: "ctx", text: `L${i}` }));
    const c = clampDiffRows(long, 200);
    expect(c.truncated).toBe(true);
    expect(c.total).toBe(250);
    expect(c.shown).toHaveLength(200);
    expect(c.shown[0].text).toBe("L0");
    expect(c.shown[199].text).toBe("L199");
  });
});
