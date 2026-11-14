/**
 * schedule/ReviewPanel.tsx — 检视（多期进度快照）面板（斑马复刻 S2）
 *
 * 对标斑马「检视时间」的交互：**计划是连续推演的，执行是离散打点的**。这个面板
 * 是"打点 + 对比 + 拉直"的唯一入口：
 *  - 打一期检视 = 把当前各任务进度存成快照（`at`=检视工作日，`date`=对应日期）；
 *  - 期列表按时间轴升序展示，标出冻结（拉直过）与当期完成量；
 *  - 对比区给两组数：**当期 vs 上一期**（完成量变化，逐任务有变化的行）与
 *    **计划应完成 vs 实际完成**（用 CPM 算到该工作日应完成的量，斑马"当期/累计"口径）；
 *  - 拉直 = 把回填写回任务进度并冻结该期（不可重复拉直，历史不可揉）。
 *
 * props 只收 cpm（页面级结果）；工程与动作一律取自 useScheduleStore，与
 * BaselinesPanel 同款结构（可整体替换进任何 Popover）。
 */
import React, { useMemo, useState } from 'react'
import { Button, Input, Popconfirm, Space, Table, Tag } from 'antd'
import { useScheduleStore } from './store'
import {
  canAddReview,
  compareReviews,
  defaultReviewDate,
  latestReview,
  projectPercent,
  reviewDate,
  reviewProgress,
  reviewWorkday,
  sortedReviews,
} from './reviews'
import type { CpmResult, SchedReview } from './types'

/** 偏差文本：+N / -N / ±0 */
function fmtDelta(n: number): string {
  return n > 0 ? `+${n}` : `${n}`
}

export const ReviewPanel: React.FC<{ cpm: CpmResult }> = ({ cpm }) => {
  const project = useScheduleStore((s) => s.project)
  const addReview = useScheduleStore((s) => s.addReview)
  const removeReview = useScheduleStore((s) => s.removeReview)
  const straightenReview = useScheduleStore((s) => s.straightenReview)

  const reviews = useMemo(() => sortedReviews(project), [project])
  const latest = useMemo(() => latestReview(project), [project])
  const [date, setDate] = useState(() => defaultReviewDate(project))
  const [msg, setMsg] = useState<string | null>(null)

  const at = reviewWorkday(project, date)
  const gate = canAddReview(project, at)
  const diff = latest ? compareReviews(project, reviews[reviews.length - 2] ?? null, latest) : null
  const prog = latest ? reviewProgress(project, latest, cpm) : null

  const doAdd = () => {
    const r = addReview(at, date)
    setMsg(r.ok ? `已记录第 ${at} 工作日的检视` : r.reason ?? '记录失败')
  }

  const doStraighten = (r: SchedReview) => {
    const res = straightenReview(r.at)
    if (!res.ok) setMsg(res.reason ?? '拉直失败')
    else if (res.changed === 0) setMsg('该期回填与当前进度一致，无需拉直')
    else setMsg(`已拉直：${res.changed} 个任务进度写回计划，该期已冻结`)
  }

  return (
    <div data-testid="sched-review-panel" style={{ width: 420 }}>
      {/* 打一期检视 */}
      <Space size={6} wrap>
        <span className="sched-dim">检视日期</span>
        <Input
          type="date"
          size="small"
          value={date}
          onChange={(e) => {
            setDate(e.target.value)
            setMsg(null)
          }}
          style={{ width: 140 }}
          data-testid="sched-review-date"
        />
        <Button size="small" type="primary" onClick={doAdd} disabled={!gate.ok} data-testid="sched-review-add">
          打一期检视
        </Button>
        <span className="sched-dim" style={{ fontSize: 12 }}>第 {at} 工作日</span>
      </Space>
      {!gate.ok && (
        <div className="sched-dim" style={{ marginTop: 4, fontSize: 12 }}>{gate.reason}</div>
      )}
      {msg && (
        <div style={{ marginTop: 6, fontSize: 12 }} data-testid="sched-review-msg">{msg}</div>
      )}

      {/* 期列表 */}
      <div style={{ margin: '10px 0 4px', fontWeight: 600, fontSize: 13 }}>
        检视序列（{reviews.length}）
      </div>
      {reviews.length === 0 ? (
        <span className="sched-dim" style={{ fontSize: 12 }}>
          还没有检视。每期检视会把当时的进度存成快照，之后再打点就能对比"哪一项变了、超前还是拖后"。
        </span>
      ) : (
        <Table<SchedReview>
          size="small"
          pagination={false}
          rowKey={(r) => String(r.at)}
          dataSource={reviews}
          columns={[
            {
              title: '检视',
              dataIndex: 'at',
              width: 132,
              render: (_v, r) => (
                <span>
                  第 {r.at} 工作日
                  <span className="sched-dim" style={{ fontSize: 11 }}>（{r.date}）</span>
                </span>
              ),
            },
            {
              title: '完成量',
              width: 66,
              align: 'right' as const,
              render: (_v, r) => `${projectPercent(project, r)}%`,
            },
            {
              title: '状态',
              width: 70,
              render: (_v, r) =>
                r.frozen ? <Tag color="blue">已冻结</Tag> : <Tag>可拉直</Tag>,
            },
            {
              title: '',
              width: 96,
              render: (_v, r) => (
                <Space size={2}>
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0, height: 20 }}
                    disabled={r.frozen}
                    onClick={() => doStraighten(r)}
                    title="把这一期的回填写回任务进度并冻结"
                  >
                    拉直
                  </Button>
                  <Popconfirm title="删除这一期检视？" onConfirm={() => removeReview(r.at)}>
                    <Button type="link" size="small" danger style={{ padding: 0, height: 20 }}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      )}

      {/* 当期：计划应完成 vs 实际完成 */}
      {latest && prog && (
        <>
          <div style={{ margin: '10px 0 4px', fontWeight: 600, fontSize: 13 }}>当期（第 {latest.at} 工作日）</div>
          <div
            data-testid="sched-review-progress"
            style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 6, fontSize: 12 }}
          >
            <div>计划应完成<br /><b>{prog.plannedPercent}%</b>（{prog.plannedWorkdays} 工日）</div>
            <div>实际完成<br /><b>{prog.actualPercent}%</b>（{prog.actualWorkdays} 工日）</div>
            <div>
              偏差<br />
              <b className={prog.gapPercent < 0 ? 'text-err' : 'text-ok'}>{fmtDelta(prog.gapPercent)}%</b>
            </div>
          </div>
          <div className="sched-dim" style={{ fontSize: 11, marginTop: 2 }}>
            共 {prog.totalWorkdays} 个有效工日（里程碑不计分母）
          </div>
        </>
      )}

      {/* 期际对比：只列有变化的行 */}
      {latest && diff && (
        <>
          <div style={{ margin: '10px 0 4px', fontWeight: 600, fontSize: 13 }}>
            较上一期（{diff.fromAt === null ? '任务当前进度' : `第 ${diff.fromAt} 工作日`}）：
            完成量 {diff.fromPercent}% → {diff.toPercent}%（{fmtDelta(diff.deltaPercent)}%）
          </div>
          <div data-testid="sched-review-diff">
            {diff.rows.length === 0 ? (
              <span className="sched-dim" style={{ fontSize: 12 }}>没有任务完成量发生变化。</span>
            ) : (
              <Table
                size="small"
                pagination={false}
                rowKey="id"
                dataSource={diff.rows}
                columns={[
                  { title: '任务', dataIndex: 'name' },
                  { title: '上期', dataIndex: 'fromPct', width: 56, align: 'right' as const, render: (v: number) => `${v}%` },
                  { title: '本期', dataIndex: 'toPct', width: 56, align: 'right' as const, render: (v: number) => `${v}%` },
                  {
                    title: '变化',
                    dataIndex: 'delta',
                    width: 62,
                    align: 'right' as const,
                    render: (v: number) => (
                      <span className={v > 0 ? 'text-ok' : 'text-err'}>{fmtDelta(v)}%</span>
                    ),
                  },
                  { title: '状态', dataIndex: 'status', width: 66 },
                ]}
              />
            )}
          </div>
        </>
      )}

      {latest && (
        <div className="sched-dim" style={{ fontSize: 11, marginTop: 8 }}>
          前锋线停在第 {latest.at} 工作日（{reviewDate(project, latest.at)}）：横道/网络图按这根线判断"应该做到哪、实际做到哪"。
        </div>
      )}
    </div>
  )
}
