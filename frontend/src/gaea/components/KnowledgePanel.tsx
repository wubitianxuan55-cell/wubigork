import { BookOpen, Clock, CloudUpload, Search, X, Plus, Pencil, Trash2, Check, X as XIcon, Save } from "../icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Modal } from "antd";
import type { FilePickResult, KnowledgeEntry, KnowledgeHistoryView, KnowledgeSaveRequest, KnowledgeSummary, SimilarView } from "../lib/types";
import { app } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import { EmptyState } from "./EmptyState";
import { KnowledgeImportModal } from "./memoryhub/KnowledgeImportModal";

const CATEGORIES = ["all", "规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他"];
const PHASES = ["all", "调查", "设计", "施工", "验收", "运维", "全程"];
const STATUSES = ["all", "现行", "已归档", "常用", "草稿"];

export function KnowledgePanel(p: { onClose: () => void; variant?: "modal" | "page" }) {
  const { onClose, variant = "modal" } = p;
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
  const [importFile, setImportFile] = useState<FilePickResult | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [similar, setSimilar] = useState<SimilarView[]>([]);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyName, setHistoryName] = useState("");
  const [historyRows, setHistoryRows] = useState<KnowledgeHistoryView[]>([]);
  const [exportMsg, setExportMsg] = useState<string | null>(null);
  const [mergeOpen, setMergeOpen] = useState(false);
  const [mergeTarget, setMergeTarget] = useState<string>("");
  const [mergeCandidates, setMergeCandidates] = useState<SimilarView[]>([]);
  const [mergeSelected, setMergeSelected] = useState<Set<string>>(new Set());
  const [mergeMsg, setMergeMsg] = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  // Edit form state
  const [form, setForm] = useState<KnowledgeSaveRequest>({
    name: "", title: "", category: "", phase: "", discipline: "",
    tags: [], status: "现行", version: 1, author: "", reviewer: "",
    source: "", body: "",
  });

  const loadList = useCallback(() => {
    setLoading(true);
    app.KnowledgeList().then((list) => {
      setEntries(list);
      setSelected((prev) => {
        const names = new Set(list.map((e) => e.name));
        return new Set([...prev].filter((n) => names.has(n)));
      });
      setLoading(false);
    });
  }, []);

  const pickImport = useCallback(async () => {
    try {
      const files = await app.PickFiles();
      const f = files?.[0];
      if (f) setImportFile(f);
    } catch { /* 原生对话框不可用时静默 */ }
  }, []);

  const toggleSelect = useCallback((name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const batchDelete = useCallback(async () => {
    if (selected.size === 0) return;
    await Promise.all([...selected].map((n) => app.KnowledgeDelete(n).catch(() => {})));
    setSelected(new Set());
    loadList();
  }, [selected, loadList]);

  const batchStatus = useCallback(async (next: string) => {
    if (selected.size === 0 || !next) return;
    for (const name of selected) {
      const e = await app.KnowledgeGet(name).catch(() => null);
      if (e) await app.KnowledgeSave({ ...e, status: next }).catch(() => {});
    }
    setSelected(new Set());
    loadList();
  }, [selected, loadList]);

  // 查重：新建/编辑且标题非空时，防抖提示疑似重复条目。
  useEffect(() => {
    const title = form.title.trim();
    if (!isAdding && !isEditing) { setSimilar([]); return; }
    if (!title) { setSimilar([]); return; }
    const timer = setTimeout(() => {
      app.KnowledgeFindSimilar(title).then((hits) => {
        const mine = isEditing ? form.name : "";
        setSimilar((hits ?? []).filter((h) => h.name !== mine));
      }).catch(() => {});
    }, 300);
    return () => clearTimeout(timer);
  }, [form.title, form.name, isAdding, isEditing]);

  const openHistory = useCallback(async (name: string) => {
    setHistoryName(name);
    setHistoryRows([]);
    setHistoryOpen(true);
    const rows = await app.KnowledgeHistory(name).catch(() => [] as KnowledgeHistoryView[]);
    setHistoryRows(rows ?? []);
  }, []);

  const doExport = useCallback(async () => {
    setExportMsg("导出中…");
    try {
      const n = await app.KnowledgeExport("");
      setExportMsg(`已导出 ${n} 条到 .gaea/exports/knowledge-*`);
      setTimeout(() => setExportMsg(null), 3000);
    } catch {
      setExportMsg("导出失败");
      setTimeout(() => setExportMsg(null), 3000);
    }
  }, []);

  const doReview = useCallback(async (entry: KnowledgeEntry) => {
    await app.KnowledgeReview(entry.name, true, "").catch(() => {});
    const fresh = await app.KnowledgeGet(entry.name).catch(() => null);
    if (fresh) setExpandedEntry(fresh);
    loadList();
  }, [loadList]);

  const openMerge = useCallback(async (entry: KnowledgeEntry) => {
    setMergeTarget(entry.name);
    setMergeSelected(new Set());
    const hits = await app.KnowledgeFindSimilar(entry.title).catch(() => [] as SimilarView[]);
    setMergeCandidates(hits ?? []);
    setMergeOpen(true);
  }, []);

  const doMerge = useCallback(async () => {
    if (mergeSelected.size === 0) return;
    const sources = [...mergeSelected];
    const target = await app.KnowledgeMerge(mergeTarget, sources).catch(() => "");
    setMergeOpen(false);
    setMergeMsg(`已把 ${sources.length} 条合并到「${target}」`);
    setTimeout(() => setMergeMsg(null), 3000);
    loadList();
  }, [mergeSelected, mergeTarget, loadList]);

  // 全文检索：query 非空时走后端全文搜索（含正文），否则走 List + 前端分类过滤。
  const doSearch = useCallback(async (q: string, cat: string, ph: string, st: string) => {
    setLoading(true);
    try {
      const list = q.trim()
        ? await app.KnowledgeSearch(q, cat, ph, st)
        : await app.KnowledgeList();
      setEntries(list);
    } catch { /* 搜索失败保持原列表 */ }
    setLoading(false);
  }, []);

  useEffect(() => { loadList(); }, [loadList]);
  useEffect(() => { searchRef.current?.focus(); }, []);
  // query/分类/阶段/状态变化时全文检索（防抖 250ms）。
  // 后端 Search 同时过滤正文与元数据，前端 filtered 仅作兜底（数据已过滤）。
  useEffect(() => {
    const timer = setTimeout(() => {
      void doSearch(query, category, phase, status);
    }, 250);
    return () => clearTimeout(timer);
  }, [query, category, phase, status, doSearch]);

  // Normalized query
  const normalizedQuery = query.trim().toLowerCase();
  // 列表数据已由 doSearch 处理（query 走后端全文，空 query 走 List），
  // 这里只做分类/阶段/状态的前端过滤。
  const filtered = useMemo(
    () => entries.filter((e) => {
      if (category !== "all" && e.category !== category) return false;
      if (phase !== "all" && (e as unknown as Record<string, unknown>).phase !== phase) return false;
      if (status !== "all" && e.status !== status) return false;
      return true;
    }),
    [entries, category, phase, status],
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
    setForm({ name: "", title: "", category: "", phase: "", discipline: "", tags: [], status: "现行", version: 1, author: "", reviewer: "", source: "", body: "" });
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

  const rootCls = variant === "page"
    ? "w-full h-full flex flex-col bg-bg"
    : "fixed inset-0 z-50 flex items-start justify-center pt-[64px] pb-8";
  return (
    <div className={rootCls} style={variant === "modal" ? { background: "var(--ds-overlay)" } : undefined}>
      <div className={variant === "page" ? "w-full h-full flex flex-col min-h-0" : "relative w-full max-w-[620px] max-h-full flex flex-col rounded-xl border border-border-soft bg-bg shadow-xl overflow-hidden"}>
        <div className="flex items-center gap-2 px-5 py-3.5 border-b border-border-soft shrink-0">
          <BookOpen size={variant === "page" ? 19 : 17} className="text-accent" />
          <h2 className={`flex-1 text-fg font-semibold ${variant === "page" ? "text-[15px]" : "text-[14px]"}`}>{t("knowledge.title")}</h2>
          {variant === "modal" && (
            <button className="inline-flex items-center justify-center w-7 h-7 border-0 rounded-md bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={onClose} aria-label={t("common.close")} type="button"><X size={15} /></button>
          )}
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
            <button className="ml-auto flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[12px] hover:bg-sidebar-hover" onClick={() => void pickImport()} type="button" title="导入 md/txt/docx/pdf/xlsx/csv">
              <CloudUpload size={13} className="text-accent" />导入
            </button>
            <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[12px] hover:bg-sidebar-hover" onClick={() => void doExport()} type="button" title="批量导出为 Markdown">
              <Save size={13} />导出
            </button>
            <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-accent text-white text-[12px] hover:opacity-90" onClick={startAdd} type="button"><Plus size={13} />{t("knowledge.new")}</button>
          </div>
          {exportMsg && <div className="text-[11px] text-accent">{exportMsg}</div>}
          {mergeMsg && <div className="text-[11px] text-amber-400">{mergeMsg}</div>}
          {selected.size > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-[11.5px] text-accent">已选 {selected.size}</span>
              <select
                value=""
                onChange={(e) => { if (e.target.value) void batchStatus(e.target.value); }}
                className="px-1.5 h-6 rounded-md bg-bg-soft text-fg-dim text-[11px] border border-border outline-none"
              >
                <option value="" disabled>改状态…</option>
                {["现行", "草稿", "常用", "已归档"].map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
              <button className="px-2 h-6 rounded-md bg-red-500/15 text-red-400 text-[11px] cursor-pointer hover:bg-red-500/25" onClick={() => void batchDelete()} type="button">批量删除</button>
            </div>
          )}
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
                  <EditForm form={form} setForm={setForm} t={t} similar={similar} />
                  <div className="flex gap-2 mt-2 justify-end">
                    <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px]" onClick={handleSave} type="button"><Save size={13} />{t("knowledge.save")}</button>
                    <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px]" onClick={cancelEdit} type="button">{t("common.cancel")}</button>
                  </div>
                </div>
              )}

              {filtered.map((entry) => (
                <div key={entry.name}>
                  {/* Card */}
                  <div className="flex items-start gap-1.5">
                    <input
                      type="checkbox"
                      checked={selected.has(entry.name)}
                      onChange={() => toggleSelect(entry.name)}
                      title="多选（批量删除/改状态）"
                      className="mt-2.5 shrink-0"
                    />
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
                  </div>

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
                            <EditForm form={form} setForm={setForm} t={t} similar={similar} />
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
                            <div className="text-fg text-[12.5px] leading-relaxed whitespace-pre-wrap break-words max-h-[320px] overflow-y-auto pr-1">{expandedEntry.body}</div>
                            <div className="flex gap-2 pt-1">
                              <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[11px] hover:bg-sidebar-hover" onClick={() => startEdit(expandedEntry)} type="button"><Pencil size={12} />{t("common.edit")}</button>
                              <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[11px] hover:bg-sidebar-hover" onClick={() => void openHistory(expandedEntry.name)} type="button"><Clock size={12} />版本历史</button>
                              {expandedEntry.status === "草稿" && (
                                <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-green-600/15 text-green-500 text-[11px] hover:bg-green-600/25" onClick={() => void doReview(expandedEntry)} type="button"><Check size={12} />审核通过</button>
                              )}
                              <button className="flex items-center gap-1 px-2 py-1 rounded-md bg-bg-soft text-fg text-[11px] hover:bg-sidebar-hover" onClick={() => void openMerge(expandedEntry)} type="button" title="把相似条目合并进本条（标签并集、来源合并、旧条目留档删除）">合并相似…</button>
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
      <Modal
        title={`版本历史：${historyName}`}
        open={historyOpen}
        onCancel={() => setHistoryOpen(false)}
        footer={null}
        width={560}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
      >
        <div className="space-y-2 max-h-[46vh] overflow-auto">
          {historyRows.length === 0 ? (
            <div className="py-6 text-center text-fg-faint text-[12px]">暂无版本历史（保存时内容变化会自动留档）</div>
          ) : (
            historyRows.map((h, i) => (
              <div key={i} className="p-2 rounded-lg bg-bg-soft/40 text-[12px]">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-fg">v{h.version}</span>
                  <span className="text-fg-faint">{h.note}</span>
                  <span className="ml-auto text-fg-faint text-[10.5px]">{h.changedAt ? new Date(h.changedAt).toLocaleString("zh-CN", { hour12: false }) : ""}</span>
                </div>
                <div className="mt-1 text-fg-dim whitespace-pre-wrap break-words max-h-[120px] overflow-y-auto">{h.body}</div>
              </div>
            ))
          )}
        </div>
      </Modal>
      <Modal
        title={`合并相似条目到「${mergeTarget}」`}
        open={mergeOpen}
        onCancel={() => setMergeOpen(false)}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        footer={
          <div className="flex items-center justify-end gap-2">
            <button className="px-3 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft text-[12px]" onClick={() => setMergeOpen(false)} type="button">取消</button>
            <button
              className="px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 disabled:opacity-50"
              onClick={() => void doMerge()}
              disabled={mergeSelected.size === 0}
              type="button"
            >
              合并 {mergeSelected.size} 条
            </button>
          </div>
        }
        width={520}
      >
        {mergeCandidates.length === 0 ? (
          <div className="py-6 text-center text-fg-faint text-[12px]">暂无相似条目</div>
        ) : (
          <div className="space-y-1.5 max-h-[40vh] overflow-auto">
            {mergeCandidates.map((s) => (
              <label key={s.name} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px] cursor-pointer">
                <input
                  type="checkbox"
                  checked={mergeSelected.has(s.name)}
                  onChange={() => setMergeSelected((prev) => {
                    const next = new Set(prev);
                    if (next.has(s.name)) next.delete(s.name);
                    else next.add(s.name);
                    return next;
                  })}
                />
                <span className="flex-1 truncate">{s.title}</span>
                <span className="shrink-0 text-fg-faint text-[10.5px]">{Math.round(s.score * 100)}% 相似</span>
              </label>
            ))}
          </div>
        )}
      </Modal>
      <KnowledgeImportModal
        open={!!importFile}
        path={importFile?.path ?? ""}
        fileName={importFile?.name ?? ""}
        onClose={() => setImportFile(null)}
        onImported={loadList}
      />
    </div>
  );
}

/** Inline edit form for knowledge entry fields */
function EditForm({ form, setForm, t, similar }: {
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
