/**
 * errText — 错误信息的统一取文（方言 A：带兜底文案）：
 * Error 实例取 message（空 message 也回退兜底），其余回退兜底文案。
 *
 * 收敛自 14 份逐字相同的本地实现（2026-10-04 第三轮审计 §3.2）。
 * 注意方言边界：`errText(err)`（无 fallback、`String(err ?? '未知错误')`）
 * 是另一方言，语义不同（Error 空 message / null 的走向不同），勿混并。
 */
export function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}
