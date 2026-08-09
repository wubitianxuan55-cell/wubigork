import { Button, Card, Popconfirm, Segmented, Space, Tag, Typography } from 'antd'
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { costToCNY, engineColor, engineIcons, engineLabel, fmtCost, isLocalEngine, USD_TO_CNY } from './utils'
import { RequestsTrendChart, TokenTrendChart, type StatsSort, type TrendRange } from './charts'
import { useModelCenter } from './context'

export function StatsSection() {
  const { callStats, statsSort, setStatsSort, trendRange, setTrendRange, loadCallStats, handleResetCallStats, trendData } = useModelCenter()
  return (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14, flexWrap: 'wrap', gap: 8 }}>
                <Space size={8}>
                  <ThunderboltOutlined style={{ color: '#fbbf24' }} />
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>模型调用统计</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                    {callStats?.since ? `统计自 ${callStats.since}` : '按引擎 / 模型维度统计调用情况与估算费用'}
                  </Typography.Text>
                </Space>
                <Space size={4}>
                  <Segmented
                    size="small"
                    value={statsSort}
                    onChange={(v) => setStatsSort(v as StatsSort)}
                    options={[
                      { value: 'calls', label: '调用最多' },
                      { value: 'tokens', label: 'Token 最多' },
                      { value: 'cost', label: '费用最高' },
                    ]}
                    style={{ fontSize: 11 }}
                  />
                  <Button size="small" icon={<ReloadOutlined />} onClick={loadCallStats} style={{ fontSize: 11 }}>刷新</Button>
                  <Popconfirm
                    title="确定清空全部模型调用统计？"
                    description="此操作不可恢复，将清空次数、Token 与费用记录。"
                    okText="清空"
                    cancelText="取消"
                    onConfirm={handleResetCallStats}
                  >
                    <Button size="small" danger style={{ fontSize: 11 }}>清空统计</Button>
                  </Popconfirm>
                </Space>
              </div>

              {!callStats || callStats.total_calls === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 48 }}>
                  <ThunderboltOutlined style={{ fontSize: 30, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text'), fontSize: 14, display: 'block' }}>暂无调用记录</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginTop: 6 }}>
                    对话、语音、办公等模块调用模型后，这里会自动统计次数、Token、耗时与估算费用
                  </Typography.Text>
                </Card>
              ) : (
                <>
                  {/* 全局指标 */}
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 10, marginBottom: 16 }}>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>总调用</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{callStats.total_calls}</div>
                      <div style={{ color: '#34d399', fontSize: 11, marginTop: 2 }}>成功 {callStats.success_calls} · 失败 {callStats.fail_calls}</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>Token 用量</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{callStats.total_tokens.toLocaleString()}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>入 {callStats.input_tokens.toLocaleString()} / 出 {callStats.output_tokens.toLocaleString()}</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>估算费用</div>
                      <div style={{ color: '#fbbf24', fontSize: 22, fontWeight: 700 }}>{fmtCost(callStats.total_cost, 'CNY')}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>美元按 1:{callStats?.usd_to_cny ?? USD_TO_CNY} 折算</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>成功率</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>
                        {((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)}%
                      </div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>{callStats.per_model.length} 个模型</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>平均耗时</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{(callStats.avg_duration_ms / 1000).toFixed(1)}s</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>累计 {(callStats.total_duration_ms / 1000).toFixed(1)}s</div>
                    </div>
                  </div>

                  {/* 用量趋势（CCSwitch 风格：请求折线 + Token 堆叠 + 费用线） */}
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10, flexWrap: 'wrap' }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>用量趋势</Typography.Text>
                      <Segmented
                        size="small"
                        value={trendRange}
                        onChange={(v) => setTrendRange(v as TrendRange)}
                        options={[
                          { value: 'today', label: '今日' },
                          { value: '7d', label: '最近 7 天' },
                          { value: '30d', label: '最近 30 天' },
                        ]}
                        style={{ fontSize: 11 }}
                      />
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 12 }}>
                      <Card size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>请求趋势</Typography.Text>
                          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>折线 = 调用 · 红点 = 失败</Typography.Text>
                        </div>
                        {trendData.length > 0 ? (
                          <RequestsTrendChart data={trendData} color="#60a5fa" />
                        ) : (
                          <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), fontSize: 12 }}>暂无趋势数据</div>
                        )}
                      </Card>
                      <Card size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>Token 趋势</Typography.Text>
                          <span style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 10, color: C('color-text-secondary') }}>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: '#60a5fa', display: 'inline-block' }} />输入</span>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: '#34d399', display: 'inline-block' }} />输出</span>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 14, height: 0, borderTop: '2px dashed #f87171', display: 'inline-block' }} />费用</span>
                          </span>
                        </div>
                        {trendData.length > 0 ? (
                          <TokenTrendChart data={trendData} />
                        ) : (
                          <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), fontSize: 12 }}>暂无趋势数据</div>
                        )}
                      </Card>
                    </div>
                  </div>

                  {/* 按引擎分组的模型用量（CCSwitch 风格：提供商卡片 + 模型行） */}
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
                          <Card key={engineId} size="small" style={{ background: 'var(--bg-glass)', border: `1px solid ${color}28`, borderRadius: 12, padding: 0 }}>
                            {/* 引擎汇总 */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderBottom: '1px solid var(--border-subtle)', background: `color-mix(in srgb, ${color} 8%, transparent)` }}>
                              <span style={{ fontSize: 18, color }}>{engineIcons[engineId]}</span>
                              <div style={{ minWidth: 0 }}>
                                <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block' }}>{engineLabel({ id: engineId })}</Typography.Text>
                                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10.5 }}>{rows.length} 个模型</Typography.Text>
                              </div>
                              <div style={{ marginLeft: 'auto', display: 'flex', gap: 18, alignItems: 'center' }}>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: C('color-text'), fontSize: 15, fontWeight: 600 }}>{calls}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>调用</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: C('color-text'), fontSize: 15, fontWeight: 600 }}>{tokens.toLocaleString()}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>Token</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: '#fbbf24', fontSize: 15, fontWeight: 600 }}>{fmtCost(engCost, 'CNY')}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>估算费用</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: succ === calls ? '#34d399' : '#fb7185', fontSize: 15, fontWeight: 600 }}>{rate}%</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>成功率</div>
                                </div>
                              </div>
                            </div>
                            {/* 模型明细（对齐 CCSwitch 模型统计字段：模型 / 请求数 / 输入输出 Token / 平均延迟 / 估算费用 / 成功率） */}
                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                              <div style={{ display: 'grid', gridTemplateColumns: 'minmax(140px, 1.6fr) 56px 128px 64px 92px 52px', gap: 8, padding: '6px 14px', borderBottom: '1px solid var(--border-subtle)', color: C('color-text-secondary'), fontSize: 10 }}>
                                <div>模型</div>
                                <div style={{ textAlign: 'right' }}>请求数</div>
                                <div style={{ textAlign: 'right' }}>输入 / 输出 Token</div>
                                <div style={{ textAlign: 'right' }}>平均延迟</div>
                                <div style={{ textAlign: 'right' }}>估算费用</div>
                                <div style={{ textAlign: 'right' }}>成功率</div>
                              </div>
                              {sorted.map(s => {
                                const r2 = s.call_count > 0 ? ((s.success_count / s.call_count) * 100).toFixed(0) : '0'
                                const share = Math.round((s.call_count / maxCalls) * 100)
                                const avgSec = s.call_count > 0 ? (s.total_duration_ms / s.call_count / 1000).toFixed(1) : '0.0'
                                const costText = fmtCost(s.estimated_cost, s.currency) || (isLocalEngine(s.engine_id) ? '免费' : '—')
                                return (
                                  <div key={s.engine_id + '|' + s.model} style={{ position: 'relative', display: 'grid', gridTemplateColumns: 'minmax(140px, 1.6fr) 56px 128px 64px 92px 52px', gap: 8, alignItems: 'center', padding: '8px 14px', borderBottom: '1px solid rgba(255,255,255,0.04)', overflow: 'hidden' }}>
                                    {/* 调用占比背景条 */}
                                    <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: `${share}%`, background: `color-mix(in srgb, ${color} 7%, transparent)`, pointerEvents: 'none' }} />
                                    <div style={{ position: 'relative', minWidth: 0 }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                        {s.model}
                                        {s.last_error && <span style={{ color: '#fb7185', marginLeft: 6 }} title={s.last_error}>⚠</span>}
                                      </Typography.Text>
                                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
                                        {s.last_called_at ? `最近 ${s.last_called_at.slice(5, 16)}` : '—'}
                                      </Typography.Text>
                                      {s.call_count >= 5 && s.fail_count / s.call_count >= 0.2 && (
                                        <Tag color="red" style={{ fontSize: 9, marginTop: 2 }}>高失败率</Tag>
                                      )}
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12 }}>{s.call_count}</Typography.Text>
                                      {s.fail_count > 0 && (
                                        <Typography.Text style={{ color: '#fb7185', fontSize: 10, display: 'block' }}>失败 {s.fail_count}</Typography.Text>
                                      )}
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 11, display: 'block' }}>入 {s.input_tokens.toLocaleString()}</Typography.Text>
                                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>出 {s.output_tokens.toLocaleString()}</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12 }}>{avgSec}s</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: costText === '—' ? C('color-text-secondary') : '#fbbf24', fontSize: 12, fontWeight: 600 }}>{costText}</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: s.fail_count > 0 ? '#fb7185' : '#34d399', fontSize: 12, fontWeight: 600 }}>{r2}%</Typography.Text>
                                    </div>
                                  </div>
                                )
                              })}
                            </div>
                          </Card>
                        )
                      })
                    })()}
                  </div>
                </>
              )}
            </div>
  )
}
