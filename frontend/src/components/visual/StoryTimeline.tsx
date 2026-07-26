import React, { useEffect, useRef, useState } from 'react'
import { Typography, Tag, Spin, Empty } from 'antd'

/**
 * StoryTimeline — 水平滚动故事时间线
 *
 * 彩色泳道展示各章场景卡片，支持缩放和悬停详情
 *
 * Props: 无（内部调用 ExtractTimeline API）
 */
interface TimelineEvent {
  chapter_num: number
  title: string
  summary: string
  emotion: string
  characters: string[]
  pov: string
  word_count: number
  key_events: string[]
  quality_score: number
}

const EMOTION_COLORS: Record<string, string> = {
  '紧张': '#f87171', '悬疑': '#f87171', '恐惧': '#ef4444',
  '温馨': '#4ade80', '浪漫': '#f472b6', '希望': '#34d399',
  '悲伤': '#60a5fa', '绝望': '#3b82f6',
  '愤怒': '#f59e0b', '战斗': '#f97316',
  '平静': '#9ca3af', '日常': '#94a3b8',
}

const getEmotionColor = (emotion: string) => {
  for (const [key, color] of Object.entries(EMOTION_COLORS)) {
    if (emotion?.includes(key)) return color
  }
  return '#c084fc'
}

const StoryTimeline: React.FC = () => {
  const [events, setEvents] = useState<TimelineEvent[]>([])
  const [totalWords, setTotalWords] = useState(0)
  const [loading, setLoading] = useState(true)
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.ExtractTimeline()
        setEvents(result?.events || [])
        setTotalWords(result?.total_words || 0)
      } catch (_) {
        setEvents([])
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载时间线..." /></div>
  }

  if (events.length === 0) {
    return <Empty description="暂无章节数据" />
  }

  const maxWords = Math.max(...events.map(e => e.word_count), 1)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* 标题栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>📅 故事时间线</Typography.Text>
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          {events.length} 章 · {totalWords.toLocaleString()} 字
        </Typography.Text>
      </div>

      {/* 水平滚动容器 */}
      <div
        ref={scrollRef}
        style={{
          display: 'flex',
          gap: 12,
          overflowX: 'auto',
          overflowY: 'visible',
          padding: '12px 4px 24px',
          scrollBehavior: 'smooth',
        }}
        onWheel={e => {
          if (scrollRef.current) {
            e.preventDefault()
            scrollRef.current.scrollLeft += e.deltaY
          }
        }}
      >
        {events.map((ev, idx) => {
          const color = getEmotionColor(ev.emotion)
          const isHovered = hoveredIdx === idx

          return (
            <div
              key={idx}
              style={{
                flexShrink: 0,
                width: 140,
                cursor: 'pointer',
                transition: 'transform 200ms ease',
                transform: isHovered ? 'translateY(-6px) scale(1.05)' : 'none',
                position: 'relative',
              }}
              onMouseEnter={() => setHoveredIdx(idx)}
              onMouseLeave={() => setHoveredIdx(null)}
            >
              {/* 章节卡片 */}
              <div
                style={{
                  background: 'var(--bg-glass)',
                  borderRadius: 'var(--radius-md)',
                  border: `2px solid ${color}`,
                  borderTop: `4px solid ${color}`,
                  padding: 10,
                  boxShadow: isHovered ? 'var(--shadow-lg)' : 'var(--shadow-sm)',
                }}
              >
                {/* 章节号 */}
                <Tag color={color} style={{ fontSize: 10, padding: '0 4px', marginBottom: 4 }}>
                  第{ev.chapter_num}章
                </Tag>

                {/* 标题 */}
                <Typography.Text
                  strong
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                  ellipsis
                >
                  {ev.title}
                </Typography.Text>

                {/* 字数条 */}
                <div
                  style={{
                    height: 3,
                    background: 'var(--bg-deep)',
                    borderRadius: 2,
                    marginBottom: 4,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      height: '100%',
                      width: `${(ev.word_count / maxWords) * 100}%`,
                      background: color,
                      borderRadius: 2,
                    }}
                  />
                </div>

                {/* 情感 + 字数 */}
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10 }}>
                  <span style={{ color }}>{ev.emotion || '—'}</span>
                  <span style={{ opacity: 0.5 }}>{ev.word_count}字</span>
                </div>

                {/* 出场角色 */}
                {ev.characters?.length > 0 && (
                  <div style={{ marginTop: 4, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                    {ev.characters.slice(0, 3).map(ch => (
                      <Tag key={ch} style={{ fontSize: 9, padding: '0 3px', margin: 0 }}>
                        {ch}
                      </Tag>
                    ))}
                    {ev.characters.length > 3 && (
                      <span style={{ fontSize: 9, opacity: 0.5 }}>+{ev.characters.length - 3}</span>
                    )}
                  </div>
                )}
              </div>

              {/* 时间轴连接线 */}
              {idx < events.length - 1 && (
                <div
                  style={{
                    position: 'absolute',
                    top: '50%',
                    right: -12,
                    width: 12,
                    height: 2,
                    background: 'var(--border-subtle)',
                  }}
                />
              )}

              {/* 悬停详情 */}
              {isHovered && ev.summary && (
                <div
                  style={{
                    position: 'absolute',
                    bottom: '100%',
                    left: '50%',
                    transform: 'translateX(-50%)',
                    background: 'var(--bg-elevated)',
                    borderRadius: 'var(--radius-md)',
                    border: '1px solid var(--border-subtle)',
                    boxShadow: 'var(--shadow-lg)',
                    padding: 8,
                    width: 200,
                    zIndex: 100,
                    fontSize: 11,
                    lineHeight: 1.5,
                    pointerEvents: 'none',
                  }}
                >
                  <Typography.Text style={{ fontSize: 11 }}>{ev.summary.slice(0, 150)}</Typography.Text>
                  {ev.quality_score > 0 && (
                    <div style={{ marginTop: 4 }}>
                      <Tag color={ev.quality_score >= 7 ? 'green' : ev.quality_score >= 5 ? 'orange' : 'red'}>
                        评分: {ev.quality_score}/10
                      </Tag>
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default StoryTimeline
