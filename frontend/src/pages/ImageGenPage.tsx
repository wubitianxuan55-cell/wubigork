import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Typography, Button, Space, message } from 'antd'
import {
  PictureOutlined, FolderOpenOutlined,
  SwapOutlined, VideoCameraOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import Lightbox from '../components/Lightbox'
import CustomTemplateModal from '../components/imagegen/CustomTemplateModal'
import TemplatePickerModal from '../components/imagegen/TemplatePickerModal'
import { ControlPanel } from '../components/imagegen/ControlPanel'
import { ResultStage } from '../components/imagegen/ResultStage'
import { HistoryRail } from '../components/imagegen/HistoryRail'
import { StatusDot } from '../components/imagegen/ui'
import {
  TEMPLATES,
  loadCustomTemplates, saveCustomTemplates, generateTemplateId,
  type Template, type CustomTemplate,
} from '../data/imageTemplates'
import {
  getImageBackendInfo, getCharacters,
  getComfyUIStatus, getSystemStats,
  getComfyUILoras,
  generateImage, startComfyUI, stopComfyUI,
  generateMedia, type MediaParams,
  openImageSaveDir, openNovelImagesDir,
  setCharacterPortrait as setPortrait,
  type SystemStats,
} from '../api/image'
import { getEngines, testEngineConnection, setActiveEngine, setEngineDefaultModel, type EngineConfig } from '../api/engines'
import { setImageBackend as setImageBackendAPI } from '../api/settings'
import type { GenResult } from '../components/imagegen/types'
import { filterLorasByModel, loraFamily, loraFamiliesForModel } from '../utils/loraFilter'
import '../components/imagegen/imagegen.css'

// ── 常量 ──

const BACKEND_OPTIONS = [
  { label: '☁️ xAI 云端', value: 'xai' },
  { label: '🏠 ComfyUI 本地', value: 'comfyui' },
  { label: '🐄 Herdsman 本地', value: 'herdsman' },
  { label: '🦙 Ollama 本地', value: 'ollama' },
]

// ── 模型分类（与 ModelCenterPage 保持一致） ──

function classifyModel(id: string): string {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle')) return 'image'
  return 'llm'
}

/** LoRA 文件名 → 可读标签（去掉扩展名/路径，下划线转空格） */
function loraLabel(name: string): string {
  const base = name.replace(/\.(safetensors|pt|bin|ckpt|sft)$/i, '')
  const rel = base.replace(/\\/g, '/')
  const file = rel.split('/').pop() || rel
  return file.replace(/_/g, ' ')
}

// ── 主组件 ──

const ImageGenPage: React.FC = () => {
  // ── 核心状态 ──
  const [mode, setMode] = useState<'txt2img' | 'img2img' | 't2v'>('txt2img')
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
  const [generating, setGenerating] = useState(false)
  const [elapsed, setElapsed] = useState(0)
  const [lastTime, setLastTime] = useState(0)
  const [genError, setGenError] = useState('')

  // ── 引擎 & 后端状态 ──
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [backendSwitching, setBackendSwitching] = useState(false)
  const [engineRunning, setEngineRunning] = useState(false)
  const [engineStarting, setEngineStarting] = useState(false)
  const [engineModelCount, setEngineModelCount] = useState(0)
  const [sysStats, setSysStats] = useState<SystemStats | null>(null)

  // ── 结果 & 历史 ──
  const [results, setResults] = useState<GenResult[]>([])
  const [history, setHistory] = useState<GenResult[]>([])
  const [lightboxIndex, setLightboxIndex] = useState(-1)
  const [characters, setCharacters] = useState<{ id: string; name: string }[]>([])

  const [templatePickerOpen, setTemplatePickerOpen] = useState(false)
  const [customTemplates, setCustomTemplates] = useState<CustomTemplate[]>(() => loadCustomTemplates())
  const [customModalOpen, setCustomModalOpen] = useState(false)
  const [editingCustom, setEditingCustom] = useState<CustomTemplate | null>(null)
  const [customLabel, setCustomLabel] = useState('')
  const [customDescription, setCustomDescription] = useState('')
  const [customSize, setCustomSize] = useState('')
  const [customPrompt, setCustomPrompt] = useState('')
  const [customNegative, setCustomNegative] = useState('')

  const generatingRef = useRef(false)

  const comfyModels = useMemo(() => [
    { label: 'Krea2 Turbo', value: 'krea2' },
    { label: 'Z-Image-Turbo', value: 'z-image-turbo' },
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
      try {
        const status = await testEngineConnection(backend)
        setEngineRunning(!!status?.connected)
        setEngineModelCount(status?.model_count || 0)
      } catch (_) { setEngineRunning(false); setEngineModelCount(0) }
    }
    check()
    const timer = setInterval(check, 8000)
    return () => clearInterval(timer)
  }, [backend])

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

  useEffect(() => {
    if (backend === 'comfyui' && engineRunning) refreshComfyLoras()
  }, [backend, engineRunning, refreshComfyLoras])

  useEffect(() => {
    refreshComfyLoras()
    const timer = setInterval(refreshComfyLoras, 30000)
    return () => clearInterval(timer)
  }, [refreshComfyLoras])

  // 切换模型时清掉不属于该模型族的已选 LoRA，避免提交无效 lora_name
  useEffect(() => {
    const allowed = loraFamiliesForModel(model)
    setSelectedLoras((prev) => prev.filter((v) => allowed.includes(loraFamily(v))))
  }, [model])

  // ── 系统状态轮询（所有后端显示 CPU+内存，GPU 仅 ComfyUI） ──
  useEffect(() => {
    const fetchStats = async () => {
      try {
        const s = await getSystemStats()
        if (s) setSysStats(s)
      } catch (_) { /* ignore */ }
    }
    fetchStats()
    const timer = setInterval(fetchStats, 5000)
    return () => clearInterval(timer)
  }, [backend])

  // ── 生成计时器 ──
  useEffect(() => {
    if (!generating) { setElapsed(0); return }
    const start = Date.now()
    const timer = setInterval(() => setElapsed(Math.round((Date.now() - start) / 1000)), 1000)
    return () => clearInterval(timer)
  }, [generating])

  // ── 生成 ──
  const handleGenerate = useCallback(async () => {
    if (!prompt.trim()) { message.warning(mode === 't2v' ? '请输入视频画面描述' : '请输入图片描述'); return }
    if (generatingRef.current) return
    if (mode === 'img2img' && !initImage) { message.warning('请先上传参考图'); return }
    if (mode !== 'txt2img' && backend !== 'comfyui') {
      message.warning('图生图 / 文生视频仅支持 ComfyUI 本地后端，请先在左侧切换引擎')
      return
    }
    generatingRef.current = true
    setGenerating(true)
    setGenError('')
    setResults([])
    setLightboxIndex(-1)
    const genStart = Date.now()
    try {
      const finalSize = size === 'custom' ? `${customWidth}x${customHeight}` : size
      const loraStr = selectedLoras.join(',')
      const mediaParams: MediaParams = {
        prompt, negative, size: finalSize, model, seed, lora: loraStr,
        count: mode === 't2v' ? 1 : count,
        mode,
      }
      if (mode === 'img2img') { mediaParams.initImage = initImage; mediaParams.denoise = denoise }
      if (mode === 't2v') { mediaParams.frames = frames; mediaParams.fps = fps }
      const res: { error?: string; images?: GenResult[]; results?: GenResult[] } = mode === 'txt2img'
        ? await generateImage(prompt, negative, finalSize, model, seed, count, loraStr)
        : await generateMedia(mediaParams)
      if (res?.error) {
        setGenError(res.error)
        message.error(res.error)
      } else if (res?.images?.length) {
        const genResults = res.images
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success(mode === 't2v' ? '✨ 视频已生成' : `✨ 已生成 ${genResults.length} 张图片`)
      } else if (res?.results?.length) {
        const genResults = res.results
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success(mode === 't2v' ? '✨ 视频已生成' : `✨ 已生成 ${genResults.length} 张图片`)
      }
    } catch (err: any) {
      setGenError(err?.message || '生成失败')
      message.error(err?.message || '生成失败')
    } finally {
      generatingRef.current = false
      setGenerating(false)
    }
  }, [prompt, negative, size, model, seed, count, customWidth, customHeight, selectedLoras,
    mode, initImage, denoise, frames, fps, backend])

  // ── 切换后端 ──
  const handleSwitchBackend = useCallback(async (newBackend: string) => {
    if (newBackend === backend) return
    setBackendSwitching(true)
    try {
      let defaultModel = ''
      if (newBackend === 'comfyui') defaultModel = 'krea2'
      else if (newBackend === 'xai') defaultModel = 'grok-imagine-image'
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
        } catch (err: any) {
          setEngineStarting(false)
          message.error(err?.message || 'ComfyUI 启动失败，请检查安装路径')
        }
      }
    } catch (err: any) { message.error(err?.message || '切换失败') }
    finally { setBackendSwitching(false) }
  }, [backend, engines])

  // ── 切换模式：非 ComfyUI 后端仅保留文生图 ──
  const handleSwitchMode = useCallback((m: 'txt2img' | 'img2img' | 't2v') => {
    setMode(m)
    setResults([])
    setLightboxIndex(-1)
  }, [])

  // ── 引擎启停 ──
  const isLocalEngine = ['comfyui', 'herdsman', 'ollama'].includes(backend)

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
        message.success(`${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 已启动`)
        setEngineStarting(false)
      }
    } catch (err: any) {
      message.error(err?.message || '启动失败')
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
    } catch (err: any) { message.error(err?.message || '停止失败') }
  }, [backend])

  // ── 目录操作 ──
  const handleOpenDir = useCallback(async () => {
    try { await openImageSaveDir() } catch (err: any) { message.error(err?.message || '打开失败') }
  }, [])
  const handleOpenNovelDir = useCallback(async () => {
    try { await openNovelImagesDir() } catch (err: any) { message.error(err?.message || '打开失败') }
  }, [])

  // ── 结果操作 ──
  const handleDownload = useCallback((i: number) => {
    const r = history[i]
    if (!r) return
    const a = document.createElement('a')
    a.href = r.image
    a.download = `gaea-${Date.now()}-seed${r.seed}.png`
    a.click()
  }, [history])

  const handleReuse = useCallback((i: number) => {
    const r = history[i]
    if (!r) return
    setPrompt(r.prompt)
    if (r.negative) setNegative(r.negative)
    if (r.seed) setSeed(r.seed)
    if (r.size) setSize(r.size)
  }, [history])

  const handleDelete = useCallback((i: number) => {
    setHistory((prev) => prev.filter((_, idx) => idx !== i))
  }, [])

  const handleSetPortrait = useCallback(async (i: number, charID: string) => {
    const r = history[i]
    if (!r) return
    try {
      await setPortrait(charID, r.image)
      message.success('已设为角色剧照')
    } catch (err: any) { message.error(err?.message || '设置失败') }
  }, [history])

  // ── 模板操作 ──
  const applyTemplate = useCallback((t: Template) => {
    setPrompt((p) => p ? p + '，' + t.prompt : t.prompt)
    const neg = t.negative
    if (neg) setNegative((n) => n ? n + ', ' + neg : neg)
  }, [])

  const openCustomAdd = useCallback(() => {
    setEditingCustom(null)
    setCustomLabel('')
    setCustomDescription('')
    setCustomSize('')
    setCustomPrompt('')
    setCustomNegative('')
    setCustomModalOpen(true)
  }, [])

  const openCustomEdit = useCallback((t: CustomTemplate) => {
    setEditingCustom(t)
    setCustomLabel(t.label)
    setCustomDescription(t.description || '')
    setCustomSize(t.size || '')
    setCustomPrompt(t.prompt)
    setCustomNegative(t.negative || '')
    setCustomModalOpen(true)
  }, [])

  const saveCustom = useCallback(() => {
    if (!customLabel.trim() || !customPrompt.trim()) {
      message.warning('标签和 Prompt 不能为空')
      return
    }
    if (editingCustom) {
      const updated = customTemplates.map((t) =>
        t.id === editingCustom.id
          ? {
              ...t,
              label: customLabel, description: customDescription, size: customSize,
              prompt: customPrompt, negative: customNegative,
            }
          : t,
      )
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    } else {
      const created: CustomTemplate = {
        id: generateTemplateId(), label: customLabel, description: customDescription, size: customSize,
        prompt: customPrompt, negative: customNegative,
      }
      const updated = [...customTemplates, created]
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    }
    setCustomModalOpen(false)
    message.success(editingCustom ? '模板已更新' : '模板已添加')
  }, [customTemplates, editingCustom, customLabel, customDescription, customSize, customPrompt, customNegative])

  const deleteCustom = useCallback((id: string) => {
    const updated = customTemplates.filter((t) => t.id !== id)
    setCustomTemplates(updated)
    saveCustomTemplates(updated)
    message.success('已删除')
  }, [customTemplates])

  // ── 渲染 ──
  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, overflow: 'hidden' }}>
      {/* 顶栏 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12, flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{
            width: 34, height: 34, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(var(--accent-rgb), 0.14)', border: '1px solid rgba(var(--accent-rgb), 0.25)',
            color: 'var(--color-primary)', fontSize: 16,
          }}>
            <PictureOutlined />
          </div>
          <div>
            <Typography.Title level={5} style={{ color: C('color-text'), margin: 0, fontSize: 16, fontWeight: 600, lineHeight: 1.2 }}>
              AI 绘梦
            </Typography.Title>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 3 }}>
              <StatusDot tone={engineStarting ? 'warn' : isLocalEngine ? (engineRunning ? 'ok' : 'idle') : 'ok'} />
              <span style={{ fontSize: 11, color: C('color-text-secondary') }}>
                {engineStarting
                  ? '引擎启动中...'
                  : isLocalEngine
                    ? (engineRunning ? `${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 运行中` : '引擎未连接')
                    : `${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 云端`}
              </span>
            </div>
          </div>
        </div>
        <Space size={6}>
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenNovelDir} title="小说图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 13, padding: '0 6px' }} />
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenDir} title="生成图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 13, padding: '0 6px' }} />
        </Space>
      </div>

      {/* 模式导航：文生图 / 图生图 / 文生视频 */}
      <div className="ig-mode-nav" role="tablist" aria-label="生成模式">
        <button
          type="button"
          role="tab"
          aria-selected={mode === 'txt2img'}
          className={`ig-mode-item${mode === 'txt2img' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('txt2img')}
        >
          <PictureOutlined /> 文生图
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mode === 'img2img'}
          className={`ig-mode-item${mode === 'img2img' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('img2img')}
        >
          <SwapOutlined /> 图生图
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mode === 't2v'}
          className={`ig-mode-item${mode === 't2v' ? ' is-active' : ''}`}
          onClick={() => handleSwitchMode('t2v')}
        >
          <VideoCameraOutlined /> 文生视频
        </button>
      </div>

      {/* 主工作区：左控制面板 + 中结果舞台 + 右历史胶片 */}
      <div style={{ flex: 1, display: 'flex', gap: 12, minHeight: 0 }}>
        {/* 左栏 — 控制面板 */}
        <div style={{ width: 350, flexShrink: 0, overflowY: 'auto', overflowX: 'hidden', paddingRight: 2 }}>
          <ControlPanel
            mode={mode}
            prompt={prompt} negative={negative}
            onPromptChange={setPrompt} onNegativeChange={setNegative}
            onOpenTemplatePicker={() => setTemplatePickerOpen(true)}
            model={model} modelOptions={modelOptions}
            onModelChange={setModel}
            size={size} onSizeChange={setSize}
            customWidth={customWidth} customHeight={customHeight}
            onCustomWidthChange={setCustomWidth} onCustomHeightChange={setCustomHeight}
            seed={seed} onSeedChange={setSeed}
            count={count} onCountChange={setCount}
            initImage={initImage} onInitImageChange={setInitImage}
            denoise={denoise} onDenoiseChange={setDenoise}
            frames={frames} onFramesChange={setFrames}
            fps={fps} onFpsChange={setFps}
            selectedLoras={selectedLoras}
            loraOptions={backend === 'comfyui' ? loraOptions : []}
            loraLoading={loraLoading}
            loraError={loraError}
            onLorasChange={setSelectedLoras}
            backend={backend} backendSwitching={backendSwitching}
            engineRunning={engineRunning} engineStarting={engineStarting} engineModelCount={engineModelCount}
            onSwitchBackend={handleSwitchBackend}
            onStartEngine={handleStartEngine} onStopEngine={handleStopEngine}
            sysStats={sysStats}
            generating={generating} elapsed={elapsed} lastTime={lastTime}
            onGenerate={handleGenerate}
          />
        </div>

        {/* 中间 — 结果舞台 */}
        <div style={{ flex: 1, overflow: 'auto', minWidth: 0 }}>
          <ResultStage
            results={results} generating={generating} error={genError} mode={mode}
            initImage={initImage}
            onPreview={(i) => setLightboxIndex(i)}
            onDownload={handleDownload}
            onReuse={handleReuse}
            onDelete={handleDelete}
            onRetry={handleGenerate}
            onOpenTemplatePicker={() => setTemplatePickerOpen(true)}
          />
        </div>

        {/* 右侧 — 历史胶片 */}
        <div style={{ width: 176, flexShrink: 0, overflowY: 'auto', overflowX: 'hidden',
          background: 'var(--gaea-glass-bg, var(--bg-elevated))', borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--md-sys-color-outline-variant)', padding: 10 }}>
          <HistoryRail
            history={history}
            selectedIndex={lightboxIndex}
            onSelect={(i) => setLightboxIndex(i)}
            onClear={() => setHistory([])}
          />
        </div>
      </div>

      {/* 灯箱 */}
      {lightboxIndex >= 0 && (
        <Lightbox
          results={history}
          index={lightboxIndex}
          characters={characters}
          onClose={() => setLightboxIndex(-1)}
          onIndexChange={setLightboxIndex}
          onDownload={handleDownload}
          onReuse={handleReuse}
          onSetPortrait={handleSetPortrait}
        />
      )}

      {/* 自定义模板弹窗 */}
      <CustomTemplateModal
        open={customModalOpen}
        editing={!!editingCustom}
        label={customLabel} onLabelChange={setCustomLabel}
        description={customDescription} onDescriptionChange={setCustomDescription}
        size={customSize} onSizeChange={setCustomSize}
        prompt={customPrompt} onPromptChange={setCustomPrompt}
        negative={customNegative} onNegativeChange={setCustomNegative}
        onSave={saveCustom}
        onCancel={() => setCustomModalOpen(false)}
      />

      {/* 模板选择弹窗 */}
      <TemplatePickerModal
        open={templatePickerOpen}
        onClose={() => setTemplatePickerOpen(false)}
        customTemplates={customTemplates}
        onSelect={applyTemplate}
        onAddCustom={openCustomAdd}
        onEditCustom={openCustomEdit}
        onDeleteCustom={deleteCustom}
      />
    </div>
  )
}

export default ImageGenPage
