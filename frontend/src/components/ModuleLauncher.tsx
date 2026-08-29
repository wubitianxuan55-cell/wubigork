/**
 * ModuleLauncher — 首页「双空间任务指挥中心」
 * 三栏布局（v4.3.2 重构）：
 *   · 左栏「书房」：work + shared 板块卡（办公/造价/记忆/模型）
 *   · 中栏：语音晶核（放大 orb + 状态 + 对话气泡，视觉焦点）
 *   · 右栏「庭院」：play + shared 板块卡（聊天/小说/绘梦/角色）
 *   · 底部横条：编程（独立窗口入口，不并入两栏）+ 设置
 * 沿用既有视觉体系：深空玻璃拟态、gaea-glow、orb 呼吸环、v3-rise。
 */
import React, { useState, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined,
  StopOutlined, RobotOutlined, UserOutlined, DashboardOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  EditOutlined, ToolOutlined, CodeOutlined, SettingOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import type { ShellSpace } from '../boards/space'
import { Tooltip } from 'antd'
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

/** 单张模块卡片（v3-card + aurora 水印 + 进入箭头微交互） */
const LauncherCard: React.FC<{
  m: LauncherModule
  idx: number
  featured: boolean
  onOpen: () => void
}> = ({ m, idx, featured, onOpen }) => {
  // icon 为图标注册表名，渲染处查表解析；未知名 → Thunderbolt 兜底（3.0 §5.2）
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={t('shell.launcher.enterModule', { name: m.name })}
      className={`ml-card v3-card is-interactive v3-rise ${featured ? 'ml-card--featured' : ''}`}
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
      {featured && (
        <div className="ml-card-foot">
          <span>{t('shell.launcher.enterWorkbench', { name: m.name })}</span>
          <ArrowRightOutlined className="ml-card-foot-arrow" />
        </div>
      )}
    </div>
  )
}

/** 语音对话气泡 */
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
 * ModuleLauncher — 首页「双空间任务指挥中心」。
 * 布局：顶部精简 Hero（eyebrow + 标题 + 副标题）＋ AI 状态细条
 * ｜ 三栏主体（左书房 / 中语音晶核 / 右庭院）＋ 底部编程/设置入口条
 * ｜ 底部信息条（会话/记忆/系统）。
 * 语音交互保留：中栏 orb 下方按钮本页直启麦克风（不跳转）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space }) => {
  // ── 板块清单（清单化数据源，3.0 §5.2 附 B #10/#11）──
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // 三栏分组：书房 = work + shared（模型中心归书房，避免 shared 两栏重复）；
  // 庭院 = play 板块（聊天/小说/绘梦/角色）；设置移到底部横条。
  const studyModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, 'work')
    .filter((m) => m.key !== 'settings') // 设置移到底部横条
  const gardenModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, 'play')
    .filter((m) => m.key !== 'settings' && m.key !== 'modelcenter') // shared 模型中心归书房，避免重复
  // 编程独立窗口入口（manifest.space=independent，不进两栏）
  const codeModule = deriveLauncherModules(activeBoards, LAUNCHER_DESC)
    .find((m) => m.key === 'code')
  const t = useT()

  // ── 项目统计（写作进度；appStore 已由壳层加载，只读消费）──
  const stats = useAppStore((s) => s.stats)
  const projectOpen = useAppStore((s) => s.projectOpen)

  // ── 遥测：引擎 + 资源（轮询 GetModelMonitor，对齐底栏遥测数据源）──
  const [monitor, setMonitor] = useState<ModelMonitor | null>(null)
  // 系统级后台轮询治理：页面不可见（窗口最小化/切走）时轮询空转零成本
  const pollable = usePollingGate()
  useEffect(() => {
    let alive = true
    const load = async () => {
      if (!pollable) return
      try {
        const m = (await App.GetModelMonitor()) as ModelMonitor
        if (alive) setMonitor(m)
      } catch (_) { /* 后端未就绪时静默，等待下一次轮询 */ }
    }
    load()
    const timer = window.setInterval(load, 3000)
    return () => { alive = false; window.clearInterval(timer) }
  }, [pollable])

  // ── 最近会话（办公工作区真实数据）──
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
      } catch (_) { /* 静默：工作区未初始化时无最近会话 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  // ── 记忆脉搏（记忆中枢聚合总览，真实数据）──
  const [memoryHub, setMemoryHub] = useState<MemoryHubLite | null>(null)
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const o = await App.GaeaMemoryHubOverview()
        if (alive) setMemoryHub(o as MemoryHubLite)
      } catch (_) { /* 静默：记忆中枢未就绪 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  // ── 语音交互（本页直启麦克风）──
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')
  const { state: voice, start, stop, interrupt } = useVoiceChat({
    onTranscript: (txt) => { setUserText(txt); setAiReply('') },
    onReply: (txt) => setAiReply(txt),
  })

  // 首页语音固定使用核心人格 gaea（与聊天板块解耦，不跟随聊天所选人格）
  const voicePersonaLabel = 'gaea'

  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    // 首页语音始终使用 gaea
    try { await App.VoiceApplySettings?.({ personalityPresetId: 'gaea' }) } catch (_) {}
    setUserText('')
    setAiReply('')
    await start()
  }, [voice.active, start, stop])

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

  // ── 状态卡数据（全部来自真实遥测/统计，不造假）──
  const ms = monitor?.stats
  const memPct = ms?.memTotal ? Math.round((ms.memUsed || 0) / ms.memTotal * 100) : 0
  const vramPct = ms?.vramTotal ? Math.round((ms.vramUsed || 0) / ms.vramTotal * 100) : 0
  // 后端未采样时 cpu/gpu 可能为 -1（哨兵），统一按「未就绪」显示 --
  const cpuVal = ms && ms.cpu != null && ms.cpu >= 0 ? ms.cpu : null
  const gpuVal = (ms?.gpuUsage ?? 0) > 0 ? (ms?.gpuUsage ?? 0) : vramPct
  const engines = monitor?.engines || []
  const engineCount = engines.length
  const localCount = engines.filter((e) => e.isLocal).length

  // 项目写作进度（与壳层 StatusBar 同公式：已写章节 / 规划章节保守估算）
  const plannedChapters = stats?.chapterCount ? Math.max(stats.chapterCount, stats.plannedChapters || 0) : 0
  const writtenChapters = stats?.chapterCount || 0
  const progressPercent = plannedChapters > 0 ? Math.round((writtenChapters / Math.max(plannedChapters, writtenChapters + 5)) * 100) : 0

  // 记忆总数 + 最近更新时间
  const memoryTotal = memoryHub
    ? (memoryHub.knowledgeCount || 0) + (memoryHub.profileCount || 0) + (memoryHub.officeCount || 0)
      + (memoryHub.costCount || 0) + (memoryHub.whisperCount || 0) + (memoryHub.pinnedCount || 0)
    : 0
  const memoryUpdated = memoryHub?.latestUpdated ? Date.parse(memoryHub.latestUpdated) : 0

  // 当前空间高亮侧（书房/庭院）
  const studyActive = space === 'work'
  const gardenActive = space === 'play'

  return (
    <div className="ml">
      <div className="ml-shell">
        {/* ── Hero：eyebrow + 大标题 + 副标题（双空间门面共用）── */}
        <section className="ml-hero ml-hero--slim" aria-label={t('shell.launcher.heroAria')}>
          <div className="ml-hero-copy">
            <span className="ml-hero-eyebrow">
              <span className="ml-hero-dot" aria-hidden="true" />
              {studyActive
                ? t('shell.hero.work.eyebrow')
                : t('shell.hero.play.eyebrow')}
              <ArrowRightOutlined className="ml-hero-eyebrow-arrow" aria-hidden="true" />
            </span>
            <h1 className="ml-hero-title">{t('shell.launcher.homeTitle')}</h1>
            <p className="ml-hero-sub">{t('shell.launcher.homeSub')}</p>
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

        {/* ── 主体：三栏（左书房 / 中语音晶核 / 右庭院）── */}
        <div className="ml-main">
          {/* 左栏 · 书房（work + shared 板块） */}
          <section className="ml-col" aria-label={t('shell.space.work')}>
            <div className={`ml-col-head${studyActive ? ' is-active' : ''}`}>
              <ToolOutlined aria-hidden="true" />
              <span>{t('shell.space.work')}</span>
              <span className="ml-col-head-desc">{t('shell.launcher.studyHint')}</span>
            </div>
            <div className="ml-col-cards">
              {studyModules.map((m, i) => (
                <LauncherCard key={m.key} m={m} idx={i} featured={false} onOpen={() => onNavigate(m.key)} />
              ))}
              {studyModules.length === 0 && (
                <div className="ml-col-empty">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>

          {/* 中栏 · 语音晶核（放大 orb，视觉焦点） */}
          <section className="ml-voice" aria-label={t('shell.launcher.voiceKernel', { state: voiceStateLabel })}>
            <div className={`ml-voice-panel v3-rise v3-rise-2 ${voiceTone}`}>
              <div className="ml-visual-grid" aria-hidden="true" />
              <div className="ml-visual-glow" aria-hidden="true" />
              <div className="ml-voice-main">
                <div className="ml-orb ml-orb--lg">
                  <span className="ml-orb-ring" aria-hidden="true" />
                  <span className="ml-orb-static" aria-hidden="true" />
                </div>
                <span className="ml-voice-eyebrow" aria-hidden="true">{t('shell.launcher.voiceOrb', { persona: voicePersonaLabel })}</span>
              </div>
              <div className="ml-voice-side">
                <div className="ml-visual-head">
                  <span className="ml-visual-title">{t('shell.launcher.voiceKernel', { state: voiceStateLabel })}</span>
                  <span className="ml-visual-model">
                    <span className="ml-visual-model-dot" aria-hidden="true" />
                    {activeModel || t('shell.launcher.localModel')}
                  </span>
                </div>
                <p className="ml-visual-sub">{t('shell.launcher.voiceSub')}</p>
                <div className="ml-voice-actions">
                  {voice.active && voice.aiSpeaking && (
                    <button className="ml-interrupt-btn" onClick={interrupt} type="button">
                      <StopOutlined /> {t('shell.launcher.voiceInterrupt')}
                    </button>
                  )}
                  <Tooltip title={voice.active ? t('shell.launcher.voiceAriaEnd') : t('shell.launcher.voiceStartTip')}>
                    <button
                      className={`ml-voice-btn ${voice.active ? 'is-active' : ''}`}
                      onClick={toggleVoice}
                      type="button"
                      aria-label={voice.active ? t('shell.launcher.voiceAriaEnd') : t('shell.launcher.voiceAriaStart')}
                    >
                      {voice.active ? <StopOutlined /> : <AudioOutlined />}
                      {voice.active ? t('shell.launcher.voiceEnd') : t('shell.launcher.voiceStart')}
                    </button>
                  </Tooltip>
                </div>
                {voice.error && (
                  <div className="ml-voice-err" role="alert">{voice.error}</div>
                )}
                {hasChat && (
                  <div className="ml-voice-chat">
                    {userText && <ChatBubble role="user" text={userText} />}
                    {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                  </div>
                )}
              </div>
            </div>
          </section>

          {/* 右栏 · 庭院（play + shared 板块） */}
          <section className="ml-col" aria-label={t('shell.space.play')}>
            <div className={`ml-col-head${gardenActive ? ' is-active' : ''}`}>
              <EditOutlined aria-hidden="true" />
              <span>{t('shell.space.play')}</span>
              <span className="ml-col-head-desc">{t('shell.launcher.gardenHint')}</span>
            </div>
            <div className="ml-col-cards">
              {gardenModules.map((m, i) => (
                <LauncherCard key={m.key} m={m} idx={i} featured={false} onOpen={() => onNavigate(m.key)} />
              ))}
              {gardenModules.length === 0 && (
                <div className="ml-col-empty">{t('shell.launcher.noModules')}</div>
              )}
            </div>
          </section>
        </div>

        {/* ── 底部入口条：编程（独立窗口）+ 设置 ── */}
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
