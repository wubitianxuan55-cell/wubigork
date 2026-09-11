import React, { memo } from 'react'
import ReactMarkdown from 'react-markdown'
import type { Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { C } from '../utils/theme'
import { genuiFenceStateKey, isGenuiFenceLang, GenuiMarkdownFence } from '../genui/markdownFence'
import { useGenuiScope } from '../genui/scope'
import { ChatCodeBlock } from './ChatCodeBlock'

/** ChatMarkdown — 聊天消息 Markdown 渲染（老栈 M3 令牌样式，供 ChatPage 使用） */
const ChatMarkdown: React.FC<{ text: string; genuiKey?: string }> = memo(function ChatMarkdown({ text, genuiKey }) {
  const genuiScope = useGenuiScope()
  const fenceKeyFor =
    genuiScope !== null && genuiKey !== undefined
      ? (body: string): string | undefined => genuiFenceStateKey(genuiScope, genuiKey, body)
      : undefined
  return (
    <div style={{ fontSize: 14, lineHeight: 1.75, wordBreak: 'break-word' }}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={chatMarkdownComponents(fenceKeyFor)}>{text}</ReactMarkdown>
    </div>
  )
})

function chatMarkdownComponents(fenceKeyFor: ((body: string) => string | undefined) | undefined): Components {
return {
  pre: ({ children }) => <>{children}</>,
  code: ({ className, children }) => {
    const text = String(children ?? '').replace(/\n$/, '')
    const match = /language-([\w-]+)/.exec(className ?? '')
    const lang = match?.[1]
    const isBlock = match !== null || text.includes('\n')
    if (lang !== undefined && isGenuiFenceLang(lang) && isBlock) {
      return <GenuiMarkdownFence code={text} stateKey={fenceKeyFor?.(text)} />
    }
    if (isBlock) {
      return <ChatCodeBlock language={lang} text={text} />
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
}

export default ChatMarkdown
