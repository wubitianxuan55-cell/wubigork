/**
 * schedule/WhatIfPanel.tsx — 工期 what-if 推演面板（v4.523）
 *
 * 消费 whatif.ts（v4.491 纯函数，此前零 UI）：「本任务工期 ±N 天，总工期与
 * 关键线路怎么变」。经 slot 挂进 TaskInspector 抽屉（保持检查器「零排程计算」
 * 契约——delta 交互态与 computeCpm 重算都在本组件，检查器只按位渲染）。
 * 只读口径：本地重算的对比快照，不改原计划、不写盘。里程碑/任务不存在/
 * 空计划按 whatIfDuration 的 reason 诚实降级，绝不伪造对比数字。
 */
import { useMemo, useState } from 'react'
import { Button, Tag } from 'antd'
import { ExperimentOutlined } from '@ant-design/icons'
import { whatIfDuration } from './whatif'
import type { SchedProject } from './types'

const STEPS: Array<[label: string, delta: number, testid: string]> = [
  ['−5', -5, 'sched-whatif-minus5'],
  ['−1', -1, 'sched-whatif-minus1'],
  ['重置', 0, 'sched-whatif-reset'],
  ['+1', 1, 'sched-whatif-plus1'],
  ['+5', 5, 'sched-whatif-plus5'],
]

export function WhatIfPanel({ project, taskId }: { project: SchedProject; taskId: string }) {
  const [delta, setDelta] = useState(0)
  // 切换任务后 delta 残留会误导（新任务的 ±N 是旧意图）——taskId 变即复位
  const [lastTask, setLastTask] = useState(taskId)
  if (lastTask !== taskId) {
    setLastTask(taskId)
    setDelta(0)
  }

  const result = useMemo(() => whatIfDuration(project, taskId, delta), [project, taskId, delta])

  const durTone = !result.ok ? '' : result.delta > 0 ? 'text-red-400' : result.delta < 0 ? 'text-emerald-500' : ''

  return (
    <div data-testid="sched-whatif">
      <div style={{ margin: '14px 0 6px', fontWeight: 600, fontSize: 13, display: 'flex', alignItems: 'center', gap: 6 }}>
        <ExperimentOutlined />
        工期推演（what-if）
      </div>
      <div className="sched-dim" style={{ fontSize: 11, marginBottom: 6 }}>
        本地重算的对比快照，不改原计划：调步进看「本任务工期变化 N 天后，总工期与关键线路怎么变」。
      </div>

      {/* 步进组：−5 / −1 / 重置 / +1 / +5；当前累计 delta 用 tag 显（重置=0） */}
      <div style={{ display: 'flex', gap: 4, alignItems: 'center', flexWrap: 'wrap' }}>
        {STEPS.map(([label, step, testid]) => (
          <Button
            key={testid}
            size="small"
            data-testid={testid}
            disabled={!result.ok}
            onClick={() => setDelta(step === 0 ? 0 : delta + step)}
          >
            {label}
          </Button>
        ))}
        <Tag data-testid="sched-whatif-delta" style={{ marginInlineEnd: 0 }}>
          {delta > 0 ? `+${delta}` : delta} 天
        </Tag>
      </div>

      <div
        data-testid="sched-whatif-result"
        style={{ marginTop: 8, padding: '8px 10px', border: '1px solid rgba(5,5,5,0.1)', borderRadius: 8 }}
      >
        {!result.ok ? (
          <span className="sched-dim">{result.reason}</span>
        ) : delta === 0 ? (
          <span>
            基线总工期 <b className="tabular-nums">{result.baseDuration}</b> 工作日——调步进开始推演。
          </span>
        ) : (
          <>
            <div>
              总工期 <b className="tabular-nums">{result.baseDuration}</b> →{' '}
              <b className={`tabular-nums ${durTone}`}>{result.whatIfDuration}</b> 工作日
              （<span className={durTone}>{result.delta > 0 ? `+${result.delta}` : result.delta}</span>）
            </div>
            {result.newlyCritical.length > 0 && (
              <div style={{ marginTop: 4, fontSize: 12 }}>
                <Tag color="red" style={{ marginInlineEnd: 4 }}>新进关键</Tag>
                {result.newlyCritical.join('、')}
              </div>
            )}
            {result.noLongerCritical.length > 0 && (
              <div style={{ marginTop: 4, fontSize: 12 }}>
                <Tag style={{ marginInlineEnd: 4 }}>脱离关键</Tag>
                {result.noLongerCritical.join('、')}
              </div>
            )}
            {result.newlyCritical.length === 0 && result.noLongerCritical.length === 0 && (
              <div className="sched-dim" style={{ marginTop: 4, fontSize: 12 }}>
                关键线路不变。
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
