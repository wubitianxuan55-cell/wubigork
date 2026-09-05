// ConsistencyPanel.test.tsx — 深检模式误报缓解 UI 单测：三档分级标签 / 原因徽标 /
// 单条忽略与恢复（localStorage 按项目记忆）
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go
vi.mock('../../../src/wailsjsCompat', () => ({
  CheckConsistency: vi.fn().mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' }),
  CheckConsistencyDeep: vi.fn(),
}))

import ConsistencyPanel from './ConsistencyPanel'
import { useAppStore } from '../../stores/appStore'
import { CheckConsistencyDeep } from '../../../src/wailsjsCompat'
import { clearIgnoredIssues, loadIgnoredFingerprints } from './consistencyIgnore'

const projPath = 'C:/books/深检项目'

function issueOf(overrides: Record<string, unknown>) {
  return {
    severity: 'error',
    category: 'status',
    entity_name: '林晚',
    description: '林晚 在第2章已死亡，但第3章仍以存活状态出场',
    location: '第3章',
    evidence: '第2章状态卡 status=dead；第3章状态卡 status=alive',
    suggestion: '确认是否为复活/回忆/幻象并在文中明确交代',
    branch: '',
    source: 'ai',
    ...overrides,
  }
}

beforeEach(() => {
  localStorage.clear()
  clearIgnoredIssues(projPath)
  useAppStore.setState({ projectOpen: true, projectPath: projPath })
  vi.mocked(CheckConsistencyDeep).mockReset()
})

describe('深检模式三档分级', () => {
  it('分级标签冲突/疑似/提示 + 原因徽标按 reason 渲染', async () => {
    vi.mocked(CheckConsistencyDeep).mockResolvedValue({
      issues: [
        issueOf({}), // error 无标注 → 冲突
        issueOf({ entity_name: '玄铁剑', category: 'item', severity: 'warning', description: '物品去留不明', reason: 'unexplained' }), // → 疑似 + 缺交代
        issueOf({ entity_name: '', severity: 'info', description: '同区域表述差异', reason: 'wording' }), // → 提示 + 措辞差异
      ],
      total_issues: 3,
      summary: '发现 3 个问题',
      chapters_scanned: 3,
      ai_available: true,
      ai_note: '',
    })

    render(<ConsistencyPanel />)
    await fireEvent.click(await screen.findByTestId('consistency-deep-run'))

    const levels = await screen.findAllByTestId('consistency-issue-level')
    // 测试环境无 LocaleProvider，非响应式 t 回退英文键值
    expect(levels.map((el) => el.textContent)).toEqual(['Conflict', 'Suspect', 'Hint'])
    const reasons = screen.getAllByTestId('consistency-issue-reason')
    expect(reasons).toHaveLength(2)
    expect(reasons[0].textContent).toBe('Lacks explicit explanation')
    expect(reasons[1].textContent).toBe('Wording difference (not a conflict)')
    // 来源徽标仍然渲染
    expect(screen.getAllByTestId('consistency-issue-source')).toHaveLength(3)
  })

  it('被忽略告警从列表消失但以计数横幅保持可见，可一键恢复', async () => {
    vi.mocked(CheckConsistencyDeep).mockResolvedValue({
      issues: [issueOf({}), issueOf({ entity_name: '玄铁剑', category: 'item', severity: 'warning', description: '物品去留不明', reason: 'unexplained' })],
      total_issues: 2,
      summary: '发现 2 个问题',
      chapters_scanned: 2,
      ai_available: true,
      ai_note: '',
    })

    render(<ConsistencyPanel />)
    await fireEvent.click(await screen.findByTestId('consistency-deep-run'))
    await screen.findAllByTestId('consistency-issue-level')
    expect(screen.queryByTestId('consistency-ignored-banner')).toBeNull()

    // 忽略第一条 → 列表剩 1 条，横幅计数 1，localStorage 有指纹
    const ignoreBtns = screen.getAllByTestId('consistency-issue-ignore')
    fireEvent.click(ignoreBtns[0])
    expect(screen.getAllByTestId('consistency-issue-level')).toHaveLength(1)
    expect(screen.getByTestId('consistency-ignored-banner').textContent).toContain('1')
    expect(loadIgnoredFingerprints(projPath)).toHaveLength(1)

    // 恢复显示 → 两条都回来，横幅消失
    fireEvent.click(screen.getByTestId('consistency-ignored-restore'))
    await waitFor(() => {
      expect(screen.getAllByTestId('consistency-issue-level')).toHaveLength(2)
    })
    expect(screen.queryByTestId('consistency-ignored-banner')).toBeNull()
    expect(loadIgnoredFingerprints(projPath)).toEqual([])
  })

  it('全部忽略时显示可恢复空态（不伪装「全部通过」）', async () => {
    vi.mocked(CheckConsistencyDeep).mockResolvedValue({
      issues: [issueOf({})],
      total_issues: 1,
      summary: '发现 1 个问题',
      chapters_scanned: 1,
      ai_available: true,
      ai_note: '',
    })

    render(<ConsistencyPanel />)
    await fireEvent.click(await screen.findByTestId('consistency-deep-run'))
    fireEvent.click(await screen.findByTestId('consistency-issue-ignore'))
    expect(screen.getByTestId('consistency-all-ignored').textContent).toContain('1')
    expect(screen.queryByText('全部通过，未发现一致性问题')).toBeNull()
  })
})
