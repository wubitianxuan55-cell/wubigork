import React, { useEffect, useState, useRef } from 'react'
import { Typography, Tag, Button, Spin, Empty, Space, Tooltip } from 'antd'
import { ReloadOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'

/**
 * CanvasCards — 拖拽式软木板
 *
 * 场景卡片网格布局，支持缩放和悬停，点击导航到章节
 * 显示章节间连接关系
 */
interface CanvasCard {
  id: string
  type: string
  title: string
  content: string
  x: number
  y: number
  width: number
  height: number
  color: string
  chapter_ref: number
}

interface CanvasEdge {
  id: string
  from_id: string
  to_id: string
  label: string
  color: string
}

const CanvasCards: React.FC = () => {
  const [cards, setCards] = useState<CanvasCard[]>([])
  const [edges, setEdges] = useState<CanvasEdge[]>([])
  const [loading, setLoading] = useState(true)
  const [scale, setScale] = useState(0.8)
  const [hoveredID, setHoveredID] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.GenerateDefaultCanvas()
        setCards(result?.cards || [])
        setEdges(result?.edges || [])
      } catch (_) {
        setCards([])
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载画布..." /></div>
  if (cards.length === 0) return <Empty description="暂无章节数据" />

  const maxX = Math.max(...cards.map(c => c.x + c.width), 400)
  const maxY = Math.max(...cards.map(c => c.y + c.height), 400)

  return (
    <div>
      {/* 工具栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Space>
          <Typography.Text strong style={{ fontSize: 13 }}>📋 软木板</Typography.Text>
          <Tag style={{ fontSize: 10 }}>{cards.length} 张卡片</Tag>
        </Space>
        <Space size={4}>
          <Tooltip title="缩小">
            <Button
              size="small" type="text"
              icon={<ZoomOutOutlined />}
              onClick={() => setScale(s => Math.max(0.3, s - 0.1))}
            />
          </Tooltip>
          <span style={{ fontSize: 10, opacity: 0.5, width: 36, textAlign: 'center' }}>
            {Math.round(scale * 100)}%
          </span>
          <Tooltip title="放大">
            <Button
              size="small" type="text"
              icon={<ZoomInOutlined />}
              onClick={() => setScale(s => Math.min(2, s + 0.1))}
            />
          </Tooltip>
          <Tooltip title="重置">
            <Button size="small" type="text" icon={<ReloadOutlined />} onClick={() => setScale(0.8)} />
          </Tooltip>
        </Space>
      </div>

      {/* 画布区域 */}
      <div
        ref={containerRef}
        style={{
          height: 420,
          overflow: 'auto',
          background: 'var(--bg-deep)',
          borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--border-subtle)',
          position: 'relative',
        }}
      >
        <div
          style={{
            position: 'relative',
            width: maxX * scale + 40,
            height: maxY * scale + 40,
            transformOrigin: 'top left',
          }}
        >
          {/* SVG 连线层 */}
          <svg
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: '100%',
              pointerEvents: 'none',
              zIndex: 0,
            }}
          >
            {edges.map(edge => {
              const from = cards.find(c => c.id === edge.from_id)
              const to = cards.find(c => c.id === edge.to_id)
              if (!from || !to) return null

              return (
                <line
                  key={edge.id}
                  x1={(from.x + from.width / 2) * scale}
                  y1={(from.y + from.height) * scale}
                  x2={(to.x + to.width / 2) * scale}
                  y2={to.y * scale}
                  stroke={edge.color}
                  strokeWidth={1}
                  strokeDasharray="4 2"
                  opacity={0.3}
                />
              )
            })}
          </svg>

          {/* 卡片层 */}
          {cards.map(card => {
            const isHovered = hoveredID === card.id
            return (
              <div
                key={card.id}
                style={{
                  position: 'absolute',
                  left: card.x * scale,
                  top: card.y * scale,
                  width: card.width * scale,
                  height: card.height * scale,
                  background: 'var(--bg-glass)',
                  backdropFilter: 'blur(8px)',
                  borderRadius: 'var(--radius-md)',
                  border: `2px solid ${card.color}`,
                  borderLeft: `4px solid ${card.color}`,
                  padding: 8 * scale,
                  cursor: 'pointer',
                  transition: 'transform 200ms ease, box-shadow 200ms ease',
                  transform: isHovered ? `scale(1.05)` : 'scale(1)',
                  boxShadow: isHovered ? 'var(--shadow-lg)' : 'var(--shadow-sm)',
                  zIndex: isHovered ? 10 : 1,
                  overflow: 'hidden',
                }}
                onMouseEnter={() => setHoveredID(card.id)}
                onMouseLeave={() => setHoveredID(null)}
              >
                <Tag
                  color={card.color}
                  style={{
                    fontSize: Math.max(8, 9 * scale),
                    padding: `0 ${3 * scale}px`,
                    marginBottom: 4 * scale,
                  }}
                >
                  {card.type}
                  {card.chapter_ref > 0 && ` · 第${card.chapter_ref}章`}
                </Tag>

                <Typography.Text
                  strong
                  style={{
                    fontSize: Math.max(9, 11 * scale),
                    display: 'block',
                    marginBottom: 4 * scale,
                  }}
                  ellipsis
                >
                  {card.title}
                </Typography.Text>

                {card.content && (
                  <Typography.Paragraph
                    style={{
                      fontSize: Math.max(8, 9 * scale),
                      color: 'var(--color-text-secondary)',
                      margin: 0,
                      lineHeight: 1.4,
                    }}
                    ellipsis
                  >
                    {card.content}
                  </Typography.Paragraph>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export default CanvasCards
