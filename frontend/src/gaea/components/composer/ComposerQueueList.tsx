// Composer 拆分产物：排队列表 —— 排队项可撤回编辑、逐条/全部取消、
// 逐条/全部插话（Steer）、拖拽排序。发送中那条钉住；编辑/删除/Steer 整列锁定。
import { useState } from "react";
import { Ban, GripVertical, Loader2, Pencil, X, Zap } from "../../icons";
import { useT } from "../../lib/i18n";
import type { ComposerQueueItem } from "./composerQueue";
import { isQueueBusy, pendingItems } from "./composerQueue";

export interface ComposerQueueListProps {
  items: ComposerQueueItem[]
  running: boolean
  onCancelItem: (index: number) => void
  onEditItem: (index: number) => void
  onSteerItem: (index: number) => void
  onSteerAll: () => void
  onCancelAll: () => void
  onReorder: (from: number, to: number) => void
}

export function ComposerQueueList({
  items, running, onCancelItem, onEditItem, onSteerItem, onSteerAll, onCancelAll, onReorder,
}: ComposerQueueListProps) {
  const t = useT()
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dragOver, setDragOver] = useState<number | null>(null)
  if (items.length === 0) return null
  const busy = isQueueBusy(items)
  const pending = pendingItems(items)
  const bulkDisabled = busy || pending.length === 0
  const clearDrag = () => { setDragFrom(null); setDragOver(null) }
  return (
    <div
      data-testid="composer-queue"
      className="mb-2 max-h-[140px] overflow-y-auto rounded-xl border border-border-soft bg-bg-elev/90 backdrop-blur-md px-2 py-1.5 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_5%,transparent)]"
    >
      <div className="flex items-center gap-1.5 text-fg-faint/50 text-[10px] font-medium px-2 pb-1 select-none">
        <span>{t("composer.queueBadge", { n: items.length })}</span>
        {busy ? (
          <span data-testid="composer-queue-sending" className="text-accent/80">{t("composer.queueSending")}</span>
        ) : (
          <span className="text-fg-faint/40">{t("composer.queueEditHint")}</span>
        )}
        <span className="ml-auto inline-flex items-center gap-0.5">
          {running && (
            <button
              type="button"
              data-testid="composer-queue-steer-all"
              className="inline-flex items-center gap-0.5 h-5 px-1.5 rounded border-0 bg-transparent text-[10px] text-fg-faint hover:text-accent hover:bg-accent/10 cursor-pointer disabled:opacity-40 disabled:cursor-default disabled:hover:text-fg-faint disabled:hover:bg-transparent"
              disabled={bulkDisabled}
              onClick={onSteerAll}
              title={busy ? t("composer.queueLockedHint") : t("composer.queueSteerAllTitle")}
            >
              <Zap size={10} />
              {t("composer.queueSteerAll")}
            </button>
          )}
          <button
            type="button"
            data-testid="composer-queue-cancel-all"
            className="inline-flex items-center gap-0.5 h-5 px-1.5 rounded border-0 bg-transparent text-[10px] text-fg-faint hover:text-err hover:bg-err/10 cursor-pointer disabled:opacity-40 disabled:cursor-default disabled:hover:text-fg-faint disabled:hover:bg-transparent"
            disabled={bulkDisabled}
            onClick={onCancelAll}
            title={busy ? t("composer.queueLockedHint") : t("composer.queueCancelAllTitle")}
          >
            <Ban size={10} />
            {t("composer.queueCancelAll")}
          </button>
        </span>
      </div>
      {items.map((item, i) => {
        const sending = item.status === "sending"
        const locked = busy
        const dragging = dragFrom === i
        const over = dragOver === i && dragFrom !== null && dragFrom !== i
        return (
          <div
            key={item.id}
            data-testid={`composer-queue-item-${i}`}
            data-status={item.status}
            onDragOver={(e) => {
              if (dragFrom === null) return
              e.preventDefault()
              e.dataTransfer.dropEffect = "move"
              if (dragOver !== i) setDragOver(i)
            }}
            onDrop={(e) => {
              e.preventDefault()
              if (dragFrom !== null && dragFrom !== i) onReorder(dragFrom, i)
              clearDrag()
            }}
            onDragEnd={clearDrag}
            className={`group flex items-center gap-1.5 py-1 pl-2 pr-1 rounded-md transition-[opacity,background-color,box-shadow] duration-100 ${sending ? "bg-accent/[0.06]" : "hover:bg-bg-soft"} ${dragging ? "opacity-40" : ""} ${over ? "shadow-[inset_0_0_0_1px_var(--accent)]" : ""}`}
          >
            {sending ? (
              <span className="shrink-0 w-4" />
            ) : (
              <button
                type="button"
                draggable
                data-testid={`composer-queue-grip-${i}`}
                className="shrink-0 inline-flex items-center justify-center w-4 h-5 border-0 rounded bg-transparent text-fg-faint/50 hover:text-fg-dim cursor-grab active:cursor-grabbing"
                title={t("composer.queueDragTitle")}
                onDragStart={(e) => {
                  e.dataTransfer.setData("text/plain", String(i))
                  e.dataTransfer.effectAllowed = "move"
                  setDragFrom(i)
                }}
                onClick={(e) => e.preventDefault()}
              >
                <GripVertical size={12} />
              </button>
            )}
            <span className="shrink-0 text-[9px] font-mono text-fg-faint/40 tabular-nums select-none">{i + 1}</span>
            <button
              type="button"
              className="flex items-center gap-1.5 flex-1 min-w-0 text-left cursor-pointer rounded px-1 py-0.5 -mx-1 hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-(color:--gaea-glow)/40 disabled:cursor-default disabled:hover:text-inherit"
              onClick={() => { if (!locked) onEditItem(i) }}
              disabled={locked}
              title={locked ? t("composer.queueLockedHint") : t("composer.queueWithdrawTitle")}
            >
              <span className="text-xs text-fg-dim flex-1 truncate">{item.text.slice(0, 80)}</span>
              {sending ? (
                <span className="inline-flex items-center gap-0.5 shrink-0 text-[10px] text-accent">
                  <Loader2 size={11} className="composer-queue-spin" />
                  {t("composer.queueSending")}
                </span>
              ) : (
                <Pencil size={11} className="shrink-0 opacity-0 group-hover:opacity-100 text-fg-faint/60 transition-opacity" />
              )}
            </button>
            {running && !sending && (
              <button
                type="button"
                data-testid={`composer-queue-steer-${i}`}
                className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint hover:text-accent hover:bg-accent/10 cursor-pointer transition-all duration-150 disabled:opacity-40 disabled:cursor-default disabled:hover:text-fg-faint disabled:hover:bg-transparent"
                disabled={locked}
                onClick={() => onSteerItem(i)}
                title={locked ? t("composer.queueLockedHint") : t("composer.queueSteerTitle")}
              >
                <Zap size={12} />
              </button>
            )}
            <button
              type="button"
              className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint hover:text-err hover:bg-err/10 cursor-pointer transition-all duration-150 disabled:opacity-40 disabled:cursor-default disabled:hover:text-fg-faint disabled:hover:bg-transparent"
              disabled={locked}
              onClick={() => onCancelItem(i)}
              title={locked ? t("composer.queueLockedHint") : t("composer.queueCancelTitle")}
            >
              <X size={12} />
            </button>
          </div>
        )
      })}
      <style>{`
        @keyframes composer-queue-spin { to { transform: rotate(360deg); } }
        .composer-queue-spin { animation: composer-queue-spin 0.9s linear infinite; }
        @media (prefers-reduced-motion: reduce) {
          .composer-queue-spin { animation: none; }
        }
      `}</style>
    </div>
  )
}
