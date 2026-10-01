// TasksFirstHome.tsx — 任务优先首页（7.3-2 板块降级为任务视图·层跃升刀，规格
// 进度计划/gaea-tasks-first-home-20260917.md）：收件箱+最近上下文成为首屏，
// 板块入口降为「能力的视图」（manifest 数据源不动、onNavigate 直达——藏≠删）。
// 复用件：SpaceSwitch/SessionList（ModuleLauncher 导出）+ TaskInboxBoard
// （TaskInboxPanel 内联板）+ deriveLauncherModules（manifest 驱动机制零改动）。
// 纯同步口径脚注：任务是清单不是调度器，应用关闭一切停止。
import { softTextStyle } from '../utils/uiStyles'
import React, { useMemo } from 'react'
import { useSyncExternalStore } from 'react'
import { Button, Tooltip } from 'antd'
import { InboxOutlined, RollbackOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { useT } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import { SHELL_SPACES, type ShellSpace } from '../boards/space'
import { CardHead, SpaceSwitch, SessionList, type LauncherData } from './ModuleLauncher'
import { TaskInboxBoard } from '../gaea/components/TaskInboxPanel'


/** 能力 chip：板块入口的紧凑形态（icon+名称，点击直达板块页）。 */
const CapabilityChip: React.FC<{ m: LauncherModule; onOpen: () => void }> = ({ m, onOpen }) => {
  const Icon = resolveBoardIcon(m.icon)
  return (
    <button
      type="button"
      data-testid={`tasks-first-cap-${m.key}`}
      className="ml-space-btn"
      style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}
      onClick={onOpen}
      title={m.desc || m.name}
    >
      {Icon && <Icon aria-hidden />}
      <span>{m.name}</span>
    </button>
  )
}

export default function TasksFirstHome({ data, onNavigate, space, onSwitchSpace, activeModel, onResumeSession }: {
  data: LauncherData
  onNavigate: (t: string) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
  /** 会话级回源（7.3-1 收口）：透传收件箱 Board。 */
  onResumeSession?: (path: string) => void
}) {
  const t = useT()
  const setHomeLayout = useAppStore((st) => st.setHomeLayout)
  const activeBoards = useSyncExternalStore(subscribeBoards, getActiveBoards)
  const allModules = useMemo(
    () => deriveLauncherModules(activeBoards, LAUNCHER_DESC, space),
    [activeBoards, space],
  )
  const spaceEntry = SHELL_SPACES.find((s) => s.id === space)

  return (
    <div className="ml ml-tasks" data-testid="tasks-first-home">
      <div className="w-board">
          <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

          {/* 指挥卡（v10 卡片化）：印章 + 标题/lede + 切回经典（回退快捷位之一） */}
          <section className="ml-card w-hero v3-rise v3-rise-1" aria-label={t('home.tasksFirstTitle')}>
            <header className="w-hero-head">
              <span className="w-seal" aria-hidden="true">{(spaceEntry ? spaceEntry.label : '')[0] || ''}</span>
              <div className="w-hero-id">
                <h1 className="w-hero-title">{t('home.tasksFirstTitle')}</h1>
                <p className="w-hero-lede">{t('home.tasksFirstLede')}</p>
              </div>
              <div className="w-hero-side">
                <Tooltip title={t('home.tasksFirstBackHint')}>
                  <Button size="small" icon={<RollbackOutlined />} data-testid="tasks-first-back"
                    onClick={() => setHomeLayout('classic')}>
                    {t('home.tasksFirstBack')}
                  </Button>
                </Tooltip>
              </div>
            </header>
          </section>

          {/* 首屏主体：收件箱大区（四档 tab+新建+状态机动作+板块跳转全量复用） */}
          <section className="ml-card t-inbox v3-rise v3-rise-2" aria-label={t('tasks.inbox.title')} data-testid="tasks-first-inbox">
            <CardHead icon={<InboxOutlined />} title={t('tasks.inbox.title')} />
            <TaskInboxBoard space={space} active onNavigate={onNavigate} onResumeSession={onResumeSession} />
          </section>

          {/* 最近上下文：最近会话（点开进聊天板块续上现场） */}
          <section className="ml-card t-sessions v3-rise v3-rise-3" aria-label={t('shell.launcher.sessions')}>
            <CardHead icon={<ThunderboltOutlined />} title={t('shell.launcher.sessions')} />
            <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
          </section>

          {/* 能力视图：全部板块紧凑 chips（藏≠删——manifest 全量，点击直达） */}
          <section className="ml-card t-caps v3-rise v3-rise-3" aria-label={t('home.tasksFirstCaps')} data-testid="tasks-first-caps">
            <CardHead icon={<ThunderboltOutlined />} title={t('home.tasksFirstCaps')} sub={t('home.capSub')} />
            <div className="t-chips">
              {allModules.map((m) => (
                <CapabilityChip key={m.key} m={m} onOpen={() => onNavigate(m.key)} />
              ))}
            </div>
          </section>

          <p className="t-foot" style={softTextStyle}>{t('home.tasksFirstFoot')}</p>
      </div>
    </div>
  )
}
