/**
 * templates.ts — 项目模板（v4.138 #10，自设计·上游空白）
 *
 * 把当前整份工程计划（任务/搭接/日历/资源…全量 SchedProject）存成本地模板，
 * 新工程从同类模板一键起步。storage 键 gaea.schedule.templates，范式与
 * chatPrefs.ts 一致：纯函数 + 可注入 storage（测试注入内存实现，运行时落
 * localStorage），写失败静默——模板属锦上添花，不打断编辑流。
 *
 * 存储结构：TemplateItem[]（JSON 序列化）。容错口径：
 *  - 坏 JSON / 非数组 → 整体回落空表；
 *  - 单条坏条目（缺名字/时间/project 缺 tasks 数组等）→ 逐条丢弃不拖累好条目；
 *  - 同名保存 = 原位覆盖（更新快照与保存时间，顺序不变），否则追加；
 *  - 上限 10 个，超出 FIFO 淘汰最旧（数组头部=最旧）。
 */
import type { SchedProject } from './types'

/** 模板条目：模板名 + 保存时间（YYYY-MM-DD HH:mm）+ 整份工程快照 */
export interface TemplateItem {
  name: string
  savedAt: string
  project: SchedProject
}

/** storage 键 */
export const TEMPLATES_KEY = 'gaea.schedule.templates'

/** 模板上限：超出 FIFO 淘汰最旧 */
export const TEMPLATES_CAP = 10

/** storage 注入面：读+写（chatPrefs 同款） */
type TplStorage = Pick<Storage, 'getItem' | 'setItem'>

/** 缺省 storage：浏览器环境取 localStorage，否则无（读回落空表、写静默失败） */
function defaultStorage(): TplStorage | undefined {
  return typeof localStorage !== 'undefined' ? localStorage : undefined
}

/** 保存时间戳（本地时间 YYYY-MM-DD HH:mm，与 store 基线时间同格式） */
function nowStamp(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

/** 单条容错：名字非空串 + 时间是字符串 + project 具备 tasks 数组才算模板条目 */
function parseItem(o: unknown): TemplateItem | null {
  if (!o || typeof o !== 'object') return null
  const it = o as Partial<TemplateItem>
  if (typeof it.name !== 'string' || it.name === '') return null
  if (typeof it.savedAt !== 'string') return null
  if (!it.project || typeof it.project !== 'object' || !Array.isArray((it.project as SchedProject).tasks)) return null
  return it as TemplateItem
}

/**
 * 读取模板列表（storage 可注入供测试）。坏 JSON/非数组整体回落空表，
 * 坏条目逐条容错丢弃，好条目照常读出。
 */
export function listTemplates(storage?: Pick<Storage, 'getItem'>): TemplateItem[] {
  const s = storage ?? defaultStorage()
  if (!s) return []
  let raw: string | null = null
  try {
    raw = s.getItem(TEMPLATES_KEY)
  } catch {
    return []
  }
  if (!raw) return []
  try {
    const o = JSON.parse(raw) as unknown
    if (!Array.isArray(o)) return []
    return o.map(parseItem).filter((v): v is TemplateItem => v !== null)
  } catch {
    return []
  }
}

/** 写回列表（写失败静默），返回入参列表便于链式刷新 UI */
function writeList(list: TemplateItem[], storage?: TplStorage): TemplateItem[] {
  const s = storage ?? defaultStorage()
  try {
    s?.setItem(TEMPLATES_KEY, JSON.stringify(list))
  } catch { /* 隐私模式等写失败忽略 */ }
  return list
}

/**
 * 另存为模板：同名覆盖（原位更新快照与保存时间，顺序不变），否则追加；
 * 上限 10 个，超出 FIFO 淘汰最旧。name 先 trim，trim 后为空视为放弃
 * （无害 no-op，原表原样返回）。savedAt 可注入（测试确定性），
 * 缺省取当前本地时间。
 */
export function saveTemplate(
  name: string,
  project: SchedProject,
  storage?: TplStorage,
  savedAt?: string,
): TemplateItem[] {
  const trimmed = name.trim()
  const list = listTemplates(storage)
  if (trimmed === '') return list
  const item: TemplateItem = { name: trimmed, savedAt: savedAt ?? nowStamp(), project }
  const idx = list.findIndex((t) => t.name === trimmed)
  if (idx >= 0) list[idx] = item
  else list.push(item)
  if (list.length > TEMPLATES_CAP) list.splice(0, list.length - TEMPLATES_CAP)
  return writeList(list, storage)
}

/** 删除模板：按名精确匹配；名字不存在 = 无害 no-op */
export function deleteTemplate(name: string, storage?: TplStorage): TemplateItem[] {
  const list = listTemplates(storage)
  const next = list.filter((t) => t.name !== name)
  if (next.length === list.length) return list
  return writeList(next, storage)
}

/** 取单个模板：命中返回条目（含整份工程快照），未命中返回 null */
export function getTemplate(name: string, storage?: Pick<Storage, 'getItem'>): TemplateItem | null {
  return listTemplates(storage).find((t) => t.name === name) ?? null
}
