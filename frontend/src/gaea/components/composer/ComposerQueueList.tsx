// Composer 拆分产物：排队列表 —— 排队项可点击撤回输入框编辑（对齐
// agentsroom 消息队列 / vm0 withdraw：点卡片调回输入框改完重新入队），
// 也保留逐条取消。
import { Pencil, X } from "../../icons";

export interface ComposerQueueListProps {
  queueDisplay: string[]
  onCancelItem: (index: number) => void
  onEditItem: (index: number) => void
}

export function ComposerQueueList({ queueDisplay, onCancelItem, onEditItem }: ComposerQueueListProps) {
  if (queueDisplay.length === 0) return null
  return (
    <div className="mb-2 max-h-[120px] overflow-y-auto rounded-xl border border-border-soft bg-bg-elev/90 backdrop-blur-md px-2 py-1.5 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_5%,transparent)]">
      <div className="flex items-center gap-1.5 text-fg-faint/50 text-[10px] font-medium px-2 pb-1 select-none">
        <span>排队中 ({queueDisplay.length})</span>
        <span className="text-fg-faint/40">· 点击可撤回编辑</span>
      </div>
      {queueDisplay.map((item, i) => (
        <div key={i} className="group flex items-center gap-1.5 py-1 pl-2 pr-1 rounded-md hover:bg-bg-soft transition-colors duration-100">
          <span className="shrink-0 text-[9px] font-mono text-fg-faint/40 tabular-nums select-none">{i + 1}</span>
          <button
            type="button"
            className="flex items-center gap-1.5 flex-1 min-w-0 text-left cursor-pointer rounded px-1 py-0.5 -mx-1 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(color:--gaea-glow)/40"
            onClick={() => onEditItem(i)}
            title="撤回输入框编辑"
          >
            <span className="text-xs text-fg-dim flex-1 truncate">{item.slice(0, 80)}</span>
            <Pencil size={11} className="shrink-0 opacity-0 group-hover:opacity-100 text-fg-faint/60 transition-opacity" />
          </button>
          <button
            className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint hover:text-err hover:bg-err/10 cursor-pointer transition-all duration-150"
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
