import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  AlertCircle,
  Check,
  FilePpt,
  Loader2,
  RefreshCw,
  Sparkles,
  Wand2,
  X,
} from "../icons";
import { app } from "../lib/bridge";
import type { PreviewResult } from "../lib/types";
import { diffDocxParagraphs } from "../lib/docxTextDiff";
import type { DiffRow } from "../lib/diff";
import type { ChangeDiff } from "../lib/planDiff";
import { ChangesDiff } from "./ChangesDiff";
import { useUpdatedFilesStore } from "../lib/store";

// GaeaPptxSlideText / GaeaPptxApplyEdit 走窄类型访问（对齐 PptxOutline 的既有
// 先例）：Go 侧绑定由主代理集成接线，接线前后均可编译；未接线（方法缺失返回
// undefined）时按「绑定不可用」诚实降级（错误态 + 重试），绝不静默假装成功。
type PptxSlideTextEntry = { index: number; paragraphs: string[] };
type PptxEditBinding = {
  PptxSlideText?: (rel: string) => Promise<PptxSlideTextEntry[]>;
  PptxApplyEdit?: (
    rel: string,
    slideIdx: number,
    target: string,
    replacement: string,
  ) => Promise<PreviewResult>;
};

// 预设动作：与 DocxPreview 框选即改同一套语义（润色/精简/翻译/扩写）。
const PRESETS = [
  { label: "润色", instruction: "润色这段文字，使表达更准确流畅，保持原意不变" },
  { label: "精简", instruction: "精简这段文字，去掉冗余表达，保留全部关键信息" },
  { label: "翻译成中文", instruction: "将这段文字翻译成规范的中文" },
  { label: "扩写", instruction: "扩写这段文字，补充必要的细节，保持原意和严谨性" },
];

// pptx 场景附加约束（设计 §3.2）：拼进 instruction 传给 GaeaOfficeEditText，
// 不动绑定本体——文本框容量有限，改写必须短句化且长度相近防溢出。
const PPTX_TEXT_CONSTRAINT = "附加约束：这是 PPT 文本框要点，请短句化表达，长度与原文相近，防止文本框溢出";

// splitSentenceUnits 把段落拆成句级单元（中英文句末标点/换行切分，标点归前句）。
// 用途：单段原文 vs 单段新文喂给段级 LCS（docxTextDiff）做对齐——未改动的句子
// 出 ctx 行（中性），改动的句子成相邻 del/add 行。
function splitSentenceUnits(text: string): string[] {
  return text
    .split(/(?<=[。！？；!?;\n])/g)
    .map((s) => s.trim())
    .filter((s) => s !== "");
}

// buildCompareRows 段级对齐（复用 docxTextDiff 的经典 LCS，不新写算法）。
// 句级单元做 LCS：未变句子 = ctx 行，改动 = 相邻 del+add 行——交给 ChangesDiff
// 的改蓝配对即自动获得行内字符级高亮（设计 §3.3 层3 白得）。DocxRow.index 是
// 段内句序号，不是 docx 段号/xlsx ref 那类用户可对照的坐标，归一为 DiffRow 时
// 不带 marker 列。
function buildCompareRows(original: string, proposal: string): DiffRow[] {
  return diffDocxParagraphs(splitSentenceUnits(original), splitSentenceUnits(proposal)).map(
    (r) => ({ type: r.type, text: r.text }),
  );
}

/**
 * PptxEditPanel 是 pptx 真编辑刀2的「编辑面」（设计
 * docs/gaea-pptx-edit-design-2026-09.md §3.2，P1 范式 = xlsx 同款
 * Plan→Apply：落盘前双栏对比 + 用户点「应用」才写盘）。
 *
 * 交互流：
 *   1. 载入：GaeaPptxSlideText 取每页段落**全文**（段落粒度 = GaeaPptxApplyEdit
 *      的 target 粒度；大纲卡 texts 是截断预览仅作导航，不能当替换目标）；
 *      段落单选即编辑目标（显示页码 + 段落序号）；载入失败诚实降级（错误态 +
 *      重试，绑定未接线按同款降级）。
 *   2. 预设动作 + 自定义指令 → GaeaOfficeEditText 生成替换文（pptx 场景约束
 *      以附加提示词拼进 instruction，绑定零改动）。
 *   3. 对比区（复用 ChangesDiff 统一渲染：句级 LCS 的相邻 del+add 对自动改蓝
 *      配对 + 行内字符级高亮，未变句子给 ctx 行、长上下文自动折叠）。
 *   4. 「应用」→ GaeaPptxApplyEdit(rel, slideIdx, target, replacement)；
 *      定位不到/原文不匹配的后端错误原样透出（宁拒不误改）；成功后把返回的
 *      新 PreviewResult 交宿主刷新预览（写盘后预览缓存已自动失效），并静默
 *      重拉段落全文对准已落盘内容。pptx 无修订标记，回滚走版本时间线（刀3）。
 *
 * 宿主形态：FilePreview pdf 分支右栏（与 PptxOutline 大纲卡同排，宿主先例 =
 * DocxPreview 的 DocxQueuePanel 右栏侧栏）。文案 zh-only 域内容层（schedule
 * 域先例，不加 shell i18n 键）。
 */
export function PptxEditPanel({
  relPath,
  fileName,
  initialSlide,
  onClose,
  onApplied,
}: {
  relPath: string;
  fileName: string;
  /** 宿主打开面板时的初始页码（来自大纲「编辑此页」/文本框条目）；面板内可跨页重选。 */
  initialSlide?: number;
  onClose: () => void;
  /** 应用成功：透传后端返回的新 PreviewResult，宿主刷新预览。 */
  onApplied: (r: PreviewResult) => void;
}) {
  // ── 载入状态机：loading / error / ready ──────────────────────────
  const [slides, setSlides] = useState<PptxSlideTextEntry[] | null>(null);
  const [loadState, setLoadState] = useState<"loading" | "error" | "ready">("loading");
  const [loadError, setLoadError] = useState("");
  const loadSeqRef = useRef(0);

  const loadSlides = useCallback(
    async (silent: boolean) => {
      const seq = ++loadSeqRef.current;
      if (!silent) {
        setLoadState("loading");
        setLoadError("");
      }
      const api = app as unknown as PptxEditBinding;
      try {
        const r = await Promise.resolve(api.PptxSlideText?.(relPath));
        if (seq !== loadSeqRef.current) return; // 已有更新的载入/卸载，丢弃过期结果
        if (!r) {
          // 绑定未接线（dev mock / Go 侧未合入 → 方法缺失）：诚实降级。
          setLoadState("error");
          setLoadError("绑定不可用（PptxSlideText 未接线，请升级客户端后重试）");
          return;
        }
        setSlides(r);
        setLoadState("ready");
      } catch (e) {
        if (seq !== loadSeqRef.current) return;
        setLoadState("error");
        setLoadError(e instanceof Error ? e.message : String(e));
      }
    },
    [relPath],
  );

  useEffect(() => {
    void loadSlides(false);
    const seq = loadSeqRef; // 快照 ref 对象：卸载后在途请求作废（react-hooks 建议，语义同 .current++）
    return () => {
      seq.current++;
    };
  }, [loadSlides]);

  // ── 编辑目标单选：{页码, 段落序号}，文本取段落全文 ────────────────
  const [sel, setSel] = useState<{ slide: number; para: number } | null>(null);
  // initialSlide 到货（载入完成）后预选该页第一段，减少一次点击。
  const initialSelRef = useRef(initialSlide);
  useEffect(() => {
    if (loadState !== "ready" || !slides) return;
    const want = initialSelRef.current;
    if (want === undefined) return;
    initialSelRef.current = undefined; // 只消费一次
    const s = slides.find((x) => x.index === want);
    if (s && s.paragraphs.length > 0) setSel((prev) => prev ?? { slide: s.index, para: 0 });
  }, [loadState, slides]);

  const selectedText = useMemo(() => {
    if (!sel) return null;
    const s = slides?.find((x) => x.index === sel.slide);
    return s?.paragraphs[sel.para] ?? null;
  }, [sel, slides]);

  // ── 生成 → 对比 → 应用（与 DocxPreview 同款状态机 + pptx 附加约束）──
  const [instruction, setInstruction] = useState("");
  const [generating, setGenerating] = useState(false);
  const [proposal, setProposal] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const [actionError, setActionError] = useState("");
  const [notice, setNotice] = useState("");
  const markUpdated = useUpdatedFilesStore((s) => s.markUpdated);

  // Esc 关闭面板（与 DocxPreview 编辑工具栏同一交互纪律）。
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // 成功提示条自动熄灭；卸载时清计时器（不向已卸载组件 setState）。
  useEffect(() => {
    if (!notice) return;
    const id = window.setTimeout(() => setNotice(""), 6000);
    return () => window.clearTimeout(id);
  }, [notice]);

  const runGenerate = useCallback(async () => {
    if (!selectedText || !instruction.trim()) return;
    setGenerating(true);
    setActionError("");
    setProposal(null);
    try {
      // 与 DocxPreview 同一消费口径：r.edited 为空按失败计，不静默。
      const r = await app.OfficeEditText(selectedText, `${instruction.trim()}。${PPTX_TEXT_CONSTRAINT}`);
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
  }, [selectedText, instruction]);

  const applyProposal = useCallback(async () => {
    if (!sel || !selectedText || !proposal) return;
    setApplying(true);
    setActionError("");
    try {
      const api = app as unknown as PptxEditBinding;
      const r = await Promise.resolve(api.PptxApplyEdit?.(relPath, sel.slide, selectedText, proposal));
      if (!r) {
        setActionError("绑定不可用（PptxApplyEdit 未接线，请升级客户端后重试）");
        return;
      }
      markUpdated(relPath);
      onApplied(r); // 宿主用返回的新 PreviewResult 刷新预览（缓存已自动失效）
      setProposal(null);
      setInstruction("");
      void loadSlides(true); // 静默重拉段落全文，对准已落盘内容
      setNotice(`已应用到第 ${sel.slide} 页：预览已刷新；如需回退，可在产物面板版本时间线回滚到应用前版本`);
    } catch (e) {
      // 定位不到/原文不匹配等后端错误原样透出（宁拒不误改）。
      setActionError(e instanceof Error ? e.message : String(e));
    } finally {
      setApplying(false);
    }
  }, [sel, selectedText, proposal, relPath, markUpdated, onApplied, loadSlides]);

  const closePanel = useCallback(() => {
    setProposal(null);
    setInstruction("");
    setActionError("");
    onClose();
  }, [onClose]);

  // 对比 diff：单段编辑天然一个 hunk（粒度 =「第 N 页·第 M 段」，页码/段号已在
  // 上方标题行给出，hunk 不再重复 label）；pptx 纯文本不传 path（无语法着色），
  // 改蓝配对 / 字符级高亮 / ctx 折叠全部由 ChangesDiff 内部完成。
  const compareDiff = useMemo<ChangeDiff | null>(
    () =>
      selectedText && proposal
        ? { kind: "diff", hunks: [{ rows: buildCompareRows(selectedText, proposal) }] }
        : null,
    [selectedText, proposal],
  );

  const paraBusy = generating || applying;

  return (
    <aside
      data-testid="pptx-edit-panel"
      className="w-72 shrink-0 border-l border-border-soft flex flex-col min-h-0 bg-bg-elev-2/60"
    >
      {/* 头部：标题 + 文件名 + 关闭 */}
      <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border-soft text-fg-dim text-[11px] shrink-0">
        <FilePpt size={12} className="text-accent shrink-0" />
        <span className="font-medium">编辑文本</span>
        <span className="text-fg-faint truncate ml-1 flex-1" title={fileName}>
          {fileName}
        </span>
        <button
          type="button"
          className="flex items-center justify-center w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg rounded shrink-0"
          data-testid="pptx-edit-close"
          onClick={closePanel}
          title="关闭编辑面板 (Esc)"
          aria-label="关闭编辑面板"
        >
          <X size={11} />
        </button>
      </div>

      {/* 载入态：转圈 / 错误+重试（诚实降级，绝不静默） */}
      {loadState === "loading" && (
        <div className="flex items-center justify-center gap-2 py-6 text-fg-faint text-[11px]" data-testid="pptx-edit-loading">
          <Loader2 size={12} className="animate-spin" />
          读取段落全文…
        </div>
      )}
      {loadState === "error" && (
        <div className="px-3 py-3 flex flex-col gap-2" data-testid="pptx-edit-load-error">
          <div className="flex items-start gap-1.5 text-err text-[11px] leading-relaxed">
            <AlertCircle size={12} className="shrink-0 mt-px" />
            <span className="break-all">段落读取失败：{loadError}</span>
          </div>
          <button
            type="button"
            className="inline-flex items-center justify-center gap-1 px-2 py-1 rounded-md border border-border-soft bg-transparent text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft self-start"
            data-testid="pptx-edit-retry"
            onClick={() => void loadSlides(false)}
          >
            <RefreshCw size={10} />
            重试
          </button>
        </div>
      )}

      {/* 就绪：段落单选（页码+段落序号）→ 指令 → 生成 → 对比 → 应用 */}
      {loadState === "ready" && slides !== null && (
        <>
          <div className="flex-1 min-h-0 overflow-auto px-1.5 py-1.5">
            {slides.length === 0 && (
              <div className="px-2 py-3 text-fg-faint text-[11px]">该演示文稿没有可解析的段落文本。</div>
            )}
            {slides.map((s) => (
              <div key={s.index} className="mb-1.5">
                <div className="px-1 py-0.5 text-[10px] text-fg-faint font-mono">第 {s.index} 页</div>
                <ul className="flex flex-col gap-px">
                  {s.paragraphs.map((p, i) => {
                    const active = sel?.slide === s.index && sel?.para === i;
                    return (
                      <li key={i}>
                        <button
                          type="button"
                          className={
                            "w-full text-left px-1.5 py-1 rounded-md border text-[11px] leading-snug cursor-pointer break-all transition-colors " +
                            (active
                              ? "border-accent/40 bg-accent/10 text-fg"
                              : "border-transparent bg-transparent text-fg-dim hover:bg-bg-soft/70")
                          }
                          data-testid={`pptx-para-${s.index}-${i}`}
                          onClick={() => {
                            setSel({ slide: s.index, para: i });
                            setProposal(null);
                            setActionError("");
                          }}
                          title={p}
                        >
                          <span className={"font-mono text-[9.5px] mr-1 " + (active ? "text-accent" : "text-fg-faint")}>
                            {s.index}.{i + 1}
                          </span>
                          {p.length > 64 ? `${p.slice(0, 64)}…` : p}
                        </button>
                      </li>
                    );
                  })}
                  {s.paragraphs.length === 0 && (
                    <li className="px-1.5 py-0.5 text-[10px] text-fg-faint">（该页无正文段落）</li>
                  )}
                </ul>
              </div>
            ))}
          </div>

          {/* 指令区：预设动作 + 自定义输入 + 生成（未选段落时禁用） */}
          <div className="border-t border-border-soft px-2.5 py-2 shrink-0">
            {proposal === null ? (
              <>
                <div className="flex items-center gap-1 text-[10.5px] text-fg-faint mb-1.5">
                  <Sparkles size={10} className="text-accent shrink-0" />
                  {sel ? (
                    <span>
                      编辑目标：第 {sel.slide} 页 第 {sel.para + 1} 段
                    </span>
                  ) : (
                    <span>先在上方选中要编辑的段落</span>
                  )}
                </div>
                <div className="flex flex-wrap gap-1 mb-1.5">
                  {PRESETS.map((p) => (
                    <button
                      key={p.label}
                      type="button"
                      className="px-1.5 py-0.5 rounded-md border border-border-soft bg-transparent text-[10.5px] text-fg-dim cursor-pointer hover:bg-accent/10 hover:text-accent hover:border-accent/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                      onClick={() => setInstruction(p.instruction)}
                      disabled={!selectedText}
                    >
                      {p.label}
                    </button>
                  ))}
                </div>
                <div className="flex gap-1.5">
                  <input
                    value={instruction}
                    onChange={(e) => setInstruction(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        void runGenerate();
                      }
                    }}
                    placeholder={selectedText ? "输入指令，如：改成更正式的表达…" : "先选中左侧段落"}
                    data-testid="pptx-edit-instruction"
                    className="flex-1 min-w-0 px-2 py-1.5 rounded-lg border border-border-soft bg-bg text-[11px] text-fg outline-none focus:border-accent/50 disabled:opacity-50"
                    disabled={!selectedText}
                  />
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-accent text-bg text-[11px] font-medium cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity shrink-0"
                    data-testid="pptx-edit-generate"
                    disabled={!selectedText || !instruction.trim() || paraBusy}
                    onClick={() => void runGenerate()}
                  >
                    {generating ? <Loader2 size={11} className="animate-spin" /> : <Wand2 size={11} />}
                    生成
                  </button>
                </div>
              </>
            ) : (
              <>
                {/* 对比区：统一 diff 查看器——相邻 del+add 对改蓝配对 + 行内
                    字符级高亮 + ctx 折叠（§3.3 层3，与版本时间线同源渲染） */}
                <div className="flex items-center gap-1 text-[10.5px] text-fg-faint mb-1.5">
                  <Sparkles size={10} className="text-accent shrink-0" />
                  <span>
                    对比预览：第 {sel?.slide} 页 第 {(sel?.para ?? 0) + 1} 段
                  </span>
                </div>
                <div data-testid="pptx-edit-compare" className="max-h-32 overflow-auto rounded-md">
                  {compareDiff && <ChangesDiff diff={compareDiff} />}
                </div>
                <div className="flex items-center justify-end gap-1.5 mt-1.5">
                  <button
                    type="button"
                    className="px-2 py-1 rounded-lg border border-border-soft bg-transparent text-[11px] text-fg-dim cursor-pointer hover:bg-bg-soft transition-colors disabled:opacity-50"
                    onClick={() => {
                      setProposal(null);
                      setInstruction("");
                    }}
                    disabled={applying}
                  >
                    重新生成
                  </button>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-accent text-bg text-[11px] font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed hover:opacity-90 transition-opacity"
                    data-testid="pptx-edit-apply"
                    disabled={applying}
                    onClick={() => void applyProposal()}
                  >
                    {applying ? <Loader2 size={11} className="animate-spin" /> : <Check size={11} />}
                    应用
                  </button>
                </div>
              </>
            )}
            {actionError && (
              <div className="mt-1.5 text-[10.5px] text-err leading-relaxed break-all" data-testid="pptx-edit-error">
                {actionError}
              </div>
            )}
            {/* P1 范式诚实标注：pptx 无修订标记，批准即落盘，回滚走版本时间线 */}
            <div className="mt-1.5 text-[9.5px] text-fg-faint leading-relaxed">
              PPT 格式不支持修订标记：点「应用」即写入文件并生效，可恢复到应用前版本。
            </div>
          </div>
        </>
      )}

      {/* 应用成功提示条（含回滚指引，指向版本时间线——刀3 接入处） */}
      {notice && (
        <div
          data-testid="pptx-edit-notice"
          className="flex items-start gap-1.5 px-2.5 py-1.5 border-t border-accent/30 bg-accent/10 text-accent text-[10.5px] leading-relaxed shrink-0"
        >
          <Check size={11} className="shrink-0 mt-px" />
          <span>{notice}</span>
        </div>
      )}
    </aside>
  );
}
