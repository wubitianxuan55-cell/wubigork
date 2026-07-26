import React from 'react'
import { Typography, Tooltip } from 'antd'

/**
 * TokenBudget — 上下文 Token 预算可视化
 *
 * 横向堆叠条形图展示各分区的 token 用量
 *
 * Props:
 *   capacity — 模型总容量
 *   used — 已使用 token
 *   sections — 各分区用量（name/used/color）
 */
interface BudgetSection {
  name: string
  used: number
  limit: number
  color: string
}

interface TokenBudgetProps {
  capacity: number
  used: number
  remaining: number
  usagePercent: number
  sections: BudgetSection[]
}

const TokenBudget: React.FC<TokenBudgetProps> = ({
  capacity,
  used,
  remaining,
  usagePercent,
  sections,
}) => {
  const formatTokens = (n: number) => {
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
    return String(n)
  }

  return (
    <div
      style={{
        background: 'var(--bg-glass)',
        borderRadius: 'var(--radius-lg)',
        border: '1px solid var(--border-subtle)',
        padding: 10,
        fontSize: 11,
      }}
    >
      {/* 头部 */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 6,
        }}
      >
        <Typography.Text strong style={{ fontSize: 12 }}>
          📊 上下文预算
        </Typography.Text>
        <Typography.Text
          type={usagePercent > 80 ? 'danger' : usagePercent > 50 ? 'warning' : 'secondary'}
          style={{ fontSize: 11 }}
        >
          {formatTokens(used)} / {formatTokens(capacity)}
          {usagePercent > 80 && ' ⚠️'}
        </Typography.Text>
      </div>

      {/* 堆叠条形图 */}
      <div
        style={{
          display: 'flex',
          height: 8,
          borderRadius: 4,
          overflow: 'hidden',
          background: 'var(--bg-deep)',
          marginBottom: 6,
        }}
      >
        {sections
          .filter(s => s.used > 0)
          .map((sec, i) => {
            const pct = (sec.used / capacity) * 100
            return (
              <Tooltip
                key={i}
                title={`${sec.name}: ${formatTokens(sec.used)} tokens`}
              >
                <div
                  style={{
                    width: `${pct}%`,
                    minWidth: 2,
                    background: sec.color,
                    transition: 'width 300ms ease',
                  }}
                />
              </Tooltip>
            )
          })}
        {/* 剩余空间 */}
        {remaining > 0 && (
          <Tooltip title={`剩余: ${formatTokens(remaining)} tokens`}>
            <div
              style={{
                flex: 1,
                background: 'rgba(255,255,255,0.03)',
              }}
            />
          </Tooltip>
        )}
      </div>

      {/* 图例 */}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        {sections
          .filter(s => s.used > 0)
          .map((sec, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: 3,
                  background: sec.color,
                  display: 'inline-block',
                }}
              />
              <Typography.Text type="secondary" style={{ fontSize: 10 }}>
                {sec.name} {formatTokens(sec.used)}
              </Typography.Text>
            </div>
          ))}
      </div>
    </div>
  )
}

export default TokenBudget
export type { TokenBudgetProps, BudgetSection }
