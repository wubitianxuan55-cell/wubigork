/**
 * chatPrefs.test.ts — 板块工作台偏好纯函数用例（刀11；刀C 增横道双栏字段）
 */
import { describe, expect, it } from 'vitest'
import { CHAT_PREFS_KEY, clampChatWidth, loadChatPrefs, saveChatPrefs } from './chatPrefs'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN } from './ganttCols'

function fakeStorage(initial: Record<string, string> = {}): Storage & { dump: () => Record<string, string> } {
  const bag = { ...initial }
  return {
    getItem: (k: string) => (k in bag ? bag[k] : null),
    setItem: (k: string, v: string) => { bag[k] = v },
    removeItem: (k: string) => { delete bag[k] },
    clear: () => { for (const k of Object.keys(bag)) delete bag[k] },
    key: (i: number) => Object.keys(bag)[i] ?? null,
    get length() { return Object.keys(bag).length },
    dump: () => ({ ...bag }),
  } as Storage & { dump: () => Record<string, string> }
}

const DEFAULTS = { collapsed: false, width: 420, ganttTableW: GANTT_LEFT_W_FULL, ganttHide: ['progress'] }

describe('clampChatWidth', () => {
  it('钳位 320~680，非有限值回落 420', () => {
    expect(clampChatWidth(200)).toBe(320)
    expect(clampChatWidth(1000)).toBe(680)
    expect(clampChatWidth(500)).toBe(500)
    expect(clampChatWidth(NaN)).toBe(420)
  })
})

describe('loadChatPrefs', () => {
  it('无存储/空值回落缺省', () => {
    expect(loadChatPrefs(fakeStorage())).toEqual(DEFAULTS)
    expect(loadChatPrefs(fakeStorage({ [CHAT_PREFS_KEY]: 'not json' }))).toEqual(DEFAULTS)
  })
  it('字段级容错：坏 width 不拖累 collapsed', () => {
    const s = fakeStorage({ [CHAT_PREFS_KEY]: '{"collapsed":true,"width":"fat"}' })
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true })
  })
  it('刀C：ganttTableW 钳位 [最小宽,全列宽]、ganttHide 过滤非法键（未知键/固定列丢弃）', () => {
    const s = fakeStorage({ [CHAT_PREFS_KEY]: JSON.stringify({ ganttTableW: 99999, ganttHide: ['ls', 'no', 'ghost', 'lf'] }) })
    expect(loadChatPrefs(s).ganttTableW).toBe(GANTT_LEFT_W_FULL)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf'])
    const s2 = fakeStorage({ [CHAT_PREFS_KEY]: '{"ganttTableW":"fat"}' })
    expect(loadChatPrefs(s2).ganttTableW).toBe(GANTT_LEFT_W_FULL)
    const s3 = fakeStorage({ [CHAT_PREFS_KEY]: JSON.stringify({ ganttTableW: 5 }) })
    expect(loadChatPrefs(s3).ganttTableW).toBe(GANTT_TABLE_W_MIN)
  })
})

describe('saveChatPrefs', () => {
  it('部分字段合并落盘，读取往返一致', () => {
    const s = fakeStorage()
    saveChatPrefs({ collapsed: true }, s)
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true })
    saveChatPrefs({ width: 555 }, s)
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true, width: 555 })
  })
  it('落盘前钳位', () => {
    const s = fakeStorage()
    saveChatPrefs({ width: 99 }, s)
    expect(JSON.parse(s.dump()[CHAT_PREFS_KEY]).width).toBe(320)
  })
  it('刀C：横道表格宽度/显隐键往返', () => {
    const s = fakeStorage()
    saveChatPrefs({ ganttTableW: 420, ganttHide: ['ls', 'lf', 'cost'] }, s)
    expect(loadChatPrefs(s).ganttTableW).toBe(420)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf', 'cost'])
    // 只改宽度不丢显隐
    saveChatPrefs({ ganttTableW: 300 }, s)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf', 'cost'])
    expect(loadChatPrefs(s).ganttTableW).toBe(300)
  })
})
