// ImageGenPage 拆分产物：生成配置/引擎/模型状态机（行为零变化，T6-10.1）
import { useCallback, useEffect, useMemo, useState } from 'react'
import { message } from 'antd'
import {
  getImageBackendInfo, getCharacters, getComfyUIStatus, getSystemStats,
  getComfyUILoras, startComfyUI, stopComfyUI,
  openImageSaveDir, openNovelImagesDir,
  type SystemStats,
} from '../api/image'
import { getEngines, testEngineConnection, setActiveEngine, setEngineDefaultModel, type EngineConfig } from '../api/engines'
import { setImageBackend as setImageBackendAPI } from '../api/settings'
import { filterLorasByModel, loraFamily, loraFamiliesForModel } from '../utils/loraFilter'
import {
  backendLabel, classifyModel, loraLabel,
  DASHSCOPE_DEFAULT_MODEL, DASHSCOPE_EDIT_MODELS,
} from '../components/imagegen/meta'
import type { ImageMode } from '../components/imagegen/types'
import { usePollingGate } from './usePollingGate'

/** 提取错误消息（unknown 收窄后统一取 message；无则用 fallback） */
function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

export function useImageGenConfig() {
  // ── 核心状态 ──
  const [mode, setMode] = useState<ImageMode>('txt2img')
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [initImage, setInitImage] = useState('')
  const [denoise, setDenoise] = useState(0.65)
  const [frames, setFrames] = useState(97)
  const [fps, setFps] = useState(8)
  const [model, setModel] = useState('krea2')
  const [seed, setSeed] = useState(0)
  const [count, setCount] = useState(1)
  const [customWidth, setCustomWidth] = useState(1024)
  const [customHeight, setCustomHeight] = useState(1024)
  const [backend, setBackend] = useState('xai')
  const [selectedLoras, setSelectedLoras] = useState<string[]>([])
  const [comfyLoras, setComfyLoras] = useState<string[]>([])
  const [loraLoading, setLoraLoading] = useState(false)
  const [loraError, setLoraError] = useState('')

  // ── 引擎 & 后端状态 ──
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [backendSwitching, setBackendSwitching] = useState(false)
  const [engineRunning, setEngineRunning] = useState(false)
  const [engineStarting, setEngineStarting] = useState(false)
  const [engineModelCount, setEngineModelCount] = useState(0)
  const [sysStats, setSysStats] = useState<SystemStats | null>(null)

  const [characters, setCharacters] = useState<{ id: string; name: string }[]>([])

  // 系统级后台轮询治理：页面不可见（窗口最小化/切走）时各轮询空转零成本
  const pollable = usePollingGate()

  const comfyModels = useMemo(() => [
    { label: 'Krea2 Turbo', value: 'krea2' },
    { label: 'Z-Image-Turbo', value: 'z-image-turbo' },
    // T6-4：flux 已有真实工作流；未装 flux1-schnell.safetensors 等模型时后端返回中文错误
    { label: 'Flux Schnell', value: 'flux' },
    { label: '流程图 / 框架图（代码渲染）', value: 'diagram' },
  ], [])

  // LoRA 选项：动态读取 ComfyUI 实际 models/loras 列表，并按当前模型族过滤，
  // 避免硬编码文件名与本地安装不一致导致提交 400，也避免跨模型误选
  const loraOptions = useMemo(
    () => (backend === 'comfyui'
      ? filterLorasByModel(model, comfyLoras).map((name) => ({ label: loraLabel(name), value: name }))
      : []),
    [backend, comfyLoras, model],
  )

  const modelOptions = useMemo(() => {
    if (backend === 'comfyui') return comfyModels
    // 百炼改图：官方编辑模型固定三档（引擎目录不含 dashscope，不能经 engines 枚举）
    if (backend === 'dashscope') {
      return DASHSCOPE_EDIT_MODELS.map((m) => ({ label: m, value: m }))
    }
    if (backend === 'xai') {
      const xaiEngine = engines.find(e => e.id === 'xai')
      const imgModels = (xaiEngine?.models || []).filter(m => classifyModel(m.id) === 'image')
      if (imgModels.length > 0) return imgModels.map(m => ({ label: m.id, value: m.id }))
      return [{ label: 'grok-imagine-image', value: 'grok-imagine-image' }]
    }
    const eng = engines.find(e => e.id === backend)
    const imgModels = (eng?.models || []).filter(m => classifyModel(m.id) === 'image')
    if (imgModels.length > 0) return imgModels.map(m => ({ label: m.id, value: m.id }))
    return model ? [{ label: model, value: model }] : []
  }, [backend, engines, model, comfyModels])

  // ── 初始化 ──
  useEffect(() => {
    (async () => {
      try {
        const info = await getImageBackendInfo()
        if (info?.backend) setBackend(info.backend)
        if (info?.model) setModel(info.model)
        const chars = await getCharacters()
        setCharacters(chars || [])
      } catch (_) { /* ignore */ }
      try {
        const list = await getEngines()
        setEngines(list || [])
      } catch (_) { /* ignore */ }
    })()
  }, [])

  // ── 引擎状态轮询 ──
  useEffect(() => {
    const isLocal = ['comfyui', 'herdsman', 'ollama'].includes(backend)
    if (!isLocal) { setEngineRunning(true); setEngineModelCount(0); return }

    // ComfyUI 用专用状态 API，其他本地引擎用通用 testEngineConnection
    if (backend === 'comfyui') {
      const check = async () => {
        if (!pollable) return
        try {
          const s = await getComfyUIStatus()
          const running = !!s?.running
          setEngineRunning(running)
          if (running) setEngineStarting(false)
          setEngineModelCount(0)
        } catch (_) { setEngineRunning(false); setEngineModelCount(0) }
      }
      check()
      const timer = setInterval(check, 5000)
      return () => clearInterval(timer)
    }

    const check = async () => {
      if (!pollable) return
      try {
        const status = await testEngineConnection(backend)
        setEngineRunning(!!status?.connected)
        setEngineModelCount(status?.model_count || 0)
      } catch (_) { setEngineRunning(false); setEngineModelCount(0) }
    }
    check()
    const timer = setInterval(check, 8000)
    return () => clearInterval(timer)
  }, [backend, pollable])

  // ── ComfyUI LoRA 列表动态加载 ──
  const refreshComfyLoras = useCallback(async () => {
    if (backend !== 'comfyui') {
      setComfyLoras([])
      setLoraError('')
      setLoraLoading(false)
      return
    }
    setLoraLoading(true)
    const { list, error } = await getComfyUILoras()
    setComfyLoras(list)
    setLoraError(error || '')
    setLoraLoading(false)
    // 已选 LoRA 若不在最新列表中则清除，避免提交无效名称
    setSelectedLoras((prev) => (list.length ? prev.filter((v) => list.includes(v)) : prev))
  }, [backend])

  // 合并原「engineRunning 就绪触发」与「挂载 + 30s 轮询」两个 effect：挂载序列只发一次请求
  // （engineRunning 初始恒为 false，就绪翻转时才立即拉取），非 ComfyUI 后端不再起 30s
  // interval（原先每 tick 推新 [] 引用导致消费者无谓重渲染）
  useEffect(() => {
    if (backend !== 'comfyui') {
      refreshComfyLoras() // 清理残留 LoRA 状态（仅一次，不起 interval）
      return
    }
    if (engineRunning && pollable) refreshComfyLoras()
    const timer = setInterval(() => { if (pollable) void refreshComfyLoras() }, 30000)
    return () => clearInterval(timer)
  }, [backend, engineRunning, pollable, refreshComfyLoras])

  // 切换模型时清掉不属于该模型族的已选 LoRA，避免提交无效 lora_name
  useEffect(() => {
    const allowed = loraFamiliesForModel(model)
    setSelectedLoras((prev) => prev.filter((v) => allowed.includes(loraFamily(v))))
  }, [model])

  // ── 系统状态轮询（所有后端显示 CPU+内存，GPU 仅 ComfyUI） ──
  useEffect(() => {
    const fetchStats = async () => {
      if (!pollable) return
      try {
        const s = await getSystemStats()
        if (s) setSysStats(s)
      } catch (_) { /* ignore */ }
    }
    fetchStats()
    const timer = setInterval(fetchStats, 5000)
    return () => clearInterval(timer)
  }, [backend, pollable])

  // ── 切换后端 ──
  const handleSwitchBackend = useCallback(async (newBackend: string) => {
    if (newBackend === backend) return
    setBackendSwitching(true)
    try {
      let defaultModel = ''
      if (newBackend === 'comfyui') defaultModel = 'krea2'
      else if (newBackend === 'xai') defaultModel = 'grok-imagine-image'
      else if (newBackend === 'dashscope') defaultModel = DASHSCOPE_DEFAULT_MODEL
      else {
        const eng = engines.find(e => e.id === newBackend)
        const img = (eng?.models || []).filter(m => classifyModel(m.id) === 'image')
        if (img.length > 0) defaultModel = img[0].id
      }
      await setImageBackendAPI(newBackend, '', defaultModel, '')
      setBackend(newBackend)
      if (defaultModel) setModel(defaultModel)
      if (newBackend !== 'comfyui') setSelectedLoras([])
      // 切换到 ComfyUI 而服务尚未启动时自动启动，避免面板 LoRA 空白、无法生图
      if (newBackend === 'comfyui') {
        try {
          const st = await getComfyUIStatus()
          if (!st?.running) {
            setEngineStarting(true)
            await startComfyUI()
            message.success('ComfyUI 正在启动，就绪后自动加载 LoRA…')
          }
        } catch (err: unknown) {
          setEngineStarting(false)
          message.error(errText(err, 'ComfyUI 启动失败，请检查安装路径'))
        }
      }
    } catch (err: unknown) { message.error(errText(err, '切换失败')) }
    finally { setBackendSwitching(false) }
  }, [backend, engines])

  // ── 引擎启停 ──
  const handleStartEngine = useCallback(async () => {
    setEngineStarting(true)
    try {
      if (backend === 'comfyui') {
        await startComfyUI()
        message.success('ComfyUI 启动中，等待连接...（约需 30-60 秒）')
        // 不清除 engineStarting，让轮询检测到运行后再切换状态
      } else {
        await setActiveEngine(backend)
        if (model) await setEngineDefaultModel(backend, model)
        setEngineRunning(true)
        message.success(`${backendLabel(backend)} 已启动`)
        setEngineStarting(false)
      }
    } catch (err: unknown) {
      message.error(errText(err, '启动失败'))
      setEngineStarting(false)
    }
  }, [backend, model])

  const handleStopEngine = useCallback(async () => {
    try {
      if (backend === 'comfyui') {
        await stopComfyUI()
        message.success('ComfyUI 已停止')
      }
      setEngineRunning(false)
    } catch (err: unknown) { message.error(errText(err, '停止失败')) }
  }, [backend])

  // ── 目录操作 ──
  const handleOpenDir = useCallback(async () => {
    try { await openImageSaveDir() } catch (err: unknown) { message.error(errText(err, '打开失败')) }
  }, [])
  const handleOpenNovelDir = useCallback(async () => {
    try { await openNovelImagesDir() } catch (err: unknown) { message.error(errText(err, '打开失败')) }
  }, [])

  return {
    mode, setMode, prompt, setPrompt, negative, setNegative, size, setSize,
    initImage, setInitImage, denoise, setDenoise, frames, setFrames, fps, setFps,
    model, setModel, seed, setSeed, count, setCount,
    customWidth, setCustomWidth, customHeight, setCustomHeight,
    backend, setBackend, selectedLoras, setSelectedLoras,
    comfyLoras, loraOptions, loraLoading, loraError, refreshComfyLoras,
    engines, backendSwitching, engineRunning, engineStarting, engineModelCount, sysStats,
    modelOptions, characters,
    handleSwitchBackend, handleStartEngine, handleStopEngine,
    handleOpenDir, handleOpenNovelDir,
  }
}
