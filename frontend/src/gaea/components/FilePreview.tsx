import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, Check, ExternalLink, File, FileText, FolderTree, Loader2, Pencil, X } from "../icons";
import { app } from "../lib/bridge";
import type { PreviewResult } from "../lib/types";
import { DocxPreview } from "./DocxPreview";
import { Markdown } from "./Markdown";
import { PptxOutline } from "./PptxOutline";
import { XlsxPreview } from "./XlsxPreview";
import { usePreviewProgress } from "../hooks/usePreviewProgress";
import { useToast } from "./Toast";

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

// v4.29 化繁为简：头部按钮统一去边框（无边框 + 悬停浅底），图标优先；
// 编辑/保存/取消等带状态语义的动作保留文字（编辑能力保留红线）。
const HEAD_BTN =
  "flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft";

export function FilePreview({
  relPath,
  onClose,
  onBackToFiles,
  embedded = false,
}: {
  relPath: string | null;
  onClose: () => void;
  onBackToFiles?: () => void;
  /** 嵌入式渲染（v4.25 A3 右栏编辑器 tab）：默认 false 行为完全不变；
   *  true 时隐藏头部文件名（tab 条已展示同名字样，280–720px 窄栏省宽），
   *  预览/编辑/OCR/docx/xlsx 能力原样保留。 */
  embedded?: boolean;
}) {
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);
  const ocrProgress = usePreviewProgress(relPath);
  const toast = useToast();
  // C5 工作区内联编辑：编辑态 / 草稿 / 脏标记 / 保存状态机 / 放弃确认
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [dirty, setDirty] = useState(false);
  const [saveState, setSaveState] = useState<"idle" | "saving" | "saved" | "failed">("idle");
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  // v4.28 B2 pptx 逐页预览：页图容器（大纲卡点页条目按 data-pptx-page 锚点滚动）
  const pptxPagesRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!relPath) { setPreview(null); return; }
    let live = true;
    setLoading(true);
    setPreview(null);
    setEditing(false);
    setDraft("");
    setDirty(false);
    setSaveState("idle");
    app.Preview(relPath)
      .then((r) => { if (live) setPreview(r); })
      .catch(() => { if (live) setPreview({ path: relPath, name: relPath.split("/").pop() ?? relPath, ext: "", size: 0, kind: "error", body: "", dataUrl: "", error: "读取文件失败" }); })
      .finally(() => { if (live) setLoading(false); });
    return () => { live = false; };
  }, [relPath]);

  // 可编辑 = 纯文本/markdown 预览且未截断（截断内容不完整，写回会丢数据）
  const editable = preview !== null && !loading && !editing && (preview.kind === "markdown" || preview.kind === "text") && !preview.truncated;

  const startEdit = useCallback(() => {
    if (!preview) return;
    setDraft(preview.body);
    setDirty(false);
    setSaveState("idle");
    setEditing(true);
  }, [preview]);

  const save = useCallback(async () => {
    if (!relPath) return;
    setSaveState("saving");
    try {
      await app.WriteFile(relPath, draft);
      setDirty(false);
      setSaveState("saved");
      toast.show("已保存到工作区", "info");
      // 保存后刷新预览（内容回读），稍后复位状态机
      const r = await app.Preview(relPath).catch(() => null);
      if (r) setPreview(r);
      window.setTimeout(() => setSaveState((s) => (s === "saving" ? s : "idle")), 0);
    } catch (e) {
      setSaveState("failed");
      toast.show(`保存失败：${String(e)}`, "error");
    }
  }, [relPath, draft, toast]);

  const cancelEdit = useCallback(() => {
    if (!dirty) {
      setEditing(false);
      setConfirmDiscard(false);
      return;
    }
    setConfirmDiscard(true);
  }, [dirty]);

  const discardAndExit = useCallback(() => {
    setEditing(false);
    setDraft("");
    setDirty(false);
    setSaveState("idle");
    setConfirmDiscard(false);
  }, []);

  // v4.28 B2：点大纲页条目 → 滚动到逐页渲染区的对应页锚点。
  //（jsdom 无 scrollIntoView，可选调用守卫；测试里注入 spy 验证。）
  const scrollToPptxPage = useCallback((page: number) => {
    const el = pptxPagesRef.current?.querySelector<HTMLElement>(`[data-pptx-page="${page}"]`);
    el?.scrollIntoView?.({ block: "start", behavior: "smooth" });
  }, []);

  // Ctrl/Cmd+S 保存
  useEffect(() => {
    if (!editing) return;
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        e.preventDefault();
        if (dirty) void save();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [editing, dirty, save]);

  if (!relPath) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-fg-faint/40 text-xs gap-2">
        <File size={26} className="opacity-30" />
        <span>选择文件以预览</span>
      </div>
    );
  }

  const fileName = preview?.name ?? relPath.split("/").pop() ?? relPath;

  return (
    <div className="flex flex-col h-full text-[12px]">
      {/* 文件标题栏 */}
      <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border-soft shrink-0">
        {onBackToFiles && (
          <button
            className={HEAD_BTN}
            onClick={onBackToFiles}
            title="返回文件列表"
          >
            <FolderTree size={10} />
            文件
          </button>
        )}
        <FileText size={13} className="text-accent shrink-0" />
        {embedded ? (
          // 嵌入式：tab 条已展示文件名，头部留 flex 占位对齐右侧操作区
          <span className="flex-1" />
        ) : (
          <span className="font-mono text-fg truncate flex-1 text-[12px]">{fileName}</span>
        )}
        {dirty && (
          <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse shrink-0" title="有未保存的修改" />
        )}
        {preview && preview.size > 0 && (
          <span className="text-fg-faint text-[10px] shrink-0">{formatSize(preview.size)}</span>
        )}
        <button
          className={HEAD_BTN}
          onClick={() => app.RevealWorkspacePath(relPath).catch(() => {})}
          title="在文件管理器中定位"
          aria-label="在文件管理器中定位"
        >
          <FolderTree size={10} />
        </button>
        <button
          className={HEAD_BTN}
          onClick={() => app.OpenWorkspacePath(relPath).catch(() => {})}
          title="在外部程序中打开"
          aria-label="在外部程序中打开"
        >
          <ExternalLink size={10} />
        </button>
        {editable && (
          <button
            className={HEAD_BTN + " text-accent"}
            onClick={startEdit}
            title="在预览中直接编辑文本文件（Ctrl+S 保存）"
          >
            <Pencil size={10} />
            编辑
          </button>
        )}
        {editing && (
          <>
            <button
              className={HEAD_BTN}
              onClick={() => void cancelEdit()}
              title="取消编辑"
            >
              <X size={10} />
              取消
            </button>
            <button
              className="flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-accent text-white text-[10px] cursor-pointer hover:opacity-90 disabled:opacity-50"
              onClick={() => void save()}
              disabled={!dirty || saveState === "saving"}
              title="保存（Ctrl+S）"
            >
              {saveState === "saving" ? <Loader2 size={10} className="animate-spin" /> : <Check size={10} />}
              {saveState === "saving" ? "保存中" : saveState === "failed" ? "重试" : "保存"}
            </button>
          </>
        )}
        <button
          className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
          onClick={onClose}
          title="关闭预览"
        >
          <X size={13} />
        </button>
      </div>

      {/* 预览内容 / 编辑区 */}
      <div className="flex-1 overflow-auto">
        {editing ? (
          <>
            {confirmDiscard && (
              <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft bg-amber-500/10 text-[11px] text-amber-500 shrink-0">
                <span className="flex-1">有未保存的修改，退出编辑将丢弃。</span>
                <button
                  type="button"
                  className="px-2 py-0.5 rounded border border-amber-500/40 text-amber-500 hover:bg-amber-500/10 cursor-pointer"
                  onClick={discardAndExit}
                >
                  放弃修改
                </button>
                <button
                  type="button"
                  className="px-2 py-0.5 rounded border border-border-soft text-fg-dim hover:bg-bg-soft cursor-pointer"
                  onClick={() => setConfirmDiscard(false)}
                >
                  继续编辑
                </button>
              </div>
            )}
            <textarea
              className="w-full h-full bg-transparent text-fg text-[12px] p-3 font-mono leading-relaxed outline-none resize-none whitespace-pre"
              value={draft}
              onChange={(e) => {
                setDraft(e.target.value);
                setDirty(true);
                setSaveState("idle");
              }}
              spellCheck={false}
              autoFocus
              aria-label="文本编辑"
            />
          </>
        ) : (
          <>
        {loading && (
          <div className="flex flex-col items-center justify-center h-full text-fg-faint text-xs gap-2">
            {ocrProgress ? (
              <>
                <Loader2 size={18} className="animate-spin text-accent" />
                <span>
                  OCR 识别中 {ocrProgress.done}/{ocrProgress.total} 页…
                </span>
              </>
            ) : (
              <>
                <Loader2 size={18} className="animate-spin text-accent" />
                <span>加载中…</span>
              </>
            )}
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
        {!loading && preview?.kind === "pdf" && (
          // v4.28 B2 pptx：kind=pdf = soffice→PDF 的逐页缩略（pages 有值，
          // 纵向铺页 + 大纲卡点页滚动）或整本 dataUrl 回退（pdftoppm 缺失时
          // 交 WebView 内嵌查看器，无页锚点）。大纲卡在右侧叠放（hint/ext 判定）。
          <div className="flex h-full min-h-0">
            <div ref={pptxPagesRef} className="flex-1 min-w-0 overflow-auto px-4 py-3" data-testid="pptx-pages">
              {preview.pages && preview.pages.length > 0 ? (
                preview.pages.map((p) => (
                  <figure key={p.page} data-pptx-page={p.page} className="mb-4">
                    <img
                      src={p.dataUrl}
                      alt={`第 ${p.page} 页`}
                      className="w-full rounded-lg border border-border-soft shadow-sm bg-bg"
                    />
                    <figcaption className="mt-1 text-center text-[10px] text-fg-faint">第 {p.page} 页</figcaption>
                  </figure>
                ))
              ) : preview.dataUrl ? (
                <embed src={preview.dataUrl} type="application/pdf" title="PDF 预览" className="w-full h-full min-h-[60vh]" />
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-fg-faint text-xs gap-2">
                  <AlertCircle size={20} className="text-amber-500/60" />
                  <span>无可渲染的页面内容，可让 AI 调用 summarize_file 获取内容摘要。</span>
                </div>
              )}
            </div>
            {(preview.hint === "outline" || preview.ext === ".pptx") && (
              <PptxOutline relPath={relPath} fileName={fileName} onPageSelect={scrollToPptxPage} />
            )}
          </div>
        )}
        {!loading && preview?.kind === "markdown" && (
          <div className="px-4 py-3 max-w-[860px] mx-auto">
            {preview.truncated && (
              <div className="mb-2 px-3 py-2 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px] leading-relaxed">
                ⚠️ 预览已截断（{preview.totalPages ? `PDF 共 ${preview.totalPages} 页` : "文件过大"}），仅展示前部内容；可让 AI 调用 summarize_file 获取全文摘要。
              </div>
            )}
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
          </>
        )}
      </div>
    </div>
  );
}
