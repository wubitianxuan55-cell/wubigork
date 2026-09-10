/**
 * ModuleLauncher — 双空间首页（v6 重排版：书斋 / 闲庭，编辑部级排印）
 *
 * v4.183 用户反馈 v4.182 版式「太 low」——同质卡片海（8 等大 Bento 瓦片 +
 * 右舷五连盒）是模板感/廉价感的根源。重排版式（信息零删除，仅形态分化）：
 *   · 书斋（work）「文书台」：文房编辑部方向——竖排空间名书脊 + 大字 masthead
 *     + Hero 命令条（AI 打字/语音直启）+ 最近文档卷宗流水（报表式 hairline 行）
 *     + 旗舰横带 + 目录式索引清单（序号大字 + 行 hover 辉光，替代等大瓦片）
 *     + 右栏单一仪表纵栏（遥测/写作/会话/记忆/晨报 hairline 分节，不再五连盒）。
 *   · 闲庭（play）「游园画廊」：画廊展示方向（variance 8 / density 3）——居中大字标题 +
 *     会客厅旗舰横幅（月洞门圆环母题）+ 竖版海报画廊大卡（圆形徽记章 +
 *     hover 环形箭头）+ 园底单条信息带（进度/继续话题/记忆/遥测 hairline 分节）。
 * 零功能删除：遥测/写作/会话/记忆两空间均可达；晨报仍仅书斋（work 记忆红线）、
 * 最近文档仅书斋（work 语义）。数据层单源：useLauncherData 一次拉取。
 * 动效沿用 v3-rise 分阶 + hover ≤2px；reduced-motion / rAF 降级全兼容。
 * 令牌纪律：零硬编码色值，全部走 --md-sys-* / --gaea-* / --color-* / --v3-*。
 */
import React, { useState, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, RobotOutlined, UserOutlined, ThunderboltOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, LAUNCHER_FEATURED, type LauncherModule } from '../boards/launcher'
import { SHELL_SPACES, type ShellSpace } from '../boards/space'
import { loadRecentFiles } from '../gaea/lib/recentFiles'
import type { AtEntry } from '../gaea/lib/types'
import { Input } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { usePollingGate } from '../hooks/usePollingGate'
import { useT, type Translator } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'
import { getModelMonitor } from '../api/engines'
import MorningBriefCard from '../gaea/components/MorningBriefCard'
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

/** 编辑部式节眉：图标 + 宽字距小标 + 渐隐细线 +（可选）副题 */
const Eyebrow: React.FC<{
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

/** 内核遥测单行（图标 + 标签/数值/副文） */
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

/** 资源三表单行（标签 + 细轨 + 数值；≥85% 转 warning 色，色/值双传达） */
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
        style={{ background: `conic-gradient(var(--gaea-glow) ${progressPercent}%, var(--color-border) 0)` }}
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

/** 会话列表内容（书斋右栏行式 / 闲庭信息带 chips 式由外层形态类切换） */
const SessionList: React.FC<{ sessions: SessionLite[]; onOpen: () => void; chips?: boolean }> = ({ sessions, onOpen, chips }) => {
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
interface LauncherData {
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
  const [recentFiles, setRecentFiles] = useState<AtEntry[]>([])
  useEffect(() => {
    setRecentFiles(loadRecentFiles().slice(0, 6))
  }, [])

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

// ─── 顶栏：空间切换器（v4.182 从 rail 迁入首页）──────────────────
// 1B 未拍板前置（长期规划阶段一）：壳层开关=界面导航，与办公引擎空间
// （办公侧栏 SpaceChip，写 session.space）是两回事——title 说清楚，防
// 「切了闲庭办公工具就没了」的误解；行为变更待拍板。
const SpaceSwitch: React.FC<{
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
}> = ({ space, onSwitchSpace, activeModel }) => {
  const t = useT()
  return (
    <div className="ml-topbar v3-rise v3-rise-1">
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
      <span className="ml-topbar-spacer" aria-hidden="true" />
      {activeModel && (
        <span className="ml-topbar-model" title={t('shell.launcher.statModel')}>
          <RobotOutlined aria-hidden="true" /> {activeModel}
        </span>
      )}
    </div>
  )
}

// ─── 书斋（work）：文书台 ────────────────────────────────────────
// 竖排书脊 masthead + Hero 命令条 + 卷宗流水 + 旗舰横带 + 目录式索引 ｜
// 右栏单一仪表纵栏（遥测/写作/会话/记忆/晨报 hairline 分节）。
const DeskHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
}> = ({ data, onNavigate, space, onSwitchSpace, activeModel }) => {
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
    try {
      app.VoiceChatText(text).catch(() => {})
    } catch { /* 后端未就绪 */ }
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
    <div className="ml ml-desk">
      <div className="ml-dock">
        {/* ═══ 左舷：顶栏 + masthead + 卷宗流水 + 索引 ═══ */}
        <div className="ml-main">
          <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

          <section className="desk-mast" aria-label={t('shell.launcher.heroAria')}>
            {/* 竖排空间名书脊（编辑部签名；信息与切换器重复，aria-hidden） */}
            <div className="desk-spine" aria-hidden="true">
              <span className="desk-spine-text">{spaceLabel}</span>
              <span className="desk-spine-rule" />
            </div>
            <div className="desk-mast-body">
              <div className="ml-hero-tags v3-rise v3-rise-1">
                <div className="ml-pill">
                  <span className="ml-pill-dot" aria-hidden="true" />
                  <span>{t('home.pill')}</span>
                  <ArrowRightOutlined className="ml-pill-arrow" aria-hidden="true" />
                </div>
                {spaceEntry && (
                  <span className="ml-space-chip" data-testid="ml-space-chip" title={t('home.spaceSwitchHint')}>
                    {spaceLabel}
                  </span>
                )}
              </div>
              <h1 className="ml-title v3-rise v3-rise-1">{t('home.title')}</h1>
              <p className="ml-sub v3-rise v3-rise-1">{t('home.sub')}</p>

              {hasChat && (
                <div className="ml-hero-chat" aria-live="polite">
                  {userText && <ChatBubble role="user" text={userText} />}
                  {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                </div>
              )}

              <div className={`ml-command v3-rise v3-rise-2 ${voiceTone}`}>
                <span className="ml-command-orb" aria-hidden="true">
                  <span className="ml-command-orb-core" />
                  <span className="ml-command-orb-ring" />
                </span>
                <Input
                  value={typedText}
                  onChange={(e) => setTypedText(e.target.value)}
                  onPressEnter={() => sendTyped()}
                  placeholder={t('home.placeholder')}
                  aria-label={t('home.placeholder')}
                  variant="borderless"
                  className="ml-command-input"
                  disabled={voice.active}
                />
                {voice.active ? (
                  <button
                    type="button"
                    className="ml-voice-btn is-active"
                    onClick={toggleVoice}
                    aria-label={t('shell.launcher.voiceAriaEnd')}
                  >
                    <StopOutlined /> <span className="ml-voice-btn-label">{t('shell.launcher.voiceEnd')}</span>
                  </button>
                ) : (
                  <button
                    type="button"
                    className="ml-voice-btn"
                    onClick={toggleVoice}
                    aria-label={t('shell.launcher.voiceAriaStart')}
                  >
                    <AudioOutlined /> <span className="ml-voice-btn-label">{t('shell.launcher.voiceStart')}</span>
                  </button>
                )}
                <button
                  type="button"
                  className="ml-command-send"
                  onClick={sendTyped}
                  disabled={!typedText.trim() || voice.active}
                  aria-label={t('shell.launcher.courtyardSend')}
                >
                  <SendOutlined />
                </button>
                <kbd className="ml-cmdk" title={t('home.cmdk')} aria-label={t('home.cmdk')}>⌘K</kbd>
              </div>

              <div className="ml-voice-status v3-rise v3-rise-2" aria-label={t('home.voiceStatusAria', { state: voiceStateLabel })}>
                <span className={`ml-voice-status-dot${voiceTone ? ` ${voiceTone}` : ''}`} aria-hidden="true" />
                <span className="ml-voice-status-label">{voiceStateLabel}</span>
                {voice.error && <span className="ml-voice-err" role="alert">{voice.error}</span>}
                {voice.active && voice.aiSpeaking && (
                  <button className="ml-interrupt-btn" onClick={interrupt} type="button">
                    <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
                  </button>
                )}
              </div>
            </div>
          </section>

          {/* ═══ 最近文档卷宗（书斋主角：文档驱动；localStorage 单源，零新 binding）═══ */}
          <section className="desk-docs v3-rise v3-rise-2" aria-label={t('home.recentDocs')} data-testid="desk-recent-docs">
            <Eyebrow icon={<FileTextOutlined />} title={t('home.recentDocs')} sub={t('home.recentDocsHint')} />
            {data.recentFiles.length > 0 ? (
              <ul className="ml-docs-list">
                {data.recentFiles.map((f, i) => (
                  <li
                    key={`${f.path}:${i}`}
                    className="ml-docs-row"
                    role="button"
                    tabIndex={0}
                    title={t('home.recentDocsHint')}
                    aria-label={`${f.name || f.path} — ${t('home.recentDocsHint')}`}
                    onClick={() => onNavigate('gaea')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        onNavigate('gaea')
                      }
                    }}
                  >
                    <span className="ml-docs-idx" aria-hidden="true">{String(i + 1).padStart(2, '0')}</span>
                    <span className="ml-docs-name">{f.name || f.path}</span>
                    <span className="ml-docs-path">{f.path}</span>
                    <ArrowRightOutlined className="ml-docs-arrow" aria-hidden="true" />
                  </li>
                ))}
              </ul>
            ) : (
              <div className="ml-desk-docs-empty">{t('home.recentDocsEmpty')}</div>
            )}
          </section>

          {/* ═══ 能力目录：旗舰横带 + 目录式索引（序号行替代等大瓦片）═══ */}
          <section className="desk-index" aria-label={t('home.capTitle')}>
            <div className="v3-rise v3-rise-3">
              <Eyebrow icon={<ThunderboltOutlined />} title={t('home.capTitle')} sub={t('home.capSub')} />
            </div>
            {featuredModule && (
              <FeaturedCard m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />
            )}
            <div className="desk-index-list">
              {indexModules.map((m, i) => (
                <IndexRow key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
              ))}
              {settingsModule && (
                <IndexRow key={settingsModule.key} m={settingsModule} idx={indexModules.length} onOpen={() => onNavigate(settingsModule.key)} />
              )}
              {indexModules.length === 0 && !featuredModule && !settingsModule && (
                <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>
        </div>

        {/* ═══ 右栏：单一仪表纵栏（遥测 / 写作 / 会话 / 记忆 / 晨报，hairline 分节）═══ */}
        <aside className="ml-side v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <div className="ml-rail">
            <section className="ml-sec" aria-label={t('home.kernel')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><ApiOutlined /></span>
                <span className="ml-sec-title">{t('home.kernel')}</span>
              </div>
              <TelemetryBody data={data} />
            </section>

            <section className="ml-sec" aria-label={t('shell.launcher.statWriting')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><FileTextOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.statWriting')}</span>
              </div>
              <WritingRing data={data} />
            </section>

            <section className="ml-sec" aria-label={t('shell.launcher.sessions')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><ClockCircleOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.sessions')}</span>
              </div>
              <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
            </section>

            <section className="ml-sec" aria-label={t('shell.launcher.memoryPulse')}>
              <div className="ml-sec-head">
                <span className="ml-sec-icon" aria-hidden="true"><HeartOutlined /></span>
                <span className="ml-sec-title">{t('shell.launcher.memoryPulse')}</span>
              </div>
              <MemoryPulse memoryHub={data.memoryHub} />
            </section>

            {/* 做梦 2.0 晨报（纯本地主动预取）：仅书斋渲染（只读 work 空间记忆）。 */}
            <MorningBriefCard />
          </div>
        </aside>
      </div>
    </div>
  )
}

/** 旗舰横带（办公 · 能力目录锚点：accent 竖条 + 徽记 + 名称/描述 + 进入） */
const FeaturedCard: React.FC<{
  m: LauncherModule
  onOpen: () => void
}> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={t('shell.launcher.enterWorkbench', { name: m.name })}
      className={`desk-featured v3-card is-interactive v3-rise`}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      <span className="desk-featured-grid" aria-hidden="true" />
      <div className="desk-featured-icon">{Icon ? <Icon /> : null}</div>
      <div className="desk-featured-body">
        <div className="desk-featured-badge">{t('home.featured')}</div>
        <div className="desk-featured-name">{m.name}</div>
        <div className="desk-featured-desc">{m.desc}</div>
      </div>
      <span className="desk-featured-enter">
        {t('shell.launcher.enterWorkbench', { name: m.name })}
        <ArrowRightOutlined className="desk-featured-arrow" />
      </span>
    </div>
  )
}

/** 目录式索引行（书斋：序号 + 徽记 + 名称 + 描述 + 悬浮箭头；替代等大瓦片） */
const IndexRow: React.FC<{
  m: LauncherModule
  idx: number
  onOpen: () => void
}> = ({ m, idx, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={t('shell.launcher.enterModule', { name: m.name })}
      className="desk-row v3-rise"
      style={{ animationDelay: `${220 + idx * 45}ms` } as React.CSSProperties}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="desk-row-num" aria-hidden="true">{String(idx + 1).padStart(2, '0')}</span>
      <span className="desk-row-icon" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="desk-row-name">{m.name}</span>
      <span className="desk-row-desc">{m.desc}</span>
      <ArrowRightOutlined className="desk-row-arrow" aria-hidden="true" />
    </div>
  )
}

// ─── 闲庭（play）：游园画廊 ──────────────────────────────────────
// 居中大字 + 月洞门旗舰横幅 + 竖版海报画廊 + 园底单条信息带。
// 无右栏无命令条——场景由画廊选择进入，信息单行化收于园底。
const GardenHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
}> = ({ data, onNavigate, space, onSwitchSpace, activeModel }) => {
  const t = useT()
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, space)
  const featuredModule = allModules.find((m) => m.key === LAUNCHER_FEATURED[space])
  const gardenCards = allModules.filter((m) => m.key !== LAUNCHER_FEATURED[space] && m.key !== 'settings')
  const settingsModule = allModules.find((m) => m.key === 'settings')
  const spaceEntry = SHELL_SPACES.find((s) => s.id === space)

  return (
    <div className="ml ml-garden">
      <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

      {/* 画廊英雄区：空间名小签 + 居中大字 + 月洞门旗舰横幅 */}
      <section className="garden-hero" aria-label={t('home.title')}>
        <div className="garden-hero-head v3-rise v3-rise-1">
          {spaceEntry && (
            <span className="garden-hero-tag" aria-hidden="true">{t(spaceEntry.labelKey) || spaceEntry.label}</span>
          )}
          <h1 className="ml-title">{t('home.title')}</h1>
          <p className="ml-sub">{t('home.sub')}</p>
        </div>
        {featuredModule && <GardenBanner m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />}
      </section>

      {/* 画廊两列海报大卡（松密度：圆徽记 + 大标题 + 描述 + hover 环形箭头） */}
      <section className="garden-gallery" aria-label={t('home.capTitle')}>
        {gardenCards.map((m, i) => (
          <GardenCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
        ))}
        {settingsModule && (
          <GardenCard m={settingsModule} idx={gardenCards.length} onOpen={() => onNavigate(settingsModule.key)} compact />
        )}
        {gardenCards.length === 0 && !featuredModule && (
          <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
        )}
      </section>

      {/* 园底单条信息带：创作进度 + 继续话题 + 记忆 + 遥测细条（hairline 分节，信息全保留） */}
      <section className="garden-foot v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
        <div className="garden-strip">
          <div className="garden-strip-sec" data-testid="garden-progress" aria-label={t('shell.launcher.statWriting')}>
            <WritingRing data={data} />
          </div>

          <div className="garden-strip-sec garden-strip-wide" data-testid="garden-sessions" aria-label={t('shell.launcher.sessions')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><ClockCircleOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.sessions')}</span>
            </div>
            <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} chips />
          </div>

          <div className="garden-strip-sec" data-testid="garden-memory" aria-label={t('shell.launcher.memoryPulse')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><HeartOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.memoryPulse')}</span>
            </div>
            <MemoryPulse memoryHub={data.memoryHub} />
          </div>

          <div className="garden-strip-sec garden-strip-wide" data-testid="garden-meters" aria-label={t('home.kernel')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><ApiOutlined /></span>
              <span className="ml-sec-title">{t('home.kernel')}</span>
            </div>
            <TelemetryBody data={data} />
          </div>
        </div>
      </section>
    </div>
  )
}

/** 闲庭旗舰横幅（会客厅：月洞门圆环母题 + 渐变大卡 + 超大徽记 + 进入箭头） */
const GardenBanner: React.FC<{ m: LauncherModule; onOpen: () => void }> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={t('shell.launcher.enterWorkbench', { name: m.name })}
      className="garden-banner v3-card is-interactive v3-rise v3-rise-2"
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      <span className="garden-banner-gate" aria-hidden="true" />
      <div className="garden-banner-icon">{Icon ? <Icon /> : null}</div>
      <div className="garden-banner-body">
        <div className="garden-banner-badge">{t('home.featured')}</div>
        <div className="garden-banner-name">{m.name}</div>
        <div className="garden-banner-desc">{m.desc}</div>
      </div>
      <span className="garden-banner-cta">
        {t('shell.launcher.enterWorkbench', { name: m.name })}
        <ArrowRightOutlined />
      </span>
    </div>
  )
}

/** 闲庭海报大卡（竖版：圆徽记 + 名称 + 描述 + 底部环形箭头；compact = 行形态） */
const GardenCard: React.FC<{
  m: LauncherModule
  idx: number
  onOpen: () => void
  compact?: boolean
}> = ({ m, idx, onOpen, compact }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={t('shell.launcher.enterModule', { name: m.name })}
      className={`garden-card v3-card is-interactive v3-rise${compact ? ' garden-card--compact' : ''}`}
      style={{ animationDelay: `${180 + idx * 60}ms` } as React.CSSProperties}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      <div className="garden-card-icon">{Icon ? <Icon /> : null}</div>
      <div className="garden-card-name">{m.name}</div>
      {!compact && <div className="garden-card-desc">{m.desc}</div>}
      {!compact && (
        <div className="garden-card-foot" aria-hidden="true">
          <span className="garden-card-go"><ArrowRightOutlined /></span>
        </div>
      )}
      {compact && <ArrowRightOutlined className="garden-card-arrow" aria-hidden="true" />}
    </div>
  )
}

/**
 * ModuleLauncher — 首页入口：数据一次拉取，按壳层空间分发书斋 / 闲庭变体。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space, onSwitchSpace }) => {
  const data = useLauncherData()
  return space === 'work' ? (
    <DeskHome data={data} onNavigate={onNavigate} space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />
  ) : (
    <GardenHome data={data} onNavigate={onNavigate} space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />
  )
}

export default ModuleLauncher
