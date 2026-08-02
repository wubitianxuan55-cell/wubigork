import React, { useState, useEffect, useRef, Suspense } from 'react'
import { Layout, Menu, Button, Space, Typography, Tooltip, Spin, Progress, Breadcrumb, Tag, notification } from 'antd'
import {
  HomeOutlined,
  SunOutlined, MoonOutlined, SearchOutlined, SettingOutlined, LoginOutlined, ConsoleSqlOutlined,
  ReadOutlined, PictureOutlined, MessageOutlined, HeartOutlined, ToolOutlined, ApiOutlined,
  FileTextOutlined, EditOutlined, TeamOutlined, EyeOutlined, BookOutlined,
  BarChartOutlined, DownOutlined,
} from '@ant-design/icons'
import SearchModal from '../components/SearchModal'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { Z_INDEX } from '../utils/zIndex'
import { useAppStore, type ThemePreset, type StatsData, type ProjectInfo } from '../stores/appStore'
import ModuleLauncher, { type LauncherTarget } from '../components/ModuleLauncher'
import * as App from '../../wailsjs/go/app/App'
const NovelPage = React.lazy(() => import('../pages/NovelPage'))
const SettingsPage = React.lazy(() => import('../pages/SettingsPage'))
const ImageGenPage = React.lazy(() => import('../pages/ImageGenPage'))
const ModelCenterPage = React.lazy(() => import('../pages/ModelCenterPage'))
const ChatPage = React.lazy(() => import('../pages/ChatPage'))
const WhisperPage = React.lazy(() => import('../pages/WhisperPage'))
const OfficePage = React.lazy(() => import('../pages/OfficePage'))
const GaeaPage = React.lazy(() => import('../pages/GaeaPage'))
const KnowledgePage = React.lazy(() => import('../pages/KnowledgePage'))
const { Header, Footer, Content } = Layout

type Page = 'home' | 'novel' | 'imagegen' | 'settings' | 'modelcenter' | 'chat' | 'whisper' | 'office' | 'gaea' | 'knowledge'

// 功能模块 key（navigate 事件校验 + Ctrl+1~4 快捷键映射；home 启动器不参与）
const allPageKeys: Page[] = ['chat', 'novel', 'imagegen', 'whisper', 'office', 'gaea', 'knowledge', 'modelcenter']

// 顶栏横向导航（含首页启动器），点击直接切换模块
const menuItems: any[] = [
  { key: 'home', icon: <HomeOutlined />, label: '首页' },
  { key: 'chat', icon: <MessageOutlined />, label: '聊天' },
  { key: 'novel', icon: <ReadOutlined />, label: '小说' },
  { key: 'imagegen', icon: <PictureOutlined />, label: '绘梦' },
  { key: 'whisper', icon: <HeartOutlined />, label: '轻语' },
  { key: 'office', icon: <FileTextOutlined />, label: '方案编写' },
  { key: 'gaea', icon: <ToolOutlined />, label: '办公' },
  { key: 'knowledge', icon: <BookOutlined />, label: '知识库' },
  { key: 'modelcenter', icon: <ApiOutlined />, label: '模型中心' },
]

const pageComponents: Record<Exclude<Page, 'home'>, React.ReactNode> = {
  novel: <NovelPage />,
  imagegen: <ImageGenPage />,
  settings: <SettingsPage />,
  modelcenter: <ModelCenterPage />,
  chat: <ChatPage />,
  whisper: <WhisperPage />,
  office: <OfficePage />,
  gaea: <GaeaPage />,
  knowledge: <KnowledgePage />,
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
  background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
  WebkitBackdropFilter: 'blur(16px) saturate(140%)',
  backdropFilter: 'blur(16px) saturate(140%)',
  borderTop: '1px solid var(--md-sys-color-outline-variant)',
  boxShadow: '0 -1px 0 var(--gaea-glow)',
  color: 'var(--md-sys-color-text-secondary)',
}

const StatusBar: React.FC<{ stats: StatsData | null; info: ProjectInfo | null }> = ({ stats, info }) => {
  // 模型监控：已启动引擎 + 系统资源（轮询 3s，防止本地模型加载过多）
  const [monitor, setMonitor] = useState<{ engines: { engine: string; name: string; model: string }[]; stats: any } | null>(null)
  const lastWarn = useRef<Record<string, number>>({})
  useEffect(() => {
    let alive = true
    // 超载警告：CPU/GPU/内存/模型过多，60s 内同类型只弹一次
    const checkOverload = (m: any) => {
      const now = Date.now()
      const ms = m?.stats
      const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
      const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
      const gpuPct = ms?.gpuUsage || vramPct
      // 模型加载预警只统计本地模型（herdsman/ollama 已启用 + ComfyUI 运行中），
      // 云端引擎（xai/deepseek）走 API 不占本机资源，不计入
      const localEngCount = (m?.engines || []).filter((e: any) => e.isLocal).length + (m?.comfyRunning ? 1 : 0)
      const engCount = localEngCount
      const warns: { key: string; title: string; desc: string }[] = []
      if ((ms?.cpu ?? 0) > 85) warns.push({ key: 'cpu', title: '⚠ CPU 负载过高', desc: `当前 CPU 使用率 ${ms.cpu}%，模型占用过大，建议停用部分模型` })
      if (gpuPct > 85) warns.push({ key: 'gpu', title: '⚠ GPU 负载过高', desc: `GPU 使用率 ${gpuPct}%（显存占用），建议停用部分本地模型` })
      if (memPct > 90) warns.push({ key: 'mem', title: '⚠ 内存占用过高', desc: `内存使用率 ${memPct}%，建议释放不用的模型` })
      if (engCount > 3) warns.push({ key: 'models', title: '⚠ 已启动模型过多', desc: `已启用 ${engCount} 个引擎，建议停用不用的模型（各功能窗口 ⚡ 一键启停）` })
      warns.forEach(w => {
        if (now - (lastWarn.current[w.key] || 0) > 60_000) {
          lastWarn.current[w.key] = now
          notification.warning({ message: w.title, description: w.desc, placement: 'topRight', duration: 6 })
        }
      })
    }
    const load = async () => {
      try {
        const m: any = await App.GetModelMonitor()
        if (!alive) return
        setMonitor(m)
        checkOverload(m)
      } catch (_) {}
    }
    load()
    const t = setInterval(load, 3000)
    return () => { alive = false; clearInterval(t) }
  }, [])
  const ms = monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
  const engLabel = (monitor?.engines || []).map(e => `${e.engine}${e.model ? '·' + String(e.model).split('/').pop() : ''}`)
  // 计算写作进度：已写章节/总大纲叶子节点（stats.chapterCount 为已有章节，保守取该值）
  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, (stats as any).plannedChapters || 0) : 0
  const writtenChapters = stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  return (
    <div style={statusBarStyle}>
      <Space size={16}>
        <span className="live-dot" style={{ width: 6, height: 6 }} />
        <span title="已启用引擎（模型中心可启停）" style={{ whiteSpace: 'nowrap' }}>
          🧠 {engLabel.length ? engLabel.join('　') : <span style={{ opacity: 0.5 }}>无启用引擎</span>}
        </span>
        {info && <span style={{ color: 'var(--md-sys-color-text)', fontWeight: 500 }}>{info.title}</span>}
        {/* 全书进度条 — 借鉴 Scrivener 写作目标 */}
        {stats && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Progress
              percent={progressPercent}
              size="small"
              showInfo={false}
              style={{ width: 100, minWidth: 60, margin: 0 }}
              strokeColor="var(--gaea-glow)"
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
        <span title="CPU 使用率">💻 CPU {ms?.cpu ?? '--'}%</span>
        <span title="内存使用率">🧠 内存 {ms ? memPct + '%' : '--'}</span>
        {ms?.gpuName ? (
          <span title={`GPU ${ms.gpuName}`}>🎮 GPU {ms.gpuUsage ? ms.gpuUsage + '%' : vramPct ? vramPct + '%' : '--'}</span>
        ) : null}
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
  home: '首页', novel: '小说', imagegen: 'AI 绘梦', settings: '设置', modelcenter: '模型引擎中心', chat: 'AI 聊天', whisper: '轻语', office: '方案编写', gaea: '办公', knowledge: '知识库',
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
    <Layout style={{ height: '100vh', display: 'flex', flexDirection: 'column', background: 'transparent' }}>
      {/* ── 未来感背景层（星云 + 网格 + 星点，fixed 且不拦截事件）── */}
      <div className="gaea-bg" aria-hidden="true">
        <div className="cyber-grid" />
        <div className="star-dots" />
        <div className="aurora-orb orb-a" />
        <div className="aurora-orb orb-b" />
        <div className="space-dust" />
      </div>
      {/* ═══ 顶栏 ═══ */}
        <Header className="scanline-top" style={{
          display: 'flex', alignItems: 'center', height: 48, padding: '0 16px',
          background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
          WebkitBackdropFilter: 'blur(20px) saturate(140%)',
          backdropFilter: 'blur(20px) saturate(140%)',
          borderBottom: '1px solid var(--md-sys-color-outline-variant)',
          boxShadow: '0 1px 0 var(--gaea-glow)',
          lineHeight: '48px',
          position: 'sticky', top: 0, zIndex: 100,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginRight: 24 }}>
            <img src="/favicon.svg" alt="gaea" style={{
              width: 26, height: 26,
              filter: 'drop-shadow(0 0 6px var(--gaea-glow))',
              transition: 'filter var(--md-sys-transition-normal)',
            }} />
            <Typography.Text strong className="gaea-brand-text" style={{
              color: 'var(--md-sys-color-primary)', fontSize: 16,
            }}>
              gaea
            </Typography.Text>
          </div>

          <Menu
            mode="horizontal"
            selectedKeys={[page]}
            items={menuItems}
            onClick={({ key }) => setPage(key as Page)}
            style={{ flex: 1, minWidth: 0, background: 'transparent', borderBottom: 'none' }}
          />

          <Space size={6}>
            {/* 4 色块 */}
            {themeKeys.map((t) => {
              const active = t === baseTheme
              return (
                <Tooltip key={t} title={themeLabels[t]}>
                  <span
                    className="theme-dot"
                    onClick={() => setTheme(t)}
                    style={{
                      width: 18, height: 18, borderRadius: '50%',
                      background: `radial-gradient(circle at 35% 30%, ${themeDots[t]}, color-mix(in srgb, ${themeDots[t]} 55%, #000))`,
                      cursor: 'pointer',
                      border: active ? '2px solid var(--gaea-glow)' : '2px solid transparent',
                      boxShadow: active ? `0 0 10px ${themeDots[t]}, 0 0 22px color-mix(in srgb, ${themeDots[t]} 45%, transparent)` : `0 0 6px color-mix(in srgb, ${themeDots[t]} 30%, transparent)`,
                      opacity: active ? 1 : 0.55,
                      transform: active ? 'scale(1.12)' : 'scale(1)',
                      transition: 'opacity 0.15s, border 0.15s, transform 0.2s, box-shadow 0.2s',
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
      <Layout style={{ flex: 1, flexDirection: 'row', background: 'transparent' }}>
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
            background: page === 'chat' ? 'var(--gaea-glass-bg, var(--md-sys-color-surface))' : 'transparent',
            overflow: page === 'chat' ? 'hidden' : 'auto',
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
          }}>
            <ErrorBoundary>
              <Suspense fallback={(
                <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', gap: 14, height: '100%' }}>
                  <Spin size="large" style={{ color: 'var(--gaea-glow)' }} />
                  <span className="live-dot" />
                  <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12, letterSpacing: '0.06em' }}>
                    正在唤醒 AI 模块…
                  </Typography.Text>
                </div>
              )}>
                {Array.from(visitedPages).map((p) => (
                  <div key={p} className="page-enter" style={{ display: p === page ? 'flex' : 'none', flex: 1, flexDirection: 'column', minHeight: 0 }}>
                    {p === 'home'
                      ? <ModuleLauncher onNavigate={(target: LauncherTarget) => setPage(target as Page)} activeModel={activeModel || undefined} />
                      : pageComponents[p]}
                  </div>
                ))}
              </Suspense>
            </ErrorBoundary>
          </Content>
        </div>

{consoleOpen && page !== 'home' && page !== 'imagegen' && page !== 'modelcenter' && page !== 'chat' && page !== 'whisper' && page !== 'office' && page !== 'gaea' && page !== 'settings' && (
  <div style={{
    width: 380, flexShrink: 0, alignSelf: 'stretch',
    maxHeight: 'calc(100vh - 80px)',
    margin: '8px 8px 8px 0',
    background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
    WebkitBackdropFilter: 'blur(24px) saturate(140%)',
    backdropFilter: 'blur(24px) saturate(140%)',
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
        <Typography.Text style={{ color: 'var(--gaea-glow)', fontSize: 10, fontWeight: 600, letterSpacing: '0.08em', textShadow: '0 0 10px var(--gaea-glow)' }}>
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

{!consoleOpen && page !== 'modelcenter' && page !== 'home' && page !== 'settings' && (
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



      {/* 底栏常驻：已启用模型监控 + CPU/内存/GPU（无项目时也显示） */}
      <Footer style={{ padding: 0 }}>
        <StatusBar stats={stats} info={projectInfo} />
      </Footer>

      {/* 搜索 */}
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} />
    </Layout>
  )
}

export default MainLayout
