import React from 'react'
import { Input, Button, Tag, Space } from 'antd'
import { PictureOutlined } from '@ant-design/icons'

const { TextArea } = Input

interface PromptBarProps {
  prompt: string
  onPromptChange: (v: string) => void
  generating: boolean
  elapsed: number
  onGenerate: () => void
}

/** 快捷风格标签 */
const QUICK_TAGS = [
  '电影级光影', '8K超高清', '概念艺术', '史诗场景',
  '黑暗奇幻', '赛博朋克', '水墨风', '油画质感',
]

/** PromptBar — 沉浸式创作输入卡 */
const PromptBar: React.FC<PromptBarProps> = ({
  prompt, onPromptChange,
  generating, elapsed, onGenerate,
}) => {
  const handleTagClick = (tag: string) => {
    // 如果 tag 已在 prompt 中则跳过
    if (prompt.includes(tag)) return
    const sep = prompt.trim() ? '，' : ''
    onPromptChange(prompt + sep + tag)
  }

  return (
    <div
      style={{
        flexShrink: 0, padding: '16px 0 0',
      }}
    >
      <div
        className="prompt-card"
        style={{
          background: 'rgba(255,255,255,0.04)',
          backdropFilter: 'blur(12px)', WebkitBackdropFilter: 'blur(12px)',
          border: '1px solid rgba(139,92,246,0.25)',
          borderRadius: 16,
          padding: 16,
          transition: 'box-shadow 0.3s, border-color 0.3s',
        }}
        onFocusCapture={() => {
          const el = document.querySelector('.prompt-card') as HTMLElement
          if (el) {
            el.style.borderColor = 'rgba(139,92,246,0.5)'
            el.style.boxShadow = '0 0 20px rgba(139,92,246,0.2)'
          }
        }}
        onBlurCapture={() => {
          const el = document.querySelector('.prompt-card') as HTMLElement
          if (el) {
            el.style.borderColor = 'rgba(139,92,246,0.25)'
            el.style.boxShadow = 'none'
          }
        }}
      >
        {/* 标题 */}
        <div style={{ marginBottom: 10, color: 'var(--color-text-secondary)', fontSize: 11, display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 14 }}>🎨</span>
          描述你心中的画面…
        </div>

        {/* TextArea */}
        <TextArea
          placeholder="悬浮云端的仙侠城市，琉璃瓦宫殿，瀑布倾泻而下，霞光万丈…"
          value={prompt}
          onChange={(e) => onPromptChange(e.target.value)}
          rows={3}
          autoSize={{ minRows: 3, maxRows: 6 }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && !generating) {
              e.preventDefault()
              onGenerate()
            }
          }}
          style={{
            background: 'rgba(0,0,0,0.25)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 12,
            color: 'var(--color-text)',
            resize: 'none',
            fontSize: 14,
            lineHeight: 1.7,
            padding: '12px 14px',
          }}
        />

        {/* 快捷标签 */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 12 }}>
          {QUICK_TAGS.map((tag) => {
            const active = prompt.includes(tag)
            return (
              <Tag
                key={tag}
                onClick={() => handleTagClick(tag)}
                style={{
                  cursor: 'pointer',
                  fontSize: 11,
                  padding: '2px 10px',
                  borderRadius: 8,
                  margin: 0,
                  border: active
                    ? '1px solid rgba(139,92,246,0.5)'
                    : '1px solid rgba(255,255,255,0.08)',
                  background: active
                    ? 'rgba(139,92,246,0.15)'
                    : 'rgba(255,255,255,0.04)',
                  color: active ? 'var(--color-primary)' : 'var(--color-text-secondary)',
                  transition: 'all 0.2s',
                  userSelect: 'none',
                }}
              >
                {tag}
              </Tag>
            )
          })}
        </div>

        {/* 分隔线 */}
        <div style={{ height: 1, background: 'var(--border-subtle)', margin: '12px 0' }} />

        {/* 底部栏：字符计数 + 按钮 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space size={8}>
            <span style={{ fontSize: 11, color: 'var(--color-text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
              ⌨ {prompt.length} 字符
            </span>
            {generating && (
              <Tag color="processing" style={{ margin: 0, fontSize: 10, borderRadius: 6 }}>
                ⏱ {elapsed}s
              </Tag>
            )}
          </Space>
          <Button
            type="primary"
            icon={generating ? undefined : <PictureOutlined />}
            onClick={onGenerate}
            loading={generating}
            style={{
              borderRadius: 10,
              minWidth: 120,
              height: 38,
              fontSize: 14,
              fontWeight: 600,
              background: generating
                ? 'var(--color-primary)'
                : 'linear-gradient(135deg, #8b5cf6, #3b82f6)',
              border: 'none',
              boxShadow: '0 4px 14px rgba(139,92,246,0.3)',
              transition: 'all 0.25s',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.boxShadow = '0 6px 20px rgba(139,92,246,0.4)'
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.boxShadow = '0 4px 14px rgba(139,92,246,0.3)'
            }}
          >
            {generating ? `生成中 ${elapsed}s` : '✨ 生成图像'}
          </Button>
        </div>
      </div>
    </div>
  )
}

export default PromptBar
