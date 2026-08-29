/**
 * ModuleLauncher — 首页「双翼 · 中庭」
 * 设计概念（v4.3.2b，遵循 design-taste-frontend 原则）：
 *   · 中庭（中央 · 磁吸核心）：语音 + 打字一体对话条 + 大 orb，视觉权重最高
 *   · 左翼「书房」：work 板块，2×2 紧凑格，冷静克制（青蓝案头感）
 *   · 右翼「庭院」：play 板块，纵向列表 + 预览感，温暖生动（粉紫院落感）
 *   · 门廊（底部）：编程（独立窗口，独立之门）+ 设置
 * 不对称原则：两翼组织方式不同（格 vs 列表），避免机械镜像；hero 让位给中庭。
 * 动效：中庭先浮现 → 两翼如门扇从两侧滑入（transform/opacity，reduced-motion 降级）。
 * 沿用既有视觉体系：深空玻璃拟态、gaea-glow、orb 呼吸环、v3-rise、--color-* 令牌。
 */
import React, { useState, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, RobotOutlined, UserOutlined, DashboardOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  EditOutlined, ToolOutlined, CodeOutlined, SettingOutlined,
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
  /** S2.1 壳层空间：当前空间（决定「书房/庭院」哪侧高亮与 hero 文案） */
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

/** 模块卡（两翼通用：图标 + 标题 + 描述；左右翼 CSS 区分性格） */
const LauncherCard: React.FC<{
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
      className={`ml-card v3-card is-interactive v3-rise`}
      style={{ animationDelay: `${80 + idx * 45}ms` } as React.CSSProperties}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen()
        }
      }}
    >
      <span className="ml-card-aurora" aria-hidden="true" />
      <div className="ml-card-icon">{Icon ? <Icon /> : <ThunderboltOutlined />}</div>
      <div className="ml-card-body">
        <div className="ml-card-name">
          {m.name}
          <ArrowRightOutlined className="ml-card-arrow" />
        </div>
        <div className="ml-card-desc">{m.desc}</div>
      </div>
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
 * ModuleLauncher — 首页「双翼 · 中庭」。
 * 布局：顶部细眉（空间状态）＋ AI 状态细条
 * ｜ 三区主体：左翼书房（2×2 格）/ 中庭对话（orb + 一体输入条 + 气泡）/ 右翼庭院（纵向列）
 * ｜ 门廊：编程（独立窗口）+ 设置
 * ｜ 底部信息条（会话/记忆/系统）。
 * 中庭输入：打字 → VoiceChatText；语音 → 本页直启麦克风；共用同一对话流。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space }) => {
  // ── 板块清单 ──
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // 书房 = work + shared（模型中心归书房）；庭院 = play；设置入门廊
  const studyModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, 'work')
    .filter((m) => m.key !== 'settings')
  const gardenModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, 'play')
    .filter((m) => m.key !== 'settings' && m.key !== 'modelcenter')
  const codeModule = deriveLauncherModules(activeBoards, LAUNCHER_DESC)
    .find((m) => m.key === 'code')
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

  // ── 中庭对话：语音 + 打字一体 ──
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

  const studyActive = space === 'work'
  const gardenActive = space === 'play'

  return (
    <div className="ml">
      <div className="ml-shell">
        {/* ── 顶部细眉：空间状态（低调，视觉权重让给中庭）── */}
        <div className="ml-masthead" aria-label={t('shell.launcher.heroAria')}>
          <span className="ml-masthead-eyebrow">
            <span className="ml-hero-dot" aria-hidden="true" />
            {studyActive ? t('shell.hero.work.eyebrow') : t('shell.hero.play.eyebrow')}
          </span>
          <span className="ml-masthead-title">{t('shell.launcher.homeTitle')}</span>
          <span className="ml-masthead-hint">{t('shell.launcher.homeSub')}</span>
        </div>

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

        {/* ── 主体：双翼 · 中庭 ── */}
        <div className="ml-main">
          {/* 左翼 · 书房（2×2 紧凑格，冷静克制） */}
          <section className="ml-wing ml-wing--study" aria-label={t('shell.space.work')}>
            <div className={`ml-wing-head${studyActive ? ' is-active' : ''}`}>
              <ToolOutlined aria-hidden="true" />
              <span>{t('shell.space.work')}</span>
              <span className="ml-wing-head-desc">{t('shell.launcher.studyHint')}</span>
            </div>
            <div className="ml-wing-grid">
              {studyModules.map((m, i) => (
                <LauncherCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
              ))}
              {studyModules.length === 0 && (
                <div className="ml-col-empty">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>

          {/* 中庭 · 对话（磁吸核心：orb + 一体输入条 + 气泡） */}
          <section className="ml-courtyard" aria-label={t('shell.launcher.voiceKernel', { state: voiceStateLabel })}>
            <div className={`ml-courtyard-panel v3-rise v3-rise-2 ${voiceTone}`}>
              <div className="ml-visual-grid" aria-hidden="true" />
              <div className="ml-visual-glow" aria-hidden="true" />
              {/* 对话气泡流（先于 orb 显示，回复可见） */}
              {hasChat && (
                <div className="ml-courtyard-chat" aria-live="polite">
                  {userText && <ChatBubble role="user" text={userText} />}
                  {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                </div>
              )}
              {/* orb + 状态 */}
              <div className="ml-courtyard-orb">
                <div className="ml-orb ml-orb--lg">
                  <span className="ml-orb-ring" aria-hidden="true" />
                  <span className="ml-orb-static" aria-hidden="true" />
                </div>
                <span className="ml-courtyard-orb-label">
                  {t('shell.launcher.voiceKernel', { state: voiceStateLabel })}
                </span>
                <span className="ml-courtyard-model">
                  <span className="ml-visual-model-dot" aria-hidden="true" />
                  {activeModel || t('shell.launcher.localModel')}
                </span>
              </div>
              {/* 一体输入条：打字 + 语音 */}
              <div className="ml-courtyard-input">
                <Input
                  value={typedText}
                  onChange={(e) => setTypedText(e.target.value)}
                  onPressEnter={() => sendTyped()}
                  placeholder={t('shell.launcher.courtyardPlaceholder')}
                  aria-label={t('shell.launcher.courtyardPlaceholder')}
                  variant="borderless"
                  className="ml-courtyard-input-field"
                  disabled={voice.active}
                />
                {voice.active ? (
                  <button
                    type="button"
                    className="ml-voice-btn is-active"
                    onClick={toggleVoice}
                    aria-label={t('shell.launcher.voiceAriaEnd')}
                  >
                    <StopOutlined /> {t('shell.launcher.voiceEnd')}
                  </button>
                ) : (
                  <button
                    type="button"
                    className="ml-voice-btn"
                    onClick={toggleVoice}
                    aria-label={t('shell.launcher.voiceAriaStart')}
                  >
                    <AudioOutlined /> {t('shell.launcher.voiceStart')}
                  </button>
                )}
                <button
                  type="button"
                  className="ml-courtyard-send"
                  onClick={sendTyped}
                  disabled={!typedText.trim() || voice.active}
                  aria-label={t('shell.launcher.courtyardSend')}
                >
                  <SendOutlined />
                </button>
              </div>
              {voice.error && (
                <div className="ml-voice-err" role="alert">{voice.error}</div>
              )}
              {voice.active && voice.aiSpeaking && (
                <button className="ml-interrupt-btn" onClick={interrupt} type="button">
                  <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
                </button>
              )}
            </div>
          </section>

          {/* 右翼 · 庭院（纵向列表，温暖生动） */}
          <section className="ml-wing ml-wing--garden" aria-label={t('shell.space.play')}>
            <div className={`ml-wing-head${gardenActive ? ' is-active' : ''}`}>
              <EditOutlined aria-hidden="true" />
              <span>{t('shell.space.play')}</span>
              <span className="ml-wing-head-desc">{t('shell.launcher.gardenHint')}</span>
            </div>
            <div className="ml-wing-list">
              {gardenModules.map((m, i) => (
                <LauncherCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
              ))}
              {gardenModules.length === 0 && (
                <div className="ml-col-empty">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>
        </div>

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
