import React, { memo, useCallback, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import type { Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CheckOutlined, CopyOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

/** 代码块头部：语言标签 + 复制按钮（对齐行业标准：深色底 + 右上角复制） */
function CodeBlockHeader({ language, text }: { language?: string; text: string }) {
  const [copied, setCopied] = useState(false)
  const copy = useCallback(async () => {
    try { await navigator.clipboard.writeText(text) } catch { /* noop */ }
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }, [text])
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '5px 12px',
      background: 'rgba(0,0,0,0.28)',
      borderBottom: '1px solid rgba(255,255,255,0.08)',
      fontSize: 10.5, userSelect: 'none',
    }}>
      <span style={{ color: C('color-text-secondary'), fontFamily: 'monospace', fontWeight: 500, textTransform: 'uppercase', letterSpacing: 0.5 }}>
        {language || 'text'}
      </span>
      <button
        onClick={copy}
        title="复制代码"
        style={{
          display: 'inline-flex', alignItems: 'center', gap: 4, cursor: 'pointer',
          border: 'none', background: 'transparent', padding: '2px 6px', borderRadius: 6,
          color: copied ? 'var(--md-sys-color-success)' : C('color-text-secondary'), fontSize: 11,
          transition: 'color 0.15s',
        }}
        onMouseEnter={(e) => { if (!copied) e.currentTarget.style.color = C('color-text') }}
        onMouseLeave={(e) => { e.currentTarget.style.color = copied ? 'var(--md-sys-color-success)' : C('color-text-secondary') }}
      >
        {copied ? <CheckOutlined style={{ fontSize: 10 }} /> : <CopyOutlined style={{ fontSize: 10 }} />}
        {copied ? '已复制' : '复制'}
      </button>
    </div>
  )
}

/** ChatMarkdown — 聊天消息 Markdown 渲染（老栈 M3 令牌样式，供 ChatPage 使用） */
const ChatMarkdown: React.FC<{ text: string }> = memo(function ChatMarkdown({ text }) {
  return (
    <div style={{ fontSize: 14, lineHeight: 1.75, wordBreak: 'break-word' }}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>{text}</ReactMarkdown>
    </div>
  )
})

const components: Components = {
  pre: ({ children }) => <>{children}</>,
  code: ({ className, children }) => {
    const text = String(children ?? '').replace(/\n$/, '')
    const match = /language-([\w-]+)/.exec(className ?? '')
    const lang = match?.[1]
    const isBlock = match !== null || text.includes('\n')
    if (isBlock) {
      return (
        /* 代码块专用深色底（#0b0e14/#e2e8f0 为行业标准暗色代码面板，不随主题，属专用色保留） */
        <div style={{ margin: '10px 0', borderRadius: 10, overflow: 'hidden', border: '1px solid rgba(255,255,255,0.09)', background: '#0b0e14' }}>
          <CodeBlockHeader language={lang} text={text} />
          <pre style={{ padding: '10px 12px', margin: 0, overflow: 'auto', fontFamily: "'Cascadia Code', Consolas, monospace", fontSize: 12.5, lineHeight: 1.55, color: '#e2e8f0', whiteSpace: 'pre' }}><code>{text}</code></pre>
        </div>
      )
    }
    return (
      <code style={{
        padding: '1px 5px', borderRadius: 5,
        background: 'var(--md-sys-color-surface-variant)', color: C('color-text'),
        fontFamily: "'Cascadia Code', Consolas, monospace", fontSize: '0.9em',
        border: '1px solid var(--md-sys-color-outline-variant)',
      }}>{children}</code>
    )
  },
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noreferrer" style={{ color: 'var(--gaea-glow, var(--md-sys-color-primary))', textDecoration: 'none' }}>
      {children}
    </a>
  ),
  table: ({ children }) => (
    <div style={{ margin: '10px 0', overflowX: 'auto', borderRadius: 10, border: '1px solid var(--md-sys-color-outline-variant)' }}>
      <table style={{ minWidth: '100%', borderCollapse: 'collapse', fontSize: 13 }}>{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th style={{
      padding: '7px 12px', textAlign: 'left', fontSize: 11.5, fontWeight: 600,
      color: C('color-text-secondary'), background: 'var(--md-sys-color-surface-variant)',
      borderBottom: '1px solid var(--md-sys-color-outline-variant)',
    }}>{children}</th>
  ),
  td: ({ children }) => (
    <td style={{ padding: '7px 12px', borderBottom: '1px solid var(--md-sys-color-outline-variant)', color: C('color-text') }}>{children}</td>
  ),
  blockquote: ({ children }) => (
    <blockquote style={{
      margin: '8px 0', paddingLeft: 12, borderLeft: '3px solid var(--gaea-glow, var(--md-sys-color-primary))',
      color: C('color-text-secondary'), fontStyle: 'italic',
    }}>{children}</blockquote>
  ),
  hr: () => <hr style={{ margin: '14px 0', border: 'none', borderTop: '1px solid var(--md-sys-color-outline-variant)' }} />,
  ol: ({ children }) => <ol style={{ margin: '8px 0', paddingLeft: 22, listStyle: 'decimal' }}>{children}</ol>,
  ul: ({ children }) => <ul style={{ margin: '8px 0', paddingLeft: 22, listStyle: 'disc' }}>{children}</ul>,
  li: ({ children }) => <li style={{ margin: '3px 0' }}>{children}</li>,
  h1: ({ children }) => <h1 style={{ margin: '14px 0 8px', fontSize: 19, fontWeight: 700, color: C('color-text') }}>{children}</h1>,
  h2: ({ children }) => <h2 style={{ margin: '12px 0 7px', fontSize: 17, fontWeight: 700, color: C('color-text') }}>{children}</h2>,
  h3: ({ children }) => <h3 style={{ margin: '10px 0 6px', fontSize: 15, fontWeight: 600, color: C('color-text') }}>{children}</h3>,
  h4: ({ children }) => <h4 style={{ margin: '9px 0 5px', fontSize: 14, fontWeight: 600, color: C('color-text') }}>{children}</h4>,
  p: ({ children }) => <p style={{ margin: '6px 0' }}>{children}</p>,
  strong: ({ children }) => <strong style={{ fontWeight: 700 }}>{children}</strong>,
}

export default ChatMarkdown
