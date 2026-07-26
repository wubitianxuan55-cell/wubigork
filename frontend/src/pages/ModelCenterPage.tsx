import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Card, Switch, Button, Input, Space, Tag, message, Spin, Collapse } from 'antd'
import {
  CloudOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ReloadOutlined, ThunderboltOutlined,
  DesktopOutlined, RocketOutlined, PictureOutlined, SoundOutlined,
  CaretRightOutlined, SettingOutlined, LoginOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import { useAppStore } from '../stores/appStore'
import SettingField from '../components/SettingField'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine,
  type EngineConfig, type ModelInfo, type EngineStatus,
} from '../api/engines'
import {
  getConfig, saveConfig,
  getImageBackendInfo, setImageBackend as setImageBackendAPI,
  getTTSConfig, getTTSStatus, saveTTSConfig,
  startTTSServer, stopTTSServer,
} from '../api/settings'
import type { TTSConfig, TTSStatus } from '../types'

type Category = 'llm' | 'image' | 'tts'

interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
}

const engineIcons: Record<string, React.ReactNode> = {
  xai: <CloudOutlined />,
  ollama: <DesktopOutlined />,
  herdsman: <RocketOutlined />,
}

const engineColors: Record<string, string> = {
  xai: '#60a5fa', ollama: '#f59e0b', herdsman: '#84cc16',
}

const engineLabels: Record<string, string> = {
  xai: 'xAI 云端', ollama: 'Ollama 本地', herdsman: 'Herdsman 本地',
}

// ── 页面组件 ──────────────────────────────────────────────

const ModelCenterPage: React.FC = () => {
  const { loggedIn, login } = useAppStore()
  const [category, setCategory] = useState<Category>('llm')
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [activeEngine, setActiveEngineState] = useState('xai')
  const [activeModel, setActiveModel] = useState('')
  const [testingEngine, setTestingEngine] = useState<string | null>(null)
  const [editingURLs, setEditingURLs] = useState<Record<string, string>>({})
  const [savingEngine, setSavingEngine] = useState<string | null>(null)
  const [engineStatuses, setEngineStatuses] = useState<Record<string, EngineStatus>>({})

  const [imageBackend, setImageBackend] = useState('xai')
  const [comfyUIURL, setComfyUIURL] = useState('http://127.0.0.1:8188')
  const [imageSaveDir, setImageSaveDir] = useState('')
  const [imageModel, setImageModel] = useState('flux')
  const [comfyUIPath, setComfyUIPath] = useState('')
  const [comfyUIPythonPath, setComfyUIPythonPath] = useState('')
  const [imageBackendSaving, setImageBackendSaving] = useState(false)
  const [config, setConfig] = useState<Record<string, string>>({})

  const [ttsConfig, setTTSConfig] = useState<TTSConfig>({
    modelPath: '', serverPath: '', port: 8765, backend: 'cuda', speed: 1.0,
  })
  const [ttsStatus, setTTSStatus] = useState<TTSStatus>({ running: false, port: 0 })

  // ── 加载 ────────────────────────────────────────────────

  const loadAll = useCallback(async () => {
    try {
      const list = await getEngines()
      setEngines(list)
      const urls: Record<string, string> = {}
      list.forEach(e => { urls[e.id] = e.base_url })
      setEditingURLs(urls)
      const active = await getActiveEngine()
      setActiveEngineState(active || 'xai')
      const ae = list.find(e => e.id === (active || 'xai'))
      if (ae?.default_model) setActiveModel(ae.default_model)
    } catch (err) { console.error('[ModelCenter] loadAll:', err) }
    finally { setLoading(false) }
  }, [])

  const loadImageBackend = useCallback(async () => {
    try {
      const info = await getImageBackendInfo()
      if (info?.backend) setImageBackend(info.backend)
      if (info?.model) setImageModel(info.model)
      const cfg = await getConfig()
      setConfig(cfg || {})
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (err) { console.error('[ModelCenter] loadImageBackend:', err) }
  }, [])

  const loadTTS = useCallback(async () => {
    try {
      const cfg = await getTTSConfig()
      if (cfg) setTTSConfig(cfg)
      const s = await getTTSStatus()
      if (s) setTTSStatus(s)
    } catch (err) { console.error('[ModelCenter] loadTTS:', err) }
  }, [])

  useEffect(() => { loadAll(); loadImageBackend(); loadTTS() }, [loadAll, loadImageBackend, loadTTS])

  // ── 操作函数 ────────────────────────────────────────────

  const handleStartModel = async (card: ModelCardData) => {
    try {
      if (activeEngine !== card.engineId) {
        await setActiveEngine(card.engineId)
        setActiveEngineState(card.engineId)
      }
      await setEngineDefaultModel(card.engineId, card.modelId)
      setEngines(prev => prev.map(e => e.id === card.engineId ? { ...e, default_model: card.modelId } : e))
      setActiveModel(card.modelId)
      message.success(`✅ 已启动 ${card.modelName}`)
    } catch (err: any) { message.error(err?.message || '启动失败') }
  }

  const handleTestConnection = async (engineID: string) => {
    setTestingEngine(engineID)
    try {
      const status = await testEngineConnection(engineID)
      setEngineStatuses(prev => ({ ...prev, [engineID]: status }))
      if (status.connected) {
        message.success(`✅ ${engineID} 连接成功，${status.model_count} 个模型`)
        const list = await getEngines()
        setEngines(list)
      } else {
        message.warning(`⚠️ ${status.error}`)
      }
    } catch (err: any) { message.error(err?.message || '失败') }
    finally { setTestingEngine(null) }
  }

  const handleRefreshModels = async (engineID: string) => {
    setTestingEngine(engineID)
    try {
      const models = await refreshEngineModels(engineID)
      setEngines(prev => prev.map(e => e.id === engineID ? { ...e, models } : e))
      message.success(`已刷新，${models.length} 个模型`)
    } catch (err: any) { message.error(err?.message || '失败') }
    finally { setTestingEngine(null) }
  }

  const handleToggleEnabled = async (engine: EngineConfig, enabled: boolean) => {
    try {
      await saveEngine({ ...engine, enabled })
      setEngines(prev => prev.map(e => e.id === engine.id ? { ...e, enabled } : e))
    } catch (err: any) { message.error(err?.message || '失败') }
  }

  const handleSaveURL = async (engine: EngineConfig) => {
    setSavingEngine(engine.id)
    try {
      const u = editingURLs[engine.id] || engine.base_url
      await saveEngine({ ...engine, base_url: u })
      setEngines(prev => prev.map(e => e.id === engine.id ? { ...e, base_url: u } : e))
      message.success('BaseURL 已保存')
    } catch (err: any) { message.error(err?.message || '失败') }
    finally { setSavingEngine(null) }
  }

  const handleSaveImageBackend = async () => {
    setImageBackendSaving(true)
    try {
      await setImageBackendAPI(imageBackend, comfyUIURL.trim(), imageModel)
      message.success('图片后端已切换')
      const keys: [string, string][] = [
        ['image_backend', imageBackend], ['comfyui_url', comfyUIURL.trim()],
        ['image_model', imageModel], ['comfyui_path', comfyUIPath.trim()],
        ['comfyui_python_path', comfyUIPythonPath.trim()], ['image_save_dir', imageSaveDir],
      ]
      for (const [k, v] of keys) { await saveConfig(k, v) }
    } catch (err: any) { message.error(err?.message || '失败') }
    finally { setImageBackendSaving(false) }
  }

  const handleSaveTTS = async () => {
    try {
      await saveTTSConfig(ttsConfig.modelPath, ttsConfig.serverPath, ttsConfig.port, ttsConfig.backend, ttsConfig.speed)
      message.success('TTS 配置已保存')
    } catch (err: any) { message.error(err?.message || '失败') }
  }

  const handleStartTTS = async () => {
    try {
      await startTTSServer(ttsConfig.modelPath, ttsConfig.port, ttsConfig.backend)
      setTTSStatus({ running: true, port: ttsConfig.port })
      message.success('TTS 已启动')
    } catch (err: any) { message.error(err?.message || '失败') }
  }

  const handleStopTTS = async () => {
    try {
      await stopTTSServer()
      setTTSStatus({ running: false, port: 0 })
      message.success('TTS 已停止')
    } catch (err: any) { message.error(err?.message || '失败') }
  }

  // ── 渲染 ────────────────────────────────────────────────

  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}><Spin size="large" /></div>
  }

  const isModelActive = (card: ModelCardData) =>
    activeEngine === card.engineId && activeModel === card.modelId

  const sidebarBtn = (cat: Category, icon: React.ReactNode, label: string) => (
    <div onClick={() => setCategory(cat)} style={{
      padding: '10px 14px', borderRadius: 'var(--radius-md)',
      cursor: 'pointer', fontSize: 13, fontWeight: 500,
      background: category === cat ? 'var(--color-primary)' : 'transparent',
      color: category === cat ? 'var(--on-primary)' : C('color-text-secondary'),
      transition: 'all 0.2s', display: 'flex', alignItems: 'center', gap: 8,
    }}>{icon} {label}</div>
  )

  const makeModels = (engine: EngineConfig): ModelCardData[] => {
    const ms: ModelCardData[] = (engine.models || []).map(m => ({
      modelId: m.id, modelName: m.id,
      engineId: engine.id, engineName: engine.name,
      engineType: engine.type, engineEnabled: engine.enabled,
      status: m.status || 'running',
    }))
    if (ms.length === 0 && engine.default_model) {
      ms.push({
        modelId: engine.default_model, modelName: engine.default_model,
        engineId: engine.id, engineName: engine.name,
        engineType: engine.type, engineEnabled: engine.enabled,
        status: 'unknown',
      })
    }
    return ms
  }

  const renderEngineCard = (engine: EngineConfig) => {
    const status = engineStatuses[engine.id]
    return (
      <Card key={engine.id} size="small" style={{
        background: 'var(--bg-glass)',
        border: `1px solid ${engine.enabled ? 'var(--border-subtle)' : 'transparent'}`,
        borderRadius: 'var(--radius-md)', opacity: engine.enabled ? 1 : 0.55,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
          <Space size={6}>
            <span style={{ color: engineColors[engine.id] }}>{engineIcons[engine.id]}</span>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{engine.name}</Typography.Text>
          </Space>
          <Switch checked={engine.enabled} onChange={v => handleToggleEnabled(engine, v)} size="small" />
        </div>
        <Space.Compact style={{ width: '100%', marginBottom: 6 }}>
          <Input size="small" placeholder="Base URL"
            value={editingURLs[engine.id] || ''}
            onChange={e => setEditingURLs(prev => ({ ...prev, [engine.id]: e.target.value }))}
            disabled={!engine.enabled}
            style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
          <Button size="small" onClick={() => handleSaveURL(engine)} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存</Button>
        </Space.Compact>
        <Space size={4}>
          {engine.type === 'xai' && !loggedIn ? (
            <Button size="small" type="primary" icon={<LoginOutlined />} onClick={() => login()} style={{ fontSize: 11 }}>登录 xAI</Button>
          ) : (
            <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>测试连接</Button>
          )}
          <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>刷新模型</Button>
        </Space>
        {status && (
          <div style={{ marginTop: 6, fontSize: 11 }}>
            {status.connected
              ? <span style={{ color: '#34d399' }}>✓ {status.model_count} 个模型 · {status.last_checked}</span>
              : <span style={{ color: '#fb7185' }}>✗ {status.error}</span>}
          </div>
        )}
      </Card>
    )
  }

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
        </div>
        <div style={{ flex: 1, overflow: 'auto', minWidth: 0 }}>

          {/* ═══ 语言模型 ═══ */}
          {category === 'llm' && (
            <>
              {engines.filter(e => e.enabled).length === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', textAlign: 'center', padding: 40 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>暂无启用的引擎。请在下方引擎管理中启用引擎并测试连接。</Typography.Text>
                </Card>
              ) : (
                engines.filter(e => e.enabled).map(engine => {
                  const models = makeModels(engine)
                  const status = engineStatuses[engine.id]
                  const color = engineColors[engine.id] || '#888'
                  return (
                    <div key={engine.id} style={{ marginBottom: 24 }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, paddingBottom: 8, borderBottom: `1px solid ${color}30` }}>
                        <Space size={8}>
                          <span style={{ fontSize: 18, color }}>{engineIcons[engine.id]}</span>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>{engine.name}</Typography.Text>
                          <Tag style={{ fontSize: 10 }}>{engineLabels[engine.id] || engine.type}</Tag>
                        </Space>
                        <Space size={4}>
                          {engine.type === 'xai' && !loggedIn ? (
                            <Button size="small" type="primary" icon={<LoginOutlined />} onClick={() => login()} style={{ fontSize: 11 }}>登录 xAI</Button>
                          ) : (
                            <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>测试连接</Button>
                          )}
                          <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>刷新模型</Button>
                        </Space>
                      </div>
                      {status && (
                        <div style={{ padding: '4px 8px', borderRadius: 'var(--radius-sm)', background: status.connected ? 'rgba(52,211,153,0.06)' : 'rgba(251,113,133,0.06)', marginBottom: 10, fontSize: 11 }}>
                          {status.connected
                            ? <span style={{ color: '#34d399' }}>✓ 已连接 · {status.model_count} 个模型 · {status.last_checked}</span>
                            : <span style={{ color: '#fb7185' }}>✗ {status.error}</span>}
                        </div>
                      )}
                      {models.length > 0 ? (
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 10 }}>
                          {models.map(card => {
                            const active = isModelActive(card)
                            return (
                              <Card key={card.modelId} size="small" style={{
                                background: active ? `linear-gradient(135deg, ${color}18, ${color}08)` : 'var(--bg-glass)',
                                border: active ? `2px solid ${color}` : '1px solid var(--border-subtle)',
                                borderRadius: 'var(--radius-md)', boxShadow: active ? `0 0 10px ${color}25` : 'none', transition: 'all 0.2s',
                              }}>
                                <Typography.Text strong style={{ color: active ? color : C('color-text'), fontSize: 14, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{card.modelName}</Typography.Text>
                                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                                  {active ? <Tag color="green" style={{ fontSize: 10, margin: 0 }}>● 运行中</Tag> : <span style={{ fontSize: 10, color: C('color-text-secondary') }}>空闲</span>}
                                  <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 'var(--radius-md)', fontSize: 11 }}>{active ? '已启动' : '启动'}</Button>
                                </div>
                              </Card>
                            )
                          })}
                        </div>
                      ) : (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, fontStyle: 'italic' }}>点击「测试连接」以发现可用模型</Typography.Text>
                      )}
                    </div>
                  )
                })
              )}
              <div style={{ marginTop: 8 }}>
                <Collapse ghost size="small" items={[{
                  key: 'engines', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />引擎管理</span>,
                  children: <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>{engines.map(e => renderEngineCard(e))}</div>,
                }]} />
              </div>
            </>
          )}

          {/* ═══ 图片生成 ═══ */}
          {category === 'image' && (
            <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)' }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>选择图片生成引擎。xAI 云端需要登录；ComfyUI 需要本地先启动服务。</Typography.Text>
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <SettingField label="后端引擎" value={imageBackend} type="select" onChange={(v: string) => setImageBackend(v)}
                  options={[{ label: '☁️ xAI 云端', value: 'xai' }, { label: '🏠 ComfyUI 本地', value: 'comfyui' }]} width={200} />
                {imageBackend === 'comfyui' && (
                  <>
                    <SettingField label="ComfyUI 服务地址" value={comfyUIURL} placeholder="http://127.0.0.1:8188" onChange={v => setComfyUIURL(v)} />
                    <SettingField label="ComfyUI 安装路径" value={comfyUIPath} placeholder="D:\\ComfyUI" onChange={v => setComfyUIPath(v)} />
                    <SettingField label="Python 解释器路径" value={comfyUIPythonPath} placeholder="D:\\ComfyUI\\python_embeded\\python.exe" onChange={v => setComfyUIPythonPath(v)} />
                    <SettingField label="生成模型" value={imageModel} type="select" onChange={(v: string) => setImageModel(v)}
                      options={[{ label: '🌊 Flux Dev', value: 'flux' }, { label: '⚡ Z-Image-Turbo', value: 'z-image-turbo' }]} width={230} />
                  </>
                )}
                <SettingField label="图片存放目录" value={imageSaveDir} placeholder="D:\\AI\\images" onChange={v => setImageSaveDir(v)} />
                <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 'var(--radius-md)' }}>💾 保存配置</Button>
              </Space>
            </Card>
          )}

          {/* ═══ 语音模型 ═══ */}
          {category === 'tts' && (
            <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)' }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>引擎链：Edge TTS → SAPI → VoxCPM（可选本地AI）。</Typography.Text>
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <SettingField label="VoxCPM 可执行文件路径" value={ttsConfig.serverPath} placeholder="留空使用 PATH" onChange={v => setTTSConfig(p => ({ ...p, serverPath: v }))} />
                <SettingField label="GGUF 模型文件路径" value={ttsConfig.modelPath} placeholder="D:\\models\\voxcpm.gguf" onChange={v => setTTSConfig(p => ({ ...p, modelPath: v }))} />
                <Space direction="horizontal">
                  <SettingField label="端口" value={ttsConfig.port} type="number" onChange={v => setTTSConfig(p => ({ ...p, port: v }))} width={100} />
                  <SettingField label="后端" value={ttsConfig.backend} type="select" onChange={v => setTTSConfig(p => ({ ...p, backend: v }))} options={[{ label: 'CUDA', value: 'cuda' }, { label: 'CPU', value: 'cpu' }, { label: 'Vulkan', value: 'vulkan' }]} width={100} />
                </Space>
                <SettingField label="语速" value={ttsConfig.speed} type="number" onChange={v => setTTSConfig(p => ({ ...p, speed: v }))} width={80} />
                <Space direction="horizontal">
                  <Button type="primary" onClick={handleSaveTTS} style={{ borderRadius: 'var(--radius-md)' }}>💾 保存</Button>
                  <Button onClick={handleStartTTS} disabled={ttsStatus.running} style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text-secondary'), borderRadius: 'var(--radius-md)' }}>▶ 启动</Button>
                  <Button onClick={handleStopTTS} disabled={!ttsStatus.running} danger style={{ borderRadius: 'var(--radius-md)' }}>⏹ 停止</Button>
                </Space>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>状态：{ttsStatus.running ? `🟢 运行中 (${ttsStatus.port})` : '⚫ 未启动'}</Typography.Text>
              </Space>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

export default ModelCenterPage
