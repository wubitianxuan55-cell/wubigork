import React from 'react'
import { Tabs, Typography } from 'antd'
import {
  AppstoreOutlined, FolderOpenOutlined, BookOutlined, FileTextOutlined, HeartOutlined,
  SoundOutlined, PictureOutlined, SettingOutlined, InfoCircleOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import AppearancePanel, { DarkModePanel } from '../components/settings/AppearancePanel'
import WorkspacePanel from '../components/settings/WorkspacePanel'
import ProposalPanel from '../components/settings/ProposalPanel'
import VoicePanel from '../components/settings/VoicePanel'
import WhisperPanel from '../components/settings/WhisperPanel'
import ImageGenPanel from '../components/settings/ImageGenPanel'
import OfficePanel from '../components/settings/OfficePanel'
import SystemPanel from '../components/settings/SystemPanel'

/** SettingsPage — 一站式设置中心（整合全部 gaea 设置参数） */
const SettingsPage: React.FC = () => {
  const tabItems = [
    {
      key: 'appearance',
      label: (<span><AppstoreOutlined style={{ marginRight: 6 }} />外观</span>),
      children: (<><AppearancePanel /><DarkModePanel /></>),
    },
    {
      key: 'workspace',
      label: (<span><BookOutlined style={{ marginRight: 6 }} />小说</span>),
      children: (<WorkspacePanel />),
    },
    {
      key: 'voice',
      label: (<span><SoundOutlined style={{ marginRight: 6 }} />语音</span>),
      children: (<VoicePanel />),
    },
    {
      key: 'whisper',
      label: (<span><HeartOutlined style={{ marginRight: 6 }} />轻语</span>),
      children: (<WhisperPanel />),
    },
    {
      key: 'imagegen',
      label: (<span><PictureOutlined style={{ marginRight: 6 }} />绘梦</span>),
      children: (<ImageGenPanel />),
    },
    {
      key: 'proposal',
      label: (<span><FileTextOutlined style={{ marginRight: 6 }} />方案</span>),
      children: (<ProposalPanel />),
    },
    {
      key: 'office',
      label: (<span><SettingOutlined style={{ marginRight: 6 }} />办公</span>),
      children: (<OfficePanel />),
    },
    {
      key: 'system',
      label: (<span><InfoCircleOutlined style={{ marginRight: 6 }} />系统</span>),
      children: (<SystemPanel />),
    },
  ]

  return (
    <div>
      {/* 标题区 */}
      <div style={{ marginBottom: 16 }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: '0 0 4px', display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{
            width: 6, height: 22, borderRadius: 3, display: 'inline-block',
            background: 'var(--gaea-glow)', boxShadow: '0 0 10px var(--gaea-glow)',
          }} />
          设置中心
        </Typography.Title>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
          整合 gaea 全部设置参数 —— 外观 / 小说 / 语音 / 轻语 / 绘梦 / 方案 / 办公 / 系统
        </Typography.Text>
      </div>

      <Tabs
        defaultActiveKey="appearance"
        items={tabItems}
        tabPosition="top"
        size="middle"
        style={{ height: '100%' }}
      />
    </div>
  )
}

export default SettingsPage
