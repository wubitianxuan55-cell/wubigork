import React, { useState, useEffect, useRef } from 'react'
import { Typography, Space, Tag, Modal, Spin, Progress } from 'antd'
import {
  BarChartOutlined,
} from '@ant-design/icons'

import { C } from '../utils/theme'

interface ChapterStats {
  num: number
  title: string
  words: number
  quality: number
}

interface StatsModalProps {
  open: boolean
  onClose: () => void
  getChapterStats?: () => Promise<ChapterStats[]>
  totalWords: number
  chapterCount: number
}

const StatsModal: React.FC<StatsModalProps> = ({ open, onClose, getChapterStats, totalWords, chapterCount }) => {
  const [loading, setLoading] = useState(false)
  const [chStats, setChStats] = useState<ChapterStats[]>([])
  const cacheRef = useRef<{ data: ChapterStats[]; time: number } | null>(null)

  useEffect(() => {
    if (!open || !getChapterStats) return
    // 缓存 30 秒内不重复请求
    if (cacheRef.current && Date.now() - cacheRef.current.time < 30000) {
      setChStats(cacheRef.current.data)
      return
    }
    setLoading(true)
    getChapterStats().then((s) => {
      setChStats(s)
      cacheRef.current = { data: s, time: Date.now() }
      setLoading(false)
    })
  }, [open])

  const maxWords = Math.max(1, ...chStats.map((c) => c.words))

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><BarChartOutlined style={{ color: C('color-primary'), marginRight: 8 }} />写作统计</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={700}
      styles={{
        body: { maxHeight: '70vh', overflow: 'auto' },
      }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
      ) : (
        <Space direction="vertical" size={20} style={{ width: '100%' }}>
          {/* 总览卡片 */}
          <div style={{ display: 'flex', gap: 12 }}>
            {[
              { label: '总字数', value: totalWords.toLocaleString(), color: '#4ade80' },
              { label: '章节数', value: chapterCount.toString(), color: '#60a5fa' },
              { label: '均字数', value: chapterCount > 0 ? Math.round(totalWords / chapterCount).toLocaleString() : '0', color: '#c084fc' },
              { label: '均品质', value: chStats.length > 0 ? (chStats.reduce((a, b) => a + b.quality, 0) / chStats.length).toFixed(1) : '-', color: '#f59e0b' },
            ].map((item) => (
              <div
                key={item.label}
                style={{
                  flex: 1, textAlign: 'center',
                  background: 'var(--bg-elevated)',
                  borderRadius: 'var(--radius-md)', padding: '12px 8px',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                <div style={{ fontSize: 24, fontWeight: 700, color: item.color }}>{item.value}</div>
                <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 4 }}>{item.label}</div>
              </div>
            ))}
          </div>

          {/* 每章字数柱状图 */}
          {chStats.length > 0 && (
            <div>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
                <BarChartOutlined style={{ marginRight: 6 }} />每章字数
              </Typography.Text>
              <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, height: 120, paddingTop: 8 }}>
                {chStats.map((c) => {
                  const h = Math.max(8, (c.words / maxWords) * 100)
                  return (
                    <div key={c.num} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 9, marginBottom: 2 }}>
                        {c.words >= 1000 ? `${(c.words / 1000).toFixed(1)}k` : c.words}
                      </Typography.Text>
                      <div
                        style={{
                          width: '100%', maxWidth: 40,
                          height: h, borderRadius: '4px 4px 0 0',
                          background: C('color-primary'),
                          opacity: 0.7 + (c.quality / 10) * 0.3,
                        }}
                        title={`第${c.num}章 ${c.title}: ${c.words}字 / 品质${c.quality}`}
                      />
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 9, marginTop: 4 }}>
                        {c.num}
                      </Typography.Text>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* 每章品质趋势 */}
          {chStats.length > 0 && (
            <div>
              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
                <BarChartOutlined style={{ marginRight: 6 }} />品质趋势
              </Typography.Text>
              <Space direction="vertical" size={6} style={{ width: '100%' }}>
                {chStats.map((c) => (
                  <div key={c.num} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag style={{ fontSize: 10, width: 44, textAlign: 'center' }}>第{c.num}章</Tag>
                    <Progress
                      percent={c.quality * 10}
                      size="small"
                      strokeColor={c.quality >= 7 ? '#4ade80' : c.quality >= 4 ? '#f59e0b' : '#f87171'}
                      style={{ flex: 1, marginBottom: 0 }}
                      format={() => `${c.quality}/10`}
                    />
                  </div>
                ))}
              </Space>
            </div>
          )}

          {chStats.length === 0 && (
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, textAlign: 'center', display: 'block' }}>
              暂无章节数据
            </Typography.Text>
          )}
        </Space>
      )}
    </Modal>
  )
}

export default StatsModal
