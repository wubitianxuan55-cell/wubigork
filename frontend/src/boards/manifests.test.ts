import { describe, expect, it } from 'vitest'
import {
  canonicalBoards, menuBoards, navigateWhitelist, shortcutMap,
  homeBoard, projectAnchorId, getBoard, boardLabel, resolveBoardIcon,
} from './manifests'
import { registerPage, getPageComponent, listRegisteredPages, clearPageRegistry } from './pageRegistry'

// 3.0 附 B 收敛映射回归：manifest 派生结果必须与旧 MainLayout 硬编码一致（像素级）。

describe('menuBoards（附 B #4：filter(inMenu) + sort(menuOrder)）', () => {
  it('菜单顺序与现状一致：首页 → 聊天 → 小说 → 绘梦 → 办公 → 记忆中枢 → 模型中心 → 角色库', () => {
    expect(menuBoards.map((b) => b.id)).toEqual([
      'home', 'chat', 'novel', 'imagegen', 'gaea', 'memoryhub', 'modelcenter', 'characterlib',
    ])
  })

  it('菜单文案与现状一致（首页/聊天/小说/绘梦/办公/记忆中枢/模型中心/角色库）', () => {
    expect(menuBoards.map((b) => b.label)).toEqual([
      '首页', '聊天', '小说', '绘梦', '办公', '记忆中枢', '模型中心', '角色库',
    ])
  })

  it('菜单图标名与现状一致（antd 图标注册表可解析）', () => {
    const expected = ['HomeOutlined', 'MessageOutlined', 'ReadOutlined', 'PictureOutlined',
      'ToolOutlined', 'DatabaseOutlined', 'ApiOutlined', 'TeamOutlined']
    expect(menuBoards.map((b) => b.icon)).toEqual(expected)
    for (const b of menuBoards) {
      expect(resolveBoardIcon(b.icon), `icon ${b.icon} 可解析`).not.toBeNull()
    }
  })

  it('settings/weixin 不进菜单（inMenu=false）', () => {
    expect(menuBoards.map((b) => b.id)).not.toContain('settings')
    expect(menuBoards.map((b) => b.id)).not.toContain('weixin')
  })
})

describe('navigateWhitelist（附 B #2：manifest 派生导航白名单）', () => {
  it('与旧 allPageKeys 一致：7 个业务板块，不含 home/settings', () => {
    expect(navigateWhitelist).toEqual([
      'chat', 'novel', 'imagegen', 'gaea', 'memoryhub', 'modelcenter', 'characterlib',
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
  it('chat/gaea = full（全出血），其余 = padded', () => {
    expect(getBoard('chat')?.layout).toBe('full')
    expect(getBoard('gaea')?.layout).toBe('full')
    for (const id of ['novel', 'imagegen', 'memoryhub', 'modelcenter', 'characterlib', 'settings']) {
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
