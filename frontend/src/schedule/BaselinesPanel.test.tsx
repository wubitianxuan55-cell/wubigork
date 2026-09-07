/**
 * BaselinesPanel.test.tsx — 多基线槽位面板 UI 用例（v4.137 #11）
 *
 * 覆盖：空槽+无活跃的旧版保存引导与守卫（循环依赖/无叶任务禁用+原因）、
 * 命名保存建槽（baseline 与 baselines 各 +1）、同名保存=原位更新不新增槽、
 * 双槽 radio 切换活跃（漂移摘要跟随）、删除活跃槽（活跃指针清空另一槽保留）、
 * 清除对比（槽位保留）、手工旧快照的漂移摘要（总工期漂移数字/推移计数）、
 * 3 槽上限提示与 FIFO 淘汰、更新基线原位写回。
 * 面板数据全部走 useScheduleStore（与被替换进 Popover 的运行形态一致），
 * 漂移期望值与 baseline.ts 同口径（工作日序号快照）。
 * v4.138 追加：签证台账 CSV 导出——无槽位隐藏、有槽位在位，点击经
 * downloadBlob 触发下载（jsdom stub URL.createObjectURL），断言文件名、
 * BOM、表头与每槽一行（漂移=当前-基线，正=拖后）。
 */

import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { BaselinesPanel } from './BaselinesPanel'
import { computeCpm } from './cpm'
import { useScheduleStore } from './store'
import type { SchedBaseline, SchedProject } from './types'

/** A(3)→B(2)→C(4) 串联，周一开工：es A=0/B=3/C=5，总工期 9，全关键 */
function chainProject(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

/** 在给定工程上把 B 工期 2→5（总工期 9→12，B/C 相对原排程推移）；保留已存槽位 */
function stretchB(p: SchedProject): SchedProject {
  return { ...p, tasks: p.tasks.map((t) => (t.id === 'B' ? { ...t, duration: 5 } : t)) }
}

/** 排程变化后页面会以新 CPM 重渲染面板（同 SchedulePage 的 useMemo 范式） */
function rerenderWith(view: ReturnType<typeof render>, p: SchedProject) {
  view.rerender(<BaselinesPanel cpm={computeCpm(p.tasks, p.links)} />)
}

const T0 = '2026-09-01 08:00'

/** 手工基线槽（rows 为工作日序号快照） */
function slotOf(name: string, duration: number, rows: SchedBaseline['rows']): SchedBaseline {
  return { name, savedAt: T0, duration, rows }
}

/** 与 chainProject 当前 CPM 完全一致的快照（总工期 9，对比无漂移） */
const CHAIN_SNAP: SchedBaseline['rows'] = {
  A: { name: '挖土', es: 0, ef: 3, dur: 3, critical: true },
  B: { name: '垫层', es: 3, ef: 5, dur: 2, critical: true },
  C: { name: '浇筑', es: 5, ef: 9, dur: 4, critical: true },
}

/** 旧排程快照（B 曾从第 0 工作日搭接，总工期 6）——对当前排程制造漂移用 */
const STALE_SNAP: SchedBaseline['rows'] = {
  A: { name: '挖土', es: 0, ef: 3, dur: 3, critical: true },
  B: { name: '垫层', es: 0, ef: 2, dur: 2, critical: true },
  C: { name: '浇筑', es: 2, ef: 6, dur: 4, critical: true },
}

function setProject(p: SchedProject) {
  useScheduleStore.setState({ project: p, selectedId: null, hydrated: true, sync: 'saved', syncError: null })
}

function renderPanel(p: SchedProject) {
  setProject(p)
  return render(<BaselinesPanel cpm={computeCpm(p.tasks, p.links)} />)
}

// antd 组件 data-testid 落点不一（Input 在 input 本体、包装组件在根层），统一取内部 input
function inputOf(testid: string): HTMLInputElement {
  const el = screen.getByTestId(testid)
  if (el instanceof HTMLInputElement) return el
  const inner = el.querySelector('input')
  if (inner) return inner
  throw new Error(`${testid} 内无 input`)
}

/** 保存新基线：点保存按钮 → 填名 → 确认（面板把「保存基线/保存为新基线」收进同一表单） */
function saveAs(name: string) {
  fireEvent.click(screen.getByTestId('sched-baseline-save'))
  fireEvent.change(inputOf('sched-baseline-name-input'), { target: { value: name } })
  fireEvent.click(screen.getByTestId('sched-baseline-save-confirm'))
}

/** 确认 Popconfirm（antd 缺省 locale 为 OK） */
async function confirmPop() {
  fireEvent.click(await screen.findByRole('button', { name: /OK|确定/ }))
}

describe('BaselinesPanel 空态与守卫', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
  })

  it('无槽位且无活跃基线：显示旧版保存引导；命名保存即建槽（baseline 与 baselines 各 +1）', () => {
    renderPanel(chainProject())
    const panel = screen.getByTestId('sched-baselines-panel')
    expect(panel.textContent).toContain('基线会固化当前排程结果')
    expect(screen.queryByTestId('sched-baseline-slot')).toBeNull()
    expect(screen.getByTestId('sched-baseline-save').textContent).toContain('保存基线')

    saveAs('开工基线')
    const s = useScheduleStore.getState().project
    expect(s.baseline?.name).toBe('开工基线')
    expect(s.baseline?.duration).toBe(9) // 总工期 9 固化进快照
    expect(Object.keys(s.baseline!.rows).sort()).toEqual(['A', 'B', 'C'])
    expect(s.baselines?.map((b) => b.name)).toEqual(['开工基线'])
    // 建槽后转入槽位视图：一行槽位 + 活跃行「当前对比」标记
    expect(screen.getAllByTestId('sched-baseline-slot')).toHaveLength(1)
    expect(screen.getByTestId('sched-baseline-slot').textContent).toContain('当前对比')
  })

  it('守卫：循环依赖 → 保存禁用并给原因「计划存在循环依赖，先修正搭接」', () => {
    const p = chainProject()
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    renderPanel(p)
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('暂不可保存：计划存在循环依赖，先修正搭接')
    expect((screen.getByTestId('sched-baseline-save') as HTMLButtonElement).disabled).toBe(true)
  })

  it('守卫：无叶任务 → 提示「计划还没有任何任务」且保存禁用', () => {
    renderPanel({
      name: '空计划',
      startDate: '2026-09-07',
      tasks: [{ id: 'G', name: '总分组', duration: 0, level: 0, progress: 0 }],
      links: [],
    })
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('暂不可保存：计划还没有任何任务')
    expect((screen.getByTestId('sched-baseline-save') as HTMLButtonElement).disabled).toBe(true)
  })
})

describe('BaselinesPanel 槽位管理', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
  })

  it('同名保存=原位更新：槽位数不变，快照跟随当前排程，更新后漂移归零', () => {
    const view = renderPanel(chainProject())
    saveAs('基线1')
    expect(useScheduleStore.getState().project.baselines).toHaveLength(1)
    // 排程变化：B 工期 2→5（总工期 9→12），活跃基线还是旧快照 → 出现漂移
    const stretched = stretchB(useScheduleStore.getState().project)
    setProject(stretched)
    rerenderWith(view, stretched)
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('总工期 9 → 12')
    saveAs('基线1') // 重名 → 原位更新该槽
    const s = useScheduleStore.getState().project
    expect(s.baselines).toHaveLength(1)
    expect(s.baseline?.name).toBe('基线1')
    expect(s.baseline?.duration).toBe(12)
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('与基线持平')
  })

  it('双槽切换活跃：点另一槽 radio → project.baseline 切换，漂移摘要跟随、「当前对比」高亮跟随', () => {
    const view = renderPanel(chainProject())
    saveAs('开工基线') // 槽1：总工期 9 快照
    const stretched = stretchB(useScheduleStore.getState().project) // 排程变化：保留槽位
    setProject(stretched)
    rerenderWith(view, stretched)
    saveAs('签证基线') // 槽2：总工期 12 快照，保存即活跃
    let s = useScheduleStore.getState().project
    expect(s.baseline?.name).toBe('签证基线')
    expect(s.baselines?.map((b) => b.name)).toEqual(['开工基线', '签证基线'])
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('与基线持平') // 与自己比

    const rows = screen.getAllByTestId('sched-baseline-slot')
    fireEvent.click(rows[0].querySelector('input[type=radio]')!) // 切回槽1
    s = useScheduleStore.getState().project
    expect(s.baseline?.name).toBe('开工基线')
    // 漂移摘要跟随：基线 9 → 当前 12，+3 拖后，B/C 推移 2、A 一致
    const panel = screen.getByTestId('sched-baselines-panel').textContent
    expect(panel).toContain('总工期 9 → 12')
    expect(panel).toContain('+3 天，拖后')
    expect(panel).toContain('推移 2')
    expect(panel).toContain('一致 1')
    expect(rows[0].textContent).toContain('当前对比')
    expect(rows[1].textContent).not.toContain('当前对比')
  })

  it('删除活跃槽（Popconfirm）：活跃指针清空（baseline=null），另一槽保留', async () => {
    const p = chainProject()
    const b1 = slotOf('开工基线', 9, CHAIN_SNAP)
    const b2 = slotOf('签证基线', 12, CHAIN_SNAP)
    p.baseline = b1
    p.baselines = [b1, b2]
    renderPanel(p)
    expect(screen.getAllByTestId('sched-baseline-slot')).toHaveLength(2)
    fireEvent.click(screen.getByRole('button', { name: '删除基线开工基线' }))
    await confirmPop()
    const s = useScheduleStore.getState().project
    expect(s.baseline).toBeNull()
    expect(s.baselines?.map((b) => b.name)).toEqual(['签证基线'])
    expect(screen.getAllByTestId('sched-baseline-slot')).toHaveLength(1)
  })

  it('清除对比（clearBaseline）：活跃指针清空但槽位全部保留', async () => {
    const p = chainProject()
    const b1 = slotOf('开工基线', 9, CHAIN_SNAP)
    const b2 = slotOf('签证基线', 12, CHAIN_SNAP)
    p.baseline = b1
    p.baselines = [b1, b2]
    renderPanel(p)
    fireEvent.click(screen.getByRole('button', { name: '清除对比' }))
    await confirmPop()
    const s = useScheduleStore.getState().project
    expect(s.baseline).toBeNull()
    expect(s.baselines?.map((b) => b.name)).toEqual(['开工基线', '签证基线'])
    expect(screen.getAllByTestId('sched-baseline-slot')).toHaveLength(2)
  })

  it('手工旧快照（rows 带旧 es/ef）→ 漂移摘要：总工期漂移数字、推移计数、偏差行', () => {
    const p = chainProject()
    const b = slotOf('评审基线', 6, STALE_SNAP)
    p.baseline = b
    p.baselines = [b]
    renderPanel(p)
    const panel = screen.getByTestId('sched-baselines-panel').textContent
    expect(panel).toContain('评审基线')
    expect(panel).toContain(`保存于 ${T0}`)
    expect(panel).toContain('总工期 6 → 9')
    expect(panel).toContain('+3 天，拖后')
    expect(panel).toContain('推移 2') // B/C 相对旧排程推移
    expect(panel).toContain('一致 1') // A 未动
    expect(panel).toContain('第 0→3 工作日') // B 行：基线 es 0 → 当前 es 3
  })

  it('槽位上限：3 槽时提示「将淘汰最旧槽位」；保存新基线按 FIFO 淘汰最旧', () => {
    const p = chainProject()
    p.baselines = [slotOf('基线A', 9, CHAIN_SNAP), slotOf('基线B', 9, CHAIN_SNAP), slotOf('基线C', 9, CHAIN_SNAP)]
    p.baseline = null
    renderPanel(p)
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('将淘汰最旧槽位')
    saveAs('基线D')
    const s = useScheduleStore.getState().project
    expect(s.baselines?.map((b) => b.name)).toEqual(['基线B', '基线C', '基线D'])
    expect(s.baseline?.name).toBe('基线D')
  })

  it('更新基线：按活跃名原位更新（槽位数不变），更新后漂移归零', () => {
    const p = chainProject()
    const b = slotOf('评审基线', 6, STALE_SNAP)
    p.baseline = b
    p.baselines = [b]
    renderPanel(p)
    fireEvent.click(screen.getByRole('button', { name: '更新基线' }))
    const s = useScheduleStore.getState().project
    expect(s.baselines).toHaveLength(1)
    expect(s.baseline?.name).toBe('评审基线')
    expect(s.baseline?.duration).toBe(9)
    expect(screen.getByTestId('sched-baselines-panel').textContent).toContain('与基线持平')
  })
})

describe('BaselinesPanel 签证台账 CSV 导出（v4.138）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.schedule.v1')
    useScheduleStore.setState({ project: chainProject(), selectedId: null, hydrated: true, sync: 'saved', savedAt: null, syncError: null, past: [], future: [] })
  })

  /** 点击导出按钮并捕获交给 downloadBlob 的 Blob 与 <a download> 文件名（jsdom 无 blob URL，先 stub） */
  async function clickLedgerAndCapture(): Promise<{ blob: Blob; filename: string }> {
    let blob: Blob | null = null
    let filename = ''
    const create = vi.fn((b: Blob) => { blob = b; return 'blob:ledger' })
    const revoke = vi.fn()
    vi.stubGlobal('URL', { ...globalThis.URL, createObjectURL: create, revokeObjectURL: revoke })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function mock(this: HTMLAnchorElement) {
      filename = this.download
    })
    try {
      fireEvent.click(screen.getByTestId('sched-baseline-ledger'))
    } finally {
      vi.unstubAllGlobals()
      click.mockRestore()
    }
    expect(create).toHaveBeenCalledTimes(1)
    expect(revoke).toHaveBeenCalledTimes(1) // downloadBlob 用完即回收 URL
    if (!blob) throw new Error('未捕获到下载 Blob')
    return { blob, filename }
  }

  it('无槽位时隐藏「导出台账」按钮', () => {
    renderPanel(chainProject())
    expect(screen.queryByTestId('sched-baseline-ledger')).toBeNull()
  })

  it('有槽位时按钮在位；点击下载 CSV：文件名/BOM/表头/每槽一行（漂移=当前-基线，正=拖后）', async () => {
    const p = chainProject()
    p.baselines = [slotOf('开工基线', 9, CHAIN_SNAP), slotOf('签证基线', 6, STALE_SNAP)]
    p.baseline = p.baselines[0]
    renderPanel(p)
    expect(screen.getByTestId('sched-baseline-ledger').textContent).toContain('导出台账')

    const { blob, filename } = await clickLedgerAndCapture()
    expect(filename).toBe('链式样板-签证台账.csv')
    expect(blob.type).toBe('text/csv;charset=utf-8')
    // \ufeff BOM：Excel 中文乱码防线。blob.text() 会按规范剥掉 BOM，改查原始字节（EF BB BF）
    const bytes = new Uint8Array(await blob.arrayBuffer())
    expect(Array.from(bytes.slice(0, 3))).toEqual([0xef, 0xbb, 0xbf])
    const text = await blob.text()
    expect(text).toContain('基线名称,保存时间,基线总工期(天),当前总工期(天),总工期漂移(天)')
    // 每槽一行：开工基线 9 vs 当前 9 漂移 0；签证基线 6 vs 当前 9 漂移 +3（拖后）
    expect(text).toContain('开工基线,2026-09-01 08:00,9,9,0')
    expect(text).toContain('签证基线,2026-09-01 08:00,6,9,3')
    // 末行不汇总：最后一行是最后一个槽位的数据行
    const lines = text.replace(/^\ufeff/, '').trimEnd().split('\r\n')
    expect(lines).toHaveLength(3)
    expect(lines[2]).toBe('签证基线,2026-09-01 08:00,6,9,3')
  })
})
