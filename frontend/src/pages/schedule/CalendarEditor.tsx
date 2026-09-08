/**
 * CalendarEditor — 工作日历设置弹层（SchedulePage 信息条「日历」Popover 内容）
 * 瘦身 P3：自 pages/SchedulePage.tsx 分片，正文零改动。
 */
import React, { useState } from 'react'
import { Button, Checkbox, Input, Space, Tag } from 'antd'
import { useScheduleStore } from '../../schedule/store'

const WEEKDAYS: { day: number; label: string }[] = [
  { day: 1, label: '一' }, { day: 2, label: '二' }, { day: 3, label: '三' },
  { day: 4, label: '四' }, { day: 5, label: '五' }, { day: 6, label: '六' }, { day: 0, label: '日' },
]

/** 日历设置弹层：周工作制 + 节假日例外 */
export const CalendarEditor: React.FC = () => {
  const project = useScheduleStore((s) => s.project)
  const setCalendar = useScheduleStore((s) => s.setCalendar)
  const cal = project.calendar ?? { workweek: [1, 2, 3, 4, 5], holidays: [] }
  const [holidayDraft, setHolidayDraft] = useState('')

  const toggleDay = (day: number, on: boolean) => {
    const next = on ? [...new Set([...cal.workweek, day])] : cal.workweek.filter((d) => d !== day)
    if (next.length === 0) return // 至少保留一个工作日
    setCalendar({ workweek: next, holidays: cal.holidays })
  }
  const addHoliday = () => {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(holidayDraft) || cal.holidays.includes(holidayDraft)) return
    setCalendar({ workweek: cal.workweek, holidays: [...cal.holidays, holidayDraft].sort() })
    setHolidayDraft('')
  }

  return (
    <div style={{ display: 'grid', gap: 8, minWidth: 280 }}>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>周工作制（至少一项）</div>
        <Space size={4} wrap>
          {WEEKDAYS.map((w) => (
            <Checkbox
              key={w.day}
              checked={cal.workweek.includes(w.day)}
              onChange={(e) => toggleDay(w.day, e.target.checked)}
            >
              周{w.label}
            </Checkbox>
          ))}
        </Space>
      </div>
      <div>
        <div className="sched-dim" style={{ marginBottom: 4 }}>节假日 / 停工日（YYYY-MM-DD）</div>
        <Space size={4} wrap style={{ marginBottom: 6 }}>
          {cal.holidays.map((h) => (
            <Tag key={h} closable onClose={() => setCalendar({ workweek: cal.workweek, holidays: cal.holidays.filter((x) => x !== h) })}>
              {h}
            </Tag>
          ))}
          {cal.holidays.length === 0 && <span className="sched-dim">无</span>}
        </Space>
        <Space size={4}>
          <Input size="small" placeholder="2026-10-01" value={holidayDraft} style={{ width: 130 }}
            onChange={(e) => setHolidayDraft(e.target.value)} onPressEnter={addHoliday} />
          <Button size="small" type="dashed" onClick={addHoliday}>添加</Button>
        </Space>
      </div>
      <span className="sched-dim">工期/时距均按工作日计算；日期轴自动跳过非工作日。</span>
    </div>
  )
}