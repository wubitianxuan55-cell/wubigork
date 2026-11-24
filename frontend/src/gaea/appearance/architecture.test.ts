import { readdirSync, readFileSync, statSync } from 'node:fs'
import { extname, join, relative, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(__dirname, '../..')
const repoFrontend = resolve(frontendRoot, '..')

function stripComments(source: string): string {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '')
}

function collectFiles(root: string, out: string[] = []): string[] {
  for (const entry of readdirSync(root)) {
    const absolute = join(root, entry)
    const info = statSync(absolute)
    if (info.isDirectory()) {
      if (entry === 'mock' || entry === 'node_modules') continue
      collectFiles(absolute, out)
    } else if (['.css', '.ts', '.tsx'].includes(extname(entry)) && !entry.includes('.test.')) {
      out.push(absolute)
    }
  }
  return out
}

describe('gaea 四层外观架构', () => {
  it('壳层按 semantic -> materials -> components 顺序加载', () => {
    const page = readFileSync(join(frontendRoot, 'pages/GaeaPage.tsx'), 'utf8')
    const semantic = page.indexOf('appearance/semantic.css')
    const materials = page.indexOf('appearance/materials.css')
    const components = page.indexOf('appearance/components.css')
    expect(semantic).toBeGreaterThan(-1)
    expect(materials).toBeGreaterThan(semantic)
    expect(components).toBeGreaterThan(materials)
  })

  it('UI 壳层与通用组件不保留颜色字面量', () => {
    const roots = [
      join(frontendRoot, 'gaea'),
      join(frontendRoot, 'layouts'),
      join(frontendRoot, 'components/settings'),
    ]
    const files = roots.flatMap((root) => collectFiles(root))
    files.push(join(frontendRoot, 'App.tsx'))
    const literal = /#(?:[0-9a-f]{3,8})\b|rgba?\(\s*\d|hsla?\(\s*\d/i
    const violations = files.flatMap((file) => {
      const source = stripComments(readFileSync(file, 'utf8'))
      return literal.test(source) ? [relative(repoFrontend, file)] : []
    })
    expect(violations).toEqual([])
  })

  it('亮暗背景分别保存，并各自映射到上游三种氛围', () => {
    const materials = readFileSync(join(frontendRoot, 'gaea/appearance/materials.css'), 'utf8')
    const app = readFileSync(join(frontendRoot, 'App.tsx'), 'utf8')
    expect(app).toContain('root.dataset.gaeaAmbienceLight')
    expect(app).toContain('root.dataset.gaeaAmbienceDark')
    for (const preset of ['stream', 'wash', 'clear']) {
      expect(materials).toContain(`[data-gaea-ambience-light="${preset}"]`)
      expect(materials).toContain(`url("#gaea-fog-${preset}")`)
    }
    for (const preset of ['corner', 'diagonal', 'galaxy']) {
      expect(materials).toContain(`[data-gaea-ambience-dark="${preset}"]`)
    }
    expect(materials).not.toContain('--gaea-chat-grid-line')
  })

  it('深色星尘使用上游确定性粒子序列与模式开关', () => {
    const stardust = readFileSync(join(frontendRoot, 'components/BloomStardust.tsx'), 'utf8')
    const app = readFileSync(join(frontendRoot, 'App.tsx'), 'utf8')
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    for (const token of ['index < 64', 'index < 26', 'index < 28', 'index < 36', 'index < 44', 'index < 72', 'index < 60']) {
      expect(stardust).toContain(token)
    }
    expect(stardust).toContain("tag: 'b'")
    expect(components).toContain(':not([data-gaea-ambience-dark="off"])')
    expect(components).toContain('[data-gaea-ambience-dark="diagonal"]')
    expect(components).toContain('[data-gaea-ambience-dark="galaxy"]')
    expect(components).toContain('@keyframes gaea-meteor')
    expect(components).toContain('isolation: isolate')
    expect(components).toContain('.gaea-app-shell {')
    expect(components).toContain('.gaea-shell-content {')
    expect(components).toContain('[data-gaea-meteor-density="sparse"]')
    expect(components).toContain('[data-gaea-meteor-speed="fast"]')
    expect(components).toContain('[data-gaea-meteor-color="solid"]')
    expect(components).toContain(
      ':root[data-gaea-mode="dark"][data-gaea-ambience-dark="galaxy"] .gaea-stardust > b.s-met.m2',
    )
    expect(components).toContain(
      ':root[data-gaea-mode="dark"][data-gaea-ambience-dark="galaxy"] .gaea-stardust > b.s-met.m3',
    )
    expect(components).toContain('.ui-reduced-motion .gaea-stardust > i')
    expect(components).toContain('.ui-reduced-motion .gaea-stardust > b.s-met')
    expect(stardust).toContain('var(--gaea-meteor-pace, 1)')
    expect(app).toContain('root.dataset.gaeaMeteorColor')
    expect(app).toContain('root.dataset.gaeaMeteorDensity')
    expect(app).toContain('root.dataset.gaeaMeteorSpeed')
  })

  it('所有一级板块根表面继承同一套玻璃材质契约', () => {
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    const pageCss = readFileSync(join(frontendRoot, 'pages/GaeaPage.tsx'), 'utf8')
    const routeContract = components.slice(components.indexOf('Route surface contract'))
    const materialList = routeContract.match(
      /:is\(([\s\S]*?)\)\s*\{\s*background-color: var\(--gaea-material-fill\)/,
    )?.[1]
    expect(pageCss).toContain("import '../gaea/appearance/components.css'")
    expect(components).toContain('.page-enter > div')
    expect(components).toContain(':not([data-gaea-surface="solid"])')
    expect(components).toContain('background-color: transparent !important')
    expect(components).toContain('background-color: var(--gaea-material-fill) !important')
    expect(components).toContain('background-image: var(--gaea-shell-sheen) !important')
    expect(components).toContain('backdrop-filter: var(--gaea-material-filter) !important')
    expect(components).toContain(
      '.page-enter .gaea-app-layout.ant-layout',
    )
    expect(materialList).not.toContain('.gaea-app-layout')
    for (const root of [
      '.chat-main',
      '.office-sidebar',
      '.ig-result-canvas',
      '.cl-detail-col',
      '.pantheon-rail',
      '.atlas-ambience',
      '.wx-main',
      '.sched-command-deck',
      '.rt-page-header',
      '.rt-section',
      '.mc-rail',
    ]) {
      expect(components).toContain(root)
    }
  })

  it('壳层只消费材质契约，玻璃态差异留在材料层', () => {
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    const materials = readFileSync(join(frontendRoot, 'gaea/appearance/materials.css'), 'utf8')
    const shellTokens = [
      '--gaea-shell-canvas-image',
      '--gaea-shell-rail-fill',
      '--gaea-shell-sidebar-fill',
      '--gaea-shell-strip-fill',
      '--gaea-shell-panel-fill',
      '--gaea-shell-composer-fill',
      '--gaea-shell-sheen',
      '--gaea-shell-filter',
    ]
    expect(components).not.toContain('data-gaea-surface="soft"')
    expect(components).not.toContain('.gaea-app-layout .v3-rail')
    expect(components).toContain('\n.v3-rail {')
    for (const token of shellTokens) {
      expect(components).toContain(`var(${token})`)
      expect(materials).toContain(`${token}:`)
    }
    expect(materials).toContain(':root[data-gaea-surface="veil"]')
    expect(materials).toContain('--gaea-material-filter: blur(24px) saturate(1.15)')
    expect(materials).toContain(':root[data-gaea-surface="matte"]')
    expect(materials).toContain('--gaea-material-filter: blur(12px) saturate(1.02)')
    expect(materials).toContain(':root[data-gaea-surface="soft"]')
    expect(materials).toContain('--gaea-material-filter: blur(3px) url("#gaea-liquid-lens") saturate(1.2)')
  })

  it('文学与聊天输入面复用办公板块的 composer 材质', () => {
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    expect(components).toContain(
      '.wx-root .gaea-app-layout .office-composer-card',
    )
    expect(components).toContain('.chat-composer.gaea-glass-shell')
    expect(components).toContain('background-color: var(--gaea-shell-composer-fill) !important')
    expect(components).toContain('background-image: var(--gaea-shell-sheen) !important')
    expect(components).toContain('backdrop-filter: var(--gaea-shell-filter) !important')
    expect(components).toContain('background: var(--gaea-shell-input-fill) !important')
    expect(components).toContain('background: var(--gaea-shell-toolbar-fill) !important')
  })

  it('表面色阶随主题分五层，面板消费同色族三种色相层', () => {
    const tokens = readFileSync(join(frontendRoot, 'theme/tokens.ts'), 'utf8')
    const semantic = readFileSync(join(frontendRoot, 'gaea/appearance/semantic.css'), 'utf8')
    const materials = readFileSync(join(frontendRoot, 'gaea/appearance/materials.css'), 'utf8')
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    const settings = readFileSync(join(frontendRoot, 'pages/settings-page.css'), 'utf8')

    for (const field of ['surfaceContainerLow', 'surfaceTint1', 'surfaceTint2', 'surfaceTint3']) {
      expect(tokens).toContain(`${field}: string`)
    }
    for (const token of ['--gaea-panel-tint-1', '--gaea-panel-tint-2', '--gaea-panel-tint-3']) {
      expect(semantic).toContain(`${token}:`)
      expect(`${materials}\n${components}`).toContain(`var(${token})`)
    }
    expect(components).toContain('.v3-card {')
    expect(components).toContain('var(--gaea-panel-tint-1)')
    expect(settings).toContain('background-color: color-mix(in srgb, var(--gaea-surface-raised) 72%, transparent)')
  })

  it('颜色令牌对齐 Bloom 0.29 的可读轨、文案阶梯与状态 tertiary', () => {
    const tokens = readFileSync(join(frontendRoot, 'theme/tokens.ts'), 'utf8')
    const app = readFileSync(join(frontendRoot, 'App.tsx'), 'utf8')
    const semantic = readFileSync(join(frontendRoot, 'gaea/appearance/semantic.css'), 'utf8')

    expect(tokens).toContain('mixOklchHex(accentRaw, black, 0.12)')
    for (const [field, transparentPercent] of [
      ['labelSecondary', 'darkMode ? 30 : 28'],
      ['labelTertiary', 'darkMode ? 35 : 32'],
      ['labelCaption', 'darkMode ? 40 : 35'],
      ['labelDimmed', 'darkMode ? 43 : 37'],
    ] as const) {
      expect(tokens).toContain(`const ${field} = mixTransparent(text, ${transparentPercent})`)
      expect(app).toContain(`'--gaea-${field.replace(/[A-Z]/g, (char) => `-${char.toLowerCase()}`)}'`)
    }
    for (const field of ['border1', 'border2', 'border3', 'border4']) {
      expect(tokens).toContain(`const ${field} =`)
      expect(app).toContain(`'--gaea-${field.replace('border', 'border-')}'`)
    }
    expect(tokens).toContain('mixBackground(success, rawSurface, darkMode ? 78 : 88)')
    expect(tokens).toContain('mixBackground(warning, rawSurface, darkMode ? 78 : 88)')
    expect(tokens).toContain('mixBackground(accent, rawSurface, darkMode ? 78 : 88)')
    expect(semantic).toContain('var(--gaea-label-secondary')
    expect(semantic).toContain('--gaea-border-4,')
    expect(semantic).toContain('--gaea-state-warning-tertiary,')
  })

  it('降低透明度偏好对已选择材质的根节点同样生效', () => {
    const materials = readFileSync(join(frontendRoot, 'gaea/appearance/materials.css'), 'utf8')
    expect(materials).toContain('@media (prefers-reduced-transparency: reduce) {\n  :root {')
    expect(materials).not.toContain(':root:not([data-gaea-surface])')
    expect(materials).toContain('--gaea-shell-trigger-filter: none')
    expect(materials).toContain('--gaea-shell-popover-filter: none')
  })

  it('顶栏配色使用六列色块网格与独立字幕，细节控制留在设置页', () => {
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    const menu = readFileSync(join(frontendRoot, 'components/AppearanceMenu.tsx'), 'utf8')
    expect(components).toContain('grid-template-columns: repeat(6, minmax(0, 1fr))')
    expect(components).toContain('.gaea-theme-menu__swatch')
    expect(components).toContain('.gaea-theme-menu__caption')
    expect(menu).toContain('THEME_GLYPHS')
    expect(menu).not.toContain('setAmbienceStyle')
    expect(menu).not.toContain('setSurfaceStyle')
  })
})
