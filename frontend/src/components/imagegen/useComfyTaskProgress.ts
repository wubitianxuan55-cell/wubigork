// useComfyTaskProgress.ts — 生成进度轮询（绘梦 useImageGenQueue 同数据源的抽出版）。
//
// 数据来源：GetComfyUITaskProgress（后端只在 ComfyUI 后端逐节点上报；其余后端
// status/percent 缺失 → 前端如实显示不定态 + 本地计时，不编造百分比）。
// 纪律：只在 active 时轮询；active 结束/卸载立即清定时器；轮询失败保留上一帧
// （单次读失败不该让进度条闪烁或清零）。
import { useEffect, useRef, useState } from 'react'
import { getComfyUITaskProgress, getImageBackendInfo } from '../../api/image'

export interface ComfyTaskProgressView {
  status: string
  elapsed: number
  /** -1 = 未知（后端未提供）；>=0 = 确定进度。 */
  percent: number
  node: string
  /** 图像后端类型（comfyui 才有实时进度）。 */
  backend: string
}

const EMPTY: ComfyTaskProgressView = { status: '', elapsed: 0, percent: -1, node: '', backend: '' }

export function useComfyTaskProgress(active: boolean, intervalMs = 1000): ComfyTaskProgressView {
  const [view, setView] = useState<ComfyTaskProgressView>(EMPTY)
  const backendRef = useRef('')

  // 后端类型只需解析一次（原罪与绘梦共用同一图像后端配置）
  useEffect(() => {
    let live = true
    getImageBackendInfo()
      .then((info) => { if (live) backendRef.current = info?.backend || '' })
      .catch(() => { /* 读不到后端类型：按「无实时进度」处理，不阻断生成 */ })
    return () => { live = false }
  }, [])

  useEffect(() => {
    if (!active) {
      setView(EMPTY)
      return
    }
    let cancelled = false
    const tick = async () => {
      try {
        const p = await getComfyUITaskProgress()
        if (cancelled) return
        setView({
          status: p.status || '',
          elapsed: p.elapsed || 0,
          percent: typeof p.percent === 'number' ? p.percent : -1,
          node: p.node || '',
          backend: backendRef.current,
        })
      } catch {
        // 单次读失败保留上一帧（进度条不闪回）
      }
    }
    void tick()
    const timer = setInterval(() => {
      // v4.365：页面隐藏时跳过本 tick（读帧+setState 不再后台空转）
      if (document.hidden) return
      void tick()
    }, intervalMs)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [active, intervalMs])

  return view
}
