// DeliverablesPanel 拆分——证据链「三步展开」区块（P3 结构版2 次批巨文件拆分）。
// 内容自 DeliverablesPanel.tsx 原文件证据链入口整体迁出，逐字节照搬；行为零变化
// （含 v4.6 失败回 Plan 的「回办公面板重新规划」按钮与 v4.16 视觉复核行）。
import { ClipboardList, FolderTree, Loader2, Rollback, Shield } from "../../icons";
import { app } from "../../lib/bridge";
import type { JournalChangeRecord, VerdictView, VerifyDiffRow } from "../../lib/types";
import { describeOp, isClaimableOp, opBatchCount, opImpact, parseOps } from "../../lib/verifyDiff";
import { useT } from "../../lib/i18n";
import { VerifyArtifactsThumbs } from "../verify/VerifyArtifactsThumbs";
import { fmtEvidenceTime, iconBtn } from "./shared";

// v4.1 证据链入口：最近变更证据卡（Apply→Verify→Journal 的 Journal 面）
export function EvidenceSection({ open, onToggleOpen, evidence, verdicts, expandedId, diffs, diffStates, onToggleExpand, onVerify, onRollback, onReplan }: {
  open: boolean;
  onToggleOpen: () => void;
  evidence: JournalChangeRecord[] | null;
  verdicts: Record<string, VerdictView>;
  expandedId: string | null;
  diffs: Record<string, VerifyDiffRow[]>;
  diffStates: Record<string, "loading" | "ok" | "none">;
  onToggleExpand: (r: JournalChangeRecord) => void;
  onVerify: (r: JournalChangeRecord) => void;
  onRollback: (r: JournalChangeRecord) => void;
  onReplan: (r: JournalChangeRecord) => void;
}) {
  const t = useT();
  return (
    <div className="shrink-0 border-t border-(color:--md-sys-color-outline-variant)">
      <button
        type="button"
        className="w-full flex items-center gap-2 px-3 py-2 text-[11px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
        onClick={onToggleOpen}
        aria-expanded={open}
      >
        <Shield size={13} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
        <span className="font-medium" style={{ color: "var(--md-sys-color-text)" }}>{t("deliverPanel.evidenceTitle")}</span>
        <span className="truncate text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {evidence ? t("deliverPanel.evidenceCount", { n: evidence.length }) : t("deliverPanel.evidenceIdle")}
        </span>
        <span className="v3-panel-spacer" />
        <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{open ? t("common.collapse") : t("common.expand")}</span>
      </button>
      {open && (
        <div className="max-h-44 overflow-y-auto px-2 pb-2 flex flex-col gap-1">
          {evidence && evidence.length === 0 ? (
            <div className="px-2 py-2 text-[10px] text-center" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {t("deliverPanel.evidenceEmpty")}
            </div>
          ) : (
            (evidence ?? []).map((r) => {
              const v = verdicts[r.id];
              const ops = r.tool === "xlsx_apply" ? parseOps(r.opsJson) : [];
              const claimable = ops.some(isClaimableOp);
              const hasOps = !!r.opsJson;
              const expanded = expandedId === r.id;
              const canRollback = !!r.baselinePath;
              const diffRows = diffs[r.id];
              const diffState = diffStates[r.id];
              return (
                <EvidenceCard
                  key={r.id}
                  r={r}
                  v={v}
                  ops={ops}
                  claimable={claimable}
                  hasOps={hasOps}
                  expanded={expanded}
                  canRollback={canRollback}
                  diffRows={diffRows}
                  diffState={diffState}
                  onToggle={onToggleExpand}
                  onVerify={onVerify}
                  onRollback={onRollback}
                  onReplan={onReplan}
                />
              );
            })
          )}
        </div>
      )}
    </div>
  );
}

// 证据卡：卡面（tool 徽标 + target + 相对时间 + 复核/回滚 + verdict 内联）→
// 展开第 1 层「声明↔实况」diff（opsJson × GaeaPreview 现取）→ 第 2 层操作回放时间线。
function EvidenceCard({ r, v, ops, claimable, hasOps, expanded, canRollback, diffRows, diffState, onToggle, onVerify, onRollback, onReplan }: {
  r: JournalChangeRecord;
  v?: VerdictView;
  ops: ReturnType<typeof parseOps>;
  claimable: boolean;
  hasOps: boolean;
  expanded: boolean;
  canRollback: boolean;
  diffRows?: VerifyDiffRow[];
  diffState?: "loading" | "ok" | "none";
  onToggle: (r: JournalChangeRecord) => void;
  onVerify: (r: JournalChangeRecord) => void;
  onRollback: (r: JournalChangeRecord) => void;
  onReplan: (r: JournalChangeRecord) => void;
}) {
  const t = useT();
  return (
    <div className="flex flex-col gap-1 px-2 py-1.5 rounded-md bg-(color:--md-sys-color-surface-container) border border-(color:--md-sys-color-outline-variant)">
      {/* 卡面：点击展开/收起（三步展开第 0 层）；带 opsJson 的卡加「可复核明细」小徽标 */}
      <div
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
        aria-label={expanded ? t("deliverPanel.collapseDetail", { tool: r.tool, target: r.target }) : t("deliverPanel.expandDetail", { tool: r.tool, target: r.target })}
        onClick={() => onToggle(r)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onToggle(r);
          }
        }}
        className="flex items-center gap-2 cursor-pointer outline-none focus-visible:ring-1 focus-visible:ring-(color:--md-sys-color-primary)"
        title={hasOps ? t("deliverPanel.cardDiffTip") : t("deliverPanel.cardTip")}
      >
        <span className="shrink-0 font-mono text-[9px] px-1 py-px rounded" style={{
          color: "var(--md-sys-color-primary)",
          background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
        }}>
          {r.tool}
        </span>
        {hasOps && (
          <span
            className="shrink-0 font-mono text-[9px] px-1 py-px rounded"
            style={{
              color: "var(--md-sys-color-warning)",
              background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
            }}
            title={t("deliverPanel.opsBadgeTip")}
          >
            {t("deliverPanel.opsBadge")}
          </span>
        )}
        <span className="min-w-0 flex-1 truncate text-[10px] font-mono" style={{ color: "var(--md-sys-color-text)" }} title={r.target}>
          {r.target}
        </span>
        <span className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {fmtEvidenceTime(r.at, t)}
        </span>
        <span className="shrink-0 flex items-center gap-1">
          <button
            type="button"
            className={iconBtn}
            onClick={(e) => { e.stopPropagation(); void onVerify(r); }}
            title={t("deliverPanel.verifyBtnTitle")}
            aria-label={t("deliverPanel.verifyBtnAria")}
          >
            <Shield size={11} />
          </button>
          <button
            type="button"
            className={iconBtn + (canRollback ? "" : " disabled:opacity-40 disabled:cursor-not-allowed")}
            disabled={!canRollback}
            onClick={(e) => { e.stopPropagation(); void onRollback(r); }}
            title={canRollback ? t("deliverPanel.rollbackTitle") : t("deliverPanel.rollbackDisabled")}
            aria-label={canRollback ? t("deliverPanel.rollbackAria") : t("deliverPanel.rollbackDisabled")}
          >
            <Rollback size={11} />
          </button>
        </span>
        <span aria-hidden className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {expanded ? "▾" : "▸"}
        </span>
      </div>
      {/* 展开区：第 1 层「声明↔实况」diff → 第 2 层操作回放时间线 */}
      {expanded && (
        <div className="flex flex-col gap-1.5 pl-1">
          {hasOps && ops.length > 0 ? (
            <>
              {claimable && diffState === "loading" && (
                <div className="flex items-center gap-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  <Loader2 size={9} className="animate-spin" />
                  {t("deliverPanel.diffLoading")}
                </div>
              )}
              {claimable && diffState === "ok" && diffRows && diffRows.length > 0 && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-[9px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {t("deliverPanel.diffTitle")}
                  </span>
                  <div className="rounded-md border border-(color:--md-sys-color-outline-variant) overflow-hidden">
                    <table className="w-full text-[9px] font-mono border-collapse">
                      <tbody>
                        {diffRows.map((row, i) => (
                          <tr key={`${row.sheet}!${row.cell}-${i}`} className="border-b border-(color:--md-sys-color-outline-variant) last:border-b-0">
                            <td className="px-1.5 py-0.5 whitespace-nowrap align-top" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                              {row.sheet}!{row.cell}
                            </td>
                            <td className="px-1.5 py-0.5 line-through max-w-[120px] truncate align-top" style={{ color: "var(--md-sys-color-text-secondary)" }} title={row.claimed}>
                              {row.claimed || t("deliverPanel.emptyCell")}
                            </td>
                            <td className="px-0.5 py-0.5 align-top" style={{ color: "var(--md-sys-color-primary)" }}>→</td>
                            <td className="px-1.5 py-0.5 max-w-[160px] truncate align-top" style={{ color: row.ok === "mismatch" ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text)" }} title={row.actual}>
                              {row.actual || t("deliverPanel.emptyCell")}
                            </td>
                            <td className="px-1.5 py-0.5 align-top whitespace-nowrap" style={{ color: row.ok === "match" ? "var(--md-sys-color-success)" : row.ok === "mismatch" ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text-secondary)" }}>
                              {row.ok === "match" ? "✓" : row.ok === "mismatch" ? "✗" : t("deliverPanel.skipped")}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  <span className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {t("deliverPanel.diffNote")}
                  </span>
                </div>
              )}
              {claimable && diffState === "none" && (
                <div className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  {t("deliverPanel.diffUnavailable")}
                </div>
              )}
              {claimable && diffState === "ok" && diffRows && diffRows.length === 0 && (
                <div className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  {t("deliverPanel.diffNoCells")}
                </div>
              )}
              {/* 第 2 层：操作回放时间线（fill_range/transform 等批量 op 折叠为单行 + 计数徽标） */}
              <div className="flex flex-col gap-0.5">
                <span className="text-[9px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  {t("deliverPanel.replayTitle")}
                </span>
                <ol className="flex flex-col gap-0.5">
                  {ops.map((op, i) => {
                    const count = opBatchCount(op);
                    return (
                      <li key={i} className="flex items-center gap-1.5">
                        <span className="shrink-0 font-mono text-[8px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{i + 1}</span>
                        <span className="shrink-0 font-mono text-[8px] px-1 py-px rounded" style={{
                          color: "var(--md-sys-color-primary)",
                          background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                        }}>
                          {op.type}
                        </span>
                        <span className="min-w-0 flex-1 truncate text-[9px]" style={{ color: "var(--md-sys-color-text)" }} title={describeOp(op)}>
                          {describeOp(op)}
                        </span>
                        <span className="shrink-0 font-mono text-[8px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                          {opImpact(op)}
                        </span>
                        {count && (
                          <span className="shrink-0 font-mono text-[8px] px-1 py-px rounded" style={{
                            color: "var(--md-sys-color-warning)",
                            background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                          }}>
                            {count}
                          </span>
                        )}
                      </li>
                    );
                  })}
                </ol>
              </div>
            </>
          ) : (
            <div className="text-[9px] font-mono leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {r.beforeSummary || t("deliverPanel.noSummary")}
            </div>
          )}
        </div>
      )}
      {/* v4.6：内联复核结论——failed 卡常驻显示「回滚 + 重新规划」入口。
          v4.16：通道 B 结果产品化——verdict 携带像素差异率时追加「视觉复核」
          行（差异率 + 页数 + 查看产物按钮；产物目录是绝对路径，
          OpenWorkspacePath 直接打开目录）。旧 verdict / 无通道 B（无
          channelBRatio）不渲染该行，向后兼容。老账收口：行内追加「查看
          缩略图」入口（VerifyArtifactsThumbs，展开才列目录/取图，失败诚实
          降级），不跳出应用即可逐页复核。 */}
      {v && (
        <div className="flex flex-col gap-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="shrink-0 text-[9px] px-1 py-px rounded font-mono" style={{
              color: v.status === "verified"
                ? "var(--md-sys-color-success)"
                : v.status === "warned"
                  ? "var(--md-sys-color-warning)"
                  : "var(--md-sys-color-destructive)",
              background: "color-mix(in srgb, currentColor 10%, transparent)",
            }}>
              {v.status === "verified" ? t("deliverPanel.verifyPass") : v.status === "warned" ? t("deliverPanel.verifyWarn") : t("deliverPanel.verifyFail")}
            </span>
            <span className="min-w-0 flex-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {v.note ?? `${v.channelA ?? ""} / ${v.channelB ?? ""}`}
            </span>
            {v.status === "failed" && r.tool === "xlsx_apply" && (
              <button
                type="button"
                className={iconBtn}
                onClick={() => onReplan(r)}
                title={t("deliverPanel.replanTitle")}
                aria-label={t("deliverPanel.replan")}
              >
                <ClipboardList size={11} />
              </button>
            )}
          </div>
          {typeof v.channelBRatio === "number" && (
            <div className="flex flex-wrap items-center gap-1.5 pl-1">
              <span className="text-[9px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {t("deliverPanel.visualVerify", { ratio: (v.channelBRatio * 100).toFixed(1), pages: v.channelBPages ?? 0 })}
              </span>
              {v.channelBArtifacts && (
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => void app.OpenWorkspacePath(v.channelBArtifacts as string).catch(() => {})}
                  title={t("deliverPanel.artifactsTitle")}
                  aria-label={t("deliverPanel.artifacts")}
                >
                  <FolderTree size={11} />
                </button>
              )}
              {v.channelBArtifacts && (
                /* 老账收口：行内逐页缩略图（before/after 成对，懒加载
                   + 诚实降级），目录布局与降级语义见 verifyArtifacts.ts */
                <VerifyArtifactsThumbs artifacts={v.channelBArtifacts} />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}