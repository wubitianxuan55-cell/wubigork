import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { playVoicePreview, playCharacterVoiceLine, stopVoicePreview, genderDefaultVoiceDesc, resolveVoiceLine } from './voicePreview'

// antd message 只桩用到的两嘴（断言试听失败的透出文案）；vi.hoisted 防 mock
// 工厂提升后引用未初始化变量
const { messageError, messageSuccess } = vi.hoisted(() => ({
  messageError: vi.fn(),
  messageSuccess: vi.fn(),
}))
vi.mock('antd', () => ({
  message: { error: messageError, success: messageSuccess, warning: vi.fn() },
}))

// jsdom 无媒体后端：Audio 换成可断言的假件（src/play/pause/onended 面）
const audioInstances: Array<{ src: string; play: ReturnType<typeof vi.fn>; pause: ReturnType<typeof vi.fn>; onended: (() => void) | null }> = []
class MockAudio {
  src = ''
  play = vi.fn(async () => {})
  pause = vi.fn()
  onended: (() => void) | null = null
  constructor(src: string) {
    this.src = src
    audioInstances.push(this)
  }
}

const previewBinding = vi.fn()
const lineAudioBinding = vi.fn()

beforeEach(() => {
  audioInstances.length = 0
  previewBinding.mockReset()
  previewBinding.mockResolvedValue({ base64: 'QUJD', mimeType: 'audio/wav' })
  lineAudioBinding.mockReset()
  lineAudioBinding.mockResolvedValue({ audio: 'QUJD', mimeType: 'audio/wav', cached: false, path: 'voices/x.wav' })
  messageError.mockReset()
  messageSuccess.mockReset()
  vi.stubGlobal('Audio', MockAudio)
  // 真壳同款 VoiceB 门面（bridge realApp 按方法名路由）
  ;(window as unknown as { go?: unknown }).go = {
    app: { VoiceB: { TTSVoicePreviewBase64: previewBinding, TTSLineAudioFor: lineAudioBinding } },
  }
})

afterEach(() => {
  stopVoicePreview()
  vi.unstubAllGlobals()
  delete (window as unknown as { go?: unknown }).go
})

describe('voicePreview（角色档案试听段播放器）', () => {
  it('genderDefaultVoiceDesc 与后端聊天链性别兜底同口径', () => {
    expect(genderDefaultVoiceDesc('male')).toBe('沉稳清晰的青年男性声线')
    expect(genderDefaultVoiceDesc('female')).toBe('温柔清晰的青年女性声线')
    expect(genderDefaultVoiceDesc('neutral')).toBe('')
    expect(genderDefaultVoiceDesc(undefined)).toBe('')
  })

  it('resolveVoiceLine 三级口径：专属台词 > 对话样本首条 > 姓名问候', () => {
    expect(resolveVoiceLine({ name: '苏念', ttsLine: '剑来。', dialogueSamples: ['别挡路。'] })).toBe('剑来。')
    expect(resolveVoiceLine({ name: '苏念', ttsLine: '', dialogueSamples: ['别挡路。', '让开'] })).toBe('别挡路。')
    expect(resolveVoiceLine({ name: '苏念', ttsLine: '', dialogueSamples: ['  ', ''] })).toContain('苏念')
    expect(resolveVoiceLine({ name: '', ttsLine: '', dialogueSamples: [] })).toBe('你好，我是「…」，很高兴认识你。')
  })

  it('playVoicePreview：显式音色/描述透传绑定，data URL 播放返回 true', async () => {
    const ok = await playVoicePreview('你好', 'clone-1', '轻快语气')
    expect(ok).toBe(true)
    expect(previewBinding).toHaveBeenCalledWith('你好', 'clone-1', '轻快语气')
    expect(audioInstances[0].src).toBe('data:audio/wav;base64,QUJD')
    expect(audioInstances[0].play).toHaveBeenCalledTimes(1)
  })

  it('连播两段：前一段被停（pause+置空），不叠音', async () => {
    await playVoicePreview('第一段', 'v', 'd')
    await playVoicePreview('第二段', 'v', 'd')
    expect(audioInstances[0].pause).toHaveBeenCalledTimes(1)
    expect(audioInstances[1].play).toHaveBeenCalledTimes(1)
  })

  it('合成失败：透出「试听失败」文案并返回 false（不播放）', async () => {
    previewBinding.mockRejectedValue(new Error('HTTP 500'))
    const ok = await playVoicePreview('你好', '', 'desc')
    expect(ok).toBe(false)
    expect(messageError).toHaveBeenCalledWith(expect.stringContaining('试听失败'))
    expect(messageError).toHaveBeenCalledWith(expect.stringContaining('HTTP 500'))
    expect(audioInstances).toHaveLength(0)
  })

  it('空 base64：按失败处理（不编造音频）', async () => {
    previewBinding.mockResolvedValue({ base64: '', mimeType: '' })
    const ok = await playVoicePreview('你好', '', 'desc')
    expect(ok).toBe(false)
    expect(messageError).toHaveBeenCalled()
  })

  // ── 台词语音文件缓存链（playCharacterVoiceLine）──────────────────────

  it('playCharacterVoiceLine：角色 JSON 透传 TTSLineAudioFor，命中静默播放', async () => {
    lineAudioBinding.mockResolvedValue({ audio: 'QUJD', mimeType: 'audio/wav', cached: true })
    const ok = await playCharacterVoiceLine({ id: 'c1', name: '苏念', ttsLine: '剑来。', gender: 'female' })
    expect(ok).toBe(true)
    const payload = JSON.parse(lineAudioBinding.mock.calls[0][0] as string)
    expect(payload).toEqual({
      id: 'c1', name: '苏念', ttsLine: '剑来。', ttsVoiceId: '', ttsDesc: '',
      gender: 'female', dialogueSamples: [],
    })
    expect(audioInstances[0].src).toBe('data:audio/wav;base64,QUJD')
    expect(messageSuccess).not.toHaveBeenCalled() // 命中=秒开，不打扰
  })

  it('未命中（cached=false）：播放 + 提示「语音文件已生成」', async () => {
    const ok = await playCharacterVoiceLine({ id: 'c1', name: '苏念' })
    expect(ok).toBe(true)
    expect(messageSuccess).toHaveBeenCalledWith(expect.stringContaining('语音文件已生成'))
  })

  it('空音频回包：按失败处理', async () => {
    lineAudioBinding.mockResolvedValue({ audio: '', mimeType: '', cached: false })
    const ok = await playCharacterVoiceLine({ id: 'c1' })
    expect(ok).toBe(false)
    expect(messageError).toHaveBeenCalledWith(expect.stringContaining('试听失败'))
  })

  it('连播两段台词：前一段被停不叠音', async () => {
    await playCharacterVoiceLine({ id: 'c1', name: 'A' })
    await playCharacterVoiceLine({ id: 'c2', name: 'B' })
    expect(audioInstances[0].pause).toHaveBeenCalledTimes(1)
    expect(lineAudioBinding).toHaveBeenCalledTimes(2)
  })
})
