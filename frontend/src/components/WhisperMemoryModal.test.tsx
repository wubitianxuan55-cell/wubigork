// WhisperMemoryModal — 键盘可达与图标按钮可访问名（v4.349 可访问性刀）
//
// 背景：本面板（角色库/轻语记忆主面板）此前只有鼠标可达——事实标题与摘要
// 是裸 onClick 的 span/div，确认/取消/编辑/删除四个图标钮只有 Tooltip
// （Tooltip 不参与可访问名计算，读屏读到的是「无名称按钮」）。
import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import WhisperMemoryModal, { type MemoryFact } from './WhisperMemoryModal'

const FACT: MemoryFact = {
  id: 'f1',
  domain: 'INNER_WORLD',
  subcategory: 'MOOD',
  subject: '喜欢雨天的窗边',
  summary: '她说下雨时喜欢坐在窗边听雨，觉得安静。',
  weight: 8.2,
  confidence: 0.9,
  tier: 'memory',
  createdAt: '2026-09-19 10:00',
  updatedAt: '2026-09-19 10:00',
}

function renderModal(facts: MemoryFact[] = [FACT]) {
  return render(<WhisperMemoryModal facts={facts} personalityID="p1" />)
}

describe('WhisperMemoryModal 键盘与可访问名（v4.349）', () => {
  it('四个图标按钮都有可访问名（编辑/删除；编辑态转为保存/取消）', () => {
    renderModal()
    expect(screen.getByLabelText('编辑记忆')).toBeTruthy()
    expect(screen.getByLabelText('删除记忆')).toBeTruthy()

    fireEvent.click(screen.getByLabelText('编辑记忆'))
    expect(screen.getByLabelText('保存修改')).toBeTruthy()
    expect(screen.getByLabelText('取消编辑')).toBeTruthy()
  })

  it('事实标题可键盘激活：role=button + tabIndex=0，Enter 打开详情', async () => {
    renderModal()
    const title = screen.getByText('喜欢雨天的窗边')
    expect(title.getAttribute('role')).toBe('button')
    expect(title.getAttribute('tabindex')).toBe('0')

    fireEvent.keyDown(title, { key: 'Enter' })
    await waitFor(() =>
      expect(document.querySelector('.ant-modal-title')?.textContent).toBe('喜欢雨天的窗边'),
    )
  })

  it('摘要可键盘激活：Enter 打开详情；空格同样激活', async () => {
    renderModal()
    const summary = screen.getByText(/她说下雨时喜欢坐在窗边听雨/)
    expect(summary.getAttribute('role')).toBe('button')
    expect(summary.getAttribute('tabindex')).toBe('0')

    fireEvent.keyDown(summary, { key: ' ' })
    await waitFor(() => expect(document.querySelector('.ant-modal-title')).toBeTruthy())
  })

  it('编辑态下摘要不参与激活（换成输入框，避免误触详情）', () => {
    renderModal()
    fireEvent.click(screen.getByLabelText('编辑记忆'))
    expect(document.querySelector('textarea')).toBeTruthy()
    const roleButtons = Array.from(document.querySelectorAll('[role="button"]')).map((e) => e.textContent ?? '')
    expect(roleButtons.some((t) => t.includes('她说下雨时喜欢坐在窗边听雨'))).toBe(false)
  })

  it('确认修改回调：保存走 WhisperUpdateFact 并回传新 facts', async () => {
    const onFactsChange = vi.fn()
    render(<WhisperMemoryModal facts={[FACT]} personalityID="p1" onFactsChange={onFactsChange} />)
    fireEvent.click(screen.getByLabelText('编辑记忆'))
    fireEvent.click(screen.getByLabelText('保存修改'))
    await waitFor(() => expect(onFactsChange).toHaveBeenCalled())
  })
})
