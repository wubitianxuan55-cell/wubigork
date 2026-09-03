/**
 * ModuleLauncher — 首页「星枢港 · 双舷驾驶舱」（v4 重构）
 *
 * 设计概念（遵循 ui-ux-pro-max AI-Native UI 范式 + design-system/gaea 星枢令牌）：
 *   · 左舷主工作台：紧凑 Hero（公告 pill + 标题 + 副标题）+ 中央命令条
 *     （AI 内核 orb / 打字 / 语音 / 发送 / ⌘K）+ 能力矩阵 Bento——
 *     办公为 4×2 旗舰大卡，其余板块 + 编程（独立窗口徽标）+ 设置全部瓦片化；
 *     v3 的快捷 chips 与门廊条是同目标二级入口，收敛进 Bento 一级面（零功能删除）；
 *   · 右舷状态栏：内核遥测（模型 / 引擎 / CPU·内存·GPU 三表）+ 写作进度环
 *     + 最近会话 + 记忆脉搏 + 做梦晨报（work 空间）——v3 的状态细条与
 *     底部信息条合并至此，一屏尽收；
 *   · 动效：v3-rise 分阶入场 + hover 位移 ≤2px（compositor-only），
 *     reduced-motion / ui-reduced-motion / gaea-raf-degraded 全降级。
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
  /** S2.1 壳层空间（晨报仅 work 空间渲染） */
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

/** Bento 瓦片（能力矩阵普通卡：图标 + 名称 + 描述 + 悬浮箭头 + 可选徽标） */
const BentoCard: React.FC<{
  m: LauncherModule
  idx: number
  onOpen: () => void
  badge?: React.ReactNode
  ariaLabel?: string
}> = ({ m, idx, onOpen, badge, ariaLabel }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={ariaLabel ?? t('shell.launcher.enterModule', { name: m.name })}
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
        {badge}
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
 * ModuleLauncher — 首页「星枢港 · 双舷驾驶舱」。
 * 布局：左舷 = 紧凑 Hero（pill / 标题 / 副标题 / 命令条 / 语音状态）
 *       + 能力矩阵 Bento（办公旗舰 4×2 + 板块/编程/设置瓦片）；
 *       右舷 = 内核遥测 + 写作进度环 + 最近会话 + 记忆脉搏 +（work）晨报。
 * 中庭输入：打字 → VoiceChatText；语音 → 本页直启麦克风；共用同一对话流。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space }) => {
  // ── 板块清单 ──
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // 全量启动器模块（manifest 驱动）；gaea 为旗舰大卡，code/settings 以瓦片编入矩阵尾部
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC)
  const featuredModule = allModules.find((m) => m.key === 'gaea')
  const bentoModules = allModules.filter((m) => m.key !== 'gaea' && m.key !== 'code' && m.key !== 'settings')
  const codeModule = allModules.find((m) => m.key === 'code')
  const settingsModule = allModules.find((m) => m.key === 'settings')
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

  // ── 内核遥测数据 ──
  const ms = monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : null
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : null
  const cpuPct = ms && ms.cpu != null && ms.cpu >= 0 ? Math.round(ms.cpu) : null
  const gpuPct = (ms?.gpuUsage ?? 0) > 0 ? Math.round(ms?.gpuUsage ?? 0) : vramPct
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

  return (
    <div className="ml">
      <div className="ml-dock">
        {/* ═══ 左舷：Hero 中轴 + 能力矩阵 ═══ */}
        <div className="ml-main">
          <section className="ml-hero" aria-label={t('shell.launcher.heroAria')}>
            <div className="ml-hero-head v3-rise v3-rise-1">
              <div className="ml-pill">
                <span className="ml-pill-dot" aria-hidden="true" />
                <span>{t('home.pill')}</span>
                <ArrowRightOutlined className="ml-pill-arrow" aria-hidden="true" />
              </div>
              <h1 className="ml-title">{t('home.title')}</h1>
              <p className="ml-sub">{t('home.sub')}</p>
            </div>

            {/* 对话气泡流（有对话时浮于命令条上方） */}
            {hasChat && (
              <div className="ml-hero-chat" aria-live="polite">
                {userText && <ChatBubble role="user" text={userText} />}
                {aiReply && <ChatBubble role="assistant" text={aiReply} />}
              </div>
            )}

            {/* 中央命令条：AI 内核 orb + 打字 + 语音 + 发送 + ⌘K */}
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

            {/* AI 状态行（语音态 + 错误） */}
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

          {/* ═══ 能力矩阵（Bento：办公旗舰 4×2 + 板块/编程/设置瓦片）═══ */}
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
              {codeModule && (
                <BentoCard
                  key={codeModule.key}
                  m={codeModule}
                  idx={bentoModules.length}
                  badge={<span className="ml-bento-badge">{t('shell.rail.independentWindow')}</span>}
                  ariaLabel={t('shell.launcher.progEntry', { name: codeModule.name })}
                  onOpen={() => onNavigate(codeModule.key)}
                />
              )}
              {settingsModule && (
                <BentoCard
                  key={settingsModule.key}
                  m={settingsModule}
                  idx={bentoModules.length + 1}
                  onOpen={() => onNavigate(settingsModule.key)}
                />
              )}
              {bentoModules.length === 0 && !featuredModule && !codeModule && !settingsModule && (
                <div className="ml-col-empty v3-rise">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>
        </div>

        {/* ═══ 右舷：状态侧栏（内核遥测 / 写作进度 / 会话 / 记忆 / 晨报）═══ */}
        <aside className="ml-side v3-rise v3-rise-3" aria-label={t('home.sideAria')}>
          <SidePanel icon={<ApiOutlined />} title={t('home.kernel')}>
            <div className="ml-panel-body">
              <KernelRow
                icon={<RobotOutlined />}
                label={t('shell.launcher.statModel')}
                value={<span className="ml-krow-strong">{activeModel || t('shell.launcher.statModelNone')}</span>}
                sub={t('shell.launcher.statModelSub')}
              />
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
                  {monitor?.comfyRunning && (
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
                  <span className="ml-ring-num">{stats ? `${progressPercent}%` : '—'}</span>
                </div>
              </div>
              <div className="ml-progress-body">
                {stats
                  ? t('shell.launcher.statWritingSub', { chapters: stats.chapterCount, words: fmtWords(stats.totalWords, t) })
                  : (projectOpen ? t('shell.launcher.statLoading') : t('shell.launcher.statNoProject'))}
              </div>
            </div>
          </SidePanel>

          <SidePanel icon={<ClockCircleOutlined />} title={t('shell.launcher.sessions')}>
            {sessions.length > 0 ? (
              <ul className="ml-sess">
                {sessions.map((s, i) => (
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
            {memoryHub ? (
              <div className="ml-memory">
                <span className="ml-krow-strong">{t('shell.launcher.memoryCount', { count: memoryTotal })}</span>
                <span className="ml-sess-meta">{t('shell.launcher.memoryUpdated', { time: fmtRel(memoryUpdated, t) })}</span>
              </div>
            ) : (
              <div className="ml-panel-empty">{t('shell.launcher.memoryIdle')}</div>
            )}
          </SidePanel>

          {/* 做梦 2.0 晨报（纯本地主动预取）：仅 work 空间渲染——play 不渲染
              = 双空间红线（晨报只读 work 空间记忆，见 MorningBriefCard）。 */}
          {space === 'work' && <MorningBriefCard />}
        </aside>
      </div>
    </div>
  )
}

export default ModuleLauncher
