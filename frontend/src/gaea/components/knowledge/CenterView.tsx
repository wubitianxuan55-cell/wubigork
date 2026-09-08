// KnowledgePanel 拆分——中区「详情 / 编辑」视图（page variant）（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件 renderCenter 整体迁出，逐字节照搬；行为零变化。
import { BookOpen, Check, Clock, Pencil, Plus, Save, Trash2, X as XIcon } from "../../icons";
import type { KnowledgeEntry, KnowledgeSaveRequest, SimilarView } from "../../lib/types";
import type { Translator } from "../../lib/i18n";
import { Markdown } from "../Markdown";
import { EditForm } from "./EditForm";
import { FOCUS_RING } from "./constants";
import { statusBadgeStyle } from "./utils";

// 中区：详情 / 编辑
export function CenterView({
  t, isAdding, isEditing, expanded, detailLoading, detailError, expandedEntry,
  form, setForm, similar, deleteConfirm,
  onStartAdd, onStartEdit, onCancelEdit, onSave, onDeleteConfirm, onSetDeleteConfirm,
  onOpenHistory, onDoReview, onOpenMerge,
}: {
  t: Translator;
  isAdding: boolean;
  isEditing: boolean;
  expanded: string | null;
  detailLoading: boolean;
  detailError: string | null;
  expandedEntry: KnowledgeEntry | null;
  form: KnowledgeSaveRequest;
  setForm: (f: KnowledgeSaveRequest) => void;
  similar?: SimilarView[];
  deleteConfirm: string | null;
  onStartAdd: () => void;
  onStartEdit: (entry: KnowledgeEntry) => void;
  onCancelEdit: () => void;
  onSave: () => void;
  onDeleteConfirm: () => void;
  onSetDeleteConfirm: (name: string | null) => void;
  onOpenHistory: (name: string) => void;
  onDoReview: (entry: KnowledgeEntry) => void;
  onOpenMerge: (entry: KnowledgeEntry) => void;
}) {
  if (isAdding) {
    return (
      <div className="flex-1 min-h-0 overflow-y-auto p-4">
        <EditForm form={form} setForm={setForm} t={t} similar={similar} />
        <div className="flex gap-2 mt-3 justify-end">
          <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px] cursor-pointer" onClick={onSave} type="button"><Save size={13} aria-hidden="true" />{t("knowledge.save")}</button>
          <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px] cursor-pointer" onClick={onCancelEdit} type="button">{t("common.cancel")}</button>
        </div>
      </div>
    );
  }
  if (!expanded) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-3 p-6 text-center">
        <div className="w-12 h-12 rounded-2xl flex items-center justify-center text-fg-faint" style={{ background: "var(--bg-soft, var(--md-sys-color-surface-container))" }}>
          <BookOpen size={22} aria-hidden="true" />
        </div>
        <p className="text-fg-faint text-[12.5px] max-w-[38ch] leading-relaxed">从左侧选择一条知识条目查看详情，或点击「新建」录入规范、案例与经验。</p>
        <button className={`flex items-center gap-1 px-3 h-8 rounded-lg bg-accent text-accent-fg text-[12px] font-medium cursor-pointer hover:opacity-90 transition-opacity ${FOCUS_RING}`} onClick={onStartAdd} type="button"><Plus size={13} aria-hidden="true" />{t("knowledge.new")}</button>
      </div>
    );
  }
  if (detailLoading) {
    return <div className="flex-1 flex items-center justify-center text-fg-faint text-[13px]">{t("common.loading")}</div>;
  }
  if (detailError) {
    return <div className="flex-1 flex items-center justify-center text-red-500 text-[13px] p-6 text-center">{detailError}</div>;
  }
  if (expandedEntry) {
    if (isEditing) {
      return (
        <div className="flex-1 min-h-0 overflow-y-auto p-4">
          <EditForm form={form} setForm={setForm} t={t} similar={similar} />
          <div className="flex gap-2 mt-3 justify-end">
            <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px] cursor-pointer" onClick={onSave} type="button"><Save size={13} aria-hidden="true" />{t("knowledge.save")}</button>
            <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px] cursor-pointer" onClick={onCancelEdit} type="button">{t("common.cancel")}</button>
          </div>
        </div>
      );
    }
    return (
      <>
        {/* 头部：标题 + 状态徽标 + 元数据 */}
        <div className="shrink-0 flex items-start gap-2 px-4 py-3 border-b border-border-soft">
          <div className="flex-1 min-w-0">
            <h3 className="text-[15px] font-semibold text-fg leading-snug break-words">{expandedEntry.title}</h3>
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[10.5px] text-fg-faint">
              {expandedEntry.author && <span>作者: {expandedEntry.author}</span>}
              {expandedEntry.phase && <span>阶段: {expandedEntry.phase}</span>}
              {expandedEntry.discipline && <span>专业: {expandedEntry.discipline}</span>}
              {expandedEntry.version > 0 && <span>版本: v{expandedEntry.version}</span>}
              {expandedEntry.reviewer && <span>审核: {expandedEntry.reviewer}</span>}
              {expandedEntry.createdAt && <span>创建: {new Date(expandedEntry.createdAt).toLocaleDateString()}</span>}
              {expandedEntry.updatedAt && <span>更新: {new Date(expandedEntry.updatedAt).toLocaleDateString()}</span>}
            </div>
          </div>
          <span className="shrink-0 text-[10.5px] font-medium px-2 py-0.5 rounded-full border" style={statusBadgeStyle(expandedEntry.status)}>{expandedEntry.status}</span>
        </div>
        {/* 正文：Markdown 渲染 */}
        <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3">
          {expandedEntry.body.trim() ? (
            <Markdown text={expandedEntry.body} />
          ) : (
            <p className="text-fg-faint text-[12px]">（无正文）</p>
          )}
        </div>
        {/* 操作条 */}
        <div className="shrink-0 flex flex-wrap items-center gap-1.5 px-4 py-2.5 border-t border-border-soft">
          <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => onStartEdit(expandedEntry)} type="button"><Pencil size={12} aria-hidden="true" />{t("common.edit")}</button>
          <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => onOpenHistory(expandedEntry.name)} type="button"><Clock size={12} aria-hidden="true" />版本历史</button>
          {expandedEntry.status === "草稿" && (
            <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-green-600/15 text-green-500 text-[11.5px] cursor-pointer hover:bg-green-600/25 transition-colors ${FOCUS_RING}`} onClick={() => onDoReview(expandedEntry)} type="button"><Check size={12} aria-hidden="true" />审核通过</button>
          )}
          <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => onOpenMerge(expandedEntry)} type="button" title="把相似条目合并进本条（标签并集、来源合并、旧条目留档删除）">合并相似…</button>
          {deleteConfirm === expandedEntry.name ? (
            <span className="flex items-center gap-1">
              <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-red-600 text-white text-[11.5px] cursor-pointer ${FOCUS_RING}`} onClick={onDeleteConfirm} type="button"><Check size={12} aria-hidden="true" />{t("common.confirm")}</button>
              <button className={`px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer ${FOCUS_RING}`} onClick={() => onSetDeleteConfirm(null)} type="button"><XIcon size={12} aria-hidden="true" /></button>
            </span>
          ) : (
            <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md text-red-500 text-[11.5px] cursor-pointer hover:bg-red-500/10 transition-colors ${FOCUS_RING}`} onClick={() => onSetDeleteConfirm(expandedEntry.name)} type="button"><Trash2 size={12} aria-hidden="true" />{t("knowledge.delete")}</button>
          )}
        </div>
      </>
    );
  }
  return null;
}