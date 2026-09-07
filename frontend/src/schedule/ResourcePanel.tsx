/**
 * schedule/ResourcePanel.tsx — 资源与成本 UI（v4.124 资源成本刀3；v4.137 #13 资源级日历）
 *
 * 板块版面三件套中的两件（docs/gaea-schedule-resource-cost-design-2026-09.md §3.4）：
 *  - ResourcePanel：页头「资源」弹层（与日历/基线弹层同范式）——资源工作表 CRUD，
 *    提交即写 store（project.resources/assignments），走既有防抖自动保存链路，无独立保存按钮；
 *    work 资源行另有「日历」入口 → ResourceCalendarEditor（个人周工作制+休假，
 *    仅影响使用视图可用性/超载判定，不改 CPM 排程；material/cost 无按天可用性语义不显示）；
 *  - TaskResourceEditor：任务行「资源」Popover（复用前置 Popover 范式）——固定成本输入
 *    + 资源多选挂载（勾选建分配/取消删分配）+ 按资源类型的 units/quantity/amount 输入，
 *    每次变更整体替换该任务分配集（set_links「整体替换入边」同语义）。
 * 成本数值一律来自 computeCosts（引擎裁决，UI 不重复实现公式）；
 * 成本列/状态栏共用的显隐与格式化口径在 costUi.ts（hasCostData/fmtCost）。
 */
import React, { useMemo, useState } from 'react'
import { Button, Checkbox, Input, InputNumber, Popconfirm, Popover, Select, Space, Tag } from 'antd'
import { CalendarOutlined, DeleteOutlined } from '@ant-design/icons'
import { computeCosts } from './cost'
import { fmtCost } from './costUi'
import type { CpmResult, ResourceType, SchedAssignment, SchedCalendar, SchedResource, SchedTask } from './types'
import { useScheduleStore } from './store'

const TYPE_LABEL: Record<ResourceType, string> = { work: '工时', material: '材料', cost: '成本' }

/** 周工作制选项（JS getDay 口径 0=周日..6=周六），展示顺序 周日~周六 */
const WEEKDAYS: { day: number; label: string }[] = [
  { day: 0, label: '日' }, { day: 1, label: '一' }, { day: 2, label: '二' }, { day: 3, label: '三' },
  { day: 4, label: '四' }, { day: 5, label: '五' }, { day: 6, label: '六' },
]

/** 项目日历缺省口径（与 store/types「缺省=周一~周五」一致） */
const DEFAULT_WORKWEEK = [1, 2, 3, 4, 5]

/** 资源费率的单位标注口径：work=元/工日、material=元/单位（含计量单位）、cost 无费率 */
function rateSuffix(r: SchedResource): string {
  return r.type === 'work' ? '元/工日' : `元/${r.unit || '单位'}`
}

/**
 * 资源个人日历编辑器（v4.137 #13）：work 资源行「日历」Popover 内容。
 * 周工作制 Checkbox 组（周日~周六，至少一项——全取消该次点击无效）+ 节假日 Tag 列表
 * （YYYY-MM-DD 校验+去重）+「跟随项目日历」清除。与费率字段同范式：直接 upsertResource
 * 落库，无草稿态。诚实口径：个人日历只影响使用视图的可用性与超载判定，不改任务排程。
 */
const ResourceCalendarEditor: React.FC<{ resourceId: string }> = ({ resourceId }) => {
  const project = useScheduleStore((s) => s.project)
  const upsertResource = useScheduleStore((s) => s.upsertResource)
  const resource = (project.resources ?? []).find((r) => r.id === resourceId)
  const [holidayDraft, setHolidayDraft] = useState('')
  if (!resource) return null
  /** 未自定义时编辑视图跟随项目日历（项目亦缺省则周一~周五），首个改动即固化为个人日历 */
  const base = resource.calendar ?? project.calendar ?? { workweek: DEFAULT_WORKWEEK, holidays: [] }

  const commit = (calendar: SchedCalendar | undefined) => {
    // 整体替换该资源（与费率字段同一动作；calendar 已在 upsertResource 清洗白名单内）
    upsertResource({ ...resource, calendar })
  }
  const toggleDay = (day: number, on: boolean) => {
    const next = on ? [...new Set([...base.workweek, day])] : base.workweek.filter((d) => d !== day)
    if (next.length === 0) return // 至少保留一个工作日：全取消时该次点击无效
    commit({ workweek: next.sort((a, b) => a - b), holidays: base.holidays })
  }
  const addHoliday = () => {
    const d = holidayDraft.trim()
    if (!/^\d{4}-\d{2}-\d{2}$/.test(d) || base.holidays.includes(d)) return // 格式校验+去重
    commit({ workweek: base.workweek, holidays: [...base.holidays, d].sort() })
    setHolidayDraft('')
  }

  return (
    <div style={{ display: 'grid', gap: 8, minWidth: 280 }} data-testid={`sched-res-cal-editor-${resource.id}`}>
      <span className="sched-dim">个人休假/停工只影响使用视图的可用性与超载判定，不改任务排程（排程按项目日历）。</span>
      {!resource.calendar && <span className="sched-dim" data-testid={`sched-res-cal-follow-${resource.id}`}>未自定义（跟随项目日历）</span>}
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>周工作制（至少一项）</div>
        <Space size={4} wrap>
          {WEEKDAYS.map((w) => (
            <Checkbox key={w.day} checked={base.workweek.includes(w.day)} onChange={(e) => toggleDay(w.day, e.target.checked)}>
              周{w.label}
            </Checkbox>
          ))}
        </Space>
      </div>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>节假日 / 个人休假（YYYY-MM-DD）</div>
        <Space size={4} wrap style={{ marginBottom: 6 }}>
          {base.holidays.map((h) => (
            <Tag key={h} closable onClose={() => commit({ workweek: base.workweek, holidays: base.holidays.filter((x) => x !== h) })}>
              {h}
            </Tag>
          ))}
          {base.holidays.length === 0 && <span className="sched-dim">无</span>}
        </Space>
        <Space size={4}>
          <Input
            size="small"
            placeholder="2026-10-01"
            value={holidayDraft}
            style={{ width: 130 }}
            onChange={(e) => setHolidayDraft(e.target.value)}
            onPressEnter={addHoliday}
          />
          <Button size="small" type="dashed" onClick={addHoliday}>添加</Button>
        </Space>
      </div>
      <Button size="small" disabled={!resource.calendar} onClick={() => commit(undefined)}>
        跟随项目日历（清除个人设置）
      </Button>
    </div>
  )
}

/**
 * 资源工作表弹层：名称/类型/费率（带单位标注）/每次使用/单位·上限/已分配任务数/资源成本合计。
 * 类型切换联动字段显隐（cost 隐藏费率与每次使用）；删除级联清除其分配（Popconfirm）；
 * work 资源名称旁有「日历」入口（primary=已自定义 / default=未自定义跟随项目日历）。
 */
export const ResourcePanel: React.FC<{ cpm: CpmResult }> = ({ cpm }) => {
  const project = useScheduleStore((s) => s.project)
  const upsertResource = useScheduleStore((s) => s.upsertResource)
  const removeResource = useScheduleStore((s) => s.removeResource)
  const resources = project.resources ?? []
  const costs = useMemo(() => computeCosts(project, cpm), [project, cpm])
  /** 每资源的已分配任务数（去重 taskId） */
  const taskCount = useMemo(() => {
    const m = new Map<string, Set<string>>()
    for (const a of project.assignments ?? []) {
      if (!m.has(a.resourceId)) m.set(a.resourceId, new Set())
      m.get(a.resourceId)!.add(a.taskId)
    }
    return m
  }, [project.assignments])

  return (
    <div className="sched-res-panel" data-testid="sched-resource-panel">
      <div className="sched-res-row sched-res-head">
        <span>名称</span>
        <span>类型</span>
        <span>费率</span>
        <span>每次使用</span>
        <span>单位/上限</span>
        <span className="sched-res-num">任务</span>
        <span className="sched-res-num">成本合计</span>
        <span />
      </div>
      {resources.length === 0 && (
        <span className="sched-dim">暂无资源：先新增（默认工时类），再到任务行「资源」入口挂载分配。</span>
      )}
      {resources.map((r) => {
        const isCost = r.type === 'cost'
        return (
          <div className="sched-res-row" key={r.id} data-testid={`sched-res-row-${r.id}`}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 4, minWidth: 0 }}>
              <Input
                size="small"
                variant="borderless"
                value={r.name}
                style={{ flex: 1, minWidth: 0, padding: 0 }}
                onChange={(e) => upsertResource({ ...r, name: e.target.value })}
              />
              {r.type === 'work' && (
                <Popover trigger="click" placement="bottom" content={<ResourceCalendarEditor resourceId={r.id} />} title={`「${r.name}」个人日历`}>
                  <Button
                    size="small"
                    type={r.calendar ? 'primary' : 'default'}
                    icon={<CalendarOutlined />}
                    aria-label={`资源日历${r.name}`}
                    title={`资源个人日历（${r.calendar ? '已自定义' : '未自定义，跟随项目日历'}）`}
                    data-testid={`sched-res-cal-${r.id}`}
                    style={{ flexShrink: 0 }}
                  />
                </Popover>
              )}
            </span>
            <Select
              size="small"
              value={r.type}
              style={{ width: '100%' }}
              options={(Object.keys(TYPE_LABEL) as ResourceType[]).map((t) => ({ value: t, label: TYPE_LABEL[t] }))}
              onChange={(v) => upsertResource({ ...r, type: v as ResourceType })}
            />
            {isCost ? (
              <span className="sched-dim sched-res-na" title="成本资源无费率：金额直接记在任务的分配上">—</span>
            ) : (
              <InputNumber
                size="small"
                min={0}
                value={r.standardRate}
                suffix={rateSuffix(r)}
                style={{ width: '100%' }}
                title={`标准费率（${rateSuffix(r)}）`}
                onChange={(v) => upsertResource({ ...r, standardRate: v ?? undefined })}
              />
            )}
            {isCost ? (
              <span className="sched-dim sched-res-na">—</span>
            ) : (
              <InputNumber
                size="small"
                min={0}
                value={r.costPerUse}
                suffix="元"
                style={{ width: '100%' }}
                title="每次使用成本（每条分配计一次）"
                onChange={(v) => upsertResource({ ...r, costPerUse: v ?? undefined })}
              />
            )}
            {r.type === 'material' ? (
              <Input
                size="small"
                variant="borderless"
                value={r.unit ?? ''}
                placeholder="t/m³"
                title="材料计量单位（展示与导出用）"
                style={{ padding: 0 }}
                onChange={(e) => upsertResource({ ...r, unit: e.target.value })}
              />
            ) : r.type === 'work' ? (
              <InputNumber
                size="small"
                min={0}
                step={0.5}
                value={r.maxUnits}
                placeholder="1"
                title="可用上限（100%=1 个满负荷当量，仅承载）"
                style={{ width: '100%' }}
                onChange={(v) => upsertResource({ ...r, maxUnits: v ?? undefined })}
              />
            ) : (
              <span className="sched-dim sched-res-na">—</span>
            )}
            <span className="sched-res-num" data-testid={`sched-res-count-${r.id}`}>{taskCount.get(r.id)?.size ?? 0}</span>
            <span className="sched-res-num sched-res-cost" data-testid={`sched-res-cost-${r.id}`}>¥{fmtCost(costs.byResource[r.id] ?? 0)}</span>
            <Popconfirm
              title="删除该资源？"
              description={`「${r.name}」的 ${taskCount.get(r.id)?.size ?? 0} 条任务分配将一并删除`}
              onConfirm={() => removeResource(r.id)}
            >
              <Button size="small" type="text" danger icon={<DeleteOutlined />} aria-label={`删除资源${r.name}`} />
            </Popconfirm>
          </div>
        )
      })}
      <Button
        size="small"
        type="dashed"
        data-testid="sched-res-add"
        onClick={() => upsertResource({ id: '', name: `新资源${resources.length + 1}`, type: 'work' })}
      >
        新增资源
      </Button>
      <span className="sched-dim">
        口径：工时=工期×投入×费率+每次使用；材料=总量×单价；成本=金额。费率单位：工时 元/工日、材料 元/单位（工作日口径）。
      </span>
    </div>
  )
}

/**
 * 任务行「资源」Popover：该任务固定成本 + 资源多选挂载 + 每条分配的
 * units（工时）/quantity（材料）/amount（成本）输入。分组行不渲染入口
 * （fail-closed：汇总唯一口径=子孙求和，模型禁止分组行分配）。
 */
export const TaskResourceEditor: React.FC<{ task: SchedTask }> = ({ task }) => {
  const project = useScheduleStore((s) => s.project)
  const setTaskAssignments = useScheduleStore((s) => s.setTaskAssignments)
  const updateTask = useScheduleStore((s) => s.updateTask)
  const resources = project.resources ?? []
  const mine = useMemo(
    () => (project.assignments ?? []).filter((a) => a.taskId === task.id),
    [project.assignments, task.id],
  )

  // 整体替换该任务分配集（每次变更都提交全量，与前置 Popover 逐击即存同范式）
  const commit = (next: SchedAssignment[]) => setTaskAssignments(task.id, next)
  const toggle = (resourceId: string, on: boolean) => {
    if (on === mine.some((a) => a.resourceId === resourceId)) return
    commit(on ? [...mine, { taskId: task.id, resourceId }] : mine.filter((a) => a.resourceId !== resourceId))
  }
  const patchField = (resourceId: string, patch: Partial<SchedAssignment>) => {
    commit(mine.map((a) => (a.resourceId === resourceId ? { ...a, ...patch } : a)))
  }

  return (
    <div style={{ display: 'grid', gap: 8, minWidth: 300 }} data-testid={`sched-task-res-panel-${task.id}`}>
      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        <span className="sched-dim" style={{ flexShrink: 0 }}>固定成本</span>
        <InputNumber
          size="small"
          min={0}
          value={task.fixedCost}
          suffix="元"
          style={{ width: 140 }}
          title="任务固定成本（与分配无关的一次性费用）"
          onChange={(v) => updateTask(task.id, { fixedCost: v ?? undefined })}
        />
      </div>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>资源挂载（勾选即建分配，取消即删）</div>
        {resources.length === 0 && (
          <span className="sched-dim">暂无资源：请先在页头「资源」中新增。</span>
        )}
        <div style={{ display: 'grid', gap: 4 }}>
          {resources.map((r) => {
            const a = mine.find((x) => x.resourceId === r.id)
            return (
              <div key={r.id} style={{ display: 'flex', gap: 6, alignItems: 'center' }} data-testid={`sched-link-row-${task.id}-${r.id}`}>
                <Checkbox
                  checked={!!a}
                  onChange={(e) => toggle(r.id, e.target.checked)}
                >
                  {r.name}
                  <span className="sched-dim">（{TYPE_LABEL[r.type]}）</span>
                </Checkbox>
                {a && r.type === 'work' && (
                  <InputNumber
                    size="small"
                    min={0}
                    step={0.5}
                    value={a.units}
                    title="投入强度（成本=工期×投入×费率）"
                    style={{ marginLeft: 'auto', width: 96 }}
                    suffix="投入"
                    onChange={(v) => patchField(r.id, { units: v ?? undefined })}
                  />
                )}
                {a && r.type === 'material' && (
                  <InputNumber
                    size="small"
                    min={0}
                    value={a.quantity}
                    title="固定总量（成本=总量×单价，不随时长变）"
                    style={{ marginLeft: 'auto', width: 110 }}
                    suffix={r.unit || '单位'}
                    onChange={(v) => patchField(r.id, { quantity: v ?? undefined })}
                  />
                )}
                {a && r.type === 'cost' && (
                  <InputNumber
                    size="small"
                    min={0}
                    value={a.amount}
                    title="该分配的固定金额（元）"
                    style={{ marginLeft: 'auto', width: 110 }}
                    suffix="元"
                    onChange={(v) => patchField(r.id, { amount: v ?? undefined })}
                  />
                )}
              </div>
            )
          })}
        </div>
      </div>
      <span className="sched-dim">工时成本随工期自动重算（改工期即改成本）；材料与成本金额固定。</span>
    </div>
  )
}
