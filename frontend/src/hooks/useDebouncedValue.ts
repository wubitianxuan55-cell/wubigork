// useDebouncedValue.ts — 通用防抖镜像值（v4.365 性能轮）。
// 高频输入（受控 textarea 每键 setState）下游的重计算/重渲染（Markdown 全文
// 解析、全文字数统计等）按 delay 防抖跟随；最终值与源一致，仅中间态降频。
import { useEffect, useState } from 'react'

export function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay)
    return () => window.clearTimeout(timer)
  }, [value, delay])

  return debounced
}
