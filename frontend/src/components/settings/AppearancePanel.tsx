import React, { useMemo, useState } from 'react'
import {
  CheckOutlined, DesktopOutlined, MoonOutlined, BgColorsOutlined,
  SunOutlined, ThunderboltOutlined, FontSizeOutlined, DashboardOutlined, CompressOutlined,
  SwapOutlined, AimOutlined, GlobalOutlined, LayoutOutlined, AppstoreOutlined, InboxOutlined } from '@ant-design/icons'
import { Button, InputNumber, Segmented, Select } from 'antd'
import { useAppStore, THEME_PRESETS, FONT_OPTIONS, type DisplayMode, type ThemePreset, type Density, type MotionPref, type HomeLayout } from '../../stores/appStore'
import { useI18n, useT, type LangPref } from '../../gaea/lib/i18n'
import type { DictKey } from '../../gaea/locales/en'
import SettingsSection from './SettingsSection'

// 主题选项：数据单一来源 THEME_PRESETS（appStore）；label/desc 经 i18n 组件内派生（原为模块级直用）。
// THEME_LABEL_KEYS 与 appStore 的 ThemePreset 联合类型锁定：新增主题时 tsc 会强制补键。
const THEME_LABEL_KEYS: Record<ThemePreset, { label: DictKey; desc: DictKey }> = {
  nightJade: { label: 'settings.appear.themeNightJadeLabel', desc: 'settings.appear.themeNightJadeDesc' },
  nightViolet: { label: 'settings.appear.themeNightVioletLabel', desc: 'settings.appear.themeNightVioletDesc' },
  nightRose: { label: 'settings.appear.themeNightRoseLabel', desc: 'settings.appear.themeNightRoseDesc' },
  nightAmber: { label: 'settings.appear.themeNightAmberLabel', desc: 'settings.appear.themeNightAmberDesc' },
  nightMoss: { label: 'settings.appear.themeNightMossLabel', desc: 'settings.appear.themeNightMossDesc' },
  nightSlate: { label: 'settings.appear.themeNightSlateLabel', desc: 'settings.appear.themeNightSlateDesc' },
}

interface ThemeOption { key: ThemePreset; label: string; desc: string; color: string }

/** 主题 Select 的 option data 形状：color/desc 随 options 透传，optionRender 经 option.data 取用 */
type ThemeSelectOption = {
  value: ThemePreset
  label: string
  color: string
  desc: string
}

/** 主题选项（label/desc 已本地化）；t 变更时重算（下拉选项与 hover 预览共用） */
function useThemeOptions(): ThemeOption[] {
  const t = useT()
  return useMemo(
    () => THEME_PRESETS.map((p) => ({ key: p.key, color: p.color, label: t(THEME_LABEL_KEYS[p.key].label), desc: t(THEME_LABEL_KEYS[p.key].desc) })),
    [t],
  )
}

/** ThemeOrb — 主题色发光球（沿用原 ThemeCard 氛围球样式）：下拉 labelRender 色点 / optionRender 图标共用 */
function ThemeOrb({ color, size = 20 }: { color: string; size?: number }) {
  return (
    <span style={{
      width: size, height: size, borderRadius: '50%', flexShrink: 0,
      background: `radial-gradient(circle at 35% 30%, ${color}, color-mix(in srgb, ${color} 45%, #000))`,
      boxShadow: `0 0 10px ${color}66, 0 0 20px ${color}33`,
    }} />
  )
}

/** AppearancePreview — 主题 + 模式实时微缩预览（hover 下拉选项时预览该主题，离开恢复当前） */
function AppearancePreview({ t, previewing }: { t: ThemeOption; previewing: boolean }) {
  const { darkMode } = useAppStore()
  const tr = useT()
  return (
    <div style={{
      borderRadius: 16, overflow: 'hidden',
      border: '1px solid var(--md-sys-color-outline-variant)',
      boxShadow: '0 8px 30px color-mix(in srgb, var(--gaea-glow) 12%, transparent)',
      transition: 'all var(--md-sys-transition-fast)',
    }}>
      {/* 预览画布：深空星云 / 晨光背景跟随主题色与模式 */}
      <div style={{
        padding: '18px 20px',
        background: darkMode
          ? `linear-gradient(135deg, #0a0f1e 0%, ${t.color}33 100%)`
          : `linear-gradient(135deg, #f8f6f1 0%, ${t.color}44 100%)`,
      }}>
        {/* 霓虹标题条（模拟设置中心玻璃区块） */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          <span style={{
            width: 3, height: 14, borderRadius: 2,
            background: 'var(--gaea-glow)', boxShadow: '0 0 8px var(--gaea-glow)',
          }} />
          <span style={{ fontSize: 13, fontWeight: 600, color: darkMode ? '#e2e8f0' : '#0f172a' }}> {/* hex-exempt 主题预览固定明暗样张 */}
            {t.label} · {darkMode ? tr('settings.appear.dark') : tr('settings.appear.light')}
          </span>
          <span style={{
            marginLeft: 'auto', display: 'inline-flex', alignItems: 'center', gap: 3,
            fontSize: 10, padding: '2px 8px', borderRadius: 999,
            color: 'var(--md-sys-color-success)',
            border: '1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)',
            background: 'color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)',
            fontWeight: 500,
          }}>{previewing ? tr('settings.appear.previewing') : (<><ThunderboltOutlined aria-hidden="true" />{tr('settings.instantBadge')}</>)}</span>
        </div>
        {/* 玻璃卡片模拟 */}
        <div style={{
          borderRadius: 12, padding: '12px 14px',
          background: darkMode ? 'color-mix(in srgb, var(--color-text) 7%, transparent)' : 'rgba(255,255,255,0.72)',
          border: darkMode ? '1px solid rgba(255,255,255,0.14)' : '1px solid rgba(0,0,0,0.10)', // hex-exempt 主题预览固定明暗样张（边框随档位）
          backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
            <span style={{
              width: 16, height: 16, borderRadius: '50%',
              background: `radial-gradient(circle at 35% 30%, ${t.color}, color-mix(in srgb, ${t.color} 45%, #000))`,
              boxShadow: `0 0 10px ${t.color}`,
            }} />
            <span style={{ fontSize: 12, fontWeight: 600, color: darkMode ? '#e2e8f0' : '#0f172a' }}>{tr('settings.appear.previewCardTitle')}</span> {/* hex-exempt 主题预览固定明暗样张 */}
          </div>
          <div style={{ fontSize: 11, color: darkMode ? '#94a3b8' : '#64748b', lineHeight: 1.7 }}> {/* hex-exempt 主题预览固定明暗样张 */}
            {tr('settings.appear.previewCardDesc', { color: t.color })}
          </div>
        </div>
      </div>
    </div>
  )
}

/** AppearancePanel — 主题色系：下拉选择 + 合并进卡的实时预览（原「外观实时预览」独立卡并入，8 卡 → 7 卡）
 *  主题下拉：optionRender 悬停即时预览，点击生效 */
const AppearancePanel: React.FC = () => {
  const t = useT()
  const { baseTheme, setTheme } = useAppStore()
  const [hovered, setHovered] = useState<ThemePreset | null>(null)
  const themeOptions = useThemeOptions()

  const previewT = themeOptions.find((x) => x.key === (hovered ?? baseTheme)) ?? themeOptions[0]

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><BgColorsOutlined /></span>}
      title={t('settings.appear.themeTitle')}
      desc={t('settings.appear.themeDesc')}
    >
      {/* 主题下拉：色点 + 名称，option 悬停即时预览（沿袭原 ThemeCard hover 特性） */}
      <Select<ThemePreset, ThemeSelectOption>
        value={baseTheme}
        onChange={(k) => setTheme(k)}
        style={{ width: 320 }}
        popupMatchSelectWidth={false}
        aria-label={t('settings.appear.themeTitle')}
        onOpenChange={(open) => { if (!open) setHovered(null) }}
        labelRender={({ label, value }) => {
          const opt = themeOptions.find((o) => o.key === value) ?? themeOptions[0]
          return (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
              <ThemeOrb color={opt.color} size={16} />
              <span>{label ?? opt.label}</span>
            </span>
          )
        }}
        optionRender={(option) => {
          const d = option.data
          return (
            <div
              onMouseEnter={() => setHovered(d.value)}
              onMouseLeave={() => setHovered(null)}
              style={{ display: 'flex', alignItems: 'center', gap: 10 }}
            >
              <ThemeOrb color={d.color} size={20} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{d.label}</div>
                <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{d.desc}</div>
              </div>
              {d.value === baseTheme && <CheckOutlined style={{ color: 'var(--gaea-glow)', fontSize: 13 }} />}
            </div>
          )
        }}
        options={themeOptions.map((o) => ({ value: o.key, label: o.label, color: o.color, desc: o.desc }))}
      />
      {/* hover 预览说明行 */}
      <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', margin: '8px 0 12px' }}>
        {t('settings.appear.livePreviewDesc')}
      </div>
      {/* 预览小标题 */}
      <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--md-sys-color-text)', marginBottom: 6 }}>
        {t('settings.appear.livePreviewTitle')}
      </div>
      <AppearancePreview t={previewT} previewing={!!hovered} />
    </SettingsSection>
  )
}

/** SegmentedRow — 行式选择条目：左侧当前项图标 + label/desc 摘要，右侧 Segmented 分段切换。
 *  紧凑替代原 ChoiceCards 卡片网格，与 FontPanel/AccentPanel 的行式风格统一（显示模式/密度/动效共用）。 */
function SegmentedRow<T extends string>({ value, onChange, options, ariaLabel }: {
  value: T
  onChange: (v: T) => void
  ariaLabel: string
  options: { key: T; label: string; desc?: string; icon?: React.ReactNode }[]
}) {
  // 左侧摘要跟随当前选中项（icon + label + desc 随切换联动）
  const current = options.find((o) => o.key === value)
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
      {current?.icon && (
        <span style={{
          width: 30, height: 30, borderRadius: 9, flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
          color: 'var(--md-sys-color-text-secondary)',
          background: 'var(--md-sys-color-surface-variant)',
          transition: 'all var(--md-sys-transition-fast)',
        }}>{current.icon}</span>
      )}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{current?.label}</div>
        {current?.desc && <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>{current.desc}</div>}
      </div>
      <Segmented
        value={value}
        onChange={(v) => onChange(v as T)}
        aria-label={ariaLabel}
        options={options.map((o) => ({
          value: o.key,
          label: (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              {o.icon && <span style={{ fontSize: 13 }}>{o.icon}</span>}
              {o.label}
            </span>
          ),
        }))}
      />
    </div>
  )
}

/** DarkModePanel — 显示模式行式条目：暗色 / 亮色 / 跟随系统 */
export const DarkModePanel: React.FC = () => {
  const t = useT()
  const { mode, systemDark, setMode } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><SwapOutlined /></span>} title={t('settings.appear.displayTitle')} desc={t('settings.appear.displayDesc')}>
      <SegmentedRow<DisplayMode>
        value={mode}
        onChange={setMode}
        ariaLabel={t('settings.appear.displayTitle')}
        options={[
          { key: 'dark',   label: t('settings.appear.modeDark'), desc: t('settings.appear.modeDarkDesc'), icon: <MoonOutlined /> },
          { key: 'light',  label: t('settings.appear.modeLight'), desc: t('settings.appear.modeLightDesc'), icon: <SunOutlined /> },
          { key: 'system', label: t('settings.appear.modeSystem'), desc: t('settings.appear.modeSystemDesc', { state: systemDark ? t('settings.appear.dark') : t('settings.appear.light') }), icon: <DesktopOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** FontPanel — 字体设置：界面字体 + 字号（FONT_OPTIONS 标签经 i18n 派生，数据仍在 appStore） */
const FONT_LABEL_KEYS: Record<string, DictKey> = {
  system: 'settings.appear.fontSystem',
  yahei: 'settings.appear.fontYahei',
  noto: 'settings.appear.fontNoto',
  songti: 'settings.appear.fontSongti',
  mono: 'settings.appear.fontMono',
}

export const FontPanel: React.FC = () => {
  const t = useT()
  const { fontFamily, fontSize, setFontFamily, setFontSize } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><FontSizeOutlined /></span>} title={t('settings.appear.fontTitle')} desc={t('settings.appear.fontDesc')}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{
            width: 30, height: 30, borderRadius: 9, flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
            color: 'var(--md-sys-color-text-secondary)',
            background: 'var(--md-sys-color-surface-variant)',
          }}><FontSizeOutlined /></span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{t('settings.appear.fontFamily')}</div>
            <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>{t('settings.appear.fontFamilyDesc')}</div>
          </div>
          <Select
            size="small"
            value={fontFamily}
            style={{ width: 180 }}
            onChange={setFontFamily}
            options={FONT_OPTIONS.map((o) => ({ value: o.key, label: FONT_LABEL_KEYS[o.key] ? t(FONT_LABEL_KEYS[o.key]) : o.label }))}
            popupMatchSelectWidth={false}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{
            width: 30, height: 30, borderRadius: 9, flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
            color: 'var(--md-sys-color-text-secondary)',
            background: 'var(--md-sys-color-surface-variant)',
          }}>A</span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{t('settings.appear.fontSize')}</div>
            <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>
              {t('settings.appear.fontSizeNow', { n: fontSize })}
              <span style={{ fontSize, color: 'var(--gaea-glow)' }}>{t('settings.appear.fontSizeSample')}</span>
            </div>
          </div>
          <InputNumber
            size="small"
            min={12} max={20} step={1}
            value={fontSize}
            onChange={(v) => { if (v) setFontSize(v) }}
            style={{ width: 80 }}
          />
        </div>
      </div>
    </SettingsSection>
  )
}

/** DensityPanel — 界面密度：标准 / 紧凑 */
export const DensityPanel: React.FC = () => {
  const t = useT()
  const { density, setDensity } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><DashboardOutlined /></span>} title={t('settings.appear.densityTitle')} desc={t('settings.appear.densityDesc')}>
      <SegmentedRow<Density>
        value={density}
        onChange={setDensity}
        ariaLabel={t('settings.appear.densityTitle')}
        options={[
          { key: 'standard', label: t('settings.appear.densityStandard'), desc: t('settings.appear.densityStandardDesc'), icon: <DashboardOutlined /> },
          { key: 'compact',  label: t('settings.appear.densityCompact'), desc: t('settings.appear.densityCompactDesc'), icon: <CompressOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** MotionPanel — 动效强度：完整 / 减弱（可访问性） */
export const MotionPanel: React.FC = () => {
  const t = useT()
  const { motion, setMotion } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><ThunderboltOutlined /></span>} title={t('settings.appear.motionTitle')} desc={t('settings.appear.motionDesc')}>
      <SegmentedRow<MotionPref>
        value={motion}
        onChange={setMotion}
        ariaLabel={t('settings.appear.motionTitle')}
        options={[
          { key: 'full',    label: t('settings.appear.motionFull'), desc: t('settings.appear.motionFullDesc'), icon: <ThunderboltOutlined /> },
          { key: 'reduced', label: t('settings.appear.motionReduced'), desc: t('settings.appear.motionReducedDesc'), icon: <MoonOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** HomeLayoutPanel — 首页形态（7.3-2 板块降级为任务视图）：经典 / 任务优先。
 *  回退开关的正式位（首页 SpaceSwitch 条的快捷钮为另一入口，同源 appStore）。 */
export const HomeLayoutPanel: React.FC = () => {
  const t = useT()
  const { homeLayout, setHomeLayout } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><LayoutOutlined /></span>} title={t('settings.homeLayoutTitle')} desc={t('settings.homeLayoutDesc')}>
      <SegmentedRow<HomeLayout>
        value={homeLayout}
        onChange={setHomeLayout}
        ariaLabel={t('settings.homeLayoutTitle')}
        options={[
          { key: 'classic', label: t('settings.homeLayoutClassic'), icon: <AppstoreOutlined /> },
          { key: 'tasks', label: t('settings.homeLayoutTasks'), icon: <InboxOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** AccentPanel — 强调色自定义：跟随主题 / 自定义取色 */
export const AccentPanel: React.FC = () => {
  const t = useT()
  const { accentColor, setAccentColor } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><AimOutlined /></span>} title={t('settings.appear.accentTitle')} desc={t('settings.appear.accentDesc')}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <span style={{
          width: 30, height: 30, borderRadius: 9, flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
          color: 'var(--md-sys-color-text-secondary)',
          background: 'var(--md-sys-color-surface-variant)',
        }}><BgColorsOutlined /></span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>
            {accentColor ? t('settings.appear.accentCustom', { color: accentColor }) : t('settings.appear.accentFollow')}
          </div>
          <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>
            {t('settings.appear.accentHint')}
          </div>
        </div>
        <label style={{
          position: 'relative', cursor: 'pointer',
          width: 38, height: 38, borderRadius: 10, flexShrink: 0,
          background: accentColor || 'var(--gaea-glow)',
          boxShadow: accentColor ? `0 0 14px ${accentColor}88` : '0 0 14px var(--gaea-glow)',
          border: '1px solid var(--md-sys-color-outline-variant)',
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <input
            type="color"
            value={accentColor || '#2dd4bf'} /* hex-exempt 默认强调色回退（与主题令牌一致） */
            onChange={(e) => setAccentColor(e.target.value)}
            style={{ position: 'absolute', inset: 0, opacity: 0, cursor: 'pointer' }}
          />
        </label>
        {accentColor && (
          <Button size="small" onClick={() => setAccentColor('')} style={{ flexShrink: 0 }}>{t('settings.appear.accentFollow')}</Button>
        )}
      </div>
    </SettingsSection>
  )
}

/** LanguagePanel — 界面语言：跟随系统 / 简体中文 / 繁體中文 / English
 *  i18n 三语字典与 setPref 早已就绪，此处是首个切换入口；偏好存 localStorage（gaea-lang），
 *  即时生效、整树重渲染。设置中心各板块面板已接入 i18n，此处保留覆盖范围说明。 */
export const LanguagePanel: React.FC = () => {
  const t = useT()
  const { pref, setPref, locale } = useI18n()
  // 语言自身名（autonym）不随界面语言翻译：始终以该语言的书写系统呈现
  const localeLabels: Record<Exclude<LangPref, ''>, string> = {
    zh: t('settings.appear.langZh'),
    'zh-TW': t('settings.appear.langZhTW'),
    en: t('settings.appear.langEn'),
  }
  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><GlobalOutlined /></span>}
      title={t('settings.appear.langTitle')}
      desc={t('settings.appear.langDesc', { label: localeLabels[locale] })}
      instant
    >
      <Select
        value={pref === '' ? 'auto' : pref}
        onChange={(v) => setPref(v === 'auto' ? '' : (v as Exclude<LangPref, ''>))}
        style={{ width: 260 }}
        aria-label={t('settings.appear.langTitle')}
        options={[
          { value: 'auto', label: t('settings.appear.langAuto', { label: localeLabels[locale] }) },
          { value: 'zh', label: localeLabels.zh },
          { value: 'zh-TW', label: localeLabels['zh-TW'] },
          { value: 'en', label: localeLabels.en },
        ]}
      />
    </SettingsSection>
  )
}

export default AppearancePanel
