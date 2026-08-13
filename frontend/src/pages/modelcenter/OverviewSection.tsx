import { Button } from 'antd'
import {
  AlertOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DashboardOutlined,
  SettingOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { EmptyState, KpiTile, ModelCard, SectionHead, StatusChip } from './ui'
import { engineLabel } from './utils'
import { useModelCenter } from './context'

export function OverviewSection() {
  const { engines, engineStatuses, callStats, setCategory, activeEngine } = useModelCenter()
  const enabled = engines.filter(e => e.enabled)
  const connected = enabled.filter(e => engineStatuses[e.id]?.connected).length
  const failedCalls = callStats?.fail_calls ?? 0
  const successRate = callStats && callStats.total_calls > 0
    ? ((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)
    : null

  const issues = (callStats?.per_model || [])
    .filter(s => s.fail_count > 0 || s.last_error)
    .map(s => ({ ...s, failRate: s.call_count > 0 ? s.fail_count / s.call_count : 0 }))
    .sort((a, b) => b.failRate - a.failRate || b.fail_count - a.fail_count)
    .slice(0, 6)

  return (
    <section className="mc-section">
      <SectionHead
        icon={<DashboardOutlined />}
        title="模型中心总览"
        desc="一眼看清引擎健康、调用健康与需要关注的模型"
      />

      <div className="mc-overview-grid">
        <KpiTile
          icon={<CheckCircleOutlined />}
          label="引擎连接"
          value={`${connected}/${enabled.length}`}
          hint="已启用引擎中连接成功"
        />
        <KpiTile
          icon={<CloseCircleOutlined />}
          label="累计失败调用"
          value={failedCalls}
          hint={successRate ? `成功率 ${successRate}%` : '暂无调用数据'}
        />
        <KpiTile
          icon={<AlertOutlined />}
          label="高失败率模型"
          value={issues.filter(s => s.failRate >= 0.2).length}
          hint="失败率 ≥ 20%"
        />
      </div>

      <div className="mc-grid">
        {enabled.map(engine => {
          const status = engineStatuses[engine.id]
          const ok = status?.connected
          return (
            <ModelCard
              key={engine.id}
              name={engineLabel(engine)}
              engineId={engine.id}
              active={activeEngine === engine.id}
              kindChip={<StatusChip>{engine.id}</StatusChip>}
              status={{
                tone: ok ? 'ok' : 'danger',
                text: ok ? `连接正常 · ${status!.model_count} 个模型` : status?.error || '尚未测试',
              }}
              action={(
                <Button size="small" icon={<SettingOutlined />} onClick={() => setCategory('engine')}>
                  管理
                </Button>
              )}
            />
          )
        })}
      </div>

      <div className="mc-panel">
        <div className="mc-panel-body">
          <div className="mc-panel-title"><AlertOutlined /> 需要关注</div>
          {issues.length === 0 ? (
            <EmptyState
              compact
              icon={<CheckCircleOutlined />}
              title="当前没有失败或异常模型"
              hint="模型被调用后，这里会汇总失败次数、失败率和最近错误"
            />
          ) : (
            <div className="mc-issue-list">
              {issues.map(s => (
                <div key={`${s.engine_id}|${s.model}`} className="mc-issue">
                  <span style={{ color: 'var(--mc-danger)' }}><AlertOutlined /></span>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <div className="mc-issue-name">{s.model}</div>
                    <div className="mc-issue-sub">
                      {engineLabel({ id: s.engine_id })} · 失败 {s.fail_count}/{s.call_count}
                    </div>
                  </div>
                  <StatusChip tone="danger">{(s.failRate * 100).toFixed(0)}%</StatusChip>
                  {s.last_error && <span className="mc-issue-err" title={s.last_error}>{s.last_error}</span>}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="mc-overview-actions">
        <Button icon={<ThunderboltOutlined />} onClick={() => setCategory('llm')}>管理语言模型</Button>
        <Button icon={<DashboardOutlined />} onClick={() => setCategory('specialty')}>查看专业模型</Button>
        <Button icon={<SettingOutlined />} onClick={() => setCategory('engine')}>测试引擎连接</Button>
      </div>
    </section>
  )
}
