import React, { useMemo, useState } from 'react'
import { Button, Input, Tag, Typography } from 'antd'
import {
  AppstoreOutlined, CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined,
  DeleteOutlined, HistoryOutlined, LoadingOutlined, MenuFoldOutlined, MenuUnfoldOutlined,
  SearchOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { HistoryRail } from './HistoryRail'
import { CATEGORIES, type Template, type CustomTemplate } from '../../data/imageTemplates'
import type { GenResult, QueueEntry, QueueStatus } from './types'

type TabKey = 'queue' | 'recent' | 'templates'

const STATUS_META: Record<QueueStatus, { label: string; icon: React.ReactNode; color: string }> = {
  pending: { label: '待执行', icon: <ClockCircleOutlined />, color: 'default' },
  running: { label: '执行中', icon: <LoadingOutlined />, color: 'processing' },
  done: { label: '完成', icon: <CheckCircleOutlined />, color: 'success' },
  failed: { label: '失败', icon: <CloseCircleOutlined />, color: 'error' },
  canceled: { label: '已取消', icon: <CloseCircleOutlined />, color: 'default' },
}

const MODE_LABEL: Record<string, string> = {
  txt2img: '文生图', img2img: '图生图', t2v: '文生视频',
}

interface Props {
  queueItems: QueueEntry[]
  onClearQueue: () => void
  onCancelQueue: () => void
  history: GenResult[]
  selectedIndex: number
  onSelectHistory: (i: number) => void
  onClearHistory: () => void
  onRegenerateMeta?: (item: GenResult) => void
  templates: Record<string, Template[]>
  customTemplates: CustomTemplate[]
  onApplyTemplate: (t: Template) => void
  onManageTemplates: () => void
}

const TABS: { key: TabKey; label: string; icon: React.ReactNode }[] = [
  { key: 'queue', label: '队列', icon: <ThunderboltOutlined /> },
  { key: 'recent', label: '最近结果', icon: <HistoryOutlined /> },
  { key: 'templates', label: '模板', icon: <AppstoreOutlined /> },
]

export const TaskCenter: React.FC<Props> = ({
  queueItems, onClearQueue, onCancelQueue,
  history, selectedIndex, onSelectHistory, onClearHistory, onRegenerateMeta,
  templates, customTemplates, onApplyTemplate, onManageTemplates,
}) => {
  const [tab, setTab] = useState<TabKey>('queue')
  const [collapsed, setCollapsed] = useState(false)
  const [keyword, setKeyword] = useState('')

  const activeRunning = queueItems.some((q) => q.status === 'pending' || q.status === 'running')
  const recentCount = history.length
  const kw = keyword.trim().toLowerCase()

  const matchedCategories = useMemo(() => {
    if (!kw) return CATEGORIES
    return CATEGORIES.filter((cat) =>
      (templates[cat.id] || []).some((t) =>
        t.label.toLowerCase().includes(kw) || (t.description || '').toLowerCase().includes(kw),
      ),
    )
  }, [kw, templates])

  const customMatch = useMemo(
    () => customTemplates.filter((t) =>
      !kw || t.label.toLowerCase().includes(kw) || (t.description || '').toLowerCase().includes(kw),
    ),
    [customTemplates, kw],
  )
  const hasAnyTemplate = customMatch.length > 0 || matchedCategories.length > 0

  if (collapsed) {
    return (
      <div className="ig-task-center is-collapsed v3-panel" role="complementary" aria-label="历史与任务队列（已收起）">
        {TABS.map((t) => {
          const count = t.key === 'queue' ? queueItems.length : t.key === 'recent' ? recentCount : customTemplates.length
          const active = tab === t.key
          return (
            <button
              key={t.key}
              type="button"
              title={t.label}
              onClick={() => { setTab(t.key); setCollapsed(false) }}
              className={`ig-task-icon${active ? ' is-active' : ''}`}
            >
              {t.icon}
              {count > 0 && <span className="ig-task-badge">{count}</span>}
            </button>
          )
        })}
        <button
          type="button"
          className="ig-task-icon"
          title="展开任务中心"
          onClick={() => setCollapsed(false)}
        >
          <MenuUnfoldOutlined />
        </button>
      </div>
    )
  }

  return (
    <div className="ig-task-center v3-panel" role="complementary" aria-label="历史与任务队列">
      <div className="ig-task-tabs" role="tablist" aria-label="任务中心">
        {TABS.map((t) => {
          const active = tab === t.key
          const count = t.key === 'queue' ? queueItems.length : t.key === 'recent' ? recentCount : customTemplates.length
          return (
            <button
              key={t.key}
              type="button"
              role="tab"
              aria-selected={active}
              className={`ig-task-tab${active ? ' is-active' : ''}`}
              onClick={() => setTab(t.key)}
            >
              {t.icon}
              {t.label}
              {count > 0 && <span className="ig-task-tab-count">{count}</span>}
            </button>
          )
        })}
        <button
          type="button"
          className="ig-task-icon"
          title="收起任务中心"
          onClick={() => setCollapsed(true)}
          style={{ marginLeft: 'auto' }}
        >
          <MenuFoldOutlined />
        </button>
      </div>

      <div className="ig-task-body" style={{ flex: 1, minHeight: 0, overflowY: 'auto', overflowX: 'hidden', padding: '10px 2px 2px' }}>
        {tab === 'queue' && (
          queueItems.length === 0 ? (
            <div className="ig-task-empty">
              <ClockCircleOutlined style={{ fontSize: 20, color: C('color-text-secondary'), opacity: 0.6 }} />
              <span>没有排队中的任务</span>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {queueItems.map((q) => {
                const meta = STATUS_META[q.status]
                const iconColor = q.status === 'running' ? 'var(--color-primary)'
                  : q.status === 'done' ? 'var(--color-success)'
                  : q.status === 'failed' ? 'var(--color-destructive)'
                  : C('color-text-secondary')
                return (
                  <div key={q.id} className="ig-queue-row">
                    <span style={{ color: iconColor }}>
                      {meta.icon}
                    </span>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{
                        fontSize: 12, color: 'var(--color-text)', whiteSpace: 'nowrap',
                        overflow: 'hidden', textOverflow: 'ellipsis',
                      }}>
                        {q.task.prompt || '(空提示词)'}
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginTop: 2 }}>
                        <Tag color={meta.color} style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>
                          {meta.label}
                        </Tag>
                        <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary') }}>
                          {MODE_LABEL[q.task.mode] || q.task.mode} · {q.task.model}
                        </Typography.Text>
                      </div>
                    </div>
                  </div>
                )
              })}
              <div style={{ display: 'flex', gap: 6, marginTop: 4 }}>
                {activeRunning && (
                  <Button size="small" block icon={<CloseCircleOutlined />} onClick={onCancelQueue}
                    style={{ borderRadius: 999, fontSize: 11 }}>
                    取消排队
                  </Button>
                )}
                <Button size="small" block icon={<DeleteOutlined />} onClick={onClearQueue}
                  style={{ borderRadius: 999, fontSize: 11 }}>
                  清空记录
                </Button>
              </div>
            </div>
          )
        )}

        {tab === 'recent' && (
          <HistoryRail
            history={history}
            selectedIndex={selectedIndex}
            onSelect={onSelectHistory}
            onClear={onClearHistory}
            onRegenerateMeta={onRegenerateMeta}
            collapsed={false}
          />
        )}

        {tab === 'templates' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <Input
              size="small"
              allowClear
              prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
              placeholder="搜索模板"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              style={{ borderRadius: 999, fontSize: 12 }}
            />

            {customMatch.length > 0 && (
              <TemplateGroup
                title="自定义"
                color="var(--color-primary)"
                items={customMatch}
                onApply={onApplyTemplate}
              />
            )}

            {matchedCategories.map((cat) => {
              const items = (templates[cat.id] || []).filter((t) =>
                !kw || t.label.toLowerCase().includes(kw) || (t.description || '').toLowerCase().includes(kw),
              )
              if (items.length === 0) return null
              return (
                <TemplateGroup
                  key={cat.id}
                  title={cat.label}
                  color={cat.color}
                  items={items}
                  onApply={onApplyTemplate}
                />
              )
            })}

            {kw && !hasAnyTemplate && (
              <div className="ig-task-empty">
                <SearchOutlined style={{ fontSize: 18, color: C('color-text-secondary'), opacity: 0.6 }} />
                <span>未找到匹配模板</span>
              </div>
            )}

            <Button size="small" block icon={<AppstoreOutlined />} onClick={onManageTemplates}
              style={{ borderRadius: 999, fontSize: 11, marginTop: 2 }}>
              管理模板库
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

const TemplateGroup: React.FC<{
  title: string
  color: string
  items: Template[]
  onApply: (t: Template) => void
}> = ({ title, color, items, onApply }) => (
  <div>
    <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 5 }}>
      <span style={{ width: 7, height: 7, borderRadius: '50%', background: color, flexShrink: 0 }} />
      <Typography.Text style={{ fontSize: 11, fontWeight: 600, letterSpacing: '0.04em', color: C('color-text-secondary') }}>
        {title}
      </Typography.Text>
      <span style={{ fontSize: 10, color: C('color-text-secondary'), opacity: 0.7 }}>{items.length}</span>
    </div>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {items.map((t) => (
        <button
          key={t.label}
          type="button"
          title={t.description || t.prompt}
          onClick={() => onApply(t)}
          className="ig-template-row"
        >
          <span style={{ fontSize: 12, color: 'var(--color-text)', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {t.label}
          </span>
          {t.description && (
            <span style={{ fontSize: 10, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {t.description}
            </span>
          )}
        </button>
      ))}
    </div>
  </div>
)
