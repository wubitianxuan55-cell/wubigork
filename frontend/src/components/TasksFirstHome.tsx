// TasksFirstHome.tsx — 任务优先首页（7.3-2 板块降级为任务视图·层跃升刀，规格
// 进度计划/gaea-tasks-first-home-20260917.md）：收件箱+最近上下文成为首屏，
// 板块入口降为「能力的视图」（manifest 数据源不动、onNavigate 直达——藏≠删）。
// 复用件：SpaceSwitch/SessionList（ModuleLauncher 导出）+ TaskInboxBoard
// （TaskInboxPanel 内联板）+ deriveLauncherModules（manifest 驱动机制零改动）。
// 纯同步口径脚注：任务是清单不是调度器，应用关闭一切停止。
import React, { useMemo } from 'react'
import { useSyncExternalStore } from 'react'
import { Button, Tooltip } from 'antd'
import { InboxOutlined, RollbackOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { useT } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'
import { getActiveBoards, subscribeBoards, resolveBoardIcon } from '../boards/manifests'
import { deriveLauncherModules, LAUNCHER_DESC, type LauncherModule } from '../boards/launcher'
import { SHELL_SPACES, type ShellSpace } from '../boards/space'
import { SpaceSwitch, SessionList, type LauncherData } from './ModuleLauncher'
import { TaskInboxBoard } from '../gaea/components/TaskInboxPanel'

const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }

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

export default function TasksFirstHome({ data, onNavigate, space, onSwitchSpace, activeModel }: {
  data: LauncherData
  onNavigate: (t: string) => void
  space: ShellSpace
  onSwitchSpace: (s: ShellSpace) => void
  activeModel?: string
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
    <div className="ml" data-testid="tasks-first-home">
      <div className="w-dock">
        <div className="w-main">
          <SpaceSwitch space={space} onSwitchSpace={onSwitchSpace} activeModel={activeModel} />

          {/* 身份区（轻量）：印章 + 标题/lede + 切回经典（回退快捷位之一） */}
          <header className="w-masthead v3-rise v3-rise-1">
            <span className="w-seal" aria-hidden="true">{(spaceEntry ? spaceEntry.label : '')[0] || ''}</span>
            <div className="w-mast-body">
              <h1 className="w-mast-title">{t('home.tasksFirstTitle')}</h1>
              <p className="w-mast-lede">{t('home.tasksFirstLede')}</p>
            </div>
            <span style={{ flex: 1 }} />
            <Tooltip title={t('home.tasksFirstBackHint')}>
              <Button size="small" icon={<RollbackOutlined />} data-testid="tasks-first-back"
                onClick={() => setHomeLayout('classic')}>
                {t('home.tasksFirstBack')}
              </Button>
            </Tooltip>
          </header>

          {/* 首屏主体：收件箱大区（四档 tab+新建+状态机动作+板块跳转全量复用） */}
          <section className="v3-rise v3-rise-2" aria-label={t('tasks.inbox.title')} data-testid="tasks-first-inbox">
            <div className="ml-sec-head" style={{ marginBottom: 8 }}>
              <span className="ml-sec-icon" aria-hidden="true"><InboxOutlined /></span>
              <span className="ml-sec-title">{t('tasks.inbox.title')}</span>
            </div>
            <TaskInboxBoard space={space} active onNavigate={onNavigate} />
          </section>

          {/* 最近上下文：最近会话（点开进聊天板块续上现场） */}
          <section className="v3-rise v3-rise-3" style={{ marginTop: 18 }} aria-label={t('shell.launcher.sessions')}>
            <div className="ml-sec-head">
              <span className="ml-sec-icon" aria-hidden="true"><ThunderboltOutlined /></span>
              <span className="ml-sec-title">{t('shell.launcher.sessions')}</span>
            </div>
            <SessionList sessions={data.sessions} onOpen={() => onNavigate('chat')} />
          </section>

          {/* 能力视图：全部板块紧凑 chips（藏≠删——manifest 全量，点击直达） */}
          <section className="v3-rise v3-rise-3" style={{ marginTop: 18 }} aria-label={t('home.tasksFirstCaps')} data-testid="tasks-first-caps">
            <div className="ml-sec-head" style={{ marginBottom: 8 }}>
              <span className="ml-sec-icon" aria-hidden="true"><ThunderboltOutlined /></span>
              <span className="ml-sec-title">{t('home.tasksFirstCaps')}</span>
              <span style={{ ...softTextStyle, marginLeft: 8 }}>{t('home.capSub')}</span>
            </div>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
              {allModules.map((m) => (
                <CapabilityChip key={m.key} m={m} onOpen={() => onNavigate(m.key)} />
              ))}
            </div>
          </section>

          <p style={{ ...softTextStyle, marginTop: 20 }}>{t('home.tasksFirstFoot')}</p>
        </div>
      </div>
    </div>
  )
}
