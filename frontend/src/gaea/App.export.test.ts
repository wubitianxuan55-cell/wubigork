import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * App.export.test.ts — 审计刀B c 回归锁（源级断言；App 巨组件不做整页渲染）：
 * 办公导出 md 分支必须收口到统一交付管线 exportConversation（ExportDeliverable，
 * 壳内落盘交付），不允许回退浏览器 <a download>（壳内不落盘，v4.162 实证）。
 * lib/export.ts 的 downloadMarkdown 已清理（v4.167.0）：无调用方的浏览器下载路径不再保留。
 */
// vitest 变换后 import.meta.url 非 file 协议，源文件按进程 cwd（frontend/）定位
const src = readFileSync(join(process.cwd(), 'src/gaea/App.tsx'), 'utf8')
const exportSrc = readFileSync(join(process.cwd(), 'src/gaea/lib/export.ts'), 'utf8')

describe('App 办公导出 md 分支收口（审计刀B c）', () => {
  it('onPick 统一走 exportConversation（ExportDeliverable 管线，零新代码路径）', () => {
    expect(src).toContain('void exportConversation(format)')
  })

  it('App 不再直接调用浏览器下载 downloadMarkdown', () => {
    expect(src).not.toMatch(/\bdownloadMarkdown\b/)
  })

  it('export.ts 已清理 downloadMarkdown，办公导出无浏览器下载回退路径', () => {
    expect(exportSrc).not.toMatch(/\bdownloadMarkdown\b/)
    expect(exportSrc).not.toMatch(/createObjectURL/)
  })
})
