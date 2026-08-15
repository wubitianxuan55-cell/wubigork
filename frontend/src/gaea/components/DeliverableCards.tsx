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

const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// DeliverableCards — 回复尾部的交付物附件卡片（对齐千问办公 / Kimi 形态）：
// 从正文中提取文件引用，按文件去重后渲染成"缩略图/图标 + 文件名 + 扩展名"卡片，
// 整卡点击打开内置预览，悬停提供外部打开 / 定位 / 复制路径；预览内编辑过的文件
// 显示「已更新」徽标。
// v3「星枢」面板语言：实底 v3-card 高光线（不叠玻璃），hover 柔光。
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
      <div className="text-[10px] uppercase tracking-wider font-medium select-none" style={{ color: "var(--md-sys-color-text-secondary)" }}>
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
              className="group flex items-center gap-2 px-2.5 py-1.5 rounded-[var(--radius-md)] max-w-full transition-all duration-200 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--md-sys-color-text)_5%,transparent)] hover:shadow-[var(--v3-glow-soft)]"
              style={{
                background: "var(--md-sys-color-surface-container)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
            >
              <span
                className="shrink-0 w-7 h-7 rounded-md flex items-center justify-center overflow-hidden"
                style={{
                  background: "color-mix(in srgb, var(--gaea-glow) 11%, transparent)",
                  color: "var(--gaea-glow)",
                }}
              >
                {isImage ? <FileThumb path={path} ext={ext} /> : <FileTypeIcon ext={ext} size={14} />}
              </span>
              <button
                type="button"
                onClick={() => openFilePreview(path)}
                title={`点击预览 ${path}`}
                className="min-w-0 text-left cursor-pointer"
              >
                <span className="flex items-center gap-1 max-w-[260px]">
                  <span className="truncate text-[12.5px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                    {baseName(path)}
                  </span>
                  {updated && (
                    <span
                      className="shrink-0 rounded-full px-1 py-px text-[9px] leading-none"
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
                  className="block max-w-[260px] truncate text-[10.5px] font-mono leading-tight"
                  style={{ color: "var(--md-sys-color-text-secondary)" }}
                >
                  {path}
                </span>
              </button>
              <span
                className="shrink-0 text-[10px] uppercase font-mono rounded px-1 py-px select-none"
                style={{
                  color: "var(--md-sys-color-text-secondary)",
                  border: "1px solid var(--md-sys-color-outline-variant)",
                }}
              >
                {ext.slice(1)}
              </span>
              <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
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
