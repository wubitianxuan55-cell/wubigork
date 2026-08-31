/**
 * ModuleLauncher — 首页「AI 多功能平台 · 星枢指挥所」（3.0 重构 v3）
 *
 * 设计概念（遵循 ui-ux-pro-max / design-taste-frontend，匹配当代 AI 平台首页范式）：
 *   · Hero 中轴：公告 pill + 巨幅标题 + 副标题 + 中央命令条（打字/语音/发送/⌘K）
 *     + 快捷建议 chips —— 对标 Poe / Perplexity / DeepSeek 的「一句话直达」范式；
 *   · AI 状态细条：活跃模型 / 引擎 / 资源 / 写作进度（真实遥测，4 列）；
 *   · 能力矩阵（Bento）：manifest 驱动，办公（gaea）为 4×2 旗舰大卡锚定全场，
 *     其余板块 2 列等宽瓦片，aurora 水印 + hover 辉光 + 箭头滑入；
 *   · 门廊：编程（独立窗口）+ 设置；底部信息条：最近会话 / 记忆脉搏 / 系统状态。
 *   · 动效：v3-rise 分阶入场 + hover 位移 ≤2px（compositor-only），
 *     reduced-motion / ui-reduced-motion / gaea-raf-degraded 全降级。
 * 令牌纪律：零硬编码色值，全部走 --md-sys-* / --gaea-* / --color-* / --v3-*。
 */
import React, { useState, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, RobotOutlined, UserOutlined, ThunderboltOutlined, DashboardOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  ReadOutlined, PictureOutlined, MessageOutlined, CodeOutlined, SettingOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import type { ShellSpace } from '../boards/space'
import { Input } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { usePollingGate } from '../hooks/usePollingGate'
import { useT, type Translator } from '../gaea/lib/i18n'
import * as App from '../../src/wailsjsCompat'
import MorningBriefCard from '../gaea/components/MorningBriefCard'
import './module-launcher.css'

/**
 * 启动器可跳转的目标页（3.0 §5.2：放宽为 string，由 manifest.id 派生，
 * 不再与 MainLayout 的 Page 字面量联合保持手工同步）。
 */
export type LauncherTarget = string

/** 语音入口信号（首页现在本页启动语音，该信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏已加装，传入提升真实感） */
  activeModel?: string
  /** S2.1 壳层空间（保留 API 兼容；新首页为全局能力矩阵，不再分区高亮） */
  space: ShellSpace
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

/** 单条 AI 状态（透明无边框信息行：图标 + 标签/数值/副文 + 渐次入场 v3-rise） */
const StatCard: React.FC<{
  icon: React.ReactNode
  label: string
  value: React.ReactNode
  sub?: React.ReactNode
  rise: string
}> = ({ icon, label, value, sub, rise }) => (
  <div className={`ml-stat v3-rise ${rise}`}>
    <span className="ml-stat-icon" aria-hidden="true">{icon}</span>
    <div className="ml-stat-body">
      <div className="ml-stat-label">{label}</div>
      <div className="ml-stat-value">{value}</div>
      {sub && <div className="ml-stat-sub">{sub}</div>}
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

/** Bento 瓦片（能力矩阵普通卡：图标 + 名称 + 描述 + 悬浮箭头） */
const BentoCard: React.FC<{
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
      className={`ml-bento v3-card is-interactive v3-rise`}
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
      <div className="ml-bento-top">
        <div className="ml-bento-icon">{Icon ? <Icon /> : null}</div>
        <ArrowRightOutlined className="ml-bento-arrow" />
      </div>
      <div className="ml-bento-name">{m.name}</div>
      <div className="ml-bento-desc">{m.desc}</div>
    </div>
  )
}

/** 旗舰大卡（4×2：办公工作台，能力矩阵锚点） */
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

/**
 * ModuleLauncher — 首页「AI 多功能平台 · 星枢指挥所」。
 * 布局：Hero 中轴（pill / 标题 / 副标题 / 命令条 / 状态 / chips）
 * ｜ AI 状态细条（4 列真实遥测）
 * ｜ 能力矩阵 Bento（办公旗舰 4×2 + 板块瓦片）
 * ｜ 门廊（编程独立窗口 + 设置）+ 底部信息条（会话 / 记忆 / 系统）。
 * 中庭输入：打字 → VoiceChatText；语音 → 本页直启麦克风；共用同一对话流。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space }) => {
  // ── 板块清单 ──
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // 全量启动器模块（manifest 驱动）；gaea 为旗舰大卡，code/settings 入门廊
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC)
  const featuredModule = allModules.find((m) => m.key === 'gaea')
  const bentoModules = allModules.filter((m) => m.key !== 'gaea' && m.key !== 'code' && m.key !== 'settings')
  const codeModule = allModules.find((m) => m.key === 'code')
  const t = useT()

  // ── 项目统计 ──
  const stats = useAppStore((s) => s.stats)
  const projectOpen = useAppStore((s) => s.projectOpen)

  // ── 遥测 ──
  const [monitor, setMonitor] = useState<ModelMonitor | null>(null)
  const pollable = usePollingGate()
  useEffect(() => {
    let alive = true
    const load = async () => {
      if (!pollable) return
      try {
        const m = (await App.GetModelMonitor()) as ModelMonitor
        if (alive) setMonitor(m)
      } catch (_) { /* 后端未就绪时静默 */ }
    }
    load()
    const timer = window.setInterval(load, 3000)
    return () => { alive = false; window.clearInterval(timer) }
  }, [pollable])

  // ── 最近会话 ──
  const [sessions, setSessions] = useState<SessionLite[]>([])
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const list = await App.GaeaListSessions()
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

  // ── 记忆脉搏 ──
  const [memoryHub, setMemoryHub] = useState<MemoryHubLite | null>(null)
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const o = await App.GaeaMemoryHubOverview()
        if (alive) setMemoryHub(o as MemoryHubLite)
      } catch (_) { /* 静默 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  // ── 命令条：语音 + 打字一体 ──
  const [typedText, setTypedText] = useState('')
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')
  const { state: voice, start, stop, interrupt } = useVoiceChat({
    onTranscript: (txt) => { setUserText(txt); setAiReply('') },
    onReply: (txt) => setAiReply(txt),
  })

  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    try { await App.VoiceApplySettings?.({ personalityPresetId: 'gaea' }) } catch (_) {}
    setUserText('')
    setAiReply('')
    await start()
  }, [voice.active, start, stop])

  // 打字发送：复用语音对话管道（VoiceChatText），回复走 voice:reply 事件
  const sendTyped = useCallback(() => {
    const text = typedText.trim()
    if (!text) return
    setUserText(text)
    setAiReply('')
    setTypedText('')
    try {
      App.VoiceChatText(text).catch(() => {})
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

  // ── 状态卡数据 ──
  const ms = monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
  const cpuVal = ms && ms.cpu != null && ms.cpu >= 0 ? ms.cpu : null
  const gpuVal = (ms?.gpuUsage ?? 0) > 0 ? (ms?.gpuUsage ?? 0) : vramPct
  const engines = monitor?.engines || []
  const engineCount = engines.length
  const localCount = engines.filter((e) => e.isLocal).length

  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, stats.plannedChapters || 0) : 0
  const writtenChapters = stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  const memoryTotal = memoryHub
    ? (memoryHub.knowledgeCount || 0) + (memoryHub.profileCount || 0) + (memoryHub.officeCount || 0)
      + (memoryHub.costCount || 0) + (memoryHub.whisperCount || 0) + (memoryHub.pinnedCount || 0)
    : 0
  const memoryUpdated = memoryHub?.latestUpdated ? Date.parse(memoryHub.latestUpdated) : 0

  // ── 快捷建议 chips（直达核心能力）──
  const suggestChips: Array<{ key: string; label: string; icon: React.ReactNode }> = [
    { key: 'novel', label: t('home.suggestNovel'), icon: <ReadOutlined /> },
    { key: 'imagegen', label: t('home.suggestImage'), icon: <PictureOutlined /> },
    { key: 'chat', label: t('home.suggestChat'), icon: <MessageOutlined /> },
    { key: 'code', label: t('home.suggestCode'), icon: <CodeOutlined /> },
  ]

  return (
    <div className="ml">
      <div className="ml-shell">
        {/* ── Hero 中轴：公告 + 标题 + 命令条 + 状态 + chips ── */}
        <section className="ml-hero" aria-label={t('shell.launcher.heroAria')}>
          <div className="ml-pill v3-rise v3-rise-1">
            <span className="ml-pill-dot" aria-hidden="true" />
            <span>{t('home.pill')}</span>
            <ArrowRightOutlined className="ml-pill-arrow" aria-hidden="true" />
          </div>
          <h1 className="ml-title v3-rise v3-rise-2">{t('home.title')}</h1>
          <p className="ml-sub v3-rise v3-rise-3">{t('home.sub')}</p>

          {/* 对话气泡流（有对话时浮于命令条上方） */}
          {hasChat && (
            <div className="ml-hero-chat" aria-live="polite">
              {userText && <ChatBubble role="user" text={userText} />}
              {aiReply && <ChatBubble role="assistant" text={aiReply} />}
            </div>
          )}

          {/* 中央命令条：AI 内核 orb + 打字 + 语音 + 发送 + ⌘K */}
          <div className={`ml-command v3-rise v3-rise-4 ${voiceTone}`}>
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

          {/* AI 状态行（语音态 + 错误） */}
          <div className="ml-voice-status v3-rise v3-rise-4" aria-label={t('home.voiceStatusAria', { state: voiceStateLabel })}>
            <span className={`ml-voice-status-dot${voiceTone ? ` ${voiceTone}` : ''}`} aria-hidden="true" />
            <span className="ml-voice-status-label">{voiceStateLabel}</span>
            {voice.error && <span className="ml-voice-err" role="alert">{voice.error}</span>}
            {voice.active && voice.aiSpeaking && (
              <button className="ml-interrupt-btn" onClick={interrupt} type="button">
                <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
              </button>
            )}
          </div>

          {/* 快捷建议 chips */}
          <div className="ml-chips v3-rise v3-rise-4">
            {suggestChips.map((c) => (
              <button
                key={c.key}
                type="button"
                className="ml-chip"
                aria-label={c.label}
                onClick={() => onNavigate(c.key)}
              >
                <span className="ml-chip-icon" aria-hidden="true">{c.icon}</span>
                {c.label}
              </button>
            ))}
          </div>
        </section>

        {/* ── AI 状态细条（真实遥测/统计，4 列）── */}
        <div className="ml-stats">
          <StatCard
            rise="v3-rise-1"
            icon={<RobotOutlined />}
            label={t('shell.launcher.statModel')}
            value={activeModel || t('shell.launcher.statModelNone')}
            sub={t('shell.launcher.statModelSub')}
          />
          <StatCard
            rise="v3-rise-2"
            icon={<ThunderboltOutlined />}
            label={t('shell.launcher.statEngines')}
            value={engineCount > 0 ? `${engineCount}` : '—'}
            sub={engineCount > 0
              ? t('shell.launcher.statEnginesSub', { local: localCount, cloud: engineCount - localCount })
              : t('shell.launcher.statNoEngines')}
          />
          <StatCard
            rise="v3-rise-3"
            icon={<DashboardOutlined />}
            label={t('shell.launcher.statResource')}
            value={ms ? t('shell.launcher.statResourceValue', { cpu: cpuVal != null ? cpuVal + '%' : '--' }) : '—'}
            sub={ms ? t('shell.launcher.statResourceSub', { mem: memPct, gpu: gpuVal }) : t('shell.launcher.statIdle')}
          />
          <StatCard
            rise="v3-rise-4"
            icon={<FileTextOutlined />}
            label={t('shell.launcher.statWriting')}
            value={stats ? `${progressPercent}%` : '—'}
            sub={stats
              ? t('shell.launcher.statWritingSub', { chapters: stats.chapterCount, words: fmtWords(stats.totalWords, t) })
              : (projectOpen ? t('shell.launcher.statLoading') : t('shell.launcher.statNoProject'))}
          />
        </div>

        {/* ── 能力矩阵（Bento：办公旗舰 4×2 + 板块瓦片）── */}
        <section className="ml-cap" aria-label={t('home.capTitle')}>
          <div className="ml-cap-head v3-rise v3-rise-1">
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
            {bentoModules.length === 0 && !featuredModule && (
              <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
            )}
          </div>
        </section>

        {/* ── 门廊：编程（独立窗口）+ 设置 ── */}
        <div className="ml-utilities v3-panel v3-rise">
          {codeModule ? (
            <button
              type="button"
              className="ml-util-entry"
              aria-label={t('shell.launcher.progEntry', { name: codeModule.name })}
              onClick={() => onNavigate(codeModule.key)}
            >
              <span className="ml-util-icon"><CodeOutlined /></span>
              <span className="ml-util-body">
                <span className="ml-util-name">{codeModule.name}</span>
                <span className="ml-util-desc">{codeModule.desc}</span>
              </span>
              <span className="ml-util-badge">{t('shell.rail.independentWindow')}</span>
              <ArrowRightOutlined className="ml-util-arrow" />
            </button>
          ) : null}
          <button
            type="button"
            className="ml-util-entry ml-util-entry--settings"
            aria-label={t('shell.launcher.openSettings')}
            onClick={() => onNavigate('settings')}
          >
            <span className="ml-util-icon"><SettingOutlined /></span>
            <span className="ml-util-body">
              <span className="ml-util-name">{t('shell.launcher.settings')}</span>
              <span className="ml-util-desc">{t('shell.launcher.settingsDesc')}</span>
            </span>
            <ArrowRightOutlined className="ml-util-arrow" />
          </button>
        </div>

        {/* ── 底部：信息条（最近会话 / 记忆脉搏 / 系统状态）── */}
        <div className="ml-info v3-panel v3-rise">
          <section className="ml-info-seg" aria-label={t('shell.launcher.sessions')}>
            <div className="ml-info-head"><ClockCircleOutlined aria-hidden="true" />{t('shell.launcher.sessions')}</div>
            {sessions.length > 0 ? (
              <ul className="ml-info-list">
                {sessions.map((s, i) => (
                  <li key={s.modTime ?? i} className="ml-info-item">
                    <span className="ml-info-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
                    <span className="ml-info-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="ml-info-empty">{t('shell.launcher.noSessions')}</div>
            )}
          </section>
          <section className="ml-info-seg" aria-label={t('shell.launcher.memoryPulse')}>
            <div className="ml-info-head"><HeartOutlined aria-hidden="true" />{t('shell.launcher.memoryPulse')}</div>
            {memoryHub ? (
              <div className="ml-info-body">
                <span className="ml-info-strong">{t('shell.launcher.memoryCount', { count: memoryTotal })}</span>
                <span className="ml-info-meta">{t('shell.launcher.memoryUpdated', { time: fmtRel(memoryUpdated, t) })}</span>
              </div>
            ) : (
              <div className="ml-info-empty">{t('shell.launcher.memoryIdle')}</div>
            )}
          </section>
          {/* 做梦 2.0 晨报（纯本地主动预取）：仅 work 空间渲染——play 不渲染
              = 双空间红线（晨报只读 work 空间记忆，见 MorningBriefCard）。 */}
          {space === 'work' && <MorningBriefCard />}
          <section className="ml-info-seg" aria-label={t('shell.launcher.sysStatus')}>
            <div className="ml-info-head"><ApiOutlined aria-hidden="true" />{t('shell.launcher.sysStatus')}</div>
            {monitor ? (
              <div className="ml-info-body">
                <span className="ml-info-strong">{t('shell.launcher.sysEngines', { count: engineCount, cpu: cpuVal != null ? cpuVal + '%' : '--' })}</span>
                <span className="ml-info-meta">
                  {t('shell.launcher.sysMeta', { mem: memPct, gpu: gpuVal })}
                  {monitor.comfyRunning ? t('shell.launcher.sysComfy') : ''}
                </span>
              </div>
            ) : (
              <div className="ml-info-empty">{t('shell.launcher.statIdle')}</div>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}

export default ModuleLauncher
