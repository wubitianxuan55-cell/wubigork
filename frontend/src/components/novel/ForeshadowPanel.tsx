// ForeshadowPanel.tsx — 伏笔登记表面板（v4.3f）
// 列表展示 GetForeshadows（内容/章节/状态徽标 planted→hinted→revealed）+ 回收率统计；
// 空态引导、加载中、错误降级。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Button, Empty, Spin, Tag, Tooltip } from 'antd'
import { FlagOutlined, ReloadOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import type { ForeshadowItemData, ForeshadowStatus } from '../../types'

const STATUS_META: Record<ForeshadowStatus, { label: string; color: string }> = {
  planted: { label: '已埋设', color: 'blue' },
  hinted: { label: '已暗示', color: 'gold' },
  revealed: { label: '已回收', color: 'green' },
}

const CATEGORY_LABELS: Record<string, string> = {
  character: '角色',
  plot: '剧情',
  world: '世界观',
  relationship: '关系',
}

function normalizeItems(raw: unknown): ForeshadowItemData[] {
  const list = ((raw as { items?: ForeshadowItemData[] } | null)?.items ?? []) as ForeshadowItemData[]
  return Array.isArray(list) ? list : []
}

interface ForeshadowPanelProps {
  /** 未打开项目时仅展示空态引导，不触发加载 */
  disabled?: boolean
}

const ForeshadowPanel: React.FC<ForeshadowPanelProps> = ({ disabled }) => {
  const [items, setItems] = useState<ForeshadowItemData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadToken = useRef(0)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setItems([])
      setLoading(false)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await App.GetForeshadows()
      if (token !== loadToken.current) return
      setItems(normalizeItems(res))
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setError(err instanceof Error ? err.message : '伏笔加载失败')
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  const stats = useMemo(() => {
    const total = items.length
    const revealed = items.filter((i) => i.status === 'revealed').length
    const hinted = items.filter((i) => i.status === 'hinted').length
    const planted = items.filter((i) => i.status === 'planted').length
    const rate = total === 0 ? 0 : Math.round((revealed / total) * 100)
    return { total, revealed, hinted, planted, rate }
  }, [items])

  const flowLegend = (
    <div className="fs-flow-legend">
      <span>埋设</span><span className="fs-arrow">→</span>
      <span>暗示</span><span className="fs-arrow">→</span>
      <span>回收</span>
    </div>
  )

  return (
    <div className="novel-panel fs-panel" style={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 4 }}>
        <span className="novel-panel-title"><FlagOutlined />伏笔登记</span>
        <div style={{ flex: 1 }} />
        <span className="novel-setting-meta">
          回收率 {stats.rate}%（{stats.revealed}/{stats.total}）
        </span>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading} disabled={disabled}>
          刷新
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
            message="伏笔加载失败"
            description={error}
            action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void load()}>重试</Button>}
          />
        ) : items.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <span>
                还没有伏笔登记。生成章节并分析后，伏笔会按 埋设 → 暗示 → 回收 自动登记在这里。
              </span>
            }
            style={{ margin: 'auto' }}
          />
        ) : (
          <div className="fs-list" style={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div className="fs-stats-row">
              {flowLegend}
              <div style={{ flex: 1 }} />
              <span className="novel-setting-meta">埋设 {stats.planted} · 暗示 {stats.hinted} · 回收 {stats.revealed}</span>
            </div>
            {items.map((it) => (
              <div key={it.id} className="fs-item">
                <div className="fs-item-head">
                  <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[it.category] || it.category}</Tag>
                  <span className="fs-item-desc">{it.description}</span>
                  <div style={{ flex: 1 }} />
                  <Tooltip title={`章节：${it.planted_in}`}>
                    <span className="novel-setting-meta">{it.planted_in}</span>
                  </Tooltip>
                  <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color={STATUS_META[it.status].color}>
                    {STATUS_META[it.status].label}
                  </Tag>
                </div>
                {it.is_long_term && (
                  <div className="fs-item-foot">
                    <Tag style={{ fontSize: 10, marginInlineEnd: 0 }} color="purple">长期伏笔</Tag>
                    {it.revealed_in && <span className="novel-setting-meta">回收于 {it.revealed_in}</span>}
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

export default ForeshadowPanel
