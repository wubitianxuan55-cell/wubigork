/** 按 Unicode 码点统计文本长度，避免中文等多字节字符被 UTF-16 length 误计。 */
export function countTextChars(text: string): number {
  return Array.from(text).length
}
