import React, { useCallback, useEffect, useState } from 'react'
import { AIConsole } from '../components/novel/AIConsole'
import NovelSidebar from '../components/novel/NovelSidebar'
import NovelInspector from '../components/novel/NovelInspector'
import {
  HomeOutlined, FileTextOutlined, UserOutlined,
  ThunderboltOutlined, BookOutlined, ExportOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { useOutlineStore } from '../stores/outlineStore'
import { sortNodes } from '../utils/outline'
import * as App from '../../src/wailsjsCompat'
import '../novel-workspace.css'
import type { OutlineNode } from '../types'

const HomePage = React.lazy(() => import('./HomePage'))
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage'))
const CharacterPage = React.lazy(() => import('./CharacterPage'))
const CreatePage = React.lazy(() => import('./CreatePage'))
const ChapterPage = React.lazy(() => import('./ChapterPage'))
const ExportPage = React.lazy(() => import('./ExportPage'))

export type NovelTab = 'home' | 'novelsetting' | 'character' | 'create' | 'chapter' | 'export'
const NOVEL_TAB_KEY = 'gaea.novel.activeTab'
const NOVEL_SIDE_KEY = 'gaea.novel.sideCollapsed'
const NOVEL_INSPECTOR_KEY = 'gaea.novel.inspectorCollapsed'

function loadActiveTab(): NovelTab {
  try {
    const v = localStorage.getItem(NOVEL_TAB_KEY)
    if (v === 'home' || v === 'novelsetting' || v === 'character' || v === 'create' || v === 'chapter' || v === 'export') {
      return v
    }
  } catch { /* ignore */ }
  return 'home'
}

function loadCollapsed(key: string): boolean {
  try { return localStorage.getItem(key) === '1' } catch { return false }
}

const tabItems = [
  { key: 'home', icon: <HomeOutlined />, label: '书架', component: HomePage },
  { key: 'novelsetting', icon: <FileTextOutlined />, label: '设定', component: NovelSettingPage },
  { key: 'character', icon: <UserOutlined />, label: '角色', component: CharacterPage },
  { key: 'create', icon: <ThunderboltOutlined />, label: '创作', component: CreatePage },
  { key: 'chapter', icon: <BookOutlined />, label: '阅读', component: ChapterPage },
  { key: 'export', icon: <ExportOutlined />, label: '导出', component: ExportPage },
] as const

/** 世界构建工作台：轨道式细条子导航 + 3 分区（侧栏 / 主视图 / 属性 inspector） */
const NovelPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<NovelTab>(loadActiveTab)
  const [sideCollapsed, setSideCollapsed] = useState<boolean>(() => loadCollapsed(NOVEL_SIDE_KEY))
  const [inspectorCollapsed, setInspectorCollapsed] = useState<boolean>(() => loadCollapsed(NOVEL_INSPECTOR_KEY))
  const [activeChapterId, setActiveChapterId] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)
  const [focusMode, setFocusMode] = useState(false)

  const projectPath = useAppStore((s) => s.projectPath)
  const projectTitle = useAppStore((s) => s.projectTitle)
  const outlines = useOutlineStore((s) => s.outlines)
  const loadOutlines = useOutlineStore((s) => s.loadOutlines)

  const changeTab = (key: NovelTab) => {
    setActiveTab(key)
    try { localStorage.setItem(NOVEL_TAB_KEY, key) } catch { /* ignore */ }
  }

  const toggleSide = () => {
    setSideCollapsed((p) => {
      const next = !p
      try { localStorage.setItem(NOVEL_SIDE_KEY, next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }
  const toggleInspector = () => {
    setInspectorCollapsed((p) => {
      const next = !p
      try { localStorage.setItem(NOVEL_INSPECTOR_KEY, next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }

  // 项目/大纲变化：刷新大纲与创作统计（侧栏与检查器共用）
  useEffect(() => {
    if (!projectPath) { setStats(null); return }
    void loadOutlines()
    App.GetStats().then((s) => {
      if (s && useAppStore.getState().projectPath === projectPath) {
        setStats(s as { totalWords: number; chapterCount: number })
      }
    }).catch(() => { /* 统计失败不阻塞 */ })
  }, [projectPath, loadOutlines])

  // 阅读页上报当前章节 → 大纲树激活项同步（空 detail 清空激活）
  useEffect(() => {
    const handler = (e: Event) => {
      const id = (e as CustomEvent<{ id?: string }>).detail?.id
      setActiveChapterId(id ?? '')
    }
    window.addEventListener('novel:chapter-active', handler)
    return () => window.removeEventListener('novel:chapter-active', handler)
  }, [])

  // 阅读页专注模式 → 壳层收起左右 zone（沉浸书写）
  useEffect(() => {
    const handler = (e: Event) => {
      const active = (e as CustomEvent<{ active?: boolean }>).detail?.active
      if (typeof active === 'boolean') setFocusMode(active)
    }
    window.addEventListener('novel:focus-mode', handler)
    return () => window.removeEventListener('novel:focus-mode', handler)
  }, [])

  // 书架卡「继续阅读」→ 切到阅读 tab（HomePage 派发 novel:goto-tab）
  useEffect(() => {
    const handler = (e: Event) => {
      const tab = (e as CustomEvent<{ tab?: NovelTab }>).detail?.tab
      if (tab) changeTab(tab)
    }
    window.addEventListener('novel:goto-tab', handler)
    return () => window.removeEventListener('novel:goto-tab', handler)
  }, [])

  // 侧栏大纲点击 → 切到阅读 tab 并定位章节（ChapterPage 监听 novel:open-chapter）
  const handleOpenChapter = useCallback((node: OutlineNode) => {
    changeTab('chapter')
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node } }))
  }, [])

  const sortedOutlines = React.useMemo(() => sortNodes(outlines), [outlines])

  return (
    <div className="novel-hub" data-novel-tab={activeTab} data-novel-focus={focusMode ? '1' : '0'}>
      {/* 轨道式细条子导航（对齐 v3 轨道语言：激活 = 主色容器 + 光条） */}
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

      {/* 3 分区工作台：侧栏 zone | 主视图 zone | 属性 inspector zone */}
      <div className="novel-workspace">
        <NovelSidebar
          outlines={sortedOutlines}
          activeKey={activeChapterId}
          collapsed={sideCollapsed}
          onToggleCollapse={toggleSide}
          onOpenChapter={handleOpenChapter}
          onGoBookshelf={() => changeTab('home')}
          projectTitle={projectTitle}
          projectPath={projectPath}
          stats={stats}
        />
        <div className="v3-grip" aria-hidden="true" />

        <main className="v3-zone novel-main-zone">
          {/* 各子页保持挂载，按需显示（切换不丢失状态） */}
          {tabItems.map((t) => (
            <div key={t.key} style={{ display: activeTab === t.key ? 'flex' : 'none', flex: 1, minWidth: 0, minHeight: 0 }}>
              <React.Suspense fallback={null}><t.component /></React.Suspense>
            </div>
          ))}
        </main>

        <div className="v3-grip" aria-hidden="true" />
        <NovelInspector
          activeTab={activeTab}
          collapsed={inspectorCollapsed}
          onToggleCollapse={toggleInspector}
          onNavigate={changeTab}
          stats={stats}
        />
      </div>

      {/* 小说专属：AI 控制台（右上角悬浮） */}
      <AIConsole />
    </div>
  )
}

export default NovelPage
