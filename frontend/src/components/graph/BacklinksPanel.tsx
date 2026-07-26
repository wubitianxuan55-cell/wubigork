import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Tag, Button, Space, Spin, Empty } from 'antd'
import { LinkOutlined, ReloadOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

/**
 * BacklinksPanel — 反向链接面板
 *
 * 显示某个实体在所有章节中的引用位置
 *
 * Props:
 *   entityName — 实体名称
 *   onNavigate — 导航到指定章节的回调 (chapterNum) => void
 */
interface BacklinkItem {
  target: string
  source_file: string
  line_number: number
  context: string
}

interface BacklinksPanelProps {
  entityName: string
  onNavigate?: (chapterNum: number) => void
}

const BacklinksPanel: React.FC<BacklinksPanelProps> = ({ entityName, onNavigate }) => {
  const [links, setLinks] = useState<BacklinkItem[]>([])
  const [loading, setLoading] = useState(false)

  const loadBacklinks = useCallback(async () => {
    if (!entityName) return
    setLoading(true)
    try {
      // @ts-ignore
      const result = await window.go.app.App.GetBacklinks(entityName)
      setLinks(Array.isArray(result) ? result : [])
    } catch (err) {
      console.warn('加载反向链接失败:', err)
      setLinks([])
    } finally {
      setLoading(false)
    }
  }, [entityName])

  useEffect(() => {
    loadBacklinks()
  }, [loadBacklinks])

  // 从 source_file 提取章节号
  const extractChapterNum = (file: string): number => {
    const match = file.match(/(\d+)\.md/)
    return match ? parseInt(match[1]) : 0
  }

  return (
    <div
      style={{
        background: 'var(--bg-glass)',
        borderRadius: 'var(--radius-lg)',
        border: '1px solid var(--border-subtle)',
        padding: 12,
        maxHeight: 400,
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
        }}
      >
        <Space>
          <LinkOutlined style={{ color: C('color-accent'), fontSize: 14 }} />
          <Typography.Text strong style={{ fontSize: 13 }}>
            「{entityName}」的引用
          </Typography.Text>
        </Space>
        <Button size="small" type="text" icon={<ReloadOutlined />} onClick={loadBacklinks} loading={loading} />
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Spin size="small" />
        </div>
      ) : links.length === 0 ? (
        <Empty description="无引用记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <div style={{ flex: 1, overflow: 'auto' }}>
          {links.map((link, i) => {
            const chNum = extractChapterNum(link.source_file)
            return (
              <div
                key={i}
                style={{
                  padding: '6px 8px',
                  marginBottom: 4,
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--border-subtle)',
                  cursor: chNum > 0 ? 'pointer' : 'default',
                  transition: 'background 150ms',
                }}
                onClick={() => {
                  if (chNum > 0) onNavigate?.(chNum)
                }}
                onMouseEnter={e => {
                  (e.currentTarget as HTMLElement).style.background = 'var(--bg-elevated)'
                }}
                onMouseLeave={e => {
                  (e.currentTarget as HTMLElement).style.background = 'transparent'
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <Tag style={{ fontSize: 10, padding: '0 4px', margin: 0 }}>
                    第{chNum}章 L{link.line_number}
                  </Tag>
                  <Typography.Text
                    style={{ fontSize: 11, color: 'var(--color-text-secondary)', flex: 1 }}
                    ellipsis
                  >
                    {link.context}
                  </Typography.Text>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div style={{ marginTop: 8, textAlign: 'center' }}>
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          共 {links.length} 处引用
        </Typography.Text>
      </div>
    </div>
  )
}

export default BacklinksPanel
export type { BacklinksPanelProps, BacklinkItem }
