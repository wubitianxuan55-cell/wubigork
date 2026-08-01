import React, { useState, useEffect, useRef, Suspense } from 'react'
import { Layout, Button, Space, Typography, Tooltip, Spin, Progress, Breadcrumb, Tag } from 'antd'
import {
  HomeOutlined,
  SunOutlined, MoonOutlined, SearchOutlined, SettingOutlined, LoginOutlined, ConsoleSqlOutlined,
  FileTextOutlined, EditOutlined, TeamOutlined, EyeOutlined,
  BarChartOutlined, DownOutlined,
} from '@ant-design/icons'
import SearchModal from '../components/SearchModal'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { Z_INDEX } from '../utils/zIndex'
import { useAppStore, type ThemePreset, type StatsData, type ProjectInfo } from '../stores/appStore'
import ModuleLauncher, { type LauncherTarget } from '../components/ModuleLauncher'
const NovelPage = React.lazy(() => import('../pages/NovelPage'))
const SettingsPage = React.lazy(() => import('../pages/SettingsPage'))
const ImageGenPage = React.lazy(() => import('../pages/ImageGenPage'))
const ModelCenterPage = React.lazy(() => import('../pages/ModelCenterPage'))
const ChatPage = React.lazy(() => import('../pages/ChatPage'))
const WhisperPage = React.lazy(() => import('../pages/WhisperPage'))
const OfficePage = React.lazy(() => import('../pages/OfficePage'))
const GaeaPage = React.lazy(() => import('../pages/GaeaPage'))
const { Header, Footer, Content } = Layout

type Page = 'home' | 'novel' | 'imagegen' | 'settings' | 'modelcenter' | 'chat' | 'whisper' | 'office' | 'gaea'

// 功能模块 key（navigate 事件校验 + Ctrl+1~4 快捷键映射；home 启动器不参与）
const allPageKeys: Page[] = ['chat', 'novel', 'imagegen', 'whisper', 'office', 'gaea', 'modelcenter']

const pageComponents: Record<Exclude<Page, 'home'>, React.ReactNode> = {
  novel: <NovelPage />,
  imagegen: <ImageGenPage />,
  settings: <SettingsPage />,
  modelcenter: <ModelCenterPage />,
  chat: <ChatPage />,
  whisper: <WhisperPage />,
  office: <OfficePage />,
  gaea: <GaeaPage />,
}

interface LogEntry {
  id: number; type: string; time: string
  model?: string; content?: string; error?: string; length?: number
  system?: string; user?: string
}

let logId = 0

// 5 色系
const themeDots: Record<ThemePreset, string> = {
  nightJade: '#2dd4bf', nightViolet: '#a78bfa', nightRose: '#fb7185', nightAmber: '#f59e0b', nightMoss: '#84cc16', nightSlate: '#94a3b8',
}
const themeLabels: Record<ThemePreset, string> = {
  nightJade: '暗夜青', nightViolet: '暗夜紫', nightRose: '暗夜玫', nightAmber: '暗夜金', nightMoss: '暗夜苔', nightSlate: '暗夜墨',
}
const themeKeys = ['nightJade', 'nightViolet', 'nightRose', 'nightAmber', 'nightMoss', 'nightSlate'] as ThemePreset[]

function fmtWords(n: number): string {
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return n.toLocaleString()
}

// ─── 底栏状态条 ─────────────────────────────────────────────
const statusBarStyle = {
  display: 'flex' as const, alignItems: 'center' as const, justifyContent: 'space-between' as const,
  padding: '0 16px', height: 32, fontSize: 12,
  background: 'var(--md-sys-color-surface-container)',
  borderTop: '1px solid var(--md-sys-color-outline-variant)',
  color: 'var(--md-sys-color-text-secondary)',
}

const StatusBar: React.FC<{ stats: StatsData | null; info: ProjectInfo | null }> = ({ stats, info }) => {
  if (!info && !stats) return null
  // 计算写作进度：已写章节/总大纲叶子节点（stats.chapterCount 为已有章节，保守取该值）
  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, (stats as any).plannedChapters || 0) : 0
  const writtenChapters = stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  return (
    <div style={statusBarStyle}>
      <Space size={16}>
        {info && <span style={{ color: 'var(--md-sys-color-text)', fontWeight: 500 }}>{info.title}</span>}
        {/* 全书进度条 — 借鉴 Scrivener 写作目标 */}
        {stats && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Progress
              percent={progressPercent}
              size="small"
              showInfo={false}
              style={{ width: 100, minWidth: 60, margin: 0 }}
              strokeColor="var(--md-sys-color-primary)"
              trailColor="var(--md-sys-color-outline-variant)"
            />
            <span style={{ fontSize: 10, whiteSpace: 'nowrap' }}>
              进度 {progressPercent}%
            </span>
          </div>
        )}
        {stats && (
          <>
            <span><FileTextOutlined style={{ marginRight: 4 }} />{stats.chapterCount} 章</span>
            <span><EditOutlined style={{ marginRight: 4 }} />{fmtWords(stats.totalWords)} 字</span>
            <span><BarChartOutlined style={{ marginRight: 4 }} />均 {fmtWords(stats.avgWordsPerChapter)} 字/章</span>
          </>
        )}
      </Space>
      <Space size={16}>
        {stats && (
          <>
            <span><TeamOutlined style={{ marginRight: 4 }} />{stats.characterCount} 角色</span>
            <span>
              <EyeOutlined style={{ marginRight: 4 }} />伏笔 {stats.foreshadowRevealed}/{stats.foreshadowTotal}
              {stats.foreshadowTotal > 0 && ` (${Math.round(stats.foreshadowRate)}%)`}
            </span>
          </>
        )}
      </Space>
    </div>
  )
}

const pageLabels: Record<Page, string> = {
  home: '首页', novel: '小说', imagegen: 'AI 绘梦', settings: '设置', modelcenter: '模型引擎中心', chat: 'AI 聊天', whisper: '轻语', office: '方案编写', gaea: '办公',
}

// ─── 主布局 ─────────────────────────────────────────────────

// ─── 主布局 ─────────────────────────────────────────────────
// ─── 主布局 ─────────────────────────────────────────────────
const MainLayout: React.FC = () => {
  const [page, setPage] = useState<Page>('home')
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [consoleOpen, setConsoleOpen] = useState(true)
  const [expandedLog, setExpandedLog] = useState<number | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const logEnd = useRef<HTMLDivElement>(null)
  const logContainerRef = useRef<HTMLDivElement>(null)
  const {
    loggedIn, login, checkLogin, baseTheme, darkMode, setTheme, toggleDarkMode,
    projectOpen, projectInfo, stats, loadProjectInfo, loadStats,
  } = useAppStore()

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [visitedPages, setVisitedPages] = useState<Set<Page>>(new Set(['home']))

  // 跟踪已访问的页面，避免切换 tab 时销毁组件丢失状态
  React.useEffect(() => {
    setVisitedPages((prev) => {
      if (prev.has(page)) return prev
      const next = new Set(prev)
      next.add(page)
      return next
    })
  }, [page])

  const [activeModel, setActiveModel] = useState('')

  useEffect(() => { checkLogin(); loadActiveModel() }, [])

  const loadActiveModel = async () => {
    try {
      // @ts-ignore
      const model = await window.go.app.App.GetActiveModel()
      setActiveModel(model || '')
    } catch (_) {}
  }

  // 监听后端模型切换事件，实时刷新右上角显示
  useEffect(() => {
    const handler = () => { loadActiveModel() }
    try {
      // @ts-ignore
      window.runtime?.EventsOn?.('model-changed', handler)
    } catch (_) {}
    return () => {
      try {
        // @ts-ignore
        window.runtime?.EventsOff?.('model-changed', handler)
      } catch (_) {}
    }
  }, [])
  useEffect(() => {
    if (projectOpen) {
      loadProjectInfo()
      loadStats()
    }
  }, [projectOpen])
  // 监听跨页面导航事件
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail
      if (detail?.page && allPageKeys.includes(detail.page as Page)) {
        setPage(detail.page as Page)
      }
    }
    window.addEventListener('navigate', handler)
    return () => window.removeEventListener('navigate', handler)
  }, [])

  // 全局快捷键
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const ctrl = e.ctrlKey || e.metaKey
      if (!ctrl) return
      // Ctrl+1~4 切换页面
      if (e.key >= '1' && e.key <= '4') {
        e.preventDefault()
        setPage(allPageKeys[Number(e.key) - 1] as Page)
      }
      // Ctrl+N 新建项目（仅在首页）
      if (e.key === 'n' && !projectOpen) {
        e.preventDefault()
        setPage('novel')
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [projectOpen])
  // 监听 XAI 实时输出事件
  useEffect(() => {
    // @ts-ignore
    if (!window.runtime?.EventsOn) return
    const handler = (ev: any) => {
      if (!ev) return
      const entry: LogEntry = {
        id: ++logId,
        type: ev.type || 'unknown',
        time: new Date().toLocaleTimeString(),
        model: ev.model,
        content: ev.content,
        error: ev.error,
        length: ev.length,
        system: ev.system,
        user: ev.user,
      }
      setLogs((prev) => [...prev.slice(-99), entry])
    }
    // @ts-ignore
    window.runtime.EventsOn('xai-output', handler)
    return () => {
      try {
        // @ts-ignore
        window.runtime?.EventsOff?.('xai-output')
      } catch (_) { /* EventsOff 可能不可用 */ }
    }
  }, [])

  useEffect(() => {
    logContainerRef.current?.scrollTo({ top: logContainerRef.current.scrollHeight, behavior: 'smooth' })
  }, [logs])

  return (
    <Layout style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: 'linear-gradient(180deg, var(--md-sys-color-surface-dim) 0%, var(--md-sys-color-surface) 100%)' }}>
      {/* ═══ 顶栏 ═══ */}
        <Header style={{
          display: 'flex', alignItems: 'center', height: 48, padding: '0 16px',
          background: 'var(--md-sys-color-surface-container)',
          borderBottom: '1px solid var(--md-sys-color-outline-variant)',
          lineHeight: '48px',
          position: 'sticky', top: 0, zIndex: 100,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginRight: 24 }}>
            <img src="/favicon.svg" alt="gaea" style={{ width: 26, height: 26 }} />
            <Typography.Text strong style={{
              color: 'var(--md-sys-color-primary)', fontSize: 16,
            }}>
              gaea
            </Typography.Text>
          </div>

          {/* 纯启动器模式：进入模块后靠「返回首页」回到卡片墙 */}
          {page !== 'home' && (
            <Button
              type="text"
              size="small"
              icon={<HomeOutlined />}
              onClick={() => setPage('home')}
              style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 13 }}
            >
              首页
            </Button>
          )}

          <div style={{ flex: 1, minWidth: 0 }} />

          <Space size={6}>
            {/* 4 色块 */}
            {themeKeys.map((t) => {
              const active = t === baseTheme
              return (
                <Tooltip key={t} title={themeLabels[t]}>
                  <span
                    onClick={() => setTheme(t)}
                    style={{
                      width: 18, height: 18, borderRadius: '50%',
                      background: themeDots[t], cursor: 'pointer',
                      border: active ? '2px solid var(--md-sys-color-text)' : '2px solid transparent',
                      opacity: active ? 1 : 0.5,
                      transition: 'opacity 0.15s, border 0.15s',
                    }}
                  />
                </Tooltip>
              )
            })}


            {/* 明/暗切换 */}
            <Tooltip title={darkMode ? '切换亮色' : '切换暗色'}>
              <Button
                type="text"
                size="small"
                icon={darkMode ? <SunOutlined /> : <MoonOutlined />}
                onClick={toggleDarkMode}
                style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 14 }}
              />
            </Tooltip>

            {projectOpen && (
              <Button type="text" size="small" icon={<SearchOutlined />}
                onClick={() => setSearchOpen(true)}
                style={{ color: 'var(--md-sys-color-text-secondary)' }}>
                搜索
              </Button>
            )}
            <Button type="text" size="small" icon={<SettingOutlined />}
              onClick={() => setPage('settings')}
              style={{ color: 'var(--md-sys-color-text-secondary)' }}>
              设置
            </Button>
            {activeModel ? (
              <Tag color="blue" style={{ fontSize: 10, cursor: 'default', margin: 0 }}>
                {activeModel}
              </Tag>
            ) : loggedIn ? (
              <Tag color="green" style={{ fontSize: 10, cursor: 'default' }}>✓ 已登录</Tag>
            ) : (
              <Button type="link" size="small" icon={<LoginOutlined />} onClick={login}
                style={{ color: 'var(--md-sys-color-primary)', fontSize: 12 }}>
                登录 xAI
              </Button>
            )}
          </Space>
        </Header>
      {/* ═══ 主体：面包屑 + 内容 + 右侧 XAI ═══ */}
      {/* ═══ 主体：面包屑 + 内容 + 右侧 XAI ═══ */}
      <Layout style={{ flex: 1, flexDirection: 'row' }}>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* 面包屑导航 */}
          {projectOpen && page !== 'novel' && page !== 'home' && (
            <div style={{
              padding: '6px 16px 0', background: 'var(--color-bg-layout)',
            }}>
              <Breadcrumb
                items={[
                  { title: <a onClick={() => setPage('novel')} style={{ color: 'var(--md-sys-color-text-secondary)', cursor: 'pointer' }}>
                    <HomeOutlined style={{ marginRight: 2 }} />{projectInfo?.title || ''}
                  </a> },
                  { title: <span style={{ color: 'var(--md-sys-color-primary)' }}>{pageLabels[page]}</span> },
                ]}
                style={{ fontSize: 12 }}
              />
            </div>
          )}
          <Content style={{
            padding: page === 'chat' ? 0 : (page === 'home' ? '16px' : '8px 16px 16px'),
            paddingBottom: page === 'chat' || page === 'home' ? 0 : '16px',
            background: page === 'chat' ? 'var(--md-sys-color-surface)' : 'var(--md-sys-color-bg-layout)',
            overflow: page === 'chat' ? 'hidden' : 'auto',
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
          }}>
            <ErrorBoundary>
              <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}><Spin size="large" /></div>}>
                {Array.from(visitedPages).map((p) => (
                  <div key={p} style={{ display: p === page ? 'flex' : 'none', flex: 1, flexDirection: 'column', minHeight: 0 }}>
                    {p === 'home'
                      ? <ModuleLauncher onNavigate={(target: LauncherTarget) => setPage(target as Page)} />
                      : pageComponents[p]}
                  </div>
                ))}
              </Suspense>
            </ErrorBoundary>
          </Content>
        </div>

{consoleOpen && page !== 'home' && page !== 'imagegen' && page !== 'modelcenter' && page !== 'chat' && page !== 'whisper' && page !== 'office' && page !== 'gaea' && (
  <div style={{
    width: 380, flexShrink: 0, alignSelf: 'stretch',
    maxHeight: 'calc(100vh - 80px)',
    margin: '8px 8px 8px 0',
    background: 'var(--md-sys-color-surface-container)',
    border: '1px solid var(--md-sys-color-outline-variant)',
    borderRadius: 'var(--md-sys-radius-lg)',
    boxShadow: 'var(--md-sys-elevation-2)',
    display: 'flex', flexDirection: 'column', fontSize: 11,
    fontFamily: 'system-ui, sans-serif',
    overflow: 'hidden',
    animation: 'slideInRight 0.25s cubic-bezier(0.16, 1, 0.3, 1)',
  }}>
    <div style={{
      padding: '8px 12px',
      borderBottom: '1px solid var(--md-sys-color-outline-variant)',
      display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    }}>
      <Space size={6}>
        <span style={{
          width: 6, height: 6, borderRadius: '50%',
          background: logs.length > 0 ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-text-secondary)',
          display: 'inline-block',
          boxShadow: logs.length > 0 ? '0 0 6px var(--md-sys-color-primary)' : 'none',
        }} />
        <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 10, fontWeight: 500 }}>
          AI 控制台
        </Typography.Text>
      </Space>
      <Space size={4}>
        <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 9, opacity: 0.6 }}>
          {logs.length}
        </Typography.Text>
        <Button type="text" size="small" onClick={() => setConsoleOpen(false)}
          style={{
            color: 'var(--md-sys-color-text-secondary)', fontSize: 12, padding: 0,
            width: 20, height: 20, borderRadius: '50%',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
          ✕
        </Button>
      </Space>
    </div>
    <div ref={logContainerRef} style={{ flex: 1, overflowY: 'scroll', maxHeight: 'calc(100vh - 200px)', padding: '8px 12px' }}>
      {logs.length === 0 ? (
        <div style={{ color: 'var(--md-sys-color-text-secondary)', textAlign: 'center', marginTop: 40, opacity: 0.5 }}>
          <ConsoleSqlOutlined style={{ fontSize: 24, marginBottom: 8 }} />
          <div>等待 AI 调用...</div>
        </div>
      ) : (
        logs.map((l) => {
          const open = expandedLog === l.id
          const tagColor = l.type === 'error' ? 'red' : l.type === 'request' ? 'blue' : l.type === 'response' ? 'green' : 'processing'
          const tagLabel = l.type === 'request' ? 'REQ' : l.type === 'response' ? 'OK' : l.type === 'error' ? 'ERR' : '···'
          const summary = l.type === 'request' ? l.model || ''
            : l.type === 'response' ? `${(l.length || 0).toLocaleString()} 字`
            : l.type === 'error' ? l.error?.slice(0, 80) || ''
            : `+${(l.content?.length || 0)} 字`
          const borderColor = l.type === 'error' ? '#f87171' : l.type === 'request' ? '#60a5fa' : l.type === 'response' ? '#4ade80' : 'transparent'
          return (
            <div key={l.id}>
              <div onClick={() => setExpandedLog(open ? null : l.id)}
                style={{ marginBottom: 1, padding: '3px 6px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, borderLeft: `2px solid ${borderColor}`, background: open ? 'var(--md-sys-color-surface-container-high)' : 'transparent', borderRadius: 2 }}>
                <span style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 9, opacity: 0.5, minWidth: 52 }}>{l.time}</span>
                <Tag color={tagColor} style={{ fontSize: 10, lineHeight: '16px', margin: 0 }}>{tagLabel}</Tag>
                <span style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 10, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{summary}</span>
              </div>
              {open && (
                <div style={{ padding: '6px 8px', marginBottom: 4, marginLeft: 8, background: 'var(--md-sys-color-surface-container-high)', borderRadius: 4, fontSize: 10, overflow: 'auto', borderLeft: '2px solid var(--md-sys-color-outline-variant)' }}>
                  {l.type === 'request' && <>
                    {l.system && <div style={{ marginBottom: 4 }}><div style={{ color: '#60a5fa', fontWeight: 600, marginBottom: 2 }}>SYSTEM:</div><pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text-secondary)', fontFamily: 'monospace', fontSize: 9 }}>{l.system}</pre></div>}
                    {l.user && <div><div style={{ color: '#f59e0b', fontWeight: 600, marginBottom: 2 }}>USER:</div><pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text-secondary)', fontFamily: 'monospace', fontSize: 9 }}>{l.user}</pre></div>}
                  </>}
                  {l.type === 'response' && l.content && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text)', fontFamily: 'monospace', fontSize: 9 }}>{l.content}</pre>}
                  {l.type === 'error' && l.error && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: '#f87171', fontFamily: 'monospace', fontSize: 9 }}>{l.error}</pre>}
                  {l.type === 'chunk' && l.content && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text)', fontFamily: 'monospace', fontSize: 9 }}>{l.content}</pre>}
                </div>
              )}
            </div>
          )
        })
      )}
      <div ref={logEnd} />
    </div>
  </div>
)}

{!consoleOpen && page !== 'modelcenter' && page !== 'home' && (
    <Button
    onClick={() => setConsoleOpen(true)}
    style={{
      position: 'fixed', right: 12, top: 56, zIndex: 1000,
      width: 28, height: 28, borderRadius: '50%',
      background: 'var(--md-sys-color-surface-container)',
      border: '1px solid var(--md-sys-color-outline-variant)',
      color: 'var(--md-sys-color-text-secondary)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      boxShadow: 'var(--md-sys-elevation-1)',
    }}
    title="AI 控制台"
  >
    <ConsoleSqlOutlined style={{ fontSize: 12 }} />
  </Button>
)}
      </Layout>



      {projectOpen && (
        <Footer style={{ padding: 0 }}>
          <StatusBar stats={stats} info={projectInfo} />
        </Footer>
      )}

      {/* 搜索 */}
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} />
    </Layout>
  )
}

export default MainLayout
