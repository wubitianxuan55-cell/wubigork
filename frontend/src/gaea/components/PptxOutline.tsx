import { useCallback, useEffect, useState } from "react";
import { AlertCircle, FilePpt, Loader2, Pencil } from "../icons";
import { app } from "../lib/bridge";
import type { PptxOutlineView } from "../lib/types";
import { useComposerInsertStore } from "../lib/store";
import { useToast } from "./Toast";

// GaeaPptxOutline 绑定由主代理接线（bindings_office.go wrapper + bridge.ts
// AppBindings/gaeaToGaea + dev mock）；此处用窄类型访问：接线前后均可编译，
// 未接线（dev mock 缺方法）时运行期优雅降级为「大纲不可用」诚实提示。
type PptxOutlineBinding = { PptxOutline?: (rel: string) => Promise<PptxOutlineView> };

// summaryOf 把该页正文文本框拼成单行摘要（大纲条目第二行；texts 后端已逐条截断）。
function summaryOf(texts: string[], max = 80): string {
  const joined = texts.join(" ｜ ").replace(/\s+/g, " ").trim();
  return joined.length > max ? joined.slice(0, max) + "…" : joined;
}

/**
 * PptxOutline 是 pptx 预览右侧的结构化大纲卡（v4.28 B2）：
 * 每页条目「第 N 页 · 标题 · 正文摘要」，点条目 → 滚动到该页（页锚点由
 * FilePreview 的逐页缩略渲染提供）；「针对第 N 页修改」→ composer 插入指令
 * 模板「请修改 <文件名> 的第 N 页：」（useComposerInsertStore.requestText，
 * 与 DocxPreview「引用到对话」走同一 composer 插入通道，不直接发送）。
 * 大纲不可用（python/python-pptx 缺失、文件解析失败等）→ 诚实提示原因，
 * 绝不假装有结构；逐页预览不受影响。
 */
export function PptxOutline({
  relPath,
  fileName,
  onPageSelect,
}: {
  relPath: string;
  fileName: string;
  onPageSelect?: (page: number) => void;
}) {
  const [view, setView] = useState<PptxOutlineView | null>(null);
  const [failed, setFailed] = useState(false);
  const toast = useToast();

  useEffect(() => {
    let live = true;
    setView(null);
    setFailed(false);
    const api = app as unknown as PptxOutlineBinding;
    Promise.resolve(api.PptxOutline?.(relPath))
      .then((r) => {
        if (!live) return;
        // 方法未接线（dev mock 缺 PptxOutline → undefined）按不可用降级，
        // 绝不停留在永久加载态。
        if (r) setView(r);
        else setFailed(true);
      })
      .catch(() => {
        if (live) setFailed(true);
      });
    return () => {
      live = false;
    };
  }, [relPath]);

  // 「针对第 N 页修改」：只往 composer 插入模板（光标接在冒号后让用户补全
  // 具体要求），不自动发送——编辑指令必须由用户确认内容后再发。
  const insertModify = useCallback(
    (page: number) => {
      useComposerInsertStore.getState().requestText(`请修改 ${fileName} 的第 ${page} 页：`);
      toast.show(`已插入「针对第 ${page} 页修改」指令，补全要求后发送`, "info");
    },
    [fileName, toast],
  );

  // 大纲不可用 / 拉取失败：同一诚实提示（原因随 view.error 透出）。
  if (failed || (view !== null && !view.available)) {
    const reason = view?.error || "绑定不可用";
    return (
      <aside data-testid="pptx-outline" className="w-56 shrink-0 border-l border-border-soft overflow-auto px-3 py-2">
        <div className="flex items-start gap-1.5 text-fg-faint text-[11px] leading-relaxed">
          <AlertCircle size={12} className="text-amber-500/70 shrink-0 mt-px" />
          <span>大纲不可用（{reason}），仅展示逐页预览。</span>
        </div>
      </aside>
    );
  }

  return (
    <aside data-testid="pptx-outline" className="w-56 shrink-0 border-l border-border-soft overflow-auto">
      <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border-soft text-fg-dim text-[11px] sticky top-0 bg-bg-elev-2/95 z-10">
        <FilePpt size={12} className="text-accent shrink-0" />
        <span className="font-medium">大纲</span>
        <span className="text-fg-faint ml-auto">{view?.slides.length ?? 0} 页</span>
      </div>
      {view === null ? (
        <div className="flex items-center justify-center gap-2 py-6 text-fg-faint text-[11px]">
          <Loader2 size={12} className="animate-spin" />
          读取大纲…
        </div>
      ) : (
        <ul className="px-1.5 py-1.5 flex flex-col gap-0.5">
          {view.slides.map((s) => {
            const summary = summaryOf(s.texts);
            return (
              <li key={s.index} className="rounded-md hover:bg-bg-soft/70">
                <button
                  type="button"
                  className="w-full text-left px-1.5 py-1.5 cursor-pointer bg-transparent border-0 p-0"
                  data-testid={`pptx-page-item-${s.index}`}
                  onClick={() => onPageSelect?.(s.index)}
                  title="点击滚动到该页"
                >
                  <div className="flex items-baseline gap-1.5 min-w-0">
                    <span className="text-[10px] text-accent/80 font-mono shrink-0">P{s.index}</span>
                    <span className="text-[11.5px] text-fg truncate">{s.title || "未命名页"}</span>
                  </div>
                  {summary && (
                    <div className="text-[10px] text-fg-faint leading-snug line-clamp-2 mt-0.5">{summary}</div>
                  )}
                </button>
                <div className="px-1.5 pb-1">
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-accent/10 hover:text-accent hover:border-accent/30"
                    data-testid={`pptx-modify-btn-${s.index}`}
                    onClick={() => insertModify(s.index)}
                    title="在输入框插入该页的修改指令模板"
                  >
                    <Pencil size={9} />
                    针对第 {s.index} 页修改
                  </button>
                </div>
              </li>
            );
          })}
          {view.slides.length === 0 && (
            <li className="px-2 py-3 text-fg-faint text-[11px]">该演示文稿没有可解析的页面。</li>
          )}
        </ul>
      )}
    </aside>
  );
}
