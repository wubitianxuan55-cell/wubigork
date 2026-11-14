import { describe, expect, it } from 'vitest'
import {
  applyCustomViewSnapshot,
  canRenameView,
  canSaveView,
  makeCustomView,
} from './views'
import type { SchedView } from './types'

const base: SchedView = { id: 'v1', name: '已有视图', kind: 'gantt' }

describe('canSaveView / canRenameView（斑马同名原文校验）', () => {
  it('空名/纯空白拒绝；重名→「存在同名视图，请重新保存」；上限 20', () => {
    expect(canSaveView('', [base]).ok).toBe(false)
    expect(canSaveView('   ', [base]).ok).toBe(false)
    expect(canSaveView('已有视图', [base]).reason).toBe('存在同名视图，请重新保存')
    const full = Array.from({ length: 20 }, (_, i) => ({ id: `x${i}`, name: `V${i}`, kind: 'gantt' as const }))
    expect(canSaveView('新视图', full).reason).toContain('上限 20')
    expect(canSaveView('新视图', [base]).ok).toBe(true)
  })
  it('改名排除自身：同名自身放行、撞他人拒绝', () => {
    expect(canRenameView('v1', '已有视图', [base]).ok).toBe(true)
    expect(canRenameView('v2', '已有视图', [base]).reason).toBe('存在同名视图，请重新保存')
    expect(canRenameView('v2', '', [base]).ok).toBe(false)
  })
})

describe('makeCustomView（快照 → SchedView，形状收窄）', () => {
  const snap = {
    hide: ['ls', 'lf', 'nope', 'ls'], // 非法键/重复由 sanitizeHide 过滤
    groupBy: ['status', 'bogus', 'critical'] as Array<'status' | 'bogus' | 'critical'>,
    sort: { field: 'start', dir: 'desc' },
    filterKind: 'milestone',
  } as never
  it('捕获列集/分组/排序/筛选档；非法组键丢弃；通过校验返回视图', () => {
    const { view, reason } = makeCustomView('管控视图', 'cv-1', snap, [base])
    expect(reason).toBeUndefined()
    expect(view).toMatchObject({ id: 'cv-1', name: '管控视图', kind: 'gantt' })
    expect(view!.hide).toEqual(['ls', 'lf'])
    expect(view!.groupBy).toEqual(['status', 'critical'])
    expect(view!.sort).toEqual({ field: 'start', dir: 'desc' })
    expect(view!.filter).toBe('milestone')
  })
  it('重名返回原因不给视图；空快照（全缺省）产出仅骨架', () => {
    expect(makeCustomView('已有视图', 'cv-2', snap, [base]).reason).toBe('存在同名视图，请重新保存')
    const empty = { hide: [], groupBy: [], sort: { field: 'none', dir: 'asc' }, filterKind: 'all' } as never
    const { view } = makeCustomView('骨架', 'cv-3', empty, [])
    expect(view!.hide).toBeUndefined()
    expect(view!.groupBy).toBeUndefined()
    expect(view!.sort).toBeDefined()
  })
})

describe('applyCustomViewSnapshot（应用容错，坏值回落缺省）', () => {
  it('合法快照原样回放；坏形输入回落 all/none/[]', () => {
    const v: SchedView = {
      id: 'cv-9', name: 'X', kind: 'gantt',
      hide: ['tf', 'junk'], groupBy: ['status'], sort: { field: 'start', dir: 'desc' }, filter: 'milestone',
    }
    expect(applyCustomViewSnapshot(v)).toEqual({ hide: ['tf'], groupBy: ['status'], sort: { field: 'start', dir: 'desc' }, filterKind: 'milestone', ok: true })
    const bad: SchedView = { id: 'cv-8', name: 'Y', kind: 'gantt', groupBy: ['nope'], sort: { field: 'zap', dir: 'sideways' as never } }
    const a = applyCustomViewSnapshot(bad)
    expect(a.hide).toEqual([])
    expect(a.groupBy).toEqual([])
    expect(a.sort).toEqual({ field: 'none', dir: 'asc' })
    expect(a.filterKind).toBe('all')
  })
  it('分组截到三级（MAX_GANTT_GROUP_DEPTH）', () => {
    const v: SchedView = { id: 'd', name: 'D', kind: 'gantt', groupBy: ['status', 'critical', 'mode', 'duration'] }
    expect(applyCustomViewSnapshot(v).groupBy).toEqual(['status', 'critical', 'mode'])
  })
})
