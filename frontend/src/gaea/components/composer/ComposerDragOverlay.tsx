// Composer 拆分产物：拖放指示器（行为零变化，T6-10.1）
export function ComposerDragOverlay({ show }: { show: boolean }) {
  if (!show) return null
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center rounded-2xl bg-accent/10 border-2 border-dashed border-accent/40 backdrop-blur-[2px] pointer-events-none animate-[fadeIn_0.15s_ease-out]">
      <div className="flex flex-col items-center gap-2 text-accent">
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" x2="12" y1="15" y2="3" />
        </svg>
        <span className="text-sm font-medium">释放以添加文件</span>
      </div>
    </div>
  )
}
