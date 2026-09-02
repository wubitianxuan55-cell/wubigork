import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { GenerationBar } from './GenerationBar'

const baseProps = {
  mode: 'txt2img' as const,
  backend: 'comfyui',
  model: 'krea2',
  count: 1,
  frames: 0,
  fps: 0,
  generating: true,
  elapsed: 12,
  lastTime: 0,
  pendingCount: 0,
  queueTotal: 0,
  needsComfy: false,
  onGenerate: vi.fn(),
  onCancel: vi.fn(),
}

describe('GenerationBar 生成进度显示', () => {
  it('ComfyUI 实时进度：显示百分比与当前节点（中文阶段名）', () => {
    const { container } = render(
      <GenerationBar {...baseProps} comfyProgress={{ status: 'running', elapsed: 12, percent: 42, node: 'KSampler' }} />,
    )
    expect(screen.getByText('全部: 42%')).toBeTruthy()
    expect(screen.getByText('当前节点: 采样中')).toBeTruthy()
    const fill = container.querySelector('.ig-progress-fill') as HTMLElement
    expect(fill.style.width).toBe('42%')
    expect(fill.className).not.toContain('is-indeterminate')
  })

  it('无实时进度时显示不定态光带（不确定宽度）', () => {
    const { container } = render(
      <GenerationBar {...baseProps} backend="xai" comfyProgress={{ status: '', elapsed: 0, percent: -1, node: '' }} />,
    )
    expect(screen.getByText('生成中…')).toBeTruthy()
    const fill = container.querySelector('.ig-progress-fill') as HTMLElement
    expect(fill.className).toContain('is-indeterminate')
  })

  it('未知节点名回退显示原始 class_type', () => {
    render(
      <GenerationBar {...baseProps} comfyProgress={{ status: 'running', elapsed: 5, percent: 10, node: 'MyCustomNode' }} />,
    )
    expect(screen.getByText('当前节点: MyCustomNode')).toBeTruthy()
  })
})

describe('GenerationBar 模式门禁', () => {
  it('img2img + xai + needsComfy：仍禁用生成按钮并提示切换引擎', () => {
    const { container } = render(
      <GenerationBar {...baseProps} mode="img2img" backend="xai" generating={false} needsComfy={true} />,
    )
    const btn = container.querySelector('.ig-gen-button') as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    expect(screen.getByText('图生图需切换至 ComfyUI / Herdsman')).toBeTruthy()
  })

  it('glm + img2img 残留态：禁用并提示 GLM 仅支持文生图', () => {
    const { container } = render(
      <GenerationBar {...baseProps} mode="img2img" backend="glm" generating={false} needsComfy={false} />,
    )
    const btn = container.querySelector('.ig-gen-button') as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    expect(screen.getByText('GLM 仅支持文生图，请切换到文生图模式或更换引擎')).toBeTruthy()
  })
})
