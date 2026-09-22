import { useState } from 'react'
import { Button, Segmented, Space, Tooltip } from 'antd'
import { ArrowRightOutlined, BarChartOutlined, FundOutlined, LeftOutlined, ReloadOutlined, RightOutlined } from '@ant-design/icons'
import { ResourceMonitor } from './ResourceMonitor'
import { EmptyState } from './ui'
import { RequestsSpark, TokenMiniBars, type TrendRange } from './charts'
import { fmtCompact, fmtCost } from './utils'
import { useModelCenterActions, useModelCenterState } from './context'

/**
 * 引擎控制台右侧「统计与资源」检查器（§2.7，v3-panel，可折叠）：
 * - 资源占用：ResourceMonitor（本地资源实时条，走 --color-* 与 --v3-* 令牌色）
 * - 调用统计：遥测读出式重设计——三格主指标 + Token 读数行 + 窄柱专用迷你图
 *   （RequestsSpark/TokenMiniBars 按 ~276px 视口设计，字标 1:1 渲染可读；
 *   旧全尺寸图压进窄柱会把 10px 轴字缩到 ~4px）。完整明细仍由
 *   头部/本块「详细统计」入口（StatsSection 抽屉）承载。
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

/** 检查器内紧凑调用统计：主指标读出 + 窄柱迷你趋势（趋势范围与「详细统计」抽屉共享） */
function StatsInspector() {
  const { callStats, trendData, trendRange, loadError } = useModelCenterState()
  const { setTrendRange, loadCallStats, openStatsDrawer } = useModelCenterActions()
  const hasStats = !!callStats && callStats.total_calls > 0
  const rate = hasStats ? (callStats.success_calls / callStats.total_calls) * 100 : 0
  const rateColor = !hasStats ? 'var(--mc-text)' : rate >= 100 ? 'var(--mc-ok)' : rate >= 90 ? 'var(--mc-text)' : 'var(--mc-danger)'
  const peak = trendData.reduce((a, b) => Math.max(a, b.calls), 0)
  return (
    <section className="mc-inspector-block" aria-label="模型调用统计">
      <div className="mc-inspector-block-head">
        <span className="mc-inspector-block-title"><BarChartOutlined /> 调用统计</span>
        <Space size={4}>
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
          <Tooltip title="刷新统计">
            <Button
              size="small"
              type="text"
              icon={<ReloadOutlined />}
              aria-label="刷新统计"
              onClick={() => void loadCallStats()}
            />
          </Tooltip>
        </Space>
      </div>

      {loadError ? (
        <div className="mc-inspector-error">
          <span>统计加载失败：{loadError}</span>
          <Button size="small" type="primary" ghost onClick={() => void loadCallStats()}>重试</Button>
        </div>
      ) : !hasStats ? (
        <EmptyState
          compact
          icon={<BarChartOutlined />}
          title="暂无调用记录"
          hint="对话、语音、办公等模块调用模型后自动统计"
        />
      ) : (
        <>
          {/* 主指标读出：调用 / 成功率 / 费用（窄柱三列，值大标签小，tabular 对齐） */}
          <div className="mc-stat-hero">
            <div className="mc-stat-cell">
              <div className="mc-stat-value">{callStats.total_calls.toLocaleString()}</div>
              <div className="mc-stat-label">总调用</div>
              <div className="mc-stat-sub">
                成功 {callStats.success_calls}
                {callStats.fail_calls > 0 && <span className="is-bad"> · 失败 {callStats.fail_calls}</span>}
              </div>
            </div>
            <div className="mc-stat-cell">
              <div className="mc-stat-value" style={{ color: rateColor }}>{rate.toFixed(1)}%</div>
              <div className="mc-stat-label">成功率</div>
              <div className="mc-stat-sub">{callStats.per_model.length} 个模型</div>
            </div>
            <div className="mc-stat-cell">
              <div className="mc-stat-value" style={{ color: 'var(--mc-warn)' }}>{fmtCost(callStats.total_cost, 'CNY')}</div>
              <div className="mc-stat-label">估算费用</div>
              <div className="mc-stat-sub">按当前汇率折算</div>
            </div>
          </div>

          {/* Token 读数行：总量 + 入/出拆分（紧凑单行，k/M 缩写） */}
          <div className="mc-stat-strip">
            <span className="mc-stat-strip-main">Token {fmtCompact(callStats.total_tokens)}</span>
            <span className="mc-stat-strip-sep" />
            <span>入 {fmtCompact(callStats.input_tokens)}</span>
            <span>出 {fmtCompact(callStats.output_tokens)}</span>
          </div>

          {trendData.length > 0 && (
            <div className="mc-mini-chart">
              <div className="mc-mini-chart-head">
                <span className="mc-mini-chart-title">请求趋势</span>
                <span className="mc-mini-chart-meta">峰值 {peak.toLocaleString()}</span>
              </div>
              <RequestsSpark data={trendData} color="var(--v3-telemetry)" />
            </div>
          )}
          {trendData.length > 0 && (
            <div className="mc-mini-chart">
              <div className="mc-mini-chart-head">
                <span className="mc-mini-chart-title">Token 分布</span>
                <span className="mc-mini-chart-meta">
                  <i className="mc-legend-swatch is-in" aria-hidden="true" />入
                  <i className="mc-legend-swatch is-out" aria-hidden="true" />出
                </span>
              </div>
              <TokenMiniBars data={trendData} />
            </div>
          )}

          <button type="button" className="mc-stat-more" onClick={openStatsDrawer}>
            查看详细统计（按引擎 / 模型明细）
            <ArrowRightOutlined />
          </button>
        </>
      )}
    </section>
  )
}
