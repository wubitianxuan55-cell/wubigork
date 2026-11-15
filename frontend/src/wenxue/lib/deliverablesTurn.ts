// deliverablesTurn — 消息尾部交付卡（DeliverableCards）的「本轮登记表」数据层。
//
// 背景：deliverableMentions(text) 从正文扫文件引用是启发式，工具写出但正文
// 未提及的路径会漏卡。后端有权威产物登记表（internal/gaea/trajectory/
// deliverable.go FoldDeliverables，经 app.DeliverableRegistry(sessionPath) 暴露），
// 这里把「本轮」的登记条目并进卡片列表。
//
// 本模块三件事：
//   1. 轮次口径映射（后端 Turn 1-based ↔ 前端 turnNo 0-based，见 backendTurnOf）；
//   2. 纯逻辑：登记条目按轮筛选 + 与正文卡合并去重 + 缺失态判定（可单测）；
//   3. 模块级共享缓存 hook：多张卡共享一次 fetch（store.ts 契约禁改，故用
//      模块级缓存而非入库）；失败静默降级为「只有正文卡」的现状。

import { useEffect, useMemo, useState } from "react";
import { app } from "./bridge";
import type { DeliverableEntry, DeliverableRegistryView, SessionMeta } from "./types";

// ── 轮次口径映射（依据 Go 折叠代码，勿凭感觉改）────────────────────────
// 后端：deliverable.go 的折叠器 f.turn 初始为 0，每遇一条 turn_started 事件
// 加一（apply 的 "turn_started" 分支）；而会话日志「每条 user 消息前写一条
// turn_started」（internal/gaea/agent/session/log.go:582 注释、:645 写入）。
// 黄金测试 deliverable_test.go 钉死：首条 user 轮内写入 → Turn=1，第二条
// user 轮 → Turn=2；无 turn_started 覆盖的写入 → Turn=0（「轮外」）。
// 前端：Transcript 的 turnNo 是 0-based 用户消息序号（turnNos map，首条
// 用户消息 → 0）。
// 因此：backendTurn = turnNo + 1（turnNo=0 → 后端 Turn=1）；Turn=0（轮外）
// 不与任何 turnNo 匹配（backendTurnOf 恒 ≥1，天然排除）。
export function backendTurnOf(turnNo: number): number {
  return turnNo + 1;
}

// turnTailSegs：turnNos（segIdx → 轮号）中每个轮「最后一段」的 segIdx 集合。
// 同一轮会被 alternatingSegments 拆成多段（文本 ↔ 过程卡交替），登记-only 卡
// 只在轮尾段合并渲染——同轮多条回复重复挂同一批「未提及交付卡」是噪声；
// 正文启发式卡不受影响，每段照常。
export function turnTailSegs(turnNos: Map<number, number>): Set<number> {
  const lastByTurn = new Map<number, number>();
  for (const [idx, turn] of turnNos) lastByTurn.set(turn, idx);
  return new Set(lastByTurn.values());
}

// ── 纯逻辑：按轮筛选 / 合并去重 / 缺失态 ─────────────────────────────
// 去重键：Windows 反斜杠统一为 /、大小写不敏感（Windows 路径语义）。
// 正文写法与登记路径（工具参数原样）常有 \ / 与大小写差异，按原串比较会漏判重复。
export function deliverablePathKey(path: string): string {
  return path.replace(/\\/g, "/").toLowerCase();
}

// 登记表中属于该轮的条目（保持后端 updatedAt 倒序——权威顺序）。
// turnNo 缺省（轮外段 / 无轮次上下文）→ 空数组：不并登记卡，维持现状。
export function registryEntriesForTurn(
  entries: DeliverableEntry[] | null | undefined,
  turnNo: number | undefined,
): DeliverableEntry[] {
  if (!entries || entries.length === 0 || turnNo == null) return [];
  const turn = backendTurnOf(turnNo);
  return entries.filter((e) => e.turn === turn);
}

export interface DeliverableCardItem {
  path: string;
  /** text=正文启发式命中（保持出现顺序，在前）；registry=登记表本轮-only（在后追加）。 */
  from: "text" | "registry";
}

// 合并：正文卡在前（deliverableMentions 的出现顺序），登记-only 卡按登记序
// 在后追加；按归一化路径键去重（正文已提及的路径不再出登记卡）。
export function mergeDeliverableCards(
  textPaths: string[],
  entries: DeliverableEntry[] | null | undefined,
  turnNo: number | undefined,
): DeliverableCardItem[] {
  const out: DeliverableCardItem[] = [];
  const seen = new Set<string>();
  for (const p of textPaths) {
    const key = deliverablePathKey(p);
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ path: p, from: "text" });
  }
  for (const e of registryEntriesForTurn(entries, turnNo)) {
    const key = deliverablePathKey(e.path);
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ path: e.path, from: "registry" });
  }
  return out;
}

// 父目录（工作区相对）：`exports/sub/a.docx` → `exports/sub`；根级文件 → ""。
// "" 传给 app.ListDir 即列工作区根（GaeaListDir 对 "" 直接用 workspace 根目录）。
export function parentDirOf(path: string): string {
  const norm = path.replace(/\\/g, "/");
  const i = norm.lastIndexOf("/");
  return i < 0 ? "" : norm.slice(0, i);
}

// 缺失态判定：登记口径 = 工具派发即登记、失败不回剔，登记-only 卡可能指向
// 尚不存在（写失败）的文件。对「已列目录核对、确认不存在」的路径标缺失；
// 目录探测失败（null）按存在处理——缺失态只做「确认不存在」，宁漏勿误。
// 返回缺失路径的归一化键集合（与 deliverablePathKey 对齐）。
export function missingRegistryKeys(
  entries: DeliverableEntry[],
  dirListings: Map<string, ReadonlySet<string> | null>,
): Set<string> {
  const out = new Set<string>();
  for (const e of entries) {
    const names = dirListings.get(parentDirOf(e.path));
    if (!names) continue; // 未探测 / 探测失败 → 不标缺失
    const name = e.path.replace(/\\/g, "/").split("/").pop() ?? e.path;
    if (!names.has(name.toLowerCase())) out.add(deliverablePathKey(e.path));
  }
  return out;
}

// ── 模块级共享缓存（多卡共享一次 fetch）────────────────────────────
// 当前会话路径来源与 App.currentSessionPath 同式：ListSessions 列表里找
// current=true 的会话（App 侧：sidebarSessions.find(s => s.current)?.path）。
// cache TTL 只用于合并「同屏多卡同时挂载」的并发请求（挂载即拉取，2s 内的
// 挂载共享同一份），保证新一轮消息挂载时拿到的登记表足够新鲜——登记时机
// 是工具派发，先于正文流式，挂载时刻拉取必含本轮条目。
const FETCH_DEDUPE_MS = 2000;
const DIR_DEDUPE_MS = 2000;

interface CacheRec<T> {
  at: number;
  promise: Promise<T>;
}

let registryCache: CacheRec<DeliverableRegistryView | null> | null = null;
const dirListingsCache = new Map<string, CacheRec<ReadonlySet<string> | null>>();

async function fetchTurnRegistry(): Promise<DeliverableRegistryView | null> {
  try {
    const sessions: SessionMeta[] = (await app.ListSessions()) ?? [];
    const currentPath = sessions.find((s) => s.current)?.path;
    if (!currentPath) return null; // 未保存草稿 / 无当前会话：降级现状
    return (await app.DeliverableRegistry(currentPath)) ?? null;
  } catch {
    return null; // 失败静默降级：只有正文卡（现状行为）
  }
}

// 组件挂载时调用的拉取入口：in-flight 或 TTL 内复用，否则重新拉取。
// 导出供测试注入时序与未来接线（如轮完成后主动失效重取）。
export function ensureTurnRegistry(now = Date.now()): Promise<DeliverableRegistryView | null> {
  if (!registryCache || now - registryCache.at > FETCH_DEDUPE_MS) {
    registryCache = { at: now, promise: fetchTurnRegistry() };
  }
  return registryCache.promise;
}

// 强制下一次挂载重新拉取 + 清空目录探测缓存（测试隔离；轮完成/会话切换
// 接线的预留失效时机——登记与存在性都随会话走，整包失效）。
export function invalidateTurnCaches(): void {
  registryCache = null;
  dirListingsCache.clear();
}

async function fetchDirNames(dir: string): Promise<ReadonlySet<string> | null> {
  try {
    const entries = await app.ListDir(dir);
    return new Set((entries ?? []).map((e) => e.name.toLowerCase()));
  } catch {
    return null; // 探测失败 → 按存在处理（不标缺失）
  }
}

function ensureDirNames(dir: string, now = Date.now()): Promise<ReadonlySet<string> | null> {
  const hit = dirListingsCache.get(dir);
  if (hit && now - hit.at > DIR_DEDUPE_MS) dirListingsCache.delete(dir);
  const fresh = dirListingsCache.get(dir);
  if (fresh) return fresh.promise;
  const rec: CacheRec<ReadonlySet<string> | null> = { at: now, promise: fetchDirNames(dir) };
  dirListingsCache.set(dir, rec);
  return rec.promise;
}

// ── hook：本轮登记条目 + 缺失集合 ───────────────────────────────────
export interface TurnDeliverables {
  /** 本轮登记条目（updatedAt 倒序）；null = 不可用（无会话 / 拉取失败 / 轮外）。 */
  entries: DeliverableEntry[];
  /** 已确认不存在的登记路径（deliverablePathKey 归一化键）。 */
  missing: ReadonlySet<string>;
}

const EMPTY_SET: ReadonlySet<string> = new Set<string>();

export function useTurnDeliverables(turnNo: number | undefined): TurnDeliverables {
  const [registry, setRegistry] = useState<DeliverableRegistryView | null>(null);
  const [missing, setMissing] = useState<ReadonlySet<string>>(EMPTY_SET);

  // 挂载时拉取（模块级共享缓存去重）；turnNo 缺省（轮外）不拉取，零开销维持现状。
  useEffect(() => {
    if (turnNo == null) return;
    let cancelled = false;
    void ensureTurnRegistry().then((reg) => {
      if (!cancelled) setRegistry(reg);
    });
    return () => {
      cancelled = true;
    };
  }, [turnNo]);

  const turnEntries = useMemo(
    () => (registry?.available ? registryEntriesForTurn(registry.entries, turnNo) : []),
    [registry, turnNo],
  );

  // 缺失态探测：登记-only 路径按父目录去重后 ListDir 核对（轻量列目录）；
  // 确认不存在才标缺失。turnEntries 仅在 registry/turnNo 变化时重新计算
  // （useMemo），故本效应每次拉取完成至多跑一次，ListDir 另有共享缓存去重。
  useEffect(() => {
    if (turnEntries.length === 0) {
      setMissing(EMPTY_SET);
      return;
    }
    let cancelled = false;
    const dirs = [...new Set(turnEntries.map((e) => parentDirOf(e.path)))];
    void Promise.all(dirs.map((d) => ensureDirNames(d))).then((listings) => {
      if (cancelled) return;
      const map = new Map<string, ReadonlySet<string> | null>();
      dirs.forEach((d, i) => map.set(d, listings[i]));
      setMissing(missingRegistryKeys(turnEntries, map));
    });
    return () => {
      cancelled = true;
    };
  }, [turnEntries]);

  return { entries: turnEntries, missing };
}
