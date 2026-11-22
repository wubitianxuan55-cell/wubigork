import { useEffect } from 'react'
import {
  FEATURED_THEME_KEYS,
  useAppStore,
  type ThemePreset,
} from '../stores/appStore'

/** Pick the next featured colour while guaranteeing it differs from the current one. */
export function pickNextFeaturedTheme(
  current: ThemePreset,
  random: () => number = Math.random,
): ThemePreset {
  const choices = FEATURED_THEME_KEYS.filter((key) => key !== current)
  if (choices.length === 0) return current
  const index = Math.min(choices.length - 1, Math.max(0, Math.floor(random() * choices.length)))
  return choices[index]
}

/**
 * Rotate through the 12 featured colours while the document is visible.
 * A manual palette change restarts the countdown through `baseTheme` in the deps.
 */
export function useThemeAutoRotation(): void {
  const minutes = useAppStore((state) => state.themeRotationMinutes)
  const baseTheme = useAppStore((state) => state.baseTheme)

  useEffect(() => {
    if (!minutes) return
    let timer: number | undefined

    const schedule = () => {
      if (timer !== undefined) window.clearTimeout(timer)
      timer = undefined
      if (document.hidden) return
      timer = window.setTimeout(() => {
        const state = useAppStore.getState()
        state.setTheme(pickNextFeaturedTheme(state.baseTheme))
      }, minutes * 60_000)
    }

    schedule()
    document.addEventListener('visibilitychange', schedule)
    return () => {
      if (timer !== undefined) window.clearTimeout(timer)
      document.removeEventListener('visibilitychange', schedule)
    }
  }, [minutes, baseTheme])
}
