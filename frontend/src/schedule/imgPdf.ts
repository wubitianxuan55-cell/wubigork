/**
 * imgPdf.ts — 最小 PDF 封装（纯函数，v4.132.0 刀D1）
 *
 * 单页 PDF 1.4 + 单个 JPEG XObject（DCTDecode 直嵌不再压缩）：
 * 图面 SVG → canvas JPEG → 本模块封装为可打印/可上报的 PDF 文件。
 * MediaBox 按像素 72/96 换算（CSS px 基准），整图铺满页。
 * 不引第三方依赖；文本段全 ASCII（字符偏移=字节偏移），xref 偏移逐一对齐，
 * 结构用例钉死（解析 xref 验证每个偏移落在对应 obj 头上）。
 */

const enc = new TextEncoder()

/** 数字 → 定宽 ASCII 字节（xref 偏移 10 位补零） */
function pad10(n: number): string {
  return String(n).padStart(10, '0')
}

/** px → pt（CSS px 96dpi → PDF 72dpi），保留 2 位小数 */
function pxToPt(px: number): string {
  return (Math.round(((px * 72) / 96) * 100) / 100).toFixed(2)
}

/**
 * JPEG 字节 → 单页 PDF 字节。wPx/hPx 为图像像素尺寸（决定 MediaBox 比例）。
 * 结构：1 Catalog / 2 Pages / 3 Page / 4 Image XObject / 5 Contents。
 */
export function jpegBytesToPdf(jpeg: Uint8Array, wPx: number, hPx: number): Uint8Array {
  if (!Number.isFinite(wPx) || !Number.isFinite(hPx) || wPx <= 0 || hPx <= 0) {
    throw new Error('图像尺寸非法')
  }
  const wPt = pxToPt(wPx)
  const hPt = pxToPt(hPx)
  const header = '%PDF-1.4\n'
  const o1 = '1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n'
  const o2 = '2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n'
  const o3 = `3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${wPt} ${hPt}] /Resources << /XObject << /Im0 4 0 R >> /ProcSet [/PDF /ImageC] >> /Contents 5 0 R >>\nendobj\n`
  const obj4Head = `4 0 obj\n<< /Type /XObject /Subtype /Image /Width ${Math.round(wPx)} /Height ${Math.round(hPx)} /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${jpeg.length} >>\nstream\n`
  const obj4Tail = '\nendstream\nendobj\n'
  const content = `q\n${wPt} 0 0 ${hPt} 0 0 cm\n/Im0 Do\nQ`
  const o5 = `5 0 obj\n<< /Length ${content.length} >>\nstream\n${content}\nendstream\nendobj\n`

  // 偏移表：文本段全 ASCII，字符偏移即字节偏移；jpeg 二进制段长度精确
  let cursor = header.length
  const off1 = cursor
  cursor += o1.length
  const off2 = cursor
  cursor += o2.length
  const off3 = cursor
  cursor += o3.length
  const off4 = cursor
  cursor += obj4Head.length + jpeg.length + obj4Tail.length
  const off5 = cursor
  cursor += o5.length
  const xrefOff = cursor
  let xref = 'xref\n0 6\n0000000000 65535 f \n'
  for (const off of [off1, off2, off3, off4, off5]) xref += `${pad10(off)} 00000 n \n`
  const trailer = `trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n${xrefOff}\n%%EOF`

  const chunks: Uint8Array[] = [
    enc.encode(header),
    enc.encode(o1),
    enc.encode(o2),
    enc.encode(o3),
    enc.encode(obj4Head),
    jpeg,
    enc.encode(obj4Tail),
    enc.encode(o5),
    enc.encode(xref),
    enc.encode(trailer),
  ]
  let total = 0
  for (const c of chunks) total += c.length
  const out = new Uint8Array(total)
  let at = 0
  for (const c of chunks) {
    out.set(c, at)
    at += c.length
  }
  return out
}

/** Uint8Array → Blob（下载用；TS BlobPart 收窄） */
export function bytesToBlob(bytes: Uint8Array, type: string): Blob {
  return new Blob([bytes as BlobPart], { type })
}
