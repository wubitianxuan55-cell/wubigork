import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  canonicalBoards, normalizeManifests, resolveBoardIcon,
  loadBoardManifests, subscribeBoards, getActiveBoards, resetActiveBoardsForTest,
} from './manifests'
import { deriveLauncherModules, LAUNCHER_DESC } from './launcher'
import { GetBoardManifests } from '../wailsjsCompat'

// 数据源 seam（§5.3 前端侧）：mock 掉 wailsjsCompat 的 GetBoardManifests，
// 隔离后端提供者，验证「启动器卡清单随 manifest 派生（v4.4 起 11 卡，含 weixin；knowledge 过滤）」。
vi.mock('../wailsjsCompat', () => ({
  GetBoardManifests: vi.fn(),
}))

// vi.mock 不改 TS 静态类型，用 loosely typed 引用以便测试注入纯对象 fixture。
const getBoardManifestsMock = GetBoardManifests as unknown as {
  mockResolvedValue(v: unknown): void
  mockReset(): void
}

// 后端 GetBoardManifests 契约形态（对齐 internal/app/board/builtins.go）：
// 12 个业务板块（含 cost 造价数据库 + code 编程 + D7 knowledge），无 home 壳层，
// v4.4 起 weixin.page="WeixinPage"、inMenu=true；v4.48 label=青鸟（原微信助手）。
const BACKEND_FIXTURE = [
  { id: 'chat', label: '聊天', icon: 'MessageOutlined', page: 'ChatPage', lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+1', menuOrder: 1, inMenu: true, featureModel: 'chat' },
  { id: 'novel', label: '小说', icon: 'ReadOutlined', page: 'NovelPage', lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+2', menuOrder: 2, inMenu: true, breadcrumb: { anchorTo: 'project' }, featureModel: 'novel' },
  { id: 'imagegen', label: '绘梦', icon: 'PictureOutlined', page: 'ImageGenPage', lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+3', menuOrder: 3, inMenu: true, featureModel: 'imagegen' },
  { id: 'gaea', label: '办公', icon: 'ToolOutlined', page: 'GaeaPage', lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+4', menuOrder: 4, inMenu: true, featureModel: 'gaea' },
  { id: 'cost', label: '造价数据库', icon: 'AccountBookOutlined', page: 'CostLibraryPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 5, inMenu: true, featureModel: 'cost' },
  { id: 'code', label: '编程', icon: 'CodeOutlined', page: 'ProgrammingPage', lazy: true, keepAlive: true, layout: 'full', menuOrder: 6, inMenu: true },
  { id: 'memoryhub', label: '记忆中枢', icon: 'DatabaseOutlined', page: 'MemoryHubPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 7, inMenu: true },
  { id: 'modelcenter', label: '模型中心', icon: 'ApiOutlined', page: 'ModelCenterPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 8, inMenu: true },
  { id: 'characterlib', label: '角色库', icon: 'TeamOutlined', page: 'CharacterLibraryPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 9, inMenu: true, featureModel: 'characterlib' },
  { id: 'settings', label: '设置', icon: 'SettingOutlined', page: 'SettingsPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 10, inMenu: false },
  { id: 'weixin', label: '青鸟', icon: 'WechatOutlined', page: 'WeixinPage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 11, inMenu: true },
  { id: 'knowledge', label: '知识库', icon: 'BookOutlined', page: 'KnowledgePage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 8, inMenu: true, featureModel: 'knowledge' },
]

describe('deriveLauncherModules（启动器清单纯函数）', () => {
  it('静态清单 → 11 卡，顺序与 LAUNCHER_DESC 对齐（v4.4 起 weixin 进清单；不含 home）', () => {
    const modules = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC)
    expect(modules.map((m) => m.key)).toEqual([
      'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib', 'settings', 'weixin',
    ])
    expect(modules.map((m) => m.key)).not.toContain('home')
  })

  it('静态 11 卡 desc 全部取 LAUNCHER_DESC 文案（name/icon 取 manifest）', () => {
    const modules = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC)
    for (const m of modules) {
      expect(LAUNCHER_DESC[m.key], m.key).toBeDefined()
      expect(m.desc, m.key).toBe(LAUNCHER_DESC[m.key])
    }
    expect(modules[0]).toEqual({ key: 'chat', name: '聊天', desc: LAUNCHER_DESC.chat, icon: 'MessageOutlined' })
  })

  it('settings 保留：inMenu=false 但 id===\'settings\' 隐式入口', () => {
    expect(canonicalBoards.find((b) => b.id === 'settings')?.inMenu).toBe(false)
    const keys = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC).map((m) => m.key)
    expect(keys).toContain('settings')
  })

  it('后端合并清单（normalizeManifests 后）→ 11 卡（knowledge 并入记忆中枢被过滤），desc 兜底 = label', () => {
    const merged = normalizeManifests(BACKEND_FIXTURE)
    const modules = deriveLauncherModules(merged, LAUNCHER_DESC)
    expect(modules.map((m) => m.key)).toEqual([
      'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib', 'settings', 'weixin',
    ])
    expect(modules.map((m) => m.key)).not.toContain('knowledge')
  })

  it('icon 字段透传图标注册表名；未知名 resolveBoardIcon 返回 null（渲染层兜底）', () => {
    const withUnknownIcon = [
      { ...canonicalBoards[1], id: 'legacy', label: '旧模块', icon: 'NoSuchIconOutlined', inMenu: true },
    ]
    const modules = deriveLauncherModules(withUnknownIcon, LAUNCHER_DESC)
    expect(modules[0].icon).toBe('NoSuchIconOutlined')
    expect(resolveBoardIcon('NoSuchIconOutlined')).toBeNull()
  })

  it('不修改入参列表（filter 新数组 + sort 不污染调用方）', () => {
    const input = [...canonicalBoards]
    const before = input.map((b) => b.id)
    deriveLauncherModules(input, LAUNCHER_DESC)
    expect(input.map((b) => b.id)).toEqual(before)
  })

  it('S2.1 双首页：按空间过滤（shared + 当前空间），工位不含乐园板块', () => {
    const work = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC, 'work')
    const workKeys = work.map((m) => m.key)
    expect(workKeys).toContain('gaea')
    expect(workKeys).toContain('cost')
    expect(workKeys).toContain('memoryhub')
    expect(workKeys).not.toContain('novel')
    expect(workKeys).not.toContain('imagegen')
    expect(workKeys).not.toContain('characterlib')
    expect(workKeys).not.toContain('code') // 编程独立窗口不进双首页
    expect(workKeys).not.toContain('chat') // P1：对话降为对话流，工位首页不出聊天卡
  })

  it('S2.1 双首页：乐园不含工位板块，含小说/绘梦/角色', () => {
    const play = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC, 'play')
    const playKeys = play.map((m) => m.key)
    expect(playKeys).toContain('novel')
    expect(playKeys).toContain('imagegen')
    expect(playKeys).toContain('characterlib')
    expect(playKeys).toContain('chat') // 乐园=会客厅沉浸对话
    expect(playKeys).not.toContain('gaea')
    expect(playKeys).not.toContain('cost')
    expect(playKeys).not.toContain('memoryhub')
    expect(playKeys).not.toContain('code')
  })

  it('不传 space = 全量（旧调用语义不变）', () => {
    const all = deriveLauncherModules(canonicalBoards, LAUNCHER_DESC)
    expect(all.map((m) => m.key)).toContain('novel')
    expect(all.map((m) => m.key)).toContain('gaea')
  })
})

describe('launcher 订阅联动（loadBoardManifests 通知 → 派生结果变化）', () => {
  beforeEach(() => {
    resetActiveBoardsForTest()
    getBoardManifestsMock.mockReset()
  })

  it('后端合并成功后 getActiveBoards 变化 → deriveLauncherModules 结果含 code（无需渲染组件）', async () => {
    // 加载前：静态 fallback → 10 卡，含 cost/code
    expect(deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC).map((m) => m.key)).toContain('code')
    expect(deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC).map((m) => m.key)).toContain('cost')
    getBoardManifestsMock.mockResolvedValue(BACKEND_FIXTURE)
    const notified = vi.fn()
    const unsub = subscribeBoards(notified)
    await loadBoardManifests()
    unsub()
    // 加载后：活动清单替换 → 订阅者收到通知，派生结果 11 卡含 cost/code（v4.4 起 +weixin）
    expect(notified).toHaveBeenCalledTimes(1)
    const keys = deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC).map((m) => m.key)
    expect(keys).toHaveLength(11)
    expect(keys).toContain('code')
    expect(keys).toContain('cost')
  })
})
