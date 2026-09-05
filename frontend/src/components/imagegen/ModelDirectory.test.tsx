// ModelDirectory（图像域 T1 画室「模型目录」创作语境视图）行为用例：
// 目录加载 → 图像相关过滤（纯 LLM 不混入）→ 能力分族落位（生图/改图/视频/识图）
// → 能力徽标 / 状态标签 → 成本诚实「未定价」（目录无成本字段，不伪装 0）→
// 「当前使用」高亮（仅 herdsman 后端 + 模型名对上）→ 卡片详情（目录实有字段 +
// 档位诚实「目录未标注」+ hint 用途建议）→ 空态 / 加载失败原文 / 目录来源异常。
// 数据源经 api/image 的 herdsmanModelCatalog（HerdsmanModelCatalog 只读绑定），
// 用例统一 vi.mock 该包装（与 VisionTrial.test 同模式）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { LocaleProvider } from '../../gaea/lib/i18n'
import { ModelDirectory } from './ModelDirectory'
import { herdsmanModelCatalog } from '../../api/image'

vi.mock('../../api/image', () => ({
  herdsmanModelCatalog: vi.fn(),
}))

const catalogMock = vi.mocked(herdsmanModelCatalog)

// 形状对齐 internal/app/herdsman_catalog.go（status/run_status 取 CLI 真实词汇）。
const CATALOG_FIXTURE = {
  models: [
    {
      name: 'z-image-turbo', display_name: 'Z-Image Turbo', type: 'image',
      capabilities: ['text-to-image', 'image-to-image'],
      installed: true, running: true, status: 'installed', run_status: 'running',
      file_size: 20401094656,
      hint: '本地文生图（19GB）：绘梦板块 herdsman 后端',
    },
    {
      name: 'flux', display_name: 'FLUX', type: 'image',
      capabilities: ['text-to-image'],
      installed: true, running: false, status: 'installed', run_status: 'stopped',
    },
    {
      name: 'ltx-video', display_name: 'LTX Video', type: 'video',
      capabilities: ['text-to-video', 'image-to-video'],
      installed: false, running: false, status: 'uninstalled', run_status: 'stopped',
    },
    {
      name: 'ovis2-8b', display_name: 'Ovis2 8B', type: 'vision',
      capabilities: ['image-understanding', 'ocr'],
      installed: true, running: false, status: 'installed', run_status: 'stopped',
    },
    {
      name: 'paddleocr', display_name: 'PaddleOCR', type: 'ocr',
      capabilities: ['ocr'],
      installed: true, running: false, status: 'installed', run_status: 'stopped',
      hint: '快速 OCR（约 90ms）：中文混合场景有错字，失败自动回退 OvisOCR2',
    },
    {
      // 非图像条目：画室视图应按能力字段过滤掉。
      name: 'qwen3-14b', display_name: 'Qwen3 14B', type: 'llm',
      capabilities: ['chat'],
      installed: true, running: false, status: 'installed', run_status: 'stopped',
      quantization: 'Q4_K_M', parameter_count: 14700000000,
    },
  ],
  total: 6,
  installed: 5,
  running: 1,
  source: 'mock',
}

function renderDirectory(props?: { backend?: string; model?: string }) {
  return render(
    <LocaleProvider>
      <ModelDirectory
        onClose={vi.fn()}
        backend={props?.backend ?? 'xai'}
        model={props?.model ?? 'krea2'}
      />
    </LocaleProvider>,
  )
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
  catalogMock.mockReset()
})

describe('ModelDirectory 画室「模型目录」创作语境视图', () => {
  it('加载目录：按能力分族展示（生图/改图/视频/识图），纯 LLM 条目不混入', async () => {
    catalogMock.mockResolvedValue(CATALOG_FIXTURE)
    renderDirectory()
    await waitFor(() => expect(screen.getByText('FLUX')).toBeTruthy())
    // 四个分族标题齐备（生图徽标与分组标题同词，用 getAllByText 容忍重复）。
    for (const group of ['生图', '改图', '视频', '识图']) {
      expect(screen.getAllByText(group).length).toBeGreaterThanOrEqual(1)
    }
    // 落位：z-image-turbo（txt2img+img2img）归入最高能力族「改图」；flux 在「生图」。
    expect(screen.getByText('Z-Image Turbo')).toBeTruthy()
    expect(screen.getByText('LTX Video')).toBeTruthy()
    expect(screen.getByText('Ovis2 8B')).toBeTruthy()
    // 纯 LLM（qwen3-14b）无图像关键词 → 不出现在画室目录视图。
    expect(screen.queryByText('Qwen3 14B')).toBeNull()
    // 概要行：图像相关 5 个 · 目录总计 6 个。
    expect(screen.getByText((c) => c.includes('图像相关 5 个') && c.includes('目录总计 6 个'))).toBeTruthy()
  })

  it('成本红线：目录无成本字段，每张卡统一诚实「未定价」，不伪装 0', async () => {
    catalogMock.mockResolvedValue(CATALOG_FIXTURE)
    renderDirectory()
    await waitFor(() => expect(screen.getByText('FLUX')).toBeTruthy())
    // 5 张图像相关卡各一条「成本: 未定价」（卡内成本行，函数匹配整段直接文本；
    // 底注与详情面板未展开不计入）。
    const costLines = screen.getAllByText((c) => c.startsWith('成本') && c.includes('未定价'))
    expect(costLines.length).toBe(5)
    expect(screen.queryByText(/¥/)).toBeNull()
  })

  it('能力徽标来自目录 capabilities（生图/改图/视频/文字识别），未安装条目诚实标注', async () => {
    catalogMock.mockResolvedValue(CATALOG_FIXTURE)
    renderDirectory()
    await waitFor(() => expect(screen.getByText('FLUX')).toBeTruthy())
    expect(screen.getAllByText('文字识别').length).toBe(2) // ovis2-8b + paddleocr
    expect(screen.getByText('未安装')).toBeTruthy() // ltx-video
    expect(screen.getByText('运行中')).toBeTruthy() // z-image-turbo
  })

  it('「当前使用」：herdsman 后端 + 当前模型在目录中 → 高亮；其它后端不标', async () => {
    catalogMock.mockResolvedValue(CATALOG_FIXTURE)
    const { rerender } = render(
      <LocaleProvider>
        <ModelDirectory onClose={vi.fn()} backend="herdsman" model="z-image-turbo" />
      </LocaleProvider>,
    )
    await waitFor(() => expect(screen.getByText('Z-Image Turbo')).toBeTruthy())
    expect(screen.getByText('当前使用')).toBeTruthy()
    // 非 herdsman 后端（如 comfyui 同名模型）不算目录内当前使用。
    rerender(
      <LocaleProvider>
        <ModelDirectory onClose={vi.fn()} backend="comfyui" model="z-image-turbo" />
      </LocaleProvider>,
    )
    await waitFor(() => expect(screen.queryByText('当前使用')).toBeNull())
  })

  it('点击卡片 → 详情仅目录实有字段：档位「目录未标注」、状态原始值、hint 用途建议', async () => {
    catalogMock.mockResolvedValue(CATALOG_FIXTURE)
    renderDirectory()
    await waitFor(() => expect(screen.getByText('Z-Image Turbo')).toBeTruthy())
    fireEvent.click(screen.getByText('Z-Image Turbo'))
    // 档位/成本诚实披露（目录未携带），状态取 CLI 原始词汇。
    expect(screen.getByText('目录未标注')).toBeTruthy()
    expect(screen.getByText('installed')).toBeTruthy()
    expect(screen.getByText('running')).toBeTruthy()
    // hint = 本机实测用途建议（目录实有字段）。
    expect(screen.getByText('本地文生图（19GB）：绘梦板块 herdsman 后端')).toBeTruthy()
  })

  it('目录为空（无图像相关模型）→ 诚实空态；底注数据源口径常驻', async () => {
    catalogMock.mockResolvedValue({ models: [], total: 0, installed: 0, running: 0, source: 'mock' })
    renderDirectory()
    await waitFor(() => expect(screen.getByText('目录中暂无图像相关模型。')).toBeTruthy())
    expect(screen.getByText((c) => c.includes('与模型中心目录同源'))).toBeTruthy()
  })

  it('调用失败：错误原文诚实呈现，不吞不伪装', async () => {
    catalogMock.mockRejectedValue(new Error('Herdsman CLI 不可用'))
    renderDirectory()
    await waitFor(() => expect(screen.getByText('模型目录加载失败：Herdsman CLI 不可用')).toBeTruthy())
  })

  it('目录来源自带 error（Go 目录字段）→ 警示行透传，已有模型照常展示', async () => {
    catalogMock.mockResolvedValue({
      ...CATALOG_FIXTURE,
      error: 'herdsman 返回部分模型信息缺失',
    })
    renderDirectory()
    await waitFor(() => expect(screen.getByText('目录来源返回异常：herdsman 返回部分模型信息缺失')).toBeTruthy())
    expect(screen.getByText('FLUX')).toBeTruthy()
  })
})
