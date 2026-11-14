/**
 * ScheduleDock.tsx — ZPert 式“预警信息”右舷。
 *
 * 复刻页的右舷没有分类页签和算法开关，只逐条展示预警。算法仍由任务检查器
 * 控制；这里保持单一信息面，避免在主布局里增加额外控件。
 */
import React from 'react'
import { Button } from 'antd'
import { CheckCircleOutlined, CloseOutlined, WarningOutlined } from '@ant-design/icons'
import type { WarnSummary } from './warn'

interface ScheduleDockProps {
  open: boolean
  warn: WarnSummary
  onNavigate: (taskId: string) => void
  onClose: () => void
}

export const ScheduleDock: React.FC<ScheduleDockProps> = ({ open, warn, onNavigate, onClose }) => {
  if (!open) return null
  const items = warn.items.filter((item) => item.level !== 'none')

  return (
    <aside className="sched-dock" data-testid="sched-dock" aria-label="预警信息">
      <div className="sched-dock-head">
        <span className="sched-dock-title">
          <WarningOutlined />
          预警信息
        </span>
        {warn.red > 0 && <span className="sched-dock-count">{warn.red}</span>}
        <Button
          size="small"
          type="text"
          icon={<CloseOutlined />}
          onClick={onClose}
          aria-label="关闭预警信息"
          title="关闭预警信息"
        />
      </div>
      <div className="sched-dock-list">
        {items.length === 0 ? (
          <div className="sched-dock-empty">
            <CheckCircleOutlined />
            <span>{warn.unassigned > 0 ? `${warn.unassigned} 项未下达要求完成，暂无已比对预警` : '暂无预警'}</span>
          </div>
        ) : items.map((item) => {
          const late = item.level === 'red'
          const timing = item.overdue === null
            ? '不可比对'
            : item.overdue > 0
              ? `滞后 ${item.overdue} 天${item.band ? ` · ${item.band}` : ''}`
              : item.overdue === 0
                ? '压线完成'
                : `提前 ${Math.abs(item.overdue)} 天`
          return (
            <button
              key={item.taskId}
              type="button"
              className={`sched-dock-item ${late ? 'is-red' : 'is-green'}`}
              onClick={() => onNavigate(item.taskId)}
              data-testid={`sched-dock-warn-${item.taskId}`}
            >
              <span className="sched-dock-item-head">
                {late ? <WarningOutlined /> : <CheckCircleOutlined />}
                <strong>{item.name}</strong>
                {item.isMilestone && <span className="sched-dock-tag">里程碑</span>}
              </span>
              <span className="sched-dock-item-meta">{timing}</span>
              {item.note && <span className="sched-dock-item-hint">{item.note}</span>}
            </button>
          )
        })}
      </div>
    </aside>
  )
}
