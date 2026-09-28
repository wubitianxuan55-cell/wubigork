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

/** 五个子页统一的 pane props（跨线契约，规格线2「常驻页快捷键门控」）。
 *
 *  `active` 语义：**本 pane 是否为当前可见页**（由 NovelPage 按 activeTab 下发）。
 *  为什么需要它：五个 pane 是**常驻挂载**的（下方 map 一次全挂，只靠
 *  `.novel-tab-pane{display:none}` 隐藏），**隐藏 ≠ 卸载**——被隐藏的子页仍在监听
 *  window。于是任何**窗口级副作用**（window keydown、全局自定义事件）都必须按
 *  `active` 门控，否则：F11 在任意子页被隐藏的阅读页 preventDefault 吞掉（浏览器
 *  全屏失效）、一次 Ctrl+S 同时保存设定与阅读页那章（归属不明）、在书架/设定页
 *  点目录树会往隐藏的阅读页塞 tab。
 *
 *  默认 `true`：子页不传也按「当前页」工作，保证既有测试/既有「单独渲染子页」
 *  用法零回归；只有 NovelPage 这个常驻壳层才显式下发。 */
type NovelPaneProps = { active?: boolean }
const HomePage = React.lazy(() => import('./HomePage')) as React.ComponentType<NovelPaneProps>
const NovelSettingPage = React.lazy(() => import('./NovelSettingPage')) as React.ComponentType<NovelPaneProps>
const CharacterPage = React.lazy(() => import('./CharacterPage')) as React.ComponentType<NovelPaneProps>
const CreatePage = React.lazy(() => import('./CreatePage')) as React.ComponentType<NovelPaneProps>
const ChapterPage = React.lazy(() => import('./ChapterPage')) as React.ComponentType<NovelPaneProps>

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

function loadCollapsed(key: string, fallback = false): boolean {
  try {
    const v = localStorage.getItem(key)
    if (v === '1') return true
    if (v === '0') return false
  } catch { /* ignore */ }
  return fallback
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
  const [inspectorCollapsed, setInspectorCollapsed] = useState<boolean>(() => loadCollapsed(NOVEL_INSPECTOR_KEY, true))
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

  // 目录树点章：先切到阅读 tab，**等本次切换提交后**再派发 novel:open-chapter。
  // 为什么不能同 tick 派发：阅读页的全局监听已按 active 门控（`ChapterPage` 的
  // open-chapter handler 只在当前页为阅读 tab 时收下事件），而同一事件处理器里
  // setActiveTab 尚未提交——此刻隐藏的阅读页拿到的 active 仍是 false，同步派发
  // 会被直接丢弃、点目录树失效。改由下方 effect 在 commit 之后派发；每次点击都
  // 塞一个新对象（引用必变），保证「已在该 tab 时再点同一个节点」也会重新派发。
  const [pendingChapter, setPendingChapter] = useState<{ node: OutlineNode } | null>(null)

  const handleOpenChapter = useCallback((node: OutlineNode) => {
    changeTab('chapter')
    setPendingChapter({ node })
  }, [])

  useEffect(() => {
    if (!pendingChapter || activeTab !== 'chapter') return
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: pendingChapter.node } }))
  }, [pendingChapter, activeTab])

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
          projectPath={projectPath}
        />
        <div className="v3-grip novel-grip-side" aria-hidden="true" />

        <main className="v3-zone novel-main-zone">
          {tabItems.map((t) => (
            <div
              key={t.key}
              className={`novel-tab-pane${activeTab === t.key ? ' is-active' : ''}`}
            >
              <React.Suspense fallback={<div className="novel-tab-skeleton" aria-hidden />}>
                {/* active 门控的唯一来源：当前 tab 为 true，其余四个常驻隐藏页为 false */}
                <t.component active={activeTab === t.key} />
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
            <div className="novel-cover-preview-path" title={coverPath}>已保存到小说目录</div>
          </div>
        ) : null}
      </Modal>
    </div>
  )
}

export default NovelPage
