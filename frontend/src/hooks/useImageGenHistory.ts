// ImageGenPage 拆分产物：历史/灯箱状态与结果操作（行为零变化，T6-10.1）
import { useCallback, useEffect, useState } from 'react'
import { message } from 'antd'
import { setCharacterPortrait as setPortrait, readFileAsDataURL } from '../api/image'
import { restoreHistoryImages } from '../components/imagegen/historyMeta'
import { downloadFileName } from '../components/imagegen/media'
import { loadHistoryMeta, saveHistoryMeta, resolveResultImage } from '../components/imagegen/meta'
import type { GenResult } from '../components/imagegen/types'

export interface UseImageGenHistoryOptions {
  setPrompt: (v: string) => void
  setNegative: (v: string) => void
  setSeed: (v: number) => void
  setSize: (v: string) => void
}

export function useImageGenHistory({ setPrompt, setNegative, setSeed, setSize }: UseImageGenHistoryOptions) {
  const [history, setHistory] = useState<GenResult[]>(() => loadHistoryMeta())
  const [lightboxIndex, setLightboxIndex] = useState(-1)

  // 历史元数据轻量持久化：小 base64 内联、大图只存 file_path（分级策略）
  useEffect(() => {
    saveHistoryMeta(history)
  }, [history])

  // 重启后历史图片恢复：无 base64 但有 file_path 的条目，经后端读取绑定回填 dataURL。
  // 仅在挂载时执行一次（history 为初始加载值）；逐条失败由 restoreHistoryImages 记录，不影响页面。
  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const restored = await restoreHistoryImages(history, readFileAsDataURL)
        if (cancelled || restored.length === 0) return
        setHistory((prev) => {
          const byPath = new Map(restored.map((it) => [it.file_path, it]))
          return prev.map((it) => (it.file_path && byPath.get(it.file_path)) || it)
        })
      } catch (err) {
        console.warn('[imagegen] 历史图片恢复流程失败', err)
      }
    })()
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ── 结果操作 ──
  const handleDownload = useCallback(async (i: number) => {
    const r = history[i]
    if (!r) return
    const href = await resolveResultImage(r)
    if (!href) {
      message.warning('图片数据不可用（本地文件缺失且无内存数据），请重新生成')
      return
    }
    const a = document.createElement('a')
    a.href = href
    // T6-4.2：按实际媒体类型命名（t2v 输出 webp 则 .webp，不再固定 .mp4）
    a.download = downloadFileName(r)
    a.click()
  }, [history])

  const handleReuse = useCallback((i: number) => {
    const r = history[i]
    if (!r) return
    setPrompt(r.prompt)
    if (r.negative) setNegative(r.negative)
    if (r.seed) setSeed(r.seed)
    if (r.size) setSize(r.size)
  }, [history, setPrompt, setNegative, setSeed, setSize])

  const handleDelete = useCallback((i: number) => {
    setHistory((prev) => prev.filter((_, idx) => idx !== i))
  }, [])

  const handleSetPortrait = useCallback(async (i: number, charID: string) => {
    const r = history[i]
    if (!r) return
    const dataUrl = await resolveResultImage(r)
    if (!dataUrl) {
      message.warning('图片数据不可用（本地文件缺失且无内存数据），请重新生成')
      return
    }
    try {
      await setPortrait(charID, dataUrl)
      message.success('已设为角色剧照')
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : '设置失败') }
  }, [history])

  return {
    history, setHistory, lightboxIndex, setLightboxIndex,
    handleDownload, handleReuse, handleDelete, handleSetPortrait,
  }
}
