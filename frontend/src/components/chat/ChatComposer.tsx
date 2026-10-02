/* eslint-disable react-refresh/only-export-components -- QUICK_REPLIES 常量导出供右侧 inspector 快捷建议复用 */
// ChatPage 拆分产物：输入岛（快捷回复 + 工具行 + 输入卡，T6-10.1）。
// 3.0 Wave 2 布局重设计（T6-10.2）：工具按钮（搜索/深度思考/语音）从输入卡内
// 移出，独立成上方细工具行（激活态胶囊 + 键盘提示）；输入卡只保留 textarea + 发送。
import React from 'react'
import { Input, Button, Tooltip } from 'antd'
import {
  GlobalOutlined, BulbOutlined, AudioOutlined, StopOutlined, PauseCircleOutlined, SendOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'

// ── 快捷情绪回复（人格模式输入区 chips；导出供右侧 inspector 快捷建议复用） ──
export const QUICK_REPLIES = [
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
  /** FE6-02：麦克风不可用（hook 降级位）——可选、缺省 false ⇒ 既有调用方零变化。
   *  降级时输入框绝不写「正在聆听…请说话」（用户会对着没插好的麦克风说话），
   *  且必须可打字（否则「请输入文字」写在一个 disabled 的框上，是另一种撒谎）。 */
  voiceDegraded?: boolean
  onToggleVoice: () => void
  sending: boolean
  forceSearch: boolean
  onToggleForceSearch: () => void
  thinking: boolean
  onToggleThinking: () => void
  onSend: () => void
  /** P1-5 停止生成（sending 时发送钮变停止） */
  onStop: () => void
  onFillInput: (label: string) => void
}

/** 工具行开关：图标 + 12px 标签，激活 = 主色胶囊（状态三重传达：色/图标/文案）。 */
const ComposerTool: React.FC<{
  active: boolean
  activeColor?: string
  onClick: () => void
  label: string
  icon: React.ReactNode
  title: string
  ariaLabel: string
}> = ({ active, activeColor, onClick, label, icon, title, ariaLabel }) => (
  <Tooltip title={title}>
    <button
      type="button"
      className={`chat-composer-tool${active ? ' is-active' : ''}`}
      aria-pressed={active}
      aria-label={ariaLabel}
      onClick={onClick}
    >
      <span className="chat-composer-tool-icon" style={active && activeColor ? { color: activeColor } : undefined}>{icon}</span>
      {label}
    </button>
  </Tooltip>
)

const ChatComposerInner: React.FC<ChatComposerProps> = ({
  mode, input, onInputChange, onKeyDown, inputRef,
  voiceOn, voiceTranscript, voiceDegraded = false, onToggleVoice, sending,
  forceSearch, onToggleForceSearch, thinking, onToggleThinking, onSend, onStop, onFillInput,
}) => (
  <div className="chat-composer-wrap">
    {mode !== 'plain' && (
      <div className="chat-quick-replies">
        {QUICK_REPLIES.map(q => (
          <button key={q.label} className="chat-quick-chip" onClick={() => onFillInput(q.text)}>{q.label}</button>
        ))}
      </div>
    )}
    {/* 工具行：搜索 / 深度思考 / 语音 + 键盘提示（与输入卡分离，激活态一目了然） */}
    <div className="chat-composer-tools">
      <ComposerTool
        active={forceSearch}
        activeColor="var(--md-sys-color-success)"
        onClick={onToggleForceSearch}
        label="搜索"
        icon={<GlobalOutlined />}
        title={forceSearch ? '联网搜索已开启：回答前先搜索网页' : '联网搜索：开启后回答前先搜索网页'}
        ariaLabel={forceSearch ? '关闭联网搜索' : '开启联网搜索'}
      />
      <ComposerTool
        active={thinking}
        onClick={onToggleThinking}
        label="深度思考"
        icon={<BulbOutlined />}
        title={thinking ? '深度思考已开启（本地模型先思考再回答）' : '深度思考（本地模型先思考再回答）'}
        ariaLabel={thinking ? '关闭深度思考' : '开启深度思考'}
      />
      <ComposerTool
        active={voiceOn}
        onClick={onToggleVoice}
        label="语音"
        icon={voiceOn ? <StopOutlined /> : <AudioOutlined />}
        // FE6-02：降级时不得再称「聆听」（未在采集音频），按钮语义改为结束语音会话
        title={voiceDegraded
          ? '结束语音（麦克风不可用，未在采集音频）'
          : voiceOn ? '结束聆听' : '语音输入（说话识别为文本对话）'}
        ariaLabel={voiceDegraded ? '结束语音' : voiceOn ? '结束聆听' : '语音输入'}
      />
      <span className="chat-composer-hint">Enter 发送 · Shift+Enter 换行</span>
    </div>
    <div className="gaea-glass-shell chat-composer">
      <Input.TextArea
        ref={inputRef}
        value={input}
        onChange={e => onInputChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={voiceDegraded
          ? '麦克风不可用，请输入文字'
          : voiceOn ? (voiceTranscript || '正在聆听…请说话') : '输入消息…'}
        // 降级时放行文字输入（语音会话仍开着，但没有音频可采）
        disabled={sending || (voiceOn && !voiceDegraded)}
        autoSize={{ minRows: 1, maxRows: 6 }}
        className="chat-input-textarea"
        style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 14, lineHeight: 1.6, padding: '6px 2px', boxShadow: 'none' }}
      />
      {sending ? (
        <Tooltip title="停止生成（已生成部分保留）">
          <Button icon={<PauseCircleOutlined />} onClick={onStop} aria-label="停止生成" data-testid="chat-stop"
            style={{ borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }} />
        </Tooltip>
      ) : (
        <Tooltip title="发送 (Enter)">
          <Button type="primary" icon={<SendOutlined />} onClick={onSend} disabled={!input.trim() || (voiceOn && !voiceDegraded)} aria-label="发送"
            style={{ background: input.trim() ? 'var(--md-sys-color-primary)' : C('color-border'), borderColor: 'transparent', borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 40%, transparent)' : 'none', flexShrink: 0 }} />
        </Tooltip>
      )}
    </div>
  </div>
)

// v4.365 性能轮：流式高频重渲染下本组件经 memo 跳过（props 稳定性由 ChatPage useCallback 保证）
export const ChatComposer = React.memo(ChatComposerInner)
