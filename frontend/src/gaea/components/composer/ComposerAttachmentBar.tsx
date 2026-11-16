// Composer 拆分产物：附件预览条（识图/OCR/移除，T6-10.1）
// P0-1 扩展：非图片附件 chip 化 —— file 型附件渲染为「图标 + 文件名 +
// 扩展名 badge」chip，点击可预览（调研 2026-08-16 P0-1）。
import { Eye, File as FileIcon, FileText, Image as ImageIcon, Loader, RefreshCw, X } from "../../icons";
import { extBadge, fileIconName } from "../../lib/fileBadge";
import { usePreviewStore } from "../../lib/store";
import type { Attachment, AttachmentUploadTask } from "../../hooks/useComposerAttachments";

export interface ComposerAttachmentBarProps {
  attachments: Attachment[]
  uploads?: AttachmentUploadTask[]
  running: boolean
  recognizingPath: string | null
  ocrPath: string | null
  onRecognize: (a: Attachment) => void
  onOCRText: (a: Attachment) => void
  onRemove: (path: string) => void
  onRetryUpload?: (id: string) => void
  onRemoveUpload?: (id: string) => void
}

function FileTypeIcon({ path, size }: { path: string; size: number }) {
  const name = path.split(/[/\\]/).pop() ?? path;
  switch (fileIconName(name)) {
    case "FileImage": return <ImageIcon size={size} className="text-accent shrink-0" />;
    default: return <FileIcon size={size} className="text-accent shrink-0" />;
  }
}

// P2-4 附件上下文占用透明化：字节数 → 可读大小（KB/MB）。
function formatBytes(n: number | undefined): string | null {
  if (typeof n !== "number" || !isFinite(n) || n <= 0) return null;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export function ComposerAttachmentBar({
  attachments, uploads = [], running, recognizingPath, ocrPath, onRecognize, onOCRText, onRemove,
  onRetryUpload, onRemoveUpload,
}: ComposerAttachmentBarProps) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  if (attachments.length === 0 && uploads.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1.5 px-1 pb-1.5">
      {uploads.map((task) => {
        const sizeLabel = formatBytes(task.size)
        const progress = Math.max(0, Math.min(100, task.progress ?? 0))
        const failed = task.status === "error"
        return (
          <div
            key={task.id}
            data-testid={`attachment-upload-${task.id}`}
            data-status={task.status}
            className={`flex min-w-[190px] max-w-[320px] flex-col gap-1 rounded-lg border px-2 py-1.5 text-xs shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_4%,transparent)] ${
              failed ? "border-err/30 bg-err/5" : "border-border-soft/70 bg-bg-elev-2/90"
            }`}
          >
            <div className="flex min-w-0 items-center gap-1.5">
              {task.type === "image" && task.previewUrl ? (
                <img src={task.previewUrl} alt="" className="h-7 w-7 shrink-0 rounded object-cover" />
              ) : (
                <FileTypeIcon path={task.name} size={15} />
              )}
              <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-fg-dim" title={task.name}>
                {task.name}
              </span>
              {sizeLabel && <span className="shrink-0 font-mono text-[9px] text-fg-faint/70">{sizeLabel}</span>}
              {failed && onRetryUpload && task.retryable && (
                <button
                  type="button"
                  className="flex h-5 w-5 shrink-0 items-center justify-center rounded border-0 bg-transparent text-accent hover:bg-accent/10"
                  title="重试上传"
                  aria-label={`重试上传 ${task.name}`}
                  onClick={() => onRetryUpload(task.id)}
                >
                  <RefreshCw size={12} />
                </button>
              )}
              {failed && onRemoveUpload && (
                <button
                  type="button"
                  className="flex h-5 w-5 shrink-0 items-center justify-center rounded border-0 bg-transparent text-fg-faint hover:bg-bg-soft hover:text-err"
                  title="移除失败任务"
                  aria-label={`移除失败上传 ${task.name}`}
                  onClick={() => onRemoveUpload(task.id)}
                >
                  <X size={12} />
                </button>
              )}
            </div>
            {task.status === "uploading" ? (
              <div className="flex items-center gap-1.5">
                <div className="h-1 flex-1 overflow-hidden rounded-full bg-border-soft/60">
                  <div className="h-full rounded-full bg-accent transition-[width] duration-150" style={{ width: `${progress}%` }} />
                </div>
                <span className="shrink-0 text-[9px] text-fg-faint">上传中{progress > 0 ? ` · ${progress}%` : ""}</span>
              </div>
            ) : (
              <div className="flex min-w-0 items-center gap-1 text-[9.5px] text-err">
                <span className="min-w-0 truncate" title={task.error}>{task.error || "上传失败"}</span>
                {!task.retryable && <span className="shrink-0 text-fg-faint">不可重试</span>}
              </div>
            )}
          </div>
        )
      })}
      {attachments.map((a) => {
        const name = a.path.split(/[/\\]/).pop() ?? a.path;
        const isImage = a.type === "image";
        const sizeLabel = formatBytes(a.size);
        return (
          <div className="flex items-center gap-1.5 pl-1.5 pr-1 py-1 bg-bg-elev-2/90 border border-border-soft/70 rounded-lg shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_4%,transparent)] text-xs" key={a.path}>
            {isImage && a.previewUrl ? (
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  className="border-0 bg-transparent rounded p-0 cursor-pointer"
                  title={`点击预览 ${a.path}`}
                  aria-label={`预览 ${name}`}
                  onClick={() => openFilePreview(a.path)}
                >
                  <img src={a.previewUrl} alt="" className="w-8 h-8 rounded object-cover" />
                </button>
                {sizeLabel && (
                  <span className="text-[9px] text-fg-faint/70 font-mono shrink-0" title="附件占用（进入上下文的体量）">{sizeLabel}</span>
                )}
              </div>
            ) : (
              <button
                type="button"
                className="flex items-center gap-1 pl-0.5 pr-1 py-0.5 border-0 bg-transparent rounded text-fg-dim cursor-pointer hover:bg-bg-soft transition-colors"
                title={`点击预览 ${a.path}`}
                aria-label={`预览 ${name}`}
                onClick={() => openFilePreview(a.path)}
              >
                <FileTypeIcon path={a.path} size={16} />
                <span className="max-w-[120px] truncate font-mono text-[11px]">{name}</span>
                <span className="shrink-0 text-[9px] uppercase text-fg-faint/70 border border-border-soft/60 rounded px-1 py-px font-mono">
                  {extBadge(name)}
                </span>
                {sizeLabel && (
                  <span className="shrink-0 text-[9px] text-fg-faint/70 font-mono" title="附件占用（进入上下文的体量）">{sizeLabel}</span>
                )}
              </button>
            )}
            {isImage && (
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
            {isImage && (
              <button
                type="button"
                className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-dim cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                title={running ? "助手回复中，稍后再试" : "提取文字：本地 uocr2 OCR 识别图中文字"}
                disabled={running || !!ocrPath}
                onClick={() => onOCRText(a)}
              >
                {ocrPath === a.path ? <Loader size={12} className="animate-spin" /> : <FileText size={12} />}
              </button>
            )}
            <button type="button" className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-faint cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" title="移除" onClick={() => onRemove(a.path)}><X size={13} /></button>
          </div>
        )
      })}
    </div>
  )
}
