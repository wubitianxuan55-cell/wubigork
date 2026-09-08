// KnowledgePanel 拆分——右栏「检索结果 / 引用」inspector（page variant）（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件 renderInspector 整体迁出，逐字节照搬；行为零变化。
import { AlertCircle, FileText, Search } from "../../icons";
import type { KnowledgeEntry, KnowledgeSummary, SimilarView } from "../../lib/types";
import { FOCUS_RING } from "./constants";
import { highlightText, sourceOf } from "./utils";

// 右栏：检索结果 / 引用 inspector
export function Inspector({ query, filtered, expandedName, expandedEntry, similar, normalizedQuery, onToggle }: {
  query: string;
  filtered: KnowledgeSummary[];
  expandedName: string | null;
  expandedEntry: KnowledgeEntry | null;
  similar: SimilarView[];
  normalizedQuery: string;
  onToggle: (name: string) => void;
}) {
  const hits = query.trim() ? filtered : [];
  return (
    <>
      {/* 检索命中 */}
      <section aria-label="检索命中">
        <h4 className="flex items-center gap-1.5 text-[10.5px] font-semibold tracking-wide text-fg-faint uppercase">
          <Search size={11} aria-hidden="true" /> 检索命中
          {query.trim() ? <span className="tabular-nums">· {hits.length}</span> : null}
        </h4>
        {query.trim() ? (
          hits.length === 0 ? (
            <p className="mt-1.5 text-[11px] text-fg-faint">无命中条目</p>
          ) : (
            <ul className="mt-1.5 flex flex-col gap-1">
              {hits.slice(0, 8).map((h) => (
                <li key={h.name}>
                  <button type="button" onClick={() => onToggle(h.name)} aria-expanded={expandedName === h.name}
                    className={`w-full text-left px-2 py-1.5 rounded-md transition-colors cursor-pointer ${expandedName === h.name ? "bg-bg-soft" : "hover:bg-bg-soft"} ${FOCUS_RING}`}>
                    <span className="block text-[11.5px] text-fg truncate">{highlightText(h.title, normalizedQuery)}</span>
                    <span className="block text-[10px] text-fg-faint truncate">
                      {sourceOf(h) || h.category}
                      {h.updatedAt ? ` · ${new Date(h.updatedAt).toLocaleDateString()}` : ""}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )
        ) : (
          <p className="mt-1.5 text-[11px] text-fg-faint leading-relaxed">在中区搜索框输入关键词，命中的条目与其来源将在此汇总。</p>
        )}
      </section>

      <div className="v3-split-h" aria-hidden="true" />

      {/* 当前条目引用信息 */}
      <section aria-label="条目引用信息">
        <h4 className="flex items-center gap-1.5 text-[10.5px] font-semibold tracking-wide text-fg-faint uppercase">
          <FileText size={11} aria-hidden="true" /> 条目引用
        </h4>
        {expandedEntry ? (
          <dl className="mt-2 flex flex-col gap-1.5 text-[11px]">
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">来源</dt><dd className="text-fg text-right break-all">{expandedEntry.source || "未标注"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">作者</dt><dd className="text-fg text-right">{expandedEntry.author || "—"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">审核</dt><dd className="text-fg text-right">{expandedEntry.reviewer || "—"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">版本</dt><dd className="text-fg text-right">v{expandedEntry.version > 0 ? expandedEntry.version : 1}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">阶段</dt><dd className="text-fg text-right">{expandedEntry.phase || "—"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">专业</dt><dd className="text-fg text-right">{expandedEntry.discipline || "—"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">创建</dt><dd className="text-fg text-right tabular-nums">{expandedEntry.createdAt ? new Date(expandedEntry.createdAt).toLocaleDateString() : "—"}</dd></div>
            <div className="flex items-baseline justify-between gap-2"><dt className="text-fg-faint shrink-0">更新</dt><dd className="text-fg text-right tabular-nums">{expandedEntry.updatedAt ? new Date(expandedEntry.updatedAt).toLocaleDateString() : "—"}</dd></div>
          </dl>
        ) : (
          <p className="mt-1.5 text-[11px] text-fg-faint leading-relaxed">从左侧选择条目后，此处显示其来源与元数据引用。</p>
        )}
      </section>

      {/* 疑似重复（新建/编辑中） */}
      {similar.length > 0 && (
        <>
          <div className="v3-split-h" aria-hidden="true" />
          <section aria-label="疑似重复">
            <h4 className="flex items-center gap-1.5 text-[10.5px] font-semibold tracking-wide text-fg-faint uppercase">
              <AlertCircle size={11} aria-hidden="true" /> 疑似重复
            </h4>
            <ul className="mt-1.5 flex flex-col gap-1">
              {similar.slice(0, 3).map((s) => (
                <li key={s.name} className="px-2 py-1.5 rounded-md bg-bg-soft/60 text-[11px]">
                  <span className="block text-fg truncate">{s.title}</span>
                  <span className="block text-[10px] text-fg-faint tabular-nums">{Math.round(s.score * 100)}% 相似</span>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </>
  );
}