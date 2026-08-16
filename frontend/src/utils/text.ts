/** 按 Unicode 码点统计文本长度，避免中文等多字节字符被 UTF-16 length 误计。 */
export function countTextChars(text: string): number {
  return Array.from(text).length
}

/** 从 AI 回复中提取可用的设定文本：优先 ```markdown / ```md 代码块，其次整个回复 */
export function extractSettingText(reply: string): string {
  const trimmed = reply.trim()
  const fenceMatch = trimmed.match(/```(?:markdown|md)\s*\n([\s\S]*?)\n?```/)
  if (fenceMatch?.[1]?.trim()) return fenceMatch[1].trim()
  return trimmed
}
