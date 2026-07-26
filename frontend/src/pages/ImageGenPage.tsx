import React, { useState, useCallback, useEffect, useRef } from 'react'
import { Typography, Tag, Input, Select, InputNumber, Button, Space, Drawer, message, Popconfirm } from 'antd'
import {
  PictureOutlined, CloudOutlined, HomeOutlined,
  DeleteOutlined, ExpandOutlined, DownloadOutlined, SyncOutlined,
  ShakeOutlined, PlusOutlined, EditOutlined,
  PlayCircleOutlined, PoweroffOutlined, FolderOpenOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import Lightbox from '../components/Lightbox'
import ResultGallery from '../components/ResultGallery'
import PromptBar from '../components/imagegen/PromptBar'
import CustomTemplateModal from '../components/imagegen/CustomTemplateModal'
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
import type { GenResult } from '../components/ResultGallery'

const { TextArea } = Input

const SIZE_OPTIONS = [
  { label: '🟦 方形 1:1 (1024)', value: '1024x1024' },
  { label: '🖼 风景 4:3', value: '1024x768' },
  { label: '🎬 宽屏 16:9', value: '1024x576' },
  { label: '📱 竖屏 9:16', value: '576x1024' },
  { label: '📐 肖像 3:4', value: '768x1024' },
  { label: '🖥 超宽 21:9', value: '1280x544' },
  { label: '✏️ 自定义', value: 'custom' },
]

const ImageGenPage: React.FC = () => {

  const [prompt, setPrompt] = useState('')
  const [negative, setNegative] = useState('')
  const [size, setSize] = useState('1024x1024')
  const [model, setModel] = useState('flux')
  const [seed, setSeed] = useState(0)
  const [count, setCount] = useState(1)
  const [customWidth, setCustomWidth] = useState(1024)
  const [customHeight, setCustomHeight] = useState(1024)

  // 模板
  const [templateCat, setTemplateCat] = useState<string | undefined>()
  const [customTemplates, setCustomTemplates] = useState<CustomTemplate[]>(() => loadCustomTemplates())
  const [customModalOpen, setCustomModalOpen] = useState(false)
  const [editingCustom, setEditingCustom] = useState<CustomTemplate | null>(null)
  const [customLabel, setCustomLabel] = useState('')
  const [customPrompt, setCustomPrompt] = useState('')
  const [customNegative, setCustomNegative] = useState('')

  const [generating, setGenerating] = useState(false)
  const [backend, setBackend] = useState('xai')
  const [comfyUIRunning, setComfyUIRunning] = useState(false)
  const [comfyUIStarting, setComfyUIStarting] = useState(false)
  const [sysStats, setSysStats] = useState<SystemStats | null>(null)
  const [showModel, setShowModel] = useState(false)
  const [elapsed, setElapsed] = useState(0)

  const [results, setResults] = useState<GenResult[]>([])
  const [history, setHistory] = useState<GenResult[]>([])
  const [lightboxIndex, setLightboxIndex] = useState(-1)
  const [showResult, setShowResult] = useState(false)
  const [characters, setCharacters] = useState<{ id: string; name: string }[]>([])

  const generatingRef = useRef(false)

  useEffect(() => {
    (async () => {
      try {
        const info = await getImageBackendInfo()
        if (info?.backend) {
          setBackend(info.backend)
          setShowModel(info.backend === 'comfyui')
        }
        if (info?.model) setModel(info.model)
        const chars = await getCharacters()
        setCharacters(chars)
      } catch (_) {}
    })()
  }, [])

  // ComfyUI 状态轮询
  useEffect(() => {
    if (backend !== 'comfyui') { setComfyUIRunning(false); return }
    const check = async () => {
      try {
        const s = await getComfyUIStatus()
        setComfyUIRunning(!!s?.running)
      } catch (_) { setComfyUIRunning(false) }
    }
    check()
    const timer = setInterval(check, 5000)
    return () => clearInterval(timer)
  }, [backend])

  // 系统状态轮询
  useEffect(() => {
    if (backend !== 'comfyui') { setSysStats(null); return }
    const fetchStats = async () => {
      try {
        const s = await getSystemStats()
        if (s) setSysStats(s)
      } catch (_) {}
    }
    fetchStats()
    const timer = setInterval(fetchStats, 3000)
    return () => clearInterval(timer)
  }, [backend])

  useEffect(() => {
    if (!generating) { setElapsed(0); return }
    const start = Date.now()
    const timer = setInterval(() => setElapsed(Math.round((Date.now() - start) / 1000)), 1000)
    return () => clearInterval(timer)
  }, [generating])

  const handleGenerate = useCallback(async () => {
    if (!prompt.trim()) { message.warning('请输入图片描述'); return }
    if (generatingRef.current) return
    generatingRef.current = true
    setGenerating(true)
    setResults([])
    setLightboxIndex(-1)
    try {
      const finalSize = size === 'custom' ? `${customWidth}x${customHeight}` : size
      const res = await generateImage(prompt, negative, finalSize, model, seed, count)
      if (res?.error) {
        message.error(res.error)
      } else if (res?.images?.length) {
        const genResults = res.images
        setResults(genResults)
        setHistory((prev) => [...genResults, ...prev.filter((h) =>
          !genResults.some((g) => g.seed === h.seed && g.prompt === h.prompt),
        )])
        setLightboxIndex(0)
        setShowResult(true)
        message.success(`✨ 已生成 ${genResults.length} 张图片`)
      }
    } catch (err: any) {
      message.error(err?.message || '生成失败')
    } finally {
      generatingRef.current = false
      setGenerating(false)
    }
  }, [prompt, negative, size, model, seed, count])

  const handleStartComfy = useCallback(async () => {
    setComfyUIStarting(true)
    try {
      await startComfyUI()
      message.success('ComfyUI 启动中，请稍候...')
    } catch (err: any) { message.error(err?.message || '启动失败') }
    finally { setComfyUIStarting(false) }
  }, [])

  const handleStopComfy = useCallback(async () => {
    try {
      await stopComfyUI()
      message.success('ComfyUI 已停止')
      setComfyUIRunning(false)
    } catch (err: any) { message.error(err?.message || '停止失败') }
  }, [])

  const handleOpenDir = useCallback(async () => {
    try { await openImageSaveDir() } catch (err: any) { message.error(err?.message || '打开失败') }
  }, [])

  const handleOpenNovelDir = useCallback(async () => {
    try { await openNovelImagesDir() } catch (err: any) { message.error(err?.message || '打开失败') }
  }, [])

  const handleReuse = useCallback((index: number) => {
    const r = history[index]
    if (!r) return
    setPrompt(r.prompt)
    if (r.negative) setNegative(r.negative)
    setSize(r.size)
    setModel(r.model)
    setSeed(r.seed)
    setLightboxIndex(-1)
  }, [history])

  const handleDelete = useCallback((index: number) => {
    const r = history[index]
    if (!r) return
    setHistory((prev) => prev.filter((_, i) => i !== index))
    setResults((prev) => prev.filter((img) => img.seed !== r.seed || img.prompt !== r.prompt))
    if (lightboxIndex === index) setLightboxIndex(-1)
    else if (lightboxIndex > index) setLightboxIndex((li) => li - 1)
  }, [history, lightboxIndex])

  const handleDownload = useCallback((index: number) => {
    const r = results[index] || history[index]
    if (!r?.image) return
    const a = document.createElement('a')
    a.href = r.image
    a.download = `wubigork-${Date.now()}-seed${r.seed}.png`
    a.click()
  }, [results, history])

  const handleSetPortrait = useCallback(async (index: number, charID: string) => {
    const r = history[index]
    if (!r?.image) return
    try {
      await setPortrait(charID, r.image)
      message.success('已设为角色剧照')
    } catch (err: any) { message.error(err?.message || '设置失败') }
  }, [history])

  // ── 自定义模板操作 ──
  const openCustomAdd = () => {
    setEditingCustom(null)
    setCustomLabel('')
    setCustomPrompt('')
    setCustomNegative('')
    setCustomModalOpen(true)
  }
  const openCustomEdit = (t: CustomTemplate) => {
    setEditingCustom(t)
    setCustomLabel(t.label)
    setCustomPrompt(t.prompt)
    setCustomNegative(t.negative)
    setCustomModalOpen(true)
  }
  const saveCustom = () => {
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
  }
  const deleteCustom = (id: string) => {
    const updated = customTemplates.filter((t) => t.id !== id)
    setCustomTemplates(updated)
    saveCustomTemplates(updated)
    message.success('已删除')
  }

  const applyTemplate = (t: Template) => {
    setPrompt((p) => p ? p + '，' + t.prompt : t.prompt)
    if (t.negative) setNegative((n) => n ? n + ', ' + t.negative : t.negative)
  }

  const currentTemplates: Template[] = templateCat
    ? (templateCat === '⭐ 自定义' ? customTemplates : TEMPLATES[templateCat] || [])
    : []

  const s = { color: C('color-text-secondary'), fontSize: 10, display: 'block', marginBottom: 8 } as const

  // ── 左侧参数面板 ──
  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* 模板 */}
      <div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
          <Typography.Text style={s}>📐 快速模板</Typography.Text>
          <Button type="text" size="small" icon={<PlusOutlined />} onClick={openCustomAdd}
            style={{ color: C('color-primary'), padding: '0 4px', fontSize: 10 }}>自定义</Button>
        </div>
        <Select value={templateCat} onChange={setTemplateCat}
          placeholder="选择模板类别…" size="small"
          style={{ width: '100%', marginBottom: 6 }}
          options={getAllCategories(customTemplates.length).map((c) => ({ label: c, value: c }))}
          allowClear
        />
        {currentTemplates.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {currentTemplates.map((t, i) => {
              const isCustom = templateCat === '⭐ 自定义'
              return (
                <Tag key={isCustom ? (t as CustomTemplate).id : i}
                  style={{ cursor: 'pointer', borderRadius: 'var(--radius-sm)', fontSize: 10, margin: 0 }}
                  onClick={() => applyTemplate(t)}
                  closable={isCustom}
                  onClose={(e) => {
                    e.preventDefault()
                    if (isCustom) {
                      if (templateCat === '⭐ 自定义' && currentTemplates.length <= 1) setTemplateCat(undefined)
                      deleteCustom((t as CustomTemplate).id)
                    }
                  }}
                >
                  <span onClick={() => isCustom && openCustomEdit(t as CustomTemplate)}
                    style={{ cursor: isCustom ? 'pointer' : undefined }}>
                    {isCustom && <EditOutlined style={{ fontSize: 9, marginRight: 2 }} />}
                    {t.label}
                  </span>
                </Tag>
              )
            })}
          </div>
        )}
      </div>

      {/* 种子 */}
      <div>
        <Typography.Text style={s}>🎲 种子</Typography.Text>
        <InputNumber value={seed || undefined} size="small"
          onChange={(v) => setSeed(v || 0)} placeholder="随机"
          min={1} max={2147483647} style={{ width: '100%' }}
          addonAfter={
            <Button type="text" size="small" icon={<ShakeOutlined />} onClick={() => setSeed(0)}
              style={{ padding: 0, height: 18 }} />
          }
        />
      </div>

      {/* 负向 prompt */}
      <div>
        <Typography.Text style={{ ...s, marginBottom: 4 }}>🚫 不想出现</Typography.Text>
        <TextArea placeholder="模糊, 低质量, 畸形手指..."
          value={negative} onChange={(e) => setNegative(e.target.value)}
          rows={2} autoSize={{ minRows: 1, maxRows: 3 }}
          style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', resize: 'none', fontSize: 12 }}
        />
      </div>

      {/* 图片参数 */}
      <div>
        <Typography.Text style={{ ...s, marginBottom: 4 }}>📐 图片参数</Typography.Text>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Select value={size} onChange={setSize} size="small" style={{ width: '100%' }}
            options={SIZE_OPTIONS} />
          {size === 'custom' && (
            <div style={{ display: 'flex', gap: 8 }}>
              <InputNumber value={customWidth} onChange={(v) => setCustomWidth(v || 1024)}
                size="small" min={256} max={2048} step={64}
                addonBefore="宽" style={{ flex: 1 }} />
              <InputNumber value={customHeight} onChange={(v) => setCustomHeight(v || 1024)}
                size="small" min={256} max={2048} step={64}
                addonBefore="高" style={{ flex: 1 }} />
            </div>
          )}
          {showModel && (
            <Select value={model} onChange={setModel} size="small" style={{ width: '100%' }}
              options={[
                { label: '🌊 Flux Dev', value: 'flux' },
                { label: '⚡ Z-Image-Turbo', value: 'z-image-turbo' },
              ]} />
          )}
          <div>
            <Typography.Text style={{ ...s, marginBottom: 4 }}>生成数量</Typography.Text>
            <Select value={count} onChange={setCount} size="small" style={{ width: '100%' }}
              options={[{ label: '1', value: 1 }, { label: '2', value: 2 }, { label: '3', value: 3 }, { label: '4', value: 4 }]} />
          </div>
        </Space>
      </div>

      {/* 系统状态 */}
      {sysStats && (
        <div style={{ marginTop: 'auto', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <Typography.Text style={{ ...s, marginBottom: 2 }}>📊 系统状态</Typography.Text>
          {/* CPU 卡片 */}
          <div style={{
            background: 'rgba(255,255,255,0.04)', borderRadius: 'var(--radius-md)',
            padding: '8px 10px',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
              <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary') }}>
                🖥 CPU
              </Typography.Text>
              <Typography.Text style={{
                fontSize: 11, fontWeight: 600,
                color: sysStats.cpu < 60 ? '#4ade80' : sysStats.cpu < 85 ? '#facc15' : '#f87171',
              }}>
                {sysStats.cpu}%
              </Typography.Text>
            </div>
            <div style={{ height: 6, background: 'rgba(255,255,255,0.08)', borderRadius: 3, overflow: 'hidden' }}>
              <div style={{
                width: `${Math.min(sysStats.cpu, 100)}%`, height: '100%',
                background: sysStats.cpu < 60 ? '#4ade80' : sysStats.cpu < 85 ? '#facc15' : '#f87171',
                borderRadius: 3, transition: 'width 0.6s ease',
              }} />
            </div>
          </div>
          {/* GPU 卡片 */}
          {sysStats.gpuName && (
            <div style={{
              background: 'rgba(255,255,255,0.04)', borderRadius: 'var(--radius-md)',
              padding: '8px 10px',
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary') }}>
                  🎮 {sysStats.gpuName.length > 12 ? sysStats.gpuName.slice(0, 12) + '…' : sysStats.gpuName}
                </Typography.Text>
                <Typography.Text style={{
                  fontSize: 11, fontWeight: 600,
                  color: '#60a5fa',
                }}>
                  {sysStats.vramUsed.toFixed(1)} / {sysStats.vramTotal.toFixed(1)} GB
                </Typography.Text>
              </div>
              <div style={{ height: 6, background: 'rgba(255,255,255,0.08)', borderRadius: 3, overflow: 'hidden' }}>
                {(() => {
                  const pct = sysStats.vramTotal > 0 ? (sysStats.vramUsed / sysStats.vramTotal) * 100 : 0
                  return (
                    <div style={{
                      width: `${Math.min(pct, 100)}%`, height: '100%',
                      background: pct < 60 ? '#4ade80' : pct < 85 ? '#facc15' : '#f87171',
                      borderRadius: 3, transition: 'width 0.6s ease',
                    }} />
                  )
                })()}
              </div>
            </div>
          )}
        </div>
      )}

      <CustomTemplateModal
        open={customModalOpen}
        editing={!!editingCustom}
        label={customLabel} onLabelChange={setCustomLabel}
        prompt={customPrompt} onPromptChange={setCustomPrompt}
        negative={customNegative} onNegativeChange={setCustomNegative}
        onSave={saveCustom}
        onCancel={() => setCustomModalOpen(false)}
      />
    </div>
  )

  // ── 右侧历史面板 ──
  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0, marginBottom: 8 }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
          📜 历史 ({history.length})
        </Typography.Text>

        <Space size={2}>
          <Button type="text" size="small" icon={<FolderOpenOutlined />}
            onClick={handleOpenNovelDir} title="小说图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }} />
          <Button type="text" size="small" icon={<PictureOutlined />}
            onClick={handleOpenDir} title="生成图片目录"
            style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }} />
          <Button type="text" size="small" icon={<DeleteOutlined />} onClick={() => setHistory([])}
            style={{ color: C('color-text-secondary'), fontSize: 10, padding: '0 4px' }}>清空</Button>
        </Space>
      </div>
      <div style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
        {history.map((h, i) => (
          <div key={i} onClick={() => setLightboxIndex(i)}
            style={{
              borderRadius: 'var(--radius-sm)', overflow: 'hidden',
              border: lightboxIndex === i ? '2px solid var(--color-primary)' : '1px solid var(--border-subtle)',
              cursor: 'pointer', flexShrink: 0, transition: 'border 0.15s',
            }}
          >
            <div style={{ position: 'relative' }}>
              <img src={h.image} alt="" style={{ width: '100%', display: 'block' }} />
              <Button type="text" size="small" danger icon={<DeleteOutlined />}
                onClick={(e) => { e.stopPropagation(); handleDelete(i) }}
                style={{
                  position: 'absolute', top: 2, right: 2,
                  color: '#fff', fontSize: 10, padding: '0 2px', height: 18,
                  background: 'rgba(0,0,0,0.5)', borderRadius: 4,
                  opacity: 0, transition: 'opacity 0.15s',
                }}
                onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.opacity = '1' }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.opacity = '0' }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  )

  return (
    <div style={{ height: 'calc(100vh - 120px)', display: 'flex', flexDirection: 'column' }}>
      {/* 顶栏 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12, flexShrink: 0 }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
          <PictureOutlined style={{ marginRight: 10 }} />AI 绘梦
        </Typography.Title>
        <Space size={8}>
          {backend === 'comfyui' && (
            <>
              <Tag color={comfyUIRunning ? 'green' : 'default'} style={{ borderRadius: 'var(--radius-md)', margin: 0 }}>
                {comfyUIRunning ? '🟢 已连接' : '⚫ 未连接'}
              </Tag>
              {comfyUIRunning ? (
                <Button size="small" danger icon={<PoweroffOutlined />} onClick={handleStopComfy}
                  style={{ borderRadius: 'var(--radius-md)', fontSize: 11 }}>停止</Button>
              ) : (
                <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                  loading={comfyUIStarting} onClick={handleStartComfy}
                  style={{ borderRadius: 'var(--radius-md)', fontSize: 11 }}>启动</Button>
              )}
            </>
          )}
          <Tag color={backend === 'comfyui' ? 'green' : 'blue'} style={{ borderRadius: 'var(--radius-md)', margin: 0 }}>
            {backend === 'comfyui'
              ? <><HomeOutlined /> 本地 {model === 'z-image-turbo' ? 'Z-Image-Turbo' : 'Flux'}</>
              : <><CloudOutlined /> xAI 云端</>}
          </Tag>
        </Space>
      </div>

      {/* 主工作区 */}
        <div style={{ flex: 1, display: 'flex', gap: 16, overflow: 'hidden' }}>
          {/* 左栏 — 参数 */}
          <div style={{ width: 200, flexShrink: 0, overflow: 'auto', paddingRight: 4 }}>
            {leftPanel}
          </div>

          {/* 中间 — 画廊 + 底部输入 */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, overflow: 'hidden' }}>
            <div style={{ flex: 1, overflow: 'auto' }}>
              <ResultGallery results={results} generating={generating}
                onPreview={(i) => setLightboxIndex(i)}
                onDownload={handleDownload}
                onReuse={handleReuse}
                onDelete={handleDelete}
              />
            </div>
            <PromptBar prompt={prompt} onPromptChange={setPrompt}
              generating={generating} elapsed={elapsed}
              onGenerate={handleGenerate}
            />
          </div>

          {/* 右栏 — 历史 */}
          {rightPanel && (
            <div style={{ width: 180, flexShrink: 0 }}>
              {rightPanel}
            </div>
          )}
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
    </div>
  )
}

export default ImageGenPage
