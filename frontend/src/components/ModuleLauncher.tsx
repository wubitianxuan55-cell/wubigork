/**
 * ModuleLauncher — 统一首页（v10「天玄」卡片工作台；v4.475 书斋/闲庭合一）
 * 首页不再按壳层空间分裂版式：一张「文书台」承载全部板块（能力矩阵全量出卡，
 * 不再按 work/play 过滤），收件箱跨空间全量（列表 scope=''；新建仍按当前壳层
 * 空间落标签，Go ValidSpace 仅收 work|play）。壳层空间继续存在——rail 分域、
 * 事件分面、每空间最后页面不动；切空间完全由板块导航隐式驱动（MainLayout
 * v4.3.2c navigateBoard 自动切换），首页顶栏不再摆放手动切换钮。
 *
 * 设计判据（v6「等大圆角卡片 + 描边」被判廉价的根因，v10 逐条避坑）：
 *   1. 卡片形态必须异质：全宽指挥卡 / 宽文档卡 / 侧列小卡 / 跨列旗舰卡 / 模块卡 /
 *      状态卡排——没有两排是等大瓦片。
 *   2. 层级靠四档排印（display / lede / body / label）+ 数据等宽；标题走负字距或
 *      中式衬线，label 走宽字距，绝不靠 13↔14px 微差。
 *   3. 深度＝双层工艺（外壳细线 + 双档投影，内芯再收一圈强调色细线）+ 色调面，
 *      描边只做 hairline；悬停抬升只给可点卡片。
 *   4. 装饰一律「版式记号」：印章 / 徽记水印 / 顶缘月华描线；不用极光斑。
 *   5. 动效走弹性曲线（--ml-ease），只动 transform/box-shadow，reduced-motion 与
 *      gaea-raf-degraded 全降级。
 *   6. 主题母题＝「天玄」：玄穹墨蓝底 + 月华金强调 + 壳层星穹透出（theme preset
 *      tianXuan 见 stores/appStore.ts；其余预设共用同一套卡片骨架）。
 *
 * 契约保持（测试与壳层依赖，勿改）：desk-recent-docs、desk-task-inbox、
 * task-inbox-open-btn。数据层单源 useLauncherData，遥测/写作/会话/记忆均可达
 * （晨报卡 2026-10-01 起撤出首页，组件保留）。闲庭画廊版式（GardenHome 系）
 * 随合一退役（v4.475）。令牌纪律：零硬编码色值，全部走 --color-* / --md-sys-* /
 * --gaea-* / --v3-*。
 */
import React, { useState, useCallback, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, ThunderboltOutlined,
  FileTextOutlined, ClockCircleOutlined, HeartOutlined, ApiOutlined,
  CheckSquareOutlined,
} from '@ant-design/icons'
// 板块清单：活动清单（静态 fallback / 后端合并）订阅驱动；图标由 manifest 图标注册表解析（3.0 §5.2）
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import TasksFirstHome from './TasksFirstHome'
import { requestSessionResume } from '../gaea/lib/pendingSessionResume'
import { openPaneFileOrPreview } from '../gaea/lib/paneFileOpen'
import { Input } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { useT } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'
import { TaskInboxPanel } from '../gaea/components/TaskInboxPanel'
import { SectionLabel, ChatBubble, WritingRing, MemoryPulse, TaskInboxEntry, TelemetryBody, SessionList, useLauncherData, type LauncherData } from './launcher_parts'
import './module-launcher.css'

export { SessionList }
export type { LauncherData }


/**
 * 启动器可跳转的目标页（3.0 §5.2：放宽为 string，由 manifest.id 派生）。
 */
export type LauncherTarget = string

/** 语音入口信号（书斋命令条本页直启语音，信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
}

// ─── 顶栏仪表条（v4.182 从 rail 迁入首页；v4.475 空间切换钮退役）────────────
// 1B 已拍板（2026-09-10，拆开）：壳层开关=界面导航，与办公引擎空间
// （办公侧栏 SpaceChip，写 session.space）两套是定局。v4.475 首页合一后
// 切空间由板块导航隐式驱动（navigateBoard 自动切换），条上只剩首页形态
// 快捷切换与日期。
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

/** 首页顶栏仪表条（v4.475 起无空间语义）：形态切换 + 分隔线 + 日期。 */
export const HomeStrip: React.FC = () => {
  return (
    <div className="ml-strip v3-rise v3-rise-1">
      <HomeLayoutToggle />
      <span className="ml-strip-rule" aria-hidden="true" />
      <span className="ml-strip-date" aria-hidden="true">{new Date().toLocaleDateString()}</span>
    </div>
  )
}

// ════════════════════════════════════════════════════════════════════
//  统一首页 · v9「案头」——masthead 身份区 + 纯命令台 + 账页×案牌 + 脉息面板
//  （v4.475 书斋/闲庭合一：本版式即唯一 classic 首页，板块全量出卡）
// ════════════════════════════════════════════════════════════════════
const DeskHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  onOpenTaskInbox: () => void
  inboxTick: number
}> = ({ data, onNavigate, onOpenTaskInbox, inboxTick }) => {
  const t = useT()
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  // 首页合一（v4.475）：不传 space = 全量板块（书斋+闲庭一屏尽收），menuOrder 序。
  // v4.476 模块卡极简化：全卡同权（无旗舰/设置特殊态），不再按旗舰拆组。
  const allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC)

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
        {/* 顶栏仪表条（形态切换 / 日期）：v4.475 空间切换钮随首页合一退役 */}
        <HomeStrip />

        {/* ═══ ① 指挥卡（全宽主角卡）：款识 + 台名 + 命令条。首页第一任务
            （发起工作）与身份、语音状态收在同一张卡里，不再分散成三条横带。 ═══ */}
        <section className="ml-card w-hero v3-rise v3-rise-1" aria-label={t('shell.launcher.heroAria')}>
          <header className="w-hero-head">
            {/* 印章：台名首字（v4.475 随合一去空间语义，不再取空间名首字） */}
            <span className="w-seal" aria-hidden="true">{(t('home.title') || '')[0] || ''}</span>
            <div className="w-hero-id">
              <h1 className="w-hero-title">{t('home.title')}</h1>
              <p className="w-hero-lede">{t('home.sub')}</p>
            </div>
            <div className="w-hero-side">
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

        {/* ═══ ③ 能力矩阵：节标 + 统一模块卡网格（v4.476 全卡同权极简化）═══ */}
        <section className="w-cap v3-rise v3-rise-3" aria-label={t('home.capTitle')}>
          <SectionLabel icon={<ThunderboltOutlined />} title={t('home.capTitle')} sub={t('home.capSub')} />
          <div className="w-modules">
            {allModules.map((m) => (
              <ModuleCard key={m.key} m={m} onOpen={() => onNavigate(m.key)} />
            ))}
            {allModules.length === 0 && (
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

          {/* 7.3-1 任务收件箱：待处理计数 + 打开收件箱（挂点 testid 契约不变）。
              v4.475 首页合一：计数跨空间全量（scope=''），面板内可跨空间处置。 */}
          <section className="ml-card w-stat" aria-label={t('shell.launcher.taskInbox')} data-testid="desk-task-inbox">
            <CardHead icon={<CheckSquareOutlined />} title={t('shell.launcher.taskInbox')} />
            <TaskInboxEntry space="" tick={inboxTick} onOpen={onOpenTaskInbox} />
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
 * 能力卡（v10 立；v4.476 极简化重画）：manifest 驱动的板块入口卡，全卡同权
 * ——无旗舰/设置特殊态、无徽记/水印/悬停箭头。卡=图标座 + 名称 + 一行描述
 * （完整描述进 title 提示，藏≠删）；hover 反馈=抬升 + 图标座点亮（Minimalism
 * /Swiss 对策：必要元素 + subtle hover，深度靠色阶与投影不靠线）。
 * aria 文案沿用 enterModule（可访问名不依赖视觉装饰）。
 */
const ModuleCard: React.FC<{
  m: LauncherModule
  onOpen: () => void
}> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  const t = useT()
  return (
    <button
      type="button"
      className="w-mod"
      aria-label={t('shell.launcher.enterModule', { name: m.name })}
      title={m.desc || m.name}
      onClick={onOpen}
    >
      <span className="w-mod-icon" aria-hidden="true">{Icon ? <Icon /> : null}</span>
      <span className="w-mod-body">
        <span className="w-mod-name">{m.name}</span>
        <span className="w-mod-desc">{m.desc}</span>
      </span>
    </button>
  )
}

/**
 * ModuleLauncher — 首页入口：数据一次拉取，按 homeLayout 分发任务优先 / 统一台
 * 两形态（v4.475 起不再有书斋/闲庭版式之分）。7.3-1：任务收件箱面板单例挂顶层
 * ——列表口径跨空间全量（scope=''），新建任务仍按当前壳层空间落标签（Go
 * ValidSpace 仅收 work|play，标签保证任务归属数据不劣化）。inboxTick 在面板
 * 收起时 +1，驱动挂点计数补一次重读（用户动作触发，非轮询）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate }) => {
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
  // 能力 chips）；classic → 统一台首页。回退开关即时切回（藏≠删）。
  const homeLayout = useAppStore((st) => st.homeLayout)
  // 新建任务的归属标签 = 当前壳层空间（rail 指示器同一数据源；仅作数据落库，
  // 不再影响首页任何渲染分支）
  const space = useAppStore((st) => st.space)

  return (
    <>
      {homeLayout === 'tasks' ? (
        <TasksFirstHome data={data} onNavigate={onNavigate} onResumeSession={resumeSessionFromInbox} />
      ) : (
        <DeskHome data={data} onNavigate={onNavigate} onOpenTaskInbox={openInbox} inboxTick={inboxTick} />
      )}
      <TaskInboxPanel open={inboxOpen} onClose={closeInbox} space={space} scope="" onNavigate={onNavigate} onResumeSession={resumeSessionFromInbox} />
    </>
  )
}

export default ModuleLauncher
