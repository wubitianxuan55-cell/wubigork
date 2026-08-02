import React, { useState } from 'react'
import { CheckOutlined, DesktopOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons'
import { Typography } from 'antd'
import { useAppStore, type DisplayMode, type ThemePreset } from '../../stores/appStore'
import SettingsSection from './SettingsSection'

const themeOptions: { key: ThemePreset; label: string; desc: string; color: string }[] = [
  { key: 'nightJade',   label: '暗夜青', desc: '深海翡翠 · 冷静专注', color: '#2dd4bf' },
  { key: 'nightViolet', label: '暗夜紫', desc: '深靛星云 · 灵感涌动', color: '#a78bfa' },
  { key: 'nightRose',   label: '暗夜玫', desc: '深褐暖调 · 温情创作', color: '#fb7185' },
  { key: 'nightAmber',  label: '暗夜金', desc: '深色暖灯 · 沉浸舒适', color: '#f59e0b' },
  { key: 'nightMoss',   label: '暗夜苔', desc: '深色林间 · 自然舒适', color: '#84cc16' },
  { key: 'nightSlate',  label: '暗夜墨', desc: '中性深灰 · 极简克制', color: '#94a3b8' },
]

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
      onMouseEnter={() => onHover(true)}
      onMouseLeave={() => onHover(false)}
      style={{
        borderRadius: 14, overflow: 'hidden', cursor: 'pointer', userSelect: 'none',
        background: 'var(--md-sys-color-surface-container)',
        border: active ? '1.5px solid var(--gaea-glow)' : '1px solid var(--md-sys-color-outline-variant)',
        boxShadow: active ? '0 0 18px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
        transition: 'all var(--md-sys-transition-fast)',
        position: 'relative',
      }}
      onMouseEnterCapture={(e) => {
        if (!active) e.currentTarget.style.borderColor = 'color-mix(in srgb, var(--gaea-glow) 45%, transparent)'
      }}
      onMouseLeaveCapture={(e) => {
        if (!active) e.currentTarget.style.borderColor = 'var(--md-sys-color-outline-variant)'
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
            marginLeft: 'auto', fontSize: 10, padding: '2px 8px', borderRadius: 999,
            color: '#34d399', border: '1px solid #34d39944', background: '#34d39914', fontWeight: 500,
          }}>{previewing ? '👆 预览中' : '⚡ 即时生效'}</span>
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

/** AppearancePanel — 外观设置：主题色系（hover 预览）+ 显示模式（含跟随系统） */
const AppearancePanel: React.FC = () => {
  const { baseTheme, setTheme } = useAppStore()
  const [hovered, setHovered] = useState<ThemePreset | null>(null)

  const previewT = themeOptions.find((x) => x.key === (hovered ?? baseTheme)) ?? themeOptions[0]

  return (
    <>
      <SettingsSection icon={<span style={{ fontSize: 15 }}>👁️</span>} title="外观实时预览" desc="当前主题与显示模式的组合效果；鼠标悬停下方主题卡可即时预览，点击才生效。" noMargin>
        <AppearancePreview t={previewT} previewing={!!hovered} />
      </SettingsSection>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>🎨</span>}
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
  const modes: { key: DisplayMode; label: string; desc: string; icon: React.ReactNode }[] = [
    { key: 'dark',   label: '暗色模式', desc: '深空星云沉浸体验', icon: <MoonOutlined /> },
    { key: 'light',  label: '亮色模式', desc: '柔和晨光风格', icon: <SunOutlined /> },
    { key: 'system', label: '跟随系统', desc: `自动跟随系统明暗（当前${systemDark ? '暗色' : '亮色'}）`, icon: <DesktopOutlined /> },
  ]
  return (
    <SettingsSection icon={<span style={{ fontSize: 15 }}>🌗</span>} title="显示模式" desc="暗色为深空星云沉浸体验，亮色为柔和晨光风格，跟随系统将随操作系统明暗自动切换。">
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 10 }}>
        {modes.map((m) => {
          const active = mode === m.key
          return (
            <div
              key={m.key}
              role="button"
              tabIndex={0}
              onClick={() => setMode(m.key)}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setMode(m.key) } }}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '13px 14px', borderRadius: 'var(--md-sys-radius-md)', cursor: 'pointer', userSelect: 'none',
                background: active ? 'color-mix(in srgb, var(--gaea-glow) 10%, var(--md-sys-color-surface-container))' : 'var(--md-sys-color-surface-container)',
                border: active ? '1.5px solid var(--gaea-glow)' : '1px solid var(--md-sys-color-outline-variant)',
                boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 25%, transparent)' : 'none',
                transition: 'all var(--md-sys-transition-fast)',
              }}
            >
              <span style={{
                width: 32, height: 32, borderRadius: 10, flexShrink: 0,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 16,
                color: active ? 'var(--gaea-glow)' : 'var(--md-sys-color-text-secondary)',
                background: m.key === 'dark'
                  ? 'linear-gradient(135deg, #1e293b, #0f172a)'
                  : m.key === 'light'
                    ? 'linear-gradient(135deg, #fef3c7, #fde68a)'
                    : 'linear-gradient(135deg, #334155, #1e293b)',
                boxShadow: active ? '0 0 10px color-mix(in srgb, var(--gaea-glow) 35%, transparent)' : 'none',
              }}>{m.icon}</span>
              <div style={{ minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{m.label}</div>
                <div style={{ fontSize: 10.5, color: 'var(--md-sys-color-text-secondary)' }}>{m.desc}</div>
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
    </SettingsSection>
  )
}

export default AppearancePanel
