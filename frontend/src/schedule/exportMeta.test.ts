/**
 * exportMeta.test.ts — 图面导出签署字段偏好（刀D1）
 *
 * 用例口径：可注入 storage 往返；trim/40 上限/日期格式逐项容错；
 * 坏 JSON 回落缺省；单字段坏不拖累另一个。
 */
import { describe, expect, it } from 'vitest'
import { EXPORT_META_KEY, loadExportMeta, saveExportMeta, todayIso } from './exportMeta'

function memStorage(): Storage & { store: Map<string, string> } {
  const store = new Map<string, string>()
  return {
    store,
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
    key: () => null,
    get length() { return store.size },
  } as Storage & { store: Map<string, string> }
}

describe('exportMeta 往返与容错', () => {
  it('保存→读取往返；trim 与 40 字上限', () => {
    const s = memStorage()
    const saved = saveExportMeta({ org: '  博耐特建设  ', designer: '张三', reviewer: '', approver: '', date: '2026-09-07' }, s)
    expect(saved.org).toBe('博耐特建设')
    expect(loadExportMeta(s)).toEqual(saved)
    expect(s.store.get(EXPORT_META_KEY)).toContain('博耐特建设')
    const long = saveExportMeta({ org: '单'.repeat(50), designer: '', reviewer: '', approver: '', date: '' }, s)
    expect(long.org).toHaveLength(40)
  })

  it('日期格式不符丢弃；坏 JSON 回落缺省；单字段坏不拖累', () => {
    const s = memStorage()
    const bad = saveExportMeta({ org: 'A', designer: '', reviewer: '', approver: '', date: '2026/09/07' }, s)
    expect(bad.date).toBe('')
    s.store.set(EXPORT_META_KEY, '{oops')
    expect(loadExportMeta(s)).toEqual({ org: '', designer: '', reviewer: '', approver: '', date: '' })
    s.store.set(EXPORT_META_KEY, JSON.stringify({ org: '保留', designer: 42, date: 'nope' }))
    const loaded = loadExportMeta(s)
    expect(loaded.org).toBe('保留')
    expect(loaded.designer).toBe('')
    expect(loaded.date).toBe('')
  })

  it('todayIso：YYYY-MM-DD 本地日期', () => {
    expect(todayIso()).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})
