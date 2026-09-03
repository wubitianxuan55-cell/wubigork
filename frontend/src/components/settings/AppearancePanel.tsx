import React, { useMemo, useState } from 'react'
import {
  CheckOutlined, DesktopOutlined, MoonOutlined, BgColorsOutlined,
  SunOutlined, ThunderboltOutlined, FontSizeOutlined, DashboardOutlined, CompressOutlined,
  EyeOutlined, SwapOutlined, AimOutlined, GlobalOutlined,
} from '@ant-design/icons'
import { Button, InputNumber, Select } from 'antd'
import { useAppStore, THEME_PRESETS, FONT_OPTIONS, type DisplayMode, type ThemePreset, type Density, type MotionPref } from '../../stores/appStore'
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

/** 主题选项（label/desc 已本地化）；t 变更时重算（hover 预览与主题卡共用） */
function useThemeOptions(): ThemeOption[] {
  const t = useT()
  return useMemo(
    () => THEME_PRESETS.map((p) => ({ key: p.key, color: p.color, label: t(THEME_LABEL_KEYS[p.key].label), desc: t(THEME_LABEL_KEYS[p.key].desc) })),
    [t],
  )
}

/** 通用「选择卡片」：多选一，选中发光对勾（外观设置各维度复用） */
function ChoiceCards<T extends string>({ options, value, onChange }: {
  options: { key: T; label: string; desc?: string; icon?: React.ReactNode }[]
  value: T
  onChange: (k: T) => void
}) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 10 }}>
      {options.map((m) => {
        const active = value === m.key
        return (
          <div
            key={m.key}
            role="button"
            tabIndex={0}
            onClick={() => onChange(m.key)}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onChange(m.key) } }}
            style={{
              display: 'flex', alignItems: 'center', gap: 10,
              padding: '13px 14px', borderRadius: 'var(--md-sys-radius-md)', cursor: 'pointer', userSelect: 'none',
              background: active ? 'color-mix(in srgb, var(--gaea-glow) 10%, var(--md-sys-color-surface-container))' : 'var(--md-sys-color-surface-container)',
              border: active ? '1.5px solid var(--gaea-glow)' : '1px solid var(--md-sys-color-outline-variant)',
              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 25%, transparent)' : 'none',
              transition: 'all var(--md-sys-transition-fast)',
            }}
          >
            {m.icon && (
              <span style={{
                width: 30, height: 30, borderRadius: 9, flexShrink: 0,
                display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
                color: active ? 'var(--gaea-glow)' : 'var(--md-sys-color-text-secondary)',
                background: active ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'var(--md-sys-color-surface-variant)',
              }}>{m.icon}</span>
            )}
            <div style={{ minWidth: 0 }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{m.label}</div>
              {m.desc && <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>{m.desc}</div>}
            </div>
            {active && (
              <span style={{
                marginLeft: 'auto', display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                width: 18, height: 18, borderRadius: '50%', background: 'var(--gaea-glow)',
                color: '#08130f', fontSize: 10, boxShadow: '0 0 8px var(--gaea-glow)', // hex-exempt 主题预览固定明暗样张
              }}><CheckOutlined /></span>
            )}
          </div>
        )
      })}
    </div>
  )
}

/** 主题预览卡：上部氛围色渐变条 + 下部名称/说明，选中态发光边框 + 对勾角标 */
function ThemeCard({ t, active, onClick, onHover }: {
  t: ThemeOption
  active: boolean
  onClick: () => void
  onHover: (hovering: boolean) => void
}) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
      onMouseEnter={(e: React.MouseEvent<HTMLDivElement>) => {
        onHover(true)
        if (!active) e.currentTarget.style.borderColor = 'color-mix(in srgb, var(--gaea-glow) 45%, transparent)'
      }}
      onMouseLeave={(e: React.MouseEvent<HTMLDivElement>) => {
        onHover(false)
        if (!active) e.currentTarget.style.borderColor = 'var(--md-sys-color-outline-variant)'
      }}
      style={{
        borderRadius: 14, overflow: 'hidden', cursor: 'pointer', userSelect: 'none',
        background: 'var(--md-sys-color-surface-container)',
        border: active ? '1.5px solid var(--gaea-glow)' : '1px solid var(--md-sys-color-outline-variant)',
        boxShadow: active ? '0 0 18px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
        transition: 'all var(--md-sys-transition-fast)',
        position: 'relative',
      }}

    >
      {/* 氛围预览条：暗底星云 + 主题色霓虹光晕 */}
      <div style={{
        height: 62,
        background: `linear-gradient(135deg, #0a0f1e 0%, ${t.color}44 55%, ${t.color}22 100%)`,
        position: 'relative',
      }}>
        <span style={{
          position: 'absolute', left: 16, top: 16, width: 22, height: 22, borderRadius: '50%',
          background: `radial-gradient(circle at 35% 30%, ${t.color}, color-mix(in srgb, ${t.color} 45%, #000))`,
          boxShadow: `0 0 12px ${t.color}, 0 0 26px color-mix(in srgb, ${t.color} 45%, transparent)`,
        }} />
        <span style={{
          position: 'absolute', right: 20, bottom: 12, width: 8, height: 8, borderRadius: '50%',
          background: `${t.color}88`, boxShadow: `0 0 8px ${t.color}`,
        }} />
      </div>
      {/* 名称区 */}
      <div style={{ padding: '10px 12px', display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: active ? 'var(--gaea-glow)' : 'var(--md-sys-color-text)' }}>
          {t.label}
        </span>
        <span style={{
          fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)',
          whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', flex: 1,
        }}>{t.desc}</span>
        {active && (
          <span style={{
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            width: 18, height: 18, borderRadius: '50%', flexShrink: 0,
            background: 'var(--gaea-glow)', color: '#08130f', fontSize: 10, // hex-exempt 主题预览固定明暗样张
            boxShadow: '0 0 8px var(--gaea-glow)',
          }}><CheckOutlined /></span>
        )}
      </div>
    </div>
  )
}

/** AppearancePreview — 主题 + 模式实时微缩预览（hover 主题卡时预览该主题，离开恢复当前） */
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
          background: darkMode ? 'rgba(255,255,255,0.07)' : 'rgba(255,255,255,0.72)',
          border: '1px solid rgba(255,255,255,0.14)',
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

/** AppearancePanel — 外观设置：主题色系（hover 预览）+ 实时预览 */
const AppearancePanel: React.FC = () => {
  const t = useT()
  const { baseTheme, setTheme } = useAppStore()
  const [hovered, setHovered] = useState<ThemePreset | null>(null)
  const themeOptions = useThemeOptions()

  const previewT = themeOptions.find((x) => x.key === (hovered ?? baseTheme)) ?? themeOptions[0]

  return (
    <>
      <SettingsSection icon={<span style={{ fontSize: 15 }}><EyeOutlined /></span>} title={t('settings.appear.livePreviewTitle')} desc={t('settings.appear.livePreviewDesc')} noMargin>
        <AppearancePreview t={previewT} previewing={!!hovered} />
      </SettingsSection>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><BgColorsOutlined /></span>}
        title={t('settings.appear.themeTitle')}
        desc={t('settings.appear.themeDesc')}
      >
        <div style={{
          display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(170px, 1fr))', gap: 10,
        }}>
          {themeOptions.map((t) => (
            <ThemeCard
              key={t.key} t={t} active={t.key === baseTheme}
              onClick={() => setTheme(t.key)}
              onHover={(h) => setHovered(h ? t.key : null)}
            />
          ))}
        </div>
      </SettingsSection>
    </>
  )
}

/** DarkModePanel — 显示模式三卡：暗色 / 亮色 / 跟随系统 */
export const DarkModePanel: React.FC = () => {
  const t = useT()
  const { mode, systemDark, setMode } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><SwapOutlined /></span>} title={t('settings.appear.displayTitle')} desc={t('settings.appear.displayDesc')}>
      <ChoiceCards<DisplayMode>
        value={mode}
        onChange={setMode}
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
      <ChoiceCards<Density>
        value={density}
        onChange={setDensity}
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
      <ChoiceCards<MotionPref>
        value={motion}
        onChange={setMotion}
        options={[
          { key: 'full',    label: t('settings.appear.motionFull'), desc: t('settings.appear.motionFullDesc'), icon: <ThunderboltOutlined /> },
          { key: 'reduced', label: t('settings.appear.motionReduced'), desc: t('settings.appear.motionReducedDesc'), icon: <MoonOutlined /> },
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
