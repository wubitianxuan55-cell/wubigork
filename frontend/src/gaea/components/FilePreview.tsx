import { useEffect, useState } from "react";
import { AlertCircle, ExternalLink, File, FileText, FolderTree, Loader2, X } from "../icons";
import { app } from "../lib/bridge";
import type { PreviewResult } from "../lib/types";
import { DocxPreview } from "./DocxPreview";
import { Markdown } from "./Markdown";
import { XlsxPreview } from "./XlsxPreview";

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export function FilePreview({
  relPath,
  onClose,
  onBackToFiles,
}: {
  relPath: string | null;
  onClose: () => void;
  onBackToFiles?: () => void;
}) {
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!relPath) { setPreview(null); return; }
    let live = true;
    setLoading(true);
    setPreview(null);
    app.Preview(relPath)
      .then((r) => { if (live) setPreview(r); })
      .catch(() => { if (live) setPreview({ path: relPath, name: relPath.split("/").pop() ?? relPath, ext: "", size: 0, kind: "error", body: "", dataUrl: "", error: "读取文件失败" }); })
      .finally(() => { if (live) setLoading(false); });
    return () => { live = false; };
  }, [relPath]);

  if (!relPath) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-fg-faint/40 text-xs gap-2">
        <File size={26} className="opacity-30" />
        <span>选择文件以预览</span>
      </div>
    );
  }

  const fileName = preview?.name ?? relPath.split("/").pop() ?? relPath;
  const kind = preview?.kind ?? "text";

  return (
    <div className="flex flex-col h-full text-[12px]">
      {/* 文件标题栏 */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border-soft shrink-0">
        {onBackToFiles && (
          <button
            className="flex items-center gap-1 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft"
            onClick={onBackToFiles}
            title="返回文件列表"
          >
            <FolderTree size={10} />
            文件
          </button>
        )}
        <FileText size={13} className="text-accent shrink-0" />
        <span className="font-mono text-fg truncate flex-1 text-[12px]">{fileName}</span>
        {preview && preview.size > 0 && (
          <span className="text-fg-faint text-[10px] shrink-0">{formatSize(preview.size)}</span>
        )}
        <button
          className="flex items-center gap-1 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft"
          onClick={() => app.RevealWorkspacePath(relPath).catch(() => {})}
          title="在文件管理器中定位"
        >
          <FolderTree size={10} />
        </button>
        <button
          className="flex items-center gap-1 px-2 py-0.5 border border-border-soft rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft"
          onClick={() => app.OpenWorkspacePath(relPath).catch(() => {})}
          title="在外部程序中打开"
        >
          <ExternalLink size={10} />
          打开
        </button>
        <button
          className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
          onClick={onClose}
          title="关闭预览"
        >
          <X size={13} />
        </button>
      </div>

      {/* 预览内容 */}
      <div className="flex-1 overflow-auto">
        {loading && (
          <div className="flex flex-col items-center justify-center h-full text-fg-faint text-xs gap-2">
            <Loader2 size={18} className="animate-spin text-accent" />
            <span>加载中…</span>
          </div>
        )}
        {!loading && preview?.kind === "image" && preview.dataUrl && (
          <div className="flex items-center justify-center p-4 min-h-full">
            <img src={preview.dataUrl} alt={fileName} className="max-w-full max-h-[62vh] object-contain rounded-lg shadow-sm" />
          </div>
        )}
        {!loading && preview?.kind === "docx" && preview.dataUrl && (
          <DocxPreview dataUrl={preview.dataUrl} fileName={fileName} relPath={relPath} />
        )}
        {!loading && preview?.kind === "xlsx" && (
          <XlsxPreview body={preview.body} fileName={fileName} relPath={relPath} />
        )}
        {!loading && preview?.kind === "markdown" && (
          <div className="px-4 py-3">
            <Markdown text={preview.body} />
          </div>
        )}
        {!loading && preview?.kind === "text" && (
          <pre className="p-3 text-[12px] text-fg-dim font-mono leading-relaxed whitespace-pre-wrap overflow-x-auto">{preview.body}</pre>
        )}
        {!loading && (preview?.kind === "unsupported" || preview?.kind === "error") && (
          <div className="flex flex-col items-center justify-center h-full text-fg-faint text-xs gap-3 p-4 text-center">
            <AlertCircle size={22} className={preview.kind === "error" ? "text-err/60" : "text-amber-500/60"} />
            <span className="text-fg-dim">{preview.error || "无法预览"}</span>
            <button
              className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md bg-accent text-bg text-[11px] font-medium cursor-pointer hover:opacity-90"
              onClick={() => app.OpenWorkspacePath(relPath).catch(() => {})}
            >
              <ExternalLink size={11} />
              在外部程序中打开
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
