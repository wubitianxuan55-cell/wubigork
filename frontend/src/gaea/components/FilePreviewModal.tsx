import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, ExternalLink, FilePdf, FileText, FolderTree, Loader2, X } from "../icons";
import { app } from "../lib/bridge";
import {
  LAZY_PAGE_ASPECT,
  LAZY_ROOT_MARGIN_PX,
  addForcedPage,
  computeInitialLazyPages,
  expandMountedPages,
  lazySupported,
  shouldRenderLazyPage,
} from "../lib/pageLazy";
import { usePreviewStore } from "../lib/store";
import type { PreviewResult } from "../lib/types";
import { DocxPreview } from "./DocxPreview";
import { Markdown } from "./Markdown";
import { PptxOutline } from "./PptxOutline";
import { usePreviewProgress } from "../hooks/usePreviewProgress";
import { useToast } from "./Toast";
import { XlsxPreview } from "./XlsxPreview";

// 可转 PDF 的扩展名（与后端 GaeaConvertToPdf 的支持范围一致）。
const PDF_CONVERT_RE = /\.(docx|xlsx|pptx|odt|html?|txt|csv|md|markdown)$/i;

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

const KIND_LABEL: Record<PreviewResult["kind"], string> = {
  image: "图片",
  docx: "Word 文档",
  xlsx: "Excel 表格",
  // v4.31 B：弹窗 pdf 分支已补齐逐页预览（收 v4.28 欠账）——pages 逐页
  // 缩略 + PptxOutline 大纲卡 + 页锚点滚动 + 「针对第 N 页修改」composer 插入，
  // 渲染结构与 FilePreview（v4.28 B2）一致；标签保持「演示文稿」。
  pdf: "演示文稿",
  markdown: "文档",
  text: "文本",
  unsupported: "不支持内联预览",
  error: "无法预览",
};

export function FilePreviewModal() {
  const { previewFile, closeFilePreview, openFilePreview } = usePreviewStore();
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);
  const ocrProgress = usePreviewProgress(previewFile);
  const [loadKey, setLoadKey] = useState(0);
  const panelRef = useRef<HTMLDivElement>(null);
  // v4.31 B：弹窗 pdf（.pptx → soffice PDF）逐页预览的页图容器——
  // 大纲卡条目按 data-pptx-page 锚点滚动（与 FilePreview v4.28 B2 同机制）。
  const pptxPagesRef = useRef<HTMLDivElement | null>(null);
  const [exportingPdf, setExportingPdf] = useState(false);
  const toast = useToast();

  // 每次切换文件重新加载
  useEffect(() => {
    setLoadKey((k) => k + 1);
  }, [previewFile]);

  useEffect(() => {
    if (!previewFile) {
      setPreview(null);
      return;
    }
    let live = true;
    setLoading(true);
    setPreview(null);
    app.Preview(previewFile)
      .then((r) => { if (live) setPreview(r); })
      .catch(() => { if (live) setPreview({ path: previewFile, name: previewFile.split("/").pop() ?? previewFile, ext: "", size: 0, kind: "error", body: "", dataUrl: "", error: "读取文件失败" }); })
      .finally(() => { if (live) setLoading(false); });
    return () => { live = false; };
  }, [previewFile, loadKey]);

  // Esc 关闭
  useEffect(() => {
    if (!previewFile) return;
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") closeFilePreview(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [previewFile, closeFilePreview]);

  // 打开时聚焦面板，禁用背景滚动
  useEffect(() => {
    if (!previewFile) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    panelRef.current?.focus();
    return () => { document.body.style.overflow = prevOverflow; };
  }, [previewFile]);

  const openExternal = useCallback(() => {
    if (previewFile) app.OpenWorkspacePath(previewFile).catch(() => {});
  }, [previewFile]);

  const reveal = useCallback(() => {
    if (previewFile) app.RevealWorkspacePath(previewFile).catch(() => {});
  }, [previewFile]);

  // v4.32 C：pdf 逐页懒加载（收 v4.31 欠账「弹窗 pdf 不虚拟化」）。判定逻辑
  // 收敛在 lib/pageLazy.ts（纯函数），这里只做 IO 接线：lazyPdf.src 记录所属
  // preview，preview 换载荷时在渲染期重置（初始窗口 + 清空强制集合，React
  // 「props 变化时调整 state」模式，避免 effect 时序上的旧集合闪烁）；mounted
  // 单向只增不减（已挂载页不卸载，杜绝滚动跳动），forced 收大纲跳转目标页。
  // 无 IntersectionObserver（jsdom/旧环境）→ lazySupported() 为 false →
  // pdf 分支全量渲染 = v4.31 行为。
  const pageElsRef = useRef(new Map<number, HTMLElement>());
  const pdfObserverRef = useRef<IntersectionObserver | null>(null);
  const [lazyPdf, setLazyPdf] = useState<{
    src: PreviewResult | null;
    mounted: ReadonlySet<number>;
    forced: ReadonlySet<number>;
  }>(() => ({ src: null, mounted: new Set<number>(), forced: new Set<number>() }));
  const pdfPages = preview?.kind === "pdf" ? preview.pages : undefined;
  const pdfPageCount = pdfPages?.length ?? 0;
  const lazyPdfPages = pdfPageCount > 0 && lazySupported();
  if (lazyPdf.src !== preview) {
    setLazyPdf({
      src: preview,
      mounted: pdfPageCount > 0 ? computeInitialLazyPages(pdfPageCount) : new Set<number>(),
      forced: new Set<number>(),
    });
  }
  // 页图容器注册表：figure 挂载即登记（data-pptx-page 既是大纲滚动锚点也是
  // IO 目标键），卸载时清掉已断连的条目。figure 可能晚于 IO effect 挂载
  // （preview 与 loading 在 .then/.finally 分两次提交，!loading 门控后到），
  // 登记时若观察器已存在则立即补 observe，保证任意挂载顺序都进观察集。
  const pdfPageAnchorRef = useCallback((el: HTMLElement | null) => {
    const els = pageElsRef.current;
    if (el) {
      const page = Number(el.getAttribute("data-pptx-page"));
      if (page > 0) {
        els.set(page, el);
        pdfObserverRef.current?.observe(el);
      }
      return;
    }
    for (const [page, node] of els) {
      if (!node.isConnected) {
        pdfObserverRef.current?.unobserve(node);
        els.delete(page);
      }
    }
  }, []);

  // IntersectionObserver 接线：页容器进入视口（rootMargin 800px，root 为弹窗
  // 内滚动容器）→ 并入挂载集合挂真身 <img>。IO 缺失时本 effect 直接跳过。
  useEffect(() => {
    if (!lazyPdfPages || typeof IntersectionObserver === "undefined") return;
    const io = new IntersectionObserver(
      (entries) => {
        const visible: number[] = [];
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          const page = Number((entry.target as HTMLElement).getAttribute("data-pptx-page"));
          if (Number.isFinite(page) && page > 0) visible.push(page);
        }
        if (visible.length === 0) return;
        setLazyPdf((prev) => ({
          ...prev,
          mounted: expandMountedPages(prev.mounted, visible, pdfPageCount),
        }));
      },
      { root: pptxPagesRef.current, rootMargin: `${LAZY_ROOT_MARGIN_PX}px` },
    );
    pdfObserverRef.current = io;
    for (const el of pageElsRef.current.values()) io.observe(el);
    return () => {
      io.disconnect();
      pdfObserverRef.current = null;
    };
  }, [lazyPdfPages, pdfPageCount]);

  // v4.31 B：点大纲页条目 → 滚动到逐页渲染区的对应页锚点
  //（jsdom 无 scrollIntoView，可选调用守卫；测试里注入 spy 验证。）
  const scrollToPptxPage = useCallback((page: number) => {
    // v4.32 C：懒加载下编程式跳转的目标页强制渲染真身——占位高度是估计值，
    // 目标页若还停留在占位盒，scrollIntoView 会跳偏；先并入强制渲染集合再滚
    //（滚动容器 scrollTop 变化后 IO 也会自然补挂邻页）。
    setLazyPdf((prev) => ({ ...prev, forced: addForcedPage(prev.forced, page) }));
    const el = pptxPagesRef.current?.querySelector<HTMLElement>(`[data-pptx-page="${page}"]`);
    el?.scrollIntoView?.({ block: "start", behavior: "smooth" });
  }, []);

  // 导出 PDF：LibreOffice 无头转换，产物入 .gaea/exports/ 并直接打开预览
  const exportPdf = useCallback(async () => {
    if (!previewFile || exportingPdf) return;
    setExportingPdf(true);
    try {
      const r = await app.ConvertToPdf(previewFile);
      toast.show(`已导出 ${r.name}`, "info");
      openFilePreview(r.path);
    } catch (e) {
      toast.show(e instanceof Error ? e.message : String(e), "warn");
    } finally {
      setExportingPdf(false);
    }
  }, [previewFile, exportingPdf, toast, openFilePreview]);

  if (!previewFile) return null;

  const name = preview?.name ?? previewFile.split("/").pop() ?? previewFile;
  const kind = preview?.kind ?? "text";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 md:p-8"
      style={{ background: "rgba(0,0,0,0.45)", backdropFilter: "blur(3px)" }}
      onClick={closeFilePreview}
      role="dialog"
      aria-modal="true"
      aria-label={`预览 ${name}`}
    >
      <div
        ref={panelRef}
        tabIndex={-1}
        className="flex flex-col w-full max-w-[1000px] h-[86vh] max-h-[86vh] rounded-xl border border-border bg-bg-elev-2 shadow-[0_24px_64px_rgba(0,0,0,0.35)] overflow-hidden outline-none"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="flex items-center gap-3 px-4 py-2.5 border-b border-border-soft shrink-0 bg-bg">
          <div className="w-8 h-8 rounded-lg bg-accent/10 flex items-center justify-center text-accent shrink-0">
            <FileText size={16} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="font-medium text-[14px] text-fg truncate leading-tight">{name}</div>
            <div className="flex items-center gap-2 text-[11px] text-fg-faint leading-tight">
              <span className="font-mono truncate">{previewFile}</span>
              {preview && (
                <>
                  <span>·</span>
                  <span className="uppercase">{preview.ext.replace(".", "")}</span>
                  <span>·</span>
                  <span>{formatSize(preview.size)}</span>
                </>
              )}
            </div>
          </div>
          {preview && kind !== "error" && (
            <span className={`shrink-0 px-2 py-0.5 rounded-full text-[10.5px] border ${kind === "unsupported" ? "border-amber-500/30 text-amber-500 bg-amber-500/5" : "border-accent/25 text-accent bg-accent/5"}`}>
              {KIND_LABEL[kind]}
            </span>
          )}
          <div className="flex items-center gap-1 shrink-0">
            {PDF_CONVERT_RE.test(previewFile) && (
              <button
                className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-accent/30 bg-accent/8 text-accent text-[12px] cursor-pointer hover:bg-accent/15 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                onClick={() => void exportPdf()}
                disabled={exportingPdf}
                title="用 LibreOffice 把当前文档转换为 PDF（产物在 .gaea/exports/）"
              >
                {exportingPdf ? <Loader2 size={12} className="animate-spin" /> : <FilePdf size={12} />}
                <span className="hidden md:inline">导出 PDF</span>
              </button>
            )}
            <button
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-border-soft bg-transparent text-fg-dim text-[12px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
              onClick={reveal}
              title="在文件管理器中定位"
            >
              <FolderTree size={12} />
              <span className="hidden md:inline">定位</span>
            </button>
            <button
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-border-soft bg-transparent text-fg-dim text-[12px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
              onClick={openExternal}
              title="在外部程序中打开"
            >
              <ExternalLink size={12} />
              <span className="hidden md:inline">打开</span>
            </button>
            <button
              className="flex items-center justify-center w-7 h-7 rounded-lg border-0 bg-transparent text-fg-faint cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
              onClick={closeFilePreview}
              title="关闭 (Esc)"
            >
              <X size={15} />
            </button>
          </div>
        </div>

        {/* 内容 */}
        <div className="flex-1 min-h-0 overflow-auto bg-bg">
          {loading && (
            <div className="flex flex-col items-center justify-center h-full gap-3 text-fg-faint text-[13px]">
              {ocrProgress ? (
                <>
                  <Loader2 size={26} className="animate-spin text-accent" />
                  <span>OCR 识别中 {ocrProgress.done}/{ocrProgress.total} 页…</span>
                </>
              ) : (
                <>
                  <Loader2 size={26} className="animate-spin text-accent" />
                  <span>正在加载预览…</span>
                </>
              )}
            </div>
          )}

          {!loading && preview?.kind === "image" && preview.dataUrl && (
            <div className="flex items-center justify-center min-h-full p-6 bg-[#0d0f12]/60">
              <img
                src={preview.dataUrl}
                alt={name}
                className="max-w-full max-h-[calc(86vh-60px)] object-contain rounded-lg shadow-2xl"
              />
            </div>
          )}

          {!loading && preview?.kind === "docx" && preview.dataUrl && (
            <DocxPreview dataUrl={preview.dataUrl} fileName={name} relPath={previewFile} />
          )}

          {!loading && preview?.kind === "xlsx" && (
            <XlsxPreview body={preview.body} fileName={name} relPath={previewFile} />
          )}

          {!loading && preview?.kind === "pdf" && (
            // v4.31 B：弹窗补齐 pdf 逐页预览（收 v4.28 欠账），渲染结构与
            // FilePreview kind=pdf 分支一致：pages 有值纵向铺页（data-pptx-page
            // 锚点供大纲卡滚动），否则回退 dataUrl 整本内嵌；右侧叠 PptxOutline
            // 大纲卡（「针对第 N 页修改」composer 插入由该组件内部完成）。
            // v4.32 C：页数上限沿用后端语义（GaeaPreview 已截断 ≤60 页）；
            // 逐页路径改单向懒加载——初始窗口外的页先渲染估计高度占位盒
            //（PreviewPageThumb 无宽高），进视口（rootMargin 800px）才挂
            // <img> 自然撑开，已挂载页不卸载；无 IO 环境全量渲染（= v4.31）。
            <div className="flex h-full min-h-0">
              <div ref={pptxPagesRef} className="flex-1 min-w-0 overflow-auto px-4 py-3" data-testid="pptx-pages">
                {preview.pages && preview.pages.length > 0 ? (
                  preview.pages.map((p) => (
                    <figure key={p.page} ref={pdfPageAnchorRef} data-pptx-page={p.page} className="mb-4">
                      {!lazyPdfPages || shouldRenderLazyPage(p.page, lazyPdf.mounted, lazyPdf.forced) ? (
                        <img
                          src={p.dataUrl}
                          alt={`第 ${p.page} 页`}
                          className="w-full rounded-lg border border-border-soft shadow-sm bg-bg"
                        />
                      ) : (
                        <div
                          aria-hidden="true"
                          className="w-full rounded-lg border border-border-soft bg-gradient-to-b from-bg-soft to-bg animate-pulse"
                          style={{ aspectRatio: LAZY_PAGE_ASPECT }}
                        />
                      )}
                      <figcaption className="mt-1 text-center text-[10px] text-fg-faint">第 {p.page} 页</figcaption>
                    </figure>
                  ))
                ) : preview.dataUrl ? (
                  <embed src={preview.dataUrl} type="application/pdf" title="PDF 预览" className="w-full h-full min-h-[60vh]" />
                ) : (
                  <div className="flex flex-col items-center justify-center h-full gap-3 text-fg-faint text-[13px]">
                    <AlertCircle size={26} className="text-amber-500/60" />
                    <span>无可渲染的页面内容，可让 AI 调用 summarize_file 获取内容摘要。</span>
                  </div>
                )}
              </div>
              {(preview.hint === "outline" || preview.ext === ".pptx") && (
                <PptxOutline relPath={previewFile} fileName={name} onPageSelect={scrollToPptxPage} />
              )}
            </div>
          )}

          {!loading && preview?.kind === "markdown" && (
            <div className="px-8 py-6 max-w-[860px] mx-auto">
              {preview.truncated && (
                <div className="mb-3 px-3 py-2 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[12px] leading-relaxed">
                  ⚠️ 预览已截断（{preview.totalPages ? `PDF 共 ${preview.totalPages} 页` : "文件过大"}），仅展示前部内容；可让 AI 调用 summarize_file 获取全文摘要。
                </div>
              )}
              <Markdown text={preview.body} autoExportMermaid={false} />
            </div>
          )}

          {!loading && preview?.kind === "text" && (
            <pre className="p-5 text-[13px] text-fg-dim font-mono leading-relaxed whitespace-pre-wrap overflow-x-auto">
              {preview.body}
            </pre>
          )}

          {!loading && (preview?.kind === "unsupported" || preview?.kind === "error") && (
            <div className="flex flex-col items-center justify-center h-full gap-3 px-8 text-center">
              <AlertCircle size={34} className={preview.kind === "error" ? "text-err/70" : "text-amber-500/70"} />
              <div className="text-[14px] text-fg-dim">{preview.error || "该文件无法预览"}</div>
              <button
                className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-accent text-bg text-[13px] font-medium cursor-pointer hover:opacity-90 transition-opacity"
                onClick={openExternal}
              >
                <ExternalLink size={13} />
                在外部程序中打开
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
