import { describe, expect, it } from 'vitest'
import { buildAoa, AOA_COL_W, AOA_MARGIN } from './aoa'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}

describe('buildAoa 教材画法', () => {
  it('顺序链 A(3)→B(2)→C(4)：FS 合并成一条链，仅终点汇出一条虚工作', () => {
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    expect(g.ok).toBe(true)
    // 5 个事件：起点、A 完成(=B 开始)、B 完成(=C 开始)、C 完成、终点
    expect(g.nodes).toHaveLength(5)
    // 4 条箭线：3 条实工作 + 1 条汇出虚工作
    expect(g.edges).toHaveLength(4)
    const tasks = g.edges.filter((e) => e.kind === 'task')
    expect(tasks.map((e) => e.dur)).toEqual([3, 2, 4])
    expect(g.edges.filter((e) => e.kind === 'dummy')).toHaveLength(1)
    // 全部关键
    expect(g.edges.every((e) => e.critical)).toBe(true)
    // 事件最早时间
    const byNum = new Map(g.nodes.map((n) => [n.num, n]))
    expect(byNum.get(1)!.es).toBe(0)
    expect(byNum.get(5)!.es).toBe(9)
  })

  it('经典虚工作案例 A→B/C→D：B、C 汇入 D 需两条虚工作', () => {
    const g = buildAoa(
      [t('A', 2), t('B', 6), t('C', 1), t('D', 5)],
      [l('A', 'B'), l('A', 'C'), l('B', 'D'), l('C', 'D')],
    )
    expect(g.ok).toBe(true)
    expect(g.nodes).toHaveLength(7)
    expect(g.edges.filter((e) => e.kind === 'task')).toHaveLength(4)
    const dummies = g.edges.filter((e) => e.kind === 'dummy')
    expect(dummies).toHaveLength(3) // E_B→S_D、E_C→S_D、E_D→END
    // 关键线路 = A→B→D（经虚工作 E_B→S_D），C 支路有时差
    const critIds = g.edges.filter((e) => e.critical).map((e) => e.taskId ?? `dummy:${e.from}>${e.to}`)
    expect(critIds).toContain('A')
    expect(critIds).toContain('B')
    expect(critIds).toContain('D')
    expect(critIds).not.toContain('C')
  })

  it('并行起算：两任务共用起点事件（一始一终），无虚工作拆分', () => {
    const g = buildAoa([t('A', 3), t('B', 2)], [])
    expect(g.ok).toBe(true)
    const starts = g.edges.filter((e) => e.kind === 'task').map((e) => e.from)
    expect(new Set(starts).size).toBe(1) // 同一出发事件
    // 汇入同一终点事件（各自一条虚工作）
    const ends = g.edges.filter((e) => e.kind === 'dummy').map((e) => e.to)
    expect(new Set(ends).size).toBe(1)
  })

  it('SS(lag=0) 合并：B 与 A 共起点（开始事件复用）', () => {
    const g = buildAoa([t('A', 5), t('B', 2)], [l('A', 'B', 'SS', 0)])
    expect(g.ok).toBe(true)
    const edgeA = g.edges.find((e) => e.taskId === 'A')!
    const edgeB = g.edges.find((e) => e.taskId === 'B')!
    expect(edgeB.from).toBe(edgeA.from)
    expect(edgeB.from).not.toBe(edgeA.to) // 不是 FS 合并
  })

  it('SS+时距：虚工作从 A 开始事件指向 B 开始事件并携带时距', () => {
    const g = buildAoa([t('A', 5), t('B', 2)], [l('A', 'B', 'SS', 3)])
    expect(g.ok).toBe(true)
    const edgeB = g.edges.find((e) => e.taskId === 'B')!
    const dummy = g.edges.find((e) => e.kind === 'dummy' && e.to === edgeB.from)!
    expect(dummy).toBeDefined()
    expect(dummy.dur).toBe(3)
    expect(dummy.label).toContain('+3')
  })
})

describe('buildAoa 分级计划（v4.142：一级汇总箭线界点衔接二级子网络）', () => {
  it('分组行=一条汇总箭线：从子网络开始界点事件连到完成界点事件，不新增事件、不扰动正逆推', () => {
    const tasks = [
      { id: 'G1', name: '前期准备', duration: 0, level: 0, progress: 0 },
      t('A', 3), t('B', 2),
    ]
    const g1 = buildAoa(tasks, [l('A', 'B')])
    const g2 = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')])
    expect(g1.ok).toBe(true)
    // 汇总箭线不新增事件、不改变正逆推：事件集与纯二级网络完全一致
    expect(g1.nodes).toHaveLength(g2.nodes.length)
    const esOf = (g: typeof g1) => new Map(g.nodes.map((n) => [n.anchor, n.es]))
    const lsOf = (g: typeof g1) => new Map(g.nodes.map((n) => [n.anchor, n.ls]))
    for (const [a, es] of esOf(g1)) expect(es).toBe(esOf(g2).get(a))
    for (const [a, ls] of lsOf(g1)) expect(ls).toBe(lsOf(g2).get(a))
    // 恰一条汇总箭线：A 开始事件 → B 完成事件，跨子网络（时长 5=界点时间差）
    const sums = g1.edges.filter((e) => e.kind === 'summary')
    expect(sums).toHaveLength(1)
    const sEdge = sums[0]
    expect(sEdge.taskId).toBe('G1')
    expect(sEdge.critical).toBe(false) // 不参与关键判定
    const nodeById = new Map(g1.nodes.map((n) => [n.id, n]))
    const bStart = nodeById.get(sEdge.from)!
    const bEnd = nodeById.get(sEdge.to)!
    expect(bStart.es).toBe(0) // 开始界点=A 的开始（S 事件）
    expect(bEnd.es).toBe(5) // 完成界点=B 的完成（最早时间=子网络工期）
    expect(sEdge.dur).toBe(5)
    expect(g1.taskEdge['G1']).toBe(sEdge.id) // 分组行可经 taskEdge 命中汇总箭线
    // 二级实/虚箭线不含分组行
    expect(g1.edges.some((e) => e.kind !== 'summary' && e.taskId === 'G1')).toBe(false)
  })

  it('空分组（无子级）不产汇总箭线；仅分组行 → 空图', () => {
    // 轮廓语义：[G1, G2, A] → G1 无子级、G2 拥有 A
    const g = buildAoa([
      { id: 'G1', name: '空分组', duration: 0, level: 0, progress: 0 },
      { id: 'G2', name: '有子分组', duration: 0, level: 0, progress: 0 },
      t('A', 2),
    ], [])
    expect(g.ok).toBe(true)
    const sums = g.edges.filter((e) => e.kind === 'summary')
    expect(sums).toHaveLength(1) // 只有 G2（拥有 A）出汇总箭线
    expect(sums[0].taskId).toBe('G2')
    const only = buildAoa([{ id: 'G1', name: '空分组', duration: 0, level: 0, progress: 0 }], [])
    expect(only.ok).toBe(true)
    expect(only.nodes).toHaveLength(0)
  })

  it('跨分组搭接把各子网络连成一张图；一级箭线共享界点事件（一始一终不被破坏）', () => {
    const tasks = [
      { id: 'G1', name: '一期', duration: 0, level: 0, progress: 0 },
      t('A', 3), t('B', 2),
      { id: 'G2', name: '二期', duration: 0, level: 0, progress: 0 },
      t('C', 4), t('D', 1),
    ]
    const g = buildAoa(tasks, [l('A', 'B'), l('B', 'D'), l('C', 'D')])
    expect(g.ok).toBe(true)
    expect(g.edges.filter((e) => e.kind === 'task')).toHaveLength(4)
    const sums = g.edges.filter((e) => e.kind === 'summary')
    expect(sums).toHaveLength(2) // G1、G2 各一条
    const byTask = new Map(sums.map((e) => [e.taskId, e]))
    expect(byTask.get('G1')!.dur).toBe(5) // A(3)+B(2)：界点 S(0)→B 完成(5)
    // C 无前置 → 汇入 S(时刻 0)；D 完成事件 es=6 → G2 界点时间差=6（跨度口径）
    expect(byTask.get('G2')!.dur).toBe(6)
    // 一始一终：S/T 各恰一个；总工期=max(6, 5)=6 不被汇总箭线扰动
    const sNodes = g.nodes.filter((n) => n.anchor === 'S')
    const tNodes = g.nodes.filter((n) => n.anchor === 'T')
    expect(sNodes).toHaveLength(1)
    expect(tNodes).toHaveLength(1)
    const last = g.nodes.reduce((a, b) => (b.num > a.num ? b : a))
    expect(last.es).toBe(6)
  })
})

describe('buildAoa 结构不变量', () => {
  const cases: { name: string; tasks: SchedTask[]; links: SchedLink[] }[] = [
    { name: '链式', tasks: [t('A', 3), t('B', 2), t('C', 4)], links: [l('A', 'B'), l('B', 'C')] },
    {
      name: '菱形+混合搭接',
      tasks: [t('A', 10), t('B', 6), t('C', 4), t('D', 3), t('E', 5)],
      links: [l('A', 'B'), l('A', 'C', 'SS', 2), l('B', 'D'), l('C', 'D', 'FF', 1), l('D', 'E'), l('B', 'E')],
    },
    { name: '多起多终', tasks: [t('A', 2), t('B', 5), t('C', 3)], links: [l('A', 'C')] },
  ]
  for (const c of cases) {
    it(`${c.name}：节点编号 i<j、每任务恰一条实箭线、无重复虚箭线`, () => {
      const g = buildAoa(c.tasks, c.links)
      expect(g.ok, g.error).toBe(true)
      const num = new Map(g.nodes.map((n) => [n.id, n.num]))
      for (const e of g.edges) {
        expect(num.get(e.from)!, `${e.from}>${e.to}`).toBeLessThan(num.get(e.to)!)
      }
      expect(g.edges.filter((e) => e.kind === 'task')).toHaveLength(c.tasks.length)
      expect(Object.keys(g.taskEdge)).toHaveLength(c.tasks.length)
      const dummyKeys = g.edges.filter((e) => e.kind === 'dummy').map((e) => `${e.from}>${e.to}`)
      expect(new Set(dummyKeys).size).toBe(dummyKeys.length)
    })
  }
})

describe('buildAoa 健壮性', () => {
  it('循环依赖拒绝', () => {
    const g = buildAoa([t('A', 1), t('B', 1)], [l('A', 'B'), l('B', 'A')])
    expect(g.ok).toBe(false)
    expect(g.error).toContain('循环依赖')
  })

  it('空项目返回空图', () => {
    const g = buildAoa([], [])
    expect(g.ok).toBe(true)
    expect(g.nodes).toHaveLength(0)
    expect(g.edges).toHaveLength(0)
  })

  it('里程碑（0 工期）参与转换不炸', () => {
    const g = buildAoa([t('A', 4), t('M', 0, { isMilestone: true })], [l('A', 'M')])
    expect(g.ok).toBe(true)
    const m = g.edges.find((e) => e.taskId === 'M')!
    expect(m.dur).toBe(0)
  })
})

describe('buildAoa 真时标布局（v4.130 刀H G4 前提）', () => {
  it('x=AOA_MARGIN+es×AOA_COL_W：es 线性，时距空洞成比例（波形线才读得出真实自由时差）', () => {
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    for (const n of g.nodes) {
      expect(n.x, `节点 ${n.id} es=${n.es}`).toBe(AOA_MARGIN + n.es * AOA_COL_W)
    }
    // end:B 事件 es=5：x 随 es 而非列排名（旧版排名会把 0/3/5 压成等距 0/1/2）
    expect(g.nodes.some((n) => n.x === AOA_MARGIN + 5 * AOA_COL_W)).toBe(true)
  })
})

describe('buildAoa 逆推锚点（v4.133 修正：total 取自正推后）', () => {
  it('长链：逆推自计算工期起，全部事件 ls≥0 且 T 事件 es=ls=计算工期', () => {
    // 修正前：total 用注册初始 es（=单任务工期 4），逆推整体平移 -5 出假负时差
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    expect(g.ok).toBe(true)
    expect(Math.max(...g.nodes.map((n) => n.es))).toBe(9)
    const endT = g.nodes.find((n) => n.anchor === 'T')!
    expect(endT.es).toBe(9)
    expect(endT.ls).toBe(9)
    expect(Math.min(...g.nodes.map((n) => n.ls)), '无假负时差').toBeGreaterThanOrEqual(0)
  })

  it('planFinish < 计算工期：逆推真从 Tp 锚起（S 事件 ls=Tp−Tc）', () => {
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { planFinish: 4 })
    const startS = g.nodes.find((n) => n.anchor === 'S')!
    expect(startS.ls).toBe(-1) // 4 − 5
    // 关键箭线（真浮时 0）：A、B 与汇出虚工作（虚工作可为关键线路一段）
    const crit = g.edges.filter((e) => e.critical)
    expect(crit).toHaveLength(3)
    expect(crit.map((e) => e.taskId)).toContain('A')
    expect(crit.map((e) => e.taskId)).toContain('B')
  })

  it('planFinish ≥ 计算工期：不收紧，逆推仍自 Tc 起（ls(S)=0）', () => {
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { planFinish: 8 })
    const startS = g.nodes.find((n) => n.anchor === 'S')!
    expect(startS.ls).toBe(0)
    expect(Math.min(...g.nodes.map((n) => n.ls))).toBeGreaterThanOrEqual(0)
  })
})
