import { memo, useCallback, useEffect, useState } from "react";
import {
  ExternalLink,
  File,
  FilePpt,
  FileSpreadsheet,
  FileText,
  Pin,
  Paperclip,
  RefreshCw,
  Sparkles,
} from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import type { FileSearchHit } from "../lib/types";
import { useToast } from "./Toast";

// 资料分组：文档 / 表格 / 演示 / PDF
const GROUPS: { key: string; label: string; re: RegExp; icon: typeof FileText }[] = [
  { key: "docs", label: "文档", re: /\.(docx?|md|markdown|txt)$/i, icon: FileText },
  { key: "sheets", label: "表格", re: /\.(xlsx?|csv)$/i, icon: FileSpreadsheet },
  { key: "slides", label: "演示", re: /\.(pptx?)$/i, icon: FilePpt },
  { key: "pdf", label: "PDF", re: /\.pdf$/i, icon: File },
];

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

// 小图标操作按钮（令牌化 + aria-label）
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// MaterialsPanel — 右侧「资料」视图：工作区 office/文本文件按类型分组、
// 最新在前；每条可预览，或一键 @ 引用为对话上下文（对标千问办公/aily 的
// 开工前资料盘点）。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 实底资料卡；固定区 = 主色容器强调。
export const MaterialsPanel = memo(function MaterialsPanel({
  onOpenFile,
}: {
  onOpenFile: (path: string) => void;
}) {
  const [items, setItems] = useState<FileSearchHit[]>([]);
  const [loading, setLoading] = useState(false);
  const [pinned, setPinned] = useState<FileSearchHit[]>([]);
  const [summarizing, setSummarizing] = useState<Set<string>>(new Set());
  const requestAt = useComposerInsertStore((s) => s.requestAt);
  const toast = useToast();

  const loadPinned = useCallback(() => {
    app.PinnedMaterials()
      .then((p) => setPinned(p ?? []))
      .catch(() => setPinned([]));
  }, []);
  const load = useCallback(() => {
    setLoading(true);
    app.Materials(120)
      .then((h) => setItems(h ?? []))
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
    loadPinned();
  }, [loadPinned]);
  useEffect(() => { load(); }, [load]);
  // 一键 @ 引用后也刷新固定清单（引用本身不改固定，仅保持面板同步）
  useEffect(() => { loadPinned(); }, [requestAt, loadPinned]);

  const reference = useCallback((path: string) => {
    requestAt(path);
    toast.show(`已引用 @${path}`, "info");
  }, [requestAt, toast]);

  const togglePin = useCallback((path: string) => {
    const isPinned = pinned.some((p) => p.path === path);
    const op = isPinned ? app.UnpinMaterial(path) : app.PinMaterial(path);
    op.then((next) => {
      setPinned(next ?? []);
      toast.show(isPinned ? `已取消固定 ${path}` : `已固定 ${path}（新会话自动带入上下文）`, "info");
    }).catch(() => {});
  }, [pinned, toast]);

  // 摘要后引用：分块摘要 → 把摘要文本插入输入框（对标千问/aily 摘要后引用）。
  const summarize = useCallback(async (path: string) => {
    setSummarizing((s) => new Set(s).add(path));
    try {
      const res = await app.SummarizeFile(path);
      useComposerInsertStore.getState().requestText(res.summary);
      toast.show(`已插入「${path}」的摘要到输入框，可编辑后发送`, "info");
    } catch (e) {
      toast.show(`摘要失败：${String(e)}`, "warn");
    } finally {
      setSummarizing((s) => { const n = new Set(s); n.delete(path); return n; });
    }
  }, [toast]);

  const total = items.length;
  const pinnedPaths = new Set(pinned.map((p) => p.path));
  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 计数 + 刷新 */}
      <div className="v3-panel-head">
        <Paperclip size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">资料</span>
        {total > 0 && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {total}
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button
          className={iconBtn}
          onClick={load}
          title="刷新资料列表"
          aria-label="刷新资料列表"
        >
          <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      {pinned.length > 0 && (
        <div className="px-2.5 pt-2 pb-1" style={{ borderBottom: "var(--v3-split)" }}>
          <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider font-medium mb-1.5" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <Pin size={11} aria-hidden className="text-(color:--gaea-glow)" />
            已固定 · {pinned.length}
            <span className="ml-auto normal-case tracking-normal" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              新会话自动带入上下文
            </span>
          </div>
          <div className="flex flex-col gap-1">
            {pinned.map((f) => (
              <div
                key={f.path}
                className="group flex items-center gap-2 px-2 py-1 rounded-[var(--radius-sm)] transition-all duration-200"
                style={{
                  background: "color-mix(in srgb, var(--md-sys-color-primary-container) 38%, transparent)",
                  border: "1px solid color-mix(in srgb, var(--gaea-glow) 22%, transparent)",
                  boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)",
                }}
              >
                <span
                  className="shrink-0 w-5 h-5 rounded flex items-center justify-center"
                  style={{
                    background: "color-mix(in srgb, var(--md-sys-color-primary-container) 60%, transparent)",
                    color: "var(--gaea-glow)",
                  }}
                >
                  <Pin size={10} aria-hidden />
                </span>
                <button
                  type="button"
                  onClick={() => onOpenFile(f.path)}
                  title={`点击预览 ${f.path}`}
                  className="min-w-0 flex-1 text-left cursor-pointer"
                >
                  <span className="block truncate text-[11.5px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>{f.name}</span>
                  <span className="block truncate text-[9.5px] font-mono leading-tight" style={{ color: "var(--md-sys-color-text-secondary)" }}>{f.path}</span>
                </button>
                <button
                  type="button"
                  className={iconBtn}
                  onClick={() => togglePin(f.path)}
                  title="取消固定"
                  aria-label="取消固定"
                >
                  <Pin size={11} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {total === 0 ? (
        <div
          className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          <FileText size={24} aria-hidden className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            工作区暂无资料
            <br />
            （docx / xlsx / pptx / pdf / md / csv）
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-2">
          {GROUPS.map((g) => {
            const rows = items.filter((f) => g.re.test(f.name));
            if (rows.length === 0) return null;
            const Icon = g.icon;
            return (
              <div key={g.key}>
                <div className="text-[10px] uppercase tracking-wider font-medium px-1 mb-1" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  {g.label} · {rows.length}
                </div>
                <div className="flex flex-col gap-1">
                  {rows.slice(0, 20).map((f) => (
                    <div
                      key={f.path}
                      className="group flex items-center gap-2 px-2 py-1 rounded-[var(--radius-md)] transition-all duration-200"
                      style={{
                        background: "var(--md-sys-color-surface-container)",
                        border: "1px solid var(--md-sys-color-outline-variant)",
                        boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)",
                      }}
                    >
                      <span
                        className="shrink-0 w-6 h-6 rounded-md flex items-center justify-center"
                        style={{
                          background: "color-mix(in srgb, var(--gaea-glow) 11%, transparent)",
                          color: "var(--gaea-glow)",
                        }}
                      >
                        <Icon size={12} aria-hidden />
                      </span>
                      <button
                        type="button"
                        onClick={() => onOpenFile(f.path)}
                        title={`点击预览 ${f.path}`}
                        className="min-w-0 flex-1 text-left cursor-pointer"
                      >
                        <span className="block truncate text-[12px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                          {f.name}
                          {f.size > 5 * 1024 * 1024 && (
                            <span className="ml-1.5 text-[9px] font-mono align-middle" style={{ color: "var(--md-sys-color-warning)" }}>
                              大文件
                            </span>
                          )}
                        </span>
                        <span className="block truncate text-[10px] font-mono leading-tight" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                          {f.path}
                          {f.size ? ` · ${fmtSize(f.size)}` : ""}
                        </span>
                      </button>
                      <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          type="button"
                          className={iconBtn}
                          onClick={() => void summarize(f.path)}
                          title="摘要后引用：分块摘要并插入输入框"
                          aria-label="摘要后引用"
                        >
                          <Sparkles size={12} className={summarizing.has(f.path) ? "animate-pulse" : ""} />
                        </button>
                        <button
                          type="button"
                          className={iconBtn}
                          onClick={() => togglePin(f.path)}
                          title={pinnedPaths.has(f.path) ? "取消固定" : "固定为常用资料（新会话自动带入）"}
                          aria-label={pinnedPaths.has(f.path) ? "取消固定" : "固定为常用资料"}
                          style={pinnedPaths.has(f.path) ? { color: "var(--gaea-glow)" } : undefined}
                        >
                          <Pin size={12} />
                        </button>
                        <button
                          type="button"
                          className={iconBtn}
                          onClick={() => reference(f.path)}
                          title="一键 @ 引用为对话上下文"
                          aria-label="一键引用为对话上下文"
                        >
                          <Paperclip size={12} />
                        </button>
                        <button
                          type="button"
                          className={iconBtn}
                          onClick={() => void app.OpenWorkspacePath(f.path).catch(() => {})}
                          title="在外部程序中打开"
                          aria-label="在外部程序中打开"
                        >
                          <ExternalLink size={12} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
});
