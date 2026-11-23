// ProposalLibraryPage — 一级板块「方案库」（与造价数据库并列）。
//
// 定位：历史方案的档案架。录入与索引只在办公板块的「方案」运行模式里由模型完成
// （proposal_save：模型定标题层级，正文由工具从源文件逐字切出）；这里只给人看：
//   方案清单 / 检索命中 → 1~4 级标题树（定位）→ 小节正文（阅读）。
// 检索与模型用的 proposal_search 同一条链：全文 + 本地语义召回 + 本地精排，
// 命中回到小节并带命中位置摘要。
//
// 四态分明：加载 / 空库（三步引导）/ 无命中 / 读取失败（不伪装成空库）。
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Modal } from "antd";
import {
  AlertCircle, BookOpen, ChevronDown, ChevronRight, Copy, FolderOpen, Inbox, Loader2, RefreshCw, Search, Trash2, X,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import { copyToClipboard } from "../gaea/lib/clipboard";
import type {
  ProposalHit, ProposalSectionBody, ProposalSectionNode, ProposalSummary,
} from "../gaea/lib/types/proposal";
import { MarkdownContent, mdStyles } from "../components/MarkdownContent";
import {
  expandAncestors, formatChars, hasChildren, indexLabel, initialCollapsed, parentOf, searchSummary,
  splitSnippet, stepRow, visibleRows,
} from "./proposal-library/logic";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "./proposal-library.css";

type Phase = "idle" | "loading" | "ready" | "failed";
type Pane = "list" | "tree" | "reader";
type Scope = "all" | "title";

interface SearchState {
  q: string;
  phase: Phase;
  mode: string;
  hits: ProposalHit[];
  note: string;
  error: string;
}

const PAGE_SEARCH_LIMIT = 30;
const NARROW_WIDTH = 820;
const LEVEL_OPTIONS = [1, 2, 3, 4] as const;
const SAMPLE_PROMPT = "把这份历史方案录入方案库，按标题层级分好：D:\\新工作\\项目目录\\技术标.docx";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

function openOfficeBoard(): void {
  window.dispatchEvent(new CustomEvent("navigate", { detail: { page: "gaea" } }));
}

function StateNote({ icon, title, children, action }: {
  icon: "loading" | "empty" | "failed";
  title: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="pl-state" role={icon === "loading" ? "status" : undefined}>
      {icon === "loading" && <Loader2 size={18} className="pl-spin" aria-hidden />}
      {icon === "empty" && <Inbox size={22} aria-hidden />}
      {icon === "failed" && <AlertCircle size={22} aria-hidden />}
      <span className="pl-state-title">{title}</span>
      {children && <span>{children}</span>}
      {action && <div className="pl-state-action">{action}</div>}
    </div>
  );
}

function Snippet({ text }: { text: string }) {
  return (
    <div className="pl-hit-snippet">
      {splitSnippet(text).map((p, i) => (p.hit ? <mark key={i}>{p.text}</mark> : <span key={i}>{p.text}</span>))}
    </div>
  );
}

export default function ProposalLibraryPage() {
  const rootRef = useRef<HTMLDivElement>(null);
  const treeRef = useRef<HTMLUListElement>(null);
  const outlineSeq = useRef(0);
  const readSeq = useRef(0);
  const searchSeq = useRef(0);

  const [narrow, setNarrow] = useState(false);
  const [pane, setPane] = useState<Pane>("tree");

  const [proposals, setProposals] = useState<ProposalSummary[]>([]);
  const [listPhase, setListPhase] = useState<Phase>("loading");
  const [listError, setListError] = useState("");

  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [outline, setOutline] = useState<ProposalSectionNode[]>([]);
  const [outlinePhase, setOutlinePhase] = useState<Phase>("idle");
  const [outlineError, setOutlineError] = useState("");
  const [collapsed, setCollapsed] = useState<Set<number>>(new Set());

  const [currentId, setCurrentId] = useState<number | null>(null);
  const [section, setSection] = useState<ProposalSectionBody | null>(null);
  const [readPhase, setReadPhase] = useState<Phase>("idle");
  const [readError, setReadError] = useState("");

  const [input, setInput] = useState("");
  const [scope, setScope] = useState<Scope>("all");
  const [typeFilter, setTypeFilter] = useState("");
  const [yearFilter, setYearFilter] = useState("");
  const [levelFilter, setLevelFilter] = useState(0);
  const [search, setSearch] = useState<SearchState | null>(null);

  const [deleteTarget, setDeleteTarget] = useState<ProposalSummary | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [copyState, setCopyState] = useState<"idle" | "done" | "failed">("idle");

  // 窄窗（三栏放不下）改单栏 + 页签切换：按容器宽度判定，不看视口。
  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const apply = () => {
      const w = el.getBoundingClientRect().width;
      setNarrow(w > 0 && w <= NARROW_WIDTH);
    };
    apply();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const loadList = useCallback(async () => {
    setListPhase("loading");
    setListError("");
    try {
      const list = (await app.GaeaProposalList()) ?? [];
      setProposals(list);
      setListPhase("ready");
    } catch (e) {
      setListError(message(e));
      setListPhase("failed");
    }
  }, []);

  const openSection = useCallback(async (id: number) => {
    setCurrentId(id);
    setReadPhase("loading");
    setReadError("");
    const seq = ++readSeq.current;
    try {
      const body = await app.GaeaProposalRead(id);
      if (seq !== readSeq.current) return;
      setSection(body);
      setReadPhase("ready");
    } catch (e) {
      if (seq !== readSeq.current) return;
      setReadError(message(e));
      setReadPhase("failed");
    }
  }, []);

  // 选中一份方案：取标题树；focusId 给了就定位到它（检索命中跳转），否则落在首个节点。
  // 切换时先清空阅读窗——不让上一份方案的小节挂在新方案的标题树旁边。
  const selectProposal = useCallback(async (name: string, focusId?: number) => {
    setSelectedName(name);
    readSeq.current++;
    setCurrentId(null);
    setSection(null);
    setReadPhase("idle");
    setOutlinePhase("loading");
    setOutlineError("");
    const seq = ++outlineSeq.current;
    try {
      const tree = (await app.GaeaProposalOutline(name)) ?? [];
      if (seq !== outlineSeq.current) return;
      let next = initialCollapsed(tree);
      if (focusId != null && tree.some((n) => n.id === focusId)) next = expandAncestors(next, tree, focusId);
      setOutline(tree);
      setCollapsed(next);
      setOutlinePhase("ready");
      const target = focusId != null && tree.some((n) => n.id === focusId) ? focusId : (tree[0]?.id ?? null);
      if (target != null) {
        await openSection(target);
      } else {
        readSeq.current++;
        setCurrentId(null);
        setSection(null);
        setReadPhase("idle");
      }
    } catch (e) {
      if (seq !== outlineSeq.current) return;
      setOutlineError(message(e));
      setOutlinePhase("failed");
      setOutline([]);
    }
  }, [openSection]);

  useEffect(() => { void loadList(); }, [loadList]);

  // 清单就绪而没有选中项（首次进入、删除后）→ 选中第一份。
  useEffect(() => {
    if (listPhase !== "ready" || selectedName) return;
    const first = proposals[0];
    if (first) void selectProposal(first.name);
  }, [listPhase, proposals, selectedName, selectProposal]);

  // 当前节点滚进标题树视野。
  useEffect(() => {
    if (currentId == null || !treeRef.current) return;
    const el = treeRef.current.querySelector<HTMLElement>(`[data-node-id="${currentId}"]`);
    el?.scrollIntoView?.({ block: "nearest" });
  }, [currentId, outlinePhase, pane, narrow]);

  const runSearch = useCallback(async () => {
    const q = input.trim();
    if (!q) {
      setSearch(null);
      return;
    }
    const seq = ++searchSeq.current;
    setSearch({ q, phase: "loading", mode: "", hits: [], note: "", error: "" });
    try {
      const res = await app.GaeaProposalSearch(q, scope === "title" ? "title" : "", levelFilter, typeFilter, yearFilter, "", PAGE_SEARCH_LIMIT);
      if (seq !== searchSeq.current) return;
      setSearch({ q, phase: "ready", mode: res?.mode ?? "", hits: res?.hits ?? [], note: res?.note ?? "", error: "" });
    } catch (e) {
      if (seq !== searchSeq.current) return;
      setSearch({ q, phase: "failed", mode: "", hits: [], note: "", error: message(e) });
    }
  }, [input, scope, levelFilter, typeFilter, yearFilter]);

  const exitSearch = () => {
    searchSeq.current++;
    setSearch(null);
  };

  const jumpToHit = (hit: ProposalHit) => {
    if (narrow) setPane("reader");
    if (hit.proposalName !== selectedName) {
      void selectProposal(hit.proposalName, hit.sectionId);
      return;
    }
    setCollapsed((c) => expandAncestors(c, outline, hit.sectionId));
    void openSection(hit.sectionId);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError("");
    try {
      await app.GaeaProposalDelete(deleteTarget.name);
    } catch (e) {
      setDeleteError(message(e));
      setDeleting(false);
      return;
    }
    setDeleting(false);
    setDeleteTarget(null);
    const wasSelected = deleteTarget.name === selectedName;
    if (wasSelected) {
      outlineSeq.current++;
      readSeq.current++;
      setSelectedName(null);
      setOutline([]);
      setCurrentId(null);
      setSection(null);
      setReadPhase("idle");
      setOutlinePhase("idle");
    }
    setProposals((list) => list.filter((p) => p.name !== deleteTarget.name));
    await loadList();
  };

  const copyPrompt = async () => {
    // 复制走仓内单源：成功才显示「已复制」，失败如实提示（不做假已复制）。
    setCopyState(await copyToClipboard(SAMPLE_PROMPT) ? "done" : "failed");
    window.setTimeout(() => setCopyState("idle"), 1600);
  };

  const selected = proposals.find((p) => p.name === selectedName) ?? null;
  const rows = useMemo(() => visibleRows(outline, collapsed), [outline, collapsed]);
  const currentIndex = currentId != null ? outline.findIndex((n) => n.id === currentId) : -1;
  const prevNode = currentIndex > 0 ? outline[currentIndex - 1] : null;
  const nextNode = currentIndex >= 0 && currentIndex < outline.length - 1 ? outline[currentIndex + 1] : null;
  const types = useMemo(() => Array.from(new Set(proposals.map((p) => p.projectType).filter(Boolean))).sort(), [proposals]);
  const years = useMemo(() => Array.from(new Set(proposals.map((p) => p.year).filter(Boolean))).sort().reverse(), [proposals]);
  const totalSections = proposals.reduce((n, p) => n + p.sections, 0);
  const libraryEmpty = listPhase === "ready" && proposals.length === 0;
  const shown = (p: Pane) => (narrow ? pane === p : true);

  const toggleNode = (id: number) =>
    setCollapsed((c) => {
      const next = new Set(c);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  // 键盘：上下/Home/End 在可见行间移动并读取；右键展开或进第一个子节，左键收起或回父节。
  const focusNode = (id: number) => {
    if (typeof window.requestAnimationFrame !== "function") return;
    window.requestAnimationFrame(() => {
      treeRef.current?.querySelector<HTMLElement>(`[data-node-id="${id}"] .pl-node-label`)?.focus();
    });
  };

  const onTreeKey = (e: KeyboardEvent) => {
    if (!rows.length) return;
    const move = (id: number | null) => {
      if (id == null) return;
      e.preventDefault();
      void openSection(id).then(() => focusNode(id));
    };
    switch (e.key) {
      case "ArrowDown":
        move(stepRow(rows, currentId, 1));
        return;
      case "ArrowUp":
        move(stepRow(rows, currentId, -1));
        return;
      case "Home":
        move(rows[0].node.id);
        return;
      case "End":
        move(rows[rows.length - 1].node.id);
        return;
      case "ArrowRight":
        if (currentId == null) return;
        e.preventDefault();
        if (collapsed.has(currentId)) toggleNode(currentId);
        else if (currentIndex >= 0 && currentIndex < outline.length - 1 && outline[currentIndex + 1].level > outline[currentIndex].level) move(outline[currentIndex + 1].id);
        return;
      case "ArrowLeft":
        if (currentId == null) return;
        e.preventDefault();
        if (!collapsed.has(currentId) && hasChildren(outline, currentId)) toggleNode(currentId);
        else move(parentOf(outline, currentId));
        return;
    }
  };

  return (
    <div ref={rootRef} className="proposal-library" data-narrow={narrow ? "true" : "false"} data-testid="proposal-library">
      <a className="pl-skip-link" href="#proposal-library-workspace">跳到方案库工作区</a>
      <style>{mdStyles}</style>

      <header className="pl-masthead">
        <div className="pl-brand">
          <span className="pl-mark" aria-hidden><BookOpen size={18} /></span>
          <div>
            <div className="pl-kicker">PROPOSAL LIBRARY</div>
            <div className="pl-title">方案库</div>
            <div className="pl-sub">历史方案 · 标题树定位 · 段落检索</div>
          </div>
        </div>
        <div className="pl-search" role="search">
          <div className="pl-search-box">
            <Search size={14} aria-hidden />
            <input
              aria-label="检索方案库"
              placeholder="关键词或自然语言问法，如：降水导致周边沉降怎么控制"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void runSearch();
              }}
            />
            {input && (
              <button type="button" className="pl-icon-btn" aria-label="清空检索词" onClick={() => { setInput(""); exitSearch(); }}>
                <X size={13} />
              </button>
            )}
          </div>
          <div className="pl-seg" role="group" aria-label="检索范围">
            <button type="button" aria-pressed={scope === "all"} onClick={() => setScope("all")}>全文 + 语义</button>
            <button type="button" aria-pressed={scope === "title"} onClick={() => setScope("title")}>只搜标题</button>
          </div>
          <select className="pl-select" aria-label="按类型筛选" value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)}>
            <option value="">全部类型</option>
            {types.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
          <select className="pl-select" aria-label="按年度筛选" value={yearFilter} onChange={(e) => setYearFilter(e.target.value)}>
            <option value="">全部年度</option>
            {years.map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
          <select className="pl-select" aria-label="按层级筛选" value={levelFilter} onChange={(e) => setLevelFilter(Number(e.target.value))}>
            <option value={0}>全部层级</option>
            {LEVEL_OPTIONS.map((lv) => <option key={lv} value={lv}>{lv} 级标题</option>)}
          </select>
          <button type="button" className="pl-btn pl-btn-primary" onClick={() => void runSearch()} disabled={!input.trim()}>
            搜索
          </button>
        </div>
      </header>

      <div className="pl-status" aria-live="polite" data-testid="proposal-search-status">
        {search ? (
          search.phase === "loading" ? (
            <span className="inline-flex items-center gap-2"><Loader2 size={13} className="pl-spin" aria-hidden />正在检索「{search.q}」…</span>
          ) : search.phase === "failed" ? (
            <span className="inline-flex items-center gap-2" role="alert">
              <AlertCircle size={13} aria-hidden />检索失败：{search.error}
              <button type="button" className="pl-btn pl-btn-ghost" onClick={() => void runSearch()}>重试</button>
            </span>
          ) : (
            <>
              <span>「{search.q}」命中 <strong>{search.hits.length}</strong> 条</span>
              <span className="pl-status-note">{searchSummary(search.mode, search.note)}</span>
              <button type="button" className="pl-btn pl-btn-ghost" onClick={exitSearch}>返回方案清单</button>
            </>
          )
        ) : listPhase === "ready" ? (
          <>
            <span>共 <strong>{proposals.length}</strong> 份方案 · <strong>{totalSections}</strong> 个标题节点</span>
            {selected && (
              <span className="pl-status-note">
                当前：《{selected.title}》{selected.status === "录入中" ? "（录入中，尚未收尾）" : ""}
              </span>
            )}
          </>
        ) : null}
      </div>

      {narrow && (
        <div className="pl-panes">
          <div className="pl-seg" role="tablist" aria-label="方案库视图">
            {([["list", search ? "命中" : "方案"], ["tree", "目录"], ["reader", "正文"]] as [Pane, string][]).map(([p, label]) => (
              <button
                key={p}
                type="button"
                role="tab"
                aria-selected={pane === p}
                onClick={() => setPane(p)}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
      )}

      {libraryEmpty ? (
        <div className="pl-empty" data-testid="proposal-empty">
          <div className="pl-empty-inner">
            <div className="pl-empty-mark" aria-hidden><BookOpen size={22} /></div>
            <h2 className="pl-empty-title">方案库还是空的</h2>
            <p className="pl-empty-lead">
              方案在办公板块的「方案」运行模式里录入：AI 读取历史方案，按标题层级存进来，
              正文原样保留。这里用来浏览章节骨架、检索段落、取材写新方案。
            </p>
            <ol className="pl-steps">
              <li className="pl-step">
                <span className="pl-step-index">01</span>
                <div>
                  <div className="pl-step-title">在办公板块切到「方案」运行模式</div>
                  <div className="pl-step-desc">参数面板里切换档位，开启新会话生效。</div>
                </div>
              </li>
              <li className="pl-step">
                <span className="pl-step-index">02</span>
                <div>
                  <div className="pl-step-title">让 AI 把历史方案录进来</div>
                  <div className="pl-step-desc">直接给出文件路径即可；长文档 AI 会分批录入，每批需要你确认一次写入。</div>
                  <div className="pl-prompt">
                    <code>{SAMPLE_PROMPT}</code>
                    <button type="button" className="pl-btn pl-btn-ghost" onClick={() => void copyPrompt()}>
                      <Copy size={13} aria-hidden />{copyState === "done" ? "已复制" : copyState === "failed" ? "复制失败" : "复制"}
                    </button>
                  </div>
                </div>
              </li>
              <li className="pl-step">
                <span className="pl-step-index">03</span>
                <div>
                  <div className="pl-step-title">回到这里浏览与检索</div>
                  <div className="pl-step-desc">写新方案时，方案模式里的 AI 也会自动查方案库取骨架和素材。</div>
                </div>
              </li>
            </ol>
            <div className="pl-empty-actions">
              <button type="button" className="pl-btn pl-btn-primary" onClick={openOfficeBoard}>
                <FolderOpen size={14} aria-hidden />去办公板块
              </button>
            </div>
          </div>
        </div>
      ) : (
        <section id="proposal-library-workspace" className="pl-body" aria-label="方案库工作区" tabIndex={-1}>
          {/* 第一栏：方案清单 / 检索命中 */}
          <div className={`pl-col${shown("list") ? " is-shown" : ""}`} aria-label={search ? "检索命中" : "方案清单"} hidden={narrow && !shown("list")}>
            {search ? (
              <>
                <div className="pl-col-head">
                  <span className="pl-col-head-title">命中 {search.phase === "ready" ? search.hits.length : "…"}</span>
                </div>
                {search.phase === "loading" && <StateNote icon="loading" title="正在检索…" />}
                {search.phase === "failed" && (
                  <StateNote icon="failed" title="检索失败" action={<button type="button" className="pl-btn" onClick={() => void runSearch()}><RefreshCw size={13} aria-hidden />重试</button>}>
                    {search.error}
                  </StateNote>
                )}
                {search.phase === "ready" && search.hits.length === 0 && (
                  <StateNote icon="empty" title="没有命中">
                    换个词或换种说法再试；两字以上的专业词最稳，也可以切到「只搜标题」按章节名找。
                  </StateNote>
                )}
                {search.phase === "ready" && search.hits.length > 0 && (
                  <div data-testid="proposal-hits">
                    {search.hits.map((hit) => (
                      <button
                        key={`${hit.proposalName}:${hit.sectionId}`}
                        type="button"
                        className={`pl-hit${hit.sectionId === currentId && hit.proposalName === selectedName ? " is-active" : ""}`}
                        onClick={() => jumpToHit(hit)}
                      >
                        <div className="pl-hit-proposal">{hit.proposalTitle}{hit.status === "录入中" ? "〔录入中〕" : ""}</div>
                        <div className="pl-hit-title">{hit.title}</div>
                        <div className="pl-hit-path">{hit.titlePath}</div>
                        <div className="pl-meta">
                          <span>{hit.level} 级</span>
                          <span>{formatChars(hit.chars)}</span>
                          {hit.childCount > 0 && <span>子节 {hit.childCount}</span>}
                        </div>
                        <Snippet text={hit.snippet} />
                      </button>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <>
                <div className="pl-col-head">
                  <span className="pl-col-head-title">方案清单</span>
                  <span>{listPhase === "ready" ? proposals.length : ""}</span>
                </div>
                {listPhase === "loading" && <StateNote icon="loading" title="正在读取方案库…" />}
                {listPhase === "failed" && (
                  <StateNote icon="failed" title="方案库读取失败" action={<button type="button" className="pl-btn" onClick={() => void loadList()}><RefreshCw size={13} aria-hidden />重试</button>}>
                    {listError || "数据库暂时读不到"}；这不是空库。
                  </StateNote>
                )}
                {listPhase === "ready" && (
                  <div data-testid="proposal-list">
                    {proposals.map((p) => (
                      <div key={p.name} className={`pl-row${p.name === selectedName ? " is-active" : ""}`}>
                        <button
                          type="button"
                          className="pl-item"
                          aria-current={p.name === selectedName ? "true" : undefined}
                          onClick={() => {
                            if (narrow) setPane("tree");
                            if (p.name !== selectedName) void selectProposal(p.name);
                          }}
                        >
                          <div className="pl-item-title" title={p.title}>{p.title}</div>
                          {p.summary && <div className="pl-item-summary">{p.summary}</div>}
                          <div className="pl-meta">
                            {p.projectType && <span>{p.projectType}</span>}
                            {p.year && <span>{p.year}</span>}
                            {p.status === "录入中" && <span className="pl-tag pl-tag-warn">录入中</span>}
                          </div>
                          <div className="pl-meta">
                            <span>{p.sections} 节</span>
                            <span>{formatChars(p.chars)}</span>
                            {indexLabel(p.indexDone, p.indexTotal) && <span className="is-pending">{indexLabel(p.indexDone, p.indexTotal)}</span>}
                          </div>
                        </button>
                        <button
                          type="button"
                          className="pl-icon-btn pl-row-delete"
                          aria-label={`删除方案 ${p.title}`}
                          title="删除方案"
                          onClick={() => setDeleteTarget(p)}
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>

          {/* 第二栏：标题树 */}
          <div className={`pl-col${shown("tree") ? " is-shown" : ""}`} aria-label="标题树" hidden={narrow && !shown("tree")}>
            <div className="pl-col-head">
              <span className="pl-col-head-title" title={selected?.title}>{selected ? selected.title : "标题树"}</span>
              {outlinePhase === "ready" && outline.length > 0 && (
                <>
                  <button type="button" className="pl-btn pl-btn-ghost" onClick={() => setCollapsed(new Set())}>全部展开</button>
                  <button type="button" className="pl-btn pl-btn-ghost" onClick={() => setCollapsed(initialCollapsed(outline))}>收起</button>
                </>
              )}
            </div>
            {outlinePhase === "idle" && <StateNote icon="empty" title="选择一份方案查看标题树" />}
            {outlinePhase === "loading" && <StateNote icon="loading" title="正在读取标题树…" />}
            {outlinePhase === "failed" && (
              <StateNote icon="failed" title="标题树读取失败" action={<button type="button" className="pl-btn" onClick={() => selectedName && void selectProposal(selectedName)}><RefreshCw size={13} aria-hidden />重试</button>}>
                {outlineError}
              </StateNote>
            )}
            {outlinePhase === "ready" && outline.length === 0 && <StateNote icon="empty" title="这份方案还没有标题节点" />}
            {outlinePhase === "ready" && outline.length > 0 && (
              <ul ref={treeRef} className="pl-tree" role="tree" aria-label={`${selected?.title ?? ""} 标题树`} onKeyDown={onTreeKey} data-testid="proposal-tree">
                {rows.map(({ node, depth }) => {
                  const active = node.id === currentId;
                  const expandable = hasChildren(outline, node.id);
                  const open = expandable && !collapsed.has(node.id);
                  return (
                    <li key={node.id} role="treeitem" aria-level={node.level} aria-selected={active} aria-expanded={expandable ? open : undefined}
                      className={`pl-node${active ? " is-active" : ""}${node.level === 1 ? " is-l1" : ""}`} data-node-id={node.id}>
                      <span style={{ width: `${depth * 16}px` }} aria-hidden />
                      {expandable ? (
                        <button
                          type="button"
                          className="pl-icon-btn pl-caret"
                          aria-label={open ? `收起 ${node.title}` : `展开 ${node.title}`}
                          aria-expanded={open}
                          tabIndex={-1}
                          onClick={() => toggleNode(node.id)}
                        >
                          {open ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
                        </button>
                      ) : (
                        <span className="pl-caret" aria-hidden />
                      )}
                      <div className="pl-node-label-wrap">
                        <button
                          type="button"
                          className="pl-node-label"
                          tabIndex={active || (currentId == null && node.id === rows[0]?.node.id) ? 0 : -1}
                          title={node.titlePath}
                          onClick={() => {
                            if (narrow) setPane("reader");
                            if (expandable && collapsed.has(node.id) && !active) toggleNode(node.id);
                            void openSection(node.id);
                          }}
                        >
                          <span className="pl-node-title">{node.title}</span>
                          <span className="pl-node-chars">{formatChars(node.chars)}</span>
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>

          {/* 第三栏：阅读窗 */}
          <div className={`pl-col${shown("reader") ? " is-shown" : ""}`} aria-label="小节正文" hidden={narrow && !shown("reader")}>
            {readPhase === "idle" && <StateNote icon="empty" title="在标题树里选一个标题阅读" />}
            {readPhase === "loading" && <StateNote icon="loading" title="正在读取小节…" />}
            {readPhase === "failed" && (
              <StateNote icon="failed" title="小节读取失败" action={<button type="button" className="pl-btn" onClick={() => currentId != null && void openSection(currentId)}><RefreshCw size={13} aria-hidden />重试</button>}>
                {readError}
              </StateNote>
            )}
            {readPhase === "ready" && section && (
              <article className="pl-reader" aria-labelledby="proposal-reader-title" data-testid="proposal-reader">
                <nav className="pl-crumbs" aria-label="标题路径">
                  {section.titlePath.split(" › ").map((crumb, i, all) => (
                    <span key={i} className={i === all.length - 1 ? "is-current" : undefined}>
                      {crumb}{i < all.length - 1 ? <span aria-hidden> › </span> : null}
                    </span>
                  ))}
                </nav>
                <h1 id="proposal-reader-title" className="pl-reader-title">{section.title}</h1>
                <div className="pl-reader-meta">
                  <span>{section.proposalTitle}{section.status === "录入中" ? "（录入中）" : ""}</span>
                  <span>本节 {formatChars(section.chars)}</span>
                  {section.children && section.children.length > 0 && <span>子节 {section.children.length}</span>}
                </div>
                {section.children && section.children.length > 0 && (
                  <div className="pl-child-chips" aria-label="子节">
                    {section.children.map((c) => (
                      <button
                        key={c.id}
                        type="button"
                        className="pl-child-chip"
                        title={c.title}
                        onClick={() => {
                          if (narrow) setPane("tree");
                          setCollapsed((cur) => expandAncestors(cur, outline, c.id));
                          void openSection(c.id);
                        }}
                      >
                        {c.title} · {formatChars(c.chars)}
                      </button>
                    ))}
                  </div>
                )}
                {section.body.trim() ? (
                  <MarkdownContent source={section.body} className="md-content pl-reader-body" />
                ) : (
                  <p className="pl-reader-body">本节标题下没有直接正文{section.children && section.children.length > 0 ? "，内容在上方的子节里" : ""}。</p>
                )}
                <div className="pl-reader-foot">
                  <button type="button" className="pl-btn" disabled={!prevNode} onClick={() => prevNode && void openSection(prevNode.id)}>
                    上一节{prevNode ? `：${prevNode.title}` : ""}
                  </button>
                  <button type="button" className="pl-btn" disabled={!nextNode} onClick={() => nextNode && void openSection(nextNode.id)}>
                    下一节{nextNode ? `：${nextNode.title}` : ""}
                  </button>
                </div>
              </article>
            )}
          </div>
        </section>
      )}

      <Modal
        title="删除方案"
        open={!!deleteTarget}
        onCancel={() => { if (!deleting) setDeleteTarget(null); }}
        onOk={() => void confirmDelete()}
        okText="删除"
        cancelText="取消"
        confirmLoading={deleting}
        okButtonProps={{ danger: true }}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={440}
      >
        {deleteTarget && (
          <div className="text-[13px] text-fg-dim leading-relaxed">
            确定删除《{deleteTarget.title}》吗？
            <div className="mt-2 text-fg-faint">
              它的 {deleteTarget.sections} 个标题节点与语义索引会一并清除，不可恢复。
              原文件不受影响，需要时可以在方案模式里重新录入。
            </div>
            {deleteError && <div className="mt-2 text-[12px] text-[var(--error)]" role="alert">删除失败：{deleteError}</div>}
          </div>
        )}
      </Modal>
    </div>
  );
}
