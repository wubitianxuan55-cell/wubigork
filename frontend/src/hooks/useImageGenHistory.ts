// ImageGenPage 拆分产物：历史/灯箱状态与结果操作（行为零变化，T6-10.1）
import { useCallback, useEffect, useState } from 'react'
import { message } from 'antd'
import { setCharacterPortrait as setPortrait, readFileAsDataURL } from '../api/image'
import { restoreHistoryImages } from '../components/imagegen/historyMeta'
import { downloadFileName } from '../components/imagegen/media'
import { loadHistoryMeta, saveHistoryMeta, resolveResultImage } from '../components/imagegen/meta'
import { inShellEnv } from '../gaea/lib/pickFile'
import { dataUrlToBlob, saveExportBlob } from '../gaea/lib/saveFile'
import type { GenResult } from '../components/imagegen/types'

export interface UseImageGenHistoryOptions {
  setPrompt: (v: string) => void
  setNegative: (v: string) => void
  setSeed: (v: number) => void
  setSize: (v: string) => void
}

/** 挂载时回填 dataURL 的最大条数（v4.351 有界化）：每条 dataURL 数 MB 级，
 *  全量回填=内存随历史无界增长；窗口外条目缩略图显示占位，选中/下载/复用时
 *  resolveResultImage 按需读文件。 */
export const RESTORE_LIMIT = 48

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
        const restored = await restoreHistoryImages(history.slice(0, RESTORE_LIMIT), readFileAsDataURL)
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
    // T6-4.2：按实际媒体类型命名（t2v 输出 webp 则 .webp，不再固定 .mp4）
    const name = downloadFileName(r)
    // 审计刀B a：壳内 <a download> 不落盘（v4.162）→ dataURL→Blob 走系统另存为；
    // 浏览器保留原 <a download> 语义。
    // v4.351：失败可见化（另存为写盘抛错原落 unhandled rejection，点了没反应；
    // 用户取消保存返回 false，不算错不提示）。
    try {
      if (inShellEnv()) {
        const saved = await saveExportBlob(dataUrlToBlob(href), name)
        if (saved) message.success(`已保存：${name}`)
        return
      }
      const a = document.createElement('a')
      a.href = href
      a.download = name
      a.click()
    } catch (err) {
      message.error(`保存失败：${err instanceof Error ? err.message : String(err)}`)
    }
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
