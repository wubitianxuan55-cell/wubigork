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
// 从正文中提取文件引用，按文件去重后渲染成纵向列表——缩略图/类型图标 + 文件名
// （主）+ 相对路径（辅），整行点击打开内置预览，悬停提供外部打开 / 定位 /
// 复制路径；预览内编辑过的文件显示「已更新」徽标。安静、低边框，突出文件名。
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
      <div className="flex items-center gap-1.5 select-none">
        <span className="text-[10px] uppercase tracking-wider font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          交付文件
        </span>
        <span
          className="rounded-full px-1.5 py-px text-[9px] font-mono leading-none"
          style={{
            color: "var(--md-sys-color-text-secondary)",
            border: "1px solid var(--md-sys-color-outline-variant)",
          }}
        >
          {items.length}
        </span>
      </div>
      <div className="flex flex-col gap-1">
        {items.map((path) => {
          const ext = extOf(path);
          const updated = updatedAt[path] != null;
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
                onClick={() => openFilePreview(path)}
                title={`点击预览 ${path}`}
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
                      已更新
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
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
});
