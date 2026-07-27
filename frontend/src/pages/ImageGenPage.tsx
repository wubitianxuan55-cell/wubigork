import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Typography, Tag, Button, Space, message } from 'antd'
import {
  PictureOutlined, FolderOpenOutlined,
  PlayCircleOutlined, PoweroffOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import PromptPanel from '../components/PromptPanel'
import GenControls from '../components/GenControls'
import GenButton from '../components/GenButton'
import ResultGallery from '../components/ResultGallery'
import HistoryStrip from '../components/HistoryStrip'
import Lightbox from '../components/Lightbox'
import CustomTemplateModal from '../components/imagegen/CustomTemplateModal'
import TemplatePickerModal from '../components/imagegen/TemplatePickerModal'
import {
  TEMPLATES, getAllCategories,
  loadCustomTemplates, saveCustomTemplates, generateTemplateId,
  type Template, type CustomTemplate,
} from '../data/imageTemplates'
import {
  getImageBackendInfo, getCharacters,
  getComfyUIStatus, getSystemStats,
  generateImage, startComfyUI, stopComfyUI,
  openImageSaveDir, openNovelImagesDir,
  setCharacterPortrait as setPortrait,
  type SystemStats,
} from '../api/image'
import { getEngines, testEngineConnection, setActiveEngine, setEngineDefaultModel, type EngineConfig } from '../api/engines'
import { setImageBackend as setImageBackendAPI } from '../api/settings'
import type { GenResult } from '../components/ResultGallery'

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
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('vox') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle')) return 'image'
  return 'llm'
}

// ── 主组件 ──

const ImageGenPage: React.FC = () => {
  // ── 核心状态 ──
  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [model, setModel] = useState('flux')
  const [seed, setSeed] = useState(0)
  const [count, setCount] = useState(1)
  const [customWidth, setCustomWidth] = useState(1024)
  const [customHeight, setCustomHeight] = useState(1024)
  const [backend, setBackend] = useState('xai')
  const [selectedLoras, setSelectedLoras] = useState<string[]>([])
  const [generating, setGenerating] = useState(false)
  const [elapsed, setElapsed] = useState(0)
  const [lastTime, setLastTime] = useState(0)

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

  const [templateCat, setTemplateCat] = useState<string | undefined>()
  const [templatePickerOpen, setTemplatePickerOpen] = useState(false)
  const [customTemplates, setCustomTemplates] = useState<CustomTemplate[]>(() => loadCustomTemplates())
  const [customModalOpen, setCustomModalOpen] = useState(false)
  const [editingCustom, setEditingCustom] = useState<CustomTemplate | null>(null)
  const [customLabel, setCustomLabel] = useState('')
  const [customPrompt, setCustomPrompt] = useState('')
  const [customNegative, setCustomNegative] = useState('')

  const generatingRef = useRef(false)

  const comfyModels = useMemo(() => [
    { label: '🌊 Flux Dev', value: 'flux' },
    { label: '⚡ Z-Image-Turbo', value: 'z-image-turbo' },
    { label: '🎨 Krea2 (FLUX)', value: 'krea2' },
  ], [])

  const loraOptions = useMemo(() => [
    { label: '✨ 细节增强', value: 'zimage\\z-image-细节增强v2.safetensors' },
    { label: '🎨 3D卡通', value: 'zimage\\z-Image-3D卡通_V1.safetensors' },
    { label: '🌫️ 朦胧光影', value: 'zimage\\z-image-朦胧氛围光影LORA_V1.0.safetensors' },
    { label: '📷 照片写实', value: 'zimage\\z-image-照片写实.safetensors' },
    { label: '👧 少女风格', value: 'zimage\\z-image-少女-ben_nd.safetensors' },
    { label: '🔞 NSFW', value: 'zimage\\NSFW_master_ZIT_000017532.safetensors' },
  ], [])

  const modelOptions = useMemo(() => {
    if (backend === 'comfyui') return comfyModels
    if (backend === 'xai') {
      const xaiEngine = engines.find(e => e.id === 'xai')
      const imgModels = (xaiEngine?.models || []).filter(m => classifyModel(m.id) === 'image')
      if (imgModels.length > 0) return imgModels.map(m => ({ label: m.id, value: m.id }))
      return [{ label: 'grok-imagine-image-quality', value: 'grok-imagine-image-quality' }]
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
        setCharacters(chars)
      } catch (_) { /* ignore */ }
      try {
        const list = await getEngines()
        setEngines(list)
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
    if (!prompt.trim()) { message.warning('请输入图片描述'); return }
    if (generatingRef.current) return
    generatingRef.current = true
    setGenerating(true)
    setResults([])
    setLightboxIndex(-1)
    const genStart = Date.now()
    try {
      const finalSize = size === 'custom' ? `${customWidth}x${customHeight}` : size
      const loraStr = selectedLoras.join(',')
      const res = await generateImage(prompt, negative, finalSize, model, seed, count, loraStr)
      if (res?.error) {
        message.error(res.error)
      } else if (res?.images?.length) {
        const genResults = res.images
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setLastTime(Math.round((Date.now() - genStart) / 1000))
        message.success(`✨ 已生成 ${genResults.length} 张图片`)
      }
    } catch (err: any) {
      message.error(err?.message || '生成失败')
    } finally {
      generatingRef.current = false
      setGenerating(false)
    }
  }, [prompt, negative, size, model, seed, count, customWidth, customHeight, selectedLoras])

  // ── 切换后端 ──
  const handleSwitchBackend = useCallback(async (newBackend: string) => {
    if (newBackend === backend) return
    setBackendSwitching(true)
    try {
      let defaultModel = ''
      if (newBackend === 'comfyui') defaultModel = 'flux'
      else if (newBackend === 'xai') defaultModel = 'grok-imagine-image-quality'
      else {
        const eng = engines.find(e => e.id === newBackend)
        const img = (eng?.models || []).filter(m => classifyModel(m.id) === 'image')
        if (img.length > 0) defaultModel = img[0].id
      }
      await setImageBackendAPI(newBackend, '', defaultModel)
      setBackend(newBackend)
      if (defaultModel) setModel(defaultModel)
    } catch (err: any) { message.error(err?.message || '切换失败') }
    finally { setBackendSwitching(false) }
  }, [backend, engines])

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
    a.download = `wubigork-${Date.now()}-seed${r.seed}.png`
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
    if (t.negative) setNegative((n) => n ? n + ', ' + t.negative : t.negative)
  }, [])

  const openCustomAdd = useCallback(() => {
    setEditingCustom(null)
    setCustomLabel('')
    setCustomPrompt('')
    setCustomNegative('')
    setCustomModalOpen(true)
  }, [])

  const openCustomEdit = useCallback((t: CustomTemplate) => {
    setEditingCustom(t)
    setCustomLabel(t.label)
    setCustomPrompt(t.prompt)
    setCustomNegative(t.negative)
    setCustomModalOpen(true)
  }, [])

  const saveCustom = useCallback(() => {
    if (!customLabel.trim() || !customPrompt.trim()) {
      message.warning('标签和 Prompt 不能为空')
      return
    }
    if (editingCustom) {
      const updated = customTemplates.map((t) =>
        t.id === editingCustom.id ? { ...t, label: customLabel, prompt: customPrompt, negative: customNegative } : t,
      )
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    } else {
      const created: CustomTemplate = { id: generateTemplateId(), label: customLabel, prompt: customPrompt, negative: customNegative }
      const updated = [...customTemplates, created]
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    }
    setCustomModalOpen(false)
    message.success(editingCustom ? '模板已更新' : '模板已添加')
  }, [customTemplates, editingCustom, customLabel, customPrompt, customNegative])

  const deleteCustom = useCallback((id: string) => {
    const updated = customTemplates.filter((t) => t.id !== id)
    setCustomTemplates(updated)
    saveCustomTemplates(updated)
    message.success('已删除')
  }, [customTemplates])

  // ── 后端选择器（注入 GenControls） ──
  const backendSelector = (
    <div>
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, display: 'block', marginBottom: 6 }}>
        🚀 引擎
      </Typography.Text>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
        {BACKEND_OPTIONS.map((b) => {
          const selected = b.value === backend
          return (
            <div
              key={b.value}
              onClick={() => handleSwitchBackend(b.value)}
              style={{
                padding: '5px 10px',
                borderRadius: 'var(--radius-sm)',
                border: selected
                  ? '1px solid var(--color-primary)'
                  : '1px solid var(--border-subtle)',
                background: selected
                  ? 'rgba(99, 102, 241, 0.12)'
                  : 'rgba(255,255,255,0.03)',
                cursor: 'pointer',
                fontSize: 11,
                fontWeight: selected ? 600 : 400,
                color: selected ? 'var(--color-primary)' : C('color-text-secondary'),
                transition: 'all 0.15s',
                textAlign: 'center' as const,
                whiteSpace: 'nowrap' as const,
                userSelect: 'none' as const,
              }}
            >
              {b.label}
            </div>
          )
        })}
      </div>
    </div>
  )

  // ── 渲染 ──
  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, overflow: 'hidden' }}>
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        marginBottom: 12, flexShrink: 0,
      }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
          <PictureOutlined style={{ marginRight: 10 }} />AI 绘梦
        </Typography.Title>
        <Space size={8}>
          {/* 引擎连接状态 */}
          {isLocalEngine && (
            <Tag color={engineRunning ? 'green' : 'default'} style={{ borderRadius: 'var(--radius-md)', margin: 0, fontSize: 11 }}>
              {engineRunning ? `🟢 已连接 (${engineModelCount}模型)` : '⚫ 未连接'}
            </Tag>
          )}
          {!isLocalEngine && (
            <Tag color="blue" style={{ borderRadius: 'var(--radius-md)', margin: 0, fontSize: 11 }}>☁️ 云端</Tag>
          )}
          <Tag color={
            backend === 'comfyui' ? 'green' : backend === 'herdsman' ? 'orange' : backend === 'ollama' ? 'purple' : 'blue'
          } style={{ borderRadius: 'var(--radius-md)', margin: 0, fontSize: 11 }}>
            {BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend}
          </Tag>
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenNovelDir} title="小说图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 4px' }} />
          <Button type="text" size="small" icon={<PictureOutlined />}
            onClick={handleOpenDir} title="生成图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 4px' }} />
        </Space>
      </div>

      {/* 主工作区：左栏控制面板 + 右栏结果 */}
      <div style={{ flex: 1, display: 'flex', gap: 16 }}>
        {/* 左栏 — 320px 控制面板 */}
        <div style={{ width: 320, flexShrink: 0, paddingRight: 4, overflowY: 'auto' }}>
          <div style={{
            background: 'rgba(255,255,255,0.03)',
            borderRadius: 'var(--radius-lg)',
            border: '1px solid var(--border-subtle)',
            padding: '14px 16px',
            display: 'flex', flexDirection: 'column', gap: 14,
          }}>
            <PromptPanel
              prompt={prompt} negative={negative}
              onPromptChange={setPrompt} onNegativeChange={setNegative}
              onTemplateSelect={applyTemplate}
              onOpenTemplatePicker={() => setTemplatePickerOpen(true)}
            />

            <div style={{ height: 1, background: 'var(--border-subtle)' }} />

            <GenControls
              size={size} model={model} seed={seed} count={count}
              modelOptions={modelOptions}
              customWidth={customWidth} customHeight={customHeight}
              onSizeChange={setSize} onModelChange={setModel}
              onSeedChange={setSeed} onCountChange={setCount}
              onCustomWidthChange={setCustomWidth}
              onCustomHeightChange={setCustomHeight}
              backendSelector={backendSelector}
              selectedLoras={selectedLoras}
              loraOptions={backend === 'comfyui' ? loraOptions : []}
              onLorasChange={setSelectedLoras}
            />
            {isLocalEngine && (() => {
              const isStarting = engineStarting
              const isRunning = engineRunning && !isStarting
              const bg = isRunning ? 'rgba(74,222,128,0.06)' : isStarting ? 'rgba(250,204,21,0.06)' : 'rgba(255,255,255,0.02)'
              const border = isRunning ? 'rgba(74,222,128,0.25)' : isStarting ? 'rgba(250,204,21,0.35)' : 'var(--border-subtle)'
              const textColor = isRunning ? '#4ade80' : isStarting ? '#facc15' : C('color-text-secondary')
              const label = isRunning ? `🟢 ${BACKEND_OPTIONS.find(b => b.value === backend)?.label || backend} 运行中`
                : isStarting ? `🟡 启动中... 等待 ${backend === 'comfyui' ? 'ComfyUI' : '引擎'} 就绪`
                : `⚫ 引擎未启动`
              return (
              <div style={{
                background: bg, borderRadius: 'var(--radius-sm)', border: `1px solid ${border}`,
                padding: '8px 10px', display: 'flex', alignItems: 'center', gap: 8,
              }}>
                <span style={{ fontSize: 11, color: textColor, flex: 1 }}>{label}</span>
                {isRunning ? (
                  <Button size="small" danger icon={<PoweroffOutlined />} onClick={handleStopEngine}
                    style={{ borderRadius: 'var(--radius-sm)', fontSize: 11, flexShrink: 0 }}>停止</Button>
                ) : (
                  <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                    loading={isStarting} onClick={handleStartEngine}
                    style={{ borderRadius: 'var(--radius-sm)', fontSize: 11, flexShrink: 0 }}>启动</Button>
                )}
              </div>
              )
            })()}
            <GenButton
              generating={generating} count={count}
              lastTime={lastTime} elapsed={elapsed}
              backend={backend} model={model}
              onGenerate={handleGenerate}
            />

            {/* 系统状态（仅 ComfyUI） */}
            {sysStats && (
              <>
                <div style={{ height: 1, background: 'var(--border-subtle)' }} />
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
                    📊 系统状态
                  </Typography.Text>
                  {/* CPU */}
                  {/* CPU */}
                  <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: 'var(--radius-sm)', padding: '6px 10px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                      <span style={{ fontSize: 10, color: C('color-text-secondary') }}>🖥 CPU</span>
                      <span style={{ fontSize: 11, fontWeight: 600, color: sysStats.cpu < 60 ? '#4ade80' : sysStats.cpu < 85 ? '#facc15' : '#f87171' }}>
                        {sysStats.cpu}%
                      </span>
                    </div>
                    <div style={{ height: 4, background: 'rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{
                        width: `${Math.min(sysStats.cpu, 100)}%`, height: '100%',
                        background: sysStats.cpu < 60 ? '#4ade80' : sysStats.cpu < 85 ? '#facc15' : '#f87171',
                        borderRadius: 2, transition: 'width 0.6s ease',
                      }} />
                    </div>
                  </div>
                  {/* 内存 */}
                  {(() => {
                    const memPct = sysStats.memTotal > 0 ? (sysStats.memUsed / sysStats.memTotal) * 100 : 0
                    const memColor = memPct < 60 ? '#4ade80' : memPct < 85 ? '#facc15' : '#f87171'
                    return (
                    <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: 'var(--radius-sm)', padding: '6px 10px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                        <span style={{ fontSize: 10, color: C('color-text-secondary') }}>🧠 内存</span>
                        <span style={{ fontSize: 11, fontWeight: 600, color: memColor }}>
                          {memPct.toFixed(0)}% <span style={{ fontWeight: 400, fontSize: 9, color: C('color-text-secondary') }}>({sysStats.memUsed.toFixed(0)}/{sysStats.memTotal.toFixed(0)}GB)</span>
                        </span>
                      </div>
                      <div style={{ height: 4, background: 'rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden' }}>
                        <div style={{
                          width: `${Math.min(memPct, 100)}%`, height: '100%',
                          background: memColor, borderRadius: 2, transition: 'width 0.6s ease',
                        }} />
                      </div>
                    </div>
                    )
                  })()}
                  {/* GPU */}
                  {sysStats.gpuName && (() => {
                    const vramPct = sysStats.vramTotal > 0 ? (sysStats.vramUsed / sysStats.vramTotal) * 100 : 0
                    const vramColor = vramPct < 60 ? '#4ade80' : vramPct < 85 ? '#facc15' : '#f87171'
                    const shortName = sysStats.gpuName.length > 20 ? sysStats.gpuName.slice(0, 20) + '…' : sysStats.gpuName
                    return (
                    <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: 'var(--radius-sm)', padding: '6px 10px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                        <span style={{ fontSize: 10, color: C('color-text-secondary') }}>🎮 {shortName}</span>
                        <span style={{ fontSize: 11, fontWeight: 600, color: vramColor }}>
                          {vramPct.toFixed(0)}% <span style={{ fontWeight: 400, fontSize: 9, color: C('color-text-secondary') }}>({sysStats.vramUsed.toFixed(0)}/{sysStats.vramTotal.toFixed(0)}GB)</span>
                        </span>
                      </div>
                      <div style={{ height: 4, background: 'rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden' }}>
                        <div style={{
                          width: `${Math.min(vramPct, 100)}%`, height: '100%',
                          background: vramColor, borderRadius: 2, transition: 'width 0.6s ease',
                        }} />
                      </div>
                    </div>
                    )
                  })()}
                </div>
              </>
            )}
          </div>
        </div>

        {/* 右栏 — 画廊 + 历史 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
          {/* 结果画廊 */}
          <div style={{ flex: 1, overflow: 'auto' }}>
            <ResultGallery
              results={results} generating={generating}
              onPreview={(i) => setLightboxIndex(i)}
              onDownload={handleDownload}
              onReuse={handleReuse}
              onDelete={handleDelete}
            />
          </div>
          {/* 历史画廊 */}
          <div style={{ flexShrink: 0 }}>
            <HistoryStrip
              history={history}
              onSelect={(i) => setLightboxIndex(i)}
              onClear={() => setHistory([])}
            />
          </div>
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
