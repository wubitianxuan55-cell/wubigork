// officeTurnProjection — U2「回合投影与审阅收口」前端纯函数层
// （docs/gaea-dsh-univer-office-distill-plan-2026-09.md §4.2 表 2/3/4/5 行）。
//
// 蒸馏自 dsh-univer-office 的 turn 投影（§1.6）：按 callId 配对 tool 的
// dispatch 与 result，三类操作独立归约——生命周期（读/打开）、写入
// （edit/apply/convert）、验证（写后回读）；失败操作记录但不提交转换；
// 乱序 / 孤儿 callId / 重复 callId 容错。零绑定面：全部消费既有 wire 形状
// （WireTool / store.Item / JournalChangeRecord / DeliverableEntry），不发明字段。
//
// 本模块四节：
//   §1 工具分类与路径提取（对齐 internal/gaea/evidence/evidence.go 的
//      isWriterTool / isProducerTool / ExtractDeliverablePaths 三口径）；
//   §2 回合投影 reducer（projectOfficeTurn + wire/items 两个适配器）；
//   §3 预览浮窗语义状态机（写弹读不弹 / 关闭优先 / 意图跨回合 / 终态清理），
//      接线在 App（office 写类工具回执 → openPaneFileOrPreview 置前）；
//   §4 草稿轻量版：draft/ready 判定（首次写盘=草稿、Plan→Apply 批准=就绪，
//      merge/discard 映射既有「保留/回滚」按钮，不新增按钮语义）+ 共享
//      JournalList 拉取薄壳（deliverablesTurn/deliverableStatus 同模式）。

import { app } from "./bridge";
import { WRITE_TOOL_NAMES } from "./changes";
import type { JournalChangeRecord, DeliverableEntry, WireEvent } from "./types";

// ── §1 工具分类与路径提取 ────────────────────────────────────────────────

// 写入类 = 写类 8 种（lib/changes.ts WRITE_TOOL_NAMES，与 Go isWriterTool 一致）
// + 生成/导出类 3 种（Go isProducerTool）+ xlsx_apply（App 层 Plan→Apply 批准
// 动作，证据卡 Tool="xlsx_apply"，internal/app/gaea_xlsx_edit.go）。
export const OFFICE_WRITE_TOOLS: ReadonlySet<string> = new Set([
  ...WRITE_TOOL_NAMES,
  "format_convert",
  "chart_gen",
  "diagram_gen",
  "xlsx_apply",
]);

// 读取类 = lib/changes.ts READ_TOOL_NAMES 白名单（read_file/grep/vision/
// format_convert）+ ls（列目录，生命周期「发现」语义）。format_convert 同时在
// 写入集合里：分类以写入优先（它落盘产物），其读取侧路径不参与回读判定。
export const OFFICE_READ_TOOLS: ReadonlySet<string> = new Set([
  "read_file",
  "grep",
  "vision",
  "format_convert",
  "ls",
]);

// Office 文档扩展名（草稿/就绪徽标与预览置前的适用范围；对齐
// DELIVERABLE_EXT_RE 的办公文档子集 + 国产格式 wps/et/dps/ofd）。
export const OFFICE_DOC_EXT_RE =
  /\.(docx?|xlsx?|xlsm|pptx?|pdf|csv|et|ods|odp|odt|rtf|wps|dps|ofd)$/i;

export function isOfficeDeliverablePath(path: string): boolean {
  return OFFICE_DOC_EXT_RE.test(path);
}

function parseArgs(args: string | undefined): Record<string, unknown> {
  if (!args) return {};
  try {
    const parsed = JSON.parse(args) as unknown;
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {};
  } catch {
    return {};
  }
}

function pushPath(out: string[], seen: Set<string>, v: unknown): void {
  if (typeof v !== "string") return;
  const p = v.trim();
  if (p === "" || seen.has(p)) return;
  seen.add(p);
  out.push(p);
}

function pushArrayKey(parsed: Record<string, unknown>, key: string, out: string[], seen: Set<string>): void {
  const arr = parsed[key];
  if (Array.isArray(arr)) for (const v of arr) pushPath(out, seen, v);
}

/**
 * 写入类工具的落盘产物路径（对齐 Go ExtractDeliverablePaths 两分支）：
 * 生成/导出类（format_convert/chart_gen/diagram_gen）只取 output——path 在
 * 这三类工具里是输入源文件，不是交付物；写类取 path/file_path/notebook_path/
 * destination 单值 + paths/file_paths 数组 + edits[].path/file_path。
 * 去重保持出现顺序。
 */
export function extractOfficeWritePaths(tool: string, args: string | undefined): string[] {
  const parsed = parseArgs(args);
  const out: string[] = [];
  const seen = new Set<string>();
  if (tool === "format_convert" || tool === "chart_gen" || tool === "diagram_gen") {
    pushPath(out, seen, parsed.output);
    return out;
  }
  for (const key of ["path", "file_path", "notebook_path", "destination"]) {
    pushPath(out, seen, parsed[key]);
  }
  pushArrayKey(parsed, "paths", out, seen);
  pushArrayKey(parsed, "file_paths", out, seen);
  if (Array.isArray(parsed.edits)) {
    for (const edit of parsed.edits as unknown[]) {
      if (!edit || typeof edit !== "object" || Array.isArray(edit)) continue;
      const e = edit as Record<string, unknown>;
      pushPath(out, seen, e.path);
      pushPath(out, seen, e.file_path);
    }
  }
  return out;
}

/** 读取类工具的目标路径（对齐 lib/changes.ts 读取键候选：source/image_path）。 */
export function extractOfficeReadPaths(args: string | undefined): string[] {
  const parsed = parseArgs(args);
  const out: string[] = [];
  const seen = new Set<string>();
  for (const key of ["path", "file_path", "source", "image_path"]) {
    pushPath(out, seen, parsed[key]);
  }
  pushArrayKey(parsed, "paths", out, seen);
  pushArrayKey(parsed, "file_paths", out, seen);
  return out;
}

// ── §2 回合投影（三类操作独立归约）──────────────────────────────────────

/** 归一化的工具事件：dispatch=发起 / result=回执。callId = WireTool.id / Item.id。 */
export interface OfficeToolEvent {
  kind: "dispatch" | "result";
  callId: string;
  tool: string;
  args?: string;
  output?: string;
  err?: string;
}

export type OfficeOpKind = "write" | "verify" | "lifecycle";

export interface OfficeCallPair {
  callId: string;
  tool: string;
  kind: OfficeOpKind;
  /** 该调用的目标路径（写入=产物路径；读取=读取目标；提取失败=空数组）。 */
  paths: string[];
  /** pending=只有 dispatch；ok/error 以 result err 字段为准。 */
  status: "pending" | "ok" | "error";
  /** 失败原因（error result 原文保留；成功/挂起为 undefined）。 */
  err?: string;
  /** 有 result 但整批中找不到同 callId 的 dispatch（乱序兜底仍计数）。 */
  orphan: boolean;
  /** 同 callId 的重复 dispatch / 重复 result（后者忽略，取首个）。 */
  duplicate: boolean;
}

/** 逐文件回合视图：写入只标 changed 与失败原因，不提交任何终态转换。 */
export interface OfficeFileTurnView {
  path: string;
  /** 成功写入次数（失败不计入）。 */
  writes: number;
  /** 失败写入次数。 */
  failedWrites: number;
  lastWriteTool?: string;
  /** 最后一次失败写入的原因（失败保留原因，不提交转换）。 */
  lastWriteErr?: string;
  /** 写后回读（验证）次数：只有落在成功写入之后的读取才计数。 */
  readBacks: number;
  /** 生命周期读取次数（未被写过的路径）。 */
  reads: number;
  /** 最近一次成功写入之后有过成功回读，且此后无失败写入。 */
  verified: boolean;
}

export interface OfficeTurnProjection {
  /** 按首现顺序（dispatch 或 result 先到者）排列的配对结果。 */
  calls: OfficeCallPair[];
  /** 三类操作视图（与 calls 同元素引用，按类过滤）。 */
  writes: OfficeCallPair[];
  verifies: OfficeCallPair[];
  lifecycle: OfficeCallPair[];
  /** 逐文件聚合（按首写顺序；只含有写入或读取痕迹的文件）。 */
  files: OfficeFileTurnView[];
  /** 本回合存在成功写入。 */
  changed: boolean;
  /** 所有被成功写入的文件都得到写后回读（verify 循环口径）。 */
  verifiedAll: boolean;
  /** 存在失败调用（原因保留在 calls[].err / files[].lastWriteErr）。 */
  hasFailure: boolean;
}

interface CallBucket {
  id: string;
  firstIndex: number;
  tool: string;
  dispatch?: { args?: string; duplicate: boolean };
  result?: { output?: string; err?: string; duplicate: boolean; orphan: boolean };
}

function classifyOp(tool: string): "write" | "read" {
  return OFFICE_WRITE_TOOLS.has(tool) ? "write" : "read";
}

/**
 * 把一个回合内的工具事件流归约为结构化 Office 回合投影。
 * 配对规则：callId 相同的 dispatch 与 result 归并；result 先到（乱序）照样
 * 配对；孤儿 result（整批无 dispatch）标记 orphan 后照常消费；重复
 * dispatch / result 只取首个并标记 duplicate。失败（err 非空）只记录，
 * 不推进任何文件的写入/验证状态。
 */
export function projectOfficeTurn(events: readonly OfficeToolEvent[]): OfficeTurnProjection {
  const buckets = new Map<string, CallBucket>();
  for (let i = 0; i < events.length; i++) {
    const ev = events[i];
    if (!ev || typeof ev.callId !== "string" || ev.callId === "") continue;
    if (typeof ev.tool !== "string" || ev.tool === "") continue;
    let b = buckets.get(ev.callId);
    if (!b) {
      b = { id: ev.callId, firstIndex: i, tool: ev.tool };
      buckets.set(ev.callId, b);
    }
    if (ev.kind === "dispatch") {
      if (b.dispatch) b.dispatch.duplicate = true;
      else b.dispatch = { args: ev.args, duplicate: false };
    } else {
      if (b.result) b.result.duplicate = true;
      else b.result = { output: ev.output, err: ev.err, duplicate: false, orphan: false };
    }
  }
  // 孤儿判定需要全量 dispatch 集合，二遍标记。
  for (const b of buckets.values()) {
    if (b.result && !b.dispatch) b.result.orphan = true;
  }

  const calls: OfficeCallPair[] = [];
  const files = new Map<string, OfficeFileTurnView>();
  // 已成功写入的路径：读取命中即升级为「验证（回读）」。
  const writtenPaths = new Set<string>();
  // 每文件「待回读」标记：成功写→true，成功回读→false。
  const pendingVerify = new Map<string, boolean>();

  const fileView = (path: string): OfficeFileTurnView => {
    let v = files.get(path);
    if (!v) {
      v = { path, writes: 0, failedWrites: 0, readBacks: 0, reads: 0, verified: false };
      files.set(path, v);
    }
    return v;
  };

  const ordered = [...buckets.values()].sort((a, b) => a.firstIndex - b.firstIndex);
  for (const b of ordered) {
    const isWrite = classifyOp(b.tool) === "write";
    const paths = isWrite
      ? extractOfficeWritePaths(b.tool, b.dispatch?.args)
      : extractOfficeReadPaths(b.dispatch?.args);
    const status = b.result ? (b.result.err ? "error" : "ok") : "pending";
    const pair: OfficeCallPair = {
      callId: b.id,
      tool: b.tool,
      kind: "lifecycle",
      paths,
      status,
      err: b.result?.err || undefined,
      orphan: b.result?.orphan ?? false,
      duplicate: (b.dispatch?.duplicate ?? false) || (b.result?.duplicate ?? false),
    };
    // 分类：写入恒为 write；读取按「是否命中已成功写入的路径」升级为 verify
    //（写后回读），否则 lifecycle。读取路径与写入路径都用原串（同一工具
    // 参数口径），不做大小写归一——verify 判定宁缺勿滥。
    if (isWrite) {
      pair.kind = "write";
      if (status === "ok") {
        for (const p of paths) {
          writtenPaths.add(p);
          pendingVerify.set(p, true);
          const v = fileView(p);
          v.writes += 1;
          v.lastWriteTool = b.tool;
          v.lastWriteErr = undefined;
          v.verified = false; // 新写入使既有回读结论失效（诚实口径：最后一次写之后才算）
        }
      } else if (status === "error") {
        for (const p of paths) {
          const v = fileView(p);
          v.failedWrites += 1;
          v.lastWriteErr = pair.err;
        }
      }
    } else if (status === "ok" && paths.some((p) => writtenPaths.has(p))) {
      pair.kind = "verify";
      for (const p of paths) {
        if (!writtenPaths.has(p)) continue;
        const v = fileView(p);
        v.readBacks += 1;
        if (pendingVerify.get(p) === true) {
          pendingVerify.set(p, false);
          v.verified = true;
        }
      }
    } else {
      pair.kind = "lifecycle";
      if (status === "ok") {
        for (const p of paths) fileView(p).reads += 1;
      }
    }
    calls.push(pair);
  }

  const writes = calls.filter((c) => c.kind === "write");
  const verifies = calls.filter((c) => c.kind === "verify");
  const lifecycle = calls.filter((c) => c.kind === "lifecycle");
  const fileViews = [...files.values()];
  return {
    calls,
    writes,
    verifies,
    lifecycle,
    files: fileViews,
    changed: writes.some((c) => c.status === "ok"),
    verifiedAll:
      writes.some((c) => c.status === "ok") &&
      fileViews.every((v) => v.writes === 0 || v.verified),
    hasFailure: calls.some((c) => c.status === "error"),
  };
}

/** wire 事件适配器：WireEvent（tool_dispatch/tool_result）→ 归一化事件。 */
export function officeToolEventsFromWire(
  events: readonly Pick<WireEvent, "kind" | "tool">[],
): OfficeToolEvent[] {
  const out: OfficeToolEvent[] = [];
  for (const e of events) {
    const t = e.tool;
    if (!t || typeof t.name !== "string" || t.name === "") continue;
    // partial=true 是只带 name 的早期预告，随后必有完整 dispatch——跳过防抖。
    if (e.kind === "tool_dispatch") {
      if (t.partial) continue;
      out.push({ kind: "dispatch", callId: t.id ?? t.name, tool: t.name, args: t.args });
    } else if (e.kind === "tool_result") {
      out.push({
        kind: "result",
        callId: t.id ?? t.name,
        tool: t.name,
        output: t.output,
        err: t.err,
      });
    }
  }
  return out;
}

/** store 条目适配器：Item（tool 卡：running/done/error/stopped）→ 归一化事件。
 *  入参用宽容形状（Item 联合的其余成员因缺 name 被跳过），免引入 store 类型。 */
export function officeToolEventsFromItems(
  items: readonly { kind: unknown; id?: string; name?: string; args?: string; status?: string; output?: string; error?: string }[],
): OfficeToolEvent[] {
  const out: OfficeToolEvent[] = [];
  for (const it of items) {
    if (!it || it.kind !== "tool" || typeof it.name !== "string" || !it.id) continue;
    out.push({ kind: "dispatch", callId: it.id, tool: it.name, args: it.args });
    if (it.status === "done") {
      out.push({ kind: "result", callId: it.id, tool: it.name, output: it.output });
    } else if (it.status === "error") {
      out.push({ kind: "result", callId: it.id, tool: it.name, err: it.error ?? "error" });
    }
    // running / stopped：只出 dispatch（stopped=中断无回执，保持 pending）。
  }
  return out;
}

// ── §3 预览浮窗语义状态机（纯函数，接线在 App）─────────────────────────
// 蒸馏上游 §1.6 浮窗拉起规则：写入类操作才主动置前；纯读取永不拉起已关闭
// 浮窗；用户手动关闭优先并清除打开意图；写意图未完成时跨回合保持；同文件
// 同回合至多自动置前一次；回合终结清理回合级状态。

export interface PreviewAutoFrontState {
  /** 待兑现的写意图（写类 dispatch 设置，回执/关闭/新意图替换时清除；跨回合保持）。 */
  intentPath: string | null;
  /** 本回合被用户手动关闭的路径：读取永不复活（终态清理复位）。 */
  closedPaths: ReadonlySet<string>;
  /** 本回合已自动置前过的路径（同文件同回合至多一次）。 */
  openedThisTurn: ReadonlySet<string>;
}

export type PreviewAutoFrontEvent =
  | { type: "writeDispatch"; path: string }
  | { type: "writeResult"; path: string; ok: boolean }
  | { type: "read"; path: string }
  | { type: "userClose"; path: string }
  | { type: "turnEnd" };

export type PreviewAutoFrontAction =
  | { type: "open"; path: string }
  | { type: "none" };

export const initialPreviewAutoFrontState: PreviewAutoFrontState = {
  intentPath: null,
  closedPaths: new Set<string>(),
  openedThisTurn: new Set<string>(),
};

function withSets(
  state: PreviewAutoFrontState,
  patch: { intentPath?: string | null; closedPaths?: ReadonlySet<string>; openedThisTurn?: ReadonlySet<string> },
): PreviewAutoFrontState {
  return {
    intentPath: patch.intentPath !== undefined ? patch.intentPath : state.intentPath,
    closedPaths: patch.closedPaths ?? state.closedPaths,
    openedThisTurn: patch.openedThisTurn ?? state.openedThisTurn,
  };
}

/**
 * 浮窗语义状态机迁移表：
 *   writeDispatch(path) → intent=path；path 移出手动关闭集（新写重新武装）
 *   writeResult(path, ok) → 有 intent 且匹配：ok →（同回合未弹过）open+记已弹、
 *     清 intent；!ok → 只清 intent（失败不弹）。无 intent（已被关闭取消）→ 不动。
 *   read(path) → 恒 no-op（读取永不弹，已关闭浮窗不复活）。
 *   userClose(path) → 记入关闭集；若 intent 命中则清除（关闭优先）。
 *   turnEnd → 清空关闭集与已弹集（终态清理复位）；intent 保留（写意图跨回合）。
 */
export function previewAutoFrontReduce(
  state: PreviewAutoFrontState,
  event: PreviewAutoFrontEvent,
): { state: PreviewAutoFrontState; action: PreviewAutoFrontAction } {
  switch (event.type) {
    case "writeDispatch": {
      if (!event.path) return { state, action: { type: "none" } };
      const closed = new Set(state.closedPaths);
      closed.delete(event.path);
      return {
        state: withSets(state, { intentPath: event.path, closedPaths: closed }),
        action: { type: "none" },
      };
    }
    case "writeResult": {
      if (state.intentPath !== event.path) return { state, action: { type: "none" } };
      if (!event.ok) {
        return { state: withSets(state, { intentPath: null }), action: { type: "none" } };
      }
      if (state.openedThisTurn.has(event.path)) {
        return { state: withSets(state, { intentPath: null }), action: { type: "none" } };
      }
      const opened = new Set(state.openedThisTurn);
      opened.add(event.path);
      return {
        state: withSets(state, { intentPath: null, openedThisTurn: opened }),
        action: { type: "open", path: event.path },
      };
    }
    case "read":
      return { state, action: { type: "none" } };
    case "userClose": {
      const closed = new Set(state.closedPaths);
      closed.add(event.path);
      return {
        state: withSets(state, {
          closedPaths: closed,
          intentPath: state.intentPath === event.path ? null : state.intentPath,
        }),
        action: { type: "none" },
      };
    }
    case "turnEnd":
      return {
        state: withSets(state, { closedPaths: new Set<string>(), openedThisTurn: new Set<string>() }),
        action: { type: "none" },
      };
    default:
      return { state, action: { type: "none" } };
  }
}

// ── §4 草稿轻量版：draft/ready 判定 + Journal 薄壳 ───────────────────────
// 口径（规划 §4.2-2 + 风险表拍板项 2，写死防撞车）：首次写盘=草稿、批准=就绪。
//   ready  ← 证据链里该文件存在 xlsx_apply（Plan→Apply 批准动作）且未 failed；
//   draft  ← 证据链/登记表里有任何写盘痕迹（未走批准动作）；
//   null   ← 非 Office 文档，或毫无写盘证据（宁缺勿误，不标）。
// merge/discard 不新增按钮：保留=既有验收「标记已验收」，回滚=既有证据卡
// 「回滚」/时间线「恢复」，本模块只产出徽标状态。

export type OfficeDeliverablePhase = "draft" | "ready";

const XLSX_APPLY_READY_STATUS = new Set(["applied", "verified"]);

function journalMatchesPath(r: JournalChangeRecord, path: string): boolean {
  return r.target.replace(/\\/g, "/").toLowerCase() === path.replace(/\\/g, "/").toLowerCase();
}

export function deliverablePhaseOf(
  path: string,
  journal: readonly JournalChangeRecord[] | null | undefined,
  registry?: Pick<DeliverableEntry, "tool"> | null,
): OfficeDeliverablePhase | null {
  if (!path || !isOfficeDeliverablePath(path)) return null;
  const recs = journal?.filter((r) => journalMatchesPath(r, path)) ?? [];
  if (recs.some((r) => r.tool === "xlsx_apply" && XLSX_APPLY_READY_STATUS.has(r.status))) {
    return "ready";
  }
  if (recs.length > 0) return "draft";
  if (registry && typeof registry.tool === "string" && registry.tool !== "") return "draft";
  return null;
}

// ── JournalList 共享薄壳（DeliverableCards 徽标数据源；多卡共享一次 fetch，
//    模式与 lib/deliverablesTurn.ts 的登记表缓存一致：2s 去重 + 失败降级）。

const JOURNAL_DEDUPE_MS = 2000;
let journalCache: { at: number; promise: Promise<JournalChangeRecord[] | null> } | null = null;

async function fetchJournal(): Promise<JournalChangeRecord[] | null> {
  try {
    if (typeof app.GaeaJournalList !== "function") return null; // 测试门面未注入 → 降级
    return (await app.GaeaJournalList(200)) ?? null;
  } catch {
    return null; // 失败静默降级：徽标不渲染（宁缺勿误）
  }
}

/** 组件挂载取证据链（草稿/就绪判定源）；in-flight 或 TTL 内复用同一 promise。 */
export function ensureOfficeJournal(now = Date.now()): Promise<JournalChangeRecord[] | null> {
  if (!journalCache || now - journalCache.at > JOURNAL_DEDUPE_MS) {
    journalCache = { at: now, promise: fetchJournal() };
  }
  return journalCache.promise;
}

/** 强制下一次取数重拉（测试隔离 / 会话切换接线预留）。 */
export function invalidateOfficeJournal(): void {
  journalCache = null;
}

// ── §5 写后预览实时跟随（U4 §4.3-2；纯逻辑，接线在 App）─────────────────
// 信号源 = office 写类工具的成功回执（write result ok 且落盘路径为 Office 文档）；
// 消费方 = 预览面「静默重载」（FilePreview 重读 app.Preview：不进 loading、不重挂、
// 不弹窗）。与 §3 浮窗状态机的分工：§3 管「要不要置前」（写弹读不弹/关闭优先/
// 意图跨回合/终态清理），本节管「已打开的预览要不要刷新」——刷新绝不打开任何
// 东西；用户已关闭的文件天然不在打开集里，既不会被刷新也不会被弹（U2 语义零回退）。

/**
 * 预览路径归一（刷新总线与打开判定共用的键口径）：反斜杠→斜杠 + 小写——
 * Windows 路径大小写不敏感，写类工具参数里的路径与 pane tab 里登记的路径
 * 可能大小写/分隔符不同；与 deliverablePhaseOf 的 journalMatchesPath 同式。
 */
export function normalizePreviewPath(path: string): string {
  return (path ?? "").trim().replace(/\\/g, "/").toLowerCase();
}

/**
 * 写类工具回执 → 「文件已更新」信号路径集（本回合信号派生的唯一口径）：
 * 失败回执不派（内容未变，刷新了也是旧内容）；非 office 写类工具不派
 * （bash/脚本写盘不打扰，与 §3 的置前范围同口径）；路径取
 * extractOfficeWritePaths 的产物口径并过滤到 Office 文档扩展名。
 */
export function officeUpdatedPathsFromResult(
  tool: string,
  args: string | undefined,
  ok: boolean,
): string[] {
  if (!ok) return [];
  if (!OFFICE_WRITE_TOOLS.has(tool)) return [];
  return extractOfficeWritePaths(tool, args).filter(isOfficeDeliverablePath);
}

export interface PreviewRefreshDeps {
  /** 目标文件此刻是否正打开在预览面（pane 活动文件 tab / 主区大预览）。 */
  isOpen: (path: string) => boolean;
  /** 执行刷新（App：paneTabs reloadTicks 递增 → FilePreview 静默重读）。 */
  refresh: (path: string) => void;
  /** 计时器注入（缺省 window.setTimeout/clearTimeout；测试注入虚拟时钟）。 */
  setTimer?: (fn: () => void, ms: number) => unknown;
  clearTimer?: (id: unknown) => void;
  /** 防抖窗口毫秒（缺省 800：合并 agent 连写，避免逐次重渲染抖动）。 */
  delayMs?: number;
}

export interface PreviewRefreshScheduler {
  /** 写类工具成功回执时逐路径喂入；同路径窗口内重复喂入合并为一次刷新。 */
  notify: (path: string) => void;
  /** 取消某路径的待刷新（会话切换/卸载卫生）。 */
  cancel: (path: string) => void;
  /** 取消全部待刷新（会话切换）。 */
  cancelAll: () => void;
  /** 测试/断言辅助：该路径当前是否有挂起的防抖计时。 */
  pending: (path: string) => boolean;
}

/**
 * 写后预览刷新调度器（防抖合并 + 到点时打开判定，纯逻辑不碰 React/store）：
 *   notify(path) → 重置该路径计时（连写合并为最后一次）；到点时再次
 *   isOpen(path) 判定——窗口内用户关闭 → 不刷新（关闭抑制，关闭优先语义）；
 *   窗口内用户（重新）打开 → 刷新（对新鲜挂载只是多一次幂等的静默重读）。
 *   isOpen/refresh 由 App 注入（App 判 pane 活动文件 tab / 主区大预览并递增
 *   reloadTicks 总线），单测用虚拟时钟钉死触发/合并/抑制三语义。
 */
export function createPreviewRefreshScheduler(deps: PreviewRefreshDeps): PreviewRefreshScheduler {
  const delayMs = deps.delayMs ?? 800;
  const setTimer = deps.setTimer ?? ((fn: () => void, ms: number) => window.setTimeout(fn, ms));
  const clearTimer = deps.clearTimer ?? ((id: unknown) => window.clearTimeout(id as number));
  const timers = new Map<string, unknown>();
  const drop = (key: string) => {
    const id = timers.get(key);
    if (id !== undefined) {
      clearTimer(id);
      timers.delete(key);
    }
  };
  return {
    notify(path) {
      const key = normalizePreviewPath(path);
      if (!key) return;
      drop(key);
      timers.set(
        key,
        setTimer(() => {
          timers.delete(key);
          if (deps.isOpen(path)) deps.refresh(path);
        }, delayMs),
      );
    },
    cancel(path) {
      drop(normalizePreviewPath(path));
    },
    cancelAll() {
      for (const key of [...timers.keys()]) drop(key);
    },
    pending(path) {
      return timers.has(normalizePreviewPath(path));
    },
  };
}
