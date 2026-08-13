import React, { useState } from 'react'
import FeatureModelBar from '../components/FeatureModelBar'
import { AIConsole } from '../components/novel/AIConsole'
import {
  HomeOutlined, FileTextOutlined, UserOutlined,
  ThunderboltOutlined, BookOutlined, ExportOutlined,
} from '@ant-design/icons'
import '../novel-workspace.css'

const HomePage = React.lazy(() => import('./HomePage'))
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage'))
const CharacterPage = React.lazy(() => import('./CharacterPage'))
const CreatePage = React.lazy(() => import('./CreatePage'))
const ChapterPage = React.lazy(() => import('./ChapterPage'))
const ExportPage = React.lazy(() => import('./ExportPage'))

type NovelTab = 'home' | 'novelsetting' | 'character' | 'create' | 'chapter' | 'export'
const NOVEL_TAB_KEY = 'gaea.novel.activeTab'

function loadActiveTab(): NovelTab {
  try {
    const v = localStorage.getItem(NOVEL_TAB_KEY)
    if (v === 'home' || v === 'novelsetting' || v === 'character' || v === 'create' || v === 'chapter' || v === 'export') {
      return v
    }
  } catch { /* ignore */ }
  return 'home'
}

const tabItems = [
  { key: 'home', icon: <HomeOutlined />, label: '书架', component: HomePage },
  { key: 'novelsetting', icon: <FileTextOutlined />, label: '设定', component: NovelSettingPage },
  { key: 'character', icon: <UserOutlined />, label: '角色', component: CharacterPage },
  { key: 'create', icon: <ThunderboltOutlined />, label: '创作', component: CreatePage },
  { key: 'chapter', icon: <BookOutlined />, label: '阅读', component: ChapterPage },
  { key: 'export', icon: <ExportOutlined />, label: '导出', component: ExportPage },
] as const

const NovelPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<NovelTab>(loadActiveTab)

  const changeTab = (key: NovelTab) => {
    setActiveTab(key)
    try { localStorage.setItem(NOVEL_TAB_KEY, key) } catch { /* ignore */ }
  }

  return (
    <div className="novel-hub">
      {/* 二级导航：平铺在顶部（与其他功能板块一致） */}
      <nav className="novel-subnav" aria-label="小说板块">
        {tabItems.map((t) => (
          <button
            key={t.key}
            type="button"
            className={`novel-subnav-item${activeTab === t.key ? ' is-active' : ''}`}
            onClick={() => changeTab(t.key)}
            aria-current={activeTab === t.key ? 'page' : undefined}
          >
            {t.icon}
            {t.label}
          </button>
        ))}
      </nav>

      {/* 内容 + AI 控制台：横向布局，控制台在右侧（还原 MainLayout 重构前的边栏设计） */}
      <div style={{ display: 'flex', flexDirection: 'row', flex: 1, minHeight: 0, minWidth: 0 }}>
        {/* 各子页保持挂载，按需显示（切换不丢失状态） */}
        <div className="novel-hub-body">
          {tabItems.map((t) => (
            <div key={t.key} style={{ display: activeTab === t.key ? 'flex' : 'none' }}>
              <React.Suspense fallback={null}><t.component /></React.Suspense>
            </div>
          ))}
        </div>
        {/* 小说专属：AI 控制台（右侧边栏） */}
        <AIConsole />
      </div>
      {/* 绑定模型卡（左下角浮动） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="novel" label="小说" />
      </div>
    </div>
  )
}

export default NovelPage
