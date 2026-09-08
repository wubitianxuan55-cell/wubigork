import { afterEach, describe, expect, it, vi } from 'vitest'
import { dataUrlToBlob, downloadBlob, saveExportBlob } from './saveFile'
import { inShellEnv } from './pickFile'
import { inShell } from '../../schedule/exportArtifact'

/**
 * saveFile.test.ts — 导出文件保存中立层用例（瘦身 P1 线1 ③「双门」）
 *
 * 壳内路径经 window.go stub 直装真壳同款 Gaea 前缀绑定面（bridge realApp
 * 代理按方法名路由，同 schedule/store.sync.test.ts / pickFile.test.ts 口径）；
 * afterEach 删 window.go 还原浏览器通道。浏览器路径 stub URL.createObjectURL
 * + spy HTMLAnchorElement.prototype.click，断言 <a download> 语义。
 */

/** 装真壳同款 window.go 绑定面（GaeaSaveFileAs 返回值即「另存为」结果路径） */
function stubSaveSurface(result: string): ReturnType<typeof vi.fn> {
  const saveFileAs = vi.fn(async () => result)
  ;(window as unknown as { go?: unknown }).go = {
    app: { SaveFileTestB: { GaeaSaveFileAs: saveFileAs } },
  }
  return saveFileAs
}

afterEach(() => {
  delete (window as unknown as { go?: unknown }).go
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('saveExportBlob（双门：壳内系统另存为 / 浏览器 a.download）', () => {
  it('壳内：走 GaeaSaveFileAs，收到文件名与 blob 内容（base64），保存成功 → true', async () => {
    const saveFileAs = stubSaveSurface('C:/导出/甘特.png')
    expect(inShellEnv()).toBe(true)
    const blob = new Blob([new Uint8Array([104, 105])], { type: 'image/png' })
    await expect(saveExportBlob(blob, '甘特.png')).resolves.toBe(true)
    expect(saveFileAs).toHaveBeenCalledTimes(1)
    expect(saveFileAs.mock.calls[0][0]).toBe('甘特.png')
    expect(saveFileAs.mock.calls[0][1]).toBe('aGk=') // "hi"
  })

  it('壳内：用户取消（SaveFileAs 返回空串）→ false，不抛错', async () => {
    const saveFileAs = stubSaveSurface('')
    await expect(saveExportBlob(new Blob(['x']), 'a.md')).resolves.toBe(false)
    expect(saveFileAs).toHaveBeenCalledTimes(1)
  })

  it('浏览器：回退 <a download>（createObjectURL + a.download=name + click + revoke）', async () => {
    const create = vi.fn(() => 'blob:mock-save')
    const revoke = vi.fn()
    vi.stubGlobal('URL', { ...globalThis.URL, createObjectURL: create, revokeObjectURL: revoke })
    const clicked: HTMLAnchorElement[] = []
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function clickCapture(this: HTMLAnchorElement) {
        clicked.push(this)
      })
    const blob = new Blob(['browser'], { type: 'text/plain' })
    await expect(saveExportBlob(blob, 'a.txt')).resolves.toBe(true)
    expect(create).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalledTimes(1)
    expect(clicked[0].download).toBe('a.txt')
    expect(clicked[0].href).toBe('blob:mock-save')
    expect(revoke).toHaveBeenCalledWith('blob:mock-save')
    click.mockRestore()
  })
})

describe('downloadBlob（浏览器下载原语，随实现体一并迁入中立层）', () => {
  it('anchor 语义：download=name、href=blob URL、点击后 revoke', () => {
    const create = vi.fn(() => 'blob:mock-dl')
    const revoke = vi.fn()
    vi.stubGlobal('URL', { ...globalThis.URL, createObjectURL: create, revokeObjectURL: revoke })
    const clicked: HTMLAnchorElement[] = []
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function clickCapture(this: HTMLAnchorElement) {
        clicked.push(this)
      })
    const blob = new Blob(['x'])
    downloadBlob(blob, 'plan.xml')
    expect(create).toHaveBeenCalledWith(blob)
    expect(clicked[0].download).toBe('plan.xml')
    expect(revoke).toHaveBeenCalledWith('blob:mock-dl')
    click.mockRestore()
  })
})

describe('dataUrlToBlob（壳内另存为前置转换：mime 保留 + 字节精确）', () => {
  it('data URL → Blob（type 取 dataURL 前缀，负载无损）', async () => {
    const blob = dataUrlToBlob('data:image/png;base64,' + btoa('png!'))
    expect(blob.type).toBe('image/png')
    await expect(blob.text()).resolves.toBe('png!')
  })
})

describe('inShell 判定合一（schedule/exportArtifact 薄委托 inShellEnv）', () => {
  it('壳内与浏览器两侧与 inShellEnv 同值', () => {
    expect(inShell()).toBe(inShellEnv())
    stubSaveSurface('x')
    expect(inShell()).toBe(inShellEnv())
    expect(inShell()).toBe(true)
  })
})
