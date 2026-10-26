// boards/space.test.ts — 壳层空间数据模型（docs/gaea-space-shell-design.md §4.2）。
// 一级导航合一（2026-10-26）：可达性/过滤/剪枝三件随 rail 分域退役，本文件
// 只守仍活的数据面（SHELL_SPACES/isShellSpace/boardSpace/isIndependentBoard）。
import { describe, expect, it } from 'vitest'
import {
  canonicalBoards,
} from './manifests'
import {
  SHELL_SPACES, isShellSpace, boardSpace, isIndependentBoard,
} from './space'

describe('SHELL_SPACES / isShellSpace', () => {
  it('壳层空间恰为 work/play 两档，标签符合书斋/闲庭（v4.182 文雅定名）', () => {
    expect(SHELL_SPACES.map((s) => s.id)).toEqual(['work', 'play'])
    expect(SHELL_SPACES[0].label).toBe('书斋')
    expect(SHELL_SPACES[1].label).toBe('闲庭')
  })

  it('isShellSpace 仅接受 work/play（localStorage 读取守卫）', () => {
    expect(isShellSpace('work')).toBe(true)
    expect(isShellSpace('play')).toBe(true)
    expect(isShellSpace('shared')).toBe(false)
    expect(isShellSpace('')).toBe(false)
    expect(isShellSpace(null)).toBe(false)
    expect(isShellSpace(undefined)).toBe(false)
  })
})

describe('boardSpace（manifest 归属解析）', () => {
  it('显式 space 优先', () => {
    expect(boardSpace({ ...canonicalBoards[0], space: 'play' })).toBe('play')
  })

  it('缺省语义：home → shared，其余板块 → work（旧数据按 work 兼容）', () => {
    expect(boardSpace(canonicalBoards.find((b) => b.id === 'home')!)).toBe('shared')
    expect(boardSpace({ ...canonicalBoards[1], space: undefined })).toBe('work')
  })
})

describe('isIndependentBoard（独立窗口分面机制）', () => {
  it('independent 判定 + canonical 现状无成员（v4.439 起空集，机制保留供复用）', () => {
    const indep = { ...canonicalBoards[1], id: 'indep-probe', space: 'independent' as const }
    expect(boardSpace(indep)).toBe('independent')
    expect(isIndependentBoard(indep)).toBe(true)
    expect(canonicalBoards.filter(isIndependentBoard)).toHaveLength(0)
  })
})

describe('canonical 空间归属表（docs §3，数据面文档化：会话空间/事件分域仍按此分）', () => {
  it('工位=办公/造价/记忆/青鸟/进度，乐园=聊天/小说/绘梦/角色/原罪；首页/模型中心/设置为 shared', () => {
    const by = (s: 'work' | 'play' | 'shared') =>
      canonicalBoards.filter((b) => boardSpace(b) === s).map((b) => b.id)
    expect(by('work')).toEqual(expect.arrayContaining(['gaea', 'cost', 'memoryhub', 'weixin', 'schedule']))
    expect(by('play')).toEqual(expect.arrayContaining(['chat', 'novel', 'imagegen', 'characterlib', 'sin']))
    expect(by('shared')).toEqual(expect.arrayContaining(['home', 'modelcenter', 'settings', 'runtime']))
    expect(canonicalBoards.filter(isIndependentBoard)).toHaveLength(0)
  })
})
