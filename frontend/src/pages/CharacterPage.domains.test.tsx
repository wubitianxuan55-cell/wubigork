// CharacterPage.domains.test.tsx — 域 handler 刻画测试（2026-10-04，第三轮审计
// §3.1 前置：hook 分域拆分前把测试网从 5 例补厚——锁各域的调用序/参数形状/
// 消息语义，拆分时零编辑复绿即等价证明）。
// 手法：vi.mock 两个 API 门面模块（components/novel/api/character 与
// api/characterlib）+ antd message 间谍 + 重型子组件桩 + 真实 store。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'

const charApi = vi.hoisted(() => ({
  getCharacters: vi.fn(),
  saveOrganization: vi.fn(),
  deleteOrganization: vi.fn(),
  setCharacterCareer: vi.fn(),
  removeCharacterCareer: vi.fn(),
  saveRelationship: vi.fn(),
  deleteRelationship: vi.fn(),
  generateCharacterFill: vi.fn(),
  generateCharacterPortrait: vi.fn(),
  mergeCharacters: vi.fn(),
  generateProtagonistRelations: vi.fn(),
}))
const libApi = vi.hoisted(() => ({
  listProjectCharacters: vi.fn(),
  associateToProject: vi.fn(),
  dissociateFromProject: vi.fn(),
  syncProjectCharacters: vi.fn(),
  importProjectCharacters: vi.fn(),
  previewProjectImport: vi.fn(),
  drawRandom: vi.fn(),
  setProjectState: vi.fn(),
}))
const messageSpies = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
}))

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>()
  return { ...actual, message: { ...actual.message, ...messageSpies } }
})
vi.mock('../components/novel/api/character', () => ({ ...charApi }))
vi.mock('../api/characterlib', () => ({ ...libApi }))

// 重型子组件桩（沿 CharacterPage.test 同款）
vi.mock('../components/RelationGraph', () => ({ default: () => <div data-testid="rel-graph-stub" /> }))
vi.mock('../components/novel/character/PortraitLightbox', () => ({ default: () => null }))
vi.mock('../components/characterlib/PortraitImg', () => ({ default: (p: { src?: string }) => (p.src ? <img src={p.src} alt="" /> : null) }))
// 组织弹窗桩：暴露 onSave/onDelete 入口供页面 handler 刻画
vi.mock('../components/novel/character/OrganizationEditModal', () => ({
  default: (p: { onSave: () => void }) => (
    <div data-testid="org-modal-stub">
      <button onClick={() => p.onSave()}>__org_save__</button>
    </div>
  ),
}))

import CharacterPage from './CharacterPage'
import { useAppStore } from '../stores/appStore'

const protagonist = {
  id: 'mc', name: '林晚', role_type: 'protagonist', status: 'Alive', gender: 'female',
  personality: '冷静', background: '', appearance: '', figure: '', motivation: '', arc: '',
}
const side = {
  id: 'sc', name: '沈青', role_type: 'antagonist', status: 'Alive', gender: 'male',
  personality: '', background: '', appearance: '', figure: '', motivation: '', arc: '',
}
const drawnA = { id: 'lib1', name: '云中鹤', roleType: 'antagonist', tags: ['剑修'] }
const drawnB = { id: 'lib2', name: '白衣客', roleType: '', tags: [] }

beforeEach(() => {
  vi.clearAllMocks()
  charApi.getCharacters.mockResolvedValue({ characters: [protagonist, side], organizations: [], relationships: [] })
  libApi.listProjectCharacters.mockResolvedValue([{ characterId: 'mc' }, { characterId: 'sc' }])
  libApi.syncProjectCharacters.mockResolvedValue(undefined)
  libApi.associateToProject.mockResolvedValue(undefined)
  libApi.importProjectCharacters.mockResolvedValue({ imported: 1, filled: 0, overwritten: 0 })
  libApi.previewProjectImport.mockResolvedValue({ conflicts: [] })
  libApi.drawRandom.mockResolvedValue([])
  charApi.generateProtagonistRelations.mockResolvedValue({ updated: 2, failed: 0, failNames: [] })
  charApi.mergeCharacters.mockResolvedValue(undefined)
  charApi.saveOrganization.mockResolvedValue(undefined)
  useAppStore.setState({ projectPath: 'C:\\proj\\demo' } as never)
})

afterEach(() => {
  document.body.innerHTML = ''
  window.localStorage.clear()
})

describe('CharacterPage 同步与回写', () => {
  it('「同步」逐序调用 syncProjectCharacters → loadData，并报成功话术', async () => {
    render(<CharacterPage />)
    await screen.findByText('林晚')
    fireEvent.click(screen.getByText('同步'))
    await waitFor(() => expect(libApi.syncProjectCharacters).toHaveBeenCalledTimes(1))
    await waitFor(() => {
      expect(messageSpies.success).toHaveBeenCalledWith('已把本书引用的角色同步到 characters.json')
    })
    // 挂载 1 次 + 同步后 reload 1 次
    expect(charApi.getCharacters.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('存在未入库角色时「同步」禁用且迁移横幅出现；「一次性迁移」零冲突直写', async () => {
    libApi.listProjectCharacters.mockResolvedValue([{ characterId: 'mc' }])
    render(<CharacterPage />)
    await screen.findByText(/检测到 1 个旧项目角色尚未进入角色库/)
    const syncBtn = (screen.getByText('同步') as HTMLElement).closest('button') as HTMLButtonElement
    expect(syncBtn.disabled).toBe(true)
    fireEvent.click(screen.getByText('一次性迁移'))
    // 零冲突：preview 后直接 importProjectCharacters({})，不弹确认弹窗
    await waitFor(() => expect(libApi.previewProjectImport).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(libApi.importProjectCharacters).toHaveBeenCalledWith({}))
    expect(screen.queryByText('回写角色库：确认要覆盖的设定')).toBeNull()
    await waitFor(() => {
      expect(messageSpies.success).toHaveBeenCalledWith('回写完成：新迁入 1 个（本书此后只引用角色库）')
    })
  })
})

describe('CharacterPage 抽卡', () => {
  async function openDraw() {
    render(<CharacterPage />)
    await screen.findByText('林晚')
    // 工具栏「抽卡」与弹窗内「抽卡」同名双匹配：先点首个（工具栏），后续操作锚定弹窗
    fireEvent.click(screen.getAllByText(/^抽\s*卡$/)[0])
    await screen.findByText('从角色库抽卡')
    return within((screen.getByText('从角色库抽卡').closest('.ant-modal') as HTMLElement))
  }

  it('「抽卡」以默认参数（5, 全部性别, 无标签, 非仅可聊天）调用 drawRandom', async () => {
    const modal = await openDraw()
    fireEvent.click(modal.getByText(/^抽\s*卡$/))
    await waitFor(() => expect(libApi.drawRandom).toHaveBeenCalledWith(5, '', '', false))
  })

  it('单个加入：associateToProject 按 roleType 落位（缺省回 supporting），先关联再同步', async () => {
    libApi.drawRandom.mockResolvedValue([drawnA, drawnB])
    // 可变引用表：加入后 loadRefs 反映新引用 → 卡片翻「已加入」Tag，第二次点击必是下一张卡
    const refs = ['mc', 'sc']
    libApi.listProjectCharacters.mockImplementation(async () => refs.map((id) => ({ characterId: id })))
    libApi.associateToProject.mockImplementation(async (id: string) => { refs.push(id) })
    const modal = await openDraw()
    fireEvent.click(modal.getByText(/^抽\s*卡$/))
    await modal.findByText('云中鹤')
    fireEvent.click(modal.getAllByText('加入')[0])
    await waitFor(() => expect(libApi.associateToProject).toHaveBeenCalledWith('lib1', 'antagonist'))
    await waitFor(() => expect(messageSpies.success).toHaveBeenCalledWith('「云中鹤」已加入本书'))
    fireEvent.click(modal.getAllByText('加入')[0])
    await waitFor(() => expect(libApi.associateToProject).toHaveBeenCalledWith('lib2', 'supporting'))
    // 关联后立刻同步到 characters.json
    expect(libApi.syncProjectCharacters).toHaveBeenCalled()
  })

  it('全部加入：只加未入库的 pending（已加入不重复调用），完成后清空结果', async () => {
    libApi.drawRandom.mockResolvedValue([drawnA, drawnB])
    libApi.listProjectCharacters.mockResolvedValue([{ characterId: 'lib1' }, { characterId: 'mc' }, { characterId: 'sc' }])
    const modal = await openDraw()
    fireEvent.click(modal.getByText(/^抽\s*卡$/))
    await modal.findByText('云中鹤')
    // 云中鹤已入库 → pending 仅白衣客
    expect(modal.getByText(/全部加入本书（1）/)).toBeTruthy()
    fireEvent.click(modal.getByText(/全部加入本书/))
    await waitFor(() => expect(libApi.associateToProject).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(messageSpies.success).toHaveBeenCalledWith('已加入 1 个角色'))
    // drawResult 清空 → 回空态话术
    await modal.findByText(/抽到的角色会显示在这里/)
  })
})

describe('CharacterPage AI 主角关系', () => {
  it('「剩余全部」不确认直接跑：mode=missing + 成功话术 + 刷新', async () => {
    render(<CharacterPage />)
    await screen.findByText('林晚')
    fireEvent.click(screen.getByTestId('char-gen-relations'))
    fireEvent.click(await screen.findByText('剩余全部（只补空白）'))
    await waitFor(() => expect(charApi.generateProtagonistRelations).toHaveBeenCalledWith('missing', ''))
    await waitFor(() => expect(messageSpies.success).toHaveBeenCalledWith('已生成 2 位角色的主角关系'))
    expect(charApi.getCharacters.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('「全部角色」走 Modal.confirm，确认后才跑 mode=all', async () => {
    render(<CharacterPage />)
    await screen.findByText('林晚')
    fireEvent.click(screen.getByTestId('char-gen-relations'))
    fireEvent.click(await screen.findByText('全部角色（覆盖重写）'))
    expect((await screen.findAllByText('AI 重写全部角色的主角关系？')).length).toBeGreaterThanOrEqual(1)
    expect(charApi.generateProtagonistRelations).not.toHaveBeenCalled()
    fireEvent.click((await screen.findAllByText('开始生成'))[0])
    await waitFor(() => expect(charApi.generateProtagonistRelations).toHaveBeenCalledWith('all', ''))
  })

  it('updated=0 走 info 引导话术；failed>0 走 warning 并点名前三个失败者', async () => {
    charApi.generateProtagonistRelations.mockResolvedValue({ updated: 0, failed: 0, failNames: [] })
    render(<CharacterPage />)
    await screen.findByText('林晚')
    fireEvent.click(screen.getByTestId('char-gen-relations'))
    fireEvent.click(await screen.findByText('剩余全部（只补空白）'))
    await waitFor(() => {
      expect(messageSpies.info).toHaveBeenCalledWith('没有需要生成的角色（剩余全部=已都有主角关系；或仅主角本人）')
    })

    charApi.generateProtagonistRelations.mockClear()
    charApi.generateProtagonistRelations.mockResolvedValue({ updated: 1, failed: 2, failNames: ['云中鹤', '白衣客', '第三者', '第四者'] })
    fireEvent.click(screen.getByTestId('char-gen-relations'))
    fireEvent.click(await screen.findByText('剩余全部（只补空白）'))
    await waitFor(() => {
      expect(messageSpies.warning).toHaveBeenCalledWith(
        expect.stringContaining('更新 1 位，失败 2 位'),
      )
    })
    // failNames 只点名前三个 + 省略号
    expect(messageSpies.warning).toHaveBeenCalledWith(expect.stringContaining('云中鹤、白衣客、第三者…'))
  })

  it('抽屉内单角色入口：mode=one + 点名该角色（主角卡无此钮）', async () => {
    render(<CharacterPage />)
    const card = await screen.findByText('沈青')
    fireEvent.click(card.closest('[class*="char-card"]') || card)
    await screen.findByText('职业体系')
    fireEvent.click(screen.getByTestId('char-detail-gen-relation'))
    await waitFor(() => expect(charApi.generateProtagonistRelations).toHaveBeenCalledWith('one', '沈青'))
  })
})

describe('CharacterPage 合并（A 并入 B 保留 B）', () => {
  it('未入库角色的「合并」：mergeCharacters(目标B, 当前A) 参数序锁定', async () => {
    libApi.listProjectCharacters.mockResolvedValue([{ characterId: 'mc' }])
    render(<CharacterPage />)
    const card = await screen.findByText('沈青')
    fireEvent.click(card.closest('[class*="char-card"]') || card)
    await screen.findByText('职业体系')
    fireEvent.click(screen.getByText('合并'))
    await screen.findByText('合并「沈青」到其他角色')
    // 选择保留目标 = 林晚（主角）
    const select = screen.getByText('选择要保留的角色…').closest('.ant-select')
    fireEvent.mouseDown(select!.querySelector('.ant-select-selector')!)
    const opt = await screen.findByText('林晚（主角）', { selector: '.ant-select-item-option-content' })
    fireEvent.click(opt)
    // 两字按钮 antd 自动插空格（「合 并」）：按 \s* 正则匹配 OK 钮
    fireEvent.click(screen.getByText(/^合\s*并$/, { selector: '.ant-modal .ant-btn-primary span' }))
    await waitFor(() => expect(charApi.mergeCharacters).toHaveBeenCalledWith('mc', 'sc'))
    await waitFor(() => {
      expect(messageSpies.success).toHaveBeenCalledWith('已合并：空缺信息已补充，关系与组织引用已重定向')
    })
  })
})

describe('CharacterPage 组织保存', () => {
  it('新建组织 → 弹窗 onSave → saveOrganization 落新组织并报成功', async () => {
    render(<CharacterPage />)
    await screen.findByText('林晚')
    // 组织 tab 的 label 是「组织 (N)」带计数：正则匹配并锚定 .ant-tabs-tab
    fireEvent.click(screen.getAllByText(/^组织 \(/).find((el) => el.closest('.ant-tabs-tab'))!)
    fireEvent.click(screen.getByText('新建组织'))
    await screen.findByTestId('org-modal-stub')
    fireEvent.click(screen.getByText('__org_save__'))
    await waitFor(() => {
      expect(charApi.saveOrganization).toHaveBeenCalledWith(
        expect.objectContaining({ name: '新组织' }),
      )
    })
    await waitFor(() => expect(messageSpies.success).toHaveBeenCalledWith('组织已保存'))
  })
})
