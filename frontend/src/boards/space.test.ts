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
  it('壳层空间恰为 work/play 两档，标签对应当前工位/乐园语义', () => {
    expect(SHELL_SPACES.map((s) => s.id)).toEqual(['work', 'play'])
    expect(SHELL_SPACES[0].label).toBe('工位')
    expect(SHELL_SPACES[1].label).toBe('乐园')
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
    expect(boardSpace({ ...canonicalBoards.find((b) => b.id === 'chat')!, space: 'play' })).toBe('play')
  })

  it('缺省语义：未声明 space 的板块按 work 兼容', () => {
    expect(boardSpace({ ...canonicalBoards.find((b) => b.id === 'gaea')!, space: undefined })).toBe('work')
  })
})

describe('isIndependentBoard（独立窗口分面机制）', () => {
  it('independent 判定 + canonical 现状无成员（v4.439 起空集，机制保留供复用）', () => {
    const indep = { ...canonicalBoards[0], id: 'indep-probe', space: 'independent' as const }
    expect(boardSpace(indep)).toBe('independent')
    expect(isIndependentBoard(indep)).toBe(true)
    expect(canonicalBoards.filter(isIndependentBoard)).toHaveLength(0)
  })
})

describe('canonical 空间归属表（docs §3，数据面文档化：会话空间/事件分域仍按此分）', () => {
  it('工位=办公/造价/记忆/青鸟/进度，乐园=聊天/小说/绘梦/角色（原罪 2026-10-27 并入小说）；模型中心/设置/运行监测为 shared', () => {
    const by = (s: 'work' | 'play' | 'shared') =>
      canonicalBoards.filter((b) => boardSpace(b) === s).map((b) => b.id)
    expect(by('work')).toEqual(expect.arrayContaining(['gaea', 'cost', 'memoryhub', 'weixin', 'schedule']))
    expect(by('play')).toEqual(expect.arrayContaining(['chat', 'novel', 'imagegen', 'characterlib']))
    expect(by('shared')).toEqual(expect.arrayContaining(['modelcenter', 'settings', 'runtime']))
    expect(canonicalBoards.filter(isIndependentBoard)).toHaveLength(0)
  })
})
