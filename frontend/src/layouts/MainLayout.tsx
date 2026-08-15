import React, { useState, useEffect, useRef, Suspense } from 'react'
import { Layout, Menu, Button, Space, Typography, Tooltip, Spin, Progress, Breadcrumb, Tag, notification, type MenuProps } from 'antd'
import {
  SunOutlined, MoonOutlined, SearchOutlined, SettingOutlined, LoginOutlined,
  FileTextOutlined, EditOutlined, TeamOutlined, EyeOutlined, BarChartOutlined,
  HomeOutlined,
} from '@ant-design/icons'
import SearchModal from '../components/SearchModal'
import SecurityBanner from '../components/SecurityBanner'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { Z_INDEX } from '../utils/zIndex'
import { useAppStore, type ThemePreset, type StatsData, type ProjectInfo } from '../stores/appStore'
import ModuleLauncher, { type LauncherTarget } from '../components/ModuleLauncher'
import * as App from '../../src/wailsjsCompat'
// 3.0 §5.2 / 附 B：12 硬编码点收敛为 manifest 驱动
import {
  type BoardId, menuBoards, navigateWhitelist, shortcutMap, homeBoard,
  projectAnchorId, getBoard, boardLabel, resolveBoardIcon,
} from '../boards/manifests'
import { getPageComponent } from '../boards/pageRegistry'
import { subscribe, BACKEND_EVENTS, FRONTEND_EVENTS } from '../events'

// ─── 页面组件：PageRegistry 集中注册（main.tsx），此处保留旧 lazy import 作为
//     过渡期 fallback（附 B #3/#5：PageRegistry 与旧 pageComponents 并行一个版本）。
//     过渡期后仅保留 registerPage，删除本组 lazy。
const legacyPageComponents: Record<string, React.ComponentType> = {
  ChatPage: React.lazy(() => import('../pages/ChatPage')),
  NovelPage: React.lazy(() => import('../pages/NovelPage')),
  ImageGenPage: React.lazy(() => import('../pages/ImageGenPage')),
  GaeaPage: React.lazy(() => import('../pages/GaeaPage')),
  MemoryHubPage: React.lazy(() => import('../pages/MemoryHubPage')),
  ModelCenterPage: React.lazy(() => import('../pages/ModelCenterPage')),
  CharacterLibraryPage: React.lazy(() => import('../pages/CharacterLibraryPage')),
  SettingsPage: React.lazy(() => import('../pages/SettingsPage')),
}

const { Header, Footer, Content } = Layout

// 附 B #1：Page 类型由 manifest.id 派生（类型级断言锁死在 manifests.ts）
type Page = BoardId

// 附 B #4：顶栏横向导航（含首页启动器）= manifest.filter(inMenu).sort(menuOrder)
const menuItems: MenuProps['items'] = menuBoards.map((b) => {
  const Icon = resolveBoardIcon(b.icon)
  return { key: b.id, icon: Icon ? <Icon /> : undefined, label: b.label }
})

/** 解析页面组件：PageRegistry（manifest.page）优先，旧 pageComponents 兜底（过渡期） */
function resolvePageComponent(p: Page): React.ComponentType | undefined {
  const m = getBoard(p)
  if (!m) return undefined
  return getPageComponent(m.page) ?? legacyPageComponents[m.page]
}

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

interface MonitorStats {
  cpu?: number
  memUsed?: number
  memTotal?: number
  vramUsed?: number
  vramTotal?: number
  gpuUsage?: number
  gpuName?: string
}
interface MonitorEngine {
  engine: string
  name: string
  model: string
  isLocal?: boolean
}
interface ModelMonitor {
  engines?: MonitorEngine[]
  stats?: MonitorStats
  comfyRunning?: boolean
}

const StatusBar: React.FC<{ stats: StatsData | null; info: ProjectInfo | null }> = ({ stats, info }) => {
  // 模型监控：已启动引擎 + 系统资源（轮询 3s，防止本地模型加载过多）
  const [monitor, setMonitor] = useState<{ engines?: MonitorEngine[]; stats?: MonitorStats } | null>(null)
  const lastWarn = useRef<Record<string, number>>({})
  useEffect(() => {
    let alive = true
    // 超载警告：CPU/GPU/内存/模型过多，60s 内同类型只弹一次
    const checkOverload = (m: ModelMonitor) => {
      const now = Date.now()
      const ms = m?.stats
      const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
      const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
      const gpuPct = ms?.gpuUsage || vramPct
      // 模型加载预警只统计本地模型（herdsman/ollama 已启用 + ComfyUI 运行中），
      // 云端引擎（xai/deepseek）走 API 不占本机资源，不计入
      const localEngCount = (m?.engines || []).filter((e: MonitorEngine) => e.isLocal).length + (m?.comfyRunning ? 1 : 0)
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
        const m = (await App.GetModelMonitor()) as ModelMonitor
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
  // 底栏只展示本地已启动模型（云端引擎走 API，不占本机资源，不列出来）
  const engLabel = (monitor?.engines || [])
    .filter((e: MonitorEngine) => e.isLocal)
    .map(e => `${e.engine}${e.model ? '·' + String(e.model).split('/').pop() : ''}`)
  // 计算写作进度：已写章节/总大纲叶子节点（stats.chapterCount 为已有章节，保守取该值）
  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, stats.plannedChapters || 0) : 0
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

// ─── 主布局 ─────────────────────────────────────────────────
const MainLayout: React.FC = () => {
  const [page, setPage] = useState<Page>(homeBoard.id)
  const {
    loggedIn, login, checkLogin, baseTheme, darkMode, setTheme, toggleDarkMode,
    projectOpen, projectInfo, stats, loadProjectInfo, loadStats,
  } = useAppStore()

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  // 附 B #10：visitedPages 初始 home = manifest.isHome
  const [visitedPages, setVisitedPages] = useState<Set<Page>>(new Set([homeBoard.id]))

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
      const model = await window.go.app.App.GetActiveModel()
      setActiveModel(model || '')
    } catch (_) {}
  }

  // 监听后端模型切换事件，实时刷新右上角显示（事件名常量收敛到 events.ts）
  useEffect(() => {
    const handler = () => { loadActiveModel() }
    return subscribe(BACKEND_EVENTS.MODEL_CHANGED, handler)
  }, [])
  useEffect(() => {
    if (projectOpen) {
      loadProjectInfo()
      loadStats()
    }
  }, [projectOpen])
  // 监听跨页面导航事件（附 B #2：白名单 = manifest 派生）
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail
      if (detail?.page && navigateWhitelist.includes(detail.page as Page)) {
        setPage(detail.page as Page)
      }
    }
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
    return () => window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
  }, [])

  // 全局快捷键（附 B #6：Ctrl+1~4 = manifest.shortcut 显式声明）
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const ctrl = e.ctrlKey || e.metaKey
      if (!ctrl) return
      if (e.key >= '1' && e.key <= '4') {
        const target = shortcutMap[`ctrl+${e.key}`]
        if (target) {
          e.preventDefault()
          setPage(target)
        }
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

  // 附 B #9：Content 布局 = manifest.layout（chat/gaea=full，其余 padded，home=isHome 特判）
  const currentLayout = getBoard(page)?.layout ?? 'padded'
  const isFullLayout = currentLayout === 'full'
  const isHomePage = page === homeBoard.id

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
        <Header style={{
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
            {/* 附 B #12：settings 隐式入口 —— inMenu=false，壳层保留右上角按钮 */}
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
      {/* ═══ Herdsman LAN 暴露安全告警（S2-1，暴露时全局展示）═══ */}
      <SecurityBanner />
      {/* ═══ 主体：面包屑 + 内容 + 右侧 XAI ═══ */}
      <Layout style={{ flex: 1, flexDirection: 'row', background: 'transparent' }}>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* 面包屑导航（附 B #8：锚点 = manifest.breadcrumb.anchorTo，novel 声明自己是项目锚点） */}
          {projectOpen && page !== projectAnchorId && page !== homeBoard.id && (
            <div style={{
              padding: '6px 16px 0', background: 'var(--color-bg-layout)',
            }}>
              <Breadcrumb
                items={[
                  { title: <a onClick={() => setPage(projectAnchorId as Page)} style={{ color: 'var(--md-sys-color-text-secondary)', cursor: 'pointer' }}>
                    <HomeOutlined style={{ marginRight: 2 }} />{projectInfo?.title || ''}
                  </a> },
                  { title: <span style={{ color: 'var(--md-sys-color-primary)' }}>{boardLabel(page)}</span> },
                ]}
                style={{ fontSize: 12 }}
              />
            </div>
          )}
          <Content style={{
            padding: isFullLayout ? 0 : (isHomePage ? '16px' : '8px 16px 16px'),
            paddingBottom: isFullLayout || isHomePage ? 0 : '16px',
            background: isFullLayout ? 'var(--gaea-glass-bg, var(--md-sys-color-surface))' : 'transparent',
            overflow: isFullLayout ? 'hidden' : 'auto',
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
                {Array.from(visitedPages).map((p) => {
                  // 附 B #11：home 特判分支 = manifest.isHome（渲染 ModuleLauncher）
                  const Comp = p === homeBoard.id ? undefined : resolvePageComponent(p)
                  return (
                    <div key={p} className="page-enter" style={{ display: p === page ? 'flex' : 'none', flex: 1, flexDirection: 'column', minHeight: 0 }}>
                      {p === homeBoard.id
                        ? <ModuleLauncher onNavigate={(target: LauncherTarget) => setPage(target as Page)} activeModel={activeModel || undefined} />
                        : Comp ? <Comp /> : null}
                    </div>
                  )
                })}
              </Suspense>
            </ErrorBoundary>
          </Content>
        </div>

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
