import { memo, useMemo } from "react";
import {
  Copy,
  ExternalLink,
  FolderTree,
} from "../icons";
import { app } from "../lib/bridge";
import { deliverableMentions } from "../lib/fileLinks";
import { usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { useToast } from "./Toast";
import { FileThumb, FileTypeIcon, IMAGE_EXT_RE } from "./FileThumb";

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

// DeliverableCards — 回复尾部的交付物附件卡片（对齐千问办公 / Kimi 形态）：
// 从正文中提取文件引用，按文件去重后渲染成"缩略图/图标 + 文件名 + 扩展名"卡片，
// 整卡点击打开内置预览，悬停提供外部打开 / 定位 / 复制路径；预览内编辑过的文件
// 显示「已更新」徽标。
export const DeliverableCards = memo(function DeliverableCards({ text }: { text: string }) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();

  const items = useMemo(() => deliverableMentions(text), [text]);
  if (items.length === 0) return null;

  const copyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show("已复制文件路径", "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  };

  return (
    <div className="mt-2 flex flex-col gap-1.5" aria-label="交付文件">
      <div className="text-[10px] uppercase tracking-wider text-fg-faint/60 font-medium select-none">
        交付文件
      </div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((path) => {
          const ext = extOf(path);
          const isImage = IMAGE_EXT_RE.test(ext);
          const updated = updatedAt[path] != null;
          return (
            <div
              key={path}
              className="group flex items-center gap-2 px-2.5 py-1.5 rounded-lg border border-border-soft bg-bg-soft/40 hover:border-accent/30 hover:bg-bg-soft/70 transition-colors max-w-full"
            >
              <span className="shrink-0 w-7 h-7 rounded-md bg-accent/10 text-accent flex items-center justify-center overflow-hidden">
                {isImage ? <FileThumb path={path} ext={ext} /> : <FileTypeIcon ext={ext} size={14} />}
              </span>
              <button
                type="button"
                onClick={() => openFilePreview(path)}
                title={`点击预览 ${path}`}
                className="min-w-0 text-left cursor-pointer"
              >
                <span className="flex items-center gap-1 max-w-[260px]">
                  <span className="truncate text-[12.5px] text-fg font-medium leading-tight">
                    {baseName(path)}
                  </span>
                  {updated && (
                    <span className="shrink-0 text-[9px] text-ok border border-ok/30 bg-ok/10 rounded-full px-1 py-px leading-none">
                      已更新
                    </span>
                  )}
                </span>
                <span className="block max-w-[260px] truncate text-[10.5px] text-fg-faint font-mono leading-tight">
                  {path}
                </span>
              </button>
              <span className="shrink-0 text-[10px] uppercase text-fg-faint/70 font-mono border border-border-soft/60 rounded px-1 py-px select-none">
                {ext.slice(1)}
              </span>
              <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
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
    </div>
  );
});
