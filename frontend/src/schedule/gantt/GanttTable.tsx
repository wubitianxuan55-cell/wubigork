/**
 * schedule/gantt/GanttTable.tsx — 表格窗格内容（自 GanttView.tsx 拆分，行为零变化）
 *
 * 列头行（宽=可见列合计，收纳时多余列裁剪不压缩）+ 表体逐行渲染：
 * 合成组头行（颜色标记/折叠态/汇总跨度）与任务行（名称内联编辑/工期+单位 chip/
 * 进度/手动开始/六时参/模式/前置右键编辑/后续/成本/自定义字段单元格）。
 * 列宽 W、行高 ROW_H、格式口径 fmtDate 等来自 ganttUtil，与画布行同源。
 */
import React from 'react'
import { Button, Dropdown, Input, InputNumber, Popover, type MenuProps } from 'antd'
import { TeamOutlined } from '@ant-design/icons'
import type { CpmResult, SchedProject, SchedTask } from '../types'
import type { CostResult } from '../cost'
import { isCd } from '../cpm'
import { fmtLinkRefs } from '../ganttLinks'
import { fmtCost } from '../costUi'
import { GANTT_CUSTOM_KEYS, type GanttCol } from '../ganttCols'
import type { GanttRow } from '../ganttGroup'
import { ROW_H, W, CUSTOM_FIELD_BY_KEY, fmtDate, groupCost, noOf, predsOf, succsOf, taskNameOf } from './ganttUtil'
import { PredEditor } from './PredEditor'
import { CustomFieldCell } from './CustomFieldCell'
import { TaskResourceEditor } from '../ResourcePanel'

interface GanttTableProps {
  leftW: number
  cols: GanttCol[]
  tbodyRef: React.Ref<HTMLDivElement>
  rows: GanttRow[]
  wbs: string[]
  project: SchedProject
  cpm: CpmResult
  selectedId: string | null
  costShown: boolean
  costs: CostResult
  groupColorSeq: number[]
  collapsedKeys: Set<string>
  colOn: (key: string) => boolean
  rowDim: (id: string) => boolean
  onSelect: (id: string) => void
  onUpdateTask: (id: string, patch: Partial<SchedTask>) => void
  onToggleCollapse: (key: string) => void
  rowMenu: (t: SchedTask, group: boolean) => MenuProps['items']
  onRowMenuClick: (key: string, t: SchedTask) => void
}

export const GanttTable: React.FC<GanttTableProps> = ({
  leftW,
  cols,
  tbodyRef,
  rows,
  wbs,
  project,
  cpm,
  selectedId,
  costShown,
  costs,
  groupColorSeq,
  collapsedKeys,
  colOn,
  rowDim,
  onSelect,
  onUpdateTask,
  onToggleCollapse,
  rowMenu,
  onRowMenuClick,
}) => {
  const cal = project.calendar
  return (
    <>
      {/* 列头行宽=全部可见列合计（leftW）：表格收纳时多余列被裁剪隐藏，
          绝不被 flex 压缩——压缩会造成列头与数据行永久错位（v4.142 修正） */}
      <div className="sched-gantt-row sched-gantt-head" style={{ height: 40, width: leftW }}>
        {cols.map((c) => (
          <div key={c.key} style={{ width: c.w }} className="sched-gantt-cell sched-th">{c.label}</div>
        ))}
      </div>
      <div className="sched-gantt-tbody" ref={tbodyRef}>
        <div style={{ width: leftW }}>
          {(() => {
            let synSeq = -1
            return rows.map((r, rowIdx) => {
            if (r.kind === 'group' && !r.key.startsWith('wbs:')) {
              // 合成组头行（分组模式）：名称+计数+汇总跨度，点击折叠/展开
              synSeq++
              const open = !collapsedKeys.has(r.key)
              return (
              <div
                key={r.key}
                data-testid={`sched-group-row-${rowIdx}`}
                className={`sched-gantt-row sched-group-row sched-group-c${synSeq % 6}`}
                style={{ height: ROW_H }}
                onClick={() => onToggleCollapse(r.key)}
                title="点击折叠/展开该组"
              >
                <div style={{ width: W.no }} className="sched-gantt-cell sched-row-no" />
                <div style={{ width: W.name }} className="sched-gantt-cell">
                  <span style={{ paddingLeft: r.level * 14, display: 'inline-flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
                    <strong className="sched-group-name">{open ? '▲' : '▶'}{r.label}</strong>
                    <span className="sched-dim" style={{ fontSize: 12 }}>{r.count} 项</span>
                  </span>
                </div>
                {colOn('wbs') && <div style={{ width: W.wbs }} className="sched-gantt-cell sched-dim" />}
                {colOn('dur') && <div style={{ width: W.dur }} className="sched-gantt-cell"><span className="sched-dim">汇总</span></div>}
                {colOn('progress') && <div style={{ width: W.progress }} className="sched-gantt-cell" />}
                {colOn('start') && <div style={{ width: W.start }} className="sched-gantt-cell sched-dim">{cpm.ok ? fmtDate(project.startDate, r.minEs, cal) : ''}</div>}
                {colOn('finish') && <div style={{ width: W.finish }} className="sched-gantt-cell sched-dim">{cpm.ok ? fmtDate(project.startDate, r.maxEf, cal) : ''}</div>}
                {colOn('ls') && <div style={{ width: W.ls }} className="sched-gantt-cell" />}
                {colOn('lf') && <div style={{ width: W.lf }} className="sched-gantt-cell" />}
                {colOn('tf') && <div style={{ width: W.tf }} className="sched-gantt-cell" />}
                {colOn('ff') && <div style={{ width: W.ff }} className="sched-gantt-cell" />}
                {colOn('mode') && <div style={{ width: W.mode }} className="sched-gantt-cell" />}
                {colOn('preds') && <div style={{ width: W.preds }} className="sched-gantt-cell" />}
                {colOn('succ') && <div style={{ width: W.succ }} className="sched-gantt-cell" />}
                {colOn('cost') && <div style={{ width: W.cost }} className="sched-gantt-cell sched-cost-cell" />}
                {/* 自定义字段列（v4.138 #14）：合成组头行留空（汇总只对叶任务有意义） */}
                {GANTT_CUSTOM_KEYS.filter((ck) => colOn(ck)).map((ck) => (
                  <div key={ck} style={{ width: W[ck] }} className="sched-gantt-cell" />
                ))}
              </div>
              )
            }
            const i = r.kind === 'task' ? r.idx : Number(r.key.slice(4))
            const t = project.tasks[i]
            const row = cpm.rows[t.id]
            const group = r.kind === 'group'
            const span = group ? { es: r.minEs, ef: r.maxEf } : null
            const manual = t.mode === 'manual'
            const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
            const colorCls = group ? ` sched-group-c${groupColorSeq[i]}` : ''
            const dimCls = !group && rowDim(t.id) ? ' sched-row-dim' : ''
            const groupKey = `wbs:${i}`
            const groupOpen = !collapsedKeys.has(groupKey)
            return (
              <Dropdown key={t.id} trigger={['contextMenu']} menu={{ items: rowMenu(t, group), onClick: (e) => onRowMenuClick(e.key, t) }}>
              <div
                className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${dimCls}${selectedId === t.id ? ' sched-row-selected' : ''}`}
                style={{ height: ROW_H }}
                onClick={() => onSelect(t.id)}
              >
                <div style={{ width: W.no }} className="sched-gantt-cell sched-row-no">{i + 1}</div>
                <div style={{ width: W.name }} className="sched-gantt-cell">
                  <span style={{ paddingLeft: t.level * 14, display: 'inline-flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
                    {group ? (
                      <strong
                        className="sched-group-name"
                        style={{ cursor: 'pointer' }}
                        onClick={(e) => { e.stopPropagation(); onToggleCollapse(groupKey) }}
                        title="点击折叠/展开"
                      >
                        {groupOpen ? '▲' : '▶'}{t.name}
                      </strong>
                    ) : (
                      <Input
                        size="small"
                        variant="borderless"
                        value={t.name}
                        onChange={(e) => onUpdateTask(t.id, { name: e.target.value })}
                        style={{ padding: 0 }}
                      />
                    )}
                    {t.isMilestone && <span className="sched-milestone-tag">里程碑</span>}
                    {!group && (
                      <Popover trigger="click" placement="left" content={<TaskResourceEditor task={t} />} title={`「${t.name}」资源与成本`}>
                        <Button
                          size="small"
                          type="text"
                          className="sched-res-entry"
                          data-testid={`sched-task-res-${t.id}`}
                          icon={<TeamOutlined />}
                          title="资源与成本（分配挂载、固定成本）"
                        >
                          {(project.assignments?.filter((a) => a.taskId === t.id).length ?? 0) || ''}
                        </Button>
                      </Popover>
                    )}
                  </span>
                </div>
                {colOn('wbs') && <div style={{ width: W.wbs }} className="sched-gantt-cell sched-dim">{wbs[i]}</div>}
                {colOn('dur') && (
                  <div style={{ width: W.dur }} className="sched-gantt-cell">
                    {group ? <span className="sched-dim">汇总</span> : (
                      <span style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                        <InputNumber
                          size="small"
                          variant="borderless"
                          min={0}
                          value={dur}
                          onChange={(v) => onUpdateTask(t.id, { duration: Number(v) || 0 })}
                          style={{ padding: 0, width: '100%', flex: 1, minWidth: 0 }}
                          title={isCd(t) ? '日历天（自然日定时）；等效工作日跨度见条形悬停' : undefined}
                        />
                        {/* 单位 chip（刀3）：cd 行常显（点击切回工作日）；wd 行 hover 行时浮现。
                            切单位不换算数值（拍板项 7）；分组行/里程碑无入口。 */}
                        {!t.isMilestone && (
                          <Button
                            size="small"
                            type={isCd(t) ? 'primary' : 'text'}
                            ghost={isCd(t)}
                            className={`sched-unit-chip${isCd(t) ? ' sched-unit-cd' : ''}`}
                            data-testid={`sched-unit-${t.id}`}
                            onClick={() => onUpdateTask(t.id, isCd(t) ? { durationUnit: 'wd' } : { durationUnit: 'cd' })}
                            title={isCd(t) ? '日历天：按自然日定时（点击切回工作日；数值不变）' : '切为日历天：按自然日定时（数值不变）'}
                          >
                            日历
                          </Button>
                        )}
                      </span>
                    )}
                  </div>
                )}
                {colOn('progress') && (
                  <div style={{ width: W.progress }} className="sched-gantt-cell">
                    {!group && (
                      <InputNumber
                        size="small"
                        variant="borderless"
                        min={0}
                        max={100}
                        value={t.progress}
                        onChange={(v) => onUpdateTask(t.id, { progress: Math.max(0, Math.min(100, Math.round(Number(v) || 0))) })}
                        style={{ padding: 0, width: '100%' }}
                        data-testid={`sched-progress-${t.id}`}
                        title="完成进度 0-100（条形进度覆盖与前锋线按此绘制）"
                      />
                    )}
                  </div>
                )}
                {colOn('start') && (
                  <div style={{ width: W.start }} className="sched-gantt-cell sched-dim">
                    {manual && !group ? (
                      <InputNumber
                        size="small"
                        variant="borderless"
                        min={0}
                        value={row?.es ?? 0}
                        onChange={(v) => onUpdateTask(t.id, { manualStart: Number(v) || 0 })}
                        style={{ padding: 0, width: '100%' }}
                        title="手动模式：锁定开始（工作日序号）"
                      />
                    ) : (
                      (group ? span : row) ? fmtDate(project.startDate, group ? span!.es : row!.es, cal) : ''
                    )}
                  </div>
                )}
                {colOn('finish') && (
                  <div style={{ width: W.finish }} className="sched-gantt-cell sched-dim">
                    {(group ? span : row) ? fmtDate(project.startDate, group ? span!.ef : row!.ef, cal) : ''}
                  </div>
                )}
                {/* 六时参（刀B）：LS/LF/TF/FF——手动任务逆推不回传，诚实显示 —（不放假数据） */}
                {colOn('ls') && (
                  <div style={{ width: W.ls }} className="sched-gantt-cell sched-dim" title="最迟开始（逆推）">
                    {group || manual ? (manual && !group ? '—' : '') : fmtDate(project.startDate, row!.ls, cal)}
                  </div>
                )}
                {colOn('lf') && (
                  <div style={{ width: W.lf }} className="sched-gantt-cell sched-dim" title="最迟完成（逆推）">
                    {group || manual ? (manual && !group ? '—' : '') : fmtDate(project.startDate, row!.lf, cal)}
                  </div>
                )}
                {colOn('tf') && (
                  <div style={{ width: W.tf }} className={`sched-gantt-cell${!group && !manual && row?.tf === 0 ? ' sched-critical-text' : ' sched-dim'}`} title={manual ? '手动模式不参与时差计算' : '总时差 = LS − ES（0=关键）'}>
                    {group ? '' : manual ? '—' : row?.tf}
                  </div>
                )}
                {colOn('ff') && (
                  <div style={{ width: W.ff }} className="sched-gantt-cell sched-dim" title={manual ? '手动模式不参与时差计算' : '自由时差 = 紧后 ES 最小值 − EF'}>
                    {group ? '' : manual ? '—' : row?.ff}
                  </div>
                )}
                {colOn('mode') && (
                  <div style={{ width: W.mode }} className="sched-gantt-cell">
                    {!group && (
                      <Button
                        size="small"
                        type={manual ? 'primary' : 'text'}
                        ghost={manual}
                        className={`sched-mode-chip${manual ? ' sched-mode-manual' : ''}`}
                        onClick={() => onUpdateTask(t.id, manual ? { mode: 'auto' } : { mode: 'manual', manualStart: row?.es ?? 0 })}
                        title={manual ? '手动模式（锁定开始，点击切回自动）' : '自动模式（CPM 排程，点击切手动）'}
                      >
                        {manual ? '手动' : '自动'}
                      </Button>
                    )}
                  </div>
                )}
                {colOn('preds') && (
                  <div style={{ width: W.preds }} className="sched-gantt-cell">
                    {!group && (
                      <Popover trigger="click" placement="left" content={<PredEditor task={t} />} title={`「${t.name}」前置任务`}>
                        <Button
                          size="small"
                          type="text"
                          className="sched-link-cell"
                          data-testid={`sched-preds-${t.id}`}
                          title={predsOf(t.id, project).map((l) => taskNameOf(project, l.from)).join('、') || '无前置，点击添加'}
                        >
                          {fmtLinkRefs(predsOf(t.id, project).map((l) => ({ id: l.from, type: l.type, lag: l.lag })), noOf(project))}
                        </Button>
                      </Popover>
                    )}
                  </div>
                )}
                {colOn('succ') && (
                  <div style={{ width: W.succ }} className="sched-gantt-cell">
                    {!group && (
                      <span
                        className="sched-dim sched-link-cell-text"
                        data-testid={`sched-succ-${t.id}`}
                        title={succsOf(t.id, project).map((l) => taskNameOf(project, l.to)).join('、') || '无后续任务'}
                      >
                        {fmtLinkRefs(succsOf(t.id, project).map((l) => ({ id: l.to, type: l.type, lag: l.lag })), noOf(project))}
                      </span>
                    )}
                  </div>
                )}
                {colOn('cost') && (
                  <div style={{ width: W.cost }} className="sched-gantt-cell sched-cost-cell" data-testid={`sched-cost-${t.id}`}>
                    {costShown && (group ? (
                      <span className="sched-dim" title="分组汇总：子孙叶任务成本求和">{fmtCost(groupCost(project, costs, i))}</span>
                    ) : (
                      <span title="明细合计：固定成本 + 分配成本（随工期实时重算）">{fmtCost(costs.rows[t.id]?.total ?? 0)}</span>
                    ))}
                  </div>
                )}
                {/* 自定义字段列（v4.138 #14）：叶行可编辑（text=Input/num=InputNumber，清空=删槽键）、分组行留空 */}
                {GANTT_CUSTOM_KEYS.filter((ck) => colOn(ck)).map((ck) => (
                  <div key={ck} style={{ width: W[ck] }} className="sched-gantt-cell">
                    {!group && <CustomFieldCell task={t} field={CUSTOM_FIELD_BY_KEY[ck]} />}
                  </div>
                ))}
              </div>
              </Dropdown>
            )
            }
          )
          })()}
        </div>
      </div>
    </>
  )
}