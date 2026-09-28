// unsavedGuard.test.tsx — 小说板块未保存保护原语（v4.421.0）
// 坑复训（本文件实测三条）：
// ①imperative Modal 的 DOM 不随 cleanup()/destroy 立即卸载 → 断言一律**限定在最新弹窗**
//   （document.querySelectorAll('.ant-modal-confirm') 取末个 + within），不用全局 role 查询；
// ②antd 弹窗标题父子双匹配（标题 wrapper 与内层 div 文本相同）→ 不用 findByText 定位标题；
// ③antd 两字按钮自动插空格（「取 消」）→ 按钮名一律正则。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, waitFor, within } from '@testing-library/react'
import { Modal } from 'antd'
import { chooseAction, chooseUnsavedAction, confirmDiscard } from './unsavedGuard'

afterEach(async () => {
  Modal.destroyAll()
  await new Promise((r) => setTimeout(r, 0))
})

const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))

/** 打开一个确认弹窗并返回「最新弹窗」的限定查询器（open 前后比对数量）。 */
async function openConfirm(trigger: () => void) {
  render(<div />)
  const before = confirms().length
  trigger()
  await waitFor(() => expect(confirms().length).toBeGreaterThan(before))
  const all = confirms()
  return within(all[all.length - 1])
}

describe('confirmDiscard（两选）', () => {
  it('点「放弃修改」调用 onDiscard 恰一次', async () => {
    const onDiscard = vi.fn()
    const scoped = await openConfirm(() =>
      confirmDiscard({ title: '章节尚未保存', message: '未保存的修改会丢失。', onDiscard }))

    expect(scoped.getByText('未保存的修改会丢失。')).toBeTruthy()
    fireEvent.click(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改/ }))
    await waitFor(() => expect(onDiscard).toHaveBeenCalledTimes(1))
  })

  it('点「取消」不触发 onDiscard（✕/Esc 同义：什么都不做）', async () => {
    const onDiscard = vi.fn()
    const scoped = await openConfirm(() => confirmDiscard({ title: '设定尚未保存', onDiscard }))

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(onDiscard).not.toHaveBeenCalled()
  })
})

describe('chooseUnsavedAction（三选）', () => {
  it('三按钮齐备；点「先保存」不触发放弃', async () => {
    const onSave = vi.fn()
    const onDiscard = vi.fn()
    const scoped = await openConfirm(() =>
      chooseUnsavedAction({ title: '有未保存的正文', onSave, onDiscard }))

    expect(scoped.getByRole('button', { name: /取\s*消/ })).toBeTruthy()
    expect(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改/ })).toBeTruthy()
    fireEvent.click(scoped.getByRole('button', { name: /先\s*保\s*存/ }))
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(onDiscard).not.toHaveBeenCalled()
  })

  it('未提供 onSave 时不渲染「先保存」', async () => {
    const scoped = await openConfirm(() =>
      chooseUnsavedAction({ title: '有未保存的设定', onDiscard: vi.fn() }))

    expect(scoped.queryByRole('button', { name: /先\s*保\s*存/ })).toBeNull()
    expect(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改/ })).toBeTruthy()
  })

  it('点「放弃修改」触发 onDiscard，不触发 onSave', async () => {
    const onSave = vi.fn()
    const onDiscard = vi.fn()
    const scoped = await openConfirm(() =>
      chooseUnsavedAction({ title: '有未保存的正文', onSave, onDiscard }))

    fireEvent.click(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改/ }))
    await waitFor(() => expect(onDiscard).toHaveBeenCalledTimes(1))
    expect(onSave).not.toHaveBeenCalled()
  })

  it('点「取消」两个回调都不触发', async () => {
    const onSave = vi.fn()
    const onDiscard = vi.fn()
    const scoped = await openConfirm(() =>
      chooseUnsavedAction({ title: '有未保存的正文', onSave, onDiscard }))

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(onSave).not.toHaveBeenCalled()
    expect(onDiscard).not.toHaveBeenCalled()
  })
})

describe('chooseAction（并列多分支）', () => {
  it('每个分支各走自己的回调，取消不触发任何分支（✕/Esc 同义）', async () => {
    const overwrite = vi.fn()
    const append = vi.fn()
    const scoped = await openConfirm(() =>
      chooseAction({
        title: '第 3 章已存在',
        message: '选择生成方式：',
        options: [
          { label: '覆盖下一章', tone: 'danger', run: overwrite },
          { label: '作为分支追加', tone: 'primary', run: append },
        ],
      }))

    expect(scoped.getByText('选择生成方式：')).toBeTruthy()
    fireEvent.click(scoped.getByRole('button', { name: /作为分支追加/ }))
    await waitFor(() => expect(append).toHaveBeenCalledTimes(1))
    expect(overwrite).not.toHaveBeenCalled()

    // 取消路径：两个分支都不许被触发（旧实现把「追加」挂在 onCancel 上）
    const scoped2 = await openConfirm(() =>
      chooseAction({
        title: '第 3 章已存在',
        options: [
          { label: '覆盖下一章', run: overwrite },
          { label: '作为分支追加', run: append },
        ],
      }))
    fireEvent.click(scoped2.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(overwrite).not.toHaveBeenCalled()
    expect(append).toHaveBeenCalledTimes(1)
  })
})
