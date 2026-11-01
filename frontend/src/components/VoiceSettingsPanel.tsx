// VoiceSettingsPanel.tsx — 聊天语音设置面板

import { useState, useEffect, useMemo, useRef } from 'react'
import { Card, Switch, Select, Slider, Button, Typography, Tag, Input, message } from 'antd'
import { AudioOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'
import { listMicDevices, loadMicDeviceId, saveMicDeviceId } from '../hooks/useVoiceChat'

const { Text } = Typography

/** 语音服务健康状态（后端 VoiceHealth 动态载荷的最小消费面） */
interface VoiceHealthInfo {
  asrReady?: boolean
  ttsReady?: boolean
  state?: string
  error?: string
}

/** VoiceGetSettings 动态载荷（bridge Record<string, unknown>）在本面板的消费面 */
interface VoiceSettingsInfo {
  ttsEnabled?: boolean
  ttsVoice?: string
  voiceMode?: string
  interruptThresholdMs?: number
  silenceThresholdMs?: number
  realtimeProvider?: string
  realtimeModel?: string
  realtimeHasKey?: boolean
  breezeVoiceDesc?: string
}

/** GetVoicePipelineConfig 动态载荷中 TTS 模型读取面 */
interface VoicePipelineInfo {
  chatTts?: { model?: string }
  tts?: { model?: string }
}

// Edge TTS 音色（在线免费，zh-CN）
const EDGE_VOICES = [
  { value: 'zh-CN-YunxiNeural', label: '云希 (男)' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 (女)' },
  { value: 'zh-CN-YunjianNeural', label: '云健 (男)' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 (女)' },
]

// Herdsman qwen3-tts 音色兜底（服务端支持列表查询失败时使用）
const HERDSMAN_VOICES = ['serena', 'vivian', 'sohee', 'aiden', 'dylan', 'eric', 'ono_anna', 'ryan', 'uncle_fu']

// xAI Grok TTS 音色单一来源（模型中心 modelcenter/utils.tsx，T6-6.4 单源收敛）
import { XAI_VOICES } from '../pages/modelcenter/utils'

// CosyVoice2 内置音色（本地服务端 /v1/audio/info 返回，查询失败时兜底）
const COSYVOICE_VOICES = ['中文女', '中文男', '英文女', '英文男']


const VOICE_LABELS: Record<string, string> = {
  serena: 'Serena (女)',
  vivian: 'Vivian (女)',
  sohee: 'Sohee (女)',
  aiden: 'Aiden (男)',
  dylan: 'Dylan (男)',
  eric: 'Eric (男)',
  ryan: 'Ryan (男)',
  ono_anna: 'Anna (女)',
  uncle_fu: 'Uncle Fu (男)',
}

export default function VoiceSettingsPanel() {
  const [health, setHealth] = useState<VoiceHealthInfo | null>(null)
  const [checking, setChecking] = useState(false)

  // 语音设置
  const [ttsEnabled, setTtsEnabled] = useState(true)
  const [ttsVoice, setTtsVoice] = useState('zh-CN-YunxiNeural')
  const [ttsModel, setTtsModel] = useState('')
  const [herdsmanVoices, setHerdsmanVoices] = useState<string[]>([])
  const [voiceMode, setVoiceMode] = useState<'vad' | 'ptt'>('vad')
  const [interruptMs, setInterruptMs] = useState(500)
  const [silenceMs, setSilenceMs] = useState(1000)

  // 实时语音 Realtime 档（v4.8.1 S1）：provider/model/key（key 回显只有 hasKey 布尔，
  // 明文永不回传前端）
  const [rtProvider, setRtProvider] = useState('')
  const [rtModel, setRtModel] = useState('')
  const [rtKey, setRtKey] = useState('')
  const [rtHasKey, setRtHasKey] = useState(false)
  const [rtSaving, setRtSaving] = useState(false)

  // 麦克风设备选择（FE6-02 终局配套）：默认设备可能是零信号虚拟音频，
  // 采集侧 getUserMedia 以此为准（localStorage 持久化，前端纯关注点）
  const [micDevices, setMicDevices] = useState<Array<{ deviceId: string; label: string; isDefault: boolean }>>([])
  const [micId, setMicId] = useState<string>(() => loadMicDeviceId())
  const refreshMicDevices = async () => {
    const list = await listMicDevices()
    setMicDevices(list)
    setMicId(loadMicDeviceId())
  }

  // ── Breeze 音色区（克隆音色 + voice design 描述） ──
  interface BreezeVoiceInfo { id: string; seconds?: number; saved?: boolean; description?: string }
  const [breezeVoices, setBreezeVoices] = useState<BreezeVoiceInfo[]>([])
  const [breezeVoicesStarted, setBreezeVoicesStarted] = useState(true)
  const [breezeVoicesLoading, setBreezeVoicesLoading] = useState(false)
  const [breezeDesc, setBreezeDesc] = useState('')
  // 克隆表单
  const [cloneName, setCloneName] = useState('')
  const [cloneDesc, setCloneDesc] = useState('')
  const [cloneRefText, setCloneRefText] = useState('')
  const [cloneSaving, setCloneSaving] = useState(false)
  const cloneFileRef = useRef<HTMLInputElement | null>(null)
  const [cloneFileB64, setCloneFileB64] = useState('')
  const [cloneFileName, setCloneFileName] = useState('')
  // 试音电平
  const [metering, setMetering] = useState(false)
  const [meterLevel, setMeterLevel] = useState(0)
  const [meterResult, setMeterResult] = useState<null | 'signal' | 'silent'>(null)
  const meterStopRef = useRef<(() => void) | null>(null)

  const refreshBreezeVoices = async () => {
    setBreezeVoicesLoading(true)
    try {
      const res = await app.BreezeListVoices() as { started?: boolean; voices?: BreezeVoiceInfo[]; error?: string }
      setBreezeVoicesStarted(!!res?.started)
      setBreezeVoices(Array.isArray(res?.voices) ? res.voices : [])
    } catch (_) {
      setBreezeVoicesStarted(false)
      setBreezeVoices([])
    } finally {
      setBreezeVoicesLoading(false)
    }
  }

  // 试音：在当前所选录音设备上采 3s，实时电平条 + 有/无信号结论（与采集同口径：
  // 频域任一字节非零 = 有信号）。
  const runMicMeter = async () => {
    if (metering) return
    setMetering(true)
    setMeterResult(null)
    setMeterLevel(0)
    let stream: MediaStream | null = null
    let ctx: AudioContext | null = null
    const savedMicId = loadMicDeviceId()
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          ...(savedMicId ? { deviceId: { exact: savedMicId } } : {}),
          echoCancellation: true,
          noiseSuppression: true,
        },
      })
      ctx = new AudioContext()
      const src = ctx.createMediaStreamSource(stream)
      const analyser = ctx.createAnalyser()
      analyser.fftSize = 256
      src.connect(analyser)
      const buf = new Uint8Array(analyser.frequencyBinCount)
      let hasSignal = false
      const deadline = Date.now() + 3000
      while (Date.now() < deadline) {
        analyser.getByteFrequencyData(buf)
        let max = 0
        for (let i = 0; i < buf.length; i++) if (buf[i] > max) max = buf[i]
        if (max > 0) hasSignal = true
        setMeterLevel(Math.min(max / 140, 1))
        await new Promise(r => setTimeout(r, 80))
      }
      setMeterResult(hasSignal ? 'signal' : 'silent')
    } catch (err) {
      setMeterResult('silent')
      message.warning(`试音失败：${err instanceof Error ? err.name : String(err)}`)
    } finally {
      stream?.getTracks().forEach(t => t.stop())
      await ctx?.close().catch(() => {})
      setMetering(false)
      setMeterLevel(0)
      meterStopRef.current = null
    }
  }

  const checkHealth = async () => {
    setChecking(true)
    try {
      const h = (await app.VoiceHealth()) as VoiceHealthInfo
      setHealth(h || { asrReady: false, ttsReady: false })
    } catch (_) {
      setHealth({ asrReady: false, ttsReady: false, error: '无法获取语音服务状态' })
    }
    void refreshMicDevices() // 插拔麦克风后「检测」即刷新设备清单
    setChecking(false)
  }

  const loadSettings = async () => {
    try {
      const [v, pipeline] = await Promise.all([
        app.VoiceGetSettings(),
        app.GetVoicePipelineConfig(),
      ])
      const settings = (v || {}) as VoiceSettingsInfo
      if (settings.ttsEnabled !== undefined) setTtsEnabled(!!settings.ttsEnabled)
      if (settings.ttsVoice) setTtsVoice(settings.ttsVoice)
      if (settings.voiceMode) setVoiceMode(settings.voiceMode === 'ptt' ? 'ptt' : 'vad')
      if (settings.interruptThresholdMs) setInterruptMs(settings.interruptThresholdMs)
      if (settings.silenceThresholdMs) setSilenceMs(settings.silenceThresholdMs)
      if (settings.realtimeProvider !== undefined) setRtProvider(String(settings.realtimeProvider || ''))
      if (settings.realtimeModel !== undefined) setRtModel(String(settings.realtimeModel || ''))
      if (settings.realtimeHasKey !== undefined) setRtHasKey(!!settings.realtimeHasKey)
      if (settings.breezeVoiceDesc !== undefined) setBreezeDesc(String(settings.breezeVoiceDesc || ''))

      const model = (pipeline as VoicePipelineInfo | undefined)?.chatTts?.model
        || (pipeline as VoicePipelineInfo | undefined)?.tts?.model
        || ''
      setTtsModel(model)
      const lower = model.toLowerCase()
      if (lower.includes('qwen3') || lower.includes('customvoice') || lower.includes('cosyvoice')) {
        try {
          const speakers = (await app.GetTTSSpeakers(model)) || []
          if (Array.isArray(speakers) && speakers.length > 0) {
            setHerdsmanVoices(speakers)
            // 当前音色不在支持列表时自动对齐到默认音色
            if (settings.ttsVoice && !speakers.includes(settings.ttsVoice)) {
              const fallback = speakers.includes('serena') ? 'serena' : speakers[0]
              setTtsVoice(fallback)
              app.VoiceApplySettings({ ttsVoice: fallback }).catch(() => {})
            }
          }
        } catch (_) { /* 查询失败使用兜底列表 */ }
      }
    } catch (_) { /* 初始化失败忽略 */ }
  }

  useEffect(() => { checkHealth(); loadSettings() }, [])

  const applyVoiceSettings = (patch: Record<string, unknown>) => {
    // v4.361：失败可见化——此前静默不落盘，用户调了音色/语速重进发现没保存。
    app.VoiceApplySettings(patch).catch(() => message.error('语音设置保存失败，请重试'))
  }

  // 保存实时语音配置（S1）：key 只在用户填了新值时下发（空=不改动；填空格并
  // 保存=清除——后端 trim 后落盘），provider/model 即时持久化。
  const saveRealtime = async () => {
    setRtSaving(true)
    try {
      const patch: Record<string, unknown> = { realtimeProvider: rtProvider, realtimeModel: rtModel }
      const key = rtKey.trim()
      if (rtKey !== '') patch.realtimeAPIKey = key // 未输入=不改动已存 key
      await app.VoiceApplySettings(patch)
      setRtKey('')
      const v = await app.VoiceGetSettings()
      if (v) setRtHasKey(!!v.realtimeHasKey)
      message.success(rtProvider ? '实时语音配置已保存' : '实时语音已关闭')
    } catch (_) {
      message.error('保存失败，请重试')
    } finally {
      setRtSaving(false)
    }
  }

  // 根据当前 TTS 模型决定音色选项：qwen3/customvoice → Herdsman 音色，其余 → Edge 音色
  const isHerdsman = useMemo(() => {
    const l = ttsModel.toLowerCase()
    return l.includes('qwen3') || l.includes('customvoice')
  }, [ttsModel])

  const isXai = useMemo(() => ttsModel.toLowerCase().includes('grok-tts'), [ttsModel])
  const isCosyvoice = useMemo(() => ttsModel.toLowerCase().includes('cosyvoice'), [ttsModel])
  // Breeze-TTS-2：无预置音色（voice 留空 = voice design，音色由描述/情感指令决定）
  const isBreeze = useMemo(() => ttsModel.toLowerCase().includes('breeze'), [ttsModel])

  const voiceOptions = useMemo(() => {
    if (isHerdsman) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES
      return list.map(v => ({ value: v, label: VOICE_LABELS[v] || v }))
    }
    if (isXai) return XAI_VOICES
    if (isCosyvoice) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES
      return list.map(v => ({ value: v, label: v }))
    }
    if (isBreeze) return [] // 无预置音色，隐藏选择器（design 模式）
    return EDGE_VOICES
  }, [isHerdsman, isXai, isCosyvoice, isBreeze, herdsmanVoices])

  const effectiveVoice = useMemo(() => {
    if (isHerdsman) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : HERDSMAN_VOICES
      return list.includes(ttsVoice) ? ttsVoice : (list.includes('serena') ? 'serena' : (list[0] || 'serena'))
    }
    if (isXai) return XAI_VOICES.some(v => v.value === ttsVoice) ? ttsVoice : 'eve'
    if (isCosyvoice) {
      const list = herdsmanVoices.length > 0 ? herdsmanVoices : COSYVOICE_VOICES
      return list.includes(ttsVoice) ? ttsVoice : '中文女'
    }
    if (isBreeze) return ttsVoice // 克隆音色 id（空=design 模式）
    return ttsVoice || 'zh-CN-YunxiNeural'
  }, [isHerdsman, isXai, isCosyvoice, isBreeze, herdsmanVoices, ttsVoice])

  // Breeze 音色列表：进入 breeze 引擎时拉一次（服务未起则提示，启动后点刷新）
  useEffect(() => {
    if (isBreeze) void refreshBreezeVoices()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isBreeze])

  // ── 角色音色预设（角色库实体→音色）：切换角色即切换音色 ──
  // 数据源=角色库 chatEnabled 实体（与聊天角色同一 id 空间）；行内编辑走
  // Get→patch 两字段→Save（全量保存前先取最新，防并发覆盖整卡）。
  interface PersonaRow { id: string; label: string; gender: string; voiceId: string; desc: string; descSaved: string; dirty?: boolean }
  const [personaRows, setPersonaRows] = useState<PersonaRow[]>([])
  const [personaLoading, setPersonaLoading] = useState(false)

  const loadPersonaRows = async () => {
    setPersonaLoading(true)
    try {
      const res = await app.CharacterList('', '', true, 1, 500) as { items?: Array<{ id: string; name: string; gender?: string; ttsVoiceId?: string; ttsDesc?: string }> }
      setPersonaRows((res?.items || []).map(c => ({
        id: c.id, label: c.name || c.id, gender: c.gender || '',
        voiceId: c.ttsVoiceId || '', desc: c.ttsDesc || '', descSaved: c.ttsDesc || '',
      })))
    } catch (_) {
      setPersonaRows([])
    } finally {
      setPersonaLoading(false)
    }
  }

  const patchPersonaVoice = async (id: string, patch: { voiceId?: string; desc?: string }) => {
    setPersonaRows(prev => prev.map(r => r.id === id ? { ...r, ...patch, dirty: true } : r))
    try {
      const res = await app.CharacterGet(id) as { character?: { [k: string]: unknown } }
      const card = res?.character
      if (!card) { message.error('角色不存在或已删除'); return }
      await app.CharacterSave(JSON.stringify({
        ...card,
        ...(patch.voiceId !== undefined ? { ttsVoiceId: patch.voiceId } : {}),
        ...(patch.desc !== undefined ? { ttsDesc: patch.desc } : {}),
      }))
      setPersonaRows(prev => prev.map(r => r.id === id ? {
        ...r, dirty: false,
        ...(patch.desc !== undefined ? { descSaved: patch.desc } : {}),
      } : r))
    } catch (err) {
      message.error(`保存角色音色失败：${err instanceof Error ? err.message : String(err)}`)
    }
  }

  const mountRef = useRef(false)
  useEffect(() => {
    if (mountRef.current) return
    mountRef.current = true
    void loadPersonaRows()
    void refreshBreezeVoices()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Text strong style={{ fontSize: 13, color: 'var(--color-text)' }}>语音设置</Text>
        <Button size="small" icon={<ReloadOutlined />} loading={checking} onClick={checkHealth}
          style={{ fontSize: 11 }}>检测</Button>
      </div>

      {/* 健康状态 */}
      {health && (
        <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 10 }}>
          <div style={{ display: 'flex', gap: 16, fontSize: 11 }}>
            <span>
              {health.asrReady ? <CheckCircleOutlined style={{ color: 'var(--color-success)' }} /> : <CloseCircleOutlined style={{ color: 'var(--color-destructive)' }} />}
              {' '}语音识别
            </span>
            <span>
              {health.ttsReady ? <CheckCircleOutlined style={{ color: 'var(--color-success)' }} /> : <CloseCircleOutlined style={{ color: 'var(--color-destructive)' }} />}
              {' '}语音合成
            </span>
            <span style={{ color: 'var(--color-text-secondary)' }}>{health.state || 'idle'}</span>
          </div>
        </Card>
      )}

      {/* TTS 开关 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>朗读回复</div>
            <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 2, opacity: 0.75 }}>AI 回复时自动朗读</div>
          </div>
          <Switch size="small" checked={ttsEnabled} onChange={v => {
            setTtsEnabled(v)
            applyVoiceSettings({ ttsEnabled: v })
          }} />
        </div>
      </Card>

      {/* TTS 音色 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>合成音色</div>
        {isHerdsman && ttsModel && (
          <div style={{ fontSize: 10, color: '#8b5cf6', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（Herdsman） */}
            当前模型：{ttsModel}（Herdsman 音色，实时从服务端获取）
          </div>
        )}
        {isXai && ttsModel && (
          <div style={{ fontSize: 10, color: '#60a5fa', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（xAI） */}
            当前模型：{ttsModel}（xAI 云端 Grok 音色）
          </div>
        )}
        {isCosyvoice && ttsModel && (
          <div style={{ fontSize: 10, color: '#f472b6', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（CosyVoice） */}
            当前模型：{ttsModel}（本地 CosyVoice，支持参考音频克隆音色）
          </div>
        )}
        {isBreeze && ttsModel && (
          <div style={{ fontSize: 10, color: '#2dd4bf', marginBottom: 6 }}> {/* hex-exempt 引擎品牌识别色（Breeze） */}
            当前模型：{ttsModel}（本地 Breeze-TTS-2）
          </div>
        )}
        {isBreeze ? (
          <>
            <Select
              size="small"
              value={ttsVoice || ''}
              onChange={v => { setTtsVoice(String(v)); applyVoiceSettings({ ttsVoice: String(v) }) }}
              style={{ width: '100%' }}
              options={[
                { value: '', label: '自动（按下方描述定音色）' },
                ...breezeVoices.map(v => ({
                  value: v.id,
                  label: `${v.id}（${(v.seconds ?? 0).toFixed(1)}s${v.saved ? '·已保存' : '·临时'}）`,
                })),
              ]}
              notFoundContent={breezeVoicesLoading ? '加载中…' : (breezeVoicesStarted ? '尚无克隆音色' : 'Breeze 服务未运行')}
              dropdownRender={menu => (
                <>
                  {menu}
                  <div style={{ padding: '4px 8px' }}>
                    <Button size="small" type="text" icon={<ReloadOutlined />} loading={breezeVoicesLoading} onClick={() => { void refreshBreezeVoices() }} style={{ fontSize: 11 }}>刷新音色列表</Button>
                  </div>
                </>
              )}
            />
            <Input.TextArea
              size="small"
              value={breezeDesc}
              onChange={e => setBreezeDesc(e.target.value)}
              onBlur={() => { applyVoiceSettings({ breezeVoiceDesc: breezeDesc }) }}
              placeholder="音色描述（voice design）：如「温柔沉稳的青年女声，吐字清晰」"
              autoSize={{ minRows: 2, maxRows: 4 }}
              style={{ marginTop: 8, fontSize: 11 }}
              maxLength={2000}
            />
            <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 4, opacity: 0.75 }}>
              未选克隆音色时按描述合成；朗读情绪（[笑]/[叹气] 等）会并入语气执导
            </div>
          </>
        ) : (
          <Select
            size="small"
            value={effectiveVoice}
            onChange={v => { setTtsVoice(v); applyVoiceSettings({ ttsVoice: v }) }}
            style={{ width: '100%' }}
            options={voiceOptions}
          />
        )}
      </Card>

      {/* Breeze 克隆音色（仅 breeze 引擎时显示）：参考音频+逐字转写 → 保存为可复用音色 */}
      {isBreeze && (
        <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>克隆音色</div>
          {!breezeVoicesStarted && (
            <div style={{ fontSize: 10, color: '#f59e0b', marginBottom: 6 }}> {/* hex-exempt 警示琥珀 */}
              Breeze 服务未运行——先在模型中心启动（克隆需要服务在线）
            </div>
          )}
          <div style={{ display: 'flex', gap: 6, marginBottom: 6 }}>
            <Input
              size="small"
              value={cloneName}
              onChange={e => setCloneName(e.target.value.replace(/[^A-Za-z0-9_-]/g, ''))}
              placeholder="名称（字母/数字/_/-，≤64）"
              style={{ fontSize: 11, flex: 1 }}
              maxLength={64}
            />
            <Input
              size="small"
              value={cloneDesc}
              onChange={e => setCloneDesc(e.target.value)}
              placeholder="备注描述（可选）"
              style={{ fontSize: 11, flex: 1 }}
              maxLength={200}
            />
          </div>
          <Input.TextArea
            size="small"
            value={cloneRefText}
            onChange={e => setCloneRefText(e.target.value)}
            placeholder="参考音频的转写文本——必须与音频逐字一致，否则克隆严重劣化"
            autoSize={{ minRows: 2, maxRows: 3 }}
            style={{ fontSize: 11, marginBottom: 6 }}
            maxLength={1000}
          />
          <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
            <Button size="small" icon={<AudioOutlined />} onClick={() => cloneFileRef.current?.click()} style={{ fontSize: 11 }} disabled={!breezeVoicesStarted}>
              {cloneFileName ? `已选：${cloneFileName.slice(0, 18)}` : '选择参考音频（WAV，5~15s 干净人声）'}
            </Button>
            <input
              ref={cloneFileRef}
              type="file"
              accept="audio/wav,audio/x-wav,.wav"
              style={{ display: 'none' }}
              onChange={e => {
                const f = e.target.files?.[0]
                if (!f) return
                setCloneFileName(f.name)
                const reader = new FileReader()
                reader.onload = () => {
                  const buf = new Uint8Array(reader.result as ArrayBuffer)
                  let bin = ''
                  for (let i = 0; i < buf.length; i += 0x8000) {
                    bin += String.fromCharCode(...buf.subarray(i, i + 0x8000))
                  }
                  setCloneFileB64(btoa(bin))
                }
                reader.readAsArrayBuffer(f)
              }}
            />
            <Button
              size="small"
              type="primary"
              icon={<PlusOutlined />}
              loading={cloneSaving}
              disabled={!breezeVoicesStarted || !cloneName || !cloneRefText || !cloneFileB64}
              style={{ fontSize: 11 }}
              onClick={async () => {
                setCloneSaving(true)
                try {
                  const res = await app.BreezeSaveVoice(cloneName, cloneDesc, cloneRefText, cloneFileB64) as { ok?: boolean; error?: string }
                  if (res?.ok) {
                    message.success(`克隆音色「${cloneName}」已保存`)
                    setCloneName(''); setCloneDesc(''); setCloneRefText(''); setCloneFileB64(''); setCloneFileName('')
                    void refreshBreezeVoices()
                  } else {
                    message.error(res?.error || '保存失败')
                  }
                } catch (err) {
                  message.error(`保存失败：${err instanceof Error ? err.message : String(err)}`)
                } finally {
                  setCloneSaving(false)
                }
              }}
            >
              保存克隆
            </Button>
          </div>
          <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 4, opacity: 0.75 }}>
            保存后在上方「合成音色」下拉选用；服务端写入 voices/&lt;名称&gt;.breeze，跨重启可用
          </div>
        </Card>
      )}

      {/* 角色音色预设（角色库实体→音色）：聊天选中该角色即用其音色。
          breeze 合成按当前人格（=角色 id）解析；描述留空回退性别缺省描述。 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
          <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>角色音色（角色库）</div>
          <Button size="small" type="text" icon={<ReloadOutlined />} loading={personaLoading} onClick={() => { void loadPersonaRows() }} style={{ fontSize: 11 }}>重载</Button>
        </div>
        {personaRows.length === 0 ? (
          <div style={{ fontSize: 10, color: 'var(--color-text-secondary)' }}>
            暂无启用聊天的角色——先在角色库创建并开启「聊天」
          </div>
        ) : (
          <div style={{ maxHeight: 240, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
            {personaRows.map(r => (
              <div key={r.id} style={{ border: '1px solid var(--color-border)', borderRadius: 6, padding: 6 }}>
                <div style={{ fontSize: 11, marginBottom: 4, display: 'flex', alignItems: 'center', gap: 6 }}>
                  {r.label}
                  {r.dirty && <span style={{ fontSize: 10, color: '#f59e0b' }}>保存中…</span>} {/* hex-exempt 保存中琥珀 */}
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  <Select
                    size="small"
                    value={r.voiceId}
                    style={{ flex: 1, minWidth: 0 }}
                    onChange={v => { void patchPersonaVoice(r.id, { voiceId: String(v) }) }}
                    options={[
                      { value: '', label: '自动（按描述）' },
                      ...breezeVoices.map(v => ({ value: v.id, label: v.id })),
                    ]}
                    notFoundContent={breezeVoicesLoading ? '…' : '无克隆音色'}
                  />
                  <Input
                    size="small"
                    value={r.desc}
                    style={{ flex: 1, minWidth: 0, fontSize: 11 }}
                    placeholder={r.gender === 'male' ? '缺省：沉稳清晰的青年男性声线' : '缺省：温柔清晰的青年女性声线'}
                    onBlur={e => {
                      const v = e.target.value
                      if (v !== r.descSaved) void patchPersonaVoice(r.id, { desc: v })
                    }}
                    onPressEnter={e => { (e.target as HTMLInputElement).blur() }}
                    onChange={e => setPersonaRows(prev => prev.map(x => x.id === r.id ? { ...x, desc: e.target.value } : x))}
                    maxLength={300}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
        <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 4, opacity: 0.75 }}>
          选克隆音色或写描述即存即生效（描述回车保存）；聊天里切到该角色，朗读与语音对话即用其音色
        </div>
      </Card>

      {/* 麦克风设备 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>录音设备</div>
        {micDevices.length === 0 ? (
          <div style={{ fontSize: 11, color: 'var(--color-text-secondary)' }}>
            未发现录音设备——插入麦克风后点「检测」刷新
          </div>
        ) : (
          <Select
            size="small"
            value={micId || 'default'}
            onChange={v => {
              const id = v === 'default' ? '' : String(v)
              setMicId(id)
              saveMicDeviceId(id)
              message.success('录音设备已切换，下次语音对话生效')
            }}
            style={{ width: '100%' }}
            options={[
              { value: 'default', label: '跟随系统默认' },
              ...micDevices.map(d => ({ value: d.deviceId, label: d.label })),
            ]}
          />
        )}
        <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginTop: 4, opacity: 0.75 }}>
          默认设备可能是无信号的虚拟音频；语音对话提示「麦克风不可用」时在此换真麦克风
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 8 }}>
          <Button size="small" loading={metering} onClick={() => { void runMicMeter() }} style={{ fontSize: 11 }}>
            试音（3 秒）
          </Button>
          <div style={{ flex: 1, height: 6, borderRadius: 3, background: 'var(--color-surface-container-high)', overflow: 'hidden' }}>
            <div style={{
              width: `${Math.round(meterLevel * 100)}%`, height: '100%', borderRadius: 3,
              background: meterResult === 'silent' ? '#f59e0b' : '#2dd4bf', /* hex-exempt 电平条（琥珀=无信号/青=有信号） */
              transition: 'width 80ms linear',
            }} />
          </div>
          {meterResult === 'signal' && (
            <span style={{ fontSize: 10, color: '#2dd4bf' }}>有信号 ✓</span> /* hex-exempt 同上 */
          )}
          {meterResult === 'silent' && (
            <span style={{ fontSize: 10, color: '#f59e0b' }}>无信号 ✗（虚拟设备/静音/未佩戴）</span> /* hex-exempt 同上 */
          )}
        </div>
      </Card>

      {/* 语音模式 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 8 }}>输入模式</div>
        <div style={{ display: 'flex', gap: 8 }}>
          <Tag.CheckableTag
            checked={voiceMode === 'vad'}
            onChange={() => { setVoiceMode('vad'); applyVoiceSettings({ voiceMode: 'vad' }) }}
            style={{ fontSize: 12, padding: '4px 12px', borderRadius: 6 }}
          >
            <AudioOutlined style={{ marginRight: 4 }} /> 自动检测 (VAD)
          </Tag.CheckableTag>
          <Tag.CheckableTag
            checked={voiceMode === 'ptt'}
            onChange={() => { setVoiceMode('ptt'); applyVoiceSettings({ voiceMode: 'ptt' }) }}
            style={{ fontSize: 12, padding: '4px 12px', borderRadius: 6 }}
          >
            <AudioOutlined /> 按住说话 (PTT)
          </Tag.CheckableTag>
        </div>
      </Card>

      {/* 打断阈值 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 4 }}>
          打断阈值: {interruptMs}ms
        </div>
        <Slider
          min={100} max={2000} step={100}
          value={interruptMs}
          onChange={v => setInterruptMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ interruptThresholdMs: v })}
          styles={{ track: { background: '#e85388' } }} // hex-exempt 品牌识别色（阈值轨道）
        />
      </Card>

      {/* 静默阈值 */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 4 }}>
          静默阈值: {silenceMs}ms
        </div>
        <Slider
          min={200} max={3000} step={100}
          value={silenceMs}
          onChange={v => setSilenceMs(v as number)}
          onAfterChange={v => applyVoiceSettings({ silenceThresholdMs: v })}
          styles={{ track: { background: '#a855f7' } }} // hex-exempt 品牌识别色（阈值轨道）
        />
      </Card>
      {/* 实时语音 Realtime 档（v4.8.1 S1） */}
      <Card size="small" style={{ background: 'var(--color-surface-container)', border: '1px solid var(--color-border)', borderRadius: 8 }} bodyStyle={{ padding: 12 }}>
        <div style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 6 }}>实时语音（实验）</div>
        <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', marginBottom: 8, opacity: 0.75 }}>
          端到端语音档：OpenAI Realtime API（需 API Key）。未配置时走现拼接管线，互不影响。
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <Select
            size="small"
            value={rtProvider || 'off'}
            onChange={v => setRtProvider(v === 'off' ? '' : v)}
            style={{ width: '100%' }}
            options={[
              { value: 'off', label: '关闭（走现拼接管线）' },
              { value: 'openai', label: 'OpenAI Realtime' },
            ]}
          />
          {rtProvider === 'openai' && (
            <>
              <Input
                size="small"
                placeholder="模型（如 gpt-realtime）"
                value={rtModel}
                onChange={e => setRtModel(e.target.value)}
              />
              <Input.Password
                size="small"
                placeholder={rtHasKey ? '已配置 Key（输入新值可更换）' : 'API Key（sk-…）'}
                value={rtKey}
                onChange={e => setRtKey(e.target.value)}
                autoComplete="new-password"
              />
              <div style={{ fontSize: 10, color: 'var(--color-text-secondary)', opacity: 0.75 }}>
                Key 经 Windows DPAPI 加密落盘，仅本机可解；界面不回显明文。
              </div>
            </>
          )}
          <Button size="small" type="primary" loading={rtSaving} onClick={() => void saveRealtime()}>
            保存
          </Button>
        </div>
      </Card>
    </div>
  )
}
