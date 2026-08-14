// Composer 拆分产物：主输入行（textarea + 停止 + 发送按钮，行为零变化，T6-10.1）
import { ArrowUp, Square, Zap } from "../../icons";
import { useT } from "../../lib/i18n";

export interface ComposerInputRowProps {
  taRef: React.RefObject<HTMLTextAreaElement>
  text: string
  onTextChange: (v: string) => void
  onPaste: (e: React.ClipboardEvent<HTMLTextAreaElement>) => void
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void
  placeholder: string
  disabled: boolean
  running: boolean
  composerHeightFixed: boolean
  dragOver: boolean
  shiftHeld: boolean
  queueLen: number
  pendingPaste: number
  attachmentsCount: number
  onDrop: (e: React.DragEvent<HTMLDivElement>) => void
  onDragOver: (e: React.DragEvent<HTMLDivElement>) => void
  onDragLeave: () => void
  onStop: () => void
  onSubmit: () => void
}

export function ComposerInputRow({
  taRef, text, onTextChange, onPaste, onKeyDown, placeholder, disabled,
  running, composerHeightFixed, dragOver, shiftHeld, queueLen,
  pendingPaste, attachmentsCount, onDrop, onDragOver, onDragLeave, onStop, onSubmit,
}: ComposerInputRowProps) {
  const t = useT()
  return (
    <div
      className={`flex gap-2 items-center shrink-0 min-h-0 bg-transparent border-0 border-b border-border-soft rounded-none px-[13px] py-2.5 ${composerHeightFixed ? "flex-1 items-start" : ""} ${dragOver ? "outline outline-1 outline-dashed outline-accent outline-offset-[-4px] bg-accent-[0.02]" : ""} ${disabled ? "opacity-50 pointer-events-none" : ""}`}
      onDrop={onDrop} onDragOver={onDragOver} onDragLeave={onDragLeave}
    >
      <span className="text-accent font-mono font-semibold text-lg leading-[1.55] shrink-0 select-none">›</span>
      <textarea
        ref={taRef}
        className={`flex-1 resize-none border-0 bg-transparent text-fg leading-[1.55] max-h-[200px] outline-none placeholder:text-fg-faint ${composerHeightFixed ? "h-full max-h-none overflow-y-auto" : ""}`}
        style={{ fieldSizing: "content" }}
        value={text} onChange={(e) => onTextChange(e.target.value)}
        onPaste={onPaste} onKeyDown={onKeyDown}
        placeholder={placeholder}
        rows={1} disabled={disabled}
      />
      {running && (
        <button className="inline-flex items-center justify-center w-[30px] h-[30px] border-0 rounded-md cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] bg-bg-elev-2 text-err hover:bg-err hover:text-white active:scale-95" onClick={onStop} title={t("composer.stop")}>
          <Square size={14} fill="currentColor" />
        </button>
      )}
      <button
        className={`inline-flex items-center justify-center w-[32px] h-[32px] border-0 rounded-full cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] active:scale-95 ${running ? (shiftHeld ? "bg-warn/20 text-warn hover:bg-warn hover:text-white shadow-[0_0_8px_var(--warn)]" : "bg-bg-elev-2 text-fg-dim hover:bg-accent hover:text-accent-fg hover:scale-105") : "bg-accent text-accent-fg hover:brightness-110"} disabled:bg-bg-elev-2 disabled:text-fg-faint disabled:cursor-default disabled:hover:scale-100 disabled:active:scale-100 disabled:shadow-none`}
        style={!running && !disabled ? {boxShadow: "var(--ds-shadow-accent-btn)"} : undefined}
        onClick={onSubmit}
        disabled={disabled || pendingPaste > 0 || (!text.trim() && attachmentsCount === 0 && (!running || queueLen === 0))}
        title={running ? (shiftHeld ? "纠正发送（Shift+Enter）" : queueLen > 0 ? `排队发送 (${queueLen})` : t("composer.queue")) : t("composer.send")}
      >
        {running && shiftHeld ? (
          <Zap size={16} />
        ) : running && queueLen > 0 ? (
          <span className="text-xs font-semibold leading-none">{queueLen}</span>
        ) : (
          <ArrowUp size={16} />
        )}
      </button>
    </div>
  )
}
