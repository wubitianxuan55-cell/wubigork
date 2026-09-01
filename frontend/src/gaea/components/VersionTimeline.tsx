// VersionTimeline — v4.28 B1「文件版本时间线」（对标 Notion 版本史 / Claude
// Artifacts 版本选择器）。
// Why：产物行的 vN 徽标此前只给「改了几次」的次数，用户既看不到每次改了什么，
// 也无法在改坏时退回上一版；证据链（JournalList）已为每次写盘留了基线快照，
// 把它按文件聚合成逐版本列表，「预览 → 恢复」才有落点。
// How：本组件是纯展示层——版本记录（调用方经 groupVersionsByPath 聚合、at
// 倒序）、预览回调（openFilePreview(baselinePath)：基线快照位于工作区内，
// Preview 的 resolvePreviewPath IsAbs 分支可直读）、恢复回调（父级负责
// RollbackRecord(id) + toast + 刷新时间线）全部经 props 注入，本组件不直接
// 触碰 bridge。每行 = 时间/工具/轮次/状态徽标 + 预览/恢复两个操作；恢复语义
// 对齐调研结论（docs/research-2026-09-01/version-timeline-diff.md §4）：
// 不做二次确认弹窗（预览即护栏），顶部常驻说明「恢复会把该文件写回所选版本，
// 当前内容成为新版本」；恢复进行中禁用全部恢复按钮，避免并发写盘竞态。
import { useCallback, useState } from "react";
import { Clock, Eye, Loader2, Rollback } from "../icons";
import type { JournalChangeRecord } from "../lib/types";
import { versionStatusText, versionTimeText } from "../lib/versionTimeline";

export interface VersionTimelineProps {
  /** 目标文件（工作区相对路径）——标题与 data-path 定位用，不做数据过滤。 */
  path: string;
  /** 该路径的版本记录（已聚合、at 倒序、只含有 baselinePath 的卡）；null = 加载中。 */
  records: JournalChangeRecord[] | null;
  /** 预览该版本：传入基线快照路径（工作区内相对/绝对路径均可被 Preview 直读）。 */
  onPreview: (baselinePath: string) => void;
  /** 恢复到该版本：父级负责 RollbackRecord(record.id) + 成功 toast + 刷新时间线。 */
  onRestore: (record: JournalChangeRecord) => void | Promise<void>;
}

// 状态徽标配色：verified/applied 成功 / warned 警告 / failed 危险 / 其余中性。
function statusColor(status: string): string {
  switch (status) {
    case "verified":
    case "applied":
      return "var(--md-sys-color-success)";
    case "warned":
      return "var(--md-sys-color-warning)";
    case "failed":
      return "var(--md-sys-color-destructive)";
    default:
      return "var(--md-sys-color-text-secondary)";
  }
}

// 小图标操作按钮：与 DeliverablesPanel 同款令牌化样式（可见焦点环走全局 :focus-visible）。
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

export function VersionTimeline({ path, records, onPreview, onRestore }: VersionTimelineProps) {
  // 恢复进行中的卡 id：期间禁用所有「恢复」按钮（避免并发写盘），本行转圈。
  const [restoringId, setRestoringId] = useState<string | null>(null);

  const handleRestore = useCallback((r: JournalChangeRecord) => {
    if (restoringId) return;
    setRestoringId(r.id);
    void Promise.resolve(onRestore(r)).finally(() => {
      setRestoringId((cur) => (cur === r.id ? null : cur));
    });
  }, [onRestore, restoringId]);

  return (
    <div
      data-testid="version-timeline"
      data-path={path}
      className="mx-1 mb-1.5 flex flex-col gap-1 rounded-lg border px-2 py-1.5"
      style={{
        borderColor: "color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
        background: "color-mix(in srgb, var(--gaea-glow) 4%, transparent)",
        color: "var(--md-sys-color-text-secondary)",
      }}
    >
      {/* 标题行：文件名 + 快照计数 */}
      <div className="flex items-center gap-1 text-[10px]">
        <Clock size={11} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="shrink-0 font-medium" style={{ color: "var(--md-sys-color-text)" }}>版本时间线</span>
        <span className="min-w-0 flex-1 truncate font-mono text-[9px]" title={path}>{path}</span>
        {records && records.length > 0 && (
          <span
            className="shrink-0 rounded px-1 py-px font-mono text-[9px]"
            style={{
              color: "var(--gaea-glow)",
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
            }}
          >
            {records.length}
          </span>
        )}
      </div>
      {/* 恢复语义常驻说明：预览即护栏，不做二次确认弹窗 */}
      <div className="text-[9px] leading-relaxed" style={{ color: "var(--md-sys-color-warning)" }}>
        恢复会把该文件写回所选版本，当前内容成为新版本
      </div>
      {records === null ? (
        <div data-testid="version-timeline-loading" className="flex items-center gap-1.5 py-1.5 text-[10px]">
          <Loader2 size={11} className="animate-spin" />
          正在加载版本记录…
        </div>
      ) : records.length === 0 ? (
        <div data-testid="version-timeline-empty" className="py-1.5 text-center text-[10px] leading-relaxed">
          暂无可回滚的版本快照
          <br />
          该文件的历史变更未保留基线文件
        </div>
      ) : (
        <ol className="flex flex-col gap-0.5">
          {records.map((r) => {
            const restoring = restoringId === r.id;
            return (
              <li
                key={r.id}
                data-testid="version-timeline-row"
                className="flex items-center gap-1.5 rounded-md px-1 py-0.5 hover:bg-(color:--md-sys-color-surface-container-high)"
                title={r.afterSummary || r.beforeSummary}
              >
                <span className="shrink-0 font-mono text-[9px] tabular-nums">{versionTimeText(r)}</span>
                <span
                  className="shrink-0 rounded px-1 py-px font-mono text-[9px]"
                  style={{
                    color: "var(--md-sys-color-primary)",
                    background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                  }}
                  title={`写入工具 ${r.tool}`}
                >
                  {r.tool}
                </span>
                <span className="shrink-0 text-[9px]">{r.turn > 0 ? `第 ${r.turn} 轮` : "轮外"}</span>
                <span
                  className="shrink-0 rounded px-1 py-px text-[9px]"
                  style={{
                    color: statusColor(r.status),
                    background: "color-mix(in srgb, currentColor 10%, transparent)",
                  }}
                >
                  {versionStatusText(r.status)}
                </span>
                <span className="min-w-0 flex-1" />
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => onPreview(r.baselinePath ?? "")}
                  disabled={!r.baselinePath}
                  title="预览该版本快照"
                  aria-label={`预览 ${versionTimeText(r)} ${r.tool} 版本快照`}
                >
                  <Eye size={11} />
                </button>
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => handleRestore(r)}
                  disabled={restoringId !== null}
                  title={`恢复到 ${versionTimeText(r)} 版本：将回滚到该时间版本`}
                  aria-label={`恢复到 ${versionTimeText(r)} ${r.tool} 版本`}
                >
                  {restoring ? <Loader2 size={11} className="animate-spin" /> : <Rollback size={11} />}
                </button>
              </li>
            );
          })}
        </ol>
      )}
    </div>
  );
}
