import { Button, Card, Tag, Typography } from 'antd'
import { AlertOutlined, CheckCircleOutlined, CloseCircleOutlined, DashboardOutlined, SettingOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineIcons, engineLabel } from './utils'
import { useModelCenter } from './context'

export function OverviewSection() {
  const { engines, engineStatuses, callStats, setCategory, activeEngine } = useModelCenter()
  const enabled = engines.filter(e => e.enabled)
  const connected = enabled.filter(e => engineStatuses[e.id]?.connected).length
  const failedCalls = callStats?.fail_calls ?? 0

  const issues = (callStats?.per_model || [])
    .filter(s => s.fail_count > 0 || s.last_error)
    .map(s => ({ ...s, failRate: s.call_count > 0 ? s.fail_count / s.call_count : 0 }))
    .sort((a, b) => b.failRate - a.failRate || b.fail_count - a.fail_count)
    .slice(0, 6)

  return (
    <section className="mc-section">
      <div className="mc-section-head">
        <div>
          <div className="mc-section-title"><DashboardOutlined /> 模型中心总览</div>
          <div className="mc-section-desc">一眼看清引擎健康、调用健康与需要关注的模型</div>
        </div>
      </div>

      <div className="mc-overview-grid" style={{ marginBottom: 16 }}>
        <div className="mc-kpi">
          <div className="mc-kpi-label"><CheckCircleOutlined /> 引擎连接</div>
          <div className="mc-kpi-value">{connected}/{enabled.length}</div>
          <div className="mc-kpi-hint">已启用引擎中连接成功</div>
        </div>
        <div className="mc-kpi">
          <div className="mc-kpi-label"><CloseCircleOutlined /> 累计失败调用</div>
          <div className="mc-kpi-value">{failedCalls}</div>
          <div className="mc-kpi-hint">{callStats?.total_calls ? `成功率 ${((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)}%` : '暂无调用数据'}</div>
        </div>
        <div className="mc-kpi">
          <div className="mc-kpi-label"><AlertOutlined /> 高失败率模型</div>
          <div className="mc-kpi-value">{issues.filter(s => s.failRate >= 0.2).length}</div>
          <div className="mc-kpi-hint">失败率 ≥ 20%</div>
        </div>
      </div>

      <div className="mc-grid" style={{ marginBottom: 16 }}>
        {enabled.map(engine => {
          const status = engineStatuses[engine.id]
          const color = engineColor(engine)
          const ok = status?.connected
          return (
            <Card key={engine.id} size="small" className={`mc-model-card${activeEngine === engine.id ? ' is-active' : ''}`} style={{ borderColor: ok ? `color-mix(in srgb, ${color} 28%, transparent)` : undefined }}>
              <div className="mc-model-name">
                <span style={{ marginRight: 8, color }}>{engineIcons[engine.id]}</span>
                {engineLabel(engine)}
              </div>
              <div className="mc-model-meta">
                <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engine.id}</Tag>
                <Tag color={ok ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>
                  {ok ? `${status!.model_count} 个模型` : '未连接'}
                </Tag>
              </div>
              <div className="mc-model-foot">
                <span className="mc-status">
                  <i className={`mc-status-dot ${ok ? 'is-running' : ''}`} />
                  {ok ? '连接正常' : status?.error || '尚未测试'}
                </span>
                <Button size="small" icon={<SettingOutlined />} onClick={() => setCategory('engine')} style={{ fontSize: 11, marginLeft: 'auto' }}>管理</Button>
              </div>
            </Card>
          )
        })}
      </div>

      <div className="mc-panel">
        <div className="mc-panel-body">
          <div className="mc-section-title" style={{ marginBottom: 12 }}><AlertOutlined /> 需要关注</div>
          {issues.length === 0 ? (
            <div className="mc-empty" style={{ minHeight: 120 }}>
              <CheckCircleOutlined className="mc-empty-icon" />
              <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>当前没有失败或异常模型</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 6 }}>模型被调用后，这里会汇总失败次数、失败率和最近错误</Typography.Text>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {issues.map(s => (
                <div key={`${s.engine_id}|${s.model}`} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 12px', borderRadius: 10, background: 'rgba(15,23,42,0.42)', border: '1px solid var(--border-subtle)' }}>
                  <span style={{ color: '#fb7185' }}><AlertOutlined /></span>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.model}</Typography.Text>
                    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>{engineLabel({ id: s.engine_id })} · 失败 {s.fail_count}/{s.call_count}</Typography.Text>
                  </div>
                  <Tag color="red" style={{ fontSize: 10, margin: 0 }}>{(s.failRate * 100).toFixed(0)}%</Tag>
                  {s.last_error && <Typography.Text style={{ color: '#fb7185', fontSize: 10, maxWidth: 240, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={s.last_error}>{s.last_error}</Typography.Text>}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div style={{ marginTop: 14, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <Button icon={<ThunderboltOutlined />} onClick={() => setCategory('llm')}>管理语言模型</Button>
        <Button icon={<DashboardOutlined />} onClick={() => setCategory('specialty')}>查看专业模型</Button>
        <Button icon={<SettingOutlined />} onClick={() => setCategory('engine')}>测试引擎连接</Button>
      </div>
    </section>
  )
}
