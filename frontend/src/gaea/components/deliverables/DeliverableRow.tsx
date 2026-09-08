// DeliverablesPanel 拆分——单条产物行（P3 结构版2 次批巨文件拆分）。
// 内容自 DeliverablesPanel.tsx 原文件产物列表行整体迁出，逐字节照搬；行为零变化
// （含 data-testid / v4.28 B1 时间线 / v4.31 A1 单版本入口 / v4.32 徽标细化 / A1 验收闭环）。
import { Fragment } from "react";
import { AlertCircle, CheckCircle, Coins, Copy, ExternalLink, FileText, FolderTree, ListTree, MessageSquare, RefreshCw, Rollback, Sparkles } from "../../icons";
import { app } from "../../lib/bridge";
import type { JournalChangeRecord } from "../../lib/types";
import type { DeliverableAcceptance } from "../../lib/deliverableStatus";
import { normalizeVersionPath } from "../../lib/versionTimeline";
import { useT } from "../../lib/i18n";
import { FileThumb } from "../FileThumb";
import { VersionTimeline } from "../VersionTimeline";
import { SPREADSHEET_EXT_RE, baseName, extOf, iconBtn } from "./shared";

// 产物行：名称按钮（新/已更新/验收徽标）+ 版本时间线入口 + 行内操作位
// （定位消息/树中定位/复制/外部打开/定位/沉淀成本库/验收标记）。
export function DeliverableRow({
  item, fresh, updated, accept, accAvailable, journal, groupedVersions, timelinePath,
  onOpen, onCopyPath, onDeposit, onLocateSource, onRevealInTree, onToggleTimeline,
  onApplyAcceptance, onRestoreVersion,
}: {
  item: { path: string; turn?: number; versions?: number };
  fresh: boolean;
  updated: boolean;
  accept: DeliverableAcceptance;
  accAvailable: boolean;
  journal: JournalChangeRecord[] | null;
  groupedVersions: Map<string, JournalChangeRecord[]>;
  timelinePath: string | null;
  onOpen: (path: string) => void;
  onCopyPath: (path: string) => void;
  onDeposit: (path: string) => void;
  onLocateSource?: (turn: number) => void;
  onRevealInTree?: (path: string) => void;
  onToggleTimeline: (normPath: string) => void;
  onApplyAcceptance: (path: string, status: DeliverableAcceptance) => void;
  onRestoreVersion: (r: JournalChangeRecord) => void;
}) {
  const t = useT();
  const { path, turn, versions } = item;
  const ext = extOf(path);
  // v4.28 B1：时间线展开 key 用归一化路径（与 JournalList target 对齐）
  const normPath = normalizeVersionPath(path);
  // v4.28 B1：vN 次数徽标——versions>1 才显示次数（>1 显示版本徽标与步进器）。
  const rev = versions && versions > 1 ? versions : undefined;
  // v4.31 A1：单版本入口——versions≤1 但该路径在 journal 分组（由
  // GaeaJournalList(200) 折叠、只留有 baselinePath 的卡）中存在条目时，
  // 同样渲染时间线入口徽标（收 v4.28 欠账「单版本无入口」）。
  const journalEntry = groupedVersions.has(normPath);
  // v4.32：单版本「版本」徽标 title 带快照数（收 v4.31 欠账「静态文案」）；
  // 快照数取不到（理论上 journalEntry 为真必有 ≥1 条）回落原静态文案。
  const snapshotCount = groupedVersions.get(normPath)?.length;
  const badgeTitle = rev
    ? t("deliverPanel.badgeUpdated", { n: rev })
    : snapshotCount
      ? t("deliverPanel.badgeSnapshots", { n: snapshotCount })
      : t("deliverPanel.badgeHistory");
  const timelineOpen = timelinePath === normPath;

  return (
    <Fragment>
      <div
        className="group flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors duration-150 hover:bg-(color:--md-sys-color-surface-container-high)"
        data-fresh={fresh ? "true" : undefined}
        style={fresh ? { background: "color-mix(in srgb, var(--md-sys-color-primary) 10%, transparent)" } : undefined}
      >
        <span
          className="shrink-0 w-8 h-8 rounded-md flex items-center justify-center overflow-hidden"
          style={{
            background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
            color: "var(--gaea-glow)",
            border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
          }}
        >
          <FileThumb path={path} ext={ext} imgClassName="w-8 h-8 object-cover rounded-md" />
        </span>
        <button
          type="button"
          onClick={() => onOpen(path)}
          title={t("msg.clickPreview", { path })}
          className="min-w-0 flex-1 text-left cursor-pointer"
        >
          <span className="flex items-center gap-1">
            <span className="truncate text-[12px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
              {baseName(path)}
            </span>
            {fresh && (
              <span
                className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                style={{
                  color: "var(--md-sys-color-primary)",
                  background: "color-mix(in srgb, var(--md-sys-color-primary) 14%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-primary) 34%, transparent)",
                }}
              >
                <Sparkles size={8} aria-hidden />
                {t("deliverPanel.fresh")}
              </span>
            )}
            {updated && (
              <span
                className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                style={{
                  color: "var(--md-sys-color-success)",
                  background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",
                }}
              >
                <FileText size={8} aria-hidden />
                {t("deliver.updated")}
              </span>
            )}
            {/* A1 验收徽标：仅显示已验收（绿）/要求修改（警示）两态；
                open 是缺省态不显示徽标（多数行初始即 open，弱化徽标反成噪声），
                悬停操作位承载标记入口。样式沿「新」「已更新」同族胶囊。 */}
            {accept === "confirmed" && (
              <span
                data-testid="deliverable-accept-badge"
                className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                style={{
                  color: "var(--md-sys-color-success)",
                  background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",
                }}
              >
                <CheckCircle size={8} aria-hidden />
                {t("accept.statusConfirmed")}
              </span>
            )}
            {accept === "redo" && (
              <span
                data-testid="deliverable-accept-badge"
                className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                style={{
                  color: "var(--md-sys-color-warning)",
                  background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-warning) 32%, transparent)",
                }}
              >
                <AlertCircle size={8} aria-hidden />
                {t("accept.statusRedo")}
              </span>
            )}
          </span>
          <span className="block truncate text-[10px] font-mono leading-tight transition-opacity duration-150 group-hover:opacity-100 opacity-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {path}
          </span>
        </button>
        {/* v4.28 B1：vN 次数徽标改为可点按钮——点开内联版本时间线
            （逐版本列表 + 预览 + 恢复）；作为名称按钮的兄弟节点避免
            button 嵌套。展开时徽标加浓提示当前处于时间线视图。
            v4.31 A1：versions≤1 但有 journal 快照的产物同样渲染
            「版本」入口徽标（收 v4.28 欠账「单版本无入口」），title
            措辞区分「更新 N 次」与「有版本历史」两种语义。
            v4.32：非 rev 分支 title 带快照数（收「静态文案」欠账）。 */}
        {(rev || journalEntry) && (
          <button
            type="button"
            onClick={() => onToggleTimeline(normPath)}
            aria-expanded={timelineOpen}
            title={badgeTitle}
            aria-label={t("deliverPanel.timelineAria", { name: baseName(path) })}
            className="shrink-0 inline-flex cursor-pointer items-center gap-0.5 rounded-full px-1.5 py-px text-[9px] leading-none font-mono transition-colors"
            style={{
              color: "var(--md-sys-color-primary)",
              background: timelineOpen
                ? "color-mix(in srgb, var(--md-sys-color-primary) 24%, transparent)"
                : "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
              border: "1px solid color-mix(in srgb, var(--md-sys-color-primary) 32%, transparent)",
            }}
          >
            <Rollback size={8} aria-hidden />
            {rev ? `v${rev}` : t("deliverPanel.versionsBadge")}
          </button>
        )}
        <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
          {turn != null && onLocateSource && (
            <button
              type="button"
              className={iconBtn}
              onClick={() => onLocateSource(turn)}
              title={t("deliverPanel.locateMessage")}
              aria-label={t("deliverPanel.locateMessage")}
            >
              <MessageSquare size={12} />
            </button>
          )}
          {onRevealInTree && (
            <button
              type="button"
              className={iconBtn}
              onClick={() => onRevealInTree(path)}
              title={t("deliverPanel.revealTreeTitle")}
              aria-label={t("deliverPanel.revealTree")}
            >
              <ListTree size={12} />
            </button>
          )}
          <button
            type="button"
            className={iconBtn}
            onClick={() => onCopyPath(path)}
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
          {SPREADSHEET_EXT_RE.test(ext) && (
            <button
              type="button"
              className={iconBtn}
              onClick={() => onDeposit(path)}
              title={t("deliverPanel.depositTitle")}
              aria-label={t("deliverPanel.deposit")}
              style={{ color: "var(--md-sys-color-warning)" }}
            >
              <Coins size={12} />
            </button>
          )}
          {/* A1 验收操作（与行内 icon 按钮同排）：open 行出「标记已验收/
              要求修改」；已标记行换「重新查看」恢复 open（记录删除，
              操作位复原）——两段式操作位，避免已定态行排三按钮。 */}
          {accAvailable && (accept === "open" ? (
            <>
              <button
                type="button"
                className={iconBtn}
                data-testid="deliverable-accept-confirm"
                onClick={() => onApplyAcceptance(path, "confirmed")}
                title={t("accept.confirmAction")}
                aria-label={t("accept.confirmAction")}
              >
                <CheckCircle size={12} />
              </button>
              <button
                type="button"
                className={iconBtn}
                data-testid="deliverable-accept-redo"
                onClick={() => onApplyAcceptance(path, "redo")}
                title={t("accept.redoAction")}
                aria-label={t("accept.redoAction")}
              >
                <AlertCircle size={12} />
              </button>
            </>
          ) : (
            <button
              type="button"
              className={iconBtn}
              data-testid="deliverable-accept-reopen"
              onClick={() => onApplyAcceptance(path, "open")}
              title={t("accept.reopenAction")}
              aria-label={t("accept.reopenAction")}
            >
              <RefreshCw size={12} />
            </button>
          ))}
        </div>
      </div>
      {/* v4.28 B1 内联版本时间线：徽标点开后在该产物行下方展开。
          records 三态——journal=null 加载态 / 空数组空态 / 有记录列表；
          预览/恢复经回调注入（open=onOpenFile ?? openFilePreview，
          onRestoreVersion=RollbackRecord + toast + 重拉时间线）。 */}
      {timelineOpen && (
        <VersionTimeline
          path={path}
          records={journal === null ? null : (groupedVersions.get(normPath) ?? [])}
          onPreview={onOpen}
          onRestore={onRestoreVersion}
        />
      )}
    </Fragment>
  );
}