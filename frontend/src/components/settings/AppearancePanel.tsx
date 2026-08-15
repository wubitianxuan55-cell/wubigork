import React, { useState } from 'react'
import {
  CheckOutlined, DesktopOutlined, MoonOutlined, BgColorsOutlined,
  SunOutlined, ThunderboltOutlined, FontSizeOutlined, DashboardOutlined, CompressOutlined,
  EyeOutlined, SwapOutlined, AimOutlined,
} from '@ant-design/icons'
import { Button, InputNumber, Select, Typography } from 'antd'
import { useAppStore, THEME_PRESETS, FONT_OPTIONS, type DisplayMode, type ThemePreset, type Density, type MotionPref } from '../../stores/appStore'
import SettingsSection from './SettingsSection'

// 主题选项：单一数据源 THEME_PRESETS（appStore，3.0 Wave 2 消除与 MainLayout/appStore 色板表三处重复）
const themeOptions: { key: ThemePreset; label: string; desc: string; color: string }[] = THEME_PRESETS

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
                color: '#08130f', fontSize: 10, boxShadow: '0 0 8px var(--gaea-glow)',
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
  t: typeof themeOptions[number]
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
            background: 'var(--gaea-glow)', color: '#08130f', fontSize: 10,
            boxShadow: '0 0 8px var(--gaea-glow)',
          }}><CheckOutlined /></span>
        )}
      </div>
    </div>
  )
}

/** AppearancePreview — 主题 + 模式实时微缩预览（hover 主题卡时预览该主题，离开恢复当前） */
function AppearancePreview({ t, previewing }: { t: typeof themeOptions[number]; previewing: boolean }) {
  const { darkMode } = useAppStore()
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
          <span style={{ fontSize: 13, fontWeight: 600, color: darkMode ? '#e2e8f0' : '#0f172a' }}>
            {t.label} · {darkMode ? '暗色' : '亮色'}
          </span>
          <span style={{
            marginLeft: 'auto', display: 'inline-flex', alignItems: 'center', gap: 3,
            fontSize: 10, padding: '2px 8px', borderRadius: 999,
            color: 'var(--md-sys-color-success)',
            border: '1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)',
            background: 'color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)',
            fontWeight: 500,
          }}>{previewing ? '预览中' : (<><ThunderboltOutlined aria-hidden="true" />即时生效</>)}</span>
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
            <span style={{ fontSize: 12, fontWeight: 600, color: darkMode ? '#e2e8f0' : '#0f172a' }}>深空星云界面</span>
          </div>
          <div style={{ fontSize: 11, color: darkMode ? '#94a3b8' : '#64748b', lineHeight: 1.7 }}>
            玻璃质感容器 · 霓虹光效标题条 · 主题氛围色 {t.color}
          </div>
        </div>
      </div>
    </div>
  )
}

/** AppearancePanel — 外观设置：主题色系（hover 预览）+ 实时预览 */
const AppearancePanel: React.FC = () => {
  const { baseTheme, setTheme } = useAppStore()
  const [hovered, setHovered] = useState<ThemePreset | null>(null)

  const previewT = themeOptions.find((x) => x.key === (hovered ?? baseTheme)) ?? themeOptions[0]

  return (
    <>
      <SettingsSection icon={<span style={{ fontSize: 15 }}><EyeOutlined /></span>} title="外观实时预览" desc="当前主题与显示模式的组合效果；鼠标悬停下方主题卡可即时预览，点击才生效。" noMargin>
        <AppearancePreview t={previewT} previewing={!!hovered} />
      </SettingsSection>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><BgColorsOutlined /></span>}
        title="主题色系"
        desc="选择全局氛围色 —— 深空星云背景、霓虹光效与玻璃质感将随主题联动。"
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
  const { mode, systemDark, setMode } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><SwapOutlined /></span>} title="显示模式" desc="暗色为深空星云沉浸体验，亮色为柔和晨光风格，跟随系统将随操作系统明暗自动切换。">
      <ChoiceCards<DisplayMode>
        value={mode}
        onChange={setMode}
        options={[
          { key: 'dark',   label: '暗色模式', desc: '深空星云沉浸体验', icon: <MoonOutlined /> },
          { key: 'light',  label: '亮色模式', desc: '柔和晨光风格', icon: <SunOutlined /> },
          { key: 'system', label: '跟随系统', desc: `自动跟随系统明暗（当前${systemDark ? '暗色' : '亮色'}）`, icon: <DesktopOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** FontPanel — 字体设置：界面字体 + 字号 */
export const FontPanel: React.FC = () => {
  const { fontFamily, fontSize, setFontFamily, setFontSize } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><FontSizeOutlined /></span>} title="字体设置" desc="界面字体与全局字号，即时生效。">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{
            width: 30, height: 30, borderRadius: 9, flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
            color: 'var(--md-sys-color-text-secondary)',
            background: 'var(--md-sys-color-surface-variant)',
          }}><FontSizeOutlined /></span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>界面字体</div>
            <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>界面与正文使用的字体族</div>
          </div>
          <Select
            size="small"
            value={fontFamily}
            style={{ width: 180 }}
            onChange={setFontFamily}
            options={FONT_OPTIONS.map((o) => ({ value: o.key, label: o.label }))}
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
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>全局字号</div>
            <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>
              当前 {fontSize}px · 预览：
              <span style={{ fontSize, color: 'var(--gaea-glow)' }}> 字小乾坤大</span>
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
  const { density, setDensity } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><DashboardOutlined /></span>} title="界面密度" desc="控件与区块的紧凑程度。">
      <ChoiceCards<Density>
        value={density}
        onChange={setDensity}
        options={[
          { key: 'standard', label: '标准', desc: '宽松留白，舒适阅读', icon: <DashboardOutlined /> },
          { key: 'compact',  label: '紧凑', desc: '信息密集，一屏更多', icon: <CompressOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** MotionPanel — 动效强度：完整 / 减弱（可访问性） */
export const MotionPanel: React.FC = () => {
  const { motion, setMotion } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><ThunderboltOutlined /></span>} title="动效强度" desc="减弱动态可减少界面动画与过渡，降低视觉负担（对齐系统「减弱动态」）。">
      <ChoiceCards<MotionPref>
        value={motion}
        onChange={setMotion}
        options={[
          { key: 'full',    label: '完整动效', desc: '玻璃光效与过渡动画', icon: <ThunderboltOutlined /> },
          { key: 'reduced', label: '减弱动态', desc: '关闭动画，更简洁专注', icon: <MoonOutlined /> },
        ]}
      />
    </SettingsSection>
  )
}

/** AccentPanel — 强调色自定义：跟随主题 / 自定义取色 */
export const AccentPanel: React.FC = () => {
  const { accentColor, setAccentColor } = useAppStore()
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}><AimOutlined /></span>} title="强调色" desc="自定义霓虹光效与主色调；留空则跟随所选主题色系。">
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <span style={{
          width: 30, height: 30, borderRadius: 9, flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
          color: 'var(--md-sys-color-text-secondary)',
          background: 'var(--md-sys-color-surface-variant)',
        }}><BgColorsOutlined /></span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>
            {accentColor ? `自定义 · ${accentColor}` : '跟随主题'}
          </div>
          <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>
            影响霓虹光效、主按钮与选中态
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
            value={accentColor || '#2dd4bf'}
            onChange={(e) => setAccentColor(e.target.value)}
            style={{ position: 'absolute', inset: 0, opacity: 0, cursor: 'pointer' }}
          />
        </label>
        {accentColor && (
          <Button size="small" onClick={() => setAccentColor('')} style={{ flexShrink: 0 }}>跟随主题</Button>
        )}
      </div>
    </SettingsSection>
  )
}

export default AppearancePanel
