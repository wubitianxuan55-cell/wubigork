import { memo, useMemo } from "react";
import type { ReactNode } from "react";
import { findFileMentions } from "../lib/fileLinks";
import { usePreviewStore } from "../lib/store";

// FileLinkText 把纯文本中的本地文件引用渲染为可点击预览按钮，
// 用于工具输出等不经过 Markdown 管道的文本表面（保留原始空白/换行）。
export const FileLinkText = memo(function FileLinkText({
  text,
  onOpen,
  compact = false,
}: {
  text: string;
  onOpen?: (path: string) => void;
  compact?: boolean;
}) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  const open = onOpen ?? openFilePreview;
  const mentions = useMemo(() => findFileMentions(text), [text]);

  if (mentions.length === 0) return <>{text}</>;

  const parts: ReactNode[] = [];
  let last = 0;
  mentions.forEach((m, i) => {
    if (m.start > last) parts.push(<span key={`t${i}`}>{text.slice(last, m.start)}</span>);
    parts.push(
      <button
        key={`f${i}`}
        type="button"
        onClick={() => open(m.path)}
        title={`点击预览 ${m.path}`}
        className={
          compact
            ? "inline-flex items-center gap-1 align-middle mx-0.5 px-1 py-px rounded bg-accent/10 text-accent font-mono text-[inherit] cursor-pointer hover:bg-accent/20 transition-colors"
            : "inline-flex items-center gap-1 align-middle mx-0.5 px-1.5 py-0.5 rounded-md border border-accent/25 bg-accent/5 text-accent text-[0.86em] font-medium cursor-pointer hover:bg-accent/15 transition-colors"
        }
      >
        {m.label}
      </button>,
    );
    last = m.end;
  });
  if (last < text.length) parts.push(<span key="t-end">{text.slice(last)}</span>);
  return <>{parts}</>;
});
