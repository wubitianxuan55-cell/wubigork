/**
 * imgPdf.test.ts — 最小 PDF 封装结构用例（刀D1）
 *
 * 用例口径：解析 xref 偏移逐一对齐（每个偏移落在对应「i 0 obj」头上）、
 * startxref 落在 xref 段、MediaBox 72/96 换算、DCTDecode 直嵌、确定性复现。
 * JPEG 内容无关结构，用带 SOI/EOI 标记的最小字节即可。
 */
import { describe, expect, it } from 'vitest'
import { jpegBytesToPdf } from './imgPdf'

/** 最小 JPEG 假体（SOI … EOI；中段内容不影响 PDF 结构） */
function fakeJpeg(n: number): Uint8Array {
  const bytes = new Uint8Array(n)
  bytes[0] = 0xff
  bytes[1] = 0xd8
  bytes[n - 2] = 0xff
  bytes[n - 1] = 0xd9
  for (let i = 2; i < n - 2; i++) bytes[i] = (i * 7) % 251
  return bytes
}

const latin1 = (bytes: Uint8Array): string => {
  let s = ''
  for (const b of bytes) s += String.fromCharCode(b)
  return s
}

describe('jpegBytesToPdf 结构', () => {
  const jpeg = fakeJpeg(64)
  const pdf = jpegBytesToPdf(jpeg, 1920, 1080)
  const text = latin1(pdf)

  it('头尾与关键结构：%PDF-1.4 / %%EOF / DCTDecode / JPEG 原样直嵌', () => {
    expect(text.startsWith('%PDF-1.4\n')).toBe(true)
    expect(text.endsWith('%%EOF')).toBe(true)
    expect(text).toContain('/DCTDecode')
    expect(text).toContain(`/Width 1920 /Height 1080`)
    expect(text).toContain(`/Length ${jpeg.length}`)
    expect(text).toContain('/Type /Catalog')
    expect(text).toContain('/Type /Pages /Kids [3 0 R] /Count 1')
  })

  it('MediaBox：CSS px 96dpi → PDF pt 72dpi', () => {
    expect(text).toContain('/MediaBox [0 0 1440.00 810.00]')
    // 非整除换算保留 2 位
    const odd = latin1(jpegBytesToPdf(fakeJpeg(8), 100, 33))
    expect(odd).toContain('/MediaBox [0 0 75.00 24.75]')
  })

  it('xref 偏移逐一对齐：每个偏移落在对应 obj 头；startxref 指向 xref 段', () => {
    expect(text).toContain('xref\n0 6\n0000000000 65535 f \n')
    const offs = [...text.matchAll(/(\d{10}) 00000 n \n/g)].map((m) => Number(m[1]))
    expect(offs).toHaveLength(5)
    offs.forEach((off, i) => {
      expect(text.slice(off, off + 10).startsWith(`${i + 1} 0 obj\n`), `obj ${i + 1} 偏移`).toBe(true)
    })
    const startxref = Number(/startxref\n(\d+)\n%%EOF$/.exec(text)![1])
    expect(text.slice(startxref, startxref + 4)).toBe('xref')
  })

  it('内容流整图铺满：cm 矩阵=MediaBox 尺寸；确定性：两次构建字节一致', () => {
    expect(text).toContain('q\n1440.00 0 0 810.00 0 0 cm\n/Im0 Do\nQ')
    expect(latin1(jpegBytesToPdf(jpeg, 1920, 1080))).toBe(text)
  })

  it('尺寸非法抛错', () => {
    expect(() => jpegBytesToPdf(jpeg, 0, 100)).toThrow(/尺寸/)
    expect(() => jpegBytesToPdf(jpeg, Number.NaN, 100)).toThrow(/尺寸/)
  })
})
