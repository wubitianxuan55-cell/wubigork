import { useCallback, useEffect, useRef, useState } from "react";
import { renderAsync } from "docx-preview";
import {
  AlertCircle,
  Check,
  FileText,
  ListTree,
  Loader2,
  MessageSquare,
  Sparkles,
  Wand2,
  X,
} from "../icons";
import { app } from "../lib/bridge";
import { useComposerInsertStore, useUpdatedFilesStore } from "../lib/store";
import { useToast } from "./Toast";
import { extractDocxParagraphs } from "../lib/docxText";
import { extractDocxOutline, linkDocxOutlineAnchors } from "../lib/docxOutline";
import type { DocxOutlineItem } from "../lib/docxOutline";
import { DocxOutline } from "./DocxOutline";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

function dataUrlToBlob(dataUrl: string): Blob {
  const comma = dataUrl.indexOf(",");
  const b64 = comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new Blob([bytes], { type: DOCX_MIME });
}

const PRESETS = [
  { label: "润色", instruction: "润色这段文字，使表达更准确流畅，保持原意不变" },
  { label: "精简", instruction: "精简这段文字，去掉冗余表达，保留全部关键信息" },
  { label: "翻译成中文", instruction: "将这段文字翻译成规范的中文" },
  { label: "扩写", instruction: "扩写这段文字，补充必要的细节，保持原意和严谨性" },
];

/**
 * DocxPreview 用 docx-preview 在浏览器内保真渲染 .docx（版式/表格/页眉页脚/
 * 修订与批注均保留）。支持「框选即改」：选中文字 → 指令 → AI 生成替换 →
 * 以 Word 修订模式（w:del + w:ins）就地写入并重渲染。
 *
 * B3 选区联动：框选工具栏内置次级「引用到对话」入口（与框选即改同选区，
 * 走既有 composer 插入通道 requestText，不抢占 AI 编辑流程）。
 * 渲染失败降级：docx-preview 抛异常时不再落死错误页，尽力提取正文段落文本
 * （docxText.ts，复用包内 jszip），降级为带提示条的纯文本视图。
 */
export function DocxPreview({
  dataUrl,
  fileName,
  relPath,
  onQuoteSelection,
}: {
  dataUrl: string;
  fileName: string;
  relPath: string;
  /** B3 选区联动：选中文本引用到对话的自定义出口；缺省走 composer 插入通道。 */
  onQuoteSelection?: (quote: string) => void;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [docDataUrl, setDocDataUrl] = useState(dataUrl);
  const markUpdated = useUpdatedFilesStore((s) => s.markUpdated);
  const toast = useToast();
  const [status, setStatus] = useState<"loading" | "done" | "error">("loading");
  const [error, setError] = useState("");
  // 渲染失败降级：正文段落文本（提取成功才有）；extractError 为提取自身的失败信息。
  const [fallbackParas, setFallbackParas] = useState<string[] | null>(null);
  const [extractError, setExtractError] = useState("");

  // 框选即改状态
  const [selected, setSelected] = useState<string | null>(null);
  const [instruction, setInstruction] = useState("");
  const [generating, setGenerating] = useState(false);
  const [proposal, setProposal] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const [actionError, setActionError] = useState("");
  const [notice, setNotice] = useState("");
  const [hasRevisions, setHasRevisions] = useState(false);
  const [flattening, setFlattening] = useState(false);
  // Word 目录（大纲导航）：标题结构来自 docxOutline.ts 纯解析，锚点链接在
  // 渲染完成后进行；目录解析失败/文档无标题结构都诚实降级，不影响版式预览。
  const [outlineItems, setOutlineItems] = useState<DocxOutlineItem[]>([]);
  const [outlineError, setOutlineError] = useState("");
  const [outlineOpen, setOutlineOpen] = useState(false);
  const outlineAnchorsRef = useRef<(HTMLElement | null)[]>([]);

  useEffect(() => {
    setDocDataUrl(dataUrl);
    setSelected(null);
    setProposal(null);
    setInstruction("");
    setActionError("");
    setHasRevisions(false);
    setFallbackParas(null);
    setExtractError("");
    outlineAnchorsRef.current = [];
    setOutlineItems([]);
    setOutlineError("");
    setOutlineOpen(false);
  }, [dataUrl]);

  useEffect(() => {
    let live = true;
    setStatus("loading");
    setError("");
    setFallbackParas(null);
    setExtractError("");
    outlineAnchorsRef.current = [];
    setOutlineItems([]);
    setOutlineError("");
    setOutlineOpen(false);

    (async () => {
      try {
        const blob = dataUrlToBlob(docDataUrl);
        if (!live) return;
        const container = containerRef.current;
        if (!container) return;
        await renderAsync(blob, container, undefined, {
          inWrapper: true,
          breakPages: true,
          renderHeaders: true,
          renderFooters: true,
          renderFootnotes: true,
          renderEndnotes: true,
          renderChanges: true,
          renderComments: true,
        });
        if (!live) return;
        // 检测文档中是否存在修订标记（ins/del），显示接受/拒绝入口
        const revCount = container.querySelectorAll("ins, del").length;
        setHasRevisions(revCount > 0);
        setStatus("done");
        // 版式渲染成功后解析标题结构并链接锚点（目录侧栏用）。解析失败只
        // 影响「目录」能力本身，绝不回滚已成功的版式预览。
        try {
          const items = await extractDocxOutline(docDataUrl);
          if (!live) return;
          outlineAnchorsRef.current = linkDocxOutlineAnchors(container, items);
          if (!live) return;
          setOutlineItems(items);
          setOutlineError("");
        } catch (e2) {
          if (!live) return;
          outlineAnchorsRef.current = [];
          setOutlineItems([]);
          setOutlineError(e2 instanceof Error ? e2.message : String(e2));
        }
      } catch (e) {
        if (!live) return;
        setError(e instanceof Error ? e.message : String(e));
        // 渲染失败 → 降级：尽力从 docx 包提取正文段落文本；提取也失败才落
        // 死错误页（两个错误如实展示），绝不静默吞掉失败原因。
        try {
          const paras = await extractDocxParagraphs(docDataUrl);
          if (!live) return;
          setFallbackParas(paras);
        } catch (e2) {
          if (!live) return;
          setExtractError(e2 instanceof Error ? e2.message : String(e2));
        }
        setStatus("error");
      }
    })();

    return () => {
      live = false;
    };
  }, [docDataUrl]);

  // Esc 关闭编辑工具栏
  useEffect(() => {
    if (!selected && !proposal) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setSelected(null);
        setProposal(null);
        setInstruction("");
        setActionError("");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [selected, proposal]);

  // 选中文本 → 弹出编辑工具栏
  const handleMouseUp = useCallback(() => {
    const sel = window.getSelection();
    const text = sel?.toString().trim();
    if (!text || !rootRef.current?.contains(sel?.anchorNode ?? null)) return;
    if (sel && sel.rangeCount > 0) {
      const range = sel.getRangeAt(0);
      if (!rootRef.current.contains(range.commonAncestorContainer)) return;
    }
    setSelected(text);
    setProposal(null);
    setInstruction("");
    setActionError("");
  }, []);

  const runGenerate = useCallback(async () => {
    if (!selected || !instruction.trim()) return;
    setGenerating(true);
    setActionError("");
    setProposal(null);
    try {
      const r = await app.OfficeEditText(selected, instruction.trim());
      if (!r?.edited) {
        setActionError("AI 未返回有效结果，请重试");
      } else {
        setProposal(r.edited);
      }
    } catch (e) {
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setGenerating(false);
    }
  }, [selected, instruction]);

  const applyProposal = useCallback(async () => {
    if (!selected || !proposal) return;
    setApplying(true);
    setActionError("");
    try {
      const r = await app.DocxApplyEdit(relPath, selected, proposal);
      setDocDataUrl(r.dataUrl);
      markUpdated(relPath);
      setSelected(null);
      setProposal(null);
      setInstruction("");
      setNotice("已应用修订：改动以修订样式呈现，可在 Word 中逐条接受/拒绝");
      window.setTimeout(() => setNotice(""), 5000);
    } catch (e) {
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  }, [selected, proposal, relPath, markUpdated]);

  const closeToolbar = useCallback(() => {
    setSelected(null);
    setProposal(null);
    setInstruction("");
    setActionError("");
  }, []);

  // B3 次级入口：把框选文本以引用块插入输入框（与 SelectionToComposer 同一
  // 引用格式与 composer 插入通道），插完收起工具栏，不打断框选即改主流程。
  const quoteToComposer = useCallback(() => {
    if (!selected) return;
    const quoted = selected
      .split("\n")
      .map((l) => `> ${l}`)
      .join("\n");
    const text = `${quoted}\n\n请基于以上内容继续处理。`;
    if (onQuoteSelection) onQuoteSelection(text);
    else useComposerInsertStore.getState().requestText(text);
    toast.show("已引用到输入框", "info");
    closeToolbar();
  }, [selected, onQuoteSelection, toast, closeToolbar]);

  const flattenRevisions = useCallback(
    async (accept: boolean) => {
      setFlattening(true);
      setActionError("");
      try {
        const r = await app.DocxAcceptChanges(relPath, accept);
        setDocDataUrl(r.dataUrl);
        markUpdated(relPath);
        setSelected(null);
        setProposal(null);
        setHasRevisions(false);
        setNotice(accept ? "已接受全部修订" : "已拒绝全部修订");
        window.setTimeout(() => setNotice(""), 4000);
      } catch (e) {
        setActionError(e instanceof Error ? e.message : String(e));
      } finally {
        setFlattening(false);
      }
    },
    [relPath, markUpdated],
  );

  const toggleOutline = useCallback(() => {
    if (outlineError) {
      toast.show(`目录不可用（${outlineError}），仍可全文预览`, "info");
      return;
    }
    setOutlineOpen((o) => !o);
  }, [outlineError, toast]);

  // 目录条目 → 版式定位：滚动到锚点段落并短暂高亮（jsdom 等无
  // scrollIntoView 的环境静默跳过滚动，仅高亮）。锚点缺失时诚实提示。
  const scrollToOutlineItem = useCallback(
    (index: number) => {
      const el = outlineAnchorsRef.current[index];
      if (!el) {
        toast.show("该目录条目在版式中未找到可定位锚点（可能位于页眉/文本框内）", "info");
        return;
      }
      if (typeof el.scrollIntoView === "function") {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }
      el.classList.add("docx-outline-flash");
      window.setTimeout(() => el.classList.remove("docx-outline-flash"), 1400);
    },
    [toast],
  );

  // 目录条目 → 「定位章节修改」：只往 composer 插入模板（光标接在冒号后让
  // 用户补全具体要求），不自动发送——与 PptxOutline 修改指令同一纪律。
  const insertOutlineModify = useCallback(
    (item: DocxOutlineItem) => {
      useComposerInsertStore
        .getState()
        .requestText(`请修改 ${fileName} 中「${item.text}」这一节：`);
      toast.show(`已插入「${item.text}」的修改指令，补全要求后发送`, "info");
    },
    [fileName, toast],
  );

  return (
    <div
      ref={rootRef}
      className="docx-preview-root relative h-full flex flex-col overflow-hidden"
      onMouseUp={handleMouseUp}
    >
      {status === "loading" && (
        <div className="flex flex-col items-center justify-center flex-1 min-h-0 gap-3 text-fg-faint text-[13px]">
          <Loader2 size={22} className="animate-spin text-accent" />
          <span>正在渲染 Word 版式…</span>
        </div>
      )}
      {status === "error" && fallbackParas === null && (
        <div className="flex flex-col items-center justify-center flex-1 min-h-0 gap-3 px-8 text-center">
          <AlertCircle size={30} className="text-err/70" />
          <div className="text-[13px] text-fg-dim max-w-[420px] break-all">
            该 Word 文档渲染失败：{error}
          </div>
          {extractError && (
            <div className="text-[11px] text-fg-faint max-w-[420px] break-all">
              纯文本降级也失败：{extractError}
            </div>
          )}
        </div>
      )}
      {status === "error" && fallbackParas !== null && (
        <div className="flex flex-col flex-1 min-h-0" data-testid="docx-fallback">
          {/* 降级提示条：诚实说明降级原因与能力边界 */}
          <div className="flex items-start gap-1.5 px-3 py-2 border-b border-amber-500/30 bg-amber-500/10 text-amber-500 text-[11px] leading-relaxed shrink-0">
            <AlertCircle size={12} className="mt-px shrink-0" />
            <span>
              Word 版式渲染失败，已降级为纯文本视图（{fallbackParas.filter(Boolean).length} 个非空段落）。
              文本由文档正文直接提取，不含图片/文本框与版式信息。
            </span>
          </div>
          <div className="flex-1 min-h-0 overflow-auto px-4 py-3">
            {fallbackParas.map((t, i) =>
              t ? (
                <p key={i} className="text-[12.5px] leading-relaxed text-fg-dim mb-2">
                  {t}
                </p>
              ) : (
                <div key={i} className="h-2" />
              ),
            )}
          </div>
        </div>
      )}
      {status === "done" && (
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft bg-bg-soft/40 text-fg-faint text-[11px] shrink-0">
          <FileText size={11} />
          <span className="truncate">{fileName}</span>
          <span>·</span>
          <span>版式保真预览</span>
          <span className="ml-auto inline-flex items-center gap-1.5 shrink-0">
            <button
              data-testid="docx-outline-toggle"
              className={
                "inline-flex items-center gap-1 px-2 py-0.5 rounded-md border text-[10px] cursor-pointer disabled:opacity-50 transition-colors " +
                (outlineOpen
                  ? "border-accent/30 bg-accent/10 text-accent hover:bg-accent/20"
                  : "border-border-soft bg-transparent text-fg-dim hover:bg-bg-soft hover:text-fg")
              }
              onClick={toggleOutline}
              title={
                outlineError
                  ? `目录不可用：${outlineError}`
                  : outlineOpen
                    ? "收起目录侧栏"
                    : "打开 Word 目录（标题大纲，点击条目定位到章节）"
              }
            >
              <ListTree size={10} />
              目录
              {!outlineError && outlineItems.length > 0 && (
                <span className="opacity-70">{outlineItems.length}</span>
              )}
            </button>
            {hasRevisions && !selected && (
              <>
              <button
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-accent/30 bg-accent/10 text-accent text-[10px] cursor-pointer hover:bg-accent/20 disabled:opacity-50 transition-colors"
                onClick={() => void flattenRevisions(true)}
                disabled={flattening}
                title="接受 gaea 的全部修订（新文生效）"
              >
                {flattening ? <Loader2 size={10} className="animate-spin" /> : <Check size={10} />}
                接受修订
              </button>
              <button
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft disabled:opacity-50 transition-colors"
                onClick={() => void flattenRevisions(false)}
                disabled={flattening}
                title="拒绝 gaea 的全部修订（恢复原文）"
              >
                <X size={10} />
                拒绝修订
              </button>
              </>
            )}
            {!hasRevisions && (
              <span className="inline-flex items-center gap-1 text-accent/80">
                <Wand2 size={10} />
                选中文字即可 AI 编辑
              </span>
            )}
          </span>
        </div>
      )}
      {/* 渲染容器常驻（renderAsync 的 ref 目标，重渲染依赖它）；目录侧栏与
          版式同排（flex 行），加载/降级态整行隐藏让位给状态块 */}
      <div className={status === "done" ? "flex flex-1 min-h-0 min-w-0" : "hidden"}>
        <div
          ref={containerRef}
          className="overflow-auto flex-1 min-w-0 docx-preview-body"
          style={status === "error" && fallbackParas !== null ? { display: "none" } : undefined}
        />
        {status === "done" && outlineOpen && (
          <DocxOutline
            items={outlineItems}
            error={outlineError || undefined}
            onNavigate={scrollToOutlineItem}
            onInsertModify={insertOutlineModify}
            onClose={() => setOutlineOpen(false)}
          />
        )}
      </div>

      {notice && (
        <div className="absolute top-9 left-1/2 -translate-x-1/2 z-30 flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-accent/30 bg-bg-elev-2 text-accent text-[12px] shadow-lg">
          <Check size={12} />
          {notice}
        </div>
      )}

      {selected && (
        <div className="absolute bottom-3 left-1/2 -translate-x-1/2 z-20 w-[min(760px,94%)] rounded-xl border border-border bg-bg-elev-2 shadow-[0_16px_48px_rgba(0,0,0,0.4)] overflow-hidden">
          {!proposal ? (
            <>
              <div className="flex items-center gap-2 px-3 py-2 border-b border-border-soft">
                <Sparkles size={13} className="text-accent shrink-0" />
                <span className="text-[12px] text-fg font-medium flex-1 truncate">
                  AI 编辑选中内容
                </span>
                <button
                  className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
                  onClick={closeToolbar}
                  title="关闭 (Esc)"
                >
                  <X size={12} />
                </button>
              </div>
              <div className="px-3 py-2">
                <div className="max-h-14 overflow-auto rounded-md bg-bg-soft/50 px-2 py-1.5 text-[11px] text-fg-dim leading-relaxed border border-border-soft">
                  {selected.length > 220 ? `${selected.slice(0, 220)}…` : selected}
                  <span className="text-fg-faint">（{selected.length} 字）</span>
                </div>
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {PRESETS.map((p) => (
                    <button
                      key={p.label}
                      className="px-2 py-1 rounded-md border border-border-soft bg-transparent text-[11px] text-fg-dim cursor-pointer hover:bg-accent/10 hover:text-accent hover:border-accent/30 transition-colors"
                      onClick={() => setInstruction(p.instruction)}
                    >
                      {p.label}
                    </button>
                  ))}
                </div>
                <div className="flex gap-2 mt-2">
                  <input
                    value={instruction}
                    onChange={(e) => setInstruction(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        runGenerate();
                      }
                    }}
                    placeholder="输入指令，如：改成更正式的表达 / 翻译成英文…"
                    className="flex-1 min-w-0 px-2.5 py-1.5 rounded-lg border border-border-soft bg-bg text-[12px] text-fg outline-none focus:border-accent/50"
                  />
                  <button
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent text-bg text-[12px] font-medium cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity"
                    disabled={!instruction.trim() || generating}
                    onClick={runGenerate}
                  >
                    {generating ? <Loader2 size={12} className="animate-spin" /> : <Wand2 size={12} />}
                    生成
                  </button>
                  {/* B3 次级入口：引用到对话（与 AI 编辑同选区，互不抢占） */}
                  <button
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg border border-border-soft bg-transparent text-[12px] text-fg-dim cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors shrink-0"
                    onClick={quoteToComposer}
                    title="把选中文本以引用块插入输入框，可编辑后发送"
                    data-testid="docx-quote-btn"
                  >
                    <MessageSquare size={12} aria-hidden />
                    引用到对话
                  </button>
                </div>
                {actionError && (
                  <div className="mt-2 text-[11px] text-err">{actionError}</div>
                )}
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center gap-2 px-3 py-2 border-b border-border-soft">
                <Sparkles size={13} className="text-accent shrink-0" />
                <span className="text-[12px] text-fg font-medium flex-1">
                  修订预览（原 → 新）
                </span>
                <button
                  className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded"
                  onClick={closeToolbar}
                  title="放弃 (Esc)"
                >
                  <X size={12} />
                </button>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2 px-3 py-2">
                <div className="min-w-0">
                  <div className="text-[10px] text-fg-faint mb-1">原文</div>
                  <div className="max-h-28 overflow-auto rounded-md bg-del-bg/25 border border-border-soft px-2 py-1.5 text-[11px] text-fg-dim leading-relaxed whitespace-pre-wrap">
                    {selected}
                  </div>
                </div>
                <div className="min-w-0">
                  <div className="text-[10px] text-fg-faint mb-1">AI 替换</div>
                  <div className="max-h-28 overflow-auto rounded-md bg-accent/5 border border-accent/25 px-2 py-1.5 text-[11px] text-fg leading-relaxed whitespace-pre-wrap">
                    {proposal}
                  </div>
                </div>
              </div>
              {actionError && (
                <div className="px-3 pb-1 text-[11px] text-err">{actionError}</div>
              )}
              <div className="flex items-center justify-end gap-2 px-3 py-2 border-t border-border-soft">
                <button
                  className="px-2.5 py-1.5 rounded-lg border border-border-soft bg-transparent text-[12px] text-fg-dim cursor-pointer hover:bg-bg-soft transition-colors"
                  onClick={() => {
                    setProposal(null);
                    setInstruction("");
                  }}
                  disabled={applying}
                >
                  重新生成
                </button>
                <button
                  className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg border border-border-soft bg-transparent text-[12px] text-fg-dim cursor-pointer hover:bg-bg-soft transition-colors"
                  onClick={closeToolbar}
                  disabled={applying}
                >
                  放弃
                </button>
                <button
                  className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent text-bg text-[12px] font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed hover:opacity-90 transition-opacity"
                  onClick={applyProposal}
                  disabled={applying}
                >
                  {applying ? <Loader2 size={12} className="animate-spin" /> : <Check size={12} />}
                  应用修订
                </button>
              </div>
            </>
          )}
        </div>
      )}

      <style>{`
        .docx-preview-root .docx-wrapper {
          background: transparent;
          padding: 16px;
        }
        .docx-preview-root .docx-wrapper section.docx {
          box-shadow: 0 8px 28px rgba(0,0,0,0.28);
          margin-bottom: 24px;
        }
        .docx-preview-body {
          background:
            radial-gradient(circle at 50% 0%, rgba(120,130,150,0.10), transparent 55%),
            var(--bg, #101318);
        }
        .docx-preview-root ::selection {
          background: rgba(92,140,255,0.32);
        }
        .docx-preview-root p.docx-outline-flash {
          outline: 2px solid rgba(92,140,255,0.72);
          outline-offset: 2px;
          background: rgba(92,140,255,0.10);
          border-radius: 3px;
        }
      `}</style>
    </div>
  );
}
