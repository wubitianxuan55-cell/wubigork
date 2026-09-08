import { describe, expect, it } from 'vitest'
import { b64ToBytes } from './bytes'

// bytes.test.ts — b64ToBytes（base64 → 字节中立层）契约锁：
// ASCII 解码、全 256 字节值无损（0x00-0xFF 无符号语义）、空串边界。
// 大文件上限语义（64MB）由调用方/后端保证，不在纯函数层断言。

describe('b64ToBytes（base64 → 字节中立层）', () => {
  it('ASCII base64 解码', () => {
    expect(new TextDecoder().decode(b64ToBytes('aGVsbG8gZ2FlYQ=='))).toBe('hello gaea')
  })

  it('全 256 字节值无损（0x00-0xFF 无符号语义）', () => {
    const src = Uint8Array.from({ length: 256 }, (_, i) => i)
    const b64 = btoa(String.fromCharCode(...src))
    expect(b64ToBytes(b64)).toEqual(src)
  })

  it('空串 → 长度 0 的 Uint8Array', () => {
    const out = b64ToBytes('')
    expect(out).toBeInstanceOf(Uint8Array)
    expect(out).toHaveLength(0)
  })
})
