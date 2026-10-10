// StoryStream.test.tsx — 故事流渲染：生成速度段（tok/s）只在数值有效时渲染，
// 流式行与缺省字段一律不渲染（不虚报速度）。
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { StoryStream } from './StoryStream'
import type { SinMessageView } from './types'

function msg(over: Partial<SinMessageView> = {}): SinMessageView {
  return {
    key: 'm1',
    messageId: 1,
    role: 'assistant',
    content: '夜雨敲窗。',
    illustrations: {},
    ...over,
  }
}

function renderStream(messages: SinMessageView[]) {
  return render(
    <StoryStream
      storyId="s1"
      messages={messages}
      onIllustrationGenerated={() => {}}
      onIllustrationError={() => {}}
    />,
  )
}

describe('StoryStream 生成速度段', () => {
  it('tokensPerSecond 有效：正文后渲染「{n} tok/s」', () => {
    const { container } = renderStream([msg({ tokensPerSecond: 18.64 })])
    const line = container.querySelector('.sin-tokspeed')
    expect(line).not.toBeNull()
    expect(line?.textContent).toBe('18.6 tok/s')
  })

  it('缺省/0/流式中：不渲染（不虚报）', () => {
    for (const m of [msg(), msg({ tokensPerSecond: 0 }), msg({ tokensPerSecond: 12.3, streaming: true })]) {
      const { container } = renderStream([m])
      expect(container.querySelector('.sin-tokspeed')).toBeNull()
    }
  })
})

describe('StoryStream 空态引导', () => {
  it('默认文案点名图文交织（板块招牌，改文案防回退）', () => {
    const { getByText } = renderStream([])
    const hint = getByText(/还没有开始写/)
    expect(hint.textContent).toContain('图文交织')
  })
})
