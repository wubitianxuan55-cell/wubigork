/**
 * ModuleLauncher — 统一首页（方向一「输入即首页」，2026-10-26 拍板）。
 *
 * 市场调研落档 docs/gaea-home-redesign-survey-2026-10.md：AI 产品首页共识
 * =「开始（输入）+ 继续（最近）+ 需要你处理的（待办）」一屏三件事。classic
 * 形态从六区（指挥卡/文档卡/侧列/工具节/板块节/状态排）收敛为中轴三段：
 *   ① 开始 —— 问候（时辰）+ 印章 + 大命令条（语音/工具/发送）+ 建议词 chips；
 *   ② 继续 —— 最近文档行 + 最近会话行（点行真打开）；
 *   ③ 待办 —— 待处理计数一行（其余状态归壳层遥测轨/运行监测板块）。
 * 退役（入口归位，藏≠删）：12 板块卡 → 常驻左栏 rail + 命令面板；工具节
 * （§7.5 刀6 tile）→ 命令条「工具」下拉（OPEN_TOOL 双通道机制原样）；状态
 * 卡排 → 待办一行；写作进度环 → 小说板块。tasks 形态（TasksFirstHome）不动。
 *
 * 视觉沿用玄墨 v11 语言（衬线大标题/描边退场/金点睛/印泥朱印章），零硬编码
 * 色值全走令牌。契约保持（测试与壳层依赖，勿改）：desk-recent-docs、
 * desk-task-inbox、task-inbox-open-btn、home-layout-*。数据层单源
 * useLauncherData（recentFiles 订阅 + 会话一次读）。
 */
import React, { useState, useCallback, useRef, useMemo, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, FileTextOutlined, ClockCircleOutlined,
  CheckSquareOutlined, BlockOutlined, BarChartOutlined, BookOutlined, ToolOutlined,
} from '@ant-design/icons'
import TasksFirstHome from './TasksFirstHome'
import { requestSessionResume } from '../gaea/lib/pendingSessionResume'
import { openPaneFileOrPreview } from '../gaea/lib/paneFileOpen'
import { loadRecentObjects, subscribeRecentObjects, type RecentObject } from '../gaea/lib/recentObjects'
import { guardNovelSwitch } from './novel/novelSwitchGuard'
import { openNovelProject } from './novel/openNovelProject'
import { Dropdown, Input } from 'antd'
import type { InputRef, MenuProps } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { useT } from '../gaea/lib/i18n'
import type { DictKey } from '../gaea/locales/en'
import { app } from '../gaea/lib/bridge'
import { TaskInboxPanel } from '../gaea/components/TaskInboxPanel'
import { SectionLabel, ChatBubble, TaskInboxEntry, SessionList, useLauncherData, type LauncherData } from './launcher_parts'
import { emitFrontendEvent, FRONTEND_EVENTS, OPEN_TOOL_FLAG } from '../events'
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
//  classic 形态 · 方向一「输入即首页」——中轴三段：开始 / 继续 / 待办
// ════════════════════════════════════════════════════════════════════

// §7.5 刀6 首页工具清单（方向一：从独立工具节移入命令条「工具」下拉）：
// 工具=动作不是房间——打开办公工作台里的临时 ToolSheet（不导航去板块）。
// 只收已落地的 ToolSheet kind（cad/schedule）；文档/表格是办公对话能力，
// 无工具面板 kind 支撑不冒充工具。
type HomeToolKind = 'cad' | 'schedule'
const HOME_TOOLS: { kind: HomeToolKind; icon: React.ReactNode; nameKey: DictKey; descKey: DictKey }[] = [
  { kind: 'cad', icon: <BlockOutlined />, nameKey: 'home.toolCad', descKey: 'home.toolCadDesc' },
  { kind: 'schedule', icon: <BarChartOutlined />, nameKey: 'home.toolSchedule', descKey: 'home.toolScheduleDesc' },
]

// 场景建议词（NN/g 规范：紧贴输入框、具体优于笼统、人工策展）。
// 点击=填入命令条（不直接发送），可改可发。
const HOME_SUGGESTIONS: DictKey[] = ['home.sug1', 'home.sug2', 'home.sug3', 'home.sug4']

/** 时辰问候键（5/12/18 点三段切换；夜深档 0~5 点） */
function greetKeyOf(hour: number): DictKey {
  if (hour < 5) return 'home.greetNight'
  if (hour < 12) return 'home.greetMorning'
  if (hour < 18) return 'home.greetAfternoon'
  return 'home.greetEvening'
}

const DeskHome: React.FC<{
  data: LauncherData
  onNavigate: (t: LauncherTarget) => void
  onOpenTaskInbox: () => void
  inboxTick: number
}> = ({ data, onNavigate, onOpenTaskInbox, inboxTick }) => {
  const t = useT()
  const inputRef = useRef<InputRef>(null)
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

  // §7.5 刀6：工具点击 = 双通道打开办公 ToolSheet。办公 App keepAlive
  // 常驻时 OPEN_TOOL 事件直达 setActiveTool；首次未挂载（lazy 懒加载）事件
  // 无人听 → sessionStorage 标记由办公 App 挂载效应补开（OPEN_STATS_FLAG 同款
  // 双通道；已挂载场景监听侧顺带清标记，无残留）。
  const openOfficeTool = useCallback((tool: HomeToolKind) => {
    try { sessionStorage.setItem(OPEN_TOOL_FLAG, tool) } catch { /* 隐私模式静默跳过补开 */ }
    emitFrontendEvent(FRONTEND_EVENTS.OPEN_TOOL, { tool })
    onNavigate('gaea')
  }, [onNavigate])

  const toolMenuItems: MenuProps['items'] = HOME_TOOLS.map((tool) => ({
    key: tool.kind,
    icon: tool.icon,
    label: (
      <span data-testid={`home-tool-${tool.kind}`} title={t(tool.descKey)}>
        {t(tool.nameKey)}
      </span>
    ),
    onClick: () => openOfficeTool(tool.kind),
  }))

  const applySuggestion = useCallback((text: string) => {
    setTypedText(text)
    inputRef.current?.focus()
  }, [])

  // ──「继续」对象组（二期 2026-10-26）：小说书 / 进度工程两类工作对象 ──
  // 订阅单源（localStorage + 写入广播）；无对象不渲染（与最近会话同款）。
  const objectsRaw = useSyncExternalStore(subscribeRecentObjects, loadRecentObjects, () => [])
  const objects = useMemo(() => objectsRaw.slice(0, 3), [objectsRaw])

  const objectKindLabel = useCallback((kind: RecentObject['kind']): string =>
    kind === 'novel' ? t('home.objKindNovel') : t('home.objKindSchedule'), [t])

  /**
   * 续做一行：
   * - schedule = 先切工程指针（store 动作，ScheduleTool/完整工作台指针驱动自动
   *   跟随）再走 OPEN_TOOL 双通道开进度工具面板；切指针失败仍开面板看当前工程
   *   （面板自身会诚实展示 syncError）。store 走动态 import——schedule 域不进
   *   主包，首页点击时才拉取。
   * - novel = 切书闸门（未保存正文先问）+ 完整打开链路（共享件 openNovelProject，
   *   与书架一份真相），落小说板块书架（书已打开、书架高亮）。
   */
  const resumeObject = useCallback((o: RecentObject) => {
    if (o.kind === 'schedule') {
      void (async () => {
        try {
          const { useScheduleStore } = await import('../schedule/store')
          await useScheduleStore.getState().openProject(o.key)
        } catch { /* 指针切换失败：仍打开面板（面板内诚实展示错误态） */ }
        openOfficeTool('schedule')
      })()
      return
    }
    const apply = async () => {
      try { await openNovelProject(o.key, o.title) } catch { /* 打开失败仍去书架，现场可见 */ }
      onNavigate('novel')
    }
    if (guardNovelSwitch(() => { void apply() }, o.title)) return
    void apply()
  }, [onNavigate, openOfficeTool])

  return (
    <div className="ml ml-work">
      <div className="w-zen">
        {/* 顶栏仪表条（形态切换 / 日期） */}
        <HomeStrip />

        {/* ═══ ① 开始：问候 + 命令条 + 建议词 —— 首页唯一主角，垂直居中 ═══ */}
        <section className="w-greet v3-rise v3-rise-1" aria-label={t('shell.launcher.heroAria')}>
          {/* 印章：台名首字（身份记号保留，玄墨语言不动） */}
          <span className="w-seal" aria-hidden="true">{(t('home.title') || '')[0] || ''}</span>
          <h1 className="w-hero-title">{t(greetKeyOf(new Date().getHours()))}</h1>
          <p className="w-hero-lede">{t('home.sub')}</p>

          {/* 命令条：单一抬升面 + focus-within 金环（键盘可达，⌘K 提示） */}
          <div className={`w-cmd ${voiceTone}`}>
            <span className="w-cmd-orb" aria-hidden="true">
              <span className="w-cmd-orb-core" />
              <span className="w-cmd-orb-ring" />
            </span>
            <Input
              ref={inputRef}
              value={typedText}
              onChange={(e) => setTypedText(e.target.value)}
              onPressEnter={() => sendTyped()}
              placeholder={t('home.placeholder')}
              aria-label={t('home.placeholder')}
              variant="borderless"
              className="w-cmd-input"
              disabled={voice.active}
            />
            <Dropdown trigger={['click']} menu={{ items: toolMenuItems }}>
              <button
                type="button"
                className="w-cmd-btn is-tools"
                data-testid="home-tools-btn"
                aria-haspopup="menu"
                aria-label={t('home.toolsTitle')}
                title={t('home.toolsSub')}
              >
                <ToolOutlined /> <span className="w-cmd-btn-label">{t('home.toolsTitle')}</span>
              </button>
            </Dropdown>
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

          {/* 建议词：填入命令条（不直接发送），可改可发 */}
          <div className="w-chips">
            {HOME_SUGGESTIONS.map((key, i) => (
              <button
                key={key}
                type="button"
                className="w-chip"
                data-testid={`home-sug-${i + 1}`}
                title={t(key)}
                onClick={() => applySuggestion(t(key))}
              >
                {t(key)}
              </button>
            ))}
          </div>

          {hasChat && (
            <div className="w-hero-chat" aria-live="polite">
              {userText && <ChatBubble role="user" text={userText} />}
              {aiReply && <ChatBubble role="assistant" text={aiReply} />}
            </div>
          )}

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

        {/* ═══ ② 继续：最近工作对象（书/工程）+ 最近文档行 + 最近会话行（点行真打开）═══ */}
        <section className="w-recent v3-rise v3-rise-2" data-testid="home-recent" aria-label={t('home.recentTitle')}>
          <SectionLabel icon={<ClockCircleOutlined />} title={t('home.recentTitle')} />
          {objects.length > 0 && (
            <ul className="w-docs-list w-objects" data-testid="home-recent-objects" aria-label={t('home.recentTitle')}>
              {objects.map((o) => (
                <li key={`${o.kind}:${o.key}`} className="w-docs-row">
                  <button
                    type="button"
                    className="w-docs-btn"
                    data-testid={`home-obj-${o.kind}`}
                    aria-label={`${o.title} — ${objectKindLabel(o.kind)}`}
                    title={objectKindLabel(o.kind)}
                    onClick={() => resumeObject(o)}
                  >
                    <span className="w-docs-mark" aria-hidden="true">
                      {o.kind === 'novel' ? <BookOutlined /> : <BarChartOutlined />}
                    </span>
                    <span className="w-docs-body">
                      <span className="w-docs-name">{o.title}</span>
                      <span className="w-docs-path w-obj-kind">{objectKindLabel(o.kind)}</span>
                    </span>
                    <ArrowRightOutlined className="w-docs-arrow" aria-hidden="true" />
                  </button>
                </li>
              ))}
            </ul>
          )}
          <div className="w-docs" data-testid="desk-recent-docs" aria-label={t('home.recentDocs')}>
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
          </div>
          {data.sessions.length > 0 && (
            <div className="w-recent-sess" aria-label={t('shell.launcher.sessions')}>
              <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
            </div>
          )}
        </section>

        {/* ═══ ③ 需要你处理的：待办一行（其余状态归遥测轨/运行监测）═══
            7.3-1 任务收件箱：待处理计数 + 打开收件箱（挂点 testid 契约不变）。
            v4.475 首页合一：计数跨空间全量（scope=''），面板内可跨空间处置。 */}
        <footer className="w-todo v3-rise v3-rise-3" data-testid="desk-task-inbox" aria-label={t('shell.launcher.taskInbox')}>
          <span className="w-todo-icon" aria-hidden="true"><CheckSquareOutlined /></span>
          <TaskInboxEntry space="" tick={inboxTick} onOpen={onOpenTaskInbox} row />
        </footer>
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
 * ModuleLauncher — 首页入口：数据一次拉取，按 homeLayout 分发任务优先 / 输入
 * 即首页两形态（v4.475 起不再有书斋/闲庭版式之分）。7.3-1：任务收件箱面板单例
 * 挂顶层——列表口径跨空间全量（scope=''），新建任务仍按当前壳层空间落标签（Go
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
  // 能力 chips）；classic → 输入即首页。回退开关即时切回（藏≠删）。
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
