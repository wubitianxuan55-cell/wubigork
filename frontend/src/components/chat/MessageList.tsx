// ChatPage 拆分产物：消息流列表（行为零变化，T6-10.1）。
// 纯展示组件：渲染 user/assistant 气泡、流式光标、推理块、错误块与复制/朗读操作。
import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  CopyOutlined, CheckOutlined, SoundOutlined, BulbOutlined,
  ReloadOutlined, CloseCircleOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import ChatMarkdown from '../ChatMarkdown'
import { MarkdownContent } from '../MarkdownContent'
import type { ChatMsg } from '../../pages/chat/types'

export interface MessageListProps {
  messages: ChatMsg[]
  streamKey: string | null
  streamText: string
  mode: string
  companionName: string
  copiedId: string | null
  speakingId: string | null
  onCopy: (content: string, id: string) => void
  onSpeak: (content: string, id: string) => void
  onRetry: (msgKey: string) => void
}

export const MessageList: React.FC<MessageListProps> = ({
  messages, streamKey, streamText, mode, companionName,
  copiedId, speakingId, onCopy, onSpeak, onRetry,
}) => (
  <div className="chat-flow v3-reading">
    {messages.map((msg, idx) => {
      const isUser = msg.role === 'user'
      const isStreaming = msg.streaming && msg.key === streamKey
      const display = isStreaming ? streamText : msg.content
      const prev = messages[idx - 1]
      const newGroup = !prev || prev.role !== msg.role
      return isUser ? (
        <div key={msg.key} className="chat-row chat-row-user">
          <div className="chat-user-capsule">{display}</div>
          {msg.content && !msg.streaming && (
            <div className="chat-msg-actions">
              <Tooltip title={copiedId === msg.key ? '已复制' : '复制'}>
                <Button type="text" size="small" icon={copiedId === msg.key ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
                  onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
              </Tooltip>
            </div>
          )}
        </div>
      ) : (
        <div key={msg.key} className="chat-row chat-row-assistant">
          {newGroup && (
            <div className="chat-assistant-name">
              <span className="ai-dot" />
              {mode === 'plain' ? 'gaea AI 助手' : `${companionName} · AI`}
            </div>
          )}
          {!msg.error && msg.reasoning && (
            <details className="chat-reasoning">
              <summary><BulbOutlined style={{ marginRight: 4 }} />思考过程</summary>
              <div className="chat-reasoning-body">{msg.reasoning}</div>
            </details>
          )}
          {msg.error ? (
            <div className="chat-error-block">
              <CloseCircleOutlined style={{ marginTop: 2, flexShrink: 0 }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                {display}
                <div style={{ marginTop: 8 }}>
                  <Button size="small" icon={<ReloadOutlined />} onClick={() => onRetry(msg.key)}
                    style={{ fontSize: 12, height: 26, borderRadius: 8 }}>
                    重试
                  </Button>
                </div>
              </div>
            </div>
          ) : (
            <div className="chat-assistant-text">
              {isStreaming ? (
                <span className="chat-streaming">
                  {display ? <><MarkdownContent source={display} className="md-content" /><span className="cursor-blink" /></> : <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>}
                </span>
              ) : (
                mode === 'plain'
                  ? <ChatMarkdown text={display} />
                  : <MarkdownContent source={display} className="md-content" />
              )}
            </div>
          )}
          {msg.content && !msg.streaming && (
            <div className="chat-msg-actions">
              <Tooltip title={copiedId === msg.key ? '已复制' : '复制'}>
                <Button type="text" size="small" icon={copiedId === msg.key ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
                  onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
              </Tooltip>
              {!msg.error && (
                <Tooltip title={speakingId === msg.key ? '朗读中…' : '朗读'}>
                  <Button type="text" size="small" icon={<SoundOutlined />} loading={speakingId === msg.key}
                    onClick={() => onSpeak(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
                </Tooltip>
              )}
            </div>
          )}
        </div>
      )
    })}
  </div>
)
