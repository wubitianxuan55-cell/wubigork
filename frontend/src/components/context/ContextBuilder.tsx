import React, { useState, useEffect, useCallback, useRef } from 'react'
import { Typography, Tag, Input, Space, Button, Spin, Empty } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import TokenBudget from './TokenBudget'
import type { BudgetSection } from './TokenBudget'

/**
 * ContextBuilder — @-mention 上下文构建器
 *
 * 允许用户通过 @ 符号引用实体（角色/地点/概念等）构建 AI 上下文
 * 显示 token 预算可视化
 *
 * Props:
 *   onContextReady — 上下文构建完成回调 (systemPrompt, userPrompt) => void
 *   defaultSystemPrompt — 默认系统提示
 */
interface EntityRef {
  name: string
  type: string
  id: string
}

interface ContextBuilderProps {
  onContextReady?: (systemPrompt: string, userPrompt: string, budgetInfo: any) => void
  defaultSystemPrompt?: string
}

const ENTITY_ICONS: Record<string, string> = {
  character: '👤',
  organization: '🏛️',
  location: '📍',
  item: '🔧',
  concept: '💡',
}

const ContextBuilder: React.FC<ContextBuilderProps> = ({ onContextReady, defaultSystemPrompt }) => {
  const [systemPrompt, setSystemPrompt] = useState(defaultSystemPrompt || '')
  const [userText, setUserText] = useState('')
  const [showMention, setShowMention] = useState(false)
  const [mentionFilter, setMentionFilter] = useState('')
  const [entities, setEntities] = useState<EntityRef[]>([])
  const [selectedEntities, setSelectedEntities] = useState<EntityRef[]>([])
  const [loading, setLoading] = useState(false)
  const [budget, setBudget] = useState<{
    capacity: number; used: number; remaining: number;
    usagePercent: number; sections: BudgetSection[];
  } | null>(null)

  const inputRef = useRef<any>(null)

  // 加载实体列表
  useEffect(() => {
    const loadEntities = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.GetAllEntityNames()
        setEntities(Array.isArray(result) ? result : [])
      } catch (_) {
        setEntities([])
      } finally {
        setLoading(false)
      }
    }
    loadEntities()
  }, [])

  // 监听 @ 输入
  const handleUserTextChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value
    setUserText(val)

    // 检测 @ 符号
    const cursorPos = e.target.selectionStart || 0
    const textBeforeCursor = val.slice(0, cursorPos)
    const lastAt = textBeforeCursor.lastIndexOf('@')

    if (lastAt >= 0 && (lastAt === 0 || textBeforeCursor[lastAt - 1]?.match(/\s/))) {
      const filter = textBeforeCursor.slice(lastAt + 1)
      setMentionFilter(filter)
      setShowMention(true)
    } else {
      setShowMention(false)
    }
  }, [])

  // 选择实体
  const selectEntity = useCallback((entity: EntityRef) => {
    setSelectedEntities(prev => {
      if (prev.find(e => e.id === entity.id)) return prev
      return [...prev, entity]
    })

    // 替换 @mention 为实体名
    const cursorPos = inputRef.current?.selectionStart || userText.length
    const textBefore = userText.slice(0, cursorPos)
    const textAfter = userText.slice(cursorPos)
    const lastAt = textBefore.lastIndexOf('@')
    if (lastAt >= 0) {
      const newText = textBefore.slice(0, lastAt) + entity.name + ' ' + textAfter
      setUserText(newText)
    }

    setShowMention(false)
    setTimeout(() => inputRef.current?.focus(), 50)
  }, [userText])

  // 移除已选实体
  const removeEntity = useCallback((id: string) => {
    setSelectedEntities(prev => prev.filter(e => e.id !== id))
  }, [])

  // 构建上下文
  const handleBuildContext = useCallback(async () => {
    try {
      const contextText = userText + '\n' + selectedEntities.map(e => `@${e.name}`).join(' ')
      // @ts-ignore
      const result = await window.go.app.App.BuildContextBudget(
        systemPrompt,
        contextText,
        '',
        selectedEntities.map(e => `${e.name}[${e.type}]`).join('\n'),
        '',
        128000,
      )
      if (result) {
        setBudget({
          capacity: result.capacity,
          used: result.used,
          remaining: result.remaining,
          usagePercent: result.usage_percent,
          sections: result.sections || [],
        })
        onContextReady?.(result.system_prompt, result.user_prompt, result)
      }
    } catch (err) {
      console.warn('构建上下文失败:', err)
    }
  }, [systemPrompt, userText, selectedEntities, onContextReady])

  // 过滤实体
  const filteredEntities = entities.filter(e =>
    !mentionFilter || e.name.toLowerCase().includes(mentionFilter.toLowerCase()),
  ).slice(0, 8)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {/* 系统提示 */}
      <div>
        <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 4 }}>
          系统提示
        </Typography.Text>
        <Input.TextArea
          value={systemPrompt}
          onChange={e => setSystemPrompt(e.target.value)}
          rows={2}
          placeholder="你是专业小说作家..."
          style={{ fontSize: 12 }}
        />
      </div>

      {/* 用户输入（支持 @mention） */}
      <div style={{ position: 'relative' }}>
        <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 4 }}>
          用户输入（使用 @ 引用实体）
        </Typography.Text>
        <Input.TextArea
          ref={inputRef}
          value={userText}
          onChange={handleUserTextChange}
          rows={4}
          placeholder="请为当前场景生成内容... @"
          style={{ fontSize: 12 }}
        />

        {/* @-mention 弹窗 */}
        {showMention && (
          <div
            style={{
              position: 'absolute',
              bottom: '100%',
              left: 0,
              right: 0,
              background: 'var(--bg-elevated)',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--border-subtle)',
              boxShadow: 'var(--shadow-lg)',
              maxHeight: 200,
              overflow: 'auto',
              zIndex: 100,
            }}
          >
            {loading ? (
              <div style={{ padding: 12, textAlign: 'center' }}>
                <Spin size="small" />
              </div>
            ) : filteredEntities.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无匹配实体" />
            ) : (
              filteredEntities.map(entity => (
                <div
                  key={entity.id}
                  onClick={() => selectEntity(entity)}
                  style={{
                    padding: '6px 10px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    borderBottom: '1px solid var(--border-subtle)',
                    transition: 'background 150ms',
                  }}
                  onMouseEnter={e => {
                    (e.currentTarget as HTMLElement).style.background = 'var(--bg-elevated)'
                  }}
                  onMouseLeave={e => {
                    (e.currentTarget as HTMLElement).style.background = 'transparent'
                  }}
                >
                  <span>{ENTITY_ICONS[entity.type] || '📌'}</span>
                  <Typography.Text style={{ fontSize: 12 }}>{entity.name}</Typography.Text>
                  <Tag style={{ fontSize: 10, padding: '0 4px', margin: '0 0 0 auto' }}>
                    {entity.type}
                  </Tag>
                </div>
              ))
            )}
          </div>
        )}
      </div>

      {/* 已选实体 */}
      {selectedEntities.length > 0 && (
        <div>
          <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 4 }}>
            已引用的实体
          </Typography.Text>
          <Space wrap size={[4, 4]}>
            {selectedEntities.map(entity => (
              <Tag
                key={entity.id}
                closable
                onClose={() => removeEntity(entity.id)}
                style={{ fontSize: 11 }}
              >
                {ENTITY_ICONS[entity.type] || '📌'} {entity.name}
              </Tag>
            ))}
          </Space>
        </div>
      )}

      {/* 构建按钮 */}
      <Button
        type="primary"
        size="small"
        onClick={handleBuildContext}
        disabled={!userText.trim()}
        icon={<SearchOutlined />}
      >
        构建上下文
      </Button>

      {/* Token 预算 */}
      {budget && (
        <TokenBudget
          capacity={budget.capacity}
          used={budget.used}
          remaining={budget.remaining}
          usagePercent={budget.usagePercent}
          sections={budget.sections}
        />
      )}
    </div>
  )
}

export default ContextBuilder
export type { ContextBuilderProps, EntityRef }
