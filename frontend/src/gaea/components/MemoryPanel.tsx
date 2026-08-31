import { Plus, RefreshCw, Search, X } from "../icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { MemorySuggestion, MemorySuggestionsView, MemoryView, SkillSuggestion } from "../lib/types";
import { app } from "../lib/bridge";
import { DocEditor } from "./DocEditor";
import { useT } from "../lib/i18n";
import { useToast } from "./Toast";
import { FactCard } from "./FactCard";
import { FilterChip } from "./FilterChip";
import { TabButton } from "./TabButton";
import { EmptyState } from "./EmptyState";
import { SuggestionCard } from "./SuggestionCard";
import { ArchivesSection } from "./ArchivesSection";

export function MemoryPanel(p: {
  view: MemoryView | null;
  onClose: () => void;
  onRemember: (scope: string, note: string) => Promise<void> | void;
  onForget: (name: string) => Promise<void> | void;
  onSaveDoc: (path: string, body: string) => Promise<void> | void;
  onSaveFact: (name: string, body: string) => Promise<void> | void;
  onChangeType: (name: string, newType: string) => Promise<void> | void;
  onAcceptMemorySuggestion: (candidate: MemorySuggestion) => Promise<void> | void;
  onAcceptSkillSuggestion: (candidate: SkillSuggestion) => Promise<void> | void;
  onAcceptMergeSuggestion: (keep: string, archive: string) => Promise<void> | void;
  onRefreshSuggestions: () => Promise<MemorySuggestionsView | null>;
}) {
  const { view, onClose, onRemember, onForget, onSaveDoc, onSaveFact, onChangeType, onAcceptMemorySuggestion, onAcceptSkillSuggestion, onAcceptMergeSuggestion, onRefreshSuggestions } = p;
  const t = useT();
  const [note, setNote] = useState("");
  const [scope, setScope] = useState("");
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [highlight, setHighlight] = useState<string | null>(null);
  const [expandedFacts, setExpandedFacts] = useState<Set<string>>(new Set());
  const [tab, setTab] = useState<"facts" | "docs" | "suggestions">("facts");
  const factRefs = useRef<Record<string, HTMLElement | null>>({});
  const searchRef = useRef<HTMLInputElement>(null);
  const noteRef = useRef<HTMLInputElement>(null);
  const highlightTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const facts = useMemo(() => view?.facts ?? [], [view?.facts]);
  const docs = view?.docs ?? [];
  const archives = view?.archives ?? [];
  const [suggestions, setSuggestions] = useState<MemorySuggestionsView | null>(null);
  const [suggestionsBusy, setSuggestionsBusy] = useState(false);
  const [acceptedSuggestions, setAcceptedSuggestions] = useState<Set<string>>(new Set());
  // 记忆开关（记忆可控性）：与后端配置同步，切换后引擎重建立即生效
  const [memoryEnabled, setMemoryEnabled] = useState(view?.enabled ?? true);
  // 晨报预载开关（v4.16 刀④ UI 补齐）：work 空间新会话自动预装配高频工作记忆
  const [morningPreload, setMorningPreload] = useState(true);
  const toast = useToast();
  const scopes = useMemo(() => view?.scopes ?? [], [view?.scopes]);
  const factNames = useMemo(() => new Set(facts.map((f) => f.name)), [facts]);
  const factTypes = useMemo(
    () => Array.from(new Set(facts.map((f) => f.type).filter(Boolean))).sort(),
    [facts],
  );

  const activeScope = scope || scopes[0]?.scope || "";
  const scopePath = scopes.find((s) => s.scope === activeScope)?.path;

  // 初始化 scope
  useEffect(() => {
    if (!scope && scopes.length > 0) setScope(scopes[0].scope);
  }, [scope, scopes]);

  useEffect(() => {
    setMemoryEnabled(view?.enabled ?? true);
  }, [view?.enabled]);

  useEffect(() => {
    let alive = true;
    app.MorningPreload()
      .then((v) => { if (alive) setMorningPreload(v); })
      .catch(() => {});
    return () => { alive = false; };
  }, []);

  const toggleMemory = useCallback(() => {
    const next = !memoryEnabled;
    setMemoryEnabled(next);
    app
      .SetMemoryEnabled(next)
      .then(() => {
        toast.show(
          next
            ? "记忆已开启：画像/规则/事实将自动带入上下文"
            : "记忆已关闭：不再注入画像/规则/事实（磁盘记忆保留）",
          "info",
        );
      })
      .catch(() => setMemoryEnabled(!next));
  }, [memoryEnabled, toast]);

  const toggleMorningPreload = useCallback(() => {
    const next = !morningPreload;
    setMorningPreload(next);
    app
      .SetMorningPreload(next)
      .then(() => {
        toast.show(
          next
            ? "晨报预载已开启：新会话自动预装配高频工作记忆（work 空间）"
            : "晨报预载已关闭：新会话不再预装配晨报块",
          "info",
        );
      })
      .catch(() => setMorningPreload(!next));
  }, [morningPreload, toast]);

  const normalizedQuery = query.trim().toLowerCase();
  const filteredFacts = useMemo(
    () =>
      facts.filter((f) => {
        if (typeFilter !== "all" && f.type !== typeFilter) return false;
        if (!normalizedQuery) return true;
        return [f.title, f.name, f.description, f.type, f.body]
          .join(" ")
          .toLowerCase()
          .includes(normalizedQuery);
      }),
    [facts, normalizedQuery, typeFilter],
  );

  // 高亮闪烁（用 ref 管理定时器，避免闭包陷阱）
  const triggerHighlight = useCallback((name: string) => {
    if (highlightTimer.current) clearTimeout(highlightTimer.current);
    setHighlight(name);
    highlightTimer.current = setTimeout(() => {
      setHighlight((h) => (h === name ? null : h));
    }, 1500);
  }, []);

  const scrollToFact = useCallback((name: string) => {
    factRefs.current[name]?.scrollIntoView({ behavior: "smooth", block: "center" });
    triggerHighlight(name);
  }, [triggerHighlight]);

  const jumpTo = useCallback((name: string) => {
    setTab("facts");
    const visible = filteredFacts.some((f) => f.name === name);
    if (!visible) {
      setQuery("");
      setTypeFilter("all");
      setTimeout(() => scrollToFact(name), 0);
    } else {
      scrollToFact(name);
    }
  }, [filteredFacts, scrollToFact]);

  const toggleFact = useCallback((name: string) => {
    setExpandedFacts((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  // 稳定的 ref callback（避免每次渲染重建）
  const setFactRef = useCallback((name: string) => (el: HTMLElement | null) => {
    factRefs.current[name] = el;
  }, []);

  const submitNote = useCallback(() => {
    if (!note.trim() || busy) return;
    setBusy(true);
    Promise.resolve(onRemember(activeScope, note.trim())).finally(() => {
      setBusy(false);
      setNote("");
    });
  }, [note, busy, activeScope, onRemember]);

  const forgetFact = useCallback(
    (name: string) => {
      setBusy(true);
      Promise.resolve(onForget(name)).finally(() => setBusy(false));
    },
    [onForget],
  );

  const saveFact = useCallback(
    (name: string, body: string) => {
      setBusy(true);
      Promise.resolve(onSaveFact(name, body)).finally(() => setBusy(false));
    },
    [onSaveFact],
  );

  const changeType = useCallback(
    (name: string, newType: string) => {
      setBusy(true);
      Promise.resolve(onChangeType(name, newType)).finally(() => setBusy(false));
    },
    [onChangeType],
  );

  // 键盘快捷键（用 useCallback 避免重复注册）
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "/" && document.activeElement === document.body && tab === "facts") {
      e.preventDefault();
      searchRef.current?.focus();
      return;
    }
    if (e.ctrlKey && e.key === "n") {
      e.preventDefault();
      noteRef.current?.focus();
    }
  }, [tab]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // 清理高亮定时器
  useEffect(() => {
    return () => {
      if (highlightTimer.current) clearTimeout(highlightTimer.current);
    };
  }, []);

  return (
    <div className="drawer-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="drawer drawer--wide" onClick={(e) => e.stopPropagation()}>
        {/* ═══ 标题栏 ═══ */}
        <div className="drawer__head">
          <div>
            <div className="drawer__title">{t("memory.title")}</div>
            {view && (
              <div className="drawer__summary">
                {t("memory.summary", { facts: facts.length, docs: docs.length })}
              </div>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={toggleMemory}
              className={`inline-flex items-center gap-1.5 px-2.5 h-7 rounded-full border text-[11px] cursor-pointer transition-colors ${
                memoryEnabled
                  ? "border-accent/30 bg-accent/10 text-accent"
                  : "border-border text-fg-faint hover:text-fg"
              }`}
              title={memoryEnabled ? "点击关闭记忆注入" : "点击开启记忆注入"}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${memoryEnabled ? "bg-accent" : "bg-fg-faint/50"}`}
              />
              记忆 {memoryEnabled ? "开" : "关"}
            </button>
            <button
              type="button"
              onClick={toggleMorningPreload}
              className={`inline-flex items-center gap-1.5 px-2.5 h-7 rounded-full border text-[11px] cursor-pointer transition-colors ${
                morningPreload
                  ? "border-accent/30 bg-accent/10 text-accent"
                  : "border-border text-fg-faint hover:text-fg"
              }`}
              title={morningPreload ? "点击关闭晨报预载（新会话不再预装配晨报块）" : "点击开启晨报预载（work 空间新会话自动预装配高频工作记忆）"}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${morningPreload ? "bg-accent" : "bg-fg-faint/50"}`}
              />
              晨报预载 {morningPreload ? "开" : "关"}
            </button>
            <button className="drawer__close" onClick={onClose} aria-label={t("common.close")}>
              <X size={18} />
            </button>
          </div>
        </div>

        {/* ═══ 快速添加区 ═══ */}
        <div className="shrink-0 mx-4 mt-3 p-3 border border-border-soft rounded-xl bg-bg-elev/40">
          <div className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider mb-2">
            {t("memory.quickAdd")}
          </div>
          <div className="flex items-center gap-2">
            <select
              className="bg-bg-soft border border-border-soft rounded-lg text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent cursor-pointer"
              value={activeScope}
              onChange={(e) => setScope(e.target.value)}
              aria-label={t("memory.whereToSave")}
            >
              {scopes.map((s) => (
                <option key={s.scope} value={s.scope}>
                  {s.scope === "user" ? t("memory.scopeUser") : s.scope === "project" ? t("memory.scopeProject") : s.scope === "local" ? t("memory.scopeLocal") : s.scope}
                </option>
              ))}
            </select>
            <input
              ref={noteRef}
              className="flex-1 bg-bg-soft border border-border-soft rounded-lg text-fg text-[12px] px-3 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
              placeholder={t("memory.notePlaceholder")}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") submitNote();
              }}
              aria-label={t("memory.notePlaceholder")}
            />
            <button
              className="shrink-0 px-3 py-1.5 border-0 rounded-lg bg-accent text-accent-fg text-[12px] font-semibold cursor-pointer hover:brightness-110 active:scale-[0.97] transition-all disabled:opacity-40"
              onClick={submitNote}
              disabled={busy || !note.trim()}
              type="button"
            >
              <Plus size={13} className="inline mr-1" />
              {t("common.add")}
            </button>
          </div>
          {scopePath && (
            <div className="mt-1.5 flex items-center gap-1">
              <span className="text-fg-faint/40 text-[10px]">{t("memory.saveTo")}:</span>
              <span className="text-fg-faint/60 text-[10px] font-mono truncate" title={scopePath}>
                {scopePath}
              </span>
            </div>
          )}
        </div>

        {/* ═══ 标签栏 ═══ */}
        <div className="shrink-0 mx-4 mt-3 flex border-b border-border-soft">
          <TabButton active={tab === "facts"} onClick={() => setTab("facts")} badge={facts.length}>
            {t("memory.facts")}
          </TabButton>
          <TabButton active={tab === "docs"} onClick={() => setTab("docs")} badge={docs.length}>
            {t("memory.docs")}
          </TabButton>
          <TabButton
            active={tab === "suggestions"}
            onClick={() => setTab("suggestions")}
            badge={suggestions ? suggestions.memories.length + suggestions.skills.length + (suggestions.merges?.length ?? 0) : 0}
          >
            {t("memory.suggestions")}
          </TabButton>
        </div>

        {/* ═══ 搜索与筛选（仅事实标签） ═══ */}
        {tab === "facts" && facts.length > 0 && (
          <div className="shrink-0 mx-4 mt-2 space-y-2">
            <div className="flex items-center gap-1.5 px-3 h-8 border border-border rounded-lg bg-bg text-fg-faint focus-within:border-accent transition-colors">
              <Search size={14} />
              <input
                ref={searchRef}
                className="flex-1 min-w-0 border-0 outline-none bg-transparent text-fg text-[12.5px] placeholder:text-fg-faint"
                placeholder={t("memory.searchPlaceholder")}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                aria-label={t("memory.searchPlaceholder")}
              />
              {query && (
                <button
                  className="bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg p-0"
                  onClick={() => setQuery("")}
                  aria-label={t("memory.clearFilters")}
                >
                  <X size={12} />
                </button>
              )}
            </div>
            <div className="flex items-center gap-1.5 flex-wrap">
              <FilterChip active={typeFilter === "all"} label={t("memory.filterAll")} onClick={() => setTypeFilter("all")} />
              {factTypes.map((ft) => (
                <FilterChip key={ft} active={typeFilter === ft} label={ft} onClick={() => setTypeFilter(ft)} />
              ))}
            </div>
          </div>
        )}

        {/* ═══ 内容区 ═══ */}
        <div className="flex-1 min-h-0 overflow-auto px-4 py-3">
          {/* ── 事实标签 ── */}
          {tab === "facts" && (
            <>
              {facts.length === 0 ? (
                <EmptyState message={t("memory.noFacts")} />
              ) : filteredFacts.length === 0 ? (
                <div className="py-10 text-center text-fg-faint text-[13px]">
                  {t("memory.noResults")}
                  {(query || typeFilter !== "all") && (
                    <button
                      className="block mx-auto mt-2 text-accent text-[12px] bg-transparent border-0 cursor-pointer hover:underline"
                      onClick={() => { setQuery(""); setTypeFilter("all"); }}
                    >
                      {t("memory.clearFilters")}
                    </button>
                  )}
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  {filteredFacts.map((fact) => (
                    <div
                      key={fact.name}
                      ref={setFactRef(fact.name)}
                      className="fact-card-wrapper"
                    >
                      <FactCard
                        fact={fact}
                        factNames={factNames}
                        expanded={expandedFacts.has(fact.name)}
                        highlight={highlight === fact.name}
                        onToggle={() => toggleFact(fact.name)}
                        onJump={jumpTo}
                        onSave={saveFact}
                        onForget={() => forgetFact(fact.name)}
                        onChangeType={changeType}
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* ── 归档区 ── */}
              {archives.length > 0 && (
                <div className="mt-4 pt-3 border-t border-border-soft/50">
                  <ArchivesSection archives={archives} />
                </div>
              )}
            </>
          )}

          {/* ── 文档标签 ── */}
          {tab === "docs" && (
            <>
              {docs.length === 0 ? (
                <EmptyState message={t("memory.noDocs")} />
              ) : (
                <DocEditor docs={docs} onSaveDoc={onSaveDoc} busy={busy} />
              )}
            </>
          )}

          {/* ── 建议标签 ── */}
          {tab === "suggestions" && (
            <div className="flex flex-col gap-3">
              {/* 扫描按钮 */}
              <button
                className="flex items-center justify-center gap-2 px-4 py-2.5 border border-border-soft rounded-lg bg-bg-soft text-fg text-[12.5px] cursor-pointer hover:bg-bg hover:border-accent transition-colors disabled:opacity-40"
                onClick={async () => {
                  setSuggestionsBusy(true);
                  const result = await onRefreshSuggestions();
                  setSuggestions(result);
                  setSuggestionsBusy(false);
                }}
                disabled={suggestionsBusy}
                type="button"
              >
                <RefreshCw size={14} className={suggestionsBusy ? "animate-spin" : ""} />
                {suggestions ? t("memory.refreshSuggestions") : t("memory.scanSuggestions")}
              </button>

              {!suggestions ? (
                <EmptyState message={t("memory.suggestionsHint")} />
              ) : suggestions.memories.length === 0 && suggestions.skills.length === 0 && (suggestions.merges?.length ?? 0) === 0 ? (
                <EmptyState message={t("memory.noCandidates")} />
              ) : (
                <>
                  {/* 记忆候选项 */}
                  {suggestions.memories.length > 0 && (
                    <>
                      <div className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider">
                        {t("memory.memoryCandidates")}
                      </div>
                      {suggestions.memories.map((s) => (
                        <SuggestionCard
                          key={s.id || s.name}
                          item={s}
                          accepted={acceptedSuggestions.has(s.id || s.name)}
                          badge={t("memory.newBadge")}
                          acceptedBadge={t("memory.savedBadge")}
                          actionLabel={t("memory.accept")}
                          onAccept={async () => {
                            await onAcceptMemorySuggestion(s);
                            setAcceptedSuggestions((prev) => new Set(prev).add(s.id || s.name));
                          }}
                        />
                      ))}
                    </>
                  )}

                  {/* 技能候选项 */}
                  {suggestions.skills.length > 0 && (
                    <>
                      <div className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider mt-2">
                        {t("memory.skillCandidates")}
                      </div>
                      {suggestions.skills.map((s) => (
                        <SuggestionCard
                          key={s.id || s.name}
                          item={s}
                          accepted={acceptedSuggestions.has(s.id || s.name)}
                          badge={t("memory.newSkillBadge")}
                          acceptedBadge={t("memory.createdBadge")}
                          actionLabel={t("memory.create")}
                          onAccept={async () => {
                            await onAcceptSkillSuggestion(s);
                            setAcceptedSuggestions((prev) => new Set(prev).add(s.id || s.name));
                          }}
                        />
                      ))}
                    </>
                  )}

                  {/* 蒸馏合并候选（做梦 2.0：确定性重复记忆，归档较旧条可逆） */}
                  {(suggestions.merges?.length ?? 0) > 0 && (
                    <>
                      <div className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider mt-2">
                        {t("memory.mergeCandidates")}
                      </div>
                      {suggestions.merges!.map((m) => (
                        <div
                          key={m.id}
                          className="rounded-lg border border-border-soft bg-bg p-2.5 flex items-start justify-between gap-2"
                        >
                          <div className="min-w-0 flex-1">
                            <div className="text-[11px] text-fg font-medium truncate">
                              {m.keepTitle || m.keep}
                            </div>
                            <div className="text-[10px] text-fg-faint truncate">
                              ← {m.archiveTitle || m.archive}
                              {m.archiveUpdatedAt ? ` · ${m.archiveUpdatedAt}` : ""}
                            </div>
                            <div className="text-[10px] text-fg-faint/70 mt-0.5">{m.reason}</div>
                          </div>
                          <button
                            type="button"
                            className="shrink-0 px-2 py-1 rounded border border-border-soft text-[10px] cursor-pointer bg-bg-soft text-fg-dim hover:text-accent hover:border-accent transition-colors disabled:opacity-40"
                            disabled={acceptedSuggestions.has(m.id)}
                            onClick={async () => {
                              await onAcceptMergeSuggestion(m.keep, m.archive);
                              setAcceptedSuggestions((prev) => new Set(prev).add(m.id));
                            }}
                          >
                            {acceptedSuggestions.has(m.id) ? t("memory.savedBadge") : t("memory.mergeAction")}
                          </button>
                        </div>
                      ))}
                    </>
                  )}

                  {/* 生成时间 */}
                  {suggestions.generatedAt && (
                    <div className="text-fg-faint/40 text-[10px] text-right">
                      {t("memory.generatedAt")} {new Date(suggestions.generatedAt).toLocaleString()}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
