// deliverableStatus 纯函数单测：验收状态机（open/confirmed/redo）+ 新版本
// 重置语义（登记表 updatedAt 前进 → 回 open）+ localStorage 薄 IO 壳降级。
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  acceptanceOf,
  acceptanceSummary,
  loadAcceptanceMap,
  saveAcceptanceMap,
  setAcceptance,
  statusKeyOf,
  type DeliverableStatusMap,
} from "./deliverableStatus";

const KEY = "gaea.deliverableAcceptance.v1";

afterEach(() => {
  try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  vi.restoreAllMocks();
});

describe("statusKeyOf 会话+产物路径归一化键", () => {
  it("反斜杠归一 + 大小写不敏感（与交付卡去重键同口径）", () => {
    expect(statusKeyOf("S\\A.jsonl", "Exports\\报表.xlsx")).toBe(
      statusKeyOf("s/a.jsonl", "exports/报表.xlsx"),
    );
    expect(statusKeyOf("s/a.jsonl", "exports/报表.xlsx")).toBe("s/a.jsonl::exports/报表.xlsx");
  });
});

describe("acceptanceOf 读验收状态", () => {
  it("无记录 → open（缺省态不落存储）", () => {
    expect(acceptanceOf({}, "s.jsonl", "a.xlsx")).toBe("open");
  });

  it("confirmed / redo 原样读出", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s.jsonl", "a.xlsx", "confirmed", 1000, 100);
    map = setAcceptance(map, "s.jsonl", "b.docx", "redo", 1000, 100);
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx")).toBe("confirmed");
    expect(acceptanceOf(map, "s.jsonl", "b.docx")).toBe("redo");
  });

  it("登记表 updatedAt 前进 → 新版本自动重置 open；持平/倒退/未知不重置", () => {
    const map = setAcceptance({}, "s.jsonl", "a.xlsx", "confirmed", 1000, 100);
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", 101)).toBe("open"); // 前进
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", 100)).toBe("confirmed"); // 持平
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", 99)).toBe("confirmed"); // 倒退
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", undefined)).toBe("confirmed"); // 未知
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", 0)).toBe("confirmed"); // 0=未知
  });

  it("versionAt=0（标记时未知版本）不因 currentUpdatedAt 重置——宁保持旧判断，勿误回待查看", () => {
    const map = setAcceptance({}, "s.jsonl", "a.xlsx", "redo", 1000); // versionAt 缺省 0
    expect(acceptanceOf(map, "s.jsonl", "a.xlsx", 999_999)).toBe("redo");
  });

  it("会话/路径键隔离：同路径不同会话互不影响，同会话不同路径互不影响", () => {
    const map = setAcceptance({}, "s1.jsonl", "a.xlsx", "confirmed", 1000);
    expect(acceptanceOf(map, "s2.jsonl", "a.xlsx")).toBe("open");
    expect(acceptanceOf(map, "s1.jsonl", "b.xlsx")).toBe("open");
    // 反斜杠/大小写变体命中同一记录
    expect(acceptanceOf(map, "S1.JSONL", "A.XLSX")).toBe("confirmed");
  });
});

describe("setAcceptance 写入", () => {
  it("写 confirmed 记录（status/at/versionAt），不改入参 map", () => {
    const prev: DeliverableStatusMap = {};
    const next = setAcceptance(prev, "s.jsonl", "a.xlsx", "confirmed", 1234, 500);
    expect(prev).toEqual({});
    expect(next["s.jsonl::a.xlsx"]).toEqual({ status: "confirmed", at: 1234, versionAt: 500 });
  });

  it("versionAt 缺省 0；redo 覆盖旧 confirmed 记录", () => {
    let map = setAcceptance({}, "s.jsonl", "a.xlsx", "confirmed", 1);
    expect(map["s.jsonl::a.xlsx"]?.versionAt).toBe(0);
    map = setAcceptance(map, "s.jsonl", "a.xlsx", "redo", 2, 300);
    expect(map["s.jsonl::a.xlsx"]).toEqual({ status: "redo", at: 2, versionAt: 300 });
  });

  it("status=open 等价清除记录（该键删除，其余记录保留）", () => {
    let map = setAcceptance({}, "s.jsonl", "a.xlsx", "confirmed", 1);
    map = setAcceptance(map, "s.jsonl", "b.docx", "redo", 1);
    map = setAcceptance(map, "s.jsonl", "a.xlsx", "open", 2);
    expect(map["s.jsonl::a.xlsx"]).toBeUndefined();
    expect(map["s.jsonl::b.docx"]).toBeTruthy();
  });
});

describe("acceptanceSummary 头部汇总计数", () => {
  it("混合状态计数：confirmed/redo/open 各归其位，total=paths.length", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s.jsonl", "a.xlsx", "confirmed", 1);
    map = setAcceptance(map, "s.jsonl", "b.docx", "redo", 1);
    expect(acceptanceSummary(map, "s.jsonl", ["a.xlsx", "b.docx", "c.md"])).toEqual({
      total: 3,
      confirmed: 1,
      redo: 1,
    });
  });

  it("currentUpdatedAtOf 命中新版本 → confirmed 回落 open 不计数（与逐行同口径）", () => {
    const map = setAcceptance({}, "s.jsonl", "a.xlsx", "confirmed", 1, 100);
    const s = acceptanceSummary(map, "s.jsonl", ["a.xlsx"], (p) =>
      p === "a.xlsx" ? 101 : undefined,
    );
    expect(s.confirmed).toBe(0);
    expect(s.total).toBe(1);
  });

  it("空列表 → 全零", () => {
    expect(acceptanceSummary({}, "s.jsonl", [])).toEqual({ total: 0, confirmed: 0, redo: 0 });
  });
});

describe("load/saveAcceptanceMap localStorage 薄 IO 壳", () => {
  it("save → load roundtrip", () => {
    const map = setAcceptance({}, "s.jsonl", "a.xlsx", "confirmed", 9, 90);
    saveAcceptanceMap(map);
    expect(loadAcceptanceMap()).toEqual(map);
  });

  it("空存储 / 损坏 JSON → 空对象（异常静默）", () => {
    expect(loadAcceptanceMap()).toEqual({});
    localStorage.setItem(KEY, "{not-json");
    expect(loadAcceptanceMap()).toEqual({});
  });

  it("save 异常静默（私密模式/配额满不抛错）", () => {
    vi.spyOn(localStorage, "setItem").mockImplementation(() => {
      throw new Error("quota exceeded");
    });
    expect(() => saveAcceptanceMap({ "s::a": { status: "confirmed", at: 1, versionAt: 0 } })).not.toThrow();
  });
});
