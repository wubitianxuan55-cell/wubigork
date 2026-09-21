import { Plus, RefreshCw, Search, X, Brain } from "../icons";
import "./context/context-view.css";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  MemorySuggestion, MemorySuggestionsView, MemoryView, SkillDistillView,
  SkillRecordResult, SkillSuggestion,
} from "../lib/types";
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
import { SkillDistillSection } from "./SkillDistillSection";
import { SkillRecordModal } from "./SkillRecordModal";

// 流程蒸馏未接线 onIgnore 时的稳定空回调（避免每次渲染重建打断 memo）。
const noopDistill = async (): Promise<void> => {};

export function MemoryPanel(p: {
  view: MemoryView | null;
  onRemember: (scope: string, note: string) => Promise<void> | void;
  onForget: (name: string) => Promise<void> | void;
  onSaveDoc: (path: string, body: string) => Promise<void> | void;
  onSaveFact: (name: string, body: string) => Promise<void> | void;
  onChangeType: (name: string, newType: string) => Promise<void> | void;
  onAcceptMemorySuggestion: (candidate: MemorySuggestion) => Promise<void> | void;
  onAcceptSkillSuggestion: (candidate: SkillSuggestion) => Promise<void> | void;
  onAcceptMergeSuggestion: (keep: string, archive: string) => Promise<void> | void;
  onRefreshSuggestions: () => Promise<MemorySuggestionsView | null>;
  // ── 流程蒸馏（7.2-2）：全部可选，缺省时分区整体隐藏（既有调用方零改动）。
  // onDraftDistill 返回蒸馏结果时，本面板本地打开 SkillRecordModal(preload)
  // 复用 7.2-1 审阅管道；保存成功经 onCrystallizeDistill 落 crystallized 审计。
  distillView?: SkillDistillView | null;
  distillLoading?: boolean;
  onRefreshDistill?: () => void;
  onDraftDistill?: (id: string) => Promise<SkillRecordResult | null | void> | SkillRecordResult | null | void;
  onIgnoreDistill?: (id: string) => Promise<void> | void;
  onCrystallizeDistill?: (patternId: string, skillName: string) => Promise<void> | void;
}) {
  const {
    view, onRemember, onForget, onSaveDoc, onSaveFact, onChangeType,
    onAcceptMemorySuggestion, onAcceptSkillSuggestion, onAcceptMergeSuggestion,
    onRefreshSuggestions, distillView, distillLoading, onRefreshDistill,
    onDraftDistill, onIgnoreDistill, onCrystallizeDistill,
  } = p;
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
  // 流程蒸馏（7.2-2）：容器层本地托管的 preload 审阅弹窗（draft 结果 +
  // 对应候选 id；保存成功经 onCrystallizeDistill 落审计后关闭）。
  const [distillDraft, setDistillDraft] = useState<{ id: string; result: SkillRecordResult } | null>(null);
  // 记忆开关（记忆可控性）：与后端配置同步，切换后引擎重建立即生效
  const [memoryEnabled, setMemoryEnabled] = useState(view?.enabled ?? true);
  // 自动做梦模式（v4.377 建议制）：off=不整理 | suggest=提炼进待确认建议
  //（默认，接受才入库）| auto=旧直写行为。循环切换，直接调绑定持久化。
  const [dreamMode, setDreamMode] = useState(view?.dreamMode ?? "suggest");
  useEffect(() => {
    setDreamMode(view?.dreamMode ?? "suggest");
  }, [view?.dreamMode]);
  // 晨报预载开关（v4.16 刀④ UI 补齐）：work 空间新会话自动预装配高频工作记忆
  const [morningPreload, setMorningPreload] = useState(true);
  // 项目本体注入开关（6.2）：work 空间新会话注入固化/项目决策带引用块
  const [projectBrief, setProjectBrief] = useState(true);
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

  // v4.377 建议制：待确认建议的忽略集合（丢弃即出队，不写库）；两步式清污
  // 状态（idle → confirm(名单) → 执行）。
  const [dismissedSuggestions, setDismissedSuggestions] = useState<Set<string>>(new Set());
  const [purgeState, setPurgeState] = useState<{ stage: "idle" } | { stage: "confirm"; names: string[] }>({ stage: "idle" });
  // 可见记忆建议（忽略后即从计数/列表移除，与技能/合并候选计数同口径）。
  const visibleMemories = useMemo(
    () => (suggestions?.memories ?? []).filter((s) => !dismissedSuggestions.has(s.id || s.name)),
    [suggestions, dismissedSuggestions],
  );

  useEffect(() => {
    let alive = true;
    app.MorningPreload()
      .then((v) => { if (alive) setMorningPreload(v); })
      .catch(() => {});
    app.MemoryBrief()
      .then((v) => { if (alive) setProjectBrief(v); })
      .catch(() => {});
    return () => { alive = false; };
  }, []);

  // 流程蒸馏首拉（7.2-2 收口）：建议 tab 打开时自动取一次候选（只读零 LLM、
  // 静默失败）——fetch 期间分区显示 loading 而非「不可用」；失败可走扫描按钮。
  const distillFetchedRef = useRef(false);
  useEffect(() => {
    if (tab !== "suggestions" || distillFetchedRef.current || !onRefreshDistill) return;
    distillFetchedRef.current = true;
    onRefreshDistill();
  }, [tab, onRefreshDistill]);

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
      .catch(() => {
        // v4.361：失败回滚可见化——开关静默弹回像 UI 抽风。
        setMemoryEnabled(!next);
        toast.show("设置失败，已还原", "error");
      });
  }, [memoryEnabled, toast]);

  // 自动做梦三态循环（v4.377 建议制）：关 → 建议 → 直写 → 关。
  const dreamModeLabel = dreamMode === "off" ? "关" : dreamMode === "auto" ? "直写" : "建议";
  const cycleDreamMode = useCallback(() => {
    const next = dreamMode === "off" ? "suggest" : dreamMode === "suggest" ? "auto" : "off";
    const prev = dreamMode;
    setDreamMode(next);
    app
      .SetDreamMode(next)
      .then(() => {
        toast.show(
          next === "off"
            ? "自动做梦已关闭：轮次结束后不再自动整理记忆"
            : next === "auto"
              ? "自动做梦直写：提炼结果直接写入记忆（旧行为，不再逐条确认）"
              : "自动做梦建议制：提炼结果进入「建议」待确认，接受才入库",
          "info",
        );
      })
      .catch(() => {
        setDreamMode(prev);
        toast.show("设置失败，已还原", "error");
      });
  }, [dreamMode, toast]);

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
      .catch(() => {
        setMorningPreload(!next);
        toast.show("设置失败，已还原", "error");
      });
  }, [morningPreload, toast]);

  const toggleProjectBrief = useCallback(() => {
    const next = !projectBrief;
    setProjectBrief(next);
    app
      .SetMemoryBrief(next)
      .then(() => {
        toast.show(
          next
            ? "项目本体已开启：新会话注入固化/项目决策事实（带引用键）"
            : "项目本体已关闭：新会话不再注入项目事实块",
          "info",
        );
      })
      .catch(() => {
        setProjectBrief(!next);
        toast.show("设置失败，已还原", "error");
      });
  }, [projectBrief, toast]);

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

  // 流程蒸馏（7.2-2）：未传任何 distill prop 的旧调用方不渲染分区；点
  // 「结晶为技能」→ onDraftDistill 拉蒸馏草稿 → 本地以 preload 打开 7.2-1
  // 审阅弹窗（LLM 只蒸馏一次，审阅编辑→保存走既有 SkillDraftSave 通道）。
  const distillWired = distillView !== undefined || onRefreshDistill !== undefined || onDraftDistill !== undefined;
  const handleDraftDistill = useCallback(
    async (id: string) => {
      if (!onDraftDistill) return;
      const res = await onDraftDistill(id);
      if (res) setDistillDraft({ id, result: res });
    },
    [onDraftDistill],
  );

  // 清污（v4.377）：两步式——先拉预览（auto_dream 审计 ∩ 现存事实），再点
  // 一次执行批量删除；名单空直接告知。删除的是旧直写路径写入的事实，
  // 用户显式接受过的（source=explicit）不在交集内。
  const handlePurgeAutoDream = useCallback(async () => {
    if (purgeState.stage === "confirm") {
      const names = purgeState.names;
      setPurgeState({ stage: "idle" });
      try {
        const n = await app.DreamPurge(names);
        toast.show(`已删除 ${n} 条自动写入的记忆`, "info");
      } catch {
        toast.show("清理失败", "error");
      }
      return;
    }
    try {
      const names = await app.DreamPurgePreview();
      if (!names || names.length === 0) {
        toast.show("没有可清理的自动写入记忆", "info");
        return;
      }
      setPurgeState({ stage: "confirm", names });
    } catch {
      toast.show("读取清理清单失败", "error");
    }
  }, [purgeState, toast]);

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
    <div className="flex h-full min-h-0 flex-col gap-3 overflow-y-auto p-3" data-testid="memory-view">
      {/* ═══ 头部卡：标题 + 记忆/晨报开关 ═══ */}
      <section className="ctx-card flex flex-wrap items-center justify-between gap-3 p-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="ctx-head-ic" aria-hidden>
            <Brain size={15} />
          </span>
          <div className="min-w-0">
            <div className="truncate text-[15px] font-semibold leading-tight text-fg">{t("memory.title")}</div>
            {view && (
              <div className="mt-0.5 text-[11px] text-fg-faint">
                {t("memory.summary", { facts: facts.length, docs: docs.length })}
              </div>
            )}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
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
            <button
              type="button"
              onClick={toggleProjectBrief}
              className={`inline-flex items-center gap-1.5 px-2.5 h-7 rounded-full border text-[11px] cursor-pointer transition-colors ${
                projectBrief
                  ? "border-accent/30 bg-accent/10 text-accent"
                  : "border-border text-fg-faint hover:text-fg"
              }`}
              title={projectBrief ? "点击关闭项目本体（新会话不再注入项目事实块）" : "点击开启项目本体（work 空间新会话注入固化/项目决策，带 [MEM:] 引用）"}
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${projectBrief ? "bg-accent" : "bg-fg-faint/50"}`}
              />
              项目本体 {projectBrief ? "开" : "关"}
            </button>
            <button
              type="button"
              data-testid="memory-dream-mode"
              onClick={cycleDreamMode}
              className={`inline-flex items-center gap-1.5 px-2.5 h-7 rounded-full border text-[11px] cursor-pointer transition-colors ${
                dreamMode === "off"
                  ? "border-border text-fg-faint hover:text-fg"
                  : dreamMode === "auto"
                    ? "border-amber-500/40 bg-amber-500/10 text-amber-500"
                    : "border-accent/30 bg-accent/10 text-accent"
              }`}
              title="自动做梦（轮次后记忆整理）：关 → 建议（提炼进「建议」待确认，接受才入库）→ 直写（旧行为，直接写入记忆）→ 关"
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${
                  dreamMode === "off" ? "bg-fg-faint/50" : dreamMode === "auto" ? "bg-amber-500" : "bg-accent"
                }`}
              />
              自动做梦 {dreamModeLabel}
            </button>
        </div>
      </section>

      {/* ═══ 三枚统计小卡：事实 / 文档 / 建议 一眼可读 ═══ */}
      <div className="grid grid-cols-3 gap-2">
        <div className="ctx-tile" data-testid="memory-kpi-facts">
          <span className="truncate text-[10px] font-medium leading-none text-fg-faint">{t("memory.facts")}</span>
          <span className="mt-0.5 truncate font-mono text-[16px] font-semibold leading-tight tabular-nums text-fg">{facts.length}</span>
        </div>
        <div className="ctx-tile" data-testid="memory-kpi-docs">
          <span className="truncate text-[10px] font-medium leading-none text-fg-faint">{t("memory.docs")}</span>
          <span className="mt-0.5 truncate font-mono text-[16px] font-semibold leading-tight tabular-nums text-fg">{docs.length}</span>
        </div>
        <div className="ctx-tile" data-testid="memory-kpi-suggestions">
          <span className="truncate text-[10px] font-medium leading-none text-fg-faint">{t("memory.suggestions")}</span>
          <span className="mt-0.5 truncate font-mono text-[16px] font-semibold leading-tight tabular-nums text-fg">
            {suggestions ? visibleMemories.length + suggestions.skills.length + (suggestions.merges?.length ?? 0) : 0}
          </span>
        </div>
      </div>

      {/* ═══ 快速添加卡 ═══ */}
      <section className="ctx-card p-3">
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
      </section>

      {/* ═══ 视图大卡：分段 + 搜索 + 内容同卡 ═══ */}
      <section className="ctx-card flex flex-col gap-2 p-2.5">
        <div className="flex items-center gap-1">
          <TabButton active={tab === "facts"} onClick={() => setTab("facts")} badge={facts.length}>
            {t("memory.facts")}
          </TabButton>
          <TabButton active={tab === "docs"} onClick={() => setTab("docs")} badge={docs.length}>
            {t("memory.docs")}
          </TabButton>
          <TabButton
            active={tab === "suggestions"}
            onClick={() => setTab("suggestions")}
            badge={suggestions ? visibleMemories.length + suggestions.skills.length + (suggestions.merges?.length ?? 0) : 0}
          >
            {t("memory.suggestions")}
          </TabButton>
        </div>

        {/* ═══ 搜索与筛选（仅事实标签） ═══ */}
        {tab === "facts" && facts.length > 0 && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-1.5 px-3 h-7 rounded-md border border-border-soft bg-bg-soft/60 text-fg-faint focus-within:border-accent focus-within:bg-bg-soft transition-colors">
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

        {/* ═══ 内容区（外层滚动收敛到记忆页根容器） ═══ */}
        <div className="flex flex-col gap-2">
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
                <section className="ctx-card p-3">
                  <ArchivesSection archives={archives} />
                </section>
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
                  // 流程蒸馏（7.2-2）：扫描/刷新按钮同时触发候选复算（各自
                  // 独立 loading，不互相阻塞；未接线时为 no-op）。
                  void onRefreshDistill?.();
                  setSuggestionsBusy(true);
                  const result = await onRefreshSuggestions();
                  // v4.354：Go nil slice 序列化为 JSON null（suggestSkillsFromMemories
                  // 在规则类记忆 <2 条时 return nil，skills 无 omitempty）——null.length
                  // 此前整页崩，收窄归一在此统一修，消费侧五处不再逐个防御。
                  if (result) {
                    result.skills ??= [];
                    result.memories ??= [];
                    result.merges ??= [];
                  }
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
              ) : visibleMemories.length === 0 && suggestions.skills.length === 0 && (suggestions.merges?.length ?? 0) === 0 ? (
                <EmptyState message={t("memory.noCandidates")} />
              ) : (
                <>
                  {/* 记忆候选项（v4.377：含自动做梦待确认建议；可忽略=出队不写库） */}
                  {visibleMemories.length > 0 && (
                    <>
                      <div className="text-fg-faint text-[10px] font-semibold uppercase tracking-wider">
                        {t("memory.memoryCandidates")}
                      </div>
                      {visibleMemories.map((s) => (
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
                          {...(s.id && s.id.startsWith("d")
                            ? {
                                onIgnore: async () => {
                                  await app.DismissMemorySuggestion(s.id);
                                  setDismissedSuggestions((prev) => new Set(prev).add(s.id || s.name));
                                },
                                ignoreLabel: t("memory.ignore"),
                              }
                            : {})}
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

              {/* 清污（v4.377）：两步确认批量删除 auto_dream 直写事实。
                  恒显示（不依赖建议列表非空）——清理污染不要求先有候选。 */}
              <button
                type="button"
                data-testid="memory-purge-auto-dream"
                className={`flex items-center justify-center gap-2 px-4 py-2 border rounded-lg text-[12px] cursor-pointer transition-colors ${
                  purgeState.stage === "confirm"
                    ? "border-red-400/60 bg-red-500/10 text-red-400 hover:bg-red-500/20"
                    : "border-border-soft bg-bg-soft text-fg-faint hover:text-fg hover:border-fg-faint"
                }`}
                onClick={handlePurgeAutoDream}
              >
                {purgeState.stage === "confirm"
                  ? t("memory.purgeConfirm", { n: purgeState.names.length })
                  : t("memory.purgeAutoDream")}
              </button>
              {purgeState.stage === "confirm" && purgeState.names.length > 0 && (
                <div className="text-fg-faint/60 text-[10px] leading-relaxed border-l-2 border-red-400/30 pl-2 break-all">
                  {purgeState.names.slice(0, 8).join("、")}
                  {purgeState.names.length > 8 ? ` …（共 ${purgeState.names.length} 条）` : ""}
                </div>
              )}

              {/* ── 流程蒸馏分区（7.2-2）：未接线（旧调用方/既有测试）时整体隐藏 ── */}
              {distillWired && (
                <SkillDistillSection
                  view={distillView ?? null}
                  loading={distillLoading}
                  onDraft={handleDraftDistill}
                  onIgnore={onIgnoreDistill ?? noopDistill}
                />
              )}
            </div>
          )}
        </div>
      </section>

      {/* 流程蒸馏 preload 审阅弹窗（7.2-2，复用 7.2-1 管道；保存成功后
          onCrystallizeDistill 落 crystallized 审计并刷新候选） */}
      {distillDraft && (
        <SkillRecordModal
          open
          preload={distillDraft.result}
          onClose={() => setDistillDraft(null)}
          onSaved={(skillName) => {
            const patternId = distillDraft.id;
            setDistillDraft(null);
            void onCrystallizeDistill?.(patternId, skillName);
          }}
        />
      )}
    </div>
  );
}
