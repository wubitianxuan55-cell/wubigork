import React, { useState } from 'react'
import { Radio, Typography } from 'antd'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import {
  loadContextPrefs, saveContextPref,
  type CtxTrendGranularity, type CtxTrendMode, type CtxBrowserSort, type CtxFileSort,
} from '../../gaea/lib/contextPrefs'

/**
 * ContextPrefsPanel —「上下文页偏好」卡（办公分组，蒸馏规划 2.5d）。
 *
 * 上下文页的趋势粒度/模式、浏览器分类内排序、文件活动排序此前是卡内瞬时
 * toggle（重开即丢），本卡补设置中心入口；存储与上下文页内 toggle 共用同一
 * localStorage 键（gaea.context.prefs，lib/contextPrefs），两处改动互通。
 * 选项文案复用上下文页既有 i18n 键（contextview.gran*、mode*、sort*、filesSort* 系列），
 * 不另造一份口径。
 *
 * 生效时机：上下文页挂载时读初值 → 切换在页面重新打开后生效（脚注如实标注）。
 */

/** 单行选择：左标签+说明、右 Radio.Group（形状对齐 ChatPanel 语音行）。 */
const ChoiceRow = <T extends string>({ label, desc, value, options, onChange }: {
  label: string
  desc: string
  value: T
  options: { value: T; label: string }[]
  onChange: (next: T) => void
}) => (
  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
    <div style={{ minWidth: 0 }}>
      <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{label}</div>
      <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', lineHeight: 1.6 }}>{desc}</div>
    </div>
    <Radio.Group
      size="small"
      value={value}
      onChange={(e) => onChange(e.target.value as T)}
      aria-label={label}
    >
      {options.map((o) => (
        <Radio.Button key={o.value} value={o.value}>{o.label}</Radio.Button>
      ))}
    </Radio.Group>
  </div>
)

const ContextPrefsPanel: React.FC = () => {
  const t = useT()
  const [granularity, setGranularity] = useState<CtxTrendGranularity>(() => loadContextPrefs().trendGranularity)
  const [mode, setMode] = useState<CtxTrendMode>(() => loadContextPrefs().trendMode)
  const [browserSort, setBrowserSort] = useState<CtxBrowserSort>(() => loadContextPrefs().browserSort)
  const [fileSort, setFileSort] = useState<CtxFileSort>(() => loadContextPrefs().fileSort)

  return (
    <SettingsSection
      title={t('settings.context.title')}
      desc={t('settings.context.desc')}
      instant
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <ChoiceRow<CtxTrendGranularity>
          label={t('settings.context.trendGranularity')}
          desc={t('settings.context.trendGranularityDesc')}
          value={granularity}
          options={[
            { value: 'step', label: t('contextview.granStep') },
            { value: 'turn', label: t('contextview.granTurn') },
          ]}
          onChange={(v) => { setGranularity(v); saveContextPref('trendGranularity', v) }}
        />
        <ChoiceRow<CtxTrendMode>
          label={t('settings.context.trendMode')}
          desc={t('settings.context.trendModeDesc')}
          value={mode}
          options={[
            { value: 'total', label: t('contextview.modeTotal') },
            { value: 'delta', label: t('contextview.modeDelta') },
          ]}
          onChange={(v) => { setMode(v); saveContextPref('trendMode', v) }}
        />
        <ChoiceRow<CtxBrowserSort>
          label={t('settings.context.browserSort')}
          desc={t('settings.context.browserSortDesc')}
          value={browserSort}
          options={[
            { value: 'time', label: t('contextview.sortTime') },
            { value: 'size', label: t('contextview.sortSize') },
          ]}
          onChange={(v) => { setBrowserSort(v); saveContextPref('browserSort', v) }}
        />
        <ChoiceRow<CtxFileSort>
          label={t('settings.context.fileSort')}
          desc={t('settings.context.fileSortDesc')}
          value={fileSort}
          options={[
            { value: 'count', label: t('contextview.filesSortCalls') },
            { value: 'recent', label: t('contextview.filesSortLatest') },
            { value: 'path', label: t('contextview.filesSortPath') },
          ]}
          onChange={(v) => { setFileSort(v); saveContextPref('fileSort', v) }}
        />
      </div>
      <Typography.Text style={{ display: 'block', marginTop: 12, fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
        {t('settings.context.footnote')}
      </Typography.Text>
    </SettingsSection>
  )
}

export default ContextPrefsPanel
