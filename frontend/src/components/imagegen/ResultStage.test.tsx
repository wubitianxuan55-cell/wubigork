import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ResultStage } from './ResultStage'

const baseProps = {
  generating: false,
  onPreview: vi.fn(),
  onDownload: vi.fn(),
  onReuse: vi.fn(),
}

function result(image: string): { image: string; seed: number; time: number; prompt: string; model: string; size: string } {
  return { image, seed: 1, time: 3, prompt: 'test', model: 'm', size: '1024x1024' }
}

describe('ResultStage 画布结果展示', () => {
  it('单张结果铺满画布：object-fit contain 且容器撑满', () => {
    const { container } = render(
      <ResultStage {...baseProps} results={[result('data:image/png;base64,AAA')]} />,
    )
    const img = container.querySelector('img') as HTMLImageElement
    expect(img).toBeTruthy()
    expect(img.style.objectFit).toBe('contain')
    expect(screen.getByText('3s')).toBeTruthy()
    expect(screen.getByTitle('预览')).toBeTruthy()
    expect(screen.getByTitle('下载')).toBeTruthy()
    expect(screen.getByTitle('复用')).toBeTruthy()
  })

  it('多张结果保持网格卡片（object-fit cover）', () => {
    const { container } = render(
      <ResultStage
        {...baseProps}
        results={[result('data:image/png;base64,AAA'), result('data:image/png;base64,BBB')]}
      />,
    )
    const imgs = container.querySelectorAll('img')
    expect(imgs.length).toBe(2)
    imgs.forEach((img) => {
      expect((img as HTMLImageElement).style.objectFit).toBe('cover')
    })
  })
})
