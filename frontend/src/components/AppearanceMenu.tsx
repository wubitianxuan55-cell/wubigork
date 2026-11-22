import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  CheckOutlined,
  DesktopOutlined,
  DownOutlined,
  MoonOutlined,
  SettingOutlined,
  SunOutlined,
} from '@ant-design/icons'
import { Popover } from 'antd'
import {
  THEME_PRESETS,
  useAppStore,
  type DisplayMode,
  type ThemePreset,
} from '../stores/appStore'
import { useT } from '../gaea/lib/i18n'
import { THEME_COPY_KEYS, THEME_GLYPHS, themePoem } from '../theme/locale'

function paletteDotStyle(meta: (typeof THEME_PRESETS)[number]): React.CSSProperties {
  return {
    background: `linear-gradient(135deg, ${meta.morandi} 0%, ${meta.lightColor} 100%)`,
  }
}

function paletteSwatchStyle(meta: (typeof THEME_PRESETS)[number]): React.CSSProperties {
  return {
    '--gaea-swatch-a': meta.morandi,
    '--gaea-swatch-b': meta.lightColor,
    '--gaea-swatch-bd': meta.color,
  } as React.CSSProperties
}

function ModeChoices({ value, onChange }: {
  value: DisplayMode
  onChange: (value: DisplayMode) => void
}) {
  const t = useT()
  const options: { value: DisplayMode; icon: React.ReactNode; label: string }[] = [
    { value: 'light', icon: <SunOutlined />, label: t('settings.appear.light') },
    { value: 'dark', icon: <MoonOutlined />, label: t('settings.appear.dark') },
    { value: 'system', icon: <DesktopOutlined />, label: t('settings.appear.modeSystem') },
  ]
  return (
    <div className="gaea-theme-menu__mode" role="radiogroup" aria-label={t('settings.appear.displayTitle')}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          role="radio"
          aria-label={option.label}
          title={option.label}
          aria-checked={value === option.value}
          data-active={value === option.value}
          onClick={() => onChange(option.value)}
        >
          {option.icon}
        </button>
      ))}
    </div>
  )
}

export const AppearanceMenu: React.FC<{
  onOpenSettings: () => void
  onClose?: () => void
  open?: boolean
}> = ({ onOpenSettings, onClose, open = false }) => {
  const t = useT()
  const baseTheme = useAppStore((state) => state.baseTheme)
  const mode = useAppStore((state) => state.mode)
  const setTheme = useAppStore((state) => state.setTheme)
  const setMode = useAppStore((state) => state.setMode)
  const [captionKey, setCaptionKey] = useState<ThemePreset>(baseTheme)
  const menuRef = useRef<HTMLDivElement>(null)
  const paletteRefs = useRef<(HTMLButtonElement | null)[]>([])

  const themes = useMemo(
    () => THEME_PRESETS.map((preset) => ({
      ...preset,
      label: t(THEME_COPY_KEYS[preset.key].label),
      desc: t(THEME_COPY_KEYS[preset.key].desc),
    })),
    [t],
  )
  const current = themes.find((theme) => theme.key === baseTheme) ?? themes[0]
  const caption = themes.find((theme) => theme.key === captionKey) ?? current

  useEffect(() => {
    if (!open) setCaptionKey(baseTheme)
  }, [baseTheme, open])

  useEffect(() => {
    if (!open || !menuRef.current) return
    if (typeof menuRef.current.scrollTo === 'function') {
      menuRef.current.scrollTo({ top: 0 })
    } else {
      menuRef.current.scrollTop = 0
    }
    const frame = requestAnimationFrame(() => {
      const currentIndex = themes.findIndex((theme) => theme.key === baseTheme)
      paletteRefs.current[Math.max(0, currentIndex)]?.focus({ preventScroll: true })
    })
    return () => cancelAnimationFrame(frame)
  }, [baseTheme, open, themes])

  const focusTheme = (index: number) => {
    const next = (index + themes.length) % themes.length
    paletteRefs.current[next]?.focus()
  }

  return (
    <div ref={menuRef} id="gaea-appearance-menu" className="gaea-theme-menu" role="dialog" aria-label={t('settings.appear.quickMenuTitle')}>
      <div className="gaea-theme-menu__head">
        <span className="gaea-theme-menu__title">{t('settings.appear.quickMenuTitle')}</span>
        <span className="gaea-theme-menu__current">
          <span className="gaea-theme-dot" style={paletteDotStyle(current)} aria-hidden="true" />
          <span>{current.label}</span>
        </span>
        {onClose && (
          <button
            type="button"
            className="gaea-theme-menu__close"
            aria-label={t('settings.appear.closeMenu')}
            title={t('settings.appear.closeMenu')}
            onClick={onClose}
          >
            ×
          </button>
        )}
      </div>

      <div className="gaea-theme-menu__section-title">{t('settings.appear.menuTheme')}</div>
      <div className="gaea-theme-menu__palette" role="listbox" aria-label={t('settings.appear.menuTheme')}>
        {themes.map((theme, index) => {
          const active = theme.key === baseTheme
          return (
            <button
              key={theme.key}
              ref={(node) => { paletteRefs.current[index] = node }}
              type="button"
              role="option"
              aria-label={`${theme.label} · ${themePoem(theme.desc)}`}
              aria-selected={active}
              data-active={active}
              className="gaea-theme-menu__option"
              title={theme.desc}
              style={{ '--gaea-row-index': index } as React.CSSProperties}
              onClick={() => {
                setTheme(theme.key)
                onClose?.()
              }}
              onMouseEnter={() => setCaptionKey(theme.key)}
              onFocus={() => setCaptionKey(theme.key)}
              onKeyDown={(event) => {
                if (event.key === 'ArrowDown') {
                  event.preventDefault()
                  focusTheme(index + 6)
                } else if (event.key === 'ArrowUp') {
                  event.preventDefault()
                  focusTheme(index - 6)
                } else if (event.key === 'ArrowRight') {
                  event.preventDefault()
                  focusTheme(index + 1)
                } else if (event.key === 'ArrowLeft') {
                  event.preventDefault()
                  focusTheme(index - 1)
                } else if (event.key === 'Home') {
                  event.preventDefault()
                  focusTheme(0)
                } else if (event.key === 'End') {
                  event.preventDefault()
                  focusTheme(themes.length - 1)
                }
              }}
            >
              <span className="gaea-theme-menu__swatch" style={paletteSwatchStyle(theme)} aria-hidden="true">
                <span className="gaea-theme-menu__glyph">{THEME_GLYPHS[theme.key]}</span>
              </span>
              <CheckOutlined className="gaea-theme-menu__check" aria-hidden="true" />
            </button>
          )
        })}
      </div>
      <div className="gaea-theme-menu__caption" role="status" aria-live="polite">
        <strong>{caption.label}</strong>
        <span>{themePoem(caption.desc)}</span>
      </div>

      <div className="gaea-theme-menu__section-title">{t('settings.appear.menuMode')}</div>
      <ModeChoices value={mode} onChange={setMode} />

      <p className="gaea-theme-menu__hint" role="note">{t('settings.appear.menuDetailHint')}</p>

      <div className="gaea-theme-menu__foot">
        <span className="gaea-theme-menu__brand">GAEA</span>
        <button type="button" className="gaea-theme-menu__more" onClick={onOpenSettings}>
          <SettingOutlined aria-hidden="true" />
          <span>{t('settings.appear.openSettings')}</span>
        </button>
      </div>
    </div>
  )
}

/** Top-bar trigger for the latest Bloom swatch-grid appearance menu. */
export const ThemeMenuTrigger: React.FC = () => {
  const t = useT()
  const baseTheme = useAppStore((state) => state.baseTheme)
  const openSettings = useAppStore((state) => state.openSettings)
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const current = THEME_PRESETS.find((theme) => theme.key === baseTheme) ?? THEME_PRESETS[0]
  const currentLabel = t(THEME_COPY_KEYS[current.key].label)
  const closeMenu = () => {
    setOpen(false)
    const focusTrigger = () => triggerRef.current?.focus({ preventScroll: true })
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(focusTrigger)
    else focusTrigger()
  }

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      trigger="click"
      placement="bottomRight"
      arrow={false}
      overlayClassName="gaea-theme-popover"
      content={(
        <AppearanceMenu
          key={open ? 'open' : 'closed'}
          open={open}
          onClose={closeMenu}
          onOpenSettings={() => {
            closeMenu()
            openSettings('general', 'appearance')
          }}
        />
      )}
    >
      <button
        ref={triggerRef}
        type="button"
        className="gaea-theme-trigger"
        aria-label={t('settings.appear.openMenuAria', { label: currentLabel })}
        aria-haspopup="dialog"
        aria-controls={open ? 'gaea-appearance-menu' : undefined}
        aria-expanded={open}
      >
        <span className="gaea-theme-dot" style={paletteDotStyle(current)} aria-hidden="true" />
        <span className="gaea-theme-trigger__name">{currentLabel}</span>
        <DownOutlined className="gaea-theme-trigger__chevron" aria-hidden="true" />
      </button>
    </Popover>
  )
}

export default ThemeMenuTrigger
