// 消息行（T-perf 性能优化）：从 MessageList 抽出的单行渲染组件，渲染契约与
// 原实现逐一对应——user/assistant 气泡、流式光标、推理块、错误块与复制/朗读操作。
// 保持「纯展示 + 浅比较友好」：props 全为原始值 / 稳定对象（msg）/ 恒稳回调，
// memo 边界在 MessageList 侧统一施加（React.memo(ChatRow)），流式 chunk 期间
// 非流式行可整行跳过重渲染。
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

export interface ChatRowProps {
  /** 行对应的消息对象：未更新的行保持同一对象引用（updateMessage 仅替换被补丁的行），memo 才能命中 */
  msg: ChatMsg
  /** 展示文本：流式行 = 最新 streamText，其余行 = msg.content */
  text: string
  /** 本行是否为当前流式行（msg.streaming && msg.key === streamKey） */
  isStreaming: boolean
  /** 是否与上一条消息（含窗口外相邻行）角色不同 → 显示 AI 名牌 */
  newGroup: boolean
  mode: string
  companionName: string
  /** 本行是否处于「已复制」态（由 copiedId === msg.key 派生为行级布尔，保证 memo 浅比较只影响相关行） */
  copied: boolean
  /** 本行是否处于「朗读中」态（由 speakingId === msg.key 派生） */
  speaking: boolean
  onCopy: (content: string, id: string) => void
  onSpeak: (content: string, id: string) => void
  onRetry: (msgKey: string) => void
}

export const ChatRow: React.FC<ChatRowProps> = ({
  msg, text, isStreaming, newGroup, mode, companionName,
  copied, speaking, onCopy, onSpeak, onRetry,
}) => {
  const display = text
  if (msg.role === 'user') {
    return (
      <div className="chat-row chat-row-user">
        <div className="chat-user-capsule">{display}</div>
        {msg.content && !msg.streaming && (
          <div className="chat-msg-actions">
            <Tooltip title={copied ? '已复制' : '复制'}>
              <Button type="text" size="small" icon={copied ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
                onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
            </Tooltip>
          </div>
        )}
      </div>
    )
  }
  return (
    <div className="chat-row chat-row-assistant">
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
          <Tooltip title={copied ? '已复制' : '复制'}>
            <Button type="text" size="small" icon={copied ? <CheckOutlined style={{ color: 'var(--md-sys-color-success)' }} /> : <CopyOutlined />}
              onClick={() => onCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
          </Tooltip>
          {!msg.error && (
            <Tooltip title={speaking ? '朗读中…' : '朗读'}>
              <Button type="text" size="small" icon={<SoundOutlined />} loading={speaking}
                onClick={() => onSpeak(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
            </Tooltip>
          )}
        </div>
      )}
    </div>
  )
}
