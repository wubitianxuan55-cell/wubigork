// booleanPref.test.ts — 方言矩阵冻结：默认开/默认关两向 × null/显式/垃圾值，
// 与四域既有口径逐格等价（收敛前各域测试继续兜底，这里钉工厂本身的规则）。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createBooleanPref } from './booleanPref'

afterEach(() => {
  window.localStorage.clear()
  vi.restoreAllMocks()
})

describe('createBooleanPref 默认开（browserPrefs 方言）', () => {
  const pref = createBooleanPref('test.pref.defaultOn', true)

  it('null → 默认开', () => {
    expect(pref.load()).toBe(true)
  })

  it('显式关值 "0"/"false" → 关', () => {
    window.localStorage.setItem('test.pref.defaultOn', '0')
    expect(pref.load()).toBe(false)
    window.localStorage.setItem('test.pref.defaultOn', 'false')
    expect(pref.load()).toBe(false)
  })

  it('"1"/"true"/垃圾值 → 一律默认开（不认显式开值之外的解释）', () => {
    for (const raw of ['1', 'true', 'garbage', '']) {
      window.localStorage.setItem('test.pref.defaultOn', raw)
      expect(pref.load()).toBe(true)
    }
  })
})

describe('createBooleanPref 默认关（subagent/tasks/deliverable 方言）', () => {
  const pref = createBooleanPref('test.pref.defaultOff', false)

  it('null → 默认关', () => {
    expect(pref.load()).toBe(false)
  })

  it('显式开值 "1"/"true" → 开', () => {
    window.localStorage.setItem('test.pref.defaultOff', '1')
    expect(pref.load()).toBe(true)
    window.localStorage.setItem('test.pref.defaultOff', 'true')
    expect(pref.load()).toBe(true)
  })

  it('"0"/"false"/垃圾值 → 一律默认关', () => {
    for (const raw of ['0', 'false', 'garbage', '']) {
      window.localStorage.setItem('test.pref.defaultOff', raw)
      expect(pref.load()).toBe(false)
    }
  })
})

describe('createBooleanPref 持久化与降级', () => {
  it('save 写 "1"/"0"，load 往返一致', () => {
    const pref = createBooleanPref('test.pref.roundtrip', false)
    pref.save(true)
    expect(window.localStorage.getItem('test.pref.roundtrip')).toBe('1')
    expect(pref.load()).toBe(true)
    pref.save(false)
    expect(window.localStorage.getItem('test.pref.roundtrip')).toBe('0')
    expect(pref.load()).toBe(false)
  })

  it('读写抛错（存储禁用）静默回落：load 回默认、save 不上抛', () => {
    // jsdom 的 localStorage 不经 Storage.prototype 分派：实例级 spy 拦截
    const getItem = vi.spyOn(window.localStorage, 'getItem').mockImplementation(() => {
      throw new Error('quota')
    })
    const setItem = vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
      throw new Error('quota')
    })
    const on = createBooleanPref('test.pref.throwing', true)
    const off = createBooleanPref('test.pref.throwing', false)
    expect(on.load()).toBe(true)
    expect(off.load()).toBe(false)
    expect(() => on.save(true)).not.toThrow()
    expect(getItem).toHaveBeenCalled()
    expect(setItem).toHaveBeenCalled()
  })
})
