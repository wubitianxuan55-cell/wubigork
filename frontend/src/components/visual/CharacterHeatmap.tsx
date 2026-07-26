import React, { useEffect, useState } from 'react'
import { Typography, Spin, Empty, Tooltip } from 'antd'

/**
 * CharacterHeatmap — 角色弧光热力图
 *
 * 矩阵图: 行=角色, 列=章节
 * 颜色深度 = 该角色在章节中的重要性（出场+提及次数+是否POV）
 */
interface HeatmapCell {
  character_name: string
  chapter_num: number
  appears: boolean
  mention_count: number
  is_pov: boolean
}

const CharacterHeatmap: React.FC = () => {
  const [cells, setCells] = useState<HeatmapCell[]>([])
  const [characters, setCharacters] = useState<string[]>([])
  const [chapterCount, setChapterCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [hovered, setHovered] = useState<{ char: string; ch: number } | null>(null)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.ExtractCharacterHeatmap()
        setCells(result?.cells || [])
        setCharacters(result?.characters || [])
        setChapterCount(result?.chapter_count || 0)
      } catch (_) {
        setCells([])
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载角色数据..." /></div>
  if (characters.length === 0 || chapterCount === 0) return <Empty description="暂无角色数据" />

  // 构建查询索引
  const cellMap = new Map<string, HeatmapCell>()
  for (const c of cells) {
    cellMap.set(`${c.character_name}-${c.chapter_num}`, c)
  }

  // 计算最大提及数（用于颜色强度）
  const maxMentions = Math.max(...cells.map(c => c.mention_count), 1)

  const getCellColor = (charName: string, chNum: number) => {
    const cell = cellMap.get(`${charName}-${chNum}`)
    if (!cell || !cell.appears) return 'transparent'

    if (cell.is_pov) return '#f87171' // POV = red

    const intensity = cell.mention_count / maxMentions
    const r = Math.round(74 + intensity * (248 - 74))
    const g = Math.round(222 + intensity * (113 - 222))
    const b = Math.round(128 + intensity * (113 - 128))
    return `rgb(${r},${g},${b})`
  }

  const cellSize = 24
  const labelWidth = 70

  return (
    <div>
      <Typography.Text strong style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>
        🔥 角色弧光热力图
      </Typography.Text>

      <div style={{ overflowX: 'auto' }}>
        <div style={{ display: 'inline-block' }}>
          {/* 表头 */}
          <div style={{ display: 'flex', marginLeft: labelWidth }}>
            {Array.from({ length: chapterCount }, (_, i) => (
              <div
                key={i}
                style={{
                  width: cellSize,
                  textAlign: 'center',
                  fontSize: 8,
                  color: 'var(--color-text-secondary)',
                  writingMode: 'vertical-rl',
                  padding: '2px 0',
                }}
              >
                {i + 1}
              </div>
            ))}
          </div>

          {/* 行 */}
          {characters.map(charName => (
            <div key={charName} style={{ display: 'flex', alignItems: 'center', marginBottom: 1 }}>
              {/* 角色名 */}
              <div
                style={{
                  width: labelWidth,
                  fontSize: 10,
                  paddingRight: 6,
                  textAlign: 'right',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {charName}
              </div>

              {/* 单元格 */}
              {Array.from({ length: chapterCount }, (_, i) => {
                const chNum = i + 1
                const cell = cellMap.get(`${charName}-${chNum}`)
                const color = getCellColor(charName, chNum)
                const isHovered = hovered?.char === charName && hovered?.ch === chNum

                return (
                  <Tooltip
                    key={i}
                    title={
                      cell?.appears
                        ? `${charName} 第${chNum}章: ${cell.mention_count}次提及${cell.is_pov ? ' (POV)' : ''}`
                        : `${charName} 第${chNum}章: 未出场`
                    }
                  >
                    <div
                      style={{
                        width: cellSize,
                        height: cellSize,
                        background: color,
                        border: isHovered ? '2px solid white' : '1px solid var(--border-subtle)',
                        borderRadius: 2,
                        cursor: 'pointer',
                        transition: 'transform 100ms',
                        transform: isHovered ? 'scale(1.3)' : 'scale(1)',
                      }}
                      onMouseEnter={() => setHovered({ char: charName, ch: chNum })}
                      onMouseLeave={() => setHovered(null)}
                    />
                  </Tooltip>
                )
              })}
            </div>
          ))}
        </div>
      </div>

      {/* 图例 */}
      <div style={{ display: 'flex', gap: 12, marginTop: 8, fontSize: 10, alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <div style={{ width: 10, height: 10, background: '#f87171', borderRadius: 2 }} />
          <span>POV</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <div style={{ width: 10, height: 10, background: 'rgb(74,222,128)', borderRadius: 2 }} />
          <span>出场</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <div style={{ width: 10, height: 10, background: 'transparent', border: '1px solid var(--border-subtle)', borderRadius: 2 }} />
          <span>未出场</span>
        </div>
      </div>
    </div>
  )
}

export default CharacterHeatmap
