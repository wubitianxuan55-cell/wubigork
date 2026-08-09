import { memo, useCallback } from "react";
import { Copy, ExternalLink, File, FileImage, FilePpt, FileSpreadsheet, FileText, FolderTree, MessageSquare, Paperclip } from "../icons";
import { app } from "../lib/bridge";
import { usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { useToast } from "./Toast";

export interface SessionDeliverable {
  path: string;
  sourceId: string;
  turn?: number;
}

const IMAGE_EXT_RE = /\.(png|jpe?g|gif|webp|svg|bmp|ico)$/i;

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

function FileTypeIcon({ ext, size }: { ext: string; size: number }) {
  if (/\.(xlsx?|csv|et|ods)$/i.test(ext)) return <FileSpreadsheet size={size} />;
  if (/\.(pptx?|dps|odp)$/i.test(ext)) return <FilePpt size={size} />;
  if (IMAGE_EXT_RE.test(ext)) return <FileImage size={size} />;
  if (/\.(docx?|pdf|md|markdown|txt|odt|rtf|wps|ofd|html?)$/i.test(ext)) return <FileText size={size} />;
  return <File size={size} />;
}

// DeliverablesPanel — 右侧「会话产物」视图（对标 Kimi 工作空间 / 千问办公产物面板）：
// 展示本次会话交付的全部文件（去重、最新在前），点击预览，悬停提供
// 外部打开 / 定位 / 复制路径；预览内编辑过的文件显示「已更新」徽标。
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

  const open = onOpenFile ?? openFilePreview;
  const copyPath = useCallback(async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show("已复制文件路径", "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  }, [toast]);

  // 最新在前
  const list = [...items].reverse();

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <FileText size={13} className="text-accent" />
          会话产物
        </span>
        {items.length > 0 && (
          <span className="text-[10px] text-fg-faint border border-border-soft/60 rounded-full px-1.5 py-px">
            {items.length}
          </span>
        )}
      </div>

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center text-fg-faint/50">
          <Paperclip size={24} className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            本轮会话暂无交付文件
            <br />
            生成/保存文件后会出现在这里
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1">
          {list.map(({ path, turn }) => {
            const ext = extOf(path);
            const updated = updatedAt[path] != null;
            return (
              <div
                key={path}
                className="group flex items-center gap-2 px-2 py-1.5 rounded-lg border border-border-soft/70 bg-bg-soft/30 hover:border-accent/30 hover:bg-bg-soft/60 transition-colors"
              >
                <span className="shrink-0 w-7 h-7 rounded-md bg-accent/10 text-accent flex items-center justify-center">
                  <FileTypeIcon ext={ext} size={14} />
                </span>
                <button
                  type="button"
                  onClick={() => open(path)}
                  title={`点击预览 ${path}`}
                  className="min-w-0 flex-1 text-left cursor-pointer"
                >
                  <span className="flex items-center gap-1">
                    <span className="truncate text-[12px] text-fg font-medium leading-tight">
                      {baseName(path)}
                    </span>
                    {updated && (
                      <span className="shrink-0 text-[9px] text-ok border border-ok/30 bg-ok/10 rounded-full px-1 py-px leading-none">
                        已更新
                      </span>
                    )}
                  </span>
                  <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">
                    {path}
                  </span>
                </button>
                <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                  {turn != null && onLocateSource && (
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                      onClick={() => onLocateSource(turn)}
                      title="跳转到生成它的消息"
                    >
                      <MessageSquare size={12} />
                    </button>
                  )}
                  <button
                    type="button"
                    className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                    onClick={() => void copyPath(path)}
                    title="复制文件路径"
                  >
                    <Copy size={12} />
                  </button>
                  <button
                    type="button"
                    className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                    onClick={() => void app.OpenWorkspacePath(path).catch(() => {})}
                    title="在外部程序中打开"
                  >
                    <ExternalLink size={12} />
                  </button>
                  <button
                    type="button"
                    className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                    onClick={() => void app.RevealWorkspacePath(path).catch(() => {})}
                    title="在文件管理器中定位"
                  >
                    <FolderTree size={12} />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
});
