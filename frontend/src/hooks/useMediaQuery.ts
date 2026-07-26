import { useState, useEffect } from 'react'

/** 响应式断点查询 hook — 移动优先 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  useEffect(() => {
    const mq = window.matchMedia(query)
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches)

    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [query])

  return matches
}

/** 预设断点 */
export const BREAKPOINTS = {
  compact:  '(max-width: 599px)',
  medium:   '(min-width: 600px) and (max-width: 899px)',
  mobile:   '(max-width: 899px)',   // compact + medium
  expanded: '(min-width: 900px)',   // 桌面端
} as const

export function useIsCompact(): boolean {
  return useMediaQuery(BREAKPOINTS.compact)
}
