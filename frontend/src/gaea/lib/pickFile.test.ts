/**
 * pickFile.test.ts — Wails 壳内文件选取共享 util 用例（审计刀A）
 *
 * 壳路径经 window.go stub 直装真壳同款 Gaea 前缀绑定面（bridge realApp
 * 代理按方法名路由，同 schedule/store.sync.test.ts 口径）；afterEach 删
 * window.go 还原浏览器/bridge-mock 通道。
 * 覆盖：取消（PickFiles 空）→ null、扩展名白名单后置校验 fail-closed
 * （GaeaPickFiles 无过滤器）、正常链路 PickFiles→ReadFileB64→还原 File
 * / dataURL（mime 按扩展名映射）。
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { inShellEnv, pickFileAsFile, pickImageAsDataUrl } from './pickFile'

type PickSurface = {
  pickFiles: ReturnType<typeof vi.fn>
  readFileB64: ReturnType<typeof vi.fn>
}

/** 装真壳同款 window.go 绑定面（bridge realApp 按方法名路由到 Gaea 前缀） */
function stubPickSurface(picked: unknown[], b64: string): PickSurface {
  const pickFiles = vi.fn(async () => picked)
  const readFileB64 = vi.fn(async () => b64)
  ;(window as unknown as { go?: unknown }).go = {
    app: { PickFileTestB: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
  }
  return { pickFiles, readFileB64 }
}

afterEach(() => {
  delete (window as unknown as { go?: unknown }).go
})

it('无 window.go（浏览器/jsdom）→ inShellEnv() 为 false，调用方走原生 input', () => {
  expect(inShellEnv()).toBe(false)
})

describe('pickFile（壳内文件选取共享 util）', () => {
  it('壳内取消（PickFiles 空数组）→ 还原 File / dataURL 都返回 null', async () => {
    const { readFileB64 } = stubPickSurface([], 'aGVsbG8=')
    expect(inShellEnv()).toBe(true)
    expect(await pickFileAsFile(['png'])).toBeNull()
    expect(await pickImageAsDataUrl()).toBeNull()
    expect(readFileB64).not.toHaveBeenCalled() // 取消不读内容
  })

  it('扩展名白名单后置校验：不合法抛错且不读内容（GaeaPickFiles 无过滤器，fail-closed）', async () => {
    const { readFileB64 } = stubPickSurface(
      [{ path: 'C:/x/a.exe', name: 'a.exe', type: 'file', size: 5 }],
      'aGVsbG8=',
    )
    await expect(pickFileAsFile(['png', 'jpg'])).rejects.toThrow('仅支持 .png/.jpg 文件')
    await expect(pickImageAsDataUrl()).rejects.toThrow(
      '仅支持 .png/.jpg/.jpeg/.webp/.gif/.bmp 文件',
    )
    expect(readFileB64).not.toHaveBeenCalled()
  })

  it('正常链路：PickFiles → ReadFileB64 → 还原 File（文件名 + 字节内容）', async () => {
    const { pickFiles, readFileB64 } = stubPickSurface(
      [{ path: 'C:/imgs/ref.png', name: 'ref.png', type: 'image', size: 5 }],
      'aGVsbG8=', // "hello"
    )
    const file = await pickFileAsFile(['png', 'jpg', 'jpeg', 'webp', 'gif', 'bmp'])
    expect(pickFiles).toHaveBeenCalledTimes(1)
    expect(readFileB64).toHaveBeenCalledWith('C:/imgs/ref.png')
    expect(file).not.toBeNull()
    expect(file!.name).toBe('ref.png')
    expect(await file!.text()).toBe('hello')
  })

  it('pickImageAsDataUrl：扩展名映射 mime（大写后缀同判）→ data URL', async () => {
    stubPickSurface(
      [{ path: 'C:/imgs/photo.JPG', name: 'photo.JPG', type: 'image', size: 5 }],
      'aGVsbG8=',
    )
    await expect(pickImageAsDataUrl()).resolves.toBe('data:image/jpeg;base64,aGVsbG8=')
  })
})
