// FE6-02（复审补齐）：语音球的顶部状态文案在麦克风不可用时不得宣称「准备聆听/
// 正在聆听」——否则语音主视觉继续宣称在听，与 WelcomeScreen/命令条的警示条自相矛盾。
// jsdom 无 canvas 2D：用 Proxy 造最小上下文（只要求不崩，断言的是文案）。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import VoiceChatOrb from './VoiceChatOrb'

function installCanvasStub() {
  const gradient = { addColorStop: () => {} }
  const ctx = new Proxy({} as Record<string, unknown>, {
    get(_t, key) {
      if (key === 'createRadialGradient' || key === 'createLinearGradient') return () => gradient
      return () => {}
    },
    set() { return true },
  })
  const original = HTMLCanvasElement.prototype.getContext
  HTMLCanvasElement.prototype.getContext = (() => ctx) as unknown as HTMLCanvasElement['getContext']
  return () => { HTMLCanvasElement.prototype.getContext = original }
}

let restore: (() => void) | null = null

afterEach(() => {
  cleanup()
  restore?.()
  restore = null
  vi.restoreAllMocks()
})

describe('VoiceChatOrb 麦克风降级文案（FE6-02 复审）', () => {
  it('micUnavailable → 显示「麦克风不可用」，不出现聆听文案', () => {
    restore = installCanvasStub()
    render(<VoiceChatOrb volume={0} listening={false} speaking={false} aiSpeaking={false} transcript="" size={120} micUnavailable />)
    expect(screen.getByText('麦克风不可用')).toBeTruthy()
    expect(screen.queryByText('准备聆听')).toBeNull()
    expect(screen.queryByText('正在聆听...')).toBeNull()
  })

  it('micUnavailable 缺省（未传）→ 既有口径零变化：listening 时仍是「准备聆听」', () => {
    restore = installCanvasStub()
    render(<VoiceChatOrb volume={0} listening speaking={false} aiSpeaking={false} transcript="" size={120} />)
    expect(screen.getByText('准备聆听')).toBeTruthy()
    expect(screen.queryByText('麦克风不可用')).toBeNull()
  })
})
