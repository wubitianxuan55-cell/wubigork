// NewCharactersModal.test.tsx — 新角色发现弹窗（T6-7.5 自订阅事件）。
// v4.421.0：订阅/退订改走 gaea/lib/wailsEvents.subscribeWailsEvent（EventsOn 返回的
// 「只摘除自己」清理函数）——用例锁定「退订只解自己那一个监听、绝不 EventsOff 全清」
// 与既有事件→弹窗→保存链路零变化。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import React from 'react'

const mocks = vi.hoisted(() => ({
  SaveCharactersBatch: vi.fn().mockResolvedValue({}),
  associateToProject: vi.fn().mockResolvedValue(undefined),
  syncProjectCharacters: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../../../gaea/lib/bridge', () => ({ app: { SaveCharactersBatch: mocks.SaveCharactersBatch } }))
vi.mock('../../../api/characterlib', () => ({
  associateToProject: mocks.associateToProject,
  syncProjectCharacters: mocks.syncProjectCharacters,
}))

import NewCharactersModal from './NewCharactersModal'

const CHANNEL = 'new-characters-discovered'
type Listener = (data: unknown) => void

/** window.runtime 桩：EventsOn 返回「只摘除自己」的退订函数（wails v2.13 桌面语义）。 */
function stubRuntime() {
  const listeners = new Map<string, Listener[]>()
  const eventsOn = vi.fn((ch: string, h: Listener) => {
    const arr = listeners.get(ch) ?? []
    arr.push(h)
    listeners.set(ch, arr)
    return () => {
      const cur = listeners.get(ch)
      if (!cur) return
      const i = cur.indexOf(h)
      if (i >= 0) cur.splice(i, 1)
    }
  })
  const eventsOff = vi.fn()
  return {
    eventsOn,
    eventsOff,
    count: (ch: string) => (listeners.get(ch) ?? []).length,
    emit: (ch: string, payload: unknown) => { for (const h of [...(listeners.get(ch) ?? [])]) h(payload) },
  }
}

let rt: ReturnType<typeof stubRuntime>
let originalRuntime: unknown

beforeEach(() => {
  vi.clearAllMocks()
  rt = stubRuntime()
  const w = window as unknown as { runtime?: unknown }
  originalRuntime = w.runtime
  w.runtime = { EventsOn: rt.eventsOn, EventsOff: rt.eventsOff }
})

afterEach(() => {
  ;(window as unknown as { runtime?: unknown }).runtime = originalRuntime
})

describe('NewCharactersModal 新角色发现（T6-7.5）', () => {
  it('订阅通道 → 事件驱动开弹窗；改名后确认走 SaveCharactersBatch', async () => {
    render(<NewCharactersModal />)
    expect(rt.eventsOn).toHaveBeenCalledWith(CHANNEL, expect.any(Function))

    act(() => { rt.emit(CHANNEL, { characters: ['林昭'], chapterNum: 3 }) })
    const input = await screen.findByDisplayValue('林昭')
    expect(screen.getByText(/第3章发现了 1 个新角色/)).toBeTruthy()

    fireEvent.change(input, { target: { value: '林昭之' } })
    fireEvent.click(screen.getByRole('button', { name: /确认添加/ }))
    await waitFor(() => expect(mocks.SaveCharactersBatch).toHaveBeenCalledWith(JSON.stringify(['林昭之'])))
  })

  it('libraryMatches 载荷：库内同名角色直接关联（走 characterlib）', async () => {
    render(<NewCharactersModal />)
    act(() => {
      rt.emit(CHANNEL, { characters: [], libraryMatches: [{ id: 'c1', name: '林昭', roleType: 'supporting' }], chapterNum: 5 })
    })
    expect(await screen.findByText(/第5章发现了 1 个新角色/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /确认添加/ }))
    await waitFor(() => expect(mocks.associateToProject).toHaveBeenCalledWith('c1', 'supporting'))
    await waitFor(() => expect(mocks.syncProjectCharacters).toHaveBeenCalledTimes(1))
    expect(mocks.SaveCharactersBatch).not.toHaveBeenCalled()
  })

  it('退订只摘自己那一个监听：同通道别人的监听不受影响，绝不 EventsOff 全清', () => {
    const foreign = vi.fn()
    rt.eventsOn(CHANNEL, foreign)
    const { unmount } = render(<NewCharactersModal />)
    expect(rt.count(CHANNEL)).toBe(2)

    unmount()
    expect(rt.count(CHANNEL)).toBe(1)
    act(() => { rt.emit(CHANNEL, { characters: ['甲'], chapterNum: 1 }) })
    expect(foreign).toHaveBeenCalledTimes(1)
    expect(rt.eventsOff).not.toHaveBeenCalled()
    // 本组件已卸载：不再开弹窗
    expect(screen.queryByText(/第1章发现了/)).toBeNull()
  })

  it('无 runtime 通道：渲染不炸（弹窗不可用但不抛错）', () => {
    delete (window as unknown as { runtime?: unknown }).runtime
    expect(() => render(<NewCharactersModal />)).not.toThrow()
  })
})
