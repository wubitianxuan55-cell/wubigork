/**
 * AttributionSection — 模型中心「成本归因」分区（阶段七 7.1-2 C 线）
 *
 * 按功能绑定键聚合的用量/成本账本视图：KPI 行（总成本/总 token/调用/成功率）
 * + 每功能一节引擎×模型明细表（「未标记」桶=历史数据，排最后）+ 组内命中时
 * 建议制改绑卡（分差/理由/证据 + 采纳/忽略；采纳后绑定面靠既有
 * feature-model-changed 事件自动刷新）。建议只在评分差超阈值时出现，
 * 采纳=用户动作（不自动改绑红线）。zh 单语（模型中心惯例）。
 */
import { Button, Table, type TableProps } from 'antd'
import {
  CheckCircleOutlined, NumberOutlined, PieChartOutlined, ReloadOutlined,
  SwapOutlined, ThunderboltOutlined, WalletOutlined,
} from '@ant-design/icons'
import type { FeatureUsageSummary, RouteSuggestion } from '../../api/engines'
import { EmptyState, KpiTile, SectionHead, StatusChip } from './ui'
import { useAttributionState } from './hooks/useAttributionState'
import { FEATURES, engineLabel, fmtCost } from './utils'

/** 「未标记」桶（feature="" 历史/未标记数据）的 testid 后缀 */
const UNMARKED_KEY = 'unmarked'

/** 功能绑定键 → 中文标签：复用 FEATURES label 映射；空串=未标记；未收录键原样透传 */
const featureLabel = (key: string): string => {
  if (!key) return '未标记'
  return FEATURES.find(f => f.key === key)?.label || key
}

/** 功能分组节：按成本降序排组内明细行 */
interface FeatureGroup {
  feature: string
  rows: FeatureUsageSummary[]
  cost: number
}

const groupCost = (rows: FeatureUsageSummary[]): number =>
  rows.reduce((a, r) => a + (r.cost_cny ?? 0), 0)

/** 账本 features 桶按 feature 分组 → 节清单：已知功能键（FEATURES 序）在前、
 *  账本独有键按成本降序居中、「未标记」（feature=""）最后；已知键无数据也
 *  出节（空表态），保证八个功能绑定键始终可查。 */
function buildGroups(features: FeatureUsageSummary[]): FeatureGroup[] {
  const byFeature = new Map<string, FeatureUsageSummary[]>()
  for (const row of features) {
    const list = byFeature.get(row.feature) || []
    list.push(row)
    byFeature.set(row.feature, list)
  }
  const known = FEATURES
    .map(f => f.key)
    .filter(k => k !== '')
  const ledgerOnly = Array.from(byFeature.keys())
    .filter(k => k !== '' && !known.includes(k))
    .sort((a, b) => groupCost(byFeature.get(b) ?? []) - groupCost(byFeature.get(a) ?? []))
  // 「未标记」桶=历史数据兜底：有行才出节，新装环境不渲染空桶噪音
  const ordered = [...known, ...ledgerOnly, ...(byFeature.has('') ? [''] : [])]
  return ordered.map(feature => {
    const rows = [...(byFeature.get(feature) ?? [])]
      .sort((a, b) => (b.cost_cny ?? 0) - (a.cost_cny ?? 0))
    return { feature, rows, cost: groupCost(rows) }
  })
}

/** 明细表列：引擎/模型/调用/Token（入+出合并）/成本/均时长/成功率 */
const columns: TableProps<FeatureUsageSummary>['columns'] = [
  {
    title: '引擎', dataIndex: 'engine_id', key: 'engine', width: 130,
    render: (id: string) => engineLabel({ id }),
  },
  {
    title: '模型', dataIndex: 'model', key: 'model', ellipsis: true,
    render: (m: string) => <span title={m}>{m || '—'}</span>,
  },
  {
    title: '调用', dataIndex: 'calls', key: 'calls', width: 80, align: 'right',
    render: (v: number) => (v ?? 0).toLocaleString(),
  },
  {
    title: 'Token（入+出）', key: 'tokens', width: 150, align: 'right',
    render: (_, r) => {
      const total = (r.tokens_in ?? 0) + (r.tokens_out ?? 0)
      return (
        <span title={`入 ${r.tokens_in?.toLocaleString() ?? 0} / 出 ${r.tokens_out?.toLocaleString() ?? 0}`}>
          {total.toLocaleString()}
        </span>
      )
    },
  },
  {
    title: '成本', dataIndex: 'cost_cny', key: 'cost', width: 90, align: 'right',
    render: (v: number) => fmtCost(v ?? 0, 'CNY'),
  },
  {
    title: '均时长', dataIndex: 'avg_ms', key: 'avg_ms', width: 90, align: 'right',
    render: (v: number) => `${Math.round(v ?? 0).toLocaleString()} ms`,
  },
  {
    title: '成功率', dataIndex: 'success_rate', key: 'rate', width: 80, align: 'right',
    render: (v: number) => `${((v ?? 0) * 100).toFixed(1)}%`,
  },
]

/** 建议卡：命中该功能时显示在组表格上方（单功能至多一条，后端契约） */
function SuggestionCard({
  suggestion,
  acting,
  onApply,
  onIgnore,
}: {
  suggestion: RouteSuggestion
  acting: boolean
  onApply: (id: string) => void
  onIgnore: (id: string) => void
}) {
  const s = suggestion
  const featureKey = s.feature || UNMARKED_KEY
  const toLabel = `${engineLabel({ id: s.to.engine_id })} · ${s.to.model}`
  return (
    <div className="mc-bind-card" data-testid={`mc-suggest-${featureKey}`} style={{ marginBottom: 8 }}>
      <div className="mc-bind-head">
        <span className="mc-bind-title">
          <SwapOutlined style={{ marginRight: 4 }} />
          改绑建议：{engineLabel({ id: s.from.engine_id })} / {s.from.model} → {toLabel}
        </span>
        <StatusChip tone="accent">评分差 +{(s.score_gap ?? 0).toFixed(2)}</StatusChip>
      </div>
      <div className="mc-bind-desc">{s.reason}</div>
      <div className="mc-bind-meta">{s.evidence}</div>
      <div className="mc-bind-row">
        <StatusChip tone={s.to.is_local ? 'ok' : 'neutral'} dot>{s.to.is_local ? '本地' : '云端'}</StatusChip>
        {s.to.success_rate != null && (
          <StatusChip tone="neutral">成功率 {((s.to.success_rate ?? 0) * 100).toFixed(1)}%</StatusChip>
        )}
        <StatusChip tone="neutral">{fmtCost(s.to.cost_cny ?? 0, 'CNY')}</StatusChip>
        <span style={{ flex: 1 }} />
        <Button
          size="small" type="primary" loading={acting}
          onClick={() => onApply(s.id)}
          data-testid={`mc-suggest-${featureKey}-apply`}
        >
          采纳
        </Button>
        <Button
          size="small" disabled={acting}
          onClick={() => onIgnore(s.id)}
          data-testid={`mc-suggest-${featureKey}-ignore`}
        >
          忽略
        </Button>
      </div>
    </div>
  )
}

export function AttributionSection() {
  const { ledger, suggestions, loading, loadError, actingId, reload, handleApply, handleIgnore } = useAttributionState()

  const groups = ledger ? buildGroups(ledger.features ?? []) : []
  const isEmpty = !loadError && !loading && (!ledger || (ledger.features ?? []).length === 0)
  const suggestFor = (feature: string): RouteSuggestion | undefined =>
    suggestions.find(s => s.feature === feature)

  return (
    <section className="mc-section" data-testid="mc-attribution">
      <SectionHead
        icon={<PieChartOutlined />}
        title="成本归因"
        desc="按功能绑定键聚合的模型用量与成本账本；历史无标记数据归「未标记」桶；本地引擎成本恒 0 属实呈现"
      />

      {loadError && (
        <div className="mc-bind-card" data-testid="mc-attribution-error" style={{ marginBottom: 12 }}>
          <div className="mc-bind-head">
            <span className="mc-bind-title">成本归因数据读取失败</span>
            <Button size="small" icon={<ReloadOutlined />} loading={loading} onClick={() => void reload()}>重试</Button>
          </div>
          <div className="mc-bind-desc">{loadError}</div>
          <div className="mc-bind-meta">旧版后端无 GaeaRouteLedger/GaeaRouteSuggestions 绑定时也会这样——升级后端后重试</div>
        </div>
      )}

      {loading && !ledger && !loadError && <div className="mc-bind-meta">加载中…</div>}

      {isEmpty && (
        <EmptyState
          icon={<PieChartOutlined />}
          title="暂无归因数据"
          hint="对话、小说、办公等功能调用模型后，这里会按功能聚合调用、Token、成本与成功率"
        />
      )}

      {ledger && (ledger.features ?? []).length > 0 && (
        <>
          <div className="mc-overview-grid" data-testid="mc-attribution-kpi">
            <KpiTile
              icon={<WalletOutlined />}
              label="总成本"
              value={fmtCost(ledger.total?.cost_cny ?? 0, 'CNY')}
              hint={`账本生成于 ${ledger.generated_at || '—'}`}
            />
            <KpiTile
              icon={<NumberOutlined />}
              label="总 Token（入+出）"
              value={((ledger.total?.tokens_in ?? 0) + (ledger.total?.tokens_out ?? 0)).toLocaleString()}
              hint={`入 ${(ledger.total?.tokens_in ?? 0).toLocaleString()} / 出 ${(ledger.total?.tokens_out ?? 0).toLocaleString()}`}
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="总调用"
              value={(ledger.total?.calls ?? 0).toLocaleString()}
            />
            <KpiTile
              icon={<CheckCircleOutlined />}
              label="成功率"
              value={`${((ledger.total?.success_rate ?? 0) * 100).toFixed(1)}%`}
              hint={`平均耗时 ${Math.round(ledger.total?.avg_ms ?? 0)} ms`}
            />
          </div>

          {groups.map(g => {
            const suggestion = suggestFor(g.feature)
            return (
              <div
                key={g.feature}
                data-testid={`mc-attribution-row-${g.feature || UNMARKED_KEY}`}
                style={{ marginTop: 16 }}
              >
                <div className="mc-bind-row" style={{ marginBottom: 6 }}>
                  <span className="mc-bind-title">{featureLabel(g.feature)}</span>
                  <span className="mc-bind-meta">
                    {g.rows.length > 0
                      ? `${g.rows.length} 个引擎×模型 · 成本 ${fmtCost(g.cost, 'CNY')}`
                      : '暂无调用记录'}
                  </span>
                </div>
                {suggestion && (
                  <SuggestionCard
                    suggestion={suggestion}
                    acting={actingId === suggestion.id}
                    onApply={(id) => void handleApply(id)}
                    onIgnore={(id) => void handleIgnore(id)}
                  />
                )}
                <Table<FeatureUsageSummary>
                  size="small"
                  columns={columns}
                  dataSource={g.rows}
                  rowKey={(r) => `${r.engine_id}|${r.model}`}
                  pagination={false}
                />
              </div>
            )
          })}

          <div className="mc-bind-meta" style={{ marginTop: 12 }}>
            「未标记」为升级前无功能维度的历史数据；改绑建议由评分差超阈值触发（宁少勿扰），
            采纳后绑定面即时生效并自动刷新。
          </div>
        </>
      )}
    </section>
  )
}
