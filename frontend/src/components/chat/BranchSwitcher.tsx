import React from 'react'
import { LeftOutlined, RightOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

// BranchSwitcher — 消息版本切换器（A 批，chat-demo BranchSwitcher 同位：
// ‹ i/n ›，busy 全局禁用；版本数 ≤1 不渲染）。
export const BranchSwitcher: React.FC<{
  ids: number[]
  index: number
  disabled: boolean
  onSelect: (id: number) => void
}> = ({ ids, index, disabled, onSelect }) => {
  if (ids.length <= 1) return null
  const btnStyle = (enabled: boolean): React.CSSProperties => ({
    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    border: 'none', background: 'transparent', cursor: enabled ? 'pointer' : 'not-allowed',
    padding: '0 3px', height: 18, borderRadius: 5,
    color: enabled ? C('color-text-secondary') : 'var(--md-sys-color-outline-variant)',
    fontSize: 10,
  })
  return (
    <span className="chat-branch-switcher" role="group" aria-label="消息版本"
      style={{ display: 'inline-flex', alignItems: 'center', gap: 1, userSelect: 'none' }}>
      <button type="button" aria-label="上一版本" disabled={disabled || index <= 0}
        style={btnStyle(!disabled && index > 0)}
        onClick={() => onSelect(ids[index - 1])}>
        <LeftOutlined style={{ fontSize: 9 }} />
      </button>
      <span style={{ color: C('color-text-secondary'), fontSize: 10.5, minWidth: 24, textAlign: 'center' }}>
        {index + 1} / {ids.length}
      </span>
      <button type="button" aria-label="下一版本" disabled={disabled || index >= ids.length - 1}
        style={btnStyle(!disabled && index < ids.length - 1)}
        onClick={() => onSelect(ids[index + 1])}>
        <RightOutlined style={{ fontSize: 9 }} />
      </button>
    </span>
  )
}
