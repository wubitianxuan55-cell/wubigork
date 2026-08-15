import { useState } from 'react'
import { Segmented } from 'antd'
import { BarChartOutlined, FundOutlined, LeftOutlined, RightOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { ResourceMonitor } from './ResourceMonitor'
import { EmptyState, KpiTile } from './ui'
import { RequestsTrendChart, TokenTrendChart, type TrendRange } from './charts'
import { fmtCost } from './utils'
import { useModelCenter } from './context'

/**
 * 引擎控制台右侧「统计与资源」检查器（§2.7，v3-panel，可折叠）：
 * - 资源占用：ResourceMonitor（本地资源实时条，走 --color-* 与 --v3-* 令牌色）
 * - 调用统计：紧凑 KPI 摘要 + 请求/Token 趋势图（charts.tsx 保留实现，仅配色令牌化，
 *   主色 --v3-telemetry）。完整明细仍由头部「详细统计」抽屉（StatsSection）承载。
 */
export function InspectorPanel() {
  const [collapsed, setCollapsed] = useState(false)
  return (
    <aside className={`v3-panel mc-inspector${collapsed ? ' is-collapsed' : ''}`} aria-label="统计与资源检查器">
      {collapsed ? (
        <div className="mc-inspector-collapsed">
          <button
            type="button"
            className="mc-inspector-toggle"
            onClick={() => setCollapsed(false)}
            aria-expanded={false}
            aria-label="展开统计与资源检查器"
            title="展开检查器"
          >
            <LeftOutlined />
          </button>
        </div>
      ) : (
        <>
          <div className="v3-panel-head">
            <span className="v3-panel-title"><FundOutlined /> 统计与资源</span>
            <span className="v3-panel-spacer" />
            <button
              type="button"
              className="mc-inspector-toggle"
              onClick={() => setCollapsed(true)}
              aria-expanded
              aria-label="折叠统计与资源检查器"
              title="折叠检查器"
            >
              <RightOutlined />
            </button>
          </div>
          <div className="mc-inspector-body">
            <ResourceMonitor />
            <StatsInspector />
          </div>
        </>
      )}
    </aside>
  )
}

/** 检查器内紧凑调用统计：KPI 摘要 + 趋势图（趋势范围与「详细统计」抽屉共享） */
function StatsInspector() {
  const { callStats, trendData, trendRange, setTrendRange } = useModelCenter()
  const hasStats = !!callStats && callStats.total_calls > 0
  return (
    <section className="mc-inspector-block" aria-label="模型调用统计">
      <div className="mc-inspector-block-head">
        <span className="mc-inspector-block-title"><BarChartOutlined /> 调用统计</span>
        {hasStats && (
          <Segmented
            size="small"
            value={trendRange}
            onChange={(v) => setTrendRange(v as TrendRange)}
            options={[
              { value: 'today', label: '今日' },
              { value: '7d', label: '7天' },
              { value: '30d', label: '30天' },
            ]}
          />
        )}
      </div>

      {!hasStats ? (
        <EmptyState
          compact
          icon={<BarChartOutlined />}
          title="暂无调用记录"
          hint="对话、语音、办公等模块调用模型后自动统计"
        />
      ) : (
        <>
          <div className="mc-inspector-kpis">
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
              hint="按当前汇率折算"
            />
            <KpiTile
              icon={<ThunderboltOutlined />}
              label="成功率"
              value={`${((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)}%`}
              hint={`${callStats.per_model.length} 个模型`}
            />
          </div>
          {trendData.length > 0 && (
            <div className="mc-inspector-charts">
              <div className="mc-inspector-chart-title">请求趋势</div>
              <RequestsTrendChart data={trendData} color="var(--v3-telemetry)" />
              <div className="mc-inspector-chart-title">Token 趋势</div>
              <TokenTrendChart data={trendData} />
            </div>
          )}
        </>
      )}
    </section>
  )
}
