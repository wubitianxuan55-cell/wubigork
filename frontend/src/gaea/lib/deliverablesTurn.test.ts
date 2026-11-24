// deliverablesTurn 单测：轮次口径映射（钉死后端 Turn 与前端 turnNo 的差一）、
// 登记条目按轮筛选、正文卡×登记卡合并去重、缺失态判定、共享拉取的失败降级。
// 轮次依据见 deliverablesTurn.ts 头注释（Go 侧 deliverable.go / log.go:582 /
// deliverable_test.go 黄金表）。
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  backendTurnOf,
  deliverablePathKey,
  ensureTurnRegistry,
  inDeliverableFolder,
  invalidateTurnCaches,
  mergeDeliverableCards,
  missingRegistryKeys,
  parentDirOf,
  registryEntriesForTurn,
  turnTailSegs,
} from "./deliverablesTurn";
import { app } from "./bridge";
import type { DeliverableEntry, SessionMeta } from "./types";

// bridge 整体替换：登记拉取/列目录的受控注入（不碰 window.go）。
vi.mock("./bridge", () => ({
  app: {
    ListSessions: vi.fn(),
    DeliverableRegistry: vi.fn(),
    ListDir: vi.fn(),
  },
}));

const mockedApp = vi.mocked(app, true);

const entry = (patch: Partial<DeliverableEntry>): DeliverableEntry => ({
  path: "p.md",
  tool: "write_file",
  turn: 1,
  updatedAt: 100,
  touches: 1,
  ...patch,
});

const session = (patch: Partial<SessionMeta>): SessionMeta => ({
  path: "s1.jsonl",
  preview: "",
  turns: 1,
  modTime: 0,
  current: false,
  ...patch,
});

beforeEach(() => {
  invalidateTurnCaches();
  vi.clearAllMocks();
});

describe("轮次口径映射（turnNo 0-based ↔ 后端 Turn 1-based）", () => {
  it("turnNo=0（首条用户消息）→ 后端 Turn=1", () => {
    expect(backendTurnOf(0)).toBe(1);
  });
  it("turnNo=n → 后端 Turn=n+1（第 n+1 条 user 消息前写的 turn_started 计数）", () => {
    expect(backendTurnOf(1)).toBe(2);
    expect(backendTurnOf(5)).toBe(6);
  });
});

describe("registryEntriesForTurn 按轮筛选", () => {
  const entries: DeliverableEntry[] = [
    entry({ path: "a.md", turn: 1 }),
    entry({ path: "b.md", turn: 2 }),
    entry({ path: "c.md", turn: 0 }), // 轮外（turn_started 之前派发）
  ];

  it("turnNo=0 只取后端 Turn=1 的条目；Turn=0（轮外）不匹配任何 turnNo", () => {
    expect(registryEntriesForTurn(entries, 0).map((e) => e.path)).toEqual(["a.md"]);
    expect(registryEntriesForTurn(entries, 1).map((e) => e.path)).toEqual(["b.md"]);
  });
  it("turnNo 缺省（轮外段）或登记表不可用 → 空数组（不并登记卡，维持现状）", () => {
    expect(registryEntriesForTurn(entries, undefined)).toEqual([]);
    expect(registryEntriesForTurn(null, 0)).toEqual([]);
    expect(registryEntriesForTurn(undefined, 3)).toEqual([]);
  });
});

describe("mergeDeliverableCards 合并与去重", () => {
  it("正文卡在前（出现顺序），登记-only 卡在后追加，turn 不匹配的不并入", () => {
    const entries: DeliverableEntry[] = [
      entry({ path: "交付/漏提.xlsx", turn: 2, updatedAt: 300 }), // 本轮（turnNo=1）
      entry({ path: "交付/上一轮.md", turn: 1, updatedAt: 200 }), // 前一轮
      entry({ path: "交付/图表.png", turn: 2, updatedAt: 250 }), // 本轮
    ];
    const merged = mergeDeliverableCards(["交付/成本.xlsx", "交付/图表.png"], entries, 1);
    expect(merged).toEqual([
      { path: "交付/成本.xlsx", from: "text" },
      { path: "交付/图表.png", from: "text" }, // 正文已提及 → 不再出登记卡
      { path: "交付/漏提.xlsx", from: "registry" }, // 登记-only 追加在后
    ]);
  });

  it("正文与登记路径按归一化键去重（反斜杠/大小写差异不算新文件）", () => {
    const entries: DeliverableEntry[] = [entry({ path: "交付\\成本测算.XLSX", turn: 1 })];
    const merged = mergeDeliverableCards(["交付/成本测算.xlsx"], entries, 0);
    expect(merged).toEqual([{ path: "交付/成本测算.xlsx", from: "text" }]);
    expect(deliverablePathKey("交付\\成本测算.XLSX")).toBe("交付/成本测算.xlsx");
  });

  it("正文无提及 + 登记表本轮有条目 → 只剩登记卡（启发式漏登补齐的场景）", () => {
    const entries: DeliverableEntry[] = [entry({ path: "交付/周报.docx", turn: 3 })];
    expect(mergeDeliverableCards([], entries, 2)).toEqual([
      { path: "交付/周报.docx", from: "registry" },
    ]);
  });

  it("turnNo 缺省 → 无论登记表有什么都只返回正文卡", () => {
    const entries: DeliverableEntry[] = [entry({ path: "交付/a.md", turn: 1 })];
    expect(mergeDeliverableCards(["交付/b.md"], entries, undefined)).toEqual([
      { path: "交付/b.md", from: "text" },
    ]);
  });
});

describe("交付文件夹过滤（2026-11-24 用户拍板）", () => {
  it("只收 交付/ 首段路径：其下可再分层；根目录/其他子文件夹/绝对/隐藏/相对跳转一律不收", () => {
    const entries: DeliverableEntry[] = [
      entry({ path: "交付/南岸/包3技术标.docx", turn: 1 }), // 其下再分层 → 收
      entry({ path: "交付\\box\\图纸.png", turn: 1 }), // 反斜杠归一后首段=交付 → 收
      entry({ path: "包3技术标.docx", turn: 1 }), // 根目录裸文件 → 不收
      entry({ path: "_draft/共享事实卡.md", turn: 1 }), // 草稿子文件夹 → 不收
      entry({ path: ".gaea/exports/方案.docx", turn: 1 }), // 隐藏分区 → 不收
      entry({ path: "D:/新工作/南岸/方案.docx", turn: 1 }), // 绝对路径 → 不收
      entry({ path: "../外部.md", turn: 1 }), // 相对跳转 → 不收
      entry({ path: "交付", turn: 1 }), // 文件夹本身 → 不收
    ];
    const merged = mergeDeliverableCards([], entries, 0);
    expect(merged.map((c) => c.path)).toEqual(["交付/南岸/包3技术标.docx", "交付\\box\\图纸.png"]);
    // 正文提及的草稿/根目录路径同样不进卡（turnNo=5 与登记轮不匹配，只回正文卡）
    const merged2 = mergeDeliverableCards(["交付/终稿.docx", "_draft/ch1.md", "根级.docx"], entries, 5);
    expect(merged2.map((c) => c.path)).toEqual(["交付/终稿.docx"]);
    expect(inDeliverableFolder("交付/x.docx")).toBe(true);
    expect(inDeliverableFolder("交付")).toBe(false);
  });
});

describe("缺失态判定 missingRegistryKeys", () => {
  it("目录列表确认不存在 → 标缺失（键归一化）；存在（大小写不敏感）→ 不标", () => {
    const entries: DeliverableEntry[] = [
      entry({ path: "reports\\丢失.docx", turn: 1 }),
      entry({ path: "reports/在的.docx", turn: 1 }),
    ];
    const listings = new Map([["reports", new Set(["在的.docx"])]]);
    const missing = missingRegistryKeys(entries, listings);
    expect(missing.has("reports/丢失.docx")).toBe(true);
    expect(missing.has(deliverablePathKey("reports/在的.docx"))).toBe(false);
  });

  it("目录探测失败（null）→ 不标缺失（宁漏勿误）；根级文件按 \"\" 目录核对", () => {
    const entries: DeliverableEntry[] = [
      entry({ path: "x/未知.md", turn: 1 }),
      entry({ path: "根级.xlsx", turn: 1 }),
    ];
    const listings = new Map<string, ReadonlySet<string> | null>([
      ["x", null], // ListDir 失败 → 不标缺失
      ["", new Set(["别的.xlsx"])], // 根目录（GaeaListDir("") = 工作区根）：根级.xlsx 不在 → 标缺失
    ]);
    const missing = missingRegistryKeys(entries, listings);
    expect(missing.size).toBe(1);
    expect(missing.has("根级.xlsx")).toBe(true);
  });
});

describe("parentDirOf 父目录", () => {
  it("正斜杠 / 反斜杠 / 根级", () => {
    expect(parentDirOf("a/b/c.md")).toBe("a/b");
    expect(parentDirOf("a\\b\\c.md")).toBe("a/b");
    expect(parentDirOf("c.md")).toBe("");
  });
});

describe("ensureTurnRegistry 拉取与失败降级", () => {
  it("ListSessions 找 current 会话 → DeliverableRegistry 返回登记表", async () => {
    mockedApp.ListSessions.mockResolvedValue([session({ path: "s1.jsonl", current: true })]);
    mockedApp.DeliverableRegistry.mockResolvedValue({
      available: true,
      entries: [entry({ path: "a.md", turn: 1 })],
      total: 1,
    });
    const reg = await ensureTurnRegistry();
    expect(reg?.entries.map((e) => e.path)).toEqual(["a.md"]);
    expect(mockedApp.ListSessions).toHaveBeenCalledTimes(1);
  });

  it("ListSessions / DeliverableRegistry 拒绝 → 静默降级 null（现状=只有正文卡）", async () => {
    mockedApp.ListSessions.mockRejectedValue(new Error("bridge down"));
    await expect(ensureTurnRegistry()).resolves.toBeNull();

    invalidateTurnCaches();
    mockedApp.ListSessions.mockResolvedValue([session({ path: "s1.jsonl", current: true })]);
    mockedApp.DeliverableRegistry.mockRejectedValue(new Error("log gone"));
    await expect(ensureTurnRegistry()).resolves.toBeNull();
  });

  it("无 current 会话（未保存草稿）→ null，不调 DeliverableRegistry", async () => {
    mockedApp.ListSessions.mockResolvedValue([session({ path: "s0.jsonl" })]);
    const reg = await ensureTurnRegistry();
    expect(reg).toBeNull();
    expect(mockedApp.DeliverableRegistry).not.toHaveBeenCalled();
  });

  it("TTL 内共享同一份 fetch（多卡一次拉取），失效后重拉", async () => {
    mockedApp.ListSessions.mockResolvedValue([session({ path: "s1.jsonl", current: true })]);
    mockedApp.DeliverableRegistry.mockResolvedValue({ available: true, entries: [], total: 0 });
    await ensureTurnRegistry(1000);
    await ensureTurnRegistry(1500); // TTL 内：复用
    expect(mockedApp.ListSessions).toHaveBeenCalledTimes(1);
    await ensureTurnRegistry(5000); // 超过 2s 去重窗：重拉
    expect(mockedApp.ListSessions).toHaveBeenCalledTimes(2);
  });
});

describe("turnTailSegs 轮尾段集合（登记-only 卡同轮去重）", () => {
  const m = (entries: [number, number][]): Map<number, number> => new Map(entries);

  it("单轮多段：只有最后一段是轮尾", () => {
    expect(turnTailSegs(m([[0, 0], [1, 0], [2, 0]]))).toEqual(new Set([2]));
  });

  it("多轮交替：每轮各留一个轮尾", () => {
    expect(turnTailSegs(m([[0, 0], [1, 0], [2, 1], [3, 1], [4, 2]]))).toEqual(new Set([1, 3, 4]));
  });

  it("空 map 与单段", () => {
    expect(turnTailSegs(m([]))).toEqual(new Set());
    expect(turnTailSegs(m([[5, 2]]))).toEqual(new Set([5]));
  });
});
