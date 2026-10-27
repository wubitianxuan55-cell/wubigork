/**
 * ModuleLauncher — 统一首页（DeepSeek 式「输入即首页」，2026-10-27 用户拍板）。
 *
 * 沿革 docs/gaea-home-redesign-survey-2026-10.md §8.3/§8.4：方向一中轴三段上线次日，
 * 用户再拍板「简洁优先，模仿 DeepSeek 网页版，功能板块置于对话框下端」（§8.3），
 * 复检再拍「印章+时辰问候退役换静态身份位、命令条放大做大气」（§8.4）。形态：
 *   ① 开始 —— 台名大标（静态品牌位，无时间态）+ 题辞 + 大命令条（语音/工具/发送，
 *      72px 抬升面）+ 板块胶囊行；垂直居中唯一主角；
 *   ② 板块 —— 命令条正下方一排功能板块胶囊（deriveLauncherModules 全量 12 模块
 *      =方向一退役的 12 板块卡以胶囊形回归 DeepSeek 位，点击直达；rail/命令面板
 *      双入口不变）。
 * 退役（入口归位，藏≠删）：最近文档行（办公板块自有最近面）、待办一行与任务
 * 优先形态 TasksFirstHome + homeLayout 分发（收件箱面板保留单例，入口移命令面板
 * 空态「打开任务收件箱」——OPEN_TASK_INBOX 事件直达，首页 keepAlive 常驻自
 * boot 恒有听者）、建议词 chips（DeepSeek 形态输入框下只有功能胶囊）、顶栏
 * 仪表条（日期归系统时钟/壳层遥测轨）。
 * 「继续」段保留工作对象组（书/工程）+ 最近会话行；两者皆空时整段不渲染——
 * 冷启首页=纯 DeepSeek 落地页，工作有现场时才长出「最近」。
 *
 * 视觉沿用玄墨 v11 语言（衬线大标题/描边退场/金点睛/印泥朱印章），零硬编码
 * 色值全走令牌。数据层单源 useLauncherData（会话一次读）。
 */
import React, { useState, useCallback, useMemo, useEffect, useSyncExternalStore } from 'react'
import {
  ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, ClockCircleOutlined,
  BlockOutlined, BarChartOutlined, BookOutlined, ToolOutlined,
} from '@ant-design/icons'
import { requestSessionResume } from '../gaea/lib/pendingSessionResume'
import { loadRecentObjects, subscribeRecentObjects, type RecentObject } from '../gaea/lib/recentObjects'
import { guardNovelSwitch } from './novel/novelSwitchGuard'
import { openNovelProject } from './novel/openNovelProject'
import { Dropdown, Input } from 'antd'
import type { MenuProps } from 'antd'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { useAppStore } from '../stores/appStore'
import { useT } from '../gaea/lib/i18n'
import type { DictKey } from '../gaea/locales/en'
import { app } from '../gaea/lib/bridge'
import { TaskInboxPanel } from '../gaea/components/TaskInboxPanel'
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC } from '../boards/launcher'
import { SectionLabel, ChatBubble, SessionList, useLauncherData } from './launcher_parts'
import { emitFrontendEvent, FRONTEND_EVENTS, OPEN_TOOL_FLAG } from '../events'
import './module-launcher.css'


/**
 * 启动器可跳转的目标页（3.0 §5.2：放宽为 string，由 manifest.id 派生）。
 */
export type LauncherTarget = string

/** 语音入口信号（书斋命令条本页直启语音，信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
}

// §7.5 刀6 首页工具清单（方向一：从独立工具节移入命令条「工具」下拉）：
// 工具=动作不是房间——打开办公工作台里的临时 ToolSheet（不导航去板块）。
// 只收已落地的 ToolSheet kind（cad/schedule）；文档/表格是办公对话能力，
// 无工具面板 kind 支撑不冒充工具。
type HomeToolKind = 'cad' | 'schedule'
const HOME_TOOLS: { kind: HomeToolKind; icon: React.ReactNode; nameKey: DictKey; descKey: DictKey }[] = [
  { kind: 'cad', icon: <BlockOutlined />, nameKey: 'home.toolCad', descKey: 'home.toolCadDesc' },
  { kind: 'schedule', icon: <BarChartOutlined />, nameKey: 'home.toolSchedule', descKey: 'home.toolScheduleDesc' },
]

/** 时辰问候键已随 §8.4 退役（印章+问候→台名大标承接身份位；键闲置留典） */

const DeskHome: React.FC<{
  onNavigate: (t: LauncherTarget) => void
}> = ({ onNavigate }) => {
  const t = useT()
  const [typedText, setTypedText] = useState('')
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')
  const data = useLauncherData()
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

  // ── 板块胶囊行（§8.3）：deriveLauncherModules 全量 12 模块（含 settings 隐式
  // 入口，=方向一退役的 12 板块卡清单），胶囊形置于命令条正下方——DeepSeek
  // 落地页的功能入口位。清单随活动板块订阅实时更新（manifest 机制零改动）。──
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  const modules = useMemo(
    () => deriveLauncherModules(activeBoards, LAUNCHER_DESC),
    [activeBoards],
  )

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

  // 「继续」段整段渲染判据：对象组或会话至少其一有内容——冷启首页保持
  // 纯 DeepSeek 落地页，不渲染只有节标的空段。
  const hasRecent = objects.length > 0 || data.sessions.length > 0

  return (
    <div className="ml ml-work">
      <div className="w-zen">
        {/* ═══ ① 开始：台名大标 + 命令条 + 板块胶囊 —— 垂直居中唯一主角 ═══ */}
        <section className="w-greet v3-rise v3-rise-1" aria-label={t('shell.launcher.heroAria')}>
          {/* §8.4：印章与时辰问候随用户拍板退役——软件名字标 gaea 承接身份位
              （DeepSeek 落地页同构：品牌名居中，静态无时间态；与 rail/favicon
              同一品牌拼写） */}
          <h1 className="w-hero-title">{t('home.brand')}</h1>
          <p className="w-hero-lede">{t('home.sub')}</p>

          {/* 命令条：单一抬升面 + focus-within 金环（键盘可达，⌘K 提示） */}
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

          {/* 板块胶囊行（§8.3 DeepSeek 位）：命令条正下方一排功能入口，点击直达 */}
          <div className="w-boards" data-testid="home-boards-row" aria-label={t('home.boardsAria')}>
            {modules.map((m) => {
              const Icon = resolveBoardIcon(m.icon)
              return (
                <button
                  key={m.key}
                  type="button"
                  className="w-board-pill"
                  data-testid={`home-board-${m.key}`}
                  title={m.desc}
                  onClick={() => onNavigate(m.key)}
                >
                  {Icon && <Icon aria-hidden />}
                  <span className="w-board-pill-label">{m.name}</span>
                </button>
              )
            })}
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

        {/* ═══ ② 继续：最近工作对象（书/工程）+ 最近会话行（点行真打开）。
            两者皆空时整段不渲染（§8.3：冷启首页=纯落地页）。═══ */}
        {hasRecent && (
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
            {data.sessions.length > 0 && (
              <div className="w-recent-sess" aria-label={t('shell.launcher.sessions')}>
                <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  )
}

/**
 * ModuleLauncher — 首页入口（§8.3 起单一形态）。任务收件箱面板单例挂顶层
 * ——列表口径跨空间全量（scope=''），新建任务仍按当前壳层空间落标签（Go
 * ValidSpace 仅收 work|play，标签保证任务归属数据不劣化）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate }) => {
  const [inboxOpen, setInboxOpen] = useState(false)
  const closeInbox = useCallback(() => setInboxOpen(false), [])

  // 收件箱入口归位（§8.3）：首页待办行与任务优先形态退役后，面板唯一入口=
  // 命令面板空态「打开任务收件箱」。首页 keepAlive 常驻（MainLayout
  // visitedPages 自 boot 含 home），监听器恒活，无需 sessionStorage 补开标记。
  useEffect(() => {
    const open = () => setInboxOpen(true)
    window.addEventListener(FRONTEND_EVENTS.OPEN_TASK_INBOX, open)
    return () => window.removeEventListener(FRONTEND_EVENTS.OPEN_TASK_INBOX, open)
  }, [])

  // 会话级回源（7.3-1 收口）：记 pending+派发事件（keepAlive 已挂载态直达），
  // 再导航到 gaea 板块——冷启态由 App 挂载消费 pending 兜底。
  const resumeSessionFromInbox = useCallback((path: string) => {
    requestSessionResume(path)
    onNavigate('gaea')
  }, [onNavigate])
  // 新建任务的归属标签 = 当前壳层空间（rail 指示器同一数据源；仅作数据落库）
  const space = useAppStore((st) => st.space)

  return (
    <>
      <DeskHome onNavigate={onNavigate} />
      <TaskInboxPanel open={inboxOpen} onClose={closeInbox} space={space} scope="" onNavigate={onNavigate} onResumeSession={resumeSessionFromInbox} />
    </>
  )
}

export default ModuleLauncher
