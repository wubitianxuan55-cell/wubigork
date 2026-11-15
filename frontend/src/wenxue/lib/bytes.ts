/**
 * bytes.ts — base64 → 字节 解码中立层（瘦身 P1 线1「双门+字节收口刀」①）
 *
 * Why: 全仓十余处 `atob + charCodeAt 循环` 手搓解码（TTS 音频块 / docx·pptx
 * zip 解包 / 导入 File 还原 / 导出字节…），写法散落且易复制错。统一收编为
 * 单一纯函数，调用方一行替换。
 *
 * 语义约定：入参为**裸 base64**（dataURL 需先由调用方截取逗号后负载）；
 * 解码失败抛 atob 原生 InvalidCharacterError 由调用方处理。大文件上限
 * （64MB）不在本层强制——语义由调用方/后端保证（如 GaeaReadFileB64 后端
 * os.Stat 拒绝超限）；未来引擎普及时可整体替换为原生
 * Uint8Array.fromBase64，调用点零改动。
 */
export function b64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes
}
