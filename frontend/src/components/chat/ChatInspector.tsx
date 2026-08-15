// ChatPage 拆分产物：对话驾驶舱右侧「上下文 · 人格」inspector（3 分区工作台，v3）
// 纯展示面板：人格卡片 / 当前模型 / 上下文摘要 / 快捷建议 —— 全部取自现有真实状态，
// 无数据时渲染优雅空态，不造假数据；不新增后端调用（当前模型复用 useFeatureModel('chat')，
// 与 FeatureModelBar 同数据源）。可折叠收起（宽度过渡，reduced-motion 下关停）。
import React from 'react'
import { Button, Popconfirm, Tooltip } from 'antd'
import {
  ApiOutlined, AudioOutlined, BulbOutlined, ClearOutlined, ExportOutlined,
  FileTextOutlined, MenuFoldOutlined, MenuUnfoldOutlined, MessageOutlined,
  RobotOutlined, SwapOutlined, TeamOutlined,
} from '@ant-design/icons'
import { useFeatureModel } from '../../hooks/useFeatureModel'
import { CompanionAvatar } from '../CompanionAvatar'
import PersonaPicker from '../PersonaPicker'
import type { ChatMsg, Personality } from '../../pages/chat/types'

export interface ChatInspectorProps {
  mode: string
  personalities: Personality[]
  activePersonality: string
  companionName: string
  emoColor: string
  messages: ChatMsg[]
  speaking: boolean
  thinking: boolean
  quickReplies: Array<{ label: string; text: string }>
  collapsed: boolean
  onToggle: () => void
  onFillInput: (text: string) => void
  onSwitchPersonality: (id: string) => void
  onExport: () => void
  onClear: () => void
  onOpenVoiceSettings: () => void
  onNavigateLib: () => void
}

/** 小节空态：图标 + 主文 + 副文（优雅降级，不造假数据） */
const SectionEmpty: React.FC<{ icon: React.ReactNode; title: string; sub?: string }> = ({ icon, title, sub }) => (
  <div className="chat-inspector-empty">
    <span className="chat-inspector-empty-icon" aria-hidden="true">{icon}</span>
    <span className="chat-inspector-empty-title">{title}</span>
    {sub && <span className="chat-inspector-empty-sub">{sub}</span>}
  </div>
)

export const ChatInspector: React.FC<ChatInspectorProps> = ({
  mode, personalities, activePersonality, companionName, emoColor,
  messages, speaking, thinking, quickReplies, collapsed, onToggle,
  onFillInput, onSwitchPersonality, onExport, onClear, onOpenVoiceSettings, onNavigateLib,
}) => {
  const model = useFeatureModel('chat')
  const persona = mode !== 'plain'
    ? personalities.find(p => p.id === mode) ?? personalities.find(p => p.id === activePersonality)
    : undefined

  // ── 上下文摘要（真实数据，只读统计） ──
  const totalChars = messages.reduce((n, m) => n + (m.content?.length || 0), 0)
  const userCount = messages.filter(m => m.role === 'user').length
  const assistantCount = messages.length - userCount
  const hasMessages = messages.length > 0

  // ── 模型状态（三重传达：色点 + 图标 + 文案） ──
  const bound = !!model.engine && !!model.model
  const modelStatus = !bound ? '未绑定' : model.enabled ? '已启用' : '已停用'
  const modelTone = !bound ? 'warning' : model.enabled ? 'ok' : 'idle'

  if (collapsed) {
    return (
      <div className="chat-inspector v3-panel is-collapsed">
        <div className="chat-inspector-collapsed">
          <Tooltip title="展开上下文面板" placement="left">
            <Button type="text" icon={<MenuUnfoldOutlined />} onClick={onToggle}
              aria-label="展开上下文面板" className="chat-inspector-collapse-btn" />
          </Tooltip>
          <span className="chat-inspector-collapsed-label" aria-hidden="true">上下文 · 人格</span>
        </div>
      </div>
    )
  }

  return (
    <div className="chat-inspector v3-panel">
      <div className="v3-panel-head">
        <span className="v3-panel-title">
          <RobotOutlined aria-hidden="true" style={{ fontSize: 13, color: 'var(--gaea-glow)' }} /> 上下文 · 人格
        </span>
        <span className="v3-panel-spacer" />
        <Tooltip title="折叠上下文面板" placement="left">
          <Button type="text" size="small" icon={<MenuFoldOutlined />} onClick={onToggle}
            aria-label="折叠上下文面板" className="chat-inspector-collapse-btn" />
        </Tooltip>
      </div>

      <div className="chat-inspector-body">
        {/* ── 人格卡片 ── */}
        <section className="chat-inspector-section" aria-label="人格">
          <div className="chat-inspector-section-title">
            <RobotOutlined aria-hidden="true" /> 人格
          </div>
          <div className="v3-card chat-inspector-card">
            {persona ? (
              <>
                <CompanionAvatar size={42} state={speaking ? 'speaking' : thinking ? 'thinking' : 'idle'} emotionColor={emoColor} />
                <div className="chat-inspector-card-main">
                  <div className="chat-inspector-card-title">{companionName}</div>
                  <div className="chat-inspector-card-sub">{persona.label}</div>
                  {persona.tags && persona.tags.length > 0 && (
                    <div className="chat-inspector-tags">
                      {persona.tags.slice(0, 3).map(t => <span key={t} className="chat-inspector-tag">{t}</span>)}
                    </div>
                  )}
                </div>
              </>
            ) : (
              <>
                <span className="chat-inspector-card-ico" aria-hidden="true"><MessageOutlined /></span>
                <div className="chat-inspector-card-main">
                  <div className="chat-inspector-card-title">普通对话</div>
                  <div className="chat-inspector-card-sub">通用 AI 助手 · 未绑定角色</div>
                </div>
              </>
            )}
            <PersonaPicker
              activeId={mode !== 'plain' ? mode : activePersonality}
              onSelect={onSwitchPersonality}
              onManage={onNavigateLib}
              placement="bottomRight"
            >
              <Tooltip title="选择角色">
                <Button type="text" size="small" icon={<SwapOutlined />} aria-label="选择角色"
                  className="chat-inspector-swap" />
              </Tooltip>
            </PersonaPicker>
          </div>
        </section>

        {/* ── 当前模型 ── */}
        <section className="chat-inspector-section" aria-label="当前模型">
          <div className="chat-inspector-section-title">
            <ApiOutlined aria-hidden="true" /> 当前模型
          </div>
          <div className="v3-card chat-inspector-card is-col">
            <div className="chat-inspector-card-head">
              <span className={`chat-inspector-status-dot is-${modelTone}`} aria-hidden="true" />
              <span className="chat-inspector-card-title">{modelStatus}</span>
            </div>
            {bound ? (
              <div className="chat-inspector-meta">
                <div className="chat-inspector-meta-row">
                  <span className="chat-inspector-key">模型</span>
                  <span className="chat-inspector-val" title={model.model}>{model.model}</span>
                </div>
                <div className="chat-inspector-meta-row">
                  <span className="chat-inspector-key">引擎</span>
                  <span className="chat-inspector-val" title={model.engine}>{model.engine}</span>
                </div>
              </div>
            ) : (
              <div className="chat-inspector-empty-sub">尚未绑定聊天模型，请到「模型中心 → 功能模型绑定」配置</div>
            )}
          </div>
        </section>

        {/* ── 上下文摘要 ── */}
        <section className="chat-inspector-section" aria-label="上下文摘要">
          <div className="chat-inspector-section-title">
            <FileTextOutlined aria-hidden="true" /> 上下文摘要
          </div>
          <div className="v3-card chat-inspector-card is-col">
            {!hasMessages ? (
              <SectionEmpty icon={<FileTextOutlined />} title="暂无上下文" sub="发送第一条消息后，这里会显示对话概览" />
            ) : (
              <div className="chat-inspector-meta">
                <div className="chat-inspector-meta-row">
                  <span className="chat-inspector-key">消息</span>
                  <span className="chat-inspector-val">{messages.length} 条（你 {userCount} · AI {assistantCount}）</span>
                </div>
                <div className="chat-inspector-meta-row">
                  <span className="chat-inspector-key">字数</span>
                  <span className="chat-inspector-val">{totalChars} 字</span>
                </div>
                <div className="chat-inspector-meta-row">
                  <span className="chat-inspector-key">状态</span>
                  <span className="chat-inspector-val">{thinking ? 'AI 正在生成…' : speaking ? '朗读中…' : '对话进行中'}</span>
                </div>
              </div>
            )}
          </div>
        </section>

        {/* ── 快捷建议 ── */}
        <section className="chat-inspector-section" aria-label="快捷建议">
          <div className="chat-inspector-section-title">
            <BulbOutlined aria-hidden="true" /> 快捷建议
          </div>
          {quickReplies.length > 0 ? (
            <div className="chat-inspector-chips">
              {quickReplies.map(q => (
                <button key={q.label} type="button" className="chat-inspector-chip" onClick={() => onFillInput(q.text)}>
                  {q.label}
                </button>
              ))}
            </div>
          ) : (
            <div className="chat-inspector-empty-sub">切到角色模式可获得快捷回应建议</div>
          )}
          <div className="chat-inspector-actions">
            <button type="button" className="chat-inspector-chip" onClick={onExport} disabled={!hasMessages}
              aria-label="导出当前会话为 Markdown">
              <ExportOutlined aria-hidden="true" /> 导出
            </button>
            <Popconfirm title="确定清空当前对话？此操作不可恢复" onConfirm={onClear} okText="清空" cancelText="取消">
              <button type="button" className="chat-inspector-chip" disabled={!hasMessages}
                aria-label="清空当前对话">
                <ClearOutlined aria-hidden="true" /> 清空
              </button>
            </Popconfirm>
            <button type="button" className="chat-inspector-chip" onClick={onOpenVoiceSettings}
              aria-label="打开语音设置">
              <AudioOutlined aria-hidden="true" /> 语音设置
            </button>
            <button type="button" className="chat-inspector-chip" onClick={onNavigateLib} aria-label="前往角色库">
              <TeamOutlined aria-hidden="true" /> 角色库
            </button>
          </div>
        </section>
      </div>
    </div>
  )
}

export default ChatInspector
