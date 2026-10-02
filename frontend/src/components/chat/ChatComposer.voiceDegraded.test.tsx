// FE6-02 收口（第 4 个、也是最后一个真实消费面）：输入岛不得在麦克风不可用时
// 写「正在聆听…请说话」——用户会对着没插好的麦克风说完整段话（且此前输入框
// disabled，连字都打不了）。降级时占位符改为「麦克风不可用，请输入文字」并放行
// 文字输入；voiceDegraded 缺省 false ⇒ 既有调用方行为逐字零变化。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { ChatComposer, type ChatComposerProps } from './ChatComposer'

function props(over: Partial<ChatComposerProps> = {}): ChatComposerProps {
  return {
    mode: 'plain',
    input: '',
    onInputChange: vi.fn(),
    onKeyDown: vi.fn(),
    inputRef: { current: null } as React.RefObject<React.ComponentRef<typeof import('antd').Input.TextArea>>,
    voiceOn: true,
    voiceTranscript: '',
    onToggleVoice: vi.fn(),
    sending: false,
    forceSearch: false,
    onToggleForceSearch: vi.fn(),
    thinking: false,
    onToggleThinking: vi.fn(),
    onSend: vi.fn(),
    onStop: vi.fn(),
    onFillInput: vi.fn(),
    ...over,
  }
}

function placeholderOf(): string {
  return (screen.getByRole('textbox') as HTMLTextAreaElement).getAttribute('placeholder') ?? ''
}

afterEach(cleanup)

describe('ChatComposer 语音降级占位符（FE6-02 收口）', () => {
  it('voiceDegraded=true + voiceOn → 占位符为「麦克风不可用，请输入文字」，绝不出现「正在聆听…请说话」', () => {
    render(<ChatComposer {...props({ voiceDegraded: true })} />)
    expect(placeholderOf()).toBe('麦克风不可用，请输入文字')
    expect(placeholderOf()).not.toContain('正在聆听')
    expect(screen.getByLabelText('结束语音')).toBeTruthy()
    expect(screen.queryByTitle('结束聆听')).toBeNull()
  })

  it('降级时必须能打字（此前 voiceOn 直接把输入框锁死，提示「请输入文字」是空话）', () => {
    const onInputChange = vi.fn()
    render(<ChatComposer {...props({ voiceDegraded: true, onInputChange })} />)
    const ta = screen.getByRole('textbox') as HTMLTextAreaElement
    expect(ta.disabled).toBe(false)
    // 受控组件：值由父级 input 决定，这里断言 onChange 通路打通（未被 disabled 吞掉）
    fireEvent.change(ta, { target: { value: '改用文字' } })
    expect(onInputChange).toHaveBeenCalledWith('改用文字')
  })

  it('未降级 + voiceOn → 既有口径零变化（仍是「正在聆听…请说话」且不可编辑）', () => {
    render(<ChatComposer {...props()} />)
    expect(placeholderOf()).toBe('正在聆听…请说话')
    expect((screen.getByRole('textbox') as HTMLTextAreaElement).disabled).toBe(true)
    expect(screen.getByLabelText('结束聆听')).toBeTruthy()
  })

  it('未降级 + 未开语音 → 仍是「输入消息…」', () => {
    render(<ChatComposer {...props({ voiceOn: false })} />)
    expect(placeholderOf()).toBe('输入消息…')
    expect((screen.getByRole('textbox') as HTMLTextAreaElement).disabled).toBe(false)
  })
})
