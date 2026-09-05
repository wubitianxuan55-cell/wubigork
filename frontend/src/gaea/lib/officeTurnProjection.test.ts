// officeTurnProjection 单测 — U2 回合投影与审阅收口（规划 §4.2 表 2/3/4/5 行）。
// 覆盖：三类操作归约（写入/验证/生命周期）、callId 配对（乱序/孤儿/重复容错）、
// 失败保留原因不提交转换、wire/items 适配器、预览浮窗语义状态机迁移表
// （写弹读不弹/关闭优先/意图跨回合/终态清理）、draft/ready 判定（首次写盘=
// 草稿、Plan→Apply 批准=就绪）与 Journal 共享薄壳降级。
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  extractOfficeReadPaths,
  extractOfficeWritePaths,
  deliverablePhaseOf,
  ensureOfficeJournal,
  initialPreviewAutoFrontState,
  invalidateOfficeJournal,
  isOfficeDeliverablePath,
  officeToolEventsFromItems,
  officeToolEventsFromWire,
  previewAutoFrontReduce,
  projectOfficeTurn,
} from "./officeTurnProjection";
import { app } from "./bridge";
import type { DeliverableEntry, JournalChangeRecord } from "./types";

vi.mock("./bridge", () => ({
  app: {
    GaeaJournalList: vi.fn(),
  },
}));

// ── 事件构造器（形状对齐 WireTool / store.Item，不发明字段）──────────────
const d = (callId: string, tool: string, args?: string) =>
  ({ kind: "dispatch", callId, tool, args }) as const;
const r = (callId: string, tool: string, err?: string) =>
  ({ kind: "result", callId, tool, err }) as const;

const rec = (over: Partial<JournalChangeRecord> & { id: string }): JournalChangeRecord => ({
  sessionId: "s1",
  space: "work",
  turn: 1,
  tool: "write_file",
  target: "docs/方案.docx",
  beforeSummary: "",
  afterSummary: "",
  at: 1000,
  status: "applied",
  ...over,
});

beforeEach(() => {
  vi.mocked(app.GaeaJournalList).mockReset();
  invalidateOfficeJournal();
});

// ── §1 路径提取 ──────────────────────────────────────────────────────────
describe("officeTurnProjection 路径提取（对齐 Go ExtractDeliverablePaths 口径）", () => {
  it("写类工具取 path/file_path/destination/paths/edits，去重保序", () => {
    expect(
      extractOfficeWritePaths(
        "multi_edit",
        JSON.stringify({ path: "a.docx", edits: [{ path: "b.docx" }, { file_path: "a.docx" }] }),
      ),
    ).toEqual(["a.docx", "b.docx"]);
    expect(
      extractOfficeWritePaths("move_file", JSON.stringify({ source: "x.docx", destination: "y.docx" })),
    ).toEqual(["y.docx"]);
    expect(
      extractOfficeWritePaths("write_file", JSON.stringify({ paths: ["p1.xlsx", "p2.xlsx"] })),
    ).toEqual(["p1.xlsx", "p2.xlsx"]);
  });

  it("生成/导出类只取 output（path 是输入源文件，不是交付物）", () => {
    expect(
      extractOfficeWritePaths("format_convert", JSON.stringify({ path: "输入.docx", output: "out/导出.md" })),
    ).toEqual(["out/导出.md"]);
    expect(extractOfficeWritePaths("chart_gen", JSON.stringify({ rel: "表.xlsx" }))).toEqual([]);
  });

  it("读取工具取 path/source/image_path；坏 JSON 与空参安全返回空", () => {
    expect(extractOfficeReadPaths(JSON.stringify({ path: "docs/x.xlsx" }))).toEqual(["docs/x.xlsx"]);
    expect(extractOfficeReadPaths(JSON.stringify({ image_path: "shot.png" }))).toEqual(["shot.png"]);
    expect(extractOfficeReadPaths("{bad json")).toEqual([]);
    expect(extractOfficeReadPaths(undefined)).toEqual([]);
  });

  it("Office 文档扩展名判定：办公文档命中，代码/图片不命中", () => {
    expect(isOfficeDeliverablePath("docs/方案.docx")).toBe(true);
    expect(isOfficeDeliverablePath("exports/成本测算.XLSX")).toBe(true);
    expect(isOfficeDeliverablePath("a/b.ofd")).toBe(true);
    expect(isOfficeDeliverablePath("src/main.ts")).toBe(false);
    expect(isOfficeDeliverablePath("img/趋势.png")).toBe(false);
  });
});

// ── §2 回合投影 ──────────────────────────────────────────────────────────
describe("officeTurnProjection 三类操作归约", () => {
  it("写入→回读→生命周期读取各归其类；回读只认成功写入之后的读取", () => {
    const p = projectOfficeTurn([
      d("c1", "read_file", JSON.stringify({ path: "docs/方案.docx" })),
      r("c1", "read_file"),
      d("c2", "edit_file", JSON.stringify({ path: "docs/方案.docx" })),
      r("c2", "edit_file"),
      d("c3", "read_file", JSON.stringify({ path: "docs/方案.docx" })),
      r("c3", "read_file"),
      d("c4", "grep", JSON.stringify({ path: "未写过.md" })),
      r("c4", "grep"),
    ]);
    expect(p.calls.map((c) => c.kind)).toEqual(["lifecycle", "write", "verify", "lifecycle"]);
    expect(p.writes).toHaveLength(1);
    expect(p.verifies).toHaveLength(1);
    expect(p.lifecycle).toHaveLength(2);
    expect(p.changed).toBe(true);
    expect(p.verifiedAll).toBe(true);
    expect(p.hasFailure).toBe(false);
    expect(p.files).toHaveLength(2);
    const f = p.files.find((v) => v.path === "docs/方案.docx")!;
    expect(f.writes).toBe(1);
    expect(f.readBacks).toBe(1);
    expect(f.verified).toBe(true);
    expect(f.reads).toBe(1);
  });

  it("失败写入：不提交转换（changed=false）、保留失败原因、verifiedAll 不虚报", () => {
    const p = projectOfficeTurn([
      d("c1", "edit_file", JSON.stringify({ path: "docs/方案.docx" })),
      r("c1", "edit_file", "Error [OFFICE_LOCKED]: 文件被占用"),
      d("c2", "read_file", JSON.stringify({ path: "docs/方案.docx" })),
      r("c2", "read_file"),
    ]);
    expect(p.changed).toBe(false);
    expect(p.hasFailure).toBe(true);
    expect(p.calls[0].status).toBe("error");
    expect(p.calls[0].err).toBe("Error [OFFICE_LOCKED]: 文件被占用");
    // 写入失败 → 该文件从未进入「已写」集合，随后的读取不算回读
    expect(p.calls[1].kind).toBe("lifecycle");
    const view = p.files.find((v) => v.path === "docs/方案.docx")!;
    expect(view.failedWrites).toBe(1);
    expect(view.lastWriteErr).toBe("Error [OFFICE_LOCKED]: 文件被占用");
    expect(view.writes).toBe(0);
    expect(view.readBacks).toBe(0);
    expect(view.verified).toBe(false);
    expect(p.verifiedAll).toBe(false);
  });

  it("乱序（result 先到）照常配对；孤儿 result 标记 orphan 且照常计数", () => {
    const p = projectOfficeTurn([
      r("c1", "write_file"),
      d("c1", "write_file", JSON.stringify({ path: "out/报告.docx" })),
      r("c9", "edit_file", undefined),
    ]);
    expect(p.calls).toHaveLength(2);
    expect(p.calls[0]).toMatchObject({ callId: "c1", status: "ok", kind: "write", orphan: false });
    expect(p.calls[1]).toMatchObject({ callId: "c9", kind: "write", orphan: true, status: "ok" });
    expect(p.changed).toBe(true);
  });

  it("重复 dispatch / 重复 result：取首个并标记 duplicate；重复 result 不重复计数", () => {
    const p = projectOfficeTurn([
      d("c1", "write_file", JSON.stringify({ path: "a.xlsx" })),
      d("c1", "write_file", JSON.stringify({ path: "a.xlsx" })),
      r("c1", "write_file"),
      r("c1", "write_file"),
    ]);
    expect(p.calls).toHaveLength(1);
    expect(p.calls[0].duplicate).toBe(true);
    const view = p.files.find((v) => v.path === "a.xlsx")!;
    expect(view.writes).toBe(1);
  });

  it("只有 dispatch 无 result：pending，不触发 changed", () => {
    const p = projectOfficeTurn([d("c1", "write_file", JSON.stringify({ path: "a.xlsx" }))]);
    expect(p.calls[0].status).toBe("pending");
    expect(p.changed).toBe(false);
    expect(p.hasFailure).toBe(false);
  });

  it("无 callId / 无工具名的坏事件直接丢弃；空输入返回空投影", () => {
    const p = projectOfficeTurn([
      d("", "write_file"),
      { kind: "result", callId: "x", tool: "" } as never,
    ]);
    expect(p.calls).toHaveLength(0);
    expect(projectOfficeTurn([]).files).toHaveLength(0);
  });

  it("第二次写入后未再回读：verified 归为 false（诚实口径）", () => {
    const p = projectOfficeTurn([
      d("c1", "write_file", JSON.stringify({ path: "a.xlsx" })),
      r("c1", "write_file"),
      d("c2", "read_file", JSON.stringify({ path: "a.xlsx" })),
      r("c2", "read_file"),
      d("c3", "edit_file", JSON.stringify({ path: "a.xlsx" })),
      r("c3", "edit_file"),
    ]);
    const view = p.files.find((v) => v.path === "a.xlsx")!;
    expect(view.writes).toBe(2);
    expect(view.readBacks).toBe(1);
    expect(view.verified).toBe(false);
    expect(p.verifiedAll).toBe(false);
  });
});

// ── §2 适配器 ────────────────────────────────────────────────────────────
describe("officeTurnProjection wire/items 适配器", () => {
  it("fromWire：tool_dispatch/tool_result → 归一化事件；partial 预告跳过；缺 id 回退工具名", () => {
    // 真实事件流混有其它 kind（text 等）：适配器必须跳过，此处整体收窄断言形状。
    const events = [
      { kind: "tool_dispatch", tool: { id: "t1", name: "write_file", args: '{"path":"a.docx"}', readOnly: false, partial: true } },
      { kind: "tool_dispatch", tool: { id: "t1", name: "write_file", args: '{"path":"a.docx"}', readOnly: false } },
      { kind: "tool_result", tool: { id: "t1", name: "write_file", output: "ok", readOnly: false } },
      { kind: "text", text: "正文" },
      { kind: "tool_dispatch", tool: { name: "grep", readOnly: true } },
    ] as unknown as Parameters<typeof officeToolEventsFromWire>[0];
    const evs = officeToolEventsFromWire(events);
    expect(evs).toHaveLength(3);
    expect(evs[0]).toEqual({ kind: "dispatch", callId: "t1", tool: "write_file", args: '{"path":"a.docx"}' });
    expect(evs[1]).toEqual({ kind: "result", callId: "t1", tool: "write_file", output: "ok", err: undefined });
    expect(evs[2]).toEqual({ kind: "dispatch", callId: "grep", tool: "grep", args: undefined });
    const proj = projectOfficeTurn(evs);
    expect(proj.changed).toBe(true);
  });

  it("fromItems：running=只有 dispatch，done=ok result，error=err result，stopped 保持 pending", () => {
    // 混入非 tool 成员（user）：适配器按 kind 过滤，整体收窄到入参形状。
    const raw = [
      { kind: "user", id: "u1", text: "改表" },
      { kind: "tool", id: "t1", name: "edit_file", args: '{"path":"a.xlsx"}', status: "running" },
      { kind: "tool", id: "t2", name: "write_file", args: '{"path":"b.docx"}', status: "done" },
      { kind: "tool", id: "t3", name: "write_file", args: '{"path":"c.docx"}', status: "error", error: "Error [OFFICE_IO]: 磁盘满" },
      { kind: "tool", id: "t4", name: "write_file", args: '{"path":"d.docx"}', status: "stopped" },
    ] as unknown as Parameters<typeof officeToolEventsFromItems>[0];
    const evs = officeToolEventsFromItems(raw);
    const proj = projectOfficeTurn(evs);
    const byId = new Map(proj.calls.map((c) => [c.callId, c]));
    expect(byId.get("t1")?.status).toBe("pending");
    expect(byId.get("t2")?.status).toBe("ok");
    expect(byId.get("t3")?.status).toBe("error");
    expect(byId.get("t3")?.err).toBe("Error [OFFICE_IO]: 磁盘满");
    expect(byId.get("t4")?.status).toBe("pending");
    expect(proj.changed).toBe(true);
    expect(proj.hasFailure).toBe(true);
  });
});

// ── §3 预览浮窗语义状态机 ────────────────────────────────────────────────
describe("previewAutoFrontReduce 浮窗语义状态机", () => {
  const dispatch = (s = initialPreviewAutoFrontState, path = "docs/方案.docx") =>
    previewAutoFrontReduce(s, { type: "writeDispatch", path });
  const result = (s: ReturnType<typeof dispatch>["state"], path = "docs/方案.docx", ok = true) =>
    previewAutoFrontReduce(s, { type: "writeResult", path, ok });

  it("写弹：写意图在回执成功时兑现为 open（写入事件才自动置前）", () => {
    const s1 = dispatch();
    expect(s1.state.intentPath).toBe("docs/方案.docx");
    const s2 = result(s1.state);
    expect(s2.action).toEqual({ type: "open", path: "docs/方案.docx" });
    expect(s2.state.intentPath).toBeNull();
    expect(s2.state.openedThisTurn.has("docs/方案.docx")).toBe(true);
  });

  it("读不弹：读取事件恒 no-op；已关闭浮窗被读取也不复活", () => {
    let s = initialPreviewAutoFrontState;
    s = previewAutoFrontReduce(s, { type: "read", path: "docs/方案.docx" }).state;
    expect(s).toEqual(initialPreviewAutoFrontState);
    // 写→弹→用户关→同回合再读：不复活
    const opened = result(dispatch().state).state;
    const closed = previewAutoFrontReduce(opened, { type: "userClose", path: "docs/方案.docx" }).state;
    const after = previewAutoFrontReduce(closed, { type: "read", path: "docs/方案.docx" });
    expect(after.state).toEqual(closed);
    expect(after.action.type).toBe("none");
  });

  it("关闭优先：dispatch 后用户关闭 → 回执到达不再弹（意图已被清除）", () => {
    const s1 = dispatch().state;
    const s2 = previewAutoFrontReduce(s1, { type: "userClose", path: "docs/方案.docx" }).state;
    expect(s2.intentPath).toBeNull();
    expect(s2.closedPaths.has("docs/方案.docx")).toBe(true);
    const s3 = result(s2);
    expect(s3.action.type).toBe("none");
  });

  it("失败不弹：写回执 error 只清意图；中断（stopped→ok=false 同口径）同理", () => {
    const s1 = dispatch().state;
    const s2 = result(s1, "docs/方案.docx", false);
    expect(s2.action.type).toBe("none");
    expect(s2.state.intentPath).toBeNull();
  });

  it("意图跨回合：写意图未兑现时 turnEnd 保持 intent，下回合回执仍兑现", () => {
    const s1 = dispatch().state;
    const s2 = previewAutoFrontReduce(s1, { type: "turnEnd" }).state;
    expect(s2.intentPath).toBe("docs/方案.docx");
    const s3 = result(s2);
    expect(s3.action).toEqual({ type: "open", path: "docs/方案.docx" });
  });

  it("终态清理：turnEnd 清空关闭集与已弹集，回合级状态复位", () => {
    let s = result(dispatch().state).state;
    s = previewAutoFrontReduce(s, { type: "userClose", path: "docs/方案.docx" }).state;
    expect(s.closedPaths.size).toBe(1);
    s = previewAutoFrontReduce(s, { type: "turnEnd" }).state;
    expect(s.closedPaths.size).toBe(0);
    expect(s.openedThisTurn.size).toBe(0);
  });

  it("同文件同回合至多自动置前一次：第二次成功写入不再 open", () => {
    let s = result(dispatch().state).state;
    s = dispatch(s).state;
    const s2 = result(s);
    expect(s2.action.type).toBe("none");
  });

  it("新写重新武装：用户关闭后新 dispatch 清除关闭标记，回执可再弹（跨回合）", () => {
    let s = result(dispatch().state).state;
    s = previewAutoFrontReduce(s, { type: "userClose", path: "docs/方案.docx" }).state;
    s = previewAutoFrontReduce(s, { type: "turnEnd" }).state;
    s = dispatch(s).state;
    expect(s.closedPaths.has("docs/方案.docx")).toBe(false);
    expect(s.intentPath).toBe("docs/方案.docx");
    expect(result(s).action.type).toBe("open");
  });

  it("空路径事件忽略；未知事件类型返回原状态", () => {
    const s1 = previewAutoFrontReduce(initialPreviewAutoFrontState, { type: "writeDispatch", path: "" });
    expect(s1.state).toEqual(initialPreviewAutoFrontState);
    const s2 = previewAutoFrontReduce(initialPreviewAutoFrontState, { type: "nope" } as never);
    expect(s2.state).toEqual(initialPreviewAutoFrontState);
    expect(s2.action.type).toBe("none");
  });
});

// ── §4 草稿轻量版：draft/ready 判定 ─────────────────────────────────────
describe("deliverablePhaseOf 草稿/就绪判定（首次写盘=草稿、批准=就绪）", () => {
  it("xlsx_apply（Plan→Apply 批准）applied/verified → ready", () => {
    const journal = [rec({ id: "e1", tool: "xlsx_apply", target: "docs/成本测算.xlsx", status: "applied" })];
    expect(deliverablePhaseOf("docs/成本测算.xlsx", journal)).toBe("ready");
    expect(
      deliverablePhaseOf("docs\\成本测算.xlsx", [
        rec({ id: "e1", tool: "xlsx_apply", target: "docs/成本测算.xlsx", status: "verified" }),
      ]),
    ).toBe("ready");
  });

  it("xlsx_apply failed 不算批准：退回 draft（写盘痕迹仍在）", () => {
    const journal = [rec({ id: "e1", tool: "xlsx_apply", target: "docs/成本测算.xlsx", status: "failed" })];
    expect(deliverablePhaseOf("docs/成本测算.xlsx", journal)).toBe("draft");
  });

  it("普通写盘（write_file/edit_file 等）→ draft；路径反斜杠归一匹配", () => {
    const journal = [rec({ id: "e1", tool: "edit_file", target: "docs\\方案.docx" })];
    expect(deliverablePhaseOf("docs/方案.docx", journal)).toBe("draft");
  });

  it("登记表兜底：journal 为空/不可用时登记条目（任意写工具）→ draft", () => {
    const entry: Pick<DeliverableEntry, "tool"> = { tool: "write_file" };
    expect(deliverablePhaseOf("docs/方案.docx", null, entry)).toBe("draft");
    expect(deliverablePhaseOf("docs/方案.docx", [], entry)).toBe("draft");
  });

  it("非 Office 文档 / 毫无写盘证据 → null（宁缺勿误，不标）", () => {
    expect(deliverablePhaseOf("exports/趋势.png", [rec({ id: "e1" })])).toBeNull();
    expect(deliverablePhaseOf("docs/方案.docx", [])).toBeNull();
    expect(deliverablePhaseOf("docs/方案.docx", null)).toBeNull();
    expect(deliverablePhaseOf("", [rec({ id: "e1" })])).toBeNull();
  });

  it("混合：先写后批准 → ready；只有别的文件的记录 → draft 兜底生效", () => {
    const journal = [
      rec({ id: "e1", tool: "write_file", target: "docs/成本测算.xlsx" }),
      rec({ id: "e2", tool: "xlsx_apply", target: "docs/成本测算.xlsx", status: "applied" }),
    ];
    expect(deliverablePhaseOf("docs/成本测算.xlsx", journal)).toBe("ready");
    expect(
      deliverablePhaseOf("docs/其他.xlsx", journal, { tool: "format_convert" }),
    ).toBe("draft");
  });
});

// ── §4 Journal 共享薄壳 ─────────────────────────────────────────────────
describe("ensureOfficeJournal 共享薄壳（2s 去重 + 失败/缺绑定降级）", () => {
  it("并发取数共享同一 promise；invalidate 后重拉", async () => {
    const recs = [rec({ id: "e1" })];
    vi.mocked(app.GaeaJournalList).mockResolvedValue(recs);
    const p1 = ensureOfficeJournal();
    const p2 = ensureOfficeJournal();
    expect(p1).toBe(p2);
    expect(app.GaeaJournalList).toHaveBeenCalledTimes(1);
    await expect(p1).resolves.toBe(recs);
    invalidateOfficeJournal();
    const p3 = ensureOfficeJournal(0); // 失效后必重拉（与 now 取值无关）
    expect(p3).not.toBe(p1);
  });

  it("绑定缺失（测试门面未注入 GaeaJournalList）与失败都降级为 null，不抛错", async () => {
    const patched = app as { GaeaJournalList?: unknown };
    const saved = patched.GaeaJournalList;
    delete patched.GaeaJournalList;
    await expect(ensureOfficeJournal()).resolves.toBeNull();
    patched.GaeaJournalList = saved;
    vi.mocked(app.GaeaJournalList).mockRejectedValue(new Error("boom"));
    invalidateOfficeJournal();
    await expect(ensureOfficeJournal()).resolves.toBeNull();
  });
});
