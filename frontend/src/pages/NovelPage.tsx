import React, { useCallback, useEffect, useState } from 'react'
import { Modal, message } from 'antd'
import { AIConsole } from '../components/novel/AIConsole'
import NovelSidebar from '../components/novel/NovelSidebar'
import NovelInspector from '../components/novel/NovelInspector'
import {
  HomeOutlined, FileTextOutlined, UserOutlined,
  ThunderboltOutlined, BookOutlined, PictureOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { useOutlineStore } from '../stores/outlineStore'
import { sortNodes } from '../utils/outline'
import { app } from '../gaea/lib/bridge'
import { PortraitImg } from '../components/characterlib/PortraitImg'
import '../novel-workspace.css'
import type { OutlineNode } from '../types'

const HomePage = React.lazy(() => import('./HomePage'))
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage'))
const CharacterPage = React.lazy(() => import('./CharacterPage'))
const CreatePage = React.lazy(() => import('./CreatePage'))
const ChapterPage = React.lazy(() => import('./ChapterPage'))

export type NovelTab = 'home' | 'novelsetting' | 'character' | 'create' | 'chapter'
const NOVEL_TAB_KEY = 'gaea.novel.activeTab'
const NOVEL_SIDE_KEY = 'gaea.novel.sideCollapsed'
const NOVEL_INSPECTOR_KEY = 'gaea.novel.inspectorCollapsed'

function loadActiveTab(): NovelTab {
  try {
    const v = localStorage.getItem(NOVEL_TAB_KEY)
    if (v === 'home' || v === 'novelsetting' || v === 'character' || v === 'create' || v === 'chapter') {
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
] as const

/** 书房工坊：身份头栏 + 模式轨 + 按页显隐的分区工作台 */
const NovelPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<NovelTab>(loadActiveTab)
  const [sideCollapsed, setSideCollapsed] = useState<boolean>(() => loadCollapsed(NOVEL_SIDE_KEY))
  const [inspectorCollapsed, setInspectorCollapsed] = useState<boolean>(() => loadCollapsed(NOVEL_INSPECTOR_KEY))
  const [activeChapterId, setActiveChapterId] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)
  const [focusMode, setFocusMode] = useState(false)
  const [coverBusy, setCoverBusy] = useState(false)
  const [coverPath, setCoverPath] = useState('')
  const [coverPreviewOpen, setCoverPreviewOpen] = useState(false)

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

  useEffect(() => {
    if (!projectPath) { setStats(null); return }
    void loadOutlines()
    app.GetStats().then((s) => {
      if (s && useAppStore.getState().projectPath === projectPath) {
        setStats(s as { totalWords: number; chapterCount: number })
      }
    }).catch(() => { /* 统计失败不阻塞 */ })
  }, [projectPath, loadOutlines])

  useEffect(() => {
    const handler = (e: Event) => {
      const id = (e as CustomEvent<{ id?: string }>).detail?.id
      setActiveChapterId(id ?? '')
    }
    window.addEventListener('novel:chapter-active', handler)
    return () => window.removeEventListener('novel:chapter-active', handler)
  }, [])

  useEffect(() => {
    const handler = (e: Event) => {
      const active = (e as CustomEvent<{ active?: boolean }>).detail?.active
      if (typeof active === 'boolean') setFocusMode(active)
    }
    window.addEventListener('novel:focus-mode', handler)
    return () => window.removeEventListener('novel:focus-mode', handler)
  }, [])

  useEffect(() => {
    const handler = (e: Event) => {
      const tab = (e as CustomEvent<{ tab?: NovelTab }>).detail?.tab
      if (tab) changeTab(tab)
    }
    window.addEventListener('novel:goto-tab', handler)
    return () => window.removeEventListener('novel:goto-tab', handler)
  }, [])

  const handleOpenChapter = useCallback((node: OutlineNode) => {
    changeTab('chapter')
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node } }))
  }, [])

  const sortedOutlines = React.useMemo(() => sortNodes(outlines), [outlines])

  const handleGenerateCover = async () => {
    if (!projectPath || coverBusy) return
    setCoverBusy(true)
    try {
      const path = await app.GenerateBookCover(projectPath, '')
      setCoverPath(path)
      setCoverPreviewOpen(true)
      message.success('封面已生成')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '封面生成失败')
    } finally {
      setCoverBusy(false)
    }
  }

  return (
    <div className="novel-hub" data-novel-tab={activeTab} data-novel-focus={focusMode ? '1' : '0'}>
      <header className="novel-atelier-bar">
        <div className="novel-atelier-identity">
          <span className="novel-atelier-kicker">书房</span>
          <span className="novel-atelier-title">
            {projectTitle || '未打开小说'}
          </span>
          {projectPath && stats ? (
            <span className="novel-atelier-meta">
              {stats.chapterCount} 章 · {stats.totalWords.toLocaleString()} 字
            </span>
          ) : (
            <span className="novel-atelier-meta">打开或新建一部小说</span>
          )}
        </div>

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

        <div className="novel-atelier-actions">
          <button
            type="button"
            className="novel-atelier-iconbtn"
            disabled={!projectPath || coverBusy}
            onClick={() => void handleGenerateCover()}
            aria-label="为当前项目生成书封"
            title="为当前项目生成书封（3:4 竖版）"
          >
            <PictureOutlined aria-hidden />
            {coverBusy ? '生成中' : '封面'}
          </button>
        </div>
      </header>

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
        <div className="v3-grip novel-grip-side" aria-hidden="true" />

        <main className="v3-zone novel-main-zone">
          {tabItems.map((t) => (
            <div
              key={t.key}
              className={`novel-tab-pane${activeTab === t.key ? ' is-active' : ''}`}
            >
              <React.Suspense fallback={<div className="novel-tab-skeleton" aria-hidden />}>
                <t.component />
              </React.Suspense>
            </div>
          ))}
        </main>

        <div className="v3-grip novel-grip-inspector" aria-hidden="true" />
        <NovelInspector
          activeTab={activeTab}
          collapsed={inspectorCollapsed}
          onToggleCollapse={toggleInspector}
          onNavigate={changeTab}
          stats={stats}
        />
      </div>

      <AIConsole />

      <Modal
        open={coverPreviewOpen}
        onCancel={() => setCoverPreviewOpen(false)}
        title="书封预览"
        footer={null}
        width={420}
      >
        {coverPath ? (
          <div className="novel-cover-preview">
            <PortraitImg
              src={coverPath}
              alt="书封"
              className="novel-cover-preview-img"
            />
            <div className="novel-cover-preview-path">{coverPath}</div>
          </div>
        ) : null}
      </Modal>
    </div>
  )
}

export default NovelPage
