/**
 * exportMeta.ts — 图面导出签署字段偏好（纯函数 + 可注入 storage，v4.132.0 刀D1）
 *
 * 键 gaea.schedule.exportMeta：{"org","designer","reviewer","approver","date"}。
 * 全部字符串逐项容错（trim、上限 40；date 须 YYYY-MM-DD 否则丢弃），
 * 单个字段坏不拖累另一个；写失败静默（偏好属锦上添花）。
 */

export interface ExportMetaPrefs {
  org: string
  designer: string
  reviewer: string
  approver: string
  date: string
}

export const EXPORT_META_KEY = 'gaea.schedule.exportMeta'

const DEFAULTS: ExportMetaPrefs = { org: '', designer: '', reviewer: '', approver: '', date: '' }

function str(v: unknown): string {
  if (typeof v !== 'string') return ''
  return v.trim().slice(0, 40)
}

function parse(raw: string | null): ExportMetaPrefs {
  if (!raw) return { ...DEFAULTS }
  try {
    const o = JSON.parse(raw) as Partial<ExportMetaPrefs>
    return {
      org: str(o.org),
      designer: str(o.designer),
      reviewer: str(o.reviewer),
      approver: str(o.approver),
      date: /^\d{4}-\d{2}-\d{2}$/.test(str(o.date)) ? str(o.date) : '',
    }
  } catch {
    return { ...DEFAULTS }
  }
}

/** 读取签署字段偏好（storage 可注入供测试） */
export function loadExportMeta(storage?: Pick<Storage, 'getItem'>): ExportMetaPrefs {
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  if (!s) return { ...DEFAULTS }
  try {
    return parse(s.getItem(EXPORT_META_KEY))
  } catch {
    return { ...DEFAULTS }
  }
}

/** 保存签署字段偏好（整体替换；字段级容错） */
export function saveExportMeta(patch: ExportMetaPrefs, storage?: Pick<Storage, 'getItem' | 'setItem'>): ExportMetaPrefs {
  const next: ExportMetaPrefs = {
    org: str(patch.org),
    designer: str(patch.designer),
    reviewer: str(patch.reviewer),
    approver: str(patch.approver),
    date: /^\d{4}-\d{2}-\d{2}$/.test(str(patch.date)) ? str(patch.date) : '',
  }
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  try {
    s?.setItem(EXPORT_META_KEY, JSON.stringify(next))
  } catch { /* 写失败忽略 */ }
  return next
}

/** 今日 ISO（YYYY-MM-DD，本地时区） */
export function todayIso(): string {
  const d = new Date()
  const p = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
