import { describe, expect, it } from 'vitest'
import { aoaGraphFromBuild, buildAoa, AOA_COL_W, AOA_MARGIN, AOA_ROW_H } from './aoa'
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

describe('buildAoa 行距自适应（v4.159：宽高比超限时放大行距，治「又矮又长」）', () => {
  it('长计划低并行度：行距按目标纵横比放大（等距且 > AOA_ROW_H，不超 2×）', () => {
    // A/B 并行各 40 天：同列多事件，w=4480 远超基础高度 ×7 → 行距放大
    const g = buildAoa([t('A', 40), t('B', 40)], [])
    expect(g.ok).toBe(true)
    const col = g.nodes.filter((n) => n.es === 40)
    expect(col.length).toBeGreaterThanOrEqual(2)
    const ys = [...new Set(col.map((n) => n.y))].sort((a, b) => a - b)
    expect(ys.length).toBeGreaterThanOrEqual(2)
    // 等距（横幅内逐行等距），且行距已放大、不超封顶
    const pitch = ys[1] - ys[0]
    for (let i = 1; i < ys.length; i++) expect(ys[i] - ys[i - 1]).toBe(pitch)
    expect(pitch).toBeGreaterThan(AOA_ROW_H)
    expect(pitch).toBeLessThanOrEqual(AOA_ROW_H * 2)
  })

  it('小图（纵横比未超限）行距保持基础值', () => {
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    const col = g.nodes.filter((n) => n.es === 9) // endC 与 T 同列两行
    expect(col).toHaveLength(2)
    expect(Math.abs(col[0].y - col[1].y)).toBe(AOA_ROW_H)
  })

  it('放大封顶 2×AOA_ROW_H：超长单链不再无限拉高', () => {
    const g = buildAoa([t('A', 200)], [])
    const col = g.nodes.filter((n) => n.es === 200) // endA 与 T 同列
    expect(col).toHaveLength(2)
    expect(Math.abs(col[0].y - col[1].y)).toBe(AOA_ROW_H * 2)
  })
})

describe('buildAoa 分级横幅（v4.160：一级分组=横幅，二级事件只在横幅内）', () => {
  it('两个分部：组内事件 band 按分组序，y 区间不打架（band 顶=min y−ROW_H/2 不相交）', () => {
    const g = buildAoa([
      { id: 'G1', name: '管网', duration: 0, level: 0, progress: 0 },
      t('A', 3), t('B', 2),
      { id: 'G2', name: '场平', duration: 0, level: 0, progress: 0 },
      t('C', 4), t('D', 1),
    ], [l('A', 'B'), l('C', 'D')])
    expect(g.ok).toBe(true)
    const bandOf = (anchor: string): number => {
      const n = g.nodes.find((x) => x.anchor === anchor)
      if (!n) throw new Error(`无锚点 ${anchor}`)
      return n.band
    }
    // 组内专属事件归各分组横幅（跨组共享的 S/T 取最小 band=归前组，不在此断言）
    expect(bandOf('end:A')).toBe(0)
    expect(bandOf('end:B')).toBe(0)
    expect(bandOf('end:C')).toBe(1)
    expect(bandOf('end:D')).toBe(1)
    // 两横幅 y 区间不打架（band 间空 1 行档：首行差=2×ROW_H）
    const yOf = (anchor: string): number => g.nodes.find((x) => x.anchor === anchor)!.y
    const b0Max = Math.max(yOf('end:A'), yOf('end:B'))
    const b1Min = Math.min(yOf('end:C'), yOf('end:D'))
    expect(b1Min - b0Max).toBeGreaterThanOrEqual(AOA_ROW_H * 2)
  })

  it('无分组行：全部事件单一横幅（退化=旧行为），共享界点归前组', () => {
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')])
    expect(new Set(g.nodes.map((n) => n.band))).toEqual(new Set([0]))
    // 跨组共享：B 挂前组开始事件（S）→ S 归前组
    const g2 = buildAoa([
      { id: 'G1', name: '一', duration: 0, level: 0, progress: 0 },
      t('A', 3),
      { id: 'G2', name: '二', duration: 0, level: 0, progress: 0 },
      t('C', 2),
    ], [l('A', 'C', 'SS', 0)])
    const s = g2.nodes.find((n) => n.anchor === 'S')!
    expect(s.band).toBe(0)
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

// ── 斑马复刻 N1：持久点-箭图元覆盖编号/层/身份 ──────────────────
// 契约：Project.aoa.dots[].id **就是**锚点键（S / T / start:<taskId> / end:<taskId>）。
// 命中者用其 num/layer；未命中者补号并跳过占用，编号必须唯一。
describe('buildAoa · 持久图元（斑马 dots 口径）', () => {
  const chain = () => buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')])

  it('未提供 persisted → numSource=derived，且不带 dotId/layer（旧行为零变化）', () => {
    const g = chain()
    expect(g.numSource).toBe('derived')
    expect(g.nodes.every((n) => n.dotId === undefined && n.layer === undefined)).toBe(true)
  })

  it('全命中 → 编号/层/身份取自持久图元，数组按编号排序', () => {
    const base = chain()
    // 反着给编号：验证"编号以持久为准"而不是派生
    const persisted = {
      dots: base.nodes.map((n, i) => ({ id: n.anchor, num: base.nodes.length - i, layer: i })),
      arrows: [],
    }
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { persisted })
    expect(g.numSource).toBe('persisted')
    expect(g.nodes.map((n) => n.num)).toEqual([...base.nodes.map((_, i) => base.nodes.length - i)].sort((p, q) => p - q))
    // 排序后数组首元素编号最小
    expect(g.nodes[0].num).toBe(1)
    // dotId 与 anchor 同值（契约），layer 逐点生效
    for (const n of g.nodes) {
      expect(n.dotId).toBe(n.anchor)
      expect(typeof n.layer).toBe('number')
    }
    // 时间与几何不受编号覆盖影响（x 仍由 ES 决定）
    const byAnchor = new Map(g.nodes.map((n) => [n.anchor, n]))
    expect(byAnchor.get('S')!.x).toBe(AOA_MARGIN)
  })

  it('部分命中 → 未命中者补号并跳过已占用编号（编号唯一）', () => {
    const base = chain()
    // 只把起点钉成编号 3，其余让算法补
    const persisted = { dots: [{ id: 'S', num: 3 }], arrows: [] }
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { persisted })
    expect(g.numSource).toBe('persisted')
    const nums = g.nodes.map((n) => n.num).sort((p, q) => p - q)
    expect(new Set(nums).size).toBe(nums.length)          // 互不重复
    const s = g.nodes.find((n) => n.anchor === 'S')!
    expect(s.num).toBe(3)
    expect(s.dotId).toBe('S')
    expect(base.nodes).toHaveLength(g.nodes.length)        // 节点集合不变
  })

  it('一个都没命中 → 退回派生编号（脏文件不应扰动现有编号）', () => {
    const persisted = { dots: [{ id: 'end:ghost', num: 9 }], arrows: [] }
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { persisted })
    expect(g.numSource).toBe('derived')
    expect(g.nodes.every((n) => n.dotId === undefined)).toBe(true)
  })

  it('箭线 id 标注：两端事件都有持久身份时按 from→to 命中', () => {
    const base = chain()
    const persisted = {
      dots: base.nodes.map((n) => ({ id: n.anchor, num: n.num })),
      arrows: base.edges.map((e, i) => ({
        id: `x${i}`,
        from: base.nodes.find((n) => n.id === e.from)!.anchor,
        to: base.nodes.find((n) => n.id === e.to)!.anchor,
        kind: e.kind,
      })),
    }
    const g = buildAoa([t('A', 3), t('B', 2)], [l('A', 'B')], { persisted })
    expect(g.edges.filter((e) => e.arrowId).length).toBe(g.edges.length)
  })
})

// ── 斑马复刻 N1 生产侧：aoaGraphFromBuild 固化 ──────────────────
describe('aoaGraphFromBuild：派生图 → 持久图元（首次手动布点时固化）', () => {
  const mk = () => buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])

  it('固化后回灌：编号不变、numSource=persisted、身份齐全', () => {
    const g = mk()
    const persisted = aoaGraphFromBuild(g)
    expect(persisted.dots.length).toBe(g.nodes.length)
    expect(persisted.arrows.length).toBe(g.edges.length)
    expect(persisted.beginDotId).toBe('S')
    expect(persisted.dots.find((d) => d.id === 'S')!.isBegin).toBe(true)
    expect(persisted.dots.find((d) => d.id === 'T')!.isEnd).toBe(true)

    const again = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')], { persisted })
    expect(again.numSource).toBe('persisted')
    const map = (x: typeof g) => new Map(x.nodes.map((n) => [n.anchor, n.num]))
    expect(map(again)).toEqual(map(g))
  })

  it('固化后插入新任务：老事件编号不变，新事件补号', () => {
    const g1 = mk()
    const persisted = aoaGraphFromBuild(g1)
    const oldNums = new Map(g1.nodes.map((n) => [n.anchor, n.num]))
    // 尾巴上加一条 C→D
    const g2 = buildAoa([t('A', 3), t('B', 2), t('C', 4), t('D', 1)], [l('A', 'B'), l('B', 'C'), l('C', 'D')], { persisted })
    expect(g2.numSource).toBe('persisted')
    for (const [anchor, num] of oldNums) {
      const n = g2.nodes.find((x) => x.anchor === anchor)
      if (n) expect(n.num).toBe(num) // 老锚点编号稳定（斑马 dots 口径）
    }
    const nums = g2.nodes.map((n) => n.num)
    expect(new Set(nums).size).toBe(nums.length) // 新老混排仍唯一
  })

  it('复用已有持久层号（prev.layer 优先于当前 band）', () => {
    const g = mk()
    const prev = { dots: [{ id: 'S', num: 1, layer: 7 }], arrows: [] }
    const out = aoaGraphFromBuild(g, prev)
    expect(out.dots.find((d) => d.id === 'S')!.layer).toBe(7)
  })

  it('箭线 id 用两端锚点拼（跨拓扑稳定，不用内部自增 id）', () => {
    const out = aoaGraphFromBuild(mk())
    expect(out.arrows.every((a) => a.id.includes('>'))).toBe(true)
    expect(new Set(out.arrows.map((a) => a.id)).size).toBe(out.arrows.length)
  })
})


// ── 逻辑网络图模式（斑马 5 模式，一次点亮批3）：x=拓扑深度，与 es 解耦 ──
describe('buildAoa logic 模式（一次点亮批3）', () => {
  it('x=拓扑深度：S 第 0 列、事件逐列递进；timescaled 保持 x=es 旧口径', () => {
    const tasks = [t('A', 3), t('B', 2)]
    const links = [l('A', 'B')]
    const logic = buildAoa(tasks, links, { mode: 'logic' })
    const depthOf = (n: { x: number }) => Math.round((n.x - AOA_MARGIN) / AOA_COL_W)
    const sL = logic.nodes.find((n) => n.anchor === 'S')!
    const tL = logic.nodes.find((n) => n.anchor === 'T')!
    // 深度单调：S 第 0 列，中间事件递进，T 收尾列（列距=拓扑步数，与 es 无关）
    expect(depthOf(sL)).toBe(0)
    const mids = logic.nodes.filter((n) => n.anchor !== 'S' && n.anchor !== 'T')
    expect(mids.length).toBeGreaterThan(0)
    for (const m of mids) {
      expect(depthOf(m)).toBeGreaterThan(depthOf(sL))
      expect(depthOf(m)).toBeLessThan(depthOf(tL))
    }
    expect(depthOf(tL)).toBeGreaterThan(0)
    // 同一拓扑在时标口径：x=es×列宽（旧口径不变）；es=3 的点在逻辑图里收到拓扑列
    const ts = buildAoa(tasks, links)
    const sTs = ts.nodes.find((n) => n.anchor === 'S')!
    const midTs = ts.nodes.find((n) => n.anchor !== 'S' && n.anchor !== 'T')!
    expect(sTs.x).toBe(AOA_MARGIN)
    expect(midTs.x).toBe(AOA_MARGIN + midTs.es * AOA_COL_W)
    const midL = logic.nodes.find((n) => n.anchor === midTs.anchor)!
    expect(Math.round((midL.x - AOA_MARGIN) / AOA_COL_W)).not.toBe(midTs.es)
  })

  it('logic 宽度按最大深度收窄（不再被工期拉长）', () => {
    // 长链 10 天：时标宽=12 列宽，逻辑宽=4 列宽
    const tasks = [t('A', 10), t('B', 10)]
    const links = [l('A', 'B')]
    const ts = buildAoa(tasks, links)
    const logic = buildAoa(tasks, links, { mode: 'logic' })
    const tsW = Math.max(...ts.nodes.map((n) => n.x)) + AOA_MARGIN + AOA_COL_W / 2
    const lgW = Math.max(...logic.nodes.map((n) => n.x)) + AOA_MARGIN + AOA_COL_W / 2
    expect(tsW).toBeGreaterThan(lgW)
    expect(lgW).toBeLessThanOrEqual(AOA_MARGIN * 2 + 4 * AOA_COL_W)
  })

  it('logic 模式环形兜底：不可达节点落 0 列不抛', () => {
    // AOA 不应有环（引擎拒绝），但派生图对脏输入需不抛——兜底 0 列
    const tasks = [t('A', 2)]
    const links = [l('A', 'A')]
    expect(() => buildAoa(tasks, links, { mode: 'logic' })).not.toThrow()
  })
})
