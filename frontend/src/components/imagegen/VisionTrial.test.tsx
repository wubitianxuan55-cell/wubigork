// VisionTrial（图像域 T1 识图「读/懂」画室试用）行为用例：
// 选图 → SavePastedImage 落盘 → 识别/OCR 动作链（缺省提示词）→ 结果诚实呈现
// （原语标注/模型名/错误原文）→ 历史最近 5 条落 localStorage → 清空。
// paste 事件与文件选择走同一漏斗（handleImageDataUrl），jsdom 无剪贴板文件，
// 用例经 input[type=file] 驱动同一链路。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { LocaleProvider } from '../../gaea/lib/i18n'
import { VisionTrial } from './VisionTrial'
import {
  readFileAsDataURL, savePastedImage, visionRead, visionUnderstand,
} from '../../api/image'

vi.mock('../../api/image', () => ({
  readFileAsDataURL: vi.fn(),
  savePastedImage: vi.fn(),
  visionUnderstand: vi.fn(),
  visionRead: vi.fn(),
}))

const readFileMock = vi.mocked(readFileAsDataURL)
const saveMock = vi.mocked(savePastedImage)
const understandMock = vi.mocked(visionUnderstand)
const readMock = vi.mocked(visionRead)

const SAVED_PATH = '.gaea/attachments/mock.png'
const DEFAULT_PROMPT = '请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。'

function renderTrial() {
  return render(
    <LocaleProvider>
      <VisionTrial onClose={vi.fn()} />
    </LocaleProvider>,
  )
}

/** 经隐藏 input[type=file] 驱动「选图 → 落盘」链路（与 paste 同一漏斗）。 */
async function pickImage(container: HTMLElement) {
  const input = container.querySelector('input[type="file"]') as HTMLInputElement
  const file = new File(['fake-png-bytes'], 'shot.png', { type: 'image/png' })
  Object.defineProperty(input, 'files', { value: [file] })
  fireEvent.change(input)
  await waitFor(() => expect(saveMock).toHaveBeenCalledWith(expect.stringContaining('data:image/png')))
  await waitFor(() => expect(screen.getAllByText(SAVED_PATH).length).toBeGreaterThan(0))
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
  localStorage.removeItem('imagehubT1.visionHistory')
  readFileMock.mockReset().mockResolvedValue('data:image/png;base64,AAA')
  saveMock.mockReset().mockResolvedValue(SAVED_PATH)
  understandMock.mockReset()
  readMock.mockReset()
})

describe('VisionTrial 识图「读/懂」画室试用', () => {
  it('选图 → SavePastedImage 落盘 → 展示预览与落盘路径', async () => {
    const { container } = renderTrial()
    await pickImage(container)
  })

  it('识别内容：缺省提示词走 GaeaRecognizeImage（识图-懂），结果区诚实呈现文本与原语标注', async () => {
    understandMock.mockResolvedValue({ text: '模拟识别：一张任务清单截图。' })
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() =>
      expect(understandMock).toHaveBeenCalledWith(SAVED_PATH, DEFAULT_PROMPT))
    await waitFor(() => expect(screen.getByText('模拟识别：一张任务清单截图。')).toBeTruthy())
    // 原语标注：识图-懂（顶条常驻 1 处 + 结果区 1 处）。
    expect(screen.getAllByText('识图-懂').length).toBeGreaterThanOrEqual(2)
  })

  it('提取文字：走 GaeaOCRText（识图-读），结果区标注识图-读', async () => {
    readMock.mockResolvedValue({ text: '模拟 OCR：营收 120 万元' })
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('提取文字'))
    await waitFor(() => expect(readMock).toHaveBeenCalledWith(SAVED_PATH))
    await waitFor(() => expect(screen.getByText('模拟 OCR：营收 120 万元')).toBeTruthy())
    expect(screen.getAllByText('识图-读').length).toBeGreaterThanOrEqual(2)
  })

  it('返回携带模型名时在结果区标注（若返回里有）', async () => {
    understandMock.mockResolvedValue({ text: '带模型名的识别结果', model: 'ovis2-8b' })
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() => expect(screen.getByText('ovis2-8b')).toBeTruthy())
  })

  it('调用失败：错误原文诚实呈现，不吞不伪装', async () => {
    understandMock.mockRejectedValue(new Error('视觉模型未就绪'))
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() => expect(screen.getByText('调用失败（原文）')).toBeTruthy())
    expect(screen.getByText('视觉模型未就绪')).toBeTruthy()
  })

  it('未载图点动作：诚实提示先粘贴或选择图片', async () => {
    renderTrial()
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() => expect(screen.getByText('请先粘贴或选择一张图片')).toBeTruthy())
    expect(understandMock).not.toHaveBeenCalled()
  })

  it('历史最近 5 条落 localStorage（文本截断），清空后回空态', async () => {
    understandMock.mockResolvedValue({ text: 'x'.repeat(900) })
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() => expect(screen.getByText('最近试用')).toBeTruthy())
    await waitFor(() => {
      const stored = JSON.parse(localStorage.getItem('imagehubT1.visionHistory') || '[]') as Array<{ text: string }>
      expect(stored).toHaveLength(1)
      expect(stored[0].text.length).toBe(600)
    })
    // 清空：localStorage 移除 + 回「暂无历史」空态。
    fireEvent.click(screen.getByText('清空'))
    await waitFor(() => expect(screen.getByText('暂无历史')).toBeTruthy())
    expect(localStorage.getItem('imagehubT1.visionHistory')).toBeNull()
  })

  it('历史回放：点击历史卡片把路径/提示词/结果装回工作区', async () => {
    understandMock.mockResolvedValue({ text: '历史条目结果' })
    readMock.mockResolvedValue({ text: 'OCR 新结果' })
    const { container } = renderTrial()
    await pickImage(container)
    fireEvent.click(screen.getByText('识别内容'))
    await waitFor(() => expect(screen.getByText('历史条目结果')).toBeTruthy())
    // 再跑一次 OCR 产出新结果，随后点历史缩略卡回放首次识别条目。
    fireEvent.click(screen.getByText('提取文字'))
    await waitFor(() => expect(screen.getByText('OCR 新结果')).toBeTruthy())
    const histCard = screen.getByTitle(
      (content: string | null) => !!content && content.includes('历史条目结果'))
    fireEvent.click(histCard)
    await waitFor(() => expect(screen.getByText('历史条目结果')).toBeTruthy())
  })
})
