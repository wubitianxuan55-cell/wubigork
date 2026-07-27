import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Card, Switch, Button, Input, Space, Tag, message, Spin, Collapse } from 'antd'
import {
  CloudOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ReloadOutlined, ThunderboltOutlined,
  DesktopOutlined, RocketOutlined, PictureOutlined, SoundOutlined,
  CaretRightOutlined, SettingOutlined, LoginOutlined, LogoutOutlined, KeyOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import SettingField from '../components/SettingField'
import { C } from '../utils/theme'
import type { TTSConfig, TTSStatus } from '../types'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
  type EngineConfig, type ModelInfo, type EngineStatus,
} from '../api/engines'
import {
  getConfig, saveConfig,
  getImageBackendInfo, setImageBackend as setImageBackendAPI,
  getTTSConfig, getTTSStatus, saveTTSConfig,
  startTTSServer, stopTTSServer,
} from '../api/settings'

type Category = 'llm' | 'image' | 'tts' | 'engine'

interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
}

type ModelKind = 'llm' | 'tts' | 'stt' | 'image'

function classifyModel(id: string): ModelKind {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('vox') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle') || lid.includes('krea')) return 'image'
  return 'llm'
}

const engineIcons: Record<string, React.ReactNode> = {
  xai: <CloudOutlined />, ollama: <DesktopOutlined />, herdsman: <RocketOutlined />, deepseek: <KeyOutlined />,
}
const engineColors: Record<string, string> = {
  xai: '#60a5fa', ollama: '#f59e0b', herdsman: '#84cc16', deepseek: '#8b5cf6',
}
const engineLabels: Record<string, string> = {
  xai: 'xAI 云端', ollama: 'Ollama 本地', herdsman: 'Herdsman 本地', deepseek: 'DeepSeek 云端',
}

const ModelCenterPage: React.FC = () => {
  const { loggedIn, login, logout } = useAppStore()
  const [category, setCategory] = useState<Category>('llm')
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [activeEngine, setActiveEngineState] = useState('xai')
  const [activeModel, setActiveModel] = useState('')
  const [testingEngine, setTestingEngine] = useState<string | null>(null)
  const [editingURLs, setEditingURLs] = useState<Record<string, string>>({})
  const [savingEngine, setSavingEngine] = useState<string | null>(null)
  const [engineStatuses, setEngineStatuses] = useState<Record<string, EngineStatus>>({})
  const [deepseekKey, setDeepseekKeyState] = useState('')
  const [deepseekKeyMasked, setDeepseekKeyMasked] = useState('')

  const [imageBackend, setImageBackend] = useState('xai')
  const [comfyUIURL, setComfyUIURL] = useState('http://127.0.0.1:8188')
  const [imageSaveDir, setImageSaveDir] = useState('')
  const [imageModel, setImageModel] = useState('flux')
  const [comfyUIPath, setComfyUIPath] = useState('')
  const [comfyUIPythonPath, setComfyUIPythonPath] = useState('')
  const [imageBackendSaving, setImageBackendSaving] = useState(false)

  const [ttsConfig, setTTSConfig] = useState<TTSConfig>({ modelPath: '', serverPath: '', port: 8765, backend: 'cuda', speed: 1.0 })
  const [ttsStatus, setTTSStatus] = useState<TTSStatus>({ running: false, port: 0 })

  const loadAll = useCallback(async () => {
    try {
      const list = await getEngines(); setEngines(list)
      const urls: Record<string, string> = {}
      list.forEach(e => { urls[e.id] = e.base_url || '' })
      try { const ae = await getActiveEngine(); if (ae) setActiveEngineState(ae) } catch (_) {}
      try {
        const ks = await getDeepseekKeyStatus()
        if (ks) { setDeepseekKeyMasked(ks.maskedKey || ''); if (ks.configured) setDeepseekKeyState(ks.maskedKey) }
      } catch (_) {}
    } catch (_) {}
    finally { setLoading(false) }
  }, [])

  const loadImageBackend = useCallback(async () => {
    try {
      const cfg: any = await getImageBackendInfo()
      if (cfg?.backend) setImageBackend(cfg.backend)
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (_) {}
  }, [])

  const loadTTS = useCallback(async () => {
    try { const c = await getTTSConfig(); if (c) setTTSConfig(c) } catch (_) {}
    try { const s = await getTTSStatus(); if (s) setTTSStatus(s) } catch (_) {}
  }, [])

  useEffect(() => { loadAll(); loadImageBackend(); loadTTS() }, [])

  const handleStartModel = async (card: ModelCardData) => {
    if (classifyModel(card.modelId) !== 'llm') return
    try {
      if (activeEngine !== card.engineId) { await setActiveEngine(card.engineId); setActiveEngineState(card.engineId) }
      await setEngineDefaultModel(card.engineId, card.modelId)
      setEngines(prev => prev.map(e => e.id === card.engineId ? { ...e, default_model: card.modelId } : e))
      setActiveModel(card.modelId)
      message.success(`已启动 ${card.modelName}`)
    } catch (err: any) { message.error(`启动失败：${err.message || err}`) }
  }

  const isModelActive = (card: ModelCardData) => activeEngine === card.engineId && activeModel === card.modelId

  const handleTestConnection = async (id: string) => {
    setTestingEngine(id)
    try { await testEngineConnection(id); await loadAll() } catch (err: any) { message.error(err.message) }
    finally { setTestingEngine(null) }
  }

  const handleRefreshModels = async (id: string) => {
    setTestingEngine(id)
    try { await refreshEngineModels(id); await loadAll() } catch (err: any) { message.error(err.message) }
    finally { setTestingEngine(null) }
  }

  const handleSaveURL = async (engine: EngineConfig) => {
    setSavingEngine(engine.id)
    try { await saveEngine({ id: engine.id, base_url: editingURLs[engine.id] || '', enabled: engine.enabled } as any); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
    finally { setSavingEngine(null) }
  }

  const handleToggleEngine = async (engine: EngineConfig, enabled: boolean) => {
    try { await saveEngine({ id: engine.id, base_url: engine.base_url || '', enabled } as any); await loadAll() }
    catch (err: any) { message.error(err.message) }
  }

  const handleSaveImageBackend = async () => {
    setImageBackendSaving(true)
    try { await setImageBackendAPI(imageBackend, comfyUIURL, imageModel); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
    finally { setImageBackendSaving(false) }
  }

  const handleSaveDeepseekKey = async () => {
    if (!deepseekKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setDeepseekKey(deepseekKey.trim())
      message.success('DeepSeek Key 已保存')
      const ks = await getDeepseekKeyStatus()
      if (ks) setDeepseekKeyMasked(ks.maskedKey || '')
    } catch (err: any) { message.error(err.message) }
  }

  const handleSaveTTS = async () => {
    try { await saveTTSConfig(ttsConfig.modelPath, ttsConfig.serverPath, ttsConfig.port, ttsConfig.backend, ttsConfig.speed); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
  }
  const handleStartTTS = async () => {
    try { await startTTSServer(ttsConfig.modelPath, ttsConfig.port, ttsConfig.backend); setTTSStatus(s => ({ ...s, running: true })); message.success('已启动') }
    catch (err: any) { message.error(err.message) }
  }
  const handleStopTTS = async () => {
    try { await stopTTSServer(); setTTSStatus(s => ({ ...s, running: false })); message.success('已停止') }
    catch (err: any) { message.error(err.message) }
  }

  const makeModels = (engine: EngineConfig): ModelCardData[] =>
    (engine.models || []).map(m => ({ modelId: m.id, modelName: m.id, engineId: engine.id, engineName: engine.name, engineType: engine.type, engineEnabled: engine.enabled, status: m.status || 'running' }))

  const allModels = engines.filter(e => e.enabled).flatMap(e => makeModels(e))
  const llmModels = allModels.filter(m => classifyModel(m.modelId) === 'llm')
  const ttsModels = allModels.filter(m => classifyModel(m.modelId) === 'tts')
  const sttModels = allModels.filter(m => classifyModel(m.modelId) === 'stt')
  const imageModels = allModels.filter(m => classifyModel(m.modelId) === 'image')

  const sidebarBtn = (key: Category, icon: React.ReactNode, label: string) => (
    <Button type={category === key ? 'primary' : 'text'} icon={icon as any}
      onClick={() => setCategory(key)}
      style={{ justifyContent: 'flex-start', textAlign: 'left', borderRadius: 8, color: category === key ? '#fff' : C('color-text-secondary'), background: category === key ? C('color-primary') : 'transparent', fontWeight: category === key ? 500 : 400, padding: '8px 14px', height: 38 }}>
      {label}
    </Button>
  )

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}><Spin size="large" /></div>

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Typography.Title level={4} style={{ color: C('color-text'), marginBottom: 16 }}>
        <ThunderboltOutlined style={{ marginRight: 8 }} />模型引擎中心
      </Typography.Title>
      <div style={{ flex: 1, display: 'flex', gap: 20, minHeight: 0 }}>
        <div style={{ width: 140, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 4 }}>
          {sidebarBtn('llm', <ThunderboltOutlined />, '语言模型')}
          {sidebarBtn('image', <PictureOutlined />, '图片生成')}
          {sidebarBtn('tts', <SoundOutlined />, '语音模型')}
          {sidebarBtn('engine', <SettingOutlined />, '引擎管理')}
        </div>

        <div style={{ flex: 1, overflow: 'auto', minWidth: 0 }}>
          {/* XAI 账户卡片 */}
          {category !== 'engine' && (
            <Card style={{ marginBottom: 20, background: loggedIn ? 'linear-gradient(135deg, rgba(52,211,153,0.06), rgba(16,185,129,0.03))' : 'linear-gradient(135deg, rgba(99,102,241,0.08), rgba(37,99,235,0.04))', border: loggedIn ? '1px solid rgba(52,211,153,0.25)' : '1px solid rgba(99,102,241,0.2)', borderRadius: 12 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Space size={12}>
                  <div style={{ width: 36, height: 36, borderRadius: 10, background: loggedIn ? 'linear-gradient(135deg, #34d399, #10b981)' : 'linear-gradient(135deg, #6366f1, #2563eb)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    {loggedIn ? <CheckCircleOutlined style={{ fontSize: 18, color: '#fff' }} /> : <CloudOutlined style={{ fontSize: 18, color: '#fff' }} />}
                  </div>
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{loggedIn ? 'xAI 已连接' : 'xAI 账户'}</Typography.Text><br />
                    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>{loggedIn ? 'Grok 模型已就绪' : '登录以使用云端模型'}</Typography.Text>
                  </div>
                </Space>
                {loggedIn ? <Button icon={<LogoutOutlined />} onClick={() => logout()} style={{ color: C('color-text-secondary'), fontSize: 12 }}>退出登录</Button>
                  : <Button type="primary" icon={<LoginOutlined />} onClick={() => login()} style={{ background: 'linear-gradient(135deg, #6366f1, #2563eb)', border: 'none', borderRadius: 8, fontWeight: 500 }}>登录 xAI</Button>}
              </div>
            </Card>
          )}

          {/* LLM */}
          {category === 'llm' && (
            <>
              {llmModels.length === 0 && (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>未发现语言模型。请在「引擎管理」中启用引擎并刷新模型。</Typography.Text>
                </Card>
              )}
              {engines.filter(e => e.enabled).map(engine => {
                const engineModels = llmModels.filter(m => m.engineId === engine.id)
                if (engineModels.length === 0) return null
                const color = engineColors[engine.id] || '#888'
                return (
                  <div key={engine.id} style={{ marginBottom: 24 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, paddingBottom: 8, borderBottom: `1px solid ${color}30` }}>
                      <Space size={8}>
                        <span style={{ fontSize: 18, color }}>{engineIcons[engine.id]}</span>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>{engine.name}</Typography.Text>
                        <Tag color={color} style={{ fontSize: 10 }}>{engineLabels[engine.id]}</Tag>
                        <Tag style={{ fontSize: 10 }}>{engineModels.length} 个</Tag>
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>刷新模型</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                      {engineModels.map(card => {
                        const active = isModelActive(card)
                        return (
                          <Card key={card.modelId} size="small" style={{ background: active ? `linear-gradient(135deg, ${color}18, ${color}08)` : 'var(--bg-glass)', border: active ? `2px solid ${color}` : '1px solid var(--border-subtle)', borderRadius: 10 }}>
                            <Typography.Text strong style={{ color: active ? color : C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{card.modelName}</Typography.Text>
                            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                              <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabels[card.engineId]}</Tag>
                              <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 8, fontSize: 11 }}>{active ? '已启动' : '启动'}</Button>
                            </div>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </>
          )}

          {/* Image */}
          {category === 'image' && (
            <>
              {imageModels.length > 0 && (
                <div style={{ marginBottom: 24 }}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>发现的图片模型</Typography.Text>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                    {imageModels.map(m => (
                      <Card key={m.modelId} size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 10 }}>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{m.modelName}</Typography.Text>
                        <Space>
                          <Tag color={engineColors[m.engineId]} style={{ fontSize: 10, margin: 0 }}>{engineLabels[m.engineId]}</Tag>
                          <Tag color="orange" style={{ fontSize: 10, margin: 0 }}>图片</Tag>
                          <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                        </Space>
                      </Card>
                    ))}
                  </div>
                </div>
              )}
              <Collapse ghost size="small" items={[{
                key: 'img-cfg', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />图片生成配置</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="后端引擎" value={imageBackend} type="select" onChange={(v: string) => setImageBackend(v)} options={[{ label: '☁️ xAI 云端', value: 'xai' }, { label: '🏠 ComfyUI 本地', value: 'comfyui' }]} width={200} />
                      {imageBackend === 'comfyui' && (
                        <>
                          <SettingField label="ComfyUI 服务地址" value={comfyUIURL} onChange={v => setComfyUIURL(v)} />
        <SettingField label="生成模型" value={imageModel} type="select" onChange={(v: string) => setImageModel(v)} options={[{ label: '🌊 Flux Dev', value: 'flux' }, { label: '⚡ Z-Image-Turbo', value: 'z-image-turbo' }, { label: '🎨 Krea2 (FLUX)', value: 'krea2' }]} width={230} />
                        </>
                      )}
                      <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>💾 保存配置</Button>
                    </Space>
                  </Card>
                ),
              }]} />
            </>
          )}

          {/* TTS / Voice */}
          {category === 'tts' && (
            <>
              {ttsModels.length === 0 && sttModels.length === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <SoundOutlined style={{ fontSize: 32, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, display: 'block' }}>未发现语音模型</Typography.Text>
                </Card>
              ) : (
                <>
                  {ttsModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🔊 TTS 文字转语音</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                        {ttsModels.map(m => (
                          <Card key={m.modelId} size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 10 }}>
                            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                            <Space>
                              <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                              <Tag color="purple" style={{ fontSize: 10 }}>TTS</Tag>
                              <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                            </Space>
                          </Card>
                        ))}
                      </div>
                    </div>
                  )}
                  {sttModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🎙️ STT 语音识别</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                        {sttModels.map(m => (
                          <Card key={m.modelId} size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 10 }}>
                            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                            <Space>
                              <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                              <Tag color="blue" style={{ fontSize: 10 }}>STT</Tag>
                              <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                            </Space>
                          </Card>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}
              <Collapse ghost size="small" items={[{
                key: 'voxcpm', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />VoxCPM 本地 TTS 配置</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="VoxCPM 可执行文件路径" value={ttsConfig.serverPath} placeholder="留空使用 PATH" onChange={v => setTTSConfig(p => ({ ...p, serverPath: v }))} />
                      <SettingField label="GGUF 模型文件路径" value={ttsConfig.modelPath} placeholder="D:\\models\\voxcpm.gguf" onChange={v => setTTSConfig(p => ({ ...p, modelPath: v }))} />
                      <Space direction="horizontal">
                        <SettingField label="端口" value={ttsConfig.port} type="number" onChange={v => setTTSConfig(p => ({ ...p, port: v }))} width={100} />
                        <SettingField label="后端" value={ttsConfig.backend} type="select" onChange={v => setTTSConfig(p => ({ ...p, backend: v }))} options={[{ label: 'CUDA', value: 'cuda' }, { label: 'CPU', value: 'cpu' }]} width={100} />
                      </Space>
                      <SettingField label="语速" value={ttsConfig.speed} type="number" onChange={v => setTTSConfig(p => ({ ...p, speed: v }))} width={80} />
                      <Space>
                        <Button type="primary" onClick={handleSaveTTS}>💾 保存</Button>
                        <Button onClick={handleStartTTS} disabled={ttsStatus.running}>▶ 启动</Button>
                        <Button onClick={handleStopTTS} disabled={!ttsStatus.running} danger>⏹ 停止</Button>
                      </Space>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>状态：{ttsStatus.running ? `🟢 运行中 (${ttsStatus.port})` : '⚫ 未启动'}</Typography.Text>
                    </Space>
                  </Card>
                ),
              }]} />
            </>
          )}

          {/* Engine Management */}
          {category === 'engine' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {engines.map(engine => {
                const color = engineColors[engine.id] || '#888'
                const em = makeModels(engine)
                const mc = { llm: em.filter(m => classifyModel(m.modelId) === 'llm').length, tts: em.filter(m => classifyModel(m.modelId) === 'tts').length, stt: em.filter(m => classifyModel(m.modelId) === 'stt').length, image: em.filter(m => classifyModel(m.modelId) === 'image').length }
                return (
                  <Card key={engine.id} size="small" style={{ background: 'var(--bg-glass)', border: engine.enabled ? `1px solid ${color}30` : '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
                      <Space size={8}>
                        <span style={{ fontSize: 20, color }}>{engineIcons[engine.id]}</span>
                        <div>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{engine.name}</Typography.Text>
                          <div style={{ marginTop: 2 }}>
                            <Tag color={color} style={{ fontSize: 10 }}>{engineLabels[engine.id]}</Tag>
                            <Switch size="small" checked={engine.enabled} onChange={(v) => handleToggleEngine(engine, v)} />
                          </div>
                        </div>
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>刷新</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                      {mc.llm > 0 && <Tag style={{ fontSize: 10 }}>💬 {mc.llm} 语言</Tag>}
                      {mc.tts > 0 && <Tag color="purple" style={{ fontSize: 10 }}>🔊 {mc.tts} TTS</Tag>}
                      {mc.stt > 0 && <Tag color="blue" style={{ fontSize: 10 }}>🎙️ {mc.stt} STT</Tag>}
                      {mc.image > 0 && <Tag color="orange" style={{ fontSize: 10 }}>🖼️ {mc.image} 图片</Tag>}
                      {em.length === 0 && <Tag style={{ fontSize: 10 }}>暂无模型</Tag>}
                    </div>
                    {engine.type !== 'xai' && engine.type !== 'deepseek' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={editingURLs[engine.id] || ''} onChange={e => setEditingURLs(prev => ({ ...prev, [engine.id]: e.target.value }))} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={() => handleSaveURL(engine)} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'deepseek' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={deepseekKey} onChange={e => setDeepseekKeyState(e.target.value)} placeholder={deepseekKeyMasked || 'sk-...'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveDeepseekKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engineStatuses[engine.id] && (
                      <div style={{ marginTop: 6, fontSize: 11 }}>
                        {engineStatuses[engine.id].connected ? <span style={{ color: '#34d399' }}>✓ 已连接</span> : <span style={{ color: '#fb7185' }}>✗ {engineStatuses[engine.id].error}</span>}
                      </div>
                    )}
                  </Card>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default ModelCenterPage
