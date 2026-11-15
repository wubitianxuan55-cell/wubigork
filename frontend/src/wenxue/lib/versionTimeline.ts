// versionTimeline.ts — v4.28 B1「文件版本时间线」纯函数层。
// Why：产物行的 vN 徽标此前只给「改了几次」的次数；证据链（GaeaJournalList，
// 与回滚同源的自动快照）已经为每次写盘留了基线快照，把它按 target 聚合成
// 逐版本列表，才能支撑「预览 → 恢复」护栏流（对标 Notion 版本史）。
// How：纯函数无副作用——groupVersionsByPath 按 target 聚合（Windows 反斜杠
// 归一为 /，保证与产物路径能对上）、at 倒序（最新在前）、只留有 baselinePath
// 的记录（无基线快照 = 不能预览/恢复，进时间线只会是死按钮）；
// versionTimeText / versionLabel 输出 HH:MM（本地时区）与「HH:MM · 工具名」；
// versionStatusText 把后端状态码翻成中文徽标文案。
import type { JournalChangeRecord } from "./types";

/** 统一路径分隔符：Windows 侧 target 可能带反斜杠，与产物路径（正斜杠）对齐。 */
export function normalizeVersionPath(path: string): string {
  return path.replace(/\\/g, "/");
}

/**
 * 按工作区相对路径聚合版本记录：同一 target 的所有证据卡归为一组。
 * - 只保留有 baselinePath 的记录（无基线快照 = 无法预览/恢复，不进时间线）；
 * - 每组内按 at 倒序（最新在前，对齐 Notion 版本史首屏）。
 */
export function groupVersionsByPath(records: JournalChangeRecord[]): Map<string, JournalChangeRecord[]> {
  const grouped = new Map<string, JournalChangeRecord[]>();
  for (const r of records) {
    if (!r.baselinePath) continue;
    const key = normalizeVersionPath(r.target);
    const bucket = grouped.get(key);
    if (bucket) bucket.push(r);
    else grouped.set(key, [r]);
  }
  for (const bucket of grouped.values()) bucket.sort((a, b) => b.at - a.at);
  return grouped;
}

/** 版本时间文本：HH:MM（本地时区 24 小时制）；at 缺失/非法 → "--:--"。 */
export function versionTimeText(record: JournalChangeRecord): string {
  const d = new Date(record.at);
  if (Number.isNaN(d.getTime())) return "--:--";
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/** 版本标签：「HH:MM · 工具名」（恢复 toast 与无障碍文案共用）。 */
export function versionLabel(record: JournalChangeRecord): string {
  return `${versionTimeText(record)} · ${record.tool}`;
}

/** 状态徽标文案：后端状态码 → 中文；未知状态透传原文，空值显示占位。 */
export function versionStatusText(status: string): string {
  switch (status) {
    case "pending_verify": return "待复核";
    case "verified": return "复核通过";
    case "warned": return "复核警告";
    case "failed": return "复核未通过";
    case "applied": return "已应用";
    case "": return "—";
    default: return status;
  }
}
