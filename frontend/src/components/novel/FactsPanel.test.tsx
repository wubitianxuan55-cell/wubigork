// FactsPanel.test.tsx — 设定事实词典面板（chinese-novelist-skill 词典蒸馏批 N8）：
// 打开拉清单/登记表单必填校验/标记已揭示与删除置脏/保存整表 JSON/揭示徽标。
// 坑复训：antd Popconfirm 确认按钮取最后挂载弹层；两字按钮用正则。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  save: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return { ...actual, app: { NovelFactsList: mocks.list, NovelFactsSave: mocks.save } }
})

import FactsPanel from './FactsPanel'
import { Modal, message as antdMessage } from 'antd'

afterEach(() => {
  Modal.destroyAll()
  antdMessage.destroy()
})

const fact = (over: Partial<Parameters<typeof FactsPanel>[0]> = {}) => ({
  id: 'f1',
  name: '倒计时',
  category: '金手指',
  first_chapter: 1,
  reader_known: '能看到倒计时',
  full_truth: '死亡倒计时',
  reveal_chapter: 80,
  revealed: false,
  ...over,
})

describe('FactsPanel 设定事实词典（蒸馏批 N8）', () => {
  beforeEach(() => {
    mocks.list.mockReset()
    mocks.save.mockReset()
    mocks.list.mockResolvedValue({ facts: [fact()] })
  })

  it('打开拉清单：名称/双栏/揭示徽标（第80章）显形；未揭示真相弱化', async () => {
    render(<FactsPanel />)
    expect(await screen.findByText('倒计时')).toBeTruthy()
    expect(screen.getByText('能看到倒计时')).toBeTruthy()
    expect(screen.getByText('死亡倒计时')).toBeTruthy()
    expect(screen.getByText('第80章')).toBeTruthy()
    expect(mocks.list).toHaveBeenCalledTimes(1)
  })

  it('登记表单：必填缺失提示不上行；齐全则新增一行并置脏可保存', async () => {
    mocks.list.mockResolvedValueOnce({ facts: [] })
    render(<FactsPanel />)

    // 保存钮初始禁用（无脏）
    expect((screen.getByTestId('facts-save') as HTMLButtonElement).disabled).toBe(true)

    fireEvent.click(screen.getByTestId('facts-add'))
    // 弹窗 OK「登 记」与外钮「登记事实」同名族——用整名锚定（^…$ 排除长名）
    const okBtn = () => screen.getByRole('button', { name: /^登\s*记$/ })
    fireEvent.click(okBtn()) // 必填缺失
    await waitFor(() => expect(screen.getByText(/名称与读者已知为必填/)).toBeTruthy())

    const areas = screen.getAllByRole('textbox')
    fireEvent.change(areas[0], { target: { value: '旧约' } }) // 名称
    fireEvent.change(areas[areas.length - 2], { target: { value: '两家世代交好' } }) // 读者已知
    fireEvent.change(areas[areas.length - 1], { target: { value: '两家有血仇' } }) // 完整真相
    fireEvent.click(okBtn())

    expect(await screen.findByText('旧约')).toBeTruthy()
    const saveBtn = screen.getByTestId('facts-save') as HTMLButtonElement
    expect(saveBtn.disabled).toBe(false)

    mocks.save.mockResolvedValueOnce({ saved: 1 })
    fireEvent.click(saveBtn)
    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(mocks.save.mock.calls[0][0])
    expect(payload).toHaveLength(1)
    expect(payload[0]).toMatchObject({ name: '旧约', reader_known: '两家世代交好', full_truth: '两家有血仇', revealed: false })
    // 保存成功后回干净态
    await waitFor(() => expect((screen.getByTestId('facts-save') as HTMLButtonElement).disabled).toBe(true))
  })

  it('标记已揭示：徽标转绿、置脏；保存回传 revealed=true', async () => {
    render(<FactsPanel />)
    await screen.findByText('倒计时')

    fireEvent.click(screen.getByRole('button', { name: /标记已揭示/ }))
    expect(screen.getByText('已揭示')).toBeTruthy()

    mocks.save.mockResolvedValueOnce({ saved: 1 })
    fireEvent.click(screen.getByTestId('facts-save'))
    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(1))
    expect(JSON.parse(mocks.save.mock.calls[0][0])[0].revealed).toBe(true)
  })

  it('删除：Popconfirm 确认后行消失并置脏', async () => {
    render(<FactsPanel />)
    await screen.findByText('倒计时')

    fireEvent.click(screen.getByRole('button', { name: /删\s*除/ }))
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定|OK/ }))

    await waitFor(() => expect(screen.queryByText('倒计时')).toBeNull())
    expect((screen.getByTestId('facts-save') as HTMLButtonElement).disabled).toBe(false)
  })

  it('保存失败如实上屏，保持脏态可重试', async () => {
    render(<FactsPanel />)
    await screen.findByText('倒计时')
    fireEvent.click(screen.getByRole('button', { name: /标记已揭示/ }))

    mocks.save.mockRejectedValueOnce(new Error('保存事实词典失败: 权限不足'))
    fireEvent.click(screen.getByTestId('facts-save'))
    await waitFor(() => expect(screen.getByText(/权限不足/)).toBeTruthy())
    expect((screen.getByTestId('facts-save') as HTMLButtonElement).disabled).toBe(false)
  })
})
