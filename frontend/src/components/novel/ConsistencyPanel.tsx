// ConsistencyPanel.tsx — 一致性检查面板（v4.3f + AI 深检 v0 + 误报缓解 v4.101）
// 展示 CheckConsistency 三类规则告警（角色属性/角色状态/时间线），带严重度/描述，
// 「重新检查」按钮 + 全部通过时的成功空态。
// 「AI 深检」：AI 逐章提取实体状态卡，本地跨章比对矛盾（死亡再出场/位置瞬移/
// 物品凭空消失·无中生有/时间倒流），与规则层合并展示（source 徽标区分来源）；
// AI 不可用时诚实降级为规则层结果并展示说明。
// 误报缓解（v4.101）：深检模式下告警按 冲突/疑似/提示 三档分级展示（后端带
// confidence/reason 标注，措辞/粒度/别名类降级附原因徽标）；每条可「忽略」并按
// 项目记忆（localStorage），被忽略条目以计数横幅保持可见、可一键恢复显示。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, InputNumber, Spin, Tag } from 'antd'
import { CheckCircleOutlined, CloseOutlined, ReloadOutlined, SafetyCertificateOutlined, ThunderboltOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
// 注意：用 i18n 模块的非响应式 t（根级 LocaleProvider 每次渲染同步 currentLocale
// 镜像），而非 useT —— NovelSettingPage 测试直接渲染本组件且外层无 Provider，
// useI18n 会抛错；非响应式 t 无 Provider 依赖，测试环境安全回退英文。
import { t as dictT } from '../../gaea/lib/i18n'
import type { DictKey } from '../../gaea/locales/en'
import { useAppStore } from '../../stores/appStore'
import {
  clampDeepChapters, deepIssueLevel, deepIssueReason, deepScanMeta,
  normalizeDeepResult, sourceBadge,
} from './consistencyDeep'
import type { DeepIssueLevel, DeepIssueReason } from './consistencyDeep'
import { clearIgnoredIssues, deepIssueFingerprint, ignoreIssue, loadIgnoredFingerprints } from './consistencyIgnore'
import type { ConsistencyCheckIssue, ConsistencyCheckReport, ConsistencyDeepIssue, ConsistencyDeepResult, ConsistencySeverity } from '../../types'

const SEVERITY_META: Record<ConsistencySeverity, { label: string; color: string }> = {
  error: { label: '错误', color: 'red' },
  warning: { label: '警告', color: 'gold' },
  info: { label: '提示', color: 'blue' },
}

/** 深检模式三档分级样式（i18n 标签，规则层-only 视图不启用分级） */
const LEVEL_META: Record<DeepIssueLevel, { labelKey: DictKey; color: string }> = {
  conflict: { labelKey: 'novelDeep.levelConflict', color: 'red' },
  suspected: { labelKey: 'novelDeep.levelSuspected', color: 'gold' },
  hint: { labelKey: 'novelDeep.levelHint', color: 'blue' },
}

/** 缓解原因徽标文案（与后端 reason 分类一一对应） */
const REASON_LABEL_KEY: Record<DeepIssueReason, DictKey> = {
  wording: 'novelDeep.reasonWording',
  granularity: 'novelDeep.reasonGranularity',
  alias: 'novelDeep.reasonAlias',
  unexplained: 'novelDeep.reasonUnexplained',
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
  const t = dictT
  const projectPath = useAppStore((s) => s.projectPath)
  const [report, setReport] = useState<ConsistencyCheckReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [deepResult, setDeepResult] = useState<ConsistencyDeepResult | null>(null)
  const [deepLoading, setDeepLoading] = useState(false)
  const [deepError, setDeepError] = useState('')
  const [deepChapters, setDeepChapters] = useState(20)
  const [ignoredFps, setIgnoredFps] = useState<string[]>([])
  const loadToken = useRef(0)

  // 忽略记忆随项目切换加载（localStorage 按项目路径隔离）
  useEffect(() => {
    setIgnoredFps(loadIgnoredFingerprints(projectPath))
  }, [projectPath])

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

  const allIssues: ConsistencyCheckIssue[] = deepResult?.issues ?? report?.issues ?? []
  // 忽略记忆只在深检视图生效（指纹按项目隔离）；被忽略条目以计数横幅保持可见
  const issues = deepResult
    ? allIssues.filter((iss) => !ignoredFps.includes(deepIssueFingerprint(iss as ConsistencyDeepIssue)))
    : allIssues
  const ignoredCount = deepResult ? allIssues.length - issues.length : 0
  const summaryText = deepResult?.summary ?? report?.summary
  const showSourceBadge = deepResult !== null

  const handleIgnore = useCallback((iss: ConsistencyDeepIssue) => {
    ignoreIssue(projectPath, iss)
    setIgnoredFps(loadIgnoredFingerprints(projectPath))
  }, [projectPath])

  const handleRestore = useCallback(() => {
    clearIgnoredIssues(projectPath)
    setIgnoredFps(loadIgnoredFingerprints(projectPath))
  }, [projectPath])

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
            {deepResult && ignoredCount > 0 && (
              <Alert
                type="info" showIcon style={{ width: '100%' }}
                data-testid="consistency-ignored-banner"
                message={t('novelDeep.ignoredSummary', { count: ignoredCount })}
                action={(
                  <Button size="small" type="text" onClick={handleRestore} data-testid="consistency-ignored-restore">
                    {t('novelDeep.restoreIgnored')}
                  </Button>
                )}
              />
            )}
            {issues.length === 0 ? (
              ignoredCount > 0 ? (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={<span data-testid="consistency-all-ignored">{t('novelDeep.allIgnored', { count: ignoredCount })}</span>}
                  style={{ margin: 'auto' }}
                />
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={
                    <span style={{ color: 'var(--color-success, #34d399)' }}>
                      <CheckCircleOutlined /> 全部通过，未发现一致性问题
                    </span>
                  }
                  style={{ margin: 'auto' }}
                />
              )
            ) : (
              <div className="fs-list" style={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
                {issues.map((iss, idx) => {
                  const deepIss = iss as ConsistencyDeepIssue
                  const badge = showSourceBadge ? sourceBadge(deepIss.source) : null
                  // 分级标签只在深检视图启用（冲突/疑似/提示）；规则层-only 视图沿用严重度
                  const level: DeepIssueLevel | null = showSourceBadge ? deepIssueLevel(deepIss) : null
                  const reason: DeepIssueReason | '' = showSourceBadge ? deepIssueReason(deepIss) : ''
                  return (
                    <div key={`${deepIss.source ?? 'rule'}-${deepIss.category}-${deepIss.entity_name}-${idx}`} className="fs-item">
                      <div className="fs-item-head">
                        {level ? (
                          <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={LEVEL_META[level].color} data-testid="consistency-issue-level">
                            {t(LEVEL_META[level].labelKey)}
                          </Tag>
                        ) : (
                          <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={SEVERITY_META[deepIss.severity]?.color ?? 'default'}>
                            {SEVERITY_META[deepIss.severity]?.label ?? deepIss.severity}
                          </Tag>
                        )}
                        <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[deepIss.category] || deepIss.category}</Tag>
                        {badge && (
                          <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color={badge.color} data-testid="consistency-issue-source">{badge.label}</Tag>
                        )}
                        {reason && (
                          <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="warning" data-testid="consistency-issue-reason">
                            {t(REASON_LABEL_KEY[reason])}
                          </Tag>
                        )}
                        <span className="fs-item-desc">
                          {deepIss.entity_name ? `${deepIss.entity_name}：` : ''}{deepIss.description}
                        </span>
                        {showSourceBadge && (
                          <Button
                            type="text" size="small" icon={<CloseOutlined />}
                            onClick={() => handleIgnore(deepIss)}
                            title={t('novelDeep.ignoreTip')}
                            data-testid="consistency-issue-ignore"
                          />
                        )}
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
