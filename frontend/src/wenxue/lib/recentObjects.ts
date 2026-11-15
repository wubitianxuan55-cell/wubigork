// 最近工作对象（localStorage 单源）——首页「继续」区跨板块续做（2026-10-26 二期）。
//
// Why: 首页方向一「继续」一期只有文档+会话两类；小说书与进度工程是更高层的
// 「工作对象」（打开即进入一段工作现场），但它们的打开动作分属各板块（书架
// handleOpen / schedule store switchProject），此前没有统一记账。仿 recentFiles
// 的单源模式收敛：各板块打开对象时 record，首页订阅展示与恢复。
//
// How: `recordRecentObject(kind, key, title)`——kind+key 为身份（novel=书路径、
// schedule=工程 rel），去重置顶；每 kind 留最近 3 条、总量 12，防单一板块刷屏。
// 写入即广播 `RECENT_OBJECTS_EVENT`；load 带解析缓存（快照引用稳定，满足
// useSyncExternalStore 的 getSnapshot 要求）。storage 读写全部 fail-safe：记账
// 是增强不是主流程，配额/隐私模式失败不得影响打开动作本身。
// cost 板块不参与：其「选中项目」是组件本地筛选态（WorkcostBillView selected），
// 无恢复语义，硬造对象行是伪续做。

export type RecentObjectKind = 'novel' | 'schedule'

export interface RecentObject {
  kind: RecentObjectKind
  /** 身份键：novel=项目路径，schedule=工程 rel */
  key: string
  /** 展示名：书名 / 工程名 */
  title: string
  /** 记账时刻（unix ms） */
  ts: number
}

const RECENT_OBJECTS_KEY = 'gaea.atRecentObjects'
/** 变化广播事件（同窗口写入即通知；storage 事件只跨窗口，覆盖不到本场景） */
const RECENT_OBJECTS_EVENT = 'gaea:recent-objects'
const MAX_PER_KIND = 3
const MAX_TOTAL = 12

/** 解析缓存：同一份快照引用稳定（useSyncExternalStore 要求）；写入/清空即失效。 */
let cache: RecentObject[] | null = null

function readFromStorage(): RecentObject[] {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENT_OBJECTS_KEY) || '[]')
    if (!Array.isArray(raw)) return []
    return raw.filter((e): e is RecentObject =>
      !!e && typeof e.kind === 'string' && typeof e.key === 'string'
        && typeof e.title === 'string' && typeof e.ts === 'number',
    )
  } catch {
    return []
  }
}

/** 读取最近工作对象（按时间倒序；损坏/空返回 []）。 */
export function loadRecentObjects(): RecentObject[] {
  if (!cache) cache = readFromStorage()
  return cache
}

/** 订阅最近对象变化（返回取消订阅函数）。 */
export function subscribeRecentObjects(cb: () => void): () => void {
  if (typeof window === 'undefined') return () => {}
  window.addEventListener(RECENT_OBJECTS_EVENT, cb)
  return () => window.removeEventListener(RECENT_OBJECTS_EVENT, cb)
}

function emit(): void {
  try { window.dispatchEvent(new Event(RECENT_OBJECTS_EVENT)) } catch { /* 非浏览器环境忽略 */ }
}

/** 每 kind 只留最近 MAX_PER_KIND 条，总量 MAX_TOTAL（防单一板块刷屏挤掉其他）。 */
function capKinds(list: RecentObject[]): RecentObject[] {
  const seen = new Map<RecentObjectKind, number>()
  return list
    .filter((o) => {
      const n = (seen.get(o.kind) ?? 0) + 1
      seen.set(o.kind, n)
      return n <= MAX_PER_KIND
    })
    .slice(0, MAX_TOTAL)
}

/** 记录一次「打开了某工作对象」：kind+key 去重置顶（幂等），落盘 + 广播。 */
export function recordRecentObject(kind: RecentObjectKind, key: string, title: string): void {
  if (!key) return
  const next = capKinds([
    { kind, key, title: title || key, ts: Date.now() },
    ...loadRecentObjects().filter((o) => !(o.kind === kind && o.key === key)),
  ])
  try {
    localStorage.setItem(RECENT_OBJECTS_KEY, JSON.stringify(next))
  } catch {
    // 配额/隐私模式等：静默失败，不影响打开动作主流程
    return
  }
  cache = next
  emit()
}

/** 测试辅助：清空最近对象（vitest 隔离用例间状态）。 */
export function clearRecentObjectsForTest(): void {
  try { localStorage.removeItem(RECENT_OBJECTS_KEY) } catch { /* ignore */ }
  cache = null
}
