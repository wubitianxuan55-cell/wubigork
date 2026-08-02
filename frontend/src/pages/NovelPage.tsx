import React, { useState } from 'react'
import { Tabs } from 'antd'
import FeatureModelBar from '../components/FeatureModelBar'
import { AIConsole } from '../components/novel/AIConsole'
import {
  HomeOutlined, FileTextOutlined, UserOutlined,
  ThunderboltOutlined, BookOutlined, ExportOutlined,
} from '@ant-design/icons'

const HomePage = React.lazy(() => import('./HomePage'))
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage'))
const CharacterPage = React.lazy(() => import('./CharacterPage'))
const CreatePage = React.lazy(() => import('./CreatePage'))
const ChapterPage = React.lazy(() => import('./ChapterPage'))
const ExportPage = React.lazy(() => import('./ExportPage'))

type NovelTab = 'home' | 'novelsetting' | 'character' | 'create' | 'chapter' | 'export'

const tabItems = [
  { key: 'home', icon: <HomeOutlined />, label: '书架', component: HomePage },
  { key: 'novelsetting', icon: <FileTextOutlined />, label: '设定', component: NovelSettingPage },
  { key: 'character', icon: <UserOutlined />, label: '角色', component: CharacterPage },
  { key: 'create', icon: <ThunderboltOutlined />, label: '创作', component: CreatePage },
  { key: 'chapter', icon: <BookOutlined />, label: '阅读', component: ChapterPage },
  { key: 'export', icon: <ExportOutlined />, label: '导出', component: ExportPage },
] as const

const NovelPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<NovelTab>('home')

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', position: 'relative' }}>
    <Tabs
      activeKey={activeTab}
      onChange={(key) => setActiveTab(key as NovelTab)}
      destroyInactiveTabPane={false}
      items={tabItems.map((t) => ({
        key: t.key,
        label: <span>{t.icon}<span style={{ marginLeft: 6 }}>{t.label}</span></span>,
        children: <React.Suspense fallback={null}><t.component /></React.Suspense>,
      }))}
      style={{ height: '100%' }}
    />
      {/* 绑定模型卡（左下角浮动） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="novel" label="小说" />
      </div>
      {/* 小说专属：AI 控制台（右上角悬浮） */}
      <AIConsole />
    </div>
  )
}

export default NovelPage
