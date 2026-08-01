import React from 'react'
import { Space, Switch, Tooltip, Typography } from 'antd'
import { SunOutlined, MoonOutlined } from '@ant-design/icons'
import { useAppStore, type ThemePreset } from '../../stores/appStore'
import SettingsSection from './SettingsSection'

const themeOptions: { key: ThemePreset; label: string; desc: string; color: string }[] = [
  { key: 'nightJade',   label: '暗夜青', desc: '深海翡翠 · 冷静专注', color: '#2dd4bf' },
  { key: 'nightViolet', label: '暗夜紫', desc: '深靛星云 · 灵感涌动', color: '#a78bfa' },
  { key: 'nightRose',   label: '暗夜玫', desc: '深褐暖调 · 温情创作', color: '#fb7185' },
  { key: 'nightAmber',  label: '暗夜金', desc: '深色暖灯 · 沉浸舒适', color: '#f59e0b' },
  { key: 'nightMoss',   label: '暗夜苔', desc: '深色林间 · 自然舒适', color: '#84cc16' },
  { key: 'nightSlate',  label: '暗夜墨', desc: '中性深灰 · 极简克制', color: '#94a3b8' },
]

/** AppearancePanel — 外观设置：主题色系 + 暗/亮模式 */
const AppearancePanel: React.FC = () => {
  const { baseTheme, setTheme } = useAppStore()

  return (
    <SettingsSection title="主题色系" desc="选择全局氛围色 —— 深空星云背景、霓虹光效与玻璃质感将随主题联动。">
      <div style={{
        display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 10,
      }}>
        {themeOptions.map((t) => {
          const active = t.key === baseTheme
          return (
            <div
              key={t.key}
              role="button"
              tabIndex={0}
              onClick={() => setTheme(t.key)}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setTheme(t.key) } }}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '12px 14px', borderRadius: 'var(--md-sys-radius-md)',
                cursor: 'pointer', userSelect: 'none',
                background: active ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'var(--md-sys-color-surface-container)',
                border: active ? '1px solid var(--gaea-glow)' : '1px solid var(--md-sys-color-outline-variant)',
                boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 25%, transparent)' : 'none',
                transition: 'all var(--md-sys-transition-fast)',
              }}
            >
              <span style={{
                width: 26, height: 26, borderRadius: '50%', flexShrink: 0,
                background: `radial-gradient(circle at 35% 30%, ${t.color}, color-mix(in srgb, ${t.color} 55%, #000))`,
                boxShadow: `0 0 10px ${t.color}, 0 0 22px color-mix(in srgb, ${t.color} 40%, transparent)`,
              }} />
              <div style={{ minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--md-sys-color-text)' }}>{t.label}</div>
                <div style={{
                  fontSize: 10, color: 'var(--md-sys-color-text-secondary)',
                  whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
                }}>{t.desc}</div>
              </div>
            </div>
          )
        })}
      </div>
    </SettingsSection>
  )
}

/** DarkModePanel — 明暗模式 */
export const DarkModePanel: React.FC = () => {
  const { darkMode, toggleDarkMode } = useAppStore()
  return (
    <SettingsSection title="显示模式" desc="暗色为深空星云沉浸体验，亮色为柔和晨光风格。">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Space size={8}>
          {darkMode ? <MoonOutlined style={{ color: 'var(--gaea-glow)' }} /> : <SunOutlined style={{ color: 'var(--gaea-glow)' }} />}
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {darkMode ? '暗色模式' : '亮色模式'}
          </Typography.Text>
          <Tooltip title={darkMode ? '切换亮色' : '切换暗色'}>
            <Switch checked={darkMode} onChange={toggleDarkMode} />
          </Tooltip>
        </Space>
      </div>
    </SettingsSection>
  )
}

export default AppearancePanel
