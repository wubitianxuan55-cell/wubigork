// SkillModal 导入半桩（审计刀C-3）：Wails 壳内动态 input.click() 不弹框
// （v4.162 实证），改走 GaeaPickFiles 系统对话框还原 File 后维持同款
// 「仅展示文件名」引导语义（真导入=手动放 skills/ 目录，不做落盘）。
// mock 口径照 schedule/store.sync.test.ts：window.go.app.<门面> 装真壳
// 同款 Gaea 前缀绑定，afterEach delete 还原。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import SkillModal from './SkillModal'
import { wailsApp } from '../lib/wailsApp'

vi.mock('../lib/wailsApp', () => ({
  wailsApp: vi.fn(),
}))

const mockedWailsApp = vi.mocked(wailsApp)

describe('SkillModal 导入（刀C-3 壳内收口）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedWailsApp.mockReturnValue({
      ListSkills: vi.fn(async () => []),
    } as unknown as ReturnType<typeof wailsApp>)
  })

  afterEach(() => {
    delete (window as unknown as { go?: unknown }).go
  })

  it('壳内导入：GaeaPickFiles+GaeaReadFileB64 取文件名（仅展示文件名半桩 parity，不点动态 input）', async () => {
    const pickFiles = vi.fn(async () => ([
      { path: 'C:/docs/my-style.md', name: 'my-style.md', type: 'file', size: 5 },
    ]))
    const readFileB64 = vi.fn(async () => 'eHg=')
    ;(window as unknown as { go?: unknown }).go = {
      app: { SkillPickTestC: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
    }
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    try {
      render(<SkillModal open onClose={vi.fn()} />)
      fireEvent.click(await screen.findByRole('button', { name: /导入 SKILL\.md/ }))
      await waitFor(() => {
        expect(pickFiles).toHaveBeenCalledTimes(1)
        expect(readFileB64).toHaveBeenCalledWith('C:/docs/my-style.md')
      })
      // 与浏览器路径同款半桩语义：剥 .md 后缀仅展示文件名
      await waitFor(() => expect(screen.getByText('my-style')).toBeTruthy())
      expect(inputClick).not.toHaveBeenCalled()
    } finally {
      inputClick.mockRestore()
    }
  })

  it('壳内选到非 .md：fail-closed 提示，不展示文件名', async () => {
    const pickFiles = vi.fn(async () => ([
      { path: 'C:/docs/a.txt', name: 'a.txt', type: 'file', size: 5 },
    ]))
    const readFileB64 = vi.fn(async () => 'eHg=')
    ;(window as unknown as { go?: unknown }).go = {
      app: { SkillPickTestC: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
    }
    render(<SkillModal open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByRole('button', { name: /导入 SKILL\.md/ }))
    await waitFor(() => expect(pickFiles).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/仅支持 \.md/)).toBeTruthy()
    expect(readFileB64).not.toHaveBeenCalled()
  })

  it('浏览器环境回退动态 input（不调 GaeaPickFiles）', async () => {
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    try {
      render(<SkillModal open onClose={vi.fn()} />)
      fireEvent.click(await screen.findByRole('button', { name: /导入 SKILL\.md/ }))
      expect(inputClick).toHaveBeenCalledTimes(1)
    } finally {
      inputClick.mockRestore()
    }
  })
})
