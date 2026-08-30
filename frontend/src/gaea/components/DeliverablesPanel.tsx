import { memo, useCallback, useEffect, useState } from "react";
import { Archive, ClipboardList, Coins, Copy, ExternalLink, FileText, FolderTree, Loader2, MessageSquare, Paperclip, Rollback, Shield } from "../icons";
import { app } from "../lib/bridge";
import type { JournalChangeRecord, VerdictView } from "../lib/types";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { FRONTEND_EVENTS, emitFrontendEvent } from "../../events";
import { useToast } from "./Toast";
import { FileThumb } from "./FileThumb";

export interface SessionDeliverable {
  path: string;
  sourceId: string;
  turn?: number;
  /** 同一文件在会话内被提及/更新的次数（≥1）；>1 显示版本徽标与步进器。 */
  versions?: number;
}

const SPREADSHEET_EXT_RE = /\.(xlsx?|csv|et|ods)$/i;

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

// 小图标操作按钮：令牌化 + 可见焦点环（全局 :focus-visible）+ aria-label
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// DeliverablesPanel — 右侧「会话产物」视图（Codex 式工作区收尾）：
// 展示本次会话交付的全部文件（去重、最新在前），点击预览，悬停提供
// 外部打开 / 定位 / 复制路径 / 沉淀成本库；预览内编辑过的文件显示「已更新」徽标。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 低边框 hover 高亮行。
export const DeliverablesPanel = memo(function DeliverablesPanel({
  items,
  onOpenFile,
  onLocateSource,
}: {
  items: SessionDeliverable[];
  onOpenFile: (path: string) => void;
  onLocateSource?: (turn: number) => void;
}) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();
  // v4.6 失败回 Plan：逐证据卡内联展示复核结论（不再只弹 toast 一闪而过）
  const [verdicts, setVerdicts] = useState<Record<string, VerdictView>>({});

  const open = onOpenFile ?? openFilePreview;
  const copyPath = useCallback(async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show("已复制文件路径", "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  }, [toast]);

  // 沉淀到成本库：把测算/表格产物一键转为 cost_save 指令进入输入框，
  // agent 读取文件后将单价明细写回成本库（来源标注该文件，同名覆盖）。
  const depositToCost = useCallback((path: string) => {
    const name = baseName(path);
    const prompt = `请读取 [${name}](${path})，用 cost_save 把其中的单价明细沉淀到成本库：逐行提取科目/单位/单价/规格，来源标注该文件；同名条目覆盖更新，完成后汇报新增/更新条数。`;
    useComposerInsertStore.getState().requestText(prompt);
    toast.show(`已把沉淀指令插入输入框，可编辑后发送`, "info");
  }, [toast]);

  // 最新在前
  const list = [...items].reverse();

  // 复制全部路径：一次拿到本次会话全部交付物清单，便于归档或继续引用。
  const copyAllPaths = useCallback(async () => {
    const paths = list.map((d) => d.path);
    try {
      await navigator.clipboard.writeText(paths.join("\n"));
      toast.show(`已复制 ${paths.length} 个文件路径`, "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  }, [list, toast]);

  // ── v4.1 证据链：最近证据卡（「证据」入口，复用产物面板挂载点）──
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const [evidence, setEvidence] = useState<JournalChangeRecord[] | null>(null);
  useEffect(() => {
    if (!evidenceOpen) return;
    let cancelled = false;
    void app
      .GaeaJournalList(15)
      .then((recs) => { if (!cancelled) setEvidence(recs ?? []); })
      .catch(() => { if (!cancelled) setEvidence([]); });
    return () => { cancelled = true };
  }, [evidenceOpen]);

  const fmtEvidenceTime = (at: number): string => {
    if (!at) return "—";
    const diff = Date.now() - at;
    const min = Math.floor(diff / 60000);
    if (min < 1) return "刚刚";
    if (min < 60) return `${min} 分钟前`;
    const h = Math.floor(min / 60);
    if (h < 24) return `${h} 小时前`;
    return new Date(at).toLocaleString();
  };

  const verifyRecord = useCallback(async (r: JournalChangeRecord) => {
    try {
      const v = await app.VerifyRecord(r.id);
      setVerdicts((prev) => ({ ...prev, [r.id]: v }));
      const label = v.status === "verified" ? "复核通过" : v.status === "warned" ? "复核警告" : "复核未通过";
      toast.show(`${label}：${v.note ?? ""}（A:${v.channelA ?? "n/a"} / B:${v.channelB ?? "n/a"}）`, v.status === "failed" ? "warn" : "info");
    } catch (e) {
      toast.show(`复核失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [toast]);

  // v4.6 失败回 Plan：xlsx_apply 复核未通过 → 一键回到办公板块重新规划
  const replanFailed = useCallback((r: JournalChangeRecord) => {
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "office" });
    toast.show(`已打开办公面板——可对 ${r.target} 重新规划后再应用`, "info");
  }, [toast]);

  const rollbackRecord = useCallback(async (r: JournalChangeRecord) => {
    try {
      await app.RollbackRecord(r.id);
      toast.show(`已回滚 ${r.target}（基线快照恢复）`, "info");
    } catch (e) {
      toast.show(`回滚失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [toast]);

  // 打包下载：把本次会话全部交付文件打成一个 zip（对标 Kimi 工作空间 /
  // WorkBuddy 会话产物打包），完成后在文件管理器中定位 zip。
  const [zipping, setZipping] = useState(false);
  const zipDeliverables = useCallback(async () => {
    if (zipping || list.length === 0) return;
    setZipping(true);
    try {
      const r = await app.ZipDeliverables(list.map((d) => d.path));
      toast.show(`已打包 ${r.entries} 个文件（${(r.bytes / 1024).toFixed(1)} KB）`, "info");
      void app.RevealWorkspacePath(r.path).catch(() => {});
    } catch (e) {
      toast.show(`打包失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    } finally {
      setZipping(false);
    }
  }, [zipping, list, toast]);

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 计数徽标 + 复制全部 */}
      <div className="v3-panel-head">
        <FileText size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">会话产物</span>
        {items.length > 0 && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {items.length}
          </span>
        )}
        <span className="v3-panel-spacer" />
        {items.length > 0 && (
          <>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void zipDeliverables()}
              disabled={zipping}
              title="打包下载：把本次会话全部交付文件打成一个 zip"
              aria-label="打包下载全部交付文件"
            >
              {zipping ? <Loader2 size={12} className="animate-spin" /> : <Archive size={12} />}
            </button>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void copyAllPaths()}
              title="复制全部文件路径"
              aria-label="复制全部文件路径"
            >
              <ClipboardList size={12} />
            </button>
          </>
        )}
      </div>

      {items.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          <Paperclip size={24} aria-hidden className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            本轮会话暂无交付文件
            <br />
            生成/保存文件后会出现在这里
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1.5">
          {list.map(({ path, turn, versions }) => {
            const ext = extOf(path);
            const updated = updatedAt[path] != null;
            const rev = versions && versions > 1 ? versions : undefined;
            return (
              <div
                key={path}
                className="group flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors duration-150 hover:bg-(color:--md-sys-color-surface-container-high)"
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
                  onClick={() => open(path)}
                  title={`点击预览 ${path}`}
                  className="min-w-0 flex-1 text-left cursor-pointer"
                >
                  <span className="flex items-center gap-1">
                    <span className="truncate text-[12px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                      {baseName(path)}
                    </span>
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
                        已更新
                      </span>
                    )}
                    {rev && (
                      <span
                        className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none font-mono"
                        style={{
                          color: "var(--md-sys-color-primary)",
                          background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                          border: "1px solid color-mix(in srgb, var(--md-sys-color-primary) 32%, transparent)",
                        }}
                        title={`会话内更新了 ${rev} 次（产物版本时间线）`}
                      >
                        <Rollback size={8} aria-hidden />
                        v{rev}
                      </span>
                    )}
                  </span>
                  <span className="block truncate text-[10px] font-mono leading-tight" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {path}
                  </span>
                </button>
                <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                  {turn != null && onLocateSource && (
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => onLocateSource(turn)}
                      title="跳转到生成它的消息"
                      aria-label="跳转到生成它的消息"
                    >
                      <MessageSquare size={12} />
                    </button>
                  )}
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void copyPath(path)}
                    title="复制文件路径"
                    aria-label="复制文件路径"
                  >
                    <Copy size={12} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void app.OpenWorkspacePath(path).catch(() => {})}
                    title="在外部程序中打开"
                    aria-label="在外部程序中打开"
                  >
                    <ExternalLink size={12} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void app.RevealWorkspacePath(path).catch(() => {})}
                    title="在文件管理器中定位"
                    aria-label="在文件管理器中定位"
                  >
                    <FolderTree size={12} />
                  </button>
                  {SPREADSHEET_EXT_RE.test(ext) && (
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => depositToCost(path)}
                      title="沉淀到成本库：把单价明细用 cost_save 写回成本库"
                      aria-label="沉淀到成本库"
                      style={{ color: "var(--md-sys-color-warning)" }}
                    >
                      <Coins size={12} />
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* v4.1 证据链入口：最近变更证据卡（Apply→Verify→Journal 的 Journal 面） */}
      <div className="shrink-0 border-t border-(color:--md-sys-color-outline-variant)">
        <button
          type="button"
          className="w-full flex items-center gap-2 px-3 py-2 text-[11px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
          onClick={() => setEvidenceOpen((v) => !v)}
          aria-expanded={evidenceOpen}
        >
          <Shield size={13} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
          <span className="font-medium" style={{ color: "var(--md-sys-color-text)" }}>证据链</span>
          <span className="truncate text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {evidence ? `${evidence.length} 条变更证据卡` : "最近变更审计"}
          </span>
          <span className="v3-panel-spacer" />
          <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{evidenceOpen ? "收起" : "展开"}</span>
        </button>
        {evidenceOpen && (
          <div className="max-h-44 overflow-y-auto px-2 pb-2 flex flex-col gap-1">
            {evidence && evidence.length === 0 ? (
              <div className="px-2 py-2 text-[10px] text-center" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                暂无证据卡——AI 改文件后这里会记录变更原文摘要
              </div>
            ) : (
              (evidence ?? []).map((r) => {
                const v = verdicts[r.id];
                return (
                <div key={r.id} className="flex flex-col gap-1 px-2 py-1.5 rounded-md bg-(color:--md-sys-color-surface-container) border border-(color:--md-sys-color-outline-variant)">
                  <div className="flex items-center gap-2">
                  <span className="shrink-0 font-mono text-[9px] px-1 py-px rounded" style={{
                    color: "var(--md-sys-color-primary)",
                    background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                  }}>
                    {r.tool}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-[10px] font-mono" style={{ color: "var(--md-sys-color-text)" }} title={r.target}>
                    {r.target}
                  </span>
                  <span className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {fmtEvidenceTime(r.at)}
                  </span>
                  <span className="shrink-0 flex items-center gap-1">
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => void verifyRecord(r)}
                      title="双通道复核（结构/引用完整性 + 视觉健全性）"
                      aria-label="复核该证据卡"
                    >
                      <Shield size={11} />
                    </button>
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => void rollbackRecord(r)}
                      title="用基线快照回滚（目标被手工修改时拒绝）"
                      aria-label="回滚该证据卡"
                    >
                      <Rollback size={11} />
                    </button>
                  </span>
                  </div>
                  {/* v4.6：内联复核结论——failed 卡常驻显示「回滚 + 重新规划」入口 */}
                  {v && (
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="shrink-0 text-[9px] px-1 py-px rounded font-mono" style={{
                        color: v.status === "verified"
                          ? "var(--md-sys-color-success)"
                          : v.status === "warned"
                            ? "var(--md-sys-color-warning)"
                            : "var(--md-sys-color-destructive)",
                        background: "color-mix(in srgb, currentColor 10%, transparent)",
                      }}>
                        {v.status === "verified" ? "复核通过" : v.status === "warned" ? "复核警告" : "复核未通过"}
                      </span>
                      <span className="min-w-0 flex-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        {v.note ?? `${v.channelA ?? ""} / ${v.channelB ?? ""}`}
                      </span>
                      {v.status === "failed" && r.tool === "xlsx_apply" && (
                        <button
                          type="button"
                          className={iconBtn}
                          onClick={() => replanFailed(r)}
                          title="回办公面板重新规划后再应用（失败回 Plan）"
                          aria-label="重新规划"
                        >
                          <ClipboardList size={11} />
                        </button>
                      )}
                    </div>
                  )}
                </div>
                );
              })
            )}
          </div>
        )}
      </div>
    </div>
  );
});
