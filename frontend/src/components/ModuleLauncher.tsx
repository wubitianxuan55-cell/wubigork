// ModuleLauncher.tsx — 首页「任务指挥中心 Mission Control」（3.0 Bento 网格）
// DeepSeek 风格 Hero：左文右卡（公告 pill + 大标题 + 双行动卡 / 右侧深色渐变 AI 视觉卡）
// ＋ AI 状态细条（活跃模型/引擎/资源/写作进度）＋ 中部模块 Bento 网格
// ＋ 底部信息条（最近会话/记忆脉搏/系统状态）；语音 orb 收进 Hero 右侧视觉卡。
// 模块卡数据源与 onNavigate 跳转逻辑保持不变（boards/launcher 纯函数派生）。
import React, { useState, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined,
  StopOutlined, RobotOutlined, UserOutlined, DashboardOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  EditOutlined, MessageOutlined, ToolOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import type { ShellSpace } from '../boards/space'
import { Tooltip } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { usePollingGate } from '../hooks/usePollingGate'
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
  /** S2.1 壳层空间：双首页按空间装配（docs/gaea-space-shell-design.md §4.5） */
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
function fmtWords(n: number): string {
  if (!n) return '0'
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return n.toLocaleString()
}

/** 相对时间（unix ms → 「刚刚 / N 分钟前 / N 小时前 / N 天前」） */
function fmtRel(ms: number): string {
  if (!ms) return '—'
  const diff = Date.now() - ms
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min} 分钟前`
  const h = Math.floor(min / 60)
  if (h < 24) return `${h} 小时前`
  const d = Math.floor(h / 24)
  if (d < 30) return `${d} 天前`
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

/** 行动引导卡（DeepSeek 式：图标 + 标题 + 描述，整卡可点） */
const HeroActionCard: React.FC<{
  icon: React.ReactNode
  title: string
  desc: string
  onClick: () => void
  rise: string
}> = ({ icon, title, desc, onClick, rise }) => (
  <button
    type="button"
    className={`ml-hero-action v3-card is-interactive v3-rise ${rise}`}
    onClick={onClick}
    aria-label={`${title}：${desc}`}
  >
    <span className="ml-hero-action-icon" aria-hidden="true">{icon}</span>
    <span className="ml-hero-action-body">
      <span className="ml-hero-action-title">{title}</span>
      <span className="ml-hero-action-desc">{desc}</span>
    </span>
    <ArrowRightOutlined className="ml-hero-action-arrow" aria-hidden="true" />
  </button>
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
  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={`进入${m.name}模块`}
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
          <span>进入{m.name}工作台</span>
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
 * ModuleLauncher — 首页「任务指挥中心」。
 * 布局：左上角语音 orb（小尺寸，呼吸 2s）＋ 顶部 AI 状态卡（遥测真实数据）
 * ｜ 中部模块 Bento 网格（onNavigate 跳转）｜ 底部信息条（会话/记忆/系统）。
 * 语音交互保留：点击 orb 下方按钮本页直启麦克风（不跳转）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel, space }) => {
  // ── 板块清单（清单化数据源，3.0 §5.2 附 B #10/#11）──
  // useSyncExternalStore 订阅活动清单：getActiveBoards 返回稳定引用（仅加载成功时
  // 替换），subscribeBoards 通知即重渲染，无手动 setState 竞态。默认未加载 = 静态
  // canonicalBoards（8 卡）；loadBoardManifests 成功并入 knowledge 后自动多出「知识库」卡。
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // S2.1 双首页：模块卡按空间装配（shared + 当前空间）；缺省 work（旧调用语义）
  const modules = deriveLauncherModules(activeBoards, LAUNCHER_DESC, space)
  // 左侧大卡：工位=办公工作台；乐园=小说创作间；缺省取首卡
  const featuredModule = modules.find((m) => m.key === (space === 'work' ? 'gaea' : 'novel')) ?? modules[0]
  const otherModules = modules.filter((m) => m.key !== featuredModule?.key)

  // 空间化文案（工位=任务工作台门面；乐园=会客厅/创作间门面）
  const hero = space === 'work'
    ? {
        eyebrow: 'GAEA 工位已就绪 · 本地 AI 办公中枢',
        title: '把活儿交给我，我来干',
        sub: '工位 = 任务工作台：办公、造价、记忆，都在一个工作台里。',
        primary: { key: 'gaea', title: '进入办公工作台', desc: '委托任务、审阅执行' },
        secondary: { key: 'cost', title: '查造价数据库', desc: '单价、定额与价格源' },
        section: '工位模块',
        sectionHint: '办公 / 造价 / 记忆 / 编程',
      }
    : {
        eyebrow: 'GAEA 乐园已就绪 · 本地 AI 创作乐园',
        title: '从灵光乍现，到星河成篇',
        sub: '乐园 = 会客厅与创作间：轻语、小说、绘梦，沉浸不打扰。',
        primary: { key: 'novel', title: '开始创作', desc: '世界观、角色与大纲' },
        secondary: { key: 'chat', title: '和 gaea 对话', desc: '与 AI 对话，激发灵感' },
        section: '乐园模块',
        sectionHint: '小说 / 绘梦 / 角色 / 会客厅',
      }

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
    const t = window.setInterval(load, 3000)
    return () => { alive = false; window.clearInterval(t) }
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
    onTranscript: (t) => { setUserText(t); setAiReply('') },
    onReply: (t) => setAiReply(t),
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
    ? 'AI 回复中…'
    : voice.listening
      ? '正在聆听…'
      : voice.active
        ? '语音待命'
        : '待机'

  const voiceTone = voice.aiSpeaking
    ? 'is-speaking'
    : voice.listening
      ? 'is-listening'
      : voice.active
        ? 'is-active'
        : ''

  const hasChat = !!userText || !!aiReply

  const heroActions = [
    { ...hero.primary, onClick: () => onNavigate(hero.primary.key) },
    { ...hero.secondary, onClick: () => onNavigate(hero.secondary.key) },
  ]

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

  return (
    <div className="ml">
      <div className="ml-shell">
        {/* ── Hero：左文右卡（参照 DeepSeek 首页风格）── */}
        <section className="ml-hero" aria-label="首页概览">
          {/* 左侧：公告 pill + 大标题 + 行动卡 */}
          <div className="ml-hero-copy">
            <span className="ml-hero-eyebrow">
              <span className="ml-hero-dot" aria-hidden="true" />
              {hero.eyebrow}
              <ArrowRightOutlined className="ml-hero-eyebrow-arrow" aria-hidden="true" />
            </span>
            <h1 className="ml-hero-title">{hero.title}</h1>
            <p className="ml-hero-sub">{hero.sub}</p>
            <div className="ml-hero-actions">
              <HeroActionCard
                rise="v3-rise-1"
                icon={heroActions[0].key === 'gaea' ? <ToolOutlined /> : <EditOutlined />}
                title={heroActions[0].title}
                desc={heroActions[0].desc}
                onClick={heroActions[0].onClick}
              />
              <HeroActionCard
                rise="v3-rise-2"
                icon={<MessageOutlined />}
                title={heroActions[1].title}
                desc={heroActions[1].desc}
                onClick={heroActions[1].onClick}
              />
            </div>
          </div>

          {/* 右侧：深色渐变视觉卡（语音晶核 · AI 在线） */}
          <div className={`ml-hero-visual v3-rise v3-rise-2 ${voiceTone}`}>
            <div className="ml-visual-grid" aria-hidden="true" />
            <div className="ml-visual-glow" aria-hidden="true" />
            <div className="ml-visual-main">
              <div className="ml-orb">
                <span className="ml-orb-ring" aria-hidden="true" />
                <span className="ml-orb-static" aria-hidden="true" />
              </div>
              <span className="ml-voice-eyebrow" aria-hidden="true">语音晶核 · {voicePersonaLabel}</span>
            </div>
            <div className="ml-visual-side">
              <div className="ml-visual-head">
                <span className="ml-visual-title">AI 内核 · {voiceStateLabel}</span>
                <span className="ml-visual-model">
                  <span className="ml-visual-model-dot" aria-hidden="true" />
                  {activeModel || '本地模型'}
                </span>
              </div>
              <p className="ml-visual-sub">说话即可交互，无需打字 — 用声音直接指挥你的创作工作台。</p>
              <div className="ml-voice-actions">
                {voice.active && voice.aiSpeaking && (
                  <button className="ml-interrupt-btn" onClick={interrupt} type="button">
                    <StopOutlined /> 打断
                  </button>
                )}
                <Tooltip title={voice.active ? '结束语音对话' : '启动麦克风，开始语音交互'}>
                  <button
                    className={`ml-voice-btn ${voice.active ? 'is-active' : ''}`}
                    onClick={toggleVoice}
                    type="button"
                    aria-label={voice.active ? '结束语音对话' : '启动语音对话'}
                  >
                    {voice.active ? <StopOutlined /> : <AudioOutlined />}
                    {voice.active ? '结束对话' : '开始语音对话'}
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

        {/* ── AI 状态细条（真实遥测/统计，4 列）── */}
        <div className="ml-stats">
          <StatCard
            rise="v3-rise-1"
            icon={<RobotOutlined />}
            label="活跃模型"
            value={activeModel || '未设置'}
            sub="当前对话模型"
          />
          <StatCard
            rise="v3-rise-2"
            icon={<ThunderboltOutlined />}
            label="已启用引擎"
            value={engineCount > 0 ? `${engineCount}` : '—'}
            sub={engineCount > 0 ? `${localCount} 本地 · ${engineCount - localCount} 云端` : '暂无引擎运行'}
          />
          <StatCard
            rise="v3-rise-3"
            icon={<DashboardOutlined />}
            label="资源占用"
            value={ms ? `CPU ${cpuVal != null ? cpuVal + '%' : '--'}` : '—'}
            sub={ms ? `内存 ${memPct}% · GPU ${gpuVal}%` : '遥测待机'}
          />
          <StatCard
            rise="v3-rise-4"
            icon={<FileTextOutlined />}
            label="项目写作进度"
            value={stats ? `${progressPercent}%` : '—'}
            sub={stats ? `${stats.chapterCount} 章 · ${fmtWords(stats.totalWords)} 字` : (projectOpen ? '统计加载中…' : '未打开项目')}
          />
        </div>

        {/* ── 中部：模块卡片 Bento 网格（数据源/跳转逻辑不变）── */}
        <section className="ml-section" aria-label="全部模块">
          <div className="ml-section-head">
            <h2>{hero.section}</h2>
            <span>{hero.sectionHint}</span>
          </div>
          <div className="ml-bento">
            {featuredModule && (
              <div className="ml-bento-side">
                <LauncherCard key={featuredModule.key} m={featuredModule} idx={0} featured onOpen={() => onNavigate(featuredModule.key)} />
              </div>
            )}
            <div className="ml-bento-grid">
              {otherModules.map((m, i) => (
                <LauncherCard key={m.key} m={m} idx={i + 1} featured={false} onOpen={() => onNavigate(m.key)} />
              ))}
            </div>
          </div>
        </section>

        {/* ── 底部：信息条（最近会话 / 记忆脉搏 / 系统状态）── */}
        <div className="ml-info v3-panel v3-rise">
          <section className="ml-info-seg" aria-label="最近会话">
            <div className="ml-info-head"><ClockCircleOutlined aria-hidden="true" />最近会话</div>
            {sessions.length > 0 ? (
              <ul className="ml-info-list">
                {sessions.map((s, i) => (
                  <li key={s.modTime ?? i} className="ml-info-item">
                    <span className="ml-info-name">{s.title || s.preview || '未命名会话'}</span>
                    <span className="ml-info-meta">{s.turns ?? 0} 轮 · {fmtRel(s.modTime || 0)}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="ml-info-empty">暂无最近会话</div>
            )}
          </section>
          <section className="ml-info-seg" aria-label="记忆脉搏">
            <div className="ml-info-head"><HeartOutlined aria-hidden="true" />记忆脉搏</div>
            {memoryHub ? (
              <div className="ml-info-body">
                <span className="ml-info-strong">{memoryTotal} 条记忆</span>
                <span className="ml-info-meta">最近更新 {fmtRel(memoryUpdated)}</span>
              </div>
            ) : (
              <div className="ml-info-empty">记忆中枢待命</div>
            )}
          </section>
          <section className="ml-info-seg" aria-label="系统状态">
            <div className="ml-info-head"><ApiOutlined aria-hidden="true" />系统状态</div>
            {monitor ? (
              <div className="ml-info-body">
                <span className="ml-info-strong">引擎 {engineCount} · CPU {cpuVal != null ? cpuVal + '%' : '--'}</span>
                <span className="ml-info-meta">
                  内存 {memPct}% · GPU {gpuVal}%
                  {monitor.comfyRunning ? ' · ComfyUI 运行中' : ''}
                </span>
              </div>
            ) : (
              <div className="ml-info-empty">遥测待机</div>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}

export default ModuleLauncher
