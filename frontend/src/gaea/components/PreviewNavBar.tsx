import { memo } from "react";

// PreviewNavBar — 多文件预览队列导航条（P1-1）。
// 会话内预览过的文件形成队列后,在预览 pane 底部显示 ← index/total →,
// 供在多文件间来回切换（对标 QwenPaw 多文件预览 / Hermes 版本步进器的
// 会话内文件导航）。单元素队列不渲染（无切换意义）。
export const PreviewNavBar = memo(function PreviewNavBar({
  index,
  total,
  onPrev,
  onNext,
}: {
  index: number;
  total: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  if (total <= 1) return null;
  return (
    <div
      className="flex items-center justify-center gap-2 py-1 border-t border-border-soft"
      style={{ background: "var(--gaea-glass-bg, var(--md-sys-color-surface-container))" }}
    >
      <button
        type="button"
        className="flex items-center gap-0.5 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-35 disabled:cursor-default"
        onClick={onPrev}
        disabled={index <= 0}
        title="上一个文件"
        aria-label="上一个文件"
      >
        ←
      </button>
      <span className="text-[10px] font-mono text-fg-dim">
        {index + 1}/{total}
      </span>
      <button
        type="button"
        className="flex items-center gap-0.5 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-35 disabled:cursor-default"
        onClick={onNext}
        disabled={index >= total - 1}
        title="下一个文件"
        aria-label="下一个文件"
      >
        →
      </button>
    </div>
  );
});
