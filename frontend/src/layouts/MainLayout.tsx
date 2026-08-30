import { wailsApp } from '../lib/wailsApp';
import React, { useState, useEffect, useRef, Suspense, useReducer, useCallback } from 'react'
import { Layout, Button, Space, Typography, Tooltip, Spin, Progress, notification } from 'antd'
import {
  SunOutlined, MoonOutlined, SearchOutlined, SettingOutlined, LoginOutlined,
  HomeOutlined, FileTextOutlined, UpOutlined, DownOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import SearchModal from '../components/SearchModal'
import SecurityBanner from '../components/SecurityBanner'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { useAppStore, THEME_PRESETS, THEME_PRESET_COLORS, THEME_PRESET_LABELS, type StatsData, type ProjectInfo } from '../stores/appStore'
import ModuleLauncher, { type LauncherTarget } from '../components/ModuleLauncher'
import * as App from '../../src/wailsjsCompat'
// 3.0「星枢 Constellation OS」壳层革命：顶栏横向菜单 → 左侧指挥轨道 + 顶部轨道条 + 底部遥测轨道。
// 板块数据源依旧 manifest 驱动（后端 GetBoardManifests + home 壳层合并，失败回退静态 canonicalBoards）。
import {
  type BoardId, resolveBoardIcon,
  subscribeBoards, loadBoardManifests,
  getActiveMenuBoards, getActiveMenuBoardsForSpace, getActiveIndependentBoards, getActiveNavigateWhitelist,
  getActiveHomeBoard, getActiveProjectAnchorId, getActiveBoard, activeBoardLabel,
} from '../boards/manifests'
import {
  isBoardReachableInSpace, boardSpace, pruneVisitedForSpace, type ShellSpace,
} from '../boards/space'
import { getPageComponent } from '../boards/pageRegistry'
import { subscribe, subscribeForSpace, BACKEND_EVENTS, FRONTEND_EVENTS } from '../events'
import { useFeatureModel } from '../hooks/useFeatureModel'
import { usePollingGate } from '../hooks/usePollingGate'
import { useT, type Translator } from '../gaea/lib/i18n'

const { Content } = Layout

// ─── S2.1 每空间最后页面持久化（docs/gaea-space-shell-design.md §4.3）───────
const SHELL_PAGE_PREFIX = 'gaea.shell.page.'

function shellPageKey(space: ShellSpace): string {
  return `${SHELL_PAGE_PREFIX}${space}`
}

function loadShellPage(space: ShellSpace): string {
  try {
    const v = localStorage.getItem(shellPageKey(space))
    if (v && getActiveBoard(v)) return v
  } catch (_) {}
  return getActiveHomeBoard().id
}

function saveShellPage(space: ShellSpace, page: string): void {
  try { localStorage.setItem(shellPageKey(space), page) } catch (_) {}
}

/** 目标页在当前空间是否可达（home 恒可达；其余按 manifest.space） */
function pageReachableInSpace(p: Page, s: ShellSpace): boolean {
  const b = getActiveBoard(p)
  if (!b) return false
  return b.isHome || isBoardReachableInSpace(b, s)
}

// ─── 页面组件：PageRegistry 集中注册（main.tsx），此处保留旧 lazy import 作为
//     过渡期 fallback（附 B #3/#5：PageRegistry 与旧 pageComponents 并行一个版本）。
//     过渡期后仅保留 registerPage，删除本组 lazy。
const legacyPageComponents: Record<string, React.ComponentType> = {
  ChatPage: React.lazy(() => import('../pages/ChatPage')),
  NovelPage: React.lazy(() => import('../pages/NovelPage')),
  ImageGenPage: React.lazy(() => import('../pages/ImageGenPage')),
  GaeaPage: React.lazy(() => import('../pages/GaeaPage')),
  MemoryHubPage: React.lazy(() => import('../pages/MemoryHubPage')),
  CostLibraryPage: React.lazy(() => import('../pages/CostLibraryPage')),
  ModelCenterPage: React.lazy(() => import('../pages/ModelCenterPage')),
  CharacterLibraryPage: React.lazy(() => import('../pages/CharacterLibraryPage')),
  SettingsPage: React.lazy(() => import('../pages/SettingsPage')),
}

// 附 B #1：Page 类型由 manifest.id 派生（类型级断言锁死在 manifests.ts）
type Page = BoardId

/** 解析页面组件：PageRegistry（manifest.page）优先，旧 pageComponents 兜底（过渡期） */
function resolvePageComponent(p: Page): React.ComponentType | undefined {
  const m = getActiveBoard(p)
  if (!m) return undefined
  return getPageComponent(m.page) ?? legacyPageComponents[m.page]
}

// 主题色点/标签：单一数据源 THEME_PRESETS（appStore，3.0 Wave 2 消除三处重复维护）
const themeKeys = THEME_PRESETS.map((t) => t.key)
const themeDots = THEME_PRESET_COLORS
const themeLabels = THEME_PRESET_LABELS

function fmtWords(n: number, t: Translator): string {
  if (n >= 10000) return (n / 10000).toFixed(1) + t('shell.launcher.fmtWan')
  return n.toLocaleString()
}

// ─── 遥测数据模型 ─────────────────────────────────────────────
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

/** 实时面积 sparkline（SVG 自绘，无新依赖；--v3-telemetry 主色 + 20% 填充 + 数字，不只靠颜色） */
function Sparkline({ data, max = 100 }: { data: number[]; max?: number }) {
  const h = 34
  const n = Math.max(data.length - 1, 1)
  const pts = data.map((v, i) => `${(i / n) * 100},${h - 2 - (Math.min(v, max) / max) * (h - 5)}`)
  const line = pts.join(' ')
  const area = `0,${h} ${line} 100,${h}`
  if (data.length < 2) {
    return <svg viewBox={`0 0 100 ${h}`} preserveAspectRatio="none" aria-hidden="true" />
  }
  return (
    <svg viewBox={`0 0 100 ${h}`} preserveAspectRatio="none" aria-hidden="true">
      <polyline className="v3-spark-fill" points={area} />
      <polyline className="v3-spark-line" points={line} />
    </svg>
  )
}

/** Spark 卡：标题 + 当前值 + sparkline */
const SparkCard: React.FC<{ label: string; value: string; data: number[]; hint?: string }> = ({ label, value, data, hint }) => (
  <div className="v3-spark-card" title={hint}>
    <div className="v3-spark-head">
      <span className="v3-tele-key">{label}</span>
      <span className="v3-tele-value">{value}</span>
    </div>
    <Sparkline data={data} />
  </div>
)

// ─── 遥测轨道（底部）────────────────────────────────────────
const TelemetryRail: React.FC<{ stats: StatsData | null; info: ProjectInfo | null; showProject: boolean }> = ({ stats, info, showProject }) => {
  const [collapsed, setCollapsed] = useState(true)
  const [monitor, setMonitor] = useState<{ engines?: MonitorEngine[]; stats?: MonitorStats } | null>(null)
  const t = useT()
  // 历史缓冲：CPU/内存/GPU 各保留最近 20 点（60s @ 3s 轮询）
  const [history, setHistory] = useState<{ cpu: number[]; mem: number[]; gpu: number[] }>({ cpu: [], mem: [], gpu: [] })
  const lastWarn = useRef<Record<string, number>>({})
  const reducedMotion = useRef(false)
  // 系统级后台轮询治理：页面不可见（窗口最小化/切走）时轮询空转零成本
  const visible = usePollingGate()
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    reducedMotion.current = mq.matches
    const onMq = (e: MediaQueryListEvent) => { reducedMotion.current = e.matches }
    mq.addEventListener?.('change', onMq)
    return () => mq.removeEventListener?.('change', onMq)
  }, [])

  useEffect(() => {
    let alive = true
    // 超载警告：CPU/GPU/内存/模型过多，60s 内同类型只弹一次
    const checkOverload = (m: ModelMonitor) => {
      const now = Date.now()
      const ms = m?.stats
      const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
      const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
      const gpuPct = ms?.gpuUsage || vramPct
      const localEngCount = (m?.engines || []).filter((e: MonitorEngine) => e.isLocal).length + (m?.comfyRunning ? 1 : 0)
      const warns: { key: string; title: string; desc: string }[] = []
      if ((ms?.cpu ?? 0) > 85) warns.push({ key: 'cpu', title: t('shell.tele.overloadCpu'), desc: t('shell.tele.overloadCpuDesc', { cpu: ms?.cpu ?? 0 }) })
      if (gpuPct > 85) warns.push({ key: 'gpu', title: t('shell.tele.overloadGpu'), desc: t('shell.tele.overloadGpuDesc', { gpu: gpuPct }) })
      if (memPct > 90) warns.push({ key: 'mem', title: t('shell.tele.overloadMem'), desc: t('shell.tele.overloadMemDesc', { mem: memPct }) })
      if (localEngCount > 3) warns.push({ key: 'models', title: t('shell.tele.overloadModels'), desc: t('shell.tele.overloadModelsDesc', { count: localEngCount }) })
      warns.forEach((w) => {
        if (now - (lastWarn.current[w.key] || 0) > 60_000) {
          lastWarn.current[w.key] = now
          notification.warning({ message: w.title, description: w.desc, placement: 'topRight', duration: 6 })
        }
      })
    }
    const load = async () => {
      if (!visible) return
      try {
        const m = (await App.GetModelMonitor()) as ModelMonitor
        if (!alive) return
        setMonitor(m)
        checkOverload(m)
        // 追加遥测历史（reduced-motion 只保留最近 2 点，静态折线；-1 哨兵按 0 处理）
        const keep = reducedMotion.current ? 2 : 20
        setHistory((prev) => {
          const ms = m?.stats
          const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
          const cpu = [...prev.cpu, Math.max(0, ms?.cpu ?? 0)].slice(-keep)
          const mem = [...prev.mem, ms?.memTotal ? Math.max(0, Math.round((ms.memUsed || 0) / ms.memTotal * 100)) : 0].slice(-keep)
          const gpu = [...prev.gpu, Math.max(0, (ms?.gpuUsage ?? 0) > 0 ? (ms?.gpuUsage ?? 0) : vramPct)].slice(-keep)
          return { cpu, mem, gpu }
        })
      } catch { /* 忽略轮询失败 */ }
    }
    load()
    const timer = setInterval(load, 3000)
    return () => { alive = false; clearInterval(timer) }
  }, [visible, t])

  const ms = monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
  // 后端未采样哨兵（-1）统一按「未就绪」显示 --
  const cpuOk = ms && ms.cpu != null && ms.cpu >= 0 ? ms.cpu : null
  const gpuOk = (ms?.gpuUsage ?? 0) > 0 ? (ms?.gpuUsage ?? 0) : vramPct
  // 只展示本地已启用引擎（云端引擎走 API，不占本机资源）
  const engLabel = (monitor?.engines || [])
    .filter((e: MonitorEngine) => e.isLocal)
    .map((e) => `${e.engine}${e.model ? '·' + String(e.model).split('/').pop() : ''}`)
  // 项目写作进度
  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, stats.plannedChapters || 0) : 0
  const writtenChapters = stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  return (
    <div className={`v3-telemetry${collapsed ? ' is-collapsed' : ''}`} aria-label={t('shell.tele.aria')}>
      <div className="v3-telemetry-inner">
        <button
          className="v3-telemetry-toggle"
          onClick={() => setCollapsed((c) => !c)}
          aria-expanded={!collapsed}
          aria-label={collapsed ? t('shell.tele.expand') : t('shell.tele.collapse')}
        >
          {collapsed ? <UpOutlined /> : <DownOutlined />}
          <span style={{ marginLeft: 4 }}>{t('shell.tele.label')}</span>
        </button>
        {/* 引擎 pods */}
        {engLabel.length ? (
          engLabel.map((e) => (
            <Tooltip key={e} title={t('shell.tele.enginesTip')}>
              <span className="v3-engine-pod"><span className="v3-pod-dot" />{e}</span>
            </Tooltip>
          ))
        ) : (
          <Tooltip title={t('shell.tele.noEnginesTip')}>
            <span className="v3-engine-pod" style={{ opacity: 0.65 }}><ThunderboltOutlined style={{ fontSize: 10 }} />{t('shell.tele.noEngines')}</span>
          </Tooltip>
        )}
        {info && showProject && (
          <span style={{ color: 'var(--color-text)', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{info.title}</span>
        )}
        {stats && showProject && (
          <>
            <span className="v3-tele-key">{t('shell.tele.progress')}</span>
            <Progress
              percent={progressPercent}
              size="small"
              showInfo={false}
              style={{ width: 90, margin: 0 }}
              strokeColor="var(--gaea-glow)"
              trailColor="var(--md-sys-color-outline-variant)"
            />
            <span className="v3-tele-value">{progressPercent}%</span>
            <span className="v3-tele-key"><FileTextOutlined />{t('shell.tele.chapters', { count: stats.chapterCount, words: fmtWords(stats.totalWords, t) })}</span>
          </>
        )}
        <span className="v3-telemetry-spacer" style={{ flex: 1 }} />
        <span className="v3-tele-key">CPU</span><span className="v3-tele-value">{cpuOk != null ? cpuOk + '%' : '--'}</span>
        <span className="v3-tele-key">{t('shell.tele.mem')}</span><span className="v3-tele-value">{ms ? memPct + '%' : '--'}</span>
        {ms?.gpuName ? (
          <>
            <span className="v3-tele-key">GPU</span>
            <span className="v3-tele-value">{gpuOk ? gpuOk + '%' : '--'}</span>
          </>
        ) : null}
      </div>
      <div className="v3-telemetry-body">
        <SparkCard label={t('shell.tele.cpuUsage')} value={cpuOk != null ? `${cpuOk}%` : '--'} data={history.cpu} hint={t('shell.tele.cpuHint', { value: cpuOk != null ? cpuOk + '%' : '--' })} />
        <SparkCard label={t('shell.tele.memUsage')} value={ms ? `${memPct}%` : '--'} data={history.mem} hint={t('shell.tele.memHint', { used: memPct, total: ms?.memTotal ?? '--' })} />
        <SparkCard label={ms?.gpuName ? t('shell.tele.gpuNamed', { name: ms.gpuName }) : t('shell.tele.gpuUsage')} value={ms?.gpuName ? (gpuOk ? gpuOk + '%' : '--') : '--'} data={history.gpu} hint={t('shell.tele.gpuHint', { vram: vramPct })} />
      </div>
    </div>
  )
}

// ─── 指挥轨道（左侧 · OS 极简窄条）────────────────────────────────
// 固定窄栏、纯图标、hover 微缩放 + 原生 tooltip（title）、激活 = 主色容器 + 底部指示光条。
const CommandRail: React.FC<{
  page: Page
  onNavigate: (p: Page) => void
  darkMode: boolean
  toggleDarkMode: () => void
}> = ({ page, onNavigate, darkMode, toggleDarkMode }) => {
  // v4.3.2c：空间切换入口已移除（首页三栏即空间入口，导航按板块自动切空间），
  // rail 展示全部 inMenu 板块（含共享），独立窗口（编程 DSH）单列。
  const boards = getActiveMenuBoards()
  // 独立窗口板块（编程 DSH）：单独入口，不混入任一空间导航
  const independentBoards = getActiveIndependentBoards()
  const t = useT()
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([])

  const moveFocus = (idx: number, dir: number) => {
    const next = (idx + dir + boards.length) % boards.length
    itemRefs.current[next]?.focus()
  }

  return (
    <nav
      className="v3-rail"
      aria-label={t('shell.rail.aria')}
    >
      <div className="v3-rail-head">
        <img src="/favicon.svg" alt="gaea" />
      </div>
      <div className="v3-rail-nav" role="menubar" aria-label={t('shell.rail.nav')}>
        {boards.map((b, i) => {
          const Icon = resolveBoardIcon(b.icon)
          const active = b.id === page
          return (
            <button
              key={b.id}
              ref={(el) => { itemRefs.current[i] = el }}
              role="menuitem"
              tabIndex={active ? 0 : -1}
              aria-current={active ? 'page' : undefined}
              aria-label={b.label}
              title={b.label}
              className={`v3-rail-item${active ? ' is-active' : ''}`}
              onClick={() => { onNavigate(b.id as Page); itemRefs.current[i]?.focus() }}
              onKeyDown={(e) => {
                if (e.key === 'ArrowDown') { e.preventDefault(); moveFocus(i, 1) }
                if (e.key === 'ArrowUp') { e.preventDefault(); moveFocus(i, -1) }
                if (e.key === 'Home') { e.preventDefault(); itemRefs.current[0]?.focus() }
                if (e.key === 'End') { e.preventDefault(); itemRefs.current[boards.length - 1]?.focus() }
              }}
            >
              <span className="v3-rail-icon">{Icon ? <Icon /> : null}</span>
            </button>
          )
        })}
      </div>
      <div className="v3-rail-foot">
        {independentBoards.length > 0 && (
          <>
            <div className="v3-rail-divider" aria-hidden="true" />
            {independentBoards.map((b) => {
              const Icon = resolveBoardIcon(b.icon)
              const active = b.id === page
              return (
                <button
                  key={b.id}
                  type="button"
                  role="menuitem"
                  tabIndex={active ? 0 : -1}
                  aria-current={active ? 'page' : undefined}
                  aria-label={b.label}
                  title={`${b.label}（${t('shell.rail.independentWindow')}）`}
                  className={`v3-rail-item${active ? ' is-active' : ''}`}
                  onClick={() => { onNavigate(b.id as Page) }}
                >
                  <span className="v3-rail-icon">{Icon ? <Icon /> : null}</span>
                </button>
              )
            })}
          </>
        )}
        <Tooltip title={darkMode ? t('shell.rail.darkOff') : t('shell.rail.darkOn')} placement="right">
          <button
            className="v3-rail-item"
            aria-label={darkMode ? t('shell.rail.darkOff') : t('shell.rail.darkOn')}
            onClick={toggleDarkMode}
          >
            <span className="v3-rail-icon">{darkMode ? <SunOutlined /> : <MoonOutlined />}</span>
          </button>
        </Tooltip>
      </div>
    </nav>
  )
}

// ─── 主布局 ─────────────────────────────────────────────────
const MainLayout: React.FC = () => {
  // S2.1：壳层空间（持久化 appStore.space）+ 每空间最后页面
  const space = useAppStore((s) => s.space)
  const setSpace = useAppStore((s) => s.setSpace)
  const t = useT()
  // 启动默认首页（3.0：gaea 每次启动都从首页启动器落地，不再恢复上次页面）；
  // 会话内跨空间切换仍按空间恢复该空间最后页面（switchSpace 内 loadShellPage）。
  const [page, setPage] = useState<Page>(() => getActiveHomeBoard().id as Page)
  const {
    loggedIn, login, checkLogin, baseTheme, darkMode, setTheme, toggleDarkMode,
    projectOpen, projectInfo, stats, loadProjectInfo, loadStats,
  } = useAppStore()

  const [searchOpen, setSearchOpen] = useState(false)
  // 附 B #10：visitedPages 初始 home = manifest.isHome
  const [visitedPages, setVisitedPages] = useState<Set<Page>>(new Set([getActiveHomeBoard().id]))

  // 页面切换 → 持久化到当前空间 key；切空间时恢复到该空间最后页面
  React.useEffect(() => {
    saveShellPage(space, page)
  }, [space, page])

  const switchSpace = useCallback((next: ShellSpace) => {
    if (next === space) return
    saveShellPage(space, page)
    setSpace(next)
    const saved = loadShellPage(next)
    const nextPage = (pageReachableInSpace(saved as Page, next) ? saved : getActiveHomeBoard().id) as Page
    setPage(nextPage)
    // S2.2 性能门控：剪枝 keepAlive 保活页——只保留 home/当前空间可达页，
    // 跨空间页面卸载，后台轮询/渲染归零（审计「keepAlive 后台轮询」风险项）
    setVisitedPages(new Set(pruneVisitedForSpace(
      new Set([...visitedPages, nextPage]),
      (id) => pageReachableInSpace(id as Page, next),
    )))
  }, [space, page, setSpace, visitedPages])

  // 数据源 seam 接线：订阅活动板块清单并触发一次加载；清单变化时重渲染菜单/白名单/快捷键/布局。
  const [, forceBoards] = useReducer((c: number) => c + 1, 0)
  useEffect(() => {
    const unsub = subscribeBoards(() => forceBoards())
    void loadBoardManifests()
    return unsub
  }, [])

  // 跟踪已访问页面，避免切换时销毁组件丢失状态（keepAlive）
  React.useEffect(() => {
    setVisitedPages((prev) => {
      if (prev.has(page)) return prev
      const next = new Set(prev)
      next.add(page)
      return next
    })
  }, [page])

  const [activeModel, setActiveModel] = useState('')
  useEffect(() => { checkLogin(); loadActiveModel() }, [checkLogin])

  // 3.0 定制：左下角悬浮模型卡（FeatureModelBar）移除后，顶栏 pill 统一展示
  // 「当前板块功能绑定的模型状态」（manifest.featureModel 驱动），无板块绑定时回退全局活跃模型。
  const curFeature = getActiveBoard(page)?.featureModel
  const fm = useFeatureModel(curFeature ?? '')
  const pillModel = fm?.model || activeModel || ''
  const pillTip = fm?.model
    ? t('shell.strip.modelTip', { model: fm.model, engine: fm.engine ?? '' })
    : activeModel
      ? t('shell.strip.modelTipSimple', { model: activeModel })
      : t('shell.strip.modelTipNone')
  // 项目上下文（小说标题/进度/字数）只在「项目锚点板块」（小说）显示；
  // 切到绘梦/办公/聊天等板块时，顶栏面包屑与底栏遥测不再残留小说信息。
  const projectContextVisible = projectOpen && !!getActiveBoard(page)?.breadcrumb?.anchorTo

  const loadActiveModel = async () => {
    try {
      const model = await wailsApp().GetActiveModel()
      setActiveModel(model || '')
    } catch { /* 忽略 */ }
  }

  // 监听后端模型切换事件，实时刷新轨道条模型 pill。v4.5.1a：走
  // subscribeForSpace——payload 带空间字段（未来按空间模型变更）时只收当前
  // 空间；无空间字段的全局变更照常放行。
  useEffect(() => {
    const handler = () => { loadActiveModel() }
    return subscribeForSpace(BACKEND_EVENTS.MODEL_CHANGED, handler, space)
  }, [space])

  useEffect(() => {
    if (projectOpen) {
      loadProjectInfo()
      loadStats()
    }
  }, [projectOpen, loadProjectInfo, loadStats])

  // 跨页面导航事件（附 B #2：白名单 = manifest 派生）
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail
      // S2.1：白名单 + 空间可达性双重校验（work 事件不打扰乐园 UI，反之亦然）
      if (detail?.page && getActiveNavigateWhitelist().includes(detail.page as Page)
        && pageReachableInSpace(detail.page as Page, space)) {
        setPage(detail.page as Page)
      }
    }
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
    return () => window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, handler)
  }, [space])

  // 全局快捷键：Ctrl+1~9 = 当前空间指挥轨道模块顺序（home 恒首位除外），
  // Ctrl+N 新建项目（仅首页；在工位时显式切到乐园——跨空间由用户显式发起）
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const ctrl = e.ctrlKey || e.metaKey
      if (!ctrl) return
      if (e.key >= '1' && e.key <= '9') {
        const targets = getActiveMenuBoardsForSpace(space).filter((b) => !b.isHome)
        const target = targets[parseInt(e.key, 10) - 1]
        if (target) {
          e.preventDefault()
          setPage(target.id as Page)
        }
        return
      }
      if (e.key === 'n' && !projectOpen) {
        e.preventDefault()
        if (space !== 'play') {
          switchSpace('play')
          // 切空间后落地到乐园小说页（loadShellPage(play) 兜底；显式指定 novel）
          setPage('novel')
        } else {
          setPage('novel')
        }
        return
      }
      // v4.7 S4.6：Ctrl+K 打开全局搜索/命令面板（顶栏 tooltip「Search (Ctrl+K)」
      // 名副其实落地）+ 面板内统一意图路由直通（SearchModal 指令预览卡）。
      // gaea 工作台内让位给其自有 CommandPalette（gaea/App.tsx document 级监听
      // 同样截获 Ctrl+K）——按当前板块豁免，避免双面板同开。
      if (e.key === 'k' && page !== 'gaea') {
        e.preventDefault()
        setSearchOpen(true)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [projectOpen, space, switchSpace, page])

  // 附 B #9：Content 布局 = manifest.layout（chat/gaea=full，其余 padded，home=isHome 特判）
  const currentLayout = getActiveBoard(page)?.layout ?? 'padded'
  const isFullLayout = currentLayout === 'full'
  const isHomePage = page === getActiveHomeBoard().id

  // v4.3.2c：统一导航入口——按目标板块的 manifest.space 自动切换壳层空间。
  // 首页三栏/rail 均走此入口：左翼书房板块 → work；右翼庭院板块 → play；
  // 编程(independent)/设置(shared) 保持当前空间。空间切换持久化 + 页面落地。
  const navigateBoard = useCallback((target: Page) => {
    const b = getActiveBoard(target)
    if (!b) return
    const targetSpace = boardSpace(b)
    if (targetSpace === 'work' || targetSpace === 'play') {
      if (targetSpace !== space) {
        saveShellPage(space, page)
        setSpace(targetSpace)
        setPage(target)
        setVisitedPages(new Set(pruneVisitedForSpace(
          new Set([...visitedPages, target]),
          (id) => pageReachableInSpace(id as Page, targetSpace),
        )))
        return
      }
    }
    setPage(target)
  }, [space, page, setSpace, visitedPages])

  // v4.5 指令中枢（S4.4）：语音等入口的意图命中 → 壳层导航（自动切空间，
  // 复用 navigateBoard 机制）。后端 intent_router.go emit。保持全局订阅不按
  // 空间过滤——跨空间导航正是该事件的存在意义（play 语音「打开办公」必须
  // 能唤醒 work 页面）。
  useEffect(() => {
    const unsub = subscribe(BACKEND_EVENTS.INTENT_NAVIGATE, (data: unknown) => {
      const detail = data as { board?: string }
      if (detail?.board && typeof detail.board === 'string') {
        navigateBoard(detail.board as Page)
      }
    })
    return unsub
  }, [navigateBoard])

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

      <div style={{ display: 'flex', flex: 1, minHeight: 0, position: 'relative' }}>
        {/* ═══ 左侧指挥轨道（OS 自动隐藏 dock：默认收进左缘，滑过热区展开） ═══ */}
        <div className="v3-rail-dock">
          <CommandRail
            page={page}
            onNavigate={navigateBoard}
            darkMode={darkMode}
            toggleDarkMode={toggleDarkMode}
          />
        </div>

        {/* ═══ 右侧列：轨道条 + 内容 + 遥测轨道 ═══ */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* 顶部轨道条 */}
          <header className="v3-strip">
            {/* 一键返回首页（非首页时显示） */}
            {page !== getActiveHomeBoard().id && (
              <Tooltip title={t('shell.strip.home')}>
                <Button
                  type="text"
                  size="small"
                  icon={<HomeOutlined />}
                  onClick={() => setPage(getActiveHomeBoard().id as Page)}
                  aria-label={t('shell.strip.home')}
                  style={{ color: 'var(--color-text-secondary)' }}
                />
              </Tooltip>
            )}
            <div className="v3-strip-context">
              {projectContextVisible && page !== getActiveProjectAnchorId() && page !== getActiveHomeBoard().id && (
                <>
                  <span
                    role="link"
                    tabIndex={0}
                    aria-label={`回到项目 ${projectInfo?.title || ''}`}
                    onClick={() => setPage(getActiveProjectAnchorId() as Page)}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setPage(getActiveProjectAnchorId() as Page) } }}
                    style={{ color: 'var(--color-text-secondary)', cursor: 'pointer' }}
                  >
                    {projectInfo?.title || ''}
                  </span>
                  <span style={{ opacity: 0.5 }}>/</span>
                  <strong>{activeBoardLabel(page)}</strong>
                </>
              )}
              {!projectContextVisible && page !== getActiveHomeBoard().id && <strong>{activeBoardLabel(page)}</strong>}
            </div>
            {/* 聊天模式切换条宿主（T6-10.2：ChatPage 经 portal 渲染进此容器；
                仅聊天板块激活时可见，其他板块隐藏） */}
            <div id="v3-chatmode-host" className="v3-strip-chatmode"
              style={{ display: page === 'chat' ? undefined : 'none' }}
              aria-hidden={page !== 'chat' || undefined} />
            {/* 编程工作台工具栏宿主（ProgrammingPage 经 portal 渲染进此容器；
                仅编程板块激活时可见，其他板块隐藏） */}
            <div id="v3-prog-host" className="v3-strip-prog"
              style={{ display: page === 'code' ? undefined : 'none' }}
              aria-hidden={page !== 'code' || undefined} />
            <div className="v3-strip-spacer" />
            {/* 模型 pill：当前板块功能绑定模型 / 全局活跃模型；点击进入模型中心 */}
            <Tooltip title={pillTip}>
              <button className="v3-model-pill" aria-label={`当前模型 ${pillModel || '未设置'}`} onClick={() => setPage('modelcenter')}>
                <span className="v3-pulse-dot" aria-hidden="true" />
                <span className="v3-pill-label">{pillModel || '选择模型'}</span>
              </button>
            </Tooltip>
            {/* 主题色点（键盘可达） */}
            <Space size={5}>
              {themeKeys.map((t) => {
                const active = t === baseTheme
                return (
                  <Tooltip key={t} title={themeLabels[t]}>
                    <span
                      className="theme-dot"
                      role="button"
                      tabIndex={0}
                      aria-label={`切换主题 ${themeLabels[t]}`}
                      onClick={() => setTheme(t)}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setTheme(t) } }}
                      style={{
                        width: 16, height: 16, borderRadius: '50%',
                        background: `radial-gradient(circle at 35% 30%, ${themeDots[t]}, color-mix(in srgb, ${themeDots[t]} 55%, #000))`,
                        cursor: 'pointer',
                        border: active ? '2px solid var(--gaea-glow)' : '2px solid transparent',
                        boxShadow: active ? `0 0 10px ${themeDots[t]}, 0 0 20px color-mix(in srgb, ${themeDots[t]} 45%, transparent)` : `0 0 6px color-mix(in srgb, ${themeDots[t]} 30%, transparent)`,
                        opacity: active ? 1 : 0.55,
                        transform: active ? 'scale(1.1)' : 'scale(1)',
                        transition: 'opacity 0.15s, border 0.15s, transform 0.2s, box-shadow 0.2s',
                      }}
                    />
                  </Tooltip>
                )
              })}
            </Space>
            {/* 搜索 */}
            {projectOpen && (
              <Tooltip title={t('shell.strip.searchTip')}>
                <Button type="text" size="small" icon={<SearchOutlined />} onClick={() => setSearchOpen(true)} aria-label={t('shell.strip.searchAria')} style={{ color: 'var(--color-text-secondary)' }} />
              </Tooltip>
            )}
            {/* 设置 */}
            <Tooltip title={t('shell.strip.settings')}>
              <Button type="text" size="small" icon={<SettingOutlined />} onClick={() => setPage('settings')} aria-label={t('shell.strip.settingsAria')} style={{ color: 'var(--color-text-secondary)' }} />
            </Tooltip>
            {/* 登录状态 */}
            {loggedIn ? (
              <Tooltip title={t('shell.strip.loggedIn')}>
                <Button type="text" size="small" icon={<LoginOutlined />} aria-label={t('shell.strip.loggedIn')} style={{ color: 'var(--color-success)' }} />
              </Tooltip>
            ) : (
              <Button type="link" size="small" icon={<LoginOutlined />} onClick={login} style={{ color: 'var(--color-primary)', fontSize: 12 }}>
                {t('shell.strip.login')}
              </Button>
            )}
          </header>

          {/* Herdsman LAN 暴露安全告警 */}
          <SecurityBanner />

          {/* 主体：面包屑 + 内容 */}
          <Layout style={{ flex: 1, flexDirection: 'row', background: 'transparent', minHeight: 0 }}>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
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
                      <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 12, letterSpacing: '0.06em' }}>
                        {t('shell.strip.loading')}
                      </Typography.Text>
                    </div>
                  )}>
                    {Array.from(visitedPages).map((p) => {
                      // 附 B #11：home 特判分支 = manifest.isHome（渲染 ModuleLauncher）
                      const Comp = p === getActiveHomeBoard().id ? undefined : resolvePageComponent(p)
                      return (
                        <div key={p} className="page-enter" style={{ display: p === page ? 'flex' : 'none', flex: 1, flexDirection: 'column', minHeight: 0 }}>
                          {p === getActiveHomeBoard().id
                            ? <ModuleLauncher
                                onNavigate={(target: LauncherTarget) => navigateBoard(target as Page)}
                                activeModel={activeModel || undefined}
                                space={space}
                              />
                            : Comp ? <Comp /> : null}
                        </div>
                      )
                    })}
                  </Suspense>
                </ErrorBoundary>
              </Content>
            </div>
          </Layout>

          {/* ═══ 底部遥测轨道 ═══ */}
          <TelemetryRail stats={stats} info={projectInfo} showProject={projectContextVisible} />
        </div>
      </div>

      {/* 搜索 */}
      <SearchModal open={searchOpen} onClose={() => setSearchOpen(false)} space={space} />
    </Layout>
  )
}

export default MainLayout
