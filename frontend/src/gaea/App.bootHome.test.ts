import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const src = readFileSync(join(process.cwd(), 'src/gaea/App.tsx'), 'utf8')

describe('办公冷启动新建会话首页', () => {
  it('启动时固定落在对话页签，不恢复上次停留的办公页签', () => {
    expect(src).toContain('const [chatTab, setChatTab] = useState<ChatTabId>("chat");')
    expect(src).not.toMatch(/const \[chatTab, setChatTab\] = useState<ChatTabId>\(\(\) =>/)
  })

  it('空会话首页不依赖 meta.ready，避免启动元数据未回填时露出空白对话区', () => {
    const block = src.split('const showCenteredComposer =')[1]?.split(';')[0] ?? ''
    expect(block).toContain('showOfficeComposer')
    expect(block).toContain('state.items.length === 0')
    expect(block).toContain('!state.running')
    expect(block).not.toContain('state.meta?.ready')
  })

  it('新建会话首页显示时不再渲染启动骨架，欢迎标题和快捷入口可直接呈现', () => {
    expect(src).toMatch(/\{!showCenteredComposer\s+&& \(\(state\.meta\?\.ready === false/)
  })
})
