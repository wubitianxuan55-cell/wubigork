import React from 'react'
import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  DownOutlined,
  ExperimentOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { Alert, Button } from 'antd'
import type {
  PantheonEvaluationCaseCoverage,
  PantheonEvaluationReport,
  PantheonEvaluationVariantSummary,
} from '../../gaea/lib/types/pantheon'

const REQUIRED_VARIANTS = [
  'single',
  'independent_synthesis',
  'roundtable',
  'targeted_challenge',
]

const VARIANT_META: Record<string, { label: string; note: string }> = {
  single: {
    label: '单席直答',
    note: '最低成本基线',
  },
  independent_synthesis: {
    label: '独立多答',
    note: '检查集成增量',
  },
  roundtable: {
    label: '圆桌会诊',
    note: '当前默认形态',
  },
  targeted_challenge: {
    label: '定向质询',
    note: '检查质疑增量',
  },
  red_team: {
    label: '红队评审',
    note: '可选高保障复核',
  },
  council_relay: {
    label: '圆桌接力',
    note: '历史路径',
  },
}
function variantMeta(variant: string) {
  return VARIANT_META[variant] || {
    label: variant || '未分类',
    note: '未登记路径',
  }
}

function formatDate(ms: number): string {
  if (!ms) return '尚未生成'
  return new Date(ms).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function completedCasesFor(report: PantheonEvaluationReport, variant: string): number {
  return report.cases.filter((item) => item.completedVariants?.includes(variant)).length
}

function latestRun(report: PantheonEvaluationReport, coverage: PantheonEvaluationCaseCoverage) {
  return [...report.runs]
    .filter((run) => run.caseKey === coverage.key)
    .sort((left, right) => right.createdAt - left.createdAt)[0]
}

const EvaluationSkeleton: React.FC = () => (
  <div
    className="pantheon-evaluation is-loading"
    aria-label="正在读取协议对照"
    aria-busy="true"
  >
    <div className="pantheon-eval-skeleton-head">
      <span />
      <span />
      <span />
    </div>
    <div className="pantheon-eval-skeleton-grid">
      <span />
      <span />
      <span />
      <span />
    </div>
    <div className="pantheon-eval-skeleton-ledger">
      <span />
      <span />
      <span />
    </div>
  </div>
)

const VariantCoverage: React.FC<{
  variant: string
  report: PantheonEvaluationReport
  summary?: PantheonEvaluationVariantSummary
}> = ({ variant, report, summary }) => {
  const meta = variantMeta(variant)
  const completed = completedCasesFor(report, variant)
  const missing = Math.max(0, report.requiredCases - completed)
  const percent = report.requiredCases > 0
    ? Math.min(100, Math.round((completed / report.requiredCases) * 100))
    : 0
  return (
    <article className={`pantheon-eval-variant${missing === 0 ? ' is-covered' : ''}`}>
      <div className="pantheon-eval-variant-copy">
        <strong>{meta.label}</strong>
        <small>{meta.note}</small>
      </div>
      <span className="pantheon-eval-variant-count">
        {completed}<small>/{report.requiredCases}</small>
      </span>
      <div
        className="pantheon-eval-variant-progress"
        role="progressbar"
        aria-label={`${meta.label}已完成议题`}
        aria-valuemin={0}
        aria-valuemax={report.requiredCases}
        aria-valuenow={Math.min(completed, report.requiredCases)}
      >
        <i style={{ '--eval-progress': `${percent}%` } as React.CSSProperties} />
      </div>
      <div className="pantheon-eval-variant-foot">
        <span>{missing === 0 ? '覆盖已满足' : `还需 ${missing} 个议题`}</span>
        <span>
          {summary?.completedSessions
            ? `${summary.completedSessions} 场 · 均 ${summary.avgRounds || 0} 轮`
            : '尚无完成样本'}
        </span>
      </div>
    </article>
  )
}

export const PantheonEvaluation: React.FC<{
  report: PantheonEvaluationReport | null
  loading: boolean
  error: string
  onRefresh: () => void
  onStart: () => void
  onOpenSession: (id: string) => void
}> = ({ report, loading, error, onRefresh, onStart, onOpenSession }) => {
  if (loading && !report) return <EvaluationSkeleton />
  if (!report) {
    return (
      <section className="pantheon-evaluation pantheon-evaluation--error">
        <div className="pantheon-eval-error-copy">
          <span className="pantheon-eyebrow">协议对照</span>
          <h2>暂时无法读取对照记录。</h2>
          <p>议事记录没有被修改。你可以重试读取，或先发起一场新的对照议题。</p>
        </div>
        <Alert type="error" showIcon message={error || '读取评测报告失败'} />
        <div className="pantheon-eval-error-actions">
          <Button icon={<ReloadOutlined />} onClick={onRefresh}>重新读取</Button>
          <Button type="primary" icon={<ArrowRightOutlined />} iconPosition="end" onClick={onStart}>
            发起对照议题
          </Button>
        </div>
      </section>
    )
  }

  const requiredVariants = report.requiredVariants?.length
    ? report.requiredVariants
    : REQUIRED_VARIANTS
  const qualified = Math.max(0, report.qualifiedCases)
  const required = Math.max(1, report.requiredCases)
  const percent = Math.min(100, Math.round((qualified / required) * 100))
  const summaryByVariant = new Map(report.variants.map((item) => [item.variant, item]))
  const gateCopy = report.gateReady
    ? '可进入协议取舍'
    : report.caseCount > 0 ? '继续补齐同题证据' : '等待第一组议题'

  return (
    <section className="pantheon-evaluation" aria-labelledby="pantheon-evaluation-title">
      <header className="pantheon-eval-hero">
        <div className="pantheon-eval-hero-copy">
          <div className="pantheon-eval-kicker">
            <span className="pantheon-eyebrow">协议对照</span>
            <span><SafetyCertificateOutlined aria-hidden /> 本地只读</span>
          </div>
          <h2 id="pantheon-evaluation-title">协议对照</h2>
          <p>
            同一议题完成四条路径后，再比较议事形态。系统复算运行证据，结论仍由你签注。
          </p>
          <div className="pantheon-eval-actions">
            <Button
              type="primary"
              size="large"
              icon={<ArrowRightOutlined />}
              iconPosition="end"
              onClick={onStart}
              data-testid="pantheon-evaluation-start"
            >
              发起对照议题
            </Button>
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={onRefresh}
              data-testid="pantheon-evaluation-refresh"
            >
              重新读取
            </Button>
            <span className="pantheon-eval-updated">读数于 {formatDate(report.generatedAt)}</span>
          </div>
        </div>

        <aside className={`pantheon-eval-gate${report.gateReady ? ' is-ready' : ''}`}>
          <div className="pantheon-eval-gate-head">
            <span><ExperimentOutlined aria-hidden /> 证据门禁</span>
            <small>{report.gateReady ? '已就绪' : '采集中'}</small>
          </div>
          <div className="pantheon-eval-gate-score" data-testid="pantheon-evaluation-gate">
            <strong>{qualified}</strong>
            <span>/ {report.requiredCases}</span>
          </div>
          <p>{gateCopy}</p>
          <div
            className="pantheon-eval-gate-progress"
            role="progressbar"
            aria-label="协议对照门禁进度"
            aria-valuemin={0}
            aria-valuemax={required}
            aria-valuenow={Math.min(qualified, required)}
          >
            <span style={{ '--eval-progress': `${percent}%` } as React.CSSProperties} />
          </div>
          <div className="pantheon-eval-gate-meta">
            <span>{report.caseCount} 个议题</span>
            <span>{report.variants.length} 类路径已有记录</span>
            <b>{percent}%</b>
          </div>
        </aside>
      </header>

      {error && (
        <Alert
          className="pantheon-eval-inline-alert"
          type="warning"
          showIcon
          message={error}
          action={<Button size="small" onClick={onRefresh}>重试</Button>}
        />
      )}

      <section className="pantheon-eval-section pantheon-eval-variants" aria-labelledby="pantheon-eval-variants-title">
        <header className="pantheon-eval-section-head">
          <div>
            <span className="pantheon-eyebrow">四链覆盖</span>
            <h3 id="pantheon-eval-variants-title">四条路径，同一组议题。</h3>
          </div>
          <p>门禁只计已完成会话。红队评审可选，不计入四链基线。</p>
        </header>
        <div className="pantheon-eval-variant-grid" data-testid="pantheon-evaluation-variants">
          {requiredVariants.map((variant) => (
            <VariantCoverage
              key={variant}
              variant={variant}
              report={report}
              summary={summaryByVariant.get(variant)}
            />
          ))}
        </div>
      </section>

      <section className="pantheon-eval-section pantheon-eval-ledger" aria-labelledby="pantheon-eval-ledger-title">
        <header className="pantheon-eval-section-head pantheon-eval-ledger-head">
          <div>
            <span className="pantheon-eyebrow">议题记录</span>
            <h3 id="pantheon-eval-ledger-title">按原始简报自动归组。</h3>
          </div>
          <span className="pantheon-eval-ledger-total">{report.caseCount} 组</span>
        </header>
        {report.cases.length === 0 ? (
          <div className="pantheon-eval-empty">
            <span aria-hidden><ExperimentOutlined /></span>
            <div>
              <strong>同题账本还是空的</strong>
              <p>先发起一个真实议题。完成后，系统会自动按材料与简报归组，无需手工登记。</p>
            </div>
            <Button onClick={onStart} icon={<ArrowRightOutlined />} iconPosition="end">
              发起第一个议题
            </Button>
          </div>
        ) : (
          <ol className="pantheon-eval-case-list" data-testid="pantheon-evaluation-cases">
            {report.cases.map((coverage) => {
              const run = latestRun(report, coverage)
              const completed = coverage.completedVariants || []
              const missing = coverage.missingVariants || []
              return (
                <li key={coverage.key} className={missing.length === 0 ? 'is-covered' : ''}>
                  <span className="pantheon-eval-case-key" aria-hidden>
                    {coverage.key.replace(/^case_/, '').slice(0, 8)}
                  </span>
                  <div className="pantheon-eval-case-copy">
                    <strong>{coverage.title || '未命名议题'}</strong>
                    <span>{report.runs.filter((item) => item.caseKey === coverage.key).length} 场记录</span>
                  </div>
                  <div className="pantheon-eval-case-coverage" aria-label="议题覆盖">
                    {completed.map((variant) => (
                      <span key={variant} className="is-complete">
                        <CheckCircleOutlined aria-hidden /> {variantMeta(variant).label}
                      </span>
                    ))}
                    {missing.map((variant) => (
                      <span key={variant} className="is-missing">
                        <ClockCircleOutlined aria-hidden /> 缺 {variantMeta(variant).label}
                      </span>
                    ))}
                  </div>
                  <div className="pantheon-eval-case-action">
                    {run ? (
                      <button
                        type="button"
                        onClick={() => onOpenSession(run.sessionId)}
                        aria-label={`打开最近纪要：${coverage.title}`}
                      >
                        最近纪要 <ArrowRightOutlined aria-hidden />
                      </button>
                    ) : (
                      <span>尚未完成</span>
                    )}
                  </div>
                </li>
              )
            })}
          </ol>
        )}
      </section>

      <section className="pantheon-eval-boundary" aria-labelledby="pantheon-eval-boundary-title">
        <header>
          <span className="pantheon-eyebrow">判定边界</span>
          <h3 id="pantheon-eval-boundary-title">机器复算，人工签注。</h3>
        </header>
        <div className="pantheon-eval-boundary-grid">
          <article>
            <CheckCircleOutlined aria-hidden />
            <div>
              <strong>可以复算</strong>
              <p>轮次、调用与重试、耗时、席位缺口、证据句柄、台账去向和用户签注。</p>
            </div>
          </article>
          <article>
            <ClockCircleOutlined aria-hidden />
            <div>
              <strong>仍须人工评审</strong>
              <p>事实错误、关键反例、真实成本、少数意见是否正确，以及是否愿意再次使用。</p>
            </div>
          </article>
        </div>
        <div className="pantheon-eval-boundary-meta">
          <span>Token / 费用：未采集</span>
          <span>CLI 模型版本：须在实验记录中固定</span>
          <span>固定投票：不在当前协议内</span>
        </div>
        <details className="pantheon-eval-warning-ledger">
          <summary>
            查看采集与判定提示
            <span>{report.warnings?.length || 0} 条</span>
            <DownOutlined aria-hidden />
          </summary>
          <ul data-testid="pantheon-evaluation-warnings">
            {(report.warnings || []).map((warning) => <li key={warning}>{warning}</li>)}
          </ul>
        </details>
      </section>
      <p className="pantheon-eval-foot-note">
        门禁未就绪时，不宣称任何议事协议更优。最近读取 {formatDate(report.generatedAt)}，共 {report.runs.length} 场原始记录。
      </p>
    </section>
  )
}
