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

  it('对话画布用独立渐隐层绘制可配置网格', () => {
    const components = readFileSync(join(frontendRoot, 'gaea/appearance/components.css'), 'utf8')
    const materials = readFileSync(join(frontendRoot, 'gaea/appearance/materials.css'), 'utf8')
    expect(components).toContain('.gaea-app-layout .chat-pane::before')
    expect(components).toContain('linear-gradient(to right, var(--gaea-chat-grid-line) 1px, transparent 1px)')
    expect(components).toContain('linear-gradient(to bottom, var(--gaea-chat-grid-line) 1px, transparent 1px)')
    expect(components).toContain('var(--gaea-chat-grid-size) var(--gaea-chat-grid-size)')
    expect(components).toContain('mask-image: var(--gaea-chat-grid-mask)')
    expect(materials).toContain('--gaea-chat-grid-size: 80px')
    expect(materials).toContain('--gaea-chat-grid-mask: radial-gradient(')
    expect(materials).toContain('--gaea-chat-grid-line: transparent')
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
      '--gaea-shell-chat-start',
      '--gaea-shell-chat-end',
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
    expect(materials).toContain('--gaea-material-filter: blur(32px) saturate(1.7) contrast(1.05)')
  })
})
