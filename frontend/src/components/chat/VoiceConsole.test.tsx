import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { VoiceConsole, type VoiceConsoleProps } from './VoiceConsole'
import type { VoiceChatState } from '../../hooks/useVoiceChat'

vi.mock('../VoiceChatOrb', () => ({ default: () => <div data-testid="voice-chat-orb" /> }))

const voiceState = (over: Partial<VoiceChatState> = {}): VoiceChatState => ({
  active: true,
  listening: true,
  speaking: false,
  aiSpeaking: false,
  transcript: '帮我整理今天的会议记录',
  finalTranscript: '',
  volume: 0.3,
  error: null,
  mode: 'vad',
  degraded: null,
  ...over,
})

function props(over: Partial<VoiceConsoleProps> = {}): VoiceConsoleProps {
  return {
    voice: voiceState(),
    voiceOn: true,
    messages: [
      { key: 'u1', role: 'user', content: '帮我整理会议记录', createdAt: '' },
      { key: 'a1', role: 'assistant', content: '我会按议题、结论和行动项整理。', createdAt: '' },
    ],
    companionName: '盖亚',
    onToggleVoice: vi.fn(),
    onInterruptVoice: vi.fn(),
    onOpenSettings: vi.fn(),
    onClose: vi.fn(),
    ...over,
  }
}

afterEach(cleanup)

describe('VoiceConsole 专用语音交互界面', () => {
  it('显示语音状态、实时转写、最近回答和基础控制', () => {
    render(<VoiceConsole {...props()} />)
    expect(screen.getByTestId('chat-voice-console')).toBeTruthy()
    expect(screen.getByText('与盖亚语音对话')).toBeTruthy()
    expect(screen.getByText('正在聆听')).toBeTruthy()
    expect(screen.getByText('帮我整理今天的会议记录')).toBeTruthy()
    expect(screen.getByText('我会按议题、结论和行动项整理。')).toBeTruthy()
    expect(screen.getByRole('button', { name: '结束语音' })).toBeTruthy()
  })

  it('AI 回答时可以打断，并保留返回文字对话入口', () => {
    const onInterruptVoice = vi.fn()
    const onClose = vi.fn()
    render(<VoiceConsole {...props({ voice: voiceState({ listening: false, aiSpeaking: true }), onInterruptVoice, onClose })} />)
    fireEvent.click(screen.getByRole('button', { name: '打断回答' }))
    expect(onInterruptVoice).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: '返回文字对话' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('麦克风降级不伪装聆听，并提供明确恢复信息', () => {
    render(<VoiceConsole {...props({ voice: voiceState({ listening: false, degraded: 'mic-unavailable' }) })} />)
    expect(screen.getByText('麦克风不可用')).toBeTruthy()
    expect(screen.getByRole('alert').textContent).toContain('未在采集声音')
    expect(screen.queryByText('正在聆听')).toBeNull()
  })

  it('暂停后可以从界面重新开始聆听', async () => {
    const onToggleVoice = vi.fn().mockResolvedValue(undefined)
    render(<VoiceConsole {...props({ voiceOn: false, voice: voiceState({ active: false, listening: false }), onToggleVoice })} />)
    fireEvent.click(screen.getByRole('button', { name: '开始聆听' }))
    await waitFor(() => expect(onToggleVoice).toHaveBeenCalledTimes(1))
  })
})
