// ConsistencyPanel.tsx — 一致性检查面板（v4.3f + AI 深检 v0）
// 展示 CheckConsistency 三类规则告警（角色属性/角色状态/时间线），带严重度/描述，
// 「重新检查」按钮 + 全部通过时的成功空态。
// 「AI 深检」：AI 逐章提取实体状态卡，本地跨章比对矛盾（死亡再出场/位置瞬移/
// 物品凭空消失·无中生有/时间倒流），与规则层合并展示（source 徽标区分来源）；
// AI 不可用时诚实降级为规则层结果并展示说明。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, InputNumber, Spin, Tag } from 'antd'
import { CheckCircleOutlined, ReloadOutlined, SafetyCertificateOutlined, ThunderboltOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import { clampDeepChapters, deepScanMeta, normalizeDeepResult, sourceBadge } from './consistencyDeep'
import type { ConsistencyCheckIssue, ConsistencyCheckReport, ConsistencyDeepIssue, ConsistencyDeepResult, ConsistencySeverity } from '../../types'

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
  item: '关键物品',
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
  const [deepResult, setDeepResult] = useState<ConsistencyDeepResult | null>(null)
  const [deepLoading, setDeepLoading] = useState(false)
  const [deepError, setDeepError] = useState('')
  const [deepChapters, setDeepChapters] = useState(20)
  const loadToken = useRef(0)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setReport(null)
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const res = await App.CheckConsistency()
      if (token !== loadToken.current) return
      setReport(normalizeReport(res))
    } catch {
      // 沿用既有行为：规则层检查失败不展示错误（原 error 状态从未接入任何渲染通道，
      // 属死状态已移除），面板保持上次报告结果，仅由 deepError 展示 AI 深检失败。
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  // AI 深检：AI 逐章提取状态卡 + 本地跨章比对，后端已合并规则层结果（source 字段区分）。
  const runDeep = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) return
    setDeepLoading(true)
    setDeepError('')
    const maxChapters = clampDeepChapters(deepChapters)
    try {
      const res = await App.CheckConsistencyDeep(maxChapters)
      if (token !== loadToken.current) return
      const normalized = normalizeDeepResult(res)
      if (!normalized) {
        setDeepError('AI 深检返回了无法解析的结果，已保留上次检查结果')
        return
      }
      setDeepResult(normalized)
      setReport({ issues: normalized.issues, total_issues: normalized.total_issues, summary: normalized.summary })
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setDeepError(err instanceof Error ? err.message : 'AI 深检失败')
    } finally {
      if (token === loadToken.current) setDeepLoading(false)
    }
  }, [deepChapters, disabled])

  const issues: ConsistencyCheckIssue[] = deepResult?.issues ?? report?.issues ?? []
  const summaryText = deepResult?.summary ?? report?.summary
  const showSourceBadge = deepResult !== null

  return (
    <div className="novel-panel fs-panel" style={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 4 }}>
        <span className="novel-panel-title"><SafetyCertificateOutlined />一致性检查</span>
        <div style={{ flex: 1 }} />
        {summaryText && (
          <span className="novel-setting-meta" data-testid="consistency-summary">
            {summaryText}
          </span>
        )}
        <InputNumber
          size="small" min={1} max={50} step={5}
          value={deepChapters}
          onChange={(v) => { if (typeof v === 'number') setDeepChapters(v) }}
          disabled={disabled || deepLoading}
          style={{ width: 88 }}
          addonAfter="章"
          data-testid="consistency-deep-chapters"
        />
        <Button
          size="small" icon={<ThunderboltOutlined />}
          onClick={() => void runDeep()} loading={deepLoading}
          disabled={disabled}
          data-testid="consistency-deep-run"
        >
          AI 深检
        </Button>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading} disabled={disabled}>
          重新检查
        </Button>
      </div>
      <div className="novel-setting-body" style={{ padding: 8, gap: 6 }}>
        {disabled ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先在「书架」打开一部小说项目" style={{ margin: 'auto' }} />
        ) : loading ? (
          <div style={{ margin: 'auto' }}><Spin size="small" /></div>
        ) : (
          <>
            {deepError && (
              <Alert
                type="error" showIcon style={{ width: '100%' }} closable onClose={() => setDeepError('')}
                message="AI 深检失败"
                description={deepError}
              />
            )}
            {deepResult && !deepResult.ai_available && (
              <Alert
                type="warning" showIcon style={{ width: '100%' }}
                message="AI 深检不可用，已降级为规则检查结果"
                description={deepResult.ai_note || 'AI 模型不可用或逐章提取失败，以下仅包含规则层检查结果。'}
              />
            )}
            {deepResult && deepScanMeta(deepResult) && (
              <div className="novel-setting-meta" data-testid="consistency-deep-meta">{deepScanMeta(deepResult)}</div>
            )}
            {issues.length === 0 ? (
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
                {issues.map((iss, idx) => {
                  const deepIss = iss as ConsistencyDeepIssue
                  const badge = showSourceBadge ? sourceBadge(deepIss.source) : null
                  return (
                    <div key={`${deepIss.source ?? 'rule'}-${deepIss.category}-${deepIss.entity_name}-${idx}`} className="fs-item">
                      <div className="fs-item-head">
                        <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={SEVERITY_META[deepIss.severity]?.color ?? 'default'}>
                          {SEVERITY_META[deepIss.severity]?.label ?? deepIss.severity}
                        </Tag>
                        <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[deepIss.category] || deepIss.category}</Tag>
                        {badge && (
                          <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={badge.color} data-testid="consistency-issue-source">{badge.label}</Tag>
                        )}
                        <span className="fs-item-desc">
                          {deepIss.entity_name ? `${deepIss.entity_name}：` : ''}{deepIss.description}
                        </span>
                      </div>
                      {deepIss.location && (
                        <div className="fs-item-foot">
                          <span className="novel-setting-meta">{deepIss.location}</span>
                        </div>
                      )}
                      {deepIss.suggestion && (
                        <div className="fs-item-foot fs-item-suggestion">
                          <span>建议：{deepIss.suggestion}</span>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

export default ConsistencyPanel
