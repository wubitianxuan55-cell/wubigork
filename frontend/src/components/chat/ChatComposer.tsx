// ChatPage 拆分产物：输入岛（快捷回复 + 输入框 + 工具按钮，行为零变化，T6-10.1）
import React from 'react'
import { Input, Button, Tooltip } from 'antd'
import {
  GlobalOutlined, BulbOutlined, AudioOutlined, StopOutlined, SendOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'

// ── 快捷情绪回复（人格模式输入区 chips） ──
const QUICK_REPLIES = [
  { label: '抱抱我', text: '能抱抱我吗，今天有点累' },
  { label: '晚安', text: '晚安，做个好梦' },
  { label: '有点低落', text: '今天心情不太好，陪我聊聊' },
  { label: '分享开心事', text: '告诉你一件开心的事' },
  { label: '深入聊聊', text: '我们来深入聊聊这个话题吧' },
]

export interface ChatComposerProps {
  mode: string
  input: string
  onInputChange: (v: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
  inputRef: React.RefObject<React.ComponentRef<typeof Input.TextArea>>
  voiceOn: boolean
  voiceTranscript: string
  onToggleVoice: () => void
  sending: boolean
  forceSearch: boolean
  onToggleForceSearch: () => void
  thinking: boolean
  onToggleThinking: () => void
  onSend: () => void
  onFillInput: (label: string) => void
}

export const ChatComposer: React.FC<ChatComposerProps> = ({
  mode, input, onInputChange, onKeyDown, inputRef,
  voiceOn, voiceTranscript, onToggleVoice, sending,
  forceSearch, onToggleForceSearch, thinking, onToggleThinking, onSend, onFillInput,
}) => (
  <div className="chat-composer-wrap">
    {mode !== 'plain' && (
      <div className="chat-quick-replies">
        {QUICK_REPLIES.map(q => (
          <button key={q.label} className="chat-quick-chip" onClick={() => onFillInput(q.text)}>{q.label}</button>
        ))}
      </div>
    )}
    <div className="gaea-glass-shell chat-composer">
      <Tooltip title={forceSearch ? '联网搜索已开启：回答前先搜索网页' : '联网搜索：开启后回答前先搜索网页'}>
        <Button type="text" icon={<GlobalOutlined />}
          onClick={onToggleForceSearch}
          style={{ color: forceSearch ? 'var(--md-sys-color-success)' : C('color-text-secondary'), borderRadius: 10, width: 36, height: 36, minWidth: 36, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: forceSearch ? 'color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)' : 'transparent', flexShrink: 0, fontSize: 15 }} />
      </Tooltip>
      <Tooltip title={thinking ? '深度思考已开启（本地模型先思考再回答）' : '深度思考（本地模型先思考再回答）'}>
        <Button type="text" icon={<BulbOutlined />}
          onClick={onToggleThinking}
          style={{ color: thinking ? 'var(--gaea-glow)' : C('color-text-secondary'), borderRadius: 10, width: 36, height: 36, minWidth: 36, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: thinking ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'transparent', flexShrink: 0, fontSize: 15 }} />
      </Tooltip>
      <Tooltip title={voiceOn ? '结束聆听' : '语音输入（说话识别为文本对话）'}>
        <Button type="text" icon={voiceOn ? <StopOutlined /> : <AudioOutlined />}
          onClick={onToggleVoice}
          style={{ color: voiceOn ? 'var(--gaea-glow)' : C('color-text-secondary'), borderRadius: 10, width: 36, height: 36, minWidth: 36, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: voiceOn ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'transparent', flexShrink: 0, fontSize: 15 }} />
      </Tooltip>
      <Input.TextArea
        ref={inputRef}
        value={input}
        onChange={e => onInputChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={voiceOn ? (voiceTranscript || '正在聆听…请说话') : '输入消息，Enter 发送 / Shift+Enter 换行'}
        disabled={sending || voiceOn}
        autoSize={{ minRows: 1, maxRows: 6 }}
        className="chat-input-textarea"
        style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 14, lineHeight: 1.6, padding: '6px 2px', boxShadow: 'none' }}
      />
      <Tooltip title="发送 (Enter)">
        <Button type="primary" icon={<SendOutlined />} onClick={onSend} loading={sending} disabled={!input.trim() || voiceOn}
          style={{ background: input.trim() ? 'var(--md-sys-color-primary)' : C('color-border'), borderColor: 'transparent', borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 40%, transparent)' : 'none', flexShrink: 0 }} />
      </Tooltip>
    </div>
  </div>
)
