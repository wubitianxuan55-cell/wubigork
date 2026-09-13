// ForeshadowPanel.test.tsx — 伏笔登记表面板关键路径
// 覆盖：手工登记→全量写回→列表出现；状态流转写回 hinted；保存失败回滚提示；删除（confirm）；
// 一致性体检（LintForeshadows）：概要行渲染 / findings Tag+说明+章节引用 / 空 findings 提示。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go。
// 组件经 gaea/lib/bridge 的 app 调用 GetForeshadows / SaveForeshadows / LintForeshadows。
vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      GetForeshadows: vi.fn().mockResolvedValue({ items: [] }),
      SaveForeshadows: vi.fn().mockResolvedValue(undefined),
      LintForeshadows: vi.fn().mockResolvedValue(null),
    },
  }
})

import ForeshadowPanel from './ForeshadowPanel'
import { app } from '../../gaea/lib/bridge'
import type { ForeshadowItemData } from '../../types'
import type { ForeshadowLintReport } from '../../gaea/lib/bridge/novel'

const EXISTING: ForeshadowItemData = {
  id: 'plot_001_abc',
  category: 'character',
  description: '主角左臂旧伤',
  planted_in: '001.md',
  status: 'planted',
  is_long_term: false,
}

describe('ForeshadowPanel 手工登记闭环', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [EXISTING] })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
  })

  it('登记伏笔：提交后 SaveForeshadows 全量写回（保留既有条目 + manual_ 新条目），列表出现', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.change(screen.getByPlaceholderText(/伏笔描述/), { target: { value: '神秘铜匣的钥匙' } })
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toHaveLength(2)
    // 既有 AI 条目原样保留（全量写回不丢数据）
    expect(payload[0]).toEqual(EXISTING)
    // 新条目：manual_ 前缀 ID、默认 planted、章节号补零
    expect(payload[1].id).toMatch(/^manual_\d+$/)
    expect(payload[1].status).toBe('planted')
    expect(payload[1].description).toBe('神秘铜匣的钥匙')
    expect(payload[1].category).toBe('plot')
    expect(payload[1].planted_in).toBe('001.md')
    expect(payload[1].is_long_term).toBe(false)

    // 列表出现新条目 + 成功提示
    expect(await screen.findByText('神秘铜匣的钥匙')).toBeTruthy()
    expect(await screen.findByText('伏笔已登记')).toBeTruthy()
  })

  it('描述为空时拒登记，不触发写回', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')
    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))
    expect(await screen.findByText('请填写伏笔描述')).toBeTruthy()
    expect(vi.mocked(app.SaveForeshadows)).not.toHaveBeenCalled()
  })

  it('状态流转：标记暗示 → 乐观更新徽标并写回 hinted', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()
    expect(screen.getByText('已埋设')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload[0].status).toBe('hinted')
    expect(await screen.findByText('已暗示')).toBeTruthy()
  })

  it('保存失败：回滚列表并提示', async () => {
    vi.mocked(app.SaveForeshadows).mockRejectedValueOnce(new Error('磁盘已满'))
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    expect(await screen.findByText(/伏笔保存失败，已回滚/)).toBeTruthy()
    // 回滚后状态徽标恢复 planted
    expect(await screen.findByText('已埋设')).toBeTruthy()
  })

  it('删除：confirm 后全量写回剩余条目', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '删除伏笔：主角左臂旧伤' }))
    // antd Popconfirm 双字按钮会插空格（「删 除」）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toEqual([])
    expect(await screen.findByText(/还没有伏笔登记/)).toBeTruthy()
  })
})

describe('ForeshadowPanel 一致性体检', () => {
  const LINT_REPORT: ForeshadowLintReport = {
    totalChapters: 12,
    items: 5,
    planted: 3,
    hinted: 1,
    revealed: 1,
    longTerm: 1,
    findings: [
      {
        code: 'ordering',
        severity: 'high',
        foreshadowId: 'manual_demo_1',
        itemDesc: '神秘铜匣的钥匙',
        message: '回收章 001.md 早于埋设章 002.md，登记序颠倒',
        chapter: '001.md',
      },
      {
        code: 'stale',
        severity: 'medium',
        foreshadowId: 'manual_demo_2',
        itemDesc: '主角左臂旧伤的来历',
        message: '已悬置 11 章未回收（第 1 章埋设），考虑回收或标记长线',
        chapter: '001.md',
      },
    ],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
    vi.mocked(app.LintForeshadows).mockResolvedValue(LINT_REPORT)
  })

  it('点击「一致性体检」→ 概要行渲染统计数字', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText(/全书 12 章 · 登记 5 条/)).toBeTruthy()
    expect(screen.getByText(/已埋 3 \/ 暗示 1 \/ 回收 1（长线 1）/)).toBeTruthy()
    expect(vi.mocked(app.LintForeshadows)).toHaveBeenCalledTimes(1)
  })

  it('findings 渲染：severity Tag（重/中）+ 说明文本 + 章节引用', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText('重')).toBeTruthy()
    expect(screen.getByText('中')).toBeTruthy()
    expect(screen.getByText('回收章 001.md 早于埋设章 002.md，登记序颠倒')).toBeTruthy()
    expect(screen.getByText('已悬置 11 章未回收（第 1 章埋设），考虑回收或标记长线')).toBeTruthy()
    expect(screen.getAllByText('→ 001.md')).toHaveLength(2)
  })

  it('findings 为空 → 「未发现一致性问题」提示', async () => {
    vi.mocked(app.LintForeshadows).mockResolvedValue({ ...LINT_REPORT, findings: [] })
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText('未发现一致性问题')).toBeTruthy()
  })
})
