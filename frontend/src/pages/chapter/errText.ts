/**
 * errText — 章节页错误信息的统一取文（自 ChapterPage 原样搬移）：
 * Error 实例取 message，其余回退兜底文案。
 */
export function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}