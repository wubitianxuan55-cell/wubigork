/**
 * exportMeta.test.ts — 图面导出签署字段偏好（刀D1）
 *
 * 用例口径：可注入 storage 往返；trim/40 上限/日期格式逐项容错；
 * 坏 JSON 回落缺省；单字段坏不拖累另一个；notes 多行往返与非字符串丢弃。
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

  it('notes 编制说明：多行文本往返；非字符串丢弃、超 600 上限截断', () => {
    const s = memStorage()
    const notes = '基础开挖至设计标高\n混凝土采用 C30\n雨季施工措施见专项方案'
    const saved = saveExportMeta({ org: '', designer: '', reviewer: '', approver: '', date: '', notes }, s)
    expect(saved.notes).toBe(notes)
    expect(loadExportMeta(s)).toEqual(saved)
    expect(s.store.get(EXPORT_META_KEY)).toContain('混凝土采用 C30')
    // 坏值容错：非字符串（数字/数组）丢弃，其余字段不拖累
    s.store.set(EXPORT_META_KEY, JSON.stringify({ org: '保留', notes: 42 }))
    let loaded = loadExportMeta(s)
    expect(loaded.org).toBe('保留')
    expect(loaded.notes).toBeUndefined()
    s.store.set(EXPORT_META_KEY, JSON.stringify({ notes: ['多行'] }))
    expect(loadExportMeta(s).notes).toBeUndefined()
    // 上限 600（多行文本不给 40 字的签署字段小口径）
    const long = saveExportMeta({ org: '', designer: '', reviewer: '', approver: '', date: '', notes: '行'.repeat(601) }, s)
    expect(long.notes).toHaveLength(600)
    loaded = loadExportMeta(s)
    expect(loaded.notes).toHaveLength(600)
  })

  it('todayIso：YYYY-MM-DD 本地日期', () => {
    expect(todayIso()).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})
