import { useCallback, useEffect, useRef, useState } from "react";
import { renderAsync } from "docx-preview";
import { AlertCircle, Check, FileText, Loader2, Sparkles, Wand2, X } from "../icons";
import { app } from "../lib/bridge";

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
 */
export function DocxPreview({
  dataUrl,
  fileName,
  relPath,
}: {
  dataUrl: string;
  fileName: string;
  relPath: string;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [docDataUrl, setDocDataUrl] = useState(dataUrl);
  const [status, setStatus] = useState<"loading" | "done" | "error">("loading");
  const [error, setError] = useState("");

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

  useEffect(() => {
    setDocDataUrl(dataUrl);
    setSelected(null);
    setProposal(null);
    setInstruction("");
    setActionError("");
    setHasRevisions(false);
  }, [dataUrl]);

  useEffect(() => {
    let live = true;
    setStatus("loading");
    setError("");

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
      } catch (e) {
        if (!live) return;
        setError(e instanceof Error ? e.message : String(e));
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
  }, [selected, proposal, relPath]);

  const closeToolbar = useCallback(() => {
    setSelected(null);
    setProposal(null);
    setInstruction("");
    setActionError("");
  }, []);

  const flattenRevisions = useCallback(
    async (accept: boolean) => {
      setFlattening(true);
      setActionError("");
      try {
        const r = await app.DocxAcceptChanges(relPath, accept);
        setDocDataUrl(r.dataUrl);
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
    [relPath],
  );

  return (
    <div ref={rootRef} className="docx-preview-root relative h-full" onMouseUp={handleMouseUp}>
      {status === "loading" && (
        <div className="flex flex-col items-center justify-center h-full gap-3 text-fg-faint text-[13px]">
          <Loader2 size={22} className="animate-spin text-accent" />
          <span>正在渲染 Word 版式…</span>
        </div>
      )}
      {status === "error" && (
        <div className="flex flex-col items-center justify-center h-full gap-3 px-8 text-center">
          <AlertCircle size={30} className="text-err/70" />
          <div className="text-[13px] text-fg-dim max-w-[420px] break-all">
            该 Word 文档渲染失败：{error}
          </div>
        </div>
      )}
      {status === "done" && (
        <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft bg-bg-soft/40 text-fg-faint text-[11px] shrink-0">
          <FileText size={11} />
          <span className="truncate">{fileName}</span>
          <span>·</span>
          <span>版式保真预览</span>
          {hasRevisions && !selected && (
            <span className="ml-auto inline-flex items-center gap-1.5 shrink-0">
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
            </span>
          )}
          {!hasRevisions && (
            <span className="ml-auto inline-flex items-center gap-1 text-accent/80">
            <Wand2 size={10} />
            选中文字即可 AI 编辑
            </span>
          )}
        </div>
      )}
      <div ref={containerRef} className="overflow-auto h-[calc(100%-30px)] docx-preview-body" />

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
      `}</style>
    </div>
  );
}
