// Composer 拆分产物：主输入行（textarea + 停止 + 排队 + 发送按钮）
import { ArrowUp, Clock, Square, Zap } from "../../icons";
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
  onQueue: () => void
}

export function ComposerInputRow({
  taRef, text, onTextChange, onPaste, onKeyDown, placeholder, disabled,
  running, composerHeightFixed, dragOver, shiftHeld, queueLen,
  pendingPaste, attachmentsCount, onDrop, onDragOver, onDragLeave, onStop, onSubmit, onQueue,
}: ComposerInputRowProps) {
  const t = useT()
  // 发送按钮 glow 呼吸：就绪待发送时（未运行、有内容）才呼吸；reduced-motion 下静态
  const breathing = !running && !disabled && (text.trim().length > 0 || attachmentsCount > 0)
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
        <button className="inline-flex items-center justify-center w-[30px] h-[30px] border-0 rounded-md cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] bg-bg-elev-2 text-err shadow-[0_0_8px_color-mix(in_srgb,var(--err)_20%,transparent)] hover:bg-err hover:text-white active:scale-95" onClick={onStop} title={t("composer.stop")}>
          <Square size={14} fill="currentColor" />
        </button>
      )}
      {running && text.trim() !== "" && (
        <button
          className="inline-flex items-center justify-center w-[30px] h-[30px] border-0 rounded-md cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] bg-transparent text-fg-faint hover:text-fg hover:bg-bg-soft active:scale-95"
          onClick={onQueue}
          title={t("composer.queueSendHint")}
        >
          <Clock size={15} />
        </button>
      )}
      <button
        className={`inline-flex items-center justify-center w-[32px] h-[32px] border-0 rounded-full cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] active:scale-95 ${running ? (shiftHeld ? "bg-warn/20 text-warn hover:bg-warn hover:text-white shadow-[0_0_8px_var(--warn)]" : "bg-bg-elev-2 text-fg-dim hover:bg-accent hover:text-accent-fg hover:scale-105") : `bg-accent text-accent-fg hover:brightness-110 ${breathing ? "v3-send-breathe" : ""}`} disabled:bg-bg-elev-2 disabled:text-fg-faint disabled:cursor-default disabled:hover:scale-100 disabled:active:scale-100 disabled:shadow-none`}
        onClick={onSubmit}
        disabled={disabled || pendingPaste > 0 || (!text.trim() && attachmentsCount === 0 && (!running || queueLen === 0))}
        title={running ? (shiftHeld ? t("composer.correctTitle") : queueLen > 0 ? t("composer.queueCount", { n: queueLen }) : t("composer.steerTitle")) : t("composer.send")}
      >
        {running && shiftHeld ? (
          <Zap size={16} />
        ) : running && queueLen > 0 ? (
          <span className="text-xs font-semibold leading-none">{queueLen}</span>
        ) : (
          <ArrowUp size={16} />
        )}
      </button>
      {/* 发送按钮呼吸光晕（令牌驱动；prefers-reduced-motion 下静态） */}
      <style>{`
        @keyframes v3-send-breathe {
          0%, 100% { box-shadow: 0 0 6px color-mix(in srgb, var(--accent) 22%, transparent), 0 2px 10px color-mix(in srgb, var(--accent) 14%, transparent); }
          50% { box-shadow: 0 0 14px color-mix(in srgb, var(--accent) 36%, transparent), 0 2px 18px color-mix(in srgb, var(--accent) 22%, transparent); }
        }
        .v3-send-breathe { animation: v3-send-breathe 2.4s ease-in-out infinite; }
        @media (prefers-reduced-motion: reduce) {
          .v3-send-breathe { animation: none; box-shadow: 0 0 8px color-mix(in srgb, var(--accent) 20%, transparent); }
        }
      `}</style>
    </div>
  )
}
