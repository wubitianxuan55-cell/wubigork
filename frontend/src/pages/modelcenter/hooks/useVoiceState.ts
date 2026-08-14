/**
 * useVoiceState — 模型中心「语音模型/功能绑定语音」状态 Hook（T6-6.4 UI 拆分）
 *
 * 归集语音管道三段激活模型（STT/LLM/TTS）、OCR 激活模型与功能绑定
 * 「聊天语音」的全部状态：配置、草稿、保存中标记、服务端音色列表，
 * 以及 xAI 固定音色选项（XAI_VOICES 单一来源自 ../utils）。
 */
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { message } from 'antd'
import * as App from '../../../wailsjsCompat'
import { setActiveOCRModel, getActiveOCRModel } from '../../../api/engines'
import { XAI_VOICES, localTTSDefaultVoice, localTTSFallbackVoices } from '../utils'
import { type VoiceCfg } from '../context'

export interface VoiceState {
  voiceCfg: VoiceCfg
  setVoiceCfg: Dispatch<SetStateAction<VoiceCfg>>
  ocrCfg: { engine: string; model: string }
  setOcrCfg: Dispatch<SetStateAction<{ engine: string; model: string }>>
  chatVoiceCfg: { engine: string; model: string }
  chatVoiceDraft: { engine: string; model: string }
  setChatVoiceDraft: Dispatch<SetStateAction<{ engine: string; model: string }>>
  chatVoiceSaving: boolean
  chatVoiceSpeakers: string[]
  chatVoiceOptions: { value: string; label: string }[]
  chatVoiceValue?: string
  loadVoiceCfg: () => Promise<void>
  handleSetVoiceModel: (kind: 'asr' | 'tts', engineId: string, modelId: string) => Promise<void>
  handleSetOCRModel: (engineId: string, modelId: string) => Promise<void>
  handleSaveChatVoice: () => Promise<void>
  handleClearChatVoice: () => Promise<void>
}

export function useVoiceState(): VoiceState {
  // 语音管道三段激活模型（STT/LLM/TTS，来自模型中心选择）
  const [voiceCfg, setVoiceCfg] = useState<VoiceCfg>({ stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '', voice: '' } })
  const [ocrCfg, setOcrCfg] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  // 功能绑定：聊天语音合成（优先于全局 TTS，便于后续扩展更多语音绑定）
  const [chatVoiceCfg, setChatVoiceCfg] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceDraft, setChatVoiceDraft] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceSaving, setChatVoiceSaving] = useState(false)
  const [chatVoiceSpeakers, setChatVoiceSpeakers] = useState<string[]>([])

  // 加载语音管道三段激活模型
  const loadVoiceCfg = useCallback(async () => {
    try {
      const cfg = await App.GetVoicePipelineConfig()
      if (cfg) {
        setVoiceCfg({
          stt: { engine: cfg.stt?.engine || '', model: cfg.stt?.model || '' },
          llm: { engine: cfg.llm?.engine || '', model: cfg.llm?.model || '' },
          tts: { engine: cfg.tts?.engine || '', model: cfg.tts?.model || '', voice: cfg.tts?.voice || '' },
        })
        setChatVoiceCfg({ engine: cfg.chatTts?.engine || '', model: cfg.chatTts?.model || '' })
        setChatVoiceDraft({ engine: cfg.chatTts?.engine || '', model: cfg.chatTts?.model || '' })
      }
    } catch (_) {}
  }, [])

  const loadOCRCfg = useCallback(async () => {
    try {
      const cfg = await getActiveOCRModel()
      if (cfg) setOcrCfg({ engine: cfg.engine || '', model: cfg.model || '' })
    } catch (_) {}
  }, [])

  useEffect(() => { void loadVoiceCfg(); void loadOCRCfg() }, [loadVoiceCfg, loadOCRCfg])

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

  const handleSetOCRModel = async (engineId: string, modelId: string) => {
    try {
      await setActiveOCRModel(engineId, modelId)
      message.success(engineId && modelId ? `已设为 OCR：${modelId}` : 'OCR 已恢复自动选择')
      loadOCRCfg()
    } catch (err: any) {
      message.error(err?.message || '设置 OCR 失败')
    }
  }

  // 保存功能绑定「聊天语音」（功能绑定 → 语音管道，空=清除绑定回退全局 TTS）
  const handleSaveChatVoice = async () => {
    const d = chatVoiceDraft
    if (!d.engine || !d.model) {
      message.warning('请选择引擎和语音模型（清除绑定请用右侧「清除」按钮）')
      return
    }
    setChatVoiceSaving(true)
    try {
      await App.SetChatVoiceModel(d.engine, d.model)
      message.success(`聊天语音已绑定：${d.model}`)
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '绑定失败')
    }
    setChatVoiceSaving(false)
  }

  // 清除功能绑定「聊天语音」
  const handleClearChatVoice = async () => {
    setChatVoiceSaving(true)
    try {
      await App.SetChatVoiceModel('', '')
      message.success('已清除聊天语音绑定（回退全局 TTS）')
      setChatVoiceDraft({ engine: '', model: '' })
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '清除失败')
    }
    setChatVoiceSaving(false)
  }

  // 聊天语音绑定卡：非 xAI 引擎时拉取服务端音色列表
  useEffect(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model || engine === 'xai') {
      setChatVoiceSpeakers([])
      return
    }
    ;(App as any).GetTTSSpeakers?.(model)
      .then((sp: any) => setChatVoiceSpeakers(Array.isArray(sp) ? sp : []))
      .catch(() => setChatVoiceSpeakers([]))
  }, [chatVoiceDraft])

  // 聊天语音绑定卡的音色选项：xAI → 固定列表（XAI_VOICES 单一来源）；其他 → 服务端列表 / 兜底
  const chatVoiceOptions = useMemo(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model) return []
    if (engine === 'xai') return XAI_VOICES
    const list = chatVoiceSpeakers.length > 0
      ? chatVoiceSpeakers
      : localTTSFallbackVoices(model)
    return list.map(v => ({ value: v, label: v }))
  }, [chatVoiceDraft, chatVoiceSpeakers])

  const chatVoiceValue = useMemo(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model) return undefined
    const cur = voiceCfg.tts.voice || ''
    if (engine === 'xai') return XAI_VOICES.some(v => v.value === cur) ? cur : 'eve'
    const list = chatVoiceSpeakers.length > 0
      ? chatVoiceSpeakers
      : localTTSFallbackVoices(model)
    if (list.includes(cur)) return cur
    return localTTSDefaultVoice(model)
  }, [chatVoiceDraft, chatVoiceSpeakers, voiceCfg.tts.voice])

  return {
    voiceCfg, setVoiceCfg,
    ocrCfg, setOcrCfg,
    chatVoiceCfg,
    chatVoiceDraft, setChatVoiceDraft,
    chatVoiceSaving,
    chatVoiceSpeakers,
    chatVoiceOptions,
    chatVoiceValue,
    loadVoiceCfg,
    handleSetVoiceModel,
    handleSetOCRModel,
    handleSaveChatVoice,
    handleClearChatVoice,
  }
}
