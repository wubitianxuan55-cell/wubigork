// KnowledgePanel 拆分——纯函数/渲染助手（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。
import type { CSSProperties, ReactNode } from "react";
import type { KnowledgeSummary } from "../../lib/types";

/** 检索关键词转义：正则元字符按字面匹配（highlightText 共用）。 */
export function escapeRegExp(q: string): string {
  return q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

// Highlight matching text（query 空时原样返回）
export function highlightText(text: string, query: string): string | ReactNode {
  if (!query) return text;
  const regex = new RegExp(`(${escapeRegExp(query)})`, "gi");
  const parts = text.split(regex);
  return parts.map((part, i) =>
    regex.test(part) ? <mark key={i} className="bg-yellow-300/30 text-fg rounded px-0.5">{part}</mark> : part
  );
}

// 列表摘要里后端会返回 phase/source 等扩展字段（TS 类型未声明），沿用原逻辑的强转。
export function phaseOf(e: KnowledgeSummary): string {
  return (e as unknown as Record<string, string>).phase;
}

export function sourceOf(e: KnowledgeSummary): string {
  return (e as unknown as Record<string, string>).source;
}

// 状态徽标配色：草稿=warning、已归档=中性、其余=accent（全部走令牌）。
export function statusBadgeStyle(st: string): CSSProperties {
  const warn = "var(--color-warning, var(--md-sys-color-warning))";
  const accent = "var(--accent, var(--md-sys-color-primary))";
  if (st === "草稿") return {
    background: `color-mix(in srgb, ${warn} 12%, transparent)`,
    color: warn,
    borderColor: `color-mix(in srgb, ${warn} 30%, transparent)`,
  };
  if (st === "已归档") return {
    background: "var(--bg-soft, var(--md-sys-color-surface-container))",
    color: "var(--fg-faint, var(--md-sys-color-text-secondary))",
    borderColor: "var(--border-soft, var(--md-sys-color-outline-variant))",
  };
  return {
    background: `color-mix(in srgb, ${accent} 12%, transparent)`,
    color: accent,
    borderColor: `color-mix(in srgb, ${accent} 30%, transparent)`,
  };
}