import React, { useState } from 'react'
import { Switch, Typography } from 'antd'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import { loadSubagentAutoOpen, saveSubagentAutoOpen } from '../../gaea/lib/subagentPrefs'
import { loadTasksAutoOpenSubagent, saveTasksAutoOpenSubagent } from '../../gaea/lib/tasksPrefs'
import { loadBrowserAutoOpen, saveBrowserAutoOpen } from '../../gaea/lib/browserPrefs'
import { loadDeliverableAutoOpen, saveDeliverableAutoOpen } from '../../gaea/lib/deliverablePrefs'

/**
 * WorkbenchPrefsPanel —「办公工作台偏好」卡（办公分组，收 v4.65 欠账）。
 *
 * gaea 工作台有一批「自动展开 / 自动切换」偏好此前只能靠 localStorage 键控
 * （各面板头部的胶囊开关不可发现），本卡在设置中心补齐入口：
 *  - gaea.subagentAutoOpen      新子代理自动展开（默认开）
 *  - gaea.tasks.autoOpenSubagent 新任务自动切任务视图（默认开）
 *  - gaea.browserAutoOpen       浏览器自动弹出（默认开）
 *  - gaea.deliverableAutoOpen   产物自动弹出（默认关）
 *
 * 零功能变更：只读写既有键、默认值与加载语义（损坏值回落）全部复用各 prefs
 * 模块。切换即时落盘；App 触发端（任务视图切换 / 浏览器 / 产物自动弹出）均为
 * 事件时直读 → 即时生效。唯 gaea.subagentAutoOpen 的分工面板胶囊为挂载时读，
 * 说明文案如实标注「面板重开后同步」。
 */

/** 单行开关：左标签+说明、右 Switch（形状对齐 ChatPanel 语音开关行）。 */
const PrefRow: React.FC<{
  label: string
  desc: string
  checked: boolean
  onToggle: (next: boolean) => void
}> = ({ label, desc, checked, onToggle }) => (
  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
    <div style={{ minWidth: 0 }}>
      <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{label}</div>
      <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', lineHeight: 1.6 }}>{desc}</div>
    </div>
    <Switch size="small" checked={checked} onChange={onToggle} aria-label={label} />
  </div>
)

const WorkbenchPrefsPanel: React.FC = () => {
  const t = useT()
  const [subagentAutoOpen, setSubagentAutoOpen] = useState(loadSubagentAutoOpen)
  const [tasksAutoOpen, setTasksAutoOpen] = useState(loadTasksAutoOpenSubagent)
  const [browserAutoOpen, setBrowserAutoOpen] = useState(loadBrowserAutoOpen)
  const [deliverableAutoOpen, setDeliverableAutoOpen] = useState(loadDeliverableAutoOpen)

  const rows: Array<{
    key: string
    label: string
    desc: string
    checked: boolean
    onToggle: (next: boolean) => void
  }> = [
    {
      key: 'subagentAutoOpen',
      label: t('settings.workbench.subagentAutoOpen'),
      desc: t('settings.workbench.subagentAutoOpenDesc'),
      checked: subagentAutoOpen,
      onToggle: (next) => { setSubagentAutoOpen(next); saveSubagentAutoOpen(next) },
    },
    {
      key: 'tasksAutoOpen',
      label: t('settings.workbench.tasksAutoOpen'),
      desc: t('settings.workbench.tasksAutoOpenDesc'),
      checked: tasksAutoOpen,
      onToggle: (next) => { setTasksAutoOpen(next); saveTasksAutoOpenSubagent(next) },
    },
    {
      key: 'browserAutoOpen',
      label: t('settings.workbench.browserAutoOpen'),
      desc: t('settings.workbench.browserAutoOpenDesc'),
      checked: browserAutoOpen,
      onToggle: (next) => { setBrowserAutoOpen(next); saveBrowserAutoOpen(next) },
    },
    {
      key: 'deliverableAutoOpen',
      label: t('settings.workbench.deliverableAutoOpen'),
      desc: t('settings.workbench.deliverableAutoOpenDesc'),
      checked: deliverableAutoOpen,
      onToggle: (next) => { setDeliverableAutoOpen(next); saveDeliverableAutoOpen(next) },
    },
  ]

  return (
    <SettingsSection
      title={t('settings.workbench.title')}
      desc={t('settings.workbench.desc')}
      instant
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        {rows.map((row) => (
          <PrefRow
            key={row.key}
            label={row.label}
            desc={row.desc}
            checked={row.checked}
            onToggle={row.onToggle}
          />
        ))}
      </div>
      <Typography.Text style={{ display: 'block', marginTop: 12, fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
        {t('settings.workbench.footnote')}
      </Typography.Text>
    </SettingsSection>
  )
}

export default WorkbenchPrefsPanel
