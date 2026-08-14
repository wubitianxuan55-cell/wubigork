import React, { useCallback, useState } from 'react'
import { Button, message } from 'antd'
import { ExperimentOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { EmptyState, KpiTile, SectionHead, StatusChip } from './ui'
import { app } from '../../gaea/lib/bridge'
import type { RetrievalEvalReport } from '../../gaea/lib/types'

/**
 * RetrievalEvalSection — 检索质量测评（阶段 5「进料与质量」）
 *
 * 对内置查询集跑一遍跨库统一检索（GaeaRetrievalEvalRun：关键词 + 语义），
 * 统计平均 recall@10 并与达标门槛（后端固定 0.8）比较，给出通过状态与
 * 逐查询命中明细。展示风格对齐 BenchmarkSection（mc-panel / KpiTile / mc-table）。
 */

// 通过/未达状态色：recall ≥ 门槛为通过（ok），否则未达（danger）。
function recallTone(recall: number, threshold: number): 'ok' | 'danger' {
  return recall >= threshold ? 'ok' : 'danger'
}

const fmtPct = (n: number) => `${(n * 100).toFixed(0)}%`

export function RetrievalEvalSection() {
  const [report, setReport] = useState<RetrievalEvalReport | null>(null)
  const [running, setRunning] = useState(false)

  const run = useCallback(async () => {
    setRunning(true)
    try {
      const r = await app.RetrievalEvalRun()
      setReport(r)
      message.success(`检索质量测评完成：recall@10 = ${fmtPct(r.recallAt10)}（${r.passed ? '通过' : '未达'}）`)
    } catch (err: any) {
      message.error(err?.message || '检索质量测评失败（需本地检索服务可用）')
    } finally {
      setRunning(false)
    }
  }, [])

  return (
    <div className="mc-drawer-body">
      <SectionHead
        icon={<ExperimentOutlined />}
        title="检索质量测评"
        desc="对内置查询集跑一遍跨库统一检索（关键词 + 语义），统计平均 recall@10（取前 10 命中中期望命中的覆盖率），与达标门槛 0.8 比较给出通过状态。"
        extra={
          <Button
            size="small"
            type="primary"
            icon={<PlayCircleOutlined />}
            loading={running}
            onClick={() => void run()}
          >
            运行检索质量测评
          </Button>
        }
      />

      {/* 结果区 */}
      <div className="mc-panel" style={{ marginTop: 12 }}>
        <div className="mc-panel-title" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ExperimentOutlined /> 测评结果
          {report && (
            <Button size="small" icon={<ReloadOutlined />} loading={running} onClick={() => void run()}>
              重新运行
            </Button>
          )}
        </div>
        {!report ? (
          <EmptyState
            icon={<ExperimentOutlined />}
            title="尚未运行测评"
            hint="点击「运行检索质量测评」，结果将在这里汇总（recall@10 / 门槛 / 通过状态 + 逐查询命中明细）"
            compact
          />
        ) : (
          <>
            <div className="mc-overview-grid" style={{ gridTemplateColumns: 'repeat(4, minmax(0,1fr))', marginTop: 8 }}>
              <KpiTile icon={<ExperimentOutlined />} label="测评查询" value={report.total}
                hint="内置查询集（成本/知识/办公/资料）" />
              <KpiTile label="平均 recall@10" value={fmtPct(report.recallAt10)}
                hint="前 10 命中期望覆盖率" />
              <KpiTile label="达标门槛" value={fmtPct(report.threshold)}
                hint="recall@10 需 ≥ 该值（后端固定）" />
              <KpiTile label="通过状态" value={
                <StatusChip tone={report.passed ? 'ok' : 'danger'} dot>
                  {report.passed ? '通过' : '未达'}
                </StatusChip>
              } hint={report.passed ? '达到门槛，检索质量合格' : '低于门槛，建议优化索引或查询'} />
            </div>

            {/* 逐查询命中明细 */}
            <div className="mc-table" style={{ marginTop: 12 }}>
              <div className="mc-table-head" style={{ gridTemplateColumns: 'minmax(150px, 1.1fr) minmax(170px, 1.2fr) minmax(200px, 1.5fr) 96px' }}>
                <div>查询</div>
                <div>期望命中</div>
                <div>实际前 10 命中</div>
                <div className="mc-table-cell-num">recall@10</div>
              </div>
              {report.perQuery.map((q, i) => (
                <div
                  key={q.query + i}
                  className="mc-table-row"
                  style={{ gridTemplateColumns: 'minmax(150px, 1.1fr) minmax(170px, 1.2fr) minmax(200px, 1.5fr) 96px', fontSize: 11.5 }}
                >
                  <div style={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={q.query}>
                    {q.query}
                  </div>
                  <div style={{ minWidth: 0, color: 'var(--mc-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={q.expected.join('、')}>
                    {q.expected.join('、') || '—'}
                  </div>
                  <div style={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={q.topHits.join('、')}>
                    {q.topHits.join('、') || '—'}
                  </div>
                  <div className="mc-table-cell-num" style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 6 }}>
                    <StatusChip tone={recallTone(q.recall, report.threshold)} dot>{fmtPct(q.recall)}</StatusChip>
                  </div>
                </div>
              ))}
            </div>

            {report.note && (
              <div style={{ color: 'var(--mc-muted)', fontSize: 11, marginTop: 8 }}>{report.note}</div>
            )}
          </>
        )}
      </div>

      <div style={{ color: 'var(--mc-muted)', fontSize: 11, marginTop: 8 }}>
        口径说明：recall@10 = 单条查询前 10 条命中中包含的期望命中数 ÷ 期望命中总数；平均 recall@10 为全部查询的均值，达标门槛后端固定 0.8。
      </div>
    </div>
  )
}

export default RetrievalEvalSection
