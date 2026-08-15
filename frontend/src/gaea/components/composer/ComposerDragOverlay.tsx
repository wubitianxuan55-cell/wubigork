// Composer 拆分产物：拖放指示器（行为零变化，T6-10.1）
import { CloudUpload } from "../../icons";

export function ComposerDragOverlay({ show }: { show: boolean }) {
  if (!show) return null
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center rounded-2xl bg-accent/10 border-2 border-dashed border-accent/40 backdrop-blur-[2px] pointer-events-none animate-[fadeIn_0.15s_ease-out]">
      <div className="flex flex-col items-center gap-2 text-accent">
        <CloudUpload size={36} />
        <span className="text-sm font-medium">释放以添加文件</span>
      </div>
    </div>
  )
}
