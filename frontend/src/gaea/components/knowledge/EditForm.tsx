// KnowledgePanel 拆分——内联编辑表单（P3 结构版2 次批巨文件拆分）。
// 内容自 KnowledgePanel.tsx 原文件整体迁出，逐字节照搬；行为零变化。
import type { KnowledgeSaveRequest, SimilarView } from "../../lib/types";
import type { Translator } from "../../lib/i18n";
import { CATEGORIES, STATUSES } from "./constants";

/** Inline edit form for knowledge entry fields */
export function EditForm({ form, setForm, t, similar }: {
  form: KnowledgeSaveRequest;
  setForm: (f: KnowledgeSaveRequest) => void;
  t: Translator;
  similar?: SimilarView[];
}) {
  const update = (partial: Partial<KnowledgeSaveRequest>) => setForm({ ...form, ...partial });

  return (
    <div className="space-y-2">
      {similar && similar.length > 0 && (
        <div className="px-2 py-1.5 rounded-md bg-amber-500/10 text-amber-400 text-[11px]">
          疑似重复：{similar.slice(0, 3).map((s) => `${s.title}（${Math.round(s.score * 100)}%）`).join("、")}
        </div>
      )}
      <div className="flex gap-2">
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none focus:border-accent" placeholder={t("knowledge.namePlaceholder")} value={form.name} onChange={(e) => update({ name: e.target.value })} disabled={!!(form.updatedAt && form.updatedAt !== "")} />
        <select className="px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" value={form.category} onChange={(e) => update({ category: e.target.value })}>
          {CATEGORIES.filter((c) => c !== "all").map((c) => (<option key={c} value={c}>{c}</option>))}
        </select>
      </div>
      <input className="w-full px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none focus:border-accent" placeholder={t("knowledge.title")} value={form.title} onChange={(e) => update({ title: e.target.value })} />
      <div className="flex gap-2">
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder={t("knowledge.phase")} value={form.phase} onChange={(e) => update({ phase: e.target.value })} />
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder="专业" value={form.discipline} onChange={(e) => update({ discipline: e.target.value })} />
      </div>
      <div className="flex gap-2">
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder={t("knowledge.tags")} value={form.tags.join(", ")} onChange={(e) => update({ tags: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })} />
        <select className="px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" value={form.status} onChange={(e) => update({ status: e.target.value })}>
          {STATUSES.filter((s) => s !== "all").map((s) => (<option key={s} value={s}>{s}</option>))}
        </select>
      </div>
      <div className="flex gap-2">
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder={t("knowledge.author")} value={form.author} onChange={(e) => update({ author: e.target.value })} />
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder="审核人" value={form.reviewer} onChange={(e) => update({ reviewer: e.target.value })} />
        <input className="flex-1 px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none" placeholder={t("knowledge.source")} value={form.source} onChange={(e) => update({ source: e.target.value })} />
      </div>
      <textarea className="w-full min-h-[150px] px-2 py-1 rounded bg-bg border border-border text-[12px] text-fg outline-none focus:border-accent font-mono" placeholder={t("knowledge.body")} value={form.body} onChange={(e) => update({ body: e.target.value })} />
    </div>
  );
}