// DeliverablesPanel 拆分——共享常量/纯函数（P3 结构版2 次批巨文件拆分）。
// 内容自 DeliverablesPanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。
import type { Translator } from "../../lib/i18n";

export const SPREADSHEET_EXT_RE = /\.(xlsx?|csv|et|ods)$/i;

export function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

export function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

// 小图标操作按钮：令牌化 + 可见焦点环（全局 :focus-visible）+ aria-label
export const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// 证据卡相对时间：分钟/小时倒推（t 文案），>24h 回落绝对时间。
export function fmtEvidenceTime(at: number, t: Translator): string {
  if (!at) return "—";
  const diff = Date.now() - at;
  const min = Math.floor(diff / 60000);
  if (min < 1) return t("deliverPanel.justNow");
  if (min < 60) return t("deliverPanel.minAgo", { n: min });
  const h = Math.floor(min / 60);
  if (h < 24) return t("deliverPanel.hourAgo", { n: h });
  return new Date(at).toLocaleString();
}

// 登记表时间：unix 秒 → 本地时间（跨会话仍是绝对时间，不随会话漂移）。
export function fmtRegistryTime(at: number): string {
  if (!at) return "—";
  const d = new Date(at * 1000);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString("zh-CN", { hour12: false, month: "numeric", day: "numeric" });
}