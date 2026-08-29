// ConsistencyPanel.tsx — 一致性检查面板（v4.3f）
// 展示 CheckConsistency 三类规则告警（角色属性/角色状态/时间线），带严重度/描述，
// 「重新检查」按钮 + 全部通过时的成功空态。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, Spin, Tag } from 'antd'
import { CheckCircleOutlined, ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import type { ConsistencyCheckIssue, ConsistencyCheckReport, ConsistencySeverity } from '../../types'

const SEVERITY_META: Record<ConsistencySeverity, { label: string; color: string }> = {
  error: { label: '错误', color: 'red' },
  warning: { label: '警告', color: 'gold' },
  info: { label: '提示', color: 'blue' },
}

const CATEGORY_LABELS: Record<string, string> = {
  attribute: '角色属性',
  timeline: '时间线',
  status: '角色状态',
  relationship: '关系',
}

function normalizeReport(raw: unknown): ConsistencyCheckReport | null {
  const r = raw as ConsistencyCheckReport | null
  if (!r || !Array.isArray(r.issues)) return null
  return r
}

interface ConsistencyPanelProps {
  /** 未打开项目时仅展示空态引导，不触发加载 */
  disabled?: boolean
}

const ConsistencyPanel: React.FC<ConsistencyPanelProps> = ({ disabled }) => {
  const [report, setReport] = useState<ConsistencyCheckReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadToken = useRef(0)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setReport(null)
      setLoading(false)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await App.CheckConsistency()
      if (token !== loadToken.current) return
      setReport(normalizeReport(res))
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setError(err instanceof Error ? err.message : '一致性检查失败')
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  const issues: ConsistencyCheckIssue[] = report?.issues ?? []

  return (
    <div className="novel-panel fs-panel" style={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 4 }}>
        <span className="novel-panel-title"><SafetyCertificateOutlined />一致性检查</span>
        <div style={{ flex: 1 }} />
        {report && (
          <span className="novel-setting-meta" data-testid="consistency-summary">
            {report.summary || `共 ${report.total_issues} 个问题`}
          </span>
        )}
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading} disabled={disabled}>
          重新检查
        </Button>
      </div>
      <div className="novel-setting-body" style={{ padding: 8 }}>
        {disabled ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先在「书架」打开一部小说项目" style={{ margin: 'auto' }} />
        ) : loading ? (
          <div style={{ margin: 'auto' }}><Spin size="small" /></div>
        ) : error ? (
          <Alert
            type="error" showIcon style={{ width: '100%' }}
            message="一致性检查失败"
            description={error}
            action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void load()}>重试</Button>}
          />
        ) : issues.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <span style={{ color: 'var(--color-success, #34d399)' }}>
                <CheckCircleOutlined /> 全部通过，未发现一致性问题
              </span>
            }
            style={{ margin: 'auto' }}
          />
        ) : (
          <div className="fs-list" style={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
            {issues.map((iss, idx) => (
              <div key={`${iss.category}-${iss.entity_name}-${idx}`} className="fs-item">
                <div className="fs-item-head">
                  <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={SEVERITY_META[iss.severity]?.color ?? 'default'}>
                    {SEVERITY_META[iss.severity]?.label ?? iss.severity}
                  </Tag>
                  <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[iss.category] || iss.category}</Tag>
                  <span className="fs-item-desc">
                    {iss.entity_name ? `${iss.entity_name}：` : ''}{iss.description}
                  </span>
                </div>
                {iss.location && (
                  <div className="fs-item-foot">
                    <span className="novel-setting-meta">{iss.location}</span>
                  </div>
                )}
                {iss.suggestion && (
                  <div className="fs-item-foot fs-item-suggestion">
                    <span>建议：{iss.suggestion}</span>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default ConsistencyPanel
