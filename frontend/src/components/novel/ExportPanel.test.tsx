// ExportPanel.test.tsx — 小说导出面板（v4.425 B6 新增）。
// 为什么要有这一件：后端 ExportAll 把**逐格式失败写进 value**（internal/export/export.go
// 的 "失败: …"），error 恒为 nil；面板此前只用 Object.keys(res).length === 0 判「没生成
// 文件」，于是四种格式全失败也会弹绿色「导出完成」——失败被说成成功。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return { ...actual, app: { ExportAll: vi.fn() } }
})

import ExportPanel from './ExportPanel'
import { app } from '../../gaea/lib/bridge'

const exportAll = vi.mocked(app.ExportAll)

const renderPanel = () => render(<ExportPanel />)
const clickExport = () => fireEvent.click(screen.getByRole('button', { name: /导出全部格式/ }))

/** 当前 antd message 通知节点快照（调用动作前先取，用增量定位「本次」的通知）。 */
const notices = (): HTMLElement[] => Array.from(document.querySelectorAll<HTMLElement>('.ant-message-notice'))

/** 本次动作新产生的通知（含文案）；避免用常驻容器的历史节点做否定断言。 */
async function newNoticeText(before: number): Promise<string> {
  await waitFor(() => expect(notices().length).toBeGreaterThan(before))
  const all = notices()
  return (all[all.length - 1]?.textContent ?? '').trim()
}

describe('ExportPanel 导出结果提示（v4.425 B6）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('全部成功：绿色「导出完成」，逐格式结果列出', async () => {
    exportAll.mockResolvedValue({
      '.txt': 'C:/novels/书/export/book.txt',
      '.md': 'C:/novels/书/export/book.md',
      '.epub': 'C:/novels/书/export/book.epub',
      '.docx': 'C:/novels/书/export/book.docx',
    } as never)
    renderPanel()
    const before = notices().length
    clickExport()

    expect(await newNoticeText(before)).toBe('导出完成')
    expect(screen.getByText('C:/novels/书/export/book.epub')).toBeTruthy()
  })

  it('四种格式全失败：不再弹「导出完成」，改报失败实数（B6 反向守卫）', async () => {
    exportAll.mockResolvedValue({
      '.txt': '失败: 章节文件读取错误',
      '.md': '失败: 章节文件读取错误',
      '.epub': '失败: 章节文件读取错误',
      '.docx': '失败: 章节文件读取错误',
    } as never)
    renderPanel()
    const before = notices().length
    clickExport()

    const text = await newNoticeText(before)
    expect(text).toContain('导出失败：4/4 个格式全部失败')
    expect(text).not.toBe('导出完成')
  })

  it('部分失败：如实给「N/M 个格式失败」的警示，不谎报全成', async () => {
    exportAll.mockResolvedValue({
      '.txt': 'C:/novels/书/export/book.txt',
      '.md': 'C:/novels/书/export/book.md',
      '.epub': '失败: epub 打包失败',
      '.docx': '失败: docx 打包失败',
    } as never)
    renderPanel()
    const before = notices().length
    clickExport()

    const text = await newNoticeText(before)
    expect(text).toContain('导出完成，但 2/4 个格式失败')
    expect(text).not.toBe('导出完成')
  })

  it('空结果：给「没有生成文件」的警示（既有口径不变）', async () => {
    exportAll.mockResolvedValue({} as never)
    renderPanel()
    const before = notices().length
    clickExport()

    expect(await newNoticeText(before)).toContain('导出完成，但没有生成文件')
    await waitFor(() => expect(screen.getByText('没有可导出的章节内容')).toBeTruthy())
  })

  it('绑定抛错：如实透出错误，不弹成功', async () => {
    exportAll.mockRejectedValue(new Error('项目未打开') as never)
    renderPanel()
    const before = notices().length
    clickExport()

    const text = await newNoticeText(before)
    expect(text).toContain('项目未打开')
    expect(text).not.toBe('导出完成')
  })
})
