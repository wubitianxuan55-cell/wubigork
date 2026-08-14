import { describe, expect, it, vi } from 'vitest'
import {
  INLINE_IMAGE_MAX,
  needsFileRestore,
  restoreHistoryImages,
  serializeHistoryMeta,
} from './historyMeta'
import type { GenResult } from './types'

const base = (over: Partial<GenResult>): GenResult => ({
  image: '', seed: 1, time: 2, prompt: 'p', model: 'krea2', size: '1024x1024', ...over,
})

describe('serializeHistoryMeta（localStorage 容量保护分级策略）', () => {
  it('keeps small inline base64 images', () => {
    const small = 'data:image/png;base64,' + 'a'.repeat(100)
    const [out] = serializeHistoryMeta([base({ image: small })])
    expect(out.image).toBe(small)
  })

  it('drops oversized base64 and keeps file_path only', () => {
    const big = 'data:image/png;base64,' + 'a'.repeat(INLINE_IMAGE_MAX + 1)
    const [out] = serializeHistoryMeta([base({ image: big, file_path: 'C:\\img\\a.png' })])
    expect(out.image).toBe('')
    expect(out.file_path).toBe('C:\\img\\a.png')
  })

  it('retains file_path alongside small inline images', () => {
    const small = 'data:image/png;base64,' + 'a'.repeat(50)
    const [out] = serializeHistoryMeta([base({ image: small, file_path: 'C:\\img\\b.png' })])
    expect(out.image).toBe(small)
    expect(out.file_path).toBe('C:\\img\\b.png')
  })

  it('strips image when only file_path is available', () => {
    const [out] = serializeHistoryMeta([base({ image: '', file_path: 'C:\\img\\c.png' })])
    expect(out.image).toBe('')
    expect(out.file_path).toBe('C:\\img\\c.png')
  })
})

describe('needsFileRestore', () => {
  it('true when no base64 but file_path exists', () => {
    expect(needsFileRestore(base({ image: '', file_path: 'C:\\img\\a.png' }))).toBe(true)
  })

  it('false when image present or no file_path', () => {
    expect(needsFileRestore(base({ image: 'data:image/png;base64,AA', file_path: 'C:\\img\\a.png' }))).toBe(false)
    expect(needsFileRestore(base({ image: '' }))).toBe(false)
    expect(needsFileRestore(base({ image: 'data:image/png;base64,AA' }))).toBe(false)
  })
})

describe('restoreHistoryImages（历史恢复路径：file_path → 后端读取回填）', () => {
  it('calls backend reader with file_path and backfills dataURL', async () => {
    const readFile = vi.fn(async () => 'data:image/png;base64,REAL')
    const out = await restoreHistoryImages(
      [
        base({ image: '', file_path: 'C:\\img\\a.png' }),
        base({ image: 'data:image/png;base64,AA' }),
        base({ image: '' }),
      ],
      readFile,
    )
    expect(readFile).toHaveBeenCalledTimes(1)
    expect(readFile).toHaveBeenCalledWith('C:\\img\\a.png')
    expect(out).toHaveLength(1)
    expect(out[0].image).toBe('data:image/png;base64,REAL')
    expect(out[0].file_path).toBe('C:\\img\\a.png')
  })

  it('records read failure and leaves item unbackfilled (no silent swallow)', async () => {
    const readFile = vi.fn(async () => { throw new Error('file missing') })
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      const out = await restoreHistoryImages([base({ image: '', file_path: 'C:\\img\\gone.png' })], readFile)
      expect(out).toHaveLength(0)
      expect(warn).toHaveBeenCalledTimes(1)
    } finally {
      warn.mockRestore()
    }
  })
})
