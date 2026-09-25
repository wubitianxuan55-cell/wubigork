// stats.ts — pure computation helpers.
// Extracted for testability: no React, no DOM, no side effects.

// ─── formatting ─────────────────────────────────────────────────

export function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, "") + "k";
  return String(n);
}
