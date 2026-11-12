import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, Radio, Typography, message } from 'antd'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import { app as gaeaApp } from '../../gaea/lib/bridge'
import { gaeaSettings } from '../../api/settings'
import type { SettingsView } from '../../gaea/lib/types'
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
      <ContextPromptGroup />
    </SettingsSection>
  )
}

/**
 * ContextPromptGroup —「系统提示词」组（刀14，dsh-context settings.group.prompt
 * 1:1）：部署级 promptPrefix/promptSuffix，拼接在所有会话系统提示词的最前/
 * 最后（含子代理——子代理 L1 从同一 compiler Fork 派生）。显式保存（引擎级
 * 写回+controller 重建即时生效，非瞬时偏好），保存载荷整体回传已加载视图，
 * 防他处设置保存互相清零。
 */
const ContextPromptGroup: React.FC = () => {
  const t = useT()
  const [prefix, setPrefix] = useState('')
  const [suffix, setSuffix] = useState('')
  const [loaded, setLoaded] = useState<SettingsView | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let alive = true
    gaeaSettings().then((v) => {
      if (!alive || !v) return
      setLoaded(v as unknown as SettingsView)
      setPrefix((v.promptPrefix as string | undefined) ?? '')
      setSuffix((v.promptSuffix as string | undefined) ?? '')
    }).catch(() => { /* 读取失败：输入框留空，保存时以报错呈现 */ })
    return () => { alive = false }
  }, [])

  const handleSave = useCallback(async () => {
    if (!loaded) {
      message.warning(t('settings.context.promptLoadFail'))
      return
    }
    setSaving(true)
    try {
      // 整体回传已加载视图 + 仅改两字段（OfficePanel 保存同样原样回传，
      // 两处互不清零）。
      await gaeaApp.SaveSettings({ ...loaded, promptPrefix: prefix, promptSuffix: suffix } as SettingsView)
      setLoaded({ ...loaded, promptPrefix: prefix, promptSuffix: suffix })
      message.success(t('settings.context.promptSaved'))
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('settings.saveFailed'))
    } finally {
      setSaving(false)
    }
  }, [loaded, prefix, suffix, t])

  const areaStyle = { fontSize: 12 } as const
  return (
    <div data-testid="ctx-prompt-group" style={{ marginTop: 16, paddingTop: 12, borderTop: '1px solid var(--md-sys-color-outline-variant, rgba(0,0,0,0.08))' }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)', marginBottom: 6 }}>
        {t('settings.context.promptGroup')}
      </div>
      <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', lineHeight: 1.6, marginBottom: 10 }}>
        {t('settings.context.promptHint')}
      </div>
      <div style={{ fontSize: 12, color: 'var(--md-sys-color-text)', marginBottom: 4 }}>{t('settings.context.promptPrefixLabel')}</div>
      <Input.TextArea
        aria-label={t('settings.context.promptPrefixLabel')}
        value={prefix}
        onChange={(e) => setPrefix(e.target.value)}
        placeholder={t('settings.context.promptPlaceholder')}
        rows={3}
        style={areaStyle}
      />
      <div style={{ fontSize: 12, color: 'var(--md-sys-color-text)', margin: '10px 0 4px' }}>{t('settings.context.promptSuffixLabel')}</div>
      <Input.TextArea
        aria-label={t('settings.context.promptSuffixLabel')}
        value={suffix}
        onChange={(e) => setSuffix(e.target.value)}
        placeholder={t('settings.context.promptPlaceholder')}
        rows={3}
        style={areaStyle}
      />
      <Button size="small" type="primary" loading={saving} onClick={handleSave} style={{ marginTop: 10 }} data-testid="ctx-prompt-save">
        {t('settings.context.promptSave')}
      </Button>
    </div>
  )
}

export default ContextPrefsPanel
