import { memo, useMemo } from "react";
import type { ReactNode } from "react";
import { findFileMentions } from "../lib/fileLinks";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { FileChip } from "./FileChip";

// FileLinkText 把纯文本中的本地文件引用渲染为可点击预览 chip，
// 用于工具输出等不经过 Markdown 管道的文本表面（保留原始空白/换行）。
// 视觉统一（P0-2）：行内文件引用全部走 FileChip —— 图标 + 文件名 + 扩展名 badge。
export const FileLinkText = memo(function FileLinkText({
  text,
  onOpen,
  compact = false,
}: {
  text: string;
  onOpen?: (path: string) => void;
  compact?: boolean;
}) {
  const open = onOpen ?? openPaneFileOrPreview;
  const mentions = useMemo(() => findFileMentions(text), [text]);

  if (mentions.length === 0) return <>{text}</>;

  const parts: ReactNode[] = [];
  let last = 0;
  mentions.forEach((m, i) => {
    if (m.start > last) parts.push(<span key={`t${i}`}>{text.slice(last, m.start)}</span>);
    parts.push(
      <FileChip
        key={`f${i}`}
        path={m.path}
        onOpen={open}
        compact={compact}
        title={`点击预览 ${m.path}`}
      />,
    );
    last = m.end;
  });
  if (last < text.length) parts.push(<span key="t-end">{text.slice(last)}</span>);
  return <>{parts}</>;
});
