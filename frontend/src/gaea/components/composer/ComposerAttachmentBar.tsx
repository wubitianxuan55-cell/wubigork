// Composer 拆分产物：附件预览条（识图/OCR/移除，行为零变化，T6-10.1）
import { FileText, Eye, Loader, X } from "../../icons";
import type { Attachment } from "../../hooks/useComposerAttachments";

export interface ComposerAttachmentBarProps {
  attachments: Attachment[]
  running: boolean
  recognizingPath: string | null
  ocrPath: string | null
  onRecognize: (a: Attachment) => void
  onOCRText: (a: Attachment) => void
  onRemove: (path: string) => void
}

export function ComposerAttachmentBar({
  attachments, running, recognizingPath, ocrPath, onRecognize, onOCRText, onRemove,
}: ComposerAttachmentBarProps) {
  if (attachments.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1.5 px-1 pb-1.5">
      {attachments.map((a) => (
        <div className="flex items-center gap-1.5 pl-1.5 pr-1 py-1 bg-bg-elev-2 border border-border-soft rounded-lg text-xs" key={a.path}>
          {a.type === "image" ? (
            <img src={a.previewUrl} alt="" className="w-8 h-8 rounded object-cover" />
          ) : (
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-accent shrink-0">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
            </svg>
          )}
          <span className="max-w-[120px] truncate text-fg-dim font-mono text-[11px]">{a.path.split("/").pop()}</span>
          {a.type === "image" && (
            <button
              type="button"
              className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-dim cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
              title={running ? "助手回复中，稍后再试" : "识图：本地视觉模型识别图片内容"}
              disabled={running || !!recognizingPath}
              onClick={() => onRecognize(a)}
            >
              {recognizingPath === a.path ? <Loader size={12} className="animate-spin" /> : <Eye size={12} />}
            </button>
          )}
          {a.type === "image" && (
            <button
              type="button"
              className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-dim cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
              title={running ? "助手回复中，稍后再试" : "提取文字：本地 OvisOCR2 识别图中文字"}
              disabled={running || !!ocrPath}
              onClick={() => onOCRText(a)}
            >
              {ocrPath === a.path ? <Loader size={12} className="animate-spin" /> : <FileText size={12} />}
            </button>
          )}
          <button type="button" className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-faint cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" title="移除" onClick={() => onRemove(a.path)}><X size={13} /></button>
        </div>
      ))}
    </div>
  )
}
