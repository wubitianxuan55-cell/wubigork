// ProposalLibraryPage — 一级板块「方案库」（与造价数据库并列）。
//
// 定位：历史方案的档案架。录入（整册重录、从文件导入）只在办公板块的「方案」运行
// 模式里由模型完成（proposal_save：模型定标题层级，正文由工具从源文件逐字切出）；
// 这里给人看、检索、直接修正入库错误：
//   方案清单 / 检索命中 → 1~4 级标题树（定位）→ 小节正文（阅读）。
// 2026-11-27：清单行「编辑」改元数据，阅读窗「编辑」改小节标题与正文。
// 2026-11-28 索引与编辑优化批：
//   编辑——结构修正（升降级 / 上下移 / 拆分 / 并入 / 增删小节；标题里的编号不改，
//     与新层级不符时在标题旁提示）、未保存守卫与 Ctrl+S / Esc、编辑⇄预览、
//     类型年度可选可填、勾选批量改、整册导出 Markdown；
//   索引——范围「全部方案 / 仅本方案」、命中定位高亮与节内查找、标题树筛选、
//     命中总数与按方案 / 按章分面（点一下收窄）、加载更多、文档顺序、多词放宽说明、
//     语义模型不可用提示与索引体检 / 重建、其他方案的同题小节对照。
// 检索与模型用的 proposal_search 同一条链：全文 + 本地语义召回 + 本地精排。
//
// 四态分明：加载 / 空库（三步引导）/ 无命中 / 读取失败（不伪装成空库）。
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Drawer, Modal } from "antd";
import {
  AlertCircle, BookOpen, ChevronDown, ChevronRight, Copy, Download, FolderOpen, Inbox, Loader2, Pencil, RefreshCw, Search, Trash2, X,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import { copyToClipboard } from "../gaea/lib/clipboard";
import { useBoardActive } from "../lib/boardActive";
import type {
  ProposalFacet, ProposalHit, ProposalIndexStatus, ProposalSectionBody, ProposalSectionNode, ProposalSectionOp, ProposalSummary, ProposalTwin,
} from "../gaea/lib/types/proposal";
import { MarkdownContent, mdStyles } from "../components/MarkdownContent";
import ReaderFind, { type LocateRequest } from "./proposal-library/Find";
import { StructureMenu } from "./proposal-library/StructureMenu";
import { IndexPanel } from "./proposal-library/IndexPanel";
import {
  changeLabel, expandAncestors, filterTree, formatChars, hasChildren, indexLabel, initialCollapsed, mismatchHint, numberMismatch, parentOf,
  queryTerms, safeFileName, searchSummary, sourceLabel, splitByTerms, splitPreview, splitSnippet, stepRow, structureFlags, structureReceipt,
  visibleRows,
} from "./proposal-library/logic";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "./proposal-library.css";

type Phase = "idle" | "loading" | "ready" | "failed";
type Pane = "list" | "tree" | "reader";
type Scope = "all" | "title";
type Range = "library" | "proposal";
type Sort = "" | "doc";
type SecView = "edit" | "preview";

interface SearchState {
  q: string;
  phase: Phase;
  mode: string;
  hits: ProposalHit[];
  total: number;
  byProposal: ProposalFacet[];
  byChapter: ProposalFacet[];
  note: string;
  /** 多词全含一个都没有，已放宽为任一词 */
  relaxed: boolean;
  /** 语义路失败的原因：本地语义模型不可用等 */
  searchError: string;
  error: string;
  loadingMore: boolean;
}

const PAGE_SEARCH_LIMIT = 30;
const NARROW_WIDTH = 820;
const LEVEL_OPTIONS = [1, 2, 3, 4] as const;
const SAMPLE_PROMPT = "把这份历史方案录入方案库，按标题层级分好：D:\\新工作\\项目目录\\技术标.docx";
const TYPE_OPTIONS_ID = "proposal-type-options";
const YEAR_OPTIONS_ID = "proposal-year-options";

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

/** 按检索词高亮一段文字（标题树筛选时的节点标题）。 */
function Marked({ text, terms }: { text: string; terms: string[] }) {
  if (!terms.length) return <>{text}</>;
  return (
    <>
      {splitByTerms(text, terms).map((p, i) => (p.hit ? <mark key={i}>{p.text}</mark> : <span key={i}>{p.text}</span>))}
    </>
  );
}

export default function ProposalLibraryPage() {
  const rootRef = useRef<HTMLDivElement>(null);
  const treeRef = useRef<HTMLUListElement>(null);
  const readerBodyRef = useRef<HTMLDivElement>(null);
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const outlineSeq = useRef(0);
  const readSeq = useRef(0);
  const searchSeq = useRef(0);
  const autoEditRef = useRef(false);

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
  // readVer：正文每重读一次加一，查找条靠它知道该清高亮、重新定位。
  const [readVer, setReadVer] = useState(0);

  const [input, setInput] = useState("");
  const [scope, setScope] = useState<Scope>("all");
  const [typeFilter, setTypeFilter] = useState("");
  const [yearFilter, setYearFilter] = useState("");
  const [levelFilter, setLevelFilter] = useState(0);
  const [search, setSearch] = useState<SearchState | null>(null);
  // 检索范围（全部方案 / 仅当前方案）、限定在某个标题之下、结果排序。
  const [range, setRange] = useState<Range>("library");
  const [under, setUnder] = useState(0);
  const [sort, setSort] = useState<Sort>("");

  const [deleteTarget, setDeleteTarget] = useState<ProposalSummary | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [copyState, setCopyState] = useState<"idle" | "done" | "failed">("idle");

  // ── 编辑面（2026-11-27）：直接修正入库错误 ──
  // 元数据：清单行「编辑」→ 弹窗改标题/类型/年度/摘要（标题即方案名，改名后按
  // 返回的新名重新定位选中行）。
  const [editTarget, setEditTarget] = useState<ProposalSummary | null>(null);
  const [editForm, setEditForm] = useState({ title: "", projectType: "", year: "", summary: "" });
  const [editBusy, setEditBusy] = useState(false);
  const [editError, setEditError] = useState("");
  // 小节：阅读窗「编辑」→ 标题 + 正文两格。保存只发真正改动的半边——没改的
  // 半边不传，避免白重嵌语义向量。
  const [secEditing, setSecEditing] = useState(false);
  const [secTitle, setSecTitle] = useState("");
  const [secBody, setSecBody] = useState("");
  const [secBusy, setSecBusy] = useState(false);
  const [secMsg, setSecMsg] = useState<{ text: string; isError: boolean } | null>(null);
  const [secView, setSecView] = useState<SecView>("edit");

  // ── 2026-11-28 编辑优化批：结构修正 ──
  const [structBusy, setStructBusy] = useState(false);
  const [structMsg, setStructMsg] = useState<{ text: string; isError: boolean } | null>(null);
  // 成功回执看几秒自己走；出错的留着，直到下一次操作或手动关掉。
  const structTimerRef = useRef<number | undefined>(undefined);
  const showStructMsg = (msg: { text: string; isError: boolean }) => {
    window.clearTimeout(structTimerRef.current);
    setStructMsg(msg);
    if (!msg.isError) structTimerRef.current = window.setTimeout(() => setStructMsg(null), 8000);
  };
  // 并入、删除要先确认；新增、拆分要问标题。
  const [confirmOp, setConfirmOp] = useState<ProposalSectionOp | null>(null);
  const [askTitle, setAskTitle] = useState<{ op: ProposalSectionOp; offset: number; title: string; saveDraft: boolean } | null>(null);

  // ── 2026-11-28：命中定位、节内查找、标题树筛选 ──
  const [locate, setLocate] = useState<LocateRequest | null>(null);
  const [findOpen, setFindOpen] = useState(false);
  const [treeFilter, setTreeFilter] = useState("");

  // ── 2026-11-28：元数据批量改、导出、索引体检、同题对照 ──
  const [checked, setChecked] = useState<string[]>([]);
  const [batchOpen, setBatchOpen] = useState(false);
  const [batchForm, setBatchForm] = useState({ projectType: "", year: "" });
  const [batchBusy, setBatchBusy] = useState(false);
  const [batchMsg, setBatchMsg] = useState("");
  const [exportMsg, setExportMsg] = useState<{ name: string; text: string; isError: boolean } | null>(null);
  const [indexStatus, setIndexStatus] = useState<ProposalIndexStatus | null>(null);
  const [twins, setTwins] = useState<ProposalTwin[] | null>(null);
  const [twinOpen, setTwinOpen] = useState<ProposalTwin | null>(null);
  const [twinBody, setTwinBody] = useState<ProposalSectionBody | null>(null);
  const [twinError, setTwinError] = useState("");

  // ── 未保存守卫：编辑小节时有改动，换节 / 换方案 / 检索跳转前先问一句 ──
  // 守卫读的是 ref。保存成功到重渲染之间这段空档里 ref 还是旧值「有改动」，会把
  // 紧随其后的重取拦下——所以保存成功时先把 ref 置回「没有」，下次渲染按 state 重算。
  const secDirty = secEditing && !!section && (secTitle !== section.title || secBody !== section.body);
  const secDirtyRef = useRef(false);
  secDirtyRef.current = secDirty;
  const sectionRef = useRef<ProposalSectionBody | null>(null);
  sectionRef.current = section;
  const guardDirtyRef = useRef((): boolean => true);
  guardDirtyRef.current = () =>
    !secDirtyRef.current || window.confirm("这一节的修改还没保存，切走后会丢失。仍要切走吗？");

  const openEdit = (p: ProposalSummary) => {
    setEditTarget(p);
    setEditForm({ title: p.title, projectType: p.projectType ?? "", year: p.year ?? "", summary: p.summary ?? "" });
    setEditError("");
  };
  const metaDirty = !!editTarget && (
    editForm.title !== editTarget.title || editForm.projectType !== (editTarget.projectType ?? "") ||
    editForm.year !== (editTarget.year ?? "") || editForm.summary !== (editTarget.summary ?? "")
  );

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

  // silent=true：切回板块时的后台重读——不清空正在看的清单（不闪「正在读取」），
  // 失败也保留旧快照，只记下错误由清单头部的重试入口带出。
  const loadList = useCallback(async (silent = false) => {
    if (!silent) setListPhase("loading");
    setListError("");
    try {
      const list = (await app.GaeaProposalList()) ?? [];
      setProposals(list);
      setListPhase("ready");
    } catch (e) {
      setListError(message(e));
      if (!silent) setListPhase("failed");
    }
  }, []);

  const openSection = useCallback(async (id: number): Promise<boolean> => {
    if (sectionRef.current?.id !== id && !guardDirtyRef.current()) return false;
    setCurrentId(id);
    setReadPhase("loading");
    setReadError("");
    const seq = ++readSeq.current;
    try {
      const body = await app.GaeaProposalRead(id);
      if (seq !== readSeq.current) return false;
      setSection(body);
      setReadPhase("ready");
      setReadVer((v) => v + 1);
      return true;
    } catch (e) {
      if (seq !== readSeq.current) return false;
      setReadError(message(e));
      setReadPhase("failed");
      return false;
    }
  }, []);

  // 选中一份方案：取标题树；focusId 给了就定位到它（检索命中跳转），否则落在首个节点。
  // 切换时先清空阅读窗——不让上一份方案的小节挂在新方案的标题树旁边。
  const selectProposal = useCallback(async (name: string, focusId?: number) => {
    if (!guardDirtyRef.current()) return;
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

  // 索引体检：挂载时读一次；读不到不打扰（检索照常，只是状态钮不出）。
  const loadIndex = useCallback(async () => {
    if (typeof app.GaeaProposalIndexStatus !== "function") return;
    try {
      setIndexStatus(await app.GaeaProposalIndexStatus());
    } catch {
      // 体检读不到不算故障：状态钮保持上次的读数。
    }
  }, []);
  useEffect(() => { void loadIndex(); }, [loadIndex]);

  // 本板块 keepAlive 常驻：切去办公板块录完新方案再切回来，组件不会重挂，挂载时
  // 那一次读取的清单就成了旧快照（2026-11-24 实弹：录了两份只看得到一份，检索却
  // 两份都能命中）。板块由隐藏转回可见时静默重读；首次挂载那次不算「切回」。
  const boardActive = useBoardActive("proposal");
  const awayRef = useRef(false);
  useEffect(() => {
    if (!boardActive) {
      awayRef.current = true;
      return;
    }
    if (!awayRef.current) return;
    awayRef.current = false;
    void loadList(true);
    void loadIndex();
  }, [boardActive, loadList, loadIndex]);

  // 清单就绪而没有选中项（首次进入、删除后）→ 选中第一份。
  useEffect(() => {
    if (listPhase !== "ready" || selectedName) return;
    const first = proposals[0];
    if (first) void selectProposal(first.name);
  }, [listPhase, proposals, selectedName, selectProposal]);

  // 选中的方案不在清单里了（别处被删，或整册重录换了标题）：清掉它的标题树与
  // 阅读窗，交给上面的规则重新选中第一份——不让已消失的方案挂在界面上。
  useEffect(() => {
    if (listPhase !== "ready" || !selectedName) return;
    if (proposals.some((p) => p.name === selectedName)) return;
    outlineSeq.current++;
    readSeq.current++;
    setSelectedName(null);
    setOutline([]);
    setCurrentId(null);
    setSection(null);
    setReadPhase("idle");
    setOutlinePhase("idle");
  }, [listPhase, proposals, selectedName]);

  // 当前节点滚进标题树视野。
  useEffect(() => {
    if (currentId == null || !treeRef.current) return;
    const el = treeRef.current.querySelector<HTMLElement>(`[data-node-id="${currentId}"]`);
    el?.scrollIntoView?.({ block: "nearest" });
  }, [currentId, outlinePhase, pane, narrow]);

  // 换节或重读：退出小节编辑态——不让上一节的草稿挂在新节上。新增 / 拆出的小节
  // 例外：直接进编辑态接着写（autoEditRef 是结构操作回来时留的记号）。
  useEffect(() => {
    if (autoEditRef.current && section) {
      autoEditRef.current = false;
      setSecTitle(section.title);
      setSecBody(section.body);
      setSecView("edit");
      setSecMsg(null);
      setSecEditing(true);
      return;
    }
    setSecEditing(false);
    setSecMsg(null);
  }, [currentId, section]);

  // ── 检索 ──
  const runSearch = useCallback(async (opts: { more?: boolean; range?: Range; sort?: Sort; under?: number; within?: string } = {}) => {
    const q = input.trim();
    if (!q) {
      setSearch(null);
      return;
    }
    const useRange = opts.range ?? range;
    const useSort = opts.sort ?? sort;
    const useUnder = opts.under ?? under;
    const within = opts.within ?? (useRange === "proposal" ? selectedName ?? "" : "");
    const prev = opts.more ? search : null;
    const offset = prev ? prev.hits.length : 0;
    const seq = ++searchSeq.current;
    if (prev) {
      setSearch({ ...prev, loadingMore: true, error: "" });
    } else {
      setSearch({ q, phase: "loading", mode: "", hits: [], total: 0, byProposal: [], byChapter: [], note: "", relaxed: false, searchError: "", error: "", loadingMore: false });
    }
    try {
      const res = await app.GaeaProposalSearch(
        q, scope === "title" ? "title" : "", levelFilter, typeFilter, yearFilter, within, useUnder, offset, useSort, PAGE_SEARCH_LIMIT,
      );
      if (seq !== searchSeq.current) return;
      const hits = res?.hits ?? [];
      const base = prev && prev.phase === "ready" ? prev.hits : [];
      setSearch({
        q, phase: "ready", mode: res?.mode ?? "", hits: [...base, ...hits],
        // 老后端不带 total：按「取到多少算多少」显示。
        total: typeof res?.total === "number" ? res.total : base.length + hits.length,
        byProposal: res?.byProposal ?? [], byChapter: res?.byChapter ?? [], note: res?.note ?? "",
        relaxed: !!res?.relaxed, searchError: res?.searchError ?? "", error: "", loadingMore: false,
      });
    } catch (e) {
      if (seq !== searchSeq.current) return;
      if (prev) {
        setSearch({ ...prev, loadingMore: false, error: message(e) });
      } else {
        setSearch({ q, phase: "failed", mode: "", hits: [], total: 0, byProposal: [], byChapter: [], note: "", relaxed: false, searchError: "", error: message(e), loadingMore: false });
      }
    }
  }, [input, scope, range, sort, under, levelFilter, typeFilter, yearFilter, selectedName, search]);

  const exitSearch = () => {
    searchSeq.current++;
    setSearch(null);
    setUnder(0);
  };

  // 范围 / 排序 / 限定标题是针对当前结果的收窄：有检索结果时改了就立刻重查。
  const refine = (next: { range?: Range; sort?: Sort; under?: number; within?: string }) => {
    if (next.range !== undefined) setRange(next.range);
    if (next.sort !== undefined) setSort(next.sort);
    if (next.under !== undefined) setUnder(next.under);
    if (input.trim()) void runSearch(next);
  };

  const jumpToHit = (hit: ProposalHit) => {
    if (!guardDirtyRef.current()) return;
    if (narrow) setPane("reader");
    // 定位用的针：命中摘要去掉【】标记；摘要只是标题路径（标题命中）时只按命中词找。
    const plain = hit.snippet.replace(/[【】]/g, "");
    const req: LocateRequest = { needle: plain === hit.titlePath ? "" : plain, fallback: hit.matched?.[0] ?? "" };
    if (hit.proposalName !== selectedName) {
      setLocate(req);
      void selectProposal(hit.proposalName, hit.sectionId);
      return;
    }
    setCollapsed((c) => expandAncestors(c, outline, hit.sectionId));
    setLocate(req);
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
    setChecked((list) => list.filter((name) => name !== deleteTarget.name));
    await loadList();
  };

  const confirmMetaEdit = async () => {
    if (!editTarget) return;
    const title = editForm.title.trim();
    if (!title) {
      setEditError("标题不能为空");
      return;
    }
    setEditBusy(true);
    setEditError("");
    try {
      const newName = await app.GaeaProposalUpdateMeta(
        editTarget.name, title, editForm.projectType.trim(), editForm.year.trim(), editForm.summary.trim(),
      );
      setEditBusy(false);
      setEditTarget(null);
      // 选中的方案改了名：重取清单后按新名重新定位（标题树与正文跟着走）。
      const wasSelected = editTarget.name === selectedName || editTarget.title === selectedName;
      await loadList();
      if (wasSelected && newName !== selectedName) await selectProposal(newName);
    } catch (e) {
      setEditError(message(e));
      setEditBusy(false);
    }
  };

  // 进小节编辑态；换节/重读后由 effect 退出（不让上一节的草稿挂在新节上）。
  const startSectionEdit = () => {
    if (!section) return;
    setSecTitle(section.title);
    setSecBody(section.body);
    setSecMsg(null);
    setSecView("edit");
    setSecEditing(true);
  };

  const saveSectionEdit = async (): Promise<boolean> => {
    if (!section) return false;
    const title = secTitle.trim();
    const updateTitle = !!title && title !== section.title;
    const updateBody = secBody !== section.body;
    if (!updateTitle && !updateBody) {
      setSecMsg({ text: "标题与正文都没有改动", isError: false });
      return false;
    }
    if (!title) {
      setSecMsg({ text: "标题不能为空", isError: true });
      return false;
    }
    setSecBusy(true);
    setSecMsg(null);
    try {
      await app.GaeaProposalUpdateSection(section.id, updateTitle ? title : section.title, updateTitle, secBody, updateBody);
      setSecBusy(false);
      secDirtyRef.current = false; // 已落盘：紧随其后的重取不该再被未保存守卫拦
      setSecEditing(false);
      const cid = section.id;
      // 改标题连带子树路径：标题树整棵重取并回到本节；只改正文：重读本节即可。
      if (updateTitle && selectedName) await selectProposal(selectedName, cid);
      else await openSection(cid);
      return true;
    } catch (e) {
      setSecMsg({ text: message(e), isError: true });
      setSecBusy(false);
      return false;
    }
  };

  // 编辑态的快捷键：Ctrl+S 保存、Esc 取消（有改动先问）。
  const saveSectionEditRef = useRef(saveSectionEdit);
  saveSectionEditRef.current = saveSectionEdit;
  useEffect(() => {
    if (!secEditing) return;
    const onKey = (e: globalThis.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "s") {
        e.preventDefault();
        void saveSectionEditRef.current();
      } else if (e.key === "Escape") {
        if (secDirtyRef.current && !window.confirm("这一节的修改还没保存，确定放弃吗？")) return;
        setSecEditing(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [secEditing]);

  // ── 结构修正 ──
  const flags = useMemo(() => structureFlags(outline, section?.id ?? 0), [outline, section]);

  const runStructure = async (op: ProposalSectionOp, offset: number, title: string, saveDraft = false) => {
    if (!section || !selectedName || structBusy) return;
    setStructBusy(true);
    setStructMsg(null);
    try {
      // 拆分时编辑框里有没保存的改动：先落盘再拆，光标位置才对得上。
      if (saveDraft && section) {
        const t = secTitle.trim();
        if ((t && t !== section.title) || secBody !== section.body) {
          await app.GaeaProposalUpdateSection(section.id, t || section.title, !!t && t !== section.title, secBody, secBody !== section.body);
        }
      }
      const res = await app.GaeaProposalSectionOp(section.id, op, offset, title);
      secDirtyRef.current = false; // 草稿已随本次操作落盘
      setSecEditing(false);
      showStructMsg({ text: structureReceipt(op, res), isError: false });
      void loadList(true);
      if (res.created > 0) autoEditRef.current = true;
      // 落点：回执给的小节（新增的、并入的目标、删除后承接的）；没给或已不在树里就留在本节。
      const focus = res.focusId || section.id;
      await selectProposal(selectedName, focus);
    } catch (e) {
      showStructMsg({ text: message(e), isError: true });
    }
    setStructBusy(false);
  };

  const pickStructure = (op: ProposalSectionOp) => {
    if (!section) return;
    if (op === "remove" || op === "mergeParent" || op === "mergePrev") {
      if (!guardDirtyRef.current()) return;
      setConfirmOp(op);
    } else if (op === "addChild" || op === "addSibling") {
      if (!guardDirtyRef.current()) return;
      setAskTitle({ op, offset: 0, title: "", saveDraft: false });
    } else {
      if (!guardDirtyRef.current()) return;
      void runStructure(op, 0, "");
    }
  };

  // 编辑态里「从光标处拆为新小节」：光标所在行及之后切出去。
  const askSplit = () => {
    const preview = splitPreview(secBody, areaRef.current?.selectionStart ?? 0);
    if (!preview) {
      setSecMsg({ text: "光标要放在要拆出去的那一段上：不能在第一行，之后要有正文", isError: true });
      return;
    }
    setSecMsg(null);
    setAskTitle({ op: "split", offset: preview.offset, title: preview.suggest, saveDraft: secDirty });
  };

  const confirmTitleOp = async () => {
    if (!askTitle || !section) return;
    const title = askTitle.title.trim();
    if (!title) return;
    const ask = askTitle;
    setAskTitle(null);
    await runStructure(ask.op, ask.offset, title, ask.saveDraft);
  };

  // ── 同题对照 ──
  useEffect(() => {
    const id = section?.id;
    if (!id || typeof app.GaeaProposalTwins !== "function") {
      setTwins(null);
      return;
    }
    let live = true;
    setTwins(null);
    app.GaeaProposalTwins(id).then((list) => { if (live) setTwins(list ?? []); }, () => { if (live) setTwins(null); });
    return () => { live = false; };
  }, [section?.id, readVer]);

  useEffect(() => {
    const twin = twinOpen;
    if (!twin) return;
    let live = true;
    setTwinBody(null);
    setTwinError("");
    app.GaeaProposalRead(twin.sectionId).then((body) => { if (live) setTwinBody(body); }, (e: unknown) => { if (live) setTwinError(message(e)); });
    return () => { live = false; };
  }, [twinOpen]);

  // ── 元数据：批量改、导出 ──
  const runBatch = async () => {
    const type = batchForm.projectType.trim();
    const year = batchForm.year.trim();
    if (!type && !year) {
      setBatchMsg("类型和年度至少填一项");
      return;
    }
    setBatchBusy(true);
    setBatchMsg("");
    const failed: string[] = [];
    for (const name of checked) {
      const p = proposals.find((it) => it.name === name);
      if (!p) continue;
      try {
        await app.GaeaProposalUpdateMeta(p.name, p.title, type || p.projectType, year || p.year, p.summary);
      } catch {
        failed.push(p.title);
      }
    }
    setBatchBusy(false);
    await loadList();
    setChecked([]);
    if (failed.length) setBatchMsg(`改了 ${checked.length - failed.length} 份；没改成：${failed.join("、")}`);
    else {
      setBatchOpen(false);
    }
  };

  const exportProposal = async (p: ProposalSummary) => {
    setExportMsg(null);
    try {
      // 另存对话框只选路径（不传内容就不写盘），图片要落同名 -assets 目录，得由后端来写。
      const picked = await app.SaveFileAs(`${safeFileName(p.title)}.md`, "");
      if (!picked) return;
      const res = await app.GaeaProposalExport(p.name, picked);
      const extra = res.images > 0 ? `，图片 ${res.images} 张在 ${res.assetsDir}` : "";
      setExportMsg({ name: p.name, text: `已导出到 ${res.path}${extra}${res.missing > 0 ? `；${res.missing} 处图片的托管文件缺失` : ""}`, isError: false });
    } catch (e) {
      setExportMsg({ name: p.name, text: message(e), isError: true });
    }
  };

  const reindex = useCallback(async () => {
    try {
      setIndexStatus(await app.GaeaProposalReindex());
    } catch (e) {
      // 结构类修复可能已经落了一部分：重读状态与清单，再把原因抛给面板显示。
      await loadIndex();
      await loadList(true);
      throw e;
    }
    await loadList(true);
  }, [loadIndex, loadList]);

  const copyPrompt = async () => {
    // 复制走仓内单源：成功才显示「已复制」，失败如实提示（不做假已复制）。
    setCopyState(await copyToClipboard(SAMPLE_PROMPT) ? "done" : "failed");
    window.setTimeout(() => setCopyState("idle"), 1600);
  };

  const selected = proposals.find((p) => p.name === selectedName) ?? null;
  const treeKeep = useMemo(() => (treeFilter.trim() ? filterTree(outline, treeFilter) : null), [outline, treeFilter]);
  const treeTerms = useMemo(() => (treeKeep ? queryTerms(treeFilter) : []), [treeKeep, treeFilter]);
  const rows = useMemo(
    () => (treeKeep
      ? outline.filter((node) => treeKeep.has(node.id)).map((node) => ({ node, depth: node.level - 1 }))
      : visibleRows(outline, collapsed)),
    [outline, collapsed, treeKeep],
  );
  const currentIndex = currentId != null ? outline.findIndex((n) => n.id === currentId) : -1;
  const prevNode = currentIndex > 0 ? outline[currentIndex - 1] : null;
  const nextNode = currentIndex >= 0 && currentIndex < outline.length - 1 ? outline[currentIndex + 1] : null;
  const types = useMemo(() => Array.from(new Set(proposals.map((p) => p.projectType).filter(Boolean))).sort(), [proposals]);
  const years = useMemo(() => Array.from(new Set(proposals.map((p) => p.year).filter(Boolean))).sort().reverse(), [proposals]);
  const totalSections = proposals.reduce((n, p) => n + p.sections, 0);
  const libraryEmpty = listPhase === "ready" && proposals.length === 0;
  const shown = (p: Pane) => (narrow ? pane === p : true);
  const underLabel = outline.find((n) => n.id === under)?.title ?? search?.byChapter.find((f) => Number(f.key) === under)?.label ?? "所选标题";
  // 检索时发现本地语义模型不可用（向量可能明明是齐的，语义检索却是坏的）。
  const searchDegraded = !!search && search.phase === "ready" && !!search.searchError;

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
      void openSection(id).then((ok) => { if (ok) focusNode(id); });
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

  const facetProposal = (key: string) => {
    if (range === "proposal" && selectedName === key) {
      refine({ range: "library", under: 0 });
      return;
    }
    refine({ range: "proposal", under: 0, within: key });
    if (selectedName !== key) void selectProposal(key);
  };

  const anyChecked = checked.length > 0;

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
          <div className="pl-seg" role="group" aria-label="检索哪些方案">
            <button type="button" aria-pressed={range === "library"} onClick={() => refine({ range: "library", under: 0 })}>全部方案</button>
            <button
              type="button"
              aria-pressed={range === "proposal"}
              disabled={!selected}
              title={selected ? `只在《${selected.title}》里找` : "先在清单里选一份方案"}
              data-testid="proposal-range-proposal"
              onClick={() => refine({ range: "proposal", under: 0 })}
            >
              仅本方案
            </button>
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
              <span>「{search.q}」共 <strong>{search.total}</strong> 处命中</span>
              {search.relaxed && <span className="pl-status-note">几个词没有同时出现的，已放宽为任一词</span>}
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

      {search?.phase === "ready" && search.searchError && (
        <div className="pl-notice is-warn" role="status" data-testid="proposal-search-degraded">
          <AlertCircle size={13} aria-hidden />
          <span>本地语义模型没有响应，这次只按关键词匹配。</span>
          <button type="button" className="pl-btn pl-btn-ghost" onClick={() => void runSearch()}>重试</button>
        </div>
      )}

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
                  <span className="pl-col-head-title">命中</span>
                  {search.phase === "ready" && (
                    <div className="pl-seg" role="group" aria-label="结果排序">
                      <button type="button" aria-pressed={sort === ""} onClick={() => refine({ sort: "" })}>相关度</button>
                      <button type="button" aria-pressed={sort === "doc"} onClick={() => refine({ sort: "doc" })}>文档顺序</button>
                    </div>
                  )}
                </div>
                {search.phase === "loading" && <StateNote icon="loading" title="正在检索…" />}
                {search.phase === "failed" && (
                  <StateNote icon="failed" title="检索失败" action={<button type="button" className="pl-btn" onClick={() => void runSearch()}><RefreshCw size={13} aria-hidden />重试</button>}>
                    {search.error}
                  </StateNote>
                )}
                {search.phase === "ready" && (
                  <>
                    {(search.byProposal.length > 0 || search.byChapter.length > 0 || under > 0) && (
                      <div className="pl-facets" data-testid="proposal-facets">
                        {search.byProposal.length > 0 && (
                          <div className="pl-facet-row">
                            <span className="pl-facet-label">按方案</span>
                            {search.byProposal.map((f) => {
                              const on = range === "proposal" && selectedName === f.key;
                              return (
                                <button
                                  key={f.key}
                                  type="button"
                                  className={`pl-facet${on ? " is-on" : ""}`}
                                  aria-pressed={on}
                                  data-testid="proposal-facet-proposal"
                                  onClick={() => facetProposal(f.key)}
                                >
                                  {f.label}<b>{f.count}</b>
                                </button>
                              );
                            })}
                          </div>
                        )}
                        {search.byChapter.length > 0 && (
                          <div className="pl-facet-row">
                            <span className="pl-facet-label">按章</span>
                            {search.byChapter.map((f) => {
                              const on = under === Number(f.key);
                              return (
                                <button
                                  key={f.key}
                                  type="button"
                                  className={`pl-facet${on ? " is-on" : ""}`}
                                  aria-pressed={on}
                                  data-testid="proposal-facet-chapter"
                                  onClick={() => refine({ under: on ? 0 : Number(f.key) })}
                                >
                                  {f.label}<b>{f.count}</b>
                                </button>
                              );
                            })}
                          </div>
                        )}
                        {under > 0 && (
                          <button type="button" className="pl-facet is-on" data-testid="proposal-under-clear" onClick={() => refine({ under: 0 })}>
                            限定在「{underLabel}」之下 ✕
                          </button>
                        )}
                      </div>
                    )}
                    {search.hits.length === 0 ? (
                      <StateNote icon="empty" title="没有命中">
                        换个词或换种说法再试；两字以上的专业词最稳，也可以切到「只搜标题」按章节名找。
                      </StateNote>
                    ) : (
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
                              {sourceLabel(hit.source) && <span className="pl-tag">{sourceLabel(hit.source)}</span>}
                            </div>
                            <Snippet text={hit.snippet} />
                          </button>
                        ))}
                      </div>
                    )}
                    {(search.hits.length < search.total || search.error) && search.hits.length > 0 && (
                      <div className="pl-hit-more">
                        <button
                          type="button"
                          className="pl-btn"
                          data-testid="proposal-load-more"
                          disabled={search.loadingMore}
                          onClick={() => void runSearch({ more: true })}
                        >
                          {search.loadingMore
                            ? <><Loader2 size={13} className="pl-spin" aria-hidden />正在读取…</>
                            : `加载更多（还有 ${search.total - search.hits.length} 条）`}
                        </button>
                        {search.error && <div className="pl-inline-error" role="alert">{search.error}</div>}
                      </div>
                    )}
                    {search.hits.length >= search.total && search.hits.length > PAGE_SEARCH_LIMIT && (
                      <p className="pl-list-end">已列出全部 {search.total} 条。</p>
                    )}
                  </>
                )}
              </>
            ) : (
              <>
                <div className="pl-col-head">
                  <span className="pl-col-head-title">方案清单</span>
                  <span>{listPhase === "ready" ? proposals.length : ""}</span>
                  {listPhase === "ready" && listError && (
                    <span className="pl-status-note" role="alert" title={listError}>刷新失败</span>
                  )}
                  {listPhase === "ready" && (
                    <button
                      type="button"
                      className="pl-icon-btn"
                      aria-label="刷新方案清单"
                      title={listError ? `刷新失败：${listError}（点击重试）` : "刷新方案清单"}
                      onClick={() => void loadList()}
                    >
                      <RefreshCw size={13} />
                    </button>
                  )}
                  {listPhase === "ready" && indexStatus && (
                    // 清单栏只在没检索时显示；检索时若发现语义模型不可用，退化的提示条另出。
                    <IndexPanel status={indexStatus} modelDown={!!searchDegraded} onReindex={reindex} />
                  )}
                </div>
                {listPhase === "loading" && <StateNote icon="loading" title="正在读取方案库…" />}
                {listPhase === "failed" && (
                  <StateNote icon="failed" title="方案库读取失败" action={<button type="button" className="pl-btn" onClick={() => void loadList()}><RefreshCw size={13} aria-hidden />重试</button>}>
                    {listError || "数据库暂时读不到"}；这不是空库。
                  </StateNote>
                )}
                {listPhase === "ready" && (
                  <div data-testid="proposal-list">
                    {anyChecked && (
                      <div className="pl-batch-bar" data-testid="proposal-batch-bar">
                        <span>已选 {checked.length} 份</span>
                        <button type="button" className="pl-btn pl-btn-ghost" data-testid="proposal-batch-open" onClick={() => { setBatchMsg(""); setBatchOpen(true); }}>
                          改类型 / 年度
                        </button>
                        <button type="button" className="pl-icon-btn" aria-label="取消选择" onClick={() => setChecked([])}>
                          <X size={13} />
                        </button>
                      </div>
                    )}
                    {exportMsg && (
                      <div className={`pl-notice${exportMsg.isError ? " is-error" : " is-ok"}`} role={exportMsg.isError ? "alert" : "status"} data-testid="proposal-export-msg">
                        {exportMsg.text}
                      </div>
                    )}
                    {proposals.map((p) => {
                      const on = checked.includes(p.name);
                      return (
                        <div key={p.name} className={`pl-row${p.name === selectedName ? " is-active" : ""}${on ? " is-checked" : ""}`}>
                          <input
                            type="checkbox"
                            className="pl-row-check"
                            aria-label={`选择方案 ${p.title}`}
                            checked={on}
                            onChange={(e) => setChecked((list) => (e.target.checked ? [...list, p.name] : list.filter((n) => n !== p.name)))}
                          />
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
                            {p.summary ? <div className="pl-item-summary">{p.summary}</div> : null}
                            <div className="pl-meta">
                              {p.projectType && <span>{p.projectType}</span>}
                              {p.year && <span>{p.year}</span>}
                              {p.status === "录入中" && <span className="pl-tag pl-tag-warn">录入中</span>}
                              {!p.summary && p.status !== "录入中" && <span className="pl-tag" title="还没有摘要：点右侧「编辑」补一段">摘要待补</span>}
                            </div>
                            <div className="pl-meta">
                              <span>{p.sections} 节</span>
                              <span>{formatChars(p.chars)}</span>
                              {indexLabel(p.indexDone, p.indexTotal) && <span className="is-pending">{indexLabel(p.indexDone, p.indexTotal)}</span>}
                            </div>
                          </button>
                          <button
                            type="button"
                            className="pl-icon-btn pl-row-export"
                            aria-label={`导出方案 ${p.title}`}
                            title="导出整册 Markdown（图片落到同名 -assets 目录）"
                            data-testid="proposal-export"
                            onClick={() => void exportProposal(p)}
                          >
                            <Download size={13} />
                          </button>
                          <button
                            type="button"
                            className="pl-icon-btn pl-row-edit"
                            aria-label={`编辑方案 ${p.title}`}
                            title="编辑标题 / 类型 / 年度 / 摘要"
                            data-testid="proposal-edit-open"
                            onClick={() => openEdit(p)}
                          >
                            <Pencil size={13} />
                          </button>
                          <button
                            type="button"
                            className="pl-icon-btn pl-row-delete"
                            aria-label={`删除方案 ${p.title}`}
                            title="删除方案"
                            onClick={() => { if (guardDirtyRef.current()) setDeleteTarget(p); }}
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      );
                    })}
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
            {outlinePhase === "ready" && outline.length > 0 && (
              <div className="pl-tree-filter">
                <Search size={12} aria-hidden />
                <input
                  aria-label="筛选标题"
                  placeholder="筛选标题：边输边留匹配的标题和它的上级"
                  value={treeFilter}
                  data-testid="proposal-tree-filter"
                  onChange={(e) => setTreeFilter(e.target.value)}
                />
                {treeFilter && (
                  <button type="button" className="pl-icon-btn" aria-label="清除标题筛选" data-testid="proposal-tree-filter-clear" onClick={() => setTreeFilter("")}>
                    <X size={12} />
                  </button>
                )}
                {treeKeep && <span className="pl-tree-filter-count">{rows.length} 项</span>}
              </div>
            )}
            {outlinePhase === "idle" && <StateNote icon="empty" title="选择一份方案查看标题树" />}
            {outlinePhase === "loading" && <StateNote icon="loading" title="正在读取标题树…" />}
            {outlinePhase === "failed" && (
              <StateNote icon="failed" title="标题树读取失败" action={<button type="button" className="pl-btn" onClick={() => selectedName && void selectProposal(selectedName)}><RefreshCw size={13} aria-hidden />重试</button>}>
                {outlineError}
              </StateNote>
            )}
            {outlinePhase === "ready" && outline.length === 0 && <StateNote icon="empty" title="这份方案还没有标题节点" />}
            {outlinePhase === "ready" && outline.length > 0 && rows.length === 0 && (
              <StateNote icon="empty" title="没有匹配的标题">换个词，或清除筛选。</StateNote>
            )}
            {outlinePhase === "ready" && rows.length > 0 && (
              <ul ref={treeRef} className="pl-tree" role="tree" aria-label={`${selected?.title ?? ""} 标题树`} onKeyDown={onTreeKey} data-testid="proposal-tree">
                {rows.map(({ node, depth }) => {
                  const active = node.id === currentId;
                  const expandable = hasChildren(outline, node.id);
                  const open = expandable && !collapsed.has(node.id);
                  const mismatch = numberMismatch(node.title, node.level);
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
                          title={mismatch ? mismatchHint(node.title, node.level) : node.titlePath}
                          onClick={() => {
                            if (narrow) setPane("reader");
                            if (expandable && collapsed.has(node.id) && !active) toggleNode(node.id);
                            void openSection(node.id);
                          }}
                        >
                          <span className="pl-node-title"><Marked text={node.title} terms={treeTerms} /></span>
                          {mismatch && (
                            <span className="pl-num-warn" role="img" aria-label={mismatchHint(node.title, node.level)} data-testid="proposal-num-warn">
                              <AlertCircle size={12} />
                            </span>
                          )}
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
          <div
            className={`pl-col${shown("reader") ? " is-shown" : ""}`}
            aria-label="小节正文"
            hidden={narrow && !shown("reader")}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "f") {
                e.preventDefault();
                setFindOpen(true);
              }
            }}
          >
            {readPhase === "idle" && <StateNote icon="empty" title="在标题树里选一个标题阅读" />}
            {readPhase === "loading" && <StateNote icon="loading" title="正在读取小节…" />}
            {readPhase === "failed" && (
              <StateNote icon="failed" title="小节读取失败" action={<button type="button" className="pl-btn" onClick={() => currentId != null && void openSection(currentId)}><RefreshCw size={13} aria-hidden />重试</button>}>
                {readError}
              </StateNote>
            )}
            {readPhase === "ready" && section && (
              secEditing ? (
                <div className="pl-reader" aria-label="编辑小节" data-testid="proposal-section-edit">
                  <nav className="pl-crumbs" aria-label="标题路径">
                    <span className="is-current">{section.titlePath}</span>
                  </nav>
                  <div className="pl-edit-tabs" role="tablist" aria-label="编辑 / 预览">
                    <button type="button" role="tab" aria-selected={secView === "edit"} onClick={() => setSecView("edit")}>编辑</button>
                    <button type="button" role="tab" aria-selected={secView === "preview"} data-testid="proposal-edit-preview" onClick={() => setSecView("preview")}>预览</button>
                    {secDirty && <span className="pl-edit-change">{changeLabel(section.body, secBody)}</span>}
                  </div>
                  {secView === "preview" ? (
                    <div className="pl-reader is-preview" data-testid="proposal-edit-preview-body">
                      <h1 className="pl-reader-title">{secTitle.trim() || section.title}</h1>
                      {secBody.trim() ? <MarkdownContent source={secBody} className="md-content pl-reader-body" /> : <p className="pl-reader-body">（正文为空）</p>}
                    </div>
                  ) : (
                    <>
                      <label className="pl-edit-field">
                        <span className="pl-edit-label">标题（连带子节标题路径）</span>
                        <input className="pl-edit-input" value={secTitle} data-testid="proposal-edit-title"
                          onChange={(e) => setSecTitle(e.target.value)} />
                      </label>
                      <label className="pl-edit-field">
                        <span className="pl-edit-label">正文（Markdown，整节替换）</span>
                        <textarea ref={areaRef} className="pl-edit-input pl-edit-area" rows={18} value={secBody} data-testid="proposal-edit-body"
                          onChange={(e) => setSecBody(e.target.value)} />
                      </label>
                      <button type="button" className="pl-btn pl-btn-ghost pl-split-btn" data-testid="proposal-split" title="光标所在行及之后的正文切成一个新的下级小节"
                        onClick={askSplit}>
                        从光标处拆为新小节
                      </button>
                    </>
                  )}
                  <div className="pl-edit-actions">
                    <button type="button" className="pl-btn pl-btn-primary" disabled={secBusy} data-testid="proposal-edit-save"
                      onClick={() => void saveSectionEdit()}>保存</button>
                    <button
                      type="button"
                      className="pl-btn"
                      disabled={secBusy}
                      onClick={() => {
                        if (secDirty && !window.confirm("这一节的修改还没保存，确定放弃吗？")) return;
                        setSecEditing(false);
                      }}
                    >
                      取消
                    </button>
                    <span className="pl-key-hint">Ctrl+S 保存 · Esc 取消</span>
                    {secMsg && (
                      <span className={secMsg.isError ? "text-[12px] text-[var(--error)]" : "text-[12px] text-fg-faint"} role={secMsg.isError ? "alert" : undefined} data-testid="proposal-edit-msg">
                        {secMsg.text}
                      </span>
                    )}
                  </div>
                </div>
              ) : (
                <article className="pl-reader" aria-labelledby="proposal-reader-title" data-testid="proposal-reader">
                  <ReaderFind
                    bodyRef={readerBodyRef}
                    version={`${section.id}:${readVer}`}
                    locate={locate}
                    onLocated={() => setLocate(null)}
                    open={findOpen}
                    onOpenChange={setFindOpen}
                  />
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
                    <button type="button" className="pl-btn pl-btn-ghost" data-testid="proposal-find-open" title="在本节里查找（Ctrl+F）" onClick={() => setFindOpen(true)}>
                      <Search size={12} aria-hidden />查找
                    </button>
                    <StructureMenu flags={flags} onPick={pickStructure} />
                    <button type="button" className="pl-btn pl-btn-ghost" data-testid="proposal-section-edit-open"
                      title="改这一节的标题或正文（正文原样录入的笔误在这里修）" onClick={() => { if (guardDirtyRef.current()) startSectionEdit(); }}>
                      <Pencil size={12} aria-hidden />编辑本节
                    </button>
                  </div>
                  {structBusy && <div className="pl-struct-msg" role="status">正在调整…</div>}
                  {structMsg && (
                    <div className={`pl-struct-msg${structMsg.isError ? " is-error" : ""}`} role={structMsg.isError ? "alert" : "status"} data-testid="proposal-structure-msg">
                      {structMsg.text}
                      {!structMsg.isError && <button type="button" className="pl-icon-btn" aria-label="关闭提示" onClick={() => setStructMsg(null)}><X size={12} /></button>}
                    </div>
                  )}
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
                  <div ref={readerBodyRef} className="pl-reader-body-wrap">
                    {section.body.trim() ? (
                      <MarkdownContent source={section.body} className="md-content pl-reader-body" />
                    ) : (
                      <p className="pl-reader-body">本节标题下没有直接正文{section.children && section.children.length > 0 ? "，内容在上方的子节里" : ""}。</p>
                    )}
                  </div>
                  {twins && twins.length > 0 && (
                    <div className="pl-twins" data-testid="proposal-twins">
                      <span className="pl-twins-label">其他方案的同题小节</span>
                      {twins.map((t) => (
                        <button
                          key={`${t.proposalName}:${t.sectionId}`}
                          type="button"
                          className="pl-child-chip pl-twin"
                          data-testid="proposal-twin"
                          title={t.titlePath}
                          onClick={() => setTwinOpen(t)}
                        >
                          {t.proposalTitle} › {t.title} · {formatChars(t.chars)}
                        </button>
                      ))}
                    </div>
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
              )
            )}
          </div>
        </section>
      )}

      <Modal
        title="编辑方案"
        open={!!editTarget}
        onCancel={() => {
          if (editBusy) return;
          if (metaDirty && !window.confirm("方案信息的修改还没保存，确定放弃吗？")) return;
          setEditTarget(null);
        }}
        onOk={() => void confirmMetaEdit()}
        okText="保存"
        cancelText="取消"
        confirmLoading={editBusy}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={480}
      >
        {editTarget && (
          <div className="flex flex-col gap-3 pt-1 text-[13px]">
            <datalist id={TYPE_OPTIONS_ID}>{types.map((t) => <option key={t} value={t} />)}</datalist>
            <datalist id={YEAR_OPTIONS_ID}>{years.map((y) => <option key={y} value={y} />)}</datalist>
            <label className="flex flex-col gap-1">
              <span className="text-fg-dim">标题（方案名随之改，库里唯一）</span>
              <input className="pl-edit-input" value={editForm.title} data-testid="proposal-meta-title"
                onChange={(e) => setEditForm((f) => ({ ...f, title: e.target.value }))} />
            </label>
            <div className="grid grid-cols-2 gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-fg-dim">类型（可选已有写法，也可手填）</span>
                <input className="pl-edit-input" list={TYPE_OPTIONS_ID} value={editForm.projectType} data-testid="proposal-meta-type"
                  onChange={(e) => setEditForm((f) => ({ ...f, projectType: e.target.value }))} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-fg-dim">年度</span>
                <input className="pl-edit-input" list={YEAR_OPTIONS_ID} value={editForm.year} data-testid="proposal-meta-year"
                  onChange={(e) => setEditForm((f) => ({ ...f, year: e.target.value }))} />
              </label>
            </div>
            <label className="flex flex-col gap-1">
              <span className="text-fg-dim">摘要</span>
              <textarea className="pl-edit-input pl-edit-area" rows={3} value={editForm.summary} data-testid="proposal-meta-summary"
                onChange={(e) => setEditForm((f) => ({ ...f, summary: e.target.value }))} />
            </label>
            {editError && <div className="text-[12px] text-[var(--error)]" role="alert" data-testid="proposal-meta-error">{editError}</div>}
          </div>
        )}
      </Modal>

      <Modal
        title={askTitle?.op === "split" ? "拆出新小节" : askTitle?.op === "addChild" ? "新增子节" : "新增同级小节"}
        open={!!askTitle}
        onCancel={() => setAskTitle(null)}
        onOk={() => void confirmTitleOp()}
        okText={askTitle?.op === "split" ? "拆分" : "新增"}
        cancelText="取消"
        okButtonProps={{ disabled: !askTitle?.title.trim() }}
        confirmLoading={structBusy}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={460}
      >
        {askTitle && (
          <div className="flex flex-col gap-3 pt-1 text-[13px]">
            <label className="flex flex-col gap-1">
              <span className="text-fg-dim">{askTitle.op === "split" ? "新小节的标题（光标所在行及之后的正文归它）" : "新小节的标题"}</span>
              <input
                className="pl-edit-input"
                data-testid="proposal-new-title"
                value={askTitle.title}
                autoFocus
                onChange={(e) => setAskTitle({ ...askTitle, title: e.target.value })}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && askTitle.title.trim()) void confirmTitleOp();
                }}
              />
            </label>
            {askTitle.op === "split" && askTitle.saveDraft && <p className="text-fg-faint">编辑框里没保存的修改会先保存，再拆分。</p>}
            <p className="text-fg-faint">标题里的编号请自己写对（例如「1.2.3」）；新增后可以直接编辑正文。</p>
          </div>
        )}
      </Modal>

      <Modal
        title={confirmOp === "remove" ? "删除小节" : "并入"}
        open={!!confirmOp}
        onCancel={() => setConfirmOp(null)}
        onOk={() => {
          const op = confirmOp;
          setConfirmOp(null);
          if (op) void runStructure(op, 0, "");
        }}
        okText={confirmOp === "remove" ? "删除" : "并入"}
        cancelText="取消"
        okButtonProps={{ danger: confirmOp === "remove" }}
        confirmLoading={structBusy}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={460}
      >
        {section && confirmOp && (
          <div className="text-[13px] text-fg-dim leading-relaxed" data-testid="proposal-structure-confirm">
            {confirmOp === "remove" ? (
              <>
                确定删除《{section.title}》吗？
                {flags.subtreeCount > 1 && <div className="mt-2 text-fg-faint">会连同 {flags.subtreeCount - 1} 个子节一起删除（约 {formatChars(flags.subtreeChars)}）。</div>}
                <div className="mt-2 text-fg-faint">全文索引与语义向量一并清除，不可恢复；原文件不受影响。</div>
              </>
            ) : confirmOp === "mergeParent" ? (
              <>
                把《{section.title}》的正文接到父节正文后面，然后去掉这个标题。
                {flags.subtreeCount > 1 && <div className="mt-2 text-fg-faint">它的 {flags.subtreeCount - 1} 个子节升一级，归到父节名下。</div>}
              </>
            ) : (
              <>把《{section.title}》的正文接到上一个同级小节的末尾，然后去掉这个标题。</>
            )}
          </div>
        )}
      </Modal>

      <Modal
        title={`批量改类型 / 年度（已选 ${checked.length} 份）`}
        open={batchOpen}
        onCancel={() => { if (!batchBusy) { setBatchOpen(false); setBatchMsg(""); } }}
        onOk={() => void runBatch()}
        okText="应用"
        cancelText="取消"
        confirmLoading={batchBusy}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        width={440}
      >
        <div className="flex flex-col gap-3 pt-1 text-[13px]">
          <datalist id={TYPE_OPTIONS_ID}>{types.map((t) => <option key={t} value={t} />)}</datalist>
          <datalist id={YEAR_OPTIONS_ID}>{years.map((y) => <option key={y} value={y} />)}</datalist>
          <p className="text-fg-dim">留空的项不动；逐份保存，哪份没成会点名。</p>
          <label className="flex flex-col gap-1">
            <span className="text-fg-dim">类型</span>
            <input className="pl-edit-input" list={TYPE_OPTIONS_ID} data-testid="proposal-batch-type" value={batchForm.projectType}
              onChange={(e) => setBatchForm((f) => ({ ...f, projectType: e.target.value }))} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-fg-dim">年度</span>
            <input className="pl-edit-input" list={YEAR_OPTIONS_ID} data-testid="proposal-batch-year" value={batchForm.year}
              onChange={(e) => setBatchForm((f) => ({ ...f, year: e.target.value }))} />
          </label>
          {batchMsg && <div className="text-[12px] text-[var(--error)]" role="alert" data-testid="proposal-batch-msg">{batchMsg}</div>}
        </div>
      </Modal>

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

      <Drawer
        title={twinOpen ? `同题对照 · ${twinOpen.proposalTitle}` : "同题对照"}
        open={!!twinOpen}
        onClose={() => setTwinOpen(null)}
        width={560}
        destroyOnHidden
        data-testid="proposal-twin-drawer"
      >
        {twinOpen && (
          <div className="pl-twin-body">
            <nav className="pl-crumbs" aria-label="标题路径"><span className="is-current">{twinOpen.titlePath}</span></nav>
            {twinError && <div className="pl-inline-error" role="alert">{twinError}</div>}
            {!twinBody && !twinError && <StateNote icon="loading" title="正在读取…" />}
            {twinBody && (
              <>
                <h2 className="pl-reader-title">{twinBody.title}</h2>
                <div className="pl-reader-meta"><span>{formatChars(twinBody.chars)}</span></div>
                {twinBody.body.trim()
                  ? <MarkdownContent source={twinBody.body} className="md-content pl-reader-body" />
                  : <p className="pl-reader-body">本节标题下没有直接正文。</p>}
              </>
            )}
            <div className="pl-reader-foot">
              <button
                type="button"
                className="pl-btn pl-btn-primary"
                data-testid="proposal-twin-jump"
                onClick={() => {
                  const twin = twinOpen;
                  setTwinOpen(null);
                  void selectProposal(twin.proposalName, twin.sectionId);
                }}
              >
                跳到这一节
              </button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}
