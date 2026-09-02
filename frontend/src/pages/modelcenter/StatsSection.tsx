import React from 'react'
import { Button, Popconfirm, Segmented } from 'antd'
import { CloudOutlined, DatabaseOutlined, DesktopOutlined, ReloadOutlined, SaveOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { EmptyState, KpiTile, StatusChip } from './ui'
import { billingModeLabel, costToCNY, engineColor, engineIcons, engineLabel, estimatePoints, findModelMeta, fmtCost, hasPointsCoef, isLocalEngine, USD_TO_CNY } from './utils'
import { RequestsTrendChart, TokenTrendChart, type StatsSort, type TrendRange } from './charts'
import { useModelCenter } from './context'
import { getUsageOverview, type UsageOverview, type UsageSide } from '../../api/engines'

// T5-3 KV 缓存命中率展示：后端 json tag 为 snake_case（与 engines.ts 类型一致），
// 全局命中率由后端计算（cache_hit_rate，0-1），云端/本地命中率由组件自算。
// 无任何缓存数据（hit+miss=0）时返回占位文案。
function fmtCacheRate(hit: number, miss: number, rate?: number): string {
  if ((hit ?? 0) + (miss ?? 0) === 0) return '暂无缓存数据'
  const r = Number.isFinite(rate) ? (rate as number) : (hit + miss > 0 ? hit / (hit + miss) : 0)
  return (r * 100).toFixed(1) + '%'
}
type OverviewCache = { cache_hit_tokens: number; cache_miss_tokens: number; cache_hit_rate: number }
function overviewCache(o: UsageOverview): UsageOverview & OverviewCache {
  return o as UsageOverview & OverviewCache
}
type SideCache = { cache_hit_tokens: number; cache_miss_tokens: number }
function sideCache(s: UsageSide): UsageSide & SideCache {
  return s as UsageSide & SideCache
}

export function StatsSection() {
  const {
    engines, callStats, statsSort, setStatsSort, trendRange, setTrendRange,
    loadCallStats, handleResetCallStats, trendData,
  } = useModelCenter()
  const [overview, setOverview] = React.useState<UsageOverview | null>(null)
  const [overviewLoading, setOverviewLoading] = React.useState(false)

  const loadOverview = React.useCallback(async () => {
    setOverviewLoading(true)
    try {
      setOverview(await getUsageOverview())
    } catch {
      setOverview(null)
    } finally {
      setOverviewLoading(false)
    }
  }, [])
  React.useEffect(() => { void loadOverview() }, [loadOverview])

  return (
    <div className="mc-drawer-body">
      <div className="mc-stats-head">
        <span className="mc-panel-title">
          <ThunderboltOutlined /> 模型调用统计
        </span>
        <span className="mc-section-extra">
          <Segmented
            size="small"
            value={statsSort}
            onChange={(v) => setStatsSort(v as StatsSort)}
            options={[
              { value: 'calls', label: '调用最多' },
              { value: 'tokens', label: 'Token 最多' },
              { value: 'cost', label: '费用最高' },
            ]}
          />
          <Button size="small" icon={<ReloadOutlined />} onClick={() => { loadCallStats(); void loadOverview() }}>刷新</Button>
          <Popconfirm
            title="确定清空全部模型调用统计？"
            description="此操作不可恢复，将清空次数、Token 与费用记录。"
            okText="清空"
            cancelText="取消"
            onConfirm={handleResetCallStats}
          >
            <Button size="small" danger>清空统计</Button>
          </Popconfirm>
        </span>
      </div>

      {callStats?.since && (
        <div style={{ color: 'var(--mc-muted)', fontSize: 11 }}>
          统计自 {callStats.since} · 按引擎 / 模型维度统计调用情况与估算费用
        </div>
      )}
      {/* B 刀：价格目录来源小注（估算费用/积分系数依据；旧 stats.json 无此字段时不显示） */}
      {callStats?.catalog_version && (
        <div style={{ color: 'var(--mc-muted)', fontSize: 11 }}>
          价格目录 {callStats.catalog_source || '内置'}（{callStats.catalog_version}）
        </div>
      )}

      {/* D3-2 本地 vs 云端分流对比 */}
      {overview && (overview.cloud.total_tokens > 0 || overview.local.total_tokens > 0) && (
        <div className="mc-panel" style={{ marginTop: 10 }}>
          <div className="mc-panel-title" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <CloudOutlined /> 本地 vs 云端 · 节省对比
            <span style={{ flex: 1 }} />
            <Button size="small" type="text" icon={<ReloadOutlined spin={overviewLoading} />}
              onClick={() => void loadOverview()} title="刷新分流统计" />
          </div>
          <div className="mc-overview-grid" style={{ gridTemplateColumns: 'repeat(3, minmax(0,1fr))' }}>
            <KpiTile icon={<CloudOutlined />} label="云端用量"
              value={fmtCost(overview.cloud.cost, 'CNY')}
              hint={`${overview.cloud.calls} 次 · ${overview.cloud.total_tokens.toLocaleString()} token · ${overview.cloud.engines.join(' / ') || '—'}`} />
            <KpiTile icon={<DesktopOutlined />} label="本地用量"
              value={overview.local.total_tokens.toLocaleString()}
              hint={`${overview.local.calls} 次 · 免费（Herdsman 等本地引擎）`} />
            <KpiTile icon={<SaveOutlined />} label="已节省"
              value={fmtCost(overview.savings.saved, 'CNY')}
              hint={`若走云端约需 ${fmtCost(overview.savings.would_cost_cloud, 'CNY')}（参考 ¥${overview.savings.ref_price_per_mtok.toFixed(2)}/百万 token）`} />
          </div>
          {/* T5-3 KV 缓存命中率：全局 + 云端/本地各自命中率（无数据时显示占位） */}
          <div className="mc-overview-grid" style={{ gridTemplateColumns: 'repeat(3, minmax(0,1fr))', marginTop: 10 }}>
            <KpiTile icon={<DatabaseOutlined />} label="全局 KV 命中率"
              value={fmtCacheRate(overviewCache(overview).cache_hit_tokens, overviewCache(overview).cache_miss_tokens, overviewCache(overview).cache_hit_rate)}
              hint={overviewCache(overview).cache_hit_tokens + overviewCache(overview).cache_miss_tokens > 0
                ? `${overviewCache(overview).cache_hit_tokens.toLocaleString()} 命中 / ${overviewCache(overview).cache_miss_tokens.toLocaleString()} 未命中`
                : undefined} />
            <KpiTile icon={<CloudOutlined />} label="云端命中率"
              value={fmtCacheRate(sideCache(overview.cloud).cache_hit_tokens, sideCache(overview.cloud).cache_miss_tokens)} />
            <KpiTile icon={<DesktopOutlined />} label="本地命中率"
              value={fmtCacheRate(sideCache(overview.local).cache_hit_tokens, sideCache(overview.local).cache_miss_tokens)} />
          </div>
          <div style={{ color: 'var(--mc-muted)', fontSize: 11, marginTop: 6 }}>{overview.savings.note}</div>
        </div>
      )}

      {!callStats || callStats.total_calls === 0 ? (
        <EmptyState
          icon={<ThunderboltOutlined />}
          title="暂无调用记录"
          hint="对话、语音、办公等模块调用模型后，这里会自动统计次数、Token、耗时与估算费用"
        />
      ) : (
        <>
          <div className="mc-overview-grid">
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="总调用"
              value={callStats.total_calls}
              hint={`成功 ${callStats.success_calls} · 失败 ${callStats.fail_calls}`}
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="Token 用量"
              value={callStats.total_tokens.toLocaleString()}
              hint={`入 ${callStats.input_tokens.toLocaleString()} / 出 ${callStats.output_tokens.toLocaleString()}`}
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="估算费用"
              value={fmtCost(callStats.total_cost, 'CNY')}
              hint={`美元按 1:${callStats?.usd_to_cny ?? USD_TO_CNY} 折算`}
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="成功率"
              value={`${((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)}%`}
              hint={`${callStats.per_model.length} 个模型`}
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="平均耗时"
              value={`${(callStats.avg_duration_ms / 1000).toFixed(1)}s`}
              hint={`累计 ${(callStats.total_duration_ms / 1000).toFixed(1)}s`}
            />
          </div>

          {/* 按引擎小计（后端 summary.engines；旧 stats.json 无此字段时整块不渲染）。
              编码套餐口径以 "<engine>@coding" 单列：Tokens/调用计入、费用 0。 */}
          {callStats.engines && Object.keys(callStats.engines).length > 0 && (
            <div className="mc-panel">
              <div className="mc-panel-body" style={{ gap: 8 }}>
                <div className="mc-field-row">
                  <span className="mc-panel-title">按引擎小计</span>
                  <span style={{ color: 'var(--mc-muted)', fontSize: 10 }}>编码套餐口径单列，费用不含积分内用量</span>
                </div>
                <div className="mc-table" style={{ border: 0 }}>
                  <div className="mc-table-head" style={{ gridTemplateColumns: 'minmax(120px, 1.6fr) minmax(90px, 1fr) 96px' }}>
                    <div>引擎</div>
                    <div className="mc-table-cell-num">Token</div>
                    <div className="mc-table-cell-num">估算费用</div>
                  </div>
                  {Object.entries(callStats.engines)
                    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
                    .map(([key, sub]) => {
                      const at = key.indexOf('@')
                      const codingKey = at > 0
                      const engName = codingKey ? `${key.slice(0, at)}（编码套餐）` : engineLabel({ id: key })
                      const costText = codingKey ? '积分内' : fmtCost(sub.estimated_cost_cny, 'CNY')
                      return (
                        <div
                          key={key}
                          className="mc-table-row"
                          style={{ gridTemplateColumns: 'minmax(120px, 1.6fr) minmax(90px, 1fr) 96px' }}
                        >
                          <div style={{ position: 'relative', minWidth: 0 }}>
                            <span style={{ color: 'var(--mc-text)', fontSize: 12, fontWeight: 500 }}>{engName}</span>
                          </div>
                          <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                            <span style={{ color: 'var(--mc-text)', fontSize: 12 }}>{sub.tokens.toLocaleString()}</span>
                          </div>
                          <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                            <span style={{ color: codingKey ? 'var(--mc-muted)' : 'var(--mc-warn)', fontSize: 12, fontWeight: 600 }}>{costText}</span>
                          </div>
                        </div>
                      )
                    })}
                </div>
              </div>
            </div>
          )}

          <div className="mc-panel">
            <div className="mc-panel-body" style={{ gap: 10 }}>
              <div className="mc-field-row">
                <span className="mc-panel-title">用量趋势</span>
                <Segmented
                  size="small"
                  value={trendRange}
                  onChange={(v) => setTrendRange(v as TrendRange)}
                  options={[
                    { value: 'today', label: '今日' },
                    { value: '7d', label: '最近 7 天' },
                    { value: '30d', label: '最近 30 天' },
                  ]}
                />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 12 }}>
                <div className="mc-panel">
                  <div className="mc-panel-body" style={{ gap: 8 }}>
                    <div className="mc-field-row">
                      <span className="mc-panel-title">请求趋势</span>
                      <span style={{ color: 'var(--mc-muted)', fontSize: 10 }}>折线 = 调用 · 红点 = 失败</span>
                    </div>
                    {trendData.length > 0 ? (
                      <RequestsTrendChart data={trendData} color="var(--v3-telemetry)" />
                    ) : (
                      <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--mc-muted)', fontSize: 12 }}>
                        暂无趋势数据
                      </div>
                    )}
                  </div>
                </div>
                <div className="mc-panel">
                  <div className="mc-panel-body" style={{ gap: 8 }}>
                    <div className="mc-field-row">
                      <span className="mc-panel-title">Token 趋势</span>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 10, fontSize: 10, color: 'var(--mc-muted)' }}>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: 'var(--color-primary)', display: 'inline-block' }} />输入</span>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: 'var(--color-success)', display: 'inline-block' }} />输出</span>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 14, height: 0, borderTop: '2px dashed var(--color-destructive)', display: 'inline-block' }} />费用</span>
                      </span>
                    </div>
                    {trendData.length > 0 ? (
                      <TokenTrendChart data={trendData} />
                    ) : (
                      <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--mc-muted)', fontSize: 12 }}>
                        暂无趋势数据
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {(() => {
              const groups = new Map<string, typeof callStats.per_model>()
              callStats.per_model.forEach(s => {
                const list = groups.get(s.engine_id) || []
                list.push(s)
                groups.set(s.engine_id, list)
              })
              return Array.from(groups.entries()).map(([engineId, rows]) => {
                const color = engineColor({ id: engineId })
                const calls = rows.reduce((a, b) => a + b.call_count, 0)
                const succ = rows.reduce((a, b) => a + b.success_count, 0)
                const tokens = rows.reduce((a, b) => a + b.total_tokens, 0)
                const engCost = rows.reduce((a, b) => a + costToCNY(b.estimated_cost ?? 0, b.currency, callStats?.usd_to_cny ?? USD_TO_CNY), 0)
                const rate = calls > 0 ? ((succ / calls) * 100).toFixed(0) : '0'
                const sorted = [...rows].sort((a, b) => {
                  if (statsSort === 'tokens') return b.total_tokens - a.total_tokens
                  if (statsSort === 'cost') return (b.estimated_cost ?? 0) - (a.estimated_cost ?? 0)
                  return b.call_count - a.call_count
                })
                const maxCalls = Math.max(...sorted.map(s => s.call_count), 1)
                return (
                  <div key={engineId} className="mc-panel">
                    <div className="mc-panel-body" style={{ gap: 0 }}>
                      <div className="mc-field-row" style={{ paddingBottom: 12, borderBottom: '1px solid var(--mc-border)' }}>
                        <span style={{ fontSize: 17, color, display: 'inline-flex' }}>{engineIcons[engineId]}</span>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ color: 'var(--mc-text)', fontSize: 13, fontWeight: 650 }}>{engineLabel({ id: engineId })}</div>
                          <div style={{ color: 'var(--mc-muted)', fontSize: 10.5 }}>{rows.length} 个模型</div>
                        </div>
                        <div className="mc-stats-metrics">
                          <div className="mc-stats-metric">
                            <div className="mc-stats-metric-value">{calls}</div>
                            <div className="mc-stats-metric-label">调用</div>
                          </div>
                          <div className="mc-stats-metric">
                            <div className="mc-stats-metric-value">{tokens.toLocaleString()}</div>
                            <div className="mc-stats-metric-label">Token</div>
                          </div>
                          <div className="mc-stats-metric">
                            <div className="mc-stats-metric-value">{fmtCost(engCost, 'CNY')}</div>
                            <div className="mc-stats-metric-label">估算费用</div>
                          </div>
                          <div className="mc-stats-metric">
                            <div className="mc-stats-metric-value" style={{ color: succ === calls ? 'var(--mc-ok)' : 'var(--mc-danger)' }}>{rate}%</div>
                            <div className="mc-stats-metric-label">成功率</div>
                          </div>
                        </div>
                      </div>
                      <div className="mc-table" style={{ border: 0, borderRadius: 0 }}>
                        <div className="mc-table-head">
                          <div>模型</div>
                          <div className="mc-table-cell-num">请求数</div>
                          <div className="mc-table-cell-num">输入 / 输出 Token</div>
                          <div className="mc-table-cell-num">平均延迟</div>
                          <div className="mc-table-cell-num">估算费用</div>
                          <div className="mc-table-cell-num">成功率</div>
                        </div>
                        {sorted.map(s => {
                          const r2 = s.call_count > 0 ? ((s.success_count / s.call_count) * 100).toFixed(0) : '0'
                          const share = Math.round((s.call_count / maxCalls) * 100)
                          const avgSec = s.call_count > 0 ? (s.total_duration_ms / s.call_count / 1000).toFixed(1) : '0.0'
                          // coding_points=GLM 编码套餐积分内调用：费用恒 0。
                          // B 刀：有积分系数时按 (入×in+缓存×cached+出×out)/10000 估算积分，
                          // 无系数（目录未下发）的 coding 行显示「—」；口径说明见模型名下标签。
                          const codingPoints = s.billing_mode === 'coding_points'
                          const rowMeta = codingPoints ? findModelMeta(engines, s.engine_id, s.model) : undefined
                          // 缓存命中 token 走明细的 cache_hit_tokens（未上报时按 0）
                          const pointsText = codingPoints && hasPointsCoef(rowMeta)
                            ? `≈${estimatePoints(s.input_tokens, s.cache_hit_tokens ?? 0, s.output_tokens, rowMeta!).toFixed(1)} 积分`
                            : ''
                          const costText = codingPoints
                            ? (pointsText || '—')
                            : (fmtCost(s.estimated_cost, s.currency) || (isLocalEngine(s.engine_id) ? '免费' : '—'))
                          return (
                            <div key={s.engine_id + '|' + s.model} className="mc-table-row">
                              <div
                                style={{
                                  position: 'absolute', left: 0, top: 0, bottom: 0,
                                  width: `${share}%`, background: `color-mix(in srgb, ${color} 7%, transparent)`,
                                  pointerEvents: 'none',
                                }}
                              />
                              <div style={{ position: 'relative', minWidth: 0 }}>
                                <div style={{ color: 'var(--mc-text)', fontSize: 12, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                  {s.model}
                                  {s.last_error && (
                                    <span style={{ color: 'var(--mc-danger)', marginLeft: 6 }} title={s.last_error}>⚠</span>
                                  )}
                                </div>
                                <div style={{ color: 'var(--mc-muted)', fontSize: 10 }}>
                                  {s.last_called_at ? `最近 ${s.last_called_at.slice(5, 16)}` : '—'}
                                </div>
                                {codingPoints && (
                                  <StatusChip tone="accent" title={billingModeLabel(s.billing_mode)}>
                                    {billingModeLabel(s.billing_mode)}
                                  </StatusChip>
                                )}
                                {s.call_count >= 5 && s.fail_count / s.call_count >= 0.2 && (
                                  <StatusChip tone="danger">高失败率</StatusChip>
                                )}
                              </div>
                              <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                                <span style={{ color: 'var(--mc-text)', fontSize: 12 }}>{s.call_count}</span>
                                {s.fail_count > 0 && (
                                  <span style={{ color: 'var(--mc-danger)', fontSize: 10, display: 'block' }}>失败 {s.fail_count}</span>
                                )}
                              </div>
                              <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                                <span style={{ color: 'var(--mc-text)', fontSize: 11, display: 'block' }}>入 {s.input_tokens.toLocaleString()}</span>
                                <span style={{ color: 'var(--mc-muted)', fontSize: 10 }}>出 {s.output_tokens.toLocaleString()}</span>
                              </div>
                              <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                                <span style={{ color: 'var(--mc-text)', fontSize: 12 }}>{avgSec}s</span>
                              </div>
                              <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                                <span style={{ color: codingPoints || costText === '—' ? 'var(--mc-muted)' : 'var(--mc-warn)', fontSize: 12, fontWeight: 600 }}>{costText}</span>
                              </div>
                              <div className="mc-table-cell-num" style={{ position: 'relative' }}>
                                <span style={{ color: s.fail_count > 0 ? 'var(--mc-danger)' : 'var(--mc-ok)', fontSize: 12, fontWeight: 600 }}>{r2}%</span>
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  </div>
                )
              })
            })()}
          </div>
        </>
      )}
    </div>
  )
}
