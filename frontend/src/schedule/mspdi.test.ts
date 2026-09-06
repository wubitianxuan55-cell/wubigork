import { describe, expect, it } from 'vitest'
import { buildProjectXml, parseProjectXml, wbsCodes } from './mspdi'
import { computeCpm } from './cpm'
import { makeSampleProject } from './sample'
import type { SchedProject } from './types'

/** 手写最小 Project XML 夹具（对齐 mspdi 口径：DayType 1=周日..7=周六、LinkLag 十分之一分钟） */
const FIXTURE = `<?xml version="1.0" encoding="UTF-8"?>
<Project xmlns="http://schemas.microsoft.com/project">
  <Name> Imported </Name>
  <StartDate>2026-03-02T08:00:00</StartDate>
  <MinutesPerDay>480</MinutesPerDay>
  <Calendars><Calendar><UID>1</UID><Name>Standard</Name><IsBaseCalendar>1</IsBaseCalendar>
    <WeekDays>
      <WeekDay><DayType>1</DayType><DayWorking>0</DayWorking></WeekDay>
      <WeekDay><DayType>2</DayType><DayWorking>1</DayWorking><WorkingTimes><WorkingTime><FromTime>08:00:00</FromTime><ToTime>17:00:00</ToTime></WorkingTime></WorkingTimes></WeekDay>
      <WeekDay><DayType>3</DayType><DayWorking>1</DayWorking></WeekDay>
      <WeekDay><DayType>4</DayType><DayWorking>1</DayWorking></WeekDay>
      <WeekDay><DayType>5</DayType><DayWorking>1</DayWorking></WeekDay>
      <WeekDay><DayType>6</DayType><DayWorking>1</DayWorking></WeekDay>
      <WeekDay><DayType>7</DayType><DayWorking>0</DayWorking></WeekDay>
    </WeekDays>
    <Exceptions>
      <Exception><TimePeriod><FromDate>2026-03-06T08:00:00</FromDate><ToDate>2026-03-06T08:00:00</ToDate></TimePeriod><DayWorking>0</DayWorking></Exception>
      <Exception><TimePeriod><FromDate>2026-04-01T08:00:00</FromDate><ToDate>2026-04-02T08:00:00</ToDate></TimePeriod><DayWorking>0</DayWorking></Exception>
    </Exceptions>
  </Calendar></Calendars>
  <Tasks>
    <Task><UID>1</UID><ID>1</ID><WBS>1</WBS><Name>前期</Name><IsSummary>1</IsSummary><OutlineLevel>1</OutlineLevel><Duration>PT40H0M0S</Duration></Task>
    <Task><UID>2</UID><ID>2</ID><WBS>1.1</WBS><Name>场地平整</Name><OutlineLevel>2</OutlineLevel><Duration>PT40H0M0S</Duration><PercentComplete>50</PercentComplete></Task>
    <Task><UID>3</UID><ID>3</ID><WBS>1.2</WBS><Name>放线</Name><OutlineLevel>2</OutlineLevel><Duration>PT8H0M0S</Duration>
      <PredecessorLink><PredecessorUID>2</PredecessorUID><Type>1</Type><LinkLag>9600</LinkLag><LagFormat>7</LagFormat></PredecessorLink>
    </Task>
    <Task><UID>4</UID><ID>4</ID><Name>开工仪式</Name><OutlineLevel>1</OutlineLevel><Duration>PT0H0M0S</Duration><Milestone>1</Milestone>
      <PredecessorLink><PredecessorUID>3</PredecessorUID><Type>1</Type><LinkLag>0</LinkLag></PredecessorLink>
    </Task>
    <Task><UID>5</UID><ID>5</ID><Name>场地围挡</Name><OutlineLevel>2</OutlineLevel><Duration>PT24H0M0S</Duration><Manual>1</Manual><ManualStart>2026-03-05T08:00:00</ManualStart></Task>
  </Tasks>
</Project>`

describe('parseProjectXml（导入）', () => {
  it('最小夹具：任务/大纲/里程碑/进度/搭接/时距/手动开工全解析', () => {
    const r = parseProjectXml(FIXTURE)
    expect(r.ok, r.error).toBe(true)
    const p = r.project!
    expect(p.name).toBe(' Imported ')
    expect(p.startDate).toBe('2026-03-02')
    // 大纲：summary→0，OutlineLevel≥2→1（两级钳制）
    expect(p.tasks.map((t) => t.level)).toEqual([0, 1, 1, 1, 1])
    expect(p.tasks[0].duration).toBe(0) // 汇总行工期 0
    expect(p.tasks[1].duration).toBe(5) // PT40H = 5 工作日
    expect(p.tasks[2].duration).toBe(1)
    expect(p.tasks[3].isMilestone).toBe(true)
    // FS + 9600 十分之一分钟 = 2 工作日时距
    expect(p.links).toEqual([{ from: 't2', to: 't3', type: 'FS', lag: 2 }, { from: 't3', to: 't4', type: 'FS', lag: 0 }])
    // 手动任务：2026-03-05（周四）= 第 3 个工作日（3/2,3/3,3/4,3/5；3/6 是节假日）
    expect(p.tasks[4].mode).toBe('manual')
    expect(p.tasks[4].manualStart).toBe(3)
    // 日历：周六上班（DayType 7=周六 DayWorking=0 → 周六休；DayType 1=周日休、2..6 上班）
    expect(p.calendar!.workweek).toEqual([1, 2, 3, 4, 5])
    expect(p.calendar!.holidays).toEqual(['2026-03-06', '2026-04-01', '2026-04-02'])
  })

  it('垃圾输入 fail-closed', () => {
    expect(parseProjectXml('not xml').ok).toBe(false)
    expect(parseProjectXml('<root/>').ok).toBe(false)
    const empty = parseProjectXml('<Project><Tasks></Tasks></Project>')
    expect(empty.ok).toBe(false)
    expect(empty.error).toContain('没有可导入的任务')
  })
})

describe('buildProjectXml（导出）', () => {
  it('样例工程导出：WBS/搭接/日历/里程碑按口径生成', () => {
    const p = makeSampleProject()
    const cpm = computeCpm(p.tasks, p.links)
    const xml = buildProjectXml(p, cpm.rows)
    expect(xml).toContain('xmlns="http://schemas.microsoft.com/project"')
    expect(xml).toContain('<IsBaseCalendar>1</IsBaseCalendar>')
    expect(xml).toContain('<Milestone>1</Milestone>') // 交付使用里程碑
    expect(xml).toContain('<LinkLag>48000</LinkLag>') // SS lag 10 天 = 10*4800
    expect(xml).toContain('<WBS>1.1</WBS>')
    expect(xml).toContain('<IsSummary>1</IsSummary>')
    expect(xml).toContain('<PercentComplete>60</PercentComplete>')
  })

  it('WBS 编码：分组 n、叶 n.k', () => {
    const p: SchedProject = {
      name: 'x', startDate: '2026-01-01',
      tasks: [
        { id: 'g', name: 'G', duration: 0, level: 0, progress: 0 },
        { id: 'a', name: 'A', duration: 1, level: 1, progress: 0 },
        { id: 'b', name: 'B', duration: 1, level: 1, progress: 0 },
        { id: 'g2', name: 'G2', duration: 0, level: 0, progress: 0 },
        { id: 'c', name: 'C', duration: 1, level: 1, progress: 0 },
      ],
      links: [],
    }
    expect(wbsCodes(p.tasks)).toEqual(['1', '1.1', '1.2', '2', '2.1'])
  })
})

describe('mspdi 往返（导出→导入 同构）', () => {
  it('样例工程 roundtrip：任务名/层级/工期/搭接/日历逐项一致', () => {
    const p = makeSampleProject()
    const cpm = computeCpm(p.tasks, p.links)
    const xml = buildProjectXml(p, cpm.rows)
    const r = parseProjectXml(xml)
    expect(r.ok, r.error).toBe(true)
    const q = r.project!
    expect(q.tasks.map((t) => [t.name, t.level, t.duration])).toEqual(p.tasks.map((t) => [t.name, t.level, t.duration]))
    // 导入侧 id 重建为 t{行序}（UID=行序+1），按行序映射回原工程后比对搭接
    const remap = (id: string) => p.tasks[Number(id.slice(1)) - 1].id
    const key = (l: { from: string; to: string; type: string; lag: number }) => `${l.from}|${l.to}|${l.type}|${l.lag}`
    expect(q.links.map((l) => key({ ...l, from: remap(l.from), to: remap(l.to) })).sort())
      .toEqual(p.links.map(key).sort())
    expect(q.calendar!.workweek).toEqual([1, 2, 3, 4, 5])
  })
})
