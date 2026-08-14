import { memo, useCallback, useEffect, useRef, useState } from "react";
import {
  ExternalLink,
  File,
  FilePpt,
  FileSpreadsheet,
  FileText,
  Loader,
  Paperclip,
  Search,
  X,
} from "../icons";
import { app, onTaskEvent } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import type { FileSemanticHit, TaskView, WorkspaceSearchHit } from "../lib/types";
import { useToast } from "./Toast";

function fileIcon(name: string) {
  if (/\.(docx?|md|markdown|txt)$/i.test(name)) return <FileText size={12} />;
  if (/\.(xlsx?|csv)$/i.test(name)) return <FileSpreadsheet size={12} />;
  if (/\.pptx?$/i.test(name)) return <FilePpt size={12} />;
  if (/\.pdf$/i.test(name)) return <File size={12} />;
  return <FileText size={12} />;
}

// 解析索引任务 result（JSON 字符串 {total, skipped}）为完成提示文案。
function indexResultText(task: TaskView): string {
  try {
    const r = JSON.parse(task.result || "{}") as { total?: number; skipped?: number };
    const skipped = r.skipped ?? 0;
    return `已索引 ${r.total ?? 0} 个文件${skipped ? `（跳过 ${skipped}）` : ""}`;
  } catch {
    return "索引完成";
  }
}

// WorkspaceSearchPanel — 右侧「搜索」视图：工作区全文检索（轻量 RAG）。
// 在 docx/xlsx/pdf/md/txt/csv 正文里定位关键词，命中片段可预览或一键 @ 引用
// （对标 Cursor 本地索引 / Cherry Studio 知识库检索）。
export const WorkspaceSearchPanel = memo(function WorkspaceSearchPanel({
  onOpenFile,
}: {
  onOpenFile: (path: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<WorkspaceSearchHit[]>([]);
  const [searching, setSearching] = useState(false);
  const [searched, setSearched] = useState(false);
  const [semantic, setSemantic] = useState(false);
  const [semHits, setSemHits] = useState<FileSemanticHit[]>([]);
  const [semSearching, setSemSearching] = useState(false);
  const [indexMsg, setIndexMsg] = useState<string | null>(null);
  const seq = useRef(0);
  // 索引任务事件订阅的注销函数（重复点击/卸载时断开，避免监听泄漏）
  const taskOff = useRef<(() => void) | null>(null);
  const requestAt = useComposerInsertStore((s) => s.requestAt);
  const toast = useToast();

  const run = useCallback((q: string) => {
    const trimmed = q.trim();
    const id = ++seq.current;
    if (!trimmed) {
      setHits([]);
      setSearching(false);
      setSearched(false);
      return;
    }
    setSearching(true);
    app.WorkspaceSearch(trimmed, 30)
      .then((h) => {
        if (id !== seq.current) return;
        setHits(h ?? []);
        setSearched(true);
      })
      .catch(() => {
        if (id !== seq.current) return;
        setHits([]);
        setSearched(true);
      })
      .finally(() => {
        if (id === seq.current) setSearching(false);
      });

    if (semantic) {
      setSemSearching(true);
      app.FileSemanticSearch(trimmed, 20)
        .then((h) => {
          if (id !== seq.current) return;
          setSemHits(h ?? []);
        })
        .catch(() => {
          if (id === seq.current) setSemHits([]);
        })
        .finally(() => {
          if (id === seq.current) setSemSearching(false);
        });
    } else {
      setSemHits([]);
    }
  }, [semantic]);

  const rebuildIndex = useCallback(async () => {
    // 重复点击时断开上一次未完成的订阅
    taskOff.current?.();
    taskOff.current = null;
    setIndexMsg("正在索引工作区…");
    try {
      const task = await app.FileIndexRebuild();
      // 终态结算：succeeded → 解析 result；failed → 错误信息；cancelled → 取消提示。
      const settle = (t: TaskView) => {
        if (t.status === "succeeded") setIndexMsg(indexResultText(t));
        else if (t.status === "failed") setIndexMsg(`索引失败：${t.error || "未知错误"}`);
        else setIndexMsg("索引已取消");
      };
      if (task.status === "queued" || task.status === "running") {
        // 任务异步执行中：订阅 gaea-task 事件，等该任务终态再结算。
        taskOff.current = onTaskEvent((t) => {
          if (t.id !== task.id) return;
          if (t.status === "queued" || t.status === "running") return; // 仍在进行
          settle(t);
          taskOff.current?.();
          taskOff.current = null;
          setTimeout(() => setIndexMsg(null), 3000);
        });
      } else {
        // 提交即终态（mock/快速完成）：直接结算
        settle(task);
        setTimeout(() => setIndexMsg(null), 3000);
      }
    } catch (e) {
      // 已有活动索引任务等后端错误（如"索引任务已在队列中"）：直接提示
      setIndexMsg(`索引失败：${String(e)}`);
      setTimeout(() => setIndexMsg(null), 3000);
    }
  }, []);

  // 300ms 防抖搜索
  useEffect(() => {
    const t = setTimeout(() => run(query), 300);
    return () => clearTimeout(t);
  }, [query, run]);

  // 卸载时断开索引任务订阅，避免事件监听泄漏
  useEffect(() => {
    return () => {
      taskOff.current?.();
    };
  }, []);

  const reference = useCallback((path: string) => {
    requestAt(path);
    toast.show(`已引用 @${path}`, "info");
  }, [requestAt, toast]);

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="px-3 py-2 border-b border-border-soft">
        <div className="flex items-center gap-1.5 font-semibold text-fg text-sm mb-2">
          <Search size={13} className="text-accent" />
          搜索
          <span className="ml-1.5 text-[10px] text-fg-faint font-normal">工作区全文</span>
          <button
            type="button"
            className={`ml-auto px-2 h-6 rounded-full text-[10.5px] transition-colors ${semantic ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"}`}
            onClick={() => setSemantic((s) => !s)}
            title="语义检索（本地 bge-m3，需先重建索引）"
          >
            语义
          </button>
          <button
            type="button"
            className="px-2 h-6 rounded-full text-[10.5px] bg-bg-elev text-fg-faint hover:text-fg border border-border"
            onClick={() => void rebuildIndex()}
            title="重建工作区文件语义索引"
          >
            重建索引
          </button>
        </div>
        {indexMsg && <div className="mt-1.5 text-[10.5px] text-accent">{indexMsg}</div>}
        <div className="relative">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索资料正文，如：成本 / 预算 / 方案…"
            className="w-full bg-bg-soft border border-border-soft rounded-md pl-8 pr-7 py-1.5 text-[12px] text-fg placeholder:text-fg-faint/60 outline-none focus:border-accent/40 transition-colors"
          />
          {query && (
            <button
              type="button"
              className="absolute right-1.5 top-1/2 -translate-y-1/2 w-5 h-5 flex items-center justify-center bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg"
              onClick={() => setQuery("")}
              title="清空"
            >
              <X size={11} />
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-2">
        {searching ? (
          <div className="flex flex-col items-center justify-center gap-2 py-10 text-fg-faint/70">
            <Loader size={18} className="animate-spin text-accent" />
            <span className="text-[11px]">正在索引工作区…</span>
          </div>
        ) : hits.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 px-6 py-10 text-center text-fg-faint/60">
            <Search size={24} className="opacity-40" />
            <span className="text-[11px] leading-relaxed">
              {searched
                ? "没有找到匹配的资料\n换个关键词试试"
                : "输入关键词搜索资料正文\n（docx / xlsx / pdf / md / csv）"}
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-1">
            <div className="px-1 pb-1 text-[10px] text-fg-faint/60">
              找到 {hits.length} 条 · 点击预览，悬停可引用
            </div>
            {hits.map((h) => (
              <div
                key={h.path}
                className="group flex items-start gap-2 px-2 py-1.5 rounded-md border border-border-soft/60 bg-bg-soft/25 hover:border-accent/30 hover:bg-bg-soft/60 transition-colors"
              >
                <span className="shrink-0 mt-0.5 w-6 h-6 rounded-md bg-accent/10 text-accent flex items-center justify-center">
                  {fileIcon(h.name)}
                </span>
                <button
                  type="button"
                  onClick={() => onOpenFile(h.path)}
                  title={`点击预览 ${h.path}`}
                  className="min-w-0 flex-1 text-left cursor-pointer"
                >
                  <span className="block truncate text-[12px] text-fg font-medium leading-tight">
                    {h.name}
                    <span className="ml-1.5 text-[9px] text-fg-faint font-mono align-middle">
                      {(h.score * 100).toFixed(0)}%
                    </span>
                    {h.truncated && (
                      <span className="ml-1.5 text-[9px] text-amber-500/90 font-mono align-middle">
                        索引截断
                      </span>
                    )}
                  </span>
                  <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">
                    {h.path}
                  </span>
                  {h.skipped ? (
                    <span className="block text-[11px] text-amber-500/90 leading-snug mt-1">
                      {h.skipped}
                    </span>
                  ) : (
                    <span className="block text-[11px] text-fg-dim leading-snug mt-1 line-clamp-2">
                      {h.snippet}
                    </span>
                  )}
                </button>
                <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    type="button"
                    className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                    onClick={() => reference(h.path)}
                    title="一键 @ 引用为对话上下文"
                  >
                    <Paperclip size={12} />
                  </button>
                  <button
                    type="button"
                    className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                    onClick={() => void app.OpenWorkspacePath(h.path).catch(() => {})}
                    title="在外部程序中打开"
                  >
                    <ExternalLink size={12} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {semantic && (semSearching || semHits.length > 0) && (
          <div className="mt-2">
            <div className="px-1 pb-1 text-[10px] text-fg-faint/60">
              语义命中（本地 bge-m3）{semSearching ? "检索中…" : ` · ${semHits.length} 条`}
            </div>
            {semSearching ? (
              <div className="flex items-center gap-2 py-4 px-1 text-fg-faint/70">
                <Loader size={14} className="animate-spin text-accent" />
                <span className="text-[11px]">语义检索中…</span>
              </div>
            ) : (
              <div className="flex flex-col gap-1">
                {semHits.map((h) => (
                  <div
                    key={h.path}
                    className="group flex items-start gap-2 px-2 py-1.5 rounded-md border border-accent/20 bg-accent/5 hover:border-accent/40 hover:bg-accent/10 transition-colors"
                  >
                    <span className="shrink-0 mt-0.5 w-6 h-6 rounded-md bg-accent/15 text-accent flex items-center justify-center">
                      {fileIcon(h.path)}
                    </span>
                    <button
                      type="button"
                      onClick={() => onOpenFile(h.path)}
                      title={`点击预览 ${h.path}`}
                      className="min-w-0 flex-1 text-left cursor-pointer"
                    >
                      <span className="block truncate text-[12px] text-fg font-medium leading-tight">
                        {h.path.split("/").pop()}
                        <span className="ml-1.5 text-[9px] text-accent font-mono align-middle">
                          {(h.score * 100).toFixed(0)}%
                        </span>
                      </span>
                      <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">{h.path}</span>
                      {h.snippet && (
                        <span className="block text-[11px] text-fg-dim leading-snug mt-1 line-clamp-2">{h.snippet}</span>
                      )}
                    </button>
                    <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        type="button"
                        className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                        onClick={() => reference(h.path)}
                        title="一键 @ 引用为对话上下文"
                      >
                        <Paperclip size={12} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
});
