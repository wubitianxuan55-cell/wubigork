// v4.26「对话流式重造」事件序号防线（前端状态线）。
//
// Why：Wails gaea-event 事件流密集到达时会丢件（store.ts 旧实现只有 turn_done
// 兜底），中途 text/tool_result 丢失导致对话窗静默；而轨迹面板读磁盘日志不受
// 害——数据在后端是完整的，只差把丢掉的部分补回前端。后端线为此给 gaea-event
// payload 附加会话内单调递增 seq，并提供新绑定 GaeaResyncEvents(afterSeq) 返回
// { seq, items }（items 为磁盘日志折叠出的前端 Item 视图 JSON）。
//
// How：本模块三层，互不掺杂——
//   1) createEventSyncTracker：纯序号簿记（首件不判缺 / 乱序回退忽略 / 跳号报缺口）；
//   2) shouldResync：纯函数冷却门（gap≥1 即补，但 5s 内只触发一次，防补拉风暴）；
//   3) noteEventSeq：事件→补拉编排（store.ts onEvent 入口调用），补拉调用经
//      setEventSyncFetcher 注入（App.tsx 挂真实 bridge；null=旁路）——本文件
//      不 import bridge 真名（bridge.ts 的 AppBindings 尚无该方法，且按约定
//      由主代理接线），测试用 mock fetcher。
// 与 reducer 完全解耦：本模块只产出 EventSyncSnapshot，由调用方转成 store 的
// resync action 落库（形状校验在 store.parseResyncItems）。

/** GaeaResyncEvents 的返回快照：seq=快照覆盖到的最后一条事件序号；items=折叠视图。 */
export interface EventSyncSnapshot {
  seq: number;
  items: unknown[];
}

/** 补拉取数函数：afterSeq=前端已见到的最新事件序号；null=未接线（旁路）。 */
export type EventSyncFetcher = (afterSeq: number) => Promise<EventSyncSnapshot>;

// fetcher 注入点：App.tsx 挂真实 bridge（app.GaeaResyncEvents）后整个防线生效；
// 不挂（null）则缺口只记不拉，行为与 v4.26 前一致。
let fetcher: EventSyncFetcher | null = null;

export function setEventSyncFetcher(fn: EventSyncFetcher | null): void {
  fetcher = fn;
}

export function getEventSyncFetcher(): EventSyncFetcher | null {
  return fetcher;
}

// ── 1) 序号簿记 ────────────────────────────────────────────────

export interface EventSyncTracker {
  /** 喂入一条事件序号，返回缺失数（0=无缺口）。 */
  feed(seq: number): number;
  /** 会话切换 / 新会话：seq 基线归零（新会话从 1 重新单调递增）。 */
  reset(): void;
  /** 已见到的最新序号（null=尚无基线）；作 fetcher 的 afterSeq。 */
  lastSeq(): number | null;
}

export function createEventSyncTracker(): EventSyncTracker {
  let last: number | null = null;
  return {
    feed(seq: number): number {
      if (!Number.isFinite(seq)) return 0; // 防御：脏序号不建立基线也不误报缺口
      if (last === null) {
        // 首件不判缺：订阅前可能已丢件，无从知晓缺了多少，以首件为基线。
        last = seq;
        return 0;
      }
      if (seq <= last) return 0; // 乱序回退 / 重复投递：忽略，不回退基线
      const gap = seq - last - 1;
      last = seq;
      return gap;
    },
    reset(): void {
      last = null;
    },
    lastSeq(): number | null {
      return last;
    },
  };
}

// ── 2) 冷却门（纯函数）────────────────────────────────────────

/** gap≥1 即补：流式丢 1 个 chunk 也是用户可见的静默缺口。 */
export const RESYNC_MIN_GAP = 1;
/** 5s 内只触发一次：密集丢件时避免补拉风暴（在途补拉落地后仍可再触发）。 */
export const RESYNC_COOLDOWN_MS = 5000;

/** 冷却门判定：缺口达标且距上次补拉已过冷却窗口。不修改任何状态（纯函数）。 */
export function shouldResync(gap: number, nowMs: number, lastResyncAtMs: number): boolean {
  if (!Number.isFinite(gap) || gap < RESYNC_MIN_GAP) return false;
  return nowMs - lastResyncAtMs >= RESYNC_COOLDOWN_MS;
}

// ── 3) 事件→补拉编排（store.ts onEvent 入口调用）──────────────

export interface EventSeqOutcome {
  gap: number; // 本次喂入检出的缺失数（0=无缺口/无 seq/乱序）
  resync: boolean; // 是否真正触发了一次补拉
}

export interface NoteEventSeqOpts {
  /** 可注入时钟（冷却判定测试用）；缺省 Date.now()。 */
  nowMs?: number;
  /** 补拉成功回调：调用方转成 store resync action。 */
  onSnapshot: (snap: EventSyncSnapshot) => void;
  /** 补拉失败回调（防线不崩，等冷却后重试）。 */
  onError?: (err: unknown) => void;
}

// 模块级单例：整个应用只有一条 gaea-event 通道，tracker / 冷却时刻 / 在途标记
// 与之同生命周期；会话切换经 resetEventSync() 归零（由 store.ts 的会话切换
// 回调调用，reducer 保持纯函数不动模块态）。
let tracker = createEventSyncTracker();
// -Infinity=从未补拉：首个缺口在任何时钟下都视为冷却窗外（0 基线会在时钟
// 读数小于冷却窗口时把首个缺口误判为「冷却中」）。
let lastResyncAt = Number.NEGATIVE_INFINITY;
let resyncInFlight = false;

/** 会话切换 / 新会话时归零序号基线与冷却态（store.ts 各 reset 调用点接线）。 */
export function resetEventSync(): void {
  tracker = createEventSyncTracker();
  lastResyncAt = Number.NEGATIVE_INFINITY;
  resyncInFlight = false;
}

/** 从事件 payload 提取可选 seq：非有限非负数一律视为「无 seq」（旧后端旁路）。 */
export function extractSeq(payload: unknown): number | null {
  if (payload === null || typeof payload !== "object") return null;
  const raw = (payload as { seq?: unknown }).seq;
  if (typeof raw !== "number" || !Number.isFinite(raw) || raw < 0) return null;
  return Math.floor(raw);
}

/**
 * noteEventSeq 事件序号防线入口：payload 带 seq 时做缺口检测，命中缺口经注入
 * 的 fetcher 异步补拉（在途去重 + 5s 冷却），快照经 onSnapshot 交回调落库。
 * 旧后端无 seq / 未挂 fetcher 时整条防线静默旁路，零行为差异。
 */
export function noteEventSeq(payload: unknown, opts: NoteEventSeqOpts): EventSeqOutcome {
  const seq = extractSeq(payload);
  if (seq === null) return { gap: 0, resync: false }; // 旧后端：无 seq，静默旁路
  const gap = tracker.feed(seq);
  if (gap <= 0) return { gap: 0, resync: false };
  const nowMs = opts.nowMs ?? Date.now();
  if (resyncInFlight) return { gap, resync: false }; // 在途补拉读的是更新日志，等它落地
  if (!getEventSyncFetcher()) return { gap, resync: false }; // 未接线：缺口只记不拉
  if (!shouldResync(gap, nowMs, lastResyncAt)) return { gap, resync: false };
  lastResyncAt = nowMs;
  resyncInFlight = true;
  const afterSeq = tracker.lastSeq() ?? seq;
  void Promise.resolve()
    .then(() => getEventSyncFetcher()?.(afterSeq))
    .then((snap) => {
      resyncInFlight = false;
      if (snap) opts.onSnapshot(snap);
    })
    .catch((err: unknown) => {
      resyncInFlight = false;
      opts.onError?.(err);
    });
  return { gap, resync: true };
}
