import { BookOpen, Search, X, Plus, Pencil, Trash2, Check, X as XIcon, Save } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { KnowledgeEntry, KnowledgeSaveRequest, KnowledgeSummary } from "../lib/types";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { EmptyState } from "./EmptyState";

const CATEGORIES = ["all", "规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他"];
const PHASES = ["all", "调查", "设计", "施工", "验收", "运维", "全程"];
const STATUSES = ["all", "现行", "已归档", "常用", "草稿"];

export function KnowledgePanel(p: { onClose: () => void }) {
  const { onClose } = p;
  const t = useT();
  const [entries, setEntries] = useState<KnowledgeSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("all");
  const [phase, setPhase] = useState("all");
  const [status, setStatus] = useState("all");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [expandedEntry, setExpandedEntry] = useState<KnowledgeEntry | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isAdding, setIsAdding] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  // Edit form state
  const [form, setForm] = useState<KnowledgeSaveRequest>({
    name: "", title: "", category: "", phase: "", discipline: "",
    tags: [], status: "现行", version: 1, author: "", reviewer: "",
    source: "", body: "", updatedAt: "", createdAt: "",
  });

  const loadList = useCallback(() => {
    setLoading(true);
    app.KnowledgeList().then((list) => {
      setEntries(list);
      setLoading(false);
    });
  }, []);

  useEffect(() => { loadList(); }, [loadList]);
  useEffect(() => { searchRef.current?.focus(); }, []);

  // Normalized query
  const normalizedQuery = query.trim().toLowerCase();
  const filtered = useMemo(
    () => entries.filter((e) => {
      if (category !== "all" && e.category !== category) return false;
      if (phase !== "all" && (e as unknown as Record<string, unknown>).phase !== phase) return false;
      if (status !== "all" && e.status !== status) return false;
      if (!normalizedQuery) return true;
      return [e.title, e.name, e.category, ...e.tags]
        .join(" ").toLowerCase().includes(normalizedQuery);
    }),
    [entries, normalizedQuery, category, phase, status],
  );

  // Highlight matching text
  const highlightText = (text: string): string | React.ReactNode => {
    if (!normalizedQuery) return text;
    const regex = new RegExp(`(${normalizedQuery.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`, "gi");
    const parts = text.split(regex);
    return parts.map((part, i) =>
      regex.test(part) ? <mark key={i} className="bg-yellow-300/30 text-fg rounded px-0.5">{part}</mark> : part
    );
  };

  const handleToggle = useCallback(async (name: string) => {
    if (isEditing) { setIsEditing(false); return; }
    if (expanded === name) { setExpanded(null); setExpandedEntry(null); setDetailError(null); setIsEditing(false); return; }
    setExpanded(name); setExpandedEntry(null); setDetailError(null); setDetailLoading(true); setIsEditing(false);
    try {
      const entry = await app.KnowledgeGet(name);
      if (entry) setExpandedEntry(entry);
      else setDetailError("条目不存在");
    } catch { setDetailError("加载失败"); }
    setDetailLoading(false);
  }, [expanded, isEditing]);

  // ── Add / Save / Delete ────────────────────────────────────────

  const startAdd = () => {
    setIsAdding(true); setIsEditing(false);
    setExpanded(null); setExpandedEntry(null);
    setForm({ name: "", title: "", category: "", phase: "", discipline: "", tags: [], status: "现行", version: 1, author: "", reviewer: "", source: "", body: "", updatedAt: "", createdAt: "" });
  };

  const startEdit = (entry: KnowledgeEntry) => {
    setIsEditing(true); setIsAdding(false);
    setForm({ name: entry.name, title: entry.title, category: entry.category, phase: entry.phase, discipline: entry.discipline, tags: entry.tags, status: entry.status, version: entry.version, author: entry.author, reviewer: entry.reviewer, source: entry.source, body: entry.body, updatedAt: entry.updatedAt, createdAt: entry.createdAt });
  };

  const cancelEdit = () => { setIsEditing(false); setIsAdding(false); };

  const handleSave = async () => {
    if (!form.name || !/^[a-zA-Z0-9_.-]+$/.test(form.name)) {
      setDetailError("名称仅允许英文字母、数字、下划线、连字符和点号");
      return;
    }
    try {
      await app.KnowledgeSave(form);
      setIsEditing(false); setIsAdding(false); setExpanded(null); setExpandedEntry(null);
      loadList();
    } catch {
      setDetailError("保存失败");
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deleteConfirm) return;
    try {
      await app.KnowledgeDelete(deleteConfirm);
      setDeleteConfirm(null);
      if (expanded === deleteConfirm) { setExpanded(null); setExpandedEntry(null); }
      loadList();
    } catch {
      setDetailError("删除失败");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[64px] pb-8" style={{ background: "var(--ds-overlay)" }}>
      <div className="relative w-full max-w-[620px] max-h-full flex flex-col rounded-xl border border-border-soft bg-bg shadow-xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-2 px-5 py-3.5 border-b border-border-soft shrink-0">
          <BookOpen size={17} className="text-accent" />
          <h2 className="flex-1 text-fg text-[14px] font-semibold">{t("knowledge.title")}</h2>
          <button className="inline-flex items-center justify-center w-7 h-7 border-0 rounded-md bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={onClose} aria-label={t("common.close")} type="button"><X size={15} /></button>
        </div>

        {/* Search & Filters */}
        <div className="shrink-0 px-4 pt-3 pb-2 space-y-2">
          <div className="flex items-center gap-1.5 px-3 h-8 border border-border rounded-lg bg-bg text-fg-faint focus-within:border-accent transition-colors">
            <Search size={14} />
            <input ref={searchRef} className="flex-1 min-w-0 border-0 outline-none bg-transparent text-fg text-[12.5px] placeholder:text-fg-faint" placeholder={t("knowledge.search")} value={query} onChange={(e) => setQuery(e.target.value)} aria-label={t("knowledge.search")} />
            {query && <button className="bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg p-0" onClick={() => setQuery("")} type="button"><X size={12} /></button>}
          </div>
          <div className="flex items-center gap-1.5 flex-wrap">
            {CATEGORIES.map((cat) => (
              <button key={cat} className={`px-2.5 py-1 rounded-full text-[11.5px] border cursor-pointer transition-colors ${category === cat ? "bg-accent text-accent-fg border-accent" : "bg-transparent text-fg-faint border-border-soft hover:border-accent hover:text-fg"}`} onClick={() => setCategory(cat)} type="button">
                {cat === "all" ? t("knowledge.all") : cat}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-1.5 flex-wrap">
            {PHASES.map((ph) => (
              <button key={ph} className={`px-2 py-0.5 rounded text-[11px] border cursor-pointer transition-colors ${phase === ph ? "bg-accent/20 text-accent border-accent/40" : "bg-transparent text-fg-faint border-border-soft hover:border-accent/30 hover:text-fg"}`} onClick={() => setPhase(ph)} type="button">
                {ph === "all" ? t("knowledge.all") : ph}
              </button>
            ))}
            <span className="text-fg-faint text-[11px] mx-1">|</span>
            {STATUSES.map((st) => (
              <button key={st} className={`px-2 py-0.5 rounded text-[11px] border cursor-pointer transition-colors ${status === st ? "bg-accent/20 text-accent border-accent/40" : "bg-transparent text-fg-faint border-border-soft hover:border-accent/30 hover:text-fg"}`} onClick={() => setStatus(st)} type="button">
                {st === "all" ? t("knowledge.all") : st}
              </button>
            ))}
            <button className="ml-auto flex items-center gap-1 px-2 py-1 rounded-md bg-accent text-white text-[12px] hover:opacity-90" onClick={startAdd} type="button"><Plus size={13} />{t("knowledge.new")}</button>
          </div>
          <div className="text-[11px] text-fg-faint">{t("knowledge.count", { n: filtered.length })}</div>
        </div>

        {/* List */}
        <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-3">
          {loading ? (
            <div className="py-10 text-center text-fg-faint text-[13px]">{t("common.loading")}</div>
          ) : entries.length === 0 ? (
            <EmptyState message={t("knowledge.empty")} />
          ) : filtered.length === 0 && (!isAdding) ? (
            <div className="py-10 text-center text-fg-faint text-[13px]">
              {t("knowledge.noMatch")}
              {(query || category !== "all" || phase !== "all" || status !== "all") && (
                <button className="block mx-auto mt-2 text-accent text-[12px] bg-transparent border-0 cursor-pointer hover:underline" onClick={() => { setQuery(""); setCategory("all"); setPhase("all"); setStatus("all"); }} type="button">
                  {t("memory.clearFilters")}
                </button>
              )}
            </div>
          ) : (
            <div className="flex flex-col gap-2 pt-2">
              {/* New entry form */}
              {isAdding && (
                <div className="p-3 rounded-lg border border-accent bg-sidebar-active">
                  <EditForm form={form} setForm={setForm} t={t} />
                  <div className="flex gap-2 mt-2 justify-end">
                    <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px]" onClick={handleSave} type="button"><Save size={13} />{t("knowledge.save")}</button>
                    <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px]" onClick={cancelEdit} type="button">{t("common.cancel")}</button>
                  </div>
                </div>
              )}

              {filtered.map((entry) => (
                <div key={entry.name}>
                  {/* Card */}
                  <button className={`w-full text-left flex flex-col gap-1 px-3 py-2.5 rounded-lg border cursor-pointer transition-colors ${expanded === entry.name ? "border-accent bg-sidebar-active" : "border-border-soft bg-bg hover:border-accent-soft hover:bg-bg-soft"}`}
                    onClick={() => void handleToggle(entry.name)} type="button">
                    <div className="flex items-start gap-2">
                      <span className="flex-1 text-fg text-[13px] font-medium leading-snug">
                        {normalizedQuery ? highlightText(entry.title) : entry.title}
                      </span>
                      <span className="shrink-0 text-[10.5px] text-accent font-medium px-1.5 py-0.5 rounded-full bg-accent/10">{entry.category}</span>
                    </div>
                    <div className="flex flex-wrap gap-1.5 text-[10px] text-fg-faint">
                      {(entry as unknown as Record<string, string>).phase && <span>{(entry as unknown as Record<string, string>).phase}</span>}
                      {(entry as unknown as Record<string, string>).phase && <span>·</span>}
                      {entry.status && <span>{entry.status}</span>}
                    </div>
                    {entry.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1">
                        {entry.tags.map((tag) => (
                          <span key={tag} className="text-[10px] text-fg-faint px-1.5 py-0.5 rounded-full bg-bg-soft">
                            {normalizedQuery ? highlightText(tag) : tag}
                          </span>
                        ))}
                      </div>
                    )}
                    <div className="text-[10.5px] text-fg-faint">{entry.updatedAt && new Date(entry.updatedAt).toLocaleDateString()}</div>
                  </button>

                  {/* Expanded detail */}
                  {expanded === entry.name && (
                    <div className="mx-2 px-3 py-3 border-l-2 border-accent/40 bg-bg-soft rounded-r-lg mt-0.5">
                      {detailLoading ? (
                        <div className="text-fg-faint text-[12px]">{t("common.loading")}</div>
                      ) : detailError ? (
                        <div className="text-red-500 text-[12px]">{detailError}</div>
                      ) : expandedEntry ? (
                        isEditing ? (
                          <div>
                            <EditForm form={form} setForm={setForm} t={t} />
                            <div className="flex gap-2 mt-3 justify-end">
                              <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px]" onClick={handleSave} type="button"><Save size={13} />{t("knowledge.save")}</button>
                              <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px]" onClick={cancelEdit} type="button">{t("common.cancel")}</button>
                            </div>
                          </div>
                        ) : (
                          <div className="space-y-2">
                            <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-fg-faint">
                              {expandedEntry.author && <span>作者: {expandedEntry.author}</span>}
                              {expandedEntry.phase && <span>阶段: {expandedEntry.phase}</span>}
                              {expandedEntry.discipline && <span>专业: {expandedEntry.discipline}</span>}
                              {expandedEntry.source && <span>来源: {expandedEntry.source}</span>}
                              {expandedEntry.version > 0 && <span>版本: v{expandedEntry.version}</span>}
                              {expandedEntry.reviewer && <span>审核: {expandedEntry.reviewer}</span>}
                              {expandedEntry.createdAt && <span>创建: {new Date(expandedEntry.createdAt).toLocaleDateString()}</span>}
                            </div>
                            <div className="text-fg text-[12.5px] leading-relaxed whitespace-pre-wrap break-words">
                              {expandedEntry.body}
                            </div>
                            <div className="flex gap-2 pt-1">
                              <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[11px] hover:bg-sidebar-hover" onClick={() => startEdit(expandedEntry)} type="button"><Pencil size={12} />{t("common.edit")}</button>
                              {deleteConfirm === entry.name ? (
                                <div className="flex items-center gap-1">
                                  <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-red-600 text-white text-[11px]" onClick={handleDeleteConfirm} type="button"><Check size={12} />{t("common.confirm")}</button>
                                  <button className="px-2 py-1 rounded-md bg-bg-soft text-fg text-[11px]" onClick={() => setDeleteConfirm(null)} type="button"><XIcon size={12} /></button>
                                </div>
                              ) : (
                                <button className="flex items-center gap-1 px-2 py-1 rounded-md text-red-500 text-[11px] hover:bg-red-500/10" onClick={() => setDeleteConfirm(entry.name)} type="button"><Trash2 size={12} />{t("knowledge.delete")}</button>
                              )}
                            </div>
                          </div>
                        )
                      ) : null}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/** Inline edit form for knowledge entry fields */
function EditForm({ form, setForm, t }: {
  form: KnowledgeSaveRequest;
  setForm: (f: KnowledgeSaveRequest) => void;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  t: (...args: any[]) => string;
}) {
  const update = (partial: Partial<KnowledgeSaveRequest>) => setForm({ ...form, ...partial });

  return (
    <div className="space-y-2">
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
