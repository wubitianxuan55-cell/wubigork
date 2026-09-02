// ImageGenPage 拆分产物：生成队列/任务执行状态机（行为零变化，T6-10.1）
import { useCallback, useEffect, useRef, useState } from 'react'
import { usePollingGate } from './usePollingGate'
import { message } from 'antd'
import {
  generateImage, cancelImageGeneration, getComfyUITaskProgress,
  generateMedia, type MediaParams,
  generateDiagram,
} from '../api/image'
import { renderMermaidToPng } from '../utils/mermaidPng'
import { markQueueCanceled, shouldSubmitNext, afterTaskStatus } from '../components/imagegen/queue'
import { backendSupportsMode } from '../components/imagegen/meta'
import type { GenResult, GenTask, QueueEntry, ImageMode } from '../components/imagegen/types'

/** handleGenerate 构建 GenTask 所需的生成配置快照（页面从 useImageGenConfig 汇总注入）。 */
export interface ImageGenQueueConfig {
  prompt: string
  mode: ImageMode
  initImage: string
  backend: string
  negative: string
  size: string
  customWidth: number
  customHeight: number
  model: string
  seed: number
  count: number
  selectedLoras: string[]
  denoise: number
  frames: number
  fps: number
}

export interface UseImageGenQueueOptions {
  setHistory: React.Dispatch<React.SetStateAction<GenResult[]>>
  setLightboxIndex: React.Dispatch<React.SetStateAction<number>>
  config: ImageGenQueueConfig
}

export function useImageGenQueue({ setHistory, setLightboxIndex, config }: UseImageGenQueueOptions) {
  const [generating, setGenerating] = useState(false)
  const [elapsed, setElapsed] = useState(0)
  const [lastTime, setLastTime] = useState(0)
  const [genError, setGenError] = useState('')
  const [comfyProgress, setComfyProgress] = useState({ status: '', elapsed: 0, percent: -1, node: '' })
  const [results, setResults] = useState<GenResult[]>([])
  // v4.5.2：ComfyUI 进度轮询接入系统级后台轮询门控
  const gate = usePollingGate()
  const [pendingCount, setPendingCount] = useState(0)
  const [queueItems, setQueueItems] = useState<QueueEntry[]>([])

  const generatingRef = useRef(false)
  const pendingRef = useRef<QueueEntry[]>([])
  const queueSeq = useRef(0)
  // 本地取消标记：收到取消后队列停止继续提交，直到用户手动发起新一轮生成
  const cancelArmedRef = useRef(false)
  const canvasRef = useRef<HTMLDivElement>(null)

  // ── 生成队列 ──
  const executeTask = useCallback(async (task: GenTask) => {
    generatingRef.current = true
    setGenerating(true)
    setGenError('')
    setResults([])
    setLightboxIndex(-1)
    const genStart = Date.now()
    let ok = true
    try {
      if (task.model === 'diagram') {
        const res = await generateDiagram(task.prompt)
        if (res?.error) {
          setGenError(res.error)
          message.error(res.error)
          return false
        }
        if (!res.code) {
          setGenError('AI 未返回有效的图表代码，请调整描述后重试')
          message.error('AI 未返回有效的图表代码，请调整描述后重试')
          return false
        }
        const png = await renderMermaidToPng(res.code)
        if (!png) {
          setGenError('图表渲染失败，请调整描述后重试')
          message.error('图表渲染失败，请调整描述后重试')
          return false
        }
        const diagramResult: GenResult = {
          image: png.dataUrl,
          seed: 0,
          time: Math.round((Date.now() - genStart) / 1000),
          prompt: task.prompt,
          negative: task.negative,
          model: '流程图 / 框架图',
          size: `${png.width}x${png.height}`,
          mode: task.mode,
          count: task.count,
          selectedLoras: task.selectedLoras,
          denoise: task.denoise,
          frames: task.frames,
          fps: task.fps,
          customWidth: task.customWidth,
          customHeight: task.customHeight,
        }
        setResults([diagramResult])
        setHistory((prev) => [diagramResult, ...prev.filter((h) =>
          !(h.prompt === diagramResult.prompt && h.model === diagramResult.model),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success('✔ 图表已生成')
        return true
      }
      const finalSize = task.size === 'custom' ? `${task.customWidth}x${task.customHeight}` : task.size
      const loraStr = task.selectedLoras.join(',')
      const mediaParams: MediaParams = {
        prompt: task.prompt, negative: task.negative, size: finalSize, model: task.model,
        seed: task.seed, lora: loraStr,
        count: task.mode === 't2v' ? 1 : task.count,
        mode: task.mode,
      }
      if (task.mode === 'img2img') { mediaParams.initImage = task.initImage; mediaParams.denoise = task.denoise }
      if (task.mode === 't2v') { mediaParams.frames = task.frames; mediaParams.fps = task.fps }
      const res: { error?: string; images?: GenResult[]; results?: GenResult[] } = task.mode === 'txt2img'
        ? await generateImage(task.prompt, task.negative, finalSize, task.model, task.seed, task.count, loraStr)
        : await generateMedia(mediaParams)
      if (res?.error) {
        ok = false
        setGenError(res.error)
        message.error(res.error)
      } else if (res?.images?.length) {
        const genResults = res.images.map((g) => ({
          ...g,
          mode: task.mode,
          count: task.count,
          selectedLoras: task.selectedLoras,
          denoise: task.denoise,
          frames: task.frames,
          fps: task.fps,
          customWidth: task.customWidth,
          customHeight: task.customHeight,
        }))
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success(task.mode === 't2v' ? '✨ 视频已生成' : `✨ 已生成 ${genResults.length} 张图片`)
      } else if (res?.results?.length) {
        const genResults = res.results.map((g) => ({
          ...g,
          mode: task.mode,
          count: task.count,
          selectedLoras: task.selectedLoras,
          denoise: task.denoise,
          frames: task.frames,
          fps: task.fps,
          customWidth: task.customWidth,
          customHeight: task.customHeight,
        }))
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success(task.mode === 't2v' ? '✨ 视频已生成' : `✨ 已生成 ${genResults.length} 张图片`)
      }
    } catch (err: unknown) {
      ok = false
      const errMsg = err instanceof Error ? err.message : '生成失败'
      setGenError(errMsg)
      message.error(errMsg)
    } finally {
      generatingRef.current = false
      setGenerating(false)
    }
    return ok
  }, [setHistory, setLightboxIndex])

  const processQueue = useCallback(async () => {
    // 本地取消标记生效（cancelArmedRef）时循环立即停止：排队项不再启动
    while (shouldSubmitNext(pendingRef.current.length, cancelArmedRef.current)) {
      const entry = pendingRef.current[0]
      pendingRef.current = pendingRef.current.slice(1)
      setPendingCount(pendingRef.current.length)
      setQueueItems((prev) => prev.map((item) => item.id === entry.id ? { ...item, status: 'running' as const } : item))
      const ok = await executeTask(entry.task)
      setQueueItems((prev) => prev.map((item) =>
        item.id === entry.id && item.status !== 'canceled'
          ? { ...item, status: afterTaskStatus(cancelArmedRef.current, ok) }
          : item,
      ))
    }
  }, [executeTask])

  const enqueueTask = useCallback((task: GenTask) => {
    // 用户手动发起新一轮生成：清除本地取消标记（后端 ComfyUI 取消标记同步自动清除）
    cancelArmedRef.current = false
    const entry: QueueEntry = { id: ++queueSeq.current, task, status: 'pending' }
    pendingRef.current = [...pendingRef.current, entry]
    setQueueItems((prev) => [...prev, entry])
    setPendingCount(pendingRef.current.length)
    if (!generatingRef.current) void processQueue()
  }, [processQueue])

  const handleGenerate = useCallback(() => {
    if (!config.prompt.trim()) { message.warning(config.mode === 't2v' ? '请输入视频画面描述' : '请输入图片描述'); return }
    if (config.mode === 'img2img' && !config.initImage) { message.warning('请先上传参考图'); return }
    // 引擎固有模式约束（百炼仅改图 / GLM 仅文生图）：残留态提交前拦截，
    // 与 ControlPanel/GenerationBar 门禁同源，避免点击后才被后端拒收。
    if (!backendSupportsMode(config.backend, config.mode)) {
      message.warning(config.backend === 'glm'
        ? 'GLM 仅支持文生图，请切换到文生图模式或更换引擎'
        : '百炼仅支持改图，请切换到图生图模式或更换引擎')
      return
    }
    if (config.mode === 'img2img' && config.backend !== 'comfyui' && config.backend !== 'herdsman' && config.backend !== 'dashscope') {
      message.warning('图生图目前支持 ComfyUI / Herdsman 本地后端 / 百炼改图，请先在左侧切换引擎')
      return
    }
    if (config.mode === 't2v' && config.backend !== 'comfyui') {
      message.warning('文生视频仅支持 ComfyUI 本地后端，请先在左侧切换引擎')
      return
    }
    const task: GenTask = {
      prompt: config.prompt, negative: config.negative, size: config.size,
      customWidth: config.customWidth, customHeight: config.customHeight,
      model: config.model, seed: config.seed, count: config.count,
      selectedLoras: config.selectedLoras, mode: config.mode, initImage: config.initImage,
      denoise: config.denoise, frames: config.frames, fps: config.fps,
    }
    enqueueTask(task)
  }, [config, enqueueTask])

  const handleRegenerateMeta = useCallback((meta: GenResult) => {
    if (!meta.prompt?.trim()) return
    const task: GenTask = {
      prompt: meta.prompt,
      negative: meta.negative || '',
      size: meta.size || '1024x1024',
      customWidth: meta.customWidth || 1024,
      customHeight: meta.customHeight || 1024,
      model: meta.model || config.model,
      seed: meta.seed || 0,
      count: meta.count || 1,
      selectedLoras: meta.selectedLoras || [],
      mode: meta.mode || 'txt2img',
      initImage: '',
      denoise: meta.denoise || 0.65,
      frames: meta.frames || 97,
      fps: meta.fps || 8,
    }
    enqueueTask(task)
  }, [config.model, enqueueTask])

  const handleCancel = useCallback(async () => {
    // 本地队列立即停止提交：已有生成中的项 + 排队中的项全部标记为取消
    setQueueItems((prev) => markQueueCanceled(prev))
    pendingRef.current = []
    setPendingCount(0)
    cancelArmedRef.current = true
    try {
      const confirmed = await cancelImageGeneration()
      // 收到确认后保持本地阻断：不再自动提交，直到用户手动发起新一轮生成
      //（enqueueTask 清除标记；后端新一轮生成会自动清除其取消标记）
      void confirmed
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '取消失败')
    }
  }, [])

  // ── 生成计时器 ──
  useEffect(() => {
    if (!generating) { setElapsed(0); return }
    const start = Date.now()
    const timer = setInterval(() => setElapsed(Math.round((Date.now() - start) / 1000)), 1000)
    return () => clearInterval(timer)
  }, [generating])

  // 生成开始时画布回到顶部，避免新一轮结果被旧滚动位置遮挡
  useEffect(() => {
    if (generating) canvasRef.current?.scrollTo({ top: 0 })
  }, [generating])

  // ComfyUI 任务状态轮询
  useEffect(() => {
    if (!generating || config.backend !== 'comfyui') {
      setComfyProgress({ status: '', elapsed: 0, percent: -1, node: '' })
      return
    }
    let cancelled = false
    const tick = async () => {
      if (!gate) return
      const p = await getComfyUITaskProgress()
      if (!cancelled) setComfyProgress({ status: p.status, elapsed: p.elapsed, percent: p.percent ?? -1, node: p.node || '' })
    }
    tick()
    const timer = setInterval(tick, 2000)
    return () => { cancelled = true; clearInterval(timer) }
  }, [generating, config.backend, gate])

  return {
    generating, elapsed, lastTime, genError, comfyProgress,
    results, setResults, pendingCount, queueItems, setQueueItems, canvasRef,
    handleGenerate, handleRegenerateMeta, handleCancel,
  }
}
