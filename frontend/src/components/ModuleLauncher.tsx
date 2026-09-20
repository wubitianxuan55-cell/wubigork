/**
 * ModuleLauncher — 双空间首页（v7 重设计：书斋「文书台」/ 闲庭「游园画廊」）
 * v9 书斋·案头：masthead+印章身份区 / 命令台独占主角 / 案牌网格 / 脉息面板
 *
 * 设计判据（为什么 v6 仍显廉价，v7 逐条改）：
 *   1. 到处都是「等大圆角卡片 + 描边」= 模板感源头 → v7 结构只用三种形态：
 *      仪表条（hairline 分隔的行）、账页（表格式行 + 等宽序号）、海报墙（真正的
 *      大小跨格，不是等大瓦片）。
 *   2. 层级只靠字号 13/14 的微差 → v7 建立 display/lede/label/caption 四档，
 *      display 走 clamp + 负字距，label 走宽字距，数据一律等宽数字。
 *   3. 深度靠描边平铺 → v7 用三级色调面（surface / container / container-high）
 *      + 精准投影（只在交互态出现），描边降到 hairline。
 *   4. 装饰用极光斑（aurora blob）= 典型 AI 味 → v7 删除，改用色调渐变 + 序号水印
 *      + 月洞门细环这类「版式记号」。
 *   5. 动效只有一次性 rise → v7 保留分阶入场，补 hover 位移/描边生长（只动
 *      transform/opacity），并给 reduced-motion 与 gaea-raf-degraded 全降级。
 *
 * 契约保持（测试与壳层依赖，勿改）：
 *   ml-space-switch / ml-space-work / ml-space-play（aria-pressed + 不影响办公引擎空间
 *   的 title）、ml-space-chip、desk-recent-docs、.garden-banner、garden-progress /
 *   garden-sessions / garden-memory / garden-meters。数据层单源 useLauncherData，
 *   零功能删除：遥测/写作/会话/记忆两空间均可达（晨报仅书斋=work 记忆红线）。
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
import type { AtEntry } from '../gaea/lib/types'
import { Input } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { usePollingGate } from '../hooks/usePollingGate'
import { useT, type Translator } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'
import { getModelMonitor } from '../api/engines'
import MorningBriefCard from '../gaea/components/MorningBriefCard'
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

/** 两位序号（水印/账页序号共用） */
const pad2 = (n: number) => String(n).padStart(2, '0')

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
      {/* 打开按钮：hairline 胶囊（令牌走 --w-accent，.ml 作用域内恒可用） */}
      <button
        type="button"
        onClick={onOpen}
        data-testid="task-inbox-open-btn"
        style={{
          border: '1px solid color-mix(in srgb, var(--w-accent) 34%, transparent)',
          color: 'var(--w-accent)',
          background: 'transparent',
          borderRadius: 999,
          padding: '3px 12px',
          fontSize: 11,
          cursor: 'pointer',
          alignSelf: row ? 'center' : 'flex-start',
        }}
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
      <KernelRow
        icon={<ThunderboltOutlined />}
        label={t('shell.launcher.statEngines')}
        value={<span className="ml-krow-strong">{engineCount > 0 ? `${engineCount}` : '—'}</span>}
        sub={engineCount > 0
          ? t('shell.launcher.statEnginesSub', { local: localCount, cloud: engineCount - localCount })
          : t('shell.launcher.statNoEngines')}
      />
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
      ) : (
        <div className="ml-panel-empty">{t('shell.launcher.statIdle')}</div>
      )}
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

  const voiceStateLabel = voice.aiSpeaking
    ? t('shell.launcher.voiceReplying')
    : voice.listening
      ? t('shell.launcher.voiceListening')
      : voice.active
        ? t('shell.launcher.voiceStandby')
        : t('shell.launcher.voiceIdle')

  const voiceTone = voice.aiSpeaking
    ? 'is-speaking'
    : voice.listening
      ? 'is-listening'
      : voice.active
        ? 'is-active'
        : ''

  const hasChat = !!userText || !!aiReply

  return (
    <div className="ml ml-work">
      <div className="w-dock">
        {/* ═══ 左舷：仪表条 + 命令台（主角）+ 账页×目录双列 ═══ */}
        <div className="w-main">
          <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

          {/* v9 身份区（masthead）：印章 + 标题/lede + 空间 chip + 就绪 pill。v8
              deck 内的 kicker 行升回独立 header——身份与状态归 masthead，命令台
              只管发起工作；印章取空间名首字（书/斋），纯装饰 aria-hidden。 */}
          <header className="w-masthead v3-rise v3-rise-1">
            <span className="w-seal" aria-hidden="true">{spaceLabel.slice(0, 1)}</span>
            <div className="w-mast-body">
              <h1 className="w-mast-title">{t('home.title')}</h1>
              <p className="w-mast-lede">{t('home.sub')}</p>
            </div>
            <div className="w-mast-side">
              {spaceEntry && (
                <span className="ml-space-chip" data-testid="ml-space-chip" title={t('home.spaceSwitchHint')}>
                  {spaceLabel}
                </span>
              )}
              <span className="w-mast-pill">{t('home.pill')}</span>
            </div>
          </header>

          {/* 命令台（v9 纯命令台）：deck-head 整块上移 masthead，台内只留气泡流 /
              命令条 / 语音状态——命令台独占主角。 */}
          <section className="w-deck v3-rise v3-rise-2" aria-label={t('shell.launcher.heroAria')}>
            {hasChat && (
              <div className="w-hero-chat" aria-live="polite">
                {userText && <ChatBubble role="user" text={userText} />}
                {aiReply && <ChatBubble role="assistant" text={aiReply} />}
              </div>
            )}

            {/* 命令条：单一抬升面 + focus-within 描边（键盘可达，⌘K 提示） */}
            <div className={`w-cmd v3-rise v3-rise-2 ${voiceTone}`}>
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

            <div className="w-voice-status v3-rise v3-rise-2" aria-label={t('home.voiceStatusAria', { state: voiceStateLabel })}>
              <span className={`w-status-dot${voiceTone ? ` ${voiceTone}` : ''}`} aria-hidden="true" />
              <span className="w-status-label">{voiceStateLabel}</span>
              {voice.error && <span className="w-voice-err" role="alert">{voice.error}</span>}
              {voice.active && voice.aiSpeaking && (
                <button className="w-interrupt-btn" onClick={interrupt} type="button">
                  <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
                </button>
              )}
            </div>
          </section>

          {/* ═══ 账页 × 目录 双列（v9：账页左=文档驱动主角，目录右=旗舰横带 +
              案牌网格，两段异质内容并排消扫视疲劳）═══ */}
          <div className="w-columns">
            {/* 最近文档账页（书斋主角：文档驱动；localStorage 单源，零新 binding） */}
            <section className="w-ledger v3-rise v3-rise-2" aria-label={t('home.recentDocs')} data-testid="desk-recent-docs">
              <SectionLabel icon={<FileTextOutlined />} title={t('home.recentDocs')} />
              {data.recentFiles.length > 0 ? (
                <ul className="w-ledger-list">
                  {data.recentFiles.map((f, i) => (
                    <li key={`${f.path}:${i}`} className="w-ledger-row">
                      <button
                        type="button"
                        className="w-ledger-btn"
                        title={t('home.recentDocsHint')}
                        aria-label={`${f.name || f.path} — ${t('home.recentDocsHint')}`}
                        onClick={() => onNavigate('gaea')}
                      >
                        <span className="w-ledger-idx" aria-hidden="true">{pad2(i + 1)}</span>
                        <span className="w-ledger-name">{f.name || f.path}</span>
                        <span className="w-ledger-path">{f.path}</span>
                        <ArrowRightOutlined className="w-ledger-arrow" aria-hidden="true" />
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="w-empty">
                  <FileTextOutlined className="w-empty-icon" aria-hidden="true" />
                  <span className="w-empty-text">{t('home.recentDocsEmpty')}</span>
                </div>
              )}
            </section>

            {/* 能力目录（v9 案牌）：旗舰横带（序号水印）不变；行式索引改案牌网格——
                徽记 + 名称/描述 + 箭头，去序号去逐项 animationDelay，铺排更疏朗 */}
            <section className="w-cap" aria-label={t('home.capTitle')}>
              <div className="v3-rise v3-rise-3">
                <SectionLabel icon={<ThunderboltOutlined />} title={t('home.capTitle')} sub={t('home.capSub')} />
              </div>
              {featuredModule && (
                <FeaturedBand
                  m={featuredModule}
                  watermark={pad2(1)}
                  onOpen={() => onNavigate(featuredModule.key)}
                />
              )}
              <div className="w-plaques">
                {indexModules.map((m) => (
                  <Plaque key={m.key} m={m} onOpen={() => onNavigate(m.key)} />
                ))}
                {settingsModule && (
                  <Plaque m={settingsModule} onOpen={() => onNavigate(settingsModule.key)} />
                )}
                {indexModules.length === 0 && !featuredModule && !settingsModule && (
                  <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
                )}
              </div>
            </section>
          </div>
        </div>

        {/* ═══ 右舷（v9 脉息面板）：晨报置顶（今天该做什么）+ w-vitals 单一脉息
            面板收拢四节——写作进度升为第一节（图形锚点），内核 / 会话 / 记忆随后 ═══ */}
        <aside className="w-rail v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          {/* 做梦 2.0 晨报（纯本地主动预取）：仅书斋渲染（只读 work 空间记忆）。 */}
          <MorningBriefCard />
          <div className="w-vitals">
            <section className="w-vital" aria-label={t('shell.launcher.statWriting')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><FileTextOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.statWriting')}</span>
              </div>
              <WritingRing data={data} />
            </section>

            <section className="w-vital" aria-label={t('home.kernel')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><ApiOutlined /></span>
                <span className="ml-sec-title">{t('home.kernel')}</span>
              </div>
              <TelemetryBody data={data} />
            </section>

            <section className="w-vital" aria-label={t('shell.launcher.sessions')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><ClockCircleOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.sessions')}</span>
              </div>
              <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
            </section>

            <section className="w-vital" aria-label={t('shell.launcher.memoryPulse')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><HeartOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.memoryPulse')}</span>
              </div>
              <MemoryPulse memoryHub={data.memoryHub} />
            </section>

            {/* 7.3-1 任务收件箱：第五节——待处理计数 + 打开收件箱（结构对齐既有四节） */}
            <section className="w-vital" aria-label={t('shell.launcher.taskInbox')} data-testid="desk-task-inbox">
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><CheckSquareOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.taskInbox')}</span>
              </div>
              <TaskInboxEntry space={space} tick={inboxTick} onOpen={onOpenTaskInbox} />
            </section>
          </div>
        </aside>
      </div>
    </div>
  )
}

/** 旗舰横带（书斋：序号水印 + 徽记 + 名称/描述 + 进入；横带而非卡片海） */
const FeaturedBand: React.FC<{
  m: LauncherModule
  watermark: string
  onOpen: () => void
}> = ({ m, watermark, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <button
      type="button"
      aria-label={t('shell.launcher.enterWorkbench', { name: m.name })}
      className="w-flagship v3-rise"
      onClick={onOpen}
    >
      <span className="w-flagship-num" aria-hidden="true">{watermark}</span>
      <span className="w-flagship-seam" aria-hidden="true" />
      <span className="w-flagship-icon" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="w-flagship-body">
        <span className="w-flagship-badge">{t('home.featured')}</span>
        <span className="w-flagship-name">{m.name}</span>
        <span className="w-flagship-desc">{m.desc}</span>
      </span>
      <span className="w-flagship-cta">
        {t('shell.launcher.enterWorkbench', { name: m.name })}
        <ArrowRightOutlined className="w-flagship-arrow" aria-hidden="true" />
      </span>
    </button>
  )
}

/** 案牌（v9：徽记 + 名称/描述 + 箭头；替代行式索引项，网格铺排，不带序号） */
const Plaque: React.FC<{ m: LauncherModule; onOpen: () => void }> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <button type="button" className="w-plaque v3-rise" onClick={onOpen}
      aria-label={t('shell.launcher.enterModule', { name: m.name })}>
      <span className="w-plaque-icon" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="w-plaque-body">
        <span className="w-plaque-name">{m.name}</span>
        <span className="w-plaque-desc">{m.desc}</span>
      </span>
      <ArrowRightOutlined className="w-plaque-arrow" aria-hidden="true" />
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
      <div className="p-wrap">
        <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

        {/* 标题区：月洞门细环 + 居中标尺 + 大字（画廊入口感，不靠卡片堆） */}
        <section className="p-hero" aria-label={t('home.title')}>
          <span className="p-gate" aria-hidden="true">
            <span className="p-gate-inner" />
          </span>
          <div className="p-kicker v3-rise v3-rise-1">
            <span className="p-kicker-rule" aria-hidden="true" />
            <span className="p-kicker-text">{spaceEntry ? (t(spaceEntry.labelKey) || spaceEntry.label) : ''}</span>
            <span className="p-kicker-rule" aria-hidden="true" />
          </div>
          <h1 className="p-display v3-rise v3-rise-1">{t('home.playTitle')}</h1>
          <p className="p-lede v3-rise v3-rise-1">{t('home.playSub')}</p>
        </section>

        {featuredModule && <GardenBanner m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />}

        {/* 海报墙：首张跨格大样（2×2），其余单格；<1180px 降两列，<760px 单列 */}
        <section className="p-wall" aria-label={t('home.capTitle')}>
          {galleryModules.map((m, i) => (
            <GardenPoster key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
          ))}
          {settingsModule && (
            <GardenPoster m={settingsModule} idx={galleryModules.length} onOpen={() => onNavigate(settingsModule.key)} compact />
          )}
          {galleryModules.length === 0 && !featuredModule && (
            <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
          )}
        </section>

        {/* 园底单条信息带：创作进度 + 继续话题 + 记忆 + 遥测（hairline 分节，信息全保留） */}
        <section className="p-foot v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <div className="p-foot-sec" data-testid="garden-progress" aria-label={t('shell.launcher.statWriting')}>
            <WritingRing data={data} />
          </div>

          <div className="p-foot-sec" data-testid="garden-sessions" aria-label={t('shell.launcher.sessions')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><ClockCircleOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.sessions')}</span>
            </div>
            <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} chips />
          </div>

          <div className="p-foot-sec" data-testid="garden-memory" aria-label={t('shell.launcher.memoryPulse')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><HeartOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.memoryPulse')}</span>
            </div>
            <MemoryPulse memoryHub={data.memoryHub} />
          </div>

          <div className="p-foot-sec" data-testid="garden-meters" aria-label={t('home.kernel')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><ApiOutlined /></span>
              <span className="ml-sec-title">{t('home.kernel')}</span>
            </div>
            <TelemetryBody data={data} />
          </div>

          {/* 7.3-1 任务收件箱：第五节。p-foot 为固定四列网格（module-launcher.css
              .p-foot grid-template-columns 210px/1.35fr/190px/1.35fr），CSS 文件不在
              本刀足迹——内联跨全列（gridColumn 1/-1 全断点成立）+ 顶 hairline 分隔，
              既有四节（garden-progress/sessions/memory/meters）位置与布局零变化。 */}
          <div
            className="p-foot-sec"
            data-testid="garden-task-inbox"
            aria-label={t('shell.launcher.taskInbox')}
            style={{ gridColumn: '1 / -1', paddingLeft: 0, borderLeft: 'none', borderTop: '1px solid var(--w-line-soft)', marginTop: 4, paddingTop: 14 }}
          >
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><CheckSquareOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.taskInbox')}</span>
            </div>
            <TaskInboxEntry space={space} tick={inboxTick} onOpen={onOpenTaskInbox} row />
          </div>
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
      className="garden-banner v3-rise v3-rise-2"
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
      className={`p-poster v3-rise${hero ? ' is-hero' : ''}${compact ? ' is-compact' : ''}`}
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
 * 既有回调）；两空间的收件箱挂点（书斋 w-vitals 第五节 / 闲庭 p-foot 第五节）
 * 都只改本组件内 inboxOpen 状态。inboxTick 在面板收起时 +1，驱动挂点计数
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
