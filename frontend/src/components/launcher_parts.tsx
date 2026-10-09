// launcher_parts.tsx — 首页启动器展示件与数据轮询（批 32 FE6-07 拆分自
// ModuleLauncher.tsx，原位搬移零逻辑改动）：遥测/会话/记忆最小类型、fmt 助手、
// 纯展示子组件（SectionLabel/KernelRow/ChatBubble/WritingRing/MemoryPulse/
// TaskInboxEntry/TelemetryBody/SessionList）+ useLauncherData 轮询 hook。
// 主文件以命名导入回接；外部消费（TasksFirstHome）经 ModuleLauncher.tsx 的
// SessionList/LauncherData re-export 通道，导入路径不变。
import React, { useState, useEffect, useMemo, useSyncExternalStore } from 'react'
import { RobotOutlined, UserOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { type ShellSpace } from '../boards/space'
import { loadRecentFiles, subscribeRecentFiles } from '../gaea/lib/recentFiles'
import type { AtEntry } from '../gaea/lib/types'
import { useT, type Translator } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'
import { getModelMonitor } from '../api/engines'
import { usePollingGate } from '../hooks/usePollingGate'
import { useAppStore } from '../stores/appStore'

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
  /** 列表口径：壳层空间或 ''（首页合一 v4.475：首页挂点传 '' = 跨空间全量） */
  space: ShellSpace | ''
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
      {/* 简化批（v4.473）：CPU/MEM/GPU 仪表条退役——壳层底部遥测轨恒常驻同屏展示
          同三项（MainLayout TelemetryRail），一屏两份属纯重复；仅保留卡内独有的
          ComfyUI 运行中指示。 */}
      {data.monitor?.comfyRunning && (
        <div className="ml-meters">
          <div className="ml-comfy">
            <span className="ml-comfy-dot" aria-hidden="true" />
            <span>{t('home.comfyRunning')}</span>
          </div>
        </div>
      )}
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
export { SectionLabel, ChatBubble, WritingRing, MemoryPulse, TaskInboxEntry, TelemetryBody, useLauncherData }
