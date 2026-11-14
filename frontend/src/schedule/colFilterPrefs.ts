/**
 * colFilterPrefs.ts — 横道列级筛选的本机偏好归一（纯函数，W5 列筛持久化）
 *
 * 背景：W5 首版给了列级筛选（`GanttFilter.colFilters`），但状态是会话态。
 * 本模块只做「本机偏好（localStorage）」这一层的**归一 + 序列化 + 签名**，
 * 不碰 `GanttView`、不碰项目文件、不碰 Go schema：
 *  - 斑马把列筛选存在视图里；gaea 的自定义字段表达式未建模，这里**诚实收窄**：
 *    只持久化 `COL_FILTER_KIND` 里已登记的五类条件
 *    （text / number / date / enum + empty 空值口径），其余一律丢弃；
 *  - 视图快照（`ViewSnapshot`）不加列筛字段，避免改 Go 侧 schema。
 *
 * 三个导出（均纯函数、不抛异常、输入脏数据只退化不崩）：
 *  - `normalizeColFilters(raw)`：未知列键/类型冲突/超限项丢弃，条件全空不留空壳；
 *  - `serializeColFilters(f)`：落盘前口径（= normalize 的强类型版，幂等）；
 *  - `colFilterSignature(f)`：稳定键序签名（去抖比较/测试判据）。
 *
 * 归一规则表（`COL_FILTER_KIND` 是权威，存量数据里的 `kind` 只做一致性校验）：
 *  | 列键类型 | 保留字段 | 规则 |
 *  | text  | text | 非字符串=无；trim；长度上限 64；trim 后空串=该条件不存在 |
 *  | number| op,num | num 必须是有限 number（NaN/Infinity/字符串=无）；op ∈ {ge,le,eq}，非法/缺失→ge |
 *  | date  | from,to | 必须 `YYYY-MM-DD` 且月 01~12、日 01~31；非法单侧丢弃，双侧皆非法=该条件不存在 |
 *  | enum  | set | 必须非空字符串数组；逐项 trim、去空、去重、上限 20 项；空集=该条件不存在 |
 *  | 公共  | empty | ∈ {any,empty,nonEmpty}；`any` 等同缺省（不写），其余原样保留 |
 *  | 键    | — | 只接受 `COL_FILTER_KIND` 里的键；未知键（含自定义字段表达式）整条丢弃 |
 *  | 冲突  | — | 值的 `kind` 与登记类型不一致＝脏数据，整条丢弃（不猜、不纠正） |
 */
import {
  COL_FILTER_KIND,
  isColFilterActive,
  type ColFilterKind,
  type ColFilterValue,
  type ColFilters,
} from './ganttFilter'

/** text 列条件长度上限（本机偏好不需要长文本；防止脏数据把 localStorage 顶爆） */
export const COL_FILTER_TEXT_MAX = 64
/** enum 列候选值上限（与 mode/marker 的候选规模同量级） */
export const COL_FILTER_ENUM_MAX = 20

/**
 * 已知列键 → 类型（用 Map 而非对象索引：脏数据里的 `__proto__`/`constructor`
 * 键在普通对象上会命中原型链，Map 只认登记过的自有条目）。
 */
const KNOWN_COL_KINDS = new Map<string, ColFilterKind>(
  Object.entries(COL_FILTER_KIND) as [string, ColFilterKind][],
)

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/

/** `YYYY-MM-DD` 且月/日在合理区间（不查闰年：区间比较只需格式与量级合理） */
function isIsoDate(v: unknown): v is string {
  if (typeof v !== 'string' || !DATE_RE.test(v)) return false
  const month = Number(v.slice(5, 7))
  const day = Number(v.slice(8, 10))
  return month >= 1 && month <= 12 && day >= 1 && day <= 31
}

/** enum 集合归一：trim、去空、去重、截上限（保持首次出现顺序） */
function parseEnumSet(raw: unknown): string[] {
  if (!Array.isArray(raw)) return []
  const set: string[] = []
  for (const item of raw) {
    if (typeof item !== 'string') continue
    const value = item.trim().slice(0, COL_FILTER_TEXT_MAX)
    if (value === '' || set.includes(value)) continue
    set.push(value)
    if (set.length >= COL_FILTER_ENUM_MAX) break
  }
  return set
}

/**
 * 单列条件归一：null = 该列无有效条件（调用方删键，不留空壳）。
 * `kind` 由调用方按 `COL_FILTER_KIND` 传入，是权威类型。
 */
function parseValue(kind: ColFilterKind, raw: unknown): ColFilterValue | null {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null
  const o = raw as Record<string, unknown>
  // 存量/手改数据里的 kind 与登记类型冲突 → 脏数据整条丢弃
  if (o.kind !== undefined && o.kind !== kind) return null
  const v: ColFilterValue = { kind }
  // 空值口径与其它条件是 AND 关系，独立于类型条件存在（只勾「只看空」也是有效条件）
  if (o.empty === 'empty' || o.empty === 'nonEmpty') v.empty = o.empty
  if (kind === 'text') {
    const text = typeof o.text === 'string' ? o.text.trim().slice(0, COL_FILTER_TEXT_MAX) : ''
    if (text !== '') v.text = text
  } else if (kind === 'number') {
    const num = typeof o.num === 'number' && Number.isFinite(o.num) ? o.num : null
    if (num !== null) {
      v.num = num
      v.op = o.op === 'le' ? 'le' : o.op === 'eq' ? 'eq' : 'ge'
    }
  } else if (kind === 'date') {
    if (isIsoDate(o.from)) v.from = o.from
    if (isIsoDate(o.to)) v.to = o.to
  } else {
    const set = parseEnumSet(o.set)
    if (set.length > 0) v.set = set
  }
  return isColFilterActive(v) ? v : null
}

/**
 * 任意未知输入 → 合法 `ColFilters`。
 * 只保留 `COL_FILTER_KIND` 里的已知列键；无有效键 → `{}`。不抛异常。
 */
export function normalizeColFilters(raw: unknown): ColFilters {
  const out: ColFilters = {}
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return out
  const o = raw as Record<string, unknown>
  for (const key of Object.keys(o)) {
    const kind = KNOWN_COL_KINDS.get(key)
    if (!kind) continue
    const value = parseValue(kind, o[key])
    if (value) out[key] = value
  }
  return out
}

/**
 * 落盘口径：去除空条件（`isColFilterActive` 为 false 的键丢弃），不存在空壳。
 * 对已归一数据幂等：`serialize(serialize(f))` 与 `serialize(f)` 深等。
 */
export function serializeColFilters(f: ColFilters): ColFilters {
  return normalizeColFilters(f)
}

/** 单列条件的规范文本（字段顺序固定；enum 集合排序，因为集合序无语义） */
function canonicalValue(v: ColFilterValue): string {
  const o: Record<string, unknown> = { kind: v.kind }
  if (v.kind === 'text' && v.text !== undefined) o.text = v.text
  if (v.kind === 'number') {
    if (v.op !== undefined) o.op = v.op
    if (v.num !== undefined) o.num = v.num
  }
  if (v.kind === 'date') {
    if (v.from !== undefined) o.from = v.from
    if (v.to !== undefined) o.to = v.to
  }
  if (v.kind === 'enum' && v.set !== undefined) o.set = [...v.set].sort()
  if (v.empty !== undefined) o.empty = v.empty
  return JSON.stringify(o)
}

/**
 * 稳定签名：键按字典序、每键字段顺序固定，`{}` = 空串。
 * 先归一，所以「脏数据」与「归一后等价数据」签名相同（去抖比较不会误判变化）；
 * enum 集合先后不同但内容相同 → 签名相同（集合语义与顺序无关）。
 */
export function colFilterSignature(f: ColFilters): string {
  const n = normalizeColFilters(f)
  return Object.keys(n)
    .sort()
    .map((key) => `${key}=${canonicalValue(n[key])}`)
    .join('|')
}
