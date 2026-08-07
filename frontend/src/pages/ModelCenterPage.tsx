import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Card, Switch, Button, Input, Space, Tag, message, Spin, Collapse, Select, Segmented } from 'antd'
import {
  CloudOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ReloadOutlined, ThunderboltOutlined,
  DesktopOutlined, RocketOutlined, PictureOutlined, SoundOutlined, AudioOutlined,
  CaretRightOutlined, SettingOutlined, LoginOutlined, LogoutOutlined, KeyOutlined,
  LinkOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import SettingField from '../components/SettingField'
import { C } from '../utils/theme'
import * as App from '../../wailsjs/go/app/App'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
  type EngineConfig, type ModelInfo, type EngineStatus,
} from '../api/engines'
import {
  getConfig, saveConfig,
  getImageBackendInfo, setImageBackend as setImageBackendAPI,
} from '../api/settings'
import { startComfyUI, stopComfyUI, getComfyUIStatus } from '../api/image'

type Category = 'llm' | 'image' | 'tts' | 'engine' | 'bind'

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
  const [loggingIn, setLoggingIn] = useState(false)
  const [imageBackend, setImageBackend] = useState('xai')
  const [comfyUIURL, setComfyUIURL] = useState('http://127.0.0.1:8188')
  const [imageSaveDir, setImageSaveDir] = useState('')
  const [imageModel, setImageModel] = useState('krea2')
  const [comfyUIPath, setComfyUIPath] = useState('')
  const [comfyUIPythonPath, setComfyUIPythonPath] = useState('')
  const [imageBackendSaving, setImageBackendSaving] = useState(false)
  const [comfyStatus, setComfyStatus] = useState<{ running: boolean; port: number }>({ running: false, port: 0 })
  const [comfyBusy, setComfyBusy] = useState(false)

  // 语音管道三段激活模型（STT/LLM/TTS，来自模型中心选择）
  const [voiceCfg, setVoiceCfg] = useState<{ stt: { engine: string; model: string }; llm: { engine: string; model: string }; tts: { engine: string; model: string } }>({ stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '' } })

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
      if (cfg?.image_model) setImageModel(cfg.image_model)
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (_) {}
    try {
      const st: any = await getComfyUIStatus()
      if (st) setComfyStatus({ running: !!st.running, port: st.port || 0 })
    } catch (_) {}
  }, [])

  const handleToggleComfy = async () => {
    setComfyBusy(true)
    try {
      if (comfyStatus.running) { await stopComfyUI(); setComfyStatus({ running: false, port: 0 }) }
      else { await startComfyUI(); setComfyStatus({ running: true, port: comfyUIURL.split(':').pop() ? 8188 : 8188 }) }
      message.success(comfyStatus.running ? 'ComfyUI 已停止' : 'ComfyUI 已启动')
    } catch (err: any) { message.error(err?.message || '操作失败') }
    finally { setComfyBusy(false) }
  }

  // 加载语音管道三段激活模型
  const loadVoiceCfg = useCallback(async () => {
    try {
      const cfg = await App.GetVoicePipelineConfig()
      if (cfg) {
        setVoiceCfg({
          stt: { engine: cfg.stt?.engine || '', model: cfg.stt?.model || '' },
          llm: { engine: cfg.llm?.engine || '', model: cfg.llm?.model || '' },
          tts: { engine: cfg.tts?.engine || '', model: cfg.tts?.model || '' },
        })
      }
    } catch (_) {}
  }, [])


  // ComfyUI 本地图片模型（本机硬编码：ComfyUI 非 LLM 引擎，模型不入引擎列表）
  const COMFY_IMAGES = [
    { modelId: 'krea2', modelName: 'Krea2 Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
    { modelId: 'z-image-turbo', modelName: 'Z-Image-Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
  ]

  // ── 功能模型绑定（聊天/轻语/小说/办公 各自独立 LLM，持久化重启不丢）──
  const FEATURES: { key: string; label: string; icon: string }[] = [
    { key: 'chat', label: '聊天', icon: '💬' },
    { key: 'whisper', label: '轻语', icon: '🫀' },
    { key: 'novel', label: '小说', icon: '📖' },
    { key: 'office', label: '方案编写', icon: '📄' },
    { key: 'gaea', label: '办公', icon: '🛠️' },
  ]
  const [featureCfg, setFeatureCfg] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureDraft, setFeatureDraft] = useState<Record<string, { engine: string; model: string }>>({})

  const loadFeatureCfg = useCallback(async () => {
    try {
      const cfg: Record<string, { engine: string; model: string }> = {}
      for (const f of ['chat', 'whisper', 'novel', 'office', 'gaea']) {
        const r: any = await App.GetFeatureModel(f)
        cfg[f] = { engine: r?.engine || '', model: r?.model || '' }
      }
      setFeatureCfg(cfg)
      setFeatureDraft(JSON.parse(JSON.stringify(cfg)))
    } catch (_) {}
  }, [])

  const handleSaveFeature = async (key: string) => {
    const d = featureDraft[key]
    if (!d?.engine || !d?.model) { message.warning('请先选择引擎和模型'); return }
    try {
      await App.SetFeatureModel(key, d.engine, d.model)
      message.success(`${FEATURES.find(f => f.key === key)?.label}模型已绑定并持久化`)
      loadFeatureCfg()
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    }
  }

  useEffect(() => { loadAll(); loadImageBackend(); loadVoiceCfg(); loadFeatureCfg() }, [loadVoiceCfg, loadFeatureCfg])

  // 设为语音识别/合成（模型中心 → 语音管道）
  const handleSetVoiceModel = async (kind: 'asr' | 'tts', engineId: string, modelId: string) => {
    try {
      if (kind === 'asr') await App.SetActiveASRModel(engineId, modelId)
      else await App.SetActiveTTSModel(engineId, modelId)
      message.success(`已设为${kind === 'asr' ? '语音识别' : '语音合成'}：${modelId}`)
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '设置失败')
    }
  }

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
    try { await setImageBackendAPI(imageBackend, comfyUIURL, imageModel, imageSaveDir); message.success('已保存') }
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
          {sidebarBtn('bind', <LinkOutlined />, '功能绑定')}
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
                  : <Button type="primary" icon={<LoginOutlined />} loading={loggingIn}
                    onClick={async () => {
                      setLoggingIn(true)
                      try {
                        await login()
                        message.success('xAI 登录成功！')
                        await loadAll()
                      } catch (err: any) {
                        message.error('登录失败：' + (err?.message || err || '未知错误，请检查浏览器是否完成了 xAI 授权'))
                      } finally {
                        setLoggingIn(false)
                      }
                    }}
                    style={{ background: 'linear-gradient(135deg, #6366f1, #2563eb)', border: 'none', borderRadius: 8, fontWeight: 500 }}>登录 xAI</Button>}
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
                            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                              <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabels[card.engineId]}</Tag>
                              {active ? (
                                <Tag color="green" style={{ fontSize: 10, margin: 0 }}>● 运行中</Tag>
                              ) : (
                                <Tag color={card.status === 'stopped' ? 'default' : 'blue'} style={{ fontSize: 10, margin: 0 }}>{card.status === 'stopped' ? '○ 已停止' : '○ 就绪'}</Tag>
                              )}
                              <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 8, fontSize: 11, marginLeft: 'auto' }}>{active ? '已启动' : '启动'}</Button>
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
          {category === 'bind' && (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
                <LinkOutlined style={{ color: C('color-text-secondary') }} />
                <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>功能模型绑定</Typography.Text>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                  各功能板块独立模型，设置后持久化（重启不丢）
                </Typography.Text>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 14 }}>
                {FEATURES.map(f => {
                  const cur = featureCfg[f.key]
                  const draft = featureDraft[f.key] || { engine: '', model: '' }
                  const engineModels = draft.engine ? llmModels.filter(m => m.engineId === draft.engine) : []
                  const bound = !!cur?.engine && !!cur?.model
                  return (
                    <Card key={f.key} size="small" style={{ background: 'var(--bg-glass)', border: bound ? '1px solid rgba(34,197,94,0.35)' : '1px solid var(--border-subtle)', borderRadius: 12 }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                        <Space size={6}>
                          <span style={{ fontSize: 16 }}>{f.icon}</span>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{f.label}</Typography.Text>
                          {f.key === 'whisper' && (
                            <>
                              <Tag color="purple" style={{ fontSize: 9, margin: 0 }}>TTS {voiceCfg.tts.model || '自动'}</Tag>
                              <Tag color="blue" style={{ fontSize: 9, margin: 0 }}>STT {voiceCfg.stt.model || '自动'}</Tag>
                            </>
                          )}
                          {f.key === 'novel' && <Tag color="orange" style={{ fontSize: 9, margin: 0 }}>剧照 {imageModel || '—'}</Tag>}
                        </Space>
                        <Tag color={bound ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{bound ? '已绑定' : '未绑定'}</Tag>
                      </div>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 10, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {bound ? `当前：${cur!.engine} / ${cur!.model}` : '尚未绑定，选择引擎和模型后点绑定'}
                      </Typography.Text>
                      <div style={{ display: 'flex', gap: 8 }}>
                        <Select size="small" placeholder="引擎" value={draft.engine || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: v, model: '' } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engines.filter(e => e.enabled).map(e => ({ value: e.id, label: engineLabels[e.id] || e.id }))} />
                        <Select size="small" placeholder="模型" value={draft.model || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: p[f.key]?.engine || '', model: v } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engineModels.map(m => ({ value: m.modelId, label: m.modelName }))} />
                      </div>
                      <Button size="small" type={bound ? 'primary' : 'default'} block onClick={() => handleSaveFeature(f.key)} style={{ marginTop: 10, fontSize: 11 }}>
                        {bound ? '更新绑定' : '绑定'}
                      </Button>
                    </Card>
                  )
                })}
                {/* 绘梦：自身界面选择 */}
                <Card size="small" style={{ background: 'rgba(255,255,255,0.02)', border: '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 16 }}>🎨</span>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>绘梦</Typography.Text>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 8, lineHeight: 1.6 }}>
                    图片模型在绘梦界面内选择（后端 / 模型 / ComfyUI 启停），无需在此重复设置
                  </Typography.Text>
                </Card>
              </div>
            </div>
          )}

          {category === 'image' && (
            <>
              {/* 图片后端 + ComfyUI + 模型 */}
              <Card style={{ marginBottom: 20, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                  {/* 后端选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片生成后端</Typography.Text>
                    <Segmented
                      value={imageBackend}
                      onChange={(v) => setImageBackend(v as string)}
                      options={[
                        { value: 'xai', label: '☁️ xAI 云端' },
                        { value: 'comfyui', label: '🏠 ComfyUI 本地' },
                        { value: 'herdsman', label: '🚀 Herdsman' },
                        { value: 'ollama', label: '🖥 Ollama' },
                      ]}
                    />
                  </div>

                  {/* ComfyUI 配置（本机默认已写死，仅展示 + 启停） */}
                  {imageBackend === 'comfyui' && (
                    <div style={{ padding: '12px 14px', borderRadius: 10, background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-subtle)' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                        <span style={{
                          width: 8, height: 8, borderRadius: '50%',
                          background: comfyStatus.running ? '#22c55e' : '#64748b',
                          boxShadow: comfyStatus.running ? '0 0 8px #22c55e' : 'none',
                        }} />
                        <Typography.Text style={{ color: C('color-text'), fontSize: 13, fontWeight: 600 }}>
                          ComfyUI {comfyStatus.running ? `运行中 (端口 ${comfyStatus.port || 8188})` : '未启动'}
                        </Typography.Text>
                        <Button size="small" type={comfyStatus.running ? 'default' : 'primary'} loading={comfyBusy}
                          onClick={handleToggleComfy} style={{ fontSize: 11 }}>
                          {comfyStatus.running ? '⏹ 停止' : '▶ 启动'}
                        </Button>
                      </div>
                      <div style={{ fontSize: 11, color: C('color-text-secondary'), marginTop: 8, lineHeight: 1.8 }}>
                        <div>URL：<span style={{ color: C('color-text') }}>{comfyUIURL}</span></div>
                        <div>启动位置：<span style={{ color: C('color-text') }}>{comfyUIPath || 'C:\\AI\\ComfyUI\\ComfyUI（默认）'}</span></div>
                        <div>Python：<span style={{ color: C('color-text') }}>{comfyUIPythonPath || 'C:\\AI\\ComfyUI\\standalone-env\\python.exe（默认）'}</span></div>
                      </div>
                    </div>
                  )}

                  {/* 图片模型选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片模型</Typography.Text>
                    <Select
                      size="middle" style={{ width: 320 }} value={imageModel}
                      onChange={setImageModel}
                      options={[
                        ...(imageBackend === 'comfyui' ? COMFY_IMAGES : []).map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })),
                        ...imageModels.map(m => ({ value: m.modelId, label: m.modelName })),
                        ...(imageBackend !== 'comfyui' ? COMFY_IMAGES.map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })) : []),
                        { value: 'grok-imagine-image-quality', label: 'Grok Imagine（xAI）' },
                      ]}
                    />
                  </div>

                  {/* 保存 */}
                  <div>
                    <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>
                      💾 保存图片后端设置
                    </Button>
                  </div>
                </div>
              </Card>

              {/* 发现的图片模型（含 ComfyUI 本地模型 krea2 / z-image-turbo） */}
              {(imageModels.length > 0 || imageBackend === 'comfyui') && (
                <div style={{ marginBottom: 24 }}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>发现的图片模型</Typography.Text>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                    {[...COMFY_IMAGES, ...imageModels.filter(m => m.engineId !== 'comfyui')].map(m => (
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
              <Collapse ghost size="small" defaultActiveKey={['img-cfg']} items={[{
                key: 'img-cfg', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />图片存储</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="图片保存目录" value={imageSaveDir} onChange={v => setImageSaveDir(v)} placeholder="默认: Pictures/gaea" />
                      <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>💾 保存</Button>
                    </Space>
                  </Card>
                ),
              }]} />
            </>
          )}

          {/* TTS / Voice */}
          {category === 'tts' && (
            <>
              {/* 三段激活模型汇总（模型中心 → 语音管道） */}
              <Card style={{ marginBottom: 16, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', fontSize: 12 }}>
                  <span style={{ color: C('color-text-secondary'), fontWeight: 600, marginRight: 4 }}>语音管道：</span>
                  <Tag color={voiceCfg.stt.model ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                    🎙️ 识别 {voiceCfg.stt.model || '自动'}
                  </Tag>
                  <Tag color={voiceCfg.llm.model ? 'green' : 'default'} style={{ fontSize: 11 }}>
                    💬 对话 {voiceCfg.llm.model || '默认'}
                  </Tag>
                  <Tag color={voiceCfg.tts.model ? 'purple' : 'default'} style={{ fontSize: 11 }}>
                    🔊 合成 {voiceCfg.tts.model || '自动'}
                  </Tag>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginLeft: 'auto' }}>
                    点击下方卡片可切换识别/合成模型（自动持久化，重启保留）
                  </Typography.Text>
                </div>
              </Card>

              {ttsModels.length === 0 && sttModels.length === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <SoundOutlined style={{ fontSize: 32, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, display: 'block' }}>未发现语音模型</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 6 }}>
                    请先在「引擎管理」中刷新模型列表（Herdsman 本地引擎可提供 whisper / qwen3-tts 等）
                  </Typography.Text>
                </Card>
              ) : (
                <>
                  {ttsModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🔊 TTS 语音合成</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {ttsModels.map(m => {
                          const active = voiceCfg.tts.engine === m.engineId && voiceCfg.tts.model === m.modelId
                          return (
                            <Card key={m.modelId} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                                <Tag color="purple" style={{ fontSize: 10 }}>TTS</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="purple" style={{ marginTop: 6, fontSize: 10 }}>● 语音合成中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<SoundOutlined />}
                                  onClick={() => handleSetVoiceModel('tts', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音合成' : '设为语音合成'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  {sttModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🎙️ STT 语音识别</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {sttModels.map(m => {
                          const active = voiceCfg.stt.engine === m.engineId && voiceCfg.stt.model === m.modelId
                          return (
                            <Card key={m.modelId} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                                <Tag color="blue" style={{ fontSize: 10 }}>STT</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="blue" style={{ marginTop: 6, fontSize: 10 }}>● 语音识别中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<AudioOutlined />}
                                  onClick={() => handleSetVoiceModel('asr', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音识别' : '设为语音识别'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </>
              )}
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
