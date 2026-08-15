// ChatPage 拆分产物：顶部模式切换条（行为零变化，T6-10.1）
import React from 'react'
import { Button, Space, Tooltip, Popconfirm } from 'antd'
import {
  MessageOutlined, HeartOutlined, GlobalOutlined, SettingOutlined,
  SwapOutlined, SoundOutlined, DownloadOutlined, ClearOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import PersonaPicker from '../PersonaPicker'

export interface ChatModeBarProps {
  mode: string
  personaLabel: string
  currentPersonalityLabel?: string
  personaPickerActiveId: string
  searchEnabled: boolean
  onToggleSearch: () => void
  onNavigateLib: () => void
  onSwitchPlain: () => void
  onSwitchPersona: () => void
  onSwitchPersonality: (id: string) => void
  onOpenVoiceSettings: () => void
  hasMessages: boolean
  onExport: () => void
  onClear: () => void
}

export const ChatModeBar: React.FC<ChatModeBarProps> = ({
  mode, personaLabel, currentPersonalityLabel, personaPickerActiveId, searchEnabled,
  onToggleSearch, onNavigateLib, onSwitchPlain, onSwitchPersona, onSwitchPersonality,
  onOpenVoiceSettings, hasMessages, onExport, onClear,
}) => (
  <div className="chat-mode-bar">
    <div className="chat-mode-seg" role="tablist" aria-label="对话模式">
      <button role="tab" aria-selected={mode === 'plain'} className={mode === 'plain' ? 'active' : ''}
        onClick={onSwitchPlain}>
        <MessageOutlined style={{ fontSize: 12 }} /> 普通对话
      </button>
      <button role="tab" aria-selected={mode !== 'plain'} className={mode !== 'plain' ? 'active' : ''}
        onClick={onSwitchPersona}>
        <HeartOutlined style={{ fontSize: 12 }} /> 角色 · {mode !== 'plain' ? personaLabel : (currentPersonalityLabel || '人格')}
      </button>
    </div>
    <div style={{ flex: 1 }} />
    <Space size={2}>
      <Tooltip title={searchEnabled ? '联网搜索已开启（自动检测搜索意图）' : '联网搜索已关闭'}>
        <Button type="text" size="small" icon={<GlobalOutlined style={{ color: searchEnabled ? 'var(--md-sys-color-success)' : C('color-text-secondary') }} />}
          onClick={onToggleSearch} style={{ padding: '0 4px', height: 24, opacity: searchEnabled ? 1 : 0.5 }} />
      </Tooltip>
      {mode !== 'plain' && (
        <>
          <Tooltip title="角色库管理">
            <Button type="text" size="small" icon={<SettingOutlined />} onClick={onNavigateLib}
              style={{ color: C('color-text-secondary'), height: 24 }} />
          </Tooltip>
          <PersonaPicker activeId={personaPickerActiveId}
            onSelect={onSwitchPersonality} onManage={onNavigateLib}>
            <Tooltip title="切换角色">
              <Button type="text" size="small" icon={<SwapOutlined />}
                style={{ color: C('color-text-secondary'), height: 24 }} />
            </Tooltip>
          </PersonaPicker>
          <Tooltip title="语音设置">
            <Button type="text" size="small" icon={<SoundOutlined />} onClick={onOpenVoiceSettings}
              style={{ color: C('color-text-secondary'), height: 24 }} />
          </Tooltip>
        </>
      )}
      {hasMessages && (
        <Tooltip title="导出为 Markdown">
          <Button type="text" size="small" icon={<DownloadOutlined />} onClick={onExport}
            style={{ color: C('color-text-secondary'), height: 24 }} />
        </Tooltip>
      )}
      <Tooltip title="清空当前对话">
        <Popconfirm title="确定清空当前对话？此操作不可恢复" onConfirm={onClear} okText="清空" cancelText="取消">
          <Button type="text" size="small" icon={<ClearOutlined />} disabled={!hasMessages}
            style={{ color: C('color-text-secondary'), height: 24, opacity: hasMessages ? 1 : 0.35 }} />
        </Popconfirm>
      </Tooltip>
    </Space>
  </div>
)
