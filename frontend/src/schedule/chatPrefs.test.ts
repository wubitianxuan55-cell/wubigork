/**
 * chatPrefs.test.ts — 左栏偏好纯函数用例（刀11）
 */
import { describe, expect, it } from 'vitest'
import { CHAT_PREFS_KEY, clampChatWidth, loadChatPrefs, saveChatPrefs } from './chatPrefs'

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
    expect(loadChatPrefs(fakeStorage())).toEqual({ collapsed: false, width: 420 })
    expect(loadChatPrefs(fakeStorage({ [CHAT_PREFS_KEY]: 'not json' }))).toEqual({ collapsed: false, width: 420 })
  })
  it('字段级容错：坏 width 不拖累 collapsed', () => {
    const s = fakeStorage({ [CHAT_PREFS_KEY]: '{"collapsed":true,"width":"fat"}' })
    expect(loadChatPrefs(s)).toEqual({ collapsed: true, width: 420 })
  })
})

describe('saveChatPrefs', () => {
  it('部分字段合并落盘，读取往返一致', () => {
    const s = fakeStorage()
    saveChatPrefs({ collapsed: true }, s)
    expect(loadChatPrefs(s)).toEqual({ collapsed: true, width: 420 })
    saveChatPrefs({ width: 555 }, s)
    expect(loadChatPrefs(s)).toEqual({ collapsed: true, width: 555 })
  })
  it('落盘前钳位', () => {
    const s = fakeStorage()
    saveChatPrefs({ width: 99 }, s)
    expect(JSON.parse(s.dump()[CHAT_PREFS_KEY]).width).toBe(320)
  })
})
