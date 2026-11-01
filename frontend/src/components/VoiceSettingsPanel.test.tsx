import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactElement } from 'react'
import { LocaleProvider } from '../gaea/lib/i18n'
import VoiceSettingsPanel from './VoiceSettingsPanel'

// EventsOn：面板订阅 tts-service-status（jsdom 无 wails 运行时，mock 成空订阅）
vi.mock('../../wailsjs/runtime/runtime', () => ({ EventsOn: vi.fn(() => () => {}) }))

const bridgeMocks = vi.hoisted(() => ({
  app: {
    VoiceHealth: vi.fn(async () => ({ asrReady: false, ttsReady: true, state: 'idle', browserAsr: true, ttsDetail: '', asrDetail: '' })),
    VoiceGetSettings: vi.fn(async () => ({
      ttsEnabled: true, ttsVoice: '', voiceMode: 'vad',
      interruptThresholdMs: 500, silenceThresholdMs: 1000,
      realtimeProvider: '', realtimeModel: '', realtimeHasKey: false, breezeVoiceDesc: '',
    })),
    VoiceApplySettings: vi.fn(async () => undefined),
    GetVoicePipelineConfig: vi.fn(async () => ({
      chatTts: { engine: '', model: '' },
      tts: { engine: 'breeze', model: 'breeze-tts-2' },
    })),
    GetTTSSpeakers: vi.fn(async () => []),
    BreezeListVoices: vi.fn(async () => ({ started: true, voices: [{ id: 'char_1' }] })),
    BreezeSaveVoice: vi.fn(async () => ({ ok: true })),
    SetChatVoiceModel: vi.fn(async () => undefined),
    StartLocalTTSService: vi.fn(async () => ({ ready: false, starting: true })),
    VoiceV2Status: vi.fn(async () => ({ sidecar: false, llm: false, breeze: true, ready: false })),
    VoiceV2Ensure: vi.fn(async () => ({ ready: false, starting: true })),
    VoiceV2Stop: vi.fn(async () => ({ sidecar: false, llm: false })),
    CharacterList: vi.fn(async () => ({ items: [] as Array<Record<string, unknown>> })),
    CharacterGet: vi.fn(async () => ({ character: {} })),
    CharacterSave: vi.fn(async () => undefined),
  },
}))

vi.mock('../gaea/lib/bridge', () => ({ app: bridgeMocks.app }))

const wrap = (ui: ReactElement) => {
  localStorage.setItem('gaea-lang', 'zh')
  return <LocaleProvider>{ui}</LocaleProvider>
}

describe('VoiceSettingsPanel 真实路由对齐（2026-11-01 重排）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })
  afterEach(cleanup)

  it('显示三级真实路由：聊天绑定（自定义档）→全局→兜底', async () => {
    bridgeMocks.app.GetVoicePipelineConfig.mockResolvedValue({
      chatTts: { engine: 'xai', model: 'grok-tts' },
      tts: { engine: 'breeze', model: 'breeze-tts-2' },
    })
    render(wrap(<VoiceSettingsPanel />))

    await waitFor(() => expect(screen.getByText('朗读引擎')).toBeTruthy())
    expect(screen.getByText('① 聊天绑定：xai/grok-tts')).toBeTruthy()
    expect(screen.getByText('② 全局 TTS：breeze/breeze-tts-2')).toBeTruthy()
    expect(screen.getByText('③ 兜底扫描：本地引擎优先，Edge 云端保底')).toBeTruthy()
    // 自定义绑定只读档如实显示
    expect(screen.getByText('自定义绑定：xai/grok-tts——在 模型中心·功能绑定 修改')).toBeTruthy()
  })

  it('点「本地克隆」档 → SetChatVoiceModel 绑定 breeze（模型取全局 breeze 模型）', async () => {
    render(wrap(<VoiceSettingsPanel />))
    await waitFor(() => expect(screen.getByText('朗读引擎')).toBeTruthy())

    fireEvent.click(screen.getByText('本地克隆（breeze）——朗读按角色库音色出声'))

    await waitFor(() => expect(bridgeMocks.app.SetChatVoiceModel).toHaveBeenCalledWith('breeze', 'breeze-tts-2'))
  })

  it('当前绑定 breeze 时点「跟随全局」→ 清除绑定回退全局 TTS', async () => {
    bridgeMocks.app.GetVoicePipelineConfig.mockResolvedValue({
      chatTts: { engine: 'breeze', model: 'breeze-tts-2' },
      tts: { engine: 'edge', model: 'edge-tts' },
    })
    render(wrap(<VoiceSettingsPanel />))
    await waitFor(() => expect(screen.getByText('① 聊天绑定：breeze/breeze-tts-2')).toBeTruthy())

    fireEvent.click(screen.getByText('跟随全局 TTS（edge/edge-tts）'))

    await waitFor(() => expect(bridgeMocks.app.SetChatVoiceModel).toHaveBeenCalledWith('', ''))
  })

  it('健康卡诚实化：ttsReady=false 显示红点+detail，识别标注浏览器路径', async () => {
    bridgeMocks.app.VoiceHealth.mockResolvedValue({
      asrReady: false, browserAsr: true, ttsReady: false, state: 'idle', asrDetail: '',
      ttsDetail: '聊天语音绑定：breeze/breeze-tts-2（本地服务离线，朗读时自动拉起）',
    })
    render(wrap(<VoiceSettingsPanel />))

    await waitFor(() => expect(screen.getByText('语音合成', { exact: false })).toBeTruthy())
    expect(screen.getByText('语音识别（浏览器）')).toBeTruthy()
    expect(screen.getByText(/本地服务离线，朗读时自动拉起/)).toBeTruthy()
  })

  it('breeze 离线 → 朗读引擎卡出「现在拉起」钮并调 StartLocalTTSService', async () => {
    bridgeMocks.app.BreezeListVoices.mockResolvedValue({ started: false, voices: [] })
    render(wrap(<VoiceSettingsPanel />))

    const btn = await screen.findByText('现在拉起 Breeze')
    fireEvent.click(btn)
    await waitFor(() => expect(bridgeMocks.app.StartLocalTTSService).toHaveBeenCalledWith('breeze'))
  })

  it('角色行 voiceId 不在克隆列表 → 「音色失效」警示', async () => {
    bridgeMocks.app.CharacterList.mockResolvedValue({
      items: [{ id: 'tsundere', name: '傲娇', gender: 'female', ttsVoiceId: 'v1', ttsDesc: '' }],
    })
    render(wrap(<VoiceSettingsPanel />))

    await waitFor(() => expect(screen.getByText('傲娇')).toBeTruthy())
    expect(screen.getByText('音色失效')).toBeTruthy()
  })
})


describe('VoiceSettingsPanel 语音对话引擎卡（M4 产品接入刀）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })
  afterEach(cleanup)

  it('默认 v1；切 V2 调 VoiceApplySettings 并显示依赖链拉起', async () => {
    render(wrap(<VoiceSettingsPanel />))
    await waitFor(() => expect(screen.getByTestId('voice-backend-card')).toBeTruthy())
    expect(screen.queryByTestId('voice-v2-ensure')).toBeNull()
    const radio = screen.getByTestId('voice-backend-radio')
    const v2btn = radio.querySelectorAll('label')[1]
    fireEvent.click(v2btn)
    await waitFor(() =>
      expect(bridgeMocks.app.VoiceApplySettings).toHaveBeenCalledWith({ voiceBackend: 'v2' }))
    await waitFor(() => expect(screen.getByTestId('voice-v2-ensure')).toBeTruthy())
    expect(screen.getByText(/breeze ✓ · 4B ✗ · sidecar ✗/)).toBeTruthy()
  })
})
