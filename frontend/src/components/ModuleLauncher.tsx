/**
 * ModuleLauncher — 双空间首页（v10「天玄」卡片工作台）
 * 书斋（work）= 卡片台：指挥卡 / 文档卡 + 侧列 / 能力卡网格 / 状态卡排
 * 闲庭（play）= 卡片画廊：展厅卡 / 会客厅旗舰卡 + 会话卡 / 海报卡墙 / 园底状态卡排
 *
 * 设计判据（v6「等大圆角卡片 + 描边」被判廉价的根因，v10 逐条避坑）：
 *   1. 卡片形态必须异质：全宽指挥卡 / 宽文档卡 / 侧列小卡 / 跨列旗舰卡 / 模块卡 /
 *      状态卡排 / 竖版海报卡墙（首张跨格）——没有两排是等大瓦片。
 *   2. 层级靠四档排印（display / lede / body / label）+ 数据等宽；标题走负字距或
 *      中式衬线，label 走宽字距，绝不靠 13↔14px 微差。
 *   3. 深度＝双层工艺（外壳细线 + 双档投影，内芯再收一圈强调色细线）+ 色调面，
 *      描边只做 hairline；悬停抬升只给可点卡片。
 *   4. 装饰一律「版式记号」：印章 / 月洞门 / 徽记水印 / 顶缘月华描线；不用极光斑。
 *   5. 动效走弹性曲线（--ml-ease），只动 transform/box-shadow，reduced-motion 与
 *      gaea-raf-degraded 全降级。
 *   6. 主题母题＝「天玄」：玄穹墨蓝底 + 月华金强调 + 壳层星穹透出（theme preset
 *      tianXuan 见 stores/appStore.ts；其余预设共用同一套卡片骨架）。
 *
 * 契约保持（测试与壳层依赖，勿改）：
 *   ml-space-switch / ml-space-work / ml-space-play（aria-pressed + 不影响办公引擎空间
 *   的 title）、ml-space-chip、desk-recent-docs、.garden-banner、garden-progress /
 *   garden-sessions / garden-memory / garden-meters。数据层单源 useLauncherData，
 *   遥测/写作/会话/记忆两空间均可达（晨报卡 2026-10-01 起撤出首页，组件保留）。
 *   令牌纪律：零硬编码色值，全部走 --color-* / --md-sys-* / --gaea-* / --v3-*。
 */
import React, { useState, useCallback, useEffect, useMemo, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, RobotOutlined, UserOutlined, ThunderboltOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  CheckSquareOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, LAUNCHER_FEATURED, type LauncherModule } from '../boards/launcher'
import { SHELL_SPACES, type ShellSpace } from '../boards/space'
import TasksFirstHome from './TasksFirstHome'
import { requestSessionResume } from '../gaea/lib/pendingSessionResume'
import { loadRecentFiles, subscribeRecentFiles } from '../gaea/lib/recentFiles'
import { openPaneFileOrPreview } from '../gaea/lib/paneFileOpen'
import type { AtEntry } from '../gaea/lib/types'
import { Input } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { usePollingGate } from '../hooks/usePollingGate'
import { useT, type Translator } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'
import { getModelMonitor } from '../api/engines'
import { TaskInboxPanel } from '../gaea/components/TaskInboxPanel'
import './module-launcher.css'

/**
 * 启动器可跳转的目标页（3.0 §5.2：放宽为 string，由 manifest.id 派生）。
 */
export type LauncherTarget = string

/** 语音入口信号（书斋命令条本页直启语音，信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏展示） */
  activeModel?: string
  /** S2.1 壳层空间（决定渲染书斋 / 闲庭变体） */
  space: ShellSpace
  /** v4.182：空间切换（首页顶栏 SpaceSwitch 直连 MainLayout.switchSpace） */
  onSwitchSpace: (s: ShellSpace) => void
}

// ── 遥测/会话/记忆的最小类型（对齐 wails 生成的 d.ts，避免引入重型类型）──
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
  name?: string
  model?: string
  isLocal?: boolean
}
interface ModelMonitor {
  engines?: MonitorEngine[]
  stats?: MonitorStats
  comfyRunning?: boolean
}
interface SessionLite {
  title?: string
  preview?: string
  turns?: number
  modTime?: number
  current?: boolean
}
interface MemoryHubLite {
  knowledgeCount?: number
  profileCount?: number
  officeCount?: number
  costCount?: number
  whisperCount?: number
  pinnedCount?: number
  latestUpdated?: string
}

/** 字数友好格式化（>=1 万显示 x.x 万） */
function fmtWords(n: number, t: Translator): string {
  if (!n) return '0'
  if (n >= 10000) return (n / 10000).toFixed(1) + t('shell.launcher.fmtWan')
  return n.toLocaleString()
}

/** 相对时间（unix ms → 「刚刚 / N 分钟前 / N 小时前 / N 天前」） */
function fmtRel(ms: number, t: Translator): string {
  if (!ms) return '—'
  const diff = Date.now() - ms
  const min = Math.floor(diff / 60000)
  if (min < 1) return t('shell.launcher.fmtJustNow')
  if (min < 60) return t('shell.launcher.fmtMin', { n: min })
  const h = Math.floor(min / 60)
  if (h < 24) return t('shell.launcher.fmtHour', { n: h })
  const d = Math.floor(h / 24)
  if (d < 30) return t('shell.launcher.fmtDay', { n: d })
  return new Date(ms).toLocaleDateString()
}

// ════════════════════════════════════════════════════════════════════
//  排版原语：节标 / 仪表行 / 细轨 / 气泡
// ════════════════════════════════════════════════════════════════════

/** 节标：宽字距小标 + 渐隐细线（仪器面板式节眉，不带卡片） */
const SectionLabel: React.FC<{
  icon: React.ReactNode
  title: string
  sub?: string
}> = ({ icon, title, sub }) => (
  <header className="ml-eyebrow">
    <span className="ml-eyebrow-icon" aria-hidden="true">{icon}</span>
    <span className="ml-eyebrow-title">{title}</span>
    <span className="ml-eyebrow-rule" aria-hidden="true" />
    {sub && <span className="ml-eyebrow-sub">{sub}</span>}
  </header>
)

/** 仪表行：图标 + 标签 + 数值（等宽）+ 副文 */
const KernelRow: React.FC<{
  icon: React.ReactNode
  label: string
  value: React.ReactNode
  sub?: React.ReactNode
}> = ({ icon, label, value, sub }) => (
  <div className="ml-krow">
    <span className="ml-krow-icon" aria-hidden="true">{icon}</span>
    <div className="ml-krow-body">
      <div className="ml-krow-label">{label}</div>
      <div className="ml-krow-value">{value}</div>
      {sub && <div className="ml-krow-sub">{sub}</div>}
    </div>
  </div>
)

/** 资源细轨（≥85% 转 warning 色：色 + 数值双传达） */
const Meter: React.FC<{ label: string; pct: number | null }> = ({ label, pct }) => {
  const hot = pct != null && pct >= 85
  return (
    <div className="ml-meter">
      <span className="ml-meter-label">{label}</span>
      <span
        className="ml-meter-track"
        role="meter"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct ?? undefined}
        aria-label={`${label} ${pct != null ? `${pct}%` : '--'}`}
      >
        <span
          className={`ml-meter-fill${hot ? ' is-hot' : ''}`}
          style={{ width: `${Math.max(0, Math.min(100, pct ?? 0))}%` }}
        />
      </span>
      <span className="ml-meter-val">{pct != null ? `${pct}%` : '--'}</span>
    </div>
  )
}

/** 语音/对话气泡 */
const ChatBubble: React.FC<{ role: 'user' | 'assistant'; text: string }> = ({ role, text }) => {
  const isUser = role === 'user'
  return (
    <div className={`ml-bubble-row ${isUser ? 'ml-bubble-user' : 'ml-bubble-ai'}`}>
      {!isUser && (
        <div className="ml-avatar ml-avatar-ai"><RobotOutlined /></div>
      )}
      <div className="ml-bubble">{text}</div>
      {isUser && (
        <div className="ml-avatar ml-avatar-user"><UserOutlined /></div>
      )}
    </div>
  )
}

/** 写作进度环（书斋右栏 / 闲庭信息带共用） */
const WritingRing: React.FC<{
  data: LauncherData
}> = ({ data }) => {
  const t = useT()
  const plannedChapters = data.stats?.chapterCount ? Math.max(data.stats.chapterCount, data.stats.plannedChapters || 0) : 0
  const writtenChapters = data.stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0
  return (
    <div className="ml-progress">
      <div
        className="ml-ring"
        aria-hidden="true"
        style={{ background: `conic-gradient(var(--gaea-glow) ${progressPercent}%, color-mix(in srgb, var(--color-border) 70%, transparent) 0)` }}
      >
        <div className="ml-ring-hole">
          <span className="ml-ring-num">{data.stats ? `${progressPercent}%` : '—'}</span>
        </div>
      </div>
      <div className="ml-progress-body">
        {data.stats
          ? t('shell.launcher.statWritingSub', { chapters: data.stats.chapterCount, words: fmtWords(data.stats.totalWords, t) })
          : (data.projectOpen ? t('shell.launcher.statLoading') : t('shell.launcher.statNoProject'))}
      </div>
    </div>
  )
}

/** 记忆脉搏内容（书斋右栏 / 闲庭信息带共用） */
const MemoryPulse: React.FC<{ memoryHub: MemoryHubLite | null }> = ({ memoryHub }) => {
  const t = useT()
  const memoryTotal = memoryHub
    ? (memoryHub.knowledgeCount || 0) + (memoryHub.profileCount || 0) + (memoryHub.officeCount || 0)
      + (memoryHub.costCount || 0) + (memoryHub.whisperCount || 0) + (memoryHub.pinnedCount || 0)
    : 0
  const memoryUpdated = memoryHub?.latestUpdated ? Date.parse(memoryHub.latestUpdated) : 0
  if (!memoryHub) return <div className="ml-panel-empty">{t('shell.launcher.memoryIdle')}</div>
  return (
    <div className="ml-memory">
      <span className="ml-krow-strong">{t('shell.launcher.memoryCount', { count: memoryTotal })}</span>
      <span className="ml-sess-meta">{t('shell.launcher.memoryUpdated', { time: fmtRel(memoryUpdated, t) })}</span>
    </div>
  )
}

/**
 * 任务收件箱挂点内容（7.3-1）：待处理计数 + 「打开收件箱」。
 * 计数取 GaeaTaskInboxList(space) 的 pending 数——挂载时一次读（与
 * desk-recent-docs 同款首屏单次读）；inboxTick 在面板收起时 +1，用户动作
 * 触发补一次重读（判据④：零轮询零定时器，读写均由用户动作触发）。
 * row=true 为闲庭园底信息带的横排形态（全宽第五节），默认纵排对齐记忆脉搏。
 */
const TaskInboxEntry: React.FC<{
  space: ShellSpace
  tick: number
  onOpen: () => void
  row?: boolean
}> = ({ space, tick, onOpen, row }) => {
  const t = useT()
  const [pending, setPending] = useState<number | null>(null)
  useEffect(() => {
    let alive = true
    app
      .GaeaTaskInboxList(space)
      .then((list: unknown) => {
        if (!alive) return
        const rows = ((list ?? []) as { status?: string }[]).filter((x) => x.status === 'pending').length
        setPending(rows)
      })
      .catch(() => { if (alive) setPending(null) })
    return () => { alive = false }
  }, [space, tick])
  return (
    <div className="ml-memory" style={row ? { flexDirection: 'row', alignItems: 'center', gap: 14 } : undefined}>
      <span className="ml-krow-strong">
        {pending === null ? '—' : t('tasks.inbox.pendingCount', { n: pending })}
      </span>
      {/* 打开按钮：中性 hairline 胶囊（v11：去金点睛滥用，hover 才点亮） */}
      <button
        type="button"
        onClick={onOpen}
        data-testid="task-inbox-open-btn"
        className="ml-inline-cta"
        style={{ alignSelf: row ? 'center' : 'flex-start' }}
      >
        {t('tasks.inbox.open')}
      </button>
    </div>
  )
}

/** 遥测三表内容（书斋右栏 / 闲庭信息带共用；engineCount=0 时主行为 —） */
const TelemetryBody: React.FC<{ data: LauncherData }> = ({ data }) => {
  const t = useT()
  const ms = data.monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : null
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : null
  const cpuPct = ms && ms.cpu != null && ms.cpu >= 0 ? Math.round(ms.cpu) : null
  const gpuPct = (ms?.gpuUsage ?? 0) > 0 ? Math.round(ms?.gpuUsage ?? 0) : vramPct
  const engines = data.monitor?.engines || []
  const engineCount = engines.length
  const localCount = engines.filter((e) => e.isLocal).length
  return (
    <div className="ml-panel-body">
      {engineCount > 0 ? (
        <KernelRow
          icon={<ThunderboltOutlined />}
          label={t('shell.launcher.statEngines')}
          value={<span className="ml-krow-strong">{engineCount}</span>}
          sub={t('shell.launcher.statEnginesSub', { local: localCount, cloud: engineCount - localCount })}
        />
      ) : (
        /* 零引擎时不渲染孤零零的「—」仪表行，直接给空态文案（v11 降噪） */
        <div className="ml-panel-empty">{t('shell.launcher.statNoEngines')}</div>
      )}
      {ms ? (
        <div className="ml-meters">
          <Meter label="CPU" pct={cpuPct} />
          <Meter label="MEM" pct={memPct} />
          <Meter label="GPU" pct={gpuPct} />
          {data.monitor?.comfyRunning && (
            <div className="ml-comfy">
              <span className="ml-comfy-dot" aria-hidden="true" />
              <span>{t('home.comfyRunning')}</span>
            </div>
          )}
        </div>
      ) : null}
      {/* v11：statIdle 兜底行退役——零引擎时 statNoEngines 已表达待机语义，
          原先两行空态（「暂无引擎运行」+「遥测待机」）语义重复 */}
    </div>
  )
}

/** 会话列表内容（书斋=账页行式 / 闲庭=软胶囊 chips，由 chips 切换） */
export const SessionList: React.FC<{ sessions: SessionLite[]; onOpen: () => void; chips?: boolean }> = ({ sessions, onOpen, chips }) => {
  const t = useT()
  if (sessions.length === 0) return <div className="ml-panel-empty">{t('shell.launcher.noSessions')}</div>
  if (chips) {
    return (
      <div className="garden-chips">
        {sessions.map((s, i) => (
          <button
            key={s.modTime ?? i}
            type="button"
            className="garden-chip"
            onClick={onOpen}
            title={s.preview || s.title || ''}
          >
            <span className="garden-chip-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
            <span className="garden-chip-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
          </button>
        ))}
      </div>
    )
  }
  return (
    <ul className="ml-sess">
      {sessions.map((s, i) => (
        <li key={s.modTime ?? i} className="ml-sess-item">
          <span className="ml-sess-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
          <span className="ml-sess-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
        </li>
      ))}
    </ul>
  )
}

/** 首页共享数据（useLauncherData 一次拉取，两变体各自选用） */
export interface LauncherData {
  stats: ReturnType<typeof useAppStore.getState>['stats']
  projectOpen: boolean
  monitor: ModelMonitor | null
  recentFiles: AtEntry[]
  sessions: SessionLite[]
  memoryHub: MemoryHubLite | null
}

function useLauncherData(): LauncherData {
  const stats = useAppStore((s) => s.stats)
  const projectOpen = useAppStore((s) => s.projectOpen)

  // 遥测（3s 轮询，不可见门控）
  const [monitor, setMonitor] = useState<ModelMonitor | null>(null)
  const pollable = usePollingGate()
  useEffect(() => {
    let alive = true
    const load = async () => {
      if (!pollable) return
      try {
        const m = (await getModelMonitor()) as ModelMonitor
        if (alive) setMonitor(m)
      } catch (_) { /* 后端未就绪时静默 */ }
    }
    load()
    const timer = window.setInterval(load, 3000)
    return () => { alive = false; window.clearInterval(timer) }
  }, [pollable])

  // 最近文档（localStorage 单源，零新 binding；书斋专用）
  // v4.349：订阅单源而非挂载时读一次——本页在 keepAlive（visitedPages）下常驻，
  // 原先「在办公打开文档 → 回首页」看到的仍是首次挂载快照。
  const recentFilesRaw = useSyncExternalStore(subscribeRecentFiles, loadRecentFiles, () => [])
  const recentFiles = useMemo(() => recentFilesRaw.slice(0, 6), [recentFilesRaw])

  // 最近会话
  const [sessions, setSessions] = useState<SessionLite[]>([])
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const list = await app.ListSessions()
        if (!alive) return
        const rows = (list || [])
          .filter((s) => !s.current)
          .sort((a, b) => (b.modTime || 0) - (a.modTime || 0))
          .slice(0, 3)
        setSessions(rows as SessionLite[])
      } catch (_) { /* 静默 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  // 记忆脉搏
  const [memoryHub, setMemoryHub] = useState<MemoryHubLite | null>(null)
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const o = await app.MemoryHubOverview()
        if (alive) setMemoryHub(o as MemoryHubLite)
      } catch (_) { /* 静默 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  return { stats, projectOpen, monitor, recentFiles, sessions, memoryHub }
}

// ─── 顶栏仪表条：空间切换器（v4.182 从 rail 迁入首页）────────────────
// 1B 已拍板（2026-09-10，拆开）：壳层开关=界面导航，与办公引擎空间
// （办公侧栏 SpaceChip，写 session.space）两套是定局；切换零扰在跑的活
// （e-check E26 锁 switchSpace 零桥接）。title 说清楚防「切了闲庭办公
// 工具就没了」的误解。
/** 首页形态快捷切换（7.3-2）：形态互切按钮，读写 appStore.homeLayout。 */
const HomeLayoutToggle: React.FC = () => {
  const t = useT()
  const homeLayout = useAppStore((st) => st.homeLayout)
  const setHomeLayout = useAppStore((st) => st.setHomeLayout)
  const tasks = homeLayout === 'tasks'
  return (
    <button
      type="button"
      data-testid={tasks ? 'home-layout-back-classic' : 'home-layout-tasks-entry'}
      aria-pressed={tasks}
      title={tasks ? t('home.tasksFirstBackHint') : t('home.tasksFirstEntryHint')}
      className="ml-space-btn"
      onClick={() => setHomeLayout(tasks ? 'classic' : 'tasks')}
    >
      {tasks ? t('home.tasksFirstBack') : t('home.tasksFirstEntry')}
    </button>
  )
}

export const SpaceSwitch: React.FC<{
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
}> = ({ space, onSwitchSpace, activeModel }) => {
  const t = useT()
  return (
    <div className="ml-strip v3-rise v3-rise-1">
      <div className="ml-space-switch" role="group" data-testid="ml-space-switch" aria-label={t('home.spaceSwitchAria')}>
        {SHELL_SPACES.map((s) => {
          const active = s.id === space
          return (
            <button
              key={s.id}
              type="button"
              data-testid={`ml-space-${s.id}`}
              aria-pressed={active}
              title={`${t(s.titleKey) || s.title}；${t('home.spaceSwitchHint')}`}
              className={`ml-space-btn${active ? ' is-active' : ''}${s.id === 'play' ? ' is-play' : ''}`}
              onClick={() => { if (!active) onSwitchSpace(s.id) }}
            >
              {t(s.labelKey) || s.label}
            </button>
          )
        })}
      </div>
      {/* 7.3-2 首页形态快捷切换：classic→「任务优先」；tasks→「切回经典」。
          SpaceSwitch 两首页共用，按钮即回退开关的快捷位（设置页另有正式开关）。 */}
      <HomeLayoutToggle />
      <span className="ml-strip-rule" aria-hidden="true" />
      <span className="ml-strip-date" aria-hidden="true">{new Date().toLocaleDateString()}</span>
      <span className="ml-strip-spacer" aria-hidden="true" />
      {activeModel && (
        <span className="ml-model-chip" title={t('shell.launcher.statModel')}>
          <span className="ml-model-dot" aria-hidden="true" />
          <RobotOutlined aria-hidden="true" />
          <span className="ml-model-name">{activeModel}</span>
        </span>
      )}
    </div>
  )
}

// ════════════════════════════════════════════════════════════════════
//  书斋（work）· v9「案头」——masthead 身份区 + 纯命令台 + 账页×案牌 + 脉息面板
// ════════════════════════════════════════════════════════════════════
const DeskHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
  onOpenTaskInbox: () => void
  inboxTick: number
}> = ({ data, onNavigate, space, onSwitchSpace, activeModel, onOpenTaskInbox, inboxTick }) => {
  const t = useT()
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, space)
  const featuredModule = allModules.find((m) => m.key === LAUNCHER_FEATURED[space])
  const indexModules = allModules.filter((m) => m.key !== LAUNCHER_FEATURED[space] && m.key !== 'settings')
  const settingsModule = allModules.find((m) => m.key === 'settings')
  const spaceEntry = SHELL_SPACES.find((s) => s.id === space)
  const spaceLabel = spaceEntry ? (t(spaceEntry.labelKey) || spaceEntry.label) : ''

  const [typedText, setTypedText] = useState('')
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')
  const { state: voice, start, stop, interrupt } = useVoiceChat({
    onTranscript: (txt) => { setUserText(txt); setAiReply('') },
    onReply: (txt) => setAiReply(txt),
  })

  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    try { await app.VoiceApplySettings({ personalityPresetId: 'gaea' }) } catch (_) {}
    setUserText('')
    setAiReply('')
    await start()
  }, [voice.active, start, stop])

  const sendTyped = useCallback(() => {
    const text = typedText.trim()
    if (!text) return
    setUserText(text)
    setAiReply('')
    setTypedText('')
    // v4.361：失败可见化——此前输入被清空、回复区空白无提示，消息石沉大海。
    try {
      app.VoiceChatText(text).catch(() => setAiReply('发送失败，请稍后再试'))
    } catch { setAiReply('发送失败，请稍后再试') }
  }, [typedText])

  // FE6-02：麦克风不可用时命令条绝不再显示「聆听中」——降级位优先于一切
  // 状态文案。此处中文直写而非走 t()：与同区块 voice.error 同口径（壳内运行时
  // 状态/错误串本就由 hook 以中文给出，属内容层运行时状态而非壳层 chrome 标签），
  // 也避免在并行波次里改动三语字典这份契约文件。
  const micDegraded = voice.degraded === 'mic-unavailable'
  const voiceStateLabel = micDegraded
    ? '麦克风不可用'
    : voice.aiSpeaking
      ? t('shell.launcher.voiceReplying')
      : voice.listening
        ? t('shell.launcher.voiceListening')
        : voice.active
          ? t('shell.launcher.voiceStandby')
          : t('shell.launcher.voiceIdle')

  const voiceTone = voice.aiSpeaking
    ? 'is-speaking'
    : micDegraded
      // 降级时不套任何「在听/在说」的动效类名（is-listening 的呼吸动画本身就是
      // 「正在聆听」的视觉语义，正是要根除的那句谎）
      ? ''
      : voice.listening
        ? 'is-listening'
        : voice.active
          ? 'is-active'
          : ''

  const hasChat = !!userText || !!aiReply

  return (
    <div className="ml ml-work">
      <div className="w-board">
        {/* 顶栏仪表条（空间切换 / 形态切换 / 日期 / 模型）：两空间共用，零改动 */}
        <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

        {/* ═══ ① 指挥卡（全宽主角卡）：款识 + 台名 + 命令条。首页第一任务
            （发起工作）与身份、语音状态收在同一张卡里，不再分散成三条横带。 ═══ */}
        <section className="ml-card w-hero v3-rise v3-rise-1" aria-label={t('shell.launcher.heroAria')}>
          <header className="w-hero-head">
            <span className="w-seal" aria-hidden="true">{spaceLabel.slice(0, 1)}</span>
            <div className="w-hero-id">
              <h1 className="w-hero-title">{t('home.title')}</h1>
              <p className="w-hero-lede">{t('home.sub')}</p>
            </div>
            <div className="w-hero-side">
              {spaceEntry && (
                <span className="ml-space-chip" data-testid="ml-space-chip" title={t('home.spaceSwitchHint')}>
                  {spaceLabel}
                </span>
              )}
              <span className="w-hero-pill">{t('home.pill')}</span>
            </div>
          </header>

          {hasChat && (
            <div className="w-hero-chat" aria-live="polite">
              {userText && <ChatBubble role="user" text={userText} />}
              {aiReply && <ChatBubble role="assistant" text={aiReply} />}
            </div>
          )}

          {/* 命令条：单一抬升面 + focus-within 描边（键盘可达，⌘K 提示） */}
          <div className={`w-cmd ${voiceTone}`}>
            <span className="w-cmd-orb" aria-hidden="true">
              <span className="w-cmd-orb-core" />
              <span className="w-cmd-orb-ring" />
            </span>
            <Input
              value={typedText}
              onChange={(e) => setTypedText(e.target.value)}
              onPressEnter={() => sendTyped()}
              placeholder={t('home.placeholder')}
              aria-label={t('home.placeholder')}
              variant="borderless"
              className="w-cmd-input"
              disabled={voice.active}
            />
            {voice.active ? (
              <button
                type="button"
                className="w-cmd-btn is-voice is-on"
                onClick={toggleVoice}
                aria-label={t('shell.launcher.voiceAriaEnd')}
              >
                <StopOutlined /> <span className="w-cmd-btn-label">{t('shell.launcher.voiceEnd')}</span>
              </button>
            ) : (
              <button
                type="button"
                className="w-cmd-btn is-voice"
                onClick={toggleVoice}
                aria-label={t('shell.launcher.voiceAriaStart')}
              >
                <AudioOutlined /> <span className="w-cmd-btn-label">{t('shell.launcher.voiceStart')}</span>
              </button>
            )}
            <button
              type="button"
              className="w-cmd-btn is-send"
              onClick={sendTyped}
              disabled={!typedText.trim() || voice.active}
              aria-label={t('shell.launcher.courtyardSend')}
            >
              <SendOutlined />
            </button>
            <kbd className="w-kbd" title={t('home.cmdk')} aria-label={t('home.cmdk')}>⌘K</kbd>
          </div>

          <div className="w-voice-status" aria-label={t('home.voiceStatusAria', { state: voiceStateLabel })}>
            <span className={`w-status-dot${voiceTone ? ` ${voiceTone}` : ''}`} aria-hidden="true" />
            <span className="w-status-label">{voiceStateLabel}</span>
            {micDegraded && (
              <span
                role="alert"
                data-testid="voice-mic-degraded"
                className="ml-1.5 rounded-md border border-amber-500/30 bg-amber-500/5 px-2 py-0.5 text-amber-500 text-[11px] leading-relaxed"
              >
                麦克风不可用，未在采集音频，本回合走文本输入
              </span>
            )}
            {voice.error && <span className="w-voice-err" role="alert">{voice.error}</span>}
            {voice.active && voice.aiSpeaking && (
              <button className="w-interrupt-btn" onClick={interrupt} type="button">
                <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
              </button>
            )}
          </div>
        </section>

        {/* ═══ ② 最近文档卡（宽卡）+ 侧列（写作进度卡）═══ */}
        <section
          className="ml-card w-docs v3-rise v3-rise-2"
          aria-label={t('home.recentDocs')}
          data-testid="desk-recent-docs"
        >
          <CardHead
            icon={<FileTextOutlined />}
            title={t('home.recentDocs')}
            sub={t('home.recentDocsHint')}
            tail={data.recentFiles.length > 0 ? <span className="ml-card-count">{data.recentFiles.length}</span> : undefined}
          />
          {data.recentFiles.length > 0 ? (
            <ul className="w-docs-list">
              {data.recentFiles.map((f, i) => (
                <li key={`${f.path}:${i}`} className="w-docs-row">
                  <button
                    type="button"
                    className="w-docs-btn"
                    title={t('home.recentDocsHint')}
                    aria-label={`${f.name || f.path} — ${t('home.recentDocsHint')}`}
                    /* v4.439：点行＝真打开该文档（此前只跳办公板块，与「在办公中打开」
                       文案不符）。经 paneFileOpen 的统一入口——办公工作台已挂载则开
                       pane 文件页签，未挂载则落预览队列，导航过去即可见。 */
                    onClick={() => { openPaneFileOrPreview(f.path); onNavigate('gaea') }}
                  >
                    <span className="w-docs-mark" aria-hidden="true"><FileTextOutlined /></span>
                    <span className="w-docs-body">
                      <span className="w-docs-name">{f.name || f.path}</span>
                      <span className="w-docs-path">{f.path}</span>
                    </span>
                    <ArrowRightOutlined className="w-docs-arrow" aria-hidden="true" />
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <div className="w-empty">
              <FileTextOutlined className="w-empty-icon" aria-hidden="true" />
              <span className="w-empty-text">{t('home.recentDocsEmpty')}</span>
              <span className="w-empty-sub">{t('home.recentDocsEmptySub')}</span>
              <button type="button" className="w-empty-cta" onClick={() => onNavigate('gaea')}>
                <ArrowRightOutlined aria-hidden="true" />
                {t('home.recentDocsEmptyCta')}
              </button>
            </div>
          )}
        </section>

        <div className="w-aside">
          <section className="ml-card w-progress v3-rise v3-rise-2" aria-label={t('shell.launcher.statWriting')}>
            <CardHead icon={<FileTextOutlined />} title={t('shell.launcher.statWriting')} />
            <WritingRing data={data} />
          </section>
          {/* 晨报（做梦 2.0）已撤出首页（2026-10-01 用户拍板「删除今日晨报」）：
              组件与数据管线保留（gaea/components/MorningBriefCard），仅撤书斋侧列入口 */}
        </div>

        {/* ═══ ③ 能力矩阵：节标 + 卡片网格（旗舰=跨列大卡，其余=模块卡）═══ */}
        <section className="w-cap v3-rise v3-rise-3" aria-label={t('home.capTitle')}>
          <SectionLabel icon={<ThunderboltOutlined />} title={t('home.capTitle')} sub={t('home.capSub')} />
          <div className="w-modules">
            {featuredModule && (
              <ModuleCard m={featuredModule} featured onOpen={() => onNavigate(featuredModule.key)} />
            )}
            {indexModules.map((m) => (
              <ModuleCard key={m.key} m={m} onOpen={() => onNavigate(m.key)} />
            ))}
            {settingsModule && (
              <ModuleCard m={settingsModule} settings onOpen={() => onNavigate(settingsModule.key)} />
            )}
            {indexModules.length === 0 && !featuredModule && !settingsModule && (
              <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
            )}
          </div>
        </section>

        {/* ═══ ④ 状态卡排：内核 / 会话 / 记忆 / 任务（自适列宽，窄屏自动换行）═══ */}
        <section className="w-stat-row v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <section className="ml-card w-stat" aria-label={t('home.kernel')}>
            <CardHead icon={<ApiOutlined />} title={t('home.kernel')} />
            <TelemetryBody data={data} />
          </section>

          <section className="ml-card w-stat" aria-label={t('shell.launcher.sessions')}>
            <CardHead icon={<ClockCircleOutlined />} title={t('shell.launcher.sessions')} />
            <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
          </section>

          <section className="ml-card w-stat" aria-label={t('shell.launcher.memoryPulse')}>
            <CardHead icon={<HeartOutlined />} title={t('shell.launcher.memoryPulse')} />
            <MemoryPulse memoryHub={data.memoryHub} />
          </section>

          {/* 7.3-1 任务收件箱：待处理计数 + 打开收件箱（挂点 testid 契约不变） */}
          <section className="ml-card w-stat" aria-label={t('shell.launcher.taskInbox')} data-testid="desk-task-inbox">
            <CardHead icon={<CheckSquareOutlined />} title={t('shell.launcher.taskInbox')} />
            <TaskInboxEntry space={space} tick={inboxTick} onOpen={onOpenTaskInbox} />
          </section>
        </section>
      </div>
    </div>
  )
}

/**
 * 卡片头（v10 卡片工作台统一头部）：图标章 + 标题/副文 + 可选尾注。
 * 导出供任务优先首页（TasksFirstHome）复用同一张卡片的头部形态。
 */
export const CardHead: React.FC<{
  icon: React.ReactNode
  title: string
  sub?: string
  tail?: React.ReactNode
}> = ({ icon, title, sub, tail }) => (
  <header className="ml-card-head">
    <span className="ml-card-icon" aria-hidden="true">{icon}</span>
    <span className="ml-card-titles">
      <span className="ml-card-title">{title}</span>
      {sub && <span className="ml-card-sub">{sub}</span>}
    </span>
    {tail}
  </header>
)

/**
 * 能力卡（v10）：manifest 驱动的板块入口卡。旗舰=跨列大卡（徽记水印 +
 * 进入胶囊），普通=模块卡（图标章 + 名称/描述 + 悬停箭头）；同一 LauncherModule
 * 数据流，交互与 aria 文案沿用 v9（旗舰 enterWorkbench / 普通 enterModule）。
 */
const ModuleCard: React.FC<{
  m: LauncherModule
  featured?: boolean
  /** v11：settings 降权为节尾横条（低频入口不占卡位） */
  settings?: boolean
  onOpen: () => void
}> = ({ m, featured, settings, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  const label = t(featured ? 'shell.launcher.enterWorkbench' : 'shell.launcher.enterModule', { name: m.name })
  return (
    <button
      type="button"
      className={`w-mod${featured ? ' is-featured' : ''}${settings ? ' is-settings' : ''}`}
      aria-label={label}
      onClick={onOpen}
    >
      {featured && <span className="w-mod-mark" aria-hidden="true">{Icon ? <Icon /> : null}</span>}
      <span className="w-mod-icon" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="w-mod-body">
        {featured && <span className="w-mod-badge">{t('home.featured')}</span>}
        <span className="w-mod-name">{m.name}</span>
        <span className="w-mod-desc">{m.desc}</span>
      </span>
      <span className="w-mod-go" aria-hidden="true">
        {featured && <span className="w-mod-go-label">{label}</span>}
        <ArrowRightOutlined className="w-mod-arrow" />
      </span>
    </button>
  )
}

// ════════════════════════════════════════════════════════════════════
//  闲庭（play）·「游园画廊」——月洞门标题区 + 旗舰横幅 + 海报墙 + 园底信息带
// ════════════════════════════════════════════════════════════════════
const GardenHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
  onOpenTaskInbox: () => void
  inboxTick: number
}> = ({ data, onNavigate, space, onSwitchSpace, activeModel, onOpenTaskInbox, inboxTick }) => {
  const t = useT()
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, space)
  const featuredModule = allModules.find((m) => m.key === LAUNCHER_FEATURED[space])
  const galleryModules = allModules.filter((m) => m.key !== LAUNCHER_FEATURED[space] && m.key !== 'settings')
  const settingsModule = allModules.find((m) => m.key === 'settings')
  const spaceEntry = SHELL_SPACES.find((s) => s.id === space)

  return (
    <div className="ml ml-play">
      <div className="p-board">
        <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

        {/* ① 展厅卡（全宽）：月洞门记号左置 + 画廊大字 + lede。v9 居中仪式区
            收进卡片，与书斋指挥卡同一条款式位（记号 + 名号 + 题辞）。 */}
        <section className="ml-card p-hero-card v3-rise v3-rise-1" aria-label={t('home.title')}>
          <span className="p-gate" aria-hidden="true">
            <span className="p-gate-inner" />
          </span>
          <div className="p-hero-id">
            <span className="p-kicker">{spaceEntry ? (t(spaceEntry.labelKey) || spaceEntry.label) : ''}</span>
            <h1 className="p-display">{t('home.playTitle')}</h1>
            <p className="p-lede">{t('home.playSub')}</p>
          </div>
        </section>

        {/* ② 会客厅旗舰卡（7 列）+ 最近会话卡（5 列） */}
        {featuredModule && <GardenBanner m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />}
        <section
          className="ml-card p-sessions v3-rise v3-rise-2"
          data-testid="garden-sessions"
          aria-label={t('shell.launcher.sessions')}
        >
          <CardHead icon={<ClockCircleOutlined />} title={t('shell.launcher.sessions')} />
          <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} chips />
        </section>

        {/* ③ 海报卡墙：首张跨格大样（2×2），其余单格；窄屏逐档降列。
            v11.2：设置横条移出墙外（避免被墙的 1fr 行高撑开成空洞） */}
        <section className="p-wall v3-rise v3-rise-3" aria-label={t('home.capTitle')}>
          {galleryModules.map((m, i) => (
            <GardenPoster key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
          ))}
          {galleryModules.length === 0 && !featuredModule && (
            <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
          )}
        </section>
        {settingsModule && (
          <GardenPoster m={settingsModule} idx={galleryModules.length} onOpen={() => onNavigate(settingsModule.key)} compact />
        )}

        {/* ④ 园底状态卡排：写作进度 / 记忆脉搏 / 内核状态 / 任务
            （v9 单条 p-foot 信息带退役；四节 testid 与信息全保留） */}
        <section className="p-stat-row v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <section className="ml-card p-stat" data-testid="garden-progress" aria-label={t('shell.launcher.statWriting')}>
            <CardHead icon={<FileTextOutlined />} title={t('shell.launcher.statWriting')} />
            <WritingRing data={data} />
          </section>

          <section className="ml-card p-stat" data-testid="garden-memory" aria-label={t('shell.launcher.memoryPulse')}>
            <CardHead icon={<HeartOutlined />} title={t('shell.launcher.memoryPulse')} />
            <MemoryPulse memoryHub={data.memoryHub} />
          </section>

          <section className="ml-card p-stat" data-testid="garden-meters" aria-label={t('home.kernel')}>
            <CardHead icon={<ApiOutlined />} title={t('home.kernel')} />
            <TelemetryBody data={data} />
          </section>

          {/* 7.3-1 任务收件箱：与书斋同款卡片形态（挂点 testid 契约不变） */}
          <section
            className="ml-card p-stat"
            data-testid="garden-task-inbox"
            aria-label={t('shell.launcher.taskInbox')}
          >
            <CardHead icon={<CheckSquareOutlined />} title={t('shell.launcher.taskInbox')} />
            <TaskInboxEntry space={space} tick={inboxTick} onOpen={onOpenTaskInbox} />
          </section>
        </section>
      </div>
    </div>
  )
}

/** 闲庭旗舰横幅（会客厅：层叠色调 + 巨型徽记水印 + 大标题 + 进入胶囊） */
const GardenBanner: React.FC<{ m: LauncherModule; onOpen: () => void }> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <button
      type="button"
      aria-label={t('shell.launcher.enterWorkbench', { name: m.name })}
      className="garden-banner ml-card v3-rise v3-rise-2"
      onClick={onOpen}
    >
      <span className="garden-banner-wash" aria-hidden="true" />
      <span className="garden-banner-mark" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="garden-banner-body">
        <span className="garden-banner-badge">{t('home.featured')}</span>
        <span className="garden-banner-name">{m.name}</span>
        <span className="garden-banner-desc">{m.desc}</span>
      </span>
      <span className="garden-banner-cta">
        {t('shell.launcher.enterWorkbench', { name: m.name })}
        <ArrowRightOutlined className="garden-banner-arrow" aria-hidden="true" />
      </span>
    </button>
  )
}

/** 闲庭海报（竖版：色调底板 + 圆徽记 + 名称 + 描述 + hover 环形箭头；首张跨格） */
const GardenPoster: React.FC<{
  m: LauncherModule
  idx: number
  onOpen: () => void
  compact?: boolean
}> = ({ m, idx, onOpen, compact }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  const hero = !compact && idx === 0
  return (
    <button
      type="button"
      aria-label={t('shell.launcher.enterModule', { name: m.name })}
      className={`p-poster ml-card v3-rise${hero ? ' is-hero' : ''}${compact ? ' is-compact' : ''}`}
      style={{ animationDelay: `${140 + idx * 55}ms` } as React.CSSProperties}
      onClick={onOpen}
    >
      {compact ? (
        <>
          <span className="p-poster-seal is-small" aria-hidden="true">{Icon ? <Icon /> : null}</span>
          <span className="p-poster-name">{m.name}</span>
          <span className="p-poster-desc">{m.desc}</span>
          <ArrowRightOutlined className="p-poster-arrow" aria-hidden="true" />
        </>
      ) : (
        <>
          <span className="p-poster-plate" aria-hidden="true" />
          {/* v11.1：首张大样的版式记号——超大徽记水印压右下，填充 414px 海报的中部空板 */}
          {hero && <span className="p-poster-mark" aria-hidden="true">{Icon ? <Icon /> : null}</span>}
          <span className="p-poster-seal" aria-hidden="true">{Icon ? <Icon /> : null}</span>
          <span className="p-poster-body">
            <span className="p-poster-name">{m.name}</span>
            <span className="p-poster-desc">{m.desc}</span>
          </span>
          <span className="p-poster-go" aria-hidden="true"><ArrowRightOutlined /></span>
        </>
      )}
    </button>
  )
}

/**
 * ModuleLauncher — 首页入口：数据一次拉取，按壳层空间分发书斋 / 闲庭变体。
 * 7.3-1：任务收件箱面板单例挂顶层（space 跟随当前 home 空间，onNavigate 复用
 * 既有回调）；两空间的收件箱挂点（书斋 w-stat-row「任务」卡 / 闲庭 p-stat-row
 * 「任务」卡）都只改本组件内 inboxOpen 状态。inboxTick 在面板收起时 +1，驱动挂点计数
 * 补一次重读（用户动作触发，非轮询）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space, onSwitchSpace }) => {
  const data = useLauncherData()
  const [inboxOpen, setInboxOpen] = useState(false)
  const [inboxTick, setInboxTick] = useState(0)
  const openInbox = useCallback(() => setInboxOpen(true), [])
  const closeInbox = useCallback(() => {
    setInboxOpen(false)
    setInboxTick((x) => x + 1)
  }, [])

  // 会话级回源（7.3-1 收口）：记 pending+派发事件（keepAlive 已挂载态直达），
  // 再导航到 gaea 板块——冷启态由 App 挂载消费 pending 兜底。
  const resumeSessionFromInbox = useCallback((path: string) => {
    requestSessionResume(path)
    onNavigate('gaea')
  }, [onNavigate])
  // 7.3-2 板块降级为任务视图：homeLayout==='tasks' → 任务优先首页（收件箱首屏+
  // 能力 chips）；classic → 既有 Desk/Garden 零变化（藏≠删，回退开关即时切回）。
  const homeLayout = useAppStore((st) => st.homeLayout)

  return (
    <>
      {homeLayout === 'tasks' ? (
        <TasksFirstHome data={data} onNavigate={onNavigate} space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} onResumeSession={resumeSessionFromInbox} />
      ) : space === 'work' ? (
        <DeskHome data={data} onNavigate={onNavigate} space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} onOpenTaskInbox={openInbox} inboxTick={inboxTick} />
      ) : (
        <GardenHome data={data} onNavigate={onNavigate} space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} onOpenTaskInbox={openInbox} inboxTick={inboxTick} />
      )}
      <TaskInboxPanel open={inboxOpen} onClose={closeInbox} space={space} onNavigate={onNavigate} onResumeSession={resumeSessionFromInbox} />
    </>
  )
}

export default ModuleLauncher
