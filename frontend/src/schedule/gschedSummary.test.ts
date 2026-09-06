import { describe, expect, it } from 'vitest'
import { isScheduleFilePath, parseSchedSummary, SCHEDULE_FILE_PATH } from './gschedSummary'
import type { SchedLink, SchedProject, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}

function project(over: Partial<SchedProject> = {}): SchedProject {
  return {
    name: '测试工程',
    startDate: '2026-01-05', // 周一
    tasks: [t('A', 3), t('B', 2)],
    links: [l('A', 'B')],
    ...over,
  }
}

describe('isScheduleFilePath 路径识别', () => {
  it('当前计划路径与反斜杠变体命中', () => {
    expect(isScheduleFilePath(SCHEDULE_FILE_PATH)).toBe(true)
    expect(isScheduleFilePath('进度计划\\当前计划.gsched.json')).toBe(true)
  })
  it('大小写不敏感；其他扩展名不命中', () => {
    expect(isScheduleFilePath('计划.GSCHED.JSON')).toBe(true)
    expect(isScheduleFilePath('a.json')).toBe(false)
    expect(isScheduleFilePath('a.gsched.jsonx')).toBe(false)
    expect(isScheduleFilePath('进度计划/说明.md')).toBe(false)
  })
})

describe('parseSchedSummary 摘要解析', () => {
  it('有效计划：CPM 摘要与竣工日期（周一开工 5 工作日 → 下周一）', () => {
    const s = parseSchedSummary(JSON.stringify(project()))
    expect(s).not.toBeNull()
    expect(s!.ok).toBe(true)
    expect(s!.duration).toBe(5)
    expect(s!.taskCount).toBe(2)
    expect(s!.groupCount).toBe(0)
    expect(s!.criticalCount).toBe(2)
    expect(s!.linkCount).toBe(1)
    expect(s!.finishDate).toBe('2026-01-12')
    expect(s!.baseline).toBeNull()
    expect(s!.deadline).toBeNull()
  })

  it('分组行计入 groupCount 不计 taskCount；里程碑计数', () => {
    const s = parseSchedSummary(JSON.stringify(project({
      tasks: [{ id: 'g', name: '分组', duration: 0, level: 0, progress: 0 }, t('A', 3, { isMilestone: false }), t('M', 0, { isMilestone: true })],
      links: [],
    })))
    expect(s!.groupCount).toBe(1)
    expect(s!.taskCount).toBe(2)
    expect(s!.milestoneCount).toBe(1)
  })

  it('循环依赖：ok=false，竣工/基线/倒排全 null', () => {
    const s = parseSchedSummary(JSON.stringify(project({ links: [l('A', 'B'), l('B', 'A')] })))
    expect(s!.ok).toBe(false)
    expect(s!.finishDate).toBeNull()
    expect(s!.baseline).toBeNull()
    expect(s!.deadline).toBeNull()
  })

  it('基线与目标竣工进入摘要', () => {
    const s = parseSchedSummary(JSON.stringify(project({
      baseline: { name: '基线一', savedAt: '2026-01-05 08:00', duration: 4, rows: { A: { name: 'A', es: 0, ef: 3, dur: 3, critical: true } } },
      deadline: '2026-01-08', // 周四：目标 4 工作日 < 当前 5 → 超期
    })))
    expect(s!.baseline).toMatchObject({ name: '基线一', duration: 4, drift: 1 })
    expect(s!.deadline).toMatchObject({ date: '2026-01-08', targetWorkdays: 4, feasible: false, overrun: 1 })
  })

  it('坏 JSON / 非工程形状返回 null（预览回落文本视图）', () => {
    expect(parseSchedSummary('not json{')).toBeNull()
    expect(parseSchedSummary('["数组"]')).toBeNull()
    expect(parseSchedSummary(JSON.stringify({ foo: 1 }))).toBeNull()
  })

  it('空任务计划：ok=true、taskCount=0、无竣工日期', () => {
    const s = parseSchedSummary(JSON.stringify(project({ tasks: [], links: [] })))
    expect(s!.ok).toBe(true)
    expect(s!.taskCount).toBe(0)
    expect(s!.finishDate).toBeNull()
  })
})
