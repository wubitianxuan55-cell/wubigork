// StudioUsagePanel.tsx — 画室消耗折叠面板（绘梦阶段一刀 E 轻量收尾，规格
// 进度计划/gaea-studio-usage-20260917.md）：本月创作记录聚合（张数×单价），
// 只读台账零计费动作。默认收起；标题带总数与合计；展开=按模型行+免费/未定价
// 注记。文案「画室消耗/创作记录」，不用积分话术（规格原文）。
import React, { useEffect, useState } from 'react'
import { Collapse, Spin, Tag, Typography } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { StudioUsageView } from '../../gaea/lib/bridge/image'

const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }

export default function StudioUsagePanel() {
  const [usage, setUsage] = useState<StudioUsageView | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setLoading(true)
    app.ImageHubMonthlyUsage('play')
      .then(setUsage)
      .catch(() => setUsage(null)) // 消耗面板失败静默收起（辅助视图不打扰创作）
      .finally(() => setLoading(false))
  }, [])

  const total = usage?.total ?? 0
  const estimated = usage?.estimated ?? ''

  return (
    <Collapse
      size="small"
      data-testid="studio-usage-panel"
      items={[{
        key: 'usage',
        label: (
          <span data-testid="studio-usage-title" style={{ fontSize: 12.5 }}>
            本月画室消耗{!loading && usage ? ` · ${total} 张 · ${estimated}` : ''}
          </span>
        ),
        children: loading ? (
          <div style={{ padding: 12, textAlign: 'center' }}><Spin size="small" /></div>
        ) : !usage || total === 0 ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>{usage?.estimated || '本月还没有创作记录'}</Typography.Text>
        ) : (
          <div data-testid="studio-usage-body" style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {(usage.byModel ?? []).map((row, i) => (
              <div key={`${row.model}-${row.backend}-${i}`} style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
                <span style={{ fontSize: 12 }}>{row.model || '（未知模型）'}</span>
                {row.backend && <Tag style={{ marginRight: 0 }}>{row.backend}</Tag>}
                <span style={{ flex: 1 }} />
                <span style={{ ...softTextStyle, fontVariantNumeric: 'tabular-nums' }}>{row.count} 张</span>
                <span style={{ ...softTextStyle, fontVariantNumeric: 'tabular-nums', minWidth: 72, textAlign: 'right' }}>
                  {row.estCost || (row.unitCost === '未定价' ? '未定价' : '—')}
                </span>
              </div>
            ))}
            <Typography.Text type="secondary" style={{ fontSize: 11, marginTop: 4 }}>
              口径：台账登记张数 × 模型目录单价（本地免费记 0；未定价如实标注不估算）。{usage.unpriced ? ` 另有 ${usage.unpriced} 张未定价。` : ''}
            </Typography.Text>
          </div>
        ),
      }]}
    />
  )
}
