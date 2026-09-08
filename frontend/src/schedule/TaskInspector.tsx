/**
 * schedule/TaskInspector.tsx — 任务检查器（对标 MS Project Task Inspector，纯受控展示件）
 *
 * 回答一个问题：本任务的日期由哪条依赖决定（蒸馏自 ProjectLibre：检查器=驱动依赖
 * +前驱/后继可点击跳转）。数据契约：任务/CPM 行/前驱后继依赖行全部由父级算好传入，
 * 本组件零排程计算，只做「驱动结论」的推导与展示；前驱/后继任务名可点击
 * （onNavigate）跳转。日期口径与项目主口径一致：工作日序号 + startDate +
 * 工作日历 → 日历日期（换算为本文件内的独立小函数，不反向依赖排程引擎）。
 */
import React, { useMemo } from 'react'
import { Button, Drawer, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { FlagOutlined, InfoCircleOutlined, LockOutlined, ThunderboltOutlined } from '@ant-design/icons'
import type { SchedLink, SchedTask, TaskCpm } from './types'

/** 依赖行（前驱/后继通用）：搭接关系与对侧任务信息由父级按视角算好 */
export interface InspectorDepRow {
  link: SchedLink
  /** 前驱表=link.from；后继表=link.to */
  otherId: string
  otherName: string
  freeSlack: number
  /** 该依赖是否在驱动本任务日期 */
  driving: boolean
}

export interface TaskInspectorProps {
  open: boolean
  task: SchedTask | null
  row: TaskCpm | null
  wbs?: string
  predecessors: InspectorDepRow[]
  successors: InspectorDepRow[]
  /** 项目开工日 YYYY-MM-DD（工作日序号换算日历日期的锚点） */
  startDate: string
  /** 工作日历（缺省周一~五；workweek 为 JS getDay 口径 0=周日..6=周六） */
  calendar?: { workweek: number[]; holidays: string[] }
  onClose: () => void
  onNavigate: (taskId: string) => void
}

/** 默认工作周：周一~五（与 calendar.ts 的 DEFAULT_CALENDAR 同口径） */
const DEFAULT_WORKWEEK = [1, 2, 3, 4, 5]
/** 日期扫描上限（天）：防呆，与 calendar.ts 的 SCAN_LIMIT 同口径 */
const SCAN_LIMIT = 3650

function parseISO(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y || 1970, (m || 1) - 1, d || 1))
}

function isoOf(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${p(d.getUTCMonth() + 1)}-${p(d.getUTCDate())}`
}

/**
 * 工作日序号 → 日历日期：从开工日起顺延数第 idx 个工作日（idx=0 即开工日当日；
 * 开工日恰为非工作日自动顺延；周末按 workweek、节假日例外命中即跳过）。
 */
function wdToISO(startISO: string, idx: number, cal?: { workweek: number[]; holidays: string[] }): string {
  const start = parseISO(startISO)
  if (!Number.isFinite(idx) || idx < 0) return isoOf(start)
  const workweek = cal?.workweek && cal.workweek.length > 0 ? cal.workweek : DEFAULT_WORKWEEK
  const holidays = cal?.holidays ?? []
  let cur = new Date(start)
  let found = -1
  for (let guard = 0; guard < SCAN_LIMIT; guard++) {
    const iso = isoOf(cur)
    if (workweek.includes(cur.getUTCDay()) && !holidays.includes(iso)) {
      found++
      if (found === idx) return iso
    }
    cur = new Date(cur.getTime() + 86400000)
  }
  return isoOf(cur)
}

/** 六时参日期单元格：MM-DD (+N)，括号内为工作日序号偏移（如 09-03 (+8)） */
function fmtWd(startISO: string, idx: number, cal?: { workweek: number[]; holidays: string[] }): string {
  return `${wdToISO(startISO, idx, cal).slice(5)} (+${idx})`
}

/** 搭接类型+时距：FS+2 / SS-1（负 lag 自带 - 号；lag=0 只显类型） */
function fmtLink(l: SchedLink): string {
  if (l.lag > 0) return `${l.type}+${l.lag}`
  if (l.lag < 0) return `${l.type}${l.lag}`
  return l.type
}

/** 依赖行稳定 key：对侧任务+搭接四元组（同名多搭接也不撞） */
function depRowKey(r: InspectorDepRow): string {
  return `${r.otherId}|${r.link.from}|${r.link.to}|${r.link.type}|${r.link.lag}`
}

/** 前驱/后继共用列：任务名可点跳转 + 类型时距 + 自由时差 + 驱动标记（红 Tag） */
function depColumns(onNavigate: (taskId: string) => void): TableProps<InspectorDepRow>['columns'] {
  return [
    {
      title: '任务名',
      dataIndex: 'otherName',
      render: (_v, r) => (
        <Button type="link" size="small" style={{ padding: 0, height: 22 }} onClick={() => onNavigate(r.otherId)}>
          {r.otherName}
        </Button>
      ),
    },
    { title: '类型时距', dataIndex: 'link', width: 84, render: (_v, r) => fmtLink(r.link) },
    { title: '自由时差', dataIndex: 'freeSlack', width: 76, align: 'right', render: (v) => String(v) },
    {
      title: '驱动',
      dataIndex: 'driving',
      width: 68,
      render: (_v, r) => (r.driving ? <Tag color="red">驱动</Tag> : <span className="sched-dim">—</span>),
    },
  ]
}

/**
 * 任务检查器抽屉：右侧 400px，标题=任务名。
 * 概要区 → 驱动结论行（最显眼）→ 六时参 2×3 网格 → 前驱/后继任务表。
 */
export const TaskInspector: React.FC<TaskInspectorProps> = ({
  open,
  task,
  row,
  wbs,
  predecessors,
  successors,
  startDate,
  calendar,
  onClose,
  onNavigate,
}) => {
  const driving = useMemo(() => predecessors.filter((p) => p.driving), [predecessors])

  const sectionHead = (text: string) => (
    <div style={{ margin: '14px 0 6px', fontWeight: 600, fontSize: 13 }}>{text}</div>
  )

  // 概要字段：WBS（有则显）/工期（里程碑换「里程碑」标签）/模式（手动附锁定开始）/进度/关键
  const summaryCells: Array<[string, React.ReactNode]> = []
  if (wbs) summaryCells.push(['WBS', wbs])
  // 工期（刀3）：cd 任务显「日历」口径与等效工作日跨度（ef−es，透明不可编辑）
  summaryCells.push(['工期', task?.isMilestone
    ? <Tag color="purple">里程碑</Tag>
    : task?.durationUnit === 'cd'
      ? `${task.duration ?? 0} 天（日历，等效 ${row ? row.ef - row.es : '—'} 工作日）`
      : `${task?.duration ?? 0} 天`])
  summaryCells.push([
    '模式',
    task?.mode === 'manual' ? `手动（锁定开始=第 ${(task.manualStart ?? 0) + 1} 工作日）` : '自动',
  ])
  summaryCells.push(['进度', `${task?.progress ?? 0}%`])
  summaryCells.push(['关键', row?.critical ? <Tag color="red">关键</Tag> : <Tag>非关键</Tag>])

  // 驱动结论（优先级：手动锁定 > 单条驱动 > 多条驱动 > 无前驱 > 前驱均不驱动）
  let verdictIcon: React.ReactNode = <FlagOutlined />
  let verdictText: React.ReactNode = '无前置，开工日开始'
  let verdictTone: 'danger' | 'warning' | undefined
  if (task?.mode === 'manual') {
    verdictIcon = <LockOutlined />
    verdictText = '手动锁定，不受依赖影响'
    verdictTone = 'warning'
  } else if (driving.length === 1) {
    const d = driving[0]
    verdictIcon = <ThunderboltOutlined />
    verdictTone = 'danger'
    verdictText = (
      <span>
        由前置【
        <Button type="link" size="small" style={{ padding: 0, height: 22 }} onClick={() => onNavigate(d.otherId)}>
          {d.otherName}
        </Button>
        】({fmtLink(d.link)}) 驱动
      </span>
    )
  } else if (driving.length > 1) {
    verdictIcon = <ThunderboltOutlined />
    verdictTone = 'danger'
    verdictText = `与 ${driving.length} 条前置同时驱动`
  } else if (predecessors.length > 0) {
    verdictIcon = <InfoCircleOutlined />
    verdictText = '前置均未驱动本任务日期'
  }

  return (
    <Drawer placement="right" width={400} open={open} closable onClose={onClose} title={task?.name ?? '任务检查器'}>
      <div data-testid="sched-inspector">
        {!task ? (
          <span className="sched-dim">未选择任务：在图面选中一个任务后查看其日期由谁决定。</span>
        ) : (
          <>
            {/* 概要区 */}
            <div
              data-testid="sched-inspector-summary"
              style={{ display: 'grid', gridTemplateColumns: '48px 1fr 48px 1fr', gap: '6px 8px', alignItems: 'center' }}
            >
              {summaryCells.map(([label, value]) => (
                <React.Fragment key={label}>
                  <span className="sched-dim">{label}</span>
                  <span>{value}</span>
                </React.Fragment>
              ))}
            </div>

            {/* 结论行：本任务日期由谁决定（检查器的核心回答，最显眼） */}
            <div
              data-testid="sched-inspector-verdict"
              style={{
                display: 'flex',
                gap: 8,
                alignItems: 'flex-start',
                marginTop: 12,
                padding: '8px 10px',
                border: '1px solid rgba(5,5,5,0.14)',
                borderRadius: 8,
              }}
            >
              <Typography.Text strong type={verdictTone} style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                {verdictIcon}
                {verdictText}
              </Typography.Text>
            </div>

            {/* 六时参 2×3 小网格：ES/EF/LS/LF 换算日历日期，TF/FF 为工作日数 */}
            {sectionHead('六时参数')}
            {row ? (
              <div data-testid="sched-inspector-cpm" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 6 }}>
                {([
                  ['ES', fmtWd(startDate, row.es, calendar)],
                  ['EF', fmtWd(startDate, row.ef, calendar)],
                  ['LS', fmtWd(startDate, row.ls, calendar)],
                  ['LF', fmtWd(startDate, row.lf, calendar)],
                  ['TF', String(row.tf)],
                  ['FF', String(row.ff)],
                ] as Array<[string, string]>).map(([label, value]) => (
                  <div key={label} style={{ padding: '4px 8px', border: '1px solid rgba(5,5,5,0.1)', borderRadius: 6 }}>
                    <div className="sched-dim" style={{ fontSize: 12 }}>{label}</div>
                    <div data-testid={`sched-inspector-${label.toLowerCase()}`}>{value}</div>
                  </div>
                ))}
              </div>
            ) : (
              <span className="sched-dim">暂无排程数据（未参与计算或存在循环依赖）。</span>
            )}

            {/* 前驱任务表 */}
            {sectionHead('前置任务')}
            <div data-testid="sched-inspector-preds">
              {predecessors.length === 0 ? (
                <span className="sched-dim">无前置任务。</span>
              ) : (
                <Table<InspectorDepRow>
                  size="small"
                  pagination={false}
                  columns={depColumns(onNavigate)}
                  dataSource={predecessors}
                  rowKey={depRowKey}
                />
              )}
            </div>

            {/* 后继任务表 */}
            {sectionHead('后继任务')}
            <div data-testid="sched-inspector-succs">
              {successors.length === 0 ? (
                <span className="sched-dim">无后继任务。</span>
              ) : (
                <Table<InspectorDepRow>
                  size="small"
                  pagination={false}
                  columns={depColumns(onNavigate)}
                  dataSource={successors}
                  rowKey={depRowKey}
                />
              )}
            </div>
          </>
        )}
      </div>
    </Drawer>
  )
}
