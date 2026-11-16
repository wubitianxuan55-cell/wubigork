import React, { useEffect } from 'react'
import { CloseOutlined, LinkOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { ChatSearchReference } from './searchCardModel'

// SearchPanel — 联网搜索来源抽屉（D 批，chat-demo SearchPanel 交互对齐：
// 单入口开合、换会话自动关、Esc 关闭）。DeepSeek 正文角标不可复刻（我们的
// 搜索协议无 in-text 引用位置），本件是「来源汇总条 → 全量来源面板」两件套
// 的面板侧：全量展示 extra.search.references（Go 侧已按 URL 去重、容量封顶）。

function safeExternalUrl(raw: string): string | null {
  try {
    const parsed = new URL(raw)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : null
  } catch {
    return null
  }
}

function hostLabel(raw: string): string {
  try {
    return new URL(raw).hostname.replace(/^www\./i, '')
  } catch {
    return raw
  }
}

export interface SourcePanelData {
  refs: ChatSearchReference[]
  queries: string[]
}

export const SearchPanel: React.FC<{
  data: SourcePanelData | null
  onClose: () => void
}> = ({ data, onClose }) => {
  // Esc 关闭（面板开启期间挂 document 级监听）
  useEffect(() => {
    if (!data) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [data, onClose])

  if (!data) return null
  const items = data.refs
    .map(ref => ({ ...ref, safeUrl: safeExternalUrl(ref.url) }))
    .filter((ref): ref is ChatSearchReference & { safeUrl: string } => Boolean(ref.safeUrl))

  return (
    <aside className="chat-source-panel" role="complementary" aria-label="搜索来源"
      style={{
        position: 'absolute', top: 0, right: 0, bottom: 0, width: 340, maxWidth: '85%',
        zIndex: 30, display: 'flex', flexDirection: 'column',
        background: C('color-bg-container'),
        borderLeft: '1px solid var(--md-sys-color-outline-variant)',
        boxShadow: '-12px 0 32px rgba(0,0,0,0.12)',
      }}>
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '12px 14px', borderBottom: '1px solid var(--md-sys-color-outline-variant)',
      }}>
        <strong style={{ fontSize: 13, color: C('color-text') }}>
          搜索来源{items.length > 0 ? `（${items.length}）` : ''}
        </strong>
        <button type="button" aria-label="关闭来源面板" onClick={onClose}
          style={{
            border: 'none', background: 'transparent', cursor: 'pointer',
            color: C('color-text-secondary'), padding: 4, borderRadius: 6,
            display: 'inline-flex', alignItems: 'center',
          }}>
          <CloseOutlined style={{ fontSize: 12 }} />
        </button>
      </div>
      {data.queries.length > 0 && (
        <div style={{ padding: '8px 14px 0', display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {data.queries.map(q => (
            <span key={q} style={{
              fontSize: 10.5, padding: '2px 8px', borderRadius: 999,
              background: 'var(--md-sys-color-surface-variant)', color: C('color-text-secondary'),
            }}>{q}</span>
          ))}
        </div>
      )}
      <div style={{ flex: 1, overflow: 'auto', padding: '10px 12px 16px', display: 'flex', flexDirection: 'column', gap: 8 }}>
        {items.length === 0 ? (
          <p style={{ fontSize: 12, color: C('color-text-secondary'), textAlign: 'center', marginTop: 32 }}>
            本轮没有可展示的来源
          </p>
        ) : items.map((ref, index) => (
          <a key={`${ref.safeUrl}-${index}`} href={ref.safeUrl} target="_blank" rel="noreferrer"
            className="chat-source-item"
            style={{
              display: 'block', padding: '10px 12px', borderRadius: 10, textDecoration: 'none',
              background: 'var(--md-sys-color-surface-variant)', color: C('color-text'),
              border: '1px solid var(--md-sys-color-outline-variant)',
            }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 10.5, color: C('color-text-secondary') }}>
              <span style={{
                width: 16, height: 16, borderRadius: 5, flexShrink: 0,
                display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                background: 'var(--md-sys-color-surface-container-highest, rgba(0,0,0,0.06))',
                fontWeight: 600,
              }}>{index + 1}</span>
              <LinkOutlined aria-hidden style={{ fontSize: 10 }} />
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {ref.source || hostLabel(ref.url)}
              </span>
            </div>
            <div style={{ fontSize: 12.5, fontWeight: 600, marginTop: 4, lineHeight: 1.45 }}>
              {ref.title || hostLabel(ref.url)}
            </div>
            {ref.snippet && (
              <div style={{ fontSize: 11.5, color: C('color-text-secondary'), marginTop: 4, lineHeight: 1.5 }}>
                {ref.snippet}
              </div>
            )}
          </a>
        ))}
      </div>
    </aside>
  )
}
