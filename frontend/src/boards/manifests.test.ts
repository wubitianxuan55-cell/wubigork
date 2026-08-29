import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  canonicalBoards, menuBoards, navigateWhitelist, shortcutMap,
  homeBoard, projectAnchorId, getBoard, boardLabel, resolveBoardIcon,
  normalizeManifests, deriveMenuBoards, deriveNavigateWhitelist,
  deriveShortcutMap, deriveHomeBoard, deriveBoard, deriveProjectAnchorId,
  loadBoardManifests, subscribeBoards, getActiveBoards,
  getActiveMenuBoards, getActiveNavigateWhitelist, getActiveShortcutMap,
  getActiveHomeBoard, getActiveBoard,
  resetActiveBoardsForTest,
} from './manifests'
import { GetBoardManifests } from '../wailsjsCompat'
import { registerPage, getPageComponent, listRegisteredPages, clearPageRegistry } from './pageRegistry'

// 数据源 seam（§5.3 前端侧）：mock 掉 wailsjsCompat 的 GetBoardManifests，
// 隔离后端提供者，验证「后端优先 / 失败回退静态 / 差集归一」。
vi.mock('../wailsjsCompat', () => ({
  GetBoardManifests: vi.fn(),
}))

// vi.mock 不改 TS 静态类型（GetBoardManifests 仍是 board.Manifest[]），
// 用 loosely typed 引用以便测试注入纯对象 fixture。
const getBoardManifestsMock = GetBoardManifests as unknown as {
  mockResolvedValue(v: unknown): void
  mockRejectedValue(e: unknown): void
  mockReset(): void
}

// 后端 GetBoardManifests 契约形态（对齐 internal/app/board/builtins.go）：
// 11 个业务板块（含 cost 造价数据库 + D7 knowledge），无 home 壳层，
// weixin.page=""、label=微信助手。
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
  { id: 'weixin', label: '微信助手', icon: 'WechatOutlined', page: '', lazy: false, keepAlive: true, layout: 'padded', menuOrder: 11, inMenu: false },
  { id: 'knowledge', label: '知识库', icon: 'BookOutlined', page: 'KnowledgePage', lazy: true, keepAlive: true, layout: 'padded', menuOrder: 8, inMenu: true, featureModel: 'knowledge' },
]

// 3.0 附 B 收敛映射回归：manifest 派生结果必须与旧 MainLayout 硬编码一致（像素级）。

describe('menuBoards（附 B #4：filter(inMenu) + sort(menuOrder)）', () => {
  it('菜单顺序与现状一致：首页 → 聊天 → 小说 → 绘梦 → 办公 → 造价数据库 → 编程 → 记忆中枢 → 模型中心 → 角色库 → 微信助手（v4.4 进菜单）', () => {
    expect(menuBoards.map((b) => b.id)).toEqual([
      'home', 'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib', 'weixin',
    ])
  })

  it('菜单文案与现状一致（首页/聊天/小说/绘梦/办公/造价数据库/编程/记忆中枢/模型中心/角色库/微信助手）', () => {
    expect(menuBoards.map((b) => b.label)).toEqual([
      '首页', '聊天', '小说', '绘梦', '办公', '造价数据库', '编程', '记忆中枢', '模型中心', '角色库', '微信助手',
    ])
  })

  it('菜单图标名与现状一致（antd 图标注册表可解析）', () => {
    const expected = ['HomeOutlined', 'MessageOutlined', 'ReadOutlined', 'PictureOutlined',
      'ToolOutlined', 'AccountBookOutlined', 'CodeOutlined', 'DatabaseOutlined', 'ApiOutlined', 'TeamOutlined', 'WechatOutlined']
    expect(menuBoards.map((b) => b.icon)).toEqual(expected)
    for (const b of menuBoards) {
      expect(resolveBoardIcon(b.icon), `icon ${b.icon} 可解析`).not.toBeNull()
    }
  })

  it('settings 不进菜单（inMenu=false）；weixin v4.4 起进菜单（inMenu=true）', () => {
    expect(menuBoards.map((b) => b.id)).not.toContain('settings')
    expect(menuBoards.map((b) => b.id)).toContain('weixin')
  })
})

describe('navigateWhitelist（附 B #2：manifest 派生导航白名单）', () => {
  it('与旧 allPageKeys 一致：9 个业务板块，不含 home/settings（v4.4 加 weixin）', () => {
    expect(navigateWhitelist).toEqual([
      'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib', 'weixin',
    ])
  })
})

describe('shortcutMap（附 B #6：Ctrl+1~4 显式声明）', () => {
  it('ctrl+1~4 → chat/novel/imagegen/gaea（与旧顺序绑定一致）', () => {
    expect(shortcutMap['ctrl+1']).toBe('chat')
    expect(shortcutMap['ctrl+2']).toBe('novel')
    expect(shortcutMap['ctrl+3']).toBe('imagegen')
    expect(shortcutMap['ctrl+4']).toBe('gaea')
  })

  it('快捷键不依赖菜单数组顺序（声明式）', () => {
    const declared = canonicalBoards.filter((b) => b.shortcut)
    expect(declared.map((b) => b.shortcut)).toEqual(['ctrl+1', 'ctrl+2', 'ctrl+3', 'ctrl+4'])
  })
})

describe('layout / breadcrumb / home（附 B #8/#9/#10/#11）', () => {
  it('chat/gaea/code = full（全出血），其余 = padded', () => {
    expect(getBoard('chat')?.layout).toBe('full')
    expect(getBoard('gaea')?.layout).toBe('full')
    expect(getBoard('code')?.layout).toBe('full')
    for (const id of ['novel', 'imagegen', 'cost', 'memoryhub', 'modelcenter', 'characterlib', 'settings']) {
      expect(getBoard(id)?.layout, id).toBe('padded')
    }
  })

  it('面包屑项目锚点 = novel（novel 声明自己是项目锚点）', () => {
    expect(projectAnchorId).toBe('novel')
    expect(getBoard('novel')?.breadcrumb?.anchorTo).toBe('novel')
  })

  it('home 板块 isHome=true 且唯一', () => {
    expect(homeBoard.id).toBe('home')
    expect(homeBoard.isHome).toBe(true)
    expect(canonicalBoards.filter((b) => b.isHome)).toHaveLength(1)
  })

  it('settings 隐式入口：inMenu=false 但仍可达（附 B #12）', () => {
    expect(getBoard('settings')?.inMenu).toBe(false)
    expect(getBoard('settings')?.page).toBe('SettingsPage')
  })
})

describe('boardLabel（附 B #7：pageLabels = manifest.label）', () => {
  it('按 id 取显示名', () => {
    expect(boardLabel('chat')).toBe('聊天')
    expect(boardLabel('novel')).toBe('小说')
    expect(boardLabel('home')).toBe('首页')
  })
  it('未知 id 返回自身（安全退化）', () => {
    expect(boardLabel('__missing__')).toBe('__missing__')
  })
})


describe('normalizeManifests（板块差集归一：后端清单 + 前端 home 壳层）', () => {
  it('差集 #2：过滤后端 knowledge（已并入记忆中枢，一级导航不单列），补前端 home 壳层', () => {
    const merged = normalizeManifests(BACKEND_FIXTURE)
    const ids = merged.map((b) => b.id)
    expect(ids).not.toContain('knowledge')
    expect(ids[0]).toBe('home')
    expect(deriveHomeBoard(merged)?.id).toBe('home')
    expect(merged.filter((b) => b.isHome)).toHaveLength(1)
    expect(merged).toHaveLength(12)
  })

  it('差集 #3：weixin 以后端为准（page=""，label=微信助手，inMenu=false）', () => {
    const merged = normalizeManifests(BACKEND_FIXTURE)
    const wx = deriveBoard(merged, 'weixin')
    expect(wx?.page).toBe('')
    expect(wx?.label).toBe('微信助手')
    expect(wx?.inMenu).toBe(false)
  })

  it('重叠 id 后端字段优先：菜单顺序/文案与静态一致（knowledge 被过滤，不尾随）', () => {
    const menu = deriveMenuBoards(normalizeManifests(BACKEND_FIXTURE))
    expect(menu.map((b) => b.id)).toEqual([
      'home', 'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib',
    ])
    expect(menu.map((b) => b.label)).toEqual([
      '首页', '聊天', '小说', '绘梦', '办公', '造价数据库', '编程', '记忆中枢', '模型中心', '角色库',
    ])
  })

  it('导航白名单 = 后端清单 + home（knowledge 已并入记忆中枢不单列）', () => {
    expect(deriveNavigateWhitelist(normalizeManifests(BACKEND_FIXTURE))).toEqual([
      'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib',
    ])
  })

  it('快捷键 ctrl+1~4 后端声明保留（像素回归）', () => {
    const sc = deriveShortcutMap(normalizeManifests(BACKEND_FIXTURE))
    expect(sc['ctrl+1']).toBe('chat')
    expect(sc['ctrl+2']).toBe('novel')
    expect(sc['ctrl+3']).toBe('imagegen')
    expect(sc['ctrl+4']).toBe('gaea')
  })

  it('布局：chat/gaea=full，其余 padded', () => {
    const merged = normalizeManifests(BACKEND_FIXTURE)
    expect(deriveBoard(merged, 'chat')?.layout).toBe('full')
    expect(deriveBoard(merged, 'gaea')?.layout).toBe('full')
    expect(deriveBoard(merged, 'code')?.layout).toBe('full')
    for (const id of ['novel', 'imagegen', 'cost', 'memoryhub', 'modelcenter', 'characterlib', 'settings']) {
      expect(deriveBoard(merged, id)?.layout, id).toBe('padded')
    }
  })

  it('面包屑项目锚点归到 novel（后端 anchorTo=project 取声明板块 id，值不参与）', () => {
    expect(deriveProjectAnchorId(normalizeManifests(BACKEND_FIXTURE))).toBe('novel')
  })

  it('knowledge 已被过滤：一级导航不并入（记忆中枢承载），图标注册表仍可解析', () => {
    expect(resolveBoardIcon('BookOutlined')).not.toBeNull()
    expect(deriveBoard(normalizeManifests(BACKEND_FIXTURE), 'knowledge')).toBeUndefined()
  })

  it('后端自带 isHome 板块时不重复补壳层', () => {
    const withHome = [
      { id: 'home', label: '首页', icon: 'HomeOutlined', page: 'home', lazy: false, keepAlive: true, layout: 'padded', menuOrder: 0, inMenu: true, isHome: true },
      ...BACKEND_FIXTURE,
    ]
    const merged = normalizeManifests(withHome)
    expect(merged.filter((b) => b.isHome)).toHaveLength(1)
    expect(merged[0].id).toBe('home')
  })

  it('空输入 → []（回退决策在 loadBoardManifests 层）', () => {
    expect(normalizeManifests([])).toEqual([])
  })
})

describe('loadBoardManifests（数据源 seam：后端优先 / fail-closed 回退静态）', () => {
  beforeEach(() => {
    resetActiveBoardsForTest()
    getBoardManifestsMock.mockReset()
  })

  it('后端成功 → 合并清单生效（home 壳层补位，knowledge 过滤）并通知订阅者', async () => {
    getBoardManifestsMock.mockResolvedValue(BACKEND_FIXTURE)
    const cb = vi.fn()
    const unsub = subscribeBoards(cb)
    const result = await loadBoardManifests()
    unsub()
    expect(result.map((b) => b.id)).toContain('home')
    expect(result.map((b) => b.id)).not.toContain('knowledge')
    expect(getActiveBoards()).toBe(result)
    expect(cb).toHaveBeenCalledTimes(1)
  })

  it('后端失败 → fail-closed 回退静态 canonicalBoards（壳层永远可用）', async () => {
    getBoardManifestsMock.mockRejectedValue(new Error('window.go 未注入（浏览器 dev mock）'))
    const result = await loadBoardManifests()
    expect(result).toBe(canonicalBoards)
    expect(getActiveBoards()).toBe(canonicalBoards)
    expect(getActiveMenuBoards().map((b) => b.id)).toEqual(menuBoards.map((b) => b.id))
  })

  it('后端返回空数组/非数组 → 回退静态清单', async () => {
    getBoardManifestsMock.mockResolvedValue([])
    expect(await loadBoardManifests()).toBe(canonicalBoards)
    getBoardManifestsMock.mockResolvedValue(undefined)
    expect(await loadBoardManifests()).toBe(canonicalBoards)
  })

  it('活动派生视图跟随合并清单（消费者侧：菜单/白名单/快捷键/布局）', async () => {
    getBoardManifestsMock.mockResolvedValue(BACKEND_FIXTURE)
    await loadBoardManifests()
    expect(getActiveMenuBoards().map((b) => b.id)).toEqual([
      'home', 'chat', 'novel', 'imagegen', 'gaea', 'cost', 'code', 'memoryhub', 'modelcenter', 'characterlib',
    ])
    expect(getActiveNavigateWhitelist()).not.toContain('knowledge')
    expect(getActiveShortcutMap()['ctrl+4']).toBe('gaea')
    expect(getActiveBoard('knowledge')).toBeUndefined()
    expect(getActiveHomeBoard().id).toBe('home')
  })

  it('订阅退订后不再收到通知', async () => {
    getBoardManifestsMock.mockResolvedValue(BACKEND_FIXTURE)
    const cb = vi.fn()
    const unsub = subscribeBoards(cb)
    await loadBoardManifests()
    unsub()
    const before = cb.mock.calls.length
    await loadBoardManifests()
    expect(cb.mock.calls.length).toBe(before)
  })

  it('resetActiveBoardsForTest 恢复静态基线（测试隔离）', async () => {
    getBoardManifestsMock.mockResolvedValue(BACKEND_FIXTURE)
    await loadBoardManifests()
    // knowledge 过滤后合并清单与静态长度相同（均为 10），但引用不同（home 壳层 + 后端字段优先）
    expect(getActiveBoards()).not.toBe(canonicalBoards)
    resetActiveBoardsForTest()
    expect(getActiveBoards()).toBe(canonicalBoards)
  })
})

describe('canonicalBoards 一致性', () => {
  it('板块 id 唯一', () => {
    const ids = canonicalBoards.map((b) => b.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
  it('菜单 menuOrder 唯一且连续', () => {
    const orders = menuBoards.map((b) => b.menuOrder)
    expect(new Set(orders).size).toBe(orders.length)
    expect(orders).toEqual([...orders].sort((a, b) => (a ?? 0) - (b ?? 0)))
  })
  it('全部板块 icon 名可被图标注册表解析', () => {
    for (const b of canonicalBoards) {
      expect(resolveBoardIcon(b.icon), `icon ${b.icon}（${b.id}）`).not.toBeNull()
    }
  })
  it('每个非 home 板块都声明了 page（PageRegistry 查找键）', () => {
    for (const b of canonicalBoards.filter((x) => !x.isHome)) {
      expect(b.page.length, `${b.id}.page 非空`).toBeGreaterThan(0)
    }
  })
})

describe('PageRegistry（附 B #3/#5：main.tsx 集中注册）', () => {
  it('registerPage/getPageComponent 往返', () => {
    clearPageRegistry()
    const Fake = () => null
    registerPage('ChatPage', Fake)
    expect(getPageComponent('ChatPage')).toBe(Fake)
    expect(listRegisteredPages()).toEqual(['ChatPage'])
  })
  it('未注册返回 undefined（走旧 pageComponents fallback）', () => {
    clearPageRegistry()
    expect(getPageComponent('NoSuchPage')).toBeUndefined()
  })
  it('重复注册覆盖（幂等）', () => {
    clearPageRegistry()
    const A = () => null
    const B = () => null
    registerPage('P', A)
    registerPage('P', B)
    expect(getPageComponent('P')).toBe(B)
  })
})
