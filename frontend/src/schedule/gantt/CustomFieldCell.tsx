/**
 * schedule/gantt/CustomFieldCell.tsx — 自定义字段单元格（自 GanttView.tsx 拆分，行为零变化）
 *
 * 仅叶任务可编辑（分组行由调用方留空）。text 槽=Input（borderless，同名称列范式）；
 * num 槽=InputNumber（无 min、可小数）。写回整体替换 patch.custom；清空文本（空串）/
 * 清空数值（null）=删除该槽键，custom 里不留空值。
 */
import React from 'react'
import { Input, InputNumber } from 'antd'
import type { SchedTask } from '../types'
import { customValueOf, type CustomFieldDef } from '../customFields'
import { useScheduleStore } from '../store'

export const CustomFieldCell: React.FC<{ task: SchedTask; field: CustomFieldDef }> = ({ task, field }) => {
  const updateTask = useScheduleStore((s) => s.updateTask)
  /** 落库：有值写槽、空值删槽（patch.custom 整体替换口径） */
  const write = (v: string | number | null) => {
    const next: Partial<Record<string, string | number>> = { ...(task.custom ?? {}) }
    if (v === null || v === '') delete next[field.key]
    else next[field.key] = v
    updateTask(task.id, { custom: next })
  }
  if (field.type === 'number') {
    const v = customValueOf(task, field.key)
    return (
      <InputNumber
        size="small"
        variant="borderless"
        value={typeof v === 'number' ? v : undefined}
        onChange={(nv) => {
          if (nv === null) write(null)
          else {
            const n = Number(nv)
            write(Number.isFinite(n) ? n : null)
          }
        }}
        style={{ padding: 0, width: '100%' }}
        data-testid={`sched-custom-${field.key}-${task.id}`}
      />
    )
  }
  return (
    <Input
      size="small"
      variant="borderless"
      value={String(customValueOf(task, field.key) ?? '')}
      onChange={(e) => write(e.target.value)}
      style={{ padding: 0 }}
      data-testid={`sched-custom-${field.key}-${task.id}`}
    />
  )
}