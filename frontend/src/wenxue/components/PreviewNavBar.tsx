import { memo, useRef } from "react";
import { X } from "../icons";

// PreviewNavBar — 多文件预览队列导航条（P1-1，C7 升级）。
// 会话内预览过的文件形成队列后，在预览 pane 底部显示文件 chip 条：
//  - 每个文件一个 chip（文件名），点击切换；
//  - chip 行尾 × 关闭；中键点击关闭（按下记录目标、弹起仍落在同一 chip 才
//    关闭——对齐插件 TabBar 的 VS Code 语义防手滑，并 preventDefault 掉
//    Chrome 中键 autoscroll）；
//  - 活动项高亮（accent 底 + 语义色）。
// 单元素队列不渲染（无切换/关闭意义，与旧行为一致）。

/** 取文件 basename（路径分隔符 / 或 \ 均处理）。 */
function baseName(rel: string): string {
  const parts = rel.split(/[/\\]/).filter(Boolean);
  return parts[parts.length - 1] ?? rel;
}

export const PreviewNavBar = memo(function PreviewNavBar({
  files,
  index,
  onJump,
  onClose,
}: {
  files: string[];
  index: number;
  onJump: (index: number) => void;
  onClose: (index: number) => void;
}) {
  // 中键关闭按下时记录的目标索引；弹起时只有仍在同一 chip 才触发关闭。
  const mousedownRef = useRef<number | null>(null);

  if (files.length <= 1) return null;

  return (
    <div
      className="flex items-center gap-1 px-2 py-1 border-t border-border-soft overflow-x-auto"
      style={{ background: "var(--gaea-glass-bg, var(--md-sys-color-surface-container))" }}
      onMouseUp={() => { mousedownRef.current = null; }}
    >
      {files.map((rel, i) => {
        const active = i === index;
        return (
          <span
            key={rel}
            className={`group inline-flex items-center gap-1 max-w-48 shrink-0 rounded-md border px-2 py-0.5 text-[10px] cursor-pointer select-none ${
              active
                ? "border-accent text-accent bg-accent/10"
                : "border-border-soft text-fg-dim hover:bg-bg-soft"
            }`}
            title={rel}
            onClick={() => onJump(i)}
            onMouseDown={(e) => {
              // 中键（button === 1）：记录目标并禁掉 Chrome autoscroll；
              // 左键不参与中键关闭流程。
              if (e.button === 1) {
                e.preventDefault();
                mousedownRef.current = i;
              }
            }}
            onMouseUp={(e) => {
              if (e.button === 1 && mousedownRef.current === i) {
                mousedownRef.current = null;
                onClose(i);
              }
            }}
            role="button"
            tabIndex={0}
            aria-label={`预览 ${baseName(rel)}`}
            aria-current={active ? "true" : undefined}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onJump(i);
              }
            }}
          >
            <span className="truncate">{baseName(rel)}</span>
            <button
              type="button"
              aria-label={`关闭 ${baseName(rel)}`}
              title="关闭"
              className="shrink-0 flex items-center justify-center w-3.5 h-3.5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) opacity-0 group-hover:opacity-100 transition-opacity"
              onClick={(e) => {
                e.stopPropagation();
                onClose(i);
              }}
            >
              <X size={9} />
            </button>
          </span>
        );
      })}
    </div>
  );
});
