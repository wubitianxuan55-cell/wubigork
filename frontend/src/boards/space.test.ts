// boards/space.test.ts — S2.1 双空间壳空间模型（docs/gaea-space-shell-design.md §4.2）
import { describe, expect, it } from 'vitest'
import {
  canonicalBoards,
} from './manifests'
import {
  SHELL_SPACES, isShellSpace, boardSpace, isBoardReachableInSpace, isIndependentBoard,
  filterBoardsForSpace, pruneVisitedForSpace,
} from './space'

describe('SHELL_SPACES / isShellSpace', () => {
  it('壳层空间恰为 work/play 两档，标签符合工位/乐园', () => {
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
    expect(boardSpace({ ...canonicalBoards[0], space: 'play' })).toBe('play')
  })

  it('缺省语义：home → shared，其余板块 → work（旧数据按 work 兼容）', () => {
    expect(boardSpace(canonicalBoards.find((b) => b.id === 'home')!)).toBe('shared')
    expect(boardSpace({ ...canonicalBoards[1], space: undefined })).toBe('work')
  })
})

describe('isBoardReachableInSpace / filterBoardsForSpace', () => {
  it('shared 两空间均可达；work/play 只在各自空间可达', () => {
    const shared = { ...canonicalBoards[1], space: 'shared' as const }
    expect(isBoardReachableInSpace(shared, 'work')).toBe(true)
    expect(isBoardReachableInSpace(shared, 'play')).toBe(true)

    const work = { ...canonicalBoards[1], space: 'work' as const }
    expect(isBoardReachableInSpace(work, 'work')).toBe(true)
    expect(isBoardReachableInSpace(work, 'play')).toBe(false)

    const play = { ...canonicalBoards[1], space: 'play' as const }
    expect(isBoardReachableInSpace(play, 'play')).toBe(true)
    expect(isBoardReachableInSpace(play, 'work')).toBe(false)
  })

  it('independent（编程 DSH 独立窗口）两空间均不可达、不进空间导航', () => {
    const code = canonicalBoards.find((b) => b.id === 'code')!
    expect(boardSpace(code)).toBe('independent')
    expect(isIndependentBoard(code)).toBe(true)
    expect(isBoardReachableInSpace(code, 'work')).toBe(false)
    expect(isBoardReachableInSpace(code, 'play')).toBe(false)
  })

  it('canonical 空间归属表（docs §3）：工位含办公/造价/记忆/知识，乐园含小说/绘梦/角色', () => {
    const workIds = filterBoardsForSpace(canonicalBoards, 'work').map((b) => b.id)
    const playIds = filterBoardsForSpace(canonicalBoards, 'play').map((b) => b.id)
    // 工位：home/shared + work 归属
    expect(workIds).toEqual(expect.arrayContaining(['home', 'gaea', 'cost', 'memoryhub', 'modelcenter', 'settings']))
    expect(workIds).not.toContain('novel')
    expect(workIds).not.toContain('imagegen')
    expect(workIds).not.toContain('characterlib')
    expect(workIds).not.toContain('chat') // P1：对话降为对话流，工位不再有独立聊天板块
    expect(workIds).not.toContain('code') // 编程独立窗口不并入工位
    // 乐园：home/shared + play 归属
    expect(playIds).toEqual(expect.arrayContaining(['home', 'chat', 'novel', 'imagegen', 'characterlib', 'modelcenter', 'settings']))
    expect(playIds).not.toContain('gaea')
    expect(playIds).not.toContain('cost')
    expect(playIds).not.toContain('memoryhub')
    expect(playIds).not.toContain('code') // 编程独立窗口不进乐园
  })

  it('过滤不修改入参（新数组）', () => {
    const input = [...canonicalBoards]
    filterBoardsForSpace(input, 'play')
    expect(input).toHaveLength(canonicalBoards.length)
  })

  it('pruneVisitedForSpace（S2.2 性能门控）：只保留谓词通过的页面', () => {
    const visited = new Set(['home', 'novel', 'gaea', 'imagegen'])
    const inWork = (id: string) => id === 'home' || id === 'gaea'
    expect(pruneVisitedForSpace(visited, inWork)).toEqual(['home', 'gaea'])
    const inPlay = (id: string) => id === 'home' || id === 'novel' || id === 'imagegen'
    expect(pruneVisitedForSpace(visited, inPlay)).toEqual(['home', 'novel', 'imagegen'])
    // 保活页被清空时结果为空数组（不抛错）
    expect(pruneVisitedForSpace(visited, () => false)).toEqual([])
  })
})
