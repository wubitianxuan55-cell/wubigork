// novelSwitchGuard.test.ts — 跨页切书未保存保护（小说板块最后一条丢稿路径）
//
// 形态：脏状态登记 + 单一拦截点（书架页）。反向守卫三条：
// ①无脏时闸门必须**零打扰**直接放行（否则每次切书都弹窗）；
// ②「先保存」全成功才放行；任一个保存失败必须中止并如实报错（不能假装切成功）；
// ③不具备保存能力（多标签脏）必须降级为两选，且✕/取消＝不切。
//
// 坑复训（沿用 unsavedGuard.test 的三条纪律）：imperative Modal 的 DOM 不随
// cleanup() 卸载 → 断言限定在**最新弹窗**；标题双匹配不用 findByText；两字按钮
// 会被 antd 插空格 → 按钮名一律正则。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, waitFor, within } from '@testing-library/react'
import { Modal } from 'antd'
import {
  dirtyNovelProviders, guardNovelSwitch, registerNovelDirtyProvider,
  resetNovelDirtyProviders, takeDiscardConfirmed,
} from './novelSwitchGuard'

afterEach(async () => {
  Modal.destroyAll()
  resetNovelDirtyProviders()
  await new Promise((r) => setTimeout(r, 0))
})

const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))

/** 当前弹窗数。注意：imperative Modal 的 DOM **不随 destroy()/cleanup 卸载**（仓内实测坑），
 *  所以断言「没有弹窗」必须用前后差值，不能写 `=== 0`。 */
const confirmCount = () => confirms().length

/** 打开一个确认弹窗并返回「最新弹窗」的限定查询器。 */
async function openConfirm(trigger: () => boolean) {
  render(<div />)
  const before = confirms().length
  const intercepted = trigger()
  await waitFor(() => expect(confirms().length).toBeGreaterThan(before))
  const all = confirms()
  return { scoped: within(all[all.length - 1]), intercepted }
}

describe('guardNovelSwitch：无脏零打扰', () => {
  it('没有任何登记/登记都不脏 → 返回 false（由调用方继续执行 proceed），不弹窗', () => {
    const before = confirmCount()
    const proceed = vi.fn()
    expect(guardNovelSwitch(proceed, '某书')).toBe(false)
    // 闸门只负责「拦不拦」，放行时由调用方执行 proceed（HomePage 的 if (...) return; apply()）
    expect(proceed).not.toHaveBeenCalled()
    expect(confirmCount()).toBe(before)
  })

  it('已登记但 dirty() 为 false → 同样返回 false 不弹窗', () => {
    const before = confirmCount()
    registerNovelDirtyProvider({ id: 'p1', label: () => '创作页正文', dirty: () => false })
    const proceed = vi.fn()
    expect(guardNovelSwitch(proceed)).toBe(false)
    expect(proceed).not.toHaveBeenCalled()
    expect(confirmCount()).toBe(before)
  })

  it('dirty() 抛错按「不脏」处理（fail-open：不把作者锁死在切不了书）', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    registerNovelDirtyProvider({
      id: 'boom', label: () => '坏探针',
      dirty: () => { throw new Error('probe boom') },
    })
    const proceed = vi.fn()
    expect(guardNovelSwitch(proceed)).toBe(false)
    expect(proceed).not.toHaveBeenCalled()
    expect(warn).toHaveBeenCalled()
    warn.mockRestore()
  })
})

describe('guardNovelSwitch：有脏 → 拦截', () => {
  it('可保存时给三选；点「放弃修改并切换」放行并记账一次确认', async () => {
    registerNovelDirtyProvider({
      id: 'create-body', label: () => '创作页正文',
      dirty: () => true, save: async () => true,
    })
    const proceed = vi.fn()
    const { scoped, intercepted } = await openConfirm(() => guardNovelSwitch(proceed, '第二本书'))

    expect(intercepted).toBe(true)
    expect(proceed).not.toHaveBeenCalled()
    expect(scoped.getByText(/切换到「第二本书」/)).toBeTruthy()
    expect(scoped.getByText(/创作页正文/)).toBeTruthy()

    fireEvent.click(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改\s*并\s*切\s*换/ }))
    await waitFor(() => expect(proceed).toHaveBeenCalledTimes(1))
    // 已确认放弃 → 各页自己的兜底提示可跳过（每个消费者各一次）
    expect(takeDiscardConfirmed('create-body')).toBe(true)
    expect(takeDiscardConfirmed('create-body')).toBe(false)
    expect(takeDiscardConfirmed('chapter-tabs')).toBe(true)
  })

  it('可保存时点「先保存再切换」：保存成功后才放行（顺序：save → proceed）', async () => {
    const order: string[] = []
    const save = vi.fn(async () => { order.push('save'); return true })
    registerNovelDirtyProvider({
      id: 'create-body', label: () => '创作页正文', dirty: () => true, save,
    })
    const proceed = vi.fn(() => { order.push('proceed') })
    const { scoped } = await openConfirm(() => guardNovelSwitch(proceed, '第二本书'))

    fireEvent.click(scoped.getByRole('button', { name: /先\s*保\s*存\s*再\s*切\s*换/ }))
    await waitFor(() => expect(proceed).toHaveBeenCalledTimes(1))
    expect(order).toEqual(['save', 'proceed'])
    // 「先保存」路径没有丢内容，不算「已确认放弃」
    expect(takeDiscardConfirmed('create-body')).toBe(false)
  })

  it('多个持有者：全部保存成功才放行', async () => {
    const saveA = vi.fn(async () => true)
    const saveB = vi.fn(async () => true)
    registerNovelDirtyProvider({ id: 'a', label: () => '创作页正文', dirty: () => true, save: saveA })
    registerNovelDirtyProvider({ id: 'b', label: () => '阅读页未保存的章节', dirty: () => true, save: saveB })
    const proceed = vi.fn()
    const { scoped } = await openConfirm(() => guardNovelSwitch(proceed))

    expect(scoped.getByText(/创作页正文、阅读页未保存的章节/)).toBeTruthy()
    fireEvent.click(scoped.getByRole('button', { name: /先\s*保\s*存\s*再\s*切\s*换/ }))
    await waitFor(() => expect(proceed).toHaveBeenCalledTimes(1))
    expect(saveA).toHaveBeenCalledTimes(1)
    expect(saveB).toHaveBeenCalledTimes(1)
  })

  it('保存失败 → 中止切换、不 proceed（不能假装切成功）', async () => {
    registerNovelDirtyProvider({
      id: 'create-body', label: () => '创作页正文',
      dirty: () => true, save: async () => false,
    })
    const proceed = vi.fn()
    const { scoped } = await openConfirm(() => guardNovelSwitch(proceed))

    fireEvent.click(scoped.getByRole('button', { name: /先\s*保\s*存\s*再\s*切\s*换/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(proceed).not.toHaveBeenCalled()
  })

  it('不具备保存能力（canSave=false）→ 降级两选：放弃修改并切换 / 取消', async () => {
    registerNovelDirtyProvider({
      id: 'chapter-tabs', label: () => '阅读页 3 个未保存章节',
      dirty: () => true, canSave: () => false, save: async () => true,
    })
    const proceed = vi.fn()
    const { scoped } = await openConfirm(() => guardNovelSwitch(proceed, '第二本书'))

    // 不给「先保存」——多标签脏无法一次存齐，不假装可以
    expect(scoped.queryByRole('button', { name: /先\s*保\s*存/ })).toBeNull()
    expect(scoped.getByText(/无法一次性自动保存/)).toBeTruthy()

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(proceed).not.toHaveBeenCalled()

    fireEvent.click(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改\s*并\s*切\s*换/ }))
    await waitFor(() => expect(proceed).toHaveBeenCalledTimes(1))
  })

  it('注销后不再拦截（组件卸载即退出闸门）', () => {
    const unregister = registerNovelDirtyProvider({
      id: 'create-body', label: () => '创作页正文', dirty: () => true, save: async () => true,
    })
    expect(dirtyNovelProviders()).toHaveLength(1)
    unregister()
    expect(dirtyNovelProviders()).toHaveLength(0)
    const before = confirmCount()
    const proceed = vi.fn()
    expect(guardNovelSwitch(proceed)).toBe(false)
    expect(proceed).not.toHaveBeenCalled()
    expect(confirmCount()).toBe(before)
  })
})
