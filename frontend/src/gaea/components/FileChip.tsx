import { memo } from "react";
import { FileImage, FilePpt, FileSpreadsheet, FileText } from "../icons";
import { extBadge, fileIconName } from "../lib/fileBadge";

// FileChip — 行内文件引用的统一视觉（P0-2 行内文件 chip 视觉统一）。
//
// 调研 docs/2026-08-16-office-file-interaction-research.md P0-2：把 FileLinkText /
// Markdown 文件链接 / 附件 chip 从「裸路径按钮」升级为「图标 + 文件名 +
// 扩展名 badge」，与 @ 菜单共用一套视觉。点击预览行为不变。
// 视觉基线取自 FileLinkText 既有按钮样式 + FileMenu 的扩展名 badge。
// 字符串渲染（流式尾部）走 lib/fileLinks.ts 的 fileChipHtml，与本组件同构。
export const FileChip = memo(function FileChip({
  path,
  label,
  onOpen,
  compact = false,
  title,
}: {
  path: string;
  /** 展示文案：默认取路径最末段文件名。 */
  label?: string;
  onOpen?: (path: string) => void;
  /** 紧凑形态（工具输出行内，更小 padding / 更短文件名）。 */
  compact?: boolean;
  title?: string;
}) {
  const name = label ?? path.split(/[/\\]/).pop() ?? path;
  const Icon = (() => {
    switch (fileIconName(name)) {
      case "FileImage": return FileImage;
      case "FileSpreadsheet": return FileSpreadsheet;
      case "FilePpt": return FilePpt;
      default: return FileText;
    }
  })();
  return (
    <button
      type="button"
      onClick={() => onOpen?.(path)}
      title={title ?? `点击预览 ${path}`}
      aria-label={`预览 ${name}`}
      className={
        compact
          ? "inline-flex items-center gap-1 align-middle mx-0.5 px-1 py-px rounded bg-accent/10 text-accent font-mono text-[inherit] cursor-pointer hover:bg-accent/20 transition-colors"
          : "inline-flex items-center gap-1 align-middle mx-0.5 px-1.5 py-0.5 rounded-md border border-accent/25 bg-accent/5 text-accent text-[0.86em] font-medium cursor-pointer hover:bg-accent/15 transition-colors"
      }
    >
      <Icon size={compact ? 10 : 12} className="shrink-0" aria-hidden />
      <span className="max-w-[220px] truncate font-mono">{name}</span>
      <span
        className="shrink-0 text-[9px] uppercase text-fg-faint/70 border border-border-soft/60 rounded px-1 py-px font-mono"
        aria-hidden
      >
        {extBadge(name)}
      </span>
    </button>
  );
});
