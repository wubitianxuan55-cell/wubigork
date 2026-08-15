// Composer 拆分产物：排队列表（行为零变化，T6-10.1）
import { X } from "../../icons";

export interface ComposerQueueListProps {
  queueDisplay: string[]
  onCancelItem: (index: number) => void
}

export function ComposerQueueList({ queueDisplay, onCancelItem }: ComposerQueueListProps) {
  if (queueDisplay.length === 0) return null
  return (
    <div className="mb-2 max-h-[120px] overflow-y-auto rounded-xl border border-border-soft bg-bg-elev/90 backdrop-blur-md px-2 py-1.5 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_5%,transparent)]">
      <div className="text-fg-faint/50 text-[10px] font-medium px-2 pb-1 select-none">排队中 ({queueDisplay.length})</div>
      {queueDisplay.map((item, i) => (
        <div key={i} className="flex items-center gap-2 py-1 px-2 rounded-md hover:bg-bg-soft group transition-colors duration-100">
          <span className="text-xs text-fg-dim flex-1 truncate">{item.slice(0, 80)}</span>
          <button
            className="opacity-0 group-hover:opacity-100 inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint hover:text-err hover:bg-err/10 cursor-pointer transition-all duration-150"
            onClick={() => onCancelItem(i)}
            title="取消排队"
          >
            <X size={12} />
          </button>
        </div>
      ))}
    </div>
  )
}
