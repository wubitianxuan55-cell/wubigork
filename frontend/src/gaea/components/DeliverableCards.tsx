import { memo, useEffect, useMemo, useState } from "react";
import {
  Copy,
  ExternalLink,
  FolderTree,
} from "../icons";
import { app } from "../lib/bridge";
import { deliverableMentions } from "../lib/fileLinks";
import {
  deliverablePathKey,
  mergeDeliverableCards,
  useTurnDeliverables,
} from "../lib/deliverablesTurn";
import {
  deliverablePhaseOf,
  ensureOfficeJournal,
  isOfficeDeliverablePath,
} from "../lib/officeTurnProjection";
import type { JournalChangeRecord } from "../lib/types";
import { useUpdatedFilesStore } from "../lib/store";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { useT } from "../lib/i18n";
import { useToast } from "./Toast";
import { FileThumb } from "./FileThumb";

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// DeliverableCards — 回复尾部的交付物列表（Codex 式收尾）：
// 两路合并——正文启发式（deliverableMentions，保持出现顺序在前）+ 权威产物
// 登记表「本轮」条目（工具写出但正文未提及的路径，在后追加，去重），渲染成
// 纵向列表——缩略图/类型图标 + 文件名（主）+ 相对路径（辅），整行点击打开
// 内置预览，悬停提供外部打开 / 定位 / 复制路径；预览内编辑过的文件显示
// 「已更新」徽标。登记口径 = 派发即登记，登记-only 卡确认文件不存在时
// 灰色淡化 + 「未生成」徽标（缺失态探测失败按存在处理，宁漏勿误）。
// 安静、低边框，突出文件名。
//
// U2 统一 Office 回合卡（docs/gaea-dsh-univer-office-distill-plan-2026-09.md
// §4.2-2/4）：Office 文档卡叠加草稿/就绪状态徽标（首次写盘=草稿、Plan→Apply
// 批准=就绪，判定源=证据链 JournalList + 本轮登记条目，见 lib/
// officeTurnProjection.deliverablePhaseOf；数据不可用时不标，宁缺勿误）。
// 本卡与 DeliverablesPanel 的 VersionTimeline 共用同一徽标语言与判定函数——
// 呈现层收敛，但本卡不复制操作 footer：保留/回滚/验收等操作仍只在登记面
// （DeliverablesPanel）与证据链入口，登记表数据源与产物置前行为不变。
export const DeliverableCards = memo(function DeliverableCards({ text, turnNo, mergeRegistry = true }: { text: string; turnNo?: number; mergeRegistry?: boolean }) {
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();
  const t = useT();
  // 本轮登记条目（模块级共享缓存，多卡一次 fetch；失败静默降级 → 仅正文卡）。
  // mergeRegistry=false（非轮尾段）不拉取：登记-only 卡只在轮尾段渲染一次，
  // 同轮多条回复不重复挂同一批卡（走查 2026-09-03 噪点）。
  const { entries, missing } = useTurnDeliverables(mergeRegistry ? turnNo : undefined);

  const textPaths = useMemo(() => deliverableMentions(text), [text]);
  const cards = useMemo(
    () => mergeDeliverableCards(textPaths, entries, turnNo),
    [textPaths, entries, turnNo],
  );
  // 草稿/就绪徽标数据源：证据链 JournalList（会话级，与版本时间线/回滚同源）。
  // 只在卡片里存在 Office 文档时拉取（共享缓存 2s 去重；失败静默 → 不标徽标）。
  // 注意：hooks 全部在 cards.length 早退之前调用（Rules of Hooks）。
  const hasOfficeCard = cards.some((c) => isOfficeDeliverablePath(c.path));
  const [journal, setJournal] = useState<JournalChangeRecord[] | null>(null);
  useEffect(() => {
    if (!hasOfficeCard) return;
    let cancelled = false;
    void ensureOfficeJournal().then((recs) => {
      if (!cancelled) setJournal(recs);
    });
    return () => {
      cancelled = true;
    };
  }, [hasOfficeCard]);
  // 本轮登记条目索引（路径归一键 → 工具名）：正文卡的草稿判定兜底。
  const entryToolByPath = useMemo(() => {
    const m = new Map<string, string>();
    for (const e of entries) m.set(deliverablePathKey(e.path), e.tool);
    return m;
  }, [entries]);
  if (cards.length === 0) return null;

  const copyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show(t("deliver.copyPathDone"), "info");
    } catch {
      toast.show(t("deliver.copyFail"), "warn");
    }
  };

  return (
    <div className="mt-2 flex flex-col gap-1.5" aria-label={t("deliver.title")}>
      <div className="flex items-center gap-1.5 select-none">
        <span className="text-[10px] uppercase tracking-wider font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {t("deliver.title")}
        </span>
        <span
          className="rounded-full px-1.5 py-px text-[9px] font-mono leading-none"
          style={{
            color: "var(--md-sys-color-text-secondary)",
            border: "1px solid var(--md-sys-color-outline-variant)",
          }}
        >
          {cards.length}
        </span>
      </div>
      <div className="flex flex-col gap-1">
        {cards.map(({ path, from }) => {
          const ext = extOf(path);
          const updated = updatedAt[path] != null;
          // 缺失态：登记-only 且已确认文件不存在（登记 = 派发即登记，写失败
          // 不回剔）→ 整行灰色淡化 +「未生成」徽标；正文卡不做缺失判定。
          const isMissing = from === "registry" && missing.has(deliverablePathKey(path));
          // U2 草稿/就绪徽标：判定源 = 证据链（JournalList）+ 本轮登记条目；
          // 已确认缺失（登记-only 未生成）的卡不标（写失败谈不上草稿）。
          const regTool = entryToolByPath.get(deliverablePathKey(path));
          const phase = isMissing
            ? null
            : deliverablePhaseOf(path, journal, regTool ? { tool: regTool } : null);
          return (
            <div
              key={path}
              className="group flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors duration-150 hover:bg-(color:--md-sys-color-surface-container-high)"
              style={isMissing ? { opacity: 0.55 } : undefined}
            >
              <span
                className="shrink-0 w-8 h-8 rounded-md flex items-center justify-center overflow-hidden"
                style={{
                  background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
                  color: "var(--gaea-glow)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
                  filter: isMissing ? "grayscale(1)" : undefined,
                }}
              >
                <FileThumb path={path} ext={ext} imgClassName="w-8 h-8 object-cover rounded-md" />
              </span>
              <button
                type="button"
                onClick={() => openPaneFileOrPreview(path)}
                title={t("msg.clickPreview", { path })}
                className="min-w-0 flex-1 text-left cursor-pointer rounded-md px-1 py-0.5 -mx-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(color:--gaea-glow)/40"
              >
                <span className="flex items-center gap-1.5 min-w-0">
                  <span className="truncate text-[13px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                    {baseName(path)}
                  </span>
                  {updated && (
                    <span
                      className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
                      style={{
                        color: "var(--md-sys-color-success)",
                        background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                        border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",
                      }}
                    >
                      {t("deliver.updated")}
                    </span>
                  )}
                  {/* U2 草稿/就绪徽标：与 VersionTimeline 头部同一判定函数、同一
                      胶囊语言（草稿=警示色/就绪=成功色）；title 说明判定口径。 */}
                  {phase === "draft" && (
                    <span
                      data-testid="office-turn-phase-draft"
                      className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
                      title={t("officeTurn.phaseDraftTitle")}
                      style={{
                        color: "var(--md-sys-color-warning)",
                        background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                        border: "1px solid color-mix(in srgb, var(--md-sys-color-warning) 32%, transparent)",
                      }}
                    >
                      {t("officeTurn.phaseDraft")}
                    </span>
                  )}
                  {phase === "ready" && (
                    <span
                      data-testid="office-turn-phase-ready"
                      className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
                      title={t("officeTurn.phaseReadyTitle")}
                      style={{
                        color: "var(--md-sys-color-success)",
                        background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                        border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",
                      }}
                    >
                      {t("officeTurn.phaseReady")}
                    </span>
                  )}
                  {isMissing && (
                    <span
                      className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
                      style={{
                        color: "var(--md-sys-color-text-secondary)",
                        background: "color-mix(in srgb, var(--md-sys-color-outline-variant) 40%, transparent)",
                        border: "1px solid var(--md-sys-color-outline-variant)",
                      }}
                    >
                      {t("deliver.missing")}
                    </span>
                  )}
                </span>
                <span
                  className="block truncate text-[10.5px] font-mono leading-tight mt-px"
                  style={{ color: "var(--md-sys-color-text-secondary)" }}
                >
                  {path}
                </span>
              </button>
              <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => void copyPath(path)}
                  title={t("deliver.copyPath")}
                  aria-label={t("deliver.copyPath")}
                >
                  <Copy size={12} />
                </button>
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => void app.OpenWorkspacePath(path).catch(() => {})}
                  title={t("deliver.openExternal")}
                  aria-label={t("deliver.openExternal")}
                >
                  <ExternalLink size={12} />
                </button>
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => void app.RevealWorkspacePath(path).catch(() => {})}
                  title={t("deliver.reveal")}
                  aria-label={t("deliver.reveal")}
                >
                  <FolderTree size={12} />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
});
