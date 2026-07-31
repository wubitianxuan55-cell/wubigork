import { useState, useEffect } from "react";
import { File, ExternalLink, AlertCircle } from "lucide-react";
import { app } from "../lib/bridge";

export function FilePreview({ relPath, onClose }: { relPath: string | null; onClose: () => void }) {
  const [preview, setPreview] = useState<{ text?: string; err?: string; isImage?: boolean; dataUrl?: string } | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!relPath) { setPreview(null); return; }
    setLoading(true);
    setPreview(null);

    const ext = relPath.split(".").pop()?.toLowerCase();
    const imageExts = ["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg"];

    if (imageExts.includes(ext ?? "")) {
      app.AttachmentDataURL(relPath).then((url) => {
        setPreview({ isImage: true, dataUrl: url });
        setLoading(false);
      }).catch(() => {
        // Not in attachments, try ReadFile's content
        app.ReadFile(relPath).then((r) => {
          if (r.err) setPreview({ err: r.err });
          else setPreview({ text: r.body ?? "(图片文件，请使用外部程序打开)" });
          setLoading(false);
        }).catch(() => {
          setPreview({ err: "无法预览" });
          setLoading(false);
        });
      });
    } else {
      app.ReadFile(relPath).then((r) => {
        if (r.err) setPreview({ err: r.err });
        else if (r.binary) setPreview({ text: `(二进制文件，${r.size ?? "?"} 字节)` });
        else setPreview({ text: r.body ?? "(空文件)" });
        setLoading(false);
      }).catch(() => {
        setPreview({ err: "无法读取文件" });
        setLoading(false);
      });
    }
  }, [relPath]);

  if (!relPath) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-fg-faint/40 text-xs gap-2">
        <File size={24} className="opacity-30" />
        <span>选择文件以预览</span>
      </div>
    );
  }

  const fileName = relPath.split("/").pop() ?? relPath;

  return (
    <div className="flex flex-col h-full text-[12px]">
      {/* 文件标题栏 */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border-soft">
        <span className="font-mono text-fg truncate flex-1 text-[12px]">{fileName}</span>
        <button
          className="flex items-center gap-1 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft"
          onClick={() => app.OpenWorkspacePath(relPath)}
          title="在外部程序中打开"
        >
          <ExternalLink size={10} />
          打开
        </button>
        <button
          className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
          onClick={onClose}
        >
          ✕
        </button>
      </div>

      {/* 预览内容 */}
      <div className="flex-1 overflow-auto">
        {loading && (
          <div className="flex items-center justify-center h-full text-fg-faint text-xs">加载中⋯</div>
        )}
        {preview?.err && (
          <div className="flex flex-col items-center justify-center h-full text-err/60 text-xs gap-2 p-4">
            <AlertCircle size={20} />
            <span>{preview.err}</span>
          </div>
        )}
        {preview?.isImage && preview.dataUrl && (
          <div className="flex items-center justify-center p-4">
            <img src={preview.dataUrl} alt={fileName} className="max-w-full max-h-[60vh] object-contain rounded-lg shadow-sm" />
          </div>
        )}
        {preview?.text !== undefined && !preview.isImage && (
          <pre className="p-3 text-[11px] text-fg-dim font-mono leading-relaxed whitespace-pre-wrap overflow-x-auto">{preview.text}</pre>
        )}
      </div>
    </div>
  );
}
