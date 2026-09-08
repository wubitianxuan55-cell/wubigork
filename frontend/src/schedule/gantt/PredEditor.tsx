/**
 * schedule/gantt/PredEditor.tsx — 前置关系编辑 Popover（自 GanttView.tsx 拆分，行为零变化）
 * 整体替换该任务入边：Select 改前置任务/搭接类型、InputNumber 改时距、可删除、可追加。
 */
import React, { useMemo } from 'react'
import { Button, InputNumber, Select } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { LinkType, SchedTask } from '../types'
import { descendantIds, useScheduleStore, type PredDraft } from '../store'

export const PredEditor: React.FC<{ task: SchedTask }> = ({ task }) => {
  const project = useScheduleStore((s) => s.project)
  const setPreds = useScheduleStore((s) => s.setPreds)
  const drafts: PredDraft[] = useMemo(
    () => project.links.filter((l) => l.to === task.id).map((l) => ({ from: l.from, type: l.type, lag: l.lag })),
    [project.links, task.id],
  )
  const banned = descendantIds(project.tasks, task.id)
  const options = project.tasks.filter((t) => !banned.has(t.id) && t.level > 0)

  const commit = (next: PredDraft[]) => setPreds(task.id, next)

  return (
    <div style={{ display: 'grid', gap: 6, minWidth: 280 }}>
      {drafts.length === 0 && <span style={{ color: 'var(--md-sys-color-outline, #999)', fontSize: 12 }}>暂无前置任务</span>}
      {drafts.map((d, i) => (
        <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          <Select
            size="small"
            value={d.from}
            style={{ flex: 1 }}
            options={options.map((o) => ({ value: o.id, label: o.name }))}
            onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, from: v } : x)))}
          />
          <Select
            size="small"
            value={d.type}
            style={{ width: 72 }}
            options={[
              { value: 'FS', label: 'FS' }, { value: 'SS', label: 'SS' },
              { value: 'FF', label: 'FF' }, { value: 'SF', label: 'SF' },
            ]}
            onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, type: v as LinkType } : x)))}
          />
          <InputNumber size="small" value={d.lag} style={{ width: 64 }} addonAfter="日" onChange={(v) => commit(drafts.map((x, j) => (j === i ? { ...x, lag: Number(v) || 0 } : x)))} />
          <Button size="small" type="text" danger icon={<DeleteOutlined />} onClick={() => commit(drafts.filter((_, j) => j !== i))} />
        </div>
      ))}
      <Button
        size="small"
        type="dashed"
        disabled={options.length === 0}
        onClick={() => commit([...drafts, { from: options[0]?.id ?? '', type: 'FS', lag: 0 }])}
      >
        添加前置
      </Button>
    </div>
  )
}