/**
 * ModuleLauncher — 双空间首页（v5 重构：书斋 / 闲庭，各有版式）
 *
 * v4.182 用户拍板：空间切换器从 rail 迁到首页顶栏（更顺手）；两首页要「各有特色，
 * 不要长得一样」。此前的单版式 Bento（仅按空间过滤瓦片）拆为两个变体：
 *   · 书斋（work）「文书台」：三栏效率台——顶栏切换器 + Hero 命令条（AI 直启）+
 *     最近文档流水（文档驱动主角）+ 能力矩阵 Bento（紧凑）+ 右舷状态栏
 *     （遥测/写作/会话/记忆/晨报）。设计 dial：密度 7 / variance 4（Flat、清单化）。
 *   · 闲庭（play）「游园画廊」：全幅画廊——顶栏切换器 + 会客厅旗舰横幅 +
 *     板块大卡两列（松、大、呼吸感）+ 创作进度 + 继续话题（最近会话 chips）+
 *     仪表细条（遥测单行化）。无右舷、无命令条——场景由画廊选择进入。
 *     设计 dial：密度 4 / variance 7（Showcase、呼吸感）。
 * 零功能删除：遥测/写作/会话/记忆在两空间均可达，仅形态不同；
 * 晨报仍仅书斋（work 记忆红线）、最近文档仅书斋（work 语义）。
 * 数据层单源：useLauncherData 顶层一次拉取，两变体各自选用。
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

/** 侧舷面板壳（玻璃容器：标题行 + 内容槽；信息层实底，不叠玻璃） */
const SidePanel: React.FC<{
  icon: React.ReactNode
  title: string
  children: React.ReactNode
}> = ({ icon, title, children }) => (
  <section className="ml-panel v3-panel" aria-label={title}>
    <div className="ml-panel-head">
      <span className="ml-panel-icon" aria-hidden="true">{icon}</span>
      <span className="ml-panel-title">{title}</span>
    </div>
    {children}
  </section>
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

/**
 * Bento 瓦片（书斋能力矩阵普通卡：图标 + 名称 + 描述 + 悬浮箭头）。
 * 宽瓦片/徽标能力保留供今后非 independent 宽板块复用。
 */
const BentoCard: React.FC<{
  m: LauncherModule
  idx: number
  onOpen: () => void
  badge?: React.ReactNode
  ariaLabel?: string
  wide?: boolean
}> = ({ m, idx, onOpen, badge, ariaLabel, wide }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={ariaLabel ?? t('shell.launcher.enterModule', { name: m.name })}
      className={`ml-bento v3-card is-interactive v3-rise${wide ? ' ml-bento--wide' : ''}`}
      style={{ animationDelay: `${140 + idx * 40}ms` } as React.CSSProperties}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      {wide ? (
        <>
          <div className="ml-bento-icon">{Icon ? <Icon /> : null}</div>
          <div className="ml-bento-wide-body">
            <div className="ml-bento-name">{m.name}</div>
            <div className="ml-bento-desc">{m.desc}</div>
          </div>
          {badge}
          <ArrowRightOutlined className="ml-bento-arrow" />
        </>
      ) : (
        <>
          <div className="ml-bento-top">
            <div className="ml-bento-icon">{Icon ? <Icon /> : null}</div>
            {badge}
            <ArrowRightOutlined className="ml-bento-arrow" />
          </div>
          <div className="ml-bento-name">{m.name}</div>
          <div className="ml-bento-desc">{m.desc}</div>
        </>
      )}
    </div>
  )
}

/** 旗舰大卡（书斋 4×2：办公工作台，能力矩阵锚点） */
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
      className={`ml-bento ml-bento--featured v3-card is-interactive v3-rise`}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      <span className="ml-featured-grid" aria-hidden="true" />
      <div className="ml-featured-badge">{t('home.featured')}</div>
      <div className="ml-featured-icon">{Icon ? <Icon /> : null}</div>
      <div className="ml-featured-name">{m.name}</div>
      <div className="ml-featured-desc">{m.desc}</div>
      <div className="ml-featured-foot">
        <span className="ml-featured-enter">
          {t('shell.launcher.enterWorkbench', { name: m.name })}
          <ArrowRightOutlined className="ml-featured-arrow" />
        </span>
      </div>
    </div>
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
              title={t(s.titleKey) || s.title}
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
// 三栏效率台：Hero 命令条 + 最近文档流水 + 能力矩阵 ｜ 右舷遥测/写作/会话/记忆/晨报。
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
  const bentoModules = allModules.filter((m) => m.key !== LAUNCHER_FEATURED[space] && m.key !== 'settings')
  const settingsModule = allModules.find((m) => m.key === 'settings')
  const spaceEntry = SHELL_SPACES.find((s) => s.id === space)

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

  const ms = data.monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : null
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : null
  const cpuPct = ms && ms.cpu != null && ms.cpu >= 0 ? Math.round(ms.cpu) : null
  const gpuPct = (ms?.gpuUsage ?? 0) > 0 ? Math.round(ms?.gpuUsage ?? 0) : vramPct
  const engines = data.monitor?.engines || []
  const engineCount = engines.length
  const localCount = engines.filter((e) => e.isLocal).length

  const plannedChapters = data.stats?.chapterCount ? Math.max(data.stats.chapterCount, data.stats.plannedChapters || 0) : 0
  const writtenChapters = data.stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  const memoryTotal = data.memoryHub
    ? (data.memoryHub.knowledgeCount || 0) + (data.memoryHub.profileCount || 0) + (data.memoryHub.officeCount || 0)
      + (data.memoryHub.costCount || 0) + (data.memoryHub.whisperCount || 0) + (data.memoryHub.pinnedCount || 0)
    : 0
  const memoryUpdated = data.memoryHub?.latestUpdated ? Date.parse(data.memoryHub.latestUpdated) : 0

  return (
    <div className="ml ml-desk">
      <div className="ml-dock">
        {/* ═══ 左舷：顶栏 + Hero 命令条 + 最近文档流水 + 能力矩阵 ═══ */}
        <div className="ml-main">
          <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

          <section className="ml-hero" aria-label={t('shell.launcher.heroAria')}>
            <div className="ml-hero-head v3-rise v3-rise-1">
              <div className="ml-hero-tags">
                <div className="ml-pill">
                  <span className="ml-pill-dot" aria-hidden="true" />
                  <span>{t('home.pill')}</span>
                  <ArrowRightOutlined className="ml-pill-arrow" aria-hidden="true" />
                </div>
                {spaceEntry && (
                  <span className="ml-space-chip" data-testid="ml-space-chip">
                    {t(spaceEntry.labelKey) || spaceEntry.label}
                  </span>
                )}
              </div>
              <h1 className="ml-title">{t('home.title')}</h1>
              <p className="ml-sub">{t('home.sub')}</p>
            </div>

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
          </section>

          {/* ═══ 最近文档流水（书斋主角：文档驱动；localStorage 单源，零新 binding）═══ */}
          <section className="ml-desk-docs v3-rise v3-rise-2" aria-label={t('home.recentDocs')} data-testid="desk-recent-docs">
            <div className="ml-desk-docs-head">
              <span className="ml-desk-docs-icon" aria-hidden="true"><FileTextOutlined /></span>
              <span className="ml-desk-docs-title">{t('home.recentDocs')}</span>
              <span className="ml-desk-docs-sub">{t('home.recentDocsHint')}</span>
            </div>
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

          {/* ═══ 能力矩阵（Bento：旗舰 4×2 + 板块瓦片 + 设置瓦片）═══ */}
          <section className="ml-cap" aria-label={t('home.capTitle')}>
            <div className="ml-cap-head v3-rise v3-rise-3">
              <span className="ml-cap-title">{t('home.capTitle')}</span>
              <span className="ml-cap-sub">{t('home.capSub')}</span>
            </div>
            <div className="ml-grid">
              {featuredModule && (
                <FeaturedCard m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />
              )}
              {bentoModules.map((m, i) => (
                <BentoCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
              ))}
              {settingsModule && (
                <BentoCard
                  key={settingsModule.key}
                  m={settingsModule}
                  idx={bentoModules.length}
                  onOpen={() => onNavigate(settingsModule.key)}
                />
              )}
              {bentoModules.length === 0 && !featuredModule && !settingsModule && (
                <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>
        </div>

        {/* ═══ 右舷：状态侧栏（遥测 / 写作 / 会话 / 记忆 / 晨报）═══ */}
        <aside className="ml-side v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <SidePanel icon={<ApiOutlined />} title={t('home.kernel')}>
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
          </SidePanel>

          <SidePanel icon={<FileTextOutlined />} title={t('shell.launcher.statWriting')}>
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
          </SidePanel>

          <SidePanel icon={<ClockCircleOutlined />} title={t('shell.launcher.sessions')}>
            {data.sessions.length > 0 ? (
              <ul className="ml-sess">
                {data.sessions.map((s, i) => (
                  <li key={s.modTime ?? i} className="ml-sess-item">
                    <span className="ml-sess-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
                    <span className="ml-sess-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="ml-panel-empty">{t('shell.launcher.noSessions')}</div>
            )}
          </SidePanel>

          <SidePanel icon={<HeartOutlined />} title={t('shell.launcher.memoryPulse')}>
            {data.memoryHub ? (
              <div className="ml-memory">
                <span className="ml-krow-strong">{t('shell.launcher.memoryCount', { count: memoryTotal })}</span>
                <span className="ml-sess-meta">{t('shell.launcher.memoryUpdated', { time: fmtRel(memoryUpdated, t) })}</span>
              </div>
            ) : (
              <div className="ml-panel-empty">{t('shell.launcher.memoryIdle')}</div>
            )}
          </SidePanel>

          {/* 做梦 2.0 晨报（纯本地主动预取）：仅书斋渲染（只读 work 空间记忆）。 */}
          <MorningBriefCard />
        </aside>
      </div>
    </div>
  )
}

// ─── 闲庭（play）：游园画廊 ──────────────────────────────────────
// 全幅画廊：会客厅旗舰横幅 + 板块大卡两列 + 创作进度 + 继续话题 chips +
// 仪表细条。无右舷无命令条——场景由画廊选择进入，信息单行化收于园底。
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

  const ms = data.monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : null
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : null
  const cpuPct = ms && ms.cpu != null && ms.cpu >= 0 ? Math.round(ms.cpu) : null
  const gpuPct = (ms?.gpuUsage ?? 0) > 0 ? Math.round(ms?.gpuUsage ?? 0) : vramPct

  const plannedChapters = data.stats?.chapterCount ? Math.max(data.stats.chapterCount, data.stats.plannedChapters || 0) : 0
  const writtenChapters = data.stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  const memoryTotal = data.memoryHub
    ? (data.memoryHub.knowledgeCount || 0) + (data.memoryHub.profileCount || 0) + (data.memoryHub.officeCount || 0)
      + (data.memoryHub.costCount || 0) + (data.memoryHub.whisperCount || 0) + (data.memoryHub.pinnedCount || 0)
    : 0
  const memoryUpdated = data.memoryHub?.latestUpdated ? Date.parse(data.memoryHub.latestUpdated) : 0

  return (
    <div className="ml ml-garden">
      <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

      {/* 画廊英雄区：标题行 + 会客厅旗舰横幅 */}
      <section className="garden-hero" aria-label={t('home.title')}>
        <div className="garden-hero-head v3-rise v3-rise-1">
          <h1 className="ml-title">{t('home.title')}</h1>
          <p className="ml-sub">{t('home.sub')}</p>
        </div>
        {featuredModule && <GardenBanner m={featuredModule} onOpen={() => onNavigate(featuredModule.key)} />}
      </section>

      {/* 画廊两列大卡（松密度：大图标 + 大标题 + 描述） */}
      <section className="garden-gallery" aria-label={t('home.capTitle')}>
        {gardenCards.map((m, i) => (
          <GardenCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
        ))}
        {gardenCards.length === 0 && !featuredModule && (
          <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
        )}
      </section>

      {/* 园底信息带：创作进度 + 继续话题 + 记忆 + 遥测细条（信息全保留，形态单行化） */}
      <section className="garden-foot v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
        <div className="garden-foot-card" data-testid="garden-progress">
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
        </div>

        <div className="garden-foot-card garden-foot-wide" data-testid="garden-sessions">
          <div className="garden-foot-head">
            <ClockCircleOutlined aria-hidden="true" /> {t('shell.launcher.sessions')}
          </div>
          {data.sessions.length > 0 ? (
            <div className="garden-chips">
              {data.sessions.map((s, i) => (
                <button
                  key={s.modTime ?? i}
                  type="button"
                  className="garden-chip"
                  onClick={() => onNavigate('chat')}
                  title={s.preview || s.title || ''}
                >
                  <span className="garden-chip-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
                  <span className="garden-chip-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
                </button>
              ))}
            </div>
          ) : (
            <div className="ml-panel-empty">{t('shell.launcher.noSessions')}</div>
          )}
        </div>

        <div className="garden-foot-card" data-testid="garden-memory">
          <div className="garden-foot-head">
            <HeartOutlined aria-hidden="true" /> {t('shell.launcher.memoryPulse')}
          </div>
          {data.memoryHub ? (
            <div className="ml-memory">
              <span className="ml-krow-strong">{t('shell.launcher.memoryCount', { count: memoryTotal })}</span>
              <span className="ml-sess-meta">{t('shell.launcher.memoryUpdated', { time: fmtRel(memoryUpdated, t) })}</span>
            </div>
          ) : (
            <div className="ml-panel-empty">{t('shell.launcher.memoryIdle')}</div>
          )}
        </div>

        <div className="garden-foot-card garden-foot-wide" data-testid="garden-meters">
          <div className="garden-foot-head">
            <ApiOutlined aria-hidden="true" /> {t('home.kernel')}
          </div>
          {ms ? (
            <div className="garden-meters">
              <Meter label="CPU" pct={cpuPct} />
              <Meter label="MEM" pct={memPct} />
              <Meter label="GPU" pct={gpuPct} />
            </div>
          ) : (
            <div className="ml-panel-empty">{t('shell.launcher.statIdle')}</div>
          )}
        </div>

        {settingsModule && (
          <GardenCard m={settingsModule} idx={0} onOpen={() => onNavigate(settingsModule.key)} compact />
        )}
      </section>
    </div>
  )
}

/** 闲庭旗舰横幅（会客厅：全宽渐变大卡 + 超大图标 + 进入箭头） */
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

/** 闲庭画廊大卡（两列：松密度、大呼吸感；compact = 园底小卡形态） */
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
      <div className="garden-card-body">
        <div className="garden-card-name">{m.name}</div>
        {!compact && <div className="garden-card-desc">{m.desc}</div>}
      </div>
      <ArrowRightOutlined className="garden-card-arrow" aria-hidden="true" />
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
