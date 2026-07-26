import React, { useState, useCallback, useEffect, useRef } from 'react'
import { Button, Space, Typography, Card, Tag, message, Spin, Tooltip } from 'antd'
import {
  ThunderboltOutlined, PlusOutlined, DeleteOutlined,
  PlayCircleOutlined, EditOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'

/**
 * BeatToProse — Beat-to-Prose 两栏面板
 *
 * 左栏: Beat 卡片列表（可编辑、拖拽重排、删除、新增）
 * 右栏: 选中 Beat 的 AI 生成 Prose（流式展示）
 *
 * Props:
 *   chapterNum — 当前章节号
 *   outlineNodeID — 大纲节点 ID（用于生成 Beats）
 *   onProseGenerated — prose 生成完成回调 (beatID, prose) => void
 */
interface BeatToProseProps {
  chapterNum: number
  outlineNodeID: string
  onProseGenerated?: (beatID: string, prose: string) => void
}

interface Beat {
  id: string
  description: string
  order: number
}

const BeatToProse: React.FC<BeatToProseProps> = ({ chapterNum, outlineNodeID, onProseGenerated }) => {
  const [beats, setBeats] = useState<Beat[]>([])
  const [selectedBeatID, setSelectedBeatID] = useState<string>('')
  const [generatingBeats, setGeneratingBeats] = useState(false)
  const [generatingProse, setGeneratingProse] = useState(false)
  const [prose, setProse] = useState('')
  const [editingBeatID, setEditingBeatID] = useState<string>('')
  const [editText, setEditText] = useState('')
  const proseRef = useRef<HTMLDivElement>(null)

  // 监听 beat-prose-stream 事件
  useEffect(() => {
    const handler = (ev: any) => {
      if (!ev?.type) return

      if (ev.type === 'chunk' && ev.beatID === selectedBeatID) {
        setProse(prev => prev + (ev.content || ''))
      } else if (ev.type === 'done' && ev.beatID === selectedBeatID) {
        setGeneratingProse(false)
        const finalProse = ev.content || prose
        if (finalProse && onProseGenerated) {
          onProseGenerated(ev.beatID, finalProse)
        }
        message.success('段落生成完成')
      } else if (ev.type === 'error') {
        setGeneratingProse(false)
        message.error(ev.error || '生成失败')
      }
    }

    try {
      // @ts-ignore
      window.runtime?.EventsOn?.('beat-prose-stream', handler)
    } catch (_) {}

    return () => {
      try {
        // @ts-ignore
        window.runtime?.EventsOff?.('beat-prose-stream', handler)
      } catch (_) {}
    }
  }, [selectedBeatID, prose, onProseGenerated])

  // 生成 Beats
  const handleGenerateBeats = useCallback(async () => {
    if (!outlineNodeID) {
      message.warning('请先选择大纲节点')
      return
    }

    setGeneratingBeats(true)
    try {
      // @ts-ignore
      const result = await window.go.app.App.GenerateBeats(outlineNodeID)
      if (Array.isArray(result)) {
        setBeats(result)
        if (result.length > 0) {
          setSelectedBeatID(result[0].id)
        }
        message.success(`生成了 ${result.length} 个节拍`)
      }
    } catch (err: any) {
      message.error(err?.message || '生成节拍失败')
    } finally {
      setGeneratingBeats(false)
    }
  }, [outlineNodeID])

  // 生成当前 Beat 的 Prose
  const handleGenerateProse = useCallback(async () => {
    if (!selectedBeatID || beats.length === 0) return

    setGeneratingProse(true)
    setProse('')

    try {
      // @ts-ignore
      await window.go.app.App.GenerateProseFromBeat(
        selectedBeatID,
        JSON.stringify(beats),
        chapterNum,
      )
    } catch (err: any) {
      setGeneratingProse(false)
      message.error(err?.message || '生成段落失败')
    }
  }, [selectedBeatID, beats, chapterNum])

  // 添加 Beat
  const handleAddBeat = useCallback(() => {
    const newBeat: Beat = {
      id: `beat-${Date.now()}`,
      description: '新节拍',
      order: beats.length + 1,
    }
    setBeats(prev => [...prev, newBeat])
    setSelectedBeatID(newBeat.id)
    setEditingBeatID(newBeat.id)
    setEditText('新节拍')
  }, [beats])

  // 删除 Beat
  const handleDeleteBeat = useCallback(
    (id: string) => {
      setBeats(prev => {
        const filtered = prev.filter(b => b.id !== id)
        // 重新编号
        return filtered.map((b, i) => ({ ...b, order: i + 1 }))
      })
      if (selectedBeatID === id) {
        setSelectedBeatID('')
        setProse('')
      }
    },
    [selectedBeatID],
  )

  // 保存编辑
  const handleSaveEdit = useCallback(
    (id: string) => {
      setBeats(prev =>
        prev.map(b => (b.id === id ? { ...b, description: editText } : b)),
      )
      setEditingBeatID('')
    },
    [editText],
  )

  const selectedBeat = beats.find(b => b.id === selectedBeatID)

  return (
    <div style={{ display: 'flex', gap: 16, height: '100%', minHeight: 400 }}>
      {/* ── 左栏: Beats 列表 ── */}
      <div
        style={{
          width: 280,
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography.Text strong style={{ fontSize: 13 }}>
            叙事节拍
          </Typography.Text>
          <Space size={4}>
            <Tooltip title="AI 生成节拍">
              <Button
                size="small"
                type="text"
                icon={<ThunderboltOutlined />}
                loading={generatingBeats}
                onClick={handleGenerateBeats}
              />
            </Tooltip>
            <Tooltip title="手动添加">
              <Button
                size="small"
                type="text"
                icon={<PlusOutlined />}
                onClick={handleAddBeat}
              />
            </Tooltip>
          </Space>
        </div>

        {generatingBeats && (
          <div style={{ textAlign: 'center', padding: 16 }}>
            <Spin size="small" />
            <Typography.Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 4 }}>
              AI 正在规划节拍...
            </Typography.Text>
          </div>
        )}

        <div style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
          {beats.map(beat => (
            <Card
              key={beat.id}
              size="small"
              style={{
                cursor: 'pointer',
                border:
                  selectedBeatID === beat.id
                    ? `2px solid ${C('color-accent')}`
                    : '1px solid var(--border-subtle)',
                background:
                  selectedBeatID === beat.id
                    ? 'var(--bg-elevated)'
                    : 'var(--bg-glass)',
                transition: 'all 150ms ease',
              }}
              onClick={() => setSelectedBeatID(beat.id)}
            >
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 4 }}>
                <Tag style={{ fontSize: 10, padding: '0 4px', lineHeight: '16px', margin: 0 }}>
                  {beat.order}
                </Tag>

                {editingBeatID === beat.id ? (
                  <input
                    value={editText}
                    onChange={e => setEditText(e.target.value)}
                    onBlur={() => handleSaveEdit(beat.id)}
                    onKeyDown={e => {
                      if (e.key === 'Enter') handleSaveEdit(beat.id)
                      if (e.key === 'Escape') setEditingBeatID('')
                    }}
                    autoFocus
                    style={{
                      flex: 1,
                      border: 'none',
                      outline: 'none',
                      background: 'transparent',
                      fontSize: 12,
                      color: 'var(--color-text)',
                    }}
                  />
                ) : (
                  <Typography.Text
                    style={{ fontSize: 12, flex: 1, lineHeight: 1.4 }}
                    onDoubleClick={() => {
                      setEditingBeatID(beat.id)
                      setEditText(beat.description)
                    }}
                  >
                    {beat.description}
                  </Typography.Text>
                )}

                <Space size={0} style={{ flexShrink: 0, opacity: 0.5 }}>
                  <Button
                    size="small"
                    type="text"
                    icon={<EditOutlined style={{ fontSize: 10 }} />}
                    onClick={e => {
                      e.stopPropagation()
                      setEditingBeatID(beat.id)
                      setEditText(beat.description)
                    }}
                  />
                  <Button
                    size="small"
                    type="text"
                    icon={<DeleteOutlined style={{ fontSize: 10 }} />}
                    onClick={e => {
                      e.stopPropagation()
                      handleDeleteBeat(beat.id)
                    }}
                  />
                </Space>
              </div>
            </Card>
          ))}

          {beats.length === 0 && !generatingBeats && (
            <div style={{ textAlign: 'center', padding: 32, opacity: 0.5 }}>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                点击 ⚡ 生成叙事节拍
                <br />
                或 + 手动添加
              </Typography.Text>
            </div>
          )}
        </div>
      </div>

      {/* ── 右栏: Prose 输出 ── */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--bg-glass)',
          borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--border-subtle)',
          overflow: 'hidden',
        }}
      >
        {/* 工具栏 */}
        <div
          style={{
            padding: '8px 12px',
            borderBottom: '1px solid var(--border-subtle)',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Typography.Text style={{ fontSize: 12, opacity: 0.7 }}>
            {selectedBeat
              ? `节拍 ${selectedBeat.order}: ${selectedBeat.description.slice(0, 40)}${selectedBeat.description.length > 40 ? '...' : ''}`
              : '选择左侧节拍'}
          </Typography.Text>

          <Button
            size="small"
            type="primary"
            icon={generatingProse ? undefined : <PlayCircleOutlined />}
            loading={generatingProse}
            disabled={!selectedBeatID}
            onClick={handleGenerateProse}
          >
            {generatingProse ? '生成中...' : '生成段落'}
          </Button>
        </div>

        {/* Prose 内容区 */}
        <div
          ref={proseRef}
          style={{
            flex: 1,
            padding: 16,
            overflow: 'auto',
            fontSize: 14,
            lineHeight: 1.8,
            color: 'var(--color-text)',
            whiteSpace: 'pre-wrap',
          }}
        >
          {generatingProse && !prose && (
            <div style={{ textAlign: 'center', padding: 32 }}>
              <Spin />
              <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8, fontSize: 12 }}>
                AI 正在展开当前节拍...
              </Typography.Text>
            </div>
          )}

          {prose && (
            <>
              {prose}
              {generatingProse && (
                <span
                  style={{
                    display: 'inline-block',
                    width: 8,
                    height: 16,
                    background: C('color-accent'),
                    animation: 'blink 1s step-end infinite',
                    verticalAlign: 'text-bottom',
                    marginLeft: 2,
                  }}
                />
              )}
            </>
          )}

          {!selectedBeatID && !generatingProse && !prose && (
            <div
              style={{
                textAlign: 'center',
                padding: 48,
                opacity: 0.4,
              }}
            >
              <Typography.Text type="secondary">
                ← 选择或生成叙事节拍开始
              </Typography.Text>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default BeatToProse
export type { BeatToProseProps, Beat }
