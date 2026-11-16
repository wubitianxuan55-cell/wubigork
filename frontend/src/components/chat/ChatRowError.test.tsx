import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { ChatRow } from './ChatRow'
import type { ChatMsg } from '../../pages/chat/types'

// B1 错误行动作测试：结构化错误码 → 「打开模型中心」+「重试」；无码 → 仅「重试」。
function errorRow(extra?: Record<string, unknown>): ChatMsg {
  return {
    key: 'kErr', role: 'assistant', content: '引擎未连接：请检查本机引擎状态后重试',
    createdAt: '2026-01-01T00:00:00Z', error: true, ...(extra ? { extra } : {}),
  }
}

const baseProps = {
  text: '',
  isStreaming: false,
  newGroup: false,
  mode: 'plain',
  companionName: '伴侣',
  copied: false,
  speaking: false,
  onCopy: vi.fn(),
  onSpeak: vi.fn(),
  onRetry: vi.fn(),
}

describe('ChatRow 错误行动作（B1）', () => {
  it('engine_unreachable：出「打开模型中心」+ 重试，原文降级为详情小字', () => {
    render(
      <ChatRow
        {...baseProps}
        msg={errorRow({ error_code: 'engine_unreachable', error_detail: 'dial tcp: connection refused' })}
        text="引擎未连接：请检查本机引擎状态后重试"
      />,
    )
    expect(screen.getByText('打开模型中心')).toBeTruthy()
    expect(screen.getByText('dial tcp: connection refused')).toBeTruthy()
    fireEvent.click(screen.getByText('重试'))
    expect(baseProps.onRetry).toHaveBeenCalledWith('kErr')
  })

  it('无错误码（旧消息形态）：仅重试按钮，零新增元素', () => {
    render(<ChatRow {...baseProps} msg={errorRow()} text="请求失败：怪错原文" />)
    expect(screen.queryByText('打开模型中心')).toBeNull()
    expect(screen.getByText('重试')).toBeTruthy()
  })
})
