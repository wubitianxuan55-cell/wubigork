import { AlertCircle, BookOpen, Check, ChevronRight, Clock, CloudUpload, FileText, PanelRightOpen, Pencil, Plus, Save, ScrollText, Search, Trash2, X, X as XIcon } from "../icons";
import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { Modal } from "antd";
import type { FilePickResult, KnowledgeEntry, KnowledgeHistoryView, KnowledgeSaveRequest, KnowledgeSummary, SimilarView } from "../lib/types";
import { app } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import { EmptyState } from "./EmptyState";
import { KnowledgeImportModal } from "./memoryhub/KnowledgeImportModal";
import { Markdown } from "./Markdown";

const CATEGORIES = ["all", "规范标准", "工程案例", "经验总结", "材料工艺", "法规政策", "调查报告", "设计方案", "其他"];
const PHASES = ["all", "调查", "设计", "施工", "验收", "运维", "全程"];
const STATUSES = ["all", "现行", "已归档", "常用", "草稿"];

// 可见焦点环（v3 规范：--gaea-glow 描边）。全局样式会把 :focus-visible 的
// outline 置 none，这里用 Tailwind 工具类显式恢复，保证键盘可达。
const FOCUS_RING = "focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-[var(--gaea-glow)]";

export function KnowledgePanel(p: { onClose: () => void; variant?: "modal" | "page" }) {
  const { onClose, variant = "modal" } = p;
  const t = useT();
  const [entries, setEntries] = useState<KnowledgeSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null); // T7-4 加载三态：失败不再无限 loading/假空列表
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
  // page variant 专用：右侧「引用/来源」inspector 折叠态（modal variant 不使用）。
  // 默认折叠：避免与宿主外壳（如 memoryhub 3 分区）嵌套时竖条过密；有检索命中/选中条目时可一键展开。
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);

  // Edit form state
  const [form, setForm] = useState<KnowledgeSaveRequest>({
    name: "", title: "", category: "", phase: "", discipline: "",
    tags: [], status: "现行", version: 1, author: "", reviewer: "",
    source: "", body: "",
  });

  const loadList = useCallback(() => {
    setLoading(true);
    setListError(null);
    app.KnowledgeList().then((list) => {
      setEntries(list);
      setSelected((prev) => {
        const names = new Set(list.map((e) => e.name));
        return new Set([...prev].filter((n) => names.has(n)));
      });
      setLoading(false);
    }).catch((err) => {
      setLoading(false);
      setListError(err instanceof Error ? err.message : String(err));
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
    // T7-4：批量删除失败可见化——不再逐条静默吞错。
    const results = await Promise.all(
      [...selected].map((n) => app.KnowledgeDelete(n).then(() => null as null | unknown).catch((err: unknown) => err)),
    );
    const failed = results.filter((r): r is unknown => r !== null);
    if (failed.length > 0) setListError(`批量删除失败 ${failed.length} 条，请重试`);
    setSelected(new Set());
    loadList();
  }, [selected, loadList]);

  const batchStatus = useCallback(async (next: string) => {
    if (selected.size === 0 || !next) return;
    // T7-4：批量改状态失败可见化——保存失败不再静默。
    let failed = 0;
    for (const name of selected) {
      const e = await app.KnowledgeGet(name).catch(() => null);
      if (e) {
        const ok = await app.KnowledgeSave({ ...e, status: next }).then(() => true).catch(() => false);
        if (!ok) failed += 1;
      } else {
        failed += 1;
      }
    }
    if (failed > 0) setListError(`批量改状态失败 ${failed} 条，请重试`);
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
    setListError(null);
    try {
      const list = q.trim()
        ? await app.KnowledgeSearch(q, cat, ph, st)
        : await app.KnowledgeList();
      setEntries(list);
    } catch (err) {
      // T7-4：搜索失败不再静默保持原列表——给出可见错误，重试按钮可重新拉取。
      setListError(err instanceof Error ? err.message : String(err));
    }
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

  // 各分类条目数（page variant 左侧分类栏计数）
  const categoryCounts = useMemo(() => {
    const m = new Map<string, number>();
    for (const e of entries) m.set(e.category, (m.get(e.category) ?? 0) + 1);
    return m;
  }, [entries]);

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

  // ── page variant 渲染助手 ──────────────────────────────────────
  // 列表摘要里后端会返回 phase/source 等扩展字段（TS 类型未声明），沿用原逻辑的强转。
  const phaseOf = (e: KnowledgeSummary): string => (e as unknown as Record<string, string>).phase;
  const sourceOf = (e: KnowledgeSummary): string => (e as unknown as Record<string, string>).source;

  // 状态徽标配色：草稿=warning、已归档=中性、其余=accent（全部走令牌）。
  const statusBadgeStyle = (st: string): CSSProperties => {
    const warn = "var(--color-warning, var(--md-sys-color-warning))";
    const accent = "var(--accent, var(--md-sys-color-primary))";
    if (st === "草稿") return {
      background: `color-mix(in srgb, ${warn} 12%, transparent)`,
      color: warn,
      borderColor: `color-mix(in srgb, ${warn} 30%, transparent)`,
    };
    if (st === "已归档") return {
      background: "var(--bg-soft, var(--md-sys-color-surface-container))",
      color: "var(--fg-faint, var(--md-sys-color-text-secondary))",
      borderColor: "var(--border-soft, var(--md-sys-color-outline-variant))",
    };
    return {
      background: `color-mix(in srgb, ${accent} 12%, transparent)`,
      color: accent,
      borderColor: `color-mix(in srgb, ${accent} 30%, transparent)`,
    };
  };

  // 中区：详情 / 编辑
  const renderCenter = () => {
    if (isAdding) {
      return (
        <div className="flex-1 min-h-0 overflow-y-auto p-4">
          <EditForm form={form} setForm={setForm} t={t} similar={similar} />
          <div className="flex gap-2 mt-3 justify-end">
            <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px] cursor-pointer" onClick={handleSave} type="button"><Save size={13} aria-hidden="true" />{t("knowledge.save")}</button>
            <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px] cursor-pointer" onClick={cancelEdit} type="button">{t("common.cancel")}</button>
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
          <button className={`flex items-center gap-1 px-3 h-8 rounded-lg bg-accent text-accent-fg text-[12px] font-medium cursor-pointer hover:opacity-90 transition-opacity ${FOCUS_RING}`} onClick={startAdd} type="button"><Plus size={13} aria-hidden="true" />{t("knowledge.new")}</button>
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
              <button className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-green-600 text-white text-[12px] cursor-pointer" onClick={handleSave} type="button"><Save size={13} aria-hidden="true" />{t("knowledge.save")}</button>
              <button className="px-2.5 py-1 rounded-md bg-bg-soft text-fg text-[12px] cursor-pointer" onClick={cancelEdit} type="button">{t("common.cancel")}</button>
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
            <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => startEdit(expandedEntry)} type="button"><Pencil size={12} aria-hidden="true" />{t("common.edit")}</button>
            <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => void openHistory(expandedEntry.name)} type="button"><Clock size={12} aria-hidden="true" />版本历史</button>
            {expandedEntry.status === "草稿" && (
              <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-green-600/15 text-green-500 text-[11.5px] cursor-pointer hover:bg-green-600/25 transition-colors ${FOCUS_RING}`} onClick={() => void doReview(expandedEntry)} type="button"><Check size={12} aria-hidden="true" />审核通过</button>
            )}
            <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer hover:bg-sidebar-hover transition-colors ${FOCUS_RING}`} onClick={() => void openMerge(expandedEntry)} type="button" title="把相似条目合并进本条（标签并集、来源合并、旧条目留档删除）">合并相似…</button>
            {deleteConfirm === expandedEntry.name ? (
              <span className="flex items-center gap-1">
                <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-red-600 text-white text-[11.5px] cursor-pointer ${FOCUS_RING}`} onClick={handleDeleteConfirm} type="button"><Check size={12} aria-hidden="true" />{t("common.confirm")}</button>
                <button className={`px-2.5 py-1.5 rounded-md bg-bg-soft text-fg text-[11.5px] cursor-pointer ${FOCUS_RING}`} onClick={() => setDeleteConfirm(null)} type="button"><XIcon size={12} aria-hidden="true" /></button>
              </span>
            ) : (
              <button className={`flex items-center gap-1 px-2.5 py-1.5 rounded-md text-red-500 text-[11.5px] cursor-pointer hover:bg-red-500/10 transition-colors ${FOCUS_RING}`} onClick={() => setDeleteConfirm(expandedEntry.name)} type="button"><Trash2 size={12} aria-hidden="true" />{t("knowledge.delete")}</button>
            )}
          </div>
        </>
      );
    }
    return null;
  };

  // 左栏：单条知识条目（page variant 紧凑行；激活 = 主色容器 + 左缘光条）
  const renderEntryRow = (entry: KnowledgeSummary) => {
    const isActive = expanded === entry.name;
    const phaseVal = phaseOf(entry);
    return (
      <div key={entry.name} className="relative">
        <div
          className={`flex items-start gap-1.5 px-2 py-1.5 rounded-lg transition-colors ${isActive ? "" : "hover:bg-bg-soft"}`}
          style={isActive ? {
            background: "var(--color-primary-container, var(--md-sys-color-primary-container))",
            boxShadow: "var(--v3-glow-faint)",
          } : undefined}
        >
          {isActive && (
            <span aria-hidden="true" className="absolute left-0 top-2 bottom-2 w-[3px] rounded-r" style={{ background: "var(--gaea-glow)", boxShadow: "0 0 8px var(--gaea-glow)" }} />
          )}
          <input
            type="checkbox"
            checked={selected.has(entry.name)}
            onChange={() => toggleSelect(entry.name)}
            title="多选（批量删除/改状态）"
            aria-label={`选择 ${entry.title}`}
            className="mt-1.5 shrink-0 cursor-pointer"
          />
          <button type="button" onClick={() => void handleToggle(entry.name)} aria-expanded={isActive}
            className="flex-1 min-w-0 text-left flex flex-col gap-0.5 cursor-pointer">
            <span className="flex items-start gap-1.5">
              <span className="flex-1 text-[12.5px] font-medium leading-snug truncate text-fg">
                {normalizedQuery ? highlightText(entry.title) : entry.title}
              </span>
              <span className="shrink-0 text-[9.5px] font-medium px-1.5 py-0.5 rounded-full"
                style={{
                  background: "var(--accent-soft, color-mix(in srgb, var(--accent, var(--md-sys-color-primary)) 12%, transparent))",
                  color: "var(--accent, var(--md-sys-color-primary))",
                }}>
                {entry.category}
              </span>
            </span>
            <span className="flex items-center gap-1.5 text-[10px] text-fg-faint">
              {phaseVal && <span>{phaseVal}</span>}
              {phaseVal && <span aria-hidden="true">·</span>}
              {entry.status && <span>{entry.status}</span>}
              {entry.updatedAt && <span className="ml-auto tabular-nums">{new Date(entry.updatedAt).toLocaleDateString()}</span>}
            </span>
            {entry.tags.length > 0 && (
              <span className="flex flex-wrap gap-1">
                {entry.tags.slice(0, 3).map((tag) => (
                  <span key={tag} className="text-[9.5px] text-fg-faint px-1.5 py-0.5 rounded-full bg-bg-soft">
                    {normalizedQuery ? highlightText(tag) : tag}
                  </span>
                ))}
                {entry.tags.length > 3 && <span className="text-[9.5px] text-fg-faint">+{entry.tags.length - 3}</span>}
              </span>
            )}
          </button>
        </div>
      </div>
    );
  };

  // 左栏：列表主体（loading / 错误重试 / 空 / 无匹配 / 条目）
  const renderListBody = () => {
    if (loading) {
      return <div className="py-10 text-center text-fg-faint text-[13px]">{t("common.loading")}</div>;
    }
    if (listError) {
      // T7-4：加载失败三态——错误信息 + 重试按钮，不再无限 loading/假空列表
      return (
        <div className="py-10 text-center">
          <div className="text-red-500 text-[13px] mb-3">加载失败：{listError}</div>
          <button className={`px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 cursor-pointer ${FOCUS_RING}`} onClick={() => void loadList()} type="button">重试</button>
        </div>
      );
    }
    if (entries.length === 0) {
      return <EmptyState message={t("knowledge.empty")} />;
    }
    if (filtered.length === 0 && !isAdding) {
      return (
        <div className="py-10 text-center text-fg-faint text-[13px]">
          {t("knowledge.noMatch")}
          {(query || category !== "all" || phase !== "all" || status !== "all") && (
            <button className="block mx-auto mt-2 text-accent text-[12px] bg-transparent border-0 cursor-pointer hover:underline" onClick={() => { setQuery(""); setCategory("all"); setPhase("all"); setStatus("all"); }} type="button">
              {t("memory.clearFilters")}
            </button>
          )}
        </div>
      );
    }
    return (
      <div className="flex flex-col gap-1 pt-1.5">
        {filtered.map(renderEntryRow)}
      </div>
    );
  };

  // 右栏：检索结果 / 引用 inspector
  const renderInspector = () => {
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
                    <button type="button" onClick={() => void handleToggle(h.name)} aria-expanded={expanded === h.name}
                      className={`w-full text-left px-2 py-1.5 rounded-md transition-colors cursor-pointer ${expanded === h.name ? "bg-bg-soft" : "hover:bg-bg-soft"} ${FOCUS_RING}`}>
                      <span className="block text-[11.5px] text-fg truncate">{highlightText(h.title)}</span>
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
  };

  const rootCls = variant === "page"
    ? "w-full h-full flex flex-col bg-bg"
    : "fixed inset-0 z-50 flex items-start justify-center pt-[64px] pb-8";
  return (
    <div className={rootCls} style={variant === "modal" ? { background: "var(--ds-overlay)" } : undefined}>
      {variant === "page" ? (
        /* ══ 3 分区工作台：左=分类/列表 · 中=搜索+详情/编辑 · 右=引用 inspector ══ */
        <div className="w-full h-full flex flex-col min-h-0">
          <div className="v3-zone-row flex-1 min-h-0 gap-3 p-3 overflow-hidden">
            {/* ── 左：知识分类 / 条目列表 ── */}
            <aside className="v3-panel v3-rise v3-rise-1 w-[272px] shrink-0 flex flex-col min-h-0 overflow-hidden" aria-label="知识分类与条目列表">
              <div className="v3-panel-head">
                <BookOpen size={14} className="text-accent" aria-hidden="true" />
                <span className="v3-panel-title">{t("knowledge.title")}</span>
                <span className="v3-panel-spacer" />
                <span className="text-[10.5px] text-fg-faint tabular-nums">{t("knowledge.count", { n: filtered.length })}</span>
              </div>
              {/* 工具栏：新建 / 导入 / 导出 */}
              <div className="shrink-0 flex items-center gap-1.5 px-3 py-2 border-b border-border-soft/60">
                <button type="button" onClick={startAdd}
                  className={`flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[12px] font-medium cursor-pointer hover:opacity-90 transition-opacity ${FOCUS_RING}`}>
                  <Plus size={13} aria-hidden="true" />{t("knowledge.new")}
                </button>
                <button type="button" onClick={() => void pickImport()} title="导入 md/txt/docx/pdf/xlsx/csv" aria-label="导入 md/txt/docx/pdf/xlsx/csv"
                  className={`inline-flex items-center justify-center w-7 h-7 rounded-lg text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors ${FOCUS_RING}`}>
                  <CloudUpload size={14} aria-hidden="true" />
                </button>
                <button type="button" onClick={() => void doExport()} title="批量导出为 Markdown" aria-label="批量导出为 Markdown"
                  className={`inline-flex items-center justify-center w-7 h-7 rounded-lg text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors ${FOCUS_RING}`}>
                  <Save size={14} aria-hidden="true" />
                </button>
              </div>
              {/* 批量操作条 */}
              {selected.size > 0 && (
                <div className="shrink-0 flex items-center gap-1.5 px-3 py-2 border-b border-border-soft/60">
                  <span className="text-[11.5px] text-accent shrink-0">已选 {selected.size}</span>
                  <select
                    value=""
                    onChange={(e) => { if (e.target.value) void batchStatus(e.target.value); }}
                    className={`px-1.5 h-6 rounded-md bg-bg-soft text-fg-dim text-[11px] border border-border outline-none ${FOCUS_RING}`}
                    aria-label="批量改状态"
                  >
                    <option value="" disabled>改状态…</option>
                    {["现行", "草稿", "常用", "已归档"].map((s) => <option key={s} value={s}>{s}</option>)}
                  </select>
                  <button className={`px-2 h-6 rounded-md bg-red-500/15 text-red-400 text-[11px] cursor-pointer hover:bg-red-500/25 ${FOCUS_RING}`} onClick={() => void batchDelete()} type="button">批量删除</button>
                </div>
              )}
              {/* 分类筛选（竖向，激活 = 主色容器 + 左缘光条） */}
              <div className="shrink-0 px-2.5 py-1.5 flex flex-col gap-0.5 border-b border-border-soft/60">
                {CATEGORIES.map((cat) => {
                  const active = category === cat;
                  const count = cat === "all" ? entries.length : (categoryCounts.get(cat) ?? 0);
                  return (
                    <button key={cat} type="button" onClick={() => setCategory(cat)} aria-pressed={active}
                      className={`relative w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-[12px] transition-colors cursor-pointer text-left ${FOCUS_RING}`}
                      style={active ? {
                        background: "var(--color-primary-container, var(--md-sys-color-primary-container))",
                        boxShadow: "var(--v3-glow-faint)",
                      } : undefined}>
                      {active && (
                        <span aria-hidden="true" className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r" style={{ background: "var(--gaea-glow)", boxShadow: "0 0 8px var(--gaea-glow)" }} />
                      )}
                      <span className={`flex-1 truncate ${active ? "text-fg" : "text-fg-dim hover:text-fg"}`}>{cat === "all" ? t("knowledge.all") : cat}</span>
                      <span className="text-[10px] tabular-nums opacity-70">{count}</span>
                    </button>
                  );
                })}
              </div>
              {/* 阶段 / 状态筛选 */}
              <div className="shrink-0 px-3 py-2 flex flex-wrap gap-1 items-center border-b border-border-soft/60">
                {PHASES.map((ph) => {
                  const active = phase === ph;
                  return (
                    <button key={ph} type="button" onClick={() => setPhase(ph)} aria-pressed={active}
                      className={`px-2 py-0.5 rounded-full text-[10.5px] border cursor-pointer transition-colors ${FOCUS_RING} ${active ? "bg-accent/20 text-accent border-accent/40" : "bg-transparent text-fg-faint border-border-soft hover:border-accent/30 hover:text-fg"}`}>
                      {ph === "all" ? t("knowledge.all") : ph}
                    </button>
                  );
                })}
                <span className="w-px h-3.5 bg-border-soft mx-0.5" aria-hidden="true" />
                {STATUSES.map((st) => {
                  const active = status === st;
                  return (
                    <button key={st} type="button" onClick={() => setStatus(st)} aria-pressed={active}
                      className={`px-2 py-0.5 rounded-full text-[10.5px] border cursor-pointer transition-colors ${FOCUS_RING} ${active ? "bg-accent/20 text-accent border-accent/40" : "bg-transparent text-fg-faint border-border-soft hover:border-accent/30 hover:text-fg"}`}>
                      {st === "all" ? t("knowledge.all") : st}
                    </button>
                  );
                })}
              </div>
              {/* 条目列表 */}
              <div className="flex-1 min-h-0 overflow-y-auto px-2.5 pb-2 pt-1.5">
                {renderListBody()}
              </div>
              {/* 状态提示 */}
              {(exportMsg || mergeMsg) && (
                <div className="shrink-0 px-3 pb-2 flex flex-col gap-0.5">
                  {exportMsg && <div className="text-[10.5px] text-accent">{exportMsg}</div>}
                  {mergeMsg && <div className="text-[10.5px] text-amber-400">{mergeMsg}</div>}
                </div>
              )}
            </aside>

            {/* ── 中：搜索 + 详情 / 编辑 ── */}
            <section className="v3-zone v3-rise v3-rise-2 flex-1 min-w-0 min-h-0 gap-3" aria-label="知识详情与编辑">
              {/* 搜索框：聚焦光晕按 v3 规范（--v3-glow-soft） */}
              <div className={`shrink-0 flex items-center gap-2 px-3 h-9 rounded-lg border border-border-soft bg-bg-soft/70 transition-all duration-200 focus-within:border-[color-mix(in_srgb,var(--gaea-glow)_40%,transparent)] focus-within:[box-shadow:var(--v3-glow-soft)]`}>
                <Search size={14} className="text-fg-faint shrink-0" aria-hidden="true" />
                <input ref={searchRef} className="flex-1 min-w-0 border-0 outline-none bg-transparent text-fg text-[12.5px] placeholder:text-fg-faint" placeholder={t("knowledge.search")} value={query} onChange={(e) => setQuery(e.target.value)} aria-label={t("knowledge.search")} />
                {query && <button className="bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg p-0" onClick={() => setQuery("")} type="button" aria-label="清空搜索"><X size={12} /></button>}
              </div>
              {/* 详情 / 编辑卡 */}
              <div className="v3-card flex-1 min-h-0 flex flex-col overflow-hidden">
                {renderCenter()}
              </div>
            </section>

            {/* ── 右：检索结果 / 引用 inspector（可折叠） ── */}
            {inspectorOpen ? (
              <aside className="v3-panel v3-rise v3-rise-3 w-[280px] shrink-0 flex flex-col min-h-0 overflow-hidden" aria-label="检索结果与引用">
                <div className="v3-panel-head">
                  <ScrollText size={14} className="text-accent" aria-hidden="true" />
                  <span className="v3-panel-title">引用 / 来源</span>
                  <span className="v3-panel-spacer" />
                  <button type="button" onClick={() => setInspectorOpen(false)} aria-expanded={inspectorOpen} aria-label="折叠引用面板" title="折叠引用面板"
                    className={`inline-flex items-center justify-center w-6 h-6 rounded-md text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors ${FOCUS_RING}`}>
                    <ChevronRight size={13} aria-hidden="true" />
                  </button>
                </div>
                <div className="flex-1 min-h-0 overflow-y-auto px-3 py-2.5 space-y-4">
                  {renderInspector()}
                </div>
              </aside>
            ) : (
              <button type="button" onClick={() => setInspectorOpen(true)} aria-expanded={false} aria-label="展开引用面板" title="展开引用面板"
                className={`v3-rise v3-rise-3 self-start shrink-0 mt-2 inline-flex flex-col items-center gap-1.5 px-1.5 py-2.5 rounded-lg border border-border-soft bg-bg-soft/70 text-fg-faint cursor-pointer hover:text-fg hover:border-accent/40 transition-colors ${FOCUS_RING}`}>
                <PanelRightOpen size={14} aria-hidden="true" />
                <span className="text-[9.5px]" style={{ writingMode: "vertical-rl" }}>引用</span>
              </button>
            )}
          </div>
        </div>
      ) : (
        /* ══ modal variant：原布局保持不变（仅视觉类，无行为改动） ══ */
        <div className="relative w-full max-w-[620px] max-h-full flex flex-col rounded-xl border border-border-soft bg-bg shadow-xl overflow-hidden">
          <div className="flex items-center gap-2 px-5 py-3.5 border-b border-border-soft shrink-0">
            <BookOpen size={17} className="text-accent" />
            <h2 className="flex-1 text-fg font-semibold text-[14px]">{t("knowledge.title")}</h2>
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
            ) : listError ? (
              // T7-4：加载失败三态——错误信息 + 重试按钮，不再无限 loading/假空列表
              <div className="py-10 text-center">
                <div className="text-red-500 text-[13px] mb-3">加载失败：{listError}</div>
                <button
                  className="px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 cursor-pointer"
                  onClick={() => void loadList()}
                  type="button"
                >
                  重试
                </button>
              </div>
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
      )}
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
